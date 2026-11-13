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

	"lumina-photo-sync/internal/model"
	"lumina-photo-sync/internal/source"
	"lumina-photo-sync/internal/sync"
	"lumina-photo-sync/internal/version"
)

type homeView struct {
	app  *PhotoSyncApp
	root fyne.CanvasObject

	// Theme-aware custom elements
	cards       []*ElevatedCard
	dividers    []*HairlineDivider
	iconTiles   []*IconTile
	pillButtons []*PillButton
	metricTiles []*MetricTile
	segTheme    *SegmentedControl
	segMode     *SegmentedControl
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
	authorText  *StyledText
	btnShowLogs *PillButton
	logLines    []string
}

func newHomeView(app *PhotoSyncApp) *homeView {
	h := &homeView{app: app}
	h.build()
	return h
}

func (h *homeView) build() {
	t := h.app.Theme

	// 1. Top Toolbar (macOS style)
	appTile := newIconTile(t, resIconAppLogo, 40, 22, 11,
		func(th *LuminaTheme) color.Color { return th.AccentColor() },
		func(th *LuminaTheme) color.Color { return color.NRGBA{R: 255, G: 255, B: 255, A: 255} },
	)
	h.titleText = newStyledText(version.AppName, t.PrimaryTextColor(), 17.5, true)
	h.versionPill = newSoftPill("v"+version.Version, t.AccentTintColor(), t.AccentColor())
	h.subText = newStyledText("图片/视频分目录 · 年/年月归档 · SHA-256 自动去重", t.MutedTextColor(), 12, false)

	titleLine := container.NewHBox(container.NewCenter(h.titleText), container.NewCenter(h.versionPill.Root))
	titleCol := container.NewVBox(titleLine, h.subText)
	headerLeft := container.NewHBox(
		container.NewCenter(appTile.Root),
		container.New(layout.NewCustomPaddedLayout(0, 0, 6, 0), titleCol),
	)

	h.segTheme = newSegmentedControl(t, []string{"浅色", "深色", "纯黑"}, themeModeToSegIndex(t.Mode()), func(idx int) {
		switch idx {
		case 0:
			h.app.setThemeMode(ThemeModeLight)
		case 1:
			h.app.setThemeMode(ThemeModeDark)
		case 2:
			h.app.setThemeMode(ThemeModeAMOLED)
		}
	})

	btnSettings := newPillButton(t, "设置", resIconSettings, PillSecondary, func() {
		h.app.showSettingsDialog()
	})

	header := container.NewBorder(
		nil, nil,
		headerLeft,
		container.NewHBox(container.NewCenter(h.segTheme.Root), container.NewCenter(btnSettings)),
	)

	// Mode Switcher Bar: [ 手机备份 | 目录整理 ]
	initialModeIdx := 0
	if h.app.Config.WorkMode == model.WorkModeOrganize {
		initialModeIdx = 1
	}
	h.segMode = newSegmentedControl(t, []string{"手机备份", "目录整理"}, initialModeIdx, func(idx int) {
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

	h.sourceNameText = newStyledText("正在检测设备...", t.PrimaryTextColor(), 14.5, true)
	h.sourcePill = newSoftPill("检测中", t.WarningTintColor(), t.WarningColor())
	h.sourceSubText = newStyledText("请连接 USB 并开启调试，或选择本地/MTP目录", t.MutedTextColor(), 12, false)

	h.srcTitleRow = container.NewHBox(container.NewCenter(h.sourceNameText), container.NewCenter(h.sourcePill.Root))
	srcInfo := container.NewVBox(
		h.srcTitleRow,
		h.sourceSubText,
	)

	h.btnSwitchSrc = newPillButton(t, "切换", nil, PillSecondary, func() {
		h.showSourceMenu()
	})
	h.btnSwitchSrc.SetTrailingIcon(resIconChevron)

	h.btnRefresh = newPillButton(t, "刷新", resIconRefresh, PillSecondary, func() {
		h.app.refreshDevicesAsync(false)
	})
	h.btnWifi = newPillButton(t, "无线", resIconWifi, PillSecondary, func() {
		h.app.showWirelessADBDialog()
	})
	h.btnPickDir = newPillButton(t, "MTP目录", resIconFolder, PillSecondary, func() {
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

	h.btnPickOrgDir = newPillButton(t, "选择待整理目录", resIconFolder, PillTinted, func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			h.app.Config.OrganizeSourceDir = uri.Path()
			h.app.saveConfig()
			h.refreshSourceUI(false)
		}, h.app.Window)
	})
	h.btnOpenOrgDir = newPillButton(t, "打开", nil, PillSecondary, func() {
		if err := openInFileManager(h.app.Config.OrganizeSourceDir); err != nil {
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

	h.targetNameText = newStyledText("尚未选择目标目录", t.PrimaryTextColor(), 14.5, true)
	h.targetPill = newSoftPill("未设置", t.WarningTintColor(), t.WarningColor())
	h.targetPathText = newStyledText("支持本地文件夹或外接移动硬盘", t.MutedTextColor(), 12, false)

	h.dstTitleRow = container.NewHBox(container.NewCenter(h.targetNameText), container.NewCenter(h.targetPill.Root))
	dstInfo := container.NewVBox(
		h.dstTitleRow,
		h.targetPathText,
	)

	h.btnChooseDst = newPillButton(t, "选择目标目录", resIconFolder, PillTinted, func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			h.app.Config.TargetDir = uri.Path()
			h.app.Config.AddRecentTarget(uri.Path())
			h.app.saveConfig()
			h.refreshTargetUI()
		}, h.app.Window)
	})
	h.btnRecentDst = newPillButton(t, "最近", nil, PillSecondary, func() {
		h.showRecentTargetsMenu()
	})
	h.btnRecentDst.SetTrailingIcon(resIconChevron)

	h.btnOpenDst = newPillButton(t, "打开", nil, PillSecondary, func() {
		if err := openInFileManager(h.app.Config.TargetDir); err != nil {
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
	h.chipPhotos = newFilterChip(t, "照片", resIconPhoto, h.app.Config.IncludePhotos, func(v bool) {
		h.app.mu.Lock()
		h.app.Config.IncludePhotos = v
		h.app.mu.Unlock()
		h.app.saveConfig()
		h.updateActionButtonsState()
	})
	h.chipVideos = newFilterChip(t, "视频", resIconVideo, h.app.Config.IncludeVideos, func(v bool) {
		h.app.mu.Lock()
		h.app.Config.IncludeVideos = v
		h.app.mu.Unlock()
		h.app.saveConfig()
		h.updateActionButtonsState()
	})
	h.chipDeleteSrc = newFilterChip(t, "移动模式 (整理后移除原文件)", resIconFolder, h.app.Config.OrganizeDeleteSource, func(v bool) {
		h.app.mu.Lock()
		h.app.Config.OrganizeDeleteSource = v
		h.app.mu.Unlock()
		h.app.saveConfig()
	})

	h.ruleText = newStyledText("结构: picture/YYYY/YYYY-MM · video/YYYY/YYYY-MM", t.MutedTextColor(), 11.5, false)

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
	h.btnBackup = newPillButton(t, "一键自动备份", resIconSync, PillPrimary, func() {
		h.app.startSync(false)
	})
	h.btnBackup.SetHero(44)

	h.btnPreview = newPillButton(t, "预览扫描", resIconSearch, PillTinted, func() {
		h.app.startSync(true)
	})
	h.btnPreview.SetHero(44)

	h.btnCancel = newPillButton(t, "停止", resIconStop, PillDanger, func() {
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

	h.phaseText = newStyledText("就绪：下次打开可直接一键执行", t.PrimaryTextColor(), 13, true)
	h.speedText = newStyledText("", t.MutedTextColor(), 12, false)
	h.progressBar = newAppleProgressBar(t)
	h.progressRow = container.NewBorder(nil, nil, h.phaseText, h.speedText)

	h.tileExported = newMetricTile(t, "新增归档", "0",
		func(th *LuminaTheme) color.Color { return th.SuccessTintColor() },
		func(th *LuminaTheme) color.Color { return th.SuccessColor() },
	)
	h.tileCached = newMetricTile(t, "已备份跳过", "0",
		func(th *LuminaTheme) color.Color { return th.AccentTintColor() },
		func(th *LuminaTheme) color.Color { return th.AccentColor() },
	)
	h.tileDedup = newMetricTile(t, "内容去重", "0",
		func(th *LuminaTheme) color.Color { return th.WarningTintColor() },
		func(th *LuminaTheme) color.Color { return th.WarningColor() },
	)
	h.tileFailed = newMetricTile(t, "失败", "0",
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
	h.lastLogText = newStyledText("最近动态：等待操作", t.MutedTextColor(), 12, false)
	h.authorText = newStyledText(fmt.Sprintf("v%s · 作者 %s", version.Version, version.Author), t.MutedTextColor(), 11.5, false)
	h.btnShowLogs = newPillButton(t, "查看明细", resIconLog, PillSecondary, func() {
		h.showLogsModal()
	})
	h.footerRow = container.NewBorder(
		nil, nil,
		container.NewCenter(h.lastLogText),
		container.NewHBox(container.NewCenter(h.authorText), h.btnShowLogs),
	)

	// Register theme-aware elements
	h.cards = []*ElevatedCard{pipelineCard, heroCard}
	h.dividers = []*HairlineDivider{div1, div2}
	h.iconTiles = []*IconTile{appTile, h.srcIconTile, dstIconTile}
	h.pillButtons = []*PillButton{
		btnSettings, h.btnSwitchSrc, h.btnRefresh, h.btnWifi, h.btnPickDir,
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
	isOrganize := h.app.Config.WorkMode == model.WorkModeOrganize

	if isOrganize {
		h.segMode.SetSelected(1)
		h.modeHint.SetText("从本地目录、SD卡或移动硬盘导入并按年月整理去重")
		h.srcIconTile.SetIcon(t, resIconFolder)
		h.phoneActionsBox.Hide()
		h.orgActionsBox.Show()
		h.chipDeleteSrc.Show()
		h.btnBackup.SetText("一键导入整理")
	} else {
		h.segMode.SetSelected(0)
		h.modeHint.SetText("连接 Android 手机，下次打开可直接点击一键自动备份")
		h.srcIconTile.SetIcon(t, resIconPhone)
		h.phoneActionsBox.Show()
		h.orgActionsBox.Hide()
		h.chipDeleteSrc.Hide()
		h.btnBackup.SetText("一键自动备份")
	}
	h.modeBarRow.Refresh()
	h.srcActionsStack.Refresh()
	h.chipsBox.Refresh()
	h.optionsRow.Refresh()
	h.btnBackup.Refresh()
	h.refreshSourceUI(true)
}

func (h *homeView) showSourceMenu() {
	h.app.mu.Lock()
	sources := append([]source.SourceDescriptor(nil), h.app.sources...)
	h.app.mu.Unlock()

	if len(sources) == 0 {
		dialog.ShowInformation("无可用设备", "当前未发现其他 ADB 设备或挂载目录，请先连接手机或点击「MTP目录」。", h.app.Window)
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
	text := "暂无同步记录。"
	if len(h.logLines) > 0 {
		text = strings.Join(h.logLines, "\n")
	}
	lbl := widget.NewLabel(text)
	lbl.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(lbl)
	scroll.SetMinSize(fyne.NewSize(520, 280))

	d := dialog.NewCustom("同步明细日志", "关闭", scroll, h.app.Window)
	d.Show()
}

func (h *homeView) applyTheme(t *LuminaTheme) {
	h.segTheme.SetSelected(themeModeToSegIndex(t.Mode()))

	h.titleText.SetColor(t.PrimaryTextColor())
	h.versionPill.Set("v"+version.Version, t.AccentTintColor(), t.AccentColor())
	h.subText.SetColor(t.MutedTextColor())
	h.authorText.SetColor(t.MutedTextColor())
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
	if len(h.app.Config.SourceDirs) == 0 {
		return "扫描默认目录"
	}
	short := make([]string, 0, len(h.app.Config.SourceDirs))
	for _, d := range h.app.Config.SourceDirs {
		short = append(short, strings.TrimPrefix(d, "/sdcard/"))
	}
	return "扫描: " + strings.Join(short, " · ")
}

func (h *homeView) refreshSourceUI(silent bool) {
	t := h.app.Theme

	if h.app.Config.WorkMode == model.WorkModeOrganize {
		orgDir := h.app.Config.OrganizeSourceDir
		desc := source.NewOrganizeFolderDescriptor(orgDir)
		if orgDir == "" {
			h.sourceNameText.SetText("尚未选择待整理目录")
			h.sourcePill.Set("未选择", t.WarningTintColor(), t.WarningColor())
			h.sourceSubText.SetText("请选择包含散乱照片或视频的文件夹（支持子目录递归扫描）")
			h.btnOpenOrgDir.SetDisabled(true)
		} else if desc.State == source.StateReady {
			h.sourceNameText.SetText(compactMiddle(filepath.Base(orgDir), 28))
			h.sourcePill.Set("已就绪", t.SuccessTintColor(), t.SuccessColor())
			h.sourceSubText.SetText(compactMiddle(orgDir, 56))
			h.btnOpenOrgDir.SetDisabled(false)
		} else {
			h.sourceNameText.SetText(compactMiddle(filepath.Base(orgDir), 28))
			h.sourcePill.Set("不可用", t.ErrorTintColor(), t.ErrorColor())
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

	if len(sources) > 1 {
		h.btnSwitchSrc.SetText(fmt.Sprintf("切换 (%d)", len(sources)))
		h.btnSwitchSrc.SetDisabled(false)
	} else {
		h.btnSwitchSrc.SetText("切换")
		h.btnSwitchSrc.SetDisabled(len(sources) == 0)
	}
	h.phoneActionsBox.Refresh()
	h.srcActionsStack.Refresh()

	if sel != nil {
		h.sourceNameText.SetText(compactMiddle(sel.Name, 34))

		switch sel.State {
		case source.StateReady:
			h.sourcePill.Set("已就绪", t.SuccessTintColor(), t.SuccessColor())
			if sel.Kind == model.SourceMounted && sel.RootPath != "" {
				h.sourceSubText.SetText(compactMiddle("目录: "+sel.RootPath, 52))
			} else {
				h.sourceSubText.SetText(compactMiddle(h.formatSourceDirs(), 52))
			}
		case source.StateUnauthorized:
			h.sourcePill.Set("等待授权", t.WarningTintColor(), t.WarningColor())
			h.sourceSubText.SetText("请在手机屏幕上点击「允许 USB 调试」并勾选始终允许")
		default:
			h.sourcePill.Set("不可用", t.ErrorTintColor(), t.ErrorColor())
			h.sourceSubText.SetText(compactMiddle(sel.StatusMessage, 52))
		}
	} else {
		h.sourceNameText.SetText("未检测到 Android 设备")
		if adbBin == "" {
			h.sourcePill.Set("未找到 ADB", t.WarningTintColor(), t.WarningColor())
			h.sourceSubText.SetText("未检测到 adb，可切换到「目录整理」或点击「MTP目录」")
		} else {
			h.sourcePill.Set("未连接", t.WellColor(), t.MutedTextColor())
			h.sourceSubText.SetText("请插入 USB 并开启「USB 调试」，或使用「无线」/「目录整理」")
		}
	}

	if !silent {
		if sel != nil {
			h.appendLog(fmt.Sprintf("设备状态: %s (%s)", sel.Name, sel.StatusMessage))
		} else {
			h.appendLog("刷新完成：暂未检测到已连接的手机")
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
	cfg := h.app.configSnapshot()
	targetDir := cfg.TargetDir
	var health sync.TargetHealth
	if probeWrite {
		health = sync.CheckTargetHealth(targetDir)
	} else {
		health = sync.CheckTargetQuick(targetDir)
	}

	if targetDir == "" {
		h.targetNameText.SetText("尚未选择备份目标")
		h.targetPill.Set("未设置", t.WarningTintColor(), t.WarningColor())
		h.targetPathText.SetText("请选择本地磁盘或外接移动硬盘中的目标文件夹")
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
	if running {
		if dryRun {
			h.phaseText.SetText("正在执行预览扫描...")
		} else {
			h.phaseText.SetText("正在启动整理同步...")
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
		h.phaseText.SetText("正在校验目标目录去重索引...")
	case model.PhaseScanning:
		h.phaseText.SetText("正在扫描媒体文件...")
	case model.PhaseSyncing:
		verb := "正在归档"
		if dryRun {
			verb = "正在比对"
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
		h.phaseText.SetText("操作已取消")
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
	h.lastLogText.SetText(compactMiddle("最近动态："+line, 64))
	h.footerRow.Refresh()
}
