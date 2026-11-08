package deej

import (
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigValidationAndCompatibility(t *testing.T) {
	for _, data := range []string{"slider_mapping:\n  0: master\n  1: [firefox.exe, discord.exe]\n", "slider_mapping:\n  0: [master]\n"} {
		s, err := ParseSettings([]byte(data))
		if err != nil || s.SliderMapping[0][0] != "master" {
			t.Fatalf("%+v %v", s, err)
		}
	}
	for _, data := range []string{"slider_mapping: {oops: master}", "slider_mapping: {-1: master}", "slider_mapping: {32: master}", "slider_mapping: {0: master, 0: mic}", "baud_rate: 0", "noise_reduction: extreme", "invert_sliders: perhaps", "slider_mapping: {0: [master, '']}"} {
		if _, err := ParseSettings([]byte(data)); err == nil {
			t.Errorf("accepted %q", data)
		}
	}
}
func TestAtomicSaveBackupAndLastGoodConfig(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	original := []byte("# keep this\ncustom_key: keep-me\nslider_mapping: {0: master}\n")
	os.WriteFile(file, original, 0600)
	next := DefaultSettings()
	next.SliderMapping[1] = []string{"discord.exe", "firefox.exe"}
	if err := SaveSettings(file, next); err != nil {
		t.Fatal(err)
	}
	backup, _ := os.ReadFile(file + ".bak")
	if string(backup) != string(original) {
		t.Fatal("previous bytes not backed up")
	}
	data, _ := os.ReadFile(file)
	if !strings.Contains(string(data), "keep-me") || !strings.Contains(string(data), "# keep this") {
		t.Fatal("unrelated YAML lost")
	}
	cc, _ := NewConfig(zap.NewNop().Sugar(), nil)
	cc.path = file
	if err := cc.Load(); err != nil {
		t.Fatal(err)
	}
	s := cc.Snapshot()
	s.SliderMapping[0][0] = "changed"
	if cc.Snapshot().SliderMapping[0][0] != "master" {
		t.Fatal("snapshot aliases live state")
	}
	os.WriteFile(file, []byte("slider_mapping: [broken"), 0600)
	if err := cc.Load(); err == nil {
		t.Fatal("invalid reload succeeded")
	}
	if cc.Snapshot().SliderMapping[0][0] != "master" {
		t.Fatal("invalid reload replaced working config")
	}
}
func TestWatcherSeesAtomicReplacementAndLatestBurst(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	s := DefaultSettings()
	if err := SaveSettings(file, s); err != nil {
		t.Fatal(err)
	}
	cc, _ := NewConfig(zap.NewNop().Sugar(), nil)
	cc.path = file
	cc.Load()
	go cc.WatchConfigFileChanges()
	defer cc.StopWatchingConfigFile()
	// Poll saves until watcher setup and the replacement debounce have both completed.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s.BaudRate = 115200
		if err := SaveSettings(file, s); err != nil {
			t.Fatal(err)
		}
		time.Sleep(150 * time.Millisecond)
		if cc.Snapshot().BaudRate == 115200 {
			return
		}
	}
	t.Fatal("atomic replacement never reloaded")
}
