//go:build e2e

package e2e

import (
	"testing"

	"github.com/apono-io/apono-cli/e2e/mockapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const getSessionRoute = "GET /api/client/v1/access-sessions/" + mockapi.PostgresSessionID

func TestAccess_List(t *testing.T) {
	cli, srv := newLoggedInCLI(t)

	sessions := decodeJSON[[]clientapi.AccessSessionClientModel](t, cli.MustRun("access", "list", "-o", "json"))
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	filtered := decodeJSON[[]clientapi.AccessSessionClientModel](t,
		cli.MustRun("access", "list", "-i", mockapi.PostgresIntegrationType+"/"+mockapi.PostgresIntegrationName, "-o", "json"))
	if len(filtered) != 1 || filtered[0].Id != mockapi.PostgresSessionID {
		t.Fatalf("integration filter not applied: %+v", filtered)
	}
	calls := srv.CallsTo("GET /api/client/v1/access-sessions")
	assertContains(t, calls[len(calls)-1].Query, "integration_id="+mockapi.PostgresIntegrationID)

	table := cli.MustRun("access", "list")
	assertContains(t, table.Stdout, "INTEGRATION NAME")
	assertContains(t, table.Stdout, mockapi.PostgresSessionID)
}

func TestAccess_ListEmpty(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	srv.Update(func(st *mockapi.State) { st.Sessions = nil })

	cli.MustFail("no active access found", "access", "list")
}

func TestAccess_UseOutputFormats(t *testing.T) {
	cli, _ := newLoggedInCLI(t)

	instructions := cli.MustRun("access", "use", mockapi.PostgresSessionID)
	assertContains(t, instructions.Stdout, "Run: psql")

	command := cli.MustRun("access", "use", mockapi.PostgresSessionID, "-o", "cli")
	assertContains(t, command.Stdout, `psql "postgresql://apono_e2e:e2e-password@localhost:5432/e2e"`)

	details := decodeJSON[map[string]any](t, cli.MustRun("access", "use", mockapi.PostgresSessionID, "-o", "json"))
	if details["command"] == nil {
		t.Fatalf("unexpected json details: %v", details)
	}

	cli.MustFail("unsupported output format: link", "access", "use", mockapi.PostgresSessionID, "-o", "link")
	cli.MustFail("access session with id missing not found", "access", "use", "missing")
}

func TestAccess_CustomInstructionMessage(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	message := "Connect through the bastion"
	srv.Update(func(st *mockapi.State) {
		st.Session(mockapi.PostgresSessionID).Details.Instructions.CustomMessage = *clientapi.NewNullableString(&message)
	})

	result := cli.MustRun("access", "use", mockapi.PostgresSessionID)
	assertContains(t, result.Stdout, "Message from your admin: "+message)
}

func TestAccess_ResetCredentials(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	srv.Update(func(st *mockapi.State) { st.Session(mockapi.PostgresSessionID).ResetPolls = 2 })

	result := cli.MustRun("access", "reset-credentials", mockapi.PostgresSessionID)
	assertContains(t, result.Stdout, "credentials reset finished successfully")

	if n := len(srv.CallsTo("POST /api/client/v1/access-sessions/" + mockapi.PostgresSessionID + "/reset-credentials")); n != 1 {
		t.Fatalf("expected 1 reset call, got %d", n)
	}
	if n := len(srv.CallsTo(getSessionRoute)); n < 3 {
		t.Fatalf("expected the CLI to poll until the credentials are new, got %d polls", n)
	}
}
