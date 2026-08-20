package apono

import (
	"testing"
	"time"

	"github.com/apono-io/apono-cli/pkg/analytics"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/interactive/flows"
)

func TestBuildAccessRequestSubmittedProperties_bundle(t *testing.T) {
	justification := "quarterly audit"
	req := &clientapi.CreateAccessRequestClientModel{
		FilterBundleIds: []string{"bundle-id"},
		Justification:   *clientapi.NewNullableString(&justification),
	}
	duration := 2 * time.Hour
	models := &flows.CreateAccessRequestWithFullModels{
		Bundles:  []clientapi.BundleClientModel{{Id: "bundle-id", Name: "prod-oncall"}},
		Duration: &duration,
	}

	submitted := buildAccessRequestSubmittedProperties(req, models)

	if submitted.RequestType != analytics.RequestTypeBundle {
		t.Errorf("request type = %q, want %q", submitted.RequestType, analytics.RequestTypeBundle)
	}
	if submitted.BundleName != "prod-oncall" {
		t.Errorf("bundle name = %q, want the resolved name not the id", submitted.BundleName)
	}
	if submitted.Justification != justification {
		t.Errorf("justification = %q, want %q", submitted.Justification, justification)
	}
	if submitted.Duration == nil || *submitted.Duration != duration {
		t.Errorf("duration = %v, want %v", submitted.Duration, duration)
	}
	if submitted.IntegrationName != "" || submitted.ResourceType != "" || submitted.Permissions != nil {
		t.Error("bundle request should not carry integration properties")
	}
}

func TestBuildAccessRequestSubmittedProperties_integration(t *testing.T) {
	req := &clientapi.CreateAccessRequestClientModel{
		FilterIntegrationIds: []string{"integration-id"},
		FilterResources: []clientapi.ResourceFilter{
			{Value: "db-1"},
			{Value: "db-2"},
		},
	}
	models := &flows.CreateAccessRequestWithFullModels{
		Integrations: []clientapi.IntegrationClientModel{{Id: "integration-id", Name: "prod-postgres"}},
		ResourceType: &clientapi.ResourceTypeClientModel{Id: "database", Name: "Database"},
		Permissions: []clientapi.PermissionClientModel{
			{Id: "select", Name: "SELECT"},
			{Id: "insert", Name: "INSERT"},
		},
	}

	submitted := buildAccessRequestSubmittedProperties(req, models)

	if submitted.RequestType != analytics.RequestTypeIntegration {
		t.Errorf("request type = %q, want %q", submitted.RequestType, analytics.RequestTypeIntegration)
	}
	if submitted.IntegrationName != "prod-postgres" {
		t.Errorf("integration name = %q, want the resolved name", submitted.IntegrationName)
	}
	if submitted.ResourceType != "Database" {
		t.Errorf("resource type = %q, want the resolved name not the id", submitted.ResourceType)
	}
	if submitted.ResourcesCount != 2 {
		t.Errorf("resources count = %d, want 2", submitted.ResourcesCount)
	}
	if len(submitted.Permissions) != 2 || submitted.Permissions[0] != "SELECT" {
		t.Errorf("permissions = %v, want resolved names", submitted.Permissions)
	}
	if submitted.BundleName != "" {
		t.Error("integration request should not carry a bundle name")
	}
}
