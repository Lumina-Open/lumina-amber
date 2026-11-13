package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"lumina-photo-sync/internal/model"
)

func TestParseADBDevices(t *testing.T) {
	sample := `List of devices attached
R58M12345AB            device usb:1-2 product:panther model:Pixel_7 device:panther transport_id:1
192.168.1.88:5555      device product:kalama model:Galaxy_S23 transport_id:2
ABCD9999               unauthorized usb:1-3 transport_id:3
`
	devs := ParseADBDevices(sample)
	if len(devs) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devs))
	}
	if devs[0].Model != "Pixel 7" || devs[0].IsWiFi || devs[0].ToDescriptor().State != StateReady {
		t.Fatalf("unexpected device 0: %+v", devs[0])
	}
	if !devs[1].IsWiFi || devs[1].Model != "Galaxy S23" {
		t.Fatalf("unexpected device 1: %+v", devs[1])
	}
	if devs[2].ToDescriptor().State != StateUnauthorized {
		t.Fatalf("expected device 2 unauthorized, got %+v", devs[2].ToDescriptor())
	}
}

func TestParseADBScanOutput(t *testing.T) {
	sample := "" +
		"2048000|1731412527|/sdcard/DCIM/Camera/IMG 20261112.jpg\n" +
		"1024|1731412527|/sdcard/DCIM/.thumbnails/thumb.jpg\n" +
		"4096000|1731412530|/sdcard/Movies/VID_001.mp4\n" +
		"512|1731412530|/sdcard/DCIM/Camera/.trashed-123-IMG.jpg\n" +
		"0|1731412530|/sdcard/DCIM/Camera/empty.jpg\n"

	files := ParseADBScanOutput("adb:test", sample, model.MediaFilter{
		IncludePhotos: true,
		IncludeVideos: true,
	})
	if len(files) != 2 {
		t.Fatalf("expected 2 valid media files, got %d: %+v", len(files), files)
	}
	if files[0].Name != "IMG 20261112.jpg" || files[0].Kind != model.MediaPhoto {
		t.Fatalf("unexpected file 0: %+v", files[0])
	}
	if files[1].Name != "VID_001.mp4" || files[1].Kind != model.MediaVideo {
		t.Fatalf("unexpected file 1: %+v", files[1])
	}

	photosOnly := ParseADBScanOutput("adb:test", sample, model.MediaFilter{
		IncludePhotos: true,
		IncludeVideos: false,
	})
	if len(photosOnly) != 1 {
		t.Fatalf("expected 1 photo when videos excluded, got %d", len(photosOnly))
	}
}

func TestMountedSourceScan(t *testing.T) {
	tmp := t.TempDir()
	dcim := filepath.Join(tmp, "DCIM", "Camera")
	thumbs := filepath.Join(tmp, "DCIM", ".thumbnails")
	_ = os.MkdirAll(dcim, 0o755)
	_ = os.MkdirAll(thumbs, 0o755)

	_ = os.WriteFile(filepath.Join(dcim, "IMG_001.jpg"), []byte("jpegdata"), 0o644)
	_ = os.WriteFile(filepath.Join(dcim, "VID_001.mp4"), []byte("mp4data"), 0o644)
	_ = os.WriteFile(filepath.Join(thumbs, "thumb.jpg"), []byte("thumb"), 0o644)
	_ = os.WriteFile(filepath.Join(dcim, ".trashed-1-IMG.jpg"), []byte("trash"), 0o644)

	ms := NewMountedSource(NewMountedDescriptor(tmp))
	files, err := ms.Scan(context.Background(), model.DefaultConfig().SourceDirs, model.MediaFilter{
		IncludePhotos: true,
		IncludeVideos: true,
	})
	if err != nil {
		t.Fatalf("scan mounted: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(files), files)
	}
}
