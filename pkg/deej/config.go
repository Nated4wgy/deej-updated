package deej

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const userConfigFilepath = "config.yaml"
const maxSliders = 32

type Settings struct {
	SliderMapping       map[int][]string `json:"slider_mapping"`
	COMPort             string           `json:"com_port"`
	BaudRate            int              `json:"baud_rate"`
	InvertSliders       bool             `json:"invert_sliders"`
	NoiseReductionLevel string           `json:"noise_reduction"`
}

func DefaultSettings() Settings {
	return Settings{map[int][]string{0: {"master"}}, "COM4", 9600, false, "default"}
}
func cloneSettings(s Settings) Settings {
	r := s
	r.SliderMapping = map[int][]string{}
	for id, t := range s.SliderMapping {
		r.SliderMapping[id] = append([]string(nil), t...)
	}
	return r
}
func (s Settings) Validate() error {
	if strings.TrimSpace(s.COMPort) == "" {
		return fmt.Errorf("choose a serial port")
	}
	if s.BaudRate < 50 || s.BaudRate > 2000000 {
		return fmt.Errorf("baud rate must be between 50 and 2000000")
	}
	switch s.NoiseReductionLevel {
	case "low", "default", "high":
	default:
		return fmt.Errorf("noise reduction must be low, default or high")
	}
	for id, targets := range s.SliderMapping {
		if id < 0 || id >= maxSliders {
			return fmt.Errorf("slider index %d must be between 0 and %d", id, maxSliders-1)
		}
		for _, target := range targets {
			if strings.TrimSpace(target) == "" {
				return fmt.Errorf("slider %d has an empty target", id)
			}
		}
	}
	return nil
}

// ParseSettings accepts scalar and list mappings without silently converting invalid indexes.
func ParseSettings(data []byte) (Settings, error) {
	s := DefaultSettings()
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return s, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return s, fmt.Errorf("configuration must be a YAML mapping")
	}
	seen := map[string]bool{}
	for i := 0; i < len(doc.Content[0].Content); i += 2 {
		key, node := doc.Content[0].Content[i].Value, doc.Content[0].Content[i+1]
		if seen[key] {
			return s, fmt.Errorf("duplicate key %q", key)
		}
		seen[key] = true
		var err error
		switch key {
		case "com_port":
			err = node.Decode(&s.COMPort)
		case "baud_rate":
			err = node.Decode(&s.BaudRate)
		case "invert_sliders":
			err = node.Decode(&s.InvertSliders)
		case "noise_reduction":
			err = node.Decode(&s.NoiseReductionLevel)
		case "slider_mapping":
			if node.Kind != yaml.MappingNode {
				return s, fmt.Errorf("slider_mapping must be a mapping")
			}
			s.SliderMapping = map[int][]string{}
			for j := 0; j < len(node.Content); j += 2 {
				id, e := strconv.Atoi(node.Content[j].Value)
				if e != nil {
					return s, fmt.Errorf("invalid slider index %q", node.Content[j].Value)
				}
				if _, ok := s.SliderMapping[id]; ok {
					return s, fmt.Errorf("duplicate slider index %d", id)
				}
				v := node.Content[j+1]
				var targets []string
				if v.Kind == yaml.ScalarNode && v.Tag == "!!str" {
					targets = []string{v.Value}
				} else if v.Kind == yaml.SequenceNode {
					if err = v.Decode(&targets); err != nil {
						return s, err
					}
				} else {
					return s, fmt.Errorf("slider %d requires a target or list", id)
				}
				for n := range targets {
					targets[n] = strings.TrimSpace(targets[n])
				}
				s.SliderMapping[id] = targets
			}
		}
		if err != nil {
			return s, fmt.Errorf("%s: %w", key, err)
		}
	}
	s.COMPort = strings.TrimSpace(s.COMPort)
	return s, s.Validate()
}
func SettingsRevision(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func ReadSettings(filename string) (Settings, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Settings{}, err
	}
	return ParseSettings(data)
}

// SaveSettings preserves unrelated YAML keys/comments and backs up the previous file.
func SaveSettings(filename string, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	data, err := os.ReadFile(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var doc yaml.Node
	if len(data) > 0 {
		if _, err = ParseSettings(data); err != nil {
			return fmt.Errorf("existing config is invalid; correct Advanced YAML first: %w", err)
		}
		if err = yaml.Unmarshal(data, &doc); err != nil {
			return err
		}
	} else {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	fields := []struct {
		key   string
		value interface{}
	}{{"slider_mapping", s.SliderMapping}, {"com_port", s.COMPort}, {"baud_rate", s.BaudRate}, {"invert_sliders", s.InvertSliders}, {"noise_reduction", s.NoiseReductionLevel}}
	mapping := doc.Content[0]
	for _, field := range fields {
		var value yaml.Node
		if err = value.Encode(field.value); err != nil {
			return err
		}
		found := false
		for i := 0; i < len(mapping.Content); i += 2 {
			if mapping.Content[i].Value == field.key {
				old := mapping.Content[i+1]
				value.HeadComment = old.HeadComment
				value.LineComment = old.LineComment
				value.FootComment = old.FootComment
				mapping.Content[i+1] = &value
				found = true
				break
			}
		}
		if !found {
			mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: field.key}, &value)
		}
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err = enc.Encode(&doc); err != nil {
		return err
	}
	enc.Close()
	if len(data) > 0 {
		if err = atomicWrite(filename+".bak", data); err != nil {
			return fmt.Errorf("backup config: %w", err)
		}
	}
	return atomicWrite(filename, out.Bytes())
}
func atomicWrite(filename string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(filename), ".deej-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err := replaceFile(name, filename); err != nil {
		return fmt.Errorf("replace %s with %s: %w", filename, name, err)
	}
	return nil
}

