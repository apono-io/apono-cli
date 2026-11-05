//go:build e2e

package e2e

import (
	"testing"

	"github.com/apono-io/apono-cli/e2e/mockapi"
)

func TestLogin_PersonalToken(t *testing.T) {
	srv := mockapi.New(t)
	cli := newCLI(t)

	result := cli.Login(srv, "default")

	assertContains(t, result.Stdout, "You successfully logged in to "+mockapi.AccountName+" as "+mockapi.UserName+" ("+mockapi.UserEmail+")")

	sessionCalls := srv.CallsTo("GET /api/client/v1/session")
	if len(sessionCalls) == 0 {
		t.Fatal("login did not validate the token with GET /session")
	}
	if got := sessionCalls[0].Authorization; got != "Bearer "+srv.Token() {
		t.Fatalf("unexpected Authorization header %q", got)
	}

	profiles := cli.MustRun("profiles", "list")
	assertContains(t, profiles.Stdout, "default")
	assertContains(t, profiles.Stdout, mockapi.AccountName)
	assertContains(t, profiles.Stdout, mockapi.UserEmail)
	assertNoUnexpectedCalls(t, srv)
}

func TestLogin_InvalidToken(t *testing.T) {
	srv := mockapi.New(t)
	cli := newCLI(t)

	cli.MustFail("failed fetching user session",
		"login", "--personal-token", "wrong-token", "--api-url", srv.URL, "--app-url", srv.URL)

	// A failed login must not leave a usable profile behind.
	cli.MustFail("run `apono login`", "inventory", "integrations")
}

func TestCommand_WithoutLogin(t *testing.T) {
	cli := newCLI(t)

	cli.MustFail("no profiles configured", "access", "list")
}

func TestProfiles_SetAndLogout(t *testing.T) {
	srv := mockapi.New(t)
	cli := newCLI(t)
	cli.Login(srv, "default")
	cli.Login(srv, "second")

	result := cli.MustRun("profiles", "set", "second")
	assertContains(t, result.Stdout, "Setting active profile to: second")

	// Commands run against the active profile.
	cli.MustRun("inventory", "integrations")

	result = cli.MustRun("logout", "second")
	assertContains(t, result.Stdout, "Logging out profile: second")

	profiles := cli.MustRun("profiles", "list")
	assertContains(t, profiles.Stdout, "default")
	assertNotContains(t, profiles.Stdout, "second")

	cli.MustFail("profile not exists", "logout", "second")
	cli.MustFail("profile not exists", "profiles", "set", "missing")
}

func TestProfileFlag_SelectsProfile(t *testing.T) {
	first := mockapi.New(t)
	second := mockapi.New(t, mockapi.WithToken("second-token"))
	cli := newCLI(t)
	cli.Login(first, "default")
	cli.Login(second, "other")

	cli.MustRun("inventory", "integrations", "--profile", "other")

	if len(second.CallsTo("GET /api/client/v1/inventory/integrations")) != 1 {
		t.Fatal("--profile other did not send the request to the second backend")
	}
	if len(first.CallsTo("GET /api/client/v1/inventory/integrations")) != 0 {
		t.Fatal("--profile other sent the request to the default profile's backend")
	}
}
