package bundler

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ResolveLocalDependencies takes a list of file paths, finds any Go files,
// parses their imports, and if they import local project packages, adds those
// files to the list.
func ResolveLocalDependencies(paths []string) []string {
	// 1. Try to find go.mod to get module name
	cwd, _ := os.Getwd()
	modData, err := os.ReadFile(filepath.Join(cwd, "go.mod"))
	if err != nil {
		return paths // Not a Go module or cannot read
	}
	
	modMatch := regexp.MustCompile(`module\s+([^\s]+)`).FindStringSubmatch(string(modData))
	if len(modMatch) < 2 {
		return paths
	}
	moduleName := modMatch[1]

	// To avoid infinite loops and duplicates
	resolved := make(map[string]bool)
	for _, p := range paths {
		abs, _ := filepath.Abs(p)
		resolved[abs] = true
	}

	var toProcess []string
	toProcess = append(toProcess, paths...)

	fset := token.NewFileSet()

	for i := 0; i < len(toProcess); i++ {
		p := toProcess[i]
		if filepath.Ext(p) != ".go" {
			continue
		}

		f, err := parser.ParseFile(fset, p, nil, parser.ImportsOnly)
		if err != nil {
			continue
		}

		for _, imp := range f.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(impPath, moduleName+"/") {
				// Resolve local path
				relPath := strings.TrimPrefix(impPath, moduleName+"/")
				dirPath := filepath.Join(cwd, filepath.FromSlash(relPath))
				
				// Read all .go files in that directory
				entries, err := os.ReadDir(dirPath)
				if err != nil {
					continue
				}

				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
						fullPath := filepath.Join(dirPath, entry.Name())
						abs, _ := filepath.Abs(fullPath)
						if !resolved[abs] {
							resolved[abs] = true
							toProcess = append(toProcess, fullPath)
						}
					}
				}
			}
		}
	}

	return toProcess
}

// GenerateMarkdown reads all the given paths and formats them into a single Markdown string
// suitable for LLM context injection.
func GenerateMarkdown(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("no files selected")
	}

	// Phase 4: Automatically resolve local dependencies
	resolvedPaths := ResolveLocalDependencies(paths)

	var sb strings.Builder
	sb.WriteString("# Bundled Context for AI\n\n")

	// Phase 5: Include Tree Structure
	cwd, _ := os.Getwd()
	sb.WriteString("## Project Structure\n```text\n")
	for _, p := range resolvedPaths {
		relPath, err := filepath.Rel(cwd, p)
		if err == nil {
			sb.WriteString("- " + relPath + "\n")
		} else {
			sb.WriteString("- " + p + "\n")
		}
	}
	sb.WriteString("```\n\n")

	for _, p := range resolvedPaths {
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			continue // Skip directories or unreadable files for now
		}

		content, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		// Detect extension for code block formatting
		ext := filepath.Ext(p)
		lang := ""
		if len(ext) > 1 {
			lang = ext[1:] // remove the dot
		}

		// Try to make the path relative for cleaner output
		relPath, err := filepath.Rel(cwd, p)
		if err == nil {
			p = relPath
		}

		sb.WriteString(fmt.Sprintf("## File: `%s`\n\n", p))
		sb.WriteString(fmt.Sprintf("```%s\n", lang))
		sb.WriteString(string(content))
		if !strings.HasSuffix(string(content), "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("```\n\n")
	}

	content := sb.String()
	// Basic secret redaction
	re := regexp.MustCompile(`(?i)(api_key|apikey|secret|token|password|passwd)["'\s]*[:=]["'\s]*([a-zA-Z0-9_\-\.]{10,})`)
	content = re.ReplaceAllString(content, `$1: "[REDACTED]"`)

	return content, nil
}
