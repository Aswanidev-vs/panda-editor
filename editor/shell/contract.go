// Package shell is the full-screen editor experience that replaces the v1
// smoke UI: it owns the workspace, the active editor view, the chrome bars,
// and every modal interaction (open, save, quit confirm, search, goto, tab
// switching). One Run call drives the whole session until the user quits.
package shell

import (
	"strconv"
	"strings"

	"github.com/Aswanidev-vs/cherry"
	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/layout"
	"github.com/Aswanidev-vs/cherry/widget"

	"github.com/Aswanidev-vs/panda-editor/editor/cli"
	"github.com/Aswanidev-vs/panda-editor/editor/document"
	"github.com/Aswanidev-vs/panda-editor/editor/editorview"
	"github.com/Aswanidev-vs/panda-editor/editor/search"
	"github.com/Aswanidev-vs/panda-editor/editor/views"
	"github.com/Aswanidev-vs/panda-editor/editor/vimmode"
	"github.com/Aswanidev-vs/panda-editor/editor/workspace"
)

// Options carries launch-time settings from main.
type Options struct {
	Args *cli.Args // parsed command line; files become tabs in order
	// VimMode enables the modal layer from startup (config wires this
	// later; the shell just holds the bool).
	VimMode bool
	// Version is shown on the welcome splash.
	Version string
}

// overlayKind enumerates what modal surface (if any) is on top.
type overlayKind int

const (
	ovNone overlayKind = iota
	ovInput             // bottom InputLine prompt (open/save-as/goto/search/ex)
	ovDialog            // centered confirm dialog
	ovHelp              // centered help popup
	ovWelcome           // centered welcome splash
)

// Shell is the internal app coordinator implementing widget.Widget and
// widget.Cursorer; it is installed as the cherry app root.
type Shell struct {
	app   *cherry.App
	opts  Options
	ws    *workspace.Workspace
	view  *editorview.View
	vim   *vimmode.Mode
	mode  string // hint-bar mode: "insert" / "search" / "dialog" / "welcome"
	msg   string // transient status message (status bar middle)
	lastH int    // last editor viewport height, for scroll clamping

	ov      overlayKind
	input   *views.InputLine
	dialog  *views.Dialog
	help    *views.Popup
	welcome *views.Popup

	searchQ search.Query
}

// Run launches the editor session and blocks until exit.
func Run(opts Options) error {
	app, err := cherry.New()
	if err != nil {
		return err
	}
	sh, err := New(app, opts)
	if err != nil {
		app.Quit()
		return err
	}
	app.SetRoot(sh)
	return app.Run()
}

// New builds the whole tree: tab bar, editor view, hint bar, status bar,
// overlays. opts carries the launch spec; app is the running cherry app
// (needed for clipboard). New returns a Shell usable as app root.
func New(app *cherry.App, opts Options) (*Shell, error) {
	s := &Shell{app: app, opts: opts, ws: workspace.New()}
	s.openCLI()
	if s.ws.Count() == 0 {
		s.ws.NewUntitled()
		s.ov = ovWelcome
		s.welcome = views.NewPopup("Welcome", &views.Welcome{Version: opts.Version})
	}
	s.view = editorview.New(app, s.ws.ActiveDoc())
	s.view.Focus()
	if opts.VimMode {
		s.vim = vimmode.New()
		s.view.SetModal(s.vim)
	}
	s.mode = s.hintMode()
	return s, nil
}

// openCLI opens the documents described by the parsed args.
func (s *Shell) openCLI() {
	args := s.opts.Args
	if args == nil {
		return
	}
	for _, fs := range args.Files {
		switch {
		case fs.Path == "-":
			s.ws.NewUntitled()
		case fs.Path == "":
			s.ws.NewUntitled()
			if fs.Line > 0 {
				s.jumpLine(s.ws.ActiveDoc(), fs.Line)
			}
		default:
			path := cli.NormalizePath(fs.Path)
			if i := s.ws.FindByPath(path); i >= 0 {
				s.ws.SetActive(i)
				continue
			}
			doc, err := document.Open(path)
			if err != nil {
				// Missing file: still open an empty buffer that remembers the path.
				doc = document.New()
				doc.SetPath(path)
			}
			s.ws.Add(doc, args.ReadOnly)
			if fs.Line > 0 {
				s.jumpLine(doc, fs.Line)
			}
		}
	}
}

func (s *Shell) jumpLine(doc *document.Document, line int) {
	if doc == nil {
		return
	}
	if line < 1 {
		line = 1
	}
	doc.GotoLine(line - 1)
	doc.EnsureCursorVisible(24)
}

// Measure fills the available region entirely (the app root gets the screen).
func (s *Shell) Measure(max geom.Size) geom.Size { return max }

