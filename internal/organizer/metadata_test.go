package organizer

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lumina-amber/internal/model"
)

func TestParseTimeFromFilename(t *testing.T) {
	cases := []struct {
		name     string
		wantYear int
		wantMon  time.Month
		wantDay  int
		wantOK   bool
	}{
		{"IMG_20261112_195527.jpg", 2026, time.November, 12, true},
		{"PXL_20250401_083015123.jpg", 2025, time.April, 1, true},
		{"VID_20240229_235959.mp4", 2024, time.February, 29, true},
		{"Screenshot_2023-08-15-14-22-10.png", 2023, time.August, 15, true},
		{"mmexport1700000000000.jpg", 2023, time.November, 15, true},
		{"IMG_20260231_120000.jpg", 0, 0, 0, false},
		{"random_wallpaper.png", 0, 0, 0, false},
	}

	for _, tc := range cases {
		got, ok := ParseTimeFromFilename(tc.name)
		if ok != tc.wantOK {
			t.Fatalf("ParseTimeFromFilename(%q) ok=%v, want %v", tc.name, ok, tc.wantOK)
		}
		if !ok {
			continue
		}
		// For unix millis, timezone may shift day +/- 1 depending on host TZ, so only check year/month for non-unix cases
		if tc.name[:8] != "mmexport" {
			if got.Year() != tc.wantYear || got.Month() != tc.wantMon || got.Day() != tc.wantDay {
				t.Fatalf("ParseTimeFromFilename(%q) = %v, want %d-%02d-%02d", tc.name, got, tc.wantYear, tc.wantMon, tc.wantDay)
			}
		} else if got.Year() != 2023 {
			t.Fatalf("ParseTimeFromFilename(%q) year = %d, want 2023", tc.name, got.Year())
		}
	}
}

func TestParseMP4CreationTime(t *testing.T) {
	// Construct a minimal synthetic MP4 with ftyp + moov -> mvhd (version 0).
	targetUTC := time.Date(2025, 7, 19, 12, 30, 0, 0, time.UTC)
	macSecs := uint32(targetUTC.Sub(macEpoch).Seconds())

	var mvhdPayload [100]byte
	mvhdPayload[0] = 0 // version 0
	binary.BigEndian.PutUint32(mvhdPayload[4:8], macSecs)

	var mvhd bytes.Buffer
	_ = binary.Write(&mvhd, binary.BigEndian, uint32(8+len(mvhdPayload)))
	mvhd.WriteString("mvhd")
	mvhd.Write(mvhdPayload[:])

	var moov bytes.Buffer
	_ = binary.Write(&moov, binary.BigEndian, uint32(8+mvhd.Len()))
	moov.WriteString("moov")
	moov.Write(mvhd.Bytes())

	var ftyp bytes.Buffer
	_ = binary.Write(&ftyp, binary.BigEndian, uint32(16))
	ftyp.WriteString("ftypisom")
	ftyp.Write([]byte{0, 0, 0, 0})

	tmp := t.TempDir()
	mp4Path := filepath.Join(tmp, "sample.mp4")
	var full bytes.Buffer
	full.Write(ftyp.Bytes())
	full.Write(moov.Bytes())
	if err := os.WriteFile(mp4Path, full.Bytes(), 0o644); err != nil {
		t.Fatalf("write sample mp4: %v", err)
	}

	got, ok := parseMP4CreationTime(mp4Path)
	if !ok {
		t.Fatalf("expected parseMP4CreationTime to succeed")
	}
	if !got.UTC().Equal(targetUTC) {
		t.Fatalf("got %v, want %v", got.UTC(), targetUTC)
	}
}

func TestBuildMediaRelPath(t *testing.T) {
	ts := time.Date(2026, time.November, 5, 10, 0, 0, 0, time.Local)
	gotPic := BuildMediaRelPath(model.MediaPhoto, ts, "/sdcard/DCIM/Camera/IMG_20261105.jpg")
	wantPic := filepath.Join("picture", "2026", "2026-11", "IMG_20261105.jpg")
	if gotPic != wantPic {
		t.Fatalf("photo path got %q, want %q", gotPic, wantPic)
	}

	gotVid := BuildMediaRelPath(model.MediaVideo, ts, "/sdcard/Movies/VID_20261105.mp4")
	wantVid := filepath.Join("video", "2026", "2026-11", "VID_20261105.mp4")
	if gotVid != wantVid {
		t.Fatalf("video path got %q, want %q", gotVid, wantVid)
	}
}
