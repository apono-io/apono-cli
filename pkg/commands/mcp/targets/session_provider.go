package targets

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
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
	client          *aponoapi.AponoClient
	allIntegrations bool
	resetMu         sync.Mutex
	resetSessions   map[string]bool              // tracks session IDs already reset
	cachedCreds     map[string]map[string]string  // session ID -> cached unmasked credentials
}

// NewSessionTargetProvider creates a new session-based target provider.
// If allIntegrations is true, all integrations are returned without database-type filtering.
func NewSessionTargetProvider(client *aponoapi.AponoClient, allIntegrations bool) *SessionTargetProvider {
	return &SessionTargetProvider{
		client:          client,
		allIntegrations: allIntegrations,
		resetSessions:   make(map[string]bool),
		cachedCreds:     make(map[string]map[string]string),
	}
}

// targetEntry maps a target ID to its session and integration
type targetEntry struct {
	session     *clientapi.AccessSessionClientModel
	integration *clientapi.IntegrationClientModel
	dbName      string // database name override (set when expanding single session to multi-db targets)
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
			// For database integrations, try to expand a single session into per-database targets
			if isDatabaseIntegrationType(integration.Type) && len(sessionIndices) == 1 {
				databases := p.discoverDatabases(ctx, integration.Id)
				if len(databases) >= 2 {
					session := &sessions[sessionIndices[0]]
					for _, dbName := range databases {
						targetID := uniqueTargetID(sanitizeName(dbName), usedIDs)
						usedIDs[targetID] = true
						result[targetID] = &targetEntry{
							session:     session,
							integration: integration,
							dbName:      dbName,
						}
						utils.McpLogf("[SessionProvider]   Target %q -> session %s, db=%s", targetID, session.Id, dbName)
					}
					continue
				}
			}

			// Default: one target per session (no database expansion)
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

// discoverDatabases returns the names of databases accessible within an integration.
// Returns nil if resource listing fails or no databases found (caller should fall back to single target).
func (p *SessionTargetProvider) discoverDatabases(ctx context.Context, integrationID string) []string {
	resourceTypes, err := services.ListResourceTypes(ctx, p.client, integrationID)
	if err != nil {
		utils.McpLogf("[SessionProvider] Failed to list resource types for %s: %v", integrationID, err)
		return nil
	}

	if len(resourceTypes) == 0 {
		utils.McpLogf("[SessionProvider] No resource types for integration %s", integrationID)
		return nil
	}

	resourceTypeID := resourceTypes[0].Id
	utils.McpLogf("[SessionProvider] Using resource type %q (%s) for integration %s", resourceTypes[0].Name, resourceTypeID, integrationID)

	resources, err := services.ListResources(ctx, p.client, integrationID, resourceTypeID, nil)
	if err != nil {
		utils.McpLogf("[SessionProvider] Failed to list resources for %s: %v", integrationID, err)
		return nil
	}

	names := make([]string, 0, len(resources))
	for _, r := range resources {
		// Use SourceId (actual database name in source system) not Name (Apono display name)
		names = append(names, r.SourceId)
		utils.McpLogf("[SessionProvider]   Resource: sourceId=%q name=%q path=%q", r.SourceId, r.Name, r.Path)
	}

	utils.McpLogf("[SessionProvider] Discovered %d databases in integration %s: %v", len(names), integrationID, names)
	return names
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
			if entry.dbName != "" {
				info.Name = fmt.Sprintf("Apono: %s / %s", entry.session.Name, entry.dbName)
			} else {
				info.Name = fmt.Sprintf("Apono: %s", entry.session.Name)
			}
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

	// Get credentials: try cache file → reset + poll for unmasked password
	credentials, err := p.getCredentials(ctx, session, targetID)
	if err != nil {
		return nil, err
	}

	// Override db_name if this target was expanded from a multi-database session
	if entry.dbName != "" {
		utils.McpLogf("[SessionProvider] Overriding db_name for %s: %q -> %q", targetID, credentials["db_name"], entry.dbName)
		credentials["db_name"] = entry.dbName
	}

	utils.McpLogf("[SessionProvider] Final credentials for %s: host=%s port=%s db_name=%s",
		targetID, credentials["host"], credentials["port"], credentials["db_name"])

	targetName := fmt.Sprintf("Apono: %s", session.Name)
	if entry.dbName != "" {
		targetName = fmt.Sprintf("Apono: %s / %s", session.Name, entry.dbName)
	}

	return &TargetDefinition{
		ID:            targetID,
		Name:          targetName,
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
	utils.McpLogf("[SessionProvider] Calling ResetAccessSessionCredentials for session %s", sessionID)
	_, httpResp, err := p.client.ClientAPI.AccessSessionsAPI.ResetAccessSessionCredentials(ctx, sessionID).Execute()
	if err != nil {
		return fmt.Errorf("failed to reset credentials: %w", err)
	}
	if httpResp != nil {
		utils.McpLogf("[SessionProvider] Reset API returned HTTP %d for session %s", httpResp.StatusCode, sessionID)
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		session, _, err := p.client.ClientAPI.AccessSessionsAPI.GetAccessSession(ctx, sessionID).Execute()
		if err != nil {
			return fmt.Errorf("failed to get session status: %w", err)
		}

		if session.Credentials.IsSet() {
			status := session.Credentials.Get().Status
			utils.McpLogf("[SessionProvider] Poll credential status for %s: %q", sessionID, status)
			if strings.EqualFold(status, "new") {
				utils.McpLogf("[SessionProvider] Credentials are fresh for session %s", sessionID)
				return nil
			}
		} else {
			utils.McpLogf("[SessionProvider] Credentials not set on session %s", sessionID)
		}

		select {
		case <-ctx.Done():
			utils.McpLogf("[SessionProvider] Context cancelled while waiting for reset of %s", sessionID)
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}

	return fmt.Errorf("timeout waiting for credentials reset")
}

// getCredentials returns credentials for a session.
// The unmasked password is only available briefly after reset (while status is "new").
// We cache it per session so all targets sharing a session reuse the same credentials.
func (p *SessionTargetProvider) getCredentials(ctx context.Context, session *clientapi.AccessSessionClientModel, targetID string) (map[string]string, error) {
	// Check if we already have cached unmasked credentials for this session
	p.resetMu.Lock()
	if cached, ok := p.cachedCreds[session.Id]; ok {
		p.resetMu.Unlock()
		// Return a copy so callers can modify (e.g., override db_name) without affecting cache
		result := make(map[string]string, len(cached))
		for k, v := range cached {
			result[k] = v
		}
		utils.McpLogf("[SessionProvider] Using cached credentials for %s (session %s)", targetID, session.Id)
		return result, nil
	}
	p.resetMu.Unlock()

	// Fetch access details
	fullDetails, _, err := p.client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, session.Id).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get access details: %w", err)
	}

	credentials := p.extractJSON(fullDetails)
	if credentials == nil {
		return nil, fmt.Errorf("no credentials available for target %q", targetID)
	}

	// Password not masked — cache and return
	if !hasOnlyMaskedPassword(fullDetails.Json) {
		p.cacheCredentials(session.Id, credentials)
		return credentials, nil
	}

	// Password is masked — try cache file first (fast, no rotation)
	if password, cacheErr := readPasswordFromCLI(fullDetails); cacheErr == nil {
		utils.McpLogf("[SessionProvider] Read password from cache file for %s (%d chars)", targetID, len(password))
		credentials["password"] = password
		p.cacheCredentials(session.Id, credentials)
		return credentials, nil
	}

	// No cache file — reset credentials once per session, then poll for unmasked password
	p.resetMu.Lock()
	alreadyReset := p.resetSessions[session.Id]
	if !alreadyReset {
		p.resetSessions[session.Id] = true
		p.resetMu.Unlock()
		utils.McpLogf("[SessionProvider] Resetting credentials for session %s (target %s)", session.Id, targetID)
		if resetErr := p.resetCredentials(ctx, session.Id); resetErr != nil {
			utils.McpLogf("[SessionProvider] Failed to reset credentials: %v", resetErr)
		}
	} else {
		p.resetMu.Unlock()
	}

	// Poll access details until password is unmasked (brief window after reset)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		fullDetails, _, err = p.client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, session.Id).Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to get access details after reset: %w", err)
		}

		if !hasOnlyMaskedPassword(fullDetails.Json) {
			credentials = p.extractJSON(fullDetails)
			utils.McpLogf("[SessionProvider] Got unmasked password for %s after reset", targetID)
			p.cacheCredentials(session.Id, credentials)
			return credentials, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}

	utils.McpLogf("[SessionProvider] WARNING: password still masked after reset for %s", targetID)
	return credentials, nil
}

