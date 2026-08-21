package input

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// Event constructors keep the golden table terse.
func kp(k Key) Event          { return KeyPress{Key: k} }
func kpm(k Key, m Mod) Event  { return KeyPress{Key: k, Mod: m} }
func kr(r rune) Event         { return KeyPress{Rune: r} }
func krm(r rune, m Mod) Event { return KeyPress{Rune: r, Mod: m} }
func mp(a MouseAction, btn, x, y int, m Mod) Event {
	return Mouse{Action: a, Button: btn, X: x, Y: y, Mod: m}
}
func mw(d WheelDir, x, y int, m Mod) Event {
	return Mouse{Action: MouseWheel, Wheel: d, X: x, Y: y, Mod: m}
}

// E builds an expected slice; no-arg calls yield nil for "silence".
func E(evs ...Event) []Event { return evs }

// feedAll feeds every byte individually and gathers all events. The events
// are copied out immediately because Feed returns a reused buffer.
func feedAll(p *Parser, in string) []Event {
	var got []Event
	for i := 0; i < len(in); i++ {
		got = append(got, p.Feed(in[i])...)
	}
	got = append(got, p.Flush()...)
	return got
}

var goldenCases = []struct {
	name string
	in   string
	want []Event
}{
	// Printable & UTF-8.
	{"ascii_letter", "a", E(kr('a'))},
	{"ascii_batch", "Ab1", E(kr('A'), kr('b'), kr('1'))},
	{"space_bar", " ", E(kr(' '))},
	{"utf8_two_byte", "\u00e9", E(kr('\u00e9'))},
	{"utf8_three_byte", "\u20ac", E(kr('\u20ac'))},
	{"utf8_four_byte", "\U0001f600", E(kr('\U0001f600'))},
	{"utf8_mixed_run", "h\u00e9llo", E(kr('h'), kr('\u00e9'), kr('l'), kr('l'), kr('o'))},

	// Control bytes.
	{"cr_is_enter", "\r", E(kp(KeyEnter))},
	{"lf_is_enter", "\n", E(kp(KeyEnter))},
	{"tab", "\t", E(kp(KeyTab))},
	{"del_is_backspace", "\x7f", E(kp(KeyBackspace))},
	{"ctrl_h_from_bs", "\x08", E(krm('h', ModCtrl))},
	{"ctrl_a", "\x01", E(krm('a', ModCtrl))},
	{"ctrl_z", "\x1a", E(krm('z', ModCtrl))},
	{"nul_is_ctrl_space", "\x00", E(krm(' ', ModCtrl))},
	{"ctrl_backslash", "\x1c", E(krm('\\', ModCtrl))},
	{"ctrl_close_bracket", "\x1d", E(krm(']', ModCtrl))},
	{"ctrl_caret", "\x1e", E(krm('^', ModCtrl))},
	{"ctrl_underscore", "\x1f", E(krm('_', ModCtrl))},

	// ESC prefix (Alt).
	{"alt_a", "\x1ba", E(krm('a', ModAlt))},
	{"alt_upper", "\x1bZ", E(krm('Z', ModAlt))},
	{"alt_digit", "\x1b5", E(krm('5', ModAlt))},
	{"alt_enter", "\x1b\r", E(kpm(KeyEnter, ModAlt))},
	{"double_esc_then_key", "\x1b\x1ba", E(krm('a', ModAlt))},
	{"alt_multibyte_rune", "\x1b\u00e9", E(krm('\u00e9', ModAlt))},

	// CSI letter finals.
	{"up", "\x1b[A", E(kp(KeyUp))},
	{"down", "\x1b[B", E(kp(KeyDown))},
	{"right", "\x1b[C", E(kp(KeyRight))},
	{"left", "\x1b[D", E(kp(KeyLeft))},
	{"home_h", "\x1b[H", E(kp(KeyHome))},
	{"end_f", "\x1b[F", E(kp(KeyEnd))},
	{"arrows_run", "\x1b[A\x1b[B\x1b[C\x1b[D",
		E(kp(KeyUp), kp(KeyDown), kp(KeyRight), kp(KeyLeft))},

	// CSI tilde codes.
	{"home_1", "\x1b[1~", E(kp(KeyHome))},
	{"home_7", "\x1b[7~", E(kp(KeyHome))},
	{"insert_2", "\x1b[2~", E(kp(KeyInsert))},
	{"delete_3", "\x1b[3~", E(kp(KeyDelete))},
	{"end_4", "\x1b[4~", E(kp(KeyEnd))},
	{"end_8", "\x1b[8~", E(kp(KeyEnd))},
	{"pgup_5", "\x1b[5~", E(kp(KeyPageUp))},
	{"pgdown_6", "\x1b[6~", E(kp(KeyPageDown))},
	{"f1_tilde", "\x1b[11~", E(kp(KeyF1))},
	{"f2_tilde", "\x1b[12~", E(kp(KeyF2))},
	{"f3_tilde", "\x1b[13~", E(kp(KeyF3))},
	{"f4_tilde", "\x1b[14~", E(kp(KeyF4))},
	{"f5_tilde", "\x1b[15~", E(kp(KeyF5))},
	{"f6_tilde", "\x1b[17~", E(kp(KeyF6))},
	{"f7_tilde", "\x1b[18~", E(kp(KeyF7))},
	{"f8_tilde", "\x1b[19~", E(kp(KeyF8))},
	{"f9_tilde", "\x1b[20~", E(kp(KeyF9))},
	{"f10_tilde", "\x1b[21~", E(kp(KeyF10))},
	{"f11_tilde", "\x1b[23~", E(kp(KeyF11))},
	{"f12_tilde", "\x1b[24~", E(kp(KeyF12))},
	{"f13_tilde", "\x1b[25~", E(kp(KeyF13))},
	{"f14_tilde", "\x1b[26~", E(kp(KeyF14))},
	{"f15_tilde", "\x1b[28~", E(kp(KeyF15))},
	{"f16_tilde", "\x1b[29~", E(kp(KeyF16))},
	{"tilde_gaps_swallowed", "\x1b[16~\x1b[22~\x1b[27~", nil},

	// CSI modifiers (value-1 bitmask).
	{"shift_up", "\x1b[1;2A", E(kpm(KeyUp, ModShift))},
	{"alt_down", "\x1b[1;3B", E(kpm(KeyDown, ModAlt))},
	{"ctrl_right", "\x1b[1;5C", E(kpm(KeyRight, ModCtrl))},
	{"ctrl_shift_left", "\x1b[1;6D", E(kpm(KeyLeft, ModShift|ModCtrl))},
	{"alt_ctrl_up", "\x1b[1;7A", E(kpm(KeyUp, ModAlt|ModCtrl))},
	{"shift_alt_ctrl_home_h", "\x1b[1;8H", E(kpm(KeyHome, ModShift|ModAlt|ModCtrl))},
	{"meta_f3", "\x1b[13;9~", E(kpm(KeyF3, ModMeta))},
	{"ctrl_delete", "\x1b[3;5~", E(kpm(KeyDelete, ModCtrl))},
	{"shift_f5", "\x1b[15;2~", E(kpm(KeyF5, ModShift))},
	{"modifier_on_home_7", "\x1b[7;5~", E(kpm(KeyHome, ModCtrl))},

	// Unknown sequences swallowed silently.
	{"unknown_final_c", "\x1b[?1;2c", nil},
	{"unknown_private_mode_reply", "\x1b[?2004h", nil},
	{"unknown_tilde_9", "\x1b[9~", nil},
	{"backtab_z_swallowed", "\x1b[Z", nil},
	{"unknown_ss3_final", "\x1bOX", nil},
	{"empty_sgr_release_swallowed", "\x1b[m", nil},
	{"stray_paste_end_marker", "\x1b[201~", nil},

	// SS3.
	{"ss3_f1_to_f4", "\x1bOP\x1bOQ\x1bOR\x1bOS",
		E(kp(KeyF1), kp(KeyF2), kp(KeyF3), kp(KeyF4))},
	{"ss3_arrows", "\x1bOA\x1bOB\x1bOC\x1bOD",
		E(kp(KeyUp), kp(KeyDown), kp(KeyRight), kp(KeyLeft))},
	{"ss3_home_end", "\x1bOH\x1bOF", E(kp(KeyHome), kp(KeyEnd))},
	{"ss3_m_enter", "\x1bOM", E(kp(KeyEnter))},

	// SGR mouse.
	{"sgr_press_left", "\x1b[<0;10;5M", E(mp(MousePress, 0, 10, 5, 0))},
	{"sgr_release_left", "\x1b[<0;10;5m", E(mp(MouseRelease, 0, 10, 5, 0))},
	{"sgr_press_middle", "\x1b[<1;3;2M", E(mp(MousePress, 1, 3, 2, 0))},
	{"sgr_press_right", "\x1b[<2;3;2M", E(mp(MousePress, 2, 3, 2, 0))},
	{"sgr_motion_no_button", "\x1b[<32;4;6M", E(mp(MouseMove, 0, 4, 6, 0))},
	{"sgr_drag_button2", "\x1b[<34;4;6M", E(mp(MouseMove, 2, 4, 6, 0))},
	{"sgr_motion_ctrl", "\x1b[<48;7;9M", E(mp(MouseMove, 0, 7, 9, ModCtrl))},
	{"sgr_wheel_up", "\x1b[<64;2;3M", E(mw(WheelUp, 2, 3, 0))},
	{"sgr_wheel_down", "\x1b[<65;2;3M", E(mw(WheelDown, 2, 3, 0))},
	{"sgr_wheel_left", "\x1b[<66;2;3M", E(mw(WheelLeft, 2, 3, 0))},
	{"sgr_wheel_right", "\x1b[<67;2;3M", E(mw(WheelRight, 2, 3, 0))},
	{"sgr_wheel_shift", "\x1b[<68;2;3M", E(mw(WheelUp, 2, 3, ModShift))},
	{"sgr_press_shift", "\x1b[<4;1;1M", E(mp(MousePress, 0, 1, 1, ModShift))},
	{"sgr_press_alt", "\x1b[<8;1;1M", E(mp(MousePress, 0, 1, 1, ModAlt))},
	{"sgr_press_ctrl", "\x1b[<16;1;1M", E(mp(MousePress, 0, 1, 1, ModCtrl))},

	// X10 mouse (raw payload bytes +32).
	{"x10_press_left", "\x1b[M !\"", E(mp(MousePress, 0, 1, 2, 0))},
	{"x10_release", "\x1b[M#!\"", E(mp(MouseRelease, 3, 1, 2, 0))}, // button bits read 3 on release
	{"x10_wheel_up", "\x1b[M`!\"", E(mw(WheelUp, 1, 2, 0))},
	{"x10_motion", "\x1b[M@!\"", E(mp(MouseMove, 0, 1, 2, 0))},

	// urxvt decimal mouse.
	{"urxvt_press", "\x1b[0;10;5M", E(mp(MousePress, 0, 10, 5, 0))},
	{"urxvt_release", "\x1b[3;10;5M", E(mp(MouseRelease, 3, 10, 5, 0))},
	{"urxvt_motion", "\x1b[32;7;8M", E(mp(MouseMove, 0, 7, 8, 0))},
	{"urxvt_wheel_down", "\x1b[65;7;8M", E(mw(WheelDown, 7, 8, 0))},

	// Bracketed paste.
	{"paste_basic", "\x1b[200~hi there\x1b[201~", E(Paste{Text: "hi there"})},
	{"paste_empty", "\x1b[200~\x1b[201~", E(Paste{Text: ""})},
	{"paste_keeps_escapes", "\x1b[200~a\x1bb[c\x1b[201~", E(Paste{Text: "a\x1bb[c"})},
	{"paste_partial_terminator_lookalike", "\x1b[200~x\x1b[20y\x1b[201~",
		E(Paste{Text: "x\x1b[20y"})},
	{"paste_esc_lookalike_then_real_end", "\x1b[200~q\x1bAz\x1b[201~",
		E(Paste{Text: "q\x1bAz"})},
	{"paste_then_key", "\x1b[200~p\x1b[201~q", E(Paste{Text: "p"}, kr('q'))},
	{"paste_newlines_literal", "\x1b[200~l1\r\nl2\x1b[201~", E(Paste{Text: "l1\r\nl2"})},

	// Focus.
	{"focus_gained", "\x1b[I", E(Focus{Gained: true})},
	{"focus_lost", "\x1b[O", E(Focus{Gained: false})},

	// Kitty keyboard protocol.
	{"kitty_plain_rune", "\x1b[97u", E(kr('a'))},
	{"kitty_unicode_rune", "\x1b[233u", E(kr('\u00e9'))},
	{"kitty_shift", "\x1b[97;2u", E(krm('a', ModShift))},
	{"kitty_alt", "\x1b[97;3u", E(krm('a', ModAlt))},
	{"kitty_ctrl", "\x1b[97;5u", E(krm('a', ModCtrl))},
	{"kitty_alt_ctrl", "\x1b[97;7u", E(krm('a', ModAlt|ModCtrl))},
	{"kitty_meta", "\x1b[97;9u", E(krm('a', ModMeta))},
	{"kitty_all_mods", "\x1b[97;16u", E(krm('a', ModShift|ModAlt|ModCtrl|ModMeta))},
	{"kitty_capital_shifted", "\x1b[65;2u", E(krm('A', ModShift))},
	{"kitty_esc", "\x1b[57344u", E(kp(KeyEscape))},
	{"kitty_enter_tab_backspace", "\x1b[57345u\x1b[57346u\x1b[57347u",
		E(kp(KeyEnter), kp(KeyTab), kp(KeyBackspace))},
	{"kitty_insert_delete", "\x1b[57348u\x1b[57349u", E(kp(KeyInsert), kp(KeyDelete))},
	{"kitty_arrows_lrud", "\x1b[57350u\x1b[57351u\x1b[57352u\x1b[57353u",
		E(kp(KeyLeft), kp(KeyRight), kp(KeyUp), kp(KeyDown))},
	{"kitty_nav_keys", "\x1b[57354u\x1b[57355u\x1b[57356u\x1b[57357u",
		E(kp(KeyPageUp), kp(KeyPageDown), kp(KeyHome), kp(KeyEnd))},
	{"kitty_f_keys", "\x1b[57364u\x1b[57368u\x1b[57375u\x1b[57383u",
		E(kp(KeyF1), kp(KeyF5), kp(KeyF12), kp(KeyF20))},
	{"kitty_ctrl_up", "\x1b[57352;5u", E(kpm(KeyUp, ModCtrl))},
	{"kitty_capslock_not_mapped", "\x1b[57358u", nil},
	{"kitty_unknown_high_codepoint", "\x1b[60000u", nil},

	// Interleaving and recovery.
	{"interleaved_stream", "\x1b[Aq\x1b[<0;1;1M\rz",
		E(kp(KeyUp), kr('q'), mp(MousePress, 0, 1, 1, 0), kp(KeyEnter), kr('z'))},
	{"csi_restarted_by_esc", "\x1b[12\x1b[C", E(kp(KeyRight))},
	{"malformed_csi_still_delivers_control", "\x1b[1;\x03A",
		E(krm('c', ModCtrl), kr('A'))},
	{"broken_utf8_reports_replacement", "\xc3xy", E(kr(utf8.RuneError), kr('x'), kr('y'))},
}

