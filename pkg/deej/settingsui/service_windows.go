package settingsui

import (
	"encoding/json"
	"fmt"
	"github.com/omriharel/deej/pkg/deej"
	"github.com/omriharel/deej/pkg/deej/util"
	"go.bug.st/serial"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type startupStore interface {
	Enabled(string, string) bool
	Set(string, bool, string) error
}
type windowsStartup struct{}

func (windowsStartup) Enabled(exe, config string) bool { return util.StartupEnabled(exe, config) }
func (windowsStartup) Set(exe string, enabled bool, config string) error {
	return util.SetStartup(exe, enabled, config)
}

type service struct {
	mu                            sync.Mutex
	filename, executable, version string
	startup                       startupStore
}
type initialState struct {
	Settings    deej.Settings `json:"settings"`
	Revision    string        `json:"revision"`
	ConfigError string        `json:"config_error"`
	Filename    string        `json:"filename"`
	Version     string        `json:"version"`
	Ports       []string      `json:"ports"`
	Startup     bool          `json:"startup"`
	HasBackup   bool          `json:"has_backup"`
}

func (s *service) initial() (initialState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := initialState{Settings: deej.DefaultSettings(), Filename: s.filename, Version: s.version, Startup: s.startup.Enabled(s.executable, s.filename)}
	data, err := os.ReadFile(s.filename)
	if err == nil {
		state.Revision = deej.SettingsRevision(data)
		state.Settings, err = deej.ParseSettings(data)
	}
	if err != nil {
		state.Settings = deej.DefaultSettings()
		if !os.IsNotExist(err) {
			state.ConfigError = err.Error()
		}
	}
	state.Ports, _ = serial.GetPortsList()
	if state.Ports == nil {
		state.Ports = []string{}
	}
	_, err = os.Stat(s.filename + ".bak")
	state.HasBackup = err == nil
	return state, nil
}

type saveRequest struct {
	Settings         deej.Settings `json:"settings"`
	ExpectedRevision string        `json:"expected_revision"`
	Startup          bool          `json:"startup"`
}

func (s *service) checkRevision(expected string) error {
	data, err := os.ReadFile(s.filename)
	if err != nil {
		if os.IsNotExist(err) && expected == "" {
			return nil
		}
		return err
	}
	if deej.SettingsRevision(data) != expected {
		return fmt.Errorf("settings changed outside this window. Reload before saving so those changes are preserved")
	}
	return nil
}
func (s *service) save(request saveRequest) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := request.Settings.Validate(); err != nil {
		return "", err
	}
	if err := s.checkRevision(request.ExpectedRevision); err != nil {
		return "", err
	}
	previous := s.startup.Enabled(s.executable, s.filename)
	// Refresh enabled legacy registrations to use --background without opening a window at login.
	if previous != request.Startup || request.Startup {
		if err := s.startup.Set(s.executable, request.Startup, s.filename); err != nil {
			return "", err
		}
	}
	if err := deej.SaveSettings(s.filename, request.Settings); err != nil {
		if previous != request.Startup {
			if rollback := s.startup.Set(s.executable, previous, s.filename); rollback != nil {
				return "", fmt.Errorf("%v; startup rollback also failed: %v", err, rollback)
			}
		}
		return "", err
	}
	data, err := os.ReadFile(s.filename)
	if err != nil {
		return "", err
	}
	return deej.SettingsRevision(data), nil
}
func (s *service) restore(expected string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkRevision(expected); err != nil {
		return err
	}
	return deej.RestoreSettingsBackup(s.filename)
}

type liveState struct {
	Live   bool               `json:"live"`
	Status deej.RuntimeStatus `json:"status"`
}

func (s *service) status() (liveState, error) {
	var result liveState
	data, err := os.ReadFile(filepath.Join(filepath.Dir(s.filename), "logs", "status.json"))
	if err != nil {
		return result, nil
	}
	if json.Unmarshal(data, &result.Status) == nil {
		age := time.Since(result.Status.UpdatedAt)
		result.Live = age >= 0 && age < 3*time.Second
	}
	return result, nil
}
func (s *service) start() error {
	cmd := exec.Command(s.executable, "--background", "--config", s.filename)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
func (s *service) openConfig() error { return launch("notepad.exe", s.filename) }
func (s *service) openFolder() error { return launch("explorer.exe", filepath.Dir(s.filename)) }
func launch(program, arg string) error {
	cmd := exec.Command(program, arg)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
