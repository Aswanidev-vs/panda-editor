package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestWatchIsIdempotent(t *testing.T) {
	w, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	dir := t.TempDir()
	file := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := w.Watch(file); err != nil {
		t.Fatalf("first Watch: %v", err)
	}
	if err := w.Watch(file); err != nil {
		t.Fatalf("second Watch of same file: %v", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.watched) != 1 {
		t.Errorf("watched = %v, want 1 entry", w.watched)
	}
	if refs := w.dirRefs[filepath.Dir(mustAbs(t, file))]; refs != 1 {
		t.Errorf("dirRefs = %d, want 1 (double-watch must not stack refs)", refs)
	}
}

func TestUnwatchRemovesUnderlyingWatch(t *testing.T) {
	events := make(chan string, 16)
	w, err := New(func(p string) { events <- p })
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	dir := t.TempDir()
	file := filepath.Join(dir, "watched.txt")
	if err := os.WriteFile(file, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := w.Watch(file); err != nil {
		t.Fatal(err)
	}

	// Sanity check: while watched, an external write triggers the callback.
	if err := os.WriteFile(file, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-events:
	case <-time.After(3 * time.Second):
		t.Fatal("callback never fired while the file was watched")
	}

	w.Unwatch(file)

	w.mu.Lock()
	refs := len(w.dirRefs)
	_, stillWatched := w.watched[mustAbs(t, file)]
	w.mu.Unlock()
	if stillWatched {
		t.Error("file should be removed from watched map")
	}
	if refs != 0 {
		t.Errorf("dirRefs has %d entries after last unwatch; underlying fsnotify handle would leak", refs)
	}

	// Let any stragglers from the earlier write land (and pass the debounce
	// window) before asserting silence.
	time.Sleep(700 * time.Millisecond)
drain:
	for {
		select {
		case <-events:
		default:
			break drain
		}
	}

	// The underlying directory watch is gone now: writes must not reach the
	// callback any more. Before the fix, Unwatch left the fsnotify handle in
	// place and this write was still delivered.
	if err := os.WriteFile(file, []byte("v3"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case p := <-events:
		t.Fatalf("callback fired for %q after Unwatch", p)
	case <-time.After(1500 * time.Millisecond):
	}
}

func TestUnwatchKeepsDirWhileOtherFilesWatched(t *testing.T) {
	w, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	for _, f := range []string{a, b} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := w.Watch(a); err != nil {
		t.Fatal(err)
	}
	if err := w.Watch(b); err != nil {
		t.Fatal(err)
	}
	w.Unwatch(a)

	w.mu.Lock()
	if refs := w.dirRefs[filepath.Dir(mustAbs(t, a))]; refs != 1 {
		t.Errorf("dirRefs = %d, want 1 while b.txt is still watched", refs)
	}
	if _, ok := w.watched[mustAbs(t, b)]; !ok {
		t.Error("b.txt should remain watched")
	}
	w.mu.Unlock()

	// Double unwatch of the same path is a safe no-op.
	w.Unwatch(a)
	w.mu.Lock()
	if n := len(w.dirRefs); n != 1 {
		t.Errorf("dirRefs = %d after double unwatch, want 1", n)
	}
	w.mu.Unlock()
}

func TestCloseIsSafeTwiceAndStopsWatching(t *testing.T) {
	w, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.Watch(file); err != nil {
		t.Fatal(err)
	}

	w.Close()
	w.Close() // must not panic

	if err := w.Watch(file); err == nil {
		t.Error("Watch after Close should fail")
	}
}