func TestGolden(t *testing.T) {
	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			got := feedAll(NewParser(), tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("input %q\n got: %#v\nwant: %#v", tc.in, got, tc.want)
			}
		})
	}
}

// Every golden case must produce identical output when the sequence is
// split across separate Feed calls with Pending() observed in between —
// which feedAll already exercises byte-by-byte; here we additionally
// assert the pending flag transitions at the right moments.
func TestPendingLifecycle(t *testing.T) {
	p := NewParser()
	if p.Pending() {
		t.Fatal("fresh parser must not be pending")
	}
	if evs := p.Feed('x'); len(evs) != 1 || p.Pending() {
		t.Fatal("plain printable byte must complete without leaving pending state")
	}

	p = NewParser()
	p.Feed(0x1b)
	if !p.Pending() {
		t.Fatal("lone ESC must be pending until disambiguated")
	}
	p.Flush()
	if p.Pending() {
		t.Fatal("Flush must clear pending state")
	}

	p = NewParser()
	for _, b := range []byte("\x1b[") {
		p.Feed(b)
		if !p.Pending() {
			t.Fatalf("byte %q mid-CSI must leave parser pending", b)
		}
	}
	p.Feed('A')
	if p.Pending() {
		t.Fatal("completed CSI must clear pending state")
	}

	p = NewParser()
	p.Feed(0xC3) // lead byte of é
	if !p.Pending() {
		t.Fatal("partial UTF-8 rune must be pending")
	}
	p.Feed(0xA9)
	if p.Pending() {
		t.Fatal("completed rune must clear pending state")
	}

	p = NewParser()
	for _, b := range []byte("\x1b[200~abc") {
		p.Feed(b)
	}
	if !p.Pending() {
		t.Fatal("open bracketed paste must stay pending")
	}
}

