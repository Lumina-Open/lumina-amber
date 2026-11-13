package organizer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"

	"lumina-amber/internal/model"
)

var (
	// Matches IMG_20261112_195527, PXL_20261112_195527123, VID_20261112_195527, 20261112_195527
	reCompactDateTime = regexp.MustCompile(`(?:^|[^0-9])(199[0-9]|20[0-9]{2})(0[1-9]|1[0-2])(0[1-9]|[12][0-9]|3[01])[_-]?([01][0-9]|2[0-3])([0-5][0-9])([0-5][0-9])(?:[0-9]{0,3}(?:[^0-9]|$))`)
	// Matches 2026-11-12, Screenshot_2026-11-12-19-55-27, 2026.11.12
	reDelimitedDate = regexp.MustCompile(`(?:^|[^0-9])(199[0-9]|20[0-9]{2})[-_.](0[1-9]|1[0-2])[-_.](0[1-9]|[12][0-9]|3[01])(?:[^0-9]|$)`)
	// Matches WeChat/social exports like mmexport1731412527000.jpg or wx_camera_1731412527000.mp4
	reUnixMillis = regexp.MustCompile(`(?:mmexport|wx_camera_|microMsg\.)(1[5-9][0-9]{11}|2[0-9]{12})`)
)

// macEpoch is 1904-01-01 00:00:00 UTC used by ISO BMFF (MP4/MOV) mvhd boxes.
var macEpoch = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)

// ExtractCaptureTime determines the best capture timestamp for a local media file.
// Priority: EXIF DateTimeOriginal -> MP4/MOV mvhd atom -> filename timestamp -> fallbackModTime.
func ExtractCaptureTime(localPath string, originalName string, fallbackModTime time.Time) time.Time {
	ext := strings.ToLower(filepath.Ext(originalName))
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(localPath))
	}

	if isEXIFCandidate(ext) {
		if t, ok := parseEXIFTime(localPath); ok {
			return t
		}
	}

	if isISOBMFFCandidate(ext) {
		if t, ok := parseMP4CreationTime(localPath); ok {
			return t
		}
	}

	if t, ok := ParseTimeFromFilename(originalName); ok {
		return t
	}

	if !fallbackModTime.IsZero() && isSaneMediaYear(fallbackModTime.Year()) {
		return fallbackModTime
	}
	return time.Now()
}

// BuildMediaRelPath returns "<picture|video>/<YYYY>/<YYYY-MM>/<filename>".
func BuildMediaRelPath(kind model.MediaKind, captureTime time.Time, fileName string) string {
	local := captureTime.In(time.Local)
	category := "picture"
	if kind == model.MediaVideo {
		category = "video"
	}
	year := fmt.Sprintf("%04d", local.Year())
	yearMonth := fmt.Sprintf("%04d-%02d", local.Year(), int(local.Month()))
	base := filepath.Base(fileName)
	return filepath.Join(category, year, yearMonth, base)
}

func isEXIFCandidate(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".tif", ".tiff", ".dng", ".cr2", ".cr3", ".nef", ".arw", ".orf", ".rw2":
		return true
	default:
		return false
	}
}

func isISOBMFFCandidate(ext string) bool {
	switch ext {
	case ".mp4", ".mov", ".m4v", ".3gp":
		return true
	default:
		return false
	}
}

func isSaneMediaYear(year int) bool {
	return year >= 1995 && year <= 2100
}

func parseEXIFTime(path string) (time.Time, bool) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return time.Time{}, false
	}

	t, err := x.DateTime()
	if err != nil || t.IsZero() || !isSaneMediaYear(t.Year()) {
		return time.Time{}, false
	}
	return t, true
}

