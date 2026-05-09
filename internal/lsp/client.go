package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// Request represents a JSON-RPC request.
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
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

// Client is an LSP client.
type Client struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	idCounter int
	mu       sync.Mutex
	pending  map[int]chan *Response
	onNotif  func(method string, params json.RawMessage)
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

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	c := &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		pending: make(map[int]chan *Response),
		onNotif: onNotif,
	}

	go c.readLoop()
	return c, nil
}

func (c *Client) readLoop() {
	reader := bufio.NewReader(c.stdout)
	for {
		// Read headers
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

		// Read body
		body := make([]byte, contentLength)
		_, err := io.ReadFull(reader, body)
		if err != nil {
			return
		}

		// Parse message
		var msg struct {
			ID     *int            `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(body, &msg); err != nil {
			continue
		}

		if msg.ID != nil {
			// Response
			c.mu.Lock()
			ch, ok := c.pending[*msg.ID]
			delete(c.pending, *msg.ID)
			c.mu.Unlock()

			if ok {
				var resp Response
				json.Unmarshal(body, &resp)
				ch <- &resp
			}
		} else if msg.Method != "" {
			// Notification
			if c.onNotif != nil {
				c.onNotif(msg.Method, msg.Params)
			}
		}
	}
}

func (c *Client) Call(method string, params interface{}) (*Response, error) {
	c.mu.Lock()
	c.idCounter++
	id := c.idCounter
	ch := make(chan *Response, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.send(req); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	return <-ch, nil
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

	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := c.stdin.Write(data); err != nil {
		return err
	}
	return nil
}

func (c *Client) Close() {
	c.stdin.Close()
	c.cmd.Process.Kill()
}
