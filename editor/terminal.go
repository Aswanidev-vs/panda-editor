package editor

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// TerminalModel manages an integrated shell process.
type TerminalModel struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	output   []string
	scroll   int
	maxLines int
	input    string
	width    int
	height   int
	shell    string
	done     bool
	onOutput func(string)
	mu       sync.Mutex
}

// NewTerminal creates a terminal model for the given shell.
func NewTerminal(shell string) *TerminalModel {
	return &TerminalModel{
		shell:    shell,
		maxLines: 500,
		width:    80,
		height:   10,
		output:   make([]string, 0, 500),
	}
}

// Start spawns the shell process and begins reading output.
func (t *TerminalModel) Start() error {
	if t.cmd != nil {
		return nil
	}
	t.done = false
	t.output = nil
	t.scroll = 0
	t.input = ""

	c := exec.Command(t.shell)
	stdin, err := c.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		return err
	}
	t.cmd = c
	t.stdin = stdin

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimRight(line, "\r\n")
			t.mu.Lock()
			wasAtBottom := t.scroll == 0
			t.output = append(t.output, line)
			if len(t.output) > t.maxLines {
				t.output = t.output[len(t.output)-t.maxLines:]
			}
			if wasAtBottom {
				t.scroll = 0
			}
			cb := t.onOutput
			t.mu.Unlock()
			if cb != nil {
				cb(line)
			}
		}
	}()

	// Read stderr (merge into output)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimRight(line, "\r\n")
			t.mu.Lock()
			wasAtBottom := t.scroll == 0
			t.output = append(t.output, line)
			if len(t.output) > t.maxLines {
				t.output = t.output[len(t.output)-t.maxLines:]
			}
			if wasAtBottom {
				t.scroll = 0
			}
			cb := t.onOutput
			t.mu.Unlock()
			if cb != nil {
				cb(line)
			}
		}
	}()

	// Wait for process exit
	go func() {
		c.Wait()
		t.mu.Lock()
		t.done = true
		t.mu.Unlock()
	}()

	return nil
}

// Stop kills the shell process.
func (t *TerminalModel) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}
	t.cmd = nil
	t.done = true
}

// Write sends a line of input to the shell's stdin.
func (t *TerminalModel) Write(line string) error {
	if t.stdin == nil {
		return fmt.Errorf("terminal not started")
	}
	_, err := io.WriteString(t.stdin, line+"\n")
	return err
}

// ScrollUp scrolls the output view up by n lines.
func (t *TerminalModel) ScrollUp(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.scroll += n
	maxScroll := 0
	if len(t.output) > t.visibleLines() {
		maxScroll = len(t.output) - t.visibleLines()
	}
	if t.scroll > maxScroll {
		t.scroll = maxScroll
	}
}

// ScrollDown scrolls the output view down by n lines.
func (t *TerminalModel) ScrollDown(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.scroll -= n
	if t.scroll < 0 {
		t.scroll = 0
	}
}

// VisibleLines returns the number of output lines that fit in the view.
func (t *TerminalModel) visibleLines() int {
	vh := t.height - 1 // reserve 1 line for input bar
	if vh < 1 {
		vh = 1
	}
	return vh
}

// View returns the terminal panel content as a string.
func (t *TerminalModel) View() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.IsRunning() {
		return "Terminal not running. Press Ctrl+` to start."
	}

	var sb strings.Builder
	vh := t.visibleLines()
	start := len(t.output) - vh - t.scroll
	if start < 0 {
		start = 0
	}
	end := start + vh
	if end > len(t.output) {
		end = len(t.output)
	}

	// Output lines
	for i := start; i < end; i++ {
		line := t.output[i]
		lineWidth := lipgloss.Width(line)
		if lineWidth > t.width {
			// Truncate by visual width
			trunc := make([]rune, 0, t.width)
			w := 0
			for _, r := range line {
				rw := lipgloss.Width(string(r))
				if w+rw > t.width {
					break
				}
				trunc = append(trunc, r)
				w += rw
			}
			line = string(trunc)
		}
		sb.WriteString(line)
		if i < end-1 {
			sb.WriteString("\n")
		}
	}

	// If no output, show blank lines
	linesPrinted := end - start
	for linesPrinted < vh {
		sb.WriteString(strings.Repeat(" ", t.width))
		linesPrinted++
		if linesPrinted < vh {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// PromptView returns the input bar line with cursor indicator.
func (t *TerminalModel) PromptView() string {
	prompt := "$ " + t.input
	cursor := "█"
	return prompt + cursor
}

// IsRunning returns whether the shell process is alive.
func (t *TerminalModel) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cmd != nil && !t.done
}

// Resize updates the terminal dimensions.
func (t *TerminalModel) Resize(w, h int) {
	t.width = w
	t.height = h
}
