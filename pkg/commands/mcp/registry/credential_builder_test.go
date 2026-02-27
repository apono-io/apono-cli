package registry

import (
	"testing"
)

func TestBuildCredentials_WithTemplates(t *testing.T) {
	def := MCPServerDefinition{
		CredentialBuilder: map[string]string{
			"database_url": "postgresql://{{.username}}:{{.password}}@{{.host}}:{{.port}}/{{.db_name}}?sslmode=require",
		},
	}

	rawFields := map[string]string{
		"host":     "db.prod.com",
		"port":     "5432",
		"username": "app_user",
		"password": "secret123",
		"db_name":  "mydb",
	}

	result, err := BuildCredentials(def, rawFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "postgresql://app_user:secret123@db.prod.com:5432/mydb?sslmode=require"
	if result["database_url"] != expected {
		t.Errorf("expected %q, got %q", expected, result["database_url"])
	}
}

func TestBuildCredentials_NoTemplates_Passthrough(t *testing.T) {
	def := MCPServerDefinition{} // no credential_builder

	rawFields := map[string]string{
		"host":     "db.prod.com",
		"password": "secret",
	}

	result, err := BuildCredentials(def, rawFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["host"] != "db.prod.com" {
		t.Errorf("expected passthrough of host")
	}
	if result["password"] != "secret" {
		t.Errorf("expected passthrough of password")
	}
}

func TestBuildCredentials_MissingField(t *testing.T) {
	def := MCPServerDefinition{
		CredentialBuilder: map[string]string{
			"database_url": "postgresql://{{.username}}:{{.password}}@{{.host}}:{{.port}}/{{.db_name}}",
		},
	}

	rawFields := map[string]string{
		"host": "db.prod.com",
		// missing username, password, port, db_name
	}

	_, err := BuildCredentials(def, rawFields)
	if err == nil {
		t.Fatal("expected error for missing template fields")
	}
}

func TestBuildCredentials_URLEncode(t *testing.T) {
	def := MCPServerDefinition{
		CredentialBuilder: map[string]string{
			"database_url": "postgresql://{{.username}}:{{urlEncode .password}}@{{.host}}:{{.port}}/{{.db_name}}",
		},
	}

	rawFields := map[string]string{
		"host":     "db.prod.com",
		"port":     "5432",
		"username": "app_user",
		"password": "p@ss/w0rd#123",
		"db_name":  "mydb",
	}

	result, err := BuildCredentials(def, rawFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "postgresql://app_user:p%40ss%2Fw0rd%23123@db.prod.com:5432/mydb"
	if result["database_url"] != expected {
		t.Errorf("expected %q, got %q", expected, result["database_url"])
	}
}
