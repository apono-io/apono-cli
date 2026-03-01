package targets

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
)

// DatabaseIntegrationKeywords lists keywords that identify database-type integrations.
// The API returns cloud-prefixed types like "azure-postgresql", "aws-rds-mysql", etc.
// We use substring matching to handle all variants.
var DatabaseIntegrationKeywords = []string{
	"postgresql",
	"mysql",
	"mariadb",
	"mssql",
	"mongodb",
}

// isDatabaseIntegrationType checks if the integration type contains a known database keyword
func isDatabaseIntegrationType(integrationType string) bool {
	lower := strings.ToLower(integrationType)
	for _, keyword := range DatabaseIntegrationKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// SessionTargetProvider discovers targets from Apono sessions and integrations
type SessionTargetProvider struct {
	client            *aponoapi.AponoClient
	allIntegrations   bool
}

// NewSessionTargetProvider creates a new session-based target provider.
// If allIntegrations is true, all integrations are returned without database-type filtering.
func NewSessionTargetProvider(client *aponoapi.AponoClient, allIntegrations bool) *SessionTargetProvider {
	return &SessionTargetProvider{
		client:          client,
		allIntegrations: allIntegrations,
	}
}

// ListTargets returns all database-type integrations with their access status
func (p *SessionTargetProvider) ListTargets(ctx context.Context) ([]TargetInfo, error) {
	// Get all integrations
	integrations, err := services.ListIntegrations(ctx, p.client)
	if err != nil {
		utils.McpLogf("[SessionProvider] Failed to list integrations: %v", err)
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}

	utils.McpLogf("[SessionProvider] Found %d total integrations", len(integrations))
	for _, integration := range integrations {
		utils.McpLogf("[SessionProvider]   Integration: id=%s name=%q type=%q", integration.Id, integration.Name, integration.Type)
	}

	// Get all active sessions
	sessions, err := services.ListAccessSessions(ctx, p.client, []string{}, []string{}, []string{})
	if err != nil {
		utils.McpLogf("[SessionProvider] Failed to list sessions: %v", err)
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	utils.McpLogf("[SessionProvider] Found %d active sessions", len(sessions))

	// Build session map: integration ID -> session
	sessionMap := make(map[string]*clientapi.AccessSessionClientModel)
	for i := range sessions {
		sessionMap[sessions[i].Integration.Id] = &sessions[i]
	}

	result := make([]TargetInfo, 0)
	for _, integration := range integrations {
		// Only include database-type integrations (unless --all-integrations is set)
		if !p.allIntegrations && !isDatabaseIntegrationType(integration.Type) {
			utils.McpLogf("[SessionProvider]   Skipping integration %q (type=%q) - not a database type", integration.Name, integration.Type)
			continue
		}

		targetID := sanitizeName(integration.Name)
		info := TargetInfo{
			ID:   targetID,
			Name: fmt.Sprintf("Apono: %s", integration.Name),
			Type: mapIntegrationTypeToBackendType(integration.Type),
		}

		if _, hasSession := sessionMap[integration.Id]; hasSession {
			info.Status = TargetStatusReady
		} else {
			info.Status = TargetStatusNeedsAccess
		}

		result = append(result, info)
	}

	utils.McpLogf("[SessionProvider] Discovered %d database targets", len(result))
	return result, nil
}

// GetTarget returns a target definition with credentials from an active session
func (p *SessionTargetProvider) GetTarget(ctx context.Context, targetID string) (*TargetDefinition, error) {
	// Find the integration matching this target ID
	integrations, err := services.ListIntegrations(ctx, p.client)
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}

	var matchedIntegration *clientapi.IntegrationClientModel
	for i, integration := range integrations {
		if sanitizeName(integration.Name) == targetID {
			matchedIntegration = &integrations[i]
			break
		}
	}

	if matchedIntegration == nil {
		return nil, fmt.Errorf("no integration found for target %q", targetID)
	}

	// Find active session for this integration
	sessions, err := services.ListAccessSessions(ctx, p.client, []string{matchedIntegration.Id}, []string{}, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		return nil, fmt.Errorf("no active session for target %q - call EnsureAccess first", targetID)
	}

	session := sessions[0]

	// Check if credentials need resetting
	if session.Credentials.IsSet() {
		creds := session.Credentials.Get()
		if creds.Status != "new" && creds.CanReset {
			utils.McpLogf("[SessionProvider] Resetting stale credentials for %s", targetID)
			if err := p.resetCredentials(ctx, session.Id); err != nil {
				utils.McpLogf("[SessionProvider] Failed to reset credentials: %v", err)
			}
		}
	}

	// Get full access details to extract credentials
	fullDetails, _, err := p.client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, session.Id).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get access details: %w", err)
	}

	// Try JSON format first (structured credentials)
	creds := fullDetails.Json
	utils.McpLogf("[SessionProvider] JSON credentials for %s: %v", targetID, maskedCreds(creds))

	var credentials map[string]string
	if len(creds) > 0 && !hasOnlyMaskedPassword(creds) {
		credentials, err = buildCredentials(matchedIntegration.Type, creds)
		if err != nil {
			return nil, fmt.Errorf("failed to build credentials: %w", err)
		}
	} else {
		// JSON not available or password is masked — extract from instructions text
		if hasOnlyMaskedPassword(creds) {
			utils.McpLogf("[SessionProvider] JSON password is masked, falling back to instructions text")
		} else {
			utils.McpLogf("[SessionProvider] No JSON credentials, trying CLI/instructions format")
		}

		credentials, err = extractCredentialsFromText(matchedIntegration.Type, fullDetails)
		if err != nil {
			return nil, fmt.Errorf("no credentials available for target %q: %w", targetID, err)
		}
	}

	return &TargetDefinition{
		ID:            targetID,
		Name:          fmt.Sprintf("Apono: %s", matchedIntegration.Name),
		Type:          mapIntegrationTypeToBackendType(matchedIntegration.Type),
		Credentials:   credentials,
		IntegrationID: matchedIntegration.Id,
	}, nil
}

