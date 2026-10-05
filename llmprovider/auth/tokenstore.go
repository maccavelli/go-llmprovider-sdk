package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// OAuthSession is a subscription login: a TokenSource that refreshes its access
// token before expiry, and after a 401, and saves each rotation to Store. It is
// safe for concurrent use, and one refresh runs at a time, across processes
// when Store is a RefreshLocker. Format a *OAuthSession: its String, GoString
// and LogValue never show the tokens.
type OAuthSession struct {
	Provider   llmprovider.ProviderID
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
	// issued is when the last refresh or exchange succeeded, in this process;
	// it measures the token's lifetime when the token is not a JWT with iat
	// (0021-MADR T3).
	issued time.Time
	// nextRefresh is when a refresh may next be tried after an early one
	// failed (0021-MADR T2); nextResave is when an unsaved rotation's save
	// may next be tried (T10).
	nextRefresh, nextResave time.Time
	// now is the session's clock; nil is time.Now. Tests set it. It is an
	// interface, not a func, so that OAuthSession stays comparable.
	now sessionClock
}

// sessionClock tells an OAuthSession the time.
type sessionClock interface{ Now() time.Time }

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

// MarshalJSON encodes the session without its tokens; see Token.MarshalJSON.
// FileTokenStore persists sessions through its own record type, unaffected.
func (s *OAuthSession) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Marshal(struct {
		Provider  llmprovider.ProviderID `json:"provider"`
		Issuer    string                 `json:"issuer"`
		ClientID  string                 `json:"client_id"`
		AccountID string                 `json:"account_id"`
		FedRAMP   bool                   `json:"fedramp"`
		TokenURL  string                 `json:"token_url"`
		Expiry    string                 `json:"expiry"`
		Access    string                 `json:"access"`
		Refresh   string                 `json:"refresh"`
	}{s.Provider, s.Issuer, s.ClientID, s.AccountID, s.FedRAMP, s.TokenURL, expiryText(s.Expiry),
		secretText(s.Access), secretText(s.Refresh)})
}

// LogValue renders the session for slog, without its tokens.
func (s *OAuthSession) LogValue() slog.Value {
	if s == nil {
		return slog.StringValue("OAuthSession(nil)")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return slog.GroupValue(
		slog.String("provider", string(s.Provider)),
		slog.String("issuer", s.Issuer),
		slog.String("client_id", s.ClientID),
		slog.String("account_id", s.AccountID),
		slog.String("expiry", expiryText(s.Expiry)),
		slog.String("access", secretText(s.Access)),
		slog.String("refresh", secretText(s.Refresh)),
	)
}

// TokenStore persists OAuth sessions, one per provider. Load returns nil and no
// error when the provider has none. FileTokenStore is the built-in one.
type TokenStore interface {
	Load(ctx context.Context, provider llmprovider.ProviderID) (*OAuthSession, error)
	Save(ctx context.Context, provider llmprovider.ProviderID, s *OAuthSession) error
	Delete(ctx context.Context, provider llmprovider.ProviderID) error
}

// RefreshLocker is implemented by a TokenStore that can serialise refreshes
// across processes; FileTokenStore does. An OAuthSession takes the lock
// around a refresh and re-reads the store once it holds it, so a refresh
// token is never spent twice: both vendors revoke the whole token family on
// reuse (0016-MADR amendment A2). The returned func releases the lock.
type RefreshLocker interface {
	LockRefresh(ctx context.Context, provider llmprovider.ProviderID) (unlock func(), err error)
}

// fileRecord is the on-disk JSON shape.
type fileRecord struct {
	Provider  llmprovider.ProviderID `json:"provider"`
	Access    string                 `json:"access"`
	Refresh   string                 `json:"refresh"`
	Expiry    time.Time              `json:"expiry"`
	Issuer    string                 `json:"issuer"`
	ClientID  string                 `json:"client_id"`
	AccountID string                 `json:"account_id"`
	FedRAMP   bool                   `json:"fedramp,omitempty"`
	TokenURL  string                 `json:"token_url"`
}

// windowsDeviceNames are the names Windows opens as a device rather than a
// file, whatever the extension (0010-MADR D12).
var windowsDeviceNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// validateProviderID refuses an id that is not a plain file name on every OS:
// empty, a path, a Windows device name with or without an extension, or an
// id ending in a dot or a space, which Windows strips (0010-MADR D12). The
// rule is the same on every OS, so a store copied onto Windows still works.
func validateProviderID(p llmprovider.ProviderID) error {
	if p == "" {
		return fmt.Errorf("%w: empty", llmprovider.ErrInvalidProvider)
	}
	id := string(p)
	base, _, _ := strings.Cut(id, ".")
	if strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") ||
		strings.HasSuffix(id, ".") || strings.HasSuffix(id, " ") ||
		windowsDeviceNames[strings.ToUpper(base)] {
		return fmt.Errorf("%w: %q", llmprovider.ErrInvalidProvider, p)
	}
	return nil
}
