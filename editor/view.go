package editor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Aswanidev-vs/panda-editor/internal/config"
	"github.com/Aswanidev-vs/panda-editor/internal/highlight"
	"github.com/Aswanidev-vs/panda-editor/internal/lsp"
	"github.com/Aswanidev-vs/panda-editor/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

func (e *Editor) View() string {
	if e.width == 0 || e.height == 0 {
		return "Loading..."
	}

	if e.mode == ViewWelcome {
		return e.renderWelcome()
	}

	var sections []string

	// Tab bar
	tabBar := e.renderTabBar()
	sections = append(sections, tabBar)

	// Main content area
	var mainContent string
	editorPanel := e.renderEditor()
	minimapPanel := e.renderMinimap()
	combinedEditor := lipgloss.JoinHorizontal(lipgloss.Top, editorPanel, minimapPanel)

	if e.fileTreeVisible {
		treePanel := e.renderFileTree()
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, treePanel, combinedEditor)
	} else {
		mainContent = combinedEditor
	}
	sections = append(sections, mainContent)

	// Status bar
	statusBar := e.renderStatusBar()
	sections = append(sections, statusBar)

	result := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Render overlays
	switch e.mode {
	case ViewFinder:
		result = e.renderFinderOverlay(result)
	case ViewCommandPalette:
		result = e.renderCommandOverlay(result)
	case ViewSearch, ViewSearchReplace:
		result = e.renderSearchOverlay(result)
	case ViewGoToLine:
		result = e.renderGoToOverlay(result)
	case ViewSaveAs:
		result = e.renderSaveAsOverlay(result)
	case ViewHelp:
		result = e.renderHelpOverlay(result)
	case ViewKeybindings:
		result = e.renderKeybindingsOverlay(result)
	case ViewKeybindEdit:
		result = e.renderKeybindingsOverlay(result)
	case ViewOpenFolder:
		result = e.renderOpenFolderOverlay(result)
	case ViewGlobalSearch:
		result = e.renderGlobalSearchOverlay(result)
	case ViewUnsavedPrompt:
		result = e.renderUnsavedPrompt(result)
	case ViewSettings:
		result = e.renderSettingsOverlay(result)
	}

	// Render autocomplete if active
	if len(e.suggestions) > 0 {
		result = e.renderAutocompleteOverlay(result)
	}

	// Render diagnostic hint if cursor is on an error
	if e.mode == ViewNormal {
		tab := e.currentTab()
		diag := e.getDiagnosticAt(tab.Buf.FilePath, tab.CursorLine, tab.CursorCol)
		if diag != nil {
			result = e.renderDiagnosticHint(result, diag)
		}
	}

	return result
}

func (e *Editor) renderTabBar() string {
	t := theme.CurrentTheme
	var sb strings.Builder

	tabBarStyle := lipgloss.NewStyle().
		Background(t.TitleBar).
		Width(e.width).
		Height(1)

	for i, tab := range e.tabs {
		icon := getFileIcon(tab.Buf.Language)
		name := tab.Buf.Name

		var prefix string
		if tab.Buf.Modified {
			prefix = "● "
		} else {
			prefix = "  "
		}

		label := prefix + icon + " " + name + " ✕"

		var tabStyle lipgloss.Style
		if i == e.activeTab {
			tabStyle = lipgloss.NewStyle().
				Background(t.TabActiveBg).
				Foreground(t.TabActiveFg).
				Bold(true).
				Padding(0, 1).
				BorderStyle(lipgloss.Border{Bottom: "▀"}).
				BorderBottom(true).
				BorderForeground(t.Accent)
		} else {
			tabStyle = lipgloss.NewStyle().
				Background(t.TabBg).
				Foreground(t.TabFg).
				Padding(0, 1)
		}

		sb.WriteString(tabStyle.Render(label))

		// Tab separator
		if i < len(e.tabs)-1 {
			sb.WriteString(lipgloss.NewStyle().Background(t.TitleBar).Foreground(t.Border).Render("│"))
		}
	}

	// Fill remaining space
	rendered := sb.String()
	remaining := e.width - lipgloss.Width(rendered)
	if remaining > 0 {
		sb.WriteString(lipgloss.NewStyle().Background(t.TitleBar).Width(remaining).Render(""))
	}

	return tabBarStyle.Render(sb.String())
}

// fileTabIcon returns a small language-specific icon for tabs
func getFileIcon(lang string) string {
	switch lang {
	case "go":
		return ""
	case "python":
		return ""
	case "javascript":
		return ""
	case "typescript":
		return ""
	case "rust":
		return ""
	case "java":
		return ""
	case "html":
		return ""
	case "css", "scss":
		return ""
	case "json":
		return ""
	case "yaml":
		return ""
	case "markdown":
		return ""
	case "shell":
		return ""
	default:
		return " "
	}
}

func getLangFromExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".html":
		return "html"
	case ".css", ".scss":
		return "css"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".sh", ".bash":
		return "shell"
	default:
		return ""
	}
}

func (e *Editor) renderFileTree() string {
	t := theme.CurrentTheme
	treeWidth := e.fileTree.Width
	treeHeight := e.editorHeight()

	e.fileTree.AdjustScroll(treeHeight - 2)

	style := lipgloss.NewStyle().
		Background(t.SidebarBg).
		Width(treeWidth).
		Height(treeHeight).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(t.Border)

	header := ""
	headerHeight := 0
	if e.mode == ViewFileTreeFilter || e.fileTree.Filter != "" {
		filterStyle := lipgloss.NewStyle().
			Background(t.SidebarBg).
			Foreground(t.Accent).
			PaddingLeft(1).
			PaddingTop(1).
			Bold(true)
		header = filterStyle.Render(" / " + e.fileTreeFilterInput.View())
		headerHeight = 2
	}

	content := e.fileTree.Render(treeHeight - 1 - headerHeight)

	// Add footer hint
	hintStyle := lipgloss.NewStyle().
		Background(t.SidebarBg).
		Foreground(t.Comment).
		Faint(true).
		PaddingLeft(1).
		PaddingTop(0).
		Width(treeWidth)

	footer := hintStyle.Render("G/↑↓ Enter /:Filter")

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, header, content, footer))
}

func (e *Editor) renderEditor() string {
	if e.splitActive {
		w := e.editorWidth()
		// Validate rightTab
		rightTab := e.rightTab
		if rightTab < 0 || rightTab >= len(e.tabs) {
			rightTab = e.activeTab
		}
		// Vertical separator
		sep := lipgloss.NewStyle().
			Foreground(theme.CurrentTheme.Border).
			Height(e.editorHeight()).
			Width(1).
			Render("│")
		return lipgloss.JoinHorizontal(lipgloss.Top, e.renderPanel(e.activeTab, w), sep, e.renderPanel(rightTab, w-1))
	}
	return e.renderPanel(e.activeTab, e.editorWidth())
}

