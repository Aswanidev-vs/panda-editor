package editor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Aswanidev-vs/panda-editor/internal/buffer"
	"github.com/Aswanidev-vs/panda-editor/internal/bundler"
	"github.com/Aswanidev-vs/panda-editor/internal/config"
	"github.com/Aswanidev-vs/panda-editor/internal/fuzzy"
	"github.com/Aswanidev-vs/panda-editor/internal/lsp"
	"github.com/Aswanidev-vs/panda-editor/internal/searcher"
	"github.com/Aswanidev-vs/panda-editor/internal/session"
	"github.com/Aswanidev-vs/panda-editor/internal/theme"
	"github.com/Aswanidev-vs/panda-editor/internal/watcher"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type ViewMode int

type tickMsg struct{}

const (
	ViewWelcome ViewMode = iota
	ViewNormal
	ViewFinder
	ViewSearch
	ViewSearchReplace
	ViewGoToLine
	ViewCommandPalette
	ViewSaveAs
	ViewHelp
	ViewFileTree
	ViewKeybindings
	ViewKeybindEdit
	ViewOpenFolder
	ViewGlobalSearch
	ViewUnsavedPrompt
	ViewFileTreeFilter
)

type ActionType int

const (
	ActionNone ActionType = iota
	ActionQuit
	ActionCloseTab
	ActionCloseSession
	ActionOpenFolder
	ActionDelete
)

type UndoState struct {
	Lines      []string
	CursorLine int
	CursorCol  int
}

type Tab struct {
	Buf           *buffer.Buffer
	CursorLine    int
	CursorCol     int
	ScrollLine    int
	ScrollCol     int
	TargetScrollL int
	TargetScrollC int
	SelectStart   int
	SelectEnd     int
	SelectActive  bool
	SelectStartL  int
	SelectStartC  int
	SearchResults []SearchResult
	UndoStack     []UndoState
	RedoStack     []UndoState
}

type SearchResult struct {
	Line  int
	Start int
	End   int
}

type DiagnosticMsg struct {
	Path        string
	Diagnostics []lsp.Diagnostic
}

type Editor struct {
	keys             KeyMap
	keyBindCfg       KeyBindConfig
	width            int
	height           int
	tabs             []Tab
	activeTab        int
	fileTree         FileTree
	fileTreeVisible  bool
	fileTreeRoot     string
	mode             ViewMode
	finderInput      textinput.Model
	finderResults    []fuzzy.Match
	finderCursor     int
	commandInput     textinput.Model
	commandResults   []Command
	commandCursor    int
	searchInput      textinput.Model
	replaceInput     textinput.Model
	searchQuery      string
	replaceQuery     string
	searchActive     bool
	gotoInput        textinput.Model
	saveAsInput      textinput.Model
	messages         []string
	messageTimer     int
	zoomLevel        int
	showHelp         bool
	lineNumberWidth  int
	statusHeight     int
	tabBarHeight     int
	fileList         []string
	clipboard        string
	isRegexSearch    bool
	caseSensitive    bool
	multiCursors     []Cursor
	kbEntries        []KeyBindEntry
	kbCursor         int
	kbScroll         int
	kbEditing        bool
	kbEditInput      textinput.Model
	kbEditIndex      int
	kbCategoryFilter string
	hasSession       bool
	sessionTimer     int
	welcomeCursor    int
	openFolderInput     textinput.Model
	fileTreeFilterInput textinput.Model
	fileWatcher      *watcher.Watcher
	fileChangeChan   chan string

	globalSearchInput   textinput.Model
	globalSearchResults []searcher.SearchResult
	searchResultChan    chan []searcher.SearchResult
	searchDoneChan      chan bool
	isSearching         bool
	globalSearchCursor  int
	finderScroll        int
	commandScroll       int
	globalSearchScroll  int
	helpScroll          int
	gitBranch           string

	lspClients      map[string]*lsp.Client
	lspDiagChan     chan DiagnosticMsg
	fileDiagnostics map[string][]lsp.Diagnostic

	gitBranchTimer int
	config         config.Config

	pendingAction ActionType
	postSaveAction ActionType
	unsavedTabIdx int
	pendingRename string
	pendingDelete string
	fileDiffs     map[string]map[int]string // [filepath][line_number]marker
	suggestions   []string
	suggestionIdx int
	splitActive   bool
	activePanel   int // 0: left, 1: right
	rightTab      int // index of tab in right panel
	bundleFormat  int // 0: Markdown, 1: XML, 2: Plain Text
}

type Cursor struct {
	Line int
	Col  int
}

func NewEditor() Editor {
	keyBindCfg := LoadKeyBindConfig()
	keys := KeyMapFromConfig(keyBindCfg)
	cfg, _ := config.LoadConfig()

	fi := textinput.New()
	fi.Placeholder = "Type to search files..."
	fi.CharLimit = 256
	fi.Width = 50

	ci := textinput.New()
	ci.Placeholder = "Type a command..."
	ci.CharLimit = 256
	ci.Width = 50

	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 256
	si.Width = 40

	ri := textinput.New()
	ri.Placeholder = "Replace with..."
	ri.CharLimit = 256
	ri.Width = 40

	gi := textinput.New()
	gi.Placeholder = "Line number..."
	gi.CharLimit = 16
	gi.Width = 20

	sai := textinput.New()
	sai.Placeholder = "Save as path..."
	sai.CharLimit = 512
	sai.Width = 50

	kbi := textinput.New()
	kbi.Placeholder = "Press new keybinding..."
	kbi.CharLimit = 64
	kbi.Width = 30

	ofi := textinput.New()
	ofi.Placeholder = "Folder path..."
	ofi.CharLimit = 512
	ofi.Width = 50

	gsi := textinput.New()
	gsi.Placeholder = "Search everywhere..."
	gsi.CharLimit = 256
	gsi.Width = 50

	ftfi := textinput.New()
	ftfi.Placeholder = "Filter tree..."
	ftfi.CharLimit = 128
	ftfi.Width = 30

	cwd, _ := os.Getwd()
	if cwd == "" {
		cwd = "."
	}

	hasSession := session.SessionExists()

	e := Editor{
		keys:              keys,
		keyBindCfg:        keyBindCfg,
		tabs:              []Tab{{Buf: buffer.New(), CursorLine: 0, CursorCol: 0, ScrollLine: 0, ScrollCol: 0, TargetScrollL: 0, TargetScrollC: 0}},
		activeTab:         0,
		fileTree:          NewFileTree(cwd),
		fileTreeVisible:   true,
		fileTreeRoot:      cwd,
		mode:              ViewWelcome,
		finderInput:       fi,
		commandInput:      ci,
		searchInput:       si,
		replaceInput:      ri,
		gotoInput:         gi,
		saveAsInput:       sai,
		kbEditInput:       kbi,
		openFolderInput:   ofi,
		zoomLevel:         0,
		lineNumberWidth:   5,
		statusHeight:      1,
		tabBarHeight:      1,
		hasSession:        hasSession,
		sessionTimer:      0,
		welcomeCursor:     0,
		fileChangeChan:    make(chan string, 10),
		globalSearchInput: gsi,
		fileTreeFilterInput: ftfi,
		searchResultChan:  make(chan []searcher.SearchResult, 10),
		searchDoneChan:    make(chan bool, 1),
		lspClients:        make(map[string]*lsp.Client),
		lspDiagChan:       make(chan DiagnosticMsg, 10),
		fileDiagnostics:   make(map[string][]lsp.Diagnostic),
		fileDiffs:         make(map[string]map[int]string),
		gitBranchTimer:    0,
		config:            cfg,
	}

	e.kbEntries = e.keyBindCfg.GetKeyBindEntries()
	e.buildFileList()

	// Try to start gopls if available
	_, err := exec.LookPath("gopls")
	if err == nil {
		client, err := lsp.NewClient("gopls", nil, func(method string, params json.RawMessage) {
			if method == "textDocument/publishDiagnostics" {
				var result struct {
					Uri         string           `json:"uri"`
					Diagnostics []lsp.Diagnostic `json:"diagnostics"`
				}
				if err := json.Unmarshal(params, &result); err == nil {
					// Convert file:/// path to absolute path
					path := strings.TrimPrefix(result.Uri, "file:///")
					if runtime.GOOS == "windows" {
						path = filepath.FromSlash(path)
					}
					e.lspDiagChan <- DiagnosticMsg{Path: path, Diagnostics: result.Diagnostics}
				}
			}
		})
		if err == nil {
			err = client.Initialize(e.fileTreeRoot)
			if err == nil {
				e.lspClients["go"] = client
			}
		}
	} else {
		// Just a log or simple message, we can't showMessage yet as TUI isn't running
		fmt.Printf("gopls not found, Go LSP disabled\n")
	}

	// Start background file watcher
	fw, err := watcher.New(func(path string) {
		e.fileChangeChan <- path
	})
	if err == nil {
		e.fileWatcher = fw
	}

	return e
}

func (e *Editor) buildFileList() {
	e.fileList = nil
	e.walkDir(e.fileTreeRoot, 0)
}

func (e *Editor) walkDir(dir string, depth int) {
	if depth > 10 {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
	for _, entry := range entries {
		name := entry.Name()
		if name == ".git" || name == "node_modules" || name == ".vscode" || name == "__pycache__" || name == ".idea" || name == "vendor" || name == ".next" || name == "dist" || name == "build" {
			continue
		}
		fullPath := filepath.Join(dir, name)
		if entry.IsDir() {
			e.walkDir(fullPath, depth+1)
		} else {
			e.fileList = append(e.fileList, fullPath)
		}
	}
}

func (e *Editor) currentTab() *Tab {
	if len(e.tabs) == 0 {
		return nil
	}
	idx := e.activeTab
	if e.splitActive && e.activePanel == 1 {
		idx = e.rightTab
	}
	if idx < 0 || idx >= len(e.tabs) {
		return &e.tabs[0]
	}
	return &e.tabs[idx]
}

func (e *Editor) currentBuf() *buffer.Buffer {
	tab := e.currentTab()
	if tab == nil {
		return nil
	}
	return tab.Buf
}

func (e *Editor) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		e.listenForFileChanges(),
		e.listenForSearchResults(),
		e.listenForSearchDone(),
		e.listenForDiagnostics(),
	)
}

func (e *Editor) listenForDiagnostics() tea.Cmd {
	return func() tea.Msg {
		msg := <-e.lspDiagChan
		return msg
	}
}

func (e *Editor) listenForSearchResults() tea.Cmd {
	return func() tea.Msg {
		results := <-e.searchResultChan
		return searcher.SearchProgressMsg{Results: results, Done: false}
	}
}

func (e *Editor) listenForSearchDone() tea.Cmd {
	return func() tea.Msg {
		<-e.searchDoneChan
		return searcher.SearchProgressMsg{Done: true}
	}
}

