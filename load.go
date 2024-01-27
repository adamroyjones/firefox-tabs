package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

	type row struct {
		Host, Profile string
		Window        int
		Title, URL    string
	}
	rows := []row{}
	for _, dataFile := range dataFiles {
		components := strings.Split(dataFile, "/")
		host, profile := components[len(components)-2], components[len(components)-1]
		profile = strings.TrimSuffix(profile, ".json")
		bs, err := os.ReadFile(dataFile)
		if err != nil {
			return fmt.Errorf("reading %q: %w", dataFile, err)
		}
		var d data
		if err := json.Unmarshal(bs, &d); err != nil {
			return fmt.Errorf("unmarshalling the data file %q: %w", dataFile, err)
		}
		for _, t := range d.Tabs {
			title := t.Title
			if len(title) > 80 {
				title = title[:77] + "..."
			}
			rows = append(rows, row{
				Host: host, Profile: profile,
				Window: t.Window,
				Title:  title, URL: t.URL,
			})
		}
	}

	const tmpl = `<!DOCTYPE html>
<html>
  <head>
    <title>firefox-tabs</title>
  </head>
  <body>
    <h1>firefox-tabs</h1>
    <table>
      <thead>
        <tr>
          <td>host</td>
          <td>profile</td>
          <td>window</td>
          <td>link</td>
        </tr>
      </thead>
      <tbody>
        {{range .}}
        <tr>
          <td>{{ .Host }}</td>
          <td>{{ .Profile }}</td>
          <td>{{ .Window }}</td>
          <td><a href={{ .URL }}>{{ .Title }}</a></td>
        </tr>
        {{end}}
      </tbody>
    </table>
  </body>
</html>
`
	parsedTmpl, err := template.New("page").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parsing the HTML template: %w", err)
	}
	f, err := os.CreateTemp("", "firefox-tabs-*.html")
	if err != nil {
		return fmt.Errorf("creating a temporary file to render: %w", err)
	}
	if err := parsedTmpl.Execute(f, rows); err != nil {
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
