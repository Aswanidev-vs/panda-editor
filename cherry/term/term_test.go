package term

import (
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/Aswanidev-vs/cherry/input"
)

// openOrSkip returns a TTY if the process is attached to a real terminal;
// otherwise it reports the rejection so the caller can skip.
func openOrSkip(t *testing.T) *TTY {
	t.Helper()
	tty, err := Open()
	if err != nil {
		if !errors.Is(err, ErrNotATerminal) {
			t.Fatalf("Open() = %v, want ErrNotATerminal or success", err)
		}
		t.Skip("stdin/stdout are redirected; interactive terminal required")
	}
	return tty
}

// TestOpenRejectsRedirectedStreams covers the CI path: under `go test`,
// stdin/stdout are pipes, so Open must refuse cleanly.
func TestOpenRejectsRedirectedStreams(t *testing.T) {
	tty, err := Open()
	if err == nil {
		tty.Close()
		t.Skip("attached to a real terminal; rejection path not exercisable")
	}
	if !errors.Is(err, ErrNotATerminal) {
		t.Fatalf("Open() error = %v, want ErrNotATerminal", err)
	}
}

// TestNoGoroutineLeakOnFailedOpen pins the lifecycle contract that failed
// Opens leave nothing running behind.
func TestNoGoroutineLeakOnFailedOpen(t *testing.T) {
	before := runtime.NumGoroutine()
	for i := 0; i < 10; i++ {
		tty, err := Open()
		if err == nil {
			tty.Close()
			t.Skip("attached to a real terminal; leak test targets the non-tty path")
		}
	}
	time.Sleep(100 * time.Millisecond)
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Fatalf("goroutines grew from %d to %d after repeated failed Opens", before, after)
	}
}

// TestSendEventPolicy verifies the watcher-side guarantees without needing a
// console: overflow drops instead of blocking, and stop wins over delivery.
func TestSendEventPolicy(t *testing.T) {
	tty := &TTY{events: make(chan input.Event, 1)}
	stop := make(chan struct{})

	tty.sendEvent(input.Resize{Width: 1, Height: 1}, stop)
	tty.sendEvent(input.Resize{Width: 2, Height: 2}, stop) // full buffer: must drop, not block

	ev := <-tty.events
	r, ok := ev.(input.Resize)
	if !ok || r.Width != 1 || r.Height != 1 {
		t.Fatalf("first buffered event = %#v, want Resize{1 1}", ev)
	}

	close(stop)
	done := make(chan struct{})
	go func() {
		tty.sendEvent(input.Resize{}, stop)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sendEvent blocked on a stopped stream")
	}
	select {
	case ev := <-tty.events:
		t.Fatalf("stopped sendEvent still delivered %#v", ev)
	default:
	}
}

// TestLifecycleOnRealTTY exercises the raw-mode lifecycle end to end. It only
// runs when invoked from a real terminal (go test ./term in a console).
func TestLifecycleOnRealTTY(t *testing.T) {
	tty := openOrSkip(t)
	defer tty.Close()

	if err := tty.Start(); err != nil {
		t.Fatalf("Start(): %v", err)
	}
	if err := tty.Start(); err != nil {
		t.Fatalf("idempotent Start(): %v", err)
	}

	sz, err := tty.Size()
	if err != nil {
		t.Fatalf("Size(): %v", err)
	}
	if sz.W <= 0 || sz.H <= 0 {
		t.Fatalf("Size() = %+v, want positive dimensions", sz)
	}
	_ = tty.VTSupported() // must not panic; value is platform-dependent

	if tty.Events() == nil {
		t.Fatal("Events() returned nil channel")
	}

	tty.Restore()
	tty.Restore() // double restore must be a no-op, not a panic

	if err := tty.Start(); err != nil {
		t.Fatalf("Start() after Restore(): %v", err)
	}

	if err := tty.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}
	if err := tty.Close(); err != nil {
		t.Fatalf("idempotent Close(): %v", err)
	}
	tty.Restore() // safe even after Close

	if err := tty.Start(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Start() after Close() = %v, want ErrClosed", err)
	}

	deadline := time.After(time.Second)
	for {
		select {
		case _, open := <-tty.Events():
			if !open {
				return // closed as promised; leftover pre-close events were drained
			}
		case <-deadline:
			t.Fatal("Events channel was not closed by Close()")
		}
	}
}
