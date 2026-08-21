package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeServer emulates an LSP server speaking Content-Length framing over one
// side of a net.Pipe. It records every incoming method/id and automatically
// answers requests, echoing the request id back as a JSON *string* the way
// many real servers do.
type fakeServer struct {
	conn    net.Conn
	mu      sync.Mutex
	methods []string
	ids     []json.RawMessage
}

func (s *fakeServer) serve() {
	r := bufio.NewReader(s.conn)
	for {
		body, err := readFrame(r)
		if err != nil {
			return
		}
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		json.Unmarshal(body, &msg)

		s.mu.Lock()
		s.methods = append(s.methods, msg.Method)
		s.ids = append(s.ids, msg.ID)
		s.mu.Unlock()

		if msg.ID == nil {
			continue // notification: no reply
		}
		// Echo the id back as a JSON string (what many servers do): decode
		// either a quoted string or a bare number into its text form.
		var idStr string
		if len(msg.ID) > 0 && msg.ID[0] == '"' {
			json.Unmarshal(msg.ID, &idStr)
		} else {
			idStr = string(bytes.TrimSpace(msg.ID))
		}
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      idStr,
			"result":  "ok",
		}
		if _, err := s.conn.Write(frameMsg(resp)); err != nil {
			return
		}
	}
}

func (s *fakeServer) received() ([]string, []json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.methods...), append([]json.RawMessage(nil), s.ids...)
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	length := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			length, _ = strconv.Atoi(strings.TrimPrefix(line, "Content-Length: "))
		}
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

func frameMsg(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return append([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))), data...)
}

// newPipedClient wires a Client to an in-memory fake server.
func newPipedClient(t *testing.T, onNotif func(string, json.RawMessage)) (*Client, *fakeServer) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	c := &Client{
		pending: make(map[string]chan *Response),
		onNotif: onNotif,
		done:    make(chan struct{}),
	}
	c.start(clientConn, clientConn, nil)
	srv := &fakeServer{conn: serverConn}
	go srv.serve()
	t.Cleanup(func() {
		c.Close()
		serverConn.Close()
	})
	return c, srv
}

func TestNormalizeID(t *testing.T) {
	cases := []struct {
		raw    string
		want   string
		wantOK bool
	}{
		{`7`, "7", true},
		{` 42 `, "42", true},
		{`"7"`, "7", true},
		{`"gopls-1"`, "gopls-1", true},
		{`null`, "", false},
		{``, "", false},
	}
	for _, tc := range cases {
		got, ok := normalizeID(json.RawMessage(tc.raw))
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("normalizeID(%s) = (%q, %v), want (%q, %v)", tc.raw, got, ok, tc.want, tc.wantOK)
		}
	}
}

// Regression: servers that echo request ids as strings must still match the
// pending call.
func TestCallMatchesStringResponseID(t *testing.T) {
	c, srv := newPipedClient(t, nil)
	resp, err := c.Call("textDocument/hover", nil)
	if err != nil {
		t.Fatalf("Call with string-id echo failed: %v", err)
	}
	if resp.IDString() != "1" {
		t.Errorf("resp.IDString() = %q, want %q", resp.IDString(), "1")
	}
	if string(resp.Result) != `"ok"` {
		t.Errorf("Result = %s, want %q", resp.Result, `"ok"`)
	}

	// The wire format we send keeps numeric ids.
	methods, ids := srv.received()
	if len(methods) != 1 || methods[0] != "textDocument/hover" {
		t.Fatalf("server saw methods %v, want [textDocument/hover]", methods)
	}
	var n int
	if err := json.Unmarshal(ids[0], &n); err != nil || n != 1 {
		t.Errorf("request id = %s (err %v), want numeric 1", ids[0], err)
	}
}

func TestNotificationsDispatchedWithParams(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	var gotMethod string
	var gotParams json.RawMessage
	done := make(chan struct{})
	c := &Client{pending: make(map[string]chan *Response), done: make(chan struct{})}
	c.onNotif = func(m string, p json.RawMessage) {
		gotMethod, gotParams = m, p
		close(done)
	}
	c.start(clientConn, clientConn, nil)
	defer c.Close()
	defer serverConn.Close()

	diag := PublishDiagnosticsParams{
		URI:         "file:///x/a.go",
		Diagnostics: []Diagnostic{{Range: Range{Start: Position{Line: 1}, End: Position{Line: 1}}, Message: "boom"}},
	}
	params, _ := json.Marshal(diag)
	serverConn.Write(frameMsg(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/publishDiagnostics",
		"params":  json.RawMessage(params),
	}))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("notification not delivered")
	}
	if gotMethod != "textDocument/publishDiagnostics" {
		t.Errorf("method = %q", gotMethod)
	}
	// Diagnostics parsing must be unaffected by the id changes.
	pd, err := ParseDiagnostics(gotParams)
	if err != nil {
		t.Fatalf("ParseDiagnostics: %v", err)
	}
	if pd.URI != diag.URI || len(pd.Diagnostics) != 1 || pd.Diagnostics[0].Message != "boom" {
		t.Errorf("parsed diagnostics = %+v", pd)
	}
}

func TestShutdownExitLifecycle(t *testing.T) {
	c, srv := newPipedClient(t, nil)

	if err := c.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if err := c.Exit(); err != nil {
		t.Fatalf("Exit: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		methods, ids := srv.received()
		if len(methods) >= 2 {
			if methods[0] != "shutdown" {
				t.Errorf("first lifecycle method = %q, want shutdown", methods[0])
			} else if ids[0] == nil {
				t.Error("shutdown should be a request carrying an id")
			}
			if methods[1] != "exit" {
				t.Errorf("second lifecycle method = %q, want exit", methods[1])
			} else if ids[1] != nil {
				t.Error("exit should be a notification without id")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("server only saw %v; expected shutdown then exit", methods)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Regression: a chatty server writing past the OS pipe buffer size to stderr
// must never block, because the client drains it in the background.
func TestStderrDrainedInBackground(t *testing.T) {
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer errW.Close()

	hookGot := make(chan []byte, 256)
	clientConn, serverConn := net.Pipe()
	c := &Client{
		pending: make(map[string]chan *Response),
		done:    make(chan struct{}),
		onStderr: func(p []byte) {
			select {
			case hookGot <- p:
			default:
			}
		},
	}
	c.start(clientConn, clientConn, errR)
	go func() {
		io.Copy(io.Discard, serverConn) // keep request writes flowing
	}()
	t.Cleanup(func() {
		c.Close()
		serverConn.Close()
	})

	payload := bytes.Repeat([]byte("x"), 200*1024) // well past pipe buffers
	writeDone := make(chan error, 1)
	go func() {
		_, werr := errW.Write(payload)
		writeDone <- werr
	}()

	select {
	case werr := <-writeDone:
		if werr != nil {
			t.Fatalf("stderr write failed: %v", werr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stderr write blocked: pipe is not being drained")
	}

	select {
	case <-hookGot:
	case <-time.After(2 * time.Second):
		t.Fatal("stderr hook was never invoked despite drained output")
	}
}

func TestCloseIsIdempotentAndRejectsCalls(t *testing.T) {
	c, _ := newPipedClient(t, nil)
	c.Close()
	c.Close() // must not panic

	if _, err := c.Call("x", nil); err == nil {
		t.Error("Call on closed client should fail")
	}
	if err := c.Notify("x", nil); err == nil {
		t.Error("Notify on closed client should fail")
	}
}
