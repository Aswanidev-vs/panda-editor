# Panda Editor

A fast terminal text editor with a two-minute learning curve — a drop-in replacement for vim/vi/nano where you can just start typing, backed by [cherry](cherry/), a TUI framework built entirely from scratch.

> **Status: the interactive editor is implemented** on the cherry framework. See [`REWRITE_PLAN.md`](REWRITE_PLAN.md) for the full plan and phase gates. The binary opens files, edits, searches, manages tabs, highlights syntax, and offers an opt-in vim mode.

## Why v2

The first version was built on Bubble Tea/Lip Gloss and grew into a 3,800-line god object with full-screen ANSI re-renders every frame. The rewrite replaces all of it:

- **Insert-first editing** — type immediately, like nano or Notepad. No modes required.
- **Visible shortcuts** — a persistent hint bar names the keys the editor actually binds, so nothing needs memorizing.
- **VS Code muscle memory** — `Ctrl+S` save, `Ctrl+C/V/X/Z` clipboard, `Ctrl+O` open, `Ctrl+F` search, `Ctrl+G` goto line, `Ctrl+W` close tab, `Ctrl+N` new buffer, `Ctrl+Q` quit, `F1` help, `Ctrl+PgUp/PgDn` switch tabs.
- **Vim when you want it** — modal editing is an opt-in config flag (`vim_mode`), not a rite of passage.
- **cherry renderer** — a retained cell grid with dirty-row diffing: idle CPU is zero and keystroke-to-screen stays under a few milliseconds even in huge files.

## The interface

The UI was rebuilt around one idea: **a precision instrument, not a document with decorations.** The metaphor is an architect's drafting table — carbon-dark substrate, hairline rules instead of heavy borders, measured gutters, small-caps micro-labels, and a single hot signal colour.

```
 ▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀
 ▌ editor/theme/theme.go                        main.go  ▸ ×     ← tab strip
│  1┆  // A screen is a retained w*h cell grid with dirty-row diffing.    │  ← gutter
│▌ 2┆  // The back buffer is what the frame should look like; the front    │
│  3┆  // buffer is what the terminal is believed to be showing. Flush      │     editor
│  4┆  // diffs the two per dirty row, emitting only changed runs.         │
│  5┆  //                                                                    │
├────────────────────────────────────────────────────────────────────────┤
│ INSERT │ main.go ● │        saved main.go        │ 12:4  UTF-8  LF │      ← status
├────────────────────────────────────────────────────────────────────────┤
│▸ ^G help   ^R save as   ^F find   ^G line   ^Q quit                    │  ← hints
 ▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀
```

- **The gutter is a measurement, not a border.** A marker rail, a right-aligned number field with a three-column floor, and a dashed tick rule that solidifies on the cursor line. The cursor-line wash spans the whole row, so the active line is one continuous band.
- **Exactly one signal colour.** The accent (ochre in dark mode, burnt ochre on paper) is reserved for UI state — mode badge, active tab, focus, cursor, readouts — and is never used for syntax. That is what keeps the chrome legible while code scrolls underneath it.
- **Two themes, one design.** *Ink* (dark, the default) and *Vellum* (light), built from the same semantic field names so no widget hard-codes a shade.
- **Modals interrupt.** Every overlay dims the editor behind it before drawing, so a dialog is the only lit thing on screen.
- **Motion costs nothing.** The editor redraws only on input, so the one animation is the terminal's own blink on the unsaved-changes dot. No frame clock, no idle CPU.

[`DESIGN.md`](DESIGN.md) is the specification: the direction, the full palette with hex values, the layout of every surface, and the reasoning behind the constraints.

### Looking at the UI

Drawing code is a poor way to judge an interface, because almost every defect worth catching is a relationship between cells rather than a line of Go. `_preview/` is a separate module that renders the real editor — real files, real lexer, real widget tree — to PNGs and to a text dump of the cell grid:

