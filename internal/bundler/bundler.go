package bundler

import (
	"encoding/xml"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// OutputFormat represents the bundle output format.
type OutputFormat int

const (
	FormatMarkdown OutputFormat = iota
	FormatXML
	FormatPlainText
)

// MaxFileSizeBytes is the default maximum file size to include (0 = no limit).
var MaxFileSizeBytes int64 = 0

// EstimateTokens gives a rough token count for a string (chars/4).
func EstimateTokens(s string) int {
	return len(s) / 4
}

// EstimateFileTokens returns estimated tokens for a file path.
func EstimateFileTokens(path string) int {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0
	}
	return int(info.Size()) / 4
}

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

// ---------- Internal: read file content with size check ----------

type fileEntry struct {
	Path    string
	RelPath string
	Lang    string
	Content string
	Skipped bool
}

func readFiles(resolvedPaths []string) []fileEntry {
	cwd, _ := os.Getwd()
	var entries []fileEntry

	for _, p := range resolvedPaths {
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			continue
		}

		relPath, err := filepath.Rel(cwd, p)
		if err != nil {
			relPath = p
		}

		// Phase 11: Max file size filter
		if MaxFileSizeBytes > 0 && info.Size() > MaxFileSizeBytes {
			entries = append(entries, fileEntry{
				Path:    p,
				RelPath: relPath,
				Skipped: true,
			})
			continue
		}

		content, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		ext := filepath.Ext(p)
		lang := ""
		if len(ext) > 1 {
			lang = ext[1:]
		}

		entries = append(entries, fileEntry{
			Path:    p,
			RelPath: relPath,
			Lang:    lang,
			Content: string(content),
		})
	}
	return entries
}

// ---------- Phase 8: Multiple output formats ----------

// GenerateBundle creates a bundle string in the given format.
func GenerateBundle(paths []string, format OutputFormat) (string, error) {
	switch format {
	case FormatXML:
		return GenerateXML(paths)
	case FormatPlainText:
		return GeneratePlainText(paths)
	default:
		return GenerateMarkdown(paths)
	}
}

// GenerateMarkdown generates Markdown output.
func GenerateMarkdown(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("no files selected")
	}

	resolvedPaths := ResolveLocalDependencies(paths)
	files := readFiles(resolvedPaths)
	cwd, _ := os.Getwd()

	var sb strings.Builder
	sb.WriteString("# Bundled Context for AI\n\n")

	// Project Structure
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

	for _, f := range files {
		if f.Skipped {
			sb.WriteString(fmt.Sprintf("## File: `%s` [SKIPPED: exceeds max size]\n\n", f.RelPath))
			continue
		}
		sb.WriteString(fmt.Sprintf("## File: `%s`\n\n", f.RelPath))
		sb.WriteString(fmt.Sprintf("```%s\n", f.Lang))
		sb.WriteString(f.Content)
		if !strings.HasSuffix(f.Content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("```\n\n")
	}

	return redactSecrets(sb.String()), nil
}

// GenerateXML generates XML output.
func GenerateXML(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("no files selected")
	}

	resolvedPaths := ResolveLocalDependencies(paths)
	files := readFiles(resolvedPaths)

	type XMLFile struct {
		XMLName xml.Name `xml:"file"`
		Path    string   `xml:"path,attr"`
		Skipped bool     `xml:"skipped,attr,omitempty"`
		Content string   `xml:",chardata"`
	}
	type XMLBundle struct {
		XMLName xml.Name  `xml:"bundle"`
		Files   []XMLFile `xml:"file"`
	}

	bundle := XMLBundle{}
	for _, f := range files {
		xf := XMLFile{Path: f.RelPath}
		if f.Skipped {
			xf.Skipped = true
		} else {
			xf.Content = f.Content
		}
		bundle.Files = append(bundle.Files, xf)
	}

	data, err := xml.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", err
	}

	output := xml.Header + string(data)
	return redactSecrets(output), nil
}

// GeneratePlainText generates plain text output.
func GeneratePlainText(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("no files selected")
	}

	resolvedPaths := ResolveLocalDependencies(paths)
	files := readFiles(resolvedPaths)

	var sb strings.Builder
	sb.WriteString("=== BUNDLED CONTEXT FOR AI ===\n\n")

	for _, f := range files {
		sb.WriteString(strings.Repeat("=", 60) + "\n")
		if f.Skipped {
			sb.WriteString(fmt.Sprintf("FILE: %s [SKIPPED: exceeds max size]\n", f.RelPath))
			sb.WriteString(strings.Repeat("=", 60) + "\n\n")
			continue
		}
		sb.WriteString(fmt.Sprintf("FILE: %s\n", f.RelPath))
		sb.WriteString(strings.Repeat("=", 60) + "\n")
		sb.WriteString(f.Content)
		if !strings.HasSuffix(f.Content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return redactSecrets(sb.String()), nil
}

// ---------- Secret redaction ----------

func redactSecrets(content string) string {
	re := regexp.MustCompile(`(?i)(api_key|apikey|secret|token|password|passwd)["'\s]*[:=]["'\s]*([a-zA-Z0-9_\-\.]{10,})`)
	return re.ReplaceAllString(content, `$1: "[REDACTED]"`)
}
