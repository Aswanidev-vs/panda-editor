package fuzzy

import (
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Match struct {
	Path       string
	Name       string
	Score      int
	Positions  []int
	IsDir      bool
}

type Result struct {
	Matches []Match
	Query   string
}

func Score(pattern, target string) (int, []int) {
	pattern = strings.ToLower(pattern)
	targetLower := strings.ToLower(target)

	if pattern == "" {
		return 0, nil
	}

	if strings.Contains(targetLower, pattern) {
		idx := strings.Index(targetLower, pattern)
		positions := make([]int, len(pattern))
		for i := range positions {
			positions[i] = idx + i
		}
		score := 100
		if idx == 0 {
			score += 50
		}
		if len(pattern) == len(target) {
			score += 100
		}
		return score, positions
	}

	positions := make([]int, 0, len(pattern))
	pIdx := 0
	lastMatch := -1
	score := 0
	consecutive := 0

	for i, r := range targetLower {
		if pIdx < len(pattern) && r == rune(pattern[pIdx]) {
			positions = append(positions, i)
			if lastMatch == i-1 {
				consecutive++
				score += consecutive * 5
			} else {
				consecutive = 0
			}
			lastMatch = i

			rr, _ := utf8.DecodeRuneInString(target[i:])
			if unicode.IsUpper(rr) {
				score += 10
			}
			if i == 0 || !unicode.IsLetter(rune(target[i-1])) {
				score += 15
			}
			if i > 0 {
				prev, _ := utf8.DecodeLastRuneInString(target[:i])
				if prev == '.' || prev == '/' || prev == '-' || prev == '_' {
					score += 10
				}
			}
			score += 1
			pIdx++
		}
	}

	if pIdx == len(pattern) {
		score -= (len(target) - len(pattern)) * 2
		return score, positions
	}

	return -1, nil
}

func Search(pattern string, paths []string) []Match {
	if pattern == "" {
		matches := make([]Match, len(paths))
		for i, p := range paths {
			matches[i] = Match{
				Path: p,
				Name: filepath.Base(p),
				Score: 0,
			}
		}
		return matches
	}

	matches := make([]Match, 0)
	for _, p := range paths {
		name := filepath.Base(p)
		score, positions := Score(pattern, name)
		if score >= 0 {
			matches = append(matches, Match{
				Path:      p,
				Name:      name,
				Score:     score,
				Positions: positions,
			})
			continue
		}
		score, positions = Score(pattern, p)
		if score >= 0 {
			matches = append(matches, Match{
				Path:      p,
				Name:      name,
				Score:     score / 2,
				Positions: positions,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if len(matches) > 50 {
		matches = matches[:50]
	}

	return matches
}

func HighlightMatches(name string, positions []int) string {
	if len(positions) == 0 {
		return name
	}
	posSet := make(map[int]bool)
	for _, p := range positions {
		posSet[p] = true
	}
	var sb strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		if posSet[i] {
			sb.WriteString(string(r))
		} else {
			sb.WriteString(string(r))
		}
	}
	return sb.String()
}
