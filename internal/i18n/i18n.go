package i18n

import (
	"fmt"
	"strings"
	"sync/atomic"
)

const (
	LangZH = "zh"
	LangEN = "en"
)

var activeLang atomic.Value

func init() {
	activeLang.Store(LangZH)
}

// Normalize coerces any language tag into a supported UI language ("zh" or "en").
func Normalize(lang string) string {
	lower := strings.ToLower(strings.TrimSpace(lang))
	if strings.HasPrefix(lower, "en") {
		return LangEN
	}
	return LangZH
}

// SetCurrent sets the process-wide default UI language used by discovery/status helpers.
func SetCurrent(lang string) string {
	n := Normalize(lang)
	activeLang.Store(n)
	return n
}

// Current returns the active process-wide UI language ("zh" or "en").
func Current() string {
	if v, ok := activeLang.Load().(string); ok && v != "" {
		return v
	}
	return LangZH
}

// S returns the catalog for the currently active language.
func S() Catalog {
	return For(Current())
}

// For returns the localized catalog for lang.
func For(lang string) Catalog {
	if Normalize(lang) == LangEN {
		return enCatalog
	}
	return zhCatalog
}

// Catalog defines all user-visible bilingual copy across Lumina Amber.
type Catalog struct {
	LangCode string

	// Top Header & Navigation
	HeaderSubtitle string
	ThemeLight     string
	ThemeDark      string
	ThemeAMOLED    string
	BtnLangToggle  string
	BtnSettings    string

	// Work Modes
	ModePhone        string
	ModeOrganize     string
	ModeHintPhone    string
	ModeHintOrganize string

	// Source Card
	DetectingDevice      string
	PillDetecting        string
	SourceSubHintDefault string
	BtnSwitch            string
	BtnRefresh           string
	BtnWifi              string
	BtnPickMTP           string
	BtnPickOrgDir        string
	BtnOpen              string
	NoOrgDirTitle        string
	PillUnselected       string
	NoOrgDirSub          string
	PillReady            string
	PillUnavailable      string
	PillWaitAuth         string
	AuthorizeHint        string
	NoAndroidDetected    string
	PillNoADB            string
	NoADBSubHint         string
	PillDisconnected     string
	DisconnectedSubHint  string
	ScanDefaultDirs      string
	ScanPrefix           string
	DirPrefix            string
	NoDeviceModalTitle   string
	NoDeviceModalMsg     string

	// Target Card
	TargetUnsetTitle   string
	PillUnset          string
	TargetUnsetHint    string
	TargetUnsetSubHint string
	BtnChooseTarget    string
	BtnRecent          string

	// Filters & Rule
	FilterPhotos     string
	FilterVideos     string
	FilterDeleteSrc  string
	ArchiveRuleHint  string
	BtnStartBackup   string
	BtnStartOrganize string
	BtnPreview       string
	BtnStop          string

	// Progress & Metrics
	PhaseIdleDefault  string
	PhasePreviewStart string
	PhaseSyncStart    string
	PhaseIndexing     string
	PhaseScanning     string
	VerbSyncing       string
	VerbComparing     string
	PhaseCancelled    string
	MetricExported    string
	MetricCached      string
	MetricDedup       string
	MetricFailed      string

	// Bottom Status & Logs
	RecentActivityPrefix string
	RecentActivityIdle   string
	BtnViewLogs          string
	LogEmpty             string
	LogModalTitle        string
	BtnClose             string
	RefreshNoneLog       string
	DeviceStatusFmt      string

	// Settings & Wireless Dialogs
	WirelessHint          string
	WirelessPlaceholder   string
	WirelessTitle         string
	BtnConnect            string
	BtnCancel             string
	ADBNotFoundErr        string
	WirelessLogPrefix     string
	MetaNotSelected       string
	MetaReadErrFmt        string
	MetaSummaryFmt        string
	ADBAutoDetectFmt      string
	ADBManualPlaceholder  string
	BtnReindexMeta        string
	InfoTitle             string
	SyncBusyReindexMsg    string
	SelectTargetFirstMsg  string
	ReindexDoneFmt        string
	BtnClearHistory       string
	SyncBusyClearMsg      string
	HistoryClearedLog     string
	AboutLabelFmt         string
	SettingsLanguageTitle string
	SettingsLangZH        string
	SettingsLangEN        string
	SettingsDirsTitle     string
	SettingsADBTitle      string
	SettingsMetaTitle     string
	SettingsAboutTitle    string
	SettingsTitle         string
	BtnSave               string
	ErrorTargetUnavailFmt string
	ErrorPrefixFmt        string
	DirEmptyErr           string
	NoSourceErr           string

	// Backend Discovery & Health
	DescMTPRootLabel      string
	DescOrganizeLabel     string
	DescGVFSMounted       string
	DescCardDeviceSuffix  string
	DescCardCameraSuffix  string
	DescDCIMDetected      string
	DescDirNotExist       string
	DescDirReady          string
	ADBStateConnected     string
	ADBStateUnauthorized  string
	ADBStateOffline       string
	ADBStateFmt           string
	VolNotSelected        string
	VolInvalidPathFmt     string
	VolExtUnmountedFmt    string
	VolMissingOrUnplugged string
	VolNoAccessFmt        string
	VolNotDir             string
	VolNotWritableFmt     string
	VolReady              string
	VolReadyFreeFmt       string

	// Sync Engine Messages
	EngineNoSource           string
	EngineIndexingLogFmt     string
	EngineVerifyingMeta      string
	EngineOpenMetaFailFmt    string
	EngineReadMetaFailFmt    string
	EngineScanTargetFailFmt  string
	EngineUpdateMetaFailFmt  string
	EngineReadHistFailFmt    string
	EngineScanningSrcFmt     string
	EngineScanSrcFailFmt     string
	EngineNoMediaFound       string
	EngineMkTempFailFmt      string
	EnginePullFailFmt        string
	EngineWriteMetaFailFmt   string
	EngineWriteHistFailFmt   string
	EngineDedupSkippedFmt    string
	EngineMkTargetDirFailFmt string
	EngineWriteTargetFailFmt string
	EngineCancelled          string
	EngineDoneFmt            string
	EnginePreviewDoneFmt     string
	EngineStatTargetFailFmt  string
	EngineSameDirErr         string
	EngineTargetInSrcErr     string
	EngineMoveSrcInTargetErr string
}