func (e *Editor) renderPanel(tabIdx int, width int) string {
	t := theme.CurrentTheme
	if tabIdx < 0 || tabIdx >= len(e.tabs) {
		return lipgloss.NewStyle().Width(width).Height(e.editorHeight()).Render(" No tab ")
	}
	tab := &e.tabs[tabIdx]
	buf := tab.Buf

	editorWidth := width
	editorHeight := e.editorHeight()

	e.lineNumberWidth = len(fmt.Sprintf("%d", buf.LineCount())) + 1
	if e.lineNumberWidth < 4 {
		e.lineNumberWidth = 4
	}

	var sb strings.Builder

	startLine := tab.ScrollLine
	endLine := startLine + editorHeight
	if endLine > buf.LineCount() {
		endLine = buf.LineCount()
	}

	// Phase 13: Find matching bracket
	matchL, matchC := -1, -1
	if tab.CursorLine >= 0 && tab.CursorLine < buf.LineCount() {
		matchL, matchC = buf.FindMatchingBracket(tab.CursorLine, tab.CursorCol)
	}

	inBlockComment := false
	if startLine > 0 {
		for i := 0; i < startLine; i++ {
			_, inBlockComment = highlight.HighlightLine(buf.GetLine(i), buf.Language, inBlockComment)
		}
	}

	absPath, _ := filepath.Abs(buf.FilePath)
	diags := e.fileDiagnostics[absPath]

	isCursorLine := false
	for lineNum := startLine; lineNum < endLine; lineNum++ {
		lineContent := buf.GetLine(lineNum)
		isCursorLine = lineNum == tab.CursorLine

		lineBg := t.Bg
		// Highlight cursor line only in active panel
		isActive := false
		if !e.splitActive {
			isActive = true
		} else if e.activePanel == 0 && tabIdx == e.activeTab {
			isActive = true
		} else if e.activePanel == 1 && tabIdx == e.rightTab {
			isActive = true
		}

		if isCursorLine && isActive {
			lineBg = t.CursorLine
		}
		baseStyle := lipgloss.NewStyle().Background(lineBg)

		// Line number
		var lineNumStr string
		if e.relativeLineNo && lineNum != tab.CursorLine {
			dist := lineNum - tab.CursorLine
			if dist < 0 {
				dist = -dist
			}
			lineNumStr = fmt.Sprintf("%*d ", e.lineNumberWidth-1, dist)
		} else {
			lineNumStr = fmt.Sprintf("%*d ", e.lineNumberWidth-1, lineNum+1)
		}

		var lineNumStyle lipgloss.Style
		if isCursorLine && isActive {
			lineNumStyle = baseStyle.Copy().
				Foreground(t.LineNumActive).
				Bold(true).
				Width(e.lineNumberWidth)
		} else {
			lineNumStyle = baseStyle.Copy().
				Foreground(t.LineNum).
				Width(e.lineNumberWidth)
		}

		sb.WriteString(lineNumStyle.Render(lineNumStr))

		// Phase 16: Git gutter
		marker := " "
		if diffs, ok := e.fileDiffs[buf.FilePath]; ok {
			if m, exists := diffs[lineNum]; exists {
				marker = m
			}
		}
		markerStyle := baseStyle.Copy()
		switch marker {
		case "+":
			markerStyle = markerStyle.Foreground(lipgloss.Color("#00ff00"))
		case "~":
			markerStyle = markerStyle.Foreground(lipgloss.Color("#ffff00"))
		case "-":
			markerStyle = markerStyle.Foreground(lipgloss.Color("#ff0000"))
		}
		sb.WriteString(markerStyle.Render(marker))

		// Gutter separator
		hasDiag := false
		if diags != nil {
			for _, d := range diags {
				if d.Range.Start.Line == lineNum {
					hasDiag = true
					break
				}
			}
		}

		if hasDiag {
			sb.WriteString(baseStyle.Copy().Foreground(lipgloss.Color("#ff0000")).Render("⊗"))
		} else if isCursorLine && isActive {
			sb.WriteString(baseStyle.Copy().Foreground(t.Accent).Render("│"))
		} else {
			sb.WriteString(baseStyle.Copy().Foreground(t.Border).Render("│"))
		}

		// Line content with syntax highlighting and scrolling
		runes := []rune(lineContent)
		scrollCol := tab.ScrollCol
		visibleCols := editorWidth - e.lineNumberWidth - 2
		if visibleCols < 5 {
			visibleCols = 5
		}

		if scrollCol > len(runes) {
			scrollCol = len(runes)
		}
		endCol := scrollCol + visibleCols
		if endCol > len(runes) {
			endCol = len(runes)
		}

		visibleRunes := runes[scrollCol:endCol]

		// Check for selection highlighting
		isSelected := func(col int) bool {
			if !tab.SelectActive {
				return false
			}
			sl, sc, el, ec := tab.SelectStartL, tab.SelectStartC, tab.CursorLine, tab.CursorCol
			if sl > el || (sl == el && sc > ec) {
				sl, sc, el, ec = el, ec, sl, sc
			}
			absCol := col + scrollCol
			if lineNum < sl || lineNum > el {
				return false
			}
			if lineNum == sl && absCol < sc {
				return false
			}
			if lineNum == el && absCol >= ec {
				return false
			}
			return true
		}

		isSearchMatch := func(col int) bool {
			if e.searchQuery == "" || !e.searchActive {
				return false
			}
			absCol := col + scrollCol
			q := e.searchQuery
			lowerLine := strings.ToLower(lineContent)
			lowerQ := strings.ToLower(q)
			idx := 0
			for {
				pos := strings.Index(lowerLine[idx:], lowerQ)
				if pos < 0 {
					return false
				}
				absStart := idx + pos
				absEnd := absStart + len(q)
				if absCol >= absStart && absCol < absEnd {
					return true
				}
				idx = absStart + 1
			}
		}

		isDiagnostic := func(col int) bool {
			if len(diags) == 0 {
				return false
			}
			absCol := col + scrollCol
			for _, d := range diags {
				if lineNum == d.Range.Start.Line && lineNum == d.Range.End.Line {
					if absCol >= d.Range.Start.Character && absCol < d.Range.End.Character {
						return true
					}
				} else if lineNum == d.Range.Start.Line && absCol >= d.Range.Start.Character {
					return true
				} else if lineNum == d.Range.End.Line && absCol < d.Range.End.Character {
					return true
				} else if lineNum > d.Range.Start.Line && lineNum < d.Range.End.Line {
					return true
				}
			}
			return false
		}

		// Syntax highlight the visible portion
		visibleStr := string(visibleRunes)
		visibleHighlighted, bcState := highlight.HighlightLine(visibleStr, buf.Language, inBlockComment)
		inBlockComment = bcState

		charIdx := 0
		dispWidth := 0
		for _, span := range visibleHighlighted {
			style := highlight.TokenToStyle(span.Token)
			for _, r := range span.Text {
				colIdx := charIdx
				if isSelected(colIdx) {
					style = lipgloss.NewStyle().Background(t.Selection).Foreground(t.Fg)
				} else if isSearchMatch(colIdx) {
					style = lipgloss.NewStyle().Background(t.Accent).Foreground(t.Bg)
				} else if isDiagnostic(colIdx) {
					style = style.Foreground(lipgloss.Color("#ff0000")).Underline(true)
				}

				rw := lipgloss.Width(string(r))
				// Cursor or Matching Bracket
				if isActive && lineNum == tab.CursorLine && colIdx+scrollCol == tab.CursorCol {
					cursorStyle := lipgloss.NewStyle().
						Background(t.Cursor).
						Foreground(t.Bg).
						Bold(true)
					sb.WriteString(cursorStyle.Render(string(r)))
				} else if isActive && lineNum == matchL && colIdx+scrollCol == matchC {
					bracketStyle := lipgloss.NewStyle().
						Background(t.Accent).
						Foreground(t.Bg).
						Bold(true)
					sb.WriteString(bracketStyle.Render(string(r)))
				} else {
					sb.WriteString(baseStyle.Copy().Inherit(style).Render(string(r)))
				}
				charIdx++
				dispWidth += rw
			}
		}

		// Cursor at end of line
		if isActive && lineNum == tab.CursorLine && tab.CursorCol >= len(runes) && len(runes) >= scrollCol && len(runes) < endCol {
			cursorStyle := lipgloss.NewStyle().
				Background(t.Cursor).
				Foreground(t.Bg).
				Bold(true)
			sb.WriteString(cursorStyle.Render(" "))
			charIdx++
			dispWidth++
		}

		// Pad to fill width
		usedWidth := e.lineNumberWidth + 2 + dispWidth
		if usedWidth < editorWidth {
			sb.WriteString(baseStyle.Copy().Render(strings.Repeat(" ", editorWidth-usedWidth)))
		}

		sb.WriteString("\n")
	}

	// Fill remaining lines with dim tilde
	for lineNum := endLine; lineNum < startLine+editorHeight; lineNum++ {
		lineNumStr := fmt.Sprintf("%*s ", e.lineNumberWidth-1, "~")
		sb.WriteString(lipgloss.NewStyle().Foreground(t.Border).Background(t.Bg).Render(lineNumStr))
		sb.WriteString(lipgloss.NewStyle().Foreground(t.Border).Background(t.Bg).Render(" "))
		sb.WriteString(lipgloss.NewStyle().Foreground(t.Border).Background(t.Bg).Render("│"))
		sb.WriteString(lipgloss.NewStyle().Background(t.Bg).Render(strings.Repeat(" ", editorWidth-e.lineNumberWidth-2)))
		sb.WriteString("\n")
	}

	editorStyle := lipgloss.NewStyle().
		Background(t.Bg).
		Width(editorWidth).
		Height(editorHeight)

	return editorStyle.Render(sb.String())
}

