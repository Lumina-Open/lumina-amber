package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"lumina-photo-sync/internal/model"
)

const configKey = "app_config"

// Database manages SQLite persistence for application preferences (AppConfig).
// Per-target media hashes and sync history live inside each target directory's meta/ subdirectory.
type Database struct {
	conn *sql.DB
	path string
}

// Open opens (or creates) the SQLite settings database at dbPath and applies migrations.
func Open(dbPath string) (*Database, error) {
	if dbPath == "" {
		return nil, errors.New("database path cannot be empty")
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	d := &Database{conn: conn, path: dbPath}
	if err := d.migrate(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return d, nil
}

// Close closes the underlying SQLite connection.
func (d *Database) Close() error {
	if d == nil || d.conn == nil {
		return nil
	}
	return d.conn.Close()
}

// Path returns the filesystem path of the settings database.
func (d *Database) Path() string {
	return d.path
}

func (d *Database) migrate() error {
	pragmas := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA synchronous=NORMAL;`,
		`PRAGMA busy_timeout=5000;`,
	}
	for _, p := range pragmas {
		if _, err := d.conn.Exec(p); err != nil {
			return fmt.Errorf("apply pragma %q: %w", p, err)
		}
	}

	// Only persist app configuration here; drop any legacy hash/history tables from older builds
	// so the app never maintains target hashes globally.
	schema := `
	CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	DROP TABLE IF EXISTS target_index;
	DROP TABLE IF EXISTS sync_history;
	`
	if _, err := d.conn.Exec(schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	return nil
}

// LoadConfig loads the persisted AppConfig, returning defaults if none is stored yet.
func (d *Database) LoadConfig() (model.AppConfig, error) {
	cfg := model.DefaultConfig()
	var raw string
	err := d.conn.QueryRow(`SELECT value FROM settings WHERE key = ?`, configKey).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("query config: %w", err)
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return model.DefaultConfig(), fmt.Errorf("decode config: %w", err)
	}
	if len(cfg.SourceDirs) == 0 {
		cfg.SourceDirs = model.DefaultConfig().SourceDirs
	}
	if cfg.WorkMode != model.WorkModeOrganize {
		cfg.WorkMode = model.WorkModePhone
	}
	if cfg.ThemeMode == "" {
		cfg.ThemeMode = "light"
	}
	return cfg, nil
}

// SaveConfig persists AppConfig atomically.
func (d *Database) SaveConfig(cfg model.AppConfig) error {
	payload, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	_, err = d.conn.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		configKey, string(payload),
	)
	if err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
