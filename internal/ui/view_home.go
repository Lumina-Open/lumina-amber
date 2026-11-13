package ui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"lumina-amber/internal/i18n"
	"lumina-amber/internal/model"
	"lumina-amber/internal/source"
	"lumina-amber/internal/sync"
	"lumina-amber/internal/version"
)

type homeView struct {
	app  *AmberApp
	root fyne.CanvasObject

	// Theme-aware custom elements
	cards       []*ElevatedCard
	dividers    []*HairlineDivider
	iconTiles   []*IconTile
	pillButtons []*PillButton
	metricTiles []*MetricTile
	segTheme    *SegmentedControl
	segMode     *SegmentedControl
	btnLang     *PillButton
	btnSettings *PillButton
	modeBarRow  *fyne.Container
	titleText   *StyledText
	versionPill *SoftPill
	subText     *StyledText
	modeHint    *StyledText
	ruleText    *StyledText

	// Source row
	srcIconTile      *IconTile
	srcTitleRow      *fyne.Container
	srcActionsStack  *fyne.Container
	sourceNameText   *StyledText
	sourceSubText    *StyledText
	sourcePill       *SoftPill
	phoneActionsBox  *fyne.Container
	orgActionsBox    *fyne.Container
	btnSwitchSrc     *PillButton
	btnRefresh       *PillButton
	btnWifi          *PillButton
	btnPickDir       *PillButton
	btnPickOrgDir    *PillButton
	btnOpenOrgDir    *PillButton

	// Target row
	dstTitleRow    *fyne.Container
	dstActions     *fyne.Container
	targetNameText *StyledText
	targetPathText *StyledText
	targetPill     *SoftPill
	btnChooseDst   *PillButton
	btnRecentDst   *PillButton
	btnOpenDst     *PillButton

	// Media chips & organize option
	chipsBox      *fyne.Container
	optionsRow    *fyne.Container
	chipPhotos    *FilterChip
	chipVideos    *FilterChip
	chipDeleteSrc *FilterChip

	// Hero action & progress
	btnBackup   *PillButton
	btnPreview  *PillButton
	btnCancel   *PillButton
	progressRow *fyne.Container
	phaseText   *StyledText
	speedText   *StyledText
	progressBar *AppleProgressBar

	tileExported *MetricTile
	tileCached   *MetricTile
	tileDedup    *MetricTile
	tileFailed   *MetricTile

	// Bottom log strip
	footerRow   *fyne.Container
	lastLogText *StyledText
	btnShowLogs *PillButton
	logLines    []string
}

func newHomeView(app *AmberApp) *homeView {
	h := &homeView{app: app}
	h.build()
	return h
}

