package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"lumina-amber/internal/db"
	"lumina-amber/internal/i18n"
	"lumina-amber/internal/model"
	"lumina-amber/internal/source"
	syncengine "lumina-amber/internal/sync"
	"lumina-amber/internal/version"
)

// AmberApp coordinates application state, persistence, device discovery, and UI rendering.
type AmberApp struct {
	FyneApp fyne.App
	Window  fyne.Window
	DB      *db.Database
	Engine  *syncengine.Engine
	Theme   *LuminaTheme
	Config  model.AppConfig

	mu             sync.Mutex
	refreshing     atomic.Bool
	adbPath        string
	sources        []source.SourceDescriptor
	selectedSource *source.SourceDescriptor
	isSyncing      bool
	cancelSync     context.CancelFunc
	stopWatcher    context.CancelFunc

	home *homeView
}

func (a *AmberApp) configSnapshot() model.AppConfig {
	a.mu.Lock()
	cfg := a.Config
	cfg.SourceDirs = append([]string(nil), a.Config.SourceDirs...)
	cfg.RecentTargets = append([]string(nil), a.Config.RecentTargets...)
	a.mu.Unlock()
	return cfg
}

// NewAmberApp initializes the database, loads saved defaults, and builds the main window.
func NewAmberApp(dbPath string) (*AmberApp, error) {
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	cfg, err := database.LoadConfig()
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("load config: %w", err)
	}
	i18n.SetCurrent(cfg.Language)

	a := app.NewWithID("org.lumina.amber")
	luminaTheme := NewLuminaTheme(ThemeMode(cfg.ThemeMode))
	a.Settings().SetTheme(luminaTheme)

	w := a.NewWindow(fmt.Sprintf("%s v%s", version.AppName, version.Version))
	w.Resize(fyne.NewSize(900, 620))
	w.CenterOnScreen()

	psa := &AmberApp{
		FyneApp: a,
		Window:  w,
		DB:      database,
		Engine:  syncengine.NewEngine(),
		Theme:   luminaTheme,
		Config:  cfg,
	}
	psa.adbPath = source.FindADBBinary(cfg.CustomADBPath)
	return psa, nil
}

// Run builds the UI, starts background device discovery, and enters the Fyne event loop.
func (a *AmberApp) Run() {
	a.home = newHomeView(a)
	a.Window.SetContent(a.home.root)

	watchCtx, cancelWatch := context.WithCancel(context.Background())
	a.stopWatcher = cancelWatch

	a.Window.SetOnClosed(func() {
		cancelWatch()
		a.mu.Lock()
		cancel := a.cancelSync
		a.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		_ = a.DB.Close()
	})

	a.refreshDevicesAsync(false)
	a.home.refreshTargetUI()
	go a.watchDevicesLoop(watchCtx)

	a.Window.ShowAndRun()
}

func (a *AmberApp) watchDevicesLoop(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			busy := a.isSyncing
			a.mu.Unlock()
			if !busy {
				a.refreshDevicesAsync(true)
			}
		}
	}
}

func (a *AmberApp) setThemeMode(mode ThemeMode) {
	a.mu.Lock()
	a.Config.ThemeMode = string(mode)
	a.mu.Unlock()
	a.Theme = NewLuminaTheme(mode)
	a.FyneApp.Settings().SetTheme(a.Theme)
	_ = a.DB.SaveConfig(a.configSnapshot())
	if a.home != nil {
		a.home.applyTheme(a.Theme)
	}
}

func (a *AmberApp) setWorkMode(mode string) {
	if mode != model.WorkModeOrganize {
		mode = model.WorkModePhone
	}
	a.mu.Lock()
	a.Config.WorkMode = mode
	a.mu.Unlock()
	a.saveConfig()
	if a.home != nil {
		a.home.refreshModeUI()
	}
}

func (a *AmberApp) lang() i18n.Catalog {
	return i18n.For(a.configSnapshot().Language)
}

func (a *AmberApp) setLanguage(lang string) {
	normalized := i18n.SetCurrent(lang)
	a.mu.Lock()
	a.Config.Language = normalized
	watching := a.stopWatcher != nil
	a.mu.Unlock()
	a.saveConfig()
	if a.home != nil {
		a.home.refreshLanguage()
	}
	if watching {
		a.refreshDevicesAsync(true)
	}
}

func (a *AmberApp) saveConfig() {
	if err := a.DB.SaveConfig(a.configSnapshot()); err != nil {
		fyne.Do(func() {
			dialog.ShowError(err, a.Window)
		})
	}
}

func (a *AmberApp) selectSource(desc source.SourceDescriptor) {
	a.mu.Lock()
	cp := desc
	a.selectedSource = &cp
	a.Config.DefaultSourceKind = desc.Kind
	a.Config.DefaultSourceID = desc.ID
	a.Config.DefaultSourceName = desc.Name
	a.mu.Unlock()

	a.saveConfig()
	if a.home != nil {
		a.home.refreshSourceUI(true)
	}
}

