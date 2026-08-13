# Panda Editor — Feature Timeline

This document tracks the evolution of Panda Editor. It logs both the
CodeGrab-inspired AI bundling features and the original feature ideas,
as well as stability and security fixes that make the editor production-ready.

---

## ✅ Completed Milestones

### Phase 1: Context Selection
*   **Feature**: Multi-Select File Explorer
*   **Description**: Mark files in the explorer with `Space` to include them in an AI context bundle.
*   **Status**: ✅ Completed. FileTree state updated with `Selected` indicators `[x]`.

### Phase 2: Context Export
*   **Feature**: Bundle Selected for AI (Command Palette)
*   **Description**: Bundles selected files into Markdown with paths and syntax-highlighted code blocks, then copies them to the system clipboard.
*   **Status**: ✅ Completed. Integrated with `atotto/clipboard`.

### Phase 3: Token Estimation & Security
*   **Feature 1**: Status Bar Token Counter (`~XXXt`).
*   **Feature 2**: Basic Secret Redaction.
*   **Status**: ✅ Completed. (Phase 3 secret redaction was later substantially expanded — see Phase 21.)

### Phase 4: Dependency Resolution
*   **Feature**: Auto-Bundle Imports (Go)
*   **Description**: Parses `import` statements with `go/parser` and recursively includes local project dependencies into the LLM bundle.
*   **Status**: ✅ Completed.

### Phase 5: Tree Structure Context
*   **Feature**: Project Tree Output
*   **Description**: The bundle output now includes a `## Project Structure` text tree at the top.
*   **Status**: ✅ Completed.

### Phase 6: .gitignore-Aware File Tree
*   **Feature**: Respect `.gitignore` rules in the file explorer sidebar.
*   **Status**: ✅ Completed.

### Phase 7: Directory Selection (Recursive)
*   **Feature**: Select/deselect entire directories for AI bundling.
*   **Status**: ✅ Completed.

### Phase 8: Multiple Output Formats
*   **Feature**: XML and Plain Text export options. Cycle with `B` in the explorer.
*   **Status**: ✅ Completed.

### Phase 9: Per-File Token Display in Explorer
*   **Feature**: Show `~120t` next to each file. Toggle with `t` in the explorer.
*   **Status**: ✅ Completed.

### Phase 10: Expand/Collapse All Directories
*   **Feature**: Toggle all directories open/closed with `e` in the file tree.
*   **Status**: ✅ Completed.

### Phase 11: Max File Size Filter
*   **Feature**: Skip files larger than 1 MB (configurable) during bundling.
*   **Status**: ✅ Completed. Also enforced when *opening* files (see Phase 22).

### Phase 12: Fuzzy Search in Explorer
*   **Feature**: Press `/` in the file explorer to filter the tree in real time.
*   **Status**: ✅ Completed.

### Phase 13: Bracket Matching & Rainbow Brackets
*   **Feature**: Highlight the matching bracket pair when the cursor is on a bracket.
*   **Status**: ✅ Matching complete. Rainbow brackets (color-coded nesting levels) — see Phase 27 for future work.

### Phase 14: Minimap
*   **Feature**: Bird's-eye code view on the right edge of the editor.
*   **Status**: ✅ Completed.

### Phase 15: Auto-Complete / Snippet Engine
*   **Feature**: Word-based autocomplete drawn from the current buffer; `Tab` or `Enter` to accept.
*   **Status**: ✅ Basic completion complete. LSP-driven completion — see Phase 28.

### Phase 16: Integrated Git Diff View
*   **Feature**: Gutter markers (`+`, `~`, `-`) showing changes relative to `git diff -U0`.
*   **Status**: ✅ Partial — markers in gutter. Side-by-side diff overlay — see Phase 29.

### Phase 17: File CRUD from Explorer
*   **Feature**: `n` create, `R` rename, `d` delete directly from the explorer.
*   **Status**: ✅ Completed.

### Phase 18: Split Panes
*   **Feature**: Vertical split-pane editing. Use `split_pane` from the command palette or add a custom keybinding.
*   **Status**: ✅ Completed. The tab bar highlights both panels' active tabs.

