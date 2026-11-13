# Changelog

All notable changes to **Lumina Amber** will be documented in this file.

## [1.0.0] - 2026-11-13

### Added
- **Bilingual Chinese / English UI (`internal/i18n`)**: Full localization across the main window, Settings, wireless ADB dialog, sync engine progress logs, and device/volume status pills with live switching (`EN` / `中文` header toggle and Settings selector) persisted in SQLite.

### Fixed
- **GitHub Actions Linux GUI Build**: Added `libwayland-dev` and `libxkbcommon-dev` dependencies required by Fyne v2.8 / GLFW 3.4 on `ubuntu-latest`.

## [0.9.0] - 2026-11-13

### Added
- **Portable Per-Target Metadata Store (`<Target>/meta/index.db`)**: Moved SHA-256 content index and sync history from the global app database into each backup target directory, making external drives portable across machines and fixing backups to newly created target directories.
- **Separated Media Year/Month Hierarchy**: Organizes photos into `picture/YYYY/YYYY-MM/` and videos into `video/YYYY/YYYY-MM/`.
- **Folder Import & Organize Mode**: Added a dedicated mode to scan any local or external folder and copy or safely move media into the unified archive layout.
- **macOS / iOS-Inspired Modern UI & Three Themes**: Redesigned UI in Fyne v2 supporting **macOS Light**, **macOS Dark**, and **AMOLED Pure Black**, with one-click remembered defaults.
- **Author & Version Identification (`v0.9.0`, `eddy`)**: Added `internal/version`, `-version` CLI flag, UI version pill & footer, Settings about panel, and synced `Makefile` / `dist/Info.plist`.
- **GitHub CI/CD Workflows**: Added `ci.yml` (Go vet, `-race` tests, Linux & macOS Universal builds) and `release.yml` (tag-triggered GitHub Releases with SHA-256 checksums).

### Performance
- **Parallel Hashing & Indexing**: Concurrent target index backfilling (`2..8` workers) and bounded worker pool for local/organize sources.
- **`sync.Pool` 1 MiB I/O Buffers**: Reduced allocations during SHA-256 hashing and file copying.
- **Single-Pass Copy + Hash (`io.MultiWriter`)**: Eliminated second read passes on newly copied files.
- **Batched SQLite Transactions + O(1) Snapshots**: 64-item transactional upserts paired with deterministic ordered commits (`maxReorderWindow = 16`).

### Fixed
- Eliminated background device-polling data races, periodic disk write probes in target health checks, and Dry-Run index mutations.
- Fixed Move-mode safety so source files are only deleted after target DB batch commits succeed and target file sizes verify.
- Improved EXIF/MP4 metadata parsing (`.cr3` support, short `mvhd` box handling, calendar date validation) and ADB/macOS `/Volumes` discovery.
