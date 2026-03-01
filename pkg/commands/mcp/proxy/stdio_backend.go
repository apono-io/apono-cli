package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apono-io/apono-cli/pkg/utils"
)

// STDIOBackend implements Backend for subprocess-based MCP servers
type STDIOBackend struct {
	id      string
	name    string
	btype   string
	command string
	args    []string
	env     map[string]string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr io.ReadCloser

	ready     bool
	requestID int64
	mu        sync.Mutex

	pending   map[interface{}]chan []byte
	pendingMu sync.Mutex

	errCh    chan error
	done     chan struct{}
	doneOnce sync.Once
	procDone chan struct{} // closed when watchProcess finishes (cmd.Wait completed)
}

// STDIOBackendConfig configures a STDIO backend
type STDIOBackendConfig struct {
	ID      string
	Name    string
	Type    string
	Command string
	Args    []string
	Env     map[string]string
}

// NewSTDIOBackend creates a new STDIO backend
func NewSTDIOBackend(cfg STDIOBackendConfig) *STDIOBackend {
	return &STDIOBackend{
		id:       cfg.ID,
		name:     cfg.Name,
		btype:    cfg.Type,
		command:  cfg.Command,
		args:     cfg.Args,
		env:      cfg.Env,
		pending:  make(map[interface{}]chan []byte),
		errCh:    make(chan error, 1),
		done:     make(chan struct{}),
		procDone: make(chan struct{}),
	}
}

func (b *STDIOBackend) ID() string   { return b.id }
func (b *STDIOBackend) Name() string { return b.name }
func (b *STDIOBackend) Type() string { return b.btype }

func (b *STDIOBackend) IsReady() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ready
}

// Start starts the subprocess
func (b *STDIOBackend) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cmd != nil {
		return fmt.Errorf("backend already started")
	}

	b.cmd = exec.Command(b.command, b.args...)

	b.cmd.Env = os.Environ()
	for key, value := range b.env {
		b.cmd.Env = append(b.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	var err error
	b.stdin, err = b.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := b.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	b.stdout = bufio.NewScanner(stdout)
	b.stdout.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max for large query results

	b.stderr, err = b.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := b.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start subprocess: %w", err)
	}

	go b.readStderr()
	go b.readStdout()
	go b.watchProcess()

	return nil
}

func (b *STDIOBackend) watchProcess() {
	defer close(b.procDone)

	if b.cmd == nil || b.cmd.Process == nil {
		return
	}

	err := b.cmd.Wait()

	b.mu.Lock()
	b.ready = false
	b.mu.Unlock()

	// Signal done so Send() unblocks even if readStdout hasn't seen EOF yet
	b.doneOnce.Do(func() { close(b.done) })

	if err != nil {
		utils.McpLogf("[%s] subprocess exited with error: %v", b.id, err)
	} else {
		utils.McpLogf("[%s] subprocess exited normally", b.id)
	}
}

func (b *STDIOBackend) readStderr() {
	scanner := bufio.NewScanner(b.stderr)
	for scanner.Scan() {
		utils.McpLogf("[%s stderr]: %s", b.id, scanner.Text())
	}
}

func (b *STDIOBackend) readStdout() {
	for b.stdout.Scan() {
		line := b.stdout.Bytes()

		lineCopy := make([]byte, len(line))
		copy(lineCopy, line)

		var resp struct {
			ID interface{} `json:"id"`
		}
		if err := json.Unmarshal(lineCopy, &resp); err != nil {
			continue
		}

		b.pendingMu.Lock()
		ch, ok := b.pending[normalizeID(resp.ID)]
		if ok {
			delete(b.pending, normalizeID(resp.ID))
		}
		b.pendingMu.Unlock()

		if ok {
			ch <- lineCopy
		}
	}

	if err := b.stdout.Err(); err != nil {
		select {
		case b.errCh <- err:
		default:
		}
	}

	b.doneOnce.Do(func() { close(b.done) })
}

func normalizeID(id interface{}) interface{} {
	if f, ok := id.(float64); ok {
		return int64(f)
	}
	return id
}

// Initialize sends an initialize request to the backend
func (b *STDIOBackend) Initialize(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req := NewJSONRPCRequest(b.nextID(), "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "apono-proxy",
			"version": "1.0.0",
		},
	})

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal initialize request: %w", err)
	}

	respBytes, err := b.Send(ctx, reqBytes)
	if err != nil {
		return fmt.Errorf("failed to send initialize request: %w", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("failed to parse initialize response: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	// Send the initialized notification
	initializedNotification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	notificationBytes, err := json.Marshal(initializedNotification)
	if err != nil {
		return fmt.Errorf("failed to marshal initialized notification: %w", err)
	}

	b.mu.Lock()
	_, err = fmt.Fprintln(b.stdin, string(notificationBytes))
	b.mu.Unlock()
	if err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	b.mu.Lock()
	b.ready = true
	b.mu.Unlock()

	return nil
}

func (b *STDIOBackend) nextID() int64 {
	return atomic.AddInt64(&b.requestID, 1)
}

// Send sends a JSON-RPC request and waits for the response
func (b *STDIOBackend) Send(ctx context.Context, request []byte) ([]byte, error) {
	var req struct {
		ID interface{} `json:"id"`
	}
	if err := json.Unmarshal(request, &req); err != nil {
		return nil, fmt.Errorf("failed to parse request: %w", err)
	}

	respCh := make(chan []byte, 1)

	b.pendingMu.Lock()
	b.pending[normalizeID(req.ID)] = respCh
	b.pendingMu.Unlock()

	defer func() {
		b.pendingMu.Lock()
		delete(b.pending, normalizeID(req.ID))
		b.pendingMu.Unlock()
	}()

	b.mu.Lock()
	_, err := fmt.Fprintln(b.stdin, string(request))
	b.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-b.done:
		return nil, fmt.Errorf("backend closed")
	case err := <-b.errCh:
		return nil, fmt.Errorf("backend error: %w", err)
	}
}

// Close stops the subprocess
func (b *STDIOBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.ready = false

	if b.stdin != nil {
		b.stdin.Close()
	}

	if b.cmd != nil && b.cmd.Process != nil {
		// Wait for watchProcess goroutine (which calls cmd.Wait) to finish,
		// rather than calling cmd.Wait again which would race/error.
		select {
		case <-b.procDone:
		case <-time.After(5 * time.Second):
			b.cmd.Process.Kill()
			<-b.procDone // wait for watchProcess to observe the kill
		}
	}

	return nil
}

// Health checks if the backend is healthy
func (b *STDIOBackend) Health(ctx context.Context) error {
	if !b.IsReady() {
		return fmt.Errorf("backend not ready")
	}

	req := NewJSONRPCRequest(b.nextID(), "ping", nil)
	reqBytes, _ := json.Marshal(req)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := b.Send(ctx, reqBytes)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// ListTools returns the list of tools from this backend
func (b *STDIOBackend) ListTools(ctx context.Context) ([]Tool, error) {
	client := NewMCPClient(b, b.id, func() interface{} { return b.nextID() })
	return client.ListTools(ctx)
}