func (e *Editor) renderStatusBar() string {
	t := theme.CurrentTheme
	tab := e.currentTab()
	buf := tab.Buf

	// Mode badge — colored indicator
	modeName := ""
	switch e.mode {
	case ViewWelcome:
		modeName = " 🐼 WELCOME "
	case ViewFinder:
		modeName = " 🔍 FINDER "
	case ViewCommandPalette:
		modeName = " ⌘ COMMAND "
	case ViewSearch:
		modeName = " 🔎 SEARCH "
	case ViewSearchReplace:
		modeName = " ⇄ REPLACE "
	case ViewGoToLine:
		modeName = " # GO TO "
	case ViewSaveAs:
		modeName = " 💾 SAVE AS "
	case ViewKeybindings, ViewKeybindEdit:
		modeName = " ⌨ KEYS "
	case ViewOpenFolder:
		modeName = " 📂 FOLDER "
	case ViewGlobalSearch:
		modeName = " 🔍 GLOBAL "
	default:
		modeName = " ✎ EDIT "
	}

	modeBadge := lipgloss.NewStyle().
		Background(t.Accent).
		Foreground(t.Bg).
		Bold(true).
		Render(modeName)

	// Left info: file name + language
	leftParts := " "
	if buf.Name != "" && buf.Name != "[scratch]" {
		leftParts += buf.Name
	} else {
		leftParts += "[scratch]"
	}
	if buf.Modified {
		leftParts += " ●"
	}
	if buf.Language != "" {
		leftParts += "  " + strings.ToUpper(buf.Language)
	}

	// Add diagnostics if any
	absPath, _ := filepath.Abs(buf.FilePath)
	if diags, ok := e.fileDiagnostics[absPath]; ok && len(diags) > 0 {
		leftParts += fmt.Sprintf("  %s %d", "⊗", len(diags))
	}

	// Add Git branch
	if e.gitBranch != "" {
		leftParts += "   " + e.gitBranch
	}

	leftStyle := lipgloss.NewStyle().
		Background(t.StatusBar).
		Foreground(t.Fg).
		Render(leftParts)

	// Right info: position, lines, message
	var rightParts []string
	if e.searchActive && e.searchQuery != "" {
		rightParts = append(rightParts, fmt.Sprintf("🔍 %s", e.searchQuery))
	}
	rightParts = append(rightParts, fmt.Sprintf("Ln %d, Col %d", tab.CursorLine+1, tab.CursorCol+1))
	rightParts = append(rightParts, fmt.Sprintf("%d lines", buf.LineCount()))

	// Token Estimation
	totalChars := 0
	for _, line := range buf.Lines {
		totalChars += len(line)
	}
	estimatedTokens := totalChars / 4
	rightParts = append(rightParts, fmt.Sprintf("~%dt", estimatedTokens))
	if len(e.multiCursors) > 0 {
		rightParts = append(rightParts, fmt.Sprintf("%d cursors", len(e.multiCursors)+1))
	}

	// Message toast
	if len(e.messages) > 0 && e.messageTimer > 0 {
		rightParts = append(rightParts, e.messages[len(e.messages)-1])
	}

	right := " " + strings.Join(rightParts, "  │  ") + " "

	rightStyle := lipgloss.NewStyle().
		Background(t.StatusBar).
		Foreground(t.Comment).
		Render(right)

	// Calculate gap
	usedWidth := lipgloss.Width(modeBadge) + lipgloss.Width(leftStyle) + lipgloss.Width(rightStyle)
	gap := e.width - usedWidth
	if gap < 0 {
		gap = 0
	}

	gapStyle := lipgloss.NewStyle().
		Background(t.StatusBar).
		Width(gap)

	statusBar := lipgloss.JoinHorizontal(lipgloss.Center,
		modeBadge,
		leftStyle,
		gapStyle.Render(""),
		rightStyle,
	)

	return statusBar
}

