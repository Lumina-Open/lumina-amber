package sync

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"lumina-amber/internal/meta"
	"lumina-amber/internal/model"
	"lumina-amber/internal/source"
)

type mockSource struct {
	id        string
	files     []model.MediaFile
	contents  map[string][]byte
	pullCount atomic.Int64
}

func (m *mockSource) Descriptor() source.SourceDescriptor {
	return source.SourceDescriptor{
		ID:            m.id,
		Name:          "Mock Phone",
		Kind:          model.SourceADB,
		State:         source.StateReady,
		StatusMessage: "Ready",
	}
}

func (m *mockSource) Scan(_ context.Context, _ []string, _ model.MediaFilter) ([]model.MediaFile, error) {
	return m.files, nil
}

func (m *mockSource) PullFile(_ context.Context, file model.MediaFile, localDest string) error {
	m.pullCount.Add(1)
	data := m.contents[file.RemotePath]
	return os.WriteFile(localDest, data, 0o644)
}

func (m *mockSource) RemoteSHA256(_ context.Context, _ model.MediaFile) (string, bool) {
	return "", false
}

func TestSyncOrganizeDedupAndIncremental(t *testing.T) {
	tmp := t.TempDir()
	targetDir := filepath.Join(tmp, "BackupTarget")
	_ = os.MkdirAll(targetDir, 0o755)

	src := &mockSource{
		id: "adb:MOCK1",
		files: []model.MediaFile{
			{
				SourceID:   "adb:MOCK1",
				RemotePath: "/sdcard/DCIM/Camera/IMG_20261112_100000.jpg",
				Name:       "IMG_20261112_100000.jpg",
				Size:       10,
				ModTime:    time.Date(2026, 11, 12, 10, 0, 0, 0, time.Local),
				Kind:       model.MediaPhoto,
			},
			{
				// Identical content under a different filename -> should be deduplicated
				SourceID:   "adb:MOCK1",
				RemotePath: "/sdcard/Pictures/WeChat/COPY_20261112_100000.jpg",
				Name:       "COPY_20261112_100000.jpg",
				Size:       10,
				ModTime:    time.Date(2026, 11, 12, 10, 5, 0, 0, time.Local),
				Kind:       model.MediaPhoto,
			},
			{
				// Same filename as file 0, different content -> should rename to _1
				SourceID:   "adb:MOCK1",
				RemotePath: "/sdcard/Pictures/Other/IMG_20261112_100000.jpg",
				Name:       "IMG_20261112_100000.jpg",
				Size:       12,
				ModTime:    time.Date(2026, 11, 12, 11, 0, 0, 0, time.Local),
				Kind:       model.MediaPhoto,
			},
		},
		contents: map[string][]byte{
			"/sdcard/DCIM/Camera/IMG_20261112_100000.jpg":     []byte("photo-aaaa"),
			"/sdcard/Pictures/WeChat/COPY_20261112_100000.jpg": []byte("photo-aaaa"),
			"/sdcard/Pictures/Other/IMG_20261112_100000.jpg":  []byte("photo-bbbbbb"),
		},
	}

	eng := NewEngine()
	opts := SyncOptions{
		Source:    src,
		TargetDir: targetDir,
		Filter:    model.MediaFilter{IncludePhotos: true, IncludeVideos: true},
	}

	stats1, err := eng.Run(context.Background(), opts, nil)
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if stats1.Exported != 2 || stats1.SkippedDedup != 1 || stats1.Failed != 0 {
		t.Fatalf("unexpected first run stats: %+v", stats1)
	}

	// Verify metadata lives inside <TargetDir>/meta/index.db
	if !meta.Exists(targetDir) {
		t.Fatalf("expected target meta index to exist at %s", meta.DBPath(targetDir))
	}
	idxCnt, histCnt, _, err := meta.ReadStats(targetDir)
	if err != nil || idxCnt != 2 || histCnt != 3 {
		t.Fatalf("unexpected target meta stats: idx=%d hist=%d err=%v", idxCnt, histCnt, err)
	}

	// Verify picture/YYYY/YYYY-MM/ layout (one of the two identical-content files won, collision file got _1 if both named IMG_)
	monthDir := filepath.Join(targetDir, "picture", "2026", "2026-11")
	entries, err := os.ReadDir(monthDir)
	if err != nil || len(entries) != 2 {
		t.Fatalf("expected 2 archived files in %s, got %d (err=%v)", monthDir, len(entries), err)
	}

	// Second run: all 3 files should be skipped from <TargetDir>/meta cache with 0 pulls!
	pullsBefore := src.pullCount.Load()
	stats2, err := eng.Run(context.Background(), opts, nil)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if stats2.SkippedCached != 3 || stats2.Exported != 0 {
		t.Fatalf("expected 3 cached skips on second run, got: %+v", stats2)
	}
	if got := src.pullCount.Load(); got != pullsBefore {
		t.Fatalf("expected 0 pulls on second run, got %d", got-pullsBefore)
	}
}

