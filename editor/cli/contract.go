// Package cli parses the command line into a launch spec, mirroring the
// vim/nano conventions promised in the README: multiple files become tabs,
// +N jumps to a line, -R opens read-only, a lone "-" means read stdin.
// It exports only data; the editor shell consumes the result.
package cli

// Args is the parsed launch specification.
type Args struct {
	// Files are absolute-ish paths in argv order; each becomes a tab.
	// Path "-" is the stdin sentinel. Path "" with Line > 0 is a deferred
	// "+N" with no following file: open an empty buffer and jump there.
	Files []FileSpec

	// ShowVersion / ShowHelp select non-interactive exits handled by main;
	// Parse returns immediately with only these set when either flag is seen.
	ShowVersion bool
	ShowHelp    bool

	// ReadOnly applies to every opened file (-R or --view).
	ReadOnly bool
}

// FileSpec is one file argument plus its optional starting line.
type FileSpec struct {
	Path string
	Line int // 1-based; 0 means default
}