func (e *Editor) tickAnimation() tea.Cmd {
	return tea.Tick(time.Millisecond*16, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (e *Editor) animate() {
	tab := e.currentTab()
	// Smooth scroll line
	diffL := tab.TargetScrollL - tab.ScrollLine
	if diffL != 0 {
		step := diffL / 4
		if step == 0 {
			if diffL > 0 { step = 1 } else { step = -1 }
		}
		tab.ScrollLine += step
	}

	// Smooth scroll col
	diffC := tab.TargetScrollC - tab.ScrollCol
	if diffC != 0 {
		step := diffC / 4
		if step == 0 {
			if diffC > 0 { step = 1 } else { step = -1 }
		}
		tab.ScrollCol += step
	}
}

func (e *Editor) listenForFileChanges() tea.Cmd {
	return func() tea.Msg {
		path := <-e.fileChangeChan
		return watcher.FileChangeMsg{Path: path}
	}
}

func (e *Editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.width = msg.Width
		e.height = msg.Height
		return e, e.tickAnimation()

	case tickMsg:
		e.animate()
		return e, e.tickAnimation()

	case tea.MouseMsg:
		return e.handleMouse(msg)

	case watcher.FileChangeMsg:
		for i := range e.tabs {
			if e.tabs[i].Buf.FilePath == msg.Path && !e.tabs[i].Buf.Modified {
				newBuf, err := buffer.Open(msg.Path)
				if err == nil {
					e.tabs[i].Buf = newBuf
					e.showMessage("File updated externally: " + newBuf.Name)
				}
			}
		}
		return e, e.listenForFileChanges()

	case searcher.SearchProgressMsg:
		if msg.Done {
			e.isSearching = false
			return e, e.listenForSearchDone()
		}
		e.globalSearchResults = append(e.globalSearchResults, msg.Results...)
		return e, e.listenForSearchResults()

	case DiagnosticMsg:
		absPath, _ := filepath.Abs(msg.Path)
		e.fileDiagnostics[absPath] = msg.Diagnostics
		return e, e.listenForDiagnostics()
	}

	// Update git info periodically
	if e.gitBranchTimer <= 0 {
		e.gitBranch = e.getGitBranch()
		tab := e.currentTab()
		if tab != nil && tab.Buf != nil && tab.Buf.FilePath != "" {
			e.fileDiffs[tab.Buf.FilePath] = e.getGitDiffs(tab.Buf.FilePath)
		}
		e.gitBranchTimer = 500 // roughly every 5-10 seconds
	} else {
		e.gitBranchTimer--
	}

	switch e.mode {
	case ViewWelcome:
		return e.updateWelcome(msg)
	case ViewFinder:
		return e.updateFinder(msg)
	case ViewCommandPalette:
		return e.updateCommandPalette(msg)
	case ViewSearch, ViewSearchReplace:
		return e.updateSearch(msg)
	case ViewGoToLine:
		return e.updateGoToLine(msg)
	case ViewSaveAs:
		return e.updateSaveAs(msg)
	case ViewHelp:
		return e.updateHelp(msg)
	case ViewKeybindings:
		return e.updateKeybindings(msg)
	case ViewKeybindEdit:
		return e.updateKeybindEdit(msg)
	case ViewOpenFolder:
		return e.updateOpenFolder(msg)
	case ViewFileTree:
		return e.updateFileTree(msg)
	case ViewGlobalSearch:
		return e.updateGlobalSearch(msg)
	case ViewUnsavedPrompt:
		return e.updateUnsavedPrompt(msg)
	case ViewFileTreeFilter:
		return e.updateFileTreeFilter(msg)
	}

	result, cmd := e.updateNormal(msg)

	// Auto-save session every ~50 key events (debounced)
	e.sessionTimer++
	if e.sessionTimer >= 50 {
		e.sessionTimer = 0
		e.saveSession()
	}

	return result, cmd
}

func (e *Editor) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseWheelUp:
		e.scrollUp(3)
	case tea.MouseWheelDown:
		e.scrollDown(3)
	}
	return e, nil
}

func (e *Editor) updateNormal(msg tea.Msg) (tea.Model, tea.Cmd) {
	tab := e.currentTab()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, e.keys.Quit):
			for i, t := range e.tabs {
				if t.Buf.Modified {
					e.activeTab = i
					e.mode = ViewUnsavedPrompt
					e.pendingAction = ActionQuit
					e.unsavedTabIdx = i
					return e, nil
				}
			}
			e.saveSession()
			return e, tea.Quit

		case key.Matches(msg, e.keys.ForceQuit):
			e.saveSession()
			return e, tea.Quit

		case key.Matches(msg, e.keys.ShowKeybindings):
			e.mode = ViewKeybindings
			e.kbCursor = 0
			e.kbScroll = 0
			e.kbEditing = false
			return e, nil

		case key.Matches(msg, e.keys.ShowHelp) || msg.String() == "f1" || msg.String() == "alt+h":
			e.mode = ViewHelp
			e.showHelp = true
			e.helpScroll = 0
			return e, nil

		case key.Matches(msg, e.keys.Save):
			return e.handleSave()

		case key.Matches(msg, e.keys.SaveAs):
			e.mode = ViewSaveAs
			e.saveAsInput.SetValue("")
			e.saveAsInput.Focus()
			return e, textinput.Blink

		case key.Matches(msg, e.keys.OpenFinder):
			e.mode = ViewFinder
			e.finderInput.SetValue("")
			e.finderInput.Focus()
			e.updateFinderResults("")
			return e, textinput.Blink

		case key.Matches(msg, e.keys.OpenExplorer):
			e.fileTreeVisible = !e.fileTreeVisible
			if e.fileTreeVisible {
				e.mode = ViewFileTree
			} else {
				e.mode = ViewNormal
			}
			return e, nil

		case key.Matches(msg, e.keys.CommandPalette):
			e.mode = ViewCommandPalette
			e.commandInput.SetValue("")
			e.commandInput.Focus()
			e.updateCommandResults("")
			return e, textinput.Blink

		case key.Matches(msg, e.keys.Search):
			e.mode = ViewSearch
			e.searchInput.SetValue("")
			e.searchInput.Focus()
			return e, textinput.Blink

		case key.Matches(msg, e.keys.SearchReplace):
			e.mode = ViewSearchReplace
			e.searchInput.SetValue("")
			e.replaceInput.SetValue("")
			e.searchInput.Focus()
			return e, textinput.Blink

		case key.Matches(msg, e.keys.GlobalSearch):
			e.mode = ViewGlobalSearch
			e.globalSearchInput.SetValue("")
			e.globalSearchInput.Focus()
			e.globalSearchResults = nil
			e.globalSearchCursor = 0
			return e, textinput.Blink

		case key.Matches(msg, e.keys.GoToLine):
			e.mode = ViewGoToLine
			e.gotoInput.SetValue("")
			e.gotoInput.Focus()
			return e, textinput.Blink

		case key.Matches(msg, e.keys.NewTab):
			e.tabs = append(e.tabs, Tab{Buf: buffer.New(), CursorLine: 0, CursorCol: 0, ScrollLine: 0, ScrollCol: 0})
			e.activeTab = len(e.tabs) - 1
			return e, nil

		case key.Matches(msg, e.keys.CloseTab):
			return e.handleCloseTab()

		case key.Matches(msg, e.keys.NextTab):
			if e.activeTab < len(e.tabs)-1 {
				e.activeTab++
			} else {
				e.activeTab = 0
			}
			return e, nil

		case key.Matches(msg, e.keys.PrevTab):
			if e.activeTab > 0 {
				e.activeTab--
			} else {
				e.activeTab = len(e.tabs) - 1
			}
			return e, nil

		case key.Matches(msg, e.keys.ToggleTheme):
			theme.ToggleTheme()
			return e, nil

		case key.Matches(msg, e.keys.ToggleSidebar):
			e.fileTreeVisible = !e.fileTreeVisible
			return e, nil

		case key.Matches(msg, e.keys.ToggleTerminal):
			cmdStr := e.keyBindCfg.TerminalCmd
			if cmdStr == "" {
				if runtime.GOOS == "windows" {
					cmdStr = "cmd"
				} else {
					cmdStr = "bash"
				}
			}
			c := exec.Command(cmdStr)
			return e, tea.ExecProcess(c, func(err error) tea.Msg {
				return nil
			})

		case key.Matches(msg, e.keys.ZoomIn):
			e.zoomLevel++
			return e, nil

		case key.Matches(msg, e.keys.ZoomOut):
			if e.zoomLevel > -3 {
				e.zoomLevel--
			}
			return e, nil

		case key.Matches(msg, e.keys.CenterCursor):
			e.centerCursor()
			return e, nil

		case key.Matches(msg, e.keys.SearchNext):
			e.searchNext()
			return e, nil

		case key.Matches(msg, e.keys.SearchPrev):
			e.searchPrev()
			return e, nil

		case key.Matches(msg, e.keys.SelectAll):
			tab.SelectActive = true
			tab.SelectStartL = 0
			tab.SelectStartC = 0
			tab.CursorLine = e.currentBuf().LineCount() - 1
			tab.CursorCol = e.currentBuf().LineLen(tab.CursorLine)
			return e, nil

		case key.Matches(msg, e.keys.SelectLine):
			tab.SelectActive = true
			tab.SelectStartL = tab.CursorLine
			tab.SelectStartC = 0
			tab.CursorCol = e.currentBuf().LineLen(tab.CursorLine)
			return e, nil

		case key.Matches(msg, e.keys.SelectUp):
			if !tab.SelectActive {
				tab.SelectActive = true
				tab.SelectStartL = tab.CursorLine
				tab.SelectStartC = tab.CursorCol
			}
			e.moveCursorUp()
			return e, nil

		case key.Matches(msg, e.keys.SelectDown):
			if !tab.SelectActive {
				tab.SelectActive = true
				tab.SelectStartL = tab.CursorLine
				tab.SelectStartC = tab.CursorCol
			}
			e.moveCursorDown()
			return e, nil

		case key.Matches(msg, e.keys.SelectLeft):
			if !tab.SelectActive {
				tab.SelectActive = true
				tab.SelectStartL = tab.CursorLine
				tab.SelectStartC = tab.CursorCol
			}
			e.moveCursorLeft()
			return e, nil

		case key.Matches(msg, e.keys.SelectRight):
			if !tab.SelectActive {
				tab.SelectActive = true
				tab.SelectStartL = tab.CursorLine
				tab.SelectStartC = tab.CursorCol
			}
			e.moveCursorRight()
			return e, nil

		case key.Matches(msg, e.keys.SelectWordLeft):
			if !tab.SelectActive {
				tab.SelectActive = true
				tab.SelectStartL = tab.CursorLine
				tab.SelectStartC = tab.CursorCol
			}
			e.moveWordLeft()
			return e, nil

		case key.Matches(msg, e.keys.SelectWordRight):
			if !tab.SelectActive {
				tab.SelectActive = true
				tab.SelectStartL = tab.CursorLine
				tab.SelectStartC = tab.CursorCol
			}
			e.moveWordRight()
			return e, nil

		case key.Matches(msg, e.keys.SelectLineStart):
			if !tab.SelectActive {
				tab.SelectActive = true
				tab.SelectStartL = tab.CursorLine
				tab.SelectStartC = tab.CursorCol
			}
			tab.CursorCol = 0
			return e, nil

		case key.Matches(msg, e.keys.SelectLineEnd):
			if !tab.SelectActive {
				tab.SelectActive = true
				tab.SelectStartL = tab.CursorLine
				tab.SelectStartC = tab.CursorCol
			}
			tab.CursorCol = e.currentBuf().LineLen(tab.CursorLine)
			return e, nil

		case key.Matches(msg, e.keys.Copy):
			return e.handleCopy()

		case key.Matches(msg, e.keys.CopyBundle):
			e.handleBundle()
			return e, nil

		case key.Matches(msg, e.keys.Cut):
			return e.handleCut()

		case key.Matches(msg, e.keys.Paste):
			return e.handlePaste()

		case key.Matches(msg, e.keys.CursorUp):
			if len(e.suggestions) > 0 {
				e.suggestionIdx--
				if e.suggestionIdx < 0 {
					e.suggestionIdx = len(e.suggestions) - 1
				}
				return e, nil
			}
			tab.SelectActive = false
			e.moveCursorUp()
			e.suggestions = nil
			return e, nil

		case key.Matches(msg, e.keys.CursorDown):
			if len(e.suggestions) > 0 {
				e.suggestionIdx++
				if e.suggestionIdx >= len(e.suggestions) {
					e.suggestionIdx = 0
				}
				return e, nil
			}
			tab.SelectActive = false
			e.moveCursorDown()
			e.suggestions = nil
			return e, nil

		case key.Matches(msg, e.keys.CursorLeft):
			tab.SelectActive = false
			e.moveCursorLeft()
			return e, nil

		case key.Matches(msg, e.keys.CursorRight):
			tab.SelectActive = false
			e.moveCursorRight()
			return e, nil

		case key.Matches(msg, e.keys.WordLeft):
			tab.SelectActive = false
			e.moveWordLeft()
			return e, nil

		case key.Matches(msg, e.keys.WordRight):
			tab.SelectActive = false
			e.moveWordRight()
			return e, nil

		case key.Matches(msg, e.keys.LineStart):
			tab.SelectActive = false
			tab.CursorCol = 0
			return e, nil

		case key.Matches(msg, e.keys.LineEnd):
			tab.SelectActive = false
			tab.CursorCol = e.currentBuf().LineLen(tab.CursorLine)
			return e, nil

		case key.Matches(msg, e.keys.FileStart):
			tab.SelectActive = false
			tab.CursorLine = 0
			tab.CursorCol = 0
			tab.ScrollLine = 0
			return e, nil

		case key.Matches(msg, e.keys.FileEnd):
			tab.SelectActive = false
			tab.CursorLine = e.currentBuf().LineCount() - 1
			tab.CursorCol = 0
			return e, nil

		case key.Matches(msg, e.keys.PageUp):
			tab.SelectActive = false
			pageSize := e.editorHeight()
			for i := 0; i < pageSize; i++ {
				e.moveCursorUp()
			}
			return e, nil

		case key.Matches(msg, e.keys.PageDown):
			tab.SelectActive = false
			pageSize := e.editorHeight()
			for i := 0; i < pageSize; i++ {
				e.moveCursorDown()
			}
			return e, nil

		case key.Matches(msg, e.keys.ScrollUp):
			e.scrollUp(1)
			return e, nil

		case key.Matches(msg, e.keys.ScrollDown):
			e.scrollDown(1)
			return e, nil

		case key.Matches(msg, e.keys.Undo):
			e.handleUndo()
			return e, nil

		case key.Matches(msg, e.keys.Redo):
			e.handleRedo()
			return e, nil

		case key.Matches(msg, e.keys.InsertNewline):
			if len(e.suggestions) > 0 {
				e.acceptSuggestion(e.suggestions[e.suggestionIdx])
				return e, nil
			}
			e.pushUndo()
			tab.SelectActive = false
			tab.CursorLine, tab.CursorCol = e.currentBuf().InsertNewline(tab.CursorLine, tab.CursorCol)
			e.ensureCursorVisible()
			e.suggestions = nil
			return e, nil

		case key.Matches(msg, e.keys.Backspace):
			e.pushUndo()
			tab.SelectActive = false
			tab.CursorLine, tab.CursorCol = e.currentBuf().Backspace(tab.CursorLine, tab.CursorCol)
			e.ensureCursorVisible()
			return e, nil

		case key.Matches(msg, e.keys.DeleteWordLeft):
			e.pushUndo()
			tab.SelectActive = false
			tab.CursorLine, tab.CursorCol = e.currentBuf().DeleteWordLeft(tab.CursorLine, tab.CursorCol)
			e.ensureCursorVisible()
			return e, nil

		case key.Matches(msg, e.keys.Delete):
			e.pushUndo()
			tab.SelectActive = false
			tab.CursorLine, tab.CursorCol = e.currentBuf().Delete(tab.CursorLine, tab.CursorCol)
			return e, nil

		case key.Matches(msg, e.keys.DeleteLine):
			e.pushUndo()
			e.currentBuf().DeleteLine(tab.CursorLine)
			if tab.CursorLine >= e.currentBuf().LineCount() {
				tab.CursorLine = e.currentBuf().LineCount() - 1
			}
			tab.CursorCol = 0
			return e, nil

		case key.Matches(msg, e.keys.DuplicateLine):
			e.pushUndo()
			e.currentBuf().DuplicateLine(tab.CursorLine)
			tab.CursorLine++
			return e, nil

		case key.Matches(msg, e.keys.MoveLineUp):
			e.pushUndo()
			tab.CursorLine = e.currentBuf().MoveLineUp(tab.CursorLine)
			e.ensureCursorVisible()
			return e, nil

		case key.Matches(msg, e.keys.MoveLineDown):
			e.pushUndo()
			tab.CursorLine = e.currentBuf().MoveLineDown(tab.CursorLine)
			e.ensureCursorVisible()
			return e, nil

		case key.Matches(msg, e.keys.IndentLine):
			if len(e.suggestions) > 0 {
				e.acceptSuggestion(e.suggestions[e.suggestionIdx])
				return e, nil
			}
			e.pushUndo()
			e.handleIndent()
			return e, nil

		case key.Matches(msg, e.keys.UnindentLine):
			e.pushUndo()
			e.handleUnindent()
			return e, nil

		case key.Matches(msg, e.keys.ToggleComment):
			e.pushUndo()
			e.handleToggleComment()
			return e, nil

		default:
			if msg.Type == tea.KeyRunes {
				if msg.String() == " " {
					e.pushUndo()
				}
				for _, r := range msg.Runes {
					tab.CursorLine, tab.CursorCol = e.currentBuf().InsertChar(tab.CursorLine, tab.CursorCol, string(r))
				}
				e.ensureCursorVisible()
				e.updateSuggestions()
				return e, nil
			}
		}
	}

	return e, nil
}