func TestPreExistingTargetDedupAndDryRun(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "PhoneDump")
	dcimDir := filepath.Join(srcDir, "DCIM", "Camera")
	_ = os.MkdirAll(dcimDir, 0o755)
	_ = os.WriteFile(filepath.Join(dcimDir, "IMG_20250501_120000.jpg"), []byte("same-image-bytes"), 0o644)
	_ = os.WriteFile(filepath.Join(dcimDir, "IMG_20260115_093000.jpg"), []byte("brand-new-image"), 0o644)

	targetDir := filepath.Join(tmp, "ExternalHDD")
	preExistingDir := filepath.Join(targetDir, "picture", "2025", "2025-05")
	_ = os.MkdirAll(preExistingDir, 0o755)
	// Pre-existing file with different filename but identical content
	_ = os.WriteFile(filepath.Join(preExistingDir, "OLD_RENAMED_PHOTO.jpg"), []byte("same-image-bytes"), 0o644)

	ms := source.NewMountedSource(source.NewMountedDescriptor(srcDir))
	eng := NewEngine()

	// DryRun should detect 1 new candidate + 1 duplicate without modifying target meta/ directory yet
	dryStats, err := eng.Run(context.Background(), SyncOptions{
		Source:     ms,
		SourceDirs: model.DefaultConfig().SourceDirs,
		Filter:     model.MediaFilter{IncludePhotos: true, IncludeVideos: true},
		TargetDir:  targetDir,
		DryRun:     true,
	}, nil)
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if dryStats.TotalScanned != 2 || dryStats.NewCandidates != 1 || dryStats.SkippedDedup != 1 {
		t.Fatalf("unexpected dry run stats: %+v", dryStats)
	}
	if meta.Exists(targetDir) {
		t.Fatalf("dry run on target without meta should not write meta/index.db")
	}

	// Real run should index pre-existing file into <targetDir>/meta/, dedup duplicate, and export new photo
	stats, err := eng.Run(context.Background(), SyncOptions{
		Source:     ms,
		SourceDirs: model.DefaultConfig().SourceDirs,
		Filter:     model.MediaFilter{IncludePhotos: true, IncludeVideos: true},
		TargetDir:  targetDir,
		DryRun:     false,
	}, nil)
	if err != nil {
		t.Fatalf("real run failed: %v", err)
	}
	if stats.Exported != 1 || stats.SkippedDedup != 1 {
		t.Fatalf("expected 1 exported + 1 dedup, got: %+v", stats)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "picture", "2026", "2026-01", "IMG_20260115_093000.jpg")); err != nil {
		t.Fatalf("expected new photo organized in picture/2026/2026-01: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "picture", "2025", "2025-05", "IMG_20250501_120000.jpg")); !os.IsNotExist(err) {
		t.Fatalf("duplicate photo should not have been copied")
	}
}

