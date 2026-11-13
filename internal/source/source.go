package source

import (
	"context"
	"path/filepath"
	"strings"

	"lumina-amber/internal/model"
)

// SourceState describes the readiness of a discovered device or volume.
type SourceState string

const (
	StateReady        SourceState = "ready"
	StateUnauthorized SourceState = "unauthorized"
	StateOffline      SourceState = "offline"
	StateUnavailable  SourceState = "unavailable"
)

// SourceDescriptor provides display and identity metadata for a selectable source.
type SourceDescriptor struct {
	ID            string
	Name          string
	Kind          model.SourceKind
	State         SourceState
	StatusMessage string
	RootPath      string // Populated for mounted directory sources
	ScanWholeRoot bool   // When true, recursively scan RootPath instead of Android subdir mapping
}

// MediaSource abstracts Android ADB devices and mounted directories/MTP shares.
type MediaSource interface {
	Descriptor() SourceDescriptor
	Scan(ctx context.Context, dirs []string, filter model.MediaFilter) ([]model.MediaFile, error)
	PullFile(ctx context.Context, file model.MediaFile, localDest string) error
	RemoteSHA256(ctx context.Context, file model.MediaFile) (string, bool)
}

var photoExts = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".heic": {}, ".heif": {},
	".webp": {}, ".dng": {}, ".raw": {}, ".arw": {}, ".cr2": {},
	".cr3": {}, ".nef": {}, ".orf": {}, ".rw2": {}, ".gif": {},
	".bmp": {}, ".tif": {}, ".tiff": {},
}

var videoExts = map[string]struct{}{
	".mp4": {}, ".mov": {}, ".m4v": {}, ".3gp": {},
	".mkv": {}, ".avi": {}, ".webm": {},
}

// ClassifyMedia checks whether filename is a supported photo or video.
func ClassifyMedia(filename string) (model.MediaKind, bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := photoExts[ext]; ok {
		return model.MediaPhoto, true
	}
	if _, ok := videoExts[ext]; ok {
		return model.MediaVideo, true
	}
	return "", false
}

// MatchesFilter returns true if the filename passes ignore rules and the active MediaFilter.
func MatchesFilter(fullPath string, filter model.MediaFilter) (model.MediaKind, bool) {
	if ShouldIgnorePath(fullPath) {
		return "", false
	}
	kind, ok := ClassifyMedia(fullPath)
	if !ok {
		return "", false
	}
	if kind == model.MediaPhoto && !filter.IncludePhotos {
		return "", false
	}
	if kind == model.MediaVideo && !filter.IncludeVideos {
		return "", false
	}
	return kind, true
}

// ShouldIgnorePath filters hidden system directories, Android trash, and thumbnail caches.
func ShouldIgnorePath(fullPath string) bool {
	norm := filepath.ToSlash(fullPath)
	parts := strings.Split(norm, "/")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, ".") {
			return true
		}
		if p == "Android" && i+1 < len(parts) && parts[i+1] == "data" {
			return true
		}
		if p == "@eaDir" || p == "Thumbs.db" {
			return true
		}
	}
	base := filepath.Base(norm)
	if strings.HasPrefix(base, ".trashed-") {
		return true
	}
	return false
}