func (e *Editor) pushUndo() {
	tab := e.currentTab()
	snapshot := make([]string, len(tab.Buf.Lines))
	copy(snapshot, tab.Buf.Lines)

	state := UndoState{
		Lines:      snapshot,
		CursorLine: tab.CursorLine,
		CursorCol:  tab.CursorCol,
	}

	tab.UndoStack = append(tab.UndoStack, state)
	if len(tab.UndoStack) > 100 {
		tab.UndoStack = tab.UndoStack[1:]
	}
	tab.RedoStack = nil
}

func (e *Editor) handleUndo() {
	tab := e.currentTab()
	if len(tab.UndoStack) == 0 {
		return
	}

	// Save current state to redo stack
	currentState := UndoState{
		Lines:      make([]string, len(tab.Buf.Lines)),
		CursorLine: tab.CursorLine,
		CursorCol:  tab.CursorCol,
	}
	copy(currentState.Lines, tab.Buf.Lines)
	tab.RedoStack = append(tab.RedoStack, currentState)

	// Pop undo stack
	prevState := tab.UndoStack[len(tab.UndoStack)-1]
	tab.UndoStack = tab.UndoStack[:len(tab.UndoStack)-1]

	// Restore
	tab.Buf.Lines = make([]string, len(prevState.Lines))
	copy(tab.Buf.Lines, prevState.Lines)
	tab.CursorLine = prevState.CursorLine
	tab.CursorCol = prevState.CursorCol
	e.ensureCursorVisible()
}

func (e *Editor) handleRedo() {
	tab := e.currentTab()
	if len(tab.RedoStack) == 0 {
		return
	}

	// Save current state to undo stack
	currentState := UndoState{
		Lines:      make([]string, len(tab.Buf.Lines)),
		CursorLine: tab.CursorLine,
		CursorCol:  tab.CursorCol,
	}
	copy(currentState.Lines, tab.Buf.Lines)
	tab.UndoStack = append(tab.UndoStack, currentState)

	// Pop redo stack
	nextState := tab.RedoStack[len(tab.RedoStack)-1]
	tab.RedoStack = tab.RedoStack[:len(tab.RedoStack)-1]

	// Restore
	tab.Buf.Lines = make([]string, len(nextState.Lines))
	copy(tab.Buf.Lines, nextState.Lines)
	tab.CursorLine = nextState.CursorLine
	tab.CursorCol = nextState.CursorCol
	e.ensureCursorVisible()
}

func (e *Editor) moveCursorUp() {
	tab := e.currentTab()
	if tab.CursorLine > 0 {
		tab.CursorLine--
		maxCol := e.currentBuf().LineLen(tab.CursorLine)
		if tab.CursorCol > maxCol {
			tab.CursorCol = maxCol
		}
	}
	e.ensureCursorVisible()
}

func (e *Editor) moveCursorDown() {
	tab := e.currentTab()
	if tab.CursorLine < e.currentBuf().LineCount()-1 {
		tab.CursorLine++
		maxCol := e.currentBuf().LineLen(tab.CursorLine)
		if tab.CursorCol > maxCol {
			tab.CursorCol = maxCol
		}
	}
	e.ensureCursorVisible()
}

func (e *Editor) moveCursorLeft() {
	tab := e.currentTab()
	if tab.CursorCol > 0 {
		tab.CursorCol--
	} else if tab.CursorLine > 0 {
		tab.CursorLine--
		tab.CursorCol = e.currentBuf().LineLen(tab.CursorLine)
	}
	e.ensureCursorVisible()
}

func (e *Editor) moveCursorRight() {
	tab := e.currentTab()
	lineLen := e.currentBuf().LineLen(tab.CursorLine)
	if tab.CursorCol < lineLen {
		tab.CursorCol++
	} else if tab.CursorLine < e.currentBuf().LineCount()-1 {
		tab.CursorLine++
		tab.CursorCol = 0
	}
	e.ensureCursorVisible()
}

func (e *Editor) moveWordLeft() {
	tab := e.currentTab()
	if tab.CursorCol == 0 {
		if tab.CursorLine > 0 {
			tab.CursorLine--
			tab.CursorCol = e.currentBuf().LineLen(tab.CursorLine)
		}
		e.ensureCursorVisible()
		return
	}
	runes := []rune(e.currentBuf().GetLine(tab.CursorLine))
	col := tab.CursorCol
	if col > len(runes) {
		col = len(runes)
	}
	for col > 0 && unicode.IsSpace(runes[col-1]) {
		col--
	}
	for col > 0 && !unicode.IsSpace(runes[col-1]) {
		col--
	}
	tab.CursorCol = col
	e.ensureCursorVisible()
}

func (e *Editor) moveWordRight() {
	tab := e.currentTab()
	runes := []rune(e.currentBuf().GetLine(tab.CursorLine))
	col := tab.CursorCol
	if col >= len(runes) {
		if tab.CursorLine < e.currentBuf().LineCount()-1 {
			tab.CursorLine++
			tab.CursorCol = 0
		}
		e.ensureCursorVisible()
		return
	}
	for col < len(runes) && !unicode.IsSpace(runes[col]) {
		col++
	}
	for col < len(runes) && unicode.IsSpace(runes[col]) {
		col++
	}
	tab.CursorCol = col
	e.ensureCursorVisible()
}

