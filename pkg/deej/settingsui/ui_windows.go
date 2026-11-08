package settingsui

import (
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	ole "github.com/go-ole/go-ole"
	"github.com/jchv/go-webview2/pkg/edge"
	"github.com/lxn/walk"
	"github.com/lxn/win"
	"github.com/omriharel/deej/pkg/deej/util"
	"go.bug.st/serial"
	"golang.org/x/sys/windows"
	"path/filepath"
	"strings"
	"unsafe"
)

//go:embed web/*
var assets embed.FS

func ReportError(err error) { walk.MsgBox(nil, "Deej", err.Error(), walk.MsgBoxIconError) }
func Show(filename, executable, version string) (resultErr error) {
	// Some WebView2 loader failures surface as a nil controller in the pinned binding.
	// Keep those failures confined to the UI process and show a recoverable error.
	defer func() {
		if failure := recover(); failure != nil {
			resultErr = fmt.Errorf("Deej could not initialize its interface (%v). Check the WebView2 Runtime installation. The tray mixer can still run with --background", failure)
		}
	}()
	// Keep a single UI per configuration and wake it when the executable is opened again.
	sum := sha256.Sum256([]byte(strings.ToLower(filename)))
	eventName, _ := windows.UTF16PtrFromString(fmt.Sprintf("Local\\DeejUI-%x", sum[:12]))
	event, err := windows.CreateEvent(nil, 0, 0, eventName)
	if err != nil && err != windows.ERROR_ALREADY_EXISTS {
		return err
	}
	defer windows.CloseHandle(event)
	release, err := util.AcquireInstance(filename + "|ui")
	if err != nil {
		if util.InstanceAlreadyRunning(err) {
			return windows.SetEvent(event)
		}
		return err
	}
	defer release()
	s := &service{filename: filename, executable: executable, version: version, startup: windowsStartup{}}
	mw, err := walk.NewMainWindow()
	if err != nil {
		return err
	}
	defer mw.Dispose()
	mw.SetLayout(walk.NewVBoxLayout())
	mw.SetTitle("Deej")
	mw.SetSize(walk.Size{Width: 1180, Height: 850})
	mw.SetMinMaxSize(walk.Size{Width: 880, Height: 650}, walk.Size{})
	if icon, err := walk.NewIconFromResourceId(1); err == nil {
		mw.SetIcon(icon)
		defer icon.Dispose()
	}
	browser := edge.NewChromium()
	browser.DataPath = filepath.Join(filepath.Dir(filename), "logs", "interface")
	browser.SetGlobalPermission(edge.CoreWebView2PermissionStateDeny)
	if !browser.Embed(uintptr(mw.AsContainerBase().Handle())) {
		return fmt.Errorf("Deej could not open its interface. Microsoft Edge WebView2 Runtime must be installed. The mixer can still run with --background")
	}
	settings, err := browser.GetSettings()
	if err != nil {
		return err
	}
	settings.PutAreDefaultContextMenusEnabled(false)
	settings.PutAreDevToolsEnabled(false)
	settings.PutIsStatusBarEnabled(false)
	settings.PutAreBrowserAcceleratorKeysEnabled(false)
	// The interface is fully embedded and needs no network or remote pages.
	browser.WebResourceRequestedCallback = func(_ *edge.ICoreWebView2WebResourceRequest, args *edge.ICoreWebView2WebResourceRequestedEventArgs) {
		response, err := browser.Environment().CreateWebResourceResponse([]byte("Offline interface"), 403, "Blocked", "Content-Type: text/plain")
		if err == nil {
			args.PutResponse(response)
			(*ole.IUnknown)(unsafe.Pointer(response)).Release()
		}
	}
	browser.AddWebResourceRequestedFilter("*", edge.COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL)
	dirty := false
	mw.Closing().Attach(func(canceled *bool, _ walk.CloseReason) {
		if dirty && walk.MsgBox(mw, "Unsaved changes", "Discard unsaved changes and close the window? The mixer will keep running.", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
			*canceled = true
		}
	})
	type request struct {
		ID     uint64          `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	browser.MessageCallback = func(message string) {
		if len(message) > 65536 {
			return
		}
		var r request
		if json.Unmarshal([]byte(message), &r) != nil {
			return
		}
		var value interface{}
		var err error
		switch r.Method {
		case "initial":
			value, err = s.initial()
		case "status":
			if signaled, e := windows.WaitForSingleObject(event, 0); e == nil && signaled == windows.WAIT_OBJECT_0 {
				mw.Show()
				win.ShowWindow(mw.Handle(), win.SW_RESTORE)
				win.SetForegroundWindow(mw.Handle())
			}
			value, err = s.status()
		case "save":
			var p saveRequest
			if err = json.Unmarshal(r.Params, &p); err == nil {
				value, err = s.save(p)
				if err == nil {
					dirty = false
				}
			}
		case "restore":
			var expected string
			if err = json.Unmarshal(r.Params, &expected); err == nil {
				err = s.restore(expected)
				if err == nil {
					dirty = false
				}
			}
		case "ports":
			value, err = serial.GetPortsList()
		case "start":
			err = s.start()
		case "config":
			err = s.openConfig()
		case "folder":
			err = s.openFolder()
		case "theme":
			var theme string
			if err = json.Unmarshal(r.Params, &theme); err == nil {
				applyWindowTheme(mw.Handle(), theme == "dark")
			}
		case "dirty":
			err = json.Unmarshal(r.Params, &dirty)
		case "close":
			mw.Synchronize(func() { mw.Close() })
		default:
			err = fmt.Errorf("unknown interface action")
		}
		response := struct {
			ID     uint64      `json:"id"`
			Result interface{} `json:"result"`
			Error  string      `json:"error,omitempty"`
		}{ID: r.ID, Result: value}
		if err != nil {
			response.Error = err.Error()
		}
		encoded, _ := json.Marshal(response)
		browser.Eval("window.deejReceive(" + string(encoded) + ")")
	}
	html, _ := assets.ReadFile("web/index.html")
	css, _ := assets.ReadFile("web/styles.css")
	js, _ := assets.ReadFile("web/app.js")
	document := strings.Replace(string(html), "/*STYLE*/", string(css), 1)
	document = strings.Replace(document, "/*SCRIPT*/", string(js), 1)
	mw.AsContainerBase().SizeChanged().Attach(func() {
		browser.Resize()
	})
	mw.BoundsChanged().Attach(func() { browser.NotifyParentWindowPositionChanged() })
	mw.Activating().Attach(func() { browser.Focus() })
	mw.Show()
	browser.Show()
	browser.NavigateToString(document)
	browser.Resize()
	mw.Starting().Attach(func() {
		browser.Show()
		browser.Resize()
	})
	mw.Run()
	return nil
}

var setDWMAttribute = windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")

func applyWindowTheme(handle win.HWND, dark bool) {
	if setDWMAttribute.Find() != nil {
		return
	}
	var enabled int32
	if dark {
		enabled = 1
	}
	setDWMAttribute.Call(uintptr(handle), 20, uintptr(unsafe.Pointer(&enabled)), unsafe.Sizeof(enabled))
}
