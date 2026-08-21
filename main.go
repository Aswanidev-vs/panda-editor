package main

import (
	"fmt"
	"os"

	"github.com/Aswanidev-vs/panda-editor/editor/cli"
	"github.com/Aswanidev-vs/panda-editor/editor/shell"
)

const version = "2.0.0"

func main() {
	args, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "panda:", err)
		os.Exit(2)
	}
	if args.ShowVersion {
		fmt.Println("panda", version)
		return
	}
	if args.ShowHelp {
		fmt.Print(helpText)
		return
	}
	opts := shell.Options{
		Args:    args,
		VimMode: os.Getenv("PANDA_VIM") == "1",
		Version: version,
	}
	if err := shell.Run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "panda:", err)
		os.Exit(1)
	}
}

const helpText = `panda — a friendly terminal editor (cherry TUI)

Usage:
  panda                 start with an empty buffer
  panda <file>...       open one tab per file
  panda +N <file>       open <file> with the cursor on line N
  panda -R <file>...    open files read-only
  panda -               read from stdin into a new buffer

Flags:
  -h, --help            show this help
  -v, --version         show the version
  -R, --view            open read-only

Editor keys (insert mode is the default):
  type to edit, ctrl+s to save, ctrl+q to quit, ctrl+f to search,
  ctrl+g to jump to a line, F1 for full help.

Set PANDA_VIM=1 to start in vim-style modal editing.
`
