package aponoapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	userIDEnvVar = "APONO_USER_ID"
)

// GetAccessSessionAccessDetailsWithUserID fetches access session details and includes userId as a query parameter.
// The userId is taken from (in order of priority):
// 1. APONO_USER_ID environment variable
// 2. The userId from the client session (from profile config)
func (c *AponoClient) GetAccessSessionAccessDetailsWithUserID(ctx context.Context, sessionID string) (*clientapi.AccessSessionDetailsClientModel, *http.Response, error) {
	// Determine userId to use
	userID := os.Getenv(userIDEnvVar)
	if userID == "" && c.Session != nil {
		userID = c.Session.UserID
	}

	// If no userId is available, fall back to the standard API call
	if userID == "" {
		return c.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, sessionID).Execute()
	}

	// Build the URL manually to add custom query parameters
	cfg := c.ClientAPI.GetConfig()
	if len(cfg.Servers) == 0 {
		return nil, nil, fmt.Errorf("no servers configured in API client")
	}

	// Build full URL with scheme and host
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "https"
	}
	host := cfg.Host
	fullURL := fmt.Sprintf("%s://%s/api/client/v1/access-sessions/%s/access-details", scheme, host, url.PathEscape(sessionID))

	// Parse URL and add query parameter
	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Add("user_id", userID)
	u.RawQuery = q.Encode()

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", cfg.UserAgent)

	// Execute request (the HTTP client already has auth configured)
	httpResp, err := cfg.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, httpResp, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if httpResp.StatusCode >= 300 {
		return nil, httpResp, fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result clientapi.AccessSessionDetailsClientModel
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, httpResp, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, httpResp, nil
}

// ListAccessSessionsWithUserID fetches access sessions list and includes userId as a query parameter.
// The userId is taken from (in order of priority):
// 1. APONO_USER_ID environment variable
// 2. The userId from the client session (from profile config)
func (c *AponoClient) ListAccessSessionsWithUserID(ctx context.Context, skip int32, bundleID, integrationID, requestID []string) (*clientapi.PaginatedClientResponseModelAccessSessionClientModel, *http.Response, error) {
	// Determine userId to use
	userID := os.Getenv(userIDEnvVar)
	if userID == "" && c.Session != nil {
		userID = c.Session.UserID
	}

	// If no userId is available, fall back to the standard API call
	if userID == "" {
		req := c.ClientAPI.AccessSessionsAPI.ListAccessSessions(ctx).Skip(skip)
		if integrationID != nil {
			req = req.IntegrationId(integrationID)
		}
		if bundleID != nil {
			req = req.BundleId(bundleID)
		}
		if requestID != nil {
			req = req.RequestId(requestID)
		}
		return req.Execute()
	}

	// Build the URL manually to add custom query parameters
	cfg := c.ClientAPI.GetConfig()
	if len(cfg.Servers) == 0 {
		return nil, nil, fmt.Errorf("no servers configured in API client")
	}

	// Build full URL with scheme and host
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "https"
	}
	host := cfg.Host
	fullURL := fmt.Sprintf("%s://%s/api/client/v1/access-sessions", scheme, host)

	// Parse URL and add query parameters
	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Add("user_id", userID)
	q.Add("skip", strconv.Itoa(int(skip)))

	// Add optional filters
	if integrationID != nil {
		for _, id := range integrationID {
			q.Add("integration_id", id)
		}
	}
	if bundleID != nil {
		for _, id := range bundleID {
			q.Add("bundle_id", id)
		}
	}
	if requestID != nil {
		for _, id := range requestID {
			q.Add("request_id", id)
		}
	}

	u.RawQuery = q.Encode()

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", cfg.UserAgent)

	// Execute request (the HTTP client already has auth configured)
	httpResp, err := cfg.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, httpResp, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if httpResp.StatusCode >= 300 {
		return nil, httpResp, fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result clientapi.PaginatedClientResponseModelAccessSessionClientModel
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, httpResp, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, httpResp, nil
}

// GetAccessSessionWithUserID fetches a single access session and includes userId as a query parameter.
// The userId is taken from (in order of priority):
// 1. APONO_USER_ID environment variable
// 2. The userId from the client session (from profile config)
func (c *AponoClient) GetAccessSessionWithUserID(ctx context.Context, sessionID string) (*clientapi.AccessSessionClientModel, *http.Response, error) {
	// Determine userId to use
	userID := os.Getenv(userIDEnvVar)
	if userID == "" && c.Session != nil {
		userID = c.Session.UserID
	}

	// If no userId is available, fall back to the standard API call
	if userID == "" {
		return c.ClientAPI.AccessSessionsAPI.GetAccessSession(ctx, sessionID).Execute()
	}

	// Build the URL manually to add custom query parameters
	cfg := c.ClientAPI.GetConfig()
	if len(cfg.Servers) == 0 {
		return nil, nil, fmt.Errorf("no servers configured in API client")
	}

	// Build full URL with scheme and host
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "https"
	}
	host := cfg.Host
	fullURL := fmt.Sprintf("%s://%s/api/client/v1/access-sessions/%s", scheme, host, url.PathEscape(sessionID))

	// Parse URL and add query parameter
	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Add("user_id", userID)
	u.RawQuery = q.Encode()

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", cfg.UserAgent)

	// Execute request (the HTTP client already has auth configured)
	httpResp, err := cfg.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, httpResp, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if httpResp.StatusCode >= 300 {
		return nil, httpResp, fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result clientapi.AccessSessionClientModel
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, httpResp, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, httpResp, nil
}

// ResetAccessSessionCredentialsWithUserID resets credentials for an access session and includes userId as a query parameter.
// The userId is taken from (in order of priority):
// 1. APONO_USER_ID environment variable
// 2. The userId from the client session (from profile config)
func (c *AponoClient) ResetAccessSessionCredentialsWithUserID(ctx context.Context, sessionID string) (*http.Response, error) {
	// Determine userId to use
	userID := os.Getenv(userIDEnvVar)
	if userID == "" && c.Session != nil {
		userID = c.Session.UserID
	}

	// If no userId is available, fall back to the standard API call
	if userID == "" {
		_, resp, err := c.ClientAPI.AccessSessionsAPI.ResetAccessSessionCredentials(ctx, sessionID).Execute()
		return resp, err
	}

	// Build the URL manually to add custom query parameters
	cfg := c.ClientAPI.GetConfig()
	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("no servers configured in API client")
	}

	// Build full URL with scheme and host
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "https"
	}
	host := cfg.Host
	fullURL := fmt.Sprintf("%s://%s/api/client/v1/access-sessions/%s/reset-credentials", scheme, host, url.PathEscape(sessionID))

	// Parse URL and add query parameter
	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Add("user_id", userID)
	u.RawQuery = q.Encode()

	// Create HTTP request (POST method)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", cfg.UserAgent)

	// Execute request (the HTTP client already has auth configured)
	httpResp, err := cfg.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body for error checking
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return httpResp, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if httpResp.StatusCode >= 300 {
		return httpResp, fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	return httpResp, nil
}
