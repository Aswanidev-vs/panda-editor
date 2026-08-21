// Package cli parses the command line into a launch spec, mirroring the
// vim/nano conventions promised in the README: multiple files become tabs,
// +N jumps to a line, -R opens read-only, a lone "-" means read stdin.
// It exports only data; the editor shell consumes the result.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Parse processes argv (without the program name). It follows vim rules:
//
//	panda -h / --help            help text
//	panda -v / --version         version line
//	panda -R file ...            read-only (also --view)
//	panda +42 file               open file at line 42 (1-based)
//	panda -                      read stdin into a new buffer
//	panda file1 file2 ...        one tab per file
//
// Unknown flags return an error describing the offending argument; a "+"
// argument not followed by a digit is a Usage error too. Bare "-" with more
// files afterwards is allowed (stdin tab + file tabs).
//
// Vim-style scanning rules also apply: flags are recognized anywhere in
// argv, not only before the first file, and -R/--view marks every file
// read-only regardless of where it appears relative to the files. The
// non-interactive exit flags (-h/--help, -v/--version) take precedence over
// any file arguments and end parsing. A +N with no file argument after it
// is legal; its line is kept on a FileSpec whose Path is "" so the caller
// can decide the meaning (stdin / empty buffer).
func Parse(argv []string) (*Args, error) {
	a := &Args{}
	var pendingLine int

	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch {
		case arg == "-h" || arg == "--help":
			return &Args{ShowHelp: true}, nil
		case arg == "-v" || arg == "--version":
			return &Args{ShowVersion: true}, nil
		case arg == "-R" || arg == "--view":
			a.ReadOnly = true
		case arg == "-":
			// stdin sentinel: keep the bare "-", never expand it.
			a.Files = append(a.Files, FileSpec{Path: "-", Line: pendingLine})
			pendingLine = 0
		case len(arg) > 1 && arg[0] == '-':
			return nil, fmt.Errorf("unknown flag: %s", arg)
		case arg[0] == '+':
			line, err := strconv.Atoi(arg[1:])
			if err != nil {
				return nil, fmt.Errorf("invalid line number: %s", arg)
			}
			pendingLine = line
		default:
			a.Files = append(a.Files, FileSpec{Path: NormalizePath(arg), Line: pendingLine})
			pendingLine = 0
		}
	}
	if pendingLine != 0 {
		a.Files = append(a.Files, FileSpec{Path: "", Line: pendingLine})
	}
	return a, nil
}

// NormalizePath resolves a file argument for opening. The stdin sentinel
// passes through untouched; "a/b"-style arguments are joined onto the
// current working directory as-is; everything else is made absolute via
// filepath.Abs, which leaves absolute paths (including Windows drive
// letters) unchanged. A leading "~" expands to the user's home directory.
func NormalizePath(arg string) string {
	if arg == "-" {
		return arg
	}
	if p, err := expandTilde(arg); err == nil {
		arg = p
	}
	if filepath.IsAbs(arg) || arg[0] == '/' {
		// Already absolute, or a unixy absolute path: keep as-is.
		return arg
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return arg
	}
	return abs
}

// expandTilde rewrites a leading "~/" (or a bare "~") to the user's home
// directory, leaving everything after it untouched.
func expandTilde(arg string) (string, error) {
	if arg != "~" && !hasTildePrefix(arg) {
		return arg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return arg, err
	}
	if arg == "~" {
		return home, nil
	}
	return filepath.Join(home, arg[2:]), nil
}

func hasTildePrefix(arg string) bool {
	if len(arg) < 3 || arg[0] != '~' {
		return false
	}
	return arg[1] == '/' || arg[1] == filepath.Separator
}
