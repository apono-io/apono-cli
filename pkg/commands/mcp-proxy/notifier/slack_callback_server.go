package notifier

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/apono-io/apono-cli/pkg/utils"
)

// CallbackServer manages the HTTP server for Slack callbacks
type CallbackServer struct {
	server  *http.Server
	handler *CallbackHandler
}

// NewCallbackServer creates a new callback server
func NewCallbackServer(port int, handler *CallbackHandler) *CallbackServer {
	mux := http.NewServeMux()
	mux.HandleFunc("/slack/interactions", handler.HandleInteraction)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &CallbackServer{
		server:  server,
		handler: handler,
	}
}

// Start starts the callback server
func (s *CallbackServer) Start() error {
	utils.McpLogf("Starting Slack callback server on %s", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("callback server error: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *CallbackServer) Shutdown(ctx context.Context) error {
	utils.McpLogf("Shutting down Slack callback server")
	return s.server.Shutdown(ctx)
}

// StartCallbackServer is a convenience function to start the callback server in a goroutine
func StartCallbackServer(port int, handler *CallbackHandler) error {
	server := NewCallbackServer(port, handler)
	return server.Start()
}
