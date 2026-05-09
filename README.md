# Panda Editor 🐼

Panda Editor is a blazingly fast, modern Terminal User Interface (TUI) text editor written entirely in Go using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework. It bridges the gap between classic terminal editors (like Vim or Nano) and modern IDEs (like VS Code), offering a sleek aesthetic, deep language intelligence, and integrated AI bundling tools.

## ✨ Features

- **Modern Aesthetic**: A beautiful UI powered by Lipgloss with customizable themes, glassmorphism-inspired overlays, and smooth animations.
- **Language Intelligence**: Syntax highlighting for 200+ languages via Chroma, and built-in Language Server Protocol (LSP) support (pre-configured for `gopls`).
- **AI-Ready Workflow (CodeGrab Integration)**:
  - **Multi-Select Explorer**: Mark files with `<kbd>Space</kbd>` to include them in an AI context bundle.
  - **Auto-Dependency Resolution**: Automatically parses Go imports and bundles local dependencies.
  - **One-Click Export**: Use the Command Palette to instantly generate a Markdown bundle of your selected files directly to your system clipboard.
  - **Token Estimator**: Live token count estimation in the status bar.
  - **Secret Redaction**: Automatically scrubs obvious API keys and tokens before copying.
- **Robust Navigation**:
  - Command Palette (`Ctrl+Shift+P`)
  - Fuzzy File Finder (`Ctrl+P`)
  - Global Search across the project
  - Multi-tab management
- **VS Code-Like Keybindings**: Familiar default shortcuts (`Ctrl+S` to save, `Ctrl+B` for sidebar, etc.) that are fully customizable.

## 📦 Installation

To install Panda Editor from source, ensure you have Go 1.21+ installed.

```sh
git clone https://github.com/Aswanidev-vs/panda-editor.git
cd panda-editor
go build -o panda.exe .
```

Then move `panda.exe` to a directory in your system's `PATH`.

## 🚀 Quick Start

1. Open your terminal and navigate to your project folder.
2. Launch the editor by running:
   ```sh
   panda
   ```
   *Optionally, pass a file or directory path as an argument.*
3. Press `Ctrl+B` to open the file explorer and use arrows (or `j`/`k`) to navigate.
4. Press `Enter` to open a file.
5. Press `Ctrl+Shift+P` to open the Command Palette and explore available actions.

## ⌨️ Keybindings

Panda Editor comes with familiar defaults that can be customized in the Keybindings menu (`Ctrl+Shift+P` -> `Keyboard Shortcuts`).

| Action | Shortcut |
| :--- | :--- |
| **Command Palette** | `Ctrl+Shift+P` |
| **Open File (Fuzzy)** | `Ctrl+P` |
| **Save File** | `Ctrl+S` |
| **Focus Explorer** | `Ctrl+B` |
| **Toggle Sidebar (Vis)**| `Ctrl+\` |
| **Global Search** | `Alt+F` |
| **New Tab** | `Ctrl+N` |
| **Next Tab** | `Ctrl+Tab` or `Ctrl+PageDown` |
| **Close Tab** | `Ctrl+W` |
| **Toggle Theme** | `Ctrl+T` |
| **Bundle for AI** | `Space` (in Explorer) to select, then use Command Palette |
| **Quit** | `Ctrl+Q` |

## 🛠️ Architecture

Panda Editor is structured around the Bubble Tea TEA (The Elm Architecture) paradigm:
- `editor/model.go`: The core state machine and event loop.
- `editor/view.go`: The UI rendering engine utilizing `lipgloss`.
- `internal/bundler`: The AST-aware AI context generation engine.
- `internal/buffer`: The text buffer management and manipulation primitives.

---
*Built with ❤️ and Go.*
