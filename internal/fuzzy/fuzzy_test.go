package fuzzy

import "testing"

func TestScoreEmptyPattern(t *testing.T) {
	score, pos := Score("", "anything")
	if score != 0 || pos != nil {
		t.Errorf("Score('') = (%d, %v), want (0, nil)", score, pos)
	}
}

func TestScoreExactSubstring(t *testing.T) {
	score, pos := Score("foo", "foobar")
	if score < 100 {
		t.Errorf("Score('foo','foobar') = %d, want >= 100", score)
	}
	if len(pos) != 3 {
		t.Errorf("positions = %v, want 3 entries", pos)
	}
}

func TestScoreFuzzyMatch(t *testing.T) {
	score, _ := Score("fb", "foobar")
	if score < 0 {
		t.Errorf("Score('fb','foobar') = %d, want >= 0", score)
	}
}

func TestScoreNoMatch(t *testing.T) {
	score, pos := Score("xyz", "abc")
	if score != -1 || pos != nil {
		t.Errorf("Score('xyz','abc') = (%d, %v), want (-1, nil)", score, pos)
	}
}

func TestSearchEmptyPatternReturnsAll(t *testing.T) {
	paths := []string{"/a.go", "/b.go", "/c.go"}
	matches := Search("", paths)
	if len(matches) != 3 {
		t.Errorf("len(matches) = %d, want 3", len(matches))
	}
}

func TestSearchFiltersAndSorts(t *testing.T) {
	paths := []string{"/foo.go", "/bar.go", "/foobar.go"}
	matches := Search("foo", paths)
	if len(matches) == 0 {
		t.Fatal("expected at least one match for 'foo'")
	}
	// Best matches should come first.
	for i := 1; i < len(matches); i++ {
		if matches[i-1].Score < matches[i].Score {
			t.Errorf("matches not sorted: %d < %d at %d", matches[i-1].Score, matches[i].Score, i)
		}
	}
}

func TestSearchCapAt50(t *testing.T) {
	paths := make([]string, 100)
	for i := range paths {
		paths[i] = "/foo" + string(rune('a'+i%26)) + ".go"
	}
	matches := Search("foo", paths)
	if len(matches) > 50 {
		t.Errorf("matches = %d, want <= 50", len(matches))
	}
}

func TestHighlightMatchesEmpty(t *testing.T) {
	got := HighlightMatches("hello", nil)
	if got != "hello" {
		t.Errorf("HighlightMatches with no positions = %q, want %q", got, "hello")
	}
}

func TestHighlightMatchesWithPositions(t *testing.T) {
	got := HighlightMatches("hello", []int{0, 4})
	if got == "hello" {
		t.Error("HighlightMatches with positions should produce ANSI output")
	}
}