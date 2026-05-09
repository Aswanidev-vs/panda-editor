package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Theme          string `json:"theme"`
	LSPEnabled     bool   `json:"lsp_enabled"`
	FontSize       int    `json:"font_size"`
	SidebarVisible bool   `json:"sidebar_visible"`
	RelativeLineNo bool   `json:"relative_line_no"`
}

func DefaultConfig() Config {
	return Config{
		Theme:          "Panda Dark",
		LSPEnabled:     true,
		FontSize:       14,
		SidebarVisible: true,
		RelativeLineNo: false,
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
	return filepath.Join(configDir(), "settings.json")
}

func LoadConfig() (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(configFilePath())
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

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
