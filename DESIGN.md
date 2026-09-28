# PANDA — UI Design Specification

Status: authoritative for the v2 UI redesign. Every surface below is a
requirement, not a suggestion. Implement against this document; if code and
this document disagree, this document wins on *appearance* and the package
doc comments win on *behaviour*.

Scope: visual redesign only. No editor feature, keybinding, or data-path
change. `cherry/` stays domain-agnostic — no editor concept may leak into it.

---

## 1. Direction: "Drafting Table"

The editor presents itself as a **precision instrument**, not a document with
decorations. The metaphor is an architect's drafting table: carbon-dark
substrate, hairline rules instead of heavy borders, measured gutters, small
caps micro-labels, and exactly one hot signal colour that behaves like
drafting ink.

Three rules govern every decision below.

1. **One signal colour.** The accent (`#E8A33D` ochre) is reserved for *UI
   state*: mode badge, active tab, focus, cursor, readouts. **Syntax colours
   never use the accent.** This is what keeps the chrome legible while code
   scrolls under it.
2. **Hierarchy through value, not hue.** Surfaces step from near-black to
   graphite in small, deliberate increments. A surface is distinguishable
   because it is *darker or lighter*, not because it is a different colour.
3. **Hairlines over boxes.** Prefer `┆ ─ ━ │` rules and tick marks to
   `╭─╮` frames. A full frame is reserved for true modals (popups, dialogs),
   which must interrupt.

Explicitly rejected: purple/indigo gradients, Neon Cyberpunk, glassmorphism,
rounded pill buttons, emoji, and Nerd Font glyphs (private-use codepoints
render as tofu on most terminals). Only Box Drawing, Block Elements, Geometric
Shapes and Arrows blocks may be used as glyphs.

---

## 2. Palette

Implemented in `editor/theme`. Two themes ship: **Ink** (dark, default) and
**Vellum** (light). Both are defined by the same semantic field names, so no
widget may hard-code a hex value.

### 2.1 Surfaces (dark → light steps)

| Field | Ink | Vellum | Used by |
|---|---|---|---|
| `Sunken` | `#100E0C` | `#EDE7DB` | gutter, tab strip, hint strip |
| `Base` | `#191614` | `#F7F2E8` | editor text area, status strip |
| `Raised` | `#221D19` | `#FDFAF3` | active tab, popup, dialog |
| `Overlay` | `#2B241E` | `#FFFFFF` | popup interior, selection of controls |
| `Rule` | `#332B23` | `#DCD1BC` | hairlines, separators, tick rules |

### 2.2 Ink (dark theme)

| Field | Hex | Note |
|---|---|---|
| `Fg` | `#E9E3D7` | bone white, warm |
| `FgDim` | `#A79C8B` | secondary text, readout values |
| `FgFaint` | `#6F6558` | comments, gutter numbers, disabled |
| `Accent` | `#E8A33D` | the single signal colour (UI only) |
| `AccentDim` | `#8A6224` | inactive ticks, disabled accent |
| `CursorLine` | `#1F1A15` | active line wash (subtle, +4% luminance) |
| `Selection` | `#3A2F1E` | selection, ochre-tinted so it reads as *ink* |
| `Error` | `#E05B4A` rust | |
| `Warning` | `#E8A33D` ochre | |
| `Success` | `#8FBF6A` moss | |

Syntax (deliberately cool-leaning so code never competes with the ochre
chrome, and low-chroma so dense lines stay calm):

| Token | Hex | Rationale |
|---|---|---|
| `Keyword` | `#E2836A` | terracotta — the "red pencil" |
| `String` | `#93B884` | sage |
| `Number` | `#7FA8C4` | dusty blue |
| `Function` | `#D9C6A2` | pale straw, near-`Fg` — the most frequent token stays quiet |
| `Type` | `#6FB3A8` | muted teal |
| `Operator` | `#B8AFA2` | warm grey |
| `Punct` | `#8A8175` | dimmer still — punctuation recedes |
| `Comment` | `#6F6558` | faint, italic |
| `Builtin` | `#9A8FC4` | one cool violet, reserved for builtins only |

