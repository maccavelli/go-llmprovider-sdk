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
	Provider  llmprovider.ProviderID
	Access    string
	Refresh   string
	Expiry    time.Time
	Issuer    string
	ClientID  string
	AccountID string
	FedRAMP   bool // id-token chatgpt_account_is_fedramp: sends X-OpenAI-Fedramp (MADR 0012 §4.4)
	TokenURL  string
	// Resource is the resource a Sign in with ChatGPT plan session's tokens
	// were granted for; it is set only on such a session, which it marks
	// (PlanUsage). Its refresh and revocation are forms, and its requests go
	// to the platform, not the Codex backend (0043-MADR D1, D2).
	Resource string
	// Scope is a plan session's granted scopes, space-separated as the token
	// response sends them (0043-MADR A1).
	Scope string
	// IDToken is a plan session's last ID token, sent as a re-sign-in's
	// id_token_hint. It is a secret, hidden as the tokens are (R33).
	IDToken string
	// HostID is the ext_agent_host_id a plan session's client was
	// registered with.
	HostID string
	// Kind is the kind of sign-in the session came from. Every sign-in sets
	// it and FileTokenStore persists it. Empty means inferred, by the rules
	// SessionKind describes, so a record saved before it existed, and a
	// session a caller builds, behave as they did (0074-MADR D5).
	Kind       SessionKind
	Store      TokenStore
	HTTPClient *http.Client
	// Logger receives a failure to save a rotated session, which the session
	// keeps and retries (0016-MADR D4). Nil logs nothing; the process-global
	// logger is never used (0015-MADR D9).
	Logger   *slog.Logger
	mu       sync.Mutex
	inflight *tokenFuture
	// storedRefresh is the refresh token Store is known to hold: set by
	// Load, by every save that succeeds, and by a session adopted from Store.
	// While it differs from Refresh, a rotation is unsaved, and the store's
	// token is one this session spent, never a sibling's rotation; it stays
	// the store's token however many saves fail in a row (0026-MADR F1).
	// "" when unknown.
	storedRefresh string
	// clientDefault reports that HTTPClient is a provider's default, given by
	// UseDefaultHTTPClient, which a caller's client replaces (0026-MADR F37).
	clientDefault bool
	// issued is when the last refresh or exchange succeeded, in this process;
	// it measures the token's lifetime when the token is not a JWT with iat
	// (0021-MADR T3).
	issued time.Time
	// identity is the application the session's refresh and revocation name
	// (0079-MADR D2): the sign-in's Identity, or the first non-zero one
	// UseIdentity gave. It lives in memory only: no store holds it, and
	// String, GoString, MarshalJSON and LogValue do not show it. A refreshed
	// session and a sibling's rotation adopted from the store leave it alone.
	identity Identity
	// nextRefresh is when a refresh may next be tried after an early one
	// failed (0021-MADR T2); nextResave is when an unsaved rotation's save
	// may next be tried (T10).
	nextRefresh, nextResave time.Time
	// now is the session's clock; nil is time.Now. Tests set it. It is an
	// interface, not a func, so that OAuthSession stays comparable.
	now sessionClock
	// onJoin, when set, is told as a caller joins the refresh in flight:
	// tests wait on it rather than sleep (0021-MADR Z12). An interface, as now
	// is, so that OAuthSession stays comparable.
	onJoin joinHook
}

// SessionKind is the kind of sign-in a stored OAuthSession came from. Its
// strings are those of the auth contract's SessionKind (0074-MADR D5). It is a
// value, safe for concurrent use while no goroutine changes it.
type SessionKind string

// The session kinds. SessionChatGPT is a Sign in with ChatGPT session for the
// Codex backend; SessionChatGPTPlan is a ChatGPT plan session on the
// platform's Responses API (LoginChatGPTPlan); SessionGrok is an xAI Grok
// sign-in; SessionKiloDevice is a Kilo device sign-in, which holds an access
// token only.
const (
	SessionChatGPT     SessionKind = "chatgpt"
	SessionChatGPTPlan SessionKind = "chatgpt_plan"
	SessionGrok        SessionKind = "grok"
	SessionKiloDevice  SessionKind = "kilo_device"
)