func TestExportToNewTargetDirAndPortableMeta(t *testing.T) {
	tmp := t.TempDir()
	src := &mockSource{
		id: "adb:MOCK_MULTI",
		files: []model.MediaFile{
			{
				SourceID:   "adb:MOCK_MULTI",
				RemotePath: "/sdcard/DCIM/Camera/IMG_20261113_120000.jpg",
				Name:       "IMG_20261113_120000.jpg",
				Size:       11,
				ModTime:    time.Date(2026, 11, 13, 12, 0, 0, 0, time.Local),
				Kind:       model.MediaPhoto,
			},
		},
		contents: map[string][]byte{
			"/sdcard/DCIM/Camera/IMG_20261113_120000.jpg": []byte("hello-photo"),
		},
	}

	targetA := filepath.Join(tmp, "TargetA")
	targetB := filepath.Join(tmp, "TargetB")
	_ = os.MkdirAll(targetA, 0o755)
	_ = os.MkdirAll(targetB, 0o755)

	eng := NewEngine()

	// Step 1: Backup to TargetA
	statsA, err := eng.Run(context.Background(), SyncOptions{
		Source:    src,
		TargetDir: targetA,
		Filter:    model.MediaFilter{IncludePhotos: true, IncludeVideos: true},
	}, nil)
	if err != nil || statsA.Exported != 1 {
		t.Fatalf("backup to TargetA failed: stats=%+v err=%v", statsA, err)
	}

	// Step 2: Backup same source to brand-new TargetB -> must export completely!
	statsB, err := eng.Run(context.Background(), SyncOptions{
		Source:    src,
		TargetDir: targetB,
		Filter:    model.MediaFilter{IncludePhotos: true, IncludeVideos: true},
	}, nil)
	if err != nil || statsB.Exported != 1 {
		t.Fatalf("backup to new TargetB failed: stats=%+v err=%v", statsB, err)
	}
	if _, err := os.Stat(filepath.Join(targetB, "picture", "2026", "2026-11", "IMG_20261113_120000.jpg")); err != nil {
		t.Fatalf("expected photo in TargetB: %v", err)
	}

	// Step 3: Move TargetB (simulating external drive mounted on another path/PC) and use a fresh Engine
	targetBMoved := filepath.Join(tmp, "TargetBMoved")
	if err := os.Rename(targetB, targetBMoved); err != nil {
		t.Fatalf("rename TargetB: %v", err)
	}
	freshEng := NewEngine()
	pullsBefore := src.pullCount.Load()
	statsMoved, err := freshEng.Run(context.Background(), SyncOptions{
		Source:    src,
		TargetDir: targetBMoved,
		Filter:    model.MediaFilter{IncludePhotos: true, IncludeVideos: true},
	}, nil)
	if err != nil {
		t.Fatalf("sync against moved TargetB failed: %v", err)
	}
	if statsMoved.SkippedCached != 1 || statsMoved.Exported != 0 || src.pullCount.Load() != pullsBefore {
		t.Fatalf("expected 1 cached skip with 0 pulls on moved target using meta/, got stats=%+v pulls=%d",
			statsMoved, src.pullCount.Load()-pullsBefore)
	}
}

func TestOrganizeFolderSeparatePictureVideoAndMove(t *testing.T) {
	tmp := t.TempDir()
	messyDir := filepath.Join(tmp, "MessyImport")
	nestedSub := filepath.Join(messyDir, "Trip")
	_ = os.MkdirAll(nestedSub, 0o755)

	rootPhoto := filepath.Join(messyDir, "IMG_20260801_101500.jpg")
	nestedVideo := filepath.Join(nestedSub, "VID_20260802_183000.mp4")
	dupPhoto := filepath.Join(nestedSub, "DUP_PHOTO.jpg")
	_ = os.WriteFile(rootPhoto, []byte("pic-unique-1"), 0o644)
	_ = os.WriteFile(nestedVideo, []byte("vid-unique-1"), 0o644)
	_ = os.WriteFile(dupPhoto, []byte("pic-unique-1"), 0o644)

	targetDir := filepath.Join(tmp, "Archive")
	_ = os.MkdirAll(targetDir, 0o755)

	eng := NewEngine()

	// Guard: same source and target must fail safely
	if _, err := eng.Run(context.Background(), SyncOptions{
		Source:    source.NewMountedSource(source.NewOrganizeFolderDescriptor(targetDir)),
		Filter:    model.MediaFilter{IncludePhotos: true, IncludeVideos: true},
		TargetDir: targetDir,
	}, nil); err == nil {
		t.Fatalf("expected error when source and target directories are identical")
	}

	// Copy organize first (DeleteSource=false): keeps source files, splits picture/ and video/
	src := source.NewMountedSource(source.NewOrganizeFolderDescriptor(messyDir))
	statsCopy, err := eng.Run(context.Background(), SyncOptions{
		Source:       src,
		Filter:       model.MediaFilter{IncludePhotos: true, IncludeVideos: true},
		TargetDir:    targetDir,
		DeleteSource: false,
	}, nil)
	if err != nil {
		t.Fatalf("copy organize failed: %v", err)
	}
	if statsCopy.Exported != 2 || statsCopy.SkippedDedup != 1 {
		t.Fatalf("expected 2 exported + 1 dedup, got: %+v", statsCopy)
	}

	picDest := filepath.Join(targetDir, "picture", "2026", "2026-08", "IMG_20260801_101500.jpg")
	vidDest := filepath.Join(targetDir, "video", "2026", "2026-08", "VID_20260802_183000.mp4")
	if _, err := os.Stat(picDest); err != nil {
		t.Fatalf("missing organized photo at %s: %v", picDest, err)
	}
	if _, err := os.Stat(vidDest); err != nil {
		t.Fatalf("missing organized video at %s: %v", vidDest, err)
	}
	// Originals still exist in copy mode
	if _, err := os.Stat(rootPhoto); err != nil {
		t.Fatalf("source photo should remain in copy mode")
	}

	// Move organize (DeleteSource=true): removes source files once verified in archive
	statsMove, err := eng.Run(context.Background(), SyncOptions{
		Source:       src,
		Filter:       model.MediaFilter{IncludePhotos: true, IncludeVideos: true},
		TargetDir:    targetDir,
		DeleteSource: true,
	}, nil)
	if err != nil {
		t.Fatalf("move organize failed: %v", err)
	}
	if statsMove.SkippedCached != 3 {
		t.Fatalf("expected 3 cached matches in move follow-up, got: %+v", statsMove)
	}
	for _, p := range []string{rootPhoto, nestedVideo, dupPhoto} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("expected source file %s to be deleted in move mode", p)
		}
	}
	// Archive files still intact
	if _, err := os.Stat(picDest); err != nil {
		t.Fatalf("archive photo disappeared after move cleanup: %v", err)
	}
	if _, err := os.Stat(vidDest); err != nil {
		t.Fatalf("archive video disappeared after move cleanup: %v", err)
	}
}

