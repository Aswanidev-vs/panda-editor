// Package layout implements the flex solver that partitions a rectangular
// area into consecutive slices along one axis. It is pure geometry: no widget
// types, no I/O, fully deterministic.
package layout

import (
	"math"
	"sort"

	"github.com/Aswanidev-vs/cherry/geom"
)

// Axis selects the direction Solve partitions along.
type Axis uint8

const (
	// Horizontal partitions the width; every rect spans the full height.
	Horizontal Axis = iota
	// Vertical partitions the height; every rect spans the full width.
	Vertical
)

// Spec declares how one slice claims space along the solve axis.
type Spec struct {
	Fixed   int     // absolute cell count; wins over Percent/Fill
	Percent float64 // share of remaining space (0-100)
	Fill    bool    // equal split of remaining after fixed+percent
	Min     int     // lower clamp post-solve
	Max     int     // upper clamp post-solve (<=0 means unbounded)
}

// Solve partitions total along axis into len(specs) consecutive rects.
// Zero-length specs returns nil. When more space is demanded than available,
// fixed claims win first and the tail collapses toward zero. Cross-axis
// dimension of every rect equals the full total.
func Solve(total geom.Size, axis Axis, specs []Spec) []geom.Rect {
	if len(specs) == 0 {
		return nil
	}

	main, cross := total.W, total.H
	if axis == Vertical {
		main, cross = total.H, total.W
	}

	pool := max(main, 0)
	sizes := make([]int, len(specs))

	// Pass 1: fixed items claim in order, capped by what is left.
	for i := range specs {
		if specs[i].Fixed <= 0 {
			continue
		}
		take := min(specs[i].Fixed, pool)
		sizes[i] = take
		pool -= take
	}

	// Pass 2: Min floors before proportional distribution, so percent/fill
	// neighbors shrink to make room. Once the pool runs dry, later items get
	// nothing further (the boundary item keeps whatever partial grant fits).
	for i := range specs {
		want := specs[i].Min - sizes[i]
		if want <= 0 {
			continue
		}
		give := min(want, pool)
		sizes[i] += give
		pool -= give
	}

	fillIdx := fillSpecs(specs)

	// Pass 3: percent shares of what remains after fixed+min.
	if pool > 0 {
		pool -= distributePercent(pool, specs, percentSpecs(specs), sizes)
	}
	// Pass 4: fill splits whatever still remains equally.
	if pool > 0 {
		distributeFill(pool, specs, fillIdx, sizes)
	}

	// Pass 5: Max clamps after distribution; freed cells get one extra pass
	// redistributed over fill items that still have headroom. Anything that
	// does not fit is left as trailing space for the caller to fill explicitly.
	freed := 0
	for i := range specs {
		if m := specs[i].Max; m > 0 && sizes[i] > m {
			freed += sizes[i] - m
			sizes[i] = m
		}
	}
	if freed > 0 {
		redistribute(freed, specs, fillIdx, sizes)
	}

	rects := make([]geom.Rect, len(specs))
	pos := 0
	for i, s := range sizes {
		if axis == Vertical {
			rects[i] = geom.Rect{
				Pos:  geom.Point{Y: pos},
				Size: geom.Size{W: cross, H: s},
			}
		} else {
			rects[i] = geom.Rect{
				Pos:  geom.Point{X: pos},
				Size: geom.Size{W: s, H: cross},
			}
		}
		pos += s
	}
	return rects
}

// percentSpecs returns indices eligible for proportional distribution:
// anything without an explicit Fixed claim. Fixed wins over Percent/Fill.
func percentSpecs(specs []Spec) []int {
	idx := make([]int, 0, len(specs))
	for i := range specs {
		if specs[i].Fixed <= 0 && specs[i].Percent > 0 {
			idx = append(idx, i)
		}
	}
	return idx
}

// fillSpecs returns indices eligible for the fill pass: no Fixed claim and no
// Percent weight (Percent wins when both are set on one spec).
func fillSpecs(specs []Spec) []int {
	idx := make([]int, 0, len(specs))
	for i := range specs {
		if specs[i].Fixed <= 0 && specs[i].Percent <= 0 && specs[i].Fill {
			idx = append(idx, i)
		}
	}
	return idx
}

// distributePercent hands each index its rounded share of pool using the
// largest-remainder method, so granted cells sum exactly to the distributed
// amount D = round(pool * sumPercent / 100). Denominator max(100, sumPercent)
// keeps oversubscribed percents (>100 combined) from demanding more than the
// pool holds while leaving sub-100 sums untouched. Returns cells granted.
func distributePercent(pool int, specs []Spec, idx []int, sizes []int) int {
	if pool <= 0 || len(idx) == 0 {
		return 0
	}
	var sumP float64
	for _, i := range idx {
		sumP += specs[i].Percent
	}
	if sumP <= 0 {
		return 0
	}

	denom := math.Max(100, sumP)
	bases := make([]int, len(idx))
	fracs := make([]float64, len(idx))
	sumBase := 0
	for k, i := range idx {
		e := float64(pool) * specs[i].Percent / denom
		bases[k] = int(math.Floor(e))
		fracs[k] = e - float64(bases[k])
		sumBase += bases[k]
	}

	granted := int(math.Round(float64(pool) * sumP / denom))
	extras := clamp(granted-sumBase, 0, len(idx))

	order := make([]int, len(idx))
	for k := range order {
		order[k] = k
	}
	// Largest fractional remainder first; ties resolve deterministically to
	// the earliest spec so identical inputs always produce identical output.
	sort.Slice(order, func(a, b int) bool {
		if fracs[order[a]] != fracs[order[b]] {
			return fracs[order[a]] > fracs[order[b]]
		}
		return order[a] < order[b]
	})

	for k, i := range idx {
		sizes[i] += bases[k]
	}
	for k := 0; k < extras; k++ {
		sizes[idx[order[k]]]++
	}
	return sumBase + extras
}

// distributeFill splits pool equally across idx with largest remainder; with
// equal weights this reduces to integer division where the first
// pool % len(idx) specs in declaration order absorb the leftover cells.
// Returns cells granted (always exactly pool).
func distributeFill(pool int, specs []Spec, idx []int, sizes []int) int {
	if pool <= 0 || len(idx) == 0 {
		return 0
	}
	base := pool / len(idx)
	extra := pool % len(idx)
	for k, i := range idx {
		give := base
		if k < extra {
			give++
		}
		sizes[i] += give
	}
	return pool
}

// redistribute hands freed cells once over fill specs that still have Max
// headroom. Shares exceeding an individual cap are dropped rather than
// re-routed — a single pass keeps behavior predictable.
func redistribute(freed int, specs []Spec, fillIdx []int, sizes []int) {
	receivers := make([]int, 0, len(fillIdx))
	for _, i := range fillIdx {
		if m := specs[i].Max; m <= 0 || sizes[i] < m {
			receivers = append(receivers, i)
		}
	}
	if len(receivers) == 0 {
		return
	}
	base := freed / len(receivers)
	extra := freed % len(receivers)
	for k, i := range receivers {
		give := base
		if k < extra {
			give++
		}
		if m := specs[i].Max; m > 0 && sizes[i]+give > m {
			give = m - sizes[i]
		}
		if give > 0 {
			sizes[i] += give
		}
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
