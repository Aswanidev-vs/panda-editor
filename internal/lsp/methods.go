package lsp

import "encoding/json"

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

func (c *Client) Initialize(rootPath string) error {
	params := InitializeParams{
		ProcessID: 0, // Not used by most servers
		RootPath:  rootPath,
		RootURI:   "file://" + rootPath,
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
			URI:        "file://" + path,
			LanguageID: languageID,
			Version:    1,
			Text:       text,
		},
	}
	return c.Notify("textDocument/didOpen", params)
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
