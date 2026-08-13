package editor

type Command struct {
	Name        string
	Description string
	Action      string
	Category    string
}

func DefaultCommands() []Command {
	return []Command{
		{Name: "Save File", Description: "Save the current file", Action: "save", Category: "File"},
		{Name: "Save As...", Description: "Save current file as...", Action: "save_as", Category: "File"},
		{Name: "Undo", Description: "Undo last edit", Action: "undo", Category: "Edit"},
		{Name: "Redo", Description: "Redo last edit", Action: "redo", Category: "Edit"},
		{Name: "New File", Description: "Open a new scratch buffer", Action: "new", Category: "File"},
		{Name: "Open File", Description: "Fuzzy find files", Action: "finder", Category: "File"},
		{Name: "Open Folder", Description: "Change workspace folder root", Action: "open_folder", Category: "File"},
		{Name: "Bundle Selected for AI", Description: "Copy selected files to clipboard as LLM context (Markdown)", Action: "bundle_ai", Category: "AI"},
		{Name: "Bundle as XML", Description: "Copy selected files to clipboard as XML", Action: "bundle_xml", Category: "AI"},
		{Name: "Bundle as Plain Text", Description: "Copy selected files to clipboard as plain text", Action: "bundle_text", Category: "AI"},
		{Name: "Toggle Sidebar", Description: "Toggle file explorer sidebar", Action: "sidebar", Category: "View"},
		{Name: "Toggle Split Pane", Description: "Split editor into two panes", Action: "split_pane", Category: "View"},
		{Name: "Cycle Theme", Description: "Switch theme (dark/light/custom)", Action: "cycle_theme", Category: "View"},
		{Name: "Search", Description: "Find in current file", Action: "search", Category: "Search"},
		{Name: "Global Search", Description: "Search across project", Action: "global_search", Category: "Search"},
		{Name: "Search & Replace", Description: "Find and replace", Action: "replace", Category: "Search"},
		{Name: "Toggle Search Case Sensitivity", Description: "Match case in current search", Action: "toggle_case", Category: "Search"},
		{Name: "Go to Line", Description: "Jump to a specific line", Action: "goto", Category: "Navigation"},
		{Name: "Select All", Description: "Select all text", Action: "select_all", Category: "Edit"},
		{Name: "Toggle Comment", Description: "Comment/uncomment selection", Action: "comment", Category: "Edit"},
		{Name: "Duplicate Line", Description: "Duplicate current line", Action: "dup_line", Category: "Edit"},
		{Name: "Delete Line", Description: "Delete current line", Action: "del_line", Category: "Edit"},
		{Name: "Move Line Up", Description: "Move line up", Action: "move_up", Category: "Edit"},
		{Name: "Move Line Down", Description: "Move line down", Action: "move_down", Category: "Edit"},
		{Name: "Indent", Description: "Indent line/selection", Action: "indent", Category: "Edit"},
		{Name: "Unindent", Description: "Unindent line/selection", Action: "unindent", Category: "Edit"},
		{Name: "Toggle Theme", Description: "Switch between light/dark theme", Action: "theme", Category: "View"},
		{Name: "Close Tab", Description: "Close current tab", Action: "close_tab", Category: "File"},
		{Name: "Next Tab", Description: "Switch to next tab", Action: "next_tab", Category: "Navigation"},
		{Name: "Previous Tab", Description: "Switch to previous tab", Action: "prev_tab", Category: "Navigation"},
		{Name: "Zoom In", Description: "Increase font size", Action: "zoom_in", Category: "View"},
		{Name: "Zoom Out", Description: "Decrease font size", Action: "zoom_out", Category: "View"},
		{Name: "Center Cursor", Description: "Center view on cursor", Action: "center", Category: "View"},
		{Name: "Reload File", Description: "Reload file from disk", Action: "reload", Category: "File"},
		{Name: "Show Recent Files", Description: "Show recently opened files", Action: "recent_files", Category: "File"},
		{Name: "Keyboard Shortcuts", Description: "View & edit keybindings", Action: "keybindings", Category: "Settings"},
		{Name: "Help", Description: "Show help overlay", Action: "help", Category: "Settings"},
		{Name: "Reset Keybindings", Description: "Reset all keybindings to defaults", Action: "reset_keybindings", Category: "Settings"},
		{Name: "Open Keybindings Config", Description: "Open keybindings.json in editor", Action: "open_keybindings_config", Category: "Settings"},
		{Name: "Open Config", Description: "Open config.json in editor", Action: "open_config", Category: "Settings"},
		{Name: "Settings UI", Description: "Open the settings browser", Action: "settings_ui", Category: "Settings"},
		{Name: "Reload Config", Description: "Reload config.json from disk", Action: "reload_config", Category: "Settings"},
		{Name: "Open Config Folder", Description: "Open .panda-editor folder in file tree", Action: "open_config_folder", Category: "Settings"},
		{Name: "Toggle Terminal", Description: "Open built-in system terminal", Action: "terminal", Category: "View"},
		{Name: "Close Session", Description: "Clear all tabs and return to welcome", Action: "close_session", Category: "System"},
		{Name: "Quit", Description: "Quit the editor", Action: "quit", Category: "System"},
	}
}
