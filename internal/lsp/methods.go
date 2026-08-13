package lsp

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"time"
)

func timeNow() int64 {
	return time.Now().UnixNano()
}

// InitializeParams as defined in LSP spec.
type InitializeParams struct {
	ProcessID int    `json:"processId"`
	RootURI   string `json:"rootUri"`
	RootPath  string `json:"rootPath"`
}

// TextDocumentItem as defined in LSP spec.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// DidOpenTextDocumentParams as defined in LSP spec.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

func fileURI(path string) string {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
	return u.String()
}

func (c *Client) Initialize(rootPath string) error {
	params := InitializeParams{
		ProcessID: 0, // Not used by most servers
		RootPath:  rootPath,
		RootURI:   fileURI(rootPath),
	}
	_, err := c.Call("initialize", params)
	if err != nil {
		return err
	}
	return c.Notify("initialized", struct{}{})
}

func (c *Client) DidOpen(path, languageID, text string) error {
	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        fileURI(path),
			LanguageID: languageID,
			Version:    1,
			Text:       text,
		},
	}
	return c.Notify("textDocument/didOpen", params)
}

// TextDocumentContentChangeEvent as defined in LSP spec.
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

// DidChangeTextDocumentParams as defined in LSP spec.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// VersionedTextDocumentIdentifier as defined in LSP spec.
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// DidChange sends a full textDocument/didChange notification to the server.
// We send the entire document text on every change (cheaper than implementing
// incremental sync for a hobbyist editor).
func (c *Client) DidChange(path, languageID, text string) error {
	version := int(timeNow()) // simple monotonic counter
	params := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI:     fileURI(path),
			Version: version,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: text},
		},
	}
	return c.Notify("textDocument/didChange", params)
}

// Diagnostics notification.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity,omitempty"`
	Message  string `json:"message"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

func ParseDiagnostics(params json.RawMessage) (*PublishDiagnosticsParams, error) {
	var p PublishDiagnosticsParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
