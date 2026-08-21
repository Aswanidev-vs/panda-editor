package input

import (
	"bytes"
	"unicode/utf8"
)

// Parser converts the raw terminal byte stream into Events incrementally.
// It is a byte-at-a-time state machine so that escape sequences split
// across reads still assemble into single events, and it never allocates
// in steady state for ordinary key input.
type Parser struct {
	state pstate
	out   []Event // scratch batch; reused between Feed/Flush calls

	csi []byte // accumulated CSI parameter/intermediate bytes

	mouseN  int // payload bytes collected toward the 3-byte X10 report
	mouseBC [3]byte

	utfNeed int               // continuation bytes still expected
	utfBuf  [utf8.UTFMax]byte // lead + continuations collected so far
	utfLen  int
	utfAlt  bool // rune was prefixed by ESC (option-key composition)

	paste     []byte // bracketed-paste content (terminator candidates excluded)
	pastePend []byte // trailing bytes that may turn out to be the terminator
	pasteM    int    // length of the terminator prefix currently matched
}

type pstate uint8

const (
	stGround   pstate = iota // plain bytes / UTF-8 text
	stEsc                    // lone ESC seen, awaiting disambiguation
	stCSI                    // inside ESC [ ... awaiting final byte
	stSS3                    // inside ESC O, one byte remains
	stMouseRaw               // collecting the 3 raw X10 mouse payload bytes
	stPaste                  // inside bracketed paste until ESC[201~
)

// maxPaste caps bracketed paste accumulation at 1 MiB; content beyond the
// cap is dropped but the paste is still delivered as a single event.
const maxPaste = 1 << 20

var pasteTerm = []byte("\x1b[201~")

// NewParser returns a parser ready to consume the terminal byte stream.
func NewParser() *Parser {
	return &Parser{csi: make([]byte, 0, 16)}
}

// Feed consumes one byte and returns every event it completed (0..n).
//
// The returned slice aliases an internal buffer that is invalidated by the
// next Feed or Flush call — forward or copy the events before feeding
// again. Pump does exactly that, which is what keeps this allocation-free.
func (p *Parser) Feed(b byte) []Event {
	switch p.state {
	case stGround:
		p.ground(b)
	case stEsc:
		p.afterEsc(b)
	case stCSI:
		p.csiByte(b)
	case stSS3:
		p.ss3(b)
	case stMouseRaw:
		p.mouseRaw(b)
	case stPaste:
		p.pasteByte(b)
	}
	return p.take()
}

// Flush resolves whatever is mid-flight at a stream boundary (EOF or an
// idle gap): a lone ESC becomes KeyEscape, truncated CSI/SS3/mouse/UTF-8
// sequences are discarded silently, and an unterminated paste is delivered
// with whatever accumulated. Afterwards the parser is reusable.
func (p *Parser) Flush() []Event {
	switch p.state {
	case stEsc:
		p.emit(KeyPress{Key: KeyEscape})
	case stPaste:
		p.stage(p.pastePend) // near-miss terminator bytes were real content
		if len(p.paste) > 0 {
			p.emit(Paste{Text: string(p.paste)})
		}
	}
	p.state = stGround
	p.clearUTF8()
	p.mouseN = 0
	p.paste = p.paste[:0]
	p.pastePend = p.pastePend[:0]
	p.pasteM = 0
	return p.take()
}

// Pending reports whether the parser sits mid-sequence and needs more
// bytes — or a Flush once a timeout decides the sequence is over.
func (p *Parser) Pending() bool {
	return p.state != stGround || p.utfNeed > 0
}

func (p *Parser) emit(e Event) { p.out = append(p.out, e) }

// take hands out the batch and resets the scratch buffer without freeing
// its capacity.
func (p *Parser) take() []Event {
	if len(p.out) == 0 {
		return nil
	}
	out := p.out
	p.out = p.out[:0]
	return out
}

// ground processes a plain byte outside any escape sequence. A partial
// UTF-8 rune takes priority: continuation bytes are folded into it, and an
// unexpected byte aborts the rune as RuneError before being reprocessed.
func (p *Parser) ground(b byte) {
	if p.utfNeed > 0 {
		if b >= 0x80 && b < 0xC0 {
			p.utfBuf[p.utfLen] = b
			p.utfLen++
			p.utfNeed--
			if p.utfNeed == 0 {
				r, _ := utf8.DecodeRune(p.utfBuf[:p.utfLen])
				p.emit(KeyPress{Rune: r, Mod: altOf(p.utfAlt)})
				p.clearUTF8()
			}
			return
		}
		// Broken sequence: emit replacement for what we had, then treat
		// the current byte as if starting fresh.
		p.emit(KeyPress{Rune: utf8.RuneError, Mod: altOf(p.utfAlt)})
		p.clearUTF8()
	}

	switch {
	case b == 0x1B:
		p.state = stEsc
	case b >= 0x20 && b < 0x7F:
		p.emit(KeyPress{Rune: rune(b)})
	case b >= 0x80:
		p.groundHigh(b)
	default:
		p.ctrlKey(b, 0) // C0 controls plus DEL
	}
}

