// Package geom holds the minimal geometry vocabulary shared by every cherry layer.
package geom

// Point is a position in character cells.
type Point struct {
	X, Y int
}

// Size is an extent in character cells.
type Size struct {
	W, H int
}

// Rect is a screen region: Pos is inclusive-top-left, Size is cell count.
type Rect struct {
	Pos  Point
	Size Size
}

func (r Rect) Right() int     { return r.Pos.X + r.Size.W }
func (r Rect) Bottom() int    { return r.Pos.Y + r.Size.H }
func (r Rect) Empty() bool    { return r.Size.W <= 0 || r.Size.H <= 0 }

// Contains reports whether p lies inside r.
func (r Rect) Contains(p Point) bool {
	return p.X >= r.Pos.X && p.Y >= r.Pos.Y && p.X < r.Right() && p.Y < r.Bottom()
}

// Intersect returns the overlapping rect, possibly empty.
func (r Rect) Intersect(o Rect) Rect {
	x1 := max(r.Pos.X, o.Pos.X)
	y1 := max(r.Pos.Y, o.Pos.Y)
	x2 := min(r.Right(), o.Right())
	y2 := min(r.Bottom(), o.Bottom())
	if x2 <= x1 || y2 <= y1 {
		return Rect{}
	}
	return Rect{Pos: Point{x1, y1}, Size: Size{x2 - x1, y2 - y1}}
}

func max(a, b int) int { if a > b { return a }; return b }
func min(a, b int) int { if a < b { return a }; return b }
