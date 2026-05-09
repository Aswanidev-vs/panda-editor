package editor

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type KeyBindConfig struct {
	CursorUp        []string `json:"cursor_up"`
	CursorDown      []string `json:"cursor_down"`
	CursorLeft      []string `json:"cursor_left"`
	CursorRight     []string `json:"cursor_right"`
	WordLeft        []string `json:"word_left"`
	WordRight       []string `json:"word_right"`
	LineStart       []string `json:"line_start"`
	LineEnd         []string `json:"line_end"`
	FileStart       []string `json:"file_start"`
	FileEnd         []string `json:"file_end"`
	PageUp          []string `json:"page_up"`
	PageDown        []string `json:"page_down"`
	GoToLine        []string `json:"go_to_line"`
	MatchingBrace   []string `json:"matching_brace"`

	InsertNewline   []string `json:"insert_newline"`
	Backspace       []string `json:"backspace"`
	DeleteWordLeft  []string `json:"delete_word_left"`
	Delete          []string `json:"delete"`
	DeleteLine      []string `json:"delete_line"`
	DuplicateLine   []string `json:"duplicate_line"`
	MoveLineUp      []string `json:"move_line_up"`
	MoveLineDown    []string `json:"move_line_down"`
	IndentLine      []string `json:"indent_line"`
	UnindentLine    []string `json:"unindent_line"`
	ToggleComment   []string `json:"toggle_comment"`

	SelectUp        []string `json:"select_up"`
	SelectDown      []string `json:"select_down"`
	SelectLeft      []string `json:"select_left"`
	SelectRight     []string `json:"select_right"`
	SelectWordLeft  []string `json:"select_word_left"`
	SelectWordRight []string `json:"select_word_right"`
	SelectLineStart []string `json:"select_line_start"`
	SelectLineEnd   []string `json:"select_line_end"`
	SelectAll       []string `json:"select_all"`
	SelectLine      []string `json:"select_line"`

	Copy            []string `json:"copy"`
	Cut             []string `json:"cut"`
	Paste           []string `json:"paste"`

	Save            []string `json:"save"`
	SaveAs          []string `json:"save_as"`
	OpenFinder      []string `json:"open_finder"`
	OpenExplorer    []string `json:"open_explorer"`
	CommandPalette  []string `json:"command_palette"`
	Search          []string `json:"search"`
	SearchReplace   []string `json:"search_replace"`
	SearchNext      []string `json:"search_next"`
	SearchPrev      []string `json:"search_prev"`
	GlobalSearch    []string `json:"global_search"`

	Undo            []string `json:"undo"`
	Redo            []string `json:"redo"`
	CopyBundle      []string `json:"copy_bundle"`

	NextTab         []string `json:"next_tab"`
	PrevTab         []string `json:"prev_tab"`
	CloseTab        []string `json:"close_tab"`
	NewTab          []string `json:"new_tab"`

	SelectFile      []string `json:"select_file"`

	ToggleSidebar   []string `json:"toggle_sidebar"`
	ZoomIn          []string `json:"zoom_in"`
	ZoomOut         []string `json:"zoom_out"`
	ToggleTheme     []string `json:"toggle_theme"`
	ScrollUp        []string `json:"scroll_up"`
	ScrollDown      []string `json:"scroll_down"`
	CenterCursor    []string `json:"center_cursor"`

	Quit            []string `json:"quit"`
	ForceQuit       []string `json:"force_quit"`

	AddCursorUp     []string `json:"add_cursor_up"`
	AddCursorDown   []string `json:"add_cursor_down"`
	SelectNextMatch []string `json:"select_next_match"`

	ShowKeybindings []string `json:"show_keybindings"`
	ShowHelp        []string `json:"show_help"`

	ToggleTerminal  []string `json:"toggle_terminal"`
	TerminalCmd     string   `json:"terminal_cmd"`
}

type KeyBindEntry struct {
	Action      string
	Label       string
	Category    string
	Description string
	Keys        []string
	Field       *[]string
}

