package editor

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	CursorUp        key.Binding
	CursorDown      key.Binding
	CursorLeft      key.Binding
	CursorRight     key.Binding
	WordLeft        key.Binding
	WordRight       key.Binding
	LineStart       key.Binding
	LineEnd         key.Binding
	FileStart       key.Binding
	FileEnd         key.Binding
	PageUp          key.Binding
	PageDown        key.Binding
	GoToLine        key.Binding
	MatchingBrace   key.Binding

	InsertNewline   key.Binding
	Backspace       key.Binding
	DeleteWordLeft  key.Binding
	Delete          key.Binding
	DeleteLine      key.Binding
	DuplicateLine   key.Binding
	MoveLineUp      key.Binding
	MoveLineDown    key.Binding
	IndentLine      key.Binding
	UnindentLine    key.Binding
	ToggleComment   key.Binding

	SelectUp        key.Binding
	SelectDown      key.Binding
	SelectLeft      key.Binding
	SelectRight     key.Binding
	SelectWordLeft  key.Binding
	SelectWordRight key.Binding
	SelectLineStart key.Binding
	SelectLineEnd   key.Binding
	SelectAll       key.Binding
	SelectLine      key.Binding

	Copy            key.Binding
	Cut             key.Binding
	Paste           key.Binding

	Save            key.Binding
	SaveAs          key.Binding
	OpenFinder      key.Binding
	OpenExplorer    key.Binding
	CommandPalette  key.Binding
	Search          key.Binding
	SearchReplace   key.Binding
	SearchNext      key.Binding
	SearchPrev      key.Binding
	GlobalSearch    key.Binding

	Undo            key.Binding
	Redo            key.Binding

	NextTab         key.Binding
	PrevTab         key.Binding
	CloseTab        key.Binding
	NewTab          key.Binding

	SelectFile      key.Binding

	ToggleSidebar   key.Binding
	ZoomIn          key.Binding
	ZoomOut         key.Binding
	ToggleTheme     key.Binding
	ScrollUp        key.Binding
	ScrollDown      key.Binding
	CenterCursor    key.Binding

	Quit            key.Binding
	ForceQuit       key.Binding

	AddCursorUp     key.Binding
	AddCursorDown   key.Binding
	SelectNextMatch key.Binding

	ShowKeybindings key.Binding
	ShowHelp        key.Binding
	CopyBundle      key.Binding
	ToggleTerminal  key.Binding
}

func keysBinding(keys []string) key.Binding {
	if len(keys) == 0 {
		return key.NewBinding(key.WithKeys("___none___"))
	}
	return key.NewBinding(key.WithKeys(keys...))
}

func DefaultKeyMap() KeyMap {
	return KeyMapFromConfig(DefaultKeyBindConfig())
}

func KeyMapFromConfig(cfg KeyBindConfig) KeyMap {
	return KeyMap{
		CursorUp:        keysBinding(cfg.CursorUp),
		CursorDown:      keysBinding(cfg.CursorDown),
		CursorLeft:      keysBinding(cfg.CursorLeft),
		CursorRight:     keysBinding(cfg.CursorRight),
		WordLeft:        keysBinding(cfg.WordLeft),
		WordRight:       keysBinding(cfg.WordRight),
		LineStart:       keysBinding(cfg.LineStart),
		LineEnd:         keysBinding(cfg.LineEnd),
		FileStart:       keysBinding(cfg.FileStart),
		FileEnd:         keysBinding(cfg.FileEnd),
		PageUp:          keysBinding(cfg.PageUp),
		PageDown:        keysBinding(cfg.PageDown),
		GoToLine:        keysBinding(cfg.GoToLine),
		MatchingBrace:   keysBinding(cfg.MatchingBrace),

		InsertNewline:   keysBinding(cfg.InsertNewline),
		Backspace:       keysBinding(cfg.Backspace),
		DeleteWordLeft:  keysBinding(cfg.DeleteWordLeft),
		Delete:          keysBinding(cfg.Delete),
		DeleteLine:      keysBinding(cfg.DeleteLine),
		DuplicateLine:   keysBinding(cfg.DuplicateLine),
		MoveLineUp:      keysBinding(cfg.MoveLineUp),
		MoveLineDown:    keysBinding(cfg.MoveLineDown),
		IndentLine:      keysBinding(cfg.IndentLine),
		UnindentLine:    keysBinding(cfg.UnindentLine),
		ToggleComment:   keysBinding(cfg.ToggleComment),

		SelectUp:        keysBinding(cfg.SelectUp),
		SelectDown:      keysBinding(cfg.SelectDown),
		SelectLeft:      keysBinding(cfg.SelectLeft),
		SelectRight:     keysBinding(cfg.SelectRight),
		SelectWordLeft:  keysBinding(cfg.SelectWordLeft),
		SelectWordRight: keysBinding(cfg.SelectWordRight),
		SelectLineStart: keysBinding(cfg.SelectLineStart),
		SelectLineEnd:   keysBinding(cfg.SelectLineEnd),
		SelectAll:       keysBinding(cfg.SelectAll),
		SelectLine:      keysBinding(cfg.SelectLine),

		Copy:            keysBinding(cfg.Copy),
		Cut:             keysBinding(cfg.Cut),
		Paste:           keysBinding(cfg.Paste),

		Save:            keysBinding(cfg.Save),
		SaveAs:          keysBinding(cfg.SaveAs),
		OpenFinder:      keysBinding(cfg.OpenFinder),
		OpenExplorer:    keysBinding(cfg.OpenExplorer),
		CommandPalette:  keysBinding(cfg.CommandPalette),
		Search:          keysBinding(cfg.Search),
		SearchReplace:   keysBinding(cfg.SearchReplace),
		SearchNext:      keysBinding(cfg.SearchNext),
		SearchPrev:      keysBinding(cfg.SearchPrev),
		GlobalSearch:    keysBinding(cfg.GlobalSearch),

		Undo:            keysBinding(cfg.Undo),
		Redo:            keysBinding(cfg.Redo),

		NextTab:         keysBinding(cfg.NextTab),
		PrevTab:         keysBinding(cfg.PrevTab),
		CloseTab:        keysBinding(cfg.CloseTab),
		NewTab:          keysBinding(cfg.NewTab),

		SelectFile:      keysBinding(cfg.SelectFile),

		ToggleSidebar:   keysBinding(cfg.ToggleSidebar),
		ZoomIn:          keysBinding(cfg.ZoomIn),
		ZoomOut:         keysBinding(cfg.ZoomOut),
		ToggleTheme:     keysBinding(cfg.ToggleTheme),
		ScrollUp:        keysBinding(cfg.ScrollUp),
		ScrollDown:      keysBinding(cfg.ScrollDown),
		CenterCursor:    keysBinding(cfg.CenterCursor),

		Quit:            keysBinding(cfg.Quit),
		ForceQuit:       keysBinding(cfg.ForceQuit),

		AddCursorUp:     keysBinding(cfg.AddCursorUp),
		AddCursorDown:   keysBinding(cfg.AddCursorDown),
		SelectNextMatch: keysBinding(cfg.SelectNextMatch),

		ShowKeybindings: keysBinding(cfg.ShowKeybindings),
		ShowHelp:        keysBinding(cfg.ShowHelp),
		CopyBundle:      keysBinding(cfg.CopyBundle),
		ToggleTerminal:  keysBinding(cfg.ToggleTerminal),
	}
}