// cacheCredentials stores unmasked credentials for a session
func (p *SessionTargetProvider) cacheCredentials(sessionID string, creds map[string]string) {
	p.resetMu.Lock()
	defer p.resetMu.Unlock()
	cached := make(map[string]string, len(creds))
	for k, v := range creds {
		cached[k] = v
	}
	p.cachedCreds[sessionID] = cached
}

// extractJSON extracts JSON credentials from access details as a string map
func (p *SessionTargetProvider) extractJSON(details *clientapi.AccessSessionDetailsClientModel) map[string]string {
	if len(details.Json) == 0 {
		return nil
	}
	credentials := make(map[string]string)
	for k, v := range details.Json {
		credentials[k] = fmt.Sprintf("%v", v)
	}
	return credentials
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

// readPasswordFromCLI extracts the real password from the local cache file referenced in the CLI field.
// The CLI field contains commands like: PGPASSWORD=$(base64 -d -i ~/.apono/cache/<session>) pgcli ...
// The password is stored base64-encoded in that cache file.
func readPasswordFromCLI(details *clientapi.AccessSessionDetailsClientModel) (string, error) {
	if !details.HasCli() {
		return "", fmt.Errorf("no CLI field in access details")
	}

	cli := details.GetCli()

	// Extract cache file path from patterns like:
	//   base64 -d -i ~/.apono/cache/<name>
	//   base64 --decode -i ~/.apono/cache/<name>
	re := regexp.MustCompile(`base64\s+(?:-d|--decode)\s+-i\s+([~\w/.@-]+)`)
	matches := re.FindStringSubmatch(cli)
	if len(matches) < 2 {
		return "", fmt.Errorf("no cache file path found in CLI: %s", cli)
	}

	cachePath := matches[1]
	// Expand ~ to home directory
	if strings.HasPrefix(cachePath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		cachePath = filepath.Join(home, cachePath[2:])
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return "", fmt.Errorf("failed to read cache file %s: %w", cachePath, err)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return "", fmt.Errorf("failed to base64 decode cache file %s: %w", cachePath, err)
	}

	password := strings.TrimSpace(string(decoded))
	if password == "" {
		return "", fmt.Errorf("empty password in cache file %s", cachePath)
	}

	return password, nil
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
