package deej

import (
	"fmt"
	"github.com/omriharel/deej/pkg/deej/util"
	"go.uber.org/zap"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	masterSessionName            = "master"
	systemSessionName            = "system"
	inputSessionName             = "mic"
	specialTargetTransformPrefix = "deej."
	specialTargetCurrentWindow   = "current"
	specialTargetAllUnmapped     = "unmapped"
)

type AudioStatus struct {
	Targets           []string `json:"targets"`
	Error             string   `json:"error,omitempty"`
	ConfigRevision    string   `json:"config_revision"`
	ConfigError       string   `json:"config_error,omitempty"`
	EffectiveSettings Settings `json:"effective_settings"`
	LastApplyMS       float64  `json:"last_apply_ms"`
	P95ApplyMS        float64  `json:"p95_apply_ms"`
	MaxApplyMS        float64  `json:"max_apply_ms"`
	LastDiscoveryMS   float64  `json:"last_discovery_ms"`
}
type sessionMap struct {
	foreground        func() ([]string, error)
	foregroundTargets map[int][]string
	deej              *Deej
	logger            *zap.SugaredLogger
	sessionFinder     SessionFinder
	m                 map[string][]Session // accessed only by the audio worker
	desired           map[int]SliderMoveEvent
	config            Settings
	configRevision    string
	refresh           chan bool
	stop              chan struct{}
	done              chan struct{}
	stopOnce          sync.Once
	reload            chan bool
	statusMu          sync.Mutex
	status            AudioStatus
	samples           []float64
	lastRefresh       time.Time
}

