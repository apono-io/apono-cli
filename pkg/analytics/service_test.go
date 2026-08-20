package analytics

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestIsInteractiveMode(t *testing.T) {
	if IsInteractiveMode(context.Background()) {
		t.Error("expected a plain context not to be interactive")
	}

	if !IsInteractiveMode(CreateInteractiveModeContext(context.Background())) {
		t.Error("expected a marked context to be interactive")
	}
}

func TestTruncatePropertyValue(t *testing.T) {
	short := strings.Repeat("a", maxPropertyValueLength)
	if got := truncatePropertyValue(short); got != short {
		t.Errorf("expected a value at the limit to pass through, got %d chars", len(got))
	}

	long := strings.Repeat("a", maxPropertyValueLength+50)
	if got := truncatePropertyValue(long); len([]rune(got)) != maxPropertyValueLength {
		t.Errorf("expected truncation to %d chars, got %d", maxPropertyValueLength, len([]rune(got)))
	}

	multibyte := strings.Repeat("é", maxPropertyValueLength+10)
	truncated := truncatePropertyValue(multibyte)
	if len([]rune(truncated)) != maxPropertyValueLength {
		t.Errorf("expected %d runes, got %d", maxPropertyValueLength, len([]rune(truncated)))
	}
	for _, r := range truncated {
		if r == '�' {
			t.Error("truncation split a multibyte character")
			break
		}
	}
}

func TestAccessRequestSubmittedProperties_bundleSendsEmptyIntegrationFields(t *testing.T) {
	duration := 4 * time.Hour
	properties := accessRequestSubmittedProperties(AccessRequestSubmittedProperties{
		RequestType:   RequestTypeBundle,
		BundleName:    "prod-oncall",
		Duration:      &duration,
		Justification: "on call",
	})

	assertProperty(t, properties, surfaceField, RequestNewAccessFlow)
	assertProperty(t, properties, requestTypeField, RequestTypeBundle)
	assertProperty(t, properties, bundleNameField, "prod-oncall")
	assertProperty(t, properties, durationField, 4.0)
	assertProperty(t, properties, justificationField, "on call")

	assertProperty(t, properties, integrationNameField, "")
	assertProperty(t, properties, resourceTypeField, "")
	assertProperty(t, properties, resourcesCountField, 0)

	permissions, ok := properties[permissionsField].([]string)
	if !ok || len(permissions) != 0 {
		t.Errorf("permissions = %v, want an empty array rather than a missing property", properties[permissionsField])
	}
}

func TestAccessRequestSubmittedProperties_integrationSendsEmptyBundleFields(t *testing.T) {
	properties := accessRequestSubmittedProperties(AccessRequestSubmittedProperties{
		RequestType:     RequestTypeIntegration,
		IntegrationName: "prod-postgres",
		ResourceType:    "Database",
		ResourcesCount:  3,
		Permissions:     []string{"SELECT", "INSERT"},
	})

	assertProperty(t, properties, requestTypeField, RequestTypeIntegration)
	assertProperty(t, properties, integrationNameField, "prod-postgres")
	assertProperty(t, properties, resourceTypeField, "Database")
	assertProperty(t, properties, resourcesCountField, 3)

	permissions, ok := properties[permissionsField].([]string)
	if !ok || len(permissions) != 2 {
		t.Fatalf("expected two permissions, got %v", properties[permissionsField])
	}

	assertProperty(t, properties, bundleNameField, "")
	assertProperty(t, properties, durationField, 0.0)
	assertProperty(t, properties, justificationField, "")
}

func TestAccessRequestSubmittedProperties_staysUnderServerPropertyLimit(t *testing.T) {
	duration := time.Hour
	properties := accessRequestSubmittedProperties(AccessRequestSubmittedProperties{
		RequestType:     RequestTypeIntegration,
		BundleName:      "bundle",
		IntegrationName: "integration",
		ResourceType:    "type",
		ResourcesCount:  1,
		Permissions:     []string{"SELECT"},
		Duration:        &duration,
		Justification:   "why",
	})

	propertiesWithCLIVersion := len(properties) + 1
	if propertiesWithCLIVersion > maxEventProperties {
		t.Errorf("event carries %d properties including cli_version, server rejects more than %d", propertiesWithCLIVersion, maxEventProperties)
	}
}

func assertProperty(t *testing.T, properties map[string]interface{}, key string, want interface{}) {
	t.Helper()

	got, ok := properties[key]
	if !ok {
		t.Errorf("missing property %q", key)
		return
	}
	if got != want {
		t.Errorf("property %q = %v, want %v", key, got, want)
	}
}