// Handle binds the global key surface ahead of the widget tree.
func (s *Shell) Handle(ev input.Event) bool {
	if s.ov != ovNone {
		return s.handleOverlay(ev)
	}
	if kp, ok := ev.(input.KeyPress); ok {
		// In vim mode, ':' and '/' open the ex / search prompt.
		if s.vim != nil && kp.Mod == 0 && kp.Key == input.KeyNone &&
			(kp.Rune == ':' || kp.Rune == '/') {
			if kp.Rune == ':' {
				s.openEx()
			} else {
				s.openSearch()
			}
			return true
		}
		if s.globalKey(kp) {
			return true
		}
	}
	return s.view.Handle(ev)
}

func (s *Shell) globalKey(kp input.KeyPress) bool {
	ctrl := kp.Mod.Has(input.ModCtrl)
	if kp.Key == input.KeyNone && ctrl {
		switch kp.Rune {
		case 'q', 'Q':
			s.quitFlow()
		case 's', 'S':
			s.saveActive()
		case 'n', 'N':
			s.newTab()
		case 'w', 'W':
			s.closeActive()
		case 'o', 'O':
			s.openOpen()
		case 'f', 'F':
			s.openSearch()
		case 'g', 'G':
			s.openGoto()
		case 'r', 'R':
			s.openSaveAs()
		default:
			return false
		}
		return true
	}
	switch kp.Key {
	case input.KeyF1:
		s.openHelp()
	case input.KeyPageUp:
		if ctrl {
			s.prevTab()
		}
	case input.KeyPageDown:
		if ctrl {
			s.nextTab()
		}
	default:
		return false
	}
	return true
}

// handleOverlay routes events to the active modal surface and keeps it modal.
func (s *Shell) handleOverlay(ev input.Event) bool {
	switch s.ov {
	case ovWelcome, ovHelp:
		// Any key dismisses the splash / help.
		if _, ok := ev.(input.KeyPress); ok {
			s.ov = ovNone
		}
		return true
	case ovInput:
		if s.input != nil {
			s.input.Handle(ev)
		}
		return true
	case ovDialog:
		if s.dialog != nil {
			s.dialog.Handle(ev)
		}
		return true
	}
	return true
}

// ---------------------------------------------------------------------------
// Overlay openers and callbacks
// ---------------------------------------------------------------------------

func (s *Shell) openSearch() {
	s.ov = ovInput
	s.mode = "search"
	s.input = views.New("Where is: ", "", s.onSearchOK, s.onSearchCancel)
}

func (s *Shell) onSearchOK(text string) {
	s.ov = ovNone
	s.mode = s.hintMode()
	if text == "" {
		return
	}
	s.searchQ = search.Query{Text: text, Regex: false, CaseSensitive: false, Word: false}
	doc := s.ws.ActiveDoc()
	if doc == nil {
		return
	}
	from := doc.Buffer().PosToOffset(doc.Cursor())
	start, end, ok := s.searchQ.FindNext(doc.Buffer(), from+1)
	if !ok {
		start, end, ok = s.searchQ.FindNext(doc.Buffer(), 0)
	}
	if ok {
		doc.SetCursor(doc.Buffer().OffsetToPos(start))
		doc.SelectTo(doc.Buffer().OffsetToPos(end))
		s.ensureVisible(doc)
		s.msg = ""
	} else {
		doc.SelectNone()
		s.msg = "not found: " + text
	}
}

func (s *Shell) onSearchCancel() {
	s.ov = ovNone
	s.mode = s.hintMode()
}

func (s *Shell) openGoto() {
	s.ov = ovInput
	s.input = views.New("Go to line: ", "", s.onGotoOK, s.onGotoCancel)
}

func (s *Shell) onGotoOK(text string) {
	s.ov = ovNone
	n, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return
	}
	doc := s.ws.ActiveDoc()
	if doc == nil {
		return
	}
	s.jumpLine(doc, n)
	s.ensureVisible(doc)
}

func (s *Shell) onGotoCancel() { s.ov = ovNone }

func (s *Shell) openSaveAs() {
	doc := s.ws.ActiveDoc()
	initial := ""
	if doc != nil {
		initial = doc.Path()
	}
	s.ov = ovInput
	s.input = views.New("Write out: ", initial, s.onSaveAsOK, s.onSaveAsCancel)
}

func (s *Shell) onSaveAsOK(text string) {
	s.ov = ovNone
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	s.saveDocAs(s.ws.ActiveDoc(), cli.NormalizePath(text))
}

func (s *Shell) onSaveAsCancel() { s.ov = ovNone }

func (s *Shell) openOpen() {
	s.ov = ovInput
	s.input = views.New("Open: ", "", s.onOpenOK, s.onOpenCancel)
}

