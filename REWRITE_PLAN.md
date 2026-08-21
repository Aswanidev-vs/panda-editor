# Panda Editor v2 — Complete Rewrite Plan

> Status: **DECISIONS CONFIRMED — awaiting owner go-ahead to start P0. No implementation has started.**

## 0. Confirmed decisions

| Decision | Choice |
|---|---|
| Syntax highlighting data | **Keep chroma** (`alecthomas/chroma` — independent of Charmbracelet) |
| Repo strategy | **In-place rewrite**, same repo + module path; old code remains until v2 supersedes |
| Binary name | **`panda`** |
| Default editing model | **Insert-first** (nano/notepad behavior); `vim_mode: true` enables modal layer |

---

## 1. Objectives

1. **Replace Charm entirely** — remove `bubbletea`, `bubbles`, `lipgloss` and build an in-repo TUI framework (`tui/`) from scratch that is *easier to use and faster than Bubble Tea*.
2. **Rewrite the application** on top of that framework as a **drop-in replacement for vim / vi / nano** with a far easier learning curve.
3. Keep the product identity: fast terminal editor with VS Code-inspired conveniences, but usable by anyone in 2 minutes without reading docs.

### Non-goals (v2 scope boundaries)
- Not a GUI/IDE. Terminal only.
- No LSP completion/hover in v2 core (diagnostics stay; completion deferred to post-v2).
- No PTY-based embedded terminal in v2 core (line-based shell panel is kept; true PTY is post-v2).

---

## 2. Findings from current-codebase audit (~11K lines)

