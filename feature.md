# Feature Timeline: CodeGrab Inspiration

This document tracks the integration of features inspired by the `CodeGrab` LLM-bundler into the Panda Editor workspace, as well as original feature ideas for the editor.

---

## ✅ Completed Milestones

### Phase 1: Context Selection
*   **Feature**: Multi-Select File Explorer
*   **Description**: Added the ability to mark files in the file explorer with `Space` to include them in an AI context bundle.
*   **Status**: ✅ Completed. FileTree state updated with `Selected` indicators `[x]`.

### Phase 2: Context Export
*   **Feature**: Bundle Selected for AI (Command Palette)
*   **Description**: Created a command palette action that gathers all selected files, structures them into a cohesive Markdown document (with paths and syntax-highlighted code blocks), and automatically copies them to the system clipboard.
*   **Status**: ✅ Completed. Integrated with `atotto/clipboard`.

### Phase 3: Token Estimation & Security
*   **Feature 1**: Status Bar Token Counter
*   **Description**: Added a live approximation (`~XXXt`) to the editor's status bar for immediate token-weight feedback.
*   **Feature 2**: Basic Secret Redaction
*   **Description**: Automated regex-based redaction step during bundling to sanitize obvious secrets (API keys, tokens).
*   **Status**: ✅ Completed.

### Phase 4: Dependency Resolution
*   **Feature**: Auto-Bundle Imports (Go)
*   **Description**: The bundler parses `import` statements using `go/parser` and recursively includes local project dependencies into the LLM bundle.
*   **Status**: ✅ Completed. Implemented with `go/parser` and AST evaluation.

### Phase 5: Tree Structure Context
*   **Feature**: Project Tree Output
*   **Description**: The bundle output now includes a `## Project Structure` text tree at the top, providing the LLM with spatial awareness of the project layout.
*   **Status**: ✅ Completed.

---

## 🚧 Planned: CodeGrab-Inspired Features

### Phase 6: .gitignore-Aware File Tree
*   **Feature**: Respect `.gitignore` rules in the file explorer sidebar
*   **Description**: Parse the project's `.gitignore` file and automatically hide matching files/directories from the tree (e.g., `node_modules/`, `*.exe`, build artifacts). Add a toggle key (`i`) to show/hide ignored files.
*   **Inspiration**: CodeGrab's `.gitignore` filter toggle (`i` key).
*   **Status**: ✅ Completed. Gitignore parser integrated into `buildTreeWithRules`.

### Phase 7: Directory Selection (Recursive)
*   **Feature**: Select/deselect entire directories for AI bundling
*   **Description**: When a user presses `Space` on a folder, recursively select/deselect all files within it. This massively speeds up the workflow for bundling an entire package.
*   **Inspiration**: CodeGrab's recursive directory selection via `Tab`/`Space`.
*   **Status**: ✅ Completed. `setSelectRecursive` propagates selection to all children.

### Phase 8: Multiple Output Formats
*   **Feature**: XML and Plain Text export options
*   **Description**: In addition to the current Markdown output, support generating bundles in XML (`<file path="...">...</file>`) and plain text formats. Allow cycling formats via a key or Command Palette option.
*   **Inspiration**: CodeGrab's `-f` / `--format` flag and `F` key to cycle formats.
*   **Status**: 🔲 Planned.

### Phase 9: Per-File Token Display in Explorer
*   **Feature**: Show token count next to each file in the sidebar
*   **Description**: Display an estimated token count (e.g., `~120t`) next to each file name in the file tree, so users can quickly gauge context cost before selecting files.
*   **Inspiration**: CodeGrab's `--show-tokens` flag.
*   **Status**: 🔲 Planned.

### Phase 10: Expand/Collapse All Directories
*   **Feature**: Toggle all directories open or closed with a single key
*   **Description**: Press `e` in the file tree to expand all directories recursively, or collapse them all if they're already expanded.
*   **Inspiration**: CodeGrab's `e` key toggle.
*   **Status**: 🔲 Planned.

### Phase 11: Max File Size Filter
*   **Feature**: Skip large files during bundling
*   **Description**: Add a configurable max file size threshold (e.g., 100KB). Files exceeding this limit are automatically excluded from the AI bundle to avoid blowing up the LLM's context window. Show a `[skipped: too large]` indicator.
*   **Inspiration**: CodeGrab's `--max-file-size` option.
*   **Status**: 🔲 Planned.

### Phase 12: Fuzzy Search in Explorer
*   **Feature**: Search files directly within the file tree
*   **Description**: Press `/` while focused on the file explorer to activate an inline fuzzy search that filters the tree in real-time, allowing rapid file discovery without leaving the sidebar.
*   **Inspiration**: CodeGrab's `/` fuzzy search mode.
*   **Status**: 🔲 Planned.

---

## 💡 Original Feature Ideas (Beyond CodeGrab)

### Phase 13: Bracket Matching & Rainbow Brackets
*   **Feature**: Highlight matching bracket pairs and colorize nested brackets
*   **Description**: When the cursor is on a bracket (`{`, `(`, `[`), highlight its matching pair. Optionally, colorize nested bracket levels with different colors for improved readability.
*   **Status**: 🔲 Planned.

### Phase 14: Minimap
*   **Feature**: A scrollable minimap panel on the right edge of the editor
*   **Description**: Show a compressed, bird's-eye view of the entire file on the right side of the editor (like VS Code). Clicking or scrolling on the minimap jumps to that region of the file.
*   **Status**: 🔲 Planned.

### Phase 15: Auto-Complete / Snippet Engine
*   **Feature**: Basic keyword auto-completion and snippet expansion
*   **Description**: When typing, show a dropdown of possible completions drawn from the current file's tokens, LSP suggestions, or user-defined snippets. Press `Tab` to accept.
*   **Status**: 🔲 Planned.

### Phase 16: Integrated Git Diff View
*   **Feature**: Side-by-side or inline diff view for uncommitted changes
*   **Description**: Show added/deleted/modified lines with green/red gutter markers (like VS Code's gutter indicators). Optionally, open a split-pane diff view for the current file vs. the last commit.
*   **Status**: 🔲 Planned.

### Phase 17: File Rename / Create / Delete from Explorer
*   **Feature**: Full CRUD operations on files directly from the sidebar
*   **Description**: Press `n` to create a new file, `R` to rename, `d` to delete (with confirmation prompt). Eliminates the need to leave the editor to manage files.
*   **Status**: 🔲 Planned.

### Phase 18: Split Panes
*   **Feature**: Horizontal/vertical split editor panes
*   **Description**: Allow users to view two files side-by-side within the terminal. Use `Ctrl+|` for vertical split and `Ctrl+-` for horizontal split.
*   **Status**: 🔲 Planned.
### phase 19 :terminal support
*   **Feature**: Terminal support
*   **Description**: Allow users to view two files side-by-side within the terminal. Use `Ctrl+|` for vertical split and `Ctrl+-` for horizontal split.
*   **Status**: 🔲 Planned.