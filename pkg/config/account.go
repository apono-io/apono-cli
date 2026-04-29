package config

import (
	"fmt"
)

func GetProfileByAccountID(accountID string) (ProfileName, error) {
	cfg, err := Get()
	if err != nil {
		return "", err
	}
	if len(cfg.Auth.Profiles) == 0 {
		return "", ErrNoProfiles
	}
	for name, sess := range cfg.Auth.Profiles {
		if sess.AccountID == accountID {
			return name, nil
		}
	}
	return "", fmt.Errorf("account_id %s: %w", accountID, ErrProfileNotExists)
}