// groundHigh classifies a high byte: a valid UTF-8 lead starts assembly,
// anything else is reported as RuneError.
func (p *Parser) groundHigh(b byte) {
	switch {
	case b >= 0xC2 && b <= 0xDF:
		p.startUTF8(b, 1)
	case b >= 0xE0 && b <= 0xEF:
		p.startUTF8(b, 2)
	case b >= 0xF0 && b <= 0xF4:
		p.startUTF8(b, 3)
	default: // stray continuation or invalid lead (C0/C1, F5..FF)
		p.emit(KeyPress{Rune: utf8.RuneError})
	}
}

func (p *Parser) startUTF8(lead byte, cont int) {
	p.utfBuf[0] = lead
	p.utfLen = 1
	p.utfNeed = cont
}

func (p *Parser) clearUTF8() {
	p.utfLen, p.utfNeed, p.utfAlt = 0, 0, false
}

func altOf(on bool) Mod {
	if on {
		return ModAlt
	}
	return 0
}

// ctrlKey maps C0 control bytes (and DEL) onto keys. extra folds in
// modifiers such as Alt from ESC-prefixed forms.
func (p *Parser) ctrlKey(b byte, extra Mod) {
	switch b {
	case '\r', '\n':
		p.emit(KeyPress{Key: KeyEnter, Mod: extra})
	case '\t':
		p.emit(KeyPress{Key: KeyTab, Mod: extra})
	case 0x7F:
		p.emit(KeyPress{Key: KeyBackspace, Mod: extra})
	case 0x08:
		p.emit(KeyPress{Rune: 'h', Mod: ModCtrl | extra})
	case 0x00:
		p.emit(KeyPress{Rune: ' ', Mod: ModCtrl | extra})
	default:
		switch {
		case b <= 26: // 0x01..0x1A -> ctrl+a..ctrl+z (tab/lf/cr handled above)
			p.emit(KeyPress{Rune: 'a' + rune(b) - 1, Mod: ModCtrl | extra})
		default: // 0x1C..0x1F -> ctrl+\ ] ^ _
			p.emit(KeyPress{Rune: rune(b) + 0x40, Mod: ModCtrl | extra})
		}
	}
}

// afterEsc disambiguates a lone ESC: '[' enters CSI, 'O' enters SS3,
// another ESC restarts (dropping the earlier one), anything else becomes
// an Alt-modified key.
func (p *Parser) afterEsc(b byte) {
	switch {
	case b == '[':
		p.state = stCSI
		p.csi = p.csi[:0]
	case b == 'O':
		p.state = stSS3
	case b == 0x1B:
		// Keep waiting; ESC ESC means the first ESC was not a keypress.
	default:
		p.state = stGround
		switch {
		case b >= 0x20 && b < 0x7F:
			p.emit(KeyPress{Rune: rune(b), Mod: ModAlt})
		case b >= 0x80:
			// macOS option-composition sends ESC + UTF-8 rune.
			p.utfAlt = true
			p.groundHigh(b)
		default:
			p.ctrlKey(b, ModAlt)
		}
	}
}

func (p *Parser) csiByte(b byte) {
	switch {
	case (b >= 0x20 && b <= 0x2F) || (b >= 0x30 && b <= 0x3F): // intermediates + params
		p.csi = append(p.csi, b)
	case b >= 0x40 && b <= 0x7E: // final byte
		p.state = stGround
		p.csiFinal(b)
	case b == 0x1B:
		p.state = stEsc // abort the partial sequence, start a fresh escape
	default:
		// Malformed byte mid-CSI: abandon the sequence but still deliver
		// the byte itself (e.g. ctrl+c typed during garbage).
		p.state = stGround
		p.ground(b)
	}
}

func (p *Parser) csiFinal(f byte) {
	params := p.csi
	switch f {
	case 'M':
		if len(params) == 0 {
			p.state = stMouseRaw // X10 mouse: three raw bytes follow
			p.mouseN = 0
			return
		}
		p.mouseDecimal(params, false)
	case 'm':
		if len(params) > 0 {
			p.mouseDecimal(params, true)
		}
	case '~':
		p.tilde(params)
	case 'u':
		p.kitty(params)
	case 'I':
		p.emit(Focus{Gained: true})
	case 'O':
		p.emit(Focus{Gained: false})
	case 'A':
		p.arrow(KeyUp, params)
	case 'B':
		p.arrow(KeyDown, params)
	case 'C':
		p.arrow(KeyRight, params)
	case 'D':
		p.arrow(KeyLeft, params)
	case 'H':
		p.arrow(KeyHome, params)
	case 'F':
		p.arrow(KeyEnd, params)
	default:
		// Unknown or private-mode response: swallow silently.
	}
}

