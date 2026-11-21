//go:build windows

package deej

import _ "embed"

// This icon is also embedded into the Windows EXE by scripts/build_windows.ps1.
// Keep trayIcon and pkg/deej/assets/logo.ico in sync so the taskbar/tray and
// executable use the same branding on Windows 11.
//go:embed assets/logo.ico
var trayIcon []byte
