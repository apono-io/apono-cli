package utils

import (
	"github.com/google/uuid"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func Contains(array []string, str string) bool {
	for _, s := range array {
		if str == s {
			return true
		}
	}
	return false
}

func FromNullableString(s clientapi.NullableString) string {
	if !s.IsSet() || s.Get() == nil {
		return ""
	}
	return *s.Get()
}