func DefaultKeyBindConfig() KeyBindConfig {
	return KeyBindConfig{
		CursorUp:        []string{"up"},
		CursorDown:      []string{"down"},
		CursorLeft:      []string{"left"},
		CursorRight:     []string{"right"},
		WordLeft:        []string{"ctrl+left", "alt+b"},
		WordRight:       []string{"ctrl+right", "alt+f"},
		LineStart:       []string{"home"},
		LineEnd:         []string{"end"},
		FileStart:       []string{"ctrl+home"},
		FileEnd:         []string{"ctrl+end"},
		PageUp:          []string{"pgup"},
		PageDown:        []string{"pgdown"},
		GoToLine:        []string{"ctrl+g"},
		MatchingBrace:   []string{"ctrl+]"},

		InsertNewline:   []string{"enter"},
		Backspace:       []string{"backspace"},
		DeleteWordLeft:  []string{"ctrl+backspace"},
		Delete:          []string{"delete"},
		DeleteLine:      []string{"ctrl+shift+k"},
		DuplicateLine:   []string{"ctrl+shift+d"},
		MoveLineUp:      []string{"alt+up"},
		MoveLineDown:    []string{"alt+down"},
		IndentLine:      []string{"tab"},
		UnindentLine:    []string{"shift+tab"},
		ToggleComment:   []string{"ctrl+/"},

		SelectUp:        []string{"shift+up"},
		SelectDown:      []string{"shift+down"},
		SelectLeft:      []string{"shift+left"},
		SelectRight:     []string{"shift+right"},
		SelectWordLeft:  []string{"ctrl+shift+left"},
		SelectWordRight: []string{"ctrl+shift+right"},
		SelectLineStart: []string{"shift+home"},
		SelectLineEnd:   []string{"shift+end"},
		SelectAll:       []string{"ctrl+a"},
			SelectLine:      []string{"ctrl+l"},

		Copy:            []string{"ctrl+c"},
		Cut:             []string{"ctrl+x"},
		Paste:           []string{"ctrl+v"},

		Save:            []string{"ctrl+s"},
		SaveAs:          []string{"ctrl+shift+s"},
		OpenFinder:      []string{"ctrl+p"},
		OpenExplorer:    []string{"ctrl+b"},
		CommandPalette:  []string{"ctrl+shift+p"},
		Search:          []string{"ctrl+f"},
		SearchReplace:   []string{"ctrl+h"},
		SearchNext:      []string{"f3"},
		SearchPrev:      []string{"shift+f3"},
		GlobalSearch:    []string{"alt+f"},

		Undo:            []string{"ctrl+z"},
		Redo:            []string{"ctrl+y", "ctrl+shift+z"},
		CopyBundle:      []string{"ctrl+shift+c"},

		NextTab:         []string{"ctrl+tab", "ctrl+pagedown"},
		PrevTab:         []string{"ctrl+shift+tab", "ctrl+pageup"},
		CloseTab:        []string{"ctrl+w"},
		NewTab:          []string{"ctrl+n"},

		SelectFile:      []string{"space"},

		ToggleSidebar:   []string{"ctrl+\\"},
		ZoomIn:          []string{"ctrl+="},
		ZoomOut:         []string{"ctrl+-"},
		ToggleTheme:     []string{"ctrl+t"},
		ScrollUp:        []string{"ctrl+up"},
		ScrollDown:      []string{"ctrl+down"},
		CenterCursor:    []string{"alt+c"},

		Quit:            []string{"ctrl+q"},
		ForceQuit:       []string{"ctrl+shift+q"},

		AddCursorUp:     []string{"ctrl+alt+up"},
		AddCursorDown:   []string{"ctrl+alt+down"},
		SelectNextMatch: []string{"ctrl+d"},

		ShowKeybindings: []string{"alt+k"},
		ShowHelp:        []string{"f1"},

		ToggleTerminal:  []string{"ctrl+j"},
		TerminalCmd:     "bash", // Default terminal, could be cmd or powershell on windows
	}
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".panda-editor")
}

