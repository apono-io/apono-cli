package proxy

import (
	"fmt"
	"strings"
)

// NamespaceSeparator is used to separate backend ID from tool names
const NamespaceSeparator = "__"

// PrefixToolName adds backend prefix to tool name
func PrefixToolName(backendID, toolName string) string {
	return backendID + NamespaceSeparator + toolName
}

// ParseToolName extracts backend ID and original tool name from prefixed name
func ParseToolName(prefixedName string) (backendID, toolName string, err error) {
	parts := strings.SplitN(prefixedName, NamespaceSeparator, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid tool name format: %s (expected backend__toolname)", prefixedName)
	}
	return parts[0], parts[1], nil
}

// HasNamespacePrefix checks if a tool name contains the namespace separator
func HasNamespacePrefix(name string) bool {
	return strings.Contains(name, NamespaceSeparator)
}
