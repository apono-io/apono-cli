package actions

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func TestCreateErrorResponse_escapesServerText(t *testing.T) {
	data := "The Apono CLI doesn't support \"1Password\" accounts.\nGo to the web portal"
	response := createErrorResponse(ErrorCodeAuthorizationFailed, "Unsupported account", data)

	var decoded jsonRPCErrorResponse
	if err := json.Unmarshal([]byte(response), &decoded); err != nil {
		t.Fatalf("response is not valid JSON: %v\n%s", err, response)
	}
	if decoded.JSONRPC != "2.0" || decoded.ID != nil || decoded.Error.Code != ErrorCodeAuthorizationFailed || decoded.Error.Data != data {
		t.Fatalf("unexpected response: %+v", decoded)
	}
}

func TestSendMcpRequest_unsupportedAccount(t *testing.T) {
	const message = "The Apono CLI doesn't support 1Password Privileged Access accounts.\nGo to the web portal"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Apono-Error-Code", "cli_unsupported_account")
		w.WriteHeader(http.StatusForbidden)
		body, _ := json.Marshal(map[string]any{"status": http.StatusForbidden, "message": message})
		_, _ = w.Write(body)
	}))
	defer server.Close()

	httpClient := aponoapi.WithUnsupportedAccountCheck(&http.Client{})
	response, statusCode := sendMcpRequest(server.URL, httpClient, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, "", false)

	if statusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, statusCode)
	}

	var decoded jsonRPCErrorResponse
	if err := json.Unmarshal([]byte(response), &decoded); err != nil {
		t.Fatalf("response is not valid JSON: %v\n%s", err, response)
	}
	if decoded.Error.Code != ErrorCodeAuthorizationFailed || decoded.Error.Data != message {
		t.Fatalf("unexpected response: %+v", decoded)
	}
}