func (a *AmberApp) refreshDevicesAsync(silent bool) {
	if !a.refreshing.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer a.refreshing.Store(false)

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		cfg := a.configSnapshot()
		adbBin := source.FindADBBinary(cfg.CustomADBPath)
		var discovered []source.SourceDescriptor

		if adbBin != "" {
			if adbDevs, err := source.ListADBDevices(ctx, adbBin); err == nil {
				discovered = append(discovered, adbDevs...)
			}
		}
		discovered = append(discovered, source.DiscoverMountedSources()...)

		// Keep remembered mounted directory accessible in the list even if temporarily unplugged.
		if cfg.DefaultSourceKind == model.SourceMounted && strings.HasPrefix(cfg.DefaultSourceID, "dir:") {
			found := false
			for _, d := range discovered {
				if d.ID == cfg.DefaultSourceID {
					found = true
					break
				}
			}
			if !found {
				rootPath := strings.TrimPrefix(cfg.DefaultSourceID, "dir:")
				discovered = append(discovered, source.NewMountedDescriptor(rootPath))
			}
		}

		a.mu.Lock()
		a.adbPath = adbBin
		prevID := ""
		if a.selectedSource != nil {
			prevID = a.selectedSource.ID
		}
		a.sources = discovered

		var nextSelected *source.SourceDescriptor
		// Priority 1: Keep active selection if still present
		if prevID != "" {
			for i := range discovered {
				if discovered[i].ID == prevID {
					cp := discovered[i]
					nextSelected = &cp
					break
				}
			}
		}
		// Priority 2: Auto-match remembered default source
		if nextSelected == nil && cfg.DefaultSourceID != "" {
			for i := range discovered {
				if discovered[i].ID == cfg.DefaultSourceID {
					cp := discovered[i]
					nextSelected = &cp
					break
				}
			}
		}
		// Priority 3: Auto-select first ready device
		if nextSelected == nil {
			for i := range discovered {
				if discovered[i].State == source.StateReady {
					cp := discovered[i]
					nextSelected = &cp
					break
				}
			}
		}
		// Priority 4: Any discovered device (e.g. unauthorized) so user sees status guidance
		if nextSelected == nil && len(discovered) > 0 {
			cp := discovered[0]
			nextSelected = &cp
		}

		a.selectedSource = nextSelected
		a.mu.Unlock()

		fyne.Do(func() {
			a.home.refreshSourceUI(silent)
			if silent {
				a.home.refreshTargetUIQuick()
			} else {
				a.home.refreshTargetUI()
			}
		})
	}()
}

func (a *AmberApp) buildActiveMediaSource() (source.MediaSource, error) {
	if a.Config.WorkMode == model.WorkModeOrganize {
		desc := source.NewOrganizeFolderDescriptor(a.Config.OrganizeSourceDir)
		if desc.State != source.StateReady {
			return nil, fmt.Errorf("%s: %s", desc.Name, desc.StatusMessage)
		}
		return source.NewMountedSource(desc), nil
	}

	a.mu.Lock()
	sel := a.selectedSource
	adbBin := a.adbPath
	a.mu.Unlock()

	if sel == nil {
		return nil, fmt.Errorf("%s", a.lang().NoSourceErr)
	}
	if sel.State != source.StateReady {
		return nil, fmt.Errorf("%s: %s", sel.Name, sel.StatusMessage)
	}

	if sel.Kind == model.SourceADB {
		return source.NewADBSource(adbBin, *sel), nil
	}
	return source.NewMountedSource(*sel), nil
}

func (a *AmberApp) startSync(dryRun bool) {
	a.mu.Lock()
	if a.isSyncing {
		a.mu.Unlock()
		return
	}
	a.isSyncing = true
	a.mu.Unlock()

	src, err := a.buildActiveMediaSource()
	if err != nil {
		a.mu.Lock()
		a.isSyncing = false
		a.mu.Unlock()
		dialog.ShowError(err, a.Window)
		return
	}

	health := syncengine.CheckTargetHealth(a.Config.TargetDir)
	if !health.Ready {
		a.mu.Lock()
		a.isSyncing = false
		a.mu.Unlock()
		dialog.ShowError(fmt.Errorf(a.lang().ErrorTargetUnavailFmt, health.StatusMessage), a.Window)
		return
	}

	desc := src.Descriptor()
	a.mu.Lock()
	if a.Config.WorkMode == model.WorkModePhone {
		a.Config.DefaultSourceKind = desc.Kind
		a.Config.DefaultSourceID = desc.ID
		a.Config.DefaultSourceName = desc.Name
	}
	a.Config.AddRecentTarget(a.Config.TargetDir)
	cfgSnap := a.Config
	cfgSnap.SourceDirs = append([]string(nil), a.Config.SourceDirs...)
	a.mu.Unlock()
	a.saveConfig()

	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.cancelSync = cancel
	a.mu.Unlock()

	a.home.setSyncRunningState(true, dryRun)

	deleteSource := cfgSnap.WorkMode == model.WorkModeOrganize && cfgSnap.OrganizeDeleteSource
	opts := syncengine.SyncOptions{
		Source:     src,
		SourceDirs: cfgSnap.SourceDirs,
		Filter: model.MediaFilter{
			IncludePhotos: cfgSnap.IncludePhotos,
			IncludeVideos: cfgSnap.IncludeVideos,
		},
		TargetDir:    cfgSnap.TargetDir,
		DryRun:       dryRun,
		DeleteSource: deleteSource,
		Lang:         cfgSnap.Language,
	}

	go func() {
		defer func() {
			cancel()
			a.mu.Lock()
			a.isSyncing = false
			a.cancelSync = nil
			a.mu.Unlock()
			fyne.Do(func() {
				a.home.setSyncRunningState(false, dryRun)
				a.home.refreshTargetUI()
			})
		}()

		_, runErr := a.Engine.Run(ctx, opts, func(p model.SyncProgress) {
			fyne.Do(func() {
				a.home.updateProgress(p, dryRun)
			})
		})

		fyne.Do(func() {
			if runErr != nil && runErr != context.Canceled {
				a.home.appendLog(fmt.Sprintf(a.lang().ErrorPrefixFmt, runErr))
				dialog.ShowError(runErr, a.Window)
			}
		})
	}()
}

func (a *AmberApp) cancelActiveSync() {
	a.mu.Lock()
	cancel := a.cancelSync
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func openInFileManager(dir string) error {
	if dir == "" {
		return fmt.Errorf("%s", i18n.S().DirEmptyErr)
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", dir)
	} else {
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}
