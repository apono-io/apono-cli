//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const mcpRoute = "POST /api/client/v1/mcp"

type jsonRPCResponse struct {
	ID     any            `json:"id"`
	Result map[string]any `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeJSONLines(t *testing.T, result Result) []jsonRPCResponse {
	t.Helper()

	var responses []jsonRPCResponse
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		var resp jsonRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("MCP stdout line is not JSON-RPC: %v\n%s", err, result)
		}
		responses = append(responses, resp)
	}
	return responses
}

func TestMCP_ProxiesJSONRPC(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	cli.Stdin = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"clientInfo":{"name":"e2e-client"}}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n"

	responses := decodeJSONLines(t, cli.MustRun("mcp"))

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}
	if responses[0].Result["echo"] != "initialize" || responses[1].Result["echo"] != "tools/list" {
		t.Fatalf("unexpected responses: %+v", responses)
	}

	calls := srv.CallsTo(mcpRoute)
	if len(calls) != 2 {
		t.Fatalf("expected 2 proxied calls, got %d", len(calls))
	}
	if calls[0].Authorization != "Bearer "+srv.Token() {
		t.Fatalf("MCP call not authenticated: %q", calls[0].Authorization)
	}
	if calls[1].UserAgent != "e2e-client" {
		t.Fatalf("expected the client name from initialize as User-Agent, got %q", calls[1].UserAgent)
	}
}

func TestMCP_AuthErrorsMapToJSONRPCCodes(t *testing.T) {
	for name, tc := range map[string]struct {
		status   int
		wantCode int
	}{
		"unauthorized": {status: http.StatusUnauthorized, wantCode: -32001},
		"forbidden":    {status: http.StatusForbidden, wantCode: -32003},
	} {
		t.Run(name, func(t *testing.T) {
			cli, srv := newLoggedInCLI(t)
			srv.SetMCPHandler(func([]byte) (int, []byte) { return tc.status, []byte(`{}`) })
			cli.Stdin = `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"

			responses := decodeJSONLines(t, cli.MustRun("mcp"))

			if len(responses) != 1 || responses[0].Error == nil || responses[0].Error.Code != tc.wantCode {
				t.Fatalf("expected JSON-RPC error %d, got %+v", tc.wantCode, responses)
			}
		})
	}
}
