package deej

import (
	"fmt"
	ole "github.com/go-ole/go-ole"
	ps "github.com/mitchellh/go-ps"
	wca "github.com/moutend/go-wca"
	"go.uber.org/zap"
	"golang.org/x/sys/windows"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"
)

type wcaSessionFinder struct {
	logger, sessionLogger *zap.SugaredLogger
	eventCtx              *ole.GUID
	mmDeviceEnumerator    *wca.IMMDeviceEnumerator
	mmNotificationClient  *wca.IMMNotificationClient
	devicesChanged        atomic.Bool
	refs                  atomic.Int32
}

func newSessionFinder(logger *zap.SugaredLogger) (SessionFinder, error) {
	return &wcaSessionFinder{logger: logger.Named("session_finder"), sessionLogger: logger.Named("sessions"), eventCtx: ole.NewGUID("{1ec920a1-7db8-44ba-9779-e5d28ed9f330}")}, nil
}
func (sf *wcaSessionFinder) GetAllSessions() (sessions []Session, resultErr error) {
	// COM is initialized by the owning audio worker, not temporarily during enumeration.
	defer func() {
		if resultErr != nil {
			for _, s := range sessions {
				s.Release()
			}
			sessions = nil
		}
	}()
	if sf.mmDeviceEnumerator == nil {
		if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &sf.mmDeviceEnumerator); err != nil {
			return nil, err
		}
	}
	if sf.mmNotificationClient == nil {
		if err := sf.registerDefaultDeviceChangeCallback(); err != nil {
			return nil, err
		}
	}
	for _, spec := range []struct {
		flow uint32
		key  string
	}{{wca.ERender, "master"}, {wca.ECapture, "mic"}} {
		var device *wca.IMMDevice
		if err := sf.mmDeviceEnumerator.GetDefaultAudioEndpoint(spec.flow, wca.EConsole, &device); err != nil {
			continue
		}
		s, err := sf.getMasterSession(device, spec.key)
		device.Release()
		if err != nil {
			sf.logger.Debugw("Default endpoint unavailable", "target", spec.key, "error", err)
			continue
		}
		sessions = append(sessions, s)
	}
	var devices *wca.IMMDeviceCollection
	if err := sf.mmDeviceEnumerator.EnumAudioEndpoints(wca.EAll, wca.DEVICE_STATE_ACTIVE, &devices); err != nil {
		return sessions, err
	}
	defer devices.Release()
	var count uint32
	if err := devices.GetCount(&count); err != nil {
		return sessions, err
	}
	// Resolve PIDs from one snapshot instead of enumerating all processes for each session.
	processNames := map[uint32]string{}
	processes, err := ps.Processes()
	if err != nil {
		return sessions, err
	}
	for _, p := range processes {
		processNames[uint32(p.Pid())] = p.Executable()
	}
	for i := uint32(0); i < count; i++ {
		var device *wca.IMMDevice
		if err := devices.Item(i, &device); err != nil {
			continue
		}
		found, err := sf.deviceSessions(device, processNames)
		device.Release()
		if err != nil {
			sf.logger.Debugw("Skipped unavailable endpoint", "error", err)
		}
		sessions = append(sessions, found...)
	}
	return sessions, nil
}
func (sf *wcaSessionFinder) Release() error {
	if sf.mmDeviceEnumerator == nil {
		return nil
	}
	if sf.mmNotificationClient != nil {
		// The pinned go-wca wrapper returns E_NOTIMPL for unregister; call its vtable directly.
		hr, _, _ := syscall.Syscall(sf.mmDeviceEnumerator.VTable().UnregisterEndpointNotificationCallback, 2, uintptr(unsafe.Pointer(sf.mmDeviceEnumerator)), uintptr(unsafe.Pointer(sf.mmNotificationClient)), 0)
		if hr != 0 {
			return ole.NewError(hr)
		}
		sf.mmNotificationClient = nil
	}
	sf.mmDeviceEnumerator.Release()
	sf.mmDeviceEnumerator = nil
	return nil
}
func (sf *wcaSessionFinder) registerDefaultDeviceChangeCallback() error {
	sf.refs.Store(1)
	sf.mmNotificationClient = &wca.IMMNotificationClient{VTable: &wca.IMMNotificationClientVtbl{}}
	v := sf.mmNotificationClient.VTable
	v.QueryInterface = syscall.NewCallback(func(this uintptr, iid *ole.GUID, out *uintptr) uintptr {
		if out == nil {
			return 0x80004003
		}
		*out = 0
		notificationIID := ole.NewGUID("{7991EEC9-7E89-4D85-8390-6C703CEC60C0}")
		if iid != nil && (*iid == *ole.IID_IUnknown || *iid == *notificationIID) {
			*out = this
			sf.refs.Add(1)
			return 0
		}
		return 0x80004002
	})
	v.AddRef = syscall.NewCallback(func(this uintptr) uintptr { return uintptr(sf.refs.Add(1)) })
	v.Release = syscall.NewCallback(func(this uintptr) uintptr { return uintptr(sf.refs.Add(-1)) })
	// Callbacks only signal; all endpoint pointers are accessed on the audio worker.
	v.OnDeviceStateChanged = syscall.NewCallback(func(this, id, state uintptr) uintptr { sf.devicesChanged.Store(true); return 0 })
	v.OnDeviceAdded = syscall.NewCallback(func(this, id uintptr) uintptr { sf.devicesChanged.Store(true); return 0 })
	v.OnDeviceRemoved = syscall.NewCallback(func(this, id uintptr) uintptr { sf.devicesChanged.Store(true); return 0 })
	v.OnDefaultDeviceChanged = syscall.NewCallback(func(this, flow, role, id uintptr) uintptr { sf.devicesChanged.Store(true); return 0 })
	v.OnPropertyValueChanged = syscall.NewCallback(func(this, id, key uintptr) uintptr { sf.devicesChanged.Store(true); return 0 })
	if err := sf.mmDeviceEnumerator.RegisterEndpointNotificationCallback(sf.mmNotificationClient); err != nil {
		return err
	}
	return nil
}
func (sf *wcaSessionFinder) DevicesChanged() bool { return sf.devicesChanged.Swap(false) }