### 2.3 Vellum (light theme)

Surfaces: `Sunken #EDE7DB`, `Base #F7F2E8`, `Raised #FDFAF3`,
`Overlay #FFFFFF`, `Rule #DCD1BC`. Text: `Fg #221E19`, `FgDim #5A5248`,
`FgFaint #948A7B`. Accent: `#A85B12` (burnt ochre). Syntax: `Keyword #9C3B1E`,
`String #4A6B2A`, `Number #2F5C7A`, `Function #6B4E14`, `Type #1F6360`,
`Operator #7A7167`, `Punct #8A8175`, `Comment #948A7B`, `Builtin #5B4B8A`.
Status: `Error #B03A2B`, `Warning #A85B12`, `Success #4A7A2E`.

---

## 3. Layout

The vertical stack is unchanged in *height* (4 chrome rows total) but
completely re-composed in weight:

```
┌─────────────────────────────────────────────────────────┐ row 0
│ ▌ editor/theme/theme.go                    main.go  ▸ × │  TAB STRIP   (Sunken)
├──────────┬──────────────────────────────────────────────┤
│  1    ┆  │  package theme                                │
│  2    ┆  │                                              │  EDITOR
│▌ 3    ┆  │▌ import "fmt"                                 │  (Base)
│  4    ┆  │                                              │
│ …      ┆  │                                              │
├──────────┴──────────────────────────────────────────────┤ row H-2
│ ⌥ insert │ main.go ● │        saved main.go        │ 12:4 │ LF │  STATUS (Base)
├─────────────────────────────────────────────────────────┤ row H-1
│ ⌃G help   ⌃R save as   ⌃F find   ⌃G line   ⌃Q quit     │  HINTS (Sunken)
└─────────────────────────────────────────────────────────┘
```

Weight order, top to bottom: **tab strip (darkest, quietest) → editor
(lightest, widest) → status strip (mid) → hint strip (darkest)**. The editor
is the brightest surface on screen; chrome recedes.

**Tab strip** — `Sunken`. Each tab is ` label ` in `FgFaint`; inactive tabs are
separated by a `·` in `Rule`. The active tab is ` label ` in `Accent` bold on
`Raised` and is preceded by a solid `▌` tick in `Accent`, so the active tab is
located by weight and position rather than by colour alone. Labels come from
`workspace.TabLabel`, which already carries the `•` modified and `RO`
markers — the strip adds no second modified indicator of its own, because two
indicators for one state is noise. The unsaved-changes signal is the blinking
dot in the status strip, which is the one place it appears. When a tab strip
would exceed the width, the leftmost tabs are dropped and a `‹` marker is
shown — never a wrapped or overlapping row.

**Gutter** — three parts, left to right:
1. a 1-column **marker rail**: `▌` in `Accent` on the cursor line, blank
   elsewhere. This is the only place the accent touches the editor body.
2. a right-aligned line number field of width `max(3, digits)` in `FgFaint`;
   the cursor line's number is `Accent` bold. Three is a floor, not a
   preference: a one- or two-digit gutter makes the numbers jitter horizontally
   as the cursor moves, and a jittering number is unreadable.
3. a 1-column **tick rule**: `┆` in `Rule` on every line, upgraded to a solid
   `│` in `AccentDim` on the cursor line. A dashed rule that solidifies on the
   active line reads as a *measurement*, not a border.

The text area starts 2 columns after the rule, so code never touches chrome.
The cursor-line wash spans the **entire row** — rail, numbers and rule included
— in `CursorLine`, so the active line is one continuous band from the left edge
to the right edge rather than a patch that starts halfway across. The rail
marker, number and rule are then painted on top of that band. Lines past EOF
show `·` in `FgFaint` at the text origin instead of an undifferentiated `~`.