func (p *Parser) arrow(k Key, params []byte) {
	p.emit(KeyPress{Key: k, Mod: xtermMods(params, 1)})
}

func (p *Parser) ss3(b byte) {
	p.state = stGround
	switch b {
	case 'P':
		p.emit(KeyPress{Key: KeyF1})
	case 'Q':
		p.emit(KeyPress{Key: KeyF2})
	case 'R':
		p.emit(KeyPress{Key: KeyF3})
	case 'S':
		p.emit(KeyPress{Key: KeyF4})
	case 'A':
		p.emit(KeyPress{Key: KeyUp})
	case 'B':
		p.emit(KeyPress{Key: KeyDown})
	case 'C':
		p.emit(KeyPress{Key: KeyRight})
	case 'D':
		p.emit(KeyPress{Key: KeyLeft})
	case 'H':
		p.emit(KeyPress{Key: KeyHome})
	case 'F':
		p.emit(KeyPress{Key: KeyEnd})
	case 'M':
		p.emit(KeyPress{Key: KeyEnter})
	default: // swallow
	}
}

func (p *Parser) tilde(params []byte) {
	n, ok := csiNum(params, 0)
	if !ok {
		return
	}
	switch n {
	case 200:
		p.state = stPaste
		p.paste = p.paste[:0]
		p.pastePend = p.pastePend[:0]
		p.pasteM = 0
		return
	case 201: // stray end marker without a start
		return
	}
	k, ok := tildeKey(n)
	if !ok {
		return
	}
	p.emit(KeyPress{Key: k, Mod: xtermMods(params, 1)})
}

// tildeKey maps xterm "~"-style function codes onto keys.
func tildeKey(n int) (Key, bool) {
	switch {
	case n == 1 || n == 7:
		return KeyHome, true
	case n == 2:
		return KeyInsert, true
	case n == 3:
		return KeyDelete, true
	case n == 4 || n == 8:
		return KeyEnd, true
	case n == 5:
		return KeyPageUp, true
	case n == 6:
		return KeyPageDown, true
	case n >= 11 && n <= 15:
		return KeyF1 + Key(n-11), true
	case n >= 17 && n <= 21:
		return KeyF6 + Key(n-17), true
	case n >= 23 && n <= 26:
		return KeyF11 + Key(n-23), true
	case n >= 28 && n <= 29:
		return KeyF15 + Key(n-28), true
	}
	return 0, false
}

// xtermMods decodes the modifier parameter at field idx of a CSI string:
// value minus one is a bitmask (1=Shift 2=Alt 4=Ctrl 8=Meta). A missing
// field means no modifiers.
func xtermMods(params []byte, idx int) Mod {
	v, ok := csiNum(params, idx)
	if !ok || v < 1 {
		return 0
	}
	return modFromMask(v - 1)
}

func modFromMask(v int) Mod {
	var m Mod
	if v&1 != 0 {
		m |= ModShift
	}
	if v&2 != 0 {
		m |= ModAlt
	}
	if v&4 != 0 {
		m |= ModCtrl
	}
	if v&8 != 0 {
		m |= ModMeta
	}
	return m
}

// csiNum returns the value of the idx-th ';'-separated decimal field;
// ok is false when the field is absent, empty or non-numeric.
func csiNum(params []byte, idx int) (int, bool) {
	start := 0
	for ; idx > 0; idx-- {
		i := bytes.IndexByte(params[start:], ';')
		if i < 0 {
			return 0, false
		}
		start += i + 1
	}
	end := len(params)
	if i := bytes.IndexByte(params[start:], ';'); i >= 0 {
		end = start + i
	}
	f := params[start:end]
	n := 0
	for _, c := range f {
		if c < '0' || c > '9' || n > 10_000_000 {
			return 0, false // guard against garbage overflowing the int
		}
		n = n*10 + int(c-'0')
	}
	if len(f) == 0 {
		return 0, false
	}
	return n, true
}

// kitty handles the Kitty keyboard protocol "CSI codepoint;mods u" form.
// Codepoints below 57344 behave like runes; functional codepoints map via
// kittyFuncKey; everything else is swallowed.
func (p *Parser) kitty(params []byte) {
	cp, ok := csiNum(params, 0)
	if !ok {
		return
	}
	var mod Mod
	if m, ok := csiNum(params, 1); ok && m >= 1 {
		mod = modFromMask(m - 1)
	}
	if cp < 57344 {
		p.emit(KeyPress{Rune: rune(cp), Mod: mod})
		return
	}
	if k, ok := kittyFuncKey(cp); ok {
		p.emit(KeyPress{Key: k, Mod: mod})
	}
}