func (sf *wcaSessionFinder) getMasterSession(device *wca.IMMDevice, key string) (*masterSession, error) {
	var volume *wca.IAudioEndpointVolume
	if err := device.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &volume); err != nil {
		return nil, err
	}
	return newMasterSession(sf.sessionLogger, volume, sf.eventCtx, key, strings.ToLower(key))
}

var propVariantClear = windows.NewLazySystemDLL("ole32.dll").NewProc("PropVariantClear")

func deviceName(device *wca.IMMDevice) (string, error) {
	var store *wca.IPropertyStore
	if err := device.OpenPropertyStore(wca.STGM_READ, &store); err != nil {
		return "", err
	}
	defer store.Release()
	value := &wca.PROPVARIANT{}
	if err := store.GetValue(&wca.PKEY_Device_FriendlyName, value); err != nil {
		return "", err
	}
	// The pinned go-wca String method copies and frees LPWSTR; it owns this allocation.
	if value.VT != ole.VT_LPWSTR {
		propVariantClear.Call(uintptr(unsafe.Pointer(value)))
		return "", fmt.Errorf("device name is not a Unicode string")
	}
	return value.String(), nil
}
func (sf *wcaSessionFinder) deviceSessions(device *wca.IMMDevice, names map[uint32]string) ([]Session, error) {
	name, err := deviceName(device)
	if err != nil {
		return nil, err
	}
	var result []Session
	if master, err := sf.getMasterSession(device, name); err == nil {
		result = append(result, master)
	}
	dispatch, err := device.QueryInterface(wca.IID_IMMEndpoint)
	if err != nil {
		return result, err
	}
	endpoint := (*wca.IMMEndpoint)(unsafe.Pointer(dispatch))
	defer endpoint.Release()
	var flow uint32
	if err := endpoint.GetDataFlow(&flow); err != nil {
		return result, err
	}
	if flow != wca.ERender {
		return result, nil
	}
	var manager *wca.IAudioSessionManager2
	if err := device.Activate(wca.IID_IAudioSessionManager2, wca.CLSCTX_ALL, nil, &manager); err != nil {
		return result, err
	}
	defer manager.Release()
	var enumerator *wca.IAudioSessionEnumerator
	if err := manager.GetSessionEnumerator(&enumerator); err != nil {
		return result, err
	}
	defer enumerator.Release()
	var count int
	if err := enumerator.GetCount(&count); err != nil {
		return result, err
	}
	for i := 0; i < count; i++ {
		s, err := sf.processSession(enumerator, i, names)
		if err != nil {
			sf.logger.Debugw("Skipped expired audio session", "error", err)
			continue
		}
		result = append(result, s)
	}
	return result, nil
}
func (sf *wcaSessionFinder) processSession(enumerator *wca.IAudioSessionEnumerator, index int, names map[uint32]string) (Session, error) {
	var base *wca.IAudioSessionControl
	if err := enumerator.GetSession(index, &base); err != nil {
		return nil, err
	}
	defer base.Release()
	dispatch, err := base.QueryInterface(wca.IID_IAudioSessionControl2)
	if err != nil {
		return nil, err
	}
	control := (*wca.IAudioSessionControl2)(unsafe.Pointer(dispatch))
	owned := false
	defer func() {
		if !owned {
			control.Release()
		}
	}()
	var state uint32
	if err := control.GetState(&state); err != nil {
		return nil, err
	}
	if state == wca.AudioSessionStateExpired {
		return nil, fmt.Errorf("session expired")
	}
	var pid uint32
	if err := control.GetProcessId(&pid); err != nil && pid == 0 && control.IsSystemSoundsSession() != nil {
		return nil, err
	}
	processName := names[pid]
	if pid != 0 && processName == "" {
		return nil, errNoSuchProcess
	}
	dispatch, err = control.QueryInterface(wca.IID_ISimpleAudioVolume)
	if err != nil {
		return nil, err
	}
	volume := (*wca.ISimpleAudioVolume)(unsafe.Pointer(dispatch))
	s, err := newWCASession(sf.sessionLogger, control, volume, pid, processName, sf.eventCtx)
	if err != nil {
		volume.Release()
		return nil, err
	}
	owned = true
	return s, nil
}