// EnsureAccess ensures the target has an active session, requesting access if needed
func (p *SessionTargetProvider) EnsureAccess(ctx context.Context, targetID string) error {
	// Find integration
	integrations, err := services.ListIntegrations(ctx, p.client)
	if err != nil {
		return fmt.Errorf("failed to list integrations: %w", err)
	}

	var matchedIntegration *clientapi.IntegrationClientModel
	for i, integration := range integrations {
		if sanitizeName(integration.Name) == targetID {
			matchedIntegration = &integrations[i]
			break
		}
	}

	if matchedIntegration == nil {
		return fmt.Errorf("no integration found for target %q", targetID)
	}

	// Check if we already have an active session
	sessions, err := services.ListAccessSessions(ctx, p.client, []string{matchedIntegration.Id}, []string{}, []string{})
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) > 0 {
		return nil // Already have access
	}

	utils.McpLogf("[SessionProvider] No active session for %s, requesting access...", targetID)

	// Create access request
	request := clientapi.NewCreateAccessRequestClientModel(
		[]string{matchedIntegration.Id}, // integration IDs
		[]string{},                       // bundle IDs
		[]string{},                       // resource type IDs
		[]string{},                       // resource IDs
		[]clientapi.ResourceFilter{},     // resource filters
		[]string{},                       // permission IDs
		[]string{},                       // access unit IDs
	)

	justification := fmt.Sprintf("Auto-requested by MCP proxy for target %s", targetID)
	request.Justification.Set(&justification)

	createdRequest, _, err := p.client.ClientAPI.AccessRequestsAPI.CreateUserAccessRequest(ctx).
		CreateAccessRequestClientModel(*request).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to create access request: %w", err)
	}

	if len(createdRequest.RequestIds) == 0 {
		return fmt.Errorf("no request IDs returned")
	}

	utils.McpLogf("[SessionProvider] Access request created: %v, waiting for approval...", createdRequest.RequestIds)

	// Poll for the request to be approved and session to become active
	return p.waitForAccess(ctx, matchedIntegration.Id)
}

