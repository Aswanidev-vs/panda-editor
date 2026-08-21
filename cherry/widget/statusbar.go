package widget

import (
	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
)

// Segment is one slot of a StatusBar.
type Segment struct {
	Text   string
	Style  cell.Style
	Flex   int  // > 0: take a share of leftover width proportional to Flex; <= 0: natural text width
	Center bool // center Text within the slot instead of left-aligning
}

// NewStatusBar returns a bar showing the given segments in order.
func NewStatusBar(segments ...Segment) *StatusBar {
	return &StatusBar{Segments: segments}
}

// StatusBar is a one-row strip of segments, typically docked to an edge.
// Each segment's background is painted across its whole slot so the bar
// reads as one continuous surface. Only the top row of the assigned rect
// is used; Measure always reports height 1.
type StatusBar struct {
	Base

	Segments []Segment
	Bg       cell.Style // fill behind slots no segment claims (default style if zero)
}

func (sb *StatusBar) Measure(max geom.Size) geom.Size {
	w := 0
	for _, sg := range sb.Segments {
		if sg.Flex <= 0 {
			w += strWidth(sg.Text)
		}
	}
	return fit(geom.Size{W: w, H: 1}, max)
}

func (sb *StatusBar) Draw(ctx *DrawCtx) {
	r := ctx.Rect
	if r.Empty() {
		return
	}
	row := geom.Rect{Pos: r.Pos, Size: geom.Size{W: r.Size.W, H: 1}}
	ctx.Screen.Fill(row, styledBlank(sb.Bg))

	right := r.Right()
	x := r.Pos.X
	slots := statusSlots(sb.Segments, row.Size.W)
	for i, sg := range sb.Segments {
		end := x + slots[i]
		if end > right {
			end = right
		}
		if end > x {
			slot := geom.Rect{Pos: geom.Point{X: x, Y: r.Pos.Y}, Size: geom.Size{W: end - x, H: 1}}
			ctx.Screen.Fill(slot, styledBlank(sg.Style))
			tx := x
			if sg.Center {
				if slack := slot.Size.W - strWidth(sg.Text); slack > 0 {
					tx += slack / 2
				}
			}
			ctx.Screen.Print(tx, r.Pos.Y, end, sg.Text, sg.Style)
		}
		x = end
	}
}

// statusSlots distributes total width w across segments. Natural segments
// claim their text width first; flex segments split whatever remains,
// proportionally to Flex with rounding remainders going to later flex
// segments so slots always sum to min(usable, w). When natural widths alone
// overflow w the result may sum beyond it; Draw clamps at the rect edge.
func statusSlots(segs []Segment, w int) []int {
	slots := make([]int, len(segs))
	flexTotal, fixed := 0, 0
	for i, sg := range segs {
		slots[i] = strWidth(sg.Text)
		if sg.Flex > 0 {
			flexTotal += sg.Flex
		} else {
			fixed += slots[i]
		}
	}
	free := w - fixed
	if free < 0 {
		free = 0
	}
	given, allocated := 0, 0
	for i, sg := range segs {
		if sg.Flex <= 0 {
			continue
		}
		given += sg.Flex
		end := free * given / flexTotal
		slots[i] = end - allocated
		allocated = end
	}
	return slots
}
