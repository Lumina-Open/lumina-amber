package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"lumina-amber/internal/i18n"
	"lumina-amber/internal/meta"
	"lumina-amber/internal/source"
	"lumina-amber/internal/version"
)

func (a *AmberApp) showWirelessADBDialog() {
	s := a.lang()
	addrEntry := widget.NewEntry()
	addrEntry.SetPlaceHolder(s.WirelessPlaceholder)

	hint := widget.NewLabel(s.WirelessHint)
	hint.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(hint, addrEntry)
	dialog.ShowCustomConfirm(s.WirelessTitle, s.BtnConnect, s.BtnCancel, content, func(ok bool) {
		if !ok {
			return
		}
		addr := strings.TrimSpace(addrEntry.Text)
		if addr == "" {
			return
		}
		adbBin := source.FindADBBinary(a.Config.CustomADBPath)
		if adbBin == "" {
			dialog.ShowError(fmt.Errorf("%s", s.ADBNotFoundErr), a.Window)
			return
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			msg, err := source.ConnectWirelessADB(ctx, adbBin, addr)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, a.Window)
					return
				}
				a.home.appendLog(s.WirelessLogPrefix + msg)
				a.refreshDevicesAsync(false)
			})
		}()
	}, a.Window)
}

func formatTargetMetaSummary(s i18n.Catalog, targetDir string) string {
	if strings.TrimSpace(targetDir) == "" {
		return s.MetaNotSelected
	}
	indexedCount, histCount, metaPath, err := meta.ReadStats(targetDir)
	if err != nil {
		return fmt.Sprintf(s.MetaReadErrFmt, err, metaPath)
	}
	return fmt.Sprintf(
		s.MetaSummaryFmt,
		indexedCount, histCount, metaPath,
	)
}

func (a *AmberApp) showSettingsDialog() {
	s := a.lang()
	cfg := a.configSnapshot()

	langIdx := 0
	selectedLang := i18n.LangZH
	if i18n.Normalize(cfg.Language) == i18n.LangEN {
		langIdx = 1
		selectedLang = i18n.LangEN
	}
	langSeg := newSegmentedControl(a.Theme, []string{s.SettingsLangZH, s.SettingsLangEN}, langIdx, func(idx int) {
		if idx == 1 {
			selectedLang = i18n.LangEN
		} else {
			selectedLang = i18n.LangZH
		}
	})

	dirsEntry := widget.NewMultiLineEntry()
	dirsEntry.SetText(strings.Join(cfg.SourceDirs, "\n"))
	dirsEntry.SetMinRowsVisible(4)

	adbEntry := widget.NewEntry()
	adbEntry.SetText(cfg.CustomADBPath)
	detected := source.FindADBBinary("")
	if detected != "" {
		adbEntry.SetPlaceHolder(fmt.Sprintf(s.ADBAutoDetectFmt, detected))
	} else {
		adbEntry.SetPlaceHolder(s.ADBManualPlaceholder)
	}

	statsLbl := widget.NewLabel(formatTargetMetaSummary(s, cfg.TargetDir))
	statsLbl.Wrapping = fyne.TextWrapWord

	btnReindex := newPillButton(a.Theme, s.BtnReindexMeta, resIconRefresh, PillTinted, func() {
		a.mu.Lock()
		busy := a.isSyncing
		curTarget := a.Config.TargetDir
		a.mu.Unlock()
		if busy {
			dialog.ShowInformation(s.InfoTitle, s.SyncBusyReindexMsg, a.Window)
			return
		}
		if strings.TrimSpace(curTarget) == "" {
			dialog.ShowInformation(s.InfoTitle, s.SelectTargetFirstMsg, a.Window)
			return
		}
		go func() {
			_ = meta.ClearIndex(curTarget)
			err := a.Engine.RefreshTargetIndex(context.Background(), curTarget, nil)
			idxCnt, _, _, _ := meta.ReadStats(curTarget)
			summary := formatTargetMetaSummary(s, curTarget)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, a.Window)
					return
				}
				statsLbl.SetText(summary)
				a.home.appendLog(fmt.Sprintf(s.ReindexDoneFmt, idxCnt))
			})
		}()
	})

	btnClearHistory := newPillButton(a.Theme, s.BtnClearHistory, nil, PillSecondary, func() {
		a.mu.Lock()
		busy := a.isSyncing
		curTarget := a.Config.TargetDir
		a.mu.Unlock()
		if busy {
			dialog.ShowInformation(s.InfoTitle, s.SyncBusyClearMsg, a.Window)
			return
		}
		if strings.TrimSpace(curTarget) == "" {
			dialog.ShowInformation(s.InfoTitle, s.SelectTargetFirstMsg, a.Window)
			return
		}
		if err := meta.ClearHistory(curTarget); err != nil {
			dialog.ShowError(err, a.Window)
			return
		}
		statsLbl.SetText(formatTargetMetaSummary(s, curTarget))
		a.home.appendLog(s.HistoryClearedLog)
	})

	aboutLbl := widget.NewLabel(fmt.Sprintf(s.AboutLabelFmt, version.AppName, version.Version, version.Author))
	aboutLbl.Wrapping = fyne.TextWrapWord

	form := container.NewVBox(
		widget.NewLabelWithStyle(s.SettingsLanguageTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		langSeg.Root,
		widget.NewSeparator(),
		widget.NewLabelWithStyle(s.SettingsDirsTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dirsEntry,
		widget.NewLabelWithStyle(s.SettingsADBTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		adbEntry,
		widget.NewSeparator(),
		widget.NewLabelWithStyle(s.SettingsMetaTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		statsLbl,
		container.NewHBox(btnReindex, btnClearHistory),
		widget.NewSeparator(),
		widget.NewLabelWithStyle(s.SettingsAboutTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		aboutLbl,
	)

	d := dialog.NewCustomConfirm(s.SettingsTitle, s.BtnSave, s.BtnCancel, form, func(ok bool) {
		if !ok {
			return
		}
		var newDirs []string
		for _, line := range strings.Split(dirsEntry.Text, "\n") {
			clean := strings.TrimSpace(line)
			if clean != "" {
				newDirs = append(newDirs, clean)
			}
		}
		a.mu.Lock()
		if len(newDirs) > 0 {
			a.Config.SourceDirs = newDirs
		}
		a.Config.CustomADBPath = strings.TrimSpace(adbEntry.Text)
		langChanged := a.Config.Language != selectedLang
		a.mu.Unlock()

		if langChanged {
			a.setLanguage(selectedLang)
		} else {
			a.saveConfig()
			a.refreshDevicesAsync(true)
		}
	}, a.Window)
	d.Resize(fyne.NewSize(560, 520))
	d.Show()
}