// waitForAccess polls until an active session exists for the integration
func (p *SessionTargetProvider) waitForAccess(ctx context.Context, integrationID string) error {
	const (
		pollInterval = 3 * time.Second
		maxWait      = 5 * time.Minute
	)

	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		sessions, err := services.ListAccessSessions(ctx, p.client, []string{integrationID}, []string{}, []string{})
		if err != nil {
			utils.McpLogf("[SessionProvider] Error checking sessions: %v", err)
		} else if len(sessions) > 0 {
			utils.McpLogf("[SessionProvider] Access granted!")
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timeout waiting for access approval (waited %v)", maxWait)
}

// resetCredentials resets credentials for a session and waits for fresh ones
func (p *SessionTargetProvider) resetCredentials(ctx context.Context, sessionID string) error {
	_, _, err := p.client.ClientAPI.AccessSessionsAPI.ResetAccessSessionCredentials(ctx, sessionID).Execute()
	if err != nil {
		return fmt.Errorf("failed to reset credentials: %w", err)
	}

	// Wait for fresh credentials
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		session, _, err := p.client.ClientAPI.AccessSessionsAPI.GetAccessSession(ctx, sessionID).Execute()
		if err != nil {
			return fmt.Errorf("failed to get session status: %w", err)
		}

		if session.Credentials.IsSet() && session.Credentials.Get().Status == "new" {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}

	return fmt.Errorf("timeout waiting for credentials reset")
}

// buildCredentials builds the credential map for a target based on integration type.
// Uses substring matching to handle cloud-prefixed types like "azure-postgresql".
func buildCredentials(integrationType string, creds map[string]interface{}) (map[string]string, error) {
	lower := strings.ToLower(integrationType)
	if strings.Contains(lower, "postgresql") {
		return buildPostgresCredentials(creds)
	}
	if strings.Contains(lower, "mysql") || strings.Contains(lower, "mariadb") {
		return buildMySQLCredentials(creds)
	}
	return buildGenericCredentials(creds)
}

func buildPostgresCredentials(creds map[string]interface{}) (map[string]string, error) {
	host := fmt.Sprintf("%v", creds["host"])
	port := fmt.Sprintf("%v", creds["port"])
	dbName := fmt.Sprintf("%v", creds["db_name"])
	username := fmt.Sprintf("%v", creds["username"])
	password := fmt.Sprintf("%v", creds["password"])

	encodedUsername := url.QueryEscape(username)
	encodedPassword := url.QueryEscape(password)

	connectionString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=require", encodedUsername, encodedPassword, host, port, dbName)

	return map[string]string{
		"database_url": connectionString,
	}, nil
}

func buildMySQLCredentials(creds map[string]interface{}) (map[string]string, error) {
	host := fmt.Sprintf("%v", creds["host"])
	port := fmt.Sprintf("%v", creds["port"])
	dbName := fmt.Sprintf("%v", creds["db_name"])
	username := fmt.Sprintf("%v", creds["username"])
	password := fmt.Sprintf("%v", creds["password"])

	connectionString := fmt.Sprintf("mysql://%s:%s@%s:%s/%s",
		url.QueryEscape(username), url.QueryEscape(password), host, port, dbName)

	return map[string]string{
		"database_url": connectionString,
	}, nil
}

func buildGenericCredentials(creds map[string]interface{}) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range creds {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result, nil
}

// mapIntegrationTypeToBackendType maps Apono integration types to backend type IDs.
// Handles cloud-prefixed types like "azure-postgresql", "aws-rds-mysql", etc.
func mapIntegrationTypeToBackendType(integrationType string) string {
	lower := strings.ToLower(integrationType)
	if strings.Contains(lower, "postgresql") {
		return "postgres"
	}
	if strings.Contains(lower, "mysql") || strings.Contains(lower, "mariadb") {
		return "mysql"
	}
	if strings.Contains(lower, "mssql") {
		return "mssql"
	}
	if strings.Contains(lower, "mongodb") {
		return "mongodb"
	}
	return lower
}

// extractCredentialsFromText tries to extract database credentials from CLI or instructions text
// when structured JSON credentials are not available
func extractCredentialsFromText(integrationType string, details *clientapi.AccessSessionDetailsClientModel) (map[string]string, error) {
	// Collect all text sources to search
	var texts []string
	if details.Cli.IsSet() && details.Cli.Get() != nil {
		texts = append(texts, *details.Cli.Get())
	}
	texts = append(texts, details.Instructions.Plain)
	texts = append(texts, details.Instructions.Markdown)

	combined := strings.Join(texts, "\n")

	// Parse individual fields first (most reliable — passwords with special chars get properly encoded)
	// Instructions format: "url: host\nport: 5432\nusername: user\npassword: pass"
	host := extractField(combined, `(?:url|host|hostname|server):\s*([^\s,;]+)`)
	port := extractField(combined, `(?:port):\s*(\d+)`)
	user := extractField(combined, `(?:username|user):\s*([^\s,;]+)`)
	pass := extractFieldToEOL(combined, `(?:password|passwd):\s*`)
	dbname := extractField(combined, `(?:dbname|database|db_name):\s*([^\s,;]+)`)

	// If no dbname from fields, try to extract from the oneliner URL path
	if dbname == "" {
		dbname = extractField(combined, `(?:postgresql|postgres)://[^/]+/([^\s?]+)`)
	}

	if host != "" && user != "" {
		if port == "" {
			port = "5432"
		}
		connStr := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=require",
			url.QueryEscape(user), url.QueryEscape(strings.TrimSpace(pass)), host, port, dbname)
		utils.McpLogf("[SessionProvider] Built connection URL from parsed fields (host=%s, port=%s, user=%s, db=%s)",
			host, port, user, dbname)
		return map[string]string{
			"database_url": connStr,
		}, nil
	}

	return nil, fmt.Errorf("could not extract credentials from access details (no JSON, no parseable fields found in CLI/instructions output)")
}