func (h *homeView) build() {
	t := h.app.Theme
	s := h.app.lang()

	// 1. Top Toolbar (macOS style)
	appTile := newIconTile(t, resIconAppLogo, 40, 22, 11,
		func(th *LuminaTheme) color.Color { return th.AccentColor() },
		func(th *LuminaTheme) color.Color { return color.NRGBA{R: 255, G: 255, B: 255, A: 255} },
	)
	h.titleText = newStyledText(version.AppName, t.PrimaryTextColor(), 17.5, true)
	h.versionPill = newSoftPill("v"+version.Version, t.AccentTintColor(), t.AccentColor())
	h.subText = newStyledText(s.HeaderSubtitle, t.MutedTextColor(), 12, false)

	titleLine := container.NewHBox(container.NewCenter(h.titleText), container.NewCenter(h.versionPill.Root))
	titleCol := container.NewVBox(titleLine, h.subText)
	headerLeft := container.NewHBox(
		container.NewCenter(appTile.Root),
		container.New(layout.NewCustomPaddedLayout(0, 0, 6, 0), titleCol),
	)

	h.segTheme = newSegmentedControl(t, []string{s.ThemeLight, s.ThemeDark, s.ThemeAMOLED}, themeModeToSegIndex(t.Mode()), func(idx int) {
		switch idx {
		case 0:
			h.app.setThemeMode(ThemeModeLight)
		case 1:
			h.app.setThemeMode(ThemeModeDark)
		case 2:
			h.app.setThemeMode(ThemeModeAMOLED)
		}
	})

	h.btnLang = newPillButton(t, s.BtnLangToggle, nil, PillSecondary, func() {
		next := i18n.LangEN
		if h.app.lang().LangCode == i18n.LangEN {
			next = i18n.LangZH
		}
		h.app.setLanguage(next)
	})

	h.btnSettings = newPillButton(t, s.BtnSettings, resIconSettings, PillSecondary, func() {
		h.app.showSettingsDialog()
	})

	header := container.NewBorder(
		nil, nil,
		headerLeft,
		container.NewHBox(
			container.NewCenter(h.segTheme.Root),
			container.NewCenter(h.btnLang),
			container.NewCenter(h.btnSettings),
		),
	)

	// Mode Switcher Bar: [ Phone Backup | Organize ]
	initialModeIdx := 0
	if h.app.Config.WorkMode == model.WorkModeOrganize {
		initialModeIdx = 1
	}
	h.segMode = newSegmentedControl(t, []string{s.ModePhone, s.ModeOrganize}, initialModeIdx, func(idx int) {
		if idx == 1 {
			h.app.setWorkMode(model.WorkModeOrganize)
		} else {
			h.app.setWorkMode(model.WorkModePhone)
		}
	})
	h.modeHint = newStyledText("", t.MutedTextColor(), 12, false)
	h.modeBarRow = container.NewBorder(nil, nil, container.NewCenter(h.segMode.Root), container.NewCenter(h.modeHint))

	// 2. Unified Pipeline Card (Source + Target + Rules in an Apple grouped inset card)
	// Row 1: Source (Phone or Organize Folder)
	h.srcIconTile = newIconTile(t, resIconPhone, 42, 22, 11,
		func(th *LuminaTheme) color.Color { return th.AccentTintColor() },
		func(th *LuminaTheme) color.Color { return th.AccentColor() },
	)

	h.sourceNameText = newStyledText(s.DetectingDevice, t.PrimaryTextColor(), 14.5, true)
	h.sourcePill = newSoftPill(s.PillDetecting, t.WarningTintColor(), t.WarningColor())
	h.sourceSubText = newStyledText(s.SourceSubHintDefault, t.MutedTextColor(), 12, false)

	h.srcTitleRow = container.NewHBox(container.NewCenter(h.sourceNameText), container.NewCenter(h.sourcePill.Root))
	srcInfo := container.NewVBox(
		h.srcTitleRow,
		h.sourceSubText,
	)

	h.btnSwitchSrc = newPillButton(t, s.BtnSwitch, nil, PillSecondary, func() {
		h.showSourceMenu()
	})
	h.btnSwitchSrc.SetTrailingIcon(resIconChevron)

	h.btnRefresh = newPillButton(t, s.BtnRefresh, resIconRefresh, PillSecondary, func() {
		h.app.refreshDevicesAsync(false)
	})
	h.btnWifi = newPillButton(t, s.BtnWifi, resIconWifi, PillSecondary, func() {
		h.app.showWirelessADBDialog()
	})
	h.btnPickDir = newPillButton(t, s.BtnPickMTP, resIconFolder, PillSecondary, func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			desc := source.NewMountedDescriptor(uri.Path())
			h.app.selectSource(desc)
			h.app.refreshDevicesAsync(false)
		}, h.app.Window)
	})

	h.phoneActionsBox = container.NewHBox(
		container.NewCenter(h.btnSwitchSrc),
		container.NewCenter(h.btnRefresh),
		container.NewCenter(h.btnWifi),
		container.NewCenter(h.btnPickDir),
	)

	h.btnPickOrgDir = newPillButton(t, s.BtnPickOrgDir, resIconFolder, PillTinted, func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			h.app.mu.Lock()
			h.app.Config.OrganizeSourceDir = uri.Path()
			h.app.mu.Unlock()
			h.app.saveConfig()
			h.refreshSourceUI(false)
		}, h.app.Window)
	})
	h.btnOpenOrgDir = newPillButton(t, s.BtnOpen, nil, PillSecondary, func() {
		if err := openInFileManager(h.app.configSnapshot().OrganizeSourceDir); err != nil {
			dialog.ShowError(err, h.app.Window)
		}
	})
	h.orgActionsBox = container.NewHBox(
		container.NewCenter(h.btnPickOrgDir),
		container.NewCenter(h.btnOpenOrgDir),
	)

	h.srcActionsStack = container.NewStack(h.phoneActionsBox, h.orgActionsBox)

	srcRow := container.NewBorder(
		nil, nil,
		container.NewCenter(h.srcIconTile.Root),
		h.srcActionsStack,
		container.New(layout.NewCustomPaddedLayout(2, 2, 10, 12), srcInfo),
	)

	div1 := newHairlineDivider(t, 52)

	// Row 2: Destination
	dstIconTile := newIconTile(t, resIconDrive, 42, 22, 11,
		func(th *LuminaTheme) color.Color { return th.IndigoTintColor() },
		func(th *LuminaTheme) color.Color { return th.IndigoColor() },
	)

	h.targetNameText = newStyledText(s.TargetUnsetTitle, t.PrimaryTextColor(), 14.5, true)
	h.targetPill = newSoftPill(s.PillUnset, t.WarningTintColor(), t.WarningColor())
	h.targetPathText = newStyledText(s.TargetUnsetHint, t.MutedTextColor(), 12, false)

	h.dstTitleRow = container.NewHBox(container.NewCenter(h.targetNameText), container.NewCenter(h.targetPill.Root))
	dstInfo := container.NewVBox(
		h.dstTitleRow,
		h.targetPathText,
	)

	h.btnChooseDst = newPillButton(t, s.BtnChooseTarget, resIconFolder, PillTinted, func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			h.app.mu.Lock()
			h.app.Config.TargetDir = uri.Path()
			h.app.Config.AddRecentTarget(uri.Path())
			h.app.mu.Unlock()
			h.app.saveConfig()
			h.refreshTargetUI()
		}, h.app.Window)
	})
	h.btnRecentDst = newPillButton(t, s.BtnRecent, nil, PillSecondary, func() {
		h.showRecentTargetsMenu()
	})
	h.btnRecentDst.SetTrailingIcon(resIconChevron)

	h.btnOpenDst = newPillButton(t, s.BtnOpen, nil, PillSecondary, func() {
		if err := openInFileManager(h.app.configSnapshot().TargetDir); err != nil {
			dialog.ShowError(err, h.app.Window)
		}
	})

	h.dstActions = container.NewHBox(
		container.NewCenter(h.btnChooseDst),
		container.NewCenter(h.btnRecentDst),
		container.NewCenter(h.btnOpenDst),
	)
	dstRow := container.NewBorder(
		nil, nil,
		container.NewCenter(dstIconTile.Root),
		h.dstActions,
		container.New(layout.NewCustomPaddedLayout(2, 2, 10, 12), dstInfo),
	)

	div2 := newHairlineDivider(t, 0)

	// Row 3: Media Filter Chips + Organize Move Option + Rule Hint
	h.chipPhotos = newFilterChip(t, s.FilterPhotos, resIconPhoto, h.app.Config.IncludePhotos, func(v bool) {
		h.app.mu.Lock()
		h.app.Config.IncludePhotos = v
		h.app.mu.Unlock()
		h.app.saveConfig()
		h.updateActionButtonsState()
	})
	h.chipVideos = newFilterChip(t, s.FilterVideos, resIconVideo, h.app.Config.IncludeVideos, func(v bool) {
		h.app.mu.Lock()
		h.app.Config.IncludeVideos = v
		h.app.mu.Unlock()
		h.app.saveConfig()
		h.updateActionButtonsState()
	})
	h.chipDeleteSrc = newFilterChip(t, s.FilterDeleteSrc, resIconFolder, h.app.Config.OrganizeDeleteSource, func(v bool) {
		h.app.mu.Lock()
		h.app.Config.OrganizeDeleteSource = v
		h.app.mu.Unlock()
		h.app.saveConfig()
	})

	h.ruleText = newStyledText(s.ArchiveRuleHint, t.MutedTextColor(), 11.5, false)

	h.chipsBox = container.NewHBox(h.chipPhotos, h.chipVideos, h.chipDeleteSrc)
	h.optionsRow = container.NewBorder(
		nil, nil,
		h.chipsBox,
		container.NewCenter(h.ruleText),
	)

	pipelineCard := newElevatedCard(t, container.NewVBox(
		srcRow,
		div1.Root,
		dstRow,
		div2.Root,
		h.optionsRow,
	))

	// 3. Hero Backup Action & Progress Card
	h.btnBackup = newPillButton(t, s.BtnStartBackup, resIconSync, PillPrimary, func() {
		h.app.startSync(false)
	})
	h.btnBackup.SetHero(44)

	h.btnPreview = newPillButton(t, s.BtnPreview, resIconSearch, PillTinted, func() {
		h.app.startSync(true)
	})
	h.btnPreview.SetHero(44)

	h.btnCancel = newPillButton(t, s.BtnStop, resIconStop, PillDanger, func() {
		h.app.cancelActiveSync()
	})
	h.btnCancel.SetHero(44)
	h.btnCancel.SetDisabled(true)

	heroButtons := container.NewBorder(
		nil, nil,
		nil,
		container.NewHBox(h.btnPreview, h.btnCancel),
		h.btnBackup,
	)

	h.phaseText = newStyledText(s.PhaseIdleDefault, t.PrimaryTextColor(), 13, true)
	h.speedText = newStyledText("", t.MutedTextColor(), 12, false)
	h.progressBar = newAppleProgressBar(t)
	h.progressRow = container.NewBorder(nil, nil, h.phaseText, h.speedText)

	h.tileExported = newMetricTile(t, s.MetricExported, "0",
		func(th *LuminaTheme) color.Color { return th.SuccessTintColor() },
		func(th *LuminaTheme) color.Color { return th.SuccessColor() },
	)
	h.tileCached = newMetricTile(t, s.MetricCached, "0",
		func(th *LuminaTheme) color.Color { return th.AccentTintColor() },
		func(th *LuminaTheme) color.Color { return th.AccentColor() },
	)
	h.tileDedup = newMetricTile(t, s.MetricDedup, "0",
		func(th *LuminaTheme) color.Color { return th.WarningTintColor() },
		func(th *LuminaTheme) color.Color { return th.WarningColor() },
	)
	h.tileFailed = newMetricTile(t, s.MetricFailed, "0",
		func(th *LuminaTheme) color.Color { return th.ErrorTintColor() },
		func(th *LuminaTheme) color.Color { return th.ErrorColor() },
	)

	metricsGrid := container.NewGridWithColumns(
		4,
		h.tileExported.Root,
		h.tileCached.Root,
		h.tileDedup.Root,
		h.tileFailed.Root,
	)

	heroCard := newElevatedCard(t, container.NewVBox(
		heroButtons,
		container.New(layout.NewCustomPaddedLayout(6, 2, 0, 0), h.progressRow),
		h.progressBar,
		container.New(layout.NewCustomPaddedLayout(6, 0, 0, 0), metricsGrid),
	))

	// 4. Minimalist Bottom Status Strip (Log modal on demand)
	h.lastLogText = newStyledText(s.RecentActivityIdle, t.MutedTextColor(), 12, false)
	h.btnShowLogs = newPillButton(t, s.BtnViewLogs, resIconLog, PillSecondary, func() {
		h.showLogsModal()
	})
	h.footerRow = container.NewBorder(
		nil, nil,
		container.NewCenter(h.lastLogText),
		h.btnShowLogs,
	)

	// Register theme-aware elements
	h.cards = []*ElevatedCard{pipelineCard, heroCard}
	h.dividers = []*HairlineDivider{div1, div2}
	h.iconTiles = []*IconTile{appTile, h.srcIconTile, dstIconTile}
	h.pillButtons = []*PillButton{
		h.btnLang, h.btnSettings, h.btnSwitchSrc, h.btnRefresh, h.btnWifi, h.btnPickDir,
		h.btnPickOrgDir, h.btnOpenOrgDir,
		h.btnChooseDst, h.btnRecentDst, h.btnOpenDst,
		h.btnBackup, h.btnPreview, h.btnCancel, h.btnShowLogs,
	}
	h.metricTiles = []*MetricTile{h.tileExported, h.tileCached, h.tileDedup, h.tileFailed}

	mainCol := container.NewVBox(
		header,
		container.New(layout.NewCustomPaddedLayout(6, 0, 0, 0), h.modeBarRow),
		container.New(layout.NewCustomPaddedLayout(4, 0, 0, 0), pipelineCard.Root),
		container.New(layout.NewCustomPaddedLayout(4, 0, 0, 0), heroCard.Root),
		container.New(layout.NewCustomPaddedLayout(2, 0, 4, 4), h.footerRow),
	)
	h.root = container.New(layout.NewCustomPaddedLayout(16, 14, 24, 24), mainCol)
	h.refreshModeUI()
}