func TestConcurrentBatchSyncAndDedup(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "BurstCamera")
	_ = os.MkdirAll(srcDir, 0o755)

	const uniqueCount = 80
	const dupCount = 40
	for i := 0; i < uniqueCount; i++ {
		name := filepath.Join(srcDir, "IMG_20261113_"+string(rune('A'+i/26))+string(rune('a'+i%26))+".jpg")
		content := []byte("unique-payload-" + filepath.Base(name))
		if err := os.WriteFile(name, content, 0o644); err != nil {
			t.Fatalf("write unique file: %v", err)
		}
	}
	for i := 0; i < dupCount; i++ {
		name := filepath.Join(srcDir, "DUP_20261113_"+string(rune('A'+i/26))+string(rune('a'+i%26))+".jpg")
		// Duplicates match the first dupCount unique payloads
		origName := "IMG_20261113_" + string(rune('A'+i/26)) + string(rune('a'+i%26)) + ".jpg"
		content := []byte("unique-payload-" + origName)
		if err := os.WriteFile(name, content, 0o644); err != nil {
			t.Fatalf("write dup file: %v", err)
		}
	}

	targetDir := filepath.Join(tmp, "ConcurrentTarget")
	_ = os.MkdirAll(targetDir, 0o755)

	eng := NewEngine()
	src := source.NewMountedSource(source.NewOrganizeFolderDescriptor(srcDir))
	stats, err := eng.Run(context.Background(), SyncOptions{
		Source:    src,
		Filter:    model.MediaFilter{IncludePhotos: true, IncludeVideos: true},
		TargetDir: targetDir,
	}, nil)
	if err != nil {
		t.Fatalf("concurrent run failed: %v", err)
	}
	if stats.Exported != uniqueCount || stats.SkippedDedup != dupCount || stats.Failed != 0 {
		t.Fatalf("unexpected concurrent stats: got %+v, want Exported=%d SkippedDedup=%d",
			stats, uniqueCount, dupCount)
	}

	// Verify meta index count matches exported files cleanly
	idxCnt, histCnt, _, err := meta.ReadStats(targetDir)
	if err != nil || idxCnt != uniqueCount || histCnt != uniqueCount+dupCount {
		t.Fatalf("unexpected meta stats after concurrent run: idx=%d hist=%d err=%v", idxCnt, histCnt, err)
	}
}

func TestCheckTargetHealth(t *testing.T) {
	if h := CheckTargetHealth(""); h.Ready {
		t.Fatalf("empty target should not be ready")
	}
	tmp := t.TempDir()
	h := CheckTargetHealth(tmp)
	if !h.Ready || h.FreeBytes == 0 {
		t.Fatalf("temp dir should be ready with free bytes > 0: %+v", h)
	}
	missing := filepath.Join(tmp, "does-not-exist")
	if hm := CheckTargetHealth(missing); hm.Ready {
		t.Fatalf("non-existent dir should not be ready")
	}
}
