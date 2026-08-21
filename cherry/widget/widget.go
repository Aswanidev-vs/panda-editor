// Package widget provides cherry's component model and built-in components.
//
// A widget owns its state, paints itself into the screen region it was given,
// and consumes input events targeted at it. Composition is plain Go: embed or
// hold child widgets and return them from Children().
package widget

import (
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/render"
)

// Widget is implemented by every UI component.
type Widget interface {
	// Measure reports preferred size within the given constraints. Called by
	// the layout solver before drawing; must not mutate visible state.
	Measure(max geom.Size) geom.Size
	// Draw paints the widget inside ctx.Rect. Called top-down after layout.
	Draw(ctx *DrawCtx)
	// Handle processes an event targeted at this widget; returning true
	// consumes it. Called bottom-up along the focus path.
	Handle(ev input.Event) bool
}

// Focusable is optionally implemented by widgets that can take keyboard
// focus (Inputs, Trees, Lists...).
type Focusable interface {
	Focus()
	Blur()
	Focused() bool
}

// DrawCtx carries the assigned region and the target screen for one draw pass.
type DrawCtx struct {
	Rect geom.Rect
	Screen *render.Screen
}

// Cursorer is optionally implemented by root widgets that decide where the
// hardware cursor rests between frames. The chassis invokes it after every
// flush when present.
type Cursorer interface {
	// CursorPos reports the (x,y) cell to place the cursor and whether the
	// cursor should be visible at all. (false) hides the cursor entirely.
	CursorPos() (geom.Point, bool)
}

// Base provides no-op defaults so simple widgets only override what they need.
type Base struct{}

func (Base) Measure(geom.Size) geom.Size          { return geom.Size{} }
func (Base) Handle(input.Event) bool              { return false }
