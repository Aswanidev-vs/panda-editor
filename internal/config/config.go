package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PerLanguage holds per-language LSP and formatting options.
type PerLanguage struct {
	LSP         string `json:"lsp,omitempty"`
	TabSize     int    `json:"tab_size,omitempty"`
	FormatOnSave bool  `json:"format_on_save,omitempty"`
}

// EditorSettings holds editor-wide options.
type EditorSettings struct {
	TabSize            int    `json:"tab_size,omitempty"`
	RelativeLineNumbers bool  `json:"relative_line_numbers,omitempty"`
	AutoSaveInterval   int    `json:"auto_save_interval,omitempty"`
	Minimap            *bool  `json:"minimap,omitempty"`
	WordWrap           bool   `json:"word_wrap,omitempty"`
}

// Behavior holds behavioral options.
type Behavior struct {
	TerminalCmd   string `json:"terminal_cmd,omitempty"`
	LSPEnabled    *bool  `json:"lsp_enabled,omitempty"`
	ClipboardSync *bool  `json:"clipboard_sync,omitempty"`
	SessionSave   *bool  `json:"session_save,omitempty"`
}

// Config is the unified user configuration.
// It replaces the old settings.json + keybindings.json model.
type Config struct {
	Editor   EditorSettings          `json:"editor,omitempty"`
	Theme    string                  `json:"theme,omitempty"`
	Colors   map[string]string       `json:"custom_colors,omitempty"`
	Lang     map[string]PerLanguage  `json:"languages,omitempty"`
	Keymap   map[string][]string     `json:"keybindings,omitempty"`
	Behavior Behavior                `json:"behavior,omitempty"`
}

// DeprecatedSettings maps old settings.json keys to the new struct.
type deprecatedSettings struct {
	Theme          string `json:"theme"`
	LSPEnabled     bool   `json:"lsp_enabled"`
	FontSize       int    `json:"font_size"`
	SidebarVisible bool   `json:"sidebar_visible"`
	RelativeLineNo bool   `json:"relative_line_no"`
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".panda-editor")
}

func configFilePath() string {
	return filepath.Join(configDir(), "config.json")
}

func oldSettingsPath() string {
	return filepath.Join(configDir(), "settings.json")
}

func oldKeybindingsPath() string {
	return filepath.Join(configDir(), "keybindings.json")
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	minimap := true
	lsp := true
	clip := true
	session := true
	return Config{
		Editor: EditorSettings{
			TabSize:             4,
			RelativeLineNumbers: false,
			AutoSaveInterval:    0,
			Minimap:             &minimap,
			WordWrap:            false,
		},
		Theme:  "Panda Dark",
		Colors: nil,
		Lang: map[string]PerLanguage{
			"go":       {LSP: "gopls", TabSize: 4, FormatOnSave: true},
			"python":   {LSP: "pylsp", TabSize: 4, FormatOnSave: false},
			"typescript": {LSP: "typescript-language-server", TabSize: 2, FormatOnSave: true},
			"javascript": {LSP: "typescript-language-server", TabSize: 2, FormatOnSave: true},
			"rust":     {LSP: "rust-analyzer", TabSize: 4, FormatOnSave: true},
		},
		Keymap: nil,
		Behavior: Behavior{
			TerminalCmd:   "cmd",
			LSPEnabled:    &lsp,
			ClipboardSync: &clip,
			SessionSave:   &session,
		},
	}
}

// DefaultConfigJSON returns the auto-generated default config.json as a
// single valid JSON document. The explanatory note is embedded as a
// "_comment" member, which json.Unmarshal ignores when decoding into
// Config, so the output round-trips back to DefaultConfig.
func DefaultConfigJSON() string {
	note := "// Panda Editor Configuration\n" +
		"// Edit this file to customize Panda Editor.\n" +
		"// Changes take effect on save — use 'Reload Config' in the command palette.\n" +
		"//\n" +
		"// Color names for custom_colors:\n" +
		"//   bg, fg, accent, accent_alt, accent_dim\n" +
		"//   comment, keyword, string, number, function, type, operator, builtin\n" +
		"//   error, warning, success\n" +
		"//   cursor, selection, cursor_line\n" +
		"//   line_num, line_num_active\n" +
		"//   status_bar, status_accent\n" +
		"//   sidebar, sidebar_bg\n" +
		"//   tab_bg, tab_active_bg, tab_fg, tab_active_fg\n" +
		"//   border, title_bar, scrollbar\n" +
		"//   gutter_bg, breadcrumb_fg, overlay_bg\n"
	doc := struct {
		Comment string `json:"_comment"`
		Config
	}{Comment: note, Config: DefaultConfig()}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadConfig loads the unified config, with fallback to old settings.json + keybindings.json.
func LoadConfig() (Config, error) {
	cfg := DefaultConfig()

	// Try new unified config.json first
	data, err := os.ReadFile(configFilePath())
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("config.json: %w", err)
		}
		return cfg, nil
	}

	// Fallback: try old settings.json
	oldData, oldErr := os.ReadFile(oldSettingsPath())
	if oldErr == nil {
		var old deprecatedSettings
		if err := json.Unmarshal(oldData, &old); err == nil {
			// Map old fields to new struct
			if old.Theme != "" {
				cfg.Theme = old.Theme
			}
			if old.LSPEnabled {
				cfg.Behavior.LSPEnabled = &old.LSPEnabled
			}
		}
	}

	// Fallback: try old keybindings.json
	kbData, kbErr := os.ReadFile(oldKeybindingsPath())
	if kbErr == nil {
		var kbMap map[string][]string
		if err := json.Unmarshal(kbData, &kbMap); err == nil {
			if cfg.Keymap == nil {
				cfg.Keymap = make(map[string][]string)
			}
			for k, v := range kbMap {
				cfg.Keymap[k] = v
			}
		}
	}

	return cfg, nil
}

// SaveConfig writes the unified config file.
func SaveConfig(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFilePath(), data, 0644)
}

// MigrateFromOld migrates old settings.json + keybindings.json to config.json.
// Returns true if a new config was written.
func MigrateFromOld() bool {
	// Check if config.json already exists
	if _, err := os.Stat(configFilePath()); err == nil {
		return false
	}

	// Check if old files exist
	oldExists := false
	if _, err := os.Stat(oldSettingsPath()); err == nil {
		oldExists = true
	}
	if _, err := os.Stat(oldKeybindingsPath()); err == nil {
		oldExists = true
	}
	if !oldExists {
		return false
	}

	cfg, _ := LoadConfig()
	if err := SaveConfig(cfg); err != nil {
		return false
	}

	// Rename old files to .bak
	if _, err := os.Stat(oldSettingsPath()); err == nil {
		os.Rename(oldSettingsPath(), oldSettingsPath()+".bak")
	}
	if _, err := os.Stat(oldKeybindingsPath()); err == nil {
		os.Rename(oldKeybindingsPath(), oldKeybindingsPath()+".bak")
	}
	return true
}

// BoolVal returns the bool value of a *bool, defaulting to the given fallback.
func BoolVal(b *bool, fallback bool) bool {
	if b == nil {
		return fallback
	}
	return *b
}

// GetLanguage returns per-language config, or a zero value if not set.
func (c Config) GetLanguage(lang string) PerLanguage {
	if c.Lang == nil {
		return PerLanguage{}
	}
	return c.Lang[strings.ToLower(lang)]
}