func themeModeToSegIndex(mode ThemeMode) int {
	switch mode {
	case ThemeModeDark:
		return 1
	case ThemeModeAMOLED:
		return 2
	default:
		return 0
	}
}

func compactMiddle(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes || maxRunes < 12 {
		return s
	}
	head := maxRunes/2 - 2
	tail := maxRunes - head - 3
	return string(runes[:head]) + "..." + string(runes[len(runes)-tail:])
}

func (h *homeView) refreshModeUI() {
	t := h.app.Theme
	s := h.app.lang()
	isOrganize := h.app.configSnapshot().WorkMode == model.WorkModeOrganize

	if isOrganize {
		h.segMode.SetSelected(1)
		h.modeHint.SetText(s.ModeHintOrganize)
		h.srcIconTile.SetIcon(t, resIconFolder)
		h.phoneActionsBox.Hide()
		h.orgActionsBox.Show()
		h.chipDeleteSrc.Show()
		h.btnBackup.SetText(s.BtnStartOrganize)
	} else {
		h.segMode.SetSelected(0)
		h.modeHint.SetText(s.ModeHintPhone)
		h.srcIconTile.SetIcon(t, resIconPhone)
		h.phoneActionsBox.Show()
		h.orgActionsBox.Hide()
		h.chipDeleteSrc.Hide()
		h.btnBackup.SetText(s.BtnStartBackup)
	}
	h.modeBarRow.Refresh()
	h.srcActionsStack.Refresh()
	h.chipsBox.Refresh()
	h.optionsRow.Refresh()
	h.btnBackup.Refresh()
	h.refreshSourceUI(true)
}