func (e *Editor) renderFinderOverlay(bg string) string {
	t := theme.CurrentTheme
	width := 60
	height := 20

	if e.width < width {
		width = e.width - 4
	}
	if e.height < height {
		height = e.height - 2
	}

	var sb strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Width(width - 2).
		Align(lipgloss.Center)
	sb.WriteString(titleStyle.Render("Quick Open"))
	sb.WriteString("\n")

	// Input
	sb.WriteString(" ")
	sb.WriteString(e.finderInput.View())
	sb.WriteString("\n\n")

	// Results - Edge Scrolling
	visibleResults := height - 6
	if e.finderCursor < e.finderScroll {
		e.finderScroll = e.finderCursor
	} else if e.finderCursor >= e.finderScroll+visibleResults {
		e.finderScroll = e.finderCursor - visibleResults + 1
	}
	
	start := e.finderScroll
	end := start + visibleResults
	if end > len(e.finderResults) {
		end = len(e.finderResults)
	}

	for i := start; i < end; i++ {
		match := e.finderResults[i]
		lang := getLangFromExt(match.Path)
		icon := getFileIcon(lang)

		var lineStyle lipgloss.Style
		if i == e.finderCursor {
			lineStyle = lipgloss.NewStyle().
				Background(t.Selection).
				Width(width-4).
				Padding(0, 1)
		} else {
			lineStyle = lipgloss.NewStyle().
				Padding(0, 1)
		}

		// Highlight matched characters in filename
		posSet := make(map[int]bool)
		for _, p := range match.Positions {
			posSet[p] = true
		}
		var nameSb strings.Builder
		runes := []rune(match.Name)
		for i, r := range runes {
			var chStyle lipgloss.Style
			if posSet[i] {
				chStyle = lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
			} else {
				chStyle = lipgloss.NewStyle().Foreground(t.Fg)
			}
			nameSb.WriteString(chStyle.Render(string(r)))
		}
		namePart := nameSb.String()
		dirStr := ""
		if match.Path != match.Name {
			dir := filepath.Dir(match.Path)
			if dir != "." {
				dir += "/"
				if len(dir) > 30 {
					dir = "..." + dir[len(dir)-27:]
				}
				dirStr = dir
			}
		}
		dirPart := lipgloss.NewStyle().Foreground(t.Comment).Render(dirStr)

		content := icon + " " + dirPart + namePart
		sb.WriteString(lineStyle.Render(content))
		sb.WriteString("\n")
	}

	// Fill remaining
	for i := end - start; i < visibleResults; i++ {
		sb.WriteString("\n")
	}

	// Hint
	hintStyle := lipgloss.NewStyle().
		Foreground(t.Comment).
		Italic(true).
		Padding(0, 1)
	sb.WriteString(hintStyle.Render("↑↓ Navigate  Enter Open  Esc Cancel"))

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Bg).
		Width(width).
		Height(height).
		Padding(0, 1)

	overlay := overlayStyle.Render(sb.String())
	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}

func (e *Editor) renderCommandOverlay(bg string) string {
	t := theme.CurrentTheme
	width := 65
	height := 22

	if e.width < width {
		width = e.width - 4
	}
	if e.height < height {
		height = e.height - 2
	}

	var sb strings.Builder

	// Icon mapping
	getIcon := func(cat string) string {
		switch cat {
		case "File":
			return "📄"
		case "Edit":
			return "✏️"
		case "View":
			return "👁️"
		case "Search":
			return "🔍"
		case "Navigation":
			return "🧭"
		case "Settings":
			return "⚙️"
		case "System":
			return "🖥️"
		default:
			return "•"
		}
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Width(width - 2).
		Align(lipgloss.Center).
		MarginBottom(1)
	sb.WriteString(titleStyle.Render("PANDA COMMAND PALETTE"))
	sb.WriteString("\n")

	sb.WriteString("  ")
	sb.WriteString(e.commandInput.View())
	sb.WriteString("\n")

	// Commands - Edge Scrolling
	visibleResults := height - 8
	if e.commandCursor < e.commandScroll {
		e.commandScroll = e.commandCursor
	} else if e.commandCursor >= e.commandScroll+visibleResults {
		e.commandScroll = e.commandCursor - visibleResults + 1
	}
	
	start := e.commandScroll
	end := start + visibleResults
	if end > len(e.commandResults) {
		end = len(e.commandResults)
	}

	for i := start; i < end; i++ {
		cmd := e.commandResults[i]
		icon := getIcon(cmd.Category)

		nameStyle := lipgloss.NewStyle().Foreground(t.Fg).Width(30)
		catStyle := lipgloss.NewStyle().Foreground(t.AccentAlt).Width(12)
		descStyle := lipgloss.NewStyle().Foreground(t.Comment)

		if i == e.commandCursor {
			nameStyle = nameStyle.Foreground(t.Bg).Background(t.Accent).Bold(true)
			catStyle = catStyle.Foreground(t.Bg).Background(t.Accent)
			descStyle = descStyle.Foreground(t.Bg).Background(t.Accent)
		}

		iconStr := lipgloss.NewStyle().Width(3).Render(icon)
		nameStr := nameStyle.Render(cmd.Name)
		catStr := catStyle.Render(cmd.Category)
		
		line := fmt.Sprintf(" %s %s %s", iconStr, nameStr, catStr)
		if i == e.commandCursor {
			sb.WriteString(lipgloss.NewStyle().Background(t.Accent).Width(width - 4).Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")

		// Optional: show description on separate line or small text
		if i == e.commandCursor {
			dLine := fmt.Sprintf("     %s", cmd.Description)
			sb.WriteString(lipgloss.NewStyle().Background(t.Accent).Foreground(t.Bg).Italic(true).Width(width - 4).Render(dLine))
			sb.WriteString("\n")
		} else {
			sb.WriteString("\n")
		}
	}

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Comment).
		Italic(true).
		Padding(0, 2)
	sb.WriteString("\n")
	sb.WriteString(hintStyle.Render("↑↓ Navigate • Enter Execute • Esc Cancel"))

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Bg).
		Width(width).
		Height(height).
		Padding(1, 0)

	overlay := overlayStyle.Render(sb.String())
	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}

func (e *Editor) renderSearchOverlay(bg string) string {
	t := theme.CurrentTheme
	width := 50
	height := 8

	if e.mode == ViewSearchReplace {
		height = 11
	}

	if e.width < width {
		width = e.width - 4
	}

	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Padding(0, 1)

	if e.mode == ViewSearchReplace {
		sb.WriteString(titleStyle.Render("Search & Replace"))
	} else {
		sb.WriteString(titleStyle.Render("Search"))
	}
	sb.WriteString("\n")

	sb.WriteString(" ")
	sb.WriteString(e.searchInput.View())
	sb.WriteString("\n")

	if e.mode == ViewSearchReplace {
		sb.WriteString(" ")
		sb.WriteString(e.replaceInput.View())
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	matchCount := 0
	if e.searchQuery != "" {
		buf := e.currentBuf()
		q := e.searchQuery
		if !e.caseSensitive {
			q = strings.ToLower(q)
		}
		for i := 0; i < buf.LineCount(); i++ {
			line := buf.GetLine(i)
			if !e.caseSensitive {
				line = strings.ToLower(line)
			}
			matchCount += strings.Count(line, q)
		}
	}

	infoStyle := lipgloss.NewStyle().
		Foreground(t.Comment).
		Padding(0, 1)
	if matchCount > 0 {
		sb.WriteString(infoStyle.Render(fmt.Sprintf("%d matches found", matchCount)))
	} else if e.searchQuery != "" {
		sb.WriteString(infoStyle.Render("No matches"))
	}
	sb.WriteString("\n")

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Comment).
		Italic(true).
		Padding(0, 1)
	if e.mode == ViewSearchReplace {
		sb.WriteString(hintStyle.Render("Enter Find  Ctrl+R Replace  Ctrl+Shift+R All  Tab Switch  Esc Cancel"))
	} else {
		sb.WriteString(hintStyle.Render("Enter Find Next  Esc Cancel"))
	}

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Bg).
		Width(width).
		Height(height).
		Padding(0, 1)

	overlay := overlayStyle.Render(sb.String())
	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}

func (e *Editor) renderGoToOverlay(bg string) string {
	t := theme.CurrentTheme
	width := 40
	height := 5

	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Padding(0, 1)
	sb.WriteString(titleStyle.Render("Go to Line"))
	sb.WriteString("\n")
	sb.WriteString(" ")
	sb.WriteString(e.gotoInput.View())
	sb.WriteString("\n")

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Comment).
		Italic(true).
		Padding(0, 1)
	sb.WriteString(hintStyle.Render("Enter Go  Esc Cancel"))

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Bg).
		Width(width).
		Height(height).
		Padding(0, 1)

	overlay := overlayStyle.Render(sb.String())
	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}

