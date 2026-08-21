package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/cherry/cell"
)

// Regression: New() must hand the frame buffer to the sgrWriter, or the
// first styled cell panics on a nil receiver.
func TestFlushStyledCellNoPanic(t *testing.T) {
	s := New(20, 3)
	s.SetColorMode(ColorRGB)
	st := cell.Style{}.Foreground(cell.RGB(10, 200, 30)).Bold(true)
	s.Print(0, 1, 20, "ok 漢", st)

	var out bytes.Buffer
	if err := s.Flush(&out); err != nil {
		t.Fatalf("flush: %v", err)
	}
	frame := out.String()
	if !strings.Contains(frame, "\x1b[0m") || !strings.Contains(frame, "38;2;10;200;30") {
		t.Fatalf("missing baseline reset or truecolor fg in %q", frame)
	}
	if !strings.Contains(frame, "ok 漢") {
		t.Fatalf("runes missing from frame %q", frame)
	}

	// Second flush with nothing repainted must stay silent.
	out.Reset()
	if err := s.Flush(&out); err != nil {
		t.Fatalf("second flush: %v", err)
	}
	if strings.Count(frame, "?2026h") > 0 && strings.Contains(out.String(), "38;2") {
		t.Fatal("unchanged frame re-emitted styles")
	}
}