func configFilePath() string {
	return filepath.Join(configDir(), "keybindings.json")
}

func LoadKeyBindConfig() KeyBindConfig {
	cfg := DefaultKeyBindConfig()

	data, err := os.ReadFile(configFilePath())
	if err != nil {
		return cfg
	}

	loaded := KeyBindConfig{}
	if err := json.Unmarshal(data, &loaded); err != nil {
		return cfg
	}

	if loaded.CursorUp != nil { cfg.CursorUp = loaded.CursorUp }
	if loaded.CursorDown != nil { cfg.CursorDown = loaded.CursorDown }
	if loaded.CursorLeft != nil { cfg.CursorLeft = loaded.CursorLeft }
	if loaded.CursorRight != nil { cfg.CursorRight = loaded.CursorRight }
	if loaded.WordLeft != nil { cfg.WordLeft = loaded.WordLeft }
	if loaded.WordRight != nil { cfg.WordRight = loaded.WordRight }
	if loaded.LineStart != nil { cfg.LineStart = loaded.LineStart }
	if loaded.LineEnd != nil { cfg.LineEnd = loaded.LineEnd }
	if loaded.FileStart != nil { cfg.FileStart = loaded.FileStart }
	if loaded.FileEnd != nil { cfg.FileEnd = loaded.FileEnd }
	if loaded.PageUp != nil { cfg.PageUp = loaded.PageUp }
	if loaded.PageDown != nil { cfg.PageDown = loaded.PageDown }
	if loaded.GoToLine != nil { cfg.GoToLine = loaded.GoToLine }
	if loaded.MatchingBrace != nil { cfg.MatchingBrace = loaded.MatchingBrace }
	if loaded.InsertNewline != nil { cfg.InsertNewline = loaded.InsertNewline }
	if loaded.Backspace != nil { cfg.Backspace = loaded.Backspace }
	if loaded.Delete != nil { cfg.Delete = loaded.Delete }
	if loaded.DeleteLine != nil { cfg.DeleteLine = loaded.DeleteLine }
	if loaded.DuplicateLine != nil { cfg.DuplicateLine = loaded.DuplicateLine }
	if loaded.MoveLineUp != nil { cfg.MoveLineUp = loaded.MoveLineUp }
	if loaded.MoveLineDown != nil { cfg.MoveLineDown = loaded.MoveLineDown }
	if loaded.IndentLine != nil { cfg.IndentLine = loaded.IndentLine }
	if loaded.UnindentLine != nil { cfg.UnindentLine = loaded.UnindentLine }
	if loaded.ToggleComment != nil { cfg.ToggleComment = loaded.ToggleComment }
	if loaded.SelectUp != nil { cfg.SelectUp = loaded.SelectUp }
	if loaded.SelectDown != nil { cfg.SelectDown = loaded.SelectDown }
	if loaded.SelectLeft != nil { cfg.SelectLeft = loaded.SelectLeft }
	if loaded.SelectRight != nil { cfg.SelectRight = loaded.SelectRight }
	if loaded.SelectWordLeft != nil { cfg.SelectWordLeft = loaded.SelectWordLeft }
	if loaded.SelectWordRight != nil { cfg.SelectWordRight = loaded.SelectWordRight }
	if loaded.SelectLineStart != nil { cfg.SelectLineStart = loaded.SelectLineStart }
	if loaded.SelectLineEnd != nil { cfg.SelectLineEnd = loaded.SelectLineEnd }
	if loaded.SelectAll != nil { cfg.SelectAll = loaded.SelectAll }
	if loaded.SelectLine != nil { cfg.SelectLine = loaded.SelectLine }
	if loaded.Copy != nil { cfg.Copy = loaded.Copy }
	if loaded.Cut != nil { cfg.Cut = loaded.Cut }
	if loaded.Paste != nil { cfg.Paste = loaded.Paste }
	if loaded.Save != nil { cfg.Save = loaded.Save }
	if loaded.SaveAs != nil { cfg.SaveAs = loaded.SaveAs }
	if loaded.OpenFinder != nil { cfg.OpenFinder = loaded.OpenFinder }
	if loaded.OpenExplorer != nil { cfg.OpenExplorer = loaded.OpenExplorer }
	if loaded.CommandPalette != nil { cfg.CommandPalette = loaded.CommandPalette }
	if loaded.Search != nil { cfg.Search = loaded.Search }
	if loaded.SearchReplace != nil { cfg.SearchReplace = loaded.SearchReplace }
	if loaded.SearchNext != nil { cfg.SearchNext = loaded.SearchNext }
	if loaded.SearchPrev != nil { cfg.SearchPrev = loaded.SearchPrev }
	if loaded.Undo != nil { cfg.Undo = loaded.Undo }
	if loaded.Redo != nil { cfg.Redo = loaded.Redo }
	if loaded.CopyBundle != nil { cfg.CopyBundle = loaded.CopyBundle }
	if loaded.NextTab != nil { cfg.NextTab = loaded.NextTab }
	if loaded.PrevTab != nil { cfg.PrevTab = loaded.PrevTab }
	if loaded.CloseTab != nil { cfg.CloseTab = loaded.CloseTab }
	if loaded.NewTab != nil { cfg.NewTab = loaded.NewTab }
	if loaded.SelectFile != nil { cfg.SelectFile = loaded.SelectFile }
	if loaded.ToggleSidebar != nil { cfg.ToggleSidebar = loaded.ToggleSidebar }
	if loaded.ZoomIn != nil { cfg.ZoomIn = loaded.ZoomIn }
	if loaded.ZoomOut != nil { cfg.ZoomOut = loaded.ZoomOut }
	if loaded.ToggleTheme != nil { cfg.ToggleTheme = loaded.ToggleTheme }
	if loaded.ScrollUp != nil { cfg.ScrollUp = loaded.ScrollUp }
	if loaded.ScrollDown != nil { cfg.ScrollDown = loaded.ScrollDown }
	if loaded.CenterCursor != nil { cfg.CenterCursor = loaded.CenterCursor }
	if loaded.Quit != nil { cfg.Quit = loaded.Quit }
	if loaded.ForceQuit != nil { cfg.ForceQuit = loaded.ForceQuit }
	if loaded.AddCursorUp != nil { cfg.AddCursorUp = loaded.AddCursorUp }
	if loaded.AddCursorDown != nil { cfg.AddCursorDown = loaded.AddCursorDown }
	if loaded.SelectNextMatch != nil { cfg.SelectNextMatch = loaded.SelectNextMatch }
	if loaded.ShowKeybindings != nil { cfg.ShowKeybindings = loaded.ShowKeybindings }
	if loaded.ShowHelp != nil { cfg.ShowHelp = loaded.ShowHelp }
	if loaded.ToggleTerminal != nil { cfg.ToggleTerminal = loaded.ToggleTerminal }
	if loaded.TerminalCmd != "" { cfg.TerminalCmd = loaded.TerminalCmd }

	return cfg
}

