package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name          string
	Bg            lipgloss.Color
	Fg            lipgloss.Color
	Accent        lipgloss.Color
	AccentAlt     lipgloss.Color
	AccentDim     lipgloss.Color
	LineNum       lipgloss.Color
	LineNumActive lipgloss.Color
	CursorLine    lipgloss.Color
	Selection     lipgloss.Color
	Cursor        lipgloss.Color
	StatusBar     lipgloss.Color
	StatusAccent  lipgloss.Color
	Sidebar       lipgloss.Color
	SidebarBg     lipgloss.Color
	TabBg         lipgloss.Color
	TabActiveBg   lipgloss.Color
	TabFg         lipgloss.Color
	TabActiveFg   lipgloss.Color
	Error         lipgloss.Color
	Warning       lipgloss.Color
	Success       lipgloss.Color
	Comment       lipgloss.Color
	Keyword       lipgloss.Color
	String        lipgloss.Color
	Number        lipgloss.Color
	Function      lipgloss.Color
	Type          lipgloss.Color
	Operator      lipgloss.Color
	Builtin       lipgloss.Color
	Border        lipgloss.Color
	TitleBar      lipgloss.Color
	Scrollbar     lipgloss.Color
	GutterBg      lipgloss.Color
	BreadcrumbFg  lipgloss.Color
	OverlayBg     lipgloss.Color
}

// Dark – refined Catppuccin Mocha-inspired palette with panda accents
var Dark = Theme{
	Name:          "Panda Dark",
	Bg:            lipgloss.Color("#1a1b26"),
	Fg:            lipgloss.Color("#c0caf5"),
	Accent:        lipgloss.Color("#7aa2f7"),
	AccentAlt:     lipgloss.Color("#bb9af7"),
	AccentDim:     lipgloss.Color("#3d59a1"),
	LineNum:       lipgloss.Color("#3b4261"),
	LineNumActive: lipgloss.Color("#737aa2"),
	CursorLine:    lipgloss.Color("#1e2030"),
	Selection:     lipgloss.Color("#283457"),
	Cursor:        lipgloss.Color("#c0caf5"),
	StatusBar:     lipgloss.Color("#16161e"),
	StatusAccent:  lipgloss.Color("#7aa2f7"),
	Sidebar:       lipgloss.Color("#a9b1d6"),
	SidebarBg:     lipgloss.Color("#16161e"),
	TabBg:         lipgloss.Color("#16161e"),
	TabActiveBg:   lipgloss.Color("#1a1b26"),
	TabFg:         lipgloss.Color("#565f89"),
	TabActiveFg:   lipgloss.Color("#c0caf5"),
	Error:         lipgloss.Color("#f7768e"),
	Warning:       lipgloss.Color("#e0af68"),
	Success:       lipgloss.Color("#9ece6a"),
	Comment:       lipgloss.Color("#565f89"),
	Keyword:       lipgloss.Color("#bb9af7"),
	String:        lipgloss.Color("#9ece6a"),
	Number:        lipgloss.Color("#ff9e64"),
	Function:      lipgloss.Color("#7aa2f7"),
	Type:          lipgloss.Color("#2ac3de"),
	Operator:      lipgloss.Color("#89ddff"),
	Builtin:       lipgloss.Color("#7dcfff"),
	Border:        lipgloss.Color("#27293d"),
	TitleBar:      lipgloss.Color("#13131e"),
	Scrollbar:     lipgloss.Color("#3b4261"),
	GutterBg:      lipgloss.Color("#16161e"),
	BreadcrumbFg:  lipgloss.Color("#565f89"),
	OverlayBg:     lipgloss.Color("#1a1b26"),
}

