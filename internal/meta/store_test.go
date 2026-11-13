package meta

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"lumina-photo-sync/internal/model"
)

func TestTargetMetaStoreLifecycleAndPortability(t *testing.T) {
	tmp := t.TempDir()
	targetA := filepath.Join(tmp, "BackupA")
	_ = os.MkdirAll(targetA, 0o755)

	// ReadStats on a brand-new folder must not create meta/ directory on disk.
	idxCnt, histCnt, _, err := ReadStats(targetA)
	if err != nil || idxCnt != 0 || histCnt != 0 {
		t.Fatalf("unexpected empty ReadStats: idx=%d hist=%d err=%v", idxCnt, histCnt, err)
	}
	if Exists(targetA) {
		t.Fatalf("ReadStats must not create meta/index.db eagerly")
	}

	s, err := Open(targetA)
	if err != nil {
		t.Fatalf("open meta store: %v", err)
	}
	if err := s.UpsertTargetEntry(model.TargetIndexEntry{
		RelPath:   "picture/2026/2026-11/IMG_001.jpg",
		MediaKind: model.MediaPhoto,
		FileSize:  1024,
		SHA256:    "abc123",
		ModTime:   time.Now().Unix(),
	}); err != nil {
		t.Fatalf("upsert target entry: %v", err)
	}
	if err := s.UpsertSyncHistory(model.SyncHistoryEntry{
		SourceID:      "adb:DEV1",
		RemotePath:    "/sdcard/DCIM/Camera/IMG_001.jpg",
		FileSize:      1024,
		RemoteModTime: 1700000000,
		SHA256:        "abc123",
		TargetRelPath: "picture/2026/2026-11/IMG_001.jpg",
		SyncedAt:      time.Now().Unix(),
	}); err != nil {
		t.Fatalf("upsert sync history: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	// Verify meta/index.db exists inside BackupA/meta/
	if !Exists(targetA) {
		t.Fatalf("expected %s to exist", DBPath(targetA))
	}

	// Simulate plugging external drive under a different path / renaming directory
	targetMoved := filepath.Join(tmp, "ExternalDriveMoved")
	if err := os.Rename(targetA, targetMoved); err != nil {
		t.Fatalf("rename target dir: %v", err)
	}

	sMoved, err := Open(targetMoved)
	if err != nil {
		t.Fatalf("reopen moved meta store: %v", err)
	}
	defer sMoved.Close()

	rel, ok, err := sMoved.HasTargetHash(1024, "abc123")
	if err != nil || !ok || rel != "picture/2026/2026-11/IMG_001.jpg" {
		t.Fatalf("expected portable hash hit after folder move, got rel=%q ok=%v err=%v", rel, ok, err)
	}
	hist, found, err := sMoved.GetSyncHistory("adb:DEV1", "/sdcard/DCIM/Camera/IMG_001.jpg")
	if err != nil || !found || hist.SHA256 != "abc123" {
		t.Fatalf("expected portable sync history hit after folder move, got found=%v hist=%+v err=%v", found, hist, err)
	}
}