// parseMP4CreationTime walks top-level ISO BMFF atoms to locate moov -> mvhd and extract creation_time.
func parseMP4CreationTime(path string) (time.Time, bool) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return time.Time{}, false
	}
	fileSize := info.Size()

	moovStart, moovEnd, ok := findAtom(f, 0, fileSize, "moov")
	if !ok {
		return time.Time{}, false
	}

	mvhdStart, mvhdEnd, ok := findAtom(f, moovStart, moovEnd, "mvhd")
	if !ok || mvhdEnd-mvhdStart < 12 {
		return time.Time{}, false
	}

	var buf [20]byte
	n, err := f.ReadAt(buf[:], mvhdStart)
	if err != nil && !errors.Is(err, io.EOF) {
		return time.Time{}, false
	}
	if n < 8 {
		return time.Time{}, false
	}

	version := buf[0]
	var secsSinceMacEpoch uint64
	switch version {
	case 0:
		secsSinceMacEpoch = uint64(binary.BigEndian.Uint32(buf[4:8]))
	case 1:
		if n < 12 {
			return time.Time{}, false
		}
		secsSinceMacEpoch = binary.BigEndian.Uint64(buf[4:12])
	default:
		return time.Time{}, false
	}

	if secsSinceMacEpoch == 0 {
		return time.Time{}, false
	}

	// Guard against overflow when converting uint64 seconds to time.Duration.
	if secsSinceMacEpoch > 1<<52 {
		return time.Time{}, false
	}

	t := macEpoch.Add(time.Duration(secsSinceMacEpoch) * time.Second).In(time.Local)
	if !isSaneMediaYear(t.Year()) {
		return time.Time{}, false
	}
	return t, true
}

func findAtom(r io.ReaderAt, start, end int64, target string) (payloadStart, atomEnd int64, found bool) {
	offset := start
	var header [16]byte

	for offset+8 <= end {
		if _, err := r.ReadAt(header[:8], offset); err != nil {
			return 0, 0, false
		}
		size32 := binary.BigEndian.Uint32(header[0:4])
		atomType := string(header[4:8])

		var atomSize int64
		var hdrSize int64 = 8

		switch size32 {
		case 0:
			atomSize = end - offset
		case 1:
			if offset+16 > end {
				return 0, 0, false
			}
			if _, err := r.ReadAt(header[8:16], offset+8); err != nil {
				return 0, 0, false
			}
			u64 := binary.BigEndian.Uint64(header[8:16])
			if u64 < 16 || u64 > uint64(end-offset) {
				return 0, 0, false
			}
			atomSize = int64(u64)
			hdrSize = 16
		default:
			atomSize = int64(size32)
		}

		if atomSize < hdrSize || offset+atomSize > end {
			return 0, 0, false
		}

		if atomType == target {
			return offset + hdrSize, offset + atomSize, true
		}

		offset += atomSize
	}
	return 0, 0, false
}

// ParseTimeFromFilename extracts a plausible date/time from common Android camera & app filenames.
func ParseTimeFromFilename(name string) (time.Time, bool) {
	base := filepath.Base(name)

	if m := reUnixMillis.FindStringSubmatch(base); len(m) == 2 {
		if ms, err := strconv.ParseInt(m[1], 10, 64); err == nil {
			t := time.UnixMilli(ms).In(time.Local)
			if isSaneMediaYear(t.Year()) {
				return t, true
			}
		}
	}

	if m := reCompactDateTime.FindStringSubmatch(base); len(m) == 7 {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		h, _ := strconv.Atoi(m[4])
		mi, _ := strconv.Atoi(m[5])
		s, _ := strconv.Atoi(m[6])
		if mo >= 1 && mo <= 12 && d >= 1 && d <= 31 && h >= 0 && h <= 23 && mi >= 0 && mi <= 59 && s >= 0 && s <= 59 {
			t := time.Date(y, time.Month(mo), d, h, mi, s, 0, time.Local)
			if t.Year() == y && int(t.Month()) == mo && t.Day() == d && isSaneMediaYear(t.Year()) {
				return t, true
			}
		}
	}

	if m := reDelimitedDate.FindStringSubmatch(base); len(m) == 4 {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		if mo >= 1 && mo <= 12 && d >= 1 && d <= 31 {
			t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.Local)
			if t.Year() == y && int(t.Month()) == mo && t.Day() == d && isSaneMediaYear(t.Year()) {
				return t, true
			}
		}
	}

	return time.Time{}, false
}
