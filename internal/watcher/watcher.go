package watcher

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileChangeMsg is sent to the Bubble Tea program when a watched file changes.
type FileChangeMsg struct {
	Path string
}

// Watcher monitors files for external changes using fsnotify.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	watched   map[string]bool
	mu        sync.Mutex
	callback  func(path string)
	debounce  map[string]time.Time
}

// New creates a new file watcher. The callback is called whenever a watched
// file is modified externally. It is safe to call from a goroutine.
func New(callback func(path string)) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher: fsw,
		watched:   make(map[string]bool),
		callback:  callback,
		debounce:  make(map[string]time.Time),
	}

	go w.listen()
	return w, nil
}

// Watch adds a file path to the watcher. It watches the parent directory
// so that file renames/recreations are also caught.
func (w *Watcher) Watch(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if w.watched[absPath] {
		return nil
	}

	dir := filepath.Dir(absPath)
	// Watch the directory (fsnotify requires watching dirs, not files)
	err = w.fsWatcher.Add(dir)
	if err != nil {
		return err
	}

	w.watched[absPath] = true
	return nil
}

// Unwatch removes a file path from the watcher.
func (w *Watcher) Unwatch(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	absPath, _ := filepath.Abs(path)
	delete(w.watched, absPath)
}

// Close shuts down the watcher.
func (w *Watcher) Close() {
	w.fsWatcher.Close()
}

// ignoredPaths filters out noise from editors, git, etc.
var ignoredSuffixes = []string{
	".swp", ".swo", "~", ".tmp", ".bak",
}

func (w *Watcher) listen() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Only care about writes and creates (external saves)
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			absPath, _ := filepath.Abs(event.Name)

			// Skip temporary/swap files
			skip := false
			for _, suffix := range ignoredSuffixes {
				if strings.HasSuffix(absPath, suffix) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}

			w.mu.Lock()
			isWatched := w.watched[absPath]

			// Debounce: ignore events within 500ms of each other for same file
			if isWatched {
				if last, ok := w.debounce[absPath]; ok {
					if time.Since(last) < 500*time.Millisecond {
						w.mu.Unlock()
						continue
					}
				}
				w.debounce[absPath] = time.Now()
			}
			w.mu.Unlock()

			if isWatched && w.callback != nil {
				w.callback(absPath)
			}

		case _, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			// Silently discard watcher errors to avoid crashing the editor
		}
	}
}