### Phase 19: Terminal Support
*   **Feature**: Integrated shell with `Ctrl+`` toggle.
*   **Status**: ✅ Completed. (Thread-safety hardened — see Phase 20.)

### Phase 20: Stability & Thread-Safety Hardening ⭐
*   **Description**: A pass over the codebase to fix crashes, races, and unsafe state mutations.
*   **Changes**:
    *   `TerminalModel` is now fully guarded by a mutex; all `input` / `output` access goes through helper methods (`SetInput`, `AppendInput`, `InputBackspace`, `PromptView`).
    *   `terminal.Start()` no longer races on `t.cmd` / `t.done`.
    *   Scanner buffer raised to 1 MB to handle long shell output lines.
    *   Git branch / diff polling is skipped when an overlay is open (no fork-bombs in finder mode).
    *   `searchResultChan` is drained between searches to prevent cross-search bleed.
    *   The unsaved-tab prompt no longer accepts `Ctrl+Y` / `Ctrl+N` as `y` / `n` (the old lower-case match triggered on Redo/Undo chord modifiers).
*   **Status**: ✅ Completed.

### Phase 21: Improved Secret Redaction ⭐
*   **Description**: The original redaction only matched JSON-style `"key": "value"`. The new redaction catches more patterns.
*   **Patterns**:
    *   JSON / YAML quoted:    `"api_key": "..."`
    *   YAML unquoted / INI:   `token: superlongtokenvalue12345`
    *   `.env` / shell export: `export API_KEY=...`
    *   HTTP headers:          `Authorization: Bearer <token>`
    *   AWS keys:              `aws_secret_access_key=...`
    *   PEM blocks:            `-----BEGIN PRIVATE KEY-----` (the marker is stripped so the bundle stays useful for documentation while the key body is dropped)
*   **Status**: ✅ Completed. Covered by `internal/bundler/bundler_test.go`.

### Phase 22: File-Size Limit on Open ⭐
*   **Description**: Opening a file larger than 25 MB now returns `ErrFileTooLarge` instead of OOM-ing the editor. The user sees a friendly error message and the buffer is not loaded.
*   **Status**: ✅ Completed. Constant is exposed as `buffer.MaxOpenBytes` so it can be tuned.

### Phase 23: Workspace-Bounded File Operations ⭐
*   **Description**: Delete and rename operations through the explorer now refuse paths that escape the workspace root.
*   **Status**: ✅ Completed via `Editor.withinProject`.

### Phase 24: Path-Safe Delete with Tab Cleanup ⭐
*   **Description**: When deleting a file from the explorer, any open tabs pointing to that path are closed before the delete, preventing dangling references.
*   **Status**: ✅ Completed.

### Phase 25: Auto-Save ⭐
*   **Description**: New `editor.auto_save_interval` config setting (seconds). When `> 0`, dirty buffers with a known file path are saved silently every N seconds while the editor is idle.
*   **Status**: ✅ Completed.

### Phase 26: Smart Backspace ⭐
*   **Description**: `Backspace` now unindents a tab/4-spaces when the cursor is in a line's leading whitespace, mirroring Shift+Tab. Falls back to regular backspace elsewhere.
*   **Status**: ✅ Completed. `buffer.SmartBackspace` is covered by `buffer_test.go`.

### Phase 27: Rainbow Brackets (planned)
*   **Description**: Color nested brackets by depth so deeply nested code is easier to follow.
*   **Status**: 🔲 Planned.

### Phase 28: LSP-Driven Auto-Complete (planned)
*   **Description**: Use LSP `textDocument/completion` for language-aware suggestions instead of just buffer words.
*   **Status**: 🔲 Planned. Note: `textDocument/didChange` is already wired up — see Phase 30.

### Phase 29: Side-by-Side Diff Overlay (planned)
*   **Description**: Open a two-pane diff view of the current file vs. `HEAD`.
*   **Status**: 🔲 Planned.

### Phase 30: LSP `didChange` Notifications ⭐
*   **Description**: The editor now sends `textDocument/didChange` to the LSP server after each save, so diagnostics and hover data stay fresh.
*   **Status**: ✅ Completed.

### Phase 31: Theme Cycle Command ⭐
*   **Feature**: `cycle_theme` command — walks through the built-in dark / light themes and any user themes in `~/.panda-editor/themes/`.
*   **Status**: ✅ Completed.

### Phase 32: Recent Files ⭐
*   **Feature**: Most-recently-opened files are tracked in memory. Use the `recent_files` command from the palette to reopen the latest. The Settings UI also shows up to five recent paths.
*   **Status**: ✅ Completed.

### Phase 33: Case-Sensitive Search Toggle ⭐
*   **Feature**: `toggle_case` command flips `e.caseSensitive`. The status bar now reflects the active state when searching.
*   **Status**: ✅ Completed.

### Phase 34: UTF-8 Safe File Reads ⭐
*   **Description**: Files that aren't valid UTF-8 are now read with a lossy decode instead of corrupting the in-memory buffer.
*   **Status**: ✅ Completed.

### Phase 35: Defensive Clipboard ⭐
*   **Description**: `safeWriteClipboard` now strips embedded NUL bytes before writing. Some hosts (notably on Windows) reject them and can panic; this prevents that crash.
*   **Status**: ✅ Completed.

### Phase 36: Test Suite ⭐
*   **Description**: Added unit tests for the four most logic-heavy packages: `internal/buffer` (29 tests), `internal/bundler` (8 tests), `internal/config` (7 tests), `internal/fuzzy` (9 tests). Run with `go test ./...`.
*   **Status**: ✅ Completed.

---

## 🛡️ Security & Stability Fixes (rolled into Phases 20–35)

*   Removed the `strings.ToLower(msg.String())` bug in the unsaved-tab prompt that accepted `Ctrl+Y` (Redo) as "Yes, save and close".
*   Guarded `os.RemoveAll` in `ActionDelete` with a `withinProject` check.
*   Closed dangling open tabs before deleting their file.
*   Made `safeWriteClipboard` strip NUL bytes that Windows hosts reject.
*   Added a 25 MB hard cap (`buffer.MaxOpenBytes`) on opened files to prevent OOM.
*   Replaced the single-pass secret redaction regex with six patterns covering JSON, YAML, .env, HTTP headers, AWS keys, and PEM blocks.
*   Tightened terminal stdout/stderr scanners to 1 MB line buffers to avoid silent line drops on long shell output.

---

## 🧭 UI Glitches Fixed

*   Tab-bar separator now uses the title-bar background to prevent a tiny color stripe between inactive tabs.
*   Tab bar now highlights both panels' active tabs in split mode (was showing only the left).
*   Settings overlay now has a proper border (was blending into the background).
*   File-tree filter header in non-filter mode shows the active filter text instead of an empty text input.
*   File-tree empty placeholder is centered correctly.
*   Global-search overlay truncation is now multi-byte / ANSI-safe (was previously able to slice mid-rune).
*   Search-overlay match count now honours the `caseSensitive` flag (was hard-coded to case-insensitive).

---

## 💡 Original Feature Ideas (Beyond CodeGrab)

| Idea | Status |
|------|--------|
| Bracket matching | ✅ Phase 13 |
| Minimap | ✅ Phase 14 |
| Auto-Complete | ✅ Phase 15 (basic) / 🔲 Phase 28 (LSP) |
| Integrated Git Diff | ✅ Phase 16 (gutter) / 🔲 Phase 29 (overlay) |
| File Rename / Create / Delete | ✅ Phase 17 |
| Split Panes | ✅ Phase 18 |
| Terminal Support | ✅ Phase 19 |
| Auto-Save | ✅ Phase 25 |
| Smart Backspace | ✅ Phase 26 |
| Theme Cycle | ✅ Phase 31 |
| Recent Files | ✅ Phase 32 |
| Case-Sensitive Search | ✅ Phase 33 |

---

## 🛠️ Architecture Notes

Panda Editor is structured around the Bubble Tea TEA paradigm:

*   `editor/model.go` — core state machine and event loop.
*   `editor/view.go` — UI rendering using `lipgloss`.
*   `editor/filetree.go` — sidebar tree (with `.gitignore` awareness).
*   `editor/terminal.go` — integrated shell with mutex-guarded state.
*   `internal/buffer` — text-buffer primitives (`Open`, `Save`, `Insert*`, `Delete*`, `FindMatchingBracket`, `SmartBackspace`). Now tested.
*   `internal/bundler` — AST-aware AI context generation with secret redaction. Now tested.
*   `internal/config` — unified `config.json` loader with backwards-compatible fallback. Now tested.
*   `internal/fuzzy` — fuzzy match scoring and ranked search. Now tested.
*   `internal/lsp` — minimal JSON-RPC LSP client.
*   `internal/searcher` — multi-threaded project-wide grep.
*   `internal/session` — tab-restore on next launch.
*   `internal/theme` — colour palettes (built-in + user JSON files).
*   `internal/watcher` — `fsnotify`-backed external-file watcher.
*   `internal/highlight` — `chroma`-backed syntax highlighting.

---

## 📋 Test Coverage

```
$ go test ./...
ok  github.com/Aswanidev-vs/panda-editor/internal/buffer     (29 tests)
ok  github.com/Aswanidev-vs/panda-editor/internal/bundler    (8 tests)
ok  github.com/Aswanidev-vs/panda-editor/internal/config     (7 tests)
ok  github.com/Aswanidev-vs/panda-editor/internal/fuzzy      (9 tests)
```