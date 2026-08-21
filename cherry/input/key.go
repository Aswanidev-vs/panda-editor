package input

// Key identifies a non-printable key. Printable text input arrives as
// KeyPress with Key == KeyNone and Rune set.
type Key uint8

const (
	KeyNone Key = iota // rune-carrying press (see KeyPress.Rune)
	KeyEscape
	KeyEnter
	KeyTab
	KeyBacktab
	KeyBackspace
	KeyDelete
	KeyInsert
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyF13
	KeyF14
	KeyF15
	KeyF16
	KeyF17
	KeyF18
	KeyF19
	KeyF20
	KeyMenu
	KeyCapsLock
	KeyScrollLock
	KeyNumLock
	KeyPrintScreen
	KeyPause
)

var keyNames = map[Key]string{
	KeyNone: "", KeyEscape: "esc", KeyEnter: "enter", KeyTab: "tab",
	KeyBacktab: "shift+tab", KeyBackspace: "backspace", KeyDelete: "delete",
	KeyInsert: "insert", KeyUp: "up", KeyDown: "down", KeyLeft: "left",
	KeyRight: "right", KeyHome: "home", KeyEnd: "end", KeyPageUp: "pgup",
	KeyPageDown: "pgdown", KeyF1: "f1", KeyF2: "f2", KeyF3: "f3", KeyF4: "f4",
	KeyF5: "f5", KeyF6: "f6", KeyF7: "f7", KeyF8: "f8", KeyF9: "f9",
	KeyF10: "f10", KeyF11: "f11", KeyF12: "f12", KeyF13: "f13", KeyF14: "f14",
	KeyF15: "f15", KeyF16: "f16", KeyF17: "f17", KeyF18: "f18", KeyF19: "f19",
	KeyF20: "f20", KeyMenu: "menu", KeyCapsLock: "capslock",
	KeyScrollLock: "scrolllock", KeyNumLock: "numlock",
	KeyPrintScreen: "printscreen", KeyPause: "pause",
}

// String returns the canonical lowercase name used in keymap specs.
func (k Key) String() string { return keyNames[k] }

// KeyFromName resolves a canonical name back to a Key; ok is false if unknown.
func KeyFromName(name string) (Key, bool) {
	for k, n := range keyNames {
		if n == name && k != KeyNone {
			return k, true
		}
	}
	return KeyNone, false
}

// Mod is a bitmask of held modifiers.
type Mod uint8

const (
	ModShift Mod = 1 << iota
	ModCtrl
	ModAlt
	ModMeta
)

func (m Mod) Has(o Mod) bool  { return m&o != 0 }
func (m Mod) String() string {
	s := ""
	if m.Has(ModCtrl) { s += "ctrl+" }
	if m.Has(ModAlt) { s += "alt+" }
	if m.Has(ModShift) { s += "shift+" }
	if m.Has(ModMeta) { s += "meta+" }
	return s
}
