//go:build linux || darwin

package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"lumina-amber/internal/i18n"
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
	s := i18n.S()
	if strings.TrimSpace(targetDir) == "" {
		return TargetHealth{
			Ready:         false,
			StatusMessage: s.VolNotSelected,
		}
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return TargetHealth{
			Ready:         false,
			StatusMessage: fmt.Sprintf(s.VolInvalidPathFmt, err),
		}
	}

	// Guard against writing to an unplugged external drive's mount path.
	if missingMount, ok := unpluggedMountRoot(absDir); ok {
		return TargetHealth{
			Ready:         false,
			StatusMessage: fmt.Sprintf(s.VolExtUnmountedFmt, missingMount),
		}
	}

	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return TargetHealth{
				Ready:         false,
				StatusMessage: s.VolMissingOrUnplugged,
			}
		}
		return TargetHealth{
			Ready:         false,
			StatusMessage: fmt.Sprintf(s.VolNoAccessFmt, err),
		}
	}
	if !info.IsDir() {
		return TargetHealth{
			Ready:         false,
			StatusMessage: s.VolNotDir,
		}
	}

	if probeWrite {
		// Lightweight writability check via creating and removing a temp file.
		probe, err := os.CreateTemp(absDir, ".lumina-write-probe-*")
		if err != nil {
			return TargetHealth{
				Ready:         false,
				StatusMessage: fmt.Sprintf(s.VolNotWritableFmt, err),
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

	msg := s.VolReady
	if freeBytes > 0 {
		msg = fmt.Sprintf(s.VolReadyFreeFmt, FormatBytes(freeBytes))
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
