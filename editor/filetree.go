package editor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Aswanidev-vs/panda-editor/internal/theme"
)

type FileNode struct {
	Name     string
	Path     string
	IsDir    bool
	Children []FileNode
	Expanded bool
	Selected bool
	Depth    int
}

type FileTree struct {
	Root            FileNode
	Width           int
	Cursor          int
	Scroll          int
	Flat            []FileNode
	gitignoreRules  []gitignoreRule
	showIgnored     bool
	ShowTokens      bool
	Filter          string
}

// ---------- Gitignore support (Phase 6) ----------

type gitignoreRule struct {
	pattern  string
	negate   bool
	dirOnly  bool
}

// parseGitignore reads a .gitignore file and returns parsed rules.
func parseGitignore(root string) []gitignoreRule {
	path := filepath.Join(root, ".gitignore")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var rules []gitignoreRule
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		rule := gitignoreRule{}

		if strings.HasPrefix(line, "!") {
			rule.negate = true
			line = line[1:]
		}

		if strings.HasSuffix(line, "/") {
			rule.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}

		rule.pattern = line
		rules = append(rules, rule)
	}
	return rules
}

// isGitignored checks whether a file/dir name matches any gitignore rule.
func isGitignored(name string, isDir bool, rules []gitignoreRule) bool {
	ignored := false
	for _, rule := range rules {
		if rule.dirOnly && !isDir {
			continue
		}
		matched, _ := filepath.Match(rule.pattern, name)
		if !matched {
			// Also try matching as a path component (e.g., "*.exe" against "foo.exe")
			matched, _ = filepath.Match(rule.pattern, filepath.Base(name))
		}
		if matched {
			if rule.negate {
				ignored = false
			} else {
				ignored = true
			}
		}
	}
	return ignored
}

// ---------- File type icons ----------

// File type icon mapping for a richer sidebar look
func fileIcon(name string, isDir bool) string {
	if isDir {
		return "📂"
	}
	return "📄"
}

func dirIcon(expanded bool) string {
	if expanded {
		return "📂"
	}
	return "📁"
}

var ignoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".vscode":      true,
	"__pycache__":  true,
	".idea":        true,
	"vendor":       true,
	".next":        true,
	"dist":         true,
	"build":        true,
	".cache":       true,
	".DS_Store":    true,
	"target":       true,
}

// ---------- Tree construction ----------

func NewFileTree(root string) FileTree {
	rules := parseGitignore(root)
	rootNode := buildTreeWithRules(root, 0, 3, rules)
	rootNode.Expanded = true

	ft := FileTree{
		Root:           rootNode,
		Width:          28,
		Cursor:         0,
		Scroll:         0,
		gitignoreRules: rules,
	}
	ft.Flatten()
	return ft
}

func buildTreeWithRules(dir string, depth, maxDepth int, rules []gitignoreRule) FileNode {
	node := FileNode{
		Name:     filepath.Base(dir),
		Path:     dir,
		IsDir:    true,
		Expanded: depth < 1,
		Depth:    depth,
	}

	if depth >= maxDepth {
		return node
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return node
	}

	// Sort: directories first, then alphabetical
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	for _, entry := range entries {
		name := entry.Name()

		// Always skip hardcoded ignored dirs
		if ignoredDirs[name] {
			continue
		}

		// Skip hidden files/dirs (except .gitignore, .env)
		if strings.HasPrefix(name, ".") && name != ".gitignore" && name != ".env" {
			if entry.IsDir() {
				continue
			}
		}

		// Phase 6: Check gitignore rules
		if isGitignored(name, entry.IsDir(), rules) {
			continue
		}

		fullPath := filepath.Join(dir, name)

		if entry.IsDir() {
			child := buildTreeWithRules(fullPath, depth+1, maxDepth, rules)
			node.Children = append(node.Children, child)
		} else {
			node.Children = append(node.Children, FileNode{
				Name:  name,
				Path:  fullPath,
				IsDir: false,
				Depth: depth + 1,
			})
		}
	}

	return node
}

// buildTree is kept for backward compat (lazy loading uses rules from tree)
func buildTree(dir string, depth, maxDepth int) FileNode {
	return buildTreeWithRules(dir, depth, maxDepth, nil)
}

// ---------- Flatten ----------

func (ft *FileTree) Flatten() {
	ft.Flat = nil
	if ft.Filter != "" {
		ft.flattenFiltered(&ft.Root)
	} else {
		ft.flattenNode(&ft.Root)
	}
}