func TestFlushResolvesLoneEscape(t *testing.T) {
	p := NewParser()
	p.Feed(0x1b)
	got := p.Flush()
	want := E(kp(KeyEscape))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	if p.Pending() {
		t.Fatal("Flush must clear pending state")
	}
	// Parser stays usable afterwards.
	if after := p.Feed('a'); !reflect.DeepEqual(after, E(kr('a'))) {
		t.Fatalf("parser unusable after Flush: %#v", after)
	}
}

func TestFlushDiscardsTruncatedSilently(t *testing.T) {
	cases := []struct{ name, in string }{
		// Note: a lone ESC is deliberately absent — Flush resolves it to
		// KeyEscape by design (see TestFlushResolvesLoneEscape).
		{"csi_no_final", "\x1b["},
		{"half_csi_params", "\x1b[12"},
		{"bare_ss3_prefix", "\x1bO"},
		{"mouse_header_only", "\x1b[M"},
		{"one_mouse_payload_byte", "\x1b[M "},
		{"two_mouse_payload_bytes", "\x1b[M !"},
		{"partial_utf8_rune", "\xe4\xb8"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := feedAll(NewParser(), tc.in)
			if len(got) != 0 {
				t.Fatalf("truncated %q must resolve silently, got %#v", tc.in, got)
			}
			p := NewParser()
			feedAll(p, tc.in)
			if p.Pending() {
				t.Fatal("Flush must clear all partial-sequence state")
			}
			if nx := p.Feed('a'); !reflect.DeepEqual(nx, E(kr('a'))) {
				t.Fatalf("parser unusable after truncation flush: %#v", nx)
			}
		})
	}
}

