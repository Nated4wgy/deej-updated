# deej for Windows 11 · v1.0.0

The first stable release of the modern, unified deej Windows 11 edition.

## Highlights
- **One app**: the tray mixer and a polished Settings dashboard are bundled in one `deej.exe`.
- **Responsive mixing**: latest slider values win, while Windows audio work stays separate from UI rendering.
- **Better Windows recovery**: reconnect after sleep, retry unavailable COM ports, recover stalled serial streams, or reconnect manually from the tray.
- **Easy configuration**: select application/device targets, see live positions and diagnostics, and safely save or restore your YAML.
- **Refreshed branding**: a teal/charcoal icon aligned with the Windows dashboard.
- **Portable**: extract ZIP and run, no installer or separate Settings executable.

## Before you upgrade
1. Keep a copy of your existing installation and `config.yaml`.
2. Extract the attached `deej-windows-amd64.zip` into a **new writable folder**.
3. Copy your existing `config.yaml` into the new folder.
4. Close older deej instances, launch the new `deej.exe`, and check your COM port.
5. Test moving sliders, saving Settings and Windows sleep/wake recovery.

The dashboard uses Microsoft Edge WebView2 Runtime (normally included with Windows 11). The physical slider controller and its deej-compatible firmware are not included in the ZIP.

**Validation:** Windows build, Go tests/vet, dashboard regression checks and race tests run automatically before publishing. Sleep/wake has been tried on one user's setup; USB controller/driver behaviour may vary.

Project: https://github.com/Nated4wgy/deej-updated
Upstream: https://github.com/omriharel/deej
