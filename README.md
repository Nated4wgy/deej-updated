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

This builds into .\bin\ by default and copies the manifest files plus the Settings icon into the output folder.

Build manually
mkdir bin -Force | Out-Null

go mod tidy

go build -ldflags "-H=windowsgui" -o .\bin\deej.exe .\pkg\deej\cmd
go build -ldflags "-H=windowsgui" -o .\bin\deej-settings.exe .\pkg\deej\cmd\settings


Copy required sidecar files next to the executables:

Copy-Item .\assets\manifests\deej.exe.manifest .\bin\deej.exe.manifest -Force
Copy-Item .\assets\manifests\deej-settings.exe.manifest .\bin\deej-settings.exe.manifest -Force
Copy-Item .\assets\icons\deej-settings.ico .\bin\deej-settings.ico -Force


Important: the Settings app relies on the manifest being next to the exe on some Windows setups. If it is missing, the app can fail to start with a Walk tooltip initialisation error.

If Go refuses to build because go.sum is missing

If you see errors about missing go.sum entries, run:

$env:GOFLAGS=""
go mod tidy


If your machine has GOFLAGS set to readonly module mode globally, clear it for the build session as above.

Configuration

The main config file is config.yaml and should live next to deej.exe.

Slider mapping format

This fork supports both formats:

Scalar form:

slider_mapping:
  0: master
  1: chrome.exe


List form:

slider_mapping:
  0:
    - master
  1:
    - chrome.exe
    - spotify.exe

Special targets

Common built in targets used by deej include:

master

system

mic

deej.current

deej.unmapped

For controlling applications, use the process name, for example chrome.exe, spotify.exe, discord.exe.

Settings app

deej-settings.exe can edit:

General serial options (COM port, baud)

Common behaviour options (invert sliders, noise reduction)

Slider mapping entries with a friendly editor

Any other YAML keys through the Advanced YAML editor

The Settings window icon is loaded from deej-settings.ico next to the exe. You can replace this file with your own ico if you want custom branding.

Release folder layout

A portable release folder typically contains:

deej.exe

deej.exe.manifest

deej-settings.exe

deej-settings.exe.manifest

deej-settings.ico

config.yaml

Optional:

preferences.yaml (if used in your build)

License and attribution

This repository remains under the MIT License, same as upstream deej. See LICENSE.

Upstream project: deej by Omri Harel.
This fork focuses on Windows 11 build and usability improvements plus a companion Settings application.


If you want, I can also add a short `RELEASE.md` that is literally a checklist for packaging a zip release (copy these files, test steps, and a quick sanity checklist like “move slider, save config, confirm app does not exit”).
::contentReference[oaicite:0]{index=0}

