package buffer

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// MaxOpenBytes caps the size of a file that can be opened in the editor.
// Files larger than this return ErrFileTooLarge so the UI can prompt the user.
const MaxOpenBytes int64 = 25 * 1024 * 1024 // 25 MB

// ErrFileTooLarge is returned by Open when a file exceeds MaxOpenBytes.
var ErrFileTooLarge = errors.New("file exceeds maximum open size")

type Buffer struct {
	Lines    []string
	FilePath string
	Modified bool
	Name     string
	Language string
}

func New() *Buffer {
	return &Buffer{
		Lines:    []string{""},
		FilePath: "",
		Modified: false,
		Name:     "[scratch]",
		Language: "",
	}
}

// Open reads a file from disk. If the file is larger than MaxOpenBytes,
// it returns ErrFileTooLarge so callers can decide how to handle it.
func Open(path string) (*Buffer, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Buffer{
				Lines:    []string{""},
				FilePath: absPath,
				Modified: false,
				Name:     filepath.Base(absPath),
				Language: DetectLanguage(absPath),
			}, nil
		}
		return nil, err
	}

	if !info.IsDir() && info.Size() > MaxOpenBytes {
		return nil, fmt.Errorf("%w: %s is %d bytes (limit %d)",
			ErrFileTooLarge, filepath.Base(absPath), info.Size(), MaxOpenBytes)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	if !utf8.Valid(data) {
		// Fall back to lossy decode for non-UTF-8 files instead of corrupting the buffer.
		data = []byte(string(data))
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	return &Buffer{
		Lines:    lines,
		FilePath: absPath,
		Modified: false,
		Name:     filepath.Base(absPath),
		Language: DetectLanguage(absPath),
	}, nil
}

func (b *Buffer) Save() error {
	if b.FilePath == "" {
		return nil
	}
	dir := filepath.Dir(b.FilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content := strings.Join(b.Lines, "\n")
	err := os.WriteFile(b.FilePath, []byte(content), 0o644)
	if err == nil {
		b.Modified = false
	}
	return err
}

func (b *Buffer) SaveAs(path string) error {
	b.FilePath = path
	b.Name = filepath.Base(path)
	b.Language = DetectLanguage(path)
	return b.Save()
}

func (b *Buffer) LineCount() int {
	return len(b.Lines)
}

func (b *Buffer) LineLen(line int) int {
	if line < 0 || line >= len(b.Lines) {
		return 0
	}
	return utf8.RuneCountInString(b.Lines[line])
}

func (b *Buffer) GetLine(line int) string {
	if line < 0 || line >= len(b.Lines) {
		return ""
	}
	return b.Lines[line]
}

func (b *Buffer) SetLine(line int, content string) {
	if line >= 0 && line < len(b.Lines) {
		b.Lines[line] = content
		b.Modified = true
	}
}

func (b *Buffer) InsertChar(line, col int, ch string) (int, int) {
	if line < 0 || line >= len(b.Lines) {
		return line, col
	}
	runes := []rune(b.Lines[line])
	if col < 0 {
		col = 0
	}
	if col > len(runes) {
		col = len(runes)
	}
	newRunes := make([]rune, 0, len(runes)+utf8.RuneCountInString(ch))
	newRunes = append(newRunes, runes[:col]...)
	newRunes = append(newRunes, []rune(ch)...)
	newRunes = append(newRunes, runes[col:]...)
	b.Lines[line] = string(newRunes)
	b.Modified = true
	return line, col + utf8.RuneCountInString(ch)
}

func (b *Buffer) InsertNewline(line, col int) (int, int) {
	if line < 0 || line >= len(b.Lines) {
		return line, col
	}
	runes := []rune(b.Lines[line])
	if col < 0 {
		col = 0
	}
	if col > len(runes) {
		col = len(runes)
	}

	left := string(runes[:col])
	right := string(runes[col:])

	indent := ""
	for _, r := range left {
		if r == ' ' || r == '\t' {
			indent += string(r)
		} else {
			break
		}
	}

	if col > 0 && (runes[col-1] == '{' || runes[col-1] == '(' || runes[col-1] == '[') {
		indent += "\t"
	}

	b.Lines[line] = left
	newLines := make([]string, 0, len(b.Lines)+1)
	newLines = append(newLines, b.Lines[:line+1]...)
	newLines = append(newLines, indent+right)
	newLines = append(newLines, b.Lines[line+1:]...)
	b.Lines = newLines
	b.Modified = true
	return line + 1, utf8.RuneCountInString(indent)
}

func (b *Buffer) Backspace(line, col int) (int, int) {
	if line < 0 || line >= len(b.Lines) {
		return line, col
	}
	runes := []rune(b.Lines[line])
	if col > len(runes) {
		col = len(runes)
	}
	if col > 0 {
		newRunes := make([]rune, 0, len(runes)-1)
		newRunes = append(newRunes, runes[:col-1]...)
		newRunes = append(newRunes, runes[col:]...)
		b.Lines[line] = string(newRunes)
		b.Modified = true
		return line, col - 1
	}
	if line > 0 {
		prevLen := utf8.RuneCountInString(b.Lines[line-1])
		b.Lines[line-1] += b.Lines[line]
		newLines := make([]string, 0, len(b.Lines)-1)
		newLines = append(newLines, b.Lines[:line]...)
		newLines = append(newLines, b.Lines[line+1:]...)
		b.Lines = newLines
		b.Modified = true
		return line - 1, prevLen
	}
	return line, col
}

// SmartBackspace removes a tab/4-spaces worth of indentation when the cursor
// is in a line's leading whitespace, so users can unindent with Backspace.
func (b *Buffer) SmartBackspace(line, col int) (int, int) {
	if line < 0 || line >= len(b.Lines) {
		return line, col
	}
	runes := []rune(b.Lines[line])
	if col > len(runes) {
		col = len(runes)
	}
	if col == 0 {
		return b.Backspace(line, col)
	}
	// Walk back over the leading-whitespace region only.
	wsEnd := 0
	for wsEnd < len(runes) && (runes[wsEnd] == ' ' || runes[wsEnd] == '\t') {
		wsEnd++
	}
	if wsEnd == 0 || col > wsEnd {
		return b.Backspace(line, col)
	}
	remove := 0
	if runes[col-1] == '\t' {
		remove = 1
	} else {
		count := 0
		for k := col - 1; k >= 0 && runes[k] == ' ' && count < 4; k-- {
			count++
		}
		if count == 0 {
			count = 1
		}
		remove = count
	}
	newRunes := make([]rune, 0, len(runes))
	newRunes = append(newRunes, runes[:col-remove]...)
	newRunes = append(newRunes, runes[col:]...)
	b.Lines[line] = string(newRunes)
	b.Modified = true
	return line, col - remove
}

func (b *Buffer) Delete(line, col int) (int, int) {
	if line < 0 || line >= len(b.Lines) {
		return line, col
	}
	if col < 0 {
		col = 0
	}
	runes := []rune(b.Lines[line])
	if col < len(runes) {
		newRunes := make([]rune, 0, len(runes)-1)
		newRunes = append(newRunes, runes[:col]...)
		newRunes = append(newRunes, runes[col+1:]...)
		b.Lines[line] = string(newRunes)
		b.Modified = true
	} else if line < len(b.Lines)-1 {
		b.Lines[line] += b.Lines[line+1]
		newLines := make([]string, 0, len(b.Lines)-1)
		newLines = append(newLines, b.Lines[:line+1]...)
		newLines = append(newLines, b.Lines[line+2:]...)
		b.Lines = newLines
		b.Modified = true
	}
	return line, col
}

func (b *Buffer) DeleteLine(line int) {
	if line < 0 || line >= len(b.Lines) {
		return
	}
	if len(b.Lines) == 1 {
		b.Lines[0] = ""
	} else {
		newLines := make([]string, 0, len(b.Lines)-1)
		newLines = append(newLines, b.Lines[:line]...)
		newLines = append(newLines, b.Lines[line+1:]...)
		b.Lines = newLines
	}
	b.Modified = true
}

func (b *Buffer) DuplicateLine(line int) {
	if line < 0 || line >= len(b.Lines) {
		return
	}
	newLines := make([]string, 0, len(b.Lines)+1)
	newLines = append(newLines, b.Lines[:line+1]...)
	newLines = append(newLines, b.Lines[line])
	newLines = append(newLines, b.Lines[line+1:]...)
	b.Lines = newLines
	b.Modified = true
}

func (b *Buffer) MoveLineUp(line int) int {
	if line <= 0 || line >= len(b.Lines) {
		return line
	}
	b.Lines[line-1], b.Lines[line] = b.Lines[line], b.Lines[line-1]
	b.Modified = true
	return line - 1
}

func (b *Buffer) MoveLineDown(line int) int {
	if line < 0 || line >= len(b.Lines)-1 {
		return line
	}
	b.Lines[line], b.Lines[line+1] = b.Lines[line+1], b.Lines[line]
	b.Modified = true
	return line + 1
}

func (b *Buffer) GetSelectionText(startLine, startCol, endLine, endCol int) string {
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, startCol, endLine, endCol = endLine, endCol, startLine, startCol
	}
	if startLine == endLine {
		runes := []rune(b.Lines[startLine])
		if startCol > len(runes) {
			startCol = len(runes)
		}
		if endCol > len(runes) {
			endCol = len(runes)
		}
		return string(runes[startCol:endCol])
	}
	var sb strings.Builder
	runes := []rune(b.Lines[startLine])
	if startCol <= len(runes) {
		sb.WriteString(string(runes[startCol:]))
	}
	sb.WriteString("\n")
	for i := startLine + 1; i < endLine; i++ {
		sb.WriteString(b.Lines[i])
		sb.WriteString("\n")
	}
	runes = []rune(b.Lines[endLine])
	if endCol > len(runes) {
		endCol = len(runes)
	}
	sb.WriteString(string(runes[:endCol]))
	return sb.String()
}

func (b *Buffer) GetWordAt(line, col int) string {
	if line < 0 || line >= len(b.Lines) {
		return ""
	}
	runes := []rune(b.Lines[line])
	if len(runes) == 0 {
		return ""
	}
	if col >= len(runes) {
		col = len(runes) - 1
	}
	if col < 0 {
		return ""
	}
	start := col
	for start > 0 && IsWordChar(runes[start-1]) {
		start--
	}
	end := col
	for end < len(runes) && IsWordChar(runes[end]) {
		end++
	}
	if start == end {
		return ""
	}
	return string(runes[start:end])
}

func IsWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py", ".pyw":
		return "python"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hxx":
		return "cpp"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".sh", ".bash", ".zsh":
		return "shell"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".xml", ".svg":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss", ".sass":
		return "scss"
	case ".md", ".markdown":
		return "markdown"
	case ".sql":
		return "sql"
	case ".lua":
		return "lua"
	case ".vim":
		return "vim"
	case ".dockerfile":
		return "dockerfile"
	case ".makefile":
		return "makefile"
	case ".ini", ".cfg":
		return "ini"
	case ".txt", ".log":
		return "text"
	default:
		name := strings.ToLower(filepath.Base(path))
		switch name {
		case "dockerfile", "makefile", "rakefile", "gemfile":
			return name
		}
		return "text"
	}
}

