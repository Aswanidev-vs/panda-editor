package buffer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	b := New()
	if b.LineCount() != 1 {
		t.Fatalf("New().LineCount() = %d, want 1", b.LineCount())
	}
	if b.LineLen(0) != 0 {
		t.Errorf("New().LineLen(0) = %d, want 0", b.LineLen(0))
	}
	if b.Modified {
		t.Error("New buffer should not be Modified")
	}
}

func TestInsertChar(t *testing.T) {
	b := New()
	l, c := b.InsertChar(0, 0, "h")
	if l != 0 || c != 1 {
		t.Fatalf("InsertChar returned (%d,%d), want (0,1)", l, c)
	}
	if b.GetLine(0) != "h" {
		t.Errorf("GetLine(0) = %q, want %q", b.GetLine(0), "h")
	}
	b.InsertChar(0, 1, "i")
	if b.GetLine(0) != "hi" {
		t.Errorf("GetLine(0) = %q, want %q", b.GetLine(0), "hi")
	}
	if !b.Modified {
		t.Error("InsertChar should mark buffer Modified")
	}
}

func TestInsertNewlineAutoIndent(t *testing.T) {
	b := New()
	b.Lines = []string{"func x() {"}
	line, col := b.InsertNewline(0, len([]rune("func x() {")))
	if line != 1 {
		t.Fatalf("InsertNewline line=%d, want 1", line)
	}
	if col != 1 {
		t.Errorf("InsertNewline col=%d, want 1 (auto-indent tab)", col)
	}
	if !strings.HasPrefix(b.GetLine(1), "\t") {
		t.Errorf("next line should start with tab, got %q", b.GetLine(1))
	}
}

func TestBackspace(t *testing.T) {
	b := New()
	b.SetLine(0, "abc")
	b.Modified = false
	l, c := b.Backspace(0, 3)
	if l != 0 || c != 2 {
		t.Fatalf("Backspace returned (%d,%d), want (0,2)", l, c)
	}
	if b.GetLine(0) != "ab" {
		t.Errorf("GetLine(0) = %q, want %q", b.GetLine(0), "ab")
	}
}

func TestSmartBackspaceUnindent(t *testing.T) {
	b := New()
	b.SetLine(0, "\t\thello")
	// Cursor between tabs and hello -> at col 2.
	_, c := b.SmartBackspace(0, 2)
	if c != 1 {
		t.Errorf("SmartBackspace at tab col=2 returned col=%d, want 1", c)
	}
	if b.GetLine(0) != "\thello" {
		t.Errorf("after SmartBackspace line = %q, want %q", b.GetLine(0), "\thello")
	}
}

func TestSmartBackspaceSpaces(t *testing.T) {
	b := New()
	b.SetLine(0, "        hello") // 8 spaces
	// Cursor at col 4 (in the middle of leading spaces).
	_, c := b.SmartBackspace(0, 4)
	if c != 0 {
		t.Errorf("SmartBackspace at spaces col=4 returned col=%d, want 0", c)
	}
	if b.GetLine(0) != "    hello" {
		t.Errorf("after SmartBackspace line = %q, want %q", b.GetLine(0), "    hello")
	}
}

func TestSmartBackspaceFallsBackToNormal(t *testing.T) {
	b := New()
	b.SetLine(0, "    hello")
	// Cursor at col 6 (between 'l' and 'l'), not in leading whitespace.
	_, c := b.SmartBackspace(0, 6)
	if c != 5 {
		t.Errorf("SmartBackspace inside word should fall back; col=%d, want 5", c)
	}
	// 'e' (at index 5) is removed: "    hllo"
	if b.GetLine(0) != "    hllo" {
		t.Errorf("after SmartBackspace line = %q, want %q", b.GetLine(0), "    hllo")
	}
}

func TestDeleteLine(t *testing.T) {
	b := New()
	b.Lines = []string{"a", "b", "c"}
	b.DeleteLine(1)
	if b.LineCount() != 2 {
		t.Fatalf("LineCount after DeleteLine = %d, want 2", b.LineCount())
	}
	if b.GetLine(0) != "a" || b.GetLine(1) != "c" {
		t.Errorf("lines after delete = (%q,%q), want (a,c)", b.GetLine(0), b.GetLine(1))
	}
}

func TestDeleteLineOnlyLine(t *testing.T) {
	b := New()
	b.Lines = []string{"only"}
	b.DeleteLine(0)
	if b.LineCount() != 1 {
		t.Errorf("LineCount after deleting only line = %d, want 1", b.LineCount())
	}
	if b.GetLine(0) != "" {
		t.Errorf("GetLine(0) after deleting only line = %q, want %q", b.GetLine(0), "")
	}
}

func TestDuplicateLine(t *testing.T) {
	b := New()
	b.Lines = []string{"a", "b", "c"}
	b.DuplicateLine(1)
	if b.LineCount() != 4 {
		t.Errorf("LineCount after DuplicateLine = %d, want 4", b.LineCount())
	}
	if b.GetLine(1) != "b" || b.GetLine(2) != "b" {
		t.Errorf("lines around dup = (%q,%q), want (b,b)", b.GetLine(1), b.GetLine(2))
	}
}

