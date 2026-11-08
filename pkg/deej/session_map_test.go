package deej

import (
	"fmt"
	"go.uber.org/zap"
	"sync"
	"testing"
	"time"
)

type fakeSession struct {
	mu       sync.Mutex
	value    float32
	released bool
	entered  chan struct{}
	unblock  chan struct{}
	once     sync.Once
}

func (s *fakeSession) Key() string        { return "master" }
func (s *fakeSession) GetVolume() float32 { s.mu.Lock(); defer s.mu.Unlock(); return s.value }
func (s *fakeSession) SetVolume(v float32) error {
	if s.entered != nil {
		s.once.Do(func() { close(s.entered); <-s.unblock })
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.released {
		return fmt.Errorf("used released session")
	}
	s.value = v
	return nil
}
func (s *fakeSession) Release() { s.mu.Lock(); s.released = true; s.mu.Unlock() }

type fakeFinder struct {
	mu      sync.Mutex
	current *fakeSession
	fail    bool
	first   *fakeSession
}

func (f *fakeFinder) GetAllSessions() ([]Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return nil, fmt.Errorf("discovery failed")
	}
	if f.first != nil {
		f.current = f.first
		f.first = nil
	} else {
		f.current = &fakeSession{}
	}
	return []Session{f.current}, nil
}
func (f *fakeFinder) Release() error { return nil }
func TestWorkerCoalescesAndSerializesRefresh(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cc, _ := NewConfig(logger, nil)
	cc.revision = "test"
	d := &Deej{config: cc}
	d.serial, _ = NewSerialIO(d, logger)
	first := &fakeSession{entered: make(chan struct{}), unblock: make(chan struct{})}
	finder := &fakeFinder{first: first}
	m, _ := newSessionMap(d, logger, finder)
	if err := m.initialize(); err != nil {
		t.Fatal(err)
	}
	defer m.release()
	d.serial.pending.publish(SliderMoveEvent{0, 0.1, time.Now()})
	select {
	case <-first.entered:
	case <-time.After(time.Second):
		t.Fatal("worker did not receive initial event")
	}
	for n := 0; n < 10000; n++ {
		d.serial.pending.publish(SliderMoveEvent{0, float32(n) / 9999, time.Now()})
	}
	m.refreshSessions(true)
	first.mu.Lock()
	released := first.released
	first.mu.Unlock()
	if released {
		t.Fatal("released session during a volume call")
	}
	close(first.unblock)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		finder.mu.Lock()
		current := finder.current
		finder.mu.Unlock()
		if current != first && current.GetVolume() == 1 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("final slider value was lost after slow audio/refresh")
}
func TestDiscoveryFailureRetainsWorkingTargets(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cc, _ := NewConfig(logger, nil)
	cc.revision = "test"
	d := &Deej{config: cc}
	d.serial, _ = NewSerialIO(d, logger)
	finder := &fakeFinder{}
	m, _ := newSessionMap(d, logger, finder)
	m.loadConfig()
	m.discover()
	old := finder.current
	finder.fail = true
	m.discover()
	m.apply(SliderMoveEvent{0, 0.8, time.Now()}, true)
	if old.GetVolume() != 0.8 {
		t.Fatal("working target lost")
	}
	m.clear()
}

func TestRefreshDoesNotRetargetStationaryForegroundSlider(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cc, _ := NewConfig(logger, nil)
	cc.current.SliderMapping = map[int][]string{0: {"deej.current"}}
	cc.revision = "test"
	d := &Deej{config: cc}
	d.serial, _ = NewSerialIO(d, logger)
	m, _ := newSessionMap(d, logger, &fakeFinder{})
	m.loadConfig()
	first, second := &fakeSession{}, &fakeSession{}
	m.m = map[string][]Session{"first.exe": {first}, "second.exe": {second}}
	focused := "first.exe"
	m.foreground = func() ([]string, error) { return []string{focused}, nil }
	event := SliderMoveEvent{0, 0.4, time.Now()}
	m.desired[0] = event
	m.apply(event, true)
	focused = "second.exe"
	m.reapply()
	if first.GetVolume() != 0.4 || second.GetVolume() != 0 {
		t.Fatal("background refresh changed the newly focused app without a slider move")
	}
	m.apply(SliderMoveEvent{0, 0.8, time.Now()}, true)
	if second.GetVolume() != 0.8 {
		t.Fatal("a new slider movement did not target the foreground app")
	}
}
