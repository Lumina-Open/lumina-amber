package db

import (
	"path/filepath"
	"testing"

	"lumina-amber/internal/model"
)

func TestConfigPersistenceAndNoHashTables(t *testing.T) {
	tmp := t.TempDir()
	d, err := Open(filepath.Join(tmp, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	// Default config when fresh
	cfg, err := d.LoadConfig()
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	if !cfg.IncludePhotos || !cfg.IncludeVideos || cfg.ThemeMode != "light" || cfg.WorkMode != model.WorkModePhone {
		t.Fatalf("unexpected default config: %+v", cfg)
	}

	// Save custom config
	cfg.WorkMode = model.WorkModeOrganize
	cfg.OrganizeSourceDir = "/tmp/messy"
	cfg.OrganizeDeleteSource = true
	cfg.TargetDir = "/mnt/usb/Photos"
	cfg.DefaultSourceID = "adb:DEVICE123"
	cfg.DefaultSourceName = "Pixel 8"
	cfg.ThemeMode = "amoled"
	cfg.AddRecentTarget("/mnt/usb/Photos")
	if err := d.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded, err := d.LoadConfig()
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if loaded.TargetDir != "/mnt/usb/Photos" ||
		loaded.DefaultSourceID != "adb:DEVICE123" ||
		loaded.ThemeMode != "amoled" ||
		loaded.WorkMode != model.WorkModeOrganize ||
		!loaded.OrganizeDeleteSource {
		t.Fatalf("loaded config mismatch: %+v", loaded)
	}

	// App DB must not maintain target_index or sync_history tables.
	for _, tbl := range []string{"target_index", "sync_history"} {
		var name string
		err := d.conn.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&name)
		if err == nil {
			t.Fatalf("app db should not maintain table %q", tbl)
		}
	}
}
