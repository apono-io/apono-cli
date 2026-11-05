//go:build e2e

package e2e

import (
	"encoding/json"
	"testing"

	"github.com/apono-io/apono-cli/e2e/mockapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	createRequestRoute = "POST /api/client/v1/access-requests"
	dryRunRoute        = "POST /api/client/v1/access-requests/dry-run"
)

func lastCreateRequestBody(t *testing.T, srv *mockapi.Server) clientapi.CreateAccessRequestClientModel {
	t.Helper()

	calls := srv.CallsTo(createRequestRoute)
	if len(calls) == 0 {
		t.Fatal("CLI did not call " + createRequestRoute)
	}

	var body clientapi.CreateAccessRequestClientModel
	if err := json.Unmarshal(calls[len(calls)-1].Body, &body); err != nil {
		t.Fatalf("create request body does not match the API model: %v\n%s", err, calls[len(calls)-1].Body)
	}
	return body
}

func TestRequests_ListAndDescribe(t *testing.T) {
	cli, _ := newLoggedInCLI(t)

	requests := decodeJSON[[]clientapi.AccessRequestClientModel](t, cli.MustRun("requests", "list", "-o", "json"))
	if len(requests) != 1 || requests[0].Id != mockapi.ActiveRequestID {
		t.Fatalf("unexpected requests: %+v", requests)
	}

	request := decodeJSON[clientapi.AccessRequestClientModel](t, cli.MustRun("requests", "describe", mockapi.ActiveRequestID, "-o", "json"))
	if request.Status.Status != "Active" {
		t.Fatalf("unexpected status %q", request.Status.Status)
	}

	table := cli.MustRun("requests", "list")
	assertContains(t, table.Stdout, "REQUEST ID")
	assertContains(t, table.Stdout, mockapi.ActiveRequestID)

	cli.MustFail("404 Not Found", "requests", "describe", "req-missing")
}

func TestRequests_AccessUnits(t *testing.T) {
	cli, _ := newLoggedInCLI(t)

	units := decodeJSON[[]clientapi.AccessUnitClientModel](t, cli.MustRun("requests", "units", mockapi.ActiveRequestID, "-o", "json"))
	if len(units) != 1 || units[0].Permission.Id != mockapi.PostgresPermissionID {
		t.Fatalf("unexpected access units: %+v", units)
	}
}

func TestRequests_CreateForBundle(t *testing.T) {
	cli, srv := newLoggedInCLI(t)

	created := decodeJSON[clientapi.AccessRequestClientModel](t,
		cli.MustRun("requests", "create", "--bundle", mockapi.BundleName, "--justification", "e2e bundle", "-o", "json"))
	if created.Status.Status != "Active" {
		t.Fatalf("unexpected status %q", created.Status.Status)
	}

	body := lastCreateRequestBody(t, srv)
	if len(body.FilterBundleIds) != 1 || body.FilterBundleIds[0] != mockapi.BundleID {
		t.Fatalf("bundle name was not resolved to its id: %+v", body.FilterBundleIds)
	}
	if got := body.Justification.Get(); got == nil || *got != "e2e bundle" {
		t.Fatalf("unexpected justification %v", got)
	}
	if len(srv.CallsTo(dryRunRoute)) != 1 {
		t.Fatal("create did not validate with a dry run first")
	}

	requests := decodeJSON[[]clientapi.AccessRequestClientModel](t, cli.MustRun("requests", "list", "-o", "json"))
	if len(requests) != 2 || requests[0].Id != created.Id {
		t.Fatalf("new request missing from list: %+v", requests)
	}
}

func TestRequests_CreateForIntegrationResources(t *testing.T) {
	cli, srv := newLoggedInCLI(t)

	cli.MustRun("requests", "create",
		"--integration", mockapi.PostgresIntegrationType+"/"+mockapi.PostgresIntegrationName,
		"--resource-type", mockapi.PostgresResourceTypeID,
		"--resources", mockapi.PostgresResourceSource,
		"--permissions", mockapi.PostgresPermissionID,
		"--justification", "e2e integration",
		"--duration", "1h",
		"--custom-field", "ticket=OPS-1",
		"-o", "json",
	)

	body := lastCreateRequestBody(t, srv)
	if len(body.FilterIntegrationIds) != 1 || body.FilterIntegrationIds[0] != mockapi.PostgresIntegrationID {
		t.Fatalf("unexpected integration ids %v", body.FilterIntegrationIds)
	}
	if len(body.FilterResources) != 1 || body.FilterResources[0].Value != mockapi.PostgresResourceID {
		t.Fatalf("resource source id was not resolved to its id: %+v", body.FilterResources)
	}
	if len(body.FilterPermissionIds) != 1 || body.FilterPermissionIds[0] != mockapi.PostgresPermissionID {
		t.Fatalf("unexpected permissions %v", body.FilterPermissionIds)
	}
	if got := body.DurationInSec.Get(); got == nil || *got != 3600 {
		t.Fatalf("unexpected duration %v", got)
	}
	if body.CustomFields["ticket"] != "OPS-1" {
		t.Fatalf("unexpected custom fields %v", body.CustomFields)
	}
}