func (e *Editor) ensureCursorVisible() {
	tab := e.currentTab()
	h := e.editorHeight()

	if tab.CursorLine < tab.TargetScrollL {
		tab.TargetScrollL = tab.CursorLine
	}
	if tab.CursorLine >= tab.TargetScrollL+h {
		tab.TargetScrollL = tab.CursorLine - h + 1
	}
	if tab.TargetScrollL < 0 {
		tab.TargetScrollL = 0
	}

	lnWidth := e.lineNumberWidth + 1
	editorWidth := e.editorWidth()
	visibleCols := editorWidth - lnWidth - 3
	if visibleCols < 10 {
		visibleCols = 10
	}

	if tab.CursorCol < tab.TargetScrollC {
		tab.TargetScrollC = tab.CursorCol
	}
	if tab.CursorCol >= tab.TargetScrollC+visibleCols {
		tab.TargetScrollC = tab.CursorCol - visibleCols + 5
	}
	if tab.TargetScrollC < 0 {
		tab.TargetScrollC = 0
	}
}

func (e *Editor) scrollUp(n int) {
	tab := e.currentTab()
	tab.TargetScrollL -= n
	if tab.TargetScrollL < 0 {
		tab.TargetScrollL = 0
	}
}

func (e *Editor) scrollDown(n int) {
	tab := e.currentTab()
	maxScroll := e.currentBuf().LineCount() - e.editorHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}
	tab.TargetScrollL += n
	if tab.TargetScrollL > maxScroll {
		tab.TargetScrollL = maxScroll
	}
}

func (e *Editor) centerCursor() {
	tab := e.currentTab()
	h := e.editorHeight()
	tab.TargetScrollL = tab.CursorLine - h/2
	if tab.TargetScrollL < 0 {
		tab.TargetScrollL = 0
	}
}

func (e *Editor) editorWidth() int {
	w := e.width
	if e.fileTreeVisible {
		w -= (e.fileTree.Width + 1)
	}
	return w
}

func (e *Editor) editorHeight() int {
	return e.height - e.tabBarHeight - e.statusHeight
}

func (e *Editor) handleSave() (tea.Model, tea.Cmd) {
	buf := e.currentBuf()
	if buf.FilePath == "" {
		e.mode = ViewSaveAs
		e.saveAsInput.SetValue("")
		e.saveAsInput.Focus()
		return e, textinput.Blink
	}
	if err := buf.Save(); err != nil {
		e.showMessage(fmt.Sprintf("Error: %v", err))
	} else {
		e.showMessage("Saved: " + buf.Name)
		if e.fileWatcher != nil {
			_ = e.fileWatcher.Watch(buf.FilePath)
		}
	}
	return e, nil
}

func (e *Editor) handleCloseTab() (tea.Model, tea.Cmd) {
	if e.currentBuf().Modified {
		e.mode = ViewUnsavedPrompt
		e.pendingAction = ActionCloseTab
		e.unsavedTabIdx = e.activeTab
		return e, nil
	}
	
	if len(e.tabs) == 1 {
		e.mode = ViewWelcome
		e.tabs = nil
		return e, nil
	}
	
	buf := e.currentBuf()
	if e.fileWatcher != nil && buf.FilePath != "" {
		e.fileWatcher.Unwatch(buf.FilePath)
	}
	e.tabs = append(e.tabs[:e.activeTab], e.tabs[e.activeTab+1:]...)
	if e.activeTab >= len(e.tabs) {
		e.activeTab = len(e.tabs) - 1
	}
	return e, nil
}

func (e *Editor) handleCopy() (tea.Model, tea.Cmd) {
	tab := e.currentTab()
	if tab.SelectActive {
		e.clipboard = e.currentBuf().GetSelectionText(
			tab.SelectStartL, tab.SelectStartC,
			tab.CursorLine, tab.CursorCol,
		)
		tab.SelectActive = false
		e.showMessage("Copied to clipboard")
	} else {
		e.clipboard = e.currentBuf().GetLine(tab.CursorLine)
		e.showMessage("Line copied")
	}
	_ = clipboard.WriteAll(e.clipboard)
	return e, nil
}

func (e *Editor) handleCut() (tea.Model, tea.Cmd) {
	e.pushUndo()
	tab := e.currentTab()
	if tab.SelectActive {
		e.clipboard = e.currentBuf().GetSelectionText(
			tab.SelectStartL, tab.SelectStartC,
			tab.CursorLine, tab.CursorCol,
		)
		// Delete selected text
		e.deleteSelection()
		tab.SelectActive = false
		_ = clipboard.WriteAll(e.clipboard)
		e.showMessage("Cut to clipboard")
	} else {
		e.clipboard = e.currentBuf().GetLine(tab.CursorLine)
		e.currentBuf().DeleteLine(tab.CursorLine)
		if tab.CursorLine >= e.currentBuf().LineCount() {
			tab.CursorLine = e.currentBuf().LineCount() - 1
		}
		tab.CursorCol = 0
		_ = clipboard.WriteAll(e.clipboard)
		e.showMessage("Line cut")
	}
	return e, nil
}

func (e *Editor) deleteSelection() {
	tab := e.currentTab()
	buf := e.currentBuf()
	sl, sc, el, ec := tab.SelectStartL, tab.SelectStartC, tab.CursorLine, tab.CursorCol
	if sl > el || (sl == el && sc > ec) {
		sl, sc, el, ec = el, ec, sl, sc
	}
	if sl == el {
		runes := []rune(buf.Lines[sl])
		if sc > len(runes) {
			sc = len(runes)
		}
		if ec > len(runes) {
			ec = len(runes)
		}
		buf.Lines[sl] = string(runes[:sc]) + string(runes[ec:])
		tab.CursorLine = sl
		tab.CursorCol = sc
	} else {
		startRunes := []rune(buf.Lines[sl])
		endRunes := []rune(buf.Lines[el])
		if sc > len(startRunes) {
			sc = len(startRunes)
		}
		if ec > len(endRunes) {
			ec = len(endRunes)
		}
		newLine := string(startRunes[:sc]) + string(endRunes[ec:])
		newLines := make([]string, 0, len(buf.Lines)-(el-sl))
		newLines = append(newLines, buf.Lines[:sl]...)
		newLines = append(newLines, newLine)
		newLines = append(newLines, buf.Lines[el+1:]...)
		buf.Lines = newLines
		tab.CursorLine = sl
		tab.CursorCol = sc
	}
	buf.Modified = true
}

func (e *Editor) handlePaste() (tea.Model, tea.Cmd) {
	e.pushUndo()
	tab := e.currentTab()
	buf := e.currentBuf()

	// Try OS clipboard first, fall back to internal
	pasteText := e.clipboard
	if osClip, err := clipboard.ReadAll(); err == nil && osClip != "" {
		pasteText = osClip
	}
	if pasteText == "" {
		return e, nil
	}
	lines := strings.Split(pasteText, "\n")
	for i, line := range lines {
		if i == 0 {
			tab.CursorLine, tab.CursorCol = buf.InsertChar(tab.CursorLine, tab.CursorCol, line)
		} else {
			tab.CursorLine, tab.CursorCol = buf.InsertNewline(tab.CursorLine, tab.CursorCol)
			tab.CursorLine, tab.CursorCol = buf.InsertChar(tab.CursorLine, tab.CursorCol, line)
		}
	}
	e.ensureCursorVisible()
	e.showMessage("Pasted")
	return e, nil
}

func (e *Editor) handleIndent() {
	tab := e.currentTab()
	buf := e.currentBuf()
	if tab.SelectActive {
		sl, el := tab.SelectStartL, tab.CursorLine
		if sl > el {
			sl, el = el, sl
		}
		for i := sl; i <= el; i++ {
			buf.SetLine(i, "\t"+buf.GetLine(i))
		}
	} else {
		line := buf.GetLine(tab.CursorLine)
		buf.SetLine(tab.CursorLine, "\t"+line)
		tab.CursorCol++
	}
}

func (e *Editor) handleUnindent() {
	tab := e.currentTab()
	buf := e.currentBuf()
	if tab.SelectActive {
		sl, el := tab.SelectStartL, tab.CursorLine
		if sl > el {
			sl, el = el, sl
		}
		for i := sl; i <= el; i++ {
			line := buf.GetLine(i)
			if len(line) > 0 && line[0] == '\t' {
				buf.SetLine(i, line[1:])
			} else if len(line) >= 4 && line[:4] == "    " {
				buf.SetLine(i, line[4:])
			}
		}
	} else {
		line := buf.GetLine(tab.CursorLine)
		if len(line) > 0 && line[0] == '\t' {
			buf.SetLine(tab.CursorLine, line[1:])
			if tab.CursorCol > 0 {
				tab.CursorCol--
			}
		} else if len(line) >= 4 && line[:4] == "    " {
			buf.SetLine(tab.CursorLine, line[4:])
			if tab.CursorCol >= 4 {
				tab.CursorCol -= 4
			} else {
				tab.CursorCol = 0
			}
		}
	}
}

func (e *Editor) handleToggleComment() {
	tab := e.currentTab()
	buf := e.currentBuf()
	lang := buf.Language
	prefix := "//"
	switch lang {
	case "python", "ruby", "shell", "yaml", "toml", "ini":
		prefix = "#"
	case "lua":
		prefix = "--"
	case "sql":
		prefix = "--"
	}

	if tab.SelectActive {
		sl, el := tab.SelectStartL, tab.CursorLine
		if sl > el {
			sl, el = el, sl
		}
		allCommented := true
		for i := sl; i <= el; i++ {
			line := strings.TrimLeft(buf.GetLine(i), " \t")
			if !strings.HasPrefix(line, prefix) {
				allCommented = false
				break
			}
		}
		for i := sl; i <= el; i++ {
			line := buf.GetLine(i)
			if allCommented {
				idx := strings.Index(line, prefix)
				if idx >= 0 {
					buf.SetLine(i, line[:idx]+line[idx+len(prefix):])
				}
			} else {
				buf.SetLine(i, prefix+" "+line)
			}
		}
	} else {
		line := buf.GetLine(tab.CursorLine)
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, prefix) {
			idx := strings.Index(line, prefix)
			if idx >= 0 {
				after := line[idx+len(prefix):]
				if len(after) > 0 && after[0] == ' ' {
					after = after[1:]
				}
				buf.SetLine(tab.CursorLine, line[:idx]+after)
			}
		} else {
			buf.SetLine(tab.CursorLine, prefix+" "+line)
		}
	}
}

func (e *Editor) searchNext() {
	if e.searchQuery == "" {
		return
	}
	tab := e.currentTab()
	buf := e.currentBuf()
	q := e.searchQuery
	if !e.caseSensitive {
		q = strings.ToLower(q)
	}

	startLine := tab.CursorLine
	startCol := tab.CursorCol + 1

	for i := 0; i < buf.LineCount(); i++ {
		line := (startLine + i) % buf.LineCount()
		searchLine := buf.GetLine(line)
		if !e.caseSensitive {
			searchLine = strings.ToLower(searchLine)
		}

		col := 0
		if i == 0 {
			col = startCol
		}
		if col > len(searchLine) {
			continue
		}
		if idx := strings.Index(searchLine[col:], q); idx >= 0 {
			tab.CursorLine = line
			tab.CursorCol = col + idx
			e.ensureCursorVisible()
			return
		}
	}
	e.showMessage("No more matches")
}