func (e *Editor) renderSaveAsOverlay(bg string) string {
	t := theme.CurrentTheme
	width := 55
	height := 5

	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Padding(0, 1)
	sb.WriteString(titleStyle.Render("Save As"))
	sb.WriteString("\n")
	sb.WriteString(" ")
	sb.WriteString(e.saveAsInput.View())
	sb.WriteString("\n")

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Comment).
		Italic(true).
		Padding(0, 1)
	sb.WriteString(hintStyle.Render("Enter Save  Esc Cancel"))

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Bg).
		Width(width).
		Height(height).
		Padding(0, 1)

	overlay := overlayStyle.Render(sb.String())
	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}

func (e *Editor) renderHelpOverlay(bg string) string {
	t := theme.CurrentTheme
	width := 70
	height := 35

	if e.width < width {
		width = e.width - 4
	}
	if e.height < height {
		height = e.height - 2
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Underline(true)

	headerStyle := lipgloss.NewStyle().
		Foreground(t.AccentAlt).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Width(25)

	descStyle := lipgloss.NewStyle().
		Foreground(t.Fg)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("  Panda Editor - Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	sections := []struct {
		title string
		items []struct{ key, desc string }
	}{
		{
			"Navigation",
			[]struct{ key, desc string }{
				{"↑/↓/←/→ or h/j/k/l", "Move cursor"},
				{"Ctrl+←/→ or Alt+B/F", "Move by word"},
				{"Home/End", "Line start/end"},
				{"Ctrl+Home/End", "File start/end"},
				{"PgUp/PgDn or Ctrl+U/D", "Page up/down"},
				{"Ctrl+G", "Go to line"},
				{"Alt+C", "Center cursor"},
				{"Ctrl+↑/↓", "Scroll up/down"},
			},
		},
		{
			"Editing",
			[]struct{ key, desc string }{
				{"Enter", "Insert newline"},
				{"Backspace / Delete", "Delete char"},
				{"Ctrl+K", "Delete line"},
				{"Ctrl+Shift+D", "Duplicate line"},
				{"Alt+↑/↓", "Move line up/down"},
				{"Tab / Shift+Tab", "Indent / Unindent"},
				{"Ctrl+/", "Toggle comment"},
				{"Ctrl+Z", "Undo"},
				{"Ctrl+Y / Ctrl+Shift+Z", "Redo"},
			},
		},
		{
			"Selection",
			[]struct{ key, desc string }{
				{"Shift+Arrow keys", "Select text"},
				{"Ctrl+Shift+←/→", "Select by word"},
				{"Shift+Home/End", "Select to line start/end"},
				{"Ctrl+A", "Select all"},
				{"Ctrl+L", "Select line"},
			},
		},
		{
			"Clipboard",
			[]struct{ key, desc string }{
				{"Ctrl+C", "Copy"},
				{"Ctrl+X", "Cut"},
				{"Ctrl+V", "Paste"},
			},
		},
		{
			"Files & Search",
			[]struct{ key, desc string }{
				{"Ctrl+S", "Save"},
				{"Ctrl+Shift+S", "Save as"},
				{"Ctrl+P", "Quick open / Fuzzy finder"},
				{"Ctrl+B", "Toggle sidebar"},
				{"Ctrl+Shift+P", "Command palette"},
				{"Ctrl+F", "Search in file"},
				{"Alt+F", "Global search"},
				{"Ctrl+H", "Search & replace"},
				{"F3 / Shift+F3", "Next/Previous match"},
				{"Ctrl+N", "New tab"},
				{"Ctrl+W", "Close tab"},
				{"Ctrl+Tab / Ctrl+Shift+Tab", "Next/Previous tab"},
			},
		},
		{
			"View",
			[]struct{ key, desc string }{
				{"Ctrl+\\", "Toggle sidebar"},
				{"Ctrl+=/−", "Zoom in/out"},
				{"Ctrl+T", "Toggle theme"},
				{"F1", "Toggle help"},
				{"Alt+K", "Keyboard shortcuts editor"},
			},
		},
		{
			"Quit",
			[]struct{ key, desc string }{
				{"Ctrl+Q", "Quit"},
				{"Ctrl+Shift+Q", "Force quit"},
			},
		},
	}

	var contentLines []string
	for _, sec := range sections {
		contentLines = append(contentLines, headerStyle.Render("  "+sec.title))
		for _, item := range sec.items {
			line := "    " + keyStyle.Render(item.key) + descStyle.Render(item.desc)
			contentLines = append(contentLines, line)
		}
		contentLines = append(contentLines, "")
	}

	visibleHeight := height - 6
	if e.helpScroll > len(contentLines)-visibleHeight && len(contentLines) > visibleHeight {
		e.helpScroll = len(contentLines) - visibleHeight
	}
	if e.helpScroll < 0 {
		e.helpScroll = 0
	}

	for i := e.helpScroll; i < len(contentLines) && i < e.helpScroll+visibleHeight; i++ {
		sb.WriteString(contentLines[i])
		sb.WriteString("\n")
	}
	
	// Fill remaining space if any
	for i := len(contentLines) - e.helpScroll; i < visibleHeight; i++ {
		sb.WriteString("\n")
	}

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Comment).
		Italic(true)
	sb.WriteString(hintStyle.Render("  Press Esc or F1 to close  |  Alt+K for keybinding editor"))

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Bg).
		Width(width).
		Height(height).
		Padding(0, 1)

	overlay := overlayStyle.Render(sb.String())
	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}

