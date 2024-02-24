package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const version = "0.0.4"

type tab struct {
	Window int    `json:"window"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

type data struct {
	Tabs      []tab  `json:"tabs"`
	Timestamp string `json:"timestamp"`
}

func main() {
	flag.Usage = func() {
		fmt.Println(`firefox-tabs: Store or load browser tabs.

Usage:
  Write the tabs out to ~/.config/firefox-tabs:
    firefox-tabs store

  Open the tabs from to ~/.config/firefox-tabs into a browser tab:
    firefox-tabs load

  Print version information and exit:
    firefox-tabs -v`)
	}

	var v bool
	flag.BoolVar(&v, "v", false, "Print version information and exit.")
	flag.Parse()

	if v {
		fmt.Println("firefox-tabs version " + version)
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	switch arg := flag.Arg(0); arg {
	case "load":
		err = load{}.run()
	case "store":
		err = store{}.run()
	default:
		err = fmt.Errorf("invalid subcommand %q: expected 'load' or 'store'", arg)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}

func programCfgDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("fetching the user configuration directory: %w", err)
	}
	return filepath.Join(cfg, "firefox-tabs"), nil
}
