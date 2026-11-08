package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/omriharel/deej/pkg/deej"
	"github.com/omriharel/deej/pkg/deej/util"
	"go.bug.st/serial"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var versionTag, gitCommit, buildType string

type sliderRow struct {
	box      *walk.Composite
	targets  *walk.LineEdit
	pick     *walk.ComboBox
	position *walk.ProgressBar
	info     *walk.Label
}

func main() {
	configPath := flag.String("config", "", "path to config.yaml")
	flag.Parse()
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	filename := filepath.Join(filepath.Dir(exe), "config.yaml")
	if *configPath != "" {
		filename, err = filepath.Abs(*configPath)
		if err != nil {
			panic(err)
		}
	}
	directory := filepath.Dir(filename)
	settings, loadErr := deej.ReadSettings(filename)
	if loadErr != nil {
		settings = deej.DefaultSettings()
	}
	var mw *walk.MainWindow
	var port *walk.ComboBox
	var baud, count *walk.NumberEdit
	var noise *walk.ComboBox
	var inverted, startup *walk.CheckBox
	var connection, message *walk.Label
	var save *walk.PushButton
	rows := make([]sliderRow, 32)
	dirty := false
	loading := true
	expectedRevision := ""
	lastTargets := ""
	diskRevision := ""
	if data, e := os.ReadFile(filename); e == nil {
		diskRevision = deej.SettingsRevision(data)
	}
	markDirty := func() {
		if !loading {
			dirty = true
			if message != nil {
				message.SetText("Unsaved changes")
			}
		}
	}
	ports, _ := serial.GetPortsList()
	if len(ports) == 0 {
		ports = []string{settings.COMPort}
	}
	countValue := 5
	for id := range settings.SliderMapping {
		if id+1 > countValue {
			countValue = id + 1
		}
	}
	builtins := []string{"master", "system", "mic", "deej.current", "deej.unmapped"}
	var children []Widget
	for i := range rows {
		id := i
		r := &rows[i]
		children = append(children, Composite{AssignTo: &r.box, MinSize: Size{Width: 680}, Layout: VBox{MarginsZero: true}, Children: []Widget{
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{Label{Text: fmt.Sprintf("Slider %d", i+1), MinSize: Size{Width: 65, Height: 0}}, ProgressBar{AssignTo: &r.position, MinValue: 0, MaxValue: 100, MinSize: Size{Width: 100, Height: 16}, MaxSize: Size{Width: 160, Height: 20}}, Label{AssignTo: &r.info, Text: "No live position", StretchFactor: 1}}},
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{
				LineEdit{AssignTo: &r.targets, MinSize: Size{Width: 300}, Text: strings.Join(settings.SliderMapping[id], "; "), StretchFactor: 3, ToolTipText: "One or more targets, separated by semicolons. App names such as firefox.exe also work when the app is closed.", OnTextChanged: markDirty},
				ComboBox{AssignTo: &r.pick, Model: builtins, CurrentIndex: 0, MinSize: Size{Width: 170, Height: 0}, StretchFactor: 1},
				PushButton{Text: "Add target", OnClicked: func() {
					value := r.pick.Text()
					if value == "" {
						return
					}
					current := strings.TrimSpace(r.targets.Text())
					if current == "" {
						r.targets.SetText(value)
					} else {
						r.targets.SetText(current + "; " + value)
					}
				}},
			}},
		}})
	}
	updateRows := func() {
		if count == nil {
			return
		}
		for i := range rows {
			if rows[i].box != nil {
				rows[i].box.SetVisible(i < int(count.Value()))
			}
		}
	}
	errorBox := func(err error) { walk.MsgBox(mw, "Could not save settings", err.Error(), walk.MsgBoxIconError) }
	saveSettings := func() {
		s := deej.Settings{SliderMapping: map[int][]string{}, COMPort: strings.TrimSpace(port.Text()), BaudRate: int(baud.Value()), InvertSliders: inverted.Checked(), NoiseReductionLevel: noise.Text()}
		for id := 0; id < int(count.Value()); id++ {
			text := strings.TrimSpace(rows[id].targets.Text())
			if text == "" {
				continue
			}
			for _, t := range strings.Split(text, ";") {
				t = strings.TrimSpace(t)
				if t == "" {
					errorBox(fmt.Errorf("slider %d has an empty target", id+1))
					return
				}
				s.SliderMapping[id] = append(s.SliderMapping[id], t)
			}
		}
		if err := s.Validate(); err != nil {
			errorBox(err)
			return
		}
		if current, e := os.ReadFile(filename); e == nil && diskRevision != "" && deej.SettingsRevision(current) != diskRevision {
			errorBox(fmt.Errorf("config changed outside this window. Close and reopen Settings to load the current file before saving"))
			return
		}
		engine := filepath.Join(directory, "deej.exe")
		wasStartup := util.StartupEnabled(engine)
		if wasStartup != startup.Checked() {
			if err := util.SetStartup(engine, startup.Checked()); err != nil {
				errorBox(err)
				return
			}
		}
		if err := deej.SaveSettings(filename, s); err != nil {
			if wasStartup != startup.Checked() {
				util.SetStartup(engine, wasStartup)
			}
			errorBox(err)
			return
		}
		data, _ := os.ReadFile(filename)
		expectedRevision = deej.SettingsRevision(data)
		diskRevision = expectedRevision
		dirty = false
		message.SetText("Saved. Waiting for the running mixer to apply settings...")
	}
	err = (MainWindow{AssignTo: &mw, Title: "Deej Settings", MinSize: Size{Width: 760, Height: 600}, Size: Size{Width: 920, Height: 760}, Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 16}, Spacing: 12}, Children: []Widget{
		Label{Text: "Deej", Font: Font{PointSize: 20, Bold: true}},
		Label{AssignTo: &connection, Text: "Checking mixer status..."},
		GroupBox{Title: "Connection and behaviour", Layout: Grid{Columns: 4}, Children: []Widget{
			Label{Text: "Serial port"}, ComboBox{AssignTo: &port, MinSize: Size{Width: 220}, Editable: true, Model: ports, Value: settings.COMPort, OnTextChanged: markDirty},
			Label{Text: "Baud rate"}, NumberEdit{AssignTo: &baud, MinSize: Size{Width: 130}, Value: float64(settings.BaudRate), MinValue: 50, MaxValue: 2000000, Decimals: 0, OnValueChanged: markDirty},
			Label{Text: "Noise reduction"}, ComboBox{AssignTo: &noise, Model: []string{"low", "default", "high"}, CurrentIndex: map[string]int{"low": 0, "default": 1, "high": 2}[settings.NoiseReductionLevel], OnCurrentIndexChanged: markDirty},
			Label{Text: "Slider count"}, NumberEdit{AssignTo: &count, MinSize: Size{Width: 130}, Value: float64(countValue), MinValue: 1, MaxValue: 32, Decimals: 0, OnValueChanged: func() { updateRows(); markDirty() }},
			CheckBox{AssignTo: &inverted, Text: "Invert slider direction", Checked: settings.InvertSliders, ColumnSpan: 2, OnCheckedChanged: markDirty},
			CheckBox{AssignTo: &startup, MinSize: Size{Width: 220}, Text: "Start mixer with Windows", Checked: util.StartupEnabled(filepath.Join(directory, "deej.exe")), ColumnSpan: 2, OnCheckedChanged: markDirty},
		}},
		Label{Text: "Low noise reduction follows smaller movements. Baud rate must match your device firmware."},
		Label{Text: "Slider mappings", Font: Font{PointSize: 11, Bold: true}},
		Label{Text: "Choose a target and click Add, or type an app name. Use semicolons to control several targets together."},
		ScrollView{Layout: VBox{Spacing: 14}, HorizontalFixed: true, StretchFactor: 1, Children: children},
		Label{AssignTo: &message, Text: "Edit settings, then click Save & apply."},
		Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{
			PushButton{Text: "Advanced YAML", OnClicked: func() {
				cmd := exec.Command("notepad.exe", filename)
				if err := cmd.Start(); err != nil {
					errorBox(err)
				} else {
					go cmd.Wait()
				}
			}},
			PushButton{Text: "Restore backup", OnClicked: func() {
				if walk.MsgBox(mw, "Restore backup", "Replace config.yaml with the previous saved version? Close and reopen Settings afterwards.", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
					return
				}
				backup := filename + ".bak"
				s, err := deej.ReadSettings(backup)
				if err != nil {
					errorBox(err)
					return
				}
				if err = deej.SaveSettings(filename, s); err != nil {
					errorBox(err)
					return
				}
				dirty = false
				mw.Close()
			}},
			HSpacer{}, PushButton{AssignTo: &save, Text: "Save && apply", OnClicked: saveSettings}, PushButton{Text: "Close", OnClicked: func() { mw.Close() }},
		}},
	}}).Create()
	if err != nil {
		walk.MsgBox(nil, "Deej Settings", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = save
	updateRows()
	loading = false
	if loadErr != nil {
		if os.IsNotExist(loadErr) {
			message.SetText("First run: choose a port and your targets, then save.")
		} else {
			message.SetText("Config could not load. Open Advanced YAML to correct it: " + loadErr.Error())
		}
	}
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if dirty && walk.MsgBox(mw, "Unsaved changes", "Discard your unsaved settings?", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
			*canceled = true
		}
	})
	// UI status polling reads only a small snapshot; it never touches Core Audio or serial.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				data, err := os.ReadFile(filepath.Join(directory, "logs", "status.json"))
				var status deej.RuntimeStatus
				if err == nil {
					err = json.Unmarshal(data, &status)
				}
				live := err == nil && time.Since(status.UpdatedAt) < 3*time.Second
				select {
				case <-stop:
					return
				default:
				}
				mw.Synchronize(func() {
					if !live {
						connection.SetText("Mixer offline — you can edit and save settings.")
						if expectedRevision != "" {
							message.SetText("Saved to disk. Start deej.exe to apply settings.")
						}
						return
					}
					text := "Device disconnected"
					if status.Serial.Connected {
						text = "Connected to " + status.Serial.Port
					}
					if status.Serial.Error != "" {
						text += " — " + status.Serial.Error
					}
					connection.SetText(text + fmt.Sprintf("   |   Response p95: %.1f ms", status.Audio.P95ApplyMS))
					if expectedRevision != "" {
						if status.Audio.ConfigError != "" {
							message.SetText("Saved, but mixer rejected config: " + status.Audio.ConfigError)
						} else if status.Audio.ConfigRevision == expectedRevision {
							message.SetText("Settings applied by the mixer.")
							expectedRevision = ""
						}
					}
					keys := append(append([]string(nil), builtins...), status.Audio.Targets...)
					joined := strings.Join(keys, "|")
					if joined != lastTargets {
						for i := range rows {
							rows[i].pick.SetModel(keys)
							rows[i].pick.SetCurrentIndex(0)
						}
						lastTargets = joined
					}
					for i := range rows {
						if i < len(status.Serial.Positions) {
							rows[i].position.SetValue(int(status.Serial.Positions[i] * 100))
							rows[i].info.SetText(strconv.Itoa(int(status.Serial.Positions[i]*100)) + "%  |  Effective: " + strings.Join(status.Audio.EffectiveSettings.SliderMapping[i], "; "))
						} else {
							rows[i].info.SetText("No live position")
						}
					}
				})
			}
		}
	}()
	mw.Run()
}