func (ft *FileTree) flattenNode(node *FileNode) {
	if node.IsDir && node.Depth == 0 {
		// Root node: don't add it, just its children
		for i := range node.Children {
			ft.flattenNode(&node.Children[i])
		}
		return
	}

	ft.Flat = append(ft.Flat, *node)
	if node.IsDir && node.Expanded {
		for i := range node.Children {
			ft.flattenNode(&node.Children[i])
		}
	}
}

func (ft *FileTree) flattenFiltered(node *FileNode) {
	if !node.IsDir {
		if strings.Contains(strings.ToLower(node.Name), strings.ToLower(ft.Filter)) {
			ft.Flat = append(ft.Flat, *node)
		}
	}
	for i := range node.Children {
		ft.flattenFiltered(&node.Children[i])
	}
}

// ---------- Toggle expand/collapse ----------

func (ft *FileTree) Toggle(idx int) {
	if idx < 0 || idx >= len(ft.Flat) {
		return
	}
	node := &ft.Flat[idx]
	if !node.IsDir {
		return
	}
	// Find and toggle in the actual tree
	ft.toggleInTree(&ft.Root, node.Path)
	ft.Flatten()
}

func (ft *FileTree) toggleInTree(node *FileNode, path string) bool {
	if node.Path == path {
		node.Expanded = !node.Expanded
		if node.Expanded && len(node.Children) == 0 {
			// Lazy-load children using gitignore rules
			*node = buildTreeWithRules(path, node.Depth, node.Depth+3, ft.gitignoreRules)
			node.Expanded = true
		}
		return true
	}
	for i := range node.Children {
		if ft.toggleInTree(&node.Children[i], path) {
			return true
		}
	}
	return false
}

// ---------- Selection (Phase 7: recursive directory selection) ----------

func (ft *FileTree) ToggleSelect(idx int) {
	if idx < 0 || idx >= len(ft.Flat) {
		return
	}
	node := &ft.Flat[idx]
	ft.toggleSelectInTree(&ft.Root, node.Path)
	ft.Flatten()
}

func (ft *FileTree) toggleSelectInTree(node *FileNode, path string) bool {
	if node.Path == path {
		newState := !node.Selected
		node.Selected = newState
		// Phase 7: If it's a directory, recursively select/deselect all children
		if node.IsDir {
			setSelectRecursive(node, newState)
		}
		return true
	}
	for i := range node.Children {
		if ft.toggleSelectInTree(&node.Children[i], path) {
			return true
		}
	}
	return false
}

// setSelectRecursive sets the Selected state on all children of a node.
func setSelectRecursive(node *FileNode, state bool) {
	for i := range node.Children {
		node.Children[i].Selected = state
		if node.Children[i].IsDir {
			setSelectRecursive(&node.Children[i], state)
		}
	}
}

// ---------- Query helpers ----------

func (ft *FileTree) SelectedPath() string {
	if ft.Cursor < 0 || ft.Cursor >= len(ft.Flat) {
		return ""
	}
	return ft.Flat[ft.Cursor].Path
}

func (ft *FileTree) SelectedNode() *FileNode {
	if ft.Cursor < 0 || ft.Cursor >= len(ft.Flat) {
		return nil
	}
	path := ft.Flat[ft.Cursor].Path
	return ft.findNode(&ft.Root, path)
}

func (ft *FileTree) findNode(node *FileNode, path string) *FileNode {
	if node.Path == path {
		return node
	}
	for i := range node.Children {
		if found := ft.findNode(&node.Children[i], path); found != nil {
			return found
		}
	}
	return nil
}

func (ft *FileTree) AdjustScroll(visibleHeight int) {
	if ft.Cursor < ft.Scroll {
		ft.Scroll = ft.Cursor
	}
	if ft.Cursor >= ft.Scroll+visibleHeight {
		ft.Scroll = ft.Cursor - visibleHeight + 1
	}
	if ft.Scroll < 0 {
		ft.Scroll = 0
	}
}

// ---------- Render ----------