func (e *Editor) renderWelcome() string {
	t := theme.CurrentTheme

	bgStyle := lipgloss.NewStyle().
		Background(t.Bg).
		Width(e.width).
		Height(e.height)

	var sb strings.Builder

	// Panda ASCII art — compact & iconic
	logo := []string{
		"  ██████╗  █████╗ ███╗   ██╗██████╗  █████╗ ",
		"  ██╔══██╗██╔══██╗████╗  ██║██╔══██╗██╔══██╗",
		"  ██████╔╝███████║██╔██╗ ██║██║  ██║███████║",
		"  ██╔═══╝ ██╔══██║██║╚██╗██║██║  ██║██╔══██║",
		"  ██║     ██║  ██║██║ ╚████║██████╔╝██║  ██║",
		"  ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝ ╚═╝  ╚═╝",
	}

	logoStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(t.AccentAlt).
		Italic(true)

	versionStyle := lipgloss.NewStyle().
		Foreground(t.Comment)

	// Center vertically
	totalHeight := len(logo) + 18
	topPad := (e.height - totalHeight) / 2
	if topPad < 0 {
		topPad = 0
	}
	for i := 0; i < topPad; i++ {
		sb.WriteString("\n")
	}

	// Logo
	for _, line := range logo {
		padding := (e.width - lipgloss.Width(line)) / 2
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(strings.Repeat(" ", padding))
		sb.WriteString(logoStyle.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Subtitle
	subtitle := "🐼  A fast, keyboard-driven TUI code editor"
	subPad := (e.width - lipgloss.Width(subtitle)) / 2
	if subPad < 0 {
		subPad = 0
	}
	sb.WriteString(strings.Repeat(" ", subPad))
	sb.WriteString(subtitleStyle.Render(subtitle))
	sb.WriteString("\n")

	// Version
	ver := "v1.1.0"
	verPad := (e.width - len(ver)) / 2
	if verPad < 0 {
		verPad = 0
	}
	sb.WriteString(strings.Repeat(" ", verPad))
	sb.WriteString(versionStyle.Render(ver))
	sb.WriteString("\n\n")

	// Separator
	sep := strings.Repeat("─", 40)
	sepPad := (e.width - len(sep)) / 2
	if sepPad < 0 {
		sepPad = 0
	}
	sb.WriteString(strings.Repeat(" ", sepPad))
	sb.WriteString(lipgloss.NewStyle().Foreground(t.Border).Render(sep))
	sb.WriteString("\n\n")

	// Menu items
	type menuItem struct {
		key   string
		icon  string
		label string
	}

	var items []menuItem
	items = append(items, menuItem{"n", "📄", "New File"})
	items = append(items, menuItem{"o", "📂", "Open File"})
	items = append(items, menuItem{"f", "📁", "Open Folder"})
	if e.hasSession {
		items = append(items, menuItem{"r", "🔄", "Restore Session"})
	}
	if len(e.tabs) > 0 {
		items = append(items, menuItem{"x", "🧹", "Exit Session"})
	}
	items = append(items, menuItem{"q", "🚪", "Quit"})

	menuWidth := 36
	for i, item := range items {
		isSelected := i == e.welcomeCursor

		var line string
		if isSelected {
			keyBadge := lipgloss.NewStyle().
				Background(t.Accent).
				Foreground(t.Bg).
				Bold(true).
				Padding(0, 1).
				Render(item.key)
			label := lipgloss.NewStyle().
				Background(t.Selection).
				Foreground(t.Fg).
				Bold(true).
				Width(menuWidth-6).
				Padding(0, 1).
				Render(item.icon + "  " + item.label)
			line = keyBadge + " " + label
		} else {
			keyBadge := lipgloss.NewStyle().
				Foreground(t.Accent).
				Bold(true).
				Padding(0, 1).
				Render(item.key)
			label := lipgloss.NewStyle().
				Foreground(t.Fg).
				Width(menuWidth-6).
				Padding(0, 1).
				Render(item.icon + "  " + item.label)
			line = keyBadge + " " + label
		}

		lineWidth := lipgloss.Width(line)
		padding := (e.width - lineWidth) / 2
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(strings.Repeat(" ", padding))
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Shortcut hints
	shortcutStyle := lipgloss.NewStyle().Foreground(t.Comment)
	shortcuts := []string{
		"F1 → Help    Alt+K → Keybindings    Ctrl+P → Quick Open",
	}
	for _, s := range shortcuts {
		pad := (e.width - lipgloss.Width(s)) / 2
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(strings.Repeat(" ", pad))
		sb.WriteString(shortcutStyle.Render(s))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Footer navigation hints
	hintStyle := lipgloss.NewStyle().
		Foreground(t.Comment).
		Italic(true)

	hints := "↑↓ Navigate  ⏎ Select  Esc Quit"
	hintPad := (e.width - lipgloss.Width(hints)) / 2
	if hintPad < 0 {
		hintPad = 0
	}
	sb.WriteString(strings.Repeat(" ", hintPad))
	sb.WriteString(hintStyle.Render(hints))

	// Session info
	if e.hasSession && e.sessionInfo != "" {
		sb.WriteString("\n\n")
		infoPad := (e.width - lipgloss.Width(e.sessionInfo)) / 2
		if infoPad < 0 {
			infoPad = 0
		}
		sb.WriteString(strings.Repeat(" ", infoPad))
		sb.WriteString(lipgloss.NewStyle().Foreground(t.Success).Render(e.sessionInfo))
	}

	return bgStyle.Render(sb.String())
}

func (e *Editor) renderUnsavedPrompt(bg string) string {
	t := theme.CurrentTheme
	width := 60
	height := 8

	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Padding(0, 1)
	sb.WriteString(titleStyle.Render("UNSAVED CHANGES"))
	sb.WriteString("\n\n")

	tab := e.tabs[e.unsavedTabIdx]
	fileName := tab.Buf.FilePath
	if fileName == "" {
		fileName = "[Untitled]"
	} else {
		fileName = filepath.Base(fileName)
	}

	msgStyle := lipgloss.NewStyle().Foreground(t.Fg).Padding(0, 2)
	sb.WriteString(msgStyle.Render(fmt.Sprintf("File '%s' has unsaved changes.", fileName)))
	sb.WriteString("\n")
	sb.WriteString(msgStyle.Render("Do you want to save them before closing?"))
	sb.WriteString("\n\n")

	hintStyle := lipgloss.NewStyle().Foreground(t.Comment).Italic(true).Padding(0, 2)
	sb.WriteString(hintStyle.Render(" Y: Save & Close  |  N: Discard  |  Esc: Cancel "))

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Bg).
		Width(width).
		Height(height).
		Padding(1, 0)

	overlay := overlayStyle.Render(sb.String())
	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}

func (e *Editor) renderOpenFolderOverlay(bg string) string {
	t := theme.CurrentTheme
	width := 55
	height := 6

	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Padding(0, 1)
	sb.WriteString(titleStyle.Render("Open Folder"))
	sb.WriteString("\n")
	sb.WriteString(" ")
	sb.WriteString(e.openFolderInput.View())
	sb.WriteString("\n")

	hintStyle := lipgloss.NewStyle().Foreground(t.Comment).Italic(true).Padding(0, 1)
	sb.WriteString(hintStyle.Render("Enter path to workspace folder"))
	sb.WriteString("\n")
	sb.WriteString(hintStyle.Render("Enter Open  Esc Cancel"))

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Bg).
		Width(width).
		Height(height).
		Padding(0, 1)

	overlay := overlayStyle.Render(sb.String())
	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}

func (e *Editor) renderKeybindingsOverlay(bg string) string {
	t := theme.CurrentTheme
	width := 78
	height := e.height - 4

	if width > e.width-4 {
		width = e.width - 4
	}
	if height < 15 {
		height = 15
	}
	if height > 40 {
		height = 40
	}

	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Underline(true)
	sb.WriteString(titleStyle.Render("  Keyboard Shortcuts Editor"))
	sb.WriteString("\n")

	if e.kbEditing {
		entry := e.kbEntries[e.kbEditIndex]
		editTitleStyle := lipgloss.NewStyle().Foreground(t.AccentAlt).Bold(true)
		sb.WriteString(editTitleStyle.Render(fmt.Sprintf("  Editing: %s (%s)", entry.Label, entry.Category)))
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(t.Comment).Render("  " + entry.Description))
		sb.WriteString("\n\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(t.Fg).Render("  Current: " + JoinKeys(entry.Keys)))
		sb.WriteString("\n\n")
		sb.WriteString("  New keybinding: ")
		sb.WriteString(e.kbEditInput.View())
		sb.WriteString("\n\n")
		editHint := lipgloss.NewStyle().Foreground(t.Comment).Italic(true)
		sb.WriteString(editHint.Render("  Format: key1, key2 (e.g. ctrl+s, ctrl+shift+s)"))
		sb.WriteString("\n")
		sb.WriteString(editHint.Render("  Enter to save  Esc to cancel"))
	} else {
		// Column headers
		headerStyle := lipgloss.NewStyle().Foreground(t.AccentAlt).Bold(true)
		sb.WriteString("\n")
		sb.WriteString(headerStyle.Render("  "))
		sb.WriteString(headerStyle.Width(22).Render("Action"))
		sb.WriteString(headerStyle.Width(14).Render("Category"))
		sb.WriteString(headerStyle.Width(36).Render("Keys"))
		sb.WriteString("\n")

		sepStyle := lipgloss.NewStyle().Foreground(t.Border)
		sb.WriteString(sepStyle.Render("  " + strings.Repeat("─", width-6)))
		sb.WriteString("\n")

		// Visible entries
		visibleHeight := height - 10
		start := e.kbScroll
		end := start + visibleHeight
		if end > len(e.kbEntries) {
			end = len(e.kbEntries)
		}

		lastCategory := ""
		for i := start; i < end; i++ {
			entry := e.kbEntries[i]

			// Category separator
			if entry.Category != lastCategory {
				if lastCategory != "" {
					sb.WriteString("\n")
				}
				catStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).PaddingLeft(2)
				sb.WriteString(catStyle.Render(entry.Category))
				sb.WriteString("\n")
				lastCategory = entry.Category
			}

			// Row style
			var rowStyle lipgloss.Style
			if i == e.kbCursor {
				rowStyle = lipgloss.NewStyle().
					Background(t.Selection).
					Foreground(t.Fg).
					Bold(true)
			} else {
				rowStyle = lipgloss.NewStyle().Foreground(t.Fg)
			}

			labelCol := lipgloss.NewStyle().Width(22).PaddingLeft(4).Render(entry.Label)
			catCol := lipgloss.NewStyle().Width(14).Foreground(t.Comment).Render(entry.Category)
			keysStr := JoinKeys(entry.Keys)
			keysCol := lipgloss.NewStyle().Foreground(t.Accent).Width(36).Render(keysStr)

			row := labelCol + catCol + keysCol
			sb.WriteString(rowStyle.Width(width - 4).Render(row))
			sb.WriteString("\n")
		}
	}

	// Hint bar
	sb.WriteString("\n")
	hintStyle := lipgloss.NewStyle().Foreground(t.Comment).Italic(true)
	if e.kbEditing {
		// hints already shown above
	} else {
		sb.WriteString(hintStyle.Render("  ↑↓ Navigate  Enter Edit  R Reset  S Save  O Open Config  Esc Close"))
	}

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Bg).
		Width(width).
		Height(height).
		Padding(0, 1)

	overlay := overlayStyle.Render(sb.String())
	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}

