module github.com/omriharel/deej

// Windows CI uses Go 1.27.3.
go 1.24.0

require (
	github.com/fsnotify/fsnotify v1.9.0
	github.com/gen2brain/beeep v0.11.2
	github.com/getlantern/systray v1.2.2
	github.com/go-ole/go-ole v1.3.0
	github.com/jchv/go-webview2 v0.0.0-20260205173254-56598839c808
	github.com/jfreymuth/pulse v0.0.0-20200608153616-84b2d752b9d4
	github.com/lxn/walk v0.0.0-20210112085537-c389da54e794
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e
	github.com/mitchellh/go-ps v1.0.0
	github.com/moutend/go-wca v0.1.2-0.20190422112502-0fa027b3d89a
	github.com/thoas/go-funk v0.7.0
	go.bug.st/serial v1.6.2
	go.uber.org/zap v1.27.1
	golang.org/x/sys v0.39.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	git.sr.ht/~jackmordaunt/go-toast v1.1.2 // indirect
	github.com/creack/goselect v0.1.2 // indirect
	github.com/esiqveland/notify v0.13.3 // indirect
	github.com/getlantern/context v0.0.0-20190109183933-c447772a6520 // indirect
	github.com/getlantern/errors v0.0.0-20190325191628-abdb3e3e36f7 // indirect
	github.com/getlantern/golog v0.0.0-20190830074920-4ef2e798c2d7 // indirect
	github.com/getlantern/hex v0.0.0-20190417191902-c6586a6fe0b7 // indirect
	github.com/getlantern/hidden v0.0.0-20190325191715-f02dbb02be55 // indirect
	github.com/getlantern/ops v0.0.0-20190325191751-d70cb0d6f85f // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/jackmordaunt/icns/v3 v3.0.1 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/sergeymakinen/go-bmp v1.0.0 // indirect
	github.com/sergeymakinen/go-ico v1.0.0-beta.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tadvi/systray v0.0.0-20190226123456-11a2b8fa57af // indirect
	go.uber.org/multierr v1.10.0 // indirect
	gopkg.in/Knetic/govaluate.v3 v3.0.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

// Notes:
// - Windows production builds are pure Go; CGO/GCC is needed only for race detection.
// - fsnotify handles replacement-aware configuration reload.
