package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Request represents a JSON-RPC request.
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC response. The ID is kept as raw JSON
// because servers legitimately echo it back either as a number or as a
// string; use IDString to get the normalized form.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// IDString returns the response id in normalized string form ("" if absent).
func (r *Response) IDString() string {
	id, _ := normalizeID(r.ID)
	return id
}

// Error represents a JSON-RPC error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Notification represents a JSON-RPC notification.
type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// DefaultCallTimeout caps how long Call waits for a response. Some servers
// (e.g. misconfigured or absent `gopls`) never reply, and we'd otherwise
// block the editor's startup goroutine forever.
const DefaultCallTimeout = 5 * time.Second

// Client is an LSP client.
type Client struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	idCounter int
	mu        sync.Mutex
	// pending is keyed by the normalized string form of the request id so a
	// response matches no matter whether the server echoes the id back as a
	// number or as a string.
	pending  map[string]chan *Response
	onNotif  func(method string, params json.RawMessage)
	onStderr func(p []byte) // optional sink for server stderr; nil => discarded
	closed   bool
	done     chan struct{}
}

// NewClient starts an LSP server and returns a client.
func NewClient(serverPath string, args []string, onNotif func(string, json.RawMessage)) (*Client, error) {
	cmd := exec.Command(serverPath, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	c := &Client{
		cmd:     cmd,
		pending: make(map[string]chan *Response),
		onNotif: onNotif,
		done:    make(chan struct{}),
	}

	c.start(stdin, stdout, stderr)
	return c, nil
}

// start wires the IO loops up to the given streams. Split out from NewClient
// so tests can drive the framing logic over in-memory pipes instead of a
// real process.
func (c *Client) start(stdin io.WriteCloser, stdout, stderr io.ReadCloser) {
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr
	go c.readLoop()
	if stderr != nil {
		go c.drainStderr()
	}
}

func (c *Client) readLoop() {
	reader := bufio.NewReader(c.stdout)
	for {
		var contentLength int
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length: ") {
				contentLength, _ = strconv.Atoi(strings.TrimPrefix(line, "Content-Length: "))
			}
		}

		if contentLength == 0 {
			continue
		}

		body := make([]byte, contentLength)
		_, err := io.ReadFull(reader, body)
		if err != nil {
			return
		}

		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(body, &msg); err != nil {
			continue
		}

		if msg.ID != nil {
			id, ok := normalizeID(msg.ID)
			if !ok {
				continue
			}
			c.mu.Lock()
			ch, ok := c.pending[id]
			delete(c.pending, id)
			c.mu.Unlock()

			if ok {
				var resp Response
				if err := json.Unmarshal(body, &resp); err == nil {
					ch <- &resp
				}
			}
		} else if msg.Method != "" {
			if c.onNotif != nil {
				c.onNotif(msg.Method, msg.Params)
			}
		}
	}
}

// drainStderr continuously consumes the server's stderr stream. Language
// servers log verbosely; an undrained pipe fills up and blocks the server
// mid-request. Output goes to onStderr when set, otherwise nowhere.
func (c *Client) drainStderr() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-c.done:
			return
		default:
		}
		n, err := c.stderr.Read(buf)
		if n > 0 && c.onStderr != nil {
			p := make([]byte, n)
			copy(p, buf[:n])
			c.onStderr(p)
		}
		if err != nil {
			return
		}
	}
}

// normalizeID reduces a JSON-RPC id (JSON number or string) to one canonical
// string key so responses match pending requests regardless of how the server
// echoes the id back.
func normalizeID(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", false
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", false
		}
		return s, true
	}
	return string(raw), true
}

// Call sends a JSON-RPC request and waits for the response, up to DefaultCallTimeout.
func (c *Client) Call(method string, params interface{}) (*Response, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("lsp: client is closed")
	}
	c.idCounter++
	reqID := c.idCounter
	idKey := strconv.Itoa(reqID)
	ch := make(chan *Response, 1)
	c.pending[idKey] = ch
	c.mu.Unlock()

	req := Request{
		JSONRPC: "2.0",
		ID:      reqID,
		Method:  method,
		Params:  params,
	}

	if err := c.send(req); err != nil {
		c.mu.Lock()
		delete(c.pending, idKey)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(DefaultCallTimeout):
		c.mu.Lock()
		delete(c.pending, idKey)
		c.mu.Unlock()
		return nil, fmt.Errorf("lsp: call %q timed out after %s", method, DefaultCallTimeout)
	}
}

func (c *Client) Notify(method string, params interface{}) error {
	req := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.send(req)
}

func (c *Client) send(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("lsp: client is closed")
	}
	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := c.stdin.Write(data); err != nil {
		return err
	}
	return nil
}

func (c *Client) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	close(c.done)
	c.stdin.Close()
	stdout := c.stdout
	stderr := c.stderr
	cmd := c.cmd
	c.mu.Unlock()

	// Unblock the read/drain loops even if the process lingers; closing an
	// os.File or net.Conn interrupts goroutines blocked in Read.
	if stdout != nil {
		stdout.Close()
	}
	if stderr != nil {
		stderr.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