func TestRequests_CreateValidation(t *testing.T) {
	cli, srv := newLoggedInCLI(t)

	cli.MustFail("either --integration, --bundle or --interactive flags must be specified", "requests", "create")
	cli.MustFail("must be specified when requesting without interactive mode",
		"requests", "create", "--integration", mockapi.PostgresIntegrationID)
	cli.MustFail("resource e2e-missing not found",
		"requests", "create", "--integration", mockapi.PostgresIntegrationID, "-t", mockapi.PostgresResourceTypeID,
		"-r", "e2e-missing", "-p", mockapi.PostgresPermissionID)
	cli.MustFail("duration must be greater than 0", "requests", "create", "--bundle", mockapi.BundleID, "--duration", "0s")

	srv.Update(func(st *mockapi.State) {
		st.DryRun = clientapi.DryRunClientResponse{
			Status: "Error",
			Errors: []clientapi.DryRunAccessRequestError{{Code: "FIELD_MISSING", Field: "justification", Details: map[string]any{}}},
		}
	})
	cli.MustFail("justification is required for this request", "requests", "create", "--bundle", mockapi.BundleID)

	srv.Update(func(st *mockapi.State) {
		st.DryRun = clientapi.DryRunClientResponse{
			Status: "Error",
			Errors: []clientapi.DryRunAccessRequestError{{
				Code: "INVALID_DURATION", Field: "duration_in_sec", Details: map[string]any{"max_duration": float64(3600)},
			}},
		}
	})
	cli.MustFail("duration is required for this request", "requests", "create", "--bundle", mockapi.BundleID, "-j", "x")
	cli.MustFail("duration is too long, maximum duration is 1.00 hours",
		"requests", "create", "--bundle", mockapi.BundleID, "-j", "x", "-d", "2h")

	if n := len(srv.CallsTo(createRequestRoute)); n != 0 {
		t.Fatalf("invalid requests must not be submitted, got %d create calls", n)
	}
}

func TestRequests_CreateWaitsUntilGranted(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	srv.Update(func(st *mockapi.State) { st.NewRequestStatus = "Pending" })

	cli.MustFail("timeout waiting for request to be granted",
		"requests", "create", "--bundle", mockapi.BundleID, "-j", "x", "--timeout", "2s")

	result := cli.MustRun("requests", "create", "--bundle", mockapi.BundleID, "-j", "x", "--no-wait", "-o", "json")
	created := decodeJSON[clientapi.AccessRequestClientModel](t, result)
	if created.Status.Status != "Pending" {
		t.Fatalf("unexpected status %q", created.Status.Status)
	}
}

func TestRequests_PendingMFAPrintsLink(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	srv.Update(func(st *mockapi.State) { st.NewRequestStatus = "PendingMFA" })

	result := cli.MustRun("requests", "create", "--bundle", mockapi.BundleID, "-j", "x")
	assertContains(t, result.Stdout, srv.URL+"/account/step-up?status=pending-mfa&request=req-new-1")
}

func TestRequests_Revoke(t *testing.T) {
	cli, srv := newLoggedInCLI(t)

	result := cli.MustRun("requests", "revoke", mockapi.ActiveRequestID, "--wait")
	assertContains(t, result.Stdout, "Request "+mockapi.ActiveRequestID+" successfully revoked")

	var status string
	srv.Update(func(st *mockapi.State) { status = st.Request(mockapi.ActiveRequestID).Status.Status })
	if status != "Revoked" {
		t.Fatalf("request was not revoked on the backend, status %q", status)
	}

	cli.MustFail("api error", "requests", "revoke", "req-missing")
}
