package searcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func writeTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"a.go":                       "hello world\nneedle here\nbye",
		"b.txt":                      "no match in this one",
		filepath.Join("sub", "c.md"): "NEEDLE uppercase\nnothing to see",
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func collect(t *testing.T, resultChan <-chan []SearchResult, doneChan <-chan bool, timeout time.Duration) []SearchResult {
	t.Helper()
	var acc []SearchResult
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case batch := <-resultChan:
			acc = append(acc, batch...)
		case <-doneChan:
			return acc
		case <-timer.C:
			t.Fatal("timed out waiting for search to finish")
		}
	}
}

func resultKeys(rs []SearchResult) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, fmt.Sprintf("%s:%d:%s", r.Path, r.LineNum, r.Content))
	}
	sort.Strings(out)
	return out
}

func TestSearchFindsMatches(t *testing.T) {
	root := writeTree(t)
	resultCh := make(chan []SearchResult)
	doneCh := make(chan bool)

	Search(Options{Root: root, Query: "needle"}, resultCh, doneCh)
	results := collect(t, resultCh, doneCh, 10*time.Second)

	keys := resultKeys(results)
	if len(keys) != 2 {
		t.Fatalf("got %d results (%v), want 2", len(keys), keys)
	}
	want := []string{
		fmt.Sprintf("%s:2:needle here", filepath.Join(root, "a.go")),
		fmt.Sprintf("%s:1:NEEDLE uppercase", filepath.Join(root, "sub", "c.md")),
	}
	for _, w := range want {
		found := false
		for _, k := range keys {
			if k == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing result %q in %v", w, keys)
		}
	}
}

// Search must behave exactly like SearchCtx(context.Background(), ...).
func TestSearchDelegatesToSearchCtx(t *testing.T) {
	root := writeTree(t)

	oldRes := make(chan []SearchResult)
	oldDone := make(chan bool)
	Search(Options{Root: root, Query: "needle"}, oldRes, oldDone)

	ctxRes := make(chan []SearchResult)
	ctxDone := make(chan bool)
	SearchCtx(context.Background(), Options{Root: root, Query: "needle"}, ctxRes, ctxDone)

	a := resultKeys(collect(t, oldRes, oldDone, 10*time.Second))
	b := resultKeys(collect(t, ctxRes, ctxDone, 10*time.Second))
	if len(a) != len(b) {
		t.Fatalf("Search found %d results, SearchCtx(background) found %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("result %d differs: %q vs %q", i, a[i], b[i])
		}
	}
}

// With an already-cancelled context nothing may be walked and no result may
// ever be produced; the search must wind down immediately.
func TestSearchCtxPreCancelledProducesNothing(t *testing.T) {
	root := writeTree(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resultCh := make(chan []SearchResult, 8)
	doneCh := make(chan bool)
	start := time.Now()
	SearchCtx(ctx, Options{Root: root, Query: "needle"}, resultCh, doneCh)

	select {
	case batch := <-resultCh:
		t.Fatalf("results produced despite cancellation: %v", batch)
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("doneChan not closed after cancellation")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("cancellation wind-down took %s; walker is not respecting ctx", elapsed)
	}
}

// Cancelling mid-walk/mid-search must close doneChan promptly and leave no
// goroutine writing to the channels afterwards.
func TestSearchCtxCancelMidWalk(t *testing.T) {
	root := writeTree(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultCh := make(chan []SearchResult)
	doneCh := make(chan bool)
	SearchCtx(ctx, Options{Root: root, Query: "needle"}, resultCh, doneCh)

	cancelled := false
	watchdog := time.After(5 * time.Second)
	for done := false; !done; {
		select {
		case <-resultCh:
			if !cancelled {
				cancel() // pull the plug as soon as work is flowing
				cancelled = true
			}
		case <-doneCh:
			done = true
		case <-watchdog:
			t.Fatal("search did not terminate after cancellation")
		}
	}

	// No orphan writes: the channels must stay quiet once done has fired.
	quiet := time.After(300 * time.Millisecond)
	for {
		select {
		case batch := <-resultCh:
			t.Fatalf("orphaned results after cancellation: %v", batch)
		case <-quiet:
			return
		}
	}
}
