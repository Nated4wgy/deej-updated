package deej

import (
	"testing"
	"time"
)

func TestSerialStreamWatchdog(t *testing.T) {
	opened := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name             string
		lastValid        time.Time
		elapsed          time.Duration
		previouslyStreamed bool
		want             bool
	}{
		{"unproven idle controller remains open", time.Time{}, 30 * time.Second, false, false},
		{"newly opened known controller gets grace period", time.Time{}, 7 * time.Second, true, false},
		{"known controller silent after reopening", time.Time{}, 9 * time.Second, true, true},
		{"healthy stream", opened.Add(4 * time.Second), 7 * time.Second, true, false},
		{"stalled established stream", opened.Add(2 * time.Second), 8 * time.Second, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := serialStreamStalled(opened, tc.lastValid, opened.Add(tc.elapsed), tc.previouslyStreamed); got != tc.want {
				t.Fatalf("stalled=%v, want %v", got, tc.want)
			}
		})
	}
}

// Wake and manual requests may arrive together. The single buffered event
// avoids goroutine buildup, UI pauses and serial Open/Close races.
func TestSerialReconnectRequestsCoalesce(t *testing.T) {
	s := &SerialIO{reconnect: make(chan struct{}, 1)}
	for i := 0; i < 1000; i++ {
		s.Reconnect()
	}
	if n := len(s.reconnect); n != 1 {
		t.Fatalf("got %d queued reconnects, want 1", n)
	}
	<-s.reconnect
	if n := len(s.reconnect); n != 0 {
		t.Fatalf("got %d queued reconnects after draining", n)
	}
}
