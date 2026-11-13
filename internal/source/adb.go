package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"lumina-amber/internal/i18n"
	"lumina-amber/internal/model"
)

// ADBDevice represents a single device entry returned by `adb devices -l`.
type ADBDevice struct {
	Serial string
	State  string // "device", "unauthorized", "offline"
	Model  string
	USB    string
	IsWiFi bool
}

// FindADBBinary locates the adb executable across standard Linux and macOS paths.
func FindADBBinary(customPath string) string {
	if customPath != "" {
		if info, err := os.Stat(customPath); err == nil && !info.IsDir() {
			return customPath
		}
	}

	if p, err := exec.LookPath("adb"); err == nil {
		return p
	}

	home, _ := os.UserHomeDir()
	var candidates []string
	if runtime.GOOS == "darwin" {
		candidates = []string{
			filepath.Join(home, "Library", "Android", "sdk", "platform-tools", "adb"),
			"/opt/homebrew/bin/adb",
			"/usr/local/bin/adb",
		}
	} else {
		candidates = []string{
			filepath.Join(home, "Android", "Sdk", "platform-tools", "adb"),
			"/usr/lib/android-sdk/platform-tools/adb",
			"/usr/bin/adb",
			"/usr/local/bin/adb",
		}
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// ListADBDevices queries connected Android devices using `adb devices -l`.
func ListADBDevices(ctx context.Context, adbBin string) ([]SourceDescriptor, error) {
	if adbBin == "" {
		return nil, errors.New("adb binary not found")
	}

	cmd := exec.CommandContext(ctx, adbBin, "devices", "-l")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("adb devices: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	devices := ParseADBDevices(string(out))
	descs := make([]SourceDescriptor, 0, len(devices))
	for _, dev := range devices {
		descs = append(descs, dev.ToDescriptor())
	}
	return descs, nil
}

// ConnectWirelessADB connects to a device over TCP/IP (`adb connect <ip:port>`).
func ConnectWirelessADB(ctx context.Context, adbBin, address string) (string, error) {
	if adbBin == "" {
		return "", errors.New("adb binary not found")
	}
	addr := strings.TrimSpace(address)
	if addr == "" {
		return "", errors.New("address cannot be empty")
	}
	if !strings.Contains(addr, ":") {
		addr += ":5555"
	}
	cmd := exec.CommandContext(ctx, adbBin, "connect", addr)
	out, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		return msg, fmt.Errorf("adb connect %s: %w (%s)", addr, err, msg)
	}
	if strings.Contains(strings.ToLower(msg), "failed") || strings.Contains(strings.ToLower(msg), "unable") {
		return msg, errors.New(msg)
	}
	return msg, nil
}

// ParseADBDevices parses `adb devices -l` output into structured ADBDevice entries.
func ParseADBDevices(output string) []ADBDevice {
	var devices []ADBDevice
	lines := strings.Split(output, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "List of devices") || strings.HasPrefix(line, "*") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		dev := ADBDevice{
			Serial: fields[0],
			State:  fields[1],
			IsWiFi: strings.Contains(fields[0], ":") || strings.Contains(fields[0], "._adb"),
		}

		for _, kv := range fields[2:] {
			if strings.HasPrefix(kv, "model:") {
				dev.Model = strings.ReplaceAll(strings.TrimPrefix(kv, "model:"), "_", " ")
			} else if strings.HasPrefix(kv, "usb:") {
				dev.USB = strings.TrimPrefix(kv, "usb:")
			}
		}
		devices = append(devices, dev)
	}
	return devices
}

// ToDescriptor converts an ADBDevice into a UI-friendly SourceDescriptor.
func (d ADBDevice) ToDescriptor() SourceDescriptor {
	name := d.Model
	if name == "" {
		name = d.Serial
	}
	connType := "USB"
	if d.IsWiFi {
		connType = "Wi-Fi"
	}

	desc := SourceDescriptor{
		ID:   "adb:" + d.Serial,
		Name: fmt.Sprintf("%s (%s)", name, connType),
		Kind: model.SourceADB,
	}

	s := i18n.S()
	switch d.State {
	case "device":
		desc.State = StateReady
		desc.StatusMessage = s.ADBStateConnected
	case "unauthorized":
		desc.State = StateUnauthorized
		desc.StatusMessage = s.ADBStateUnauthorized
	case "offline":
		desc.State = StateOffline
		desc.StatusMessage = s.ADBStateOffline
	default:
		desc.State = StateUnavailable
		desc.StatusMessage = fmt.Sprintf(s.ADBStateFmt, d.State)
	}
	return desc
}

// ADBSource implements MediaSource over the adb CLI.
type ADBSource struct {
	adbBin string
	serial string
	desc   SourceDescriptor
}

// NewADBSource creates an ADBSource for the given descriptor and adb binary path.
func NewADBSource(adbBin string, desc SourceDescriptor) *ADBSource {
	serial := strings.TrimPrefix(desc.ID, "adb:")
	return &ADBSource{
		adbBin: adbBin,
		serial: serial,
		desc:   desc,
	}
}

func (s *ADBSource) Descriptor() SourceDescriptor {
	return s.desc
}

// Scan queries all target phone directories in a single fast shell invocation.
func (s *ADBSource) Scan(ctx context.Context, dirs []string, filter model.MediaFilter) ([]model.MediaFile, error) {
	if s.adbBin == "" {
		return nil, errors.New("adb binary not configured")
	}
	if len(dirs) == 0 {
		dirs = model.DefaultConfig().SourceDirs
	}

	var quotedDirs []string
	for _, d := range dirs {
		d = strings.TrimSpace(d)
		if d != "" {
			quotedDirs = append(quotedDirs, posixShellQuote(d))
		}
	}
	if len(quotedDirs) == 0 {
		return nil, nil
	}

	// Single-shot POSIX shell loop using toybox find + stat to scan thousands of files in < 1s.
	// Using "$d/." follows symlinks such as /sdcard -> /storage/self/primary.
	script := fmt.Sprintf(
		`for d in %s; do if [ -d "$d" ]; then find "$d/." -type f ! -path '*/.*' ! -name '.*' ! -name '.trashed-*' -exec stat -c '%%s|%%Y|%%n' {} + 2>/dev/null; fi; done`,
		strings.Join(quotedDirs, " "),
	)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, s.adbBin, "-s", s.serial, "shell", script)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// If stdout has valid entries despite partial permission warnings, process them; otherwise error.
		if stdout.Len() == 0 {
			return nil, fmt.Errorf("adb scan failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
		}
	}

	return ParseADBScanOutput(s.desc.ID, stdout.String(), filter), nil
}