func (s *Shell) onOpenOK(text string) {
	s.ov = ovNone
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	path := cli.NormalizePath(text)
	if i := s.ws.FindByPath(path); i >= 0 {
		s.ws.SetActive(i)
		s.syncView()
		return
	}
	doc, err := document.Open(path)
	if err != nil {
		s.msg = "cannot open: " + err.Error()
		return
	}
	s.ws.Add(doc, s.opts.Args != nil && s.opts.Args.ReadOnly)
	s.syncView()
}

func (s *Shell) onOpenCancel() { s.ov = ovNone }

func (s *Shell) openEx() {
	s.ov = ovInput
	s.input = views.New(": ", "", s.onExOK, s.onExCancel)
}

func (s *Shell) onExOK(text string) {
	s.ov = ovNone
	executed, quit, save, path := s.vim.HandleCmd(text)
	if !executed {
		return
	}
	doc := s.ws.ActiveDoc()
	if save {
		if path != "" {
			s.saveDocAs(doc, cli.NormalizePath(path))
		} else {
			s.saveDoc(doc)
		}
	}
	if g := s.vim.GotoLinePending(); g > 0 {
		s.jumpLine(doc, g)
		s.ensureVisible(doc)
	}
	if quit {
		s.app.Quit()
	}
}

func (s *Shell) onExCancel() { s.ov = ovNone }

func (s *Shell) openHelp() {
	s.ov = ovHelp
	s.help = views.NewPopup("Help", &widget.Text{
		Content: helpText,
		Style:   cell.Style{}.Foreground(cell.Indexed(252)).Background(cell.Indexed(236)),
	})
}

// ---------------------------------------------------------------------------
// Tab / file operations
// ---------------------------------------------------------------------------

func (s *Shell) newTab() {
	s.ws.NewUntitled()
	s.syncView()
}

func (s *Shell) closeActive() {
	i := s.ws.Active()
	if i < 0 || !s.ws.Close(i) {
		return
	}
	if s.ws.Count() == 0 {
		s.ws.NewUntitled()
		s.ov = ovWelcome
		s.welcome = views.NewPopup("Welcome", &views.Welcome{Version: s.opts.Version})
	}
	s.syncView()
}

func (s *Shell) prevTab() {
	n := s.ws.Count()
	if n == 0 {
		return
	}
	s.ws.SetActive((s.ws.Active() - 1 + n) % n)
	s.syncView()
}

func (s *Shell) nextTab() {
	n := s.ws.Count()
	if n == 0 {
		return
	}
	s.ws.SetActive((s.ws.Active() + 1) % n)
	s.syncView()
}

func (s *Shell) syncView() {
	doc := s.ws.ActiveDoc()
	if doc == nil {
		return
	}
	s.view.SetDocument(doc)
	s.view.Focus()
}

func (s *Shell) saveActive() {
	doc := s.ws.ActiveDoc()
	if doc == nil {
		return
	}
	if doc.Path() == "" {
		s.openSaveAs()
		return
	}
	s.saveDoc(doc)
}

func (s *Shell) saveDoc(doc *document.Document) {
	if doc == nil {
		return
	}
	if err := doc.Buffer().Save(doc.Path()); err != nil {
		s.msg = "save failed: " + err.Error()
		return
	}
	s.msg = "saved " + doc.Path()
}

func (s *Shell) saveDocAs(doc *document.Document, path string) {
	if doc == nil || path == "" {
		return
	}
	if err := doc.Buffer().SaveAs(path); err != nil {
		s.msg = "save failed: " + err.Error()
		return
	}
	doc.SetPath(path)
	s.msg = "saved " + path
}

func (s *Shell) quitFlow() {
	doc := s.ws.ActiveDoc()
	if doc != nil && doc.Modified() {
		s.ov = ovDialog
		s.mode = "dialog"
		s.dialog = views.NewDialog("Save changes? (y/n/esc)",
			[]string{"yes", "no"}, s.onQuitDialog)
		return
	}
	s.app.Quit()
}

func (s *Shell) onQuitDialog(choice string) {
	s.ov = ovNone
	s.mode = s.hintMode()
	switch choice {
	case "yes":
		s.saveActive()
		s.app.Quit()
	case "no":
		s.app.Quit()
	}
}

func (s *Shell) ensureVisible(doc *document.Document) {
	h := s.lastH
	if h <= 0 {
		h = 24
	}
	doc.EnsureCursorVisible(h)
}

// ---------------------------------------------------------------------------
// Drawing
// ---------------------------------------------------------------------------

