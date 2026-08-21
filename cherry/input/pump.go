package input

import (
	"io"
	"time"
)

// escGap is how long Pump waits for the next byte before deciding a lone
// ESC is the entire keypress: terminals emit ESC-prefixed sequences as one
// burst, while a real ESC keypress stands alone in the stream.
const escGap = 40 * time.Millisecond

// Pump reads the terminal byte stream from r until r errors, parses it
// with an internal Parser and forwards every resulting event to out,
// merging events from extra (resizes, interrupts, synthetic keys, ...) in
// arrival order. When r returns an error — including io.EOF — Pump flushes
// any partially received sequence (a pending lone ESC becomes KeyEscape),
// calls onErr exactly once if non-nil, and exits.
//
// Contract: out MUST be buffered (>=64) and MUST be drained continuously
// for as long as the program runs. Pump never closes out; a send that
// cannot complete blocks the goroutine forever. Closing extra is allowed
// and merely detaches it. A nil extra is fine too.
func Pump(r io.Reader, extra <-chan Event, out chan<- Event, onErr func(error)) {
	go pump(r, extra, out, onErr)
}

func pump(r io.Reader, extra <-chan Event, out chan<- Event, onErr func(error)) {
	p := NewParser()

	type res struct {
		b   []byte
		err error
	}
	rc := make(chan res, 1)
	go func() {
		defer close(rc)
		buf := make([]byte, 512)
		for {
			n, err := r.Read(buf)
			b := make([]byte, n)
			copy(b, buf[:n])
			rc <- res{b, err}
			if err != nil {
				return
			}
		}
	}()

	// The parser cannot interrupt a blocking Read, so bytes arrive via rc
	// and a timer races them only while a sequence is Pending().
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	stopTimer := func() {
		if !timer.Stop() {
			select { // drain an already-fired timer without blocking
			case <-timer.C:
			default:
			}
		}
	}
	armTimer := func() {
		stopTimer()
		timer.Reset(escGap)
	}

	send := func(evs []Event) {
		for _, e := range evs {
			out <- e
		}
	}

	errDone := false
	fail := func(err error) {
		send(p.Flush()) // trailing events precede the failure report
		if onErr != nil && !errDone {
			errDone = true
			onErr(err)
		}
	}

	handleChunk := func(rr res) bool {
		for _, b := range rr.b {
			send(p.Feed(b))
		}
		if rr.err != nil {
			fail(rr.err)
			return false
		}
		if p.Pending() {
			armTimer()
		} else {
			stopTimer()
		}
		return true
	}

	extraCh := extra // nil disables the extra case entirely
	for {
		select {
		case rr, ok := <-rc:
			if !ok {
				fail(io.ErrUnexpectedEOF) // reader goroutine vanished
				return
			}
			if !handleChunk(rr) {
				return
			}
		case e, ok := <-extraCh:
			if !ok {
				extraCh = nil
				continue
			}
			out <- e
		case <-timer.C:
			// Bytes may already be queued behind the timer; prefer them
			// over declaring the escape lone.
			select {
			case rr, ok := <-rc:
				if !ok {
					fail(io.ErrUnexpectedEOF)
					return
				}
				if !handleChunk(rr) {
					return
				}
			default:
				stopTimer()
				send(p.Flush())
			}
		}
	}
}
