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

	// cacheDirMarker is how launcher commands refer to the session's cached password file.
	cacheDirMarker = ".apono/cache/"
)

func cachedPasswordPath(sessionID string) string {
	return filepath.Clean(filepath.Join(utils.DefaultCacheDir(), sessionID))
}

func readCachedPassword(sessionID string) (string, error) {
	raw, err := os.ReadFile(cachedPasswordPath(sessionID))
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
