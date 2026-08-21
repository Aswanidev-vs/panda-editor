package render

import "io"

// SetCursor moves the hardware cursor to cell (x,y) and reveals it, or hides
// it entirely when either coordinate is negative. It is meant to be called
// after Flush so the cursor lands on top of the freshly painted frame.
func (s *Screen) SetCursor(out io.Writer, x, y int) error {
	if x < 0 || y < 0 {
		s.buf.Reset()
		s.buf.WriteString("\x1b[?25l")
		_, err := out.Write(s.buf.Bytes())
		s.lastX, s.lastY = -1, -1
		return err
	}
	s.buf.Reset()
	writeCUP(s.buf, y+1, x+1)
	s.buf.WriteString("\x1b[?25h")
	s.lastX, s.lastY = x, y
	_, err := out.Write(s.buf.Bytes())
	return err
}
