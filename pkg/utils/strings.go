package utils

import "github.com/google/uuid"

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
