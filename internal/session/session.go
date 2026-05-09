package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type TabState struct {
	FilePath   string `json:"file_path"`
	CursorLine int    `json:"cursor_line"`
	CursorCol  int    `json:"cursor_col"`
	ScrollLine int    `json:"scroll_line"`
	ScrollCol  int    `json:"scroll_col"`
	IsScratch  bool   `json:"is_scratch"`
	Content    string `json:"content,omitempty"`
}

type SessionState struct {
	Version       int         `json:"version"`
	ActiveTab     int         `json:"active_tab"`
	Tabs          []TabState  `json:"tabs"`
	SidebarOpen   bool        `json:"sidebar_open"`
	ThemeName     string      `json:"theme_name"`
	ZoomLevel     int         `json:"zoom_level"`
	LastDir       string      `json:"last_dir"`
	SavedAt       string      `json:"saved_at"`
	WindowWidth   int         `json:"window_width"`
	WindowHeight  int         `json:"window_height"`
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".panda-editor")
}

func sessionFilePath() string {
	return filepath.Join(configDir(), "session.json")
}

func hasSessionFile() bool {
	_, err := os.Stat(sessionFilePath())
	return err == nil
}

func SaveSession(state SessionState) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	state.Version = 1
	state.SavedAt = time.Now().Format(time.RFC3339)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sessionFilePath(), data, 0o644)
}

func LoadSession() (SessionState, error) {
	state := SessionState{
		Version:   1,
		ActiveTab: 0,
		SidebarOpen: true,
		ThemeName: "Panda Dark",
		Tabs:      []TabState{},
	}

	data, err := os.ReadFile(sessionFilePath())
	if err != nil {
		return state, err
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}

	return state, nil
}

func DeleteSession() error {
	return os.Remove(sessionFilePath())
}

func SessionExists() bool {
	if !hasSessionFile() {
		return false
	}
	state, err := LoadSession()
	if err != nil {
		return false
	}
	return state.Version > 0 && len(state.Tabs) > 0
}
