# Changelog

Notable user-facing changes to the Windows 11 edition of deej.

## [1.0.0] - 2026-11-21

### Added
- Modern unified Settings dashboard inside `deej.exe`, with live physical slider positions, searchable targets, device preferences, diagnostics, and light/dark themes.
- Single-app startup modes (`--background`, `--settings`, `--check-config`) and per-user Windows startup support.
- Windows 11 USB serial sleep/wake notifications, deferred reconnect and recovery of stalled serial frames.
- **Reconnect controller** tray action and automatic retry of unavailable COM ports.
- Atomic configuration writes, backups, restore and external-edit conflict detection.
- Branded Windows icon matching the dark/teal dashboard and refreshed project visuals.

### Improved
- Low-latency slider handling that keeps only the newest pending slider value.
- Dedicated Windows audio-session ownership to avoid conflicting volume updates and session teardown.
- More resilient session discovery, configuration reload and process mapping.
- Developer test suite, Windows CI build, dashboard regression tests and race detection.
- Portable release packaging with SHA-256 checksums and build provenance.

### Kept compatible
- Existing `config.yaml` scalar or list slider mappings and supported deej controller firmware.
- Master, system, mic, current-window, unmapped and application session mappings.
- Original upstream project attribution and MIT license.

### Upgrade notes
- This release uses **one `deej.exe`**; the separate legacy `deej-settings.exe` is no longer required.
- Extract into a new folder, copy your existing `config.yaml`, and test the COM port, slider mappings, sleep/wake recovery and Settings before replacing your old installation.
- Microsoft Edge WebView2 Runtime is required for the embedded dashboard (not for background mixer operation).
- Windows USB devices that re-enumerate under a new COM number still require a port selection update.

## Earlier development builds

Initial Windows 11 compatibility and separate Settings companion application were followed by unified WebView2 settings, responsive audio and serial recovery. Earlier development tags remain in GitHub Releases for reference.