```sh
go -C _preview run .              # preview-out/*.png, eight UI states
go -C _preview run . -text        # the frames as text: exact about layout
go -C _preview run . -cols 60     # the narrow-terminal case
go -C _preview run . -marks       # a comparison sheet of splash-mark designs
```

The text dump is how the quit dialog was caught filling the entire screen; the PNGs are the only way to judge colour. It is the cheapest review tool in the repo — use it for any UI change.

## Repository layout

```
panda_editor/
├── main.go            CLI entry point (panda)
├── DESIGN.md          the interface specification
├── editor/            application layer (views, workspace, commands)
│   ├── theme/         design system: surfaces, ink, signal, syntax, glyphs
│   ├── editorview/    the editor body: gutter, text, cursor, selection
│   ├── views/         chrome: status bar, hint bar, popups, dialogs, splash
│   ├── shell/         app root: layout, overlays, key routing
│   ├── highlight/     chroma-backed syntax highlighting
│   ├── document/      buffer wrapper: scroll, selection, undo grouping
│   └── textbuf/       gap-buffer text core: grouped undo, CRLF/BOM-safe IO
├── internal/          stdlib-only support packages
│   ├── lsp/           JSON-RPC LSP client (diagnostics)
│   ├── bundler/       AI context packager (secret-redacting)
│   ├── fuzzy/         fuzzy matcher · searcher/ parallel grep
│   └── watcher/       fsnotify wrapper · session/ · config/
├── _preview/          UI screenshot + text-dump harness (own go.mod)
└── cherry/            independent TUI framework module (own go.mod)
    ├── term/          raw mode: unix termios + Windows VT console
    ├── input/         escape-sequence parser: keys/mouse/paste/kitty
    ├── cell/          cell grid primitives, styles, Unicode width tables
    ├── render/        double-buffered diffing flusher, color downgrade
    ├── layout/        flex solver (fixed/percent/fill)
    └── widget/        component model + built-in components
```

`cherry` is a **nested Go module**: compiler-isolated from the editor, releasable standalone later — the parent wires it in with one `replace` directive. `_preview` is a second nested module, so it can depend on `golang.org/x/image` for font rasterisation without touching the editor's own dependency graph.

## Building

Requires Go 1.23+.

```sh
make build        # produces panda.exe (panda on unix)
./panda -v
```

Or directly: `go build -o panda .`

## Development

```sh
make test          # editor module tests
make test-cherry   # cherry framework tests
make test-all      # everything incl. vet across both modules
```

cherry's layers import strictly downward (`term → input → cell → render → layout → widget → app`); nothing outside cherry imports its internals except through its public API.

Two rules the theme package enforces, because violating either is invisible until it matters:

- **No `cell.Style` in a package-level var.** Styles capture `theme.Current` at init and silently ignore any later theme switch. Resolve them inside `Draw`, or via `Palette.Roles()`, which belongs to a palette rather than to the package. `TestThemeIsResolvedPerFrameNotFrozenAtInit` fails if this regresses.
- **No literal glyph runes in drawing code.** Name them from `theme.Glyph`, and keep every glyph single-width — a two-cell glyph shifts every column after it. `TestEveryGlyphIsOneCellWide` walks the vocabulary by reflection so a glyph added later cannot skip the check.

## Roadmap

Phased delivery, each phase ending runnable — see [`REWRITE_PLAN.md`](REWRITE_PLAN.md):

| Phase | Milestone |
|---|---|
| P0–P5 | cherry core + text engine *(done)* |
| P6 | daily-drivable single-pane editor MVP *(done)* |
| P7 | tabs, search, goto, save/quit flows, chrome bars, syntax highlight *(done)* |
| P8 | LSP diagnostics, git gutter, minimap, AI bundler |
| P9 | opt-in vim mode *(done)*, explorer, sessions |
| P10 | perf benchmarks + cross-platform hardening |

Two themes ship (Ink and Vellum) and the theme is swappable at runtime, but there is no UI for choosing one yet — that belongs with the rest of P9.
