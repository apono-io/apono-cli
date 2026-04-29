package config

import (
	"fmt"
)

// GetProfileByAccountID scans configured profiles and returns the name and
// session of the first profile whose account_id matches.
//
// Returns ErrNoProfiles if no profiles are configured at all, or a wrapped
// ErrProfileNotExists when no profile matches the given account ID.
//
// Iteration over the profiles map is non-deterministic; callers must not rely
// on a stable winner if multiple profiles share the same account_id (which
// shouldn't happen in practice — `apono login` to the same account
// overwrites rather than duplicates).
func GetProfileByAccountID(accountID string) (ProfileName, *SessionConfig, error) {
	cfg, err := Get()
	if err != nil {
		return "", nil, err
	}
	if len(cfg.Auth.Profiles) == 0 {
		return "", nil, ErrNoProfiles
	}
	for name, sess := range cfg.Auth.Profiles {
		if sess.AccountID == accountID {
			session := sess
			return name, &session, nil
		}
	}
	return "", nil, fmt.Errorf("account_id %s: %w", accountID, ErrProfileNotExists)
}