func (h *homeView) refreshLanguage() {
	s := h.app.lang()
	h.subText.SetText(s.HeaderSubtitle)
	h.segTheme.SetLabels([]string{s.ThemeLight, s.ThemeDark, s.ThemeAMOLED})
	h.btnLang.SetText(s.BtnLangToggle)
	h.btnSettings.SetText(s.BtnSettings)
	h.segMode.SetLabels([]string{s.ModePhone, s.ModeOrganize})
	h.btnRefresh.SetText(s.BtnRefresh)
	h.btnWifi.SetText(s.BtnWifi)
	h.btnPickDir.SetText(s.BtnPickMTP)
	h.btnPickOrgDir.SetText(s.BtnPickOrgDir)
	h.btnOpenOrgDir.SetText(s.BtnOpen)
	h.btnChooseDst.SetText(s.BtnChooseTarget)
	h.btnRecentDst.SetText(s.BtnRecent)
	h.btnOpenDst.SetText(s.BtnOpen)
	h.chipPhotos.SetLabel(s.FilterPhotos)
	h.chipVideos.SetLabel(s.FilterVideos)
	h.chipDeleteSrc.SetLabel(s.FilterDeleteSrc)
	h.ruleText.SetText(s.ArchiveRuleHint)
	h.btnPreview.SetText(s.BtnPreview)
	h.btnCancel.SetText(s.BtnStop)
	h.tileExported.SetTitle(s.MetricExported)
	h.tileCached.SetTitle(s.MetricCached)
	h.tileDedup.SetTitle(s.MetricDedup)
	h.tileFailed.SetTitle(s.MetricFailed)
	h.btnShowLogs.SetText(s.BtnViewLogs)

	h.app.mu.Lock()
	syncing := h.app.isSyncing
	h.app.mu.Unlock()
	if !syncing {
		h.phaseText.SetText(s.PhaseIdleDefault)
	}
	if len(h.logLines) == 0 {
		h.lastLogText.SetText(s.RecentActivityIdle)
	}
	h.refreshModeUI()
	h.refreshTargetUIQuick()
	h.root.Refresh()
}

