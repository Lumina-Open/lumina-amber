package meta

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"lumina-photo-sync/internal/model"
)

const (
	// DirName is the subdirectory inside every backup target directory storing metadata.
	DirName = "meta"
	// DBFileName is the portable SQLite metadata index inside DirName.
	DBFileName = "index.db"
)

// DirPath returns the path to the target's meta subdirectory.
func DirPath(targetDir string) string {
	return filepath.Join(targetDir, DirName)
}

// DBPath returns the path to the target's meta index database.
func DBPath(targetDir string) string {
	return filepath.Join(DirPath(targetDir), DBFileName)
}

// Exists reports whether targetDir already contains a meta index database on disk.
func Exists(targetDir string) bool {
	if strings.TrimSpace(targetDir) == "" {
		return false
	}
	info, err := os.Stat(DBPath(targetDir))
	return err == nil && !info.IsDir()
}

// Store manages per-target photo/video hashes and sync history inside <targetDir>/meta/index.db.
// All stored paths are relative to targetDir so moving or unplugging the drive preserves every index.
type Store struct {
	conn       *sql.DB
	targetRoot string
	path       string
}

// Open opens (or creates) <targetDir>/meta/index.db and ensures schema is migrated.
func Open(targetDir string) (*Store, error) {
	if strings.TrimSpace(targetDir) == "" {
		return nil, errors.New("target directory cannot be empty")
	}

	targetRoot, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolve target root: %w", err)
	}

	metaDir := DirPath(targetRoot)
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		return nil, fmt.Errorf("create meta directory: %w", err)
	}

	dbPath := DBPath(targetRoot)
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open target meta sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	s := &Store{
		conn:       conn,
		targetRoot: targetRoot,
		path:       dbPath,
	}
	if err := s.migrate(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying SQLite connection so removable drives can eject cleanly.
func (s *Store) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

// Path returns the filesystem path of the meta SQLite file.
func (s *Store) Path() string {
	return s.path
}

func (s *Store) migrate() error {
	// Use DELETE journal mode so portable drives (exFAT/FAT32/NTFS/SMB) keep a clean single index.db
	// without lingering -wal/-shm shared-memory sidecars.
	pragmas := []string{
		`PRAGMA journal_mode=DELETE;`,
		`PRAGMA synchronous=NORMAL;`,
		`PRAGMA busy_timeout=5000;`,
	}
	for _, p := range pragmas {
		if _, err := s.conn.Exec(p); err != nil {
			return fmt.Errorf("apply pragma %q: %w", p, err)
		}
	}

	schema := `
	CREATE TABLE IF NOT EXISTS target_index (
		rel_path   TEXT PRIMARY KEY,
		media_kind TEXT NOT NULL DEFAULT '',
		file_size  INTEGER NOT NULL,
		sha256     TEXT NOT NULL,
		mtime      INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_meta_target_hash
		ON target_index (file_size, sha256);

	CREATE TABLE IF NOT EXISTS sync_history (
		source_id       TEXT NOT NULL,
		remote_path     TEXT NOT NULL,
		file_size       INTEGER NOT NULL,
		remote_mtime    INTEGER NOT NULL,
		sha256          TEXT NOT NULL,
		target_rel_path TEXT NOT NULL,
		synced_at       INTEGER NOT NULL,
		PRIMARY KEY (source_id, remote_path)
	);
	`
	if _, err := s.conn.Exec(schema); err != nil {
		return fmt.Errorf("create target meta schema: %w", err)
	}
	return nil
}

// LoadTargetIndex returns all indexed entries inside this target directory keyed by RelPath.
func (s *Store) LoadTargetIndex() (map[string]model.TargetIndexEntry, error) {
	rows, err := s.conn.Query(`SELECT rel_path, media_kind, file_size, sha256, mtime FROM target_index`)
	if err != nil {
		return nil, fmt.Errorf("query target_index: %w", err)
	}
	defer rows.Close()

	result := make(map[string]model.TargetIndexEntry)
	for rows.Next() {
		var e model.TargetIndexEntry
		var kindStr string
		if err := rows.Scan(&e.RelPath, &kindStr, &e.FileSize, &e.SHA256, &e.ModTime); err != nil {
			return nil, fmt.Errorf("scan target_index: %w", err)
		}
		e.MediaKind = model.MediaKind(kindStr)
		result[e.RelPath] = e
	}
	return result, rows.Err()
}

// HistoryKey uniquely identifies a synced source file inside a target's meta history.
type HistoryKey struct {
	SourceID   string
	RemotePath string
}

// UpsertTargetEntry inserts or updates a single target file record.
func (s *Store) UpsertTargetEntry(e model.TargetIndexEntry) error {
	return s.UpsertTargetEntriesBatch([]model.TargetIndexEntry{e})
}

// UpsertTargetEntriesBatch inserts or updates multiple target file records inside one transaction.
func (s *Store) UpsertTargetEntriesBatch(entries []model.TargetIndexEntry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := s.conn.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(
		`INSERT INTO target_index (rel_path, media_kind, file_size, sha256, mtime)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(rel_path) DO UPDATE SET
		   media_kind = excluded.media_kind,
		   file_size  = excluded.file_size,
		   sha256     = excluded.sha256,
		   mtime      = excluded.mtime`,
	)
	if err != nil {
		return fmt.Errorf("prepare target_index upsert: %w", err)
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.Exec(e.RelPath, string(e.MediaKind), e.FileSize, e.SHA256, e.ModTime); err != nil {
			return fmt.Errorf("upsert target_index: %w", err)
		}
	}
	return tx.Commit()
}

// DeleteTargetEntries removes stale records for files no longer present on disk.
func (s *Store) DeleteTargetEntries(relPaths []string) error {
	if len(relPaths) == 0 {
		return nil
	}
	tx, err := s.conn.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`DELETE FROM target_index WHERE rel_path = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, rp := range relPaths {
		if _, err := stmt.Exec(rp); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ClearTargetIndex deletes all target_index rows to force a clean re-index.
func (s *Store) ClearTargetIndex() error {
	_, err := s.conn.Exec(`DELETE FROM target_index`)
	return err
}

// HasTargetHash checks whether a file with matching size and SHA-256 already exists in this target.
func (s *Store) HasTargetHash(size int64, sha256Hex string) (string, bool, error) {
	var relPath string
	err := s.conn.QueryRow(
		`SELECT rel_path FROM target_index WHERE file_size = ? AND sha256 = ? LIMIT 1`,
		size, sha256Hex,
	).Scan(&relPath)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return relPath, true, nil
}

// HasTargetSize checks whether any indexed file in this target has the given byte size.
func (s *Store) HasTargetSize(size int64) (bool, error) {
	var dummy int
	err := s.conn.QueryRow(
		`SELECT 1 FROM target_index WHERE file_size = ? LIMIT 1`,
		size,
	).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetSyncHistory looks up a cached sync history entry for a source file in this target.
func (s *Store) GetSyncHistory(sourceID, remotePath string) (model.SyncHistoryEntry, bool, error) {
	var e model.SyncHistoryEntry
	e.SourceID = sourceID
	e.RemotePath = remotePath

	err := s.conn.QueryRow(
		`SELECT file_size, remote_mtime, sha256, target_rel_path, synced_at
		 FROM sync_history
		 WHERE source_id = ? AND remote_path = ?`,
		sourceID, remotePath,
	).Scan(&e.FileSize, &e.RemoteModTime, &e.SHA256, &e.TargetRelPath, &e.SyncedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return e, false, nil
	}
	if err != nil {
		return e, false, err
	}
	return e, true, nil
}

// LoadSyncHistory loads all cached sync history entries into memory for O(1) incremental checks.
func (s *Store) LoadSyncHistory() (map[HistoryKey]model.SyncHistoryEntry, error) {
	rows, err := s.conn.Query(`SELECT source_id, remote_path, file_size, remote_mtime, sha256, target_rel_path, synced_at FROM sync_history`)
	if err != nil {
		return nil, fmt.Errorf("query sync_history: %w", err)
	}
	defer rows.Close()

	result := make(map[HistoryKey]model.SyncHistoryEntry)
	for rows.Next() {
		var e model.SyncHistoryEntry
		if err := rows.Scan(&e.SourceID, &e.RemotePath, &e.FileSize, &e.RemoteModTime, &e.SHA256, &e.TargetRelPath, &e.SyncedAt); err != nil {
			return nil, fmt.Errorf("scan sync_history: %w", err)
		}
		result[HistoryKey{SourceID: e.SourceID, RemotePath: e.RemotePath}] = e
	}
	return result, rows.Err()
}

// UpsertSyncHistory stores or updates a sync history entry.
func (s *Store) UpsertSyncHistory(e model.SyncHistoryEntry) error {
	return s.UpsertSyncHistoryBatch([]model.SyncHistoryEntry{e})
}

// UpsertSyncHistoryBatch stores or updates multiple sync history rows inside one transaction.
func (s *Store) UpsertSyncHistoryBatch(entries []model.SyncHistoryEntry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := s.conn.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(
		`INSERT INTO sync_history (source_id, remote_path, file_size, remote_mtime, sha256, target_rel_path, synced_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(source_id, remote_path) DO UPDATE SET
		   file_size       = excluded.file_size,
		   remote_mtime    = excluded.remote_mtime,
		   sha256          = excluded.sha256,
		   target_rel_path = excluded.target_rel_path,
		   synced_at       = excluded.synced_at`,
	)
	if err != nil {
		return fmt.Errorf("prepare sync_history upsert: %w", err)
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.Exec(e.SourceID, e.RemotePath, e.FileSize, e.RemoteModTime, e.SHA256, e.TargetRelPath, e.SyncedAt); err != nil {
			return fmt.Errorf("upsert sync_history: %w", err)
		}
	}
	return tx.Commit()
}

// ClearSyncHistory removes all cached source-to-target history rows.
func (s *Store) ClearSyncHistory() error {
	_, err := s.conn.Exec(`DELETE FROM sync_history`)
	return err
}

// CountStats returns summary counts of indexed target files and cached sync history rows.
func (s *Store) CountStats() (indexedCount int, historyCount int, err error) {
	if err = s.conn.QueryRow(`SELECT COUNT(*) FROM target_index`).Scan(&indexedCount); err != nil {
		return 0, 0, err
	}
	if err = s.conn.QueryRow(`SELECT COUNT(*) FROM sync_history`).Scan(&historyCount); err != nil {
		return 0, 0, err
	}
	return indexedCount, historyCount, nil
}

// ReadStats reads counts from targetDir's meta store without creating files if meta doesn't exist yet.
func ReadStats(targetDir string) (indexedCount int, historyCount int, metaDBPath string, err error) {
	if strings.TrimSpace(targetDir) == "" {
		return 0, 0, "", nil
	}
	absDir, absErr := filepath.Abs(targetDir)
	if absErr != nil {
		absDir = targetDir
	}
	metaDBPath = DBPath(absDir)
	if !Exists(absDir) {
		return 0, 0, metaDBPath, nil
	}
	s, err := Open(absDir)
	if err != nil {
		return 0, 0, metaDBPath, err
	}
	defer func() { _ = s.Close() }()

	indexedCount, historyCount, err = s.CountStats()
	return indexedCount, historyCount, s.Path(), err
}

// ClearIndex clears target_index inside targetDir's meta store if it exists.
func ClearIndex(targetDir string) error {
	if !Exists(targetDir) {
		return nil
	}
	s, err := Open(targetDir)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	return s.ClearTargetIndex()
}

// ClearHistory clears sync_history inside targetDir's meta store if it exists.
func ClearHistory(targetDir string) error {
	if !Exists(targetDir) {
		return nil
	}
	s, err := Open(targetDir)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	return s.ClearSyncHistory()
}