func TestPasteCapTruncatesToOneMB(t *testing.T) {
	content := strings.Repeat("x", maxPaste+11)
	in := "\x1b[200~" + content + "\x1b[201~after"
	got := feedAll(NewParser(), in)
	want := E(Paste{Text: content[:maxPaste]}, kr('a'), kr('f'), kr('t'), kr('e'), kr('r'))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("overflow paste: got %d events, want single truncated Paste + 5 runes", len(got))
	}
	pst, ok := got[0].(Paste)
	if !ok {
		t.Fatalf("expected Paste, got %T", got[0])
	}
	if pst.Text != strings.Repeat("x", maxPaste) {
		t.Fatalf("truncated length = %d, want %d", len(pst.Text), maxPaste)
	}
	// Stream resumes normally after the real terminator despite the cap.
	p := NewParser()
	feedAll(p, in[:len(in)-len("after")])
	if tail := p.Feed('q'); !reflect.DeepEqual(tail, E(kr('q'))) {
		t.Fatalf("post-paste input lost after overflow: %#v", tail)
	}
}

func TestPasteAtExactCapIsNotTruncated(t *testing.T) {
	content := strings.Repeat("y", maxPaste)
	in := "\x1b[200~" + content + "\x1b[201~"
	got := feedAll(NewParser(), in)
	want := E(Paste{Text: content})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exact-cap paste length = %d, want %d", len(got[0].(Paste).Text), maxPaste)
	}
}