func placeOverlay(x, y int, overlay, bg string) string {
	overlayLines := strings.Split(overlay, "\n")
	bgLines := strings.Split(bg, "\n")

	bH := len(bgLines)
	oH := len(overlayLines)

	// Result slice
	result := make([]string, bH)
	copy(result, bgLines)

	for i := 0; i < oH; i++ {
		targetLine := y + i
		if targetLine < 0 || targetLine >= bH {
			continue
		}

		ol := overlayLines[i]
		bl := bgLines[targetLine]

		result[targetLine] = overlayAnsi(bl, ol, x)
	}

	return strings.Join(result, "\n")
}

func toCells(s string) []string {
	var cells []string
	var currentStyle string

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\x1b' {
			start := i
			i++
			if i < len(runes) && runes[i] == '[' {
				i++
				for i < len(runes) && !((runes[i] >= 'A' && runes[i] <= 'Z') || (runes[i] >= 'a' && runes[i] <= 'z') || runes[i] == 'm') {
					i++
				}
			}
			if i < len(runes) {
				seq := string(runes[start : i+1])
				if seq == "\x1b[0m" {
					currentStyle = ""
				} else {
					currentStyle += seq
				}
			}
		} else {
			// Normal character
			char := string(runes[i])
			w := lipgloss.Width(char)
			if w == 0 {
				if len(cells) > 0 {
					cells[len(cells)-1] += char
				}
				continue
			}

			// Add styled character
			cell := currentStyle + char + "\x1b[0m"
			cells = append(cells, cell)
			
			// Handle wide characters by adding placeholders
			for k := 1; k < w; k++ {
				cells = append(cells, "") // Marker for occupied column
			}
		}
	}
	return cells
}

func overlayAnsi(bg, fg string, x int) string {
	bgCells := toCells(bg)
	fgCells := toCells(fg)

	// Pad background if needed
	if x > len(bgCells) {
		for len(bgCells) < x {
			bgCells = append(bgCells, " ")
		}
	}

	for i, cell := range fgCells {
		idx := x + i
		if idx >= 0 && idx < 2000 { // Safety limit
			if idx < len(bgCells) {
				// If we overwrite a cell, we must handle multi-column character boundaries
				
				// 1. If we overwrite a placeholder, we must clear the parent character
				if bgCells[idx] == "" && idx > 0 {
					for p := idx - 1; p >= 0; p-- {
						if bgCells[p] != "" {
							bgCells[p] = " " // Replace parent with space
							break
						}
					}
				}

				// 2. If we overwrite a parent of placeholders, we must clear them
				for p := idx + 1; p < len(bgCells) && bgCells[p] == ""; p++ {
					bgCells[p] = " "
				}

				bgCells[idx] = cell
			} else {
				bgCells = append(bgCells, cell)
			}
		}
	}

	var sb strings.Builder
	for _, c := range bgCells {
		if c == "" {
			continue // Skip placeholders as the main char already covers these columns
		}
		sb.WriteString(c)
	}
	return sb.String()
}
func (e *Editor) renderGlobalSearchOverlay(bg string) string {
	t := theme.CurrentTheme
	width := 80
	height := 25

	if e.width < width+4 {
		width = e.width - 4
	}
	if e.height < height+4 {
		height = e.height - 4
	}

	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Width(width - 2).
		Align(lipgloss.Center).
		MarginBottom(1)
	sb.WriteString(headerStyle.Render("🔍 GLOBAL SEARCH"))
	sb.WriteString("\n")

	sb.WriteString(e.globalSearchInput.View())
	sb.WriteString("\n\n")

	if e.isSearching {
		sb.WriteString(lipgloss.NewStyle().Foreground(t.AccentAlt).Render("Searching..."))
		sb.WriteString("\n")
	}

	pathStyle := lipgloss.NewStyle().Foreground(t.Accent)
	lineStyle := lipgloss.NewStyle().Foreground(t.Comment)
	matchStyle := lipgloss.NewStyle().Foreground(t.Fg)
	selectedStyle := lipgloss.NewStyle().Background(t.Selection).Width(width - 2)

	// Global Search Results - Edge Scrolling
	visibleResults := height - 8
	if e.globalSearchCursor < e.globalSearchScroll {
		e.globalSearchScroll = e.globalSearchCursor
	} else if e.globalSearchCursor >= e.globalSearchScroll+visibleResults {
		e.globalSearchScroll = e.globalSearchCursor - visibleResults + 1
	}
	
	start := e.globalSearchScroll
	end := start + visibleResults
	if end > len(e.globalSearchResults) {
		end = len(e.globalSearchResults)
	}

	for i := start; i < end; i++ {
		res := e.globalSearchResults[i]
		path := filepath.Base(res.Path)
		if len(path) > 20 {
			path = path[:17] + "..."
		}

		pStr := pathStyle.Render(fmt.Sprintf("%-20s", path))
		lStr := lineStyle.Render(fmt.Sprintf(":%-4d", res.LineNum))
		cStr := matchStyle.Render(res.Content)

		line := pStr + lStr + " " + cStr
		if lipgloss.Width(line) > width-4 {
			line = line[:width-7] + "..."
		}

		if i == e.globalSearchCursor {
			sb.WriteString(selectedStyle.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	if len(e.globalSearchResults) == 0 && !e.isSearching && e.globalSearchInput.Value() != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(t.Comment).Render("No results found."))
		sb.WriteString("\n")
	}

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Comment).
		Italic(true)
	sb.WriteString("\n")
	sb.WriteString(hintStyle.Render("Enter: Search/Open | Tab: Focus Input | Esc: Close"))

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Bg).
		Width(width).
		Height(height).
		Padding(1, 2)

	overlay := overlayStyle.Render(sb.String())
	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}