// Draw lays out tab bar / editor / hint bar / status bar top to bottom and
// paints any active overlay on top.
func (s *Shell) Draw(ctx *widget.DrawCtx) {
	rects := layout.Solve(ctx.Rect.Size, layout.Vertical, []layout.Spec{
		{Fixed: 1},  // tab bar
		{Fill: true}, // editor
		{Fixed: 1},  // hint bar
		{Fixed: 1},  // status bar
	})
	s.lastH = rects[1].Size.H

	s.drawTabBar(ctx, rects[0])
	s.view.Draw(&widget.DrawCtx{Rect: rects[1], Screen: ctx.Screen})
	s.drawHintBar(ctx, rects[2])

	switch s.ov {
	case ovInput:
		s.input.Draw(&widget.DrawCtx{Rect: rects[3], Screen: ctx.Screen})
	default:
		s.drawStatusBar(ctx, rects[3])
	}

	switch s.ov {
	case ovDialog:
		if s.dialog != nil {
			s.dialog.Draw(ctx)
		}
	case ovHelp:
		if s.help != nil {
			s.help.Draw(ctx)
		}
	case ovWelcome:
		if s.welcome != nil {
			s.welcome.Draw(ctx)
		}
	}
}

func (s *Shell) drawTabBar(ctx *widget.DrawCtx, r geom.Rect) {
	if r.Empty() {
		return
	}
	ctx.Screen.Fill(r, styledBlank(shTabBG))
	x := r.Pos.X
	for i := 0; i < s.ws.Count(); i++ {
		label := s.ws.TabLabel(i)
		st := shTab
		if i == s.ws.Active() {
			st = shTabActive
		}
		x = ctx.Screen.Print(x, r.Pos.Y, r.Right(), " "+label, st)
		x = ctx.Screen.Print(x, r.Pos.Y, r.Right(), " ", shTabBG)
	}
}

func (s *Shell) drawHintBar(ctx *widget.DrawCtx, r geom.Rect) {
	if r.Empty() {
		return
	}
	h := views.HintBar(s.hintMode())
	h.Draw(&widget.DrawCtx{Rect: r, Screen: ctx.Screen})
}

func (s *Shell) drawStatusBar(ctx *widget.DrawCtx, r geom.Rect) {
	if r.Empty() {
		return
	}
	doc := s.ws.ActiveDoc()
	mode := "INSERT"
	if s.vim != nil {
		mode = s.vim.State().String()
	}
	file := "Untitled"
	var ln, col int = 1, 1
	var mod, ro bool
	if doc != nil {
		if p := doc.Path(); p != "" {
			file = p
		}
		mod = doc.Modified()
		ro = doc.ReadOnly()
		c := doc.Cursor()
		ln, col = c.Line+1, c.Col+1
	}
	segs := views.StatusBarText(mode, file, s.msg, ln, col, mod, ro)
	sb := widget.NewStatusBar(segs...)
	sb.Draw(&widget.DrawCtx{Rect: r, Screen: ctx.Screen})
}

func (s *Shell) hintMode() string {
	switch s.ov {
	case ovDialog:
		return "dialog"
	case ovHelp, ovWelcome:
		return "welcome"
	case ovInput:
		return "search"
	}
	return "insert"
}

// CursorPos reports the editor's cursor, hidden while an overlay is up.
func (s *Shell) CursorPos() (geom.Point, bool) {
	if s.ov != ovNone || s.view == nil {
		return geom.Point{}, false
	}
	return s.view.CursorPos()
}

// ---------------------------------------------------------------------------
// Palette
// ---------------------------------------------------------------------------

var (
	shTabBG    = cell.Style{}.Foreground(cell.Indexed(240)).Background(cell.Indexed(236))
	shTab      = cell.Style{}.Foreground(cell.Indexed(252)).Background(cell.Indexed(236))
	shTabActive = cell.Style{}.Foreground(cell.Indexed(236)).Background(cell.Indexed(75)).Bold(true)
)

func styledBlank(st cell.Style) cell.Cell { return cell.Cell{Rune: ' ', Style: st, Width: 1} }

const helpText = `PANDA — terminal editor (cherry TUI)

Global
  ctrl+q  quit (asks to save dirty buffers)
  ctrl+s  save        ctrl+n  new buffer
  ctrl+w  close tab   ctrl+o  open file
  ctrl+f  search      ctrl+g  goto line
  ctrl+r  save as     F1      this help
  ctrl+pgup / pgdn    switch tab

Editing (insert mode is default)
  type / backspace / delete / arrows / home / end
  ctrl+z undo   ctrl+y redo   ctrl+a select all
  ctrl+k cut     ctrl+x cut    ctrl+c copy   ctrl+v paste
  ctrl+left / right  word move   pgup / pgdn  page move

Vim mode (start with PANDA_VIM=1)
  i a A o O  enter insert   h j k l  move
  w b e  word move   0 ^ $  line ends   gg G  doc ends
  x  delete char   dd  delete line   D  delete to end
  dw de d$  delete motion   yy  yank   p P  paste
  v V  visual select   r  replace char   u ctrl+r  undo/redo
  :w :q :wq :e <path> :N  ex commands
  /  search`
