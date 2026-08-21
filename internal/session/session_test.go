package session

import (
	"path/filepath"
	"reflect"
	"testing"
)

func overrideHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

func TestSaveLoadRoundTripAtomic(t *testing.T) {
	dir := overrideHome(t)

	want := SessionState{
		ActiveTab: 2,
		Tabs: []TabState{
			{FilePath: "/tmp/a.go", CursorLine: 3, CursorCol: 5, ScrollLine: 1},
			{FilePath: "", IsScratch: true, Content: "scratch text"},
		},
		SidebarOpen:  true,
		ThemeName:    "Panda Dark",
		ZoomLevel:    120,
		LastDir:      dir,
		WindowWidth:  1024,
		WindowHeight: 768,
	}

	if err := SaveSession(want); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	if !SessionExists() {
		t.Fatal("session file should exist after Save")
	}

	got, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}
	if got.SavedAt == "" {
		t.Error("SavedAt should be stamped by Save")
	}
	if got.ActiveTab != want.ActiveTab || !reflect.DeepEqual(got.Tabs, want.Tabs) {
		t.Errorf("tabs round-trip mismatch:\n got %+v\nwant %+v", got.Tabs, want.Tabs)
	}
	if !got.SidebarOpen || got.ThemeName != want.ThemeName || got.ZoomLevel != want.ZoomLevel ||
		got.WindowWidth != want.WindowWidth || got.WindowHeight != want.WindowHeight {
		t.Errorf("state round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}

	// Regression: the atomic temp+rename save must not leave *.tmp droppings
	// behind in the session directory.
	matches, err := filepath.Glob(filepath.Join(dir, ".panda-editor", "*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("temp files left behind after Save: %v", matches)
	}
}

func TestSaveOverwritesExistingSession(t *testing.T) {
	overrideHome(t)

	first := SessionState{ActiveTab: 0, Tabs: []TabState{{FilePath: "a.txt"}}}
	if err := SaveSession(first); err != nil {
		t.Fatalf("first SaveSession: %v", err)
	}
	second := SessionState{ActiveTab: 1, Tabs: []TabState{{FilePath: "b.txt"}, {FilePath: "c.txt"}}}
	if err := SaveSession(second); err != nil {
		t.Fatalf("second SaveSession (rename over existing): %v", err)
	}

	got, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if got.ActiveTab != 1 || len(got.Tabs) != 2 || got.Tabs[1].FilePath != "c.txt" {
		t.Errorf("stale content survived overwrite: %+v", got)
	}
}

func TestDeleteSession(t *testing.T) {
	overrideHome(t)
	if err := SaveSession(SessionState{Tabs: []TabState{{FilePath: "a.txt"}}}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	if err := DeleteSession(); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if SessionExists() {
		t.Error("session should not exist after delete")
	}
}
