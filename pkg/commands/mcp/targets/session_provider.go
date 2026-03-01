package targets

import (
	"context"
	"fmt"
	"regexp"
	"sort"
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

// targetEntry maps a target ID to its session and integration
type targetEntry struct {
	session     *clientapi.AccessSessionClientModel
	integration *clientapi.IntegrationClientModel
}

// sessionTargetMapping builds a consistent mapping of target IDs to sessions/integrations.
// Each session becomes its own target. Integrations without sessions get a "needs_access" entry.
func (p *SessionTargetProvider) sessionTargetMapping(ctx context.Context) (map[string]*targetEntry, error) {
	integrations, err := services.ListIntegrations(ctx, p.client)
	if err != nil {
		utils.McpLogf("[SessionProvider] Failed to list integrations: %v", err)
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}

	utils.McpLogf("[SessionProvider] Found %d total integrations", len(integrations))

	sessions, err := services.ListAccessSessions(ctx, p.client, []string{}, []string{}, []string{})
	if err != nil {
		utils.McpLogf("[SessionProvider] Failed to list sessions: %v", err)
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	utils.McpLogf("[SessionProvider] Found %d active sessions", len(sessions))

	// Group sessions by integration ID, sorted by session ID for stable ordering
	integrationSessions := make(map[string][]int) // integration ID -> indices into sessions slice
	for i := range sessions {
		intID := sessions[i].Integration.Id
		integrationSessions[intID] = append(integrationSessions[intID], i)
	}
	for _, indices := range integrationSessions {
		sort.Slice(indices, func(a, b int) bool {
			return sessions[indices[a]].Id < sessions[indices[b]].Id
		})
	}

	result := make(map[string]*targetEntry)
	usedIDs := make(map[string]bool)

	for i := range integrations {
		integration := &integrations[i]

		if !p.allIntegrations && !isDatabaseIntegrationType(integration.Type) {
			utils.McpLogf("[SessionProvider]   Skipping integration %q (type=%q) - not a database type", integration.Name, integration.Type)
			continue
		}

		sessionIndices, hasSessions := integrationSessions[integration.Id]
		if hasSessions {
			// One target per session
			for _, idx := range sessionIndices {
				session := &sessions[idx]
				targetID := uniqueTargetID(sanitizeName(session.Name), usedIDs)
				usedIDs[targetID] = true
				result[targetID] = &targetEntry{
					session:     session,
					integration: integration,
				}
				utils.McpLogf("[SessionProvider]   Target %q -> session %s (%s)", targetID, session.Id, session.Name)
			}
		} else {
			// No session — needs_access entry for the integration
			targetID := uniqueTargetID(sanitizeName(integration.Name), usedIDs)
			usedIDs[targetID] = true
			result[targetID] = &targetEntry{
				session:     nil,
				integration: integration,
			}
			utils.McpLogf("[SessionProvider]   Target %q -> no session (needs access)", targetID)
		}
	}

	return result, nil
}

// uniqueTargetID returns a unique target ID by appending a numeric suffix on collision
func uniqueTargetID(base string, used map[string]bool) string {
	if !used[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !used[candidate] {
			return candidate
		}
	}
}

// ListTargets returns all database-type integrations with their access status,
// creating one target per active session
func (p *SessionTargetProvider) ListTargets(ctx context.Context) ([]TargetInfo, error) {
	mapping, err := p.sessionTargetMapping(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]TargetInfo, 0, len(mapping))
	for targetID, entry := range mapping {
		info := TargetInfo{
			ID:   targetID,
			Type: mapIntegrationTypeToBackendType(entry.integration.Type),
		}

		if entry.session != nil {
			info.Name = fmt.Sprintf("Apono: %s", entry.session.Name)
			info.Status = TargetStatusReady
		} else {
			info.Name = fmt.Sprintf("Apono: %s", entry.integration.Name)
			info.Status = TargetStatusNeedsAccess
		}

		result = append(result, info)
	}

	utils.McpLogf("[SessionProvider] Discovered %d database targets", len(result))
	return result, nil
}

// GetTarget returns a target definition with credentials from the specific session
func (p *SessionTargetProvider) GetTarget(ctx context.Context, targetID string) (*TargetDefinition, error) {
	mapping, err := p.sessionTargetMapping(ctx)
	if err != nil {
		return nil, err
	}

	entry, ok := mapping[targetID]
	if !ok {
		return nil, fmt.Errorf("no target found for %q", targetID)
	}

	if entry.session == nil {
		return nil, fmt.Errorf("no active session for target %q - call EnsureAccess first", targetID)
	}

	session := entry.session

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

	// Get full access details to extract credentials for this specific session
	fullDetails, _, err := p.client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, session.Id).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get access details: %w", err)
	}

	// Try JSON format first (structured credentials)
	creds := fullDetails.Json
	utils.McpLogf("[SessionProvider] JSON credentials for %s: %v", targetID, maskedCreds(creds))

	var credentials map[string]string
	if len(creds) > 0 && !hasOnlyMaskedPassword(creds) {
		credentials = make(map[string]string)
		for k, v := range creds {
			credentials[k] = fmt.Sprintf("%v", v)
		}
	} else {
		// JSON not available or password is masked — extract from instructions text
		if hasOnlyMaskedPassword(creds) {
			utils.McpLogf("[SessionProvider] JSON password is masked, falling back to instructions text")
		} else {
			utils.McpLogf("[SessionProvider] No JSON credentials, trying CLI/instructions format")
		}

		credentials, err = extractCredentialsFromText(entry.integration.Type, fullDetails)
		if err != nil {
			return nil, fmt.Errorf("no credentials available for target %q: %w", targetID, err)
		}
	}

	return &TargetDefinition{
		ID:            targetID,
		Name:          fmt.Sprintf("Apono: %s", session.Name),
		Type:          mapIntegrationTypeToBackendType(entry.integration.Type),
		Credentials:   credentials,
		IntegrationID: entry.integration.Id,
		SessionID:     session.Id,
	}, nil
}

// EnsureAccess ensures the target has an active session, requesting access if needed
func (p *SessionTargetProvider) EnsureAccess(ctx context.Context, targetID string) error {
	mapping, err := p.sessionTargetMapping(ctx)
	if err != nil {
		return err
	}

	entry, ok := mapping[targetID]
	if !ok {
		return fmt.Errorf("no target found for %q", targetID)
	}

	// Already has a session — nothing to do
	if entry.session != nil {
		return nil
	}

	utils.McpLogf("[SessionProvider] No active session for %s, requesting access...", targetID)

	// Create access request for the integration
	request := clientapi.NewCreateAccessRequestClientModel(
		[]string{entry.integration.Id}, // integration IDs
		[]string{},                      // bundle IDs
		[]string{},                      // resource type IDs
		[]string{},                      // resource IDs
		[]clientapi.ResourceFilter{},    // resource filters
		[]string{},                      // permission IDs
		[]string{},                      // access unit IDs
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
	return p.waitForAccess(ctx, entry.integration.Id)
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
		utils.McpLogf("[SessionProvider] Extracted credential fields from text (host=%s, port=%s, user=%s, db=%s)",
			host, port, user, dbname)
		result := map[string]string{
			"host":     host,
			"username": user,
			"password": strings.TrimSpace(pass),
		}
		if port != "" {
			result["port"] = port
		}
		if dbname != "" {
			result["db_name"] = dbname
		}
		return result, nil
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
