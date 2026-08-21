package textbuf

import (
	"os"
	"unicode/utf16"
)

// EOLKind classifies the newline style of a buffer's current content.
type EOLKind uint8

const (
	EOLLF    EOLKind = iota // only bare LF (or no newlines at all)
	EOLCRLF                 // every LF is preceded by CR
	EOLMixed                // both bare LF and CRLF occur
)

type bomKind uint8

const (
	bomNone bomKind = iota
	bomUTF8
	bomUTF16LE
	bomUTF16BE
)

// Open reads path into a buffer. A leading BOM is stripped and remembered so
// Save can restore it; UTF-16 files are transcoded to UTF-8 for editing and
// transcoded back on Save. Content bytes are otherwise kept verbatim.
func Open(path string) (*Buffer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	kind, body := splitBOM(data)
	switch kind {
	case bomUTF16LE:
		body = utf16BytesToUTF8(body, false)
	case bomUTF16BE:
		body = utf16BytesToUTF8(body, true)
	}
	b := newFromBytes(body)
	b.bom = kind
	return b, nil
}

func splitBOM(data []byte) (kind bomKind, rest []byte) {
	switch {
	case len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF:
		return bomUTF8, data[3:]
	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE:
		return bomUTF16LE, data[2:]
	case len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF:
		return bomUTF16BE, data[2:]
	default:
		return bomNone, data
	}
}

func utf16BytesToUTF8(data []byte, bigEndian bool) []byte {
	units := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 { // dangling odd byte is dropped
		if bigEndian {
			units = append(units, uint16(data[i])<<8|uint16(data[i+1]))
		} else {
			units = append(units, uint16(data[i+1])<<8|uint16(data[i]))
		}
	}
	return []byte(string(utf16.Decode(units)))
}

func utf8ToUTF16Bytes(s string, bigEndian bool) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 2*len(units))
	for i, u := range units {
		if bigEndian {
			out[2*i], out[2*i+1] = byte(u>>8), byte(u)
		} else {
			out[2*i], out[2*i+1] = byte(u), byte(u>>8)
		}
	}
	return out
}

// DominantEOL counts '\n' versus "\r\n" occurrences in the current content.
func (b *Buffer) DominantEOL() EOLKind {
	lf, crlf := 0, 0
	prevCR := false
	for i, n := 0, b.Len(); i < n; i++ {
		c := b.byteAt(i)
		if c == '\n' {
			lf++
			if prevCR {
				crlf++
			}
		}
		prevCR = c == '\r'
	}
	switch {
	case lf == 0 || crlf == 0:
		return EOLLF
	case lf == crlf:
		return EOLCRLF
	default:
		return EOLMixed
	}
}

// Save writes the buffer to path in the encoding it was opened with: the
// original BOM is re-prepended and UTF-16 content is re-encoded. No line
// ending conversion ever happens. Missing parent directories are not created;
// the os error is returned as-is.
func (b *Buffer) Save(path string) error {
	content := b.Text()
	var out []byte
	switch b.bom {
	case bomUTF8:
		out = append([]byte{0xEF, 0xBB, 0xBF}, content...)
	case bomUTF16LE:
		out = append([]byte{0xFF, 0xFE}, utf8ToUTF16Bytes(content, false)...)
	case bomUTF16BE:
		out = append([]byte{0xFE, 0xFF}, utf8ToUTF16Bytes(content, true)...)
	default:
		out = []byte(content)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return err
	}
	b.savedEdits = b.edits
	return nil
}

// SaveAs writes the current content to path verbatim as UTF-8 with no BOM and
// no conversion of any kind — the escape hatch for brand-new buffers.
func (b *Buffer) SaveAs(path string) error {
	if err := os.WriteFile(path, []byte(b.Text()), 0o644); err != nil {
		return err
	}
	b.savedEdits = b.edits
	return nil
}
