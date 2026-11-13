package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"lumina-amber/internal/db"
	"lumina-amber/internal/model"
	"lumina-amber/internal/source"
	syncengine "lumina-amber/internal/sync"
)

func TestRenderHomeViews(t *testing.T) {
	tmp := t.TempDir()
	database, err := db.Open(filepath.Join(tmp, "ui.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	targetDir := filepath.Join(tmp, "MyPassport", "Photos")
	_ = os.MkdirAll(targetDir, 0o755)

	testApp := test.NewApp()
	defer testApp.Quit()

	win := testApp.NewWindow("Lumina Amber")
	win.Resize(fyne.NewSize(900, 620))

	orgDir := filepath.Join(tmp, "UnsortedPhotos")
	_ = os.MkdirAll(orgDir, 0o755)

	cfg := model.DefaultConfig()
	cfg.TargetDir = targetDir
	cfg.OrganizeSourceDir = orgDir
	cfg.AddRecentTarget(targetDir)

	psa := &AmberApp{
		FyneApp: testApp,
		Window:  win,
		DB:      database,
		Engine:  syncengine.NewEngine(),
		Theme:   NewLuminaTheme(ThemeModeLight),
		Config:  cfg,
		sources: []source.SourceDescriptor{
			{
				ID:            "adb:SAMPLE001",
				Name:          "Pixel 8 Pro (USB)",
				Kind:          model.SourceADB,
				State:         source.StateReady,
				StatusMessage: "已连接就绪",
			},
		},
	}
	cp := psa.sources[0]
	psa.selectedSource = &cp

	testApp.Settings().SetTheme(psa.Theme)
	hv := newHomeView(psa)
	psa.home = hv
	win.SetContent(hv.root)
	hv.refreshSourceUI(true)
	hv.refreshTargetUI()
	hv.updateProgress(model.SyncProgress{
		Phase:       model.PhaseSyncing,
		Current:     142,
		Total:       380,
		CurrentFile: "IMG_20261113_095500.jpg",
		BytesPerSec: 38 * 1024 * 1024,
		Stats: model.SyncStats{
			Exported:      142,
			SkippedCached: 95,
			SkippedDedup:  18,
			Failed:        0,
		},
	}, false)

	previewDir := os.Getenv("UI_PREVIEW_DIR")
	if previewDir != "" {
		_ = os.MkdirAll(previewDir, 0o755)
		savePNG(t, win.Canvas().Capture(), filepath.Join(previewDir, "ui_light.png"))
	}

	psa.setThemeMode(ThemeModeDark)
	if previewDir != "" {
		savePNG(t, win.Canvas().Capture(), filepath.Join(previewDir, "ui_dark.png"))
	}

	psa.setThemeMode(ThemeModeAMOLED)
	if previewDir != "" {
		savePNG(t, win.Canvas().Capture(), filepath.Join(previewDir, "ui_amoled.png"))
	}

	psa.setThemeMode(ThemeModeLight)
	psa.setWorkMode(model.WorkModeOrganize)
	if previewDir != "" {
		savePNG(t, win.Canvas().Capture(), filepath.Join(previewDir, "ui_organize.png"))
	}

	psa.setWorkMode(model.WorkModePhone)
	psa.setLanguage("en")
	if previewDir != "" {
		savePNG(t, win.Canvas().Capture(), filepath.Join(previewDir, "ui_en.png"))
	}
}

func savePNG(t *testing.T, img image.Image, outPath string) {
	t.Helper()
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("create %s: %v", outPath, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode %s: %v", outPath, err)
	}
}
