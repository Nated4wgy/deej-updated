package deej

import (
	"strings"
	"testing"
	"time"
)

func TestFrameDecoderResynchronizes(t *testing.T) {
	var d frameDecoder
	var frames [][]int
	emit := func(v []int) { frames = append(frames, v) }
	d.feed([]byte("0|10"), emit)
	d.feed([]byte("23\r\n1|9999\n2|3\n"+strings.Repeat("9", 500)+"\n4|5\n"), emit)
	if len(frames) != 3 || frames[0][1] != 1023 || frames[1][0] != 2 || frames[2][0] != 4 {
		t.Fatalf("%v", frames)
	}
	for _, line := range []string{"", "-1|0", "0|1024", "0||2", "0|+1", strings.Repeat("0|", 32) + "0"} {
		if _, err := parseSliderFrame(line); err == nil {
			t.Errorf("accepted %q", line)
		}
	}
}
func TestLatestSlidersDoesNotWaitForAudio(t *testing.T) {
	q := newLatestSliders()
	done := make(chan struct{})
	go func() {
		for n := 0; n < 100000; n++ {
			q.publish(SliderMoveEvent{SliderID: n % 5, PercentValue: float32(n)})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("producer blocked on slow consumer")
	}
	events := q.take()
	if len(events) != 5 {
		t.Fatalf("%d pending events", len(events))
	}
	for _, e := range events {
		if e.PercentValue != float32(99995+e.SliderID) {
			t.Fatalf("stale event %+v", e)
		}
	}
}
