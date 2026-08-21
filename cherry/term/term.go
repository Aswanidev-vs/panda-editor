// Package term is cherry's terminal backend: it owns raw-mode entry/exit,
// blocking byte I/O against the console, size queries, and asynchronous
// resize notification. Open refuses redirected streams so tests and CI can
// detect the absence of a real tty and skip interactive paths.
package term

import (
	"errors"
	"io"
	"os"
	"sync"

	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
)

// eventBuffer bounds the async event queue. Overflow drops instead of
// blocking: a resize burst must never stall its producer goroutine.
const eventBuffer = 8

var (
	// ErrNotATerminal is returned by Open when stdin or stdout is redirected;
	// callers check errors.Is to skip tty-dependent work.
	ErrNotATerminal = errors.New("term: stdin/stdout does not refer to a terminal")
	// ErrClosed is returned by Start on a TTY after Close.
	ErrClosed = errors.New("term: tty is closed")
)

var (
	_ io.Reader = (*TTY)(nil)
	_ io.Writer = (*TTY)(nil)
)

// TTY is an exclusive handle on the controlling terminal. Lifecycle state
// below is shared across platforms; OS handles live in platformTTY (see the
// build-tagged files).
//
// Locking invariant: goroutines spawned by Start must never acquire t.mu.
// They touch only state fixed at Open time plus sendEvent, which is what
// allows Restore to wg.Wait while holding the lock.
type TTY struct {
	plat platformTTY

	mu          sync.Mutex
	rawActive   bool          // console currently in raw mode
	running     bool          // resize watcher alive
	closed      bool          // Close finished
	vtSupported bool          // VT sequences usable in both directions (downgraded on legacy Windows consoles)
	stopCh      chan struct{} // non-nil while running; closed to stop the watcher
	events      chan input.Event
	wg          sync.WaitGroup
	closeOnce   sync.Once
}

// Open wraps os.Stdin/os.Stdout as a TTY. It fails when either stream is not
// attached to a terminal (pipe, file) without mutating any console state, so
// a failed Open leaves the process environment untouched.
func Open() (*TTY, error) {
	plat, err := newPlatform(os.Stdin, os.Stdout)
	if err != nil {
		return nil, err
	}
	return &TTY{
		plat:        plat,
		vtSupported: true,
		events:      make(chan input.Event, eventBuffer),
	}, nil
}

// Start enters raw mode and starts the resize watcher. Idempotent while raw
// mode is active; after Restore it may be called again to re-enter raw mode.
func (t *TTY) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return ErrClosed
	}
	if !t.rawActive {
		if err := t.enterRaw(); err != nil {
			return err
		}
		t.rawActive = true
	}
	if !t.running {
		t.stopCh = make(chan struct{})
		stop := t.stopCh
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			t.watchResizes(stop)
		}()
		t.running = true
	}
	return nil
}

// Restore returns the console to its pre-Open state and stops background
// goroutines. Safe to call any number of times in any state.
func (t *TTY) Restore() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.rawActive {
		t.restoreRaw()
		t.rawActive = false
	}
	if t.running {
		close(t.stopCh)
		t.stopCh = nil
		t.wg.Wait()
		t.running = false
	}
}

// Close restores the terminal, stops background goroutines, and closes the
// Events channel. Idempotent. A Read already blocked in raw mode is not
// interrupted; callers shutting down should unblock their reader separately.
func (t *TTY) Close() error {
	t.closeOnce.Do(func() {
		t.Restore()
		// Safe ordering: Restore waited for every watcher to return, so no
		// send can race the channel close.
		close(t.events)
		t.mu.Lock()
		t.closed = true
		t.mu.Unlock()
	})
	return nil
}

// Events streams asynchronous terminal events: Resize, plus Interrupt on
// platforms where ^C cannot surface as a raw input byte (legacy Windows
// consoles). The channel is closed only by Close.
func (t *TTY) Events() <-chan input.Event { return t.events }

// VTSupported reports whether the console accepted virtual-terminal mode on
// both directions. Always true on Unix; false means legacy Windows conhost,
// where escape sequences will not round-trip and callers should degrade.
func (t *TTY) VTSupported() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.vtSupported
}

// Size returns the current viewport in character cells. Deliberately
// lock-free: resize watchers call it concurrently with lifecycle methods,
// and it reads only state fixed at Open time.
func (t *TTY) Size() (geom.Size, error) { return t.size() }

// Read implements io.Reader: blocking raw bytes from the console. Escape
// sequences and control codes — including ^C as 0x03 once raw mode has
// disabled signal processing — arrive verbatim for the input parser.
func (t *TTY) Read(p []byte) (int, error) { return os.Stdin.Read(p) }

// Write implements io.Writer, emitting raw bytes to stdout.
func (t *TTY) Write(p []byte) (int, error) { return os.Stdout.Write(p) }

// sendEvent delivers ev without ever blocking: stop is checked first so a
// shutting-down watcher exits promptly, and a full buffer drops the event
// because resize bursts must not stall the producer.
func (t *TTY) sendEvent(ev input.Event, stop <-chan struct{}) {
	select {
	case <-stop:
		return
	default:
	}
	select {
	case t.events <- ev:
	default:
	}
}