func extractField(text, pattern string) string {
	re := regexp.MustCompile(`(?i)` + pattern)
	if matches := re.FindStringSubmatch(text); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractFieldToEOL extracts the value after a key prefix up to end of line.
// Handles passwords with special characters like @, ^, (, etc.
func extractFieldToEOL(text, prefixPattern string) string {
	re := regexp.MustCompile(`(?im)` + prefixPattern + `(.+)$`)
	if matches := re.FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// hasOnlyMaskedPassword checks if the JSON credentials have a masked password (e.g., "*****")
func hasOnlyMaskedPassword(creds map[string]interface{}) bool {
	if pw, ok := creds["password"]; ok {
		if pwStr, ok := pw.(string); ok {
			return strings.TrimSpace(pwStr) != "" && strings.Trim(pwStr, "*") == ""
		}
	}
	return false
}

// maskedCreds returns a copy of creds with sensitive values masked for logging
func maskedCreds(creds map[string]interface{}) map[string]interface{} {
	masked := make(map[string]interface{}, len(creds))
	for k, v := range creds {
		if k == "password" {
			masked[k] = "***"
		} else {
			masked[k] = v
		}
	}
	return masked
}

func credKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// sanitizeName converts a name to a safe format for use in target IDs
func sanitizeName(name string) string {
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result += string(r)
		} else if r >= 'A' && r <= 'Z' {
			result += string(r + 32) // lowercase
		} else if r == ' ' || r == '_' {
			result += "-"
		}
	}
	return result
}
