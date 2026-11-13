package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"lumina-amber/internal/meta"
	"lumina-amber/internal/model"
	"lumina-amber/internal/organizer"
	"lumina-amber/internal/source"
)

const (
	stagingDirName   = ".lumina-sync-tmp"
	dbBatchSize      = 64
	copyBufBytes     = 1024 * 1024
	maxReorderWindow = 16
)

var ioBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, copyBufBytes)
		return &buf
	},
}

// SyncOptions configures a single backup or preview scan run.
type SyncOptions struct {
	Source       source.MediaSource
	SourceDirs   []string
	Filter       model.MediaFilter
	TargetDir    string
	DryRun       bool
	DeleteSource bool // When true and Source is a mounted folder, removes source file after verified archive/dedup
}

// Engine orchestrates per-target meta indexing, concurrent hashing, two-tier deduplication, and atomic archive layout.
type Engine struct{}

// NewEngine creates a new stateless sync Engine. All hashes and history live inside <TargetDir>/meta/.
func NewEngine() *Engine {
	return &Engine{}
}

type sizeHashKey struct {
	size   int64
	sha256 string
}

type indexTask struct {
	absPath  string
	relSlash string
	kind     model.MediaKind
	size     int64
	modTime  int64
}

type indexResult struct {
	entry model.TargetIndexEntry
	err   error
}

func clampWorkers(n, minW, maxW int) int {
	if n < minW {
		return minW
	}
	if n > maxW {
		return maxW
	}
	return n
}

// RefreshTargetIndex incrementally indexes files in targetDir concurrently into <targetDir>/meta/index.db.
func (e *Engine) RefreshTargetIndex(ctx context.Context, targetDir string, onProgress func(model.SyncProgress)) error {
	targetRoot, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	store, err := meta.Open(targetRoot)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	_, err = refreshTargetIndexWithStore(ctx, targetRoot, store, onProgress)
	return err
}

