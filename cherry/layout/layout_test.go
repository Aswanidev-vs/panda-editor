package layout

import (
	"reflect"
	"testing"

	"github.com/Aswanidev-vs/cherry/geom"
)

func rect(x, y, w, h int) geom.Rect {
	return geom.Rect{Pos: geom.Point{X: x, Y: y}, Size: geom.Size{W: w, H: h}}
}

func TestSolve(t *testing.T) {
	tests := []struct {
		name  string
		total geom.Size
		axis  Axis
		specs []Spec
		want  []geom.Rect
	}{
		{
			name:  "pure fill equal split gives first item the remainder cell",
			total: geom.Size{W: 100, H: 40},
			axis:  Horizontal,
			specs: []Spec{{Fill: true}, {Fill: true}, {Fill: true}},
			want:  []geom.Rect{rect(0, 0, 34, 40), rect(34, 0, 33, 40), rect(67, 0, 33, 40)},
		},
		{
			name:  "percent plus fill mix leaves leftover pool to fill",
			total: geom.Size{W: 100, H: 50},
			axis:  Horizontal,
			specs: []Spec{{Percent: 25}, {Percent: 25}, {Fill: true}},
			want:  []geom.Rect{rect(0, 0, 25, 50), rect(25, 0, 25, 50), rect(50, 0, 50, 50)},
		},
		{
			name:  "percent largest remainder rounds to exact total, ties favor earliest",
			total: geom.Size{W: 100, H: 10},
			axis:  Horizontal,
			specs: []Spec{{Percent: 33.33}, {Percent: 33.33}, {Percent: 33.33}},
			want:  []geom.Rect{rect(0, 0, 34, 10), rect(34, 0, 33, 10), rect(67, 0, 33, 10)},
		},
		{
			name:  "fixed overflow claims head and collapses the tail",
			total: geom.Size{W: 10, H: 4},
			axis:  Horizontal,
			specs: []Spec{{Fixed: 6}, {Fixed: 6}, {Fixed: 6}},
			want:  []geom.Rect{rect(0, 0, 6, 4), rect(6, 0, 4, 4), rect(10, 0, 0, 4)},
		},
		{
			name:  "min floor forces percent neighbors to shrink",
			total: geom.Size{W: 20, H: 5},
			axis:  Horizontal,
			specs: []Spec{{Percent: 50, Min: 14}, {Percent: 50}},
			want:  []geom.Rect{rect(0, 0, 17, 5), rect(17, 0, 3, 5)},
		},
		{
			name:  "min raise consumes pool so later items get zero",
			total: geom.Size{W: 5, H: 3},
			axis:  Horizontal,
			specs: []Spec{{Fixed: 4, Min: 5}, {Fill: true}},
			want:  []geom.Rect{rect(0, 0, 5, 3), rect(5, 0, 0, 3)},
		},
		{
			name:  "max clamp redistributes freed cells across remaining fills",
			total: geom.Size{W: 30, H: 4},
			axis:  Horizontal,
			specs: []Spec{{Fill: true, Max: 5}, {Fill: true}, {Fill: true}},
			want:  []geom.Rect{rect(0, 0, 5, 4), rect(5, 0, 13, 4), rect(18, 0, 12, 4)},
		},
		{
			name:  "max clamp without fill leaves trailing space unused",
			total: geom.Size{W: 10, H: 4},
			axis:  Horizontal,
			specs: []Spec{{Fixed: 8, Max: 3}, {Fixed: 2}},
			want:  []geom.Rect{rect(0, 0, 3, 4), rect(3, 0, 2, 4)},
		},
		{
			name:  "max clamp on percent item frees cells to a later fill",
			total: geom.Size{W: 20, H: 2},
			axis:  Horizontal,
			specs: []Spec{{Percent: 100, Max: 6}, {Fill: true}},
			want:  []geom.Rect{rect(0, 0, 6, 2), rect(6, 0, 14, 2)},
		},
		{
			name:  "fixed wins over percent and fill on the same spec",
			total: geom.Size{W: 10, H: 4},
			axis:  Horizontal,
			specs: []Spec{{Fixed: 4, Percent: 90, Fill: true}, {Fill: true}},
			want:  []geom.Rect{rect(0, 0, 4, 4), rect(4, 0, 6, 4)},
		},
		{
			name:  "oversubscribed percents normalize instead of overflowing",
			total: geom.Size{W: 10, H: 4},
			axis:  Horizontal,
			specs: []Spec{{Percent: 60}, {Percent: 60}},
			want:  []geom.Rect{rect(0, 0, 5, 4), rect(5, 0, 5, 4)},
		},
		{
			name:  "odd pool rounding still sums exactly",
			total: geom.Size{W: 10, H: 4},
			axis:  Horizontal,
			specs: []Spec{{Percent: 33.33}, {Percent: 33.33}, {Percent: 33.33}},
			want:  []geom.Rect{rect(0, 0, 4, 4), rect(4, 0, 3, 4), rect(7, 0, 3, 4)},
		},
		{
			name:  "zero total yields zero-width rects spanning cross axis",
			total: geom.Size{W: 0, H: 7},
			axis:  Horizontal,
			specs: []Spec{{Fill: true}, {Fill: true}},
			want:  []geom.Rect{rect(0, 0, 0, 7), rect(0, 0, 0, 7)},
		},
		{
			name:  "negative total behaves like zero",
			total: geom.Size{W: -5, H: 7},
			axis:  Horizontal,
			specs: []Spec{{Fill: true}, {Fill: true}},
			want:  []geom.Rect{rect(0, 0, 0, 7), rect(0, 0, 0, 7)},
		},
		{
			name:  "empty specs returns nil",
			total: geom.Size{W: 80, H: 24},
			axis:  Horizontal,
			specs: nil,
			want:  nil,
		},
		{
			name:  "vertical mirrors the horizontal solve",
			total: geom.Size{W: 40, H: 100},
			axis:  Vertical,
			specs: []Spec{{Fill: true}, {Fill: true}, {Fill: true}},
			want:  []geom.Rect{rect(0, 0, 40, 34), rect(0, 34, 40, 33), rect(0, 67, 40, 33)},
		},
		{
			name:  "single fill swallows everything including cross size",
			total: geom.Size{W: 7, H: 3},
			axis:  Horizontal,
			specs: []Spec{{Fill: true}},
			want:  []geom.Rect{rect(0, 0, 7, 3)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Solve(tt.total, tt.axis, tt.specs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Solve(%v, %v, %v)\n got: %v\nwant: %v", tt.total, tt.axis, tt.specs, got, tt.want)
			}
		})
	}
}