func newSessionMap(d *Deej, logger *zap.SugaredLogger, finder SessionFinder) (*sessionMap, error) {
	return &sessionMap{foreground: util.GetCurrentWindowProcessNames, foregroundTargets: map[int][]string{}, deej: d, logger: logger.Named("sessions"), sessionFinder: finder, m: map[string][]Session{}, desired: map[int]SliderMoveEvent{}, refresh: make(chan bool, 1), stop: make(chan struct{}), done: make(chan struct{}), reload: d.config.SubscribeToChanges()}, nil
}
func (m *sessionMap) Status() AudioStatus {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	s := m.status
	s.Targets = append([]string(nil), s.Targets...)
	s.EffectiveSettings = cloneSettings(s.EffectiveSettings)
	return s
}
func (m *sessionMap) initialize() error {
	ready := make(chan error, 1)
	go m.run(ready)
	return <-ready
}
func (m *sessionMap) release() error { m.stopOnce.Do(func() { close(m.stop) }); <-m.done; return nil }
func (m *sessionMap) refreshSessions(force bool) {
	select {
	case m.refresh <- force:
	default:
	}
}
func (m *sessionMap) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(m.done)
	if err := initializeAudioThread(); err != nil {
		ready <- err
		return
	}
	defer uninitializeAudioThread()
	defer m.sessionFinder.Release()
	defer m.clear()
	m.loadConfig()
	m.discover()
	ready <- nil
	// Discovery is bounded in frequency and never blocks serial ingestion. Core Audio pointers
	// remain on this initialized MTA thread throughout discovery, application and release.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		default:
		}
		select {
		case <-m.stop:
			return
		case <-m.reload:
			m.loadConfig()
			m.reapply()
		case force := <-m.refresh:
			if force || time.Since(m.lastRefresh) >= time.Second {
				m.discover()
				m.reapply()
			}
		case <-m.deej.serial.pending.notify:
			for _, event := range m.deej.serial.pending.take() {
				m.desired[event.SliderID] = event
				m.apply(event, true)
			}
		case <-ticker.C:
			changed := false
			if f, ok := m.sessionFinder.(interface{ DevicesChanged() bool }); ok {
				changed = f.DevicesChanged()
			}
			if changed || time.Since(m.lastRefresh) >= 5*time.Second {
				m.discover()
				m.reapply()
			}
		}
	}
}
func (m *sessionMap) loadConfig() {
	m.foregroundTargets = map[int][]string{}
	m.config, m.configRevision, _ = m.deej.config.SnapshotState()
	if m.configRevision == "" {
		m.config.SliderMapping = map[int][]string{}
	}
	m.statusMu.Lock()
	m.status.ConfigRevision = m.configRevision
	m.status.EffectiveSettings = cloneSettings(m.config)
	m.statusMu.Unlock()
}
func (m *sessionMap) discover() {
	start := time.Now()
	m.lastRefresh = start
	sessions, err := m.sessionFinder.GetAllSessions()
	if err != nil {
		m.statusMu.Lock()
		m.status.Error = err.Error()
		m.status.LastDiscoveryMS = float64(time.Since(start).Microseconds()) / 1000
		m.statusMu.Unlock()
		m.logger.Warnw("Audio discovery failed; keeping usable targets", "error", err)
		return
	}
	m.clear()
	for _, s := range sessions {
		key := strings.ToLower(s.Key())
		m.m[key] = append(m.m[key], s)
	}
	keys := make([]string, 0, len(m.m))
	for key := range m.m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	m.statusMu.Lock()
	m.status.Targets = keys
	m.status.Error = ""
	m.status.LastDiscoveryMS = float64(time.Since(start).Microseconds()) / 1000
	m.statusMu.Unlock()
}
func (m *sessionMap) clear() {
	for _, sessions := range m.m {
		for _, s := range sessions {
			s.Release()
		}
	}
	m.m = map[string][]Session{}
}
func (m *sessionMap) reapply() {
	// Do not replay an old value ahead of a newer sample received during discovery.
	for _, event := range m.deej.serial.pending.take() {
		m.desired[event.SliderID] = event
		delete(m.foregroundTargets, event.SliderID)
	}
	for id := 0; id < maxSliders; id++ {
		if event, ok := m.desired[id]; ok {
			m.apply(event, false)
		}
	}
}
func (m *sessionMap) apply(event SliderMoveEvent, measure bool) {
	if measure {
		delete(m.foregroundTargets, event.SliderID)
	}
	applied := false
	for _, target := range m.config.SliderMapping[event.SliderID] {
		keys := m.resolveTarget(target)
		if strings.EqualFold(target, "deej.current") {
			if previous, ok := m.foregroundTargets[event.SliderID]; ok {
				keys = previous
			} else {
				m.foregroundTargets[event.SliderID] = append([]string(nil), keys...)
			}
		}
		for _, key := range keys {
			for _, session := range m.m[key] {
				// Events already pass a noise filter. Avoid a synchronous GetVolume before every Set.
				if err := session.SetVolume(event.PercentValue); err != nil {
					m.statusMu.Lock()
					m.status.Error = err.Error()
					m.statusMu.Unlock()
					m.refreshSessions(false)
				} else {
					applied = true
				}
			}
		}
	}
	if measure && applied && !event.ReceivedAt.IsZero() {
		elapsed := float64(time.Since(event.ReceivedAt).Microseconds()) / 1000
		m.samples = append(m.samples, elapsed)
		if len(m.samples) > 256 {
			m.samples = m.samples[1:]
		}
		sorted := append([]float64(nil), m.samples...)
		sort.Float64s(sorted)
		m.statusMu.Lock()
		m.status.LastApplyMS = elapsed
		m.status.P95ApplyMS = sorted[(len(sorted)-1)*95/100]
		if elapsed > m.status.MaxApplyMS {
			m.status.MaxApplyMS = elapsed
		}
		m.statusMu.Unlock()
	}
}
func (m *sessionMap) resolveTarget(target string) []string {
	target = strings.ToLower(target)
	switch target {
	case "deej.current":
		names, err := m.foreground()
		if err != nil {
			return nil
		}
		for i := range names {
			names[i] = strings.ToLower(names[i])
		}
		return names
	case "deej.unmapped":
		mapped := map[string]bool{"master": true, "mic": true, "system": true}
		for _, targets := range m.config.SliderMapping {
			for _, t := range targets {
				mapped[strings.ToLower(t)] = true
			}
		}
		var keys []string
		for key, sessions := range m.m {
			if mapped[key] {
				continue
			}
			if len(sessions) > 0 {
				if endpoint, ok := sessions[0].(interface{ IsMaster() bool }); ok && endpoint.IsMaster() {
					continue
				}
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return keys
	default:
		return []string{target}
	}
}
func (m *sessionMap) String() string {
	return fmt.Sprintf("<%d audio targets>", len(m.Status().Targets))
}
