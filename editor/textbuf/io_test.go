package textbuf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Open → Save must be byte-identical for a UTF-8 file with BOM and CRLF
// endings: no normalization anywhere.
func TestRoundtripCRLFWithBOM(t *testing.T) {
	orig := append([]byte{0xEF, 0xBB, 0xBF}, []byte("first\r\nsecond\r\nthird\r\n")...)
	src := writeFile(t, "crlf_bom.txt", orig)

	b, err := Open(src)
	if err != nil {
		t.Fatal(err)
	}
	if b.DominantEOL() != EOLCRLF {
		t.Errorf("DominantEOL = %v, want EOLCRLF", b.DominantEOL())
	}
	dst := filepath.Join(t.TempDir(), "out.txt")
	if err := b.Save(dst); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dst); !bytes.Equal(got, orig) {
		t.Fatalf("roundtrip mismatch:\n got %x\nwant %x", got, orig)
	}
	if b.Modified() {
		t.Error("freshly opened buffer should not be modified")
	}
}

// Mixed line endings survive untouched and are classified as mixed.
func TestRoundtripMixedEOL(t *testing.T) {
	orig := []byte("a\r\nb\nc\r\nd")
	src := writeFile(t, "mixed.txt", orig)

	b, err := Open(src)
	if err != nil {
		t.Fatal(err)
	}
	if b.DominantEOL() != EOLMixed {
		t.Errorf("DominantEOL = %v, want EOLMixed", b.DominantEOL())
	}
	dst := filepath.Join(t.TempDir(), "out.txt")
	if err := b.Save(dst); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dst); !bytes.Equal(got, orig) {
		t.Fatalf("roundtrip mismatch: got %q, want %q", got, orig)
	}
}

func TestDominantEOLPureLFAndNone(t *testing.T) {
	if got := FromString("a\nb\nc").DominantEOL(); got != EOLLF {
		t.Errorf("LF doc DominantEOL = %v", got)
	}
	if got := New().DominantEOL(); got != EOLLF {
		t.Errorf("empty buffer DominantEOL = %v, want EOLLF", got)
	}
}

func utf16Bytes(s string, bigEndian bool) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, 2+2*len(units))
	if bigEndian {
		out = append(out, 0xFE, 0xFF)
	} else {
		out = append(out, 0xFF, 0xFE)
	}
	for _, u := range units {
		if bigEndian {
			out = append(out, byte(u>>8), byte(u))
		} else {
			out = append(out, byte(u), byte(u>>8))
		}
	}
	return out
}

// UTF-16LE roundtrips through edit-space as UTF-8 and back to identical bytes.
func TestRoundtripUTF16LE(t *testing.T) {
	text := "héllo\nwörld\n日本語\r\n"
	orig := utf16Bytes(text, false)
	src := writeFile(t, "utf16le.txt", orig)

	b, err := Open(src)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.Text(); got != text {
		t.Fatalf("decoded Text = %q, want %q", got, text)
	}
	if b.Line(2) != "日本語\r" {
		t.Errorf("Line(2) = %q, CR should survive as content", b.Line(2))
	}
	if b.Modified() {
		t.Error("opened buffer should not be modified")
	}
	dst := filepath.Join(t.TempDir(), "out.txt")
	if err := b.Save(dst); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dst); !bytes.Equal(got, orig) {
		t.Fatalf("UTF-16LE roundtrip mismatch:\n got %x\nwant %x", got, orig)
	}
}

func TestRoundtripUTF16BE(t *testing.T) {
	text := "line one\nline two"
	orig := utf16Bytes(text, true)
	src := writeFile(t, "utf16be.txt", orig)

	b, err := Open(src)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.Text(); got != text {
		t.Fatalf("decoded Text = %q, want %q", got, text)
	}
	dst := filepath.Join(t.TempDir(), "out.txt")
	if err := b.Save(dst); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dst); !bytes.Equal(got, orig) {
		t.Fatalf("UTF-16BE roundtrip mismatch:\n got %x\nwant %x", got, orig)
	}
}

// Editing a UTF-16 file keeps BOM and encoding after Save.
func TestEditUTF16ThenSaveStaysUTF16(t *testing.T) {
	src := writeFile(t, "in.txt", utf16Bytes("abc\ndef\n", false))
	b, err := Open(src)
	if err != nil {
		t.Fatal(err)
	}
	b.InsertRune(Pos{Line: 1, Col: 3}, '!')
	dst := filepath.Join(t.TempDir(), "out.txt")
	if err := b.Save(dst); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Text(); got != "abc\ndef!\n" {
		t.Fatalf("reopened Text = %q, want %q", got, "abc\ndef!\n")
	}
	want := utf16Bytes("abc\ndef!\n", false)
	if got := readFile(t, dst); !bytes.Equal(got, want) {
		t.Fatalf("saved bytes not UTF-16LE+BOM: %x", got)
	}
}

// SaveAs never converts: even a UTF-16-opened buffer lands as plain UTF-8.
func TestSaveAsWritesPlainUTF8(t *testing.T) {
	src := writeFile(t, "in.txt", utf16Bytes("böm\r\nfile", false))
	b, err := Open(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "out.txt")
	if err := b.SaveAs(dst); err != nil {
		t.Fatal(err)
	}
	want := []byte("böm\r\nfile") // raw content, no BOM, CR untouched
	if got := readFile(t, dst); !bytes.Equal(got, want) {
		t.Fatalf("SaveAs wrote %x, want %x", got, want)
	}
}

func TestSaveMissingParentDirReturnsError(t *testing.T) {
	dir := t.TempDir()
	b := FromString("data")
	err := b.Save(filepath.Join(dir, "nope", "out.txt"))
	if err == nil {
		t.Fatal("Save into missing directory should return the os error")
	}
	err = b.SaveAs(filepath.Join(dir, "nope", "out.txt"))
	if err == nil {
		t.Fatal("SaveAs into missing directory should return the os error")
	}
}

func TestOpenMissingFileErrors(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "ghost.txt")); err == nil {
		t.Fatal("Open of missing file should error")
	}
}

func TestSaveClearsModifiedFlag(t *testing.T) {
	b := FromString("content")
	b.Insert(b.Len(), " more")
	if !b.Modified() {
		t.Fatal("should be dirty before save")
	}
	dst := filepath.Join(t.TempDir(), "out.txt")
	if err := b.Save(dst); err != nil {
		t.Fatal(err)
	}
	if b.Modified() {
		t.Fatal("Save should clear the modified flag")
	}
}

func TestUTF8BOMStrippedForEditing(t *testing.T) {
	src := writeFile(t, "bom.txt", []byte{0xEF, 0xBB, 0xBF, 'h', 'i'})
	b, err := Open(src)
	if err != nil {
		t.Fatal(err)
	}
	if b.Text() != "hi" || b.Line(0)[0] != 'h' {
		t.Fatalf("BOM leaked into content: %q", b.Text())
	}
}