type CanonicalConfig struct {
	mu                        sync.RWMutex
	current                   Settings
	revision, lastError, path string
	logger                    *zap.SugaredLogger
	consumers                 []chan bool
	stop                      chan struct{}
	stopOnce                  sync.Once
}

func NewConfig(logger *zap.SugaredLogger, _ Notifier) (*CanonicalConfig, error) {
	return &CanonicalConfig{current: DefaultSettings(), path: userConfigFilepath, logger: logger.Named("config"), stop: make(chan struct{})}, nil
}
func (cc *CanonicalConfig) Snapshot() Settings {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cloneSettings(cc.current)
}
func (cc *CanonicalConfig) State() (string, string) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.revision, cc.lastError
}
func (cc *CanonicalConfig) Load() error {
	data, err := os.ReadFile(cc.path)
	var next Settings
	if err == nil {
		next, err = ParseSettings(data)
	}
	if err == nil {
		// Only merge mappings actually present in legacy preferences; other defaults are irrelevant.
		if prefs, e := os.ReadFile(filepath.Join(filepath.Dir(cc.path), "logs", "preferences.yaml")); e == nil {
			var raw map[string]yaml.Node
			if e = yaml.Unmarshal(prefs, &raw); e != nil {
				err = e
			} else if _, ok := raw["slider_mapping"]; ok {
				internal, e := ParseSettings(prefs)
				if e != nil {
					err = e
				} else {
					for id, targets := range internal.SliderMapping {
						for _, target := range targets {
							exists := false
							for _, v := range next.SliderMapping[id] {
								if strings.EqualFold(v, target) {
									exists = true
								}
							}
							if !exists {
								next.SliderMapping[id] = append(next.SliderMapping[id], target)
							}
						}
					}
				}
			}
		} else if !os.IsNotExist(e) {
			err = e
		}
	}
	cc.mu.Lock()
	if err != nil {
		cc.lastError = err.Error()
		cc.mu.Unlock()
		return err
	}
	changed := cc.revision != SettingsRevision(data)
	cc.current = cloneSettings(next)
	cc.revision = SettingsRevision(data)
	cc.lastError = ""
	cc.mu.Unlock()
	if changed {
		cc.notify()
	}
	return nil
}
func (cc *CanonicalConfig) SubscribeToChanges() chan bool {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	ch := make(chan bool, 1)
	cc.consumers = append(cc.consumers, ch)
	return ch
}
func (cc *CanonicalConfig) notify() {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	for _, ch := range cc.consumers {
		select {
		case ch <- true:
		default:
		}
	}
}
func (cc *CanonicalConfig) WatchConfigFileChanges() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cc.logger.Warnw("Config watcher unavailable", "error", err)
		return
	}
	defer watcher.Close()
	if err = watcher.Add(filepath.Dir(cc.path)); err != nil {
		cc.logger.Warnw("Watch config directory", "error", err)
		return
	}
	// Close the load-to-watch startup gap after subscribing to directory changes.
	if err := cc.Load(); err != nil {
		cc.logger.Debugw("Initial watched config unavailable", "error", err)
	}
	var timer *time.Timer
	var tick <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-cc.stop:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) == filepath.Base(cc.path) && event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
				if timer != nil {
					timer.Stop()
				}
				timer = time.NewTimer(100 * time.Millisecond)
				tick = timer.C
			}
		case <-tick:
			tick = nil
			if err := cc.Load(); err != nil {
				cc.logger.Warnw("Rejected config; keeping last good settings", "error", err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			cc.logger.Warnw("Config watcher error", "error", err)
		}
	}
}
func (cc *CanonicalConfig) StopWatchingConfigFile() { cc.stopOnce.Do(func() { close(cc.stop) }) }

// SnapshotState correlates the immutable settings and acknowledgement revision under one lock.
func (cc *CanonicalConfig) SnapshotState() (Settings, string, string) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cloneSettings(cc.current), cc.revision, cc.lastError
}

// RestoreSettingsBackup restores the exact validated backup, even if the current YAML is corrupt.
func RestoreSettingsBackup(filename string) error {
	data, err := os.ReadFile(filename + ".bak")
	if err != nil {
		return err
	}
	if _, err = ParseSettings(data); err != nil {
		return fmt.Errorf("backup is invalid: %w", err)
	}
	if current, err := os.ReadFile(filename); err == nil {
		if err = atomicWrite(filename+".before-restore", current); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return atomicWrite(filename, data)
}