// TestSolveInvariants walks a deterministic grid of inputs and asserts the
// structural guarantees every caller relies on: exact slice length, non-negative
// sizes, cumulative positions from origin, full cross-axis span, and no main-axis
// overrun of the available space.
func TestSolveInvariants(t *testing.T) {
	specSets := [][]Spec{
		{{Fill: true}},
		{{Fill: true}, {Fill: true}, {Fill: true}},
		{{Fixed: 3}, {Percent: 45.5}, {Fill: true}},
		{{Percent: 70}, {Percent: 70}, {Percent: 70}},
		{{Fixed: 2, Min: 6}, {Fill: true, Max: 4}, {Percent: 30, Min: 1, Max: 9}},
		{{}, {}, {}},
		{{Fixed: 9, Max: 2}, {Fixed: 1, Min: 3}, {Fill: true, Min: 2, Max: 5}},
	}
	totals := []geom.Size{{W: 0, H: 0}, {W: 1, H: 9}, {W: 13, H: 7}, {W: 24, H: 5}}
	axes := []Axis{Horizontal, Vertical}

	for _, total := range totals {
		for _, axis := range axes {
			mainTotal, crossTotal := total.W, total.H
			if axis == Vertical {
				mainTotal, crossTotal = total.H, total.W
			}
			for _, specs := range specSets {
				rects := Solve(total, axis, specs)
				if len(rects) != len(specs) {
					t.Fatalf("Solve(%v,%v,%v): got %d rects, want %d", total, axis, specs, len(rects), len(specs))
				}
				pos := 0
				for i, r := range rects {
					if r.Size.W < 0 || r.Size.H < 0 {
						t.Fatalf("negative size in %+v for specs %+v", r, specs)
					}
					if axis == Vertical {
						if r.Pos.Y != pos || r.Pos.X != 0 || r.Size.W != crossTotal {
							t.Fatalf("vertical rect %d = %+v, want y=%d x=0 w=%d", i, r, pos, crossTotal)
						}
					} else if r.Pos.X != pos || r.Pos.Y != 0 || r.Size.H != crossTotal {
						t.Fatalf("horizontal rect %d = %+v, want x=%d y=0 h=%d", i, r, pos, crossTotal)
					}
					main := r.Size.W
					if axis == Vertical {
						main = r.Size.H
					}
					pos += main
				}
				if pos > max(mainTotal, 0) {
					t.Fatalf("main axis overrun: used %d > %d for specs %+v", pos, mainTotal, specs)
				}
			}
		}
	}
}

// TestSolveDeterministic pins that repeated solves produce byte-identical
// output, guarding against hidden map-order or time dependence.
func TestSolveDeterministic(t *testing.T) {
	total := geom.Size{W: 37, H: 11}
	specs := []Spec{{Percent: 41.7}, {Fixed: 2, Min: 1, Max: 30}, {Fill: true, Min: 2, Max: 20}, {Fill: true}}
	first := Solve(total, Horizontal, specs)
	for range 10 {
		if next := Solve(total, Horizontal, specs); !reflect.DeepEqual(first, next) {
			t.Fatalf("nondeterministic solve:\n first: %v\n later: %v", first, next)
		}
	}
}