// Light – clean, professional light theme
var Light = Theme{
	Name:          "Panda Light",
	Bg:            lipgloss.Color("#fafafa"),
	Fg:            lipgloss.Color("#383a42"),
	Accent:        lipgloss.Color("#4078f2"),
	AccentAlt:     lipgloss.Color("#a626a4"),
	AccentDim:     lipgloss.Color("#b4c7f5"),
	LineNum:       lipgloss.Color("#d4d4d4"),
	LineNumActive: lipgloss.Color("#838387"),
	CursorLine:    lipgloss.Color("#f2f2f2"),
	Selection:     lipgloss.Color("#d7e4fc"),
	Cursor:        lipgloss.Color("#526fff"),
	StatusBar:     lipgloss.Color("#eaeaeb"),
	StatusAccent:  lipgloss.Color("#4078f2"),
	Sidebar:       lipgloss.Color("#383a42"),
	SidebarBg:     lipgloss.Color("#f0f0f0"),
	TabBg:         lipgloss.Color("#eaeaeb"),
	TabActiveBg:   lipgloss.Color("#fafafa"),
	TabFg:         lipgloss.Color("#a0a1a7"),
	TabActiveFg:   lipgloss.Color("#383a42"),
	Error:         lipgloss.Color("#e45649"),
	Warning:       lipgloss.Color("#c18401"),
	Success:       lipgloss.Color("#50a14f"),
	Comment:       lipgloss.Color("#a0a1a7"),
	Keyword:       lipgloss.Color("#a626a4"),
	String:        lipgloss.Color("#50a14f"),
	Number:        lipgloss.Color("#c18401"),
	Function:      lipgloss.Color("#4078f2"),
	Type:          lipgloss.Color("#0184bc"),
	Operator:      lipgloss.Color("#0184bc"),
	Builtin:       lipgloss.Color("#e45649"),
	Border:        lipgloss.Color("#e0e0e0"),
	TitleBar:      lipgloss.Color("#e0e0e2"),
	Scrollbar:     lipgloss.Color("#c4c4c4"),
	GutterBg:      lipgloss.Color("#f5f5f5"),
	BreadcrumbFg:  lipgloss.Color("#a0a1a7"),
	OverlayBg:     lipgloss.Color("#fafafa"),
}

// FromMap builds a Theme from a color name→hex map.
// Only keys present in the map are set; missing fields keep their zero value.
func FromMap(colors map[string]string) Theme {
	return Theme{
		Name:          colors["name"],
		Bg:            lipgloss.Color(colors["bg"]),
		Fg:            lipgloss.Color(colors["fg"]),
		Accent:        lipgloss.Color(colors["accent"]),
		AccentAlt:     lipgloss.Color(colors["accent_alt"]),
		AccentDim:     lipgloss.Color(colors["accent_dim"]),
		LineNum:       lipgloss.Color(colors["line_num"]),
		LineNumActive: lipgloss.Color(colors["line_num_active"]),
		CursorLine:    lipgloss.Color(colors["cursor_line"]),
		Selection:     lipgloss.Color(colors["selection"]),
		Cursor:        lipgloss.Color(colors["cursor"]),
		StatusBar:     lipgloss.Color(colors["status_bar"]),
		StatusAccent:  lipgloss.Color(colors["status_accent"]),
		Sidebar:       lipgloss.Color(colors["sidebar"]),
		SidebarBg:     lipgloss.Color(colors["sidebar_bg"]),
		TabBg:         lipgloss.Color(colors["tab_bg"]),
		TabActiveBg:   lipgloss.Color(colors["tab_active_bg"]),
		TabFg:         lipgloss.Color(colors["tab_fg"]),
		TabActiveFg:   lipgloss.Color(colors["tab_active_fg"]),
		Error:         lipgloss.Color(colors["error"]),
		Warning:       lipgloss.Color(colors["warning"]),
		Success:       lipgloss.Color(colors["success"]),
		Comment:       lipgloss.Color(colors["comment"]),
		Keyword:       lipgloss.Color(colors["keyword"]),
		String:        lipgloss.Color(colors["string"]),
		Number:        lipgloss.Color(colors["number"]),
		Function:      lipgloss.Color(colors["function"]),
		Type:          lipgloss.Color(colors["type"]),
		Operator:      lipgloss.Color(colors["operator"]),
		Builtin:       lipgloss.Color(colors["builtin"]),
		Border:        lipgloss.Color(colors["border"]),
		TitleBar:      lipgloss.Color(colors["title_bar"]),
		Scrollbar:     lipgloss.Color(colors["scrollbar"]),
		GutterBg:      lipgloss.Color(colors["gutter_bg"]),
		BreadcrumbFg:  lipgloss.Color(colors["breadcrumb_fg"]),
		OverlayBg:     lipgloss.Color(colors["overlay_bg"]),
	}
}

// LoadThemeFromFile reads a theme from a JSON file.
// The JSON should be a flat {"name": "...", "bg": "#...", ...} map.
func LoadThemeFromFile(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, err
	}
	var colors map[string]string
	if err := json.Unmarshal(data, &colors); err != nil {
		return Theme{}, err
	}
	t := FromMap(colors)
	if t.Name == "" {
		name := filepath.Base(path)
		t.Name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	return t, nil
}

