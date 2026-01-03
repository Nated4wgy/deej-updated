# CHANGELOG.md

## deej-win11 release clean

This changelog covers the major changes made while updating deej for Windows 11, adding a Settings app, and stabilising runtime behaviour. It focuses on user visible and reliability changes, not every small refactor.

## [Win11 Clean Release] - 2026-01-03

### Added
- Windows Settings companion app (deej-settings.exe)
  - Simple UI for editing all common config options
  - Slider mapping editor with dropdown target types plus normal exe entry for apps
  - Advanced YAML editor for full control of config.yaml
- Settings app window icon
  - Ships a dedicated deej-settings.ico and uses it instead of the default Windows icon
- Start with Windows support
  - Toggle is available from the tray and the Settings UI
  - Uses HKCU Run entry so it works per user without admin rights
- Windows GUI reliability fix for Settings app startup
  - Ships an external application manifest (deej-settings.exe.manifest) so Walk loads common controls v6
  - Prevents the Walk startup failure TTM_ADDTOOL failed
- Safer config reload while sliders are moving
  - Prevents the main app from crashing when config.yaml is saved and sessions are refreshed

### Improved
- Windows startup behaviour
  - Main app resolves working directory to the folder that contains deej.exe to avoid config path issues when launched at login
- Windows COM session handling
  - More tolerant COM initialisation and session discovery behaviour on Windows 11
- Serial behaviour
  - Improved stability around serial disconnect and reconnect scenarios

### Changed
- Config slider mapping compatibility
  - Accepts either scalar values or lists for slider_mapping entries
  - Example: `0: master` and `0: [master]` both work
- Release packaging workflow
  - build_windows.ps1 builds both executables and copies required sidecar files into the output directory

### Removed
- Debug logging bloat from the tray build
  - Removed verbose debug tray options
  - No automatic logs folder creation during normal usage

## Notes
- If you run the GUI builds without manifests next to the executables, Windows can fall back to older common controls and the Settings app may fail to start.
- If you need troubleshooting logs in the future, build without `-H=windowsgui` to get a console and stderr output.
md
Copy code
# README.md

# deej-win11

A Windows 11 focused fork of deej with a companion Settings app. The main mixer behaviour remains the same: an Arduino (or compatible microcontroller) sends slider values over serial and the Windows client applies volume changes to audio sessions.

This fork adds a clean Settings UI, improves Windows 11 reliability, and includes a simple release workflow for building portable binaries.

## What is included

- deej.exe
  - The main Windows tray application that reads the serial device and controls volume
- deej-settings.exe
  - A companion Settings UI for editing config.yaml and slider mappings
- Sidecar files for Windows GUI correctness and polish
  - deej.exe.manifest
  - deej-settings.exe.manifest
  - deej-settings.ico

## Major improvements in this fork

- Windows 11 friendly startup behaviour
  - deej runs correctly when started at login and can still find config.yaml next to the exe
- Settings app
  - General settings editor
  - Slider mapping editor with dropdown target types and app exe entry
  - Advanced YAML editor for full config control
- Start with Windows toggle
  - Available from tray and Settings UI
- Stability fix when saving settings
  - Saving config.yaml from the Settings app no longer crashes deej during session refresh

## Quick start

1. Build or download the binaries into a folder.
2. Put config.yaml in the same folder as deej.exe.
3. Run deej.exe.
4. Open the Settings app from the tray menu.

## Building on Windows 11

### Prerequisites

- Go installed
- A working C toolchain for Windows builds that use systray (CGO)
  - A common approach is MSYS2 with mingw-w64 GCC installed
  - If your environment already builds getlantern systray projects, you are fine

### Build using the included script

From the repo root:

```powershell
.\scripts\build_windows.ps1