func (e *Editor) searchPrev() {
	if e.searchQuery == "" {
		return
	}
	tab := e.currentTab()
	buf := e.currentBuf()
	q := e.searchQuery
	if !e.caseSensitive {
		q = strings.ToLower(q)
	}

	startLine := tab.CursorLine
	startCol := tab.CursorCol - 1

	for i := 0; i < buf.LineCount(); i++ {
		line := (startLine - i + buf.LineCount()) % buf.LineCount()
		searchLine := buf.GetLine(line)
		if !e.caseSensitive {
			searchLine = strings.ToLower(searchLine)
		}

		endCol := len(searchLine)
		if i == 0 && startCol >= 0 {
			endCol = startCol
		}
		if idx := strings.LastIndex(searchLine[:endCol], q); idx >= 0 {
			tab.CursorLine = line
			tab.CursorCol = idx
			e.ensureCursorVisible()
			return
		}
	}
	e.showMessage("No more matches")
}

func (e *Editor) showMessage(msg string) {
	e.messages = append(e.messages, msg)
	if len(e.messages) > 5 {
		e.messages = e.messages[1:]
	}
	e.messageTimer = 150
}

func (e *Editor) openFile(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		e.showMessage(fmt.Sprintf("Error: %v", err))
		return
	}

	for i, t := range e.tabs {
		if t.Buf.FilePath == absPath {
			e.activeTab = i
			return
		}
	}

	buf, err := buffer.Open(absPath)
	if err != nil {
		e.showMessage(fmt.Sprintf("Error: %v", err))
		return
	}

	e.tabs = append(e.tabs, Tab{
		Buf:        buf,
		CursorLine: 0,
		CursorCol:  0,
		ScrollLine: 0,
		ScrollCol:  0,
	})
	e.activeTab = len(e.tabs) - 1

	if e.fileWatcher != nil {
		_ = e.fileWatcher.Watch(absPath)
	}

	// Notify LSP
	if client, ok := e.lspClients[buf.Language]; ok {
		_ = client.DidOpen(absPath, buf.Language, strings.Join(buf.Lines, "\n"))
	}
}

func (e *Editor) updateFinderResults(query string) {
	e.finderResults = fuzzy.Search(query, e.fileList)
	e.finderCursor = 0
}

func (e *Editor) updateCommandResults(query string) {
	if query == "" {
		e.commandResults = DefaultCommands()
		return
	}
	var filtered []Command
	lq := strings.ToLower(query)
	for _, cmd := range DefaultCommands() {
		if strings.Contains(strings.ToLower(cmd.Name), lq) ||
			strings.Contains(strings.ToLower(cmd.Description), lq) {
			filtered = append(filtered, cmd)
		}
	}
	e.commandResults = filtered
	e.commandCursor = 0
}

func (e *Editor) updateFinder(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			e.mode = ViewNormal
			e.finderInput.Blur()
			return e, nil
		case "enter":
			if e.finderCursor >= 0 && e.finderCursor < len(e.finderResults) {
				e.openFile(e.finderResults[e.finderCursor].Path)
			}
			e.mode = ViewNormal
			e.finderInput.Blur()
			return e, nil
		case "up", "ctrl+k":
			if e.finderCursor > 0 {
				e.finderCursor--
			}
			return e, nil
		case "down", "ctrl+j":
			if e.finderCursor < len(e.finderResults)-1 {
				e.finderCursor++
			}
			return e, nil
		case "ctrl+n":
			if e.finderCursor < len(e.finderResults)-1 {
				e.finderCursor++
			}
			return e, nil
		case "ctrl+p":
			if e.finderCursor > 0 {
				e.finderCursor--
			}
			return e, nil
		}
	}

	var cmd tea.Cmd
	oldVal := e.finderInput.Value()
	e.finderInput, cmd = e.finderInput.Update(msg)
	if e.finderInput.Value() != oldVal {
		e.updateFinderResults(e.finderInput.Value())
	}
	return e, cmd
}

func (e *Editor) updateCommandPalette(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			e.mode = ViewNormal
			e.commandInput.Blur()
			return e, nil
		case "enter":
			if e.commandCursor >= 0 && e.commandCursor < len(e.commandResults) {
				model, cmd := e.executeCommand(e.commandResults[e.commandCursor].Action)
				e.mode = ViewNormal
				e.commandInput.Blur()
				return model, cmd
			}
			e.mode = ViewNormal
			e.commandInput.Blur()
			return e, nil
		case "up", "ctrl+k":
			if e.commandCursor > 0 {
				e.commandCursor--
			}
			return e, nil
		case "down", "ctrl+j":
			if e.commandCursor < len(e.commandResults)-1 {
				e.commandCursor++
			}
			return e, nil
		}
	}

	var cmd tea.Cmd
	oldVal := e.commandInput.Value()
	e.commandInput, cmd = e.commandInput.Update(msg)
	if e.commandInput.Value() != oldVal {
		e.updateCommandResults(e.commandInput.Value())
	}
	return e, cmd
}

func (e *Editor) executeCommand(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "save":
		return e.handleSave()

	case "save_as":
		e.mode = ViewSaveAs
		e.saveAsInput.SetValue("")
		e.saveAsInput.Focus()
		return e, textinput.Blink

	case "new":
		e.tabs = append(e.tabs, Tab{Buf: buffer.New()})
		e.activeTab = len(e.tabs) - 1
		return e, nil

	case "finder":
		e.mode = ViewFinder
		e.finderInput.SetValue("")
		e.finderInput.Focus()
		e.updateFinderResults("")
		return e, textinput.Blink

	case "sidebar":
		e.fileTreeVisible = !e.fileTreeVisible
		return e, nil

	case "bundle_ai":
		paths := e.fileTree.GetSelectedPaths()
		if len(paths) == 0 {
			e.showMessage("No files selected for AI bundle")
			return e, nil
		}
		md, err := bundler.GenerateMarkdown(paths)
		if err != nil {
			e.showMessage("Failed to generate bundle: " + err.Error())
			return e, nil
		}
		if err := clipboard.WriteAll(md); err != nil {
			e.showMessage("Failed to write to clipboard: " + err.Error())
			return e, nil
		}
		e.showMessage(fmt.Sprintf("Bundled %d files to clipboard (Markdown)", len(paths)))
		return e, nil

	case "bundle_xml":
		paths := e.fileTree.GetSelectedPaths()
		if len(paths) == 0 {
			e.showMessage("No files selected for AI bundle")
			return e, nil
		}
		out, err := bundler.GenerateXML(paths)
		if err != nil {
			e.showMessage("Failed to generate XML bundle: " + err.Error())
			return e, nil
		}
		if err := clipboard.WriteAll(out); err != nil {
			e.showMessage("Failed to write to clipboard: " + err.Error())
			return e, nil
		}
		e.showMessage(fmt.Sprintf("Bundled %d files to clipboard (XML)", len(paths)))
		return e, nil

	case "bundle_text":
		paths := e.fileTree.GetSelectedPaths()
		if len(paths) == 0 {
			e.showMessage("No files selected for AI bundle")
			return e, nil
		}
		out, err := bundler.GeneratePlainText(paths)
		if err != nil {
			e.showMessage("Failed to generate text bundle: " + err.Error())
			return e, nil
		}
		if err := clipboard.WriteAll(out); err != nil {
			e.showMessage("Failed to write to clipboard: " + err.Error())
			return e, nil
		}
		e.showMessage(fmt.Sprintf("Bundled %d files to clipboard (Text)", len(paths)))
		return e, nil

	case "search":
		e.mode = ViewSearch
		e.searchInput.SetValue("")
		e.searchInput.Focus()
		return e, textinput.Blink

	case "global_search":
		e.mode = ViewGlobalSearch
		e.globalSearchInput.SetValue("")
		e.globalSearchInput.Focus()
		e.globalSearchResults = nil
		e.globalSearchCursor = 0
		return e, textinput.Blink

	case "replace":
		e.mode = ViewSearchReplace
		e.searchInput.SetValue("")
		e.replaceInput.SetValue("")
		e.searchInput.Focus()
		return e, textinput.Blink

	case "goto":
		e.mode = ViewGoToLine
		e.gotoInput.SetValue("")
		e.gotoInput.Focus()
		return e, textinput.Blink

	case "select_all":
		tab := e.currentTab()
		tab.SelectActive = true
		tab.SelectStartL = 0
		tab.SelectStartC = 0
		tab.CursorLine = e.currentBuf().LineCount() - 1
		tab.CursorCol = e.currentBuf().LineLen(tab.CursorLine)
		return e, nil

	case "comment":
		e.handleToggleComment()
		return e, nil

	case "dup_line":
		e.currentBuf().DuplicateLine(e.currentTab().CursorLine)
		e.currentTab().CursorLine++
		return e, nil

	case "del_line":
		e.currentBuf().DeleteLine(e.currentTab().CursorLine)
		return e, nil

	case "move_up":
		e.currentTab().CursorLine = e.currentBuf().MoveLineUp(e.currentTab().CursorLine)
		return e, nil

	case "move_down":
		e.currentTab().CursorLine = e.currentBuf().MoveLineDown(e.currentTab().CursorLine)
		return e, nil

	case "indent":
		e.handleIndent()
		return e, nil

	case "unindent":
		e.handleUnindent()
		return e, nil

	case "theme":
		theme.ToggleTheme()
		return e, nil

	case "close_tab":
		return e.handleCloseTab()

	case "close_session":
		for i, t := range e.tabs {
			if t.Buf.Modified {
				e.activeTab = i
				e.mode = ViewUnsavedPrompt
				e.pendingAction = ActionCloseSession
				e.unsavedTabIdx = i
				return e, nil
			}
		}
		e.closeSession()
		return e, nil

	case "open_folder":
		e.showOpenFolder()
		return e, nil

	case "next_tab":
		if e.activeTab < len(e.tabs)-1 {
			e.activeTab++
		} else {
			e.activeTab = 0
		}
		return e, nil

	case "prev_tab":
		if e.activeTab > 0 {
			e.activeTab--
		} else {
			e.activeTab = len(e.tabs) - 1
		}
		return e, nil

	case "zoom_in":
		e.zoomLevel++
		return e, nil

	case "zoom_out":
		if e.zoomLevel > -3 {
			e.zoomLevel--
		}
		return e, nil

	case "center":
		e.centerCursor()
		return e, nil

	case "reload":
		buf := e.currentBuf()
		if buf.FilePath != "" {
			newBuf, err := buffer.Open(buf.FilePath)
			if err != nil {
				e.showMessage(fmt.Sprintf("Error: %v", err))
			} else {
				e.currentTab().Buf = newBuf
				e.showMessage("Reloaded: " + newBuf.Name)
			}
		}
		return e, nil

	case "quit":
		for i, t := range e.tabs {
			if t.Buf.Modified {
				e.activeTab = i
				e.mode = ViewUnsavedPrompt
				e.pendingAction = ActionQuit
				e.unsavedTabIdx = i
				return e, nil
			}
		}
		return e, tea.Quit

	case "help":
		e.mode = ViewHelp
		e.showHelp = true
		return e, nil

	case "open_settings":
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".panda-editor", "settings.json")
		buf, err := buffer.Open(path)
		if err != nil {
			// If file doesn't exist, config.SaveConfig will create it
			config.SaveConfig(config.DefaultConfig())
			buf, _ = buffer.Open(path)
		}
		e.tabs = append(e.tabs, Tab{Buf: buf})
		e.activeTab = len(e.tabs) - 1
		e.mode = ViewNormal
		return e, nil

	case "open_keybindings_config":
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".panda-editor", "keybindings.json")
		buf, err := buffer.Open(path)
		if err != nil {
			// Save default keybind config first
			data, _ := json.MarshalIndent(e.keyBindCfg, "", "  ")
			os.MkdirAll(filepath.Dir(path), 0755)
			os.WriteFile(path, data, 0644)
			buf, _ = buffer.Open(path)
		}
		e.tabs = append(e.tabs, Tab{Buf: buf})
		e.activeTab = len(e.tabs) - 1
		e.mode = ViewNormal
		return e, nil


	case "keybindings":
		e.mode = ViewKeybindings
		e.kbCursor = 0
		e.kbScroll = 0
		e.kbEditing = false
		return e, nil

	case "reset_keybindings":
		if err := ResetKeyBindConfig(); err != nil {
			e.showMessage(fmt.Sprintf("Error: %v", err))
		} else {
			e.keyBindCfg = DefaultKeyBindConfig()
			e.keys = KeyMapFromConfig(e.keyBindCfg)
			e.kbEntries = e.keyBindCfg.GetKeyBindEntries()
			e.rebuildKeyMap()
			e.showMessage("Keybindings reset to defaults")
		}
		return e, nil

	case "undo":
		e.handleUndo()
		return e, nil

	case "redo":
		e.handleRedo()
		return e, nil

	case "terminal":
		cmdStr := e.keyBindCfg.TerminalCmd
		if cmdStr == "" {
			if runtime.GOOS == "windows" {
				cmdStr = "cmd"
			} else {
				cmdStr = "bash"
			}
		}
		c := exec.Command(cmdStr)
		return e, tea.ExecProcess(c, func(err error) tea.Msg {
			return nil
		})

	default:
		return e, nil
	}
}

