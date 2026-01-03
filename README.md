# README.md

# deej-updated

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
