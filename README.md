# Panda Editor 🐼

Panda Editor is a blazingly fast, modern Terminal User Interface (TUI) text editor written entirely in Go using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework. It bridges the gap between classic terminal editors (like Vim or Nano) and modern IDEs (like VS Code), offering a sleek aesthetic, deep language intelligence, and integrated AI bundling tools.

## ✨ Features

- **Modern Aesthetic**: A beautiful UI powered by Lipgloss with customizable themes, glassmorphism-inspired overlays, and smooth animations.
- **Language Intelligence**: 
  - Syntax highlighting for 20+ languages, including **multi-line block comment** (`/* */`) preservation across lines.
  - **LSP Diagnostics**: Integration with `gopls` and other language servers for real-time error reporting (displayed as gutter markers `⊗` and diagnostic hints).
  - **Word-based Autocomplete**: Smart heuristic suggestions as you type.
- **Split-Pane Editing**: Side-by-side editing for multitasking (`Ctrl+\`).
- **Integrated Terminal**: Built-in shell panel (Ctrl+backtick or Alt+backtick) — run commands without leaving the editor.
- **Minimap**: A bird's-eye code view for quick navigation in large files.
- **File Icons**: Language-specific Unicode icons in the tab bar for quick visual identification.
- **File Tree Scrollbar**: Visual scroll indicator when the file list exceeds the viewport.
- **Deep Customization**: Full editor customization via `~/.panda-editor/config.json` — themes, editor settings, per-language LSP config, and keybindings all in one place. Supported by a Settings UI overlay (`Ctrl+,`) and custom theme files (`~/.panda-editor/themes/*.json`).
- **AI-Ready Workflow (Bundler)**:
  - **Multi-Select Explorer**: Mark files with `<kbd>Space</kbd>` to include them in an AI context bundle.
  - **Multiple Formats**: Export bundles as **Markdown**, **XML**, or **Plain Text**.
  - **One-Click Export**: Press `<kbd>Ctrl+Shift+C</kbd>` to bundle all selected files and copy them to your system clipboard instantly.
  - **Auto-Dependency Resolution**: Intelligent parsing of imports to bundle local dependencies.
  - **Token Estimator**: Real-time token-weight estimation in the status bar (~4 characters per token).
- **Git Integration**:
  - Current branch displayed in the status bar.
  - **Gutter Markers**: Visual indicators for Added (`+`), Modified (`~`), and Deleted (`-`) lines relative to the Git index.
- **Explorer CRUD & Search**: 
  - Create (`n`), Rename (`R`), and Delete (`d`) files directly from the explorer.
  - Fuzzy search within the file tree (`/`).
- **Robust Navigation**:
  - Command Palette (`Ctrl+Shift+P`)
  - Fuzzy File Finder (`Ctrl+P`)
  - Global Project Search (`Alt+F`)
  - Multi-tab management and session restoration.
- **VS Code-Like Keybindings**: Familiar default shortcuts that are fully customizable.

## 📦 Installation

Ensure you have Go 1.21+ installed.

### Go Install (Recommended)

```sh
go install github.com/Aswanidev-vs/panda-editor@latest
```

This will compile and install the `panda-editor` binary directly into your `$GOPATH/bin`.

### Build from Source

```sh
git clone https://github.com/Aswanidev-vs/panda-editor.git
cd panda-editor
go build -o panda.exe .
```

Then move `panda.exe` (or `panda` on Linux/macOS) to a directory in your system's `PATH`.

## 🚀 Quick Start

1. Open your terminal and navigate to your project folder.
2. Launch the editor by running:
   ```sh
   panda
   ```
3. Press `Ctrl+B` to open the file explorer and use arrows (or `j`/`k`) to navigate.
4. Press `Space` to select files for AI bundling, then `Ctrl+Shift+C` to copy the context.
5. Press `Ctrl+Shift+P` to open the Command Palette and explore all available actions.

## ⌨️ Keybindings

| Action | Shortcut |
| :--- | :--- |
| **Command Palette** | `Ctrl+Shift+P` |
| **Open File (Fuzzy)** | `Ctrl+P` |
| **Save File** | `Ctrl+S` |
| **Quick Bundle (AI)** | `Ctrl+Shift+C` |
| **Focus Sidebar** | `Ctrl+B` |
| **Split Pane** | `Ctrl+\` |
| **Global Search** | `Alt+F` |
| **Search in File** | `Ctrl+F` |
| **Go To Line** | `Ctrl+G` |
| **New Tab** | `Ctrl+N` |
| **Next Tab** | `Ctrl+Tab` |
| **Toggle Theme** | `Ctrl+T` |
| **Settings UI** | `Ctrl+,` |
| **Reload Config** | via Command Palette |
| **Toggle Terminal** | `` Ctrl+` `` or `` Alt+` `` |
| **Show Help** | `F1` |
| **Quit** | `Ctrl+Q` |

### Explorer Keybindings
- `j` / `k`: Navigate
- `Enter`: Open File / Expand Directory
- `Space`: Select for Bundling
- `e`: Expand/Collapse All
- `/`: Filter Tree
- `n`: New File
- `R`: Rename
- `d`: Delete
- `t`: Toggle Token Display
- `B`: Cycle Bundle Format (Markdown -> XML -> Text)

## ⚙️ Customization

Panda Editor is built around a **unified config file** at `~/.panda-editor/config.json`:

```json
{
  "editor": {
    "tab_size": 4,
    "relative_line_numbers": false,
    "minimap": true
  },
  "theme": "Panda Dark",
  "custom_colors": { "accent": "#ff0000" },
  "languages": {
    "go": { "lsp": "gopls", "tab_size": 4, "format_on_save": true }
  },
  "keybindings": { "save": ["ctrl+s"] },
  "behavior": { "terminal_cmd": "cmd", "lsp_enabled": true }
}
```

Three ways to customize:
1. **Settings UI** — Press `Ctrl+,` or use Command Palette → "Settings UI"
2. **Direct edit** — Command Palette → "Open Config" edits `config.json` in-editor
3. **Custom themes** — Place `.json` theme files in `~/.panda-editor/themes/`

See `guide.md` for detailed customization instructions.

## 🛠️ Architecture

Panda Editor is structured around the Bubble Tea TEA (The Elm Architecture) paradigm:
- `editor/model.go`: The core state machine and event loop.
- `editor/view.go`: The UI rendering engine utilizing `lipgloss`.
- `internal/bundler`: AST-aware AI context generation engine.
- `internal/buffer`: Text buffer management and manipulation primitives.
- `internal/lsp`: Lightweight LSP client for language intelligence.

---
*Built with ❤️ and Go.*