func (h *homeView) showSourceMenu() {
	s := h.app.lang()
	h.app.mu.Lock()
	sources := append([]source.SourceDescriptor(nil), h.app.sources...)
	h.app.mu.Unlock()

	if len(sources) == 0 {
		dialog.ShowInformation(s.NoDeviceModalTitle, s.NoDeviceModalMsg, h.app.Window)
		return
	}

	items := make([]*fyne.MenuItem, 0, len(sources))
	for _, src := range sources {
		s := src
		label := s.Name
		if s.State != source.StateReady {
			label = fmt.Sprintf("%s (%s)", s.Name, s.StatusMessage)
		}
		items = append(items, fyne.NewMenuItem(label, func() {
			h.app.selectSource(s)
		}))
	}

	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(h.btnSwitchSrc)
	pos.Y += h.btnSwitchSrc.Size().Height + 4
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), h.app.Window.Canvas(), pos)
}

func (h *homeView) showRecentTargetsMenu() {
	recents := h.app.Config.RecentTargets
	if len(recents) == 0 {
		return
	}

	items := make([]*fyne.MenuItem, 0, len(recents))
	for _, r := range recents {
		path := r
		items = append(items, fyne.NewMenuItem(path, func() {
			h.app.Config.TargetDir = path
			h.app.Config.AddRecentTarget(path)
			h.app.saveConfig()
			h.refreshTargetUI()
		}))
	}

	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(h.btnRecentDst)
	pos.Y += h.btnRecentDst.Size().Height + 4
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), h.app.Window.Canvas(), pos)
}