// MergeColors merges override colors into a base Theme and returns a new Theme.
func MergeColors(base Theme, overrides map[string]string) Theme {
	t := base
	if v, ok := overrides["name"]; ok { t.Name = v }
	if v, ok := overrides["bg"]; ok { t.Bg = lipgloss.Color(v) }
	if v, ok := overrides["fg"]; ok { t.Fg = lipgloss.Color(v) }
	if v, ok := overrides["accent"]; ok { t.Accent = lipgloss.Color(v) }
	if v, ok := overrides["accent_alt"]; ok { t.AccentAlt = lipgloss.Color(v) }
	if v, ok := overrides["accent_dim"]; ok { t.AccentDim = lipgloss.Color(v) }
	if v, ok := overrides["line_num"]; ok { t.LineNum = lipgloss.Color(v) }
	if v, ok := overrides["line_num_active"]; ok { t.LineNumActive = lipgloss.Color(v) }
	if v, ok := overrides["cursor_line"]; ok { t.CursorLine = lipgloss.Color(v) }
	if v, ok := overrides["selection"]; ok { t.Selection = lipgloss.Color(v) }
	if v, ok := overrides["cursor"]; ok { t.Cursor = lipgloss.Color(v) }
	if v, ok := overrides["status_bar"]; ok { t.StatusBar = lipgloss.Color(v) }
	if v, ok := overrides["status_accent"]; ok { t.StatusAccent = lipgloss.Color(v) }
	if v, ok := overrides["sidebar"]; ok { t.Sidebar = lipgloss.Color(v) }
	if v, ok := overrides["sidebar_bg"]; ok { t.SidebarBg = lipgloss.Color(v) }
	if v, ok := overrides["tab_bg"]; ok { t.TabBg = lipgloss.Color(v) }
	if v, ok := overrides["tab_active_bg"]; ok { t.TabActiveBg = lipgloss.Color(v) }
	if v, ok := overrides["tab_fg"]; ok { t.TabFg = lipgloss.Color(v) }
	if v, ok := overrides["tab_active_fg"]; ok { t.TabActiveFg = lipgloss.Color(v) }
	if v, ok := overrides["error"]; ok { t.Error = lipgloss.Color(v) }
	if v, ok := overrides["warning"]; ok { t.Warning = lipgloss.Color(v) }
	if v, ok := overrides["success"]; ok { t.Success = lipgloss.Color(v) }
	if v, ok := overrides["comment"]; ok { t.Comment = lipgloss.Color(v) }
	if v, ok := overrides["keyword"]; ok { t.Keyword = lipgloss.Color(v) }
	if v, ok := overrides["string"]; ok { t.String = lipgloss.Color(v) }
	if v, ok := overrides["number"]; ok { t.Number = lipgloss.Color(v) }
	if v, ok := overrides["function"]; ok { t.Function = lipgloss.Color(v) }
	if v, ok := overrides["type"]; ok { t.Type = lipgloss.Color(v) }
	if v, ok := overrides["operator"]; ok { t.Operator = lipgloss.Color(v) }
	if v, ok := overrides["builtin"]; ok { t.Builtin = lipgloss.Color(v) }
	if v, ok := overrides["border"]; ok { t.Border = lipgloss.Color(v) }
	if v, ok := overrides["title_bar"]; ok { t.TitleBar = lipgloss.Color(v) }
	if v, ok := overrides["scrollbar"]; ok { t.Scrollbar = lipgloss.Color(v) }
	if v, ok := overrides["gutter_bg"]; ok { t.GutterBg = lipgloss.Color(v) }
	if v, ok := overrides["breadcrumb_fg"]; ok { t.BreadcrumbFg = lipgloss.Color(v) }
	if v, ok := overrides["overlay_bg"]; ok { t.OverlayBg = lipgloss.Color(v) }
	return t
}

// ThemeDir returns the path to the user themes directory.
func ThemeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".panda-editor", "themes")
}

// ListUserThemes returns paths to all JSON theme files in the themes directory.
func ListUserThemes() []string {
	dir := ThemeDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	return paths
}

var CurrentTheme = Dark

func SetTheme(t Theme) {
	CurrentTheme = t
}

func ToggleTheme() {
	if CurrentTheme.Name == Dark.Name {
		CurrentTheme = Light
	} else {
		CurrentTheme = Dark
	}
}
