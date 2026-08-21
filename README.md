# Panda Editor

A fast terminal text editor with a two-minute learning curve — a drop-in replacement for vim/vi/nano where you can just start typing, backed by [cherry](cherry/), a TUI framework built entirely from scratch.

> **Status: the interactive editor is implemented** on the cherry framework. See [`REWRITE_PLAN.md`](REWRITE_PLAN.md) for the full plan and phase gates. The binary opens files, edits, searches, manages tabs, highlights syntax, and offers an opt-in vim mode.

## Why v2

The first version was built on Bubble Tea/Lip Gloss and grew into a 3,800-line god object with full-screen ANSI re-renders every frame. The rewrite replaces all of it:

- **Insert-first editing** — type immediately, like nano or Notepad. No modes required.
- **Visible shortcuts** — a persistent hint bar shows context-aware keys, so nothing needs memorizing.
- **VS Code muscle memory** — `Ctrl+S` save, `Ctrl+C/V/X/Z` clipboard, `Ctrl+O` open, `Ctrl+F` search, `Ctrl+G` goto line, `Ctrl+W` close tab, `Ctrl+N` new buffer, `Ctrl+Q` quit, `F1` help, `Ctrl+PgUp/PgDn` switch tabs.
- **Vim when you want it** — modal editing is an opt-in config flag (`vim_mode`), not a rite of passage.
- **cherry renderer** — a retained cell grid with dirty-row diffing: idle CPU is zero and keystroke-to-screen stays under a few milliseconds even in huge files.

## Repository layout

```
panda_editor/
├── main.go            CLI entry point (panda)
├── editor/            application layer (views, workspace, commands)
│   └── textbuf/       gap-buffer text core: grouped undo, CRLF/BOM-safe IO
├── internal/          stdlib-only support packages
│   ├── lsp/           JSON-RPC LSP client (diagnostics)
│   ├── bundler/       AI context packager (secret-redacting)
│   ├── fuzzy/         fuzzy matcher · searcher/ parallel grep
│   ├── watcher/       fsnotify wrapper · session/ · config/
└── cherry/            independent TUI framework module (own go.mod)
    ├── term/          raw mode: unix termios + Windows VT console
    ├── input/         escape-sequence parser: keys/mouse/paste/kitty
    ├── cell/          cell grid primitives, styles, Unicode width tables
    ├── render/        double-buffered diffing flusher, color downgrade
    ├── layout/        flex solver (fixed/percent/fill)
    └── widget/        component model + built-in components
```

cherry is a **nested Go module**: compiler-isolated from the editor, releasable standalone later — the parent wires it in with one `replace` directive.

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

## Roadmap

Phased delivery, each phase ending runnable — see [`REWRITE_PLAN.md`](REWRITE_PLAN.md):

| Phase | Milestone |
|---|---|
| P0–P5 | cherry core + text engine *(done)* |
| P6 | daily-drivable single-pane editor MVP *(done)* |
| P7 | tabs, search, goto, save/quit flows, chrome bars, syntax highlight *(done)* |
| P8 | LSP diagnostics, git gutter, minimap, AI bundler |
| P9 | opt-in vim mode *(done)*, explorer, themes, sessions |
| P10 | perf benchmarks + cross-platform hardening |
