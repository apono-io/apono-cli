//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/apono-io/apono-cli/e2e/mockapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func TestInventory_Integrations(t *testing.T) {
	cli, _ := newLoggedInCLI(t)

	integrations := decodeJSON[[]clientapi.IntegrationClientModel](t, cli.MustRun("inventory", "integrations", "-o", "json"))
	if len(integrations) != 2 {
		t.Fatalf("expected 2 integrations, got %d", len(integrations))
	}

	table := cli.MustRun("inventory", "integrations")
	assertContains(t, table.Stdout, "ID")
	assertContains(t, table.Stdout, mockapi.PostgresIntegrationName)
	assertContains(t, table.Stdout, mockapi.KubeIntegrationName)

	yaml := cli.MustRun("inventory", "integrations", "-o", "yaml")
	assertContains(t, yaml.Stdout, "name: "+mockapi.PostgresIntegrationName)
}

func TestInventory_Bundles(t *testing.T) {
	cli, _ := newLoggedInCLI(t)

	bundles := decodeJSON[[]clientapi.BundleClientModel](t, cli.MustRun("inventory", "bundles", "-o", "json"))
	if len(bundles) != 1 || bundles[0].Name != mockapi.BundleName {
		t.Fatalf("unexpected bundles: %+v", bundles)
	}
}

func TestInventory_ResourceTypesResourcesPermissions(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	integrationRef := mockapi.PostgresIntegrationType + "/" + mockapi.PostgresIntegrationName

	resourceTypes := decodeJSON[[]clientapi.ResourceTypeClientModel](t,
		cli.MustRun("inventory", "resource-types", "-i", integrationRef, "-o", "json"))
	if len(resourceTypes) != 1 || resourceTypes[0].Id != mockapi.PostgresResourceTypeID {
		t.Fatalf("unexpected resource types: %+v", resourceTypes)
	}
	calls := srv.CallsTo("GET /api/client/v1/inventory/resource-types")
	assertContains(t, calls[len(calls)-1].Query, "integration-id="+mockapi.PostgresIntegrationID)

	resources := decodeJSON[[]clientapi.ResourceClientModel](t,
		cli.MustRun("inventory", "resources", "-i", mockapi.PostgresIntegrationID, "-t", mockapi.PostgresResourceTypeID, "-o", "json"))
	if len(resources) != 1 || resources[0].SourceId != mockapi.PostgresResourceSource {
		t.Fatalf("unexpected resources: %+v", resources)
	}

	permissions := decodeJSON[[]clientapi.PermissionClientModel](t,
		cli.MustRun("inventory", "permissions", "-i", integrationRef, "-t", mockapi.PostgresResourceTypeID, "-o", "json"))
	if len(permissions) != 1 || permissions[0].Id != mockapi.PostgresPermissionID {
		t.Fatalf("unexpected permissions: %+v", permissions)
	}
}

func TestInventory_UnknownIntegration(t *testing.T) {
	cli, _ := newLoggedInCLI(t)

	cli.MustFail("integration postgresql/missing not found",
		"inventory", "resources", "-i", "postgresql/missing", "-t", mockapi.PostgresResourceTypeID)
}

func TestInventory_RequiredFlags(t *testing.T) {
	cli, _ := newLoggedInCLI(t)

	cli.MustFail(`required flag(s) "integration", "type" not set`, "inventory", "resources")
}

func TestInventory_BackendError(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	srv.FailRoute("GET /api/client/v1/inventory/integrations", http.StatusInternalServerError)

	cli.MustFail("500", "inventory", "integrations")
}
