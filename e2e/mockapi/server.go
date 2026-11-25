// Package mockapi is an in-memory fake of the Apono client API used by the CLI end-to-end tests.
//
// Responses are built from the generated pkg/clientapi models, so a change to the OpenAPI spec
// that breaks the fixtures fails to compile instead of silently drifting.
package mockapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	// DefaultToken is the personal token the mock accepts unless overridden with WithToken.
	DefaultToken = "e2e-personal-token"

	apiPrefix = "/api/client/v1"
	pageLimit = 100
)

// Call is a single request received by the mock.
type Call struct {
	Method        string
	Path          string
	Query         string
	Body          []byte
	Authorization string
	UserAgent     string
}

// MCPHandler answers a proxied MCP JSON-RPC request body.
type MCPHandler func(body []byte) (status int, response []byte)

// Server is a stateful fake Apono backend.
type Server struct {
	*httptest.Server

	t     testing.TB
	token string

	mu         sync.Mutex
	calls      []Call
	unexpected []Call
	faults     map[string]int
	state      *State
	mcpHandler MCPHandler
	nextReqSeq int
}

// Option configures a Server.
type Option func(*Server)

// WithToken overrides the personal token that the mock accepts.
func WithToken(token string) Option {
	return func(s *Server) { s.token = token }
}

// WithState replaces the default fixtures.
func WithState(state *State) Option {
	return func(s *Server) { s.state = state }
}

// New starts a mock server that is closed when the test ends.
func New(t testing.TB, opts ...Option) *Server {
	t.Helper()

	s := &Server{
		t:      t,
		token:  DefaultToken,
		faults: map[string]int{},
		state:  DefaultState(),
		mcpHandler: func(body []byte) (int, []byte) {
			return http.StatusOK, echoMCP(body)
		},
	}
	for _, opt := range opts {
		opt(s)
	}

	s.Server = httptest.NewServer(s.middleware(s.routes()))
	t.Cleanup(s.Close)

	return s
}

// Token returns the personal token the mock accepts.
func (s *Server) Token() string { return s.token }

// Update runs fn on the fixtures under the server lock. Tests must read and change the state
// only through Update: the handlers run on server goroutines, and the race detector cannot see
// that the CLI process has already exited.
func (s *Server) Update(fn func(st *State)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.state)
}

// SetMCPHandler overrides how POST /api/client/v1/mcp answers.
func (s *Server) SetMCPHandler(h MCPHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mcpHandler = h
}

// FailRoute makes every request for "METHOD /path" answer with the given status code.
// The path is the concrete request path, for example "GET /api/client/v1/access-sessions".
func (s *Server) FailRoute(methodAndPath string, status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.faults[methodAndPath] = status
}

// Calls returns every request received so far.
func (s *Server) Calls() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Call(nil), s.calls...)
}

// CallsTo returns the requests received for "METHOD /path".
func (s *Server) CallsTo(methodAndPath string) []Call {
	var result []Call
	for _, c := range s.Calls() {
		if c.Method+" "+c.Path == methodAndPath {
			result = append(result, c)
		}
	}
	return result
}