func refreshTargetIndexWithStore(
	ctx context.Context,
	targetRoot string,
	store *meta.Store,
	onProgress func(model.SyncProgress),
) (map[string]model.TargetIndexEntry, error) {
	existing, err := store.LoadTargetIndex()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(existing))
	var tasks []indexTask

	err = walkTargetMedia(ctx, targetRoot, func(path, rel string, kind model.MediaKind, info fs.FileInfo) error {
		seen[rel] = struct{}{}
		modUnix := info.ModTime().Unix()
		if prev, ok := existing[rel]; ok && prev.FileSize == info.Size() && prev.ModTime == modUnix && prev.SHA256 != "" {
			return nil
		}
		tasks = append(tasks, indexTask{
			absPath:  path,
			relSlash: rel,
			kind:     kind,
			size:     info.Size(),
			modTime:  modUnix,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(tasks) > 0 {
		results := hashIndexTasksConcurrently(ctx, tasks)
		batch := make([]model.TargetIndexEntry, 0, dbBatchSize)
		indexed := 0

		for res := range results {
			if res.err != nil {
				continue
			}
			existing[res.entry.RelPath] = res.entry
			batch = append(batch, res.entry)
			indexed++

			if len(batch) >= dbBatchSize {
				if err := store.UpsertTargetEntriesBatch(batch); err != nil {
					return nil, err
				}
				batch = batch[:0]
			}
			if onProgress != nil && (indexed%25 == 0 || indexed == len(tasks)) {
				onProgress(model.SyncProgress{
					Phase:       model.PhaseIndexing,
					Current:     indexed,
					Total:       len(tasks),
					CurrentFile: filepath.Base(res.entry.RelPath),
					LogLine:     fmt.Sprintf("正在并行建立目标目录 meta 去重索引 (%d/%d)...", indexed, len(tasks)),
				})
			}
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if len(batch) > 0 {
			if err := store.UpsertTargetEntriesBatch(batch); err != nil {
				return nil, err
			}
		}
	}

	var stale []string
	for rel := range existing {
		if _, ok := seen[rel]; !ok {
			stale = append(stale, rel)
			delete(existing, rel)
		}
	}
	if len(stale) > 0 {
		if err := store.DeleteTargetEntries(stale); err != nil {
			return nil, err
		}
	}
	return existing, nil
}

func scanTargetReadOnly(ctx context.Context, targetRoot string, cached map[string]model.TargetIndexEntry) (map[string]model.TargetIndexEntry, error) {
	var tasks []indexTask
	out := make(map[string]model.TargetIndexEntry, len(cached))
	err := walkTargetMedia(ctx, targetRoot, func(path, rel string, kind model.MediaKind, info fs.FileInfo) error {
		modUnix := info.ModTime().Unix()
		if prev, ok := cached[rel]; ok && prev.FileSize == info.Size() && prev.ModTime == modUnix && prev.SHA256 != "" {
			out[rel] = prev
			return nil
		}
		tasks = append(tasks, indexTask{
			absPath:  path,
			relSlash: rel,
			kind:     kind,
			size:     info.Size(),
			modTime:  modUnix,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return out, nil
	}
	for res := range hashIndexTasksConcurrently(ctx, tasks) {
		if res.err == nil {
			out[res.entry.RelPath] = res.entry
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return out, nil
}

func hashIndexTasksConcurrently(ctx context.Context, tasks []indexTask) <-chan indexResult {
	workers := clampWorkers(runtime.NumCPU(), 2, 8)
	if len(tasks) < workers {
		workers = len(tasks)
	}

	jobs := make(chan indexTask, workers*2)
	results := make(chan indexResult, workers*2)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range jobs {
				if ctx.Err() != nil {
					return
				}
				hashHex, _, err := hashFileSHA256(ctx, task.absPath)
				if err != nil {
					results <- indexResult{err: err}
					continue
				}
				results <- indexResult{
					entry: model.TargetIndexEntry{
						RelPath:   task.relSlash,
						MediaKind: task.kind,
						FileSize:  task.size,
						SHA256:    hashHex,
						ModTime:   task.modTime,
					},
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, task := range tasks {
			select {
			case <-ctx.Done():
				return
			case jobs <- task:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

func walkTargetMedia(ctx context.Context, targetRoot string, fn func(absPath, relSlash string, kind model.MediaKind, info fs.FileInfo) error) error {
	return filepath.WalkDir(targetRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == stagingDirName ||
				(d.Name() == meta.DirName && filepath.Dir(path) == targetRoot) ||
				(path != targetRoot && strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		kind, ok := source.ClassifyMedia(d.Name())
		if !ok {
			return nil
		}

		info, err := d.Info()
		if err != nil || info.Size() <= 0 {
			return nil
		}

		relOS, err := filepath.Rel(targetRoot, path)
		if err != nil {
			return nil
		}
		return fn(path, filepath.ToSlash(relOS), kind, info)
	})
}

type resultKind int

const (
	resCachedSkip resultKind = iota
	resPreDedup
	resDryCandidate
	resPrepared
	resFailed
	resCancelled
)

type workResult struct {
	idx         int
	file        model.MediaFile
	kind        resultKind
	stagePath   string
	shaHex      string
	actualSize  int64
	captureTime time.Time
	existingRel string
	errMsg      string
}

type readSnapshot struct {
	sizeSet    map[int64]bool
	hashMap    map[sizeHashKey]string
	historyMap map[meta.HistoryKey]model.SyncHistoryEntry
}

func (r *readSnapshot) lookupCache(sourceID, remotePath string, size, modUnix int64) (string, bool) {
	hist, ok := r.historyMap[meta.HistoryKey{SourceID: sourceID, RemotePath: remotePath}]
	if !ok || hist.FileSize != size || hist.RemoteModTime != modUnix {
		return "", false
	}
	existingRel, stillInTarget := r.hashMap[sizeHashKey{size: size, sha256: hist.SHA256}]
	return existingRel, stillInTarget
}

// Run executes either a dry-run preview scan or a concurrent incremental backup with deterministic ordered commits.
func (e *Engine) Run(ctx context.Context, opts SyncOptions, onProgress func(model.SyncProgress)) (model.SyncStats, error) {
	var stats model.SyncStats

	if opts.Source == nil {
		return stats, errors.New("未选择来源设备")
	}

	health := CheckTargetHealth(opts.TargetDir)
	if !health.Ready {
		return stats, errors.New(health.StatusMessage)
	}

	targetRoot, err := filepath.Abs(opts.TargetDir)
	if err != nil {
		return stats, err
	}

	if srcDesc := opts.Source.Descriptor(); srcDesc.Kind == model.SourceMounted {
		if err := validateMountedSourceTarget(srcDesc.RootPath, targetRoot, opts.DeleteSource); err != nil {
			return stats, err
		}
	}

	emit := func(p model.SyncProgress) {
		if onProgress != nil {
			onProgress(p)
		}
	}

	// Phase 1: Load or refresh per-target deduplication metadata from <TargetDir>/meta/
	emit(model.SyncProgress{
		Phase:   model.PhaseIndexing,
		LogLine: "正在并行校验目标目录 meta 索引...",
	})

	var store *meta.Store
	var targetEntries map[string]model.TargetIndexEntry
	historyEntries := make(map[meta.HistoryKey]model.SyncHistoryEntry)

	if opts.DryRun {
		var cached map[string]model.TargetIndexEntry
		if meta.Exists(targetRoot) {
			roStore, openErr := meta.Open(targetRoot)
			if openErr != nil {
				return stats, fmt.Errorf("打开目标 meta 索引失败: %w", openErr)
			}
			cached, err = roStore.LoadTargetIndex()
			if err == nil {
				historyEntries, err = roStore.LoadSyncHistory()
			}
			_ = roStore.Close()
			if err != nil {
				return stats, fmt.Errorf("读取目标 meta 索引失败: %w", err)
			}
		}
		targetEntries, err = scanTargetReadOnly(ctx, targetRoot, cached)
		if err != nil {
			return stats, fmt.Errorf("扫描目标目录失败: %w", err)
		}
	} else {
		store, err = meta.Open(targetRoot)
		if err != nil {
			return stats, fmt.Errorf("打开目标 meta 索引失败: %w", err)
		}
		defer func() { _ = store.Close() }()

		targetEntries, err = refreshTargetIndexWithStore(ctx, targetRoot, store, onProgress)
		if err != nil {
			return stats, fmt.Errorf("目标 meta 索引更新失败: %w", err)
		}
		historyEntries, err = store.LoadSyncHistory()
		if err != nil {
			return stats, fmt.Errorf("读取目标同步历史失败: %w", err)
		}
	}

	// Phase 2: Scan source device
	emit(model.SyncProgress{
		Phase:   model.PhaseScanning,
		LogLine: fmt.Sprintf("正在扫描 %s...", opts.Source.Descriptor().Name),
	})
	files, err := opts.Source.Scan(ctx, opts.SourceDirs, opts.Filter)
	if err != nil {
		return stats, fmt.Errorf("扫描设备失败: %w", err)
	}

	stats.TotalScanned = len(files)
	if len(files) == 0 {
		emit(model.SyncProgress{
			Phase:   model.PhaseCompleted,
			Stats:   stats,
			LogLine: "未发现符合条件的照片或视频",
		})
		return stats, nil
	}

	stagingDir := filepath.Join(targetRoot, stagingDirName)
	if !opts.DryRun {
		if err := os.MkdirAll(stagingDir, 0o755); err != nil {
			return stats, fmt.Errorf("创建临时目录失败: %w", err)
		}
		defer cleanEmptyStaging(stagingDir)
	}

	// Build immutable read-only snapshot for concurrent workers and mutable state exclusively owned by committer.
	snap := &readSnapshot{
		sizeSet:    make(map[int64]bool, len(targetEntries)),
		hashMap:    make(map[sizeHashKey]string, len(targetEntries)),
		historyMap: historyEntries,
	}
	liveHashes := make(map[sizeHashKey]string, len(targetEntries))
	reservedPaths := make(map[string]struct{}, len(targetEntries))

	for rel, entry := range targetEntries {
		reservedPaths[rel] = struct{}{}
		snap.sizeSet[entry.FileSize] = true
		k := sizeHashKey{size: entry.FileSize, sha256: entry.SHA256}
		if _, exists := snap.hashMap[k]; !exists {
			snap.hashMap[k] = rel
			liveHashes[k] = rel
		}
	}

	isMountedSource := opts.Source.Descriptor().Kind == model.SourceMounted
	workers := 2
	if isMountedSource {
		workers = clampWorkers(runtime.NumCPU(), 2, 4)
	}
	if opts.DryRun && isMountedSource {
		workers = clampWorkers(runtime.NumCPU(), 2, 6)
	}
	if len(files) < workers {
		workers = len(files)
	}

	type fileJob struct {
		idx  int
		file model.MediaFile
	}

	jobs := make(chan fileJob, workers*2)
	results := make(chan workResult, workers*2)
	var seqCounter atomic.Int64
	var wg sync.WaitGroup

	// Bounded window semaphore avoids buffering excessive staged files ahead of the ordered committer.
	windowSem := make(chan struct{}, maxReorderWindow)

	processJob := func(job fileJob) workResult {
		f := job.file
		if ctx.Err() != nil {
			return workResult{idx: job.idx, file: f, kind: resCancelled}
		}

		// Tier 1: O(1) snapshot sync_history cache check (0-byte transfer)
		if existingRel, hit := snap.lookupCache(f.SourceID, f.RemotePath, f.Size, f.ModTime.Unix()); hit {
			return workResult{idx: job.idx, file: f, kind: resCachedSkip, existingRel: existingRel}
		}

		// Tier 2a: Pre-pull remote SHA-256 check on ADB when same file size exists in target
		if !isMountedSource && snap.sizeSet[f.Size] {
			if remoteHash, ok := opts.Source.RemoteSHA256(ctx, f); ok {
				if existingRel, dup := snap.hashMap[sizeHashKey{size: f.Size, sha256: remoteHash}]; dup {
					return workResult{
						idx:         job.idx,
						file:        f,
						kind:        resPreDedup,
						shaHex:      remoteHash,
						existingRel: existingRel,
					}
				}
			}
		}

		// Mounted source optimization: if file size matches target, hash in-place before writing any staging copy
		// (also runs in DryRun so Preview accurately reflects Tier-2 content duplicates).
		if isMountedSource && snap.sizeSet[f.Size] {
			if shaHex, actualSize, err := hashFileSHA256(ctx, f.RemotePath); err == nil {
				if existingRel, dup := snap.hashMap[sizeHashKey{size: actualSize, sha256: shaHex}]; dup {
					return workResult{
						idx:         job.idx,
						file:        f,
						kind:        resPreDedup,
						shaHex:      shaHex,
						actualSize:  actualSize,
						existingRel: existingRel,
					}
				}
			}
		}

		if opts.DryRun {
			return workResult{idx: job.idx, file: f, kind: resDryCandidate}
		}

		// Pull / single-pass copy+hash to staging inside targetRoot
		partID := seqCounter.Add(1)
		stagePath := filepath.Join(stagingDir, fmt.Sprintf("part_%d_%d_%s", job.idx, partID, sanitizeBaseName(f.Name)))

		var shaHex string
		var actualSize int64
		var pullErr error
		if isMountedSource {
			shaHex, actualSize, pullErr = copyAndHashFile(ctx, f.RemotePath, stagePath)
		} else {
			if err := opts.Source.PullFile(ctx, f, stagePath); err != nil {
				pullErr = err
			} else {
				shaHex, actualSize, pullErr = hashFileSHA256(ctx, stagePath)
			}
		}
		if pullErr != nil {
			_ = os.Remove(stagePath)
			if ctx.Err() != nil {
				return workResult{idx: job.idx, file: f, kind: resCancelled}
			}
			return workResult{
				idx:    job.idx,
				file:   f,
				kind:   resFailed,
				errMsg: fmt.Sprintf("导出失败 [%s]: %v", f.Name, pullErr),
			}
		}

		// Extract EXIF / MP4 capture time concurrently in worker
		captureTime := organizer.ExtractCaptureTime(stagePath, f.Name, f.ModTime)
		return workResult{
			idx:         job.idx,
			file:        f,
			kind:        resPrepared,
			stagePath:   stagePath,
			shaHex:      shaHex,
			actualSize:  actualSize,
			captureTime: captureTime,
		}
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				results <- processJob(job)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for idx, f := range files {
			select {
			case <-ctx.Done():
				return
			case windowSem <- struct{}{}:
			}
			select {
			case <-ctx.Done():
				<-windowSem
				return
			case jobs <- fileJob{idx: idx, file: f}:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	type deleteReq struct {
		remotePath     string
		targetRelSlash string
		expectedSize   int64
	}

	// Single ordered committer loop preserves exact deterministic winner/collision ordering (0..len(files)-1)
	startedAt := time.Now()
	pending := make(map[int]workResult, maxReorderWindow)
	pendingTarget := make([]model.TargetIndexEntry, 0, dbBatchSize)
	pendingHistory := make([]model.SyncHistoryEntry, 0, dbBatchSize)
	pendingDelete := make([]deleteReq, 0, dbBatchSize)
	var flushErr error

	flushBatches := func(force bool) error {
		if flushErr != nil {
			return flushErr
		}
		if store != nil {
			if force || len(pendingTarget) >= dbBatchSize {
				if len(pendingTarget) > 0 {
					if err := store.UpsertTargetEntriesBatch(pendingTarget); err != nil {
						flushErr = fmt.Errorf("写入目标 meta 索引失败: %w", err)
						return flushErr
					}
					pendingTarget = pendingTarget[:0]
				}
			}
			if force || len(pendingHistory) >= dbBatchSize {
				if len(pendingHistory) > 0 {
					if err := store.UpsertSyncHistoryBatch(pendingHistory); err != nil {
						flushErr = fmt.Errorf("写入目标同步历史失败: %w", err)
						return flushErr
					}
					pendingHistory = pendingHistory[:0]
				}
			}
		}
		if force || len(pendingDelete) >= dbBatchSize {
			for _, del := range pendingDelete {
				maybeRemoveMountedSource(opts, targetRoot, del.remotePath, del.targetRelSlash, del.expectedSize)
			}
			pendingDelete = pendingDelete[:0]
		}
		return nil
	}
	defer func() { _ = flushBatches(true) }()

	cleanupUncommitted := func() {
		for _, res := range pending {
			if res.stagePath != "" {
				_ = os.Remove(res.stagePath)
			}
		}
		for res := range results {
			if res.stagePath != "" {
				_ = os.Remove(res.stagePath)
			}
			select {
			case <-windowSem:
			default:
			}
		}
	}

	commitOne := func(res workResult, completed int) {
		f := res.file
		var logLine string

		switch res.kind {
		case resCachedSkip:
			stats.SkippedCached++
			if !opts.DryRun && opts.DeleteSource {
				pendingDelete = append(pendingDelete, deleteReq{
					remotePath:     f.RemotePath,
					targetRelSlash: res.existingRel,
					expectedSize:   f.Size,
				})
			}

		case resPreDedup:
			stats.SkippedDedup++
			if !opts.DryRun {
				pendingHistory = append(pendingHistory, model.SyncHistoryEntry{
					SourceID:      f.SourceID,
					RemotePath:    f.RemotePath,
					FileSize:      f.Size,
					RemoteModTime: f.ModTime.Unix(),
					SHA256:        res.shaHex,
					TargetRelPath: res.existingRel,
					SyncedAt:      time.Now().Unix(),
				})
				if opts.DeleteSource {
					expectedSz := res.actualSize
					if expectedSz <= 0 {
						expectedSz = f.Size
					}
					pendingDelete = append(pendingDelete, deleteReq{
						remotePath:     f.RemotePath,
						targetRelSlash: res.existingRel,
						expectedSize:   expectedSz,
					})
				}
				if isMountedSource {
					logLine = fmt.Sprintf("已去重跳过: %s (与 %s 内容一致)", f.Name, res.existingRel)
				}
			}

		case resDryCandidate:
			stats.NewCandidates++
			stats.BytesTotal += f.Size

		case resFailed:
			stats.Failed++
			logLine = res.errMsg

		case resPrepared:
			k := sizeHashKey{size: res.actualSize, sha256: res.shaHex}
			if existingRel, dup := liveHashes[k]; dup {
				_ = os.Remove(res.stagePath)
				stats.SkippedDedup++
				pendingHistory = append(pendingHistory, model.SyncHistoryEntry{
					SourceID:      f.SourceID,
					RemotePath:    f.RemotePath,
					FileSize:      f.Size,
					RemoteModTime: f.ModTime.Unix(),
					SHA256:        res.shaHex,
					TargetRelPath: existingRel,
					SyncedAt:      time.Now().Unix(),
				})
				if opts.DeleteSource {
					pendingDelete = append(pendingDelete, deleteReq{
						remotePath:     f.RemotePath,
						targetRelSlash: existingRel,
						expectedSize:   res.actualSize,
					})
				}
				logLine = fmt.Sprintf("已去重跳过: %s (与 %s 内容一致)", f.Name, existingRel)
			} else {
				desiredRelOS := organizer.BuildMediaRelPath(f.Kind, res.captureTime, f.Name)
				finalAbs, finalRelOS, err := resolveUniqueTargetPathLocked(targetRoot, desiredRelOS, reservedPaths)
				if err != nil {
					_ = os.Remove(res.stagePath)
					stats.Failed++
					logLine = fmt.Sprintf("创建目标目录失败 [%s]: %v", f.Name, err)
				} else if err := moveStageFile(res.stagePath, finalAbs); err != nil {
					_ = os.Remove(res.stagePath)
					stats.Failed++
					logLine = fmt.Sprintf("写入目标文件失败 [%s]: %v", f.Name, err)
				} else {
					tsToApply := f.ModTime
					if tsToApply.IsZero() {
						tsToApply = res.captureTime
					}
					_ = os.Chtimes(finalAbs, tsToApply, tsToApply)
					finalInfo, statErr := os.Stat(finalAbs)
					finalMTime := tsToApply.Unix()
					if statErr == nil {
						finalMTime = finalInfo.ModTime().Unix()
					}

					finalRelSlash := filepath.ToSlash(finalRelOS)
					reservedPaths[finalRelSlash] = struct{}{}
					liveHashes[k] = finalRelSlash

					pendingTarget = append(pendingTarget, model.TargetIndexEntry{
						RelPath:   finalRelSlash,
						MediaKind: f.Kind,
						FileSize:  res.actualSize,
						SHA256:    res.shaHex,
						ModTime:   finalMTime,
					})
					pendingHistory = append(pendingHistory, model.SyncHistoryEntry{
						SourceID:      f.SourceID,
						RemotePath:    f.RemotePath,
						FileSize:      f.Size,
						RemoteModTime: f.ModTime.Unix(),
						SHA256:        res.shaHex,
						TargetRelPath: finalRelSlash,
						SyncedAt:      time.Now().Unix(),
					})
					if opts.DeleteSource {
						pendingDelete = append(pendingDelete, deleteReq{
							remotePath:     f.RemotePath,
							targetRelSlash: finalRelSlash,
							expectedSize:   res.actualSize,
						})
					}

					stats.Exported++
					stats.BytesCopied += res.actualSize
				}
			}
		}

		if err := flushBatches(false); err != nil && logLine == "" {
			logLine = err.Error()
		}

		elapsed := time.Since(startedAt).Seconds()
		var bps float64
		if elapsed > 0.1 {
			bps = float64(stats.BytesCopied) / elapsed
		}
		emit(model.SyncProgress{
			Phase:       model.PhaseSyncing,
			Current:     completed,
			Total:       len(files),
			CurrentFile: f.Name,
			BytesPerSec: bps,
			Stats:       stats,
			LogLine:     logLine,
		})
	}

	nextIdx := 0
	for res := range results {
		select {
		case <-windowSem:
		default:
		}
		if res.kind == resCancelled || ctx.Err() != nil {
			if res.stagePath != "" {
				_ = os.Remove(res.stagePath)
			}
			cleanupUncommitted()
			_ = flushBatches(true)
			emit(model.SyncProgress{
				Phase:   model.PhaseCancelled,
				Current: nextIdx,
				Total:   len(files),
				Stats:   stats,
				LogLine: "备份已取消",
			})
			return stats, context.Canceled
		}

		pending[res.idx] = res
		for {
			ordered, ok := pending[nextIdx]
			if !ok {
				break
			}
			delete(pending, nextIdx)
			nextIdx++
			commitOne(ordered, nextIdx)
		}
	}

	if err := flushBatches(true); err != nil {
		return stats, err
	}

	if ctx.Err() != nil {
		cleanupUncommitted()
		emit(model.SyncProgress{
			Phase:   model.PhaseCancelled,
			Current: nextIdx,
			Total:   len(files),
			Stats:   stats,
			LogLine: "备份已取消",
		})
		return stats, ctx.Err()
	}

	summary := fmt.Sprintf(
		"备份完成：新增 %d，缓存跳过 %d，内容去重 %d，失败 %d",
		stats.Exported, stats.SkippedCached, stats.SkippedDedup, stats.Failed,
	)
	if opts.DryRun {
		summary = fmt.Sprintf(
			"预览完成：待导出新文件 %d (%s)，缓存跳过 %d，内容去重 %d",
			stats.NewCandidates, FormatBytes(uint64(stats.BytesTotal)), stats.SkippedCached, stats.SkippedDedup,
		)
	}

	emit(model.SyncProgress{
		Phase:   model.PhaseCompleted,
		Current: len(files),
		Total:   len(files),
		Stats:   stats,
		LogLine: summary,
	})
	return stats, nil
}

func copyAndHashFile(ctx context.Context, srcPath, dstPath string) (string, int64, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", 0, fmt.Errorf("open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", 0, fmt.Errorf("create staging file: %w", err)
	}

	hasher := sha256.New()
	bufPtr := ioBufPool.Get().(*[]byte)
	buf := *bufPtr
	defer ioBufPool.Put(bufPtr)

	var total int64
	for {
		if err := ctx.Err(); err != nil {
			_ = dst.Close()
			return "", 0, err
		}
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				_ = dst.Close()
				return "", 0, fmt.Errorf("write staging file: %w", werr)
			}
			_, _ = hasher.Write(buf[:n])
			total += int64(n)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			_ = dst.Close()
			return "", 0, fmt.Errorf("read source file: %w", rerr)
		}
	}

	if err := dst.Sync(); err != nil {
		_ = dst.Close()
		return "", 0, err
	}
	if err := dst.Close(); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hasher.Sum(nil)), total, nil
}

func moveStageFile(stagePath, finalAbs string) error {
	if err := os.Rename(stagePath, finalAbs); err == nil {
		return nil
	}
	in, err := os.Open(stagePath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(finalAbs)
	if err != nil {
		return err
	}

	bufPtr := ioBufPool.Get().(*[]byte)
	buf := *bufPtr
	defer ioBufPool.Put(bufPtr)

	if _, err := io.CopyBuffer(out, in, buf); err != nil {
		_ = out.Close()
		_ = os.Remove(finalAbs)
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		_ = os.Remove(finalAbs)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(finalAbs)
		return err
	}
	return os.Remove(stagePath)
}

func hashFileSHA256(ctx context.Context, path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	bufPtr := ioBufPool.Get().(*[]byte)
	buf := *bufPtr
	defer ioBufPool.Put(bufPtr)

	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return "", 0, err
		}
		n, rerr := f.Read(buf)
		if n > 0 {
			_, _ = h.Write(buf[:n])
			total += int64(n)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return "", 0, rerr
		}
	}
	return hex.EncodeToString(h.Sum(nil)), total, nil
}

func resolveUniqueTargetPathLocked(targetRoot, desiredRelOS string, reserved map[string]struct{}) (absPath string, relOS string, err error) {
	dirRel := filepath.Dir(desiredRelOS)
	base := filepath.Base(desiredRelOS)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	parentAbs := filepath.Join(targetRoot, dirRel)
	if err := os.MkdirAll(parentAbs, 0o755); err != nil {
		return "", "", err
	}

	candidateRel := desiredRelOS
	candidateAbs := filepath.Join(targetRoot, candidateRel)
	for suffix := 1; ; suffix++ {
		relSlash := filepath.ToSlash(candidateRel)
		if _, taken := reserved[relSlash]; !taken {
			_, statErr := os.Stat(candidateAbs)
			if os.IsNotExist(statErr) {
				return candidateAbs, candidateRel, nil
			}
			if statErr != nil {
				return "", "", fmt.Errorf("检查目标路径失败: %w", statErr)
			}
		}
		candidateName := fmt.Sprintf("%s_%d%s", stem, suffix, ext)
		candidateRel = filepath.Join(dirRel, candidateName)
		candidateAbs = filepath.Join(targetRoot, candidateRel)
	}
}

func sanitizeBaseName(name string) string {
	base := filepath.Base(name)
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', 0:
			return '_'
		default:
			return r
		}
	}, base)
}

func cleanEmptyStaging(stagingDir string) {
	entries, err := os.ReadDir(stagingDir)
	if err == nil && len(entries) == 0 {
		_ = os.Remove(stagingDir)
	}
}

func validateMountedSourceTarget(sourceDir, targetRoot string, deleteSource bool) error {
	if sourceDir == "" {
		return nil
	}
	srcAbs, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil
	}
	srcClean := filepath.Clean(srcAbs)
	dstClean := filepath.Clean(targetRoot)

	if srcClean == dstClean {
		return errors.New("目标目录不能与来源目录相同，请选择单独的整理目标目录")
	}
	if isSubpath(dstClean, srcClean) {
		return errors.New("目标目录不能位于来源目录内部")
	}
	if deleteSource && isSubpath(srcClean, dstClean) {
		return errors.New("移动整理模式下，来源目录不能位于目标目录内部")
	}
	return nil
}

func isSubpath(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func maybeRemoveMountedSource(opts SyncOptions, targetRoot, remotePath, targetRelSlash string, expectedSize int64) {
	if !opts.DeleteSource || opts.DryRun || opts.Source.Descriptor().Kind != model.SourceMounted {
		return
	}
	if targetRelSlash == "" || remotePath == "" {
		return
	}
	targetAbs := filepath.Clean(filepath.Join(targetRoot, filepath.FromSlash(targetRelSlash)))
	srcAbs := filepath.Clean(remotePath)
	if targetAbs == srcAbs {
		return
	}
	if info, err := os.Stat(targetAbs); err == nil && !info.IsDir() && (expectedSize <= 0 || info.Size() == expectedSize) {
		_ = os.Remove(srcAbs)
	}
}
