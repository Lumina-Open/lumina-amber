package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"lumina-amber/internal/ui"
	"lumina-amber/internal/version"
)

func main() {
	var dbPathFlag string
	var showVersion bool
	flag.StringVar(&dbPathFlag, "db", "", "Path to the lumina_amber.db database file")
	flag.BoolVar(&showVersion, "version", false, "Print version information and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s v%s (author: %s)\n", version.AppName, version.Version, version.Author)
		return
	}

	dbPath := dbPathFlag
	if dbPath == "" {
		cfgDir, err := os.UserConfigDir()
		if err != nil || cfgDir == "" {
			homeDir, _ := os.UserHomeDir()
			cfgDir = filepath.Join(homeDir, ".config")
		}
		dbPath = filepath.Join(cfgDir, "lumina-amber", "lumina_amber.db")
	}

	app, err := ui.NewAmberApp(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error initializing %s: %v\n", version.AppName, err)
		os.Exit(1)
	}

	app.Run()
}
