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
//
// All mutable state is guarded by `mu`. Callers that need to read or write
// `input`, `scroll`, or `output` MUST hold the lock (use the helper methods).
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
	t.mu.Lock()
	t.done = false
	t.output = nil
	t.scroll = 0
	t.input = ""
	t.mu.Unlock()

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
	t.mu.Lock()
	t.cmd = c
	t.stdin = stdin
	t.mu.Unlock()

	// Use a larger buffer for shell commands that may produce long lines
	// (e.g. minified JSON, npm install output).
	const maxLineSize = 1024 * 1024

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64*1024), maxLineSize)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimRight(line, "\r\n")
			t.appendLine(line)
		}
	}()

	// Read stderr (merge into output)
	go func() {
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 64*1024), maxLineSize)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimRight(line, "\r\n")
			t.appendLine(line)
		}
	}()

	// Wait for process exit
	go func() {
		_ = c.Wait()
		t.mu.Lock()
		t.done = true
		t.mu.Unlock()
	}()

	return nil
}

func (t *TerminalModel) appendLine(line string) {
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

// Stop kills the shell process.
func (t *TerminalModel) Stop() {
	t.mu.Lock()
	cmd := t.cmd
	t.cmd = nil
	t.done = true
	t.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// Write sends a line of input to the shell's stdin.
func (t *TerminalModel) Write(line string) error {
	t.mu.Lock()
	stdin := t.stdin
	t.mu.Unlock()
	if stdin == nil {
		return fmt.Errorf("terminal not started")
	}
	_, err := io.WriteString(stdin, line+"\n")
	return err
}

// SetInput replaces the typed input buffer.
func (t *TerminalModel) SetInput(s string) {
	t.mu.Lock()
	t.input = s
	t.mu.Unlock()
}

// AppendInput appends a single rune/character to the input buffer.
func (t *TerminalModel) AppendInput(s string) {
	t.mu.Lock()
	t.input += s
	t.mu.Unlock()
}

// InputBackspace removes the last rune from the input buffer.
func (t *TerminalModel) InputBackspace() {
	t.mu.Lock()
	runes := []rune(t.input)
	if len(runes) > 0 {
		t.input = string(runes[:len(runes)-1])
	}
	t.mu.Unlock()
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
	if t.scroll < 0 {
		t.scroll = 0
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
// The panel reserves 2 rows for the border, 1 for the title and 1 for the
// input bar, so the scrollable output area is height - 4.
func (t *TerminalModel) visibleLines() int {
	vh := t.height - 4
	if vh < 1 {
		vh = 1
	}
	return vh
}

// View returns the terminal panel content as a string.
func (t *TerminalModel) View() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.IsRunningLocked() {
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
	t.mu.Lock()
	input := t.input
	t.mu.Unlock()
	prompt := "$ " + input
	cursor := "█"
	return prompt + cursor
}

// IsRunning returns whether the shell process is alive.
func (t *TerminalModel) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.IsRunningLocked()
}

// IsRunningLocked reports the running state without acquiring the mutex.
// Callers MUST hold mu.
func (t *TerminalModel) IsRunningLocked() bool {
	return t.cmd != nil && !t.done
}

// Resize updates the terminal dimensions.
func (t *TerminalModel) Resize(w, h int) {
	t.mu.Lock()
	t.width = w
	t.height = h
	t.mu.Unlock()
}