**Status strip** — instrument cluster, `Base` background, hairline `│`
dividers in `Rule` between segments. Segments, left to right:
` mode badge ` (uppercase, `Accent` bold, on `Raised`, padded 1 cell each side)
│ `file name` (`Fg`) │ `●` modified dot (blinking `Accent`) │ `[RO]` badge
(`Warning`) │ centred transient message (`FgDim`, italic) │ right cluster:
`12:4` position │ `UTF-8` │ `LF`.
Position is rendered `Ln 12, Col 4` → `12:4` to keep the cluster compact.

Two constraints the layout imposes, both learned from rendering it:

- **The right cluster is a single segment, so it is a single style.** A
  `widget.Segment` carries one style, which means the faint `LN`/`COL` labels
  cannot be styled separately from the bold value. The cluster is therefore
  drawn in the readout value style throughout. Splitting it would need cherry
  to support mixed-style runs in one segment; until it does, the compact form
  carries the hierarchy on its own.
- **The path is elided against a budget the shell computes**
  (`views.ElidePath`). Segments are laid out left to right, so an unshortened
  absolute path claims the whole row and pushes the position off the screen
  entirely on any narrow terminal. The path keeps its tail and is prefixed
  with `…`, because the file name identifies the buffer and the root of the
  path does not.

**Hint strip** — `Sunken`, `FgFaint`, keys in `AccentDim`. The strips must name
the keys panda actually binds (`ctrl+g` help, `ctrl+r` save as, `ctrl+f`
find, `ctrl+g` line — disambiguated as `help` / `line`), not nano's `^O Write
Out`. Keys are uppercased and separated by two spaces; a leading `▸` marks the
strip as chrome.

**Modals** — every overlay (help, welcome, quit confirm) draws first a
**scrim**: the whole viewport is re-blended toward `Sunken` by 65%, so the
editor visibly recedes behind the modal. The shell applies it, between
painting the editor and painting the overlay, because the shell owns the
frame composition; a container widget restyling every cell it is handed is a
side effect no caller expects. The frame is `BorderRounded` in `AccentDim` on
`Overlay`, with the title in `Accent` bold, and a 1-cell `PadT/PadB` so
content never touches the frame.

A dialog **centres itself** at its measured size inside the rect it is given,
like a popup does. The shell hands every overlay the whole viewport, so a
dialog that painted its rect as given would cover the editor entirely and read
as a crash rather than a question. `Measure` and `Draw` must agree exactly on
that size: when they disagree the dialog does not look wrong, it silently
loses its action row off the bottom.

**Welcome splash** — centred column, composed not stacked:
a six-row block-glyph panda mark (see §5), a heavy rule in `Accent` as wide
as the mark, the
wordmark `P  A  N  D  A` in `Fg` bold over a light rule, `version 2.0.0` in
`FgFaint`, a blank row, then a **two-column key grid** (`^O` open │ `^G` help
on one row, `^N` new │ `^Q` quit on the next) with the keys in the dimmed
accent.

**Dialog** — the actions are no longer reverse-video text. Each action is a
`[ label ]` chip: the selected chip is `Accent` bold on `Raised` with a `▸`
pointer, unselected chips are `FgDim` on `Overlay` in brackets. The title
carries the question, the body is `FgDim`.

---

## 4. Motion

The editor redraws only on input events, so motion must be either
terminal-native or event-driven. **No ticker is introduced.**

- **Blink** (`cell.AttrBlink`) is the only animation: the dirty-buffer `●` in
  the tab strip and the modified dot in the status strip. This is the
  deliberate "recording indicator" detail and costs zero redraws.
- **Event-driven state**: the transient message in the status strip is the
  feedback channel for save/search results — it must appear and be readable
  without any timer, so it is plain styled text, never a fade.

If a future version adds a frame clock, the reserved hooks are the spinner in
the hint strip and a cursor-block pulse. Do not pre-build for them.

---

## 5. The one memorable thing

The splash mark: a panda face drawn in Block Elements, ochre on graphite,
sitting above the wordmark. It uses only `▛▜▙▟▀▄█░` (all width 1 in cherry's
table), so it can never break alignment.

Two decisions carry the whole design, and both came out of comparison rather
than intuition:

- **The ears get their own rows, with a blank row between them and the head.**
  Ears fused into the head's top edge read as a bump on a skull.
- **The face needs both eye patches and a solid muzzle.** A hollow outline
  with nothing in the middle reads as a mask; a thin nose alone reads as a
  smudge. The patches are `░`, which in a single ink colour renders lighter
  than the `█` outline, so the mark gets its second tone for free.

```
    ▄▀▄     ▄▀▄    
   █▛▀▜█   █▛▀▜█   
                   
  ▛▀▀▀▀▀▀▀▀▀▀▀▀▀▜  
  █ ░░░ ███ ░░░ █  
  ▙▄▄▄▄▄▄▄▄▄▄▄▄▄▟  