func (h *homeView) showLogsModal() {
	s := h.app.lang()
	text := s.LogEmpty
	if len(h.logLines) > 0 {
		text = strings.Join(h.logLines, "\n")
	}
	lbl := widget.NewLabel(text)
	lbl.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(lbl)
	scroll.SetMinSize(fyne.NewSize(520, 280))

	d := dialog.NewCustom(s.LogModalTitle, s.BtnClose, scroll, h.app.Window)
	d.Show()
}

func (h *homeView) applyTheme(t *LuminaTheme) {
	h.segTheme.SetSelected(themeModeToSegIndex(t.Mode()))

	h.titleText.SetColor(t.PrimaryTextColor())
	h.versionPill.Set("v"+version.Version, t.AccentTintColor(), t.AccentColor())
	h.subText.SetColor(t.MutedTextColor())
	h.modeHint.SetColor(t.MutedTextColor())
	h.ruleText.SetColor(t.MutedTextColor())
	h.sourceNameText.SetColor(t.PrimaryTextColor())
	h.sourceSubText.SetColor(t.MutedTextColor())
	h.targetNameText.SetColor(t.PrimaryTextColor())
	h.targetPathText.SetColor(t.MutedTextColor())
	h.phaseText.SetColor(t.PrimaryTextColor())
	h.speedText.SetColor(t.MutedTextColor())
	h.lastLogText.SetColor(t.MutedTextColor())

	for _, c := range h.cards {
		c.ApplyTheme(t)
	}
	for _, d := range h.dividers {
		d.ApplyTheme(t)
	}
	for _, it := range h.iconTiles {
		it.ApplyTheme(t)
	}
	for _, pb := range h.pillButtons {
		pb.ApplyTheme(t)
	}
	for _, mt := range h.metricTiles {
		mt.ApplyTheme(t)
	}
	h.segTheme.ApplyTheme(t)
	h.segMode.ApplyTheme(t)
	h.chipPhotos.ApplyTheme(t)
	h.chipVideos.ApplyTheme(t)
	h.chipDeleteSrc.ApplyTheme(t)
	h.progressBar.ApplyTheme(t)

	h.refreshModeUI()
	h.refreshTargetUI()
}

func (h *homeView) formatSourceDirs() string {
	s := h.app.lang()
	dirs := h.app.configSnapshot().SourceDirs
	if len(dirs) == 0 {
		return s.ScanDefaultDirs
	}
	short := make([]string, 0, len(dirs))
	for _, d := range dirs {
		short = append(short, strings.TrimPrefix(d, "/sdcard/"))
	}
	return s.ScanPrefix + strings.Join(short, " · ")
}