// TestPumpForwardsMergesAndTimesOutLoneEsc covers Pump end-to-end,
// including the 40ms lone-ESC deadline (the only intentional sleep).
func TestPumpForwardsMergesAndTimesOutLoneEsc(t *testing.T) {
	pr, pw := io.Pipe()
	out := make(chan Event, 64)
	extra := make(chan Event, 2)
	errCh := make(chan error, 1)
	Pump(pr, extra, out, func(err error) { errCh <- err })

	pw.Write([]byte("a"))      //nolint:errcheck // pipe never fails here
	pw.Write([]byte("\x1b[B")) // arrow split into its own write
	expectEqual(t, recvEvents(t, out, 2), E(kr('a'), kp(KeyDown)))

	extra <- Focus{Gained: true}
	expectEqual(t, recvEvents(t, out, 1), E(Focus{Gained: true}))

	pw.Write([]byte("\x1b")) // lone ESC: resolved by the idle gap, not a sleep in the test
	expectEqual(t, recvEvents(t, out, 1), E(kp(KeyEscape)))

	close(extra) // detaching extra must not stop key pumping
	pw.Write([]byte("z"))
	expectEqual(t, recvEvents(t, out, 1), E(kr('z')))

	pw.Close() // reader sees io.EOF
	select {
	case err := <-errCh:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("onErr got %v, want io.EOF", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("onErr was not called after reader EOF")
	}
}

func TestPumpNilExtraAndSingleErrorCallback(t *testing.T) {
	pr, pw := io.Pipe()
	out := make(chan Event, 64)
	errCh := make(chan error, 2)
	Pump(pr, nil, out, func(err error) { errCh <- err }) // nil extra is legal

	pw.Write([]byte("\x1b[<0;1;1M")) //nolint:errcheck
	expectEqual(t, recvEvents(t, out, 1), E(mp(MousePress, 0, 1, 1, 0)))

	boom := errors.New("boom")
	pw.CloseWithError(boom)
	select {
	case err := <-errCh:
		if !errors.Is(err, boom) {
			t.Fatalf("onErr got %v, want %v", err, boom)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("onErr was not called after reader error")
	}
	select { // exactly one callback
	case err := <-errCh:
		t.Fatalf("onErr called twice: second=%v", err)
	case <-time.After(150 * time.Millisecond):
	}
}

func expectEqual(t *testing.T, got, want []Event) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func recvEvents(t *testing.T, out <-chan Event, n int) []Event {
	t.Helper()
	got := make([]Event, 0, n)
	deadline := time.After(3 * time.Second)
	for len(got) < n {
		select {
		case e := <-out:
			got = append(got, e)
		case <-deadline:
			t.Fatalf("timeout waiting for events: have %d/%d (%#v)", len(got), n, got)
		}
	}
	return got
}
