package registry

import (
	"bytes"
	"fmt"
	"net/url"
	"text/template"
)

// templateFuncs provides helper functions available in credential templates.
var templateFuncs = template.FuncMap{
	"urlEncode": url.QueryEscape,
}

// BuildCredentials renders credential templates from an MCPServerDefinition
// against raw session fields. If no CredentialBuilder is defined, raw fields
// are passed through as-is.
func BuildCredentials(def MCPServerDefinition, rawFields map[string]string) (map[string]string, error) {
	if len(def.CredentialBuilder) == 0 {
		result := make(map[string]string, len(rawFields))
		for k, v := range rawFields {
			result[k] = v
		}
		return result, nil
	}

	result := make(map[string]string, len(def.CredentialBuilder))
	for key, tmplStr := range def.CredentialBuilder {
		tmpl, err := template.New(key).Funcs(templateFuncs).Option("missingkey=error").Parse(tmplStr)
		if err != nil {
			return nil, fmt.Errorf("invalid template for credential %q: %w", key, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, rawFields); err != nil {
			return nil, fmt.Errorf("failed to render credential %q: %w", key, err)
		}

		result[key] = buf.String()
	}

	return result, nil
}
