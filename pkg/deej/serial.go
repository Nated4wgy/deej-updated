package deej

import (
	"errors"
	"fmt"
	"github.com/omriharel/deej/pkg/deej/util"
	"go.bug.st/serial"
	"go.uber.org/zap"

	"strconv"
	"strings"
	"sync"
	"time"
)

type SliderMoveEvent struct {
	SliderID     int
	PercentValue float32
	ReceivedAt   time.Time
}

// latestSliders bounds pending work to one value per physical slider, never a FIFO of old movements.
type latestSliders struct {
	mu     sync.Mutex
	values map[int]SliderMoveEvent
	notify chan struct{}
}

func newLatestSliders() *latestSliders {
	return &latestSliders{values: map[int]SliderMoveEvent{}, notify: make(chan struct{}, 1)}
}
func (q *latestSliders) publish(e SliderMoveEvent) {
	q.mu.Lock()
	q.values[e.SliderID] = e
	q.mu.Unlock()
	select {
	case q.notify <- struct{}{}:
	default:
	}
}
func (q *latestSliders) take() []SliderMoveEvent {
	q.mu.Lock()
	defer q.mu.Unlock()
	events := make([]SliderMoveEvent, 0, len(q.values))
	for id := 0; id < maxSliders; id++ {
		if e, ok := q.values[id]; ok {
			events = append(events, e)
		}
	}
	q.values = map[int]SliderMoveEvent{}
	return events
}

type SerialStatus struct {
	Connected bool      `json:"connected"`
	Port      string    `json:"port"`
	Error     string    `json:"error,omitempty"`
	Positions []float32 `json:"positions"`
	LastFrame time.Time `json:"last_frame"`
}
type SerialIO struct {
	deej    *Deej
	logger  *zap.SugaredLogger
	pending *latestSliders
	mu      sync.Mutex
	port    serial.Port
	status  SerialStatus
	stop    chan struct{}
	done    chan struct{}
	started bool
	reload  chan bool

	// reconnect is a coalescing request channel. Only the serial owner goroutine
	// opens and closes ports; callers never race an in-flight Read with Close.
	reconnect chan struct{}
	// everStreamed remembers whether this controller has actually provided
	// frames, so a change-only firmware is not repeatedly reconnected on idle.
	everStreamed bool
}

func NewSerialIO(d *Deej, logger *zap.SugaredLogger) (*SerialIO, error) {
	return &SerialIO{deej: d, logger: logger.Named("serial"), pending: newLatestSliders(), stop: make(chan struct{}), done: make(chan struct{}), reload: d.config.SubscribeToChanges(), reconnect: make(chan struct{}, 1)}, nil
}
func (s *SerialIO) Status() SerialStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := s.status
	result.Positions = append([]float32(nil), s.status.Positions...)
	return result
}
// Reconnect asks the serial owner to reopen the currently configured port.
// A full channel means a reconnect is already pending; never block the tray
// or a Windows power notification callback while trying to request another.
func (s *SerialIO) Reconnect() {
	select {
	case s.reconnect <- struct{}{}:
	default:
	}
}

func (s *SerialIO) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return fmt.Errorf("serial already started")
	}
	s.started = true
	go s.run()
	return nil
}
func (s *SerialIO) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	// The connection owner closes after its bounded read timeout; no concurrent Close/Read.
	s.mu.Unlock()
	<-s.done
}
func (s *SerialIO) run() {
	defer close(s.done)

	// Windows reports resume even if the old COM handle still looks valid.
	// Never close it from that callback: wake the owner and let its bounded
	// Read finish before it releases and reopens the connection.
	unsubscribe, subscriptionErr := subscribeSystemResume(s.Reconnect)
	if subscriptionErr != nil {
		s.logger.Warnw("Windows resume notifications unavailable; serial watchdog still active", "error", subscriptionErr)
	} else {
		defer unsubscribe()
	}

	backoff := time.Second
	for {
		select {
		case <-s.stop:
			return
		default:
		}
		cfg := s.deej.config.Snapshot()
		p, err := serial.Open(cfg.COMPort, &serial.Mode{BaudRate: cfg.BaudRate})
		if err != nil {
			s.mu.Lock()
			s.status.Connected = false
			s.status.Port = cfg.COMPort
			s.status.Error = err.Error()
			s.mu.Unlock()
			s.logger.Warnw("Serial connection unavailable; retrying", "error", err)
			select {
			case <-s.stop:
				return
			case <-s.reload:
			case <-s.reconnect:
				backoff = time.Second
			case <-time.After(backoff):
			}
			if backoff < 5*time.Second {
				backoff += time.Second
			}
			continue
		}
		if err = p.SetReadTimeout(100 * time.Millisecond); err != nil {
			p.Close()
			s.logger.Warnw("Set serial timeout", "error", err)
			select {
			case <-s.stop:
				return
			case <-time.After(time.Second):
			}
			continue
		}
		s.mu.Lock()
		select {
		case <-s.stop:
			s.mu.Unlock()
			p.Close()
			return
		default:
		}
		s.port = p
		s.status = SerialStatus{Connected: true, Port: cfg.COMPort}
		s.mu.Unlock()
		backoff = time.Second
		s.logger.Infow("Serial connected", "port", cfg.COMPort)
		err = s.readConnection(p, cfg)
		// The connection owner is the only goroutine calling Close. This is
		// important on Windows: a concurrent Close and overlapped Read can
		// interfere with the serial driver during USB sleep/wake transitions.
		s.mu.Lock()
		_ = p.Close()
		s.port = nil
		s.status.Connected = false
		if err != nil {
			s.status.Error = err.Error()
		} else {
			s.status.Error = ""
		}
		s.mu.Unlock()
		if err != nil && !errors.Is(err, errReconnectRequested) {
			s.logger.Warnw("Serial connection interrupted; reopening", "port", cfg.COMPort, "error", err)
		}

		// Windows needs time to re-enumerate a USB controller after resume.
		// Initial / normal retries remain fast; the explicit resume path
		// waits a little longer before reopening that COM port.
		delay := 100 * time.Millisecond
		if errors.Is(err, errReconnectRequested) {
			delay = 1200 * time.Millisecond
		}
		select {
		case <-s.stop:
			return
		case <-s.reload:
		case <-time.After(delay):
		}
	}
}

