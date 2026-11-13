package model

import "time"

// SourceKind identifies how a media source is connected.
type SourceKind string

const (
	SourceADB     SourceKind = "adb"
	SourceMounted SourceKind = "mounted"
)

// MediaKind classifies a media asset as photo or video.
type MediaKind string

const (
	MediaPhoto MediaKind = "photo"
	MediaVideo MediaKind = "video"
)

// MediaFilter controls which media types are included during scanning and syncing.
type MediaFilter struct {
	IncludePhotos bool
	IncludeVideos bool
}

// WorkMode selects between Android phone backup and local/external folder organization.
const (
	WorkModePhone    = "phone"
	WorkModeOrganize = "organize"
)

// AppConfig persists the user's remembered defaults and preferences.
type AppConfig struct {
	WorkMode             string     `json:"work_mode"`
	DefaultSourceKind    SourceKind `json:"default_source_kind"`
	DefaultSourceID      string     `json:"default_source_id"`
	DefaultSourceName    string     `json:"default_source_name"`
	OrganizeSourceDir    string     `json:"organize_source_dir"`
	OrganizeDeleteSource bool       `json:"organize_delete_source"`
	SourceDirs           []string   `json:"source_dirs"`
	IncludePhotos        bool       `json:"include_photos"`
	IncludeVideos        bool       `json:"include_videos"`
	TargetDir            string     `json:"target_dir"`
	RecentTargets        []string   `json:"recent_targets"`
	ThemeMode            string     `json:"theme_mode"` // "light", "dark", or "amoled"
	Language             string     `json:"language"`   // "zh" or "en"
	CustomADBPath        string     `json:"custom_adb_path"`
}

// DefaultConfig returns sensible out-of-the-box configuration defaults.
func DefaultConfig() AppConfig {
	return AppConfig{
		WorkMode:          WorkModePhone,
		DefaultSourceKind: SourceADB,
		SourceDirs: []string{
			"/sdcard/DCIM",
			"/sdcard/Pictures",
			"/sdcard/Movies",
		},
		IncludePhotos: true,
		IncludeVideos: true,
		ThemeMode:     "light",
		Language:      "zh",
	}
}

// AddRecentTarget records a target directory at the front of RecentTargets (max 5).
func (c *AppConfig) AddRecentTarget(dir string) {
	if dir == "" {
		return
	}
	updated := make([]string, 0, 5)
	updated = append(updated, dir)
	for _, existing := range c.RecentTargets {
		if existing != dir && existing != "" {
			updated = append(updated, existing)
			if len(updated) >= 5 {
				break
			}
		}
	}
	c.RecentTargets = updated
}

// MediaFile represents a single photo or video discovered on a source device.
type MediaFile struct {
	SourceID   string
	RemotePath string
	Name       string
	Size       int64
	ModTime    time.Time
	Kind       MediaKind
}

// TargetIndexEntry records an already-stored file inside a destination root's meta index.
type TargetIndexEntry struct {
	RelPath   string
	MediaKind MediaKind
	FileSize  int64
	SHA256    string
	ModTime   int64
}

// SyncHistoryEntry records a completed sync decision for instant incremental skipping
// inside the destination root's meta index.
type SyncHistoryEntry struct {
	SourceID      string
	RemotePath    string
	FileSize      int64
	RemoteModTime int64
	SHA256        string
	TargetRelPath string
	SyncedAt      int64
}

// SyncPhase represents the current stage of a backup or scan operation.
type SyncPhase string

const (
	PhaseIdle      SyncPhase = "idle"
	PhaseIndexing  SyncPhase = "indexing"
	PhaseScanning  SyncPhase = "scanning"
	PhaseSyncing   SyncPhase = "syncing"
	PhaseCompleted SyncPhase = "completed"
	PhaseCancelled SyncPhase = "cancelled"
)

// SyncStats aggregates real-time counters for a sync or dry-run scan.
type SyncStats struct {
	TotalScanned  int
	Exported      int
	SkippedCached int
	SkippedDedup  int
	Failed        int
	BytesCopied   int64
	BytesTotal    int64
	NewCandidates int // Populated in dry-run preview mode
}

// SyncProgress carries live status updates from the sync engine to the UI.
type SyncProgress struct {
	Phase       SyncPhase
	Current     int
	Total       int
	CurrentFile string
	BytesPerSec float64
	Stats       SyncStats
	LogLine     string
}
