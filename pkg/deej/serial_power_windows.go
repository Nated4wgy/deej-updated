//go:build windows

package deej

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// RegisterSuspendResumeNotification is supported on Windows 8 and newer.
// DEVICE_NOTIFY_CALLBACK lets the background mixer receive power events
// without creating a hidden window or depending on the Settings dashboard.
const (
	deviceNotifyCallback       = 2
	pbtAPMResumeAutomatic      = 0x12
)

type deviceNotifySubscribeParameters struct {
	Callback uintptr
	Context  uintptr
}

var (
	user32Power = windows.NewLazySystemDLL("user32.dll")
	registerSuspendResume = user32Power.NewProc("RegisterSuspendResumeNotification")
	unregisterSuspendResume = user32Power.NewProc("UnregisterSuspendResumeNotification")
)

// subscribeSystemResume queues a reconnect on every automatic wake-up,
// including unattended wakes. Windows can resume before USB serial hardware
// has finished re-enumerating, so SerialIO deliberately reconnects later.
//
// The notification callback only queues work. It NEVER reads, closes or opens
// COM ports, and never touches Windows audio COM objects.
func subscribeSystemResume(notify func()) (func(), error) {
	if err := registerSuspendResume.Find(); err != nil {
		return nil, fmt.Errorf("find Windows resume notification API: %w", err)
	}
	if err := unregisterSuspendResume.Find(); err != nil {
		return nil, fmt.Errorf("find Windows resume unregister API: %w", err)
	}

	callback := syscall.NewCallback(func(_ uintptr, eventType uintptr, _ uintptr) uintptr {
		if eventType == pbtAPMResumeAutomatic {
			notify()
		}
		return 0 // ERROR_SUCCESS: callback handled the event.
	})
	params := &deviceNotifySubscribeParameters{Callback: callback}

	handle, _, callErr := registerSuspendResume.Call(
		uintptr(unsafe.Pointer(params)), deviceNotifyCallback,
	)
	if handle == 0 {
		return nil, fmt.Errorf("register Windows resume notification: %v", callErr)
	}
	return func() {
		unregisterSuspendResume.Call(handle)
		// Keep subscription memory alive until Windows has unregistered the
		// native callback; the Go runtime retains the generated callback stub.
		runtime.KeepAlive(params)
		runtime.KeepAlive(callback)
	}, nil
}
