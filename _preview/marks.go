package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"

	"github.com/Aswanidev-vs/cherry/cell"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// markCandidate is one proposed splash mark. Every row must be the same
// display width, or the mark will not centre and the wordmark under it will
// sit off-axis.
type markCandidate struct {
	name string
	note string
	rows []string
}

// The finalists. They share a construction — two round ears over a rounded
// head, two eye patches and a nose — and differ only in whether the ears are
// detached from the head, how heavy the patches are, and how wide the mark is.
var markCandidates = []markCandidate{
	{
		name: "H solid muzzle",
		note: "detached ears, patches and a solid muzzle between them",
		rows: []string{
			"    ▄▀▄     ▄▀▄    ",
			"   █▛▀▜█   █▛▀▜█   ",
			"                   ",
			"  ▛▀▀▀▀▀▀▀▀▀▀▀▀▀▜  ",
			"  █ ░░░ ███ ░░░ █  ",
			"  ▙▄▄▄▄▄▄▄▄▄▄▄▄▄▟  ",
		},
	},
	{
		name: "L face",
		note: "a real two-row face: a row of eyes and a row of mouth",
		rows: []string{
			"    ▄▀▄     ▄▀▄    ",
			"   █▛▀▜█   █▛▀▜█   ",
			"                   ",
			"  ▛▀▀▀▀▀▀▀▀▀▀▀▀▀▜  ",
			"  █ ░░░     ░░░ █  ",
			"  █     ▄▄▄     █  ",
			"  ▙▄▄▄▄▄▄▄▄▄▄▄▄▄▟  ",
		},
	},
	{
		name: "M compact solid",
		note: "the winner at seventeen columns",
		rows: []string{
			"   ▄▀▄     ▄▀▄   ",
			"  █▛▀▜█   █▛▀▜█  ",
			"                 ",
			"  ▛▀▀▀▀▀▀▀▀▀▀▀▜  ",
			"  █ ░░ ███ ░░ █  ",
			"  ▙▄▄▄▄▄▄▄▄▄▄▄▟  ",
		},
	},
}

// checkMarkWidths reports any candidate whose rows disagree on display width.
// A mark one cell short on a single row is invisible in code review and
// glaring on screen, so it is worth asserting.
func checkMarkWidths() error {
	var bad []string
	for _, c := range markCandidates {
		w := -1
		for i, r := range c.rows {
			n := cell.StringWidth(r)
			if w == -1 {
				w = n
				continue
			}
			if n != w {
				bad = append(bad, fmt.Sprintf("%s row %d is %d cells, row 0 is %d", c.name, i, n, w))
			}
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("candidate rows are not uniform:\n  %s", strings.Join(bad, "\n  "))
	}
	return nil
}

// writeMarkSheet renders every candidate large and side by side so they can be
// compared in one look, which is the only way to judge whether a mark reads
// as its subject at a glance.
func writeMarkSheet(path string, f *faces) error {
	cols := 2
	rowsPerMark := 0
	for _, c := range markCandidates {
		if len(c.rows) > rowsPerMark {
			rowsPerMark = len(c.rows)
		}
	}
	cellW := 0
	for _, c := range markCandidates {
		for _, r := range c.rows {
			if w := f.cellWidth(r); w > cellW {
				cellW = w
			}
		}
	}
	cellW += 6
	cellH := f.cellH * (rowsPerMark + 3)
	sheetW := cellW * cols
	sheetH := cellH * ((len(markCandidates) + cols - 1) / cols)

	img := image.NewRGBA(image.Rect(0, 0, sheetW, sheetH))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0x19, 0x16, 0x14, 0xFF}}, image.Point{}, draw.Src)

	accent := color.RGBA{0xE8, 0xA3, 0x3D, 0xFF} // the signal colour
	label := color.RGBA{0xA7, 0x9C, 0x8B, 0xFF}  // captions

	for i, c := range markCandidates {
		cx := (i % cols) * cellW
		cy := (i / cols) * cellH
		put(img, f, cx+3, cy, c.name, label)
		put(img, f, cx+3, cy+f.cellH, c.note, label)
		for r, row := range c.rows {
			put(img, f, cx+3, cy+(r+3)*f.cellH, row, accent)
		}
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}

// put draws s at a pixel position.
func put(img *image.RGBA, f *faces, x, y int, s string, c color.Color) {
	d := &font.Drawer{Dst: img, Src: image.NewUniform(c), Face: f.bold}
	d.Dot = fixed.P(x, y+f.ascent)
	d.DrawString(s)
}

// cellWidth reports how many terminal cells s occupies, which is what decides
// whether a mark row lines up with its neighbours.
func (f *faces) cellWidth(s string) int { return cell.StringWidth(s) * f.cellW }
