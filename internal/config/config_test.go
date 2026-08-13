package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBoolVal(t *testing.T) {
	if !BoolVal(nil, true) {
		t.Error("BoolVal(nil, true) should return true")
	}
	if BoolVal(nil, false) {
		t.Error("BoolVal(nil, false) should return false")
	}
	tr := true
	fa := false
	if !BoolVal(&tr, false) {
		t.Error("BoolVal(&true, false) should return true")
	}
	if BoolVal(&fa, true) {
		t.Error("BoolVal(&false, true) should return false")
	}
}

func TestDefaultConfigValid(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Editor.TabSize == 0 {
		t.Error("Default TabSize should be > 0")
	}
	if cfg.Theme == "" {
		t.Error("Default Theme should be set")
	}
	if cfg.Lang == nil {
		t.Error("Default Lang should be initialized")
	}
	if _, ok := cfg.Lang["go"]; !ok {
		t.Error("Default config should include go language entry")
	}
}

func TestGetLanguage(t *testing.T) {
	cfg := DefaultConfig()
	lang := cfg.GetLanguage("go")
	if lang.LSP != "gopls" {
		t.Errorf("GetLanguage(go).LSP = %q, want %q", lang.LSP, "gopls")
	}
	lang = cfg.GetLanguage("nonexistent")
	if lang.LSP != "" {
		t.Errorf("GetLanguage(nonexistent).LSP = %q, want empty", lang.LSP)
	}
}

func TestSaveLoadConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	cfg := DefaultConfig()
	cfg.Theme = "Custom Theme"
	cfg.Editor.TabSize = 8

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loaded.Theme != "Custom Theme" {
		t.Errorf("loaded Theme = %q, want %q", loaded.Theme, "Custom Theme")
	}
	if loaded.Editor.TabSize != 8 {
		t.Errorf("loaded TabSize = %d, want 8", loaded.Editor.TabSize)
	}
}

func TestLoadConfigFallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig with no file should not error: %v", err)
	}
	if cfg.Theme == "" {
		t.Error("fallback config should still have a Theme")
	}
}

func TestConfigFilePath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	want := filepath.Join(dir, ".panda-editor", "config.json")
	got := configFilePath()
	if got != want {
		t.Errorf("configFilePath = %q, want %q", got, want)
	}
}

func TestEnsureConfigDirCreated(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	if err := os.MkdirAll(configDir(), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if _, err := os.Stat(configDir()); err != nil {
		t.Errorf("expected config dir to exist: %v", err)
	}
}