// SwitchText formats the source switcher label with optional count.
func (c Catalog) SwitchText(count int) string {
	if count > 1 {
		return fmt.Sprintf("%s (%d)", c.BtnSwitch, count)
	}
	return c.BtnSwitch
}

var zhCatalog = Catalog{
	LangCode: LangZH,

	HeaderSubtitle: "图片/视频分目录 · 年/年月归档 · SHA-256 自动去重",
	ThemeLight:     "浅色",
	ThemeDark:      "深色",
	ThemeAMOLED:    "纯黑",
	BtnLangToggle:  "EN",
	BtnSettings:    "设置",

	ModePhone:        "手机备份",
	ModeOrganize:     "目录整理",
	ModeHintPhone:    "连接 Android 手机，下次打开可直接点击一键自动备份",
	ModeHintOrganize: "从本地目录、SD卡或移动硬盘导入并按年月整理去重",

	DetectingDevice:      "正在检测设备...",
	PillDetecting:        "检测中",
	SourceSubHintDefault: "请连接 USB 并开启调试，或选择本地/MTP目录",
	BtnSwitch:            "切换",
	BtnRefresh:           "刷新",
	BtnWifi:              "无线",
	BtnPickMTP:           "MTP目录",
	BtnPickOrgDir:        "选择待整理目录",
	BtnOpen:              "打开",
	NoOrgDirTitle:        "尚未选择待整理目录",
	PillUnselected:       "未选择",
	NoOrgDirSub:          "请选择包含散乱照片或视频的文件夹（支持子目录递归扫描）",
	PillReady:            "已就绪",
	PillUnavailable:      "不可用",
	PillWaitAuth:         "等待授权",
	AuthorizeHint:        "请在手机屏幕上点击「允许 USB 调试」并勾选始终允许",
	NoAndroidDetected:    "未检测到 Android 设备",
	PillNoADB:            "未找到 ADB",
	NoADBSubHint:         "未检测到 adb，可切换到「目录整理」或点击「MTP目录」",
	PillDisconnected:     "未连接",
	DisconnectedSubHint:  "请插入 USB 并开启「USB 调试」，或使用「无线」/「目录整理」",
	ScanDefaultDirs:      "扫描默认目录",
	ScanPrefix:           "扫描: ",
	DirPrefix:            "目录: ",
	NoDeviceModalTitle:   "无可用设备",
	NoDeviceModalMsg:     "当前未发现其他 ADB 设备或挂载目录，请先连接手机或点击「MTP目录」。",

	TargetUnsetTitle:   "尚未选择备份目标",
	PillUnset:          "未设置",
	TargetUnsetHint:    "支持本地文件夹或外接移动硬盘",
	TargetUnsetSubHint: "请选择本地磁盘或外接移动硬盘中的目标文件夹",
	BtnChooseTarget:    "选择目标目录",
	BtnRecent:          "最近",

	FilterPhotos:     "照片",
	FilterVideos:     "视频",
	FilterDeleteSrc:  "移动模式 (整理后移除原文件)",
	ArchiveRuleHint:  "结构: picture/YYYY/YYYY-MM · video/YYYY/YYYY-MM",
	BtnStartBackup:   "一键自动备份",
	BtnStartOrganize: "一键导入整理",
	BtnPreview:       "预览扫描",
	BtnStop:          "停止",

	PhaseIdleDefault:  "就绪：下次打开可直接一键执行",
	PhasePreviewStart: "正在执行预览扫描...",
	PhaseSyncStart:    "正在启动整理同步...",
	PhaseIndexing:     "正在校验目标目录去重索引...",
	PhaseScanning:     "正在扫描媒体文件...",
	VerbSyncing:       "正在归档",
	VerbComparing:     "正在比对",
	PhaseCancelled:    "操作已取消",
	MetricExported:    "新增归档",
	MetricCached:      "已备份跳过",
	MetricDedup:       "内容去重",
	MetricFailed:      "失败",

	RecentActivityPrefix: "最近动态：",
	RecentActivityIdle:   "最近动态：等待操作",
	BtnViewLogs:          "查看明细",
	LogEmpty:             "暂无同步记录。",
	LogModalTitle:        "同步明细日志",
	BtnClose:             "关闭",
	RefreshNoneLog:       "刷新完成：暂未检测到已连接的手机",
	DeviceStatusFmt:      "设备状态: %s (%s)",

	WirelessHint:          "在手机「开发者选项 → 无线调试」查看 IP 与端口，输入后即可无线备份。",
	WirelessPlaceholder:   "例如 192.168.1.50:5555",
	WirelessTitle:         "无线 ADB 连接",
	BtnConnect:            "连接",
	BtnCancel:             "取消",
	ADBNotFoundErr:        "未找到 adb 可执行文件，请先在设置中指定或安装 platform-tools",
	WirelessLogPrefix:     "无线 ADB: ",
	MetaNotSelected:       "尚未选择目标目录\n照片/视频的哈希索引与增量记录将随备份保存在目标目录的 meta/ 子目录中（切换新目录互不影响）",
	MetaReadErrFmt:        "读取目标 meta 索引失败: %v\n路径: %s",
	MetaSummaryFmt:        "当前目标已索引文件: %d · 增量缓存记录: %d\n目标元数据路径: %s",
	ADBAutoDetectFmt:      "留空自动检测: %s",
	ADBManualPlaceholder:  "未检测到 adb，可手动填写路径",
	BtnReindexMeta:        "重建目标 meta 索引",
	InfoTitle:             "提示",
	SyncBusyReindexMsg:    "备份或扫描正在进行中，请等待完成后再重建索引。",
	SelectTargetFirstMsg:  "请先选择目标备份目录",
	ReindexDoneFmt:        "已重建目标 meta 索引，共 %d 个文件",
	BtnClearHistory:       "清空当前目标增量缓存",
	SyncBusyClearMsg:      "备份或扫描正在进行中，请等待完成后再清理缓存。",
	HistoryClearedLog:     "已清空当前目标增量缓存（下次备份将重新比对目标 meta/ 哈希索引）",
	AboutLabelFmt:         "%s v%s · 作者: %s · 100%% 离线运行",
	SettingsLanguageTitle: "界面语言 / Language",
	SettingsLangZH:        "简体中文",
	SettingsLangEN:        "English",
	SettingsDirsTitle:     "手机端扫描目录（每行一个路径）",
	SettingsADBTitle:      "ADB 路径（可选，默认自动检测）",
	SettingsMetaTitle:     "目标目录 meta 哈希索引与缓存",
	SettingsAboutTitle:    "关于软件",
	SettingsTitle:         "偏好设置",
	BtnSave:               "保存",
	ErrorTargetUnavailFmt: "目标存储不可用：%s",
	ErrorPrefixFmt:        "错误: %v",
	DirEmptyErr:           "目录为空",
	NoSourceErr:           "未检测到可用手机或目录，请连接手机或选择本地/MTP目录",

	DescMTPRootLabel:      "目录",
	DescOrganizeLabel:     "待整理目录",
	DescGVFSMounted:       "GVFS MTP 已挂载",
	DescCardDeviceSuffix:  " (存储卡/设备)",
	DescCardCameraSuffix:  " (存储卡/相机)",
	DescDCIMDetected:      "检测到 DCIM 目录",
	DescDirNotExist:       "目录不存在或未挂载",
	DescDirReady:          "目录已就绪",
	ADBStateConnected:     "已连接就绪",
	ADBStateUnauthorized:  "请在手机上点击「允许 USB 调试」",
	ADBStateOffline:       "设备离线，请重新插拔或重连",
	ADBStateFmt:           "状态: %s",
	VolNotSelected:        "尚未选择目标目录",
	VolInvalidPathFmt:     "路径无效: %v",
	VolExtUnmountedFmt:    "外接存储未挂载: %s",
	VolMissingOrUnplugged: "目标目录不存在或硬盘未连接",
	VolNoAccessFmt:        "无法访问目标目录: %v",
	VolNotDir:             "目标路径不是文件夹",
	VolNotWritableFmt:     "目标目录不可写: %v",
	VolReady:              "已就绪",
	VolReadyFreeFmt:       "已就绪 · 剩余 %s",

	EngineNoSource:           "未选择来源设备",
	EngineIndexingLogFmt:     "正在并行建立目标目录 meta 去重索引 (%d/%d)...",
	EngineVerifyingMeta:      "正在并行校验目标目录 meta 索引...",
	EngineOpenMetaFailFmt:    "打开目标 meta 索引失败: %w",
	EngineReadMetaFailFmt:    "读取目标 meta 索引失败: %w",
	EngineScanTargetFailFmt:  "扫描目标目录失败: %w",
	EngineUpdateMetaFailFmt:  "目标 meta 索引更新失败: %w",
	EngineReadHistFailFmt:    "读取目标同步历史失败: %w",
	EngineScanningSrcFmt:     "正在扫描 %s...",
	EngineScanSrcFailFmt:     "扫描设备失败: %w",
	EngineNoMediaFound:       "未发现符合条件的照片或视频",
	EngineMkTempFailFmt:      "创建临时目录失败: %w",
	EnginePullFailFmt:        "导出失败 [%s]: %v",
	EngineWriteMetaFailFmt:   "写入目标 meta 索引失败: %w",
	EngineWriteHistFailFmt:   "写入目标同步历史失败: %w",
	EngineDedupSkippedFmt:    "已去重跳过: %s (与 %s 内容一致)",
	EngineMkTargetDirFailFmt: "创建目标目录失败 [%s]: %v",
	EngineWriteTargetFailFmt: "写入目标文件失败 [%s]: %v",
	EngineCancelled:          "备份已取消",
	EngineDoneFmt:            "备份完成：新增 %d，缓存跳过 %d，内容去重 %d，失败 %d",
	EnginePreviewDoneFmt:     "预览完成：待导出新文件 %d (%s)，缓存跳过 %d，内容去重 %d",
	EngineStatTargetFailFmt:  "检查目标路径失败: %w",
	EngineSameDirErr:         "目标目录不能与来源目录相同，请选择单独的整理目标目录",
	EngineTargetInSrcErr:     "目标目录不能位于来源目录内部",
	EngineMoveSrcInTargetErr: "移动整理模式下，来源目录不能位于目标目录内部",
}