func (e *Editor) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			e.mode = ViewNormal
			e.searchInput.Blur()
			e.replaceInput.Blur()
			e.searchActive = false
			return e, nil
		case "enter":
			e.searchQuery = e.searchInput.Value()
			e.searchActive = true
			if e.searchQuery != "" {
				e.searchNext()
			}
			return e, nil
		case "ctrl+r":
			if e.mode == ViewSearchReplace {
				e.replaceCurrent()
			}
			return e, nil
		case "ctrl+shift+r":
			if e.mode == ViewSearchReplace {
				e.replaceAll()
			}
			return e, nil
		case "tab":
			if e.mode == ViewSearchReplace {
				if e.searchInput.Focused() {
					e.searchInput.Blur()
					e.replaceInput.Focus()
				} else {
					e.replaceInput.Blur()
					e.searchInput.Focus()
				}
			}
			return e, nil
		}
	}

	var cmd tea.Cmd
	if e.searchInput.Focused() {
		e.searchInput, cmd = e.searchInput.Update(msg)
	} else if e.replaceInput.Focused() {
		e.replaceInput, cmd = e.replaceInput.Update(msg)
	}
	return e, cmd
}

func (e *Editor) replaceCurrent() {
	tab := e.currentTab()
	buf := e.currentBuf()

	q := e.searchQuery
	if q == "" {
		q = e.searchInput.Value()
	}
	if q == "" {
		return
	}

	line := buf.GetLine(tab.CursorLine)
	r := e.replaceInput.Value()

	lineRunes := []rune(line)
	patternRunes := []rune(q)

	start := tab.CursorCol
	if start < 0 {
		start = 0
	}
	if start > len(lineRunes) {
		start = len(lineRunes)
	}

	matchIdx := -1
	// rune-level search
	if len(patternRunes) == 0 {
		return
	}
	for i := start; i+len(patternRunes) <= len(lineRunes); i++ {
		ok := true
		for j := 0; j < len(patternRunes); j++ {
			r1 := lineRunes[i+j]
			r2 := patternRunes[j]
			if !e.caseSensitive {
				r1 = unicode.ToLower(r1)
				r2 = unicode.ToLower(r2)
			}
			if r1 != r2 {
				ok = false
				break
			}
		}
		if ok {
			matchIdx = i
			break
		}
	}

	if matchIdx < 0 {
		e.showMessage("No match at cursor")
		return
	}

	// Replace using the original rune slice (so cursor/cols stay consistent).
	newRunes := make([]rune, 0, len(lineRunes)+len([]rune(r))-len(patternRunes))
	newRunes = append(newRunes, lineRunes[:matchIdx]...)
	newRunes = append(newRunes, []rune(r)...)
	newRunes = append(newRunes, lineRunes[matchIdx+len(patternRunes):]...)

	buf.SetLine(tab.CursorLine, string(newRunes))
	buf.Modified = true

	// Cursor after replacement: end of inserted text.
	tab.CursorCol = matchIdx + len([]rune(r))
	e.ensureCursorVisible()

	e.showMessage("Replaced")
	e.searchNext()
}

func (e *Editor) replaceAll() {
	q := e.searchQuery
	if q == "" {
		q = e.searchInput.Value()
	}
	if q == "" {
		return
	}
	r := e.replaceInput.Value()
	buf := e.currentBuf()
	count := 0
	for i := 0; i < buf.LineCount(); i++ {
		line := buf.GetLine(i)
		if e.caseSensitive {
			newLine := strings.ReplaceAll(line, q, r)
			if newLine != line {
				buf.SetLine(i, newLine)
				count += strings.Count(line, q)
			}
		} else {
			lineRunes := []rune(line)
			patternRunes := []rune(q)
			replaceRunes := []rune(r)
			if len(patternRunes) == 0 {
				continue
			}

			var newRunes []rune
			j := 0
			replacedInLine := false
			for j <= len(lineRunes)-len(patternRunes) {
				ok := true
				for k := 0; k < len(patternRunes); k++ {
					if unicode.ToLower(lineRunes[j+k]) != unicode.ToLower(patternRunes[k]) {
						ok = false
						break
					}
				}
				if ok {
					newRunes = append(newRunes, replaceRunes...)
					j += len(patternRunes)
					count++
					replacedInLine = true
				} else {
					newRunes = append(newRunes, lineRunes[j])
					j++
				}
			}
			if j < len(lineRunes) {
				newRunes = append(newRunes, lineRunes[j:]...)
			}
			if replacedInLine {
				buf.SetLine(i, string(newRunes))
			}
		}
	}
	e.showMessage(fmt.Sprintf("Replaced %d occurrences", count))
}

func (e *Editor) updateGoToLine(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			e.mode = ViewNormal
			e.gotoInput.Blur()
			return e, nil
		case "enter":
			var lineNum int
			fmt.Sscanf(e.gotoInput.Value(), "%d", &lineNum)
			if lineNum > 0 {
				tab := e.currentTab()
				if lineNum > e.currentBuf().LineCount() {
					lineNum = e.currentBuf().LineCount()
				}
				tab.CursorLine = lineNum - 1
				tab.CursorCol = 0
				e.ensureCursorVisible()
			}
			e.mode = ViewNormal
			e.gotoInput.Blur()
			return e, nil
		}
	}

	var cmd tea.Cmd
	e.gotoInput, cmd = e.gotoInput.Update(msg)
	return e, cmd
}

func (e *Editor) updateSaveAs(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			e.mode = ViewNormal
			e.saveAsInput.Blur()
			return e, nil
		case "enter":
			path := e.saveAsInput.Value()
			if path != "" {
				// Phase 17: Handle rename
				if e.pendingRename != "" {
					if err := os.Rename(e.pendingRename, path); err != nil {
						e.showMessage("Rename failed: " + err.Error())
					} else {
						e.showMessage("Renamed to: " + filepath.Base(path))
						e.fileTree = NewFileTree(e.fileTreeRoot)
						e.buildFileList()
						// Update any open tab pointing to old path
						for i := range e.tabs {
							if e.tabs[i].Buf.FilePath == e.pendingRename {
								e.tabs[i].Buf.FilePath = path
								e.tabs[i].Buf.Name = filepath.Base(path)
							}
						}
					}
					e.pendingRename = ""
					e.mode = ViewFileTree
					e.saveAsInput.Blur()
					return e, nil
				}
				if err := e.currentBuf().SaveAs(path); err != nil {
					e.showMessage(fmt.Sprintf("Error: %v", err))
				} else {
					e.showMessage("Saved: " + e.currentBuf().Name)
					e.buildFileList()
					e.fileTree = NewFileTree(e.fileTreeRoot)
				}
			}
			e.mode = ViewNormal
			e.saveAsInput.Blur()
			if e.postSaveAction != ActionNone {
				e.pendingAction = e.postSaveAction
				e.postSaveAction = ActionNone
				return e.executePendingAction()
			}
			return e, nil
		}
	}

	var cmd tea.Cmd
	e.saveAsInput, cmd = e.saveAsInput.Update(msg)
	return e, cmd
}

func (e *Editor) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		s := msg.String()
		if s == "esc" || s == "f1" || s == "alt+h" {
			e.mode = ViewNormal
			e.showHelp = false
		} else if s == "up" || s == "ctrl+p" {
			if e.helpScroll > 0 {
				e.helpScroll--
			}
		} else if s == "down" || s == "ctrl+n" {
			e.helpScroll++
		}
	}
	return e, nil
}

func (e *Editor) updateKeybindings(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			e.mode = ViewNormal
			return e, nil
		case "up":
			if e.kbCursor > 0 {
				e.kbCursor--
			}
			if e.kbCursor < e.kbScroll {
				e.kbScroll = e.kbCursor
			}
			return e, nil
		case "down":
			if e.kbCursor < len(e.kbEntries)-1 {
				e.kbCursor++
			}
			return e, nil
		case "enter":
			if e.kbCursor >= 0 && e.kbCursor < len(e.kbEntries) {
				e.kbEditing = true
				e.kbEditIndex = e.kbCursor
				e.kbEditInput.SetValue("")
				e.kbEditInput.Focus()
				return e, textinput.Blink
			}
			return e, nil
		case "r":
			if err := ResetKeyBindConfig(); err != nil {
				e.showMessage(fmt.Sprintf("Error: %v", err))
			} else {
				e.keyBindCfg = DefaultKeyBindConfig()
				e.keys = KeyMapFromConfig(e.keyBindCfg)
				e.kbEntries = e.keyBindCfg.GetKeyBindEntries()
				e.showMessage("Keybindings reset to defaults")
			}
			return e, nil
		case "s":
			if err := SaveKeyBindConfig(e.keyBindCfg); err != nil {
				e.showMessage(fmt.Sprintf("Error saving: %v", err))
			} else {
				e.showMessage("Keybindings saved to " + configFilePath())
			}
			return e, nil
		case "o":
			cfgPath := configFilePath()
			dir := configDir()
			os.MkdirAll(dir, 0o755)
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				SaveKeyBindConfig(e.keyBindCfg)
			}
			e.mode = ViewNormal
			e.openFile(cfgPath)
			return e, nil
		case "home":
			e.kbCursor = 0
			e.kbScroll = 0
			return e, nil
		case "end":
			e.kbCursor = len(e.kbEntries) - 1
			return e, nil
		case "pgup":
			e.kbCursor -= 10
			if e.kbCursor < 0 {
				e.kbCursor = 0
			}
			if e.kbCursor < e.kbScroll {
				e.kbScroll = e.kbCursor
			}
			return e, nil
		case "pgdown":
			e.kbCursor += 10
			if e.kbCursor >= len(e.kbEntries) {
				e.kbCursor = len(e.kbEntries) - 1
			}
			return e, nil
		}
	}
	return e, nil
}