// Unexpected returns requests that did not match any mocked route.
func (s *Server) Unexpected() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Call(nil), s.unexpected...)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		r.Body = io.NopCloser(strings.NewReader(string(body)))

		call := Call{
			Method:        r.Method,
			Path:          r.URL.Path,
			Query:         r.URL.RawQuery,
			Body:          body,
			Authorization: r.Header.Get("Authorization"),
			UserAgent:     r.Header.Get("User-Agent"),
		}

		s.mu.Lock()
		s.calls = append(s.calls, call)
		status, faulted := s.faults[r.Method+" "+r.URL.Path]
		s.mu.Unlock()

		if r.Header.Get("Authorization") != "Bearer "+s.token {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"status": http.StatusUnauthorized, "message": "unauthorized"})
			return
		}

		if faulted {
			writeJSON(w, status, map[string]any{"status": status, "message": fmt.Sprintf("mock fault %d", status)})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET "+apiPrefix+"/session", s.getUserSession)
	mux.HandleFunc("GET "+apiPrefix+"/cli-notifications", s.listCliNotifications)
	mux.HandleFunc("POST "+apiPrefix+"/analytics", s.accepted)
	mux.HandleFunc("POST "+apiPrefix+"/analytics/bulk", s.accepted)
	mux.HandleFunc("POST "+apiPrefix+"/logs", s.submitLog)

	mux.HandleFunc("GET "+apiPrefix+"/access-sessions", s.listAccessSessions)
	mux.HandleFunc("GET "+apiPrefix+"/access-sessions/{id}", s.getAccessSession)
	mux.HandleFunc("GET "+apiPrefix+"/access-sessions/{id}/access-details", s.getAccessDetails)
	mux.HandleFunc("POST "+apiPrefix+"/access-sessions/{id}/reset-credentials", s.resetCredentials)

	mux.HandleFunc("GET "+apiPrefix+"/access-requests", s.listAccessRequests)
	mux.HandleFunc("POST "+apiPrefix+"/access-requests", s.createAccessRequest)
	mux.HandleFunc("POST "+apiPrefix+"/access-requests/dry-run", s.dryRunAccessRequest)
	mux.HandleFunc("GET "+apiPrefix+"/access-requests/{id}", s.getAccessRequest)
	mux.HandleFunc("POST "+apiPrefix+"/access-requests/{id}/revoke", s.revokeAccessRequest)
	mux.HandleFunc("GET "+apiPrefix+"/access-groups/{id}/units", s.getAccessGroupUnits)

	mux.HandleFunc("GET "+apiPrefix+"/inventory/integrations", s.listIntegrations)
	mux.HandleFunc("GET "+apiPrefix+"/inventory/bundles", s.listBundles)
	mux.HandleFunc("GET "+apiPrefix+"/inventory/resource-types", s.listResourceTypes)
	mux.HandleFunc("GET "+apiPrefix+"/inventory/resources", s.listResources)
	mux.HandleFunc("GET "+apiPrefix+"/inventory/permissions", s.listPermissions)

	mux.HandleFunc("POST "+apiPrefix+"/mcp", s.mcp)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.unexpected = append(s.unexpected, s.calls[len(s.calls)-1])
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]any{"status": http.StatusNotFound, "message": "not mocked: " + r.Method + " " + r.URL.Path})
	})

	return mux
}

func (s *Server) accepted(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) submitLog(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, clientapi.LogEntrySubmissionResponse{Status: "ok"})
}

func (s *Server) getUserSession(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.state.UserSession)
}

func (s *Server) listCliNotifications(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, clientapi.CliNotificationsClientListModel{Notifications: []clientapi.CliNotificationClientModel{}})
}

func (s *Server) mcp(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	handler := s.mcpHandler
	s.mu.Unlock()

	status, response := handler(body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(response)
}

// echoMCP answers any JSON-RPC request with a result that echoes the method name.
func echoMCP(body []byte) []byte {
	var req struct {
		ID     any    `json:"id"`
		Method string `json:"method"`
	}
	_ = json.Unmarshal(body, &req)
	resp, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"result":  map[string]any{"echo": req.Method},
	})
	return resp
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func pagination(count int) clientapi.PaginationClientInfoModel {
	total := int32(count) //nolint:gosec // fixture sizes are tiny
	return clientapi.PaginationClientInfoModel{
		Limit:  pageLimit,
		Offset: 0,
		Total:  *clientapi.NewNullableInt32(&total),
	}
}

// page returns the slice for the requested skip; the CLI stops paging once a page is shorter than the limit.
func page[T any](items []T, r *http.Request) []T {
	var skip int
	_, _ = fmt.Sscanf(r.URL.Query().Get("skip"), "%d", &skip)
	if skip >= len(items) {
		return []T{}
	}
	return items[skip:min(skip+pageLimit, len(items))]
}