func (h *homeView) refreshSourceUI(silent bool) {
	t := h.app.Theme
	s := h.app.lang()
	cfg := h.app.configSnapshot()

	if cfg.WorkMode == model.WorkModeOrganize {
		orgDir := cfg.OrganizeSourceDir
		desc := source.NewOrganizeFolderDescriptor(orgDir)
		if orgDir == "" {
			h.sourceNameText.SetText(s.NoOrgDirTitle)
			h.sourcePill.Set(s.PillUnselected, t.WarningTintColor(), t.WarningColor())
			h.sourceSubText.SetText(s.NoOrgDirSub)
			h.btnOpenOrgDir.SetDisabled(true)
		} else if desc.State == source.StateReady {
			h.sourceNameText.SetText(compactMiddle(filepath.Base(orgDir), 28))
			h.sourcePill.Set(s.PillReady, t.SuccessTintColor(), t.SuccessColor())
			h.sourceSubText.SetText(compactMiddle(orgDir, 56))
			h.btnOpenOrgDir.SetDisabled(false)
		} else {
			h.sourceNameText.SetText(compactMiddle(filepath.Base(orgDir), 28))
			h.sourcePill.Set(s.PillUnavailable, t.ErrorTintColor(), t.ErrorColor())
			h.sourceSubText.SetText(compactMiddle(orgDir+" ("+desc.StatusMessage+")", 56))
			h.btnOpenOrgDir.SetDisabled(true)
		}
		h.srcTitleRow.Refresh()
		h.updateActionButtonsState()
		return
	}

	h.app.mu.Lock()
	adbBin := h.app.adbPath
	sources := append([]source.SourceDescriptor(nil), h.app.sources...)
	sel := h.app.selectedSource
	h.app.mu.Unlock()

	h.btnSwitchSrc.SetText(s.SwitchText(len(sources)))
	h.btnSwitchSrc.SetDisabled(len(sources) <= 1)
	h.phoneActionsBox.Refresh()
	h.srcActionsStack.Refresh()

	if sel != nil {
		h.sourceNameText.SetText(compactMiddle(sel.Name, 34))

		switch sel.State {
		case source.StateReady:
			h.sourcePill.Set(s.PillReady, t.SuccessTintColor(), t.SuccessColor())
			if sel.Kind == model.SourceMounted && sel.RootPath != "" {
				h.sourceSubText.SetText(compactMiddle(s.DirPrefix+sel.RootPath, 52))
			} else {
				h.sourceSubText.SetText(compactMiddle(h.formatSourceDirs(), 52))
			}
		case source.StateUnauthorized:
			h.sourcePill.Set(s.PillWaitAuth, t.WarningTintColor(), t.WarningColor())
			h.sourceSubText.SetText(s.AuthorizeHint)
		default:
			h.sourcePill.Set(s.PillUnavailable, t.ErrorTintColor(), t.ErrorColor())
			h.sourceSubText.SetText(compactMiddle(sel.StatusMessage, 52))
		}
	} else {
		h.sourceNameText.SetText(s.NoAndroidDetected)
		if adbBin == "" {
			h.sourcePill.Set(s.PillNoADB, t.WarningTintColor(), t.WarningColor())
			h.sourceSubText.SetText(s.NoADBSubHint)
		} else {
			h.sourcePill.Set(s.PillDisconnected, t.WellColor(), t.MutedTextColor())
			h.sourceSubText.SetText(s.DisconnectedSubHint)
		}
	}

	if !silent {
		if sel != nil {
			h.appendLog(fmt.Sprintf(s.DeviceStatusFmt, sel.Name, sel.StatusMessage))
		} else {
			h.appendLog(s.RefreshNoneLog)
		}
	}
	h.srcTitleRow.Refresh()
	h.updateActionButtonsState()
}

func (h *homeView) refreshTargetUI() {
	h.refreshTargetUIWithProbe(true)
}

func (h *homeView) refreshTargetUIQuick() {
	h.refreshTargetUIWithProbe(false)
}

func (h *homeView) refreshTargetUIWithProbe(probeWrite bool) {
	t := h.app.Theme
	s := h.app.lang()
	cfg := h.app.configSnapshot()
	targetDir := cfg.TargetDir
	var health sync.TargetHealth
	if probeWrite {
		health = sync.CheckTargetHealth(targetDir)
	} else {
		health = sync.CheckTargetQuick(targetDir)
	}

	if targetDir == "" {
		h.targetNameText.SetText(s.TargetUnsetTitle)
		h.targetPill.Set(s.PillUnset, t.WarningTintColor(), t.WarningColor())
		h.targetPathText.SetText(s.TargetUnsetSubHint)
		h.btnOpenDst.SetDisabled(true)
	} else {
		displayName := filepath.Base(targetDir)
		if displayName == "" || displayName == "/" {
			displayName = targetDir
		}
		h.targetNameText.SetText(compactMiddle(displayName, 26))
		h.targetPathText.SetText(compactMiddle(targetDir, 56))

		if health.Ready {
			h.targetPill.Set(health.StatusMessage, t.SuccessTintColor(), t.SuccessColor())
			h.btnOpenDst.SetDisabled(false)
		} else {
			h.targetPill.Set(health.StatusMessage, t.ErrorTintColor(), t.ErrorColor())
			h.btnOpenDst.SetDisabled(true)
		}
	}

	if len(h.app.Config.RecentTargets) > 0 {
		h.btnRecentDst.Show()
	} else {
		h.btnRecentDst.Hide()
	}

	h.dstActions.Refresh()
	h.dstTitleRow.Refresh()
	h.updateActionButtonsStateWithReady(health.Ready)
}

