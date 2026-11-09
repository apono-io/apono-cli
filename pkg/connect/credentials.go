package connect

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	passwordPlaceholder = "__APONO_PASSWORD__"

	passwordEncodingURL = "url"

	redactedMarker = "***"
)

func readCachedPassword(sessionID string) (string, error) {
	cacheDir := filepath.Clean(utils.DefaultCacheDir())
	path := filepath.Join(cacheDir, sessionID)
	if rel, err := filepath.Rel(cacheDir, path); err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid session id %q: cache path escapes cache directory", sessionID)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // path is confined to the cache directory above
	if err != nil {
		return "", fmt.Errorf("read cache file: %w", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return "", fmt.Errorf("decode cache content: %w", err)
	}
	return strings.TrimRight(string(decoded), "\n\r"), nil
}

func encodePassword(raw, encoding string) string {
	switch encoding {
	case passwordEncodingURL:
		return url.QueryEscape(raw)
	default:
		return raw
	}
}

func withoutSecrets(err error, secrets []string) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	seen := make(map[string]bool, len(secrets))
	for _, secret := range secrets {
		if secret == "" || seen[secret] {
			continue
		}
		seen[secret] = true
		message = strings.ReplaceAll(message, secret, redactedMarker)
	}
	return errors.New(message)
}
