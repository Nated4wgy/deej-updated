package settingsui

import (
	"github.com/omriharel/deej/pkg/deej"
	"os"
	"path/filepath"
	"testing"
)

type fakeStartup struct {
	enabled bool
	writes  int
}

func (s *fakeStartup) Enabled(string, string) bool { return s.enabled }
func (s *fakeStartup) Set(_ string, enabled bool, _ string) error {
	s.enabled = enabled
	s.writes++
	return nil
}
func TestSaveRejectsConcurrentConfigAndPreservesExternalEdits(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	deej.SaveSettings(file, deej.DefaultSettings())
	startup := &fakeStartup{}
	s := &service{filename: file, executable: "deej.exe", startup: startup}
	state, err := s.initial()
	if err != nil {
		t.Fatal(err)
	}
	external := deej.DefaultSettings()
	external.COMPort = "COM9"
	deej.SaveSettings(file, external)
	if _, err = s.save(saveRequest{Settings: state.Settings, ExpectedRevision: state.Revision, Startup: true}); err == nil {
		t.Fatal("concurrent edits were overwritten")
	}
	actual, _ := deej.ReadSettings(file)
	if actual.COMPort != "COM9" || startup.writes != 0 {
		t.Fatal("save conflict changed configuration or startup")
	}
}
func TestSaveAndRestoreKeepConfigAndRevisionInSync(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	original := deej.DefaultSettings()
	deej.SaveSettings(file, original)
	s := &service{filename: file, executable: "deej.exe", startup: &fakeStartup{}}
	state, _ := s.initial()
	state.Settings.SliderMapping[1] = []string{"discord.exe", "firefox.exe"}
	revision, err := s.save(saveRequest{Settings: state.Settings, ExpectedRevision: state.Revision})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(file)
	if revision != deej.SettingsRevision(data) {
		t.Fatal("acknowledged the wrong revision")
	}
	if err = s.restore(revision); err != nil {
		t.Fatal(err)
	}
	restored, _ := deej.ReadSettings(file)
	if len(restored.SliderMapping[1]) != 0 {
		t.Fatal("backup not restored")
	}
}
