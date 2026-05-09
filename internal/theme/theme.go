package theme

import "github.com/charmbracelet/lipgloss"

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
