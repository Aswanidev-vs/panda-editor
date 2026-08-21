package cli

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// expectedArg computes the Path a test expects for a file argument: files
// marked as absolute pass through, the stdin sentinel is untouched, and
// everything else is resolved against the test working directory exactly
// like NormalizePath does.
func expectedArg(t *testing.T, arg string, absolute bool) string {
	t.Helper()
	if arg == "-" || arg == "" || absolute {
		return arg
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", arg, err)
	}
	return abs
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		argv     []string
		want     *Args // file paths written as argv-style values
		wantErr  string
		absolute bool // flag every path in want.Files as already absolute
	}{
		{
			name: "empty argv",
			argv: nil,
			want: &Args{},
		},
		{
			name: "short help flag",
			argv: []string{"-h"},
			want: &Args{ShowHelp: true},
		},
		{
			name: "long help flag",
			argv: []string{"--help"},
			want: &Args{ShowHelp: true},
		},
		{
			name: "short version flag beats files",
			argv: []string{"-v", "file.txt"},
			want: &Args{ShowVersion: true},
		},
		{
			name: "long version flag beats files",
			argv: []string{"--version", "a.txt", "b.txt"},
			want: &Args{ShowVersion: true},
		},
		{
			name: "plus line applies to following file",
			argv: []string{"+42", "file.txt"},
			want: &Args{Files: []FileSpec{{Path: "file.txt", Line: 42}}},
		},
		{
			name: "plus one on single file",
			argv: []string{"+1", "single.txt"},
			want: &Args{Files: []FileSpec{{Path: "single.txt", Line: 1}}},
		},
		{
			name: "bare dash is stdin sentinel",
			argv: []string{"-"},
			want: &Args{Files: []FileSpec{{Path: "-"}}},
		},
		{
			name: "read only mixed with files keeps order",
			argv: []string{"-R", "file1", "+3", "file2"},
			want: &Args{
				ReadOnly: true,
				Files: []FileSpec{
					{Path: "file1"},
					{Path: "file2", Line: 3},
				},
			},
		},
		{
			name: "multiple files preserve order",
			argv: []string{"one.txt", "two.txt", "three.txt"},
			want: &Args{Files: []FileSpec{
				{Path: "one.txt"},
				{Path: "two.txt"},
				{Path: "three.txt"},
			}},
		},
		{
			name: "read only after first file applies globally",
			argv: []string{"a.txt", "-R", "b.txt"},
			want: &Args{
				ReadOnly: true,
				Files: []FileSpec{
					{Path: "a.txt"},
					{Path: "b.txt"},
				},
			},
		},
		{
			name: "long view flag equals dash R",
			argv: []string{"--view", "c.txt"},
			want: &Args{ReadOnly: true, Files: []FileSpec{{Path: "c.txt"}}},
		},
		{
			name: "help after first file still wins and stops parsing",
			argv: []string{"a.txt", "--help"},
			want: &Args{ShowHelp: true},
		},
		{
			name: "pending line without file becomes empty path",
			argv: []string{"+42"},
			want: &Args{Files: []FileSpec{{Path: "", Line: 42}}},
		},
		{
			name: "pending line after trailing files",
			argv: []string{"a.txt", "+9"},
			want: &Args{Files: []FileSpec{
				{Path: "a.txt"},
				{Path: "", Line: 9},
			}},
		},
		{
			name:     "absolute path stays as is",
			argv:     []string{"G:/fastbeam/absolute.txt"},
			want:     &Args{Files: []FileSpec{{Path: "G:/fastbeam/absolute.txt"}}},
			absolute: true,
		},
		{
			name:    "unknown flag",
			argv:    []string{"--wat"},
			wantErr: "--wat",
		},
		{
			name:    "unknown flag after file",
			argv:    []string{"a.txt", "-Q"},
			wantErr: "-Q",
		},
		{
			name:    "plus without digits",
			argv:    []string{"+abc"},
			wantErr: "+abc",
		},
		{
			name:    "plus alone is not a line number",
			argv:    []string{"+"},
			wantErr: "+",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.argv)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Parse(%v): expected error containing %q, got nil", tt.argv, tt.wantErr)
				}
				if got != nil {
					t.Fatalf("Parse(%v): expected nil Args on error, got %+v", tt.argv, got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Parse(%v): error %q does not contain %q", tt.argv, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%v): unexpected error: %v", tt.argv, err)
			}
			if got.ShowVersion != tt.want.ShowVersion {
				t.Errorf("ShowVersion = %v, want %v", got.ShowVersion, tt.want.ShowVersion)
			}
			if got.ShowHelp != tt.want.ShowHelp {
				t.Errorf("ShowHelp = %v, want %v", got.ShowHelp, tt.want.ShowHelp)
			}
			if got.ReadOnly != tt.want.ReadOnly {
				t.Errorf("ReadOnly = %v, want %v", got.ReadOnly, tt.want.ReadOnly)
			}
			if len(got.Files) != len(tt.want.Files) {
				t.Fatalf("Files = %#v, want %#v", got.Files, tt.want.Files)
			}
			for i, want := range tt.want.Files {
				wantPath := expectedArg(t, want.Path, tt.absolute)
				if got.Files[i].Path != wantPath || got.Files[i].Line != want.Line {
					t.Errorf("Files[%d] = %+v, want FileSpec{Path: %q, Line: %d}",
						i, got.Files[i], wantPath, want.Line)
				}
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("filepath.Abs(.): %v", err)
	}
	tests := []struct {
		arg  string
		want string
	}{
		{"-", "-"},
		{"a.txt", filepath.Join(cwd, "a.txt")},
		{"sub/file.txt", filepath.Join(cwd, "sub", "file.txt")},
		{"G:/fastbeam/file.txt", "G:/fastbeam/file.txt"},
	}
	if runtime.GOOS != "windows" {
		tests = append(tests, struct{ arg, want string }{"/etc/hosts", "/etc/hosts"})
	}
	for _, tt := range tests {
		if got := NormalizePath(tt.arg); got != tt.want {
			t.Errorf("NormalizePath(%q) = %q, want %q", tt.arg, got, tt.want)
		}
	}
}
