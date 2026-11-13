//go:build linux || darwin

package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// TargetHealth describes the availability and free space of a destination directory.
type TargetHealth struct {
	Ready         bool
	FreeBytes     uint64
	TotalBytes    uint64
	StatusMessage string
}

// CheckTargetHealth verifies that targetDir exists, is writable, and resides on a mounted volume.
func CheckTargetHealth(targetDir string) TargetHealth {
	return checkTargetHealth(targetDir, true)
}

// CheckTargetQuick verifies targetDir availability and free space without writing temp files,
// suitable for periodic background UI refreshes so external HDDs can spin down normally.
func CheckTargetQuick(targetDir string) TargetHealth {
	return checkTargetHealth(targetDir, false)
}

func checkTargetHealth(targetDir string, probeWrite bool) TargetHealth {
	if strings.TrimSpace(targetDir) == "" {
		return TargetHealth{
			Ready:         false,
			StatusMessage: "尚未选择目标目录",
		}
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return TargetHealth{
			Ready:         false,
			StatusMessage: fmt.Sprintf("路径无效: %v", err),
		}
	}

	// Guard against writing to an unplugged external drive's mount path.
	if missingMount, ok := unpluggedMountRoot(absDir); ok {
		return TargetHealth{
			Ready:         false,
			StatusMessage: fmt.Sprintf("外接存储未挂载: %s", missingMount),
		}
	}

	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return TargetHealth{
				Ready:         false,
				StatusMessage: "目标目录不存在或硬盘未连接",
			}
		}
		return TargetHealth{
			Ready:         false,
			StatusMessage: fmt.Sprintf("无法访问目标目录: %v", err),
		}
	}
	if !info.IsDir() {
		return TargetHealth{
			Ready:         false,
			StatusMessage: "目标路径不是文件夹",
		}
	}

	if probeWrite {
		// Lightweight writability check via creating and removing a temp file.
		probe, err := os.CreateTemp(absDir, ".lumina-write-probe-*")
		if err != nil {
			return TargetHealth{
				Ready:         false,
				StatusMessage: fmt.Sprintf("目标目录不可写: %v", err),
			}
		}
		probeName := probe.Name()
		_ = probe.Close()
		_ = os.Remove(probeName)
	}

	var fsStat syscall.Statfs_t
	var freeBytes, totalBytes uint64
	if err := syscall.Statfs(absDir, &fsStat); err == nil {
		bsize := uint64(fsStat.Bsize)
		freeBytes = uint64(fsStat.Bavail) * bsize
		totalBytes = uint64(fsStat.Blocks) * bsize
	}

	msg := "已就绪"
	if freeBytes > 0 {
		msg = fmt.Sprintf("已就绪 · 剩余 %s", FormatBytes(freeBytes))
	}
	return TargetHealth{
		Ready:         true,
		FreeBytes:     freeBytes,
		TotalBytes:    totalBytes,
		StatusMessage: msg,
	}
}

// unpluggedMountRoot detects paths pointing inside common Linux/macOS external drive roots
// whose volume mount directory itself is missing.
func unpluggedMountRoot(absDir string) (string, bool) {
	clean := filepath.ToSlash(absDir)
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")

	if runtime.GOOS == "darwin" {
		// /Volumes/<DriveName>/...
		if len(parts) >= 2 && parts[0] == "Volumes" && parts[1] != "" {
			volRoot := filepath.Join("/Volumes", parts[1])
			if _, err := os.Stat(volRoot); os.IsNotExist(err) {
				return volRoot, true
			}
		}
	} else if runtime.GOOS == "linux" {
		// /media/<user>/<DriveName> or /run/media/<user>/<DriveName>
		if len(parts) >= 3 && parts[0] == "media" && parts[2] != "" {
			volRoot := filepath.Join("/", parts[0], parts[1], parts[2])
			if _, err := os.Stat(volRoot); os.IsNotExist(err) {
				return volRoot, true
			}
		}
		if len(parts) >= 4 && parts[0] == "run" && parts[1] == "media" && parts[3] != "" {
			volRoot := filepath.Join("/", parts[0], parts[1], parts[2], parts[3])
			if _, err := os.Stat(volRoot); os.IsNotExist(err) {
				return volRoot, true
			}
		}
	}
	return "", false
}

// FormatBytes formats a byte count into human-readable units.
func FormatBytes(b uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.2f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