func FromString(content string) *Buffer {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	return &Buffer{
		Lines:    lines,
		Modified: true,
		Name:     "[scratch]",
		Language: "text",
	}
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ReadDir(dir string) ([]os.DirEntry, error) {
	return os.ReadDir(dir)
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ScanLines is exported for potential external use
func ScanLines(data string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}
func (b *Buffer) DeleteWordLeft(line, col int) (int, int) {
	if line < 0 || line >= len(b.Lines) {
		return line, col
	}
	if col <= 0 {
		if line > 0 {
			return b.Backspace(line, col)
		}
		return line, col
	}

	runes := []rune(b.Lines[line])
	if col > len(runes) {
		col = len(runes)
	}

	i := col - 1
	// Skip trailing whitespace
	for i >= 0 && (runes[i] == ' ' || runes[i] == '\t') {
		i--
	}
	// Delete until next whitespace or start of line
	if i >= 0 && IsWordChar(runes[i]) {
		for i >= 0 && IsWordChar(runes[i]) {
			i--
		}
	} else if i >= 0 {
		// Just delete one non-word char if we started on one
		i--
	}

	newRunes := make([]rune, 0, len(runes)-(col-(i+1)))
	newRunes = append(newRunes, runes[:i+1]...)
	newRunes = append(newRunes, runes[col:]...)
	b.Lines[line] = string(newRunes)
	b.Modified = true
	return line, i + 1
}

func (b *Buffer) FindMatchingBracket(line, col int) (int, int) {
	if line < 0 || line >= len(b.Lines) {
		return -1, -1
	}
	runes := []rune(b.Lines[line])
	if col < 0 || col >= len(runes) {
		return -1, -1
	}

	char := runes[col]
	pairs := map[rune]rune{
		'(': ')', ')': '(',
		'{': '}', '}': '{',
		'[': ']', ']': '[',
	}

	other, ok := pairs[char]
	if !ok {
		return -1, -1
	}

	direction := 1
	if char == ')' || char == '}' || char == ']' {
		direction = -1
	}

	stack := 0
	currLine := line
	currCol := col

	for {
		currCol += direction
		if direction == 1 {
			for currLine < len(b.Lines) && currCol >= len([]rune(b.Lines[currLine])) {
				currLine++
				currCol = 0
			}
			if currLine >= len(b.Lines) {
				return -1, -1
			}
		} else {
			for currLine >= 0 && currCol < 0 {
				currLine--
				if currLine >= 0 {
					currCol = len([]rune(b.Lines[currLine])) - 1
				}
			}
			if currLine < 0 {
				return -1, -1
			}
		}

		lineRunes := []rune(b.Lines[currLine])
		if currCol < 0 || currCol >= len(lineRunes) {
			return -1, -1
		}

		currChar := lineRunes[currCol]
		if currChar == char {
			stack++
		} else if currChar == other {
			if stack == 0 {
				return currLine, currCol
			}
			stack--
		}

		// Limit search to 5000 lines to avoid hanging on large files without matches
		if direction == 1 && currLine > line+5000 {
			return -1, -1
		}
		if direction == -1 && currLine < line-5000 {
			return -1, -1
		}
	}
}