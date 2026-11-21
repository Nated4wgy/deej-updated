# Roadmap

## Implemented in the responsive-audio/settings branch

- Architecture-correct generated Windows resources and one portable build script.
- Bounded latest-value delivery and validated incremental frame decoding.
- Reconnecting serial owner with bounded read timeout and shutdown.
- Persistent Windows MTA audio owner; serialized discovery/application/release.
- Atomic device-change signals, fewer process lookups and error-path cleanup.
- Typed config snapshots, scalar/list compatibility, validated atomic saves/backups and replacement-aware watching.
- Native Settings with offline editing, connection/noise options, grouped/discovered targets, live positions/effective mappings, save acknowledgement, startup toggle and restore.
- Regression tests and automated GitHub recovery bundles.

## Hardware acceptance before stable release

1. Compare master, one app, groups and foreground targeting with repeatable physical movements.
2. Record p50/p95/p99 and final settling, after idle, during discovery and with unavailable targets. Separate received-frame timing from physical-to-audible response.
3. Exercise USB unplug/replug, sleep/resume, default-device swaps, app launch/exit, no microphone and no output.
4. Verify save while moving sliders, login startup, high DPI, keyboard navigation and backup recovery.
5. Run a multi-hour soak for handles, memory and reconnect behaviour.

## Follow-up

- Session-created notifications and incremental discovery to reduce full scans further.
- Stable device identifiers and profiles, with friendly-name import compatibility.
- Per-slider calibration/filter settings based on measured noise.
- If serial transit dominates, a separately versioned higher-baud firmware package; never change one end silently.
- Signed releases and a reviewed updater when distribution warrants the signing cost.

Preserve source commits and successful ZIPs together. Do not infer hardware response from synthetic tests.