func (ft *FileTree) Render(visibleHeight int) string {
	t := theme.CurrentTheme
	var sb strings.Builder

	// Header with subtle styling
	headerStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		PaddingLeft(1).
		PaddingBottom(0)
	sb.WriteString(headerStyle.Render("🐼 EXPLORER"))
	sb.WriteString("\n")

	// Thin separator
	sepStyle := lipgloss.NewStyle().Foreground(t.Border)
	sb.WriteString(sepStyle.Render(strings.Repeat("─", ft.Width-2)))
	sb.WriteString("\n")

	if len(ft.Flat) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(t.Comment).
			Italic(true).
			PaddingLeft(2)
		sb.WriteString(emptyStyle.Render("(empty)"))
		return sb.String()
	}

	renderHeight := visibleHeight - 2 // Account for header + separator
	if renderHeight < 1 {
		renderHeight = 1
	}

	start := ft.Scroll
	end := start + renderHeight
	if end > len(ft.Flat) {
		end = len(ft.Flat)
	}

	for i := start; i < end; i++ {
		node := ft.Flat[i]
		isSelected := i == ft.Cursor

		// Build indent with tree guides
		indent := ""
		depth := node.Depth - 1
		if depth < 0 {
			depth = 0
		}
		for d := 0; d < depth; d++ {
			indent += "  "
		}

		// Icon
		var icon string
		if node.IsDir {
			icon = dirIcon(node.Expanded)
			if node.Expanded {
				indent += "▾ "
			} else {
				indent += "▸ "
			}
		} else {
			indent += "  "
			icon = fileIcon(node.Name, false)
		}

		// Checkbox
		checkbox := "[ ] "
		if node.Selected {
			checkbox = "[x] "
		}

		// Truncate name if too long
		maxNameLen := ft.Width - len([]rune(indent)) - len([]rune(checkbox)) - 5
		displayName := node.Name
		if maxNameLen > 0 && len(displayName) > maxNameLen {
			displayName = displayName[:maxNameLen-1] + "…"
		}

		// Phase 9: Per-file token display
		tokenSuffix := ""
		if ft.ShowTokens && !node.IsDir {
			info, err := os.Stat(node.Path)
			if err == nil {
				tokens := int(info.Size()) / 4
				if tokens > 1000 {
					tokenSuffix = fmt.Sprintf(" ~%dk", tokens/1000)
				} else {
					tokenSuffix = fmt.Sprintf(" ~%dt", tokens)
				}
			}
		}

		line := indent + checkbox + icon + " " + displayName + tokenSuffix

		var style lipgloss.Style
		if isSelected {
			style = lipgloss.NewStyle().
				Background(t.Selection).
				Foreground(t.Fg).
				Bold(true).
				Width(ft.Width - 2)
		} else if node.IsDir {
			style = lipgloss.NewStyle().
				Foreground(t.Accent).
				Width(ft.Width - 2)
		} else {
			style = lipgloss.NewStyle().
				Foreground(t.Sidebar).
				Width(ft.Width - 2)
		}

		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	// Fill remaining space
	for i := end - start; i < renderHeight; i++ {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(t.Border).
			Width(ft.Width - 2).
			Render(""))
		sb.WriteString("\n")
	}

	// Scrollbar indicator
	if len(ft.Flat) > renderHeight {
		scrollPercent := float64(ft.Cursor) / float64(len(ft.Flat)-1)
		scrollInfo := lipgloss.NewStyle().
			Foreground(t.Comment).
			Italic(true).
			PaddingLeft(1).
			Render(strings.Repeat("░", int(scrollPercent*float64(ft.Width-4))) +
				"█" +
				strings.Repeat("░", ft.Width-4-int(scrollPercent*float64(ft.Width-4))))
		_ = scrollInfo // only used if needed
	}

	return sb.String()
}


// ---------- Phase 10: Expand/Collapse All ----------

func (ft *FileTree) ToggleExpandAll() {
	// Check if most dirs are expanded or collapsed
	expanded := 0
	collapsed := 0
	ft.countExpandState(&ft.Root, &expanded, &collapsed)

	newState := expanded <= collapsed // if more collapsed, expand all; else collapse
	ft.setExpandRecursive(&ft.Root, newState)
	ft.Flatten()
}

func (ft *FileTree) countExpandState(node *FileNode, expanded, collapsed *int) {
	if node.IsDir && node.Depth > 0 {
		if node.Expanded {
			*expanded++
		} else {
			*collapsed++
		}
	}
	for i := range node.Children {
		ft.countExpandState(&node.Children[i], expanded, collapsed)
	}
}

func (ft *FileTree) setExpandRecursive(node *FileNode, state bool) {
	if node.IsDir {
		node.Expanded = state
		if state && len(node.Children) == 0 {
			// Lazy-load children
			*node = buildTreeWithRules(node.Path, node.Depth, node.Depth+3, ft.gitignoreRules)
			node.Expanded = true
		}
	}
	for i := range node.Children {
		ft.setExpandRecursive(&node.Children[i], state)
	}
}

func (ft *FileTree) GetSelectedPaths() []string {
	var paths []string
	ft.Root.collectSelected(&paths)
	return paths
}

func (n *FileNode) collectSelected(paths *[]string) {
	if n.Selected && !n.IsDir {
		*paths = append(*paths, n.Path)
	}
	for i := range n.Children {
		n.Children[i].collectSelected(paths)
	}
}