// ParseADBScanOutput parses `size|epoch_mtime|remote_path` lines into MediaFile items.
func ParseADBScanOutput(sourceID, output string, filter model.MediaFilter) []model.MediaFile {
	lines := strings.Split(output, "\n")
	files := make([]model.MediaFile, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))

	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r\n")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}

		size, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil || size <= 0 {
			continue
		}
		epochSec, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil {
			continue
		}
		remotePath := strings.Replace(parts[2], "/./", "/", 1)
		if _, dup := seen[remotePath]; dup {
			continue
		}
		seen[remotePath] = struct{}{}

		kind, ok := MatchesFilter(remotePath, filter)
		if !ok {
			continue
		}

		files = append(files, model.MediaFile{
			SourceID:   sourceID,
			RemotePath: remotePath,
			Name:       filepath.Base(remotePath),
			Size:       size,
			ModTime:    time.Unix(epochSec, 0),
			Kind:       kind,
		})
	}
	return files
}

// PullFile pulls a file from the Android device to localDest.
func (s *ADBSource) PullFile(ctx context.Context, file model.MediaFile, localDest string) error {
	cmd := exec.CommandContext(ctx, s.adbBin, "-s", s.serial, "pull", file.RemotePath, localDest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("adb pull %q: %w (%s)", file.RemotePath, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// RemoteSHA256 computes SHA-256 directly on the Android device via toybox sha256sum
// to skip pulling duplicate large files over USB.
func (s *ADBSource) RemoteSHA256(ctx context.Context, file model.MediaFile) (string, bool) {
	script := "sha256sum " + posixShellQuote(file.RemotePath) + " 2>/dev/null"
	cmd := exec.CommandContext(ctx, s.adbBin, "-s", s.serial, "shell", script)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 || len(fields[0]) != 64 {
		return "", false
	}
	return strings.ToLower(fields[0]), true
}

func posixShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
