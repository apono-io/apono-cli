package aponoapi

import (
	"context"
	"sync"

	"golang.org/x/oauth2"
)

// TokenUpdateFunc is a function that accepts an oauth2 Token upon refresh, and
// returns an error if it should not be used.
type TokenUpdateFunc func(*oauth2.Token) error

func NewRefreshableTokenSource(
	ctx context.Context,
	cfg oauth2.Config,
	token *oauth2.Token,
	f TokenUpdateFunc,
) oauth2.TokenSource {
	return &refreshableTokenSource{
		new: cfg.TokenSource(ctx, token),
		t:   token,
		f:   f,
	}
}

// refreshableTokenSource is essentially `oauth2.reuseTokenSource` with `TokenUpdateFunc` added.
type refreshableTokenSource struct {
	new oauth2.TokenSource
	mu  sync.Mutex // guards t
	t   *oauth2.Token
	f   TokenUpdateFunc // called when token refreshed so new refresh token can be persisted
}

// Token returns the current token if it's still valid, else will
// refresh the current token (using r.Context for HTTP client information) and return the new one.
func (s *refreshableTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.t.Valid() {
		return s.t, nil
	}

	t, err := s.new.Token()
	if err != nil {
		return nil, err
	}

	s.t = t
	return t, s.f(t)
}