func TestMoveLineUpDown(t *testing.T) {
	b := New()
	b.Lines = []string{"a", "b", "c"}
	newLine := b.MoveLineDown(0)
	if newLine != 1 {
		t.Errorf("MoveLineDown(0) = %d, want 1", newLine)
	}
	if b.GetLine(0) != "b" || b.GetLine(1) != "a" {
		t.Errorf("after MoveLineDown(0) = (%q,%q), want (b,a)", b.GetLine(0), b.GetLine(1))
	}
	newLine = b.MoveLineUp(1)
	if newLine != 0 {
		t.Errorf("MoveLineUp(1) = %d, want 0", newLine)
	}
}

func TestGetSelectionText(t *testing.T) {
	b := New()
	b.Lines = []string{"hello", "world", "foo"}
	// Select "lo wo" across two lines (col 3..6 spanning lines 0..1)
	got := b.GetSelectionText(0, 3, 1, 2)
	if got != "lo\nwo" {
		t.Errorf("GetSelectionText = %q, want %q", got, "lo\nwo")
	}
}

func TestIsWordChar(t *testing.T) {
	cases := []struct {
		r    rune
		want bool
	}{
		{'a', true}, {'Z', true}, {'5', true}, {'_', true},
		{'-', false}, {' ', false}, {'.', false},
	}
	for _, c := range cases {
		if got := IsWordChar(c.r); got != c.want {
			t.Errorf("IsWordChar(%q) = %v, want %v", c.r, got, c.want)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	cases := map[string]string{
		"foo.go":         "go",
		"main.py":        "python",
		"app.js":         "javascript",
		"x.ts":           "typescript",
		"lib.rs":         "rust",
		"Dockerfile":     "dockerfile",
		"unknown.xyz":    "text",
		"data.json":      "json",
		"config.yaml":    "yaml",
		"config.yml":     "yaml",
		"page.html":      "html",
		"style.css":      "css",
		"README.md":      "markdown",
		"script.sh":      "shell",
		"queries.sql":    "sql",
		"plugin.lua":     "lua",
		"build.toml":     "toml",
		"image.png":      "text", // unknown ext
		"random_file":    "text",
	}
	for path, want := range cases {
		got := DetectLanguage(path)
		if got != want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestFromString(t *testing.T) {
	b := FromString("hello\nworld")
	if b.LineCount() != 2 {
		t.Errorf("LineCount = %d, want 2", b.LineCount())
	}
	if b.GetLine(0) != "hello" || b.GetLine(1) != "world" {
		t.Errorf("lines = (%q,%q), want (hello,world)", b.GetLine(0), b.GetLine(1))
	}
	if !b.Modified {
		t.Error("FromString should mark buffer Modified")
	}
}

func TestFindMatchingBracket(t *testing.T) {
	b := New()
	b.Lines = []string{"(hello)"}
	l, c := b.FindMatchingBracket(0, 0)
	if l != 0 || c != 6 {
		t.Errorf("FindMatchingBracket at '(' = (%d,%d), want (0,6)", l, c)
	}
	l, c = b.FindMatchingBracket(0, 6)
	if l != 0 || c != 0 {
		t.Errorf("FindMatchingBracket at ')' = (%d,%d), want (0,0)", l, c)
	}
	// No match for non-bracket
	l, c = b.FindMatchingBracket(0, 1)
	if l != -1 || c != -1 {
		t.Errorf("FindMatchingBracket at letter = (%d,%d), want (-1,-1)", l, c)
	}
}

func TestDeleteWordLeft(t *testing.T) {
	b := New()
	b.SetLine(0, "hello world")
	b.Modified = false
	l, c := b.DeleteWordLeft(0, len("hello world"))
	if l != 0 || c != 6 {
		t.Errorf("DeleteWordLeft end returned (%d,%d), want (0,6)", l, c)
	}
	if b.GetLine(0) != "hello " {
		t.Errorf("after DeleteWordLeft line = %q, want %q", b.GetLine(0), "hello ")
	}
}

func TestOpenFileSizeLimit(t *testing.T) {
	// Create a temp file just over the limit using a sparse layout.
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	// Seek to MaxOpenBytes + 1
	_, err = f.WriteAt([]byte("x"), MaxOpenBytes)
	if err != nil {
		t.Fatalf("WriteAt failed: %v", err)
	}
	_ = f.Close()

	_, err = Open(path)
	if err == nil {
		t.Fatal("Open on oversized file should fail")
	}
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("Open error should wrap ErrFileTooLarge, got %v", err)
	}
}

func TestOpenFileSmall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\n"), 0644); err != nil {
		t.Fatal(err)
	}
	b, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if b.LineCount() != 4 {
		t.Errorf("LineCount = %d, want 4", b.LineCount())
	}
	if b.LineCount() >= 1 && b.GetLine(0) != "a" {
		t.Errorf("GetLine(0) = %q, want %q", b.GetLine(0), "a")
	}
}

func TestGetWordAt(t *testing.T) {
	b := New()
	b.SetLine(0, "hello world")
	got := b.GetWordAt(0, 2)
	if got != "hello" {
		t.Errorf("GetWordAt(0,2) = %q, want %q", got, "hello")
	}
	got = b.GetWordAt(0, 8)
	if got != "world" {
		t.Errorf("GetWordAt(0,8) = %q, want %q", got, "world")
	}
}