func (e *Editor) updateKeybindEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			e.kbEditing = false
			e.kbEditInput.Blur()
			return e, nil
		case "enter":
			newKeys := e.kbEditInput.Value()
			if newKeys != "" && e.kbEditIndex >= 0 && e.kbEditIndex < len(e.kbEntries) {
				parsed := parseKeysInput(newKeys)
				if len(parsed) > 0 {
					*e.kbEntries[e.kbEditIndex].Field = parsed
					e.kbEntries = e.keyBindCfg.GetKeyBindEntries()
					e.rebuildKeyMap()
					e.showMessage(fmt.Sprintf("Updated %s → %s", e.kbEntries[e.kbEditIndex].Label, JoinKeys(parsed)))
				} else {
					e.showMessage("Invalid key format")
				}
			}
			e.kbEditing = false
			e.kbEditInput.Blur()
			return e, nil
		}
	}

	var cmd tea.Cmd
	e.kbEditInput, cmd = e.kbEditInput.Update(msg)
	return e, cmd
}

func parseKeysInput(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	keys := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			keys = append(keys, strings.ToLower(p))
		}
	}
	return keys
}

func (e *Editor) rebuildKeyMap() {
	e.keys = KeyMapFromConfig(e.keyBindCfg)
}

func (e *Editor) saveSession() {
	state := session.SessionState{
		ActiveTab:    e.activeTab,
		SidebarOpen:  e.fileTreeVisible,
		ThemeName:    theme.CurrentTheme.Name,
		ZoomLevel:    e.zoomLevel,
		LastDir:      e.fileTreeRoot,
		WindowWidth:  e.width,
		WindowHeight: e.height,
	}

	for _, tab := range e.tabs {
		ts := session.TabState{
			CursorLine: tab.CursorLine,
			CursorCol:  tab.CursorCol,
			ScrollLine: tab.ScrollLine,
			ScrollCol:  tab.ScrollCol,
		}
		if tab.Buf.FilePath != "" {
			ts.FilePath = tab.Buf.FilePath
			ts.IsScratch = false
		} else {
			ts.IsScratch = true
			if tab.Buf.Modified {
				ts.Content = strings.Join(tab.Buf.Lines, "\n")
			}
		}
		state.Tabs = append(state.Tabs, ts)
	}

	if err := session.SaveSession(state); err != nil {
		// silently fail
	}
}

func (e *Editor) restoreSession() {
	state, err := session.LoadSession()
	if err != nil {
		e.mode = ViewNormal
		return
	}

	if state.ThemeName == theme.Light.Name {
		theme.SetTheme(theme.Light)
	}

	e.zoomLevel = state.ZoomLevel
	e.fileTreeVisible = state.SidebarOpen

	if state.WindowWidth > 0 {
		e.width = state.WindowWidth
	}
	if state.WindowHeight > 0 {
		e.height = state.WindowHeight
	}

	e.tabs = nil
	for _, ts := range state.Tabs {
		tab := Tab{
			CursorLine: ts.CursorLine,
			CursorCol:  ts.CursorCol,
			ScrollLine: ts.ScrollLine,
			ScrollCol:  ts.ScrollCol,
		}
		if ts.IsScratch {
			if ts.Content != "" {
				tab.Buf = buffer.FromString(ts.Content)
			} else {
				tab.Buf = buffer.New()
			}
		} else if ts.FilePath != "" {
			buf, err := buffer.Open(ts.FilePath)
			if err != nil {
				tab.Buf = buffer.New()
				tab.Buf.Name = filepath.Base(ts.FilePath) + " (not found)"
			} else {
				tab.Buf = buf
				tab.CursorLine = ts.CursorLine
				tab.CursorCol = ts.CursorCol
				tab.ScrollLine = ts.ScrollLine
				tab.ScrollCol = ts.ScrollCol
			}
		} else {
			tab.Buf = buffer.New()
		}
		e.tabs = append(e.tabs, tab)
	}

	if len(e.tabs) == 0 {
		e.tabs = []Tab{{Buf: buffer.New()}}
	}

	if state.ActiveTab >= 0 && state.ActiveTab < len(e.tabs) {
		e.activeTab = state.ActiveTab
	} else {
		e.activeTab = 0
	}

	if state.LastDir != "" {
		if _, err := os.Stat(state.LastDir); err == nil {
			e.fileTreeRoot = state.LastDir
			e.fileTree = NewFileTree(state.LastDir)
			e.buildFileList()
		}
	}

	e.mode = ViewNormal
	e.showMessage("Session restored from " + state.SavedAt)
}

func (e *Editor) getWelcomeActions() []string {
	actions := []string{"new", "open", "folder"}
	if e.hasSession {
		actions = append(actions, "restore")
	}
	if len(e.tabs) > 0 {
		actions = append(actions, "exit_session")
	}
	actions = append(actions, "quit")
	return actions
}

func (e *Editor) welcomeItemCount() int {
	return len(e.getWelcomeActions())
}

func (e *Editor) welcomeItemAction(cursor int) string {
	actions := e.getWelcomeActions()
	if cursor >= 0 && cursor < len(actions) {
		return actions[cursor]
	}
	return ""
}

func (e *Editor) updateWelcome(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+q", "esc":
			return e, tea.Quit

		case "n":
			e.mode = ViewNormal
			if len(e.tabs) == 0 {
				e.tabs = []Tab{{Buf: buffer.New()}}
				e.activeTab = 0
			}
			return e, nil

		case "enter":
			action := e.welcomeItemAction(e.welcomeCursor)
			switch action {
			case "new":
				e.mode = ViewNormal
				if len(e.tabs) == 0 {
					e.tabs = []Tab{{Buf: buffer.New()}}
					e.activeTab = 0
				}
			case "restore":
				e.restoreSession()
			case "open":
				e.mode = ViewFinder
				e.finderInput.SetValue("")
				e.finderInput.Focus()
				e.updateFinderResults("")
				return e, textinput.Blink
			case "folder":
				e.mode = ViewOpenFolder
				e.openFolderInput.SetValue(e.fileTreeRoot)
				e.openFolderInput.Focus()
				return e, textinput.Blink
			case "exit_session":
				e.closeSession()
			case "quit":
				return e, tea.Quit
			}
			return e, nil

		case "up", "k":
			if e.welcomeCursor > 0 {
				e.welcomeCursor--
			}
			return e, nil

		case "down", "j":
			if e.welcomeCursor < e.welcomeItemCount()-1 {
				e.welcomeCursor++
			}
			return e, nil

		case "o":
			e.mode = ViewFinder
			e.finderInput.SetValue("")
			e.finderInput.Focus()
			e.updateFinderResults("")
			return e, textinput.Blink

		case "f":
			e.mode = ViewOpenFolder
			e.openFolderInput.SetValue(e.fileTreeRoot)
			e.openFolderInput.Focus()
			return e, textinput.Blink

		case "x":
			if len(e.tabs) > 0 {
				e.closeSession()
			}
			return e, nil

		case "r":
			if e.hasSession {
				e.restoreSession()
				return e, nil
			}

		case "f1":
			e.mode = ViewHelp
			e.showHelp = true
			return e, nil

		case "alt+k":
			e.mode = ViewKeybindings
			e.kbCursor = 0
			e.kbScroll = 0
			return e, nil
		}
	}
	return e, nil
}

func (e *Editor) updateUnsavedPrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch strings.ToLower(msg.String()) {
		case "y":
			// Save and continue
			tab := &e.tabs[e.unsavedTabIdx]
			if tab.Buf.FilePath == "" {
				e.mode = ViewSaveAs
				e.postSaveAction = e.pendingAction
				e.saveAsInput.SetValue("")
				e.saveAsInput.Focus()
				return e, textinput.Blink
			}
			e.saveTab(e.unsavedTabIdx)
			return e.executePendingAction()
		case "n":
			// Discard and continue
			if e.unsavedTabIdx >= 0 && e.unsavedTabIdx < len(e.tabs) {
				e.tabs[e.unsavedTabIdx].Buf.Modified = false
			}
			return e.executePendingAction()
		case "esc":
			e.mode = ViewNormal
			e.pendingAction = ActionNone
			e.postSaveAction = ActionNone
		}
	}
	return e, nil
}

func (e *Editor) showOpenFolder() {
	e.openFolderInput.SetValue(e.fileTreeRoot)
	e.openFolderInput.Focus()
	e.mode = ViewOpenFolder
}

func (e *Editor) updateOpenFolder(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			e.mode = ViewNormal
			e.openFolderInput.Blur()
			return e, nil
		case "enter":
			path := e.openFolderInput.Value()
			if path == "" {
				e.mode = ViewNormal
				e.openFolderInput.Blur()
				return e, nil
			}
			absPath, err := filepath.Abs(path)
			if err != nil {
				e.showMessage(fmt.Sprintf("Error: %v", err))
				return e, nil
			}
			info, err := os.Stat(absPath)
			if err != nil {
				e.showMessage(fmt.Sprintf("Path not found: %s", absPath))
				return e, nil
			}
			if !info.IsDir() {
				e.showMessage("Not a directory: " + absPath)
				return e, nil
			}
			e.fileTreeRoot = absPath
			e.fileTree = NewFileTree(absPath)
			e.buildFileList()
			e.showMessage("Opened folder: " + absPath)
			e.mode = ViewNormal
			e.openFolderInput.Blur()
			return e, nil
		}
	}
	var cmd tea.Cmd
	e.openFolderInput, cmd = e.openFolderInput.Update(msg)
	return e, cmd
}

func OpenFileInEditor(e *Editor, path string) {
	e.openFile(path)
}

func (e *Editor) updateFileTree(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, e.keys.SelectFile) {
			e.fileTree.ToggleSelect(e.fileTree.Cursor)
			return e, nil
		}

		switch msg.String() {
		case "esc":
			e.mode = ViewNormal
			return e, nil

		case "up", "k":
			if e.fileTree.Cursor > 0 {
				e.fileTree.Cursor--
				e.fileTree.AdjustScroll(e.editorHeight())
			}
			return e, nil

		case "down", "j":
			if e.fileTree.Cursor < len(e.fileTree.Flat)-1 {
				e.fileTree.Cursor++
				e.fileTree.AdjustScroll(e.editorHeight())
			}
			return e, nil

		case "enter":
			node := e.fileTree.SelectedNode()
			if node == nil {
				return e, nil
			}
			if node.IsDir {
				e.fileTree.Toggle(e.fileTree.Cursor)
				e.fileTree.AdjustScroll(e.editorHeight())
			} else {
				e.openFile(node.Path)
				e.mode = ViewNormal
			}
			return e, nil

		case "right", "l":
			node := e.fileTree.SelectedNode()
			if node != nil && node.IsDir && !node.Expanded {
				e.fileTree.Toggle(e.fileTree.Cursor)
				e.fileTree.AdjustScroll(e.editorHeight())
			}
			return e, nil

		case "left", "h":
			node := e.fileTree.SelectedNode()
			if node != nil && node.IsDir && node.Expanded {
				e.fileTree.Toggle(e.fileTree.Cursor)
				e.fileTree.AdjustScroll(e.editorHeight())
			}
			return e, nil
		case "home":
			e.fileTree.Cursor = 0
			e.fileTree.AdjustScroll(e.editorHeight())
			return e, nil

		case "end":
			if len(e.fileTree.Flat) > 0 {
				e.fileTree.Cursor = len(e.fileTree.Flat) - 1
				e.fileTree.AdjustScroll(e.editorHeight())
			}
			return e, nil

		// Phase 10: Expand/Collapse all
		case "e":
			e.fileTree.ToggleExpandAll()
			e.fileTree.AdjustScroll(e.editorHeight())
			return e, nil

		// Phase 9: Toggle per-file token display
		case "t":
			e.fileTree.ShowTokens = !e.fileTree.ShowTokens
			if e.fileTree.ShowTokens {
				e.showMessage("Token display: ON")
			} else {
				e.showMessage("Token display: OFF")
			}
			return e, nil

		// Phase 8: Toggle bundle format
		case "B":
			e.bundleFormat = (e.bundleFormat + 1) % 3
			formats := []string{"Markdown", "XML", "Plain Text"}
			e.showMessage("Bundle format: " + formats[e.bundleFormat])
			return e, nil

		// Phase 17: File CRUD from explorer
		case "n":
			node := e.fileTree.SelectedNode()
			dir := e.fileTreeRoot
			if node != nil {
				if node.IsDir {
					dir = node.Path
				} else {
					dir = filepath.Dir(node.Path)
				}
			}
			e.mode = ViewSaveAs
			e.saveAsInput.SetValue(dir + string(os.PathSeparator))
			e.saveAsInput.Focus()
			return e, textinput.Blink

		case "R":
			node := e.fileTree.SelectedNode()
			if node == nil || node.Depth == 0 {
				return e, nil
			}
			e.mode = ViewSaveAs
			e.saveAsInput.SetValue(node.Path)
			e.saveAsInput.Focus()
			e.pendingRename = node.Path
			return e, textinput.Blink

		case "d":
			node := e.fileTree.SelectedNode()
			if node == nil || node.Depth == 0 {
				return e, nil
			}
			e.pendingDelete = node.Path
			e.mode = ViewUnsavedPrompt
			e.pendingAction = ActionDelete
			return e, nil

		// Phase 12: Fuzzy search within explorer
		case "/":
			e.mode = ViewFileTreeFilter
			e.fileTreeFilterInput.Focus()
			return e, textinput.Blink
		}

		if key.Matches(msg, e.keys.OpenExplorer) {
			e.mode = ViewNormal
			return e, nil
		}
	}
	return e, nil
}

