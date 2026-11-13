package source

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"lumina-amber/internal/model"
)

// DiscoverMountedSources detects Linux GVFS MTP mounts (and optional camera/SD mounts).
func DiscoverMountedSources() []SourceDescriptor {
	var sources []SourceDescriptor
	seen := make(map[string]struct{})

	if runtime.GOOS == "linux" {
		uid := fmt.Sprintf("%d", os.Getuid())
		gvfsRoot := filepath.Join("/run/user", uid, "gvfs")
		if entries, err := os.ReadDir(gvfsRoot); err == nil {
			for _, ent := range entries {
				name := ent.Name()
				if strings.HasPrefix(name, "mtp:") || strings.HasPrefix(name, "gphoto2:") {
					full := filepath.Join(gvfsRoot, name)
					if _, ok := seen[full]; ok {
						continue
					}
					seen[full] = struct{}{}
					sources = append(sources, SourceDescriptor{
						ID:            "dir:" + full,
						Name:          formatGVFSName(name),
						Kind:          model.SourceMounted,
						State:         StateReady,
						StatusMessage: "GVFS MTP 已挂载",
						RootPath:      full,
					})
				}
			}
		}

		// Also surface removable DCIM mounts under /media/$USER or /run/media/$USER if they contain DCIM.
		if u, err := user.Current(); err == nil {
			for _, base := range []string{filepath.Join("/media", u.Username), filepath.Join("/run/media", u.Username)} {
				if entries, err := os.ReadDir(base); err == nil {
					for _, ent := range entries {
						if !ent.IsDir() {
							continue
						}
						vol := filepath.Join(base, ent.Name())
						if hasDCIMSubdir(vol) {
							if _, ok := seen[vol]; ok {
								continue
							}
							seen[vol] = struct{}{}
							sources = append(sources, SourceDescriptor{
								ID:            "dir:" + vol,
								Name:          ent.Name() + " (存储卡/设备)",
								Kind:          model.SourceMounted,
								State:         StateReady,
								StatusMessage: "检测到 DCIM 目录",
								RootPath:      vol,
							})
						}
					}
				}
			}
		}
	} else if runtime.GOOS == "darwin" {
		if entries, err := os.ReadDir("/Volumes"); err == nil {
			for _, ent := range entries {
				if !ent.IsDir() {
					continue
				}
				vol := filepath.Join("/Volumes", ent.Name())
				if hasDCIMSubdir(vol) {
					if _, ok := seen[vol]; ok {
						continue
					}
					seen[vol] = struct{}{}
					sources = append(sources, SourceDescriptor{
						ID:            "dir:" + vol,
						Name:          ent.Name() + " (存储卡/相机)",
						Kind:          model.SourceMounted,
						State:         StateReady,
						StatusMessage: "检测到 DCIM 目录",
						RootPath:      vol,
					})
				}
			}
		}
	}

	return sources
}

func formatGVFSName(raw string) string {
	clean := strings.TrimPrefix(raw, "mtp:host=")
	clean = strings.TrimPrefix(clean, "gphoto2:host=")
	clean = strings.ReplaceAll(clean, "_", " ")
	if len(clean) > 36 {
		clean = clean[:33] + "..."
	}
	return clean + " (MTP)"
}

func hasDCIMSubdir(root string) bool {
	info, err := os.Stat(filepath.Join(root, "DCIM"))
	return err == nil && info.IsDir()
}

func hasMetaDB(metaDir string) bool {
	info, err := os.Stat(filepath.Join(metaDir, "index.db"))
	return err == nil && !info.IsDir()
}

// NewMountedDescriptor creates a SourceDescriptor for an arbitrary user-chosen directory.
func NewMountedDescriptor(rootPath string) SourceDescriptor {
	return newDirDescriptor(rootPath, "目录", false)
}

// NewOrganizeFolderDescriptor creates a SourceDescriptor that recursively scans all files in rootPath.
func NewOrganizeFolderDescriptor(rootPath string) SourceDescriptor {
	return newDirDescriptor(rootPath, "待整理目录", true)
}

func newDirDescriptor(rootPath, suffix string, wholeRoot bool) SourceDescriptor {
	if rootPath == "" {
		return SourceDescriptor{
			ID:            "dir:",
			Name:          "尚未选择待整理目录",
			Kind:          model.SourceMounted,
			State:         StateUnavailable,
			StatusMessage: "请选择包含照片或视频的文件夹",
			ScanWholeRoot: wholeRoot,
		}
	}
	abs, err := filepath.Abs(rootPath)
	if err == nil {
		rootPath = abs
	}
	info, err := os.Stat(rootPath)
	if err != nil || !info.IsDir() {
		return SourceDescriptor{
			ID:            "dir:" + rootPath,
			Name:          filepath.Base(rootPath) + " (" + suffix + ")",
			Kind:          model.SourceMounted,
			State:         StateUnavailable,
			StatusMessage: "目录不存在或未挂载",
			RootPath:      rootPath,
			ScanWholeRoot: wholeRoot,
		}
	}
	return SourceDescriptor{
		ID:            "dir:" + rootPath,
		Name:          filepath.Base(rootPath) + " (" + suffix + ")",
		Kind:          model.SourceMounted,
		State:         StateReady,
		StatusMessage: "目录已就绪",
		RootPath:      rootPath,
		ScanWholeRoot: wholeRoot,
	}
}