```

Nineteen columns by six rows. The rule beneath it is built from
`theme.MarkWidth` — the *column* count, not `len()` of a row, which for
multi-byte block glyphs is roughly three times the width and would burst out
of the popup.

---

## 6. Implementation contract

`editor/theme` is the single source of truth and must expose:

- `Palette` — every semantic field in §2. Existing field names
  (`Bg`, `Fg`, `Accent`, `LineNum`, `CursorLine`, `Selection`, `Comment`,
  `Keyword`, `String`, `Number`, `Function`, `Type`, `Operator`, `Builtin`,
  `Border`, `GutterBg`, …) are **kept** so existing consumers keep compiling;
  new surfaces use the new fields.
- `Ink` (dark) and `Vellum` (light) as the two palettes; `Dark`/`Light` remain
  as aliases for compatibility. `Current` starts as `Ink`.
- `Glyph` — named constants for every glyph used above. **No literal glyph
  runes in widget code.**
- `Styles` — precomputed `cell.Style` values for recurring roles
  (mode badge, readout label/value, hairline, tick, dim text, scrim target),
  so widgets never assemble styles by hand.
- `Mix(a, b cell.Color, t float64) cell.Color` — the blend used by the scrim
  and by any surface step. `t=0` returns `a`, `t=1` returns `b`.
- `Scrim(r geom.Rect, sc *render.Screen)` — re-blends every cell in `r`
  toward `Sunken`, reading what is already there. This requires
  `render.Screen.CellAt`, which exists.

**Hard constraint:** no `cell.Style` may be captured into a package-level
`var` initialised from `theme.Current`, because the theme is swappable at
runtime and such a `var` goes stale. Resolve styles inside `Draw`, or build
them from `Palette.Roles()`, which belongs to a palette rather than to the
package. This is asserted, not just documented: `TestThemeIsResolvedPerFrame
NotFrozenAtInit` in `editor/shell` switches themes and requires the rendered
cells to change and to change back.

**Constraint on colour fidelity:** the whole point of the palette is exact
shades, so `Mix` must operate in 24-bit RGB and must degrade gracefully to
`cell.Indexed`/`DefaultColor` inputs.

---

## 7. Verifying the result

The UI cannot be reviewed by reading drawing code, because almost every
defect worth catching is a relationship between cells rather than a line of
Go. `_preview/` is a separate module that renders the real editor — real
files, real lexer, real widget tree — into PNGs and into a text dump of the
cell grid:

```
go -C _preview run .              → preview-out/*.png
go -C _preview run . -text        → the frames as text
go -C _preview run . -cols 60     → the narrow-terminal case
```

The text dump is exact about layout, which is what catches misalignment,
truncation and overflow; the PNGs are the only way to judge colour. Both were
needed: the text dump is how the dialog was found filling the entire screen,
and the PNGs are how the composition was checked.

Two rules for the harness itself, learned the hard way: label each frame
*before* printing it, and never screenshot while another agent is mid-edit —
a render of a half-written file looks exactly like a design bug.
