# deej-updated

One Windows app for your physical audio sliders. **deej.exe** contains the responsive mixer and a modern, local settings dashboard. No companion settings executable or external interface assets are required.

## Run

1. Download a Windows ZIP from [Releases](https://github.com/Nated4wgy/deej-updated/releases) and extract it into a writable folder.
2. Exit any older mixer or serial monitor using the controller.
3. Run **deej.exe**. The mixer starts in the tray and opens the dashboard. Configure your COM port, matching firmware baud rate and slider targets, then save.
4. Close the window whenever you like; the mixer keeps running. Open it again from the tray's Settings menu or by opening deej.exe again.

The startup toggle runs quietly in the tray at Windows login. The dashboard uses WebView2 Runtime, normally included with Windows 11. If it is missing, install Microsoft's Evergreen WebView2 Runtime; the background mixer does not depend on the interface being open. The runtime is already present on the development machine. The app loads embedded local content and blocks web-resource requests; it does not serve a local website or need an Internet connection for settings.

For an upgrade, use a fresh folder and copy your existing config.yaml into it. Keep the previous release available until you have tried the new build. Your YAML and firmware remain compatible; no baud-rate or noise-setting changes are made automatically.

## Dashboard

- Live slider cards show physical positions. Add one or more targets with the searchable picker; targets appear as removable chips.
- Choose discovered apps/devices, system controls, or type an app executable for an application that is not running.
- Device & preferences contains serial settings, noise filtering, direction inversion and Windows startup.
- Diagnostics shows the connection, discovered audio targets and recent received-sample-to-volume-call timing. This excludes controller/USB transit and audible playback delay.
- Light and dark themes are remembered locally. Keyboard navigation and Ctrl+S are supported.
- Saving validates the config, preserves unrelated YAML, creates config.yaml.bak and atomically replaces the file. Invalid edits keep the mixer's last good configuration.
- Advanced YAML opens Notepad. Use Reload configuration afterwards. Concurrent edits are detected rather than silently overwritten. Restore backup can recover even a corrupt current file, retaining it as config.yaml.before-restore.

Built-in targets: **master**, **system**, **mic**, **deej.current** (foreground app when moved), **deej.unmapped** (unassigned app sessions). Named devices use the full Windows friendly name, matched case insensitively. Supports up to 32 sliders.

Original scalar/list YAML mappings still work:

~~~yaml
slider_mapping:
  0: master
  1: [discord.exe, firefox.exe]
  2: deej.current
com_port: COM4
baud_rate: 9600
invert_sliders: false
noise_reduction: default
~~~

YAML indexes start at zero; UI labels start at one. Legacy mappings in logs/preferences.yaml are still merged. Move wanted legacy targets into config.yaml and remove the legacy mapping file if the UI should be the only mapping source.

## Single executable, independent audio work

The same executable has two runtime roles. The tray mixer owns serial/audio work. Its interface runs in UI-only mode in a separate process of **the same deej.exe**, so rendering and editing do not block slider processing. WebView2 also uses its standard Windows helper processes. Closing the interface releases those resources without stopping the mixer. Reopening reuses the existing interface for that config.

Launch options:

~~~text
deej.exe                       Start mixer and show interface
deej.exe --background          Start mixer in tray only
deej.exe --settings            Open interface without starting another mixer
deej.exe --check-config        Validate config and exit
~~~

Add **--config C:/path/config.yaml** for an explicit configuration location. The default is config.yaml beside the executable, including at login. Use a writable portable folder rather than Program Files.

Serial ingestion retains only the latest pending value per slider. A persistent Windows audio owner serializes discovery, volume writes and COM release. The previously validated audio pipeline is retained. Sessions refresh every five seconds and physical positions are reapplied; a stationary foreground slider remains bound to its last moved target. High noise reduction still requires larger movements and can feel stepped.

## Build and tests

Windows amd64 is the release target. Install Go 1.27.3 (CI baseline) and Git; put Go on PATH. Production builds need no C compiler. Race detection needs GCC/CGO. Node 22+ and npm are used only for interface regression tests, not to run the app.

~~~powershell
git clone https://github.com/Nated4wgy/deej-updated.git
cd deej-updated
./scripts/build_windows.ps1
npm ci
npm test
~~~

The script installs pinned rsrc v0.10.2, generates amd64 icon/manifest resources, verifies dependencies, runs Go tests/vet and builds **bin/deej.exe**. An existing output config is not overwritten. The interface is embedded; no HTML/CSS/JS files, DLL sidecar or settings EXE must be copied alongside it. UI tests use installed Microsoft Edge headlessly with fixture data, separate from user browser profiles and the controller.

Successful GitHub push builds run all checks, including race and UI tests, then preserve a source-linked ZIP as a prerelease. Tagged v* releases are normal releases. Artifacts are also kept for 90 days; release assets remain recoverable beyond artifact expiry unless deleted. Each bundle includes source/toolchain identity, binary SHA-256 checksums, license notices and a sample config. Personal configuration and logs are not uploaded.

See [validation](docs/validation.md), [unified app notes](docs/unified-app.md) and [roadmap](docs/roadmap.md). The original Linux backend remains in source; this interface and the release validation target Windows amd64.

MIT; see [LICENSE](LICENSE) and THIRD-PARTY-NOTICES.txt. Based on [deej by Omri Harel](https://github.com/omriharel/deej).