// MountedSource implements MediaSource for local directories, SD cards, and GVFS MTP mounts.
type MountedSource struct {
	desc SourceDescriptor
}

// NewMountedSource creates a MountedSource for the given descriptor.
func NewMountedSource(desc SourceDescriptor) *MountedSource {
	return &MountedSource{desc: desc}
}

func (s *MountedSource) Descriptor() SourceDescriptor {
	return s.desc
}

// Scan walks the mounted root directory (resolving standard Android subdirs when present).
func (s *MountedSource) Scan(ctx context.Context, dirs []string, filter model.MediaFilter) ([]model.MediaFile, error) {
	root := s.desc.RootPath
	if root == "" {
		root = strings.TrimPrefix(s.desc.ID, "dir:")
	}

	scanRoots := []string{root}
	if !s.desc.ScanWholeRoot {
		scanRoots = resolveMountedRoots(root, dirs)
	}
	var files []model.MediaFile
	seen := make(map[string]struct{})

	for _, sr := range scanRoots {
		err := filepath.WalkDir(sr, func(path string, d fs.DirEntry, walkErr error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if walkErr != nil {
				return nil // Skip unreadable entries gracefully
			}
			relPath, relErr := filepath.Rel(sr, path)
			if relErr != nil {
				relPath = d.Name()
			}
			if d.IsDir() {
				if d.Name() == ".lumina-sync-tmp" ||
					(d.Name() == "meta" && filepath.Dir(path) == sr && hasMetaDB(path)) ||
					(path != sr && ShouldIgnorePath(relPath)) {
					return filepath.SkipDir
				}
				return nil
			}
			kind, ok := MatchesFilter(relPath, filter)
			if !ok {
				return nil
			}
			info, err := d.Info()
			if err != nil || info.Size() <= 0 {
				return nil
			}
			if _, dup := seen[path]; dup {
				return nil
			}
			seen[path] = struct{}{}

			files = append(files, model.MediaFile{
				SourceID:   s.desc.ID,
				RemotePath: path,
				Name:       d.Name(),
				Size:       info.Size(),
				ModTime:    info.ModTime(),
				Kind:       kind,
			})
			return nil
		})
		if err != nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return files, nil
}

// resolveMountedRoots maps Android config dirs like /sdcard/DCIM onto GVFS MTP or SD roots when applicable,
// falling back to scanning the chosen root directly.
func resolveMountedRoots(root string, configDirs []string) []string {
	// GVFS MTP mounts often have "Internal shared storage/DCIM" or "内部存储设备/DCIM"
	var candidates []string
	storageRoots := []string{root}
	if entries, err := os.ReadDir(root); err == nil {
		for _, ent := range entries {
			if ent.IsDir() {
				sub := filepath.Join(root, ent.Name())
				if hasDCIMSubdir(sub) {
					storageRoots = append(storageRoots, sub)
				}
			}
		}
	}

	for _, sr := range storageRoots {
		for _, cfgDir := range configDirs {
			rel := strings.TrimPrefix(cfgDir, "/sdcard/")
			rel = strings.TrimPrefix(rel, "/storage/emulated/0/")
			rel = strings.TrimPrefix(rel, "/")
			if rel == "" {
				continue
			}
			joined := filepath.Join(sr, filepath.FromSlash(rel))
			if info, err := os.Stat(joined); err == nil && info.IsDir() {
				candidates = append(candidates, joined)
			}
		}
	}

	if len(candidates) > 0 {
		return candidates
	}
	return []string{root}
}

// PullFile streams a file from the mounted source directory into localDest.
func (s *MountedSource) PullFile(ctx context.Context, file model.MediaFile, localDest string) error {
	src, err := os.Open(file.RemotePath)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(localDest)
	if err != nil {
		return fmt.Errorf("create staging file: %w", err)
	}
	defer func() {
		_ = dst.Close()
	}()

	buf := make([]byte, 1024*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return fmt.Errorf("write staging file: %w", werr)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fmt.Errorf("read source file: %w", rerr)
		}
	}
	return dst.Sync()
}

// RemoteSHA256 is not used for mounted filesystems since PullFile + local hashing is already direct I/O.
func (s *MountedSource) RemoteSHA256(_ context.Context, _ model.MediaFile) (string, bool) {
	return "", false
}