func SaveKeyBindConfig(cfg KeyBindConfig) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFilePath(), data, 0o644)
}

func ResetKeyBindConfig() error {
	return SaveKeyBindConfig(DefaultKeyBindConfig())
}

func JoinKeys(keys []string) string {
	if len(keys) == 0 {
		return "(none)"
	}
	result := keys[0]
	for i := 1; i < len(keys); i++ {
		result += ", " + keys[i]
	}
	return result
}

func (cfg *KeyBindConfig) GetKeyBindEntries() []KeyBindEntry {
	return []KeyBindEntry{
		{"cursor_up", "Cursor Up", "Navigation", "Move cursor up", cfg.CursorUp, &cfg.CursorUp},
		{"cursor_down", "Cursor Down", "Navigation", "Move cursor down", cfg.CursorDown, &cfg.CursorDown},
		{"cursor_left", "Cursor Left", "Navigation", "Move cursor left", cfg.CursorLeft, &cfg.CursorLeft},
		{"cursor_right", "Cursor Right", "Navigation", "Move cursor right", cfg.CursorRight, &cfg.CursorRight},
		{"word_left", "Word Left", "Navigation", "Move one word left", cfg.WordLeft, &cfg.WordLeft},
		{"word_right", "Word Right", "Navigation", "Move one word right", cfg.WordRight, &cfg.WordRight},
		{"line_start", "Line Start", "Navigation", "Go to line start", cfg.LineStart, &cfg.LineStart},
		{"line_end", "Line End", "Navigation", "Go to line end", cfg.LineEnd, &cfg.LineEnd},
		{"file_start", "File Start", "Navigation", "Go to file start", cfg.FileStart, &cfg.FileStart},
		{"file_end", "File End", "Navigation", "Go to file end", cfg.FileEnd, &cfg.FileEnd},
		{"page_up", "Page Up", "Navigation", "Scroll page up", cfg.PageUp, &cfg.PageUp},
		{"page_down", "Page Down", "Navigation", "Scroll page down", cfg.PageDown, &cfg.PageDown},
		{"go_to_line", "Go to Line", "Navigation", "Jump to line number", cfg.GoToLine, &cfg.GoToLine},
		{"matching_brace", "Matching Brace", "Navigation", "Jump to matching bracket", cfg.MatchingBrace, &cfg.MatchingBrace},

		{"insert_newline", "Insert Newline", "Editing", "Insert a new line", cfg.InsertNewline, &cfg.InsertNewline},
		{"backspace", "Backspace", "Editing", "Delete before cursor", cfg.Backspace, &cfg.Backspace},
		{"delete", "Delete", "Editing", "Delete at cursor", cfg.Delete, &cfg.Delete},
		{"delete_line", "Delete Line", "Editing", "Delete current line", cfg.DeleteLine, &cfg.DeleteLine},
		{"duplicate_line", "Duplicate Line", "Editing", "Duplicate current line", cfg.DuplicateLine, &cfg.DuplicateLine},
		{"move_line_up", "Move Line Up", "Editing", "Move line up", cfg.MoveLineUp, &cfg.MoveLineUp},
		{"move_line_down", "Move Line Down", "Editing", "Move line down", cfg.MoveLineDown, &cfg.MoveLineDown},
		{"indent_line", "Indent", "Editing", "Indent line/selection", cfg.IndentLine, &cfg.IndentLine},
		{"unindent_line", "Unindent", "Editing", "Unindent line/selection", cfg.UnindentLine, &cfg.UnindentLine},
		{"toggle_comment", "Toggle Comment", "Editing", "Comment/uncomment", cfg.ToggleComment, &cfg.ToggleComment},

		{"select_up", "Select Up", "Selection", "Extend selection up", cfg.SelectUp, &cfg.SelectUp},
		{"select_down", "Select Down", "Selection", "Extend selection down", cfg.SelectDown, &cfg.SelectDown},
		{"select_left", "Select Left", "Selection", "Extend selection left", cfg.SelectLeft, &cfg.SelectLeft},
		{"select_right", "Select Right", "Selection", "Extend selection right", cfg.SelectRight, &cfg.SelectRight},
		{"select_word_left", "Select Word Left", "Selection", "Extend by word left", cfg.SelectWordLeft, &cfg.SelectWordLeft},
		{"select_word_right", "Select Word Right", "Selection", "Extend by word right", cfg.SelectWordRight, &cfg.SelectWordRight},
		{"select_line_start", "Select to Start", "Selection", "Select to line start", cfg.SelectLineStart, &cfg.SelectLineStart},
		{"select_line_end", "Select to End", "Selection", "Select to line end", cfg.SelectLineEnd, &cfg.SelectLineEnd},
		{"select_all", "Select All", "Selection", "Select all text", cfg.SelectAll, &cfg.SelectAll},
		{"select_line", "Select Line", "Selection", "Select current line", cfg.SelectLine, &cfg.SelectLine},

		{"copy", "Copy", "Clipboard", "Copy selection/line", cfg.Copy, &cfg.Copy},
		{"cut", "Cut", "Clipboard", "Cut selection/line", cfg.Cut, &cfg.Cut},
		{"paste", "Paste", "Clipboard", "Paste from clipboard", cfg.Paste, &cfg.Paste},

		{"save", "Save", "Files", "Save current file", cfg.Save, &cfg.Save},
		{"save_as", "Save As", "Files", "Save as new file", cfg.SaveAs, &cfg.SaveAs},
		{"open_finder", "Quick Open", "Files", "Fuzzy file finder", cfg.OpenFinder, &cfg.OpenFinder},
		{"open_explorer", "Toggle Explorer", "Files", "Toggle sidebar", cfg.OpenExplorer, &cfg.OpenExplorer},
		{"command_palette", "Command Palette", "Files", "Open commands", cfg.CommandPalette, &cfg.CommandPalette},
		{"search", "Search", "Files", "Find in file", cfg.Search, &cfg.Search},
		{"search_replace", "Search & Replace", "Files", "Find and replace", cfg.SearchReplace, &cfg.SearchReplace},
		{"search_next", "Next Match", "Files", "Jump to next match", cfg.SearchNext, &cfg.SearchNext},
		{"search_prev", "Prev Match", "Files", "Jump to prev match", cfg.SearchPrev, &cfg.SearchPrev},
		{"global_search", "Global Search", "Files", "Search across all files", cfg.GlobalSearch, &cfg.GlobalSearch},

		{"undo", "Undo", "Editing", "Undo last action", cfg.Undo, &cfg.Undo},
		{"redo", "Redo", "Editing", "Redo last action", cfg.Redo, &cfg.Redo},

		{"next_tab", "Next Tab", "Tabs", "Switch to next tab", cfg.NextTab, &cfg.NextTab},
		{"prev_tab", "Previous Tab", "Tabs", "Switch to prev tab", cfg.PrevTab, &cfg.PrevTab},
		{"close_tab", "Close Tab", "Tabs", "Close current tab", cfg.CloseTab, &cfg.CloseTab},
		{"new_tab", "New Tab", "Tabs", "Open new tab", cfg.NewTab, &cfg.NewTab},

		{"select_file", "Select File", "Explorer", "Select file for AI bundle", cfg.SelectFile, &cfg.SelectFile},

		{"toggle_sidebar", "Toggle Sidebar", "View", "Toggle file tree", cfg.ToggleSidebar, &cfg.ToggleSidebar},
		{"zoom_in", "Zoom In", "View", "Increase zoom", cfg.ZoomIn, &cfg.ZoomIn},
		{"zoom_out", "Zoom Out", "View", "Decrease zoom", cfg.ZoomOut, &cfg.ZoomOut},
		{"toggle_theme", "Toggle Theme", "View", "Switch dark/light", cfg.ToggleTheme, &cfg.ToggleTheme},
		{"scroll_up", "Scroll Up", "View", "Scroll view up", cfg.ScrollUp, &cfg.ScrollUp},
		{"scroll_down", "Scroll Down", "View", "Scroll view down", cfg.ScrollDown, &cfg.ScrollDown},
		{"center_cursor", "Center Cursor", "View", "Center on cursor", cfg.CenterCursor, &cfg.CenterCursor},

		{"quit", "Quit", "General", "Quit editor", cfg.Quit, &cfg.Quit},
		{"force_quit", "Force Quit", "General", "Force quit", cfg.ForceQuit, &cfg.ForceQuit},
		{"show_keybindings", "Keybinding Editor", "General", "Edit keybindings", cfg.ShowKeybindings, &cfg.ShowKeybindings},
		{"show_help", "Help", "General", "Show help overlay", cfg.ShowHelp, &cfg.ShowHelp},
		{"copy_bundle", "Copy AI Bundle", "AI", "Copy selected files as context", cfg.CopyBundle, &cfg.CopyBundle},
		{"toggle_terminal", "Toggle Terminal", "General", "Open external terminal", cfg.ToggleTerminal, &cfg.ToggleTerminal},
	}
}

