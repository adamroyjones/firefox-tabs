package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

//go:embed index.html.tmpl
var indexHTMLTmpl string

type load struct{}

func (l load) run() error {
	cfg, err := programCfgDir()
	if err != nil {
		return fmt.Errorf("finding the program configuration directory: %w", err)
	}
	dataDir := filepath.Join(cfg, "data")

	dataFiles, err := filepath.Glob(dataDir + "/*/*.json")
	if err != nil {
		return fmt.Errorf("looking for synced files: %w", err)
	}

	// host -> profile -> window -> [tab]
	hostToMap := make(map[string]map[string]map[int][]tab)
	for _, dataFile := range dataFiles {
		components := strings.Split(dataFile, "/")
		host, profile := components[len(components)-2], components[len(components)-1]
		profile = strings.TrimSuffix(profile, ".json")
		bs, err := os.ReadFile(dataFile)
		if err != nil {
			return fmt.Errorf("reading %q: %w", dataFile, err)
		}

		profileToMap, ok := hostToMap[host]
		if !ok {
			profileToMap = make(map[string]map[int][]tab)
			hostToMap[host] = profileToMap
		}

		var d data
		if err := json.Unmarshal(bs, &d); err != nil {
			return fmt.Errorf("unmarshalling the data file %q: %w", dataFile, err)
		}

		windowToTabs := make(map[int][]tab)
		for _, t := range d.Tabs {
			if utf8.RuneCountInString(t.Title) > 80 {
				for i := 77; i >= 0; i-- {
					if utf8.ValidString(t.Title[:i]) {
						t.Title = t.Title[:i] + "..."
						break
					}
				}
			}
			windowToTabs[t.Window] = append(windowToTabs[t.Window], t)
		}
		profileToMap[profile] = windowToTabs
	}

	parsedIndexHTMLTmpl, err := template.New("page").Parse(indexHTMLTmpl)
	if err != nil {
		return fmt.Errorf("parsing the HTML template: %w", err)
	}
	f, err := os.CreateTemp("", "firefox-tabs-*.html")
	if err != nil {
		return fmt.Errorf("creating a temporary file to render: %w", err)
	}
	if err := parsedIndexHTMLTmpl.Execute(f, hostToMap); err != nil {
		f.Close()
		return fmt.Errorf("executing the HTML template: %w", err)
	}
	f.Close()

	fmt.Printf("Attempting to open a browser at %s...\n", f.Name())
	if err := exec.Command("xdg-open", f.Name()).Run(); err != nil {
		return fmt.Errorf("failed to open the browser using xdg-open (URL: %q): %w", f.Name(), err)
	}
	return nil
}