func (h *homeView) updateActionButtonsState() {
	cfg := h.app.configSnapshot()
	targetReady := sync.CheckTargetQuick(cfg.TargetDir).Ready
	h.updateActionButtonsStateWithReady(targetReady)
}

func (h *homeView) updateActionButtonsStateWithReady(targetReady bool) {
	h.app.mu.Lock()
	busy := h.app.isSyncing
	sel := h.app.selectedSource
	cfg := h.app.Config
	h.app.mu.Unlock()

	if busy {
		h.btnBackup.SetDisabled(true)
		h.btnPreview.SetDisabled(true)
		h.btnCancel.SetDisabled(false)
		return
	}

	h.btnCancel.SetDisabled(true)

	var sourceReady bool
	if cfg.WorkMode == model.WorkModeOrganize {
		sourceReady = source.NewOrganizeFolderDescriptor(cfg.OrganizeSourceDir).State == source.StateReady
	} else {
		sourceReady = sel != nil && sel.State == source.StateReady
	}

	filterReady := cfg.IncludePhotos || cfg.IncludeVideos
	ready := sourceReady && targetReady && filterReady
	h.btnBackup.SetDisabled(!ready)
	h.btnPreview.SetDisabled(!ready)
}

func (h *homeView) setSyncRunningState(running bool, dryRun bool) {
	s := h.app.lang()
	if running {
		if dryRun {
			h.phaseText.SetText(s.PhasePreviewStart)
		} else {
			h.phaseText.SetText(s.PhaseSyncStart)
		}
		h.speedText.SetText("")
		h.progressBar.SetValue(0)
		h.tileExported.SetValue("0")
		h.tileCached.SetValue("0")
		h.tileDedup.SetValue("0")
		h.tileFailed.SetValue("0")
		h.progressRow.Refresh()
	}
	h.updateActionButtonsState()
}

func (h *homeView) updateProgress(p model.SyncProgress, dryRun bool) {
	s := h.app.lang()
	h.tileExported.SetValue(fmt.Sprintf("%d", p.Stats.Exported))
	h.tileCached.SetValue(fmt.Sprintf("%d", p.Stats.SkippedCached))
	h.tileDedup.SetValue(fmt.Sprintf("%d", p.Stats.SkippedDedup))
	h.tileFailed.SetValue(fmt.Sprintf("%d", p.Stats.Failed))

	var pct float64
	if p.Total > 0 {
		pct = float64(p.Current) / float64(p.Total)
		h.progressBar.SetValue(pct)
	}

	switch p.Phase {
	case model.PhaseIndexing:
		h.phaseText.SetText(s.PhaseIndexing)
	case model.PhaseScanning:
		h.phaseText.SetText(s.PhaseScanning)
	case model.PhaseSyncing:
		verb := s.VerbSyncing
		if dryRun {
			verb = s.VerbComparing
		}
		h.phaseText.SetText(fmt.Sprintf("%s %d / %d · %s", verb, p.Current, p.Total, compactMiddle(filepath.Base(p.CurrentFile), 36)))
		if p.BytesPerSec > 0 {
			h.speedText.SetText(fmt.Sprintf("%s/s · %d%%", sync.FormatBytes(uint64(p.BytesPerSec)), int(pct*100)))
		} else if p.Total > 0 {
			h.speedText.SetText(fmt.Sprintf("%d%%", int(pct*100)))
		}
	case model.PhaseCompleted:
		h.progressBar.SetValue(1)
		h.phaseText.SetText(p.LogLine)
		h.speedText.SetText("100%")
	case model.PhaseCancelled:
		h.phaseText.SetText(s.PhaseCancelled)
		h.speedText.SetText("")
	}
	h.progressRow.Refresh()

	if p.LogLine != "" {
		h.appendLog(p.LogLine)
	}
}

func (h *homeView) appendLog(line string) {
	if line == "" {
		return
	}
	tsLine := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), line)
	h.logLines = append(h.logLines, tsLine)
	if len(h.logLines) > 120 {
		h.logLines = h.logLines[len(h.logLines)-120:]
	}
	h.lastLogText.SetText(compactMiddle(h.app.lang().RecentActivityPrefix+line, 64))
	h.footerRow.Refresh()
}
