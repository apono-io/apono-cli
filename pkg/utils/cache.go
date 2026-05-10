package utils

import (
	"os"
	"path/filepath"
)

func DefaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".apono", "cache")
	}
	return filepath.Join(home, ".apono", "cache")
}
