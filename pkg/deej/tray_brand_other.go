//go:build !windows

package deej

import "github.com/omriharel/deej/pkg/deej/icon"

// Preserve the upstream tray icon on Linux; Windows uses a new .ico resource.
var trayIcon = icon.DeejLogo
