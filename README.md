# deej-updated

A portable Windows mixer for a physical deej slider controller. The Go tray engine reads serial positions and controls Windows audio sessions. A native Settings app edits connection options and mappings without requiring YAML knowledge.

## Run

1. Download a Windows ZIP from [Releases](https://github.com/Nated4wgy/deej-updated/releases). Development releases are marked prerelease.
2. Extract the whole ZIP to a writable folder. Keep **deej.exe** and **deej-settings.exe** together.
3. Run **deej-settings.exe**, choose the COM port and firmware's baud rate, configure sliders, and save.
4. Run **deej.exe**. Open Settings from its tray icon whenever needed. Offline editing also works.

Both programs use config.yaml beside the executable, including at Windows login. The startup toggle registers the mixer for the current user. Use a writable portable folder, rather than Program Files. An explicit **--config C:/path/config.yaml** is supported; keep the filename config.yaml.

Exit the old mixer before trying the new one: only one program can open the controller. For an initial trial, copy the old config into the new folder, leaving the old installation intact. No firmware upgrade is required; 9600 baud remains supported.

## Settings

- One or more targets per slider; separate groups with semicolons in the UI.
- Choose discovered apps/devices or type an executable name for an app that is closed.
- Built-ins: **master** (default output), **system**, **mic** (default input), **deej.current** (foreground app), **deej.unmapped** (unassigned app sessions).
- Named devices use their full Windows friendly name. Matching is case insensitive.
- Invert direction and choose low/default/high noise reduction. High filtering requires larger movements and can feel stepped.
- Live position, effective mappings and received-sample-to-volume-call timing appear while the mixer runs. Timing excludes device sampling, USB transit and audible output delay.
- Save backs up the previous file to config.yaml.bak, validates and atomically replaces the file. Invalid external edits retain working settings.
- Advanced YAML opens Notepad. Reopen Settings after editing externally to avoid overwriting concurrent changes.

Original scalar and list mappings remain supported:

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

YAML indexes start at zero; UI labels start at one. Supports up to 32 sliders. Legacy mappings in logs/preferences.yaml are merged; the UI shows effective mappings. Move wanted legacy targets into the main config and remove the legacy mapping file if you want the UI to be the sole mapping source.

## Response and reliability

Serial ingestion no longer waits for Windows audio calls. Pending movement is coalesced to the newest value per slider. Audio interfaces have one persistent initialized owner thread. Discovery, volume writes and release are serialized. Device notifications signal the worker, and failed discovery retains usable targets. Serial reconnect uses bounded backoff; shutdown uses a bounded read timeout.

Sessions are also refreshed every five seconds to find new apps, and physical positions are reapplied afterwards. Physical sliders are authoritative: a Windows mixer adjustment can be overwritten at the next refresh. Foreground targeting uses a short cache and remains experimental for unusual child-process audio routing.

These changes remove known backlog and lifetime hazards. Physical-to-audible response still needs validation with your controller and audio devices. Synthetic timings are not hardware latency measurements.

## Build Windows amd64

Install Go 1.27.3 (CI baseline), Git, and put Go on PATH. The module declares Go 1.24 as its minimum language version. Production builds require no C compiler; race detection requires GCC and CGO.

~~~powershell
git clone https://github.com/Nated4wgy/deej-updated.git
cd deej-updated
./scripts/build_windows.ps1
~~~

The script installs pinned github.com/akavel/rsrc@v0.10.2, regenerates amd64 resources from the tracked icon/manifest, verifies modules, runs tests/vet and builds both apps into bin/. Resources are embedded; manifest sidecars are unnecessary. An existing output config is never overwritten.

The bundle includes BUILD-INFO.txt (source commit/toolchain), SHA256SUMS.txt, sample config and license. Use an ordinary writable temp directory if a restricted environment prevents file replacement. For race checks use **CGO_ENABLED=1 go test -race ./pkg/deej/...** with GCC available.

## Recovery

Windows GitHub Actions checks pushes to master and the development branch, and pull requests. Every successful push creates a durable prerelease ZIP tied to the exact source commit. Tagged v* releases are normal releases. CI artifacts are also retained for 90 days; release assets remain beyond that unless deleted.

Recover by downloading a release ZIP, or checking out its tag/commit and running the build script. Keep your own configuration backup: personal config and logs are not uploaded. See [validation](docs/validation.md) and [roadmap](docs/roadmap.md).

## License

MIT; see [LICENSE](LICENSE). Based on [deej by Omri Harel](https://github.com/omriharel/deej). The original Linux backend remains in the tree; native Settings and release checks target Windows amd64. Linux runtime behaviour is not validated by this release.