// A bounded incremental decoder survives partial, oversized and malformed lines.
type frameDecoder struct {
	line    []byte
	discard bool
}

func (d *frameDecoder) feed(data []byte, emit func([]int)) {
	for _, b := range data {
		if b == '\n' {
			if !d.discard {
				if values, err := parseSliderFrame(strings.TrimSuffix(string(d.line), "\r")); err == nil {
					emit(values)
				}
			}
			d.line = d.line[:0]
			d.discard = false
			continue
		}
		if d.discard {
			continue
		}
		if len(d.line) >= maxSliders*5 {
			d.line = d.line[:0]
			d.discard = true
			continue
		}
		d.line = append(d.line, b)
	}
}
func parseSliderFrame(line string) ([]int, error) {
	parts := strings.Split(line, "|")
	if len(parts) > maxSliders || line == "" {
		return nil, fmt.Errorf("invalid frame size")
	}
	values := make([]int, len(parts))
	for i, p := range parts {
		if len(p) < 1 || len(p) > 4 {
			return nil, fmt.Errorf("invalid sample")
		}
		for _, b := range p {
			if b < '0' || b > '9' {
				return nil, fmt.Errorf("invalid sample")
			}
		}
		n, err := strconv.Atoi(p)
		if err != nil || n > 1023 {
			return nil, fmt.Errorf("sample out of range")
		}
		values[i] = n
	}
	return values, nil
}
const (
	// The stock firmware emits frames approximately every 10 ms. Several
	// seconds without a valid frame means the connection is almost certainly
	// stalled, not that the user has left the sliders untouched.
	serialFrameStallTimeout = 5 * time.Second
	// When re-opening a previously active controller, also detect a COM
	// handle that opens successfully but never delivers its first frame.
	serialInitialFrameTimeout = 8 * time.Second
)

var (
	errReconnectRequested = errors.New("serial reconnect requested")
	errSerialStreamStalled = errors.New("serial controller stopped sending valid frames")
)

// serialStreamStalled deliberately does NOT time out the first-ever open of a
// controller that has never streamed. Some third-party firmwares send only on
// changes, and we should not constantly reset those while they are idle.
func serialStreamStalled(openedAt, lastValidFrame, now time.Time, previouslyStreamed bool) bool {
	if !lastValidFrame.IsZero() {
		return now.Sub(lastValidFrame) >= serialFrameStallTimeout
	}
	return previouslyStreamed && now.Sub(openedAt) >= serialInitialFrameTimeout
}

func (s *SerialIO) readConnection(p serial.Port, cfg Settings) error {
	var decoder frameDecoder
	var previous []float32
	buffer := make([]byte, 256)
	openedAt := time.Now()
	var lastValidFrame time.Time
	// Only the serial owner reads/writes everStreamed, avoiding synchronization
	// with the UI/status polling and preserving the existing audio pipeline.
	previouslyStreamed := s.everStreamed
	for {
		select {
		case <-s.stop:
			return nil
		case <-s.reconnect:
			return errReconnectRequested
		case <-s.reload:
			next := s.deej.config.Snapshot()
			if next.COMPort != cfg.COMPort || next.BaudRate != cfg.BaudRate {
				return nil
			}
			cfg = next
			previous = nil
		default:
		}
		n, err := p.Read(buffer)
		if err != nil {
			return err
		}
		if n == 0 {
			if serialStreamStalled(openedAt, lastValidFrame, time.Now(), previouslyStreamed) {
				return errSerialStreamStalled
			}
			continue
		}
		decoder.feed(buffer[:n], func(raw []int) {
			received := time.Now()
			lastValidFrame = received
			s.everStreamed = true
			if len(previous) != len(raw) {
				previous = make([]float32, len(raw))
				for i := range previous {
					previous[i] = -1
				}
			}
			positions := make([]float32, len(raw))
			for id, v := range raw {
				value := util.NormalizeScalar(float32(v) / 1023)
				if cfg.InvertSliders {
					value = 1 - value
				}
				positions[id] = value
				if util.SignificantlyDifferent(previous[id], value, cfg.NoiseReductionLevel) {
					previous[id] = value
					s.pending.publish(SliderMoveEvent{id, value, received})
				}
			}
			s.mu.Lock()
			s.status.Positions = positions
			s.status.LastFrame = received
			s.mu.Unlock()
		})
		if serialStreamStalled(openedAt, lastValidFrame, time.Now(), previouslyStreamed) {
			return errSerialStreamStalled
		}
	}
}