var enCatalog = Catalog{
	LangCode: LangEN,

	HeaderSubtitle: "Picture & Video · Year/Month Archive · SHA-256 Dedup",
	ThemeLight:     "Light",
	ThemeDark:      "Dark",
	ThemeAMOLED:    "OLED",
	BtnLangToggle:  "中文",
	BtnSettings:    "Settings",

	ModePhone:        "Phone Backup",
	ModeOrganize:     "Organize",
	ModeHintPhone:    "Connect an Android device for instant one-click backup next time",
	ModeHintOrganize: "Import photos & videos from any folder or drive and organize by date",

	DetectingDevice:      "Detecting devices...",
	PillDetecting:        "Scanning",
	SourceSubHintDefault: "Connect via USB Debugging, or choose a local/MTP folder",
	BtnSwitch:            "Switch",
	BtnRefresh:           "Refresh",
	BtnWifi:              "Wi-Fi",
	BtnPickMTP:           "MTP Folder",
	BtnPickOrgDir:        "Choose Folder",
	BtnOpen:              "Open",
	NoOrgDirTitle:        "No source folder selected",
	PillUnselected:       "Unset",
	NoOrgDirSub:          "Select a folder with photos or videos (scanned recursively)",
	PillReady:            "Ready",
	PillUnavailable:      "Unavailable",
	PillWaitAuth:         "Authorize",
	AuthorizeHint:        "Tap 'Allow USB debugging' on your phone and check Always allow",
	NoAndroidDetected:    "No Android device detected",
	PillNoADB:            "No ADB",
	NoADBSubHint:         "adb not found — switch to Organize mode or pick an MTP Folder",
	PillDisconnected:     "Disconnected",
	DisconnectedSubHint:  "Plug in USB with Debugging on, or use Wi-Fi / Organize mode",
	ScanDefaultDirs:      "Scan default folders",
	ScanPrefix:           "Scan: ",
	DirPrefix:            "Path: ",
	NoDeviceModalTitle:   "No Other Devices",
	NoDeviceModalMsg:     "No other ADB devices or mounted folders found. Connect a phone or click MTP Folder.",

	TargetUnsetTitle:   "No target directory selected",
	PillUnset:          "Unset",
	TargetUnsetHint:    "Supports local folders or external drives",
	TargetUnsetSubHint: "Choose a destination folder on a local or external drive",
	BtnChooseTarget:    "Choose Target",
	BtnRecent:          "Recent",

	FilterPhotos:     "Photos",
	FilterVideos:     "Videos",
	FilterDeleteSrc:  "Move mode (delete source after verify)",
	ArchiveRuleHint:  "Layout: picture/YYYY/YYYY-MM · video/YYYY/YYYY-MM",
	BtnStartBackup:   "One-Click Backup",
	BtnStartOrganize: "Organize Now",
	BtnPreview:       "Dry Run",
	BtnStop:          "Stop",

	PhaseIdleDefault:  "Ready for one-click backup",
	PhasePreviewStart: "Running dry-run preview...",
	PhaseSyncStart:    "Starting archive sync...",
	PhaseIndexing:     "Verifying target meta index...",
	PhaseScanning:     "Scanning media files...",
	VerbSyncing:       "Archiving",
	VerbComparing:     "Checking",
	PhaseCancelled:    "Operation cancelled",
	MetricExported:    "Archived",
	MetricCached:      "Cached",
	MetricDedup:       "Deduplicated",
	MetricFailed:      "Failed",

	RecentActivityPrefix: "Latest: ",
	RecentActivityIdle:   "Latest: Idle",
	BtnViewLogs:          "Details",
	LogEmpty:             "No sync logs yet.",
	LogModalTitle:        "Sync Activity Log",
	BtnClose:             "Close",
	RefreshNoneLog:       "Refresh complete: no connected phone detected",
	DeviceStatusFmt:      "Device: %s (%s)",

	WirelessHint:          "Check IP & port under Developer Options → Wireless debugging on your phone.",
	WirelessPlaceholder:   "e.g., 192.168.1.50:5555",
	WirelessTitle:         "Wireless ADB Connect",
	BtnConnect:            "Connect",
	BtnCancel:             "Cancel",
	ADBNotFoundErr:        "adb binary not found; specify its path in Settings or install platform-tools",
	WirelessLogPrefix:     "Wireless ADB: ",
	MetaNotSelected:       "No target directory selected.\nSHA-256 index and sync history are stored inside <Target>/meta/ independently per drive.",
	MetaReadErrFmt:        "Failed to read target meta index: %v\nPath: %s",
	MetaSummaryFmt:        "Indexed files: %d · Cached records: %d\nMetadata path: %s",
	ADBAutoDetectFmt:      "Leave empty for auto-detect: %s",
	ADBManualPlaceholder:  "adb not found; enter binary path manually",
	BtnReindexMeta:        "Rebuild Target Meta Index",
	InfoTitle:             "Notice",
	SyncBusyReindexMsg:    "Backup or scan is running. Please wait until it finishes.",
	SelectTargetFirstMsg:  "Please select a target directory first",
	ReindexDoneFmt:        "Rebuilt target meta index: %d files",
	BtnClearHistory:       "Clear Incremental Cache",
	SyncBusyClearMsg:      "Backup or scan is running. Please wait until it finishes.",
	HistoryClearedLog:     "Cleared target incremental cache (next run will re-check meta/ hashes)",
	AboutLabelFmt:         "%s v%s · Author: %s · 100%% Offline",
	SettingsLanguageTitle: "Language / 界面语言",
	SettingsLangZH:        "简体中文",
	SettingsLangEN:        "English",
	SettingsDirsTitle:     "Phone Scan Folders (one path per line)",
	SettingsADBTitle:      "Custom ADB Path (optional)",
	SettingsMetaTitle:     "Target meta/ Hash Index & Cache",
	SettingsAboutTitle:    "About",
	SettingsTitle:         "Preferences",
	BtnSave:               "Save",
	ErrorTargetUnavailFmt: "Target unavailable: %s",
	ErrorPrefixFmt:        "Error: %v",
	DirEmptyErr:           "Path is empty",
	NoSourceErr:           "No phone or folder available; connect a device or choose an MTP/local folder",

	DescMTPRootLabel:      "Folder",
	DescOrganizeLabel:     "Source Folder",
	DescGVFSMounted:       "GVFS MTP mounted",
	DescCardDeviceSuffix:  " (Storage/Device)",
	DescCardCameraSuffix:  " (Card/Camera)",
	DescDCIMDetected:      "DCIM folder detected",
	DescDirNotExist:       "Folder missing or unmounted",
	DescDirReady:          "Ready",
	ADBStateConnected:     "Connected & Ready",
	ADBStateUnauthorized:  "Allow USB debugging on your phone",
	ADBStateOffline:       "Device offline; reconnect USB",
	ADBStateFmt:           "State: %s",
	VolNotSelected:        "No target selected",
	VolInvalidPathFmt:     "Invalid path: %v",
	VolExtUnmountedFmt:    "External drive not mounted: %s",
	VolMissingOrUnplugged: "Target missing or drive disconnected",
	VolNoAccessFmt:        "Cannot access target: %v",
	VolNotDir:             "Target path is not a folder",
	VolNotWritableFmt:     "Target not writable: %v",
	VolReady:              "Ready",
	VolReadyFreeFmt:       "Ready · %s free",

	EngineNoSource:           "No source device selected",
	EngineIndexingLogFmt:     "Building target meta index (%d/%d)...",
	EngineVerifyingMeta:      "Verifying target meta index...",
	EngineOpenMetaFailFmt:    "Open target meta index failed: %w",
	EngineReadMetaFailFmt:    "Read target meta index failed: %w",
	EngineScanTargetFailFmt:  "Scan target folder failed: %w",
	EngineUpdateMetaFailFmt:  "Update target meta index failed: %w",
	EngineReadHistFailFmt:    "Read target sync history failed: %w",
	EngineScanningSrcFmt:     "Scanning %s...",
	EngineScanSrcFailFmt:     "Scan source failed: %w",
	EngineNoMediaFound:       "No matching photos or videos found",
	EngineMkTempFailFmt:      "Create staging dir failed: %w",
	EnginePullFailFmt:        "Export failed [%s]: %v",
	EngineWriteMetaFailFmt:   "Write target meta index failed: %w",
	EngineWriteHistFailFmt:   "Write target sync history failed: %w",
	EngineDedupSkippedFmt:    "Deduplicated: %s (matches %s)",
	EngineMkTargetDirFailFmt: "Create target directory failed [%s]: %v",
	EngineWriteTargetFailFmt: "Write target file failed [%s]: %v",
	EngineCancelled:          "Backup cancelled",
	EngineDoneFmt:            "Completed: %d archived, %d cached, %d deduplicated, %d failed",
	EnginePreviewDoneFmt:     "Dry run complete: %d new (%s), %d cached, %d deduplicated",
	EngineStatTargetFailFmt:  "Inspect target path failed: %w",
	EngineSameDirErr:         "Target folder cannot be the same as source folder",
	EngineTargetInSrcErr:     "Target folder cannot be inside source folder",
	EngineMoveSrcInTargetErr: "In move mode, source folder cannot be inside target folder",
}
