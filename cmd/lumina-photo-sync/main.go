package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"lumina-photo-sync/internal/ui"
	"lumina-photo-sync/internal/version"
)

func main() {
	var dbPathFlag string
	var showVersion bool
	flag.StringVar(&dbPathFlag, "db", "", "Path to the lumina_sync.db database file")
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
		dbPath = filepath.Join(cfgDir, "lumina-photo-sync", "lumina_sync.db")
	}

	app, err := ui.NewPhotoSyncApp(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error initializing Lumina Photo Sync: %v\n", err)
		os.Exit(1)
	}

	app.Run()
}
