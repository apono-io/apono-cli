package connect

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	passwordPlaceholder = "__APONO_PASSWORD__"

	passwordEncodingURL = "url"

	redactedMarker = "***"
)

const (
	awsErrorLinePrefix    = "aws: [ERROR]: "
	maxAwsErrorLineLength = 500
	awsAccountIDLength    = 12
)

var awsErrorMessageStarts = []string{
	"An error occurred (",
	"The SSO session",
	"Error loading SSO Token",
	"Error when retrieving token from sso",
	"Unable to locate credentials",
	"The config profile (",
	"The source_profile ",
}

var (
	digitRun              = regexp.MustCompile(`\d+`)
	awsAssumedRoleSession = regexp.MustCompile(`(assumed-role/[^/\s]+/)\S+`)
	awsIamUserName        = regexp.MustCompile(`(:user/)\S+`)
	emailAddress          = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
)

func awsErrorLines(output string) (kept []string, withheld int) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isAwsErrorLine(line) {
			kept = append(kept, withoutAwsIdentity(line))
			continue
		}
		withheld++
	}
	return kept, withheld
}

func isAwsErrorLine(line string) bool {
	if len(line) > maxAwsErrorLineLength {
		return false
	}
	message := strings.TrimPrefix(line, awsErrorLinePrefix)
	for _, start := range awsErrorMessageStarts {
		if strings.HasPrefix(message, start) {
			return true
		}
	}
	return false
}

func withoutAwsIdentity(line string) string {
	line = digitRun.ReplaceAllStringFunc(line, func(digits string) string {
		if len(digits) == awsAccountIDLength {
			return redactedMarker
		}
		return digits
	})
	line = awsAssumedRoleSession.ReplaceAllString(line, "${1}"+redactedMarker)
	line = awsIamUserName.ReplaceAllString(line, "${1}"+redactedMarker)
	return emailAddress.ReplaceAllString(line, redactedMarker)
}

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
