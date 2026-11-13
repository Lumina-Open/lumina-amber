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

	"lumina-amber/internal/meta"
	"lumina-amber/internal/source"
	"lumina-amber/internal/version"
)

func (a *AmberApp) showWirelessADBDialog() {
	addrEntry := widget.NewEntry()
	addrEntry.SetPlaceHolder("例如 192.168.1.50:5555")

	hint := widget.NewLabel("在手机「开发者选项 → 无线调试」查看 IP 与端口，输入后即可无线备份。")
	hint.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(hint, addrEntry)
	dialog.ShowCustomConfirm("无线 ADB 连接", "连接", "取消", content, func(ok bool) {
		if !ok {
			return
		}
		addr := strings.TrimSpace(addrEntry.Text)
		if addr == "" {
			return
		}
		adbBin := source.FindADBBinary(a.Config.CustomADBPath)
		if adbBin == "" {
			dialog.ShowError(fmt.Errorf("未找到 adb 可执行文件，请先在设置中指定或安装 platform-tools"), a.Window)
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
				a.home.appendLog("无线 ADB: " + msg)
				a.refreshDevicesAsync(false)
			})
		}()
	}, a.Window)
}

func formatTargetMetaSummary(targetDir string) string {
	if strings.TrimSpace(targetDir) == "" {
		return "尚未选择目标目录\n照片/视频的哈希索引与增量记录将随备份保存在目标目录的 meta/ 子目录中（切换新目录互不影响）"
	}
	indexedCount, histCount, metaPath, err := meta.ReadStats(targetDir)
	if err != nil {
		return fmt.Sprintf("读取目标 meta 索引失败: %v\n路径: %s", err, metaPath)
	}
	return fmt.Sprintf(
		"当前目标已索引文件: %d · 增量缓存记录: %d\n目标元数据路径: %s",
		indexedCount, histCount, metaPath,
	)
}

func (a *AmberApp) showSettingsDialog() {
	dirsEntry := widget.NewMultiLineEntry()
	dirsEntry.SetText(strings.Join(a.Config.SourceDirs, "\n"))
	dirsEntry.SetMinRowsVisible(4)

	adbEntry := widget.NewEntry()
	adbEntry.SetText(a.Config.CustomADBPath)
	detected := source.FindADBBinary("")
	if detected != "" {
		adbEntry.SetPlaceHolder("留空自动检测: " + detected)
	} else {
		adbEntry.SetPlaceHolder("未检测到 adb，可手动填写路径")
	}

	statsLbl := widget.NewLabel(formatTargetMetaSummary(a.Config.TargetDir))
	statsLbl.Wrapping = fyne.TextWrapWord

	btnReindex := newPillButton(a.Theme, "重建目标 meta 索引", resIconRefresh, PillTinted, func() {
		a.mu.Lock()
		busy := a.isSyncing
		a.mu.Unlock()
		if busy {
			dialog.ShowInformation("提示", "备份或扫描正在进行中，请等待完成后再重建索引。", a.Window)
			return
		}
		if strings.TrimSpace(a.Config.TargetDir) == "" {
			dialog.ShowInformation("提示", "请先选择目标备份目录", a.Window)
			return
		}
		targetDir := a.Config.TargetDir
		go func() {
			_ = meta.ClearIndex(targetDir)
			err := a.Engine.RefreshTargetIndex(context.Background(), targetDir, nil)
			idxCnt, _, _, _ := meta.ReadStats(targetDir)
			summary := formatTargetMetaSummary(targetDir)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, a.Window)
					return
				}
				statsLbl.SetText(summary)
				a.home.appendLog(fmt.Sprintf("已重建目标 meta 索引，共 %d 个文件", idxCnt))
			})
		}()
	})

	btnClearHistory := newPillButton(a.Theme, "清空当前目标增量缓存", nil, PillSecondary, func() {
		a.mu.Lock()
		busy := a.isSyncing
		a.mu.Unlock()
		if busy {
			dialog.ShowInformation("提示", "备份或扫描正在进行中，请等待完成后再清理缓存。", a.Window)
			return
		}
		if strings.TrimSpace(a.Config.TargetDir) == "" {
			dialog.ShowInformation("提示", "请先选择目标备份目录", a.Window)
			return
		}
		if err := meta.ClearHistory(a.Config.TargetDir); err != nil {
			dialog.ShowError(err, a.Window)
			return
		}
		statsLbl.SetText(formatTargetMetaSummary(a.Config.TargetDir))
		a.home.appendLog("已清空当前目标增量缓存（下次备份将重新比对目标 meta/ 哈希索引）")
	})

	aboutLbl := widget.NewLabel(fmt.Sprintf("%s v%s · 作者: %s · 100%% 离线运行", version.AppName, version.Version, version.Author))
	aboutLbl.Wrapping = fyne.TextWrapWord

	form := container.NewVBox(
		widget.NewLabelWithStyle("手机端扫描目录（每行一个路径）", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dirsEntry,
		widget.NewLabelWithStyle("ADB 路径（可选，默认自动检测）", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		adbEntry,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("目标目录 meta 哈希索引与缓存", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		statsLbl,
		container.NewHBox(btnReindex, btnClearHistory),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("关于软件", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		aboutLbl,
	)

	d := dialog.NewCustomConfirm("偏好设置", "保存", "取消", form, func(ok bool) {
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
		if len(newDirs) > 0 {
			a.Config.SourceDirs = newDirs
		}
		a.Config.CustomADBPath = strings.TrimSpace(adbEntry.Text)
		a.saveConfig()
		a.refreshDevicesAsync(true)
	}, a.Window)
	d.Resize(fyne.NewSize(540, 450))
	d.Show()
}
