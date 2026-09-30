package llmprovider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	// Logger receives a failure to save a rotated session, which the session
	// keeps and retries (0016-MADR D4). Nil logs nothing; the process-global
	// logger is never used (0015-MADR D9).
	Logger   *slog.Logger
	mu       sync.Mutex
	inflight *tokenFuture
	// spentRefresh is the refresh token the last refresh spent, while its
	// rotated session is not yet saved; "" once saved.
	spentRefresh string
}

// String renders the session without its access or refresh token
// (0016-MADR D5). It takes the session's lock, so it must not be called while
// the lock is held. Format a *OAuthSession: copying the struct is a vet error,
// and a copy has no String method.
func (s *OAuthSession) String() string {
	if s == nil {
		return "OAuthSession(nil)"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("OAuthSession{Provider:%s Issuer:%s ClientID:%s AccountID:%s FedRAMP:%t TokenURL:%s Expiry:%s Access:%s Refresh:%s}",
		s.Provider, s.Issuer, s.ClientID, s.AccountID, s.FedRAMP, s.TokenURL, expiryText(s.Expiry),
		secretText(s.Access), secretText(s.Refresh))
}

// GoString renders the session for %#v, without its tokens.
func (s *OAuthSession) GoString() string { return "&llmprovider." + s.String() }

// LogValue renders the session for slog, without its tokens.
func (s *OAuthSession) LogValue() slog.Value {
	if s == nil {
		return slog.StringValue("OAuthSession(nil)")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return slog.GroupValue(
		slog.String("provider", s.Provider),
		slog.String("issuer", s.Issuer),
		slog.String("client_id", s.ClientID),
		slog.String("account_id", s.AccountID),
		slog.String("expiry", expiryText(s.Expiry)),
		slog.String("access", secretText(s.Access)),
		slog.String("refresh", secretText(s.Refresh)),
	)
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