func (e *Editor) getDiagnosticAt(path string, line, col int) *lsp.Diagnostic {
	absPath, _ := filepath.Abs(path)
	diags, ok := e.fileDiagnostics[absPath]
	if !ok {
		return nil
	}

	for _, d := range diags {
		if line == d.Range.Start.Line && line == d.Range.End.Line {
			if col >= d.Range.Start.Character && col < d.Range.End.Character {
				return &d
			}
		} else if line == d.Range.Start.Line && col >= d.Range.Start.Character {
			return &d
		} else if line == d.Range.End.Line && col < d.Range.End.Character {
			return &d
		} else if line > d.Range.Start.Line && line < d.Range.End.Line {
			return &d
		}
	}
	return nil
}

func (e *Editor) renderDiagnosticHint(bg string, diag *lsp.Diagnostic) string {
	t := theme.CurrentTheme

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#ff0000")).
		Background(t.Bg).
		Padding(0, 1).
		MaxWidth(50)

	content := lipgloss.NewStyle().Foreground(t.Fg).Render("🐼 Panda Hint: " + diag.Message)
	overlay := style.Render(content)

	// Position it near the cursor
	tab := e.currentTab()
	lineInView := tab.CursorLine - tab.ScrollLine
	colInView := tab.CursorCol - tab.ScrollCol + e.lineNumberWidth + 3

	// Clamp positions
	x := colInView
	y := lineInView + 2 // Show 1 line below cursor

	if y+lipgloss.Height(overlay) >= e.height-2 {
		y = lineInView - lipgloss.Height(overlay) // Show above cursor if no space below
	}
	if x < 0 {
		x = 0
	}
	if x+lipgloss.Width(overlay) >= e.width-2 {
		x = e.width - lipgloss.Width(overlay) - 2
	}

	return placeOverlay(x, y, overlay, bg)
}

func (e *Editor) renderAutocompleteOverlay(bg string) string {
	t := theme.CurrentTheme
	tab := e.currentTab()

	var sb strings.Builder
	for i, s := range e.suggestions {
		style := lipgloss.NewStyle().Background(t.TitleBar).Foreground(t.Fg).Padding(0, 1).Width(20)
		if i == e.suggestionIdx {
			style = style.Background(t.Accent).Foreground(t.Bg).Bold(true)
		}
		sb.WriteString(style.Render(s))
		if i < len(e.suggestions)-1 {
			sb.WriteString("\n")
		}
	}

	overlay := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.Accent).
		Background(t.TitleBar).
		Render(sb.String())

	// Calculate cursor position
	x := e.lineNumberWidth + 2 + tab.CursorCol - tab.ScrollCol
	if e.fileTreeVisible {
		x += e.fileTree.Width
	}
	y := tab.CursorLine - tab.ScrollLine + 2 // +1 for tab bar, +1 for offset

	return placeOverlay(x, y, overlay, bg)
}

func (e *Editor) renderMinimap() string {
	t := theme.CurrentTheme
	tab := e.currentTab()
	buf := tab.Buf

	height := e.editorHeight()
	width := 4

	var sb strings.Builder
	totalLines := buf.LineCount()
	for i := 0; i < height; i++ {
		lineIdx := 0
		if height > 1 && totalLines > 0 {
			lineIdx = i * (totalLines - 1) / (height - 1)
		}
		if lineIdx >= totalLines {
			lineIdx = totalLines - 1
		}

		line := buf.GetLine(lineIdx)
		dots := " "
		if len(line) > 0 {
			dots = "░"
			if len(line) > 30 {
				dots = "▒"
			}
			if len(line) > 60 {
				dots = "▓"
			}
		}

		style := lipgloss.NewStyle().Foreground(t.Comment)
		if lineIdx >= tab.ScrollLine && lineIdx < tab.ScrollLine+height {
			style = style.Foreground(t.Accent)
		}
		sb.WriteString(style.Render(dots))
		if i < height-1 {
			sb.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(t.Border).
		Render(sb.String())
}

func (e *Editor) renderSettingsOverlay(bg string) string {
	t := theme.CurrentTheme

	// Build settings display lines
	var items []string

	items = append(items, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("── Editor ──"))
	items = append(items, fmt.Sprintf("  Tab Size:             %d", e.config.Editor.TabSize))
	items = append(items, fmt.Sprintf("  Relative Line Nums:  %v", e.relativeLineNo))
	items = append(items, fmt.Sprintf("  Auto Save Interval:  %d sec", e.config.Editor.AutoSaveInterval))
	minimap := "OFF"
	if e.config.Editor.Minimap != nil && *e.config.Editor.Minimap {
		minimap = "ON"
	}
	items = append(items, fmt.Sprintf("  Minimap:              %s", minimap))
	items = append(items, fmt.Sprintf("  Word Wrap:            %v", e.config.Editor.WordWrap))

	items = append(items, "")
	items = append(items, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("── Theme ──"))
	items = append(items, fmt.Sprintf("  Current Theme:  %s", e.config.Theme))

	lspEnabled := "YES"
	if !config.BoolVal(e.config.Behavior.LSPEnabled, true) {
		lspEnabled = "NO"
	}
	items = append(items, "")
	items = append(items, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("── Behavior ──"))
	items = append(items, fmt.Sprintf("  Terminal:       %s", e.config.Behavior.TerminalCmd))
	items = append(items, fmt.Sprintf("  LSP Enabled:    %s", lspEnabled))
	items = append(items, fmt.Sprintf("  Session Save:   %v", config.BoolVal(e.config.Behavior.SessionSave, true)))

	items = append(items, "")
	items = append(items, lipgloss.NewStyle().Foreground(t.Comment).Italic(true).Render("  Press 'o' to open config file"))
	items = append(items, lipgloss.NewStyle().Foreground(t.Comment).Italic(true).Render("  Press 'r' to reload config"))
	items = append(items, lipgloss.NewStyle().Foreground(t.Comment).Italic(true).Render("  Press esc to close"))

	// Render the overlay

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().
		Background(t.OverlayBg).
		Foreground(t.Fg).
		Width(50).
		Padding(1, 2).
		Render(strings.Join(items, "\n")))

	overlay := sb.String()

	oW := lipgloss.Width(overlay)
	oH := lipgloss.Height(overlay)
	return placeOverlay(e.width/2-oW/2, e.height/2-oH/2, overlay, bg)
}
