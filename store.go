package main

import (
	"cmp"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/pierrec/lz4/v4"
)

const rfc3339ms = "2006-01-02T15:04:05.999Z07:00"

type profile struct{ name, path string }

type store struct{}

func (s store) run() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding the home directory for the current user: %w", err)
	}
	dir := filepath.Join(home, ".mozilla", "firefox")

	profiles, err := s.extractProfiles(dir)
	if err != nil {
		return fmt.Errorf("extracting the Firefox profiles: %w", err)
	}

	cfg, err := programCfgDir()
	if err != nil {
		return fmt.Errorf("finding the program configuration directory: %w", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("fetching the machine's hostname: %w", err)
	}
	dataDir := filepath.Join(cfg, "data", hostname)
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("creating %q: %w", dataDir, err)
	}

	for _, profile := range profiles {
		jsonLZ4Path := filepath.Join(dir, profile.path, "sessionstore-backups", "recovery.jsonlz4")
		outputFile := filepath.Join(dataDir, profile.name+".json")
		if !s.shouldExtract(jsonLZ4Path, outputFile) {
			continue
		}

		bs, err := s.extractJSON(jsonLZ4Path)
		if err != nil {
			return fmt.Errorf("extracting JSON (path: %q): %w", jsonLZ4Path, err)
		}
		tabs, err := s.extractTabs(bs)
		if err != nil {
			return fmt.Errorf("extracting tabs from %q: %w", jsonLZ4Path, err)
		}

		w, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("creating %q: %w", outputFile, err)
		}
		defer w.Close()
		d := data{Tabs: tabs, Timestamp: time.Now().UTC().Format(rfc3339ms)}
		bs, err = json.MarshalIndent(d, "", "  ")
		if err != nil {
			return fmt.Errorf("preparing the JSON to write to %q: %w", outputFile, err)
		}
		if _, err := w.Write(append(bs, byte('\n'))); err != nil {
			return fmt.Errorf("writing to %q: %w", outputFile, err)
		}
	}

	return nil
}

func (s store) extractProfiles(dir string) ([]profile, error) {
	profilesPath := filepath.Join(dir, "profiles.ini")
	bs, err := os.ReadFile(profilesPath)
	if err != nil {
		return nil, fmt.Errorf("reading the profiles file %q: %w", profilesPath, err)
	}

	// This could be replaced with a full INI or TOML parser, but...
	profiles := []profile{}
	blocks := strings.Split(string(bs), "\n\n")
	for _, block := range blocks {
		lines := strings.Split(block, "\n")

		header := lines[0]
		header = strings.TrimSuffix(strings.TrimPrefix(header, "["), "]")
		if !strings.HasPrefix(header, "Profile") {
			continue
		}

		prof := profile{}
		for _, line := range lines[1:] {
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			if k == "Name" {
				prof.name = v
			}
			if k == "Path" {
				prof.path = v
			}
		}
		if prof.name == "" || prof.path == "" {
			return nil, fmt.Errorf("the Name and Path fields must both be set for the profile %q in %q", header, profilesPath)
		}
		profiles = append(profiles, prof)
	}
	return profiles, nil
}

func (s store) shouldExtract(jsonLZ4Path, outputFile string) bool {
	jsonLZ4FileInfo, err := os.Stat(jsonLZ4Path)
	if os.IsNotExist(err) {
		return false
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		return true
	}
	bs, err := os.ReadFile(outputFile)
	if err != nil {
		return true
	}
	var d struct {
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(bs, &d); err != nil {
		return true
	}
	t, err := time.Parse(rfc3339ms, d.Timestamp)
	if err != nil {
		return true
	}
	return t.Before(jsonLZ4FileInfo.ModTime())
}

func (s store) extractJSON(path string) ([]byte, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading the file containing tabs (%q): %w", path, err)
	}

	// This is a Mozilla LZ4 JSON file. This is structured as follows:
	//
	// - "mozLz4": A magic string to identify the format?
	// - "0": A version number for the format?
	// - A NUL byte.
	// - A (little-endian) uint32 that corresponds the length of the uncompressed
	//   payload.
	// - An LZ4 block that contains the compressed data.
	const prefix = "mozLz40\x00"
	if d := len(bs); d < 12 {
		return nil, fmt.Errorf("expected %q to contain at least 12 bytes (length: %d)", path, d)
	}
	if !slices.Equal(bs[:8], []byte(prefix)) {
		return nil, fmt.Errorf("failed to find a magic string %q in the file %q", prefix, path)
	}
	size, block := binary.LittleEndian.Uint32(bs[8:12]), bs[12:]
	out := make([]byte, size)
	bytesRead, err := lz4.UncompressBlock(block, out)
	if err != nil {
		return nil, fmt.Errorf("decompressing the LZ4 block from %q: %w", path, err)
	}
	if bytesRead != int(size) {
		return nil, fmt.Errorf("expected to uncompressed size to be %d, but it's %d", size, bytesRead)
	}
	return out, nil
}

func (s store) extractTabs(bs []byte) (tabs []tab, err error) {
	var d struct {
		Windows []struct {
			Tabs []struct {
				Entries []tab `json:"entries"`
				Index   int   `json:"index"`
			} `json:"tabs"`
		} `json:"windows"`
	}
	if err := json.Unmarshal(bs, &d); err != nil {
		return nil, fmt.Errorf("unmarhsalling the JSON in the .jsonlz4 file: %w", err)
	}

	out := []tab{}
	for _, window := range d.Windows {
		for _, tab := range window.Tabs {
			// "index" treats the array as 1-indexed, it seems.
			out = append(out, tab.Entries[tab.Index-1])
		}
	}
	slices.SortFunc(out, func(fst, snd tab) int {
		return cmp.Compare(strings.ToLower(fst.Title), strings.ToLower(snd.Title))
	})
	return out, nil
}
