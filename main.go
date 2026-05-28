package main

import (
	"fmt"
	"os"

	"github.com/Aswanidev-vs/panda-editor/editor"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	e := editor.NewEditor()
	if len(os.Args) > 1 {
		editor.OpenFileInEditor(&e, os.Args[1])
	}
	p := tea.NewProgram(&e, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

