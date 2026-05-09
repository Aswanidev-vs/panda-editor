package searcher

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SearchResult represents a match found in a file.
type SearchResult struct {
	Path    string
	LineNum int
	Content string
}

// SearchProgressMsg is sent to indicate search progress.
type SearchProgressMsg struct {
	Results []SearchResult
	Done    bool
}

// Search options.
type Options struct {
	Root          string
	Query         string
	CaseSensitive bool
	IgnoreDirs    []string
}

// Search performs a multi-threaded search across files.
func Search(opts Options, resultChan chan<- []SearchResult, doneChan chan<- bool) {
	go func() {
		defer close(doneChan)

		files := make(chan string, 100)
		results := make(chan SearchResult, 100)
		var wg sync.WaitGroup

		// Worker pool size
		numWorkers := 8
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for path := range files {
					matches, err := grepFile(path, opts.Query, opts.CaseSensitive)
					if err == nil {
						for _, m := range matches {
							results <- m
						}
					}
				}
			}()
		}

		// Result collector (batches results to reduce message frequency)
		collectorDone := make(chan bool)
		go func() {
			var batch []SearchResult
			for res := range results {
				batch = append(batch, res)
				if len(batch) >= 20 {
					resultChan <- batch
					batch = nil
				}
			}
			if len(batch) > 0 {
				resultChan <- batch
			}
			collectorDone <- true
		}()

		// File walker
		_ = filepath.Walk(opts.Root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				name := info.Name()
				for _, ignore := range opts.IgnoreDirs {
					if name == ignore {
						return filepath.SkipDir
					}
				}
				if strings.HasPrefix(name, ".") && name != "." {
					return filepath.SkipDir
				}
				return nil
			}

			// Only search text files (rough check)
			ext := strings.ToLower(filepath.Ext(path))
			if isTextFile(ext) {
				files <- path
			}
			return nil
		})

		close(files)
		wg.Wait()
		close(results)
		<-collectorDone
	}()
}

func grepFile(path string, query string, caseSensitive bool) ([]SearchResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []SearchResult
	scanner := bufio.NewScanner(file)
	lineNum := 1

	q := query
	if !caseSensitive {
		q = strings.ToLower(q)
	}

	for scanner.Scan() {
		content := scanner.Text()
		searchContent := content
		if !caseSensitive {
			searchContent = strings.ToLower(content)
		}

		if strings.Contains(searchContent, q) {
			matches = append(matches, SearchResult{
				Path:    path,
				LineNum: lineNum,
				Content: strings.TrimSpace(content),
			})
		}
		lineNum++
		// Cap results per file to avoid massive output
		if len(matches) > 100 {
			break
		}
	}

	return matches, scanner.Err()
}

func isTextFile(ext string) bool {
	switch ext {
	case ".go", ".js", ".ts", ".py", ".c", ".cpp", ".h", ".hpp", ".java", ".rs", ".md", ".txt", ".json", ".yaml", ".yml", ".toml", ".css", ".html", ".sh", ".sql", ".lua":
		return true
	default:
		return false
	}
}
