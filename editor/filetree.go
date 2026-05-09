package editor

import (
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
	Root     FileNode
	Width    int
	Cursor   int
	Scroll   int
	Flat     []FileNode
}

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

func NewFileTree(root string) FileTree {
	rootNode := buildTree(root, 0, 3)
	rootNode.Expanded = true

	ft := FileTree{
		Root:   rootNode,
		Width:  28,
		Cursor: 0,
		Scroll: 0,
	}
	ft.Flatten()
	return ft
}

func buildTree(dir string, depth, maxDepth int) FileNode {
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

		// Skip hidden files and ignored directories
		if strings.HasPrefix(name, ".") && name != ".gitignore" && name != ".env" {
			if ignoredDirs[name] {
				continue
			}
			// Still skip most hidden dirs
			if entry.IsDir() {
				continue
			}
		}

		if ignoredDirs[name] {
			continue
		}

		fullPath := filepath.Join(dir, name)

		if entry.IsDir() {
			child := buildTree(fullPath, depth+1, maxDepth)
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

func (ft *FileTree) Flatten() {
	ft.Flat = nil
	ft.flattenNode(&ft.Root)
}

func (ft *FileTree) flattenNode(node *FileNode) {
	if node.IsDir && node.Depth == 0 {
		// Root node: don't add it, just its children
		if node.Expanded {
			for i := range node.Children {
				ft.flattenNode(&node.Children[i])
			}
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
			// Lazy-load children
			*node = buildTree(path, node.Depth, node.Depth+3)
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
		node.Selected = !node.Selected
		return true
	}
	for i := range node.Children {
		if ft.toggleSelectInTree(&node.Children[i], path) {
			return true
		}
	}
	return false
}

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

		line := indent + checkbox + icon + " " + displayName

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

func (ft *FileTree) GetSelectedPaths() []string {
	var paths []string
	var walk func(node *FileNode)
	walk = func(node *FileNode) {
		if node.Selected {
			paths = append(paths, node.Path)
		}
		for i := range node.Children {
			walk(&node.Children[i])
		}
	}
	walk(&ft.Root)
	return paths
}