func (e *Editor) updateFileTreeFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			e.mode = ViewFileTree
			e.fileTreeFilterInput.Blur()
			return e, nil
		case "enter":
			e.mode = ViewFileTree
			e.fileTreeFilterInput.Blur()
			return e, nil
		}
	}

	var cmd tea.Cmd
	oldVal := e.fileTreeFilterInput.Value()
	e.fileTreeFilterInput, cmd = e.fileTreeFilterInput.Update(msg)
	newVal := e.fileTreeFilterInput.Value()

	if oldVal != newVal {
		e.fileTree.Filter = newVal
		e.fileTree.Flatten()
		e.fileTree.Cursor = 0
	}

	return e, cmd
}

func (e *Editor) updateGlobalSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			e.mode = ViewNormal
			e.globalSearchInput.Blur()
			return e, nil
		case "enter":
			if e.globalSearchInput.Focused() {
				e.globalSearchInput.Blur()
				e.triggerGlobalSearch()
				return e, nil
			} else {
				// Open selected search result
				if e.globalSearchCursor >= 0 && e.globalSearchCursor < len(e.globalSearchResults) {
					res := e.globalSearchResults[e.globalSearchCursor]
					e.openFile(res.Path)
					// Go to line
					tab := e.currentTab()
					tab.CursorLine = res.LineNum - 1
					if tab.CursorLine < 0 {
						tab.CursorLine = 0
					}
					tab.CursorCol = 0
					e.ensureCursorVisible()
					e.mode = ViewNormal
				}
				return e, nil
			}
		case "up", "ctrl+k":
			if e.globalSearchCursor > 0 {
				e.globalSearchCursor--
			}
			return e, nil
		case "down", "ctrl+j":
			if e.globalSearchCursor < len(e.globalSearchResults)-1 {
				e.globalSearchCursor++
			}
			return e, nil
		case "tab":
			if e.globalSearchInput.Focused() {
				e.globalSearchInput.Blur()
			} else {
				e.globalSearchInput.Focus()
			}
			return e, textinput.Blink
		}
	}

	var cmd tea.Cmd
	e.globalSearchInput, cmd = e.globalSearchInput.Update(msg)
	return e, cmd
}

func (e *Editor) triggerGlobalSearch() {
	query := e.globalSearchInput.Value()
	if query == "" {
		return
	}

	e.globalSearchResults = nil
	e.isSearching = true
	e.globalSearchCursor = 0

	opts := searcher.Options{
		Root:          e.fileTreeRoot,
		Query:         query,
		CaseSensitive: false,
		IgnoreDirs:    []string{".git", "node_modules", "vendor", "dist", "build"},
	}

	searcher.Search(opts, e.searchResultChan, e.searchDoneChan)
}

func (e *Editor) getGitBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = e.fileTreeRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (e *Editor) getGitDiffs(path string) map[int]string {
	if path == "" {
		return nil
	}
	cmd := exec.Command("git", "diff", "-U0", path)
	cmd.Dir = e.fileTreeRoot
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	diffs := make(map[int]string)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			// Parse @@ -L,C +L,C @@
			parts := strings.Split(line, " ")
			if len(parts) < 3 {
				continue
			}
			// parts[2] is +L,C or +L
			target := strings.TrimPrefix(parts[2], "+")
			targetParts := strings.Split(target, ",")
			startLine, _ := strconv.Atoi(targetParts[0])
			count := 1
			if len(targetParts) > 1 {
				count, _ = strconv.Atoi(targetParts[1])
			}

			// parts[1] is -L,C or -L
			source := strings.TrimPrefix(parts[1], "-")
			sourceParts := strings.Split(source, ",")
			sourceLen := 1
			if len(sourceParts) > 1 {
				sourceLen, _ = strconv.Atoi(sourceParts[1])
			}

			if sourceLen == 0 && count > 0 {
				// Added
				for i := 0; i < count; i++ {
					diffs[startLine+i-1] = "+"
				}
			} else if sourceLen > 0 && count > 0 {
				// Modified
				for i := 0; i < count; i++ {
					diffs[startLine+i-1] = "~"
				}
			} else if sourceLen > 0 && count == 0 {
				// Deleted
				diffs[startLine-1] = "-"
			}
		}
	}
	return diffs
}

func (e *Editor) closeSession() {
	e.tabs = nil
	e.activeTab = 0
	e.mode = ViewWelcome
	e.showMessage("Session closed")
}

func (e *Editor) executePendingAction() (tea.Model, tea.Cmd) {
	action := e.pendingAction
	e.mode = ViewNormal

	switch action {
	case ActionQuit:
		// Check for next unsaved tab
		for i, t := range e.tabs {
			if t.Buf.Modified {
				e.mode = ViewUnsavedPrompt
				e.unsavedTabIdx = i
				return e, nil
			}
		}
		e.saveSession()
		return e, tea.Quit

	case ActionCloseTab:
		if e.unsavedTabIdx >= 0 && e.unsavedTabIdx < len(e.tabs) {
			e.tabs = append(e.tabs[:e.unsavedTabIdx], e.tabs[e.unsavedTabIdx+1:]...)
			if e.activeTab >= len(e.tabs) && len(e.tabs) > 0 {
				e.activeTab = len(e.tabs) - 1
			}
			if len(e.tabs) == 0 {
				e.mode = ViewWelcome
			}
		}
		e.pendingAction = ActionNone
		return e, nil

	case ActionCloseSession:
		// Check for next unsaved tab
		for i, t := range e.tabs {
			if t.Buf.Modified {
				e.mode = ViewUnsavedPrompt
				e.unsavedTabIdx = i
				return e, nil
			}
		}
		e.closeSession()
		e.pendingAction = ActionNone
		return e, nil

	case ActionDelete:
		if e.pendingDelete != "" {
			err := os.Remove(e.pendingDelete)
			if err != nil {
				// Try removing directory
				err = os.RemoveAll(e.pendingDelete)
			}
			if err != nil {
				e.showMessage("Delete failed: " + err.Error())
			} else {
				e.showMessage("Deleted: " + filepath.Base(e.pendingDelete))
				e.fileTree = NewFileTree(e.fileTreeRoot)
				e.buildFileList()
			}
			e.pendingDelete = ""
		}
		e.pendingAction = ActionNone
		return e, nil
	}
	return e, nil
}

func (e *Editor) saveTab(idx int) {
	if idx < 0 || idx >= len(e.tabs) {
		return
	}
	tab := &e.tabs[idx]
	if tab.Buf.FilePath == "" {
		return
	}
	err := tab.Buf.Save()
	if err != nil {
		e.showMessage("Save failed: " + err.Error())
	}
}

func (e *Editor) updateSuggestions() {
	tab := e.currentTab()
	if tab == nil || tab.Buf == nil {
		e.suggestions = nil
		return
	}

	line := tab.Buf.GetLine(tab.CursorLine)
	runes := []rune(line)
	if tab.CursorCol <= 0 || tab.CursorCol > len(runes) {
		e.suggestions = nil
		return
	}

	// Get word before cursor
	start := tab.CursorCol - 1
	for start >= 0 && buffer.IsWordChar(runes[start]) {
		start--
	}
	prefix := string(runes[start+1 : tab.CursorCol])
	if len(prefix) < 2 {
		e.suggestions = nil
		return
	}

	// Simple word-based completion from current buffer
	words := make(map[string]bool)
	for _, l := range tab.Buf.Lines {
		for _, w := range strings.FieldsFunc(l, func(r rune) bool { return !buffer.IsWordChar(r) }) {
			if strings.HasPrefix(w, prefix) && w != prefix {
				words[w] = true
			}
		}
	}

	var suggestions []string
	for w := range words {
		suggestions = append(suggestions, w)
	}
	sort.Strings(suggestions)
	if len(suggestions) > 8 {
		suggestions = suggestions[:8] // Limit to 8
	}
	e.suggestions = suggestions
	e.suggestionIdx = 0
}

func (e *Editor) acceptSuggestion(suggestion string) {
	tab := e.currentTab()
	line := tab.Buf.GetLine(tab.CursorLine)
	runes := []rune(line)

	start := tab.CursorCol - 1
	for start >= 0 && buffer.IsWordChar(runes[start]) {
		start--
	}

	newRunes := append(runes[:start+1], []rune(suggestion)...)
	newRunes = append(newRunes, runes[tab.CursorCol:]...)
	tab.Buf.SetLine(tab.CursorLine, string(newRunes))
	tab.CursorCol = start + 1 + len(suggestion)
	e.suggestions = nil
}

func (e *Editor) handleBundle() {
	paths := e.fileTree.GetSelectedPaths()
	if len(paths) == 0 {
		e.showMessage("No files selected -- use Space in explorer")
		return
	}

	format := bundler.FormatMarkdown
	switch e.bundleFormat {
	case 1:
		format = bundler.FormatXML
	case 2:
		format = bundler.FormatPlainText
	}

	bundle, err := bundler.GenerateBundle(paths, format)
	if err != nil {
		e.showMessage("Bundle error: " + err.Error())
		return
	}

	// Try to copy to system clipboard
	err = clipboard.WriteAll(bundle)
	if err != nil {
		// Fallback to internal clipboard if system clipboard fails
		e.clipboard = bundle
		e.showMessage(fmt.Sprintf("System clipboard failed, stored internally -- %d files", len(paths)))
	} else {
		e.showMessage(fmt.Sprintf("Copied bundle -- %d files -- to clipboard", len(paths)))
	}
}
