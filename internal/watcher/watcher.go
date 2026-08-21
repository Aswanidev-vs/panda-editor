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
	// dirRefs counts how many watched files rely on each watched directory,
	// so Unwatch can release the underlying fsnotify handle exactly when the
	// last file in that directory goes away.
	dirRefs  map[string]int
	mu       sync.Mutex
	callback func(path string)
	debounce map[string]time.Time
	closed   bool
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
		dirRefs:   make(map[string]int),
		callback:  callback,
		debounce:  make(map[string]time.Time),
	}

	go w.listen()
	return w, nil
}

// Watch adds a file path to the watcher. It watches the parent directory
// so that file renames/recreations are also caught. Watching the same file
// twice is a no-op.
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
	w.dirRefs[dir]++

	w.watched[absPath] = true
	return nil
}

// Unwatch removes a file path from the watcher. Once no watched file relies
// on a directory any more, the underlying fsnotify watch is removed too.
func (w *Watcher) Unwatch(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	absPath, _ := filepath.Abs(path)
	if !w.watched[absPath] {
		return
	}
	delete(w.watched, absPath)

	dir := filepath.Dir(absPath)
	w.dirRefs[dir]--
	if w.dirRefs[dir] <= 0 {
		delete(w.dirRefs, dir)
		_ = w.fsWatcher.Remove(dir)
	}
}

// Close shuts down the watcher and releases every watch handle. It is safe
// to call more than once.
func (w *Watcher) Close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	fsw := w.fsWatcher
	w.watched = make(map[string]bool)
	w.dirRefs = make(map[string]int)
	w.debounce = make(map[string]time.Time)
	w.mu.Unlock()

	// Closing the fsnotify watcher shuts its event channels; the listen
	// goroutine sees them closed and exits, releasing all OS-level handles.
	_ = fsw.Close()
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