// kittyFuncKey maps Kitty functional codepoints (57344+) onto keys for the
// common subset. Note the arrow order is left/right/up/down, unlike CSI.
func kittyFuncKey(cp int) (Key, bool) {
	switch cp {
	case 57344:
		return KeyEscape, true
	case 57345:
		return KeyEnter, true
	case 57346:
		return KeyTab, true
	case 57347:
		return KeyBackspace, true
	case 57348:
		return KeyInsert, true
	case 57349:
		return KeyDelete, true
	case 57350:
		return KeyLeft, true
	case 57351:
		return KeyRight, true
	case 57352:
		return KeyUp, true
	case 57353:
		return KeyDown, true
	case 57354:
		return KeyPageUp, true
	case 57355:
		return KeyPageDown, true
	case 57356:
		return KeyHome, true
	case 57357:
		return KeyEnd, true
	}
	if cp >= 57364 && cp <= 57383 { // F1..F20
		return KeyF1 + Key(cp-57364), true
	}
	return 0, false
}

func (p *Parser) mouseRaw(b byte) {
	p.mouseBC[p.mouseN] = b
	p.mouseN++
	if p.mouseN < 3 {
		return
	}
	p.state = stGround
	code := int(p.mouseBC[0]) - 32
	// X10/urxvt-style encoding signals release via button bits == 3.
	p.mouseEvent(code, int(p.mouseBC[1])-32, int(p.mouseBC[2])-32, code&3 == 3)
}

// mouseDecimal handles SGR ("<b;x;y") and urxvt ("b;x;y") reports. release
// comes from the SGR 'm' suffix; urxvt instead encodes release as button
// bits == 3 like the X10 encoding.
func (p *Parser) mouseDecimal(params []byte, release bool) {
	sgr := params[0] == '<'
	rest := params
	if sgr {
		rest = rest[1:]
	}
	b, ok1 := csiNum(rest, 0)
	x, ok2 := csiNum(rest, 1)
	y, ok3 := csiNum(rest, 2)
	if !ok1 || !ok2 || !ok3 {
		return
	}
	if !sgr {
		release = b&3 == 3
	}
	p.mouseEvent(b, x, y, release)
}

// mouseEvent builds the Mouse event from a decoded report code. The xterm
// wire encoding overlays shift=4 alt=8 ctrl=16 onto the button code; bit
// 32 means motion and bit 64 wheel. On X10-style releases the button bits
// read 3 and carry no identity, so Button stays 3 there.
func (p *Parser) mouseEvent(code, x, y int, release bool) {
	var mod Mod
	if code&4 != 0 {
		mod |= ModShift
	}
	if code&8 != 0 {
		mod |= ModAlt
	}
	if code&16 != 0 {
		mod |= ModCtrl
	}
	btn := code & 3
	switch {
	case code&64 != 0: // wheel: low bits are the direction index
		p.emit(Mouse{Action: MouseWheel, Wheel: WheelDir(btn), X: x, Y: y, Mod: mod})
	case code&32 != 0: // motion while dragging (or all-motion tracking)
		p.emit(Mouse{Action: MouseMove, Button: btn, X: x, Y: y, Mod: mod})
	default:
		act := MousePress
		if release {
			act = MouseRelease
		}
		p.emit(Mouse{Action: act, Button: btn, X: x, Y: y, Mod: mod})
	}
}

// pasteByte accumulates paste content verbatim — including ESC bytes —
// until the real terminator arrives. Terminator-shaped content is staged
// aside and flushed back if the next byte breaks the match.
func (p *Parser) pasteByte(b byte) {
	if b == pasteTerm[p.pasteM] {
		p.pastePend = append(p.pastePend, b)
		p.pasteM++
		if p.pasteM == len(pasteTerm) {
			p.emit(Paste{Text: string(p.paste)})
			p.state = stGround
			p.paste = p.paste[:0]
			p.pastePend = p.pastePend[:0]
			p.pasteM = 0
		}
		return
	}
	p.stage(p.pastePend)
	p.pastePend = p.pastePend[:0]
	p.pasteM = 0
	if b == pasteTerm[0] {
		p.pastePend = append(p.pastePend, b)
		p.pasteM = 1
		return
	}
	if len(p.paste) < maxPaste { // cap: keep consuming, drop the overflow
		p.paste = append(p.paste, b)
	}
}

func (p *Parser) stage(bs []byte) {
	for _, b := range bs {
		if len(p.paste) < maxPaste {
			p.paste = append(p.paste, b)
		}
	}
}
