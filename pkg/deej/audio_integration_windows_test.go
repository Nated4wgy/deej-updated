package deej

import (
	"go.uber.org/zap"
	"os"
	"testing"
	"time"
)

// Opt-in: enumerate and release real Windows audio objects without changing volume or opening serial.
func TestWindowsAudioIntegration(t *testing.T) {
	if os.Getenv("DEEJ_AUDIO_INTEGRATION") != "1" {
		t.Skip("set DEEJ_AUDIO_INTEGRATION=1 for a read-only audio check")
	}
	logger := zap.NewNop().Sugar()
	cc, _ := NewConfig(logger, nil)
	d := &Deej{config: cc}
	d.serial, _ = NewSerialIO(d, logger)
	finder, err := newSessionFinder(logger)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := newSessionMap(d, logger, finder)
	if err = m.initialize(); err != nil {
		t.Fatal(err)
	}
	defer m.release()
	status := m.Status()
	if status.Error != "" {
		t.Fatal(status.Error)
	}
	t.Logf("%d targets discovered in %.2f ms", len(status.Targets), status.LastDiscoveryMS)
	m.refreshSessions(true)
	time.Sleep(150 * time.Millisecond)
	if status = m.Status(); status.Error != "" {
		t.Fatal(status.Error)
	}
}
