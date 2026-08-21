package render

import "strings"

// DetectCapabilities derives the color fidelity and synchronized-update support
// from environment variables (pass os.Getenv). Precedence follows the common
// convention:
//
//  1. COLORTERM containing "truecolor" or "24bit" -> ColorRGB
//  2. TERM containing "256color"                  -> Color256
//  3. WT_SESSION set (Windows Terminal)           -> ColorRGB
//  4. TERM_PROGRAM naming vscode / iTerm / WezTerm /
//     kitty / ghostty / alacritty                 -> ColorRGB
//  5. otherwise                                   -> Color16
//
// Synchronized updates (DECSET 2026 bracketing) are enabled conservatively:
// only for terminals known to support them (kitty, ghostty, alacritty,
// wezterm, iterm, foot, contour) or Windows Terminal. Unknown terminals simply
// ignore DECSET 2026, so enabling there would be harmless — but staying quiet
// keeps byte streams minimal on dumb transports.
func DetectCapabilities(getenv func(string) string) (mode ColorMode, syncOutput bool) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	colorterm := getenv("COLORTERM")
	term := getenv("TERM")
	program := getenv("TERM_PROGRAM")

	switch {
	case containsFold(colorterm, "truecolor") || containsFold(colorterm, "24bit"):
		mode = ColorRGB
	case strings.Contains(term, "256color"):
		mode = Color256
	case getenv("WT_SESSION") != "":
		mode = ColorRGB
	case termProgramTruecolor(program):
		mode = ColorRGB
	default:
		mode = Color16
	}

	syncOutput = getenv("WT_SESSION") != ""
	if !syncOutput {
		hay := strings.ToLower(term + "\x00" + colorterm + "\x00" + program)
		for _, t := range [...]string{"kitty", "ghostty", "alacritty", "wezterm", "iterm", "foot", "contour"} {
			if strings.Contains(hay, t) {
				syncOutput = true
				break
			}
		}
	}
	return mode, syncOutput
}

func termProgramTruecolor(program string) bool {
	p := strings.ToLower(program)
	for _, t := range [...]string{"vscode", "iterm", "wezterm", "kitty", "ghostty", "alacritty"} {
		if strings.Contains(p, t) {
			return true
		}
	}
	return false
}

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), sub)
}