// joinHook is told when a caller joins a refresh in flight.
type joinHook interface{ joined() }

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
	text := fmt.Sprintf("OAuthSession{Provider:%s Issuer:%s ClientID:%s AccountID:%s FedRAMP:%t TokenURL:%s Expiry:%s Access:%s Refresh:%s",
		s.Provider, s.Issuer, s.ClientID, s.AccountID, s.FedRAMP, s.TokenURL, expiryText(s.Expiry),
		secretText(s.Access), secretText(s.Refresh))
	if s.Kind != "" {
		text += fmt.Sprintf(" Kind:%s", s.Kind)
	}
	if s.planFields() {
		text += fmt.Sprintf(" Resource:%s Scope:%q HostID:%s IDToken:%s", s.Resource, s.Scope, s.HostID, secretText(s.IDToken))
	}
	return text + "}"
}

// planFields reports that the session carries a plan session's fields, which
// its formatters then show; another session's forms are as they were.
func (s *OAuthSession) planFields() bool {
	return s.Resource != "" || s.Scope != "" || s.IDToken != "" || s.HostID != ""
}

// GoString renders the session for %#v, without its tokens.
func (s *OAuthSession) GoString() string { return "&auth." + s.String() }

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
		Resource  string                 `json:"resource,omitempty"`
		Scope     string                 `json:"scope,omitempty"`
		HostID    string                 `json:"host_id,omitempty"`
		IDToken   string                 `json:"id_token,omitempty"`
		Kind      SessionKind            `json:"kind,omitempty"`
	}{s.Provider, s.Issuer, s.ClientID, s.AccountID, s.FedRAMP, s.TokenURL, expiryText(s.Expiry),
		secretText(s.Access), secretText(s.Refresh), s.Resource, s.Scope, s.HostID, secretText(s.IDToken), s.Kind})
}

// LogValue renders the session for slog, without its tokens.
func (s *OAuthSession) LogValue() slog.Value {
	if s == nil {
		return slog.StringValue("OAuthSession(nil)")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	attrs := []slog.Attr{
		slog.String("provider", string(s.Provider)),
		slog.String("issuer", s.Issuer),
		slog.String("client_id", s.ClientID),
		slog.String("account_id", s.AccountID),
		slog.String("expiry", expiryText(s.Expiry)),
		slog.String("access", secretText(s.Access)),
		slog.String("refresh", secretText(s.Refresh)),
	}
	if s.Kind != "" {
		attrs = append(attrs, slog.String("kind", string(s.Kind)))
	}
	if s.planFields() {
		attrs = append(attrs,
			slog.String("resource", s.Resource),
			slog.String("scope", s.Scope),
			slog.String("host_id", s.HostID),
			slog.String("id_token", secretText(s.IDToken)),
		)
	}
	return slog.GroupValue(attrs...)
}

// TokenStore persists OAuth sessions, one per provider. Load returns nil and no
// error when the provider has none. FileTokenStore is the built-in one. A
// TokenStore shared by several OAuthSessions must be safe for concurrent use:
// each session loads and saves from its own refresh, while the others do.
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
// LockRefresh must be safe for concurrent use, since sessions sharing the
// store call it at once, and must exclude the process's other callers as it
// does other processes'.
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
	// A plan session's fields (0043-MADR D3, A1); absent from every other
	// session's record, so an older record reads as before.
	Resource string `json:"resource,omitempty"`
	Scope    string `json:"scope,omitempty"`
	IDToken  string `json:"id_token,omitempty"`
	HostID   string `json:"host_id,omitempty"`
	// Kind is the session's kind (0074-MADR D5); absent from an older
	// record, which then infers it.
	Kind SessionKind `json:"kind,omitempty"`
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
