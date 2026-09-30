package llmprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OAuthSession is declared here as a minimal stub so Phase 2's FileTokenStore
// can be tested in isolation. Phase 3 will extend this type with the full
// refresh + persist-before-use behaviour and the locked method set.
type OAuthSession struct {
	Provider   string
	Access     string
	Refresh    string
	Expiry     time.Time
	Issuer     string
	ClientID   string
	AccountID  string
	FedRAMP    bool // id-token chatgpt_account_is_fedramp: sends X-OpenAI-Fedramp (MADR 0012 §4.4)
	TokenURL   string
	Store      TokenStore
	HTTPClient *http.Client
	mu         sync.Mutex
	inflight   *tokenFuture
}

type tokenFuture struct {
	done chan struct{}
	tok  Token
	err  error
}

// TokenStore interface (Phase 2 introduces this; Phase 3 adds methods that
// depend on it).
type TokenStore interface {
	Load(ctx context.Context, provider string) (*OAuthSession, error)
	Save(ctx context.Context, provider string, s *OAuthSession) error
	Delete(ctx context.Context, provider string) error
}

// RefreshLocker is implemented by a TokenStore that can serialise refreshes
// across processes; FileTokenStore does. An OAuthSession takes the lock
// around a refresh and re-reads the store once it holds it, so a refresh
// token is never spent twice: both vendors revoke the whole token family on
// reuse (0016-MADR amendment A2). The returned func releases the lock.
type RefreshLocker interface {
	LockRefresh(ctx context.Context, provider string) (unlock func(), err error)
}

// fileRecord is the on-disk JSON shape.
type fileRecord struct {
	Provider  string    `json:"provider"`
	Access    string    `json:"access"`
	Refresh   string    `json:"refresh"`
	Expiry    time.Time `json:"expiry"`
	Issuer    string    `json:"issuer"`
	ClientID  string    `json:"client_id"`
	AccountID string    `json:"account_id"`
	FedRAMP   bool      `json:"fedramp,omitempty"`
	TokenURL  string    `json:"token_url"`
}

// ErrInvalidProvider is returned when a provider id is empty or path-traverses.
var ErrInvalidProvider = errors.New("invalid provider id")

func validateProviderID(p string) error {
	if p == "" {
		return fmt.Errorf("%w: empty", ErrInvalidProvider)
	}
	if strings.ContainsAny(p, `/\`) || strings.Contains(p, "..") {
		return fmt.Errorf("%w: %q", ErrInvalidProvider, p)
	}
	return nil
}