| Area | Verdict |
|---|---|
| `editor/model.go` (3,863 ln) | God object: ~90 fields, 16 modes, 71 `key.Matches` sites, undo/multi-cursor logic living in UI layer. **Rewrite.** |
| `editor/view.go` (2,351 ln) | Full-screen ANSI string rebuilt every frame; overlays re-parse escape codes char-by-char; per-rune `lipgloss.Style.Render`; re-lexes lines `[0, scrollOffset)` every frame for comment state. **Rewrite** (this is why it feels slow). |
| `internal/buffer` | Correct + tested, but `[]string` lines with full-copy undo snapshots (100 deep) per keystroke; CRLF silently converted to LF; no BOM. **Rewrite around gap buffer.** |
| `internal/highlight` | Chroma used well (per-line lex + block-comment threading), zero caching, lipgloss-coupled output, ~250 ln dead code. **Rewrite renderer, keep technique, add cache.** |
| `internal/{fuzzy,session,config,bundler,lsp,searcher,watcher}` | Stdlib-only, clean seams. **Salvage** (with small fixes: LSP string-IDs, watcher leak, config's broken `DefaultConfigJSON`, session atomic writes, searcher cancellation). |
| `internal/theme` | lipgloss.Color fields + global mutable state. **Port palettes to hex strings.** |

---

## 3. Deliverable A — `tui/`: the from-scratch framework ("the Charm killer")

Zero third-party UI deps. Only allowed external packages project-wide: `golang.org/x/sys` (raw mode / Windows console), `github.com/fsnotify/fsnotify`, `github.com/alecthomas/chroma` (syntax data — independent project, not Charm). Clipboard moves to OSC52 + native APIs (drops `atotto/clipboard`).

### 3.1 Module layout

```
tui/
├── term/      # raw mode (unix termios + win32 console), resize signals,
│              # alt-screen, suspend/resume, size queries
├── input/     # unified event parser: CSI/SS3 keys, chords, mouse (1000–1006),
│              # bracketed paste, focus events, kitty keyboard protocol,
│              # Windows INPUT_RECORDs → one Event type
├── cell/      # Cell{Rune,Style,Width}, style model (fg/bg/attrs),
│              # color downgrade 24bit→256→16→8, embedded width tables
│              # (generated data files — wide/CJK/zero-width, not a lib dep)
├── render/    # double-buffered grid + dirty-row diffing flusher,
│              # cursor-move optimizer, synchronized-output (DECSET 2026)
│              # = flicker-free, idle CPU = 0
├── layout/    # Flex engine: Row/Col with Fixed/Fill/Percent sizing,
│              # padding/border/title resolution in ONE pass
├── widget/    # built-in components (see 3.3)
└── app.go     # event loop, focus manager, keymap engine, z-order stack
```

### 3.2 Programming model (why it beats Bubble Tea on dev speed)

Bubble Tea makes you write Model + Update(Msg type-switch) + View(string math) and compose lipgloss strings by hand. Our model:

```go
app := tui.New()
root := ui.Col(
    ui.NewTabs(),                                  // built-in component
    ui.Row(ui.NewTree(dir).Fixed(30), editorPane), // flex row
    statusBar,
)
app.SetRoot(root)
app.Keys().Bind("ctrl+s", Actions.Save).
           Bind("ctrl+q", Quit, tui.ScopeGlobal)
app.Run()
```

- Components own their state and handle their own input — no Msg plumbing, no type-switch soup.
- Layout is declarative flex constraints — no manual width arithmetic or string joining.
- Overlays/modals are widgets pushed onto a z-stack — no hand-composited ANSI (deletes the worst part of the old view.go).
- One keymap registry drives key handling **and** palette **and** help hints from the same action table.

### 3.3 Built-in components shipped in v1 of the framework

| Group | Components |
|---|---|
| Primitives | Box (borders/padding/title), Text (wrap/truncate/align), Spacer, Filler |
| Inputs | Input (single-line, history), TextArea (multi-line, selection, clipboard) |
| Lists | ListView (virtualized, filtered), Menu, **CommandPalette** (fuzzy built-in), TreeView, TableView |
| Containers | Tabs, Split (resizable, nestable), ScrollView, Stack (z-order), Modal, Popover (anchor-aware) |
| Feedback | StatusBar, Toasts, ProgressBar, Spinner |
| Platform | Theme system, FocusManager (auto tab-order), KeymapEngine (chords, scopes, sequences, which-key hints) |

That single table replaces essentially all of `bubbles` plus ~2,300 lines of overlay code in old view.go.

### 3.4 Performance design (vs current + vs bubbletea)

| Technique | Effect |
|---|---|
| Retained cell-grid + dirty-row diff | Only changed cells hit the wire; scroll of 10k-line file costs O(viewport) |
| Synchronized output (DECSET 2026) | Zero flicker during composite frames |
| Render-on-dirty only | Idle CPU 0% (old code ticked at 60fps forever) |
| Overlay compositing at cell level | Kills per-frame ANSI re-parsing entirely |

Targets: keystroke→screen <4 ms typical; open 25 MB file <500 ms; memory ≈2× file size; zero allocations per idle frame.

---

## 4. Deliverable B — the editor ("panda"): nano-simple, VS Code-familiar, vim-capable

### 4.1 Learning-curve tiers (the core product decision)

- **Tier 0 — zero learning (default):** just type. Arrows + mouse move cursor. `Ctrl+S/X/C/V/Z/Y` work like everywhere else. A **nano-style persistent hint bar** shows context-aware shortcuts so nothing needs memorizing. `Ctrl+Q` quits with save-confirm dialog.
- **Tier 1 — VS Code muscle memory:** `Ctrl+P` fuzzy finder, `Ctrl+Shift+P` palette, `Ctrl+F/H` search/replace, `Ctrl+B` sidebar, `` Ctrl+` `` terminal, tabs/splits — identical to today's defaults.
- **Tier 2 — discoverability:** command palette lists *every* action with its binding; F1 help overlay auto-generated from the action registry (never stale).
- **Tier 3 — power users, opt-in:** `"vim_mode": true` enables modal editing (normal/visual/visual-block + motions d/c/y/gg/G/w/b/e/f/t/%…) implemented as a keymap layer over the same actions. Vim migrants get `:w`/`:q`/`:` ex-style commands routed into the palette. Nano migrants: `M-\` style meta keys also mapped where they don't conflict.

### 4.2 Drop-in CLI compatibility

```
panda [flags] [file...]     # multiple files → tabs
panda +42 file.txt          # open at line (vim/nano convention)
cat x | panda -             # stdin editing
panda -R file               # read-only (-v alias, vim view mode)
```
Exit codes: 0 success, 1 error, 2 unsaved-abort. Restores terminal perfectly on any exit path (`$EDITOR`-safe).

### 4.3 Text core (`editor/textbuf`) — replaces internal/buffer

- **Gap buffer over bytes + line-offset index**: O(1) amortized local edits, no per-keystroke whole-file copies.
- **Command-pattern undo with coalescing** (typing bursts merge until pause/mode change); memory-bounded, redo preserved across cursor moves.
- **True multi-cursor** in buffer ops (old code replayed ops per cursor from the UI layer).
- Preserves CRLF/LF per file (flag on load, saved back verbatim) + BOM preserved; invalid UTF-8 handled explicitly, not silently mangled.
- Selection model: anchor+head per cursor.

### 4.4 Rendering pipeline

- Per-line highlight cache keyed by `(line-content-hash, block-comment-state-in)` → invalidate only edited lines. Fixes the biggest measured hotspot.
- Viewport paints cached spans straight into the cell grid; search hits/selection/git-gutter are cell-level layers, composited once.

### 4.5 App structure (kills the god object)

```
editor/
├── app.go         # wiring + lifecycle (~200 ln)
├── workspace.go   # tabs, panes, sessions
├── commands.go    # THE action registry (drives keys + palette + help)
├── keymap/        # default map, vim layer, config merge
├── views/         # one file per screen, each a plain tui component
│   editorview / explorer / finder / palette / searchreplace / goto /
│   help / settings / terminal / minimap / statusbar / tabbar /
│   welcome / dialogs
└── textbuf/, highlight/, lsp/, search/, fs/, theme/, config/, session/
```

Adding a new overlay = new component file + one registry entry. (Old cost: touch 5 places.)

### 4.6 Feature parity vs v1

| Kept (v2 launch) | Salvaged nearly as-is | Deferred (post-v2) |
|---|---|---|
| Tabs, split panes, explorer CRUD + gitignore + filter, fuzzy finder, palette, search/replace (+regex, case toggle), global search, go-to-line, undo/redo, multi-cursor, autocomplete (word-based), LSP diagnostics + gutter, git branch + gutter markers, minimap, themes (+custom JSON), settings UI, keybinding editor UI, sessions, autosave, welcome screen, integrated line-shell, AI bundler (Ctrl+Shift+C), recent files, smooth scroll, zoom, relative line numbers, mouse support | `fuzzy`, `session`, `config`, `bundler`, `watcher`, `lsp`, `searcher` packages ported with fixes listed in §2 | Rainbow brackets (old Phase 27), LSP completion/hover (28), side-by-side diff (29), PTY terminal |

---

## 5. Implementation phases

| Phase | Scope | Exit criteria |
|---|---|---|
| **P0** Scaffold | New package tree, drop charm imports from go.mod, CI green | `go build ./...` clean, zero charm deps |
| **P1** term + input | Raw mode both OSes, event parser + golden tests for 200+ byte sequences | Headless fake-tty test harness passes |
| **P2** cell + render | Grid, styles, diffing flusher, color downgrade, width tables | Golden-frame tests; flicker-free resize demo |
| **P3** layout + core widgets | Flex engine, Box/Text/Input/List/Menu/Tabs/Split/Modal/Palette | Demo app exercising each widget |
| **P4** app loop | Focus manager, keymap engine (scopes/chords/sequences), z-stack | Keymap unit tests; which-key hints render |
| **P5** textbuf | Gap buffer, undo coalescing, multi-cursor, CRLF/BOM | Ported 29 buffer tests + new ones pass; fuzz edits stable |
| **P6** Editor MVP | Single pane: open/save/edit/search/goto, hint bar, status bar | Usable as basic `$EDITOR` today |
| **P7** Full shell | Tabs/splits/explorer/finder/palette/global-search/settings/keybind-editor | All §4.6 "kept" features interactive |
| **P8** Intelligence | Highlight cache, LSP diagnostics, git gutter/minimap, terminal panel, AI bundler | Parity walkthrough checklist done |
| **P9** vim_mode + polish | Modal layer, ex commands, themes, sessions, welcome | vim smoke-test script; docs updated |
| **P10** Hardening | Perf benchmarks vs targets, cross-platform (Win/mac/Linux) manual matrix, README/feature.md rewrite | Release build |

Each phase ends runnable; P6 onward you can daily-drive it incrementally.

## 6. Testing strategy

- Unit: textbuf (fuzz + property), input parser (golden sequences), diff renderer (golden frames), keymap, layout solver.
- Integration: headless driver (fake tty) scripting real key streams against the running app.
- Perf: benchmarks for open/edit/scroll/render on synthetic 25 MB file, tracked per phase.

## 7. Risks & mitigations

| Risk | Mitigation |
|---|---|
| Windows console quirks (ConPTY-less raw mode, IME) | P1 lands Windows first-class via INPUT_RECORD path; manual matrix in P10 |
| Width/grapheme edge cases (emoji ZWJ, CJK) | Generated width tables + best-effort clustering documented; golden-frame cases |
| Scope creep delaying MVP | Hard phase gates; P6 = shippable editor before any intelligence features |
| Losing v1 muscle memory | Keymap config importer reads old `~/.panda-editor/keybindings.json` |

---

## 8. Decisions I need from you before starting

1. **Keep `chroma`** for syntax-highlight data (it's independent of Charmbracelet)? Recommended: yes — writing tokenizers for 20+ languages from scratch is weeks of low-value work.
2. **Repo layout:** rewrite in-place in this repo (recommended — git history intact, `refer/` untouched) or fresh repo?
3. **Binary name:** keep `panda`?
4. **Default mode:** pure insert-mode-by-default with vim as opt-in (recommended), or vim-first?
