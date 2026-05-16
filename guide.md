# Panda Editor — Keybindings Guide

This guide lists the **default** keybindings as defined in `editor/keybindconfig.go` and grouped by function/category.

> Notes:
> - Some actions have **multiple keys** (e.g. `Ctrl+Left` or `Alt+B`).
> - Key names follow the same format used by the keybinding editor (e.g. `ctrl+shift+s`, `f3`, `shift+tab`).
> - **Tab bar** shows language-specific file icons (Go → ``, Python → ``, JS → ``, etc.).
> - **Multi-line block comments** (`/* */`) are highlighted across lines, not reset per-line.

---

## Navigation (Cursor / Movement)

| Action | Keys |
|---|---|
| Cursor Up | `up` |
| Cursor Down | `down` |
| Cursor Left | `left` |
| Cursor Right | `right` |
| Word Left | `ctrl+left`, `alt+b` |
| Word Right | `ctrl+right`, `alt+f` |
| Line Start | `home` |
| Line End | `end` |
| File Start | `ctrl+home` |
| File End | `ctrl+end` |
| Page Up | `pgup` |
| Page Down | `pgdown` |
| Go to Line | `ctrl+g` |
| Matching Brace | `ctrl+]` |

---

## Editing

| Action | Keys |
|---|---|
| Insert Newline | `enter` |
| Backspace | `backspace` |
| Delete | `delete` |
| Delete Line | `ctrl+shift+k` |
| Duplicate Line | `ctrl+shift+d` |
| Move Line Up | `alt+up` |
| Move Line Down | `alt+down` |
| Indent Line / Selection | `tab` |
| Unindent Line / Selection | `shift+tab` |
| Toggle Comment | `ctrl+/` |
| Undo | `ctrl+z` |
| Redo | `ctrl+y` |

---

## Selection

| Action | Keys |
|---|---|
| Select Up (extend) | `shift+up` |
| Select Down (extend) | `shift+down` |
| Select Left (extend) | `shift+left` |
| Select Right (extend) | `shift+right` |
| Select Word Left | `ctrl+shift+left` |
| Select Word Right | `ctrl+shift+right` |
| Select to Line Start | `shift+home` |
| Select to Line End | `shift+end` |
| Select All | `ctrl+a` |
| Select Line (current line) | `ctrl+l` |

---

## Clipboard

| Action | Keys |
|---|---|
| Copy | `ctrl+c` |
| Cut | `ctrl+x` |
| Paste | `ctrl+v` |

---

## Files & Search

| Action | Keys |
|---|---|
| Save | `ctrl+s` |
| Save As | `ctrl+shift+s` |
| Quick Open (Fuzzy Finder) | `ctrl+p` |
| Focus Explorer (File Tree) | `ctrl+b` |
| Command Palette | `ctrl+shift+p` |
| Search (find in file) | `ctrl+f` |
| Search & Replace | `ctrl+h` |
| Next Match | `f3` |
| Previous Match | `shift+f3` |

---

## Tabs

| Action | Keys |
|---|---|
| Next Tab | `ctrl+tab`, `ctrl+pagedown` |
| Previous Tab | `ctrl+shift+tab`, `ctrl+pageup` |
| Close Tab | `ctrl+w` |
| New Tab | `ctrl+n` |

---

## View (Theme / Zoom / Scrolling)

| Action | Keys |
|---|---|
| Toggle Sidebar | `ctrl+\` |
| Zoom In | `ctrl+=` |
| Zoom Out | `ctrl+-` |
| Toggle Theme (dark/light) | `ctrl+t` |
| Scroll Up | `ctrl+up` |
| Scroll Down | `ctrl+down` |
| Center Cursor | `alt+c` |

---

## General

| Action | Keys |
|---|---|
| Quit | `ctrl+q` |
| Force Quit | `ctrl+shift+q` |
| Keybinding Editor | `ctrl+k` |
| Help Overlay | `f1` |
| Toggle Terminal | `ctrl+j` |

---

# Overlay / Mode Controls (UI-specific)

These apply when you open a special view/overlay.

## Quick Open (Finder) Overlay
- `esc`: cancel / back to normal mode
- `enter`: open selected file
- `up` / `ctrl+k`: move selection up
- `down` / `ctrl+j`: move selection down
- `ctrl+p` / `ctrl+n`: alternate navigation (as shown by UI)
- Matched characters in the filename are highlighted in the accent color
  
## Command Palette Overlay
- `esc`: cancel / back to normal mode
- `enter`: execute selected command
- `up` / `ctrl+k`: move selection up
- `down` / `ctrl+j`: move selection down

## Search Overlay (Find)
- `esc`: cancel / back to normal mode
- `enter`: activate search and move to first match
- `F3`: next match
- `Shift+F3`: previous match

## Search & Replace Overlay
- `esc`: cancel / back to normal mode
- `enter`: find next / activate
- `ctrl+r`: replace current match
- `ctrl+shift+r`: replace all matches
- `tab`: switch focus between “Find” and “Replace” inputs

## Go to Line Overlay
- `esc`: cancel / back to normal mode
- `enter`: jump to the entered line number

## Save As Overlay
- `esc`: cancel / back to normal mode
- `enter`: save to entered path

## Help Overlay
- `esc`: close help

## Keybindings Editor Overlay
- `esc`: close editor / back to normal mode
- `enter`: edit selected keybinding
- In editing mode:
  - type new keys (comma-separated) and `enter` to apply
  - `esc` to cancel editing
- Navigating the list:
  - `up` / `down`
  - additional list navigation is also available via paging/home/end keys

## Open Folder Overlay
- `esc`: cancel / back to normal mode
- `enter`: open folder at entered path

## File Explorer (Sidebar Focus)
- `ctrl+b`: focus file explorer / return to editor
- `esc`: return to editor
- `up` / `k`: move cursor up
- `down` / `j`: move cursor down
- `enter`: open file / toggle folder
- `right` / `l`: expand folder
- `left` / `h`: collapse folder
- `home`: jump to top
- `end`: jump to bottom
- A **scrollbar** (█) appears on the right when files exceed viewport height
