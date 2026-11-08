package deej

import (
	"go.uber.org/zap"
	"os"
	"testing"
	"time"
)

// Opt-in. Opens the specified controller but never creates audio sessions or changes volume.
func TestSerialControllerIntegration(t *testing.T) {
	port := os.Getenv("DEEJ_SERIAL_INTEGRATION")
	if port == "" {
		t.Skip("set DEEJ_SERIAL_INTEGRATION to the controller port")
	}
	logger := zap.NewNop().Sugar()
	cc, _ := NewConfig(logger, nil)
	cc.current.COMPort = port
	cc.current.SliderMapping = map[int][]string{}
	d := &Deej{config: cc}
	s, _ := NewSerialIO(d, logger)
	d.serial = s
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		status := s.Status()
		if len(status.Positions) > 0 {
			t.Logf("received %d sliders on %s", len(status.Positions), port)
			start := time.Now()
			s.Stop()
			if time.Since(start) > 500*time.Millisecond {
				t.Fatal("shutdown exceeded bounded timeout")
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no valid controller data: %+v", s.Status())
}
