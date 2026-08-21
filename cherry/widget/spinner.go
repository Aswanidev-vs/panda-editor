package widget

import (
	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
)

// Spinner frame presets. FramesDots uses braille, FramesLine plain ASCII,
// FramesCircle quadrant glyphs.
var (
	FramesDots   = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	FramesLine   = []string{"|", "/", "-", "\\"}
	FramesCircle = []string{"◐", "◓", "◑", "◒"}
)

// Spinner renders one animation frame. It owns no timer and spawns no
// goroutine: apps advance it by calling Tick on whatever cadence they
// schedule, then trigger a redraw.
type Spinner struct {
	Base

	Frames []string
	Style  cell.Style
	Frame  int // index of the frame Tick will move past; wraps via modulo
}

// NewSpinner returns a spinner using the given frames and style.
func NewSpinner(frames []string, style cell.Style) *Spinner {
	return &Spinner{Frames: frames, Style: style}
}

// Tick advances to the next frame, wrapping to zero at the end.
func (s *Spinner) Tick() {
	if len(s.Frames) == 0 {
		return
	}
	s.Frame = (s.Frame + 1) % len(s.Frames)
}

func (s *Spinner) Measure(max geom.Size) geom.Size {
	return fit(geom.Size{W: strWidth(s.frame()), H: 1}, max)
}

func (s *Spinner) Draw(ctx *DrawCtx) {
	r := ctx.Rect
	frame := s.frame()
	if r.Empty() || frame == "" {
		return
	}
	// Vertically centered, flush left; Print clips oversized frames.
	y := r.Pos.Y + r.Size.H/2
	if y >= r.Bottom() {
		y = r.Bottom() - 1
	}
	ctx.Screen.Print(r.Pos.X, y, r.Right(), frame, s.Style)
}

// frame resolves the current frame without normalizing Frame itself, so a
// Draw pass never mutates visible state.
func (s *Spinner) frame() string {
	if len(s.Frames) == 0 {
		return ""
	}
	return s.Frames[s.Frame%len(s.Frames)]
}
