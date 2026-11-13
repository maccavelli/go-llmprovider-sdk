package llmprovider_test

// The checks the person-gated live tests run after their sign-in, untagged so
// that an offline twin (live_twins_test.go) runs each against a fake and a
// planted break of a check is caught with no sign-in (0046-MADR D3). The live
// tests call them with the real session; the twins with a fake's.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/grok"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/kilo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// liveOpenAI builds OpenAI from a session, or fails the test.
func liveOpenAI(t *testing.T, src llmprovider.TokenSource, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := openai.New(append([]llmprovider.Option{llmprovider.WithTokenSource(src), llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func liveGrok(t *testing.T, src llmprovider.TokenSource, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := grok.New(append([]llmprovider.Option{llmprovider.WithTokenSource(src), llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func userText(text string) *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: "user", Text: text}}}
}

// transientSkip is how a check skips a transient failure: the live suite's
// llmprovider.SkipIfTransient where a live test already skipped, noSkip where
// it did not, and noSkip in every twin.
type transientSkip func(*testing.T, error)

func noSkip(*testing.T, error) {}

// checkWord asks p for word alone, and fails unless the reply holds it.
func checkWord(ctx context.Context, t *testing.T, p llmprovider.Provider, word string, skip transientSkip) {
	t.Helper()
	out, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word "+word))
	skip(t, err)
	if err != nil || !strings.Contains(strings.ToUpper(out), word) {
		t.Fatalf("Generate = %q, %v", out, err)
	}
}

// checkKiloSession checks a Kilo device login's session: an access token with
// no refresh and no expiry, the account's profile (counts only), and one
// generation on model.
func checkKiloSession(ctx context.Context, t *testing.T, session *auth.OAuthSession, model string, skip transientSkip,
	opts ...llmprovider.Option) {
	t.Helper()
	if session.Access == "" || session.Refresh != "" || !session.Expiry.IsZero() {
		t.Fatalf("session: access set %t, refresh set %t, expiry %v; want an access token only, with no refresh and no expiry",
			session.Access != "", session.Refresh != "", session.Expiry)
	}
	account, err := auth.KiloProfile(ctx, session.Access, opts...)
	if err != nil {
		t.Fatalf("KiloProfile: %v", err)
	}
	t.Logf("profile: %d organizations, personal account %t", len(account.Organizations), account.HasPersonalAccount)
	p, err := kilo.New(append([]llmprovider.Option{llmprovider.WithTokenSource(session), kilo.WithDataCollection(true),
		llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatalf("kilo.New: %v", err)
	}
	out, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word ALPHA"))
	skip(t, err)
	if err != nil || strings.TrimSpace(out) == "" {
		t.Fatalf("Generate = %q, %v", out, err)
	}
}

// checkPlanKind checks a session from LoginChatGPTPlan: a plan session, not a
// ChatGPT one, under an issued client, with plan usage granted and a host id.
// It logs no token and no client id.
func checkPlanKind(t *testing.T, s *auth.OAuthSession) {
	t.Helper()
	if !s.PlanUsage() || s.ChatGPT() || !strings.HasPrefix(s.ClientID, "oaia") ||
		!strings.Contains(" "+s.Scope+" ", " chatgpt.tokens.use.direct ") || s.HostID == "" {
		t.Fatalf("session: plan %v, ChatGPT %v, issued client %v (%d chars), scope has plan usage %v, host id %v",
			s.PlanUsage(), s.ChatGPT(), strings.HasPrefix(s.ClientID, "oaia"), len(s.ClientID),
			strings.Contains(s.Scope, "chatgpt.tokens.use.direct"), s.HostID != "")
	}
	t.Logf("FINDING sign-in: issued client of %d chars; token URL %s", len(s.ClientID), s.TokenURL)
}

// planStoreRoundTrip saves signedIn to a FileTokenStore, and returns what the
// store gives back, bound to it.
func planStoreRoundTrip(ctx context.Context, t *testing.T, signedIn *auth.OAuthSession) (*auth.OAuthSession, auth.TokenStore) {
	t.Helper()
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, signedIn); err != nil {
		t.Fatalf("Save: %v", err)
	}
	session, err := store.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil || session == nil || !session.PlanUsage() || session.ClientID != signedIn.ClientID ||
		session.Scope != signedIn.Scope || session.HostID != signedIn.HostID || session.IDToken == "" {
		t.Fatalf("the stored session does not round-trip: err %v", err)
	}
	session.Store = store
	return session, store
}

// planRequests lists the account's models with probes asked for, then
// generates, streams and calls a forced tool on the first. opts reach every
// provider and listing it builds.
func planRequests(ctx context.Context, t *testing.T, session *auth.OAuthSession, skip transientSkip, opts ...llmprovider.Option) {
	t.Helper()
	lister := liveOpenAI(t, session, "", append([]llmprovider.Option{llmprovider.WithModelProbes(true)}, opts...)...).(llmprovider.ModelLister)
	models, err := lister.ListModels(ctx)
	skip(t, err)
	if err != nil || len(models) == 0 {
		t.Fatalf("ListModels = %v, %v", models, err)
	}
	t.Logf("FINDING listing: %d listed models, the first %q", len(models), models[0])
	raw, err := catalog.ListRaw(ctx, llmprovider.ProviderOpenAI, session, opts...)
	if err != nil || len(raw.Models) < len(models) || raw.Models[0].ID == "" {
		t.Fatalf("ListRaw: %d entries, %v", len(raw.Models), err)
	}
	p := liveOpenAI(t, session, models[0], opts...)

	out, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word ALPHA"))
	skip(t, err)
	if err != nil || !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Fatalf("Generate = %q, %v", out, err)
	}

	var streamed strings.Builder
	done := false
	for ev, err := range llmprovider.Stream(ctx, p, userText("Reply with only the word BRAVO")) {
		skip(t, err)
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if ev.Type == llmprovider.EventTextDelta {
			streamed.WriteString(ev.Text)
		}
		done = done || ev.Type == llmprovider.EventDone
	}
	if !done || !strings.Contains(strings.ToUpper(streamed.String()), "BRAVO") {
		t.Fatalf("Stream = %q, done %v", streamed.String(), done)
	}

	tool := llmprovider.Tool{Name: "get_weather", Description: "Get the weather for a city", Schema: map[string]any{
		"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []string{"city"}}}
	call, err := llmprovider.GenerateToolCall(ctx, p, &llmprovider.Request{
		Input: userText("What is the weather in Paris?").Input, Tools: []llmprovider.Tool{tool},
		ToolChoice: llmprovider.ForceTool(tool.Name)})
	skip(t, err)
	if err != nil || call.Name != tool.Name || !strings.Contains(call.Arguments, "Paris") {
		t.Fatalf("forced tool = %+v, %v", call, err)
	}
}

// planRefresh forces a refresh, and checks that the token rotated and that
// the store holds the rotation with its plan fields.
func planRefresh(ctx context.Context, t *testing.T, session *auth.OAuthSession, store auth.TokenStore) {
	t.Helper()
	firstAccess, err := session.Token(ctx)
	if err != nil {
		t.Fatal(err)
	}
	session.Invalidate()
	refreshed, err := session.Token(ctx)
	if err != nil || refreshed.Value == firstAccess.Value || !refreshed.Expiry.After(time.Now()) {
		t.Fatalf("forced refresh: err %v, new token %v", err, refreshed.Value != firstAccess.Value)
	}
	if again, err := store.Load(ctx, llmprovider.ProviderOpenAI); err != nil || again == nil || again.Scope == "" ||
		again.HostID != session.HostID {
		t.Fatalf("the refreshed session was not saved with its plan fields: %v", err)
	}
	t.Log("FINDING refresh: the token rotated and the store holds it")
}

// planRevoke revokes the session, and checks that a refresh afterwards is
// refused.
func planRevoke(ctx context.Context, t *testing.T, session *auth.OAuthSession) {
	t.Helper()
	if err := auth.RevokeOAuthSession(ctx, session); err != nil {
		t.Fatalf("RevokeOAuthSession: %v", err)
	}
	session.Invalidate()
	if _, err := session.Token(ctx); err == nil {
		t.Fatal("a refresh after revocation succeeded")
	} else if !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Logf("FINDING refresh after revocation: refused, but not as ErrAuthFailure: %v", err)
	} else {
		t.Log("FINDING refresh after revocation: refused, ErrAuthFailure")
	}
}

// probeIdentityName is the name 0076's probe signs in under, which is no
// real product's (0076-PLAN Phase 4).
const probeIdentityName = "llmprovider-probe"

// checkAuthorizeIdentity reports whether the authorize URL raw carries name
// as its one param, OpenAI's originator or Grok's referrer (0076-MADR D3),
// and fails the test when it does not. It fails with t.Errorf only, so a
// sign-in's OpenURL, which runs on a goroutine of its own, may call it.
func checkAuthorizeIdentity(t *testing.T, raw, param, name string) bool {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Errorf("authorize URL: %v", err)
		return false
	}
	if got := u.Query()[param]; len(got) != 1 || got[0] != name {
		t.Errorf("authorize URL %s = %q, want [%q]", param, got, name)
		return false
	}
	return true
}

// identityRecorder is a sign-in's transport that keeps the referrer values
// of each form it posts with one, which is Grok's device form, and nothing
// else: no token, code or other part of any request is kept or logged.
type identityRecorder struct {
	base http.RoundTripper

	mu        sync.Mutex
	referrers [][]string
}

func (r *identityRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body == nil || req.Body == http.NoBody ||
		!strings.HasPrefix(req.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		return r.base.RoundTrip(req)
	}
	raw, err := io.ReadAll(req.Body)
	if closeErr := req.Body.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	if form, err := url.ParseQuery(string(raw)); err == nil && form.Has("referrer") {
		r.mu.Lock()
		r.referrers = append(r.referrers, form["referrer"])
		r.mu.Unlock()
	}
	sent := req.Clone(req.Context())
	sent.Body = io.NopCloser(bytes.NewReader(raw))
	sent.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(raw)), nil }
	return r.base.RoundTrip(sent)
}

// checkDeviceReferrer checks that the sign-in posted one form with a
// referrer, Grok's device form, and that its referrer is name alone
// (0076-MADR D3).
func checkDeviceReferrer(t *testing.T, rec *identityRecorder, name string) {
	t.Helper()
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.referrers) != 1 || len(rec.referrers[0]) != 1 || rec.referrers[0][0] != name {
		t.Errorf("device form referrers = %q, want one form with [%q]", rec.referrers, name)
	}
}

// checkIdentitySession binds session to a scratch FileTokenStore in
// t.TempDir, so that a refresh saves there and never to a real store, and
// then uses it with no generation: its provider's raw model listing
// (catalog.ListRaw: no probe, no listing cache, no models.dev, no fallback),
// which must hold a model. It logs the count only. opts reach the listing.
func checkIdentitySession(ctx context.Context, t *testing.T, session *auth.OAuthSession, opts ...llmprovider.Option) {
	t.Helper()
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, session.Provider, session); err != nil {
		t.Fatalf("Save: %v", err)
	}
	session.Store = store
	raw, err := catalog.ListRaw(ctx, session.Provider, session, opts...)
	if err != nil || len(raw.Models) == 0 {
		t.Fatalf("FINDING use: %s listed %d models with the session, %v; want one or more", session.Provider,
			len(raw.Models), err)
	}
	t.Logf("FINDING use: %s listed %d models with the session", session.Provider, len(raw.Models))
}

// checkIdentityRevoked revokes session at its issuer, and checks that a
// refresh afterwards is refused, as planRevoke does.
func checkIdentityRevoked(ctx context.Context, t *testing.T, session *auth.OAuthSession) {
	t.Helper()
	if err := auth.RevokeOAuthSession(ctx, session); err != nil {
		t.Fatalf("FINDING revocation: RevokeOAuthSession: %v", err)
	}
	session.Invalidate()
	if _, err := session.Token(ctx); err == nil {
		t.Fatal("FINDING revocation: a refresh after revocation succeeded")
	} else {
		t.Logf("FINDING revocation: revoked; a refresh afterwards is refused, as ErrAuthFailure %t",
			errors.Is(err, llmprovider.ErrAuthFailure))
	}
}

// pasteFileLimit is the most pasteFromFile reads from its file.
const pasteFileLimit = 8 << 10

// pasteInterval is how often pasteFromFile looks at its file. A twin sets it
// shorter.
var pasteInterval = time.Second

// pasteRecord is what TestLive_0074GrokPaste knows of the paste: whether
// InputCode returned a value, and the value's shape (pasteShape). It never
// holds the value.
type pasteRecord struct {
	mu       sync.Mutex
	returned bool
	shape    string
}

func (r *pasteRecord) set(shape string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.returned, r.shape = true, shape
}

func (r *pasteRecord) get() (returned bool, shape string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.returned, r.shape
}

// pasteFromFile is an OAuthFlowOptions.InputCode that reads a person's paste
// from the file at path, because go test gives the test binary no standard
// input (0074-PLAN Phase 7 step 4). It looks every pasteInterval until the
// file holds a newline-terminated line, reads at most pasteFileLimit bytes,
// removes the file, records the line's shape in rec and returns the line. It
// returns ctx's error when ctx ends first. The line is a secret: it is
// never logged, and no error holds it.
func pasteFromFile(path string, rec *pasteRecord) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		ticker := time.NewTicker(pasteInterval)
		defer ticker.Stop()
		for {
			data, err := readPasteFile(path)
			if err != nil {
				return "", err
			}
			line, _, found := strings.Cut(string(data), "\n")
			if !found {
				if len(data) >= pasteFileLimit {
					if err := os.Remove(path); err != nil {
						return "", fmt.Errorf("paste file: %w", err)
					}
					return "", fmt.Errorf("paste file: no newline within %d bytes", pasteFileLimit)
				}
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-ticker.C:
				}
				continue
			}
			if err := os.Remove(path); err != nil {
				return "", fmt.Errorf("paste file: %w", err)
			}
			line = strings.TrimSuffix(line, "\r")
			rec.set(pasteShape(line))
			return line, nil
		}
	}
}

// readPasteFile reads at most pasteFileLimit bytes of path; a file that is
// not there yet reads as empty.
func readPasteFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("paste file: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, pasteFileLimit))
	if err != nil {
		return nil, fmt.Errorf("paste file: %w", err)
	}
	return data, nil
}

// pasteShape says what a paste is, as the SDK's own parser reads it
// (auth's parseOAuthCallback): a URL, when it parses with a scheme, and
// otherwise a bare code. A URL's scheme (http or https), whether its host is
// a loopback address, and whether its path is /callback are named, and
// nothing else of it: the result never holds any part of raw.
func pasteShape(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "an empty line"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return "a bare code"
	}
	query := u.Query()
	hasCode, hasState := query.Get("code") != "", query.Get("state") != ""
	var kind string
	switch {
	case query.Get("error") != "":
		kind = "a URL with an error"
	case hasCode && hasState:
		kind = "a callback URL with code and state"
	case hasCode:
		kind = "a URL with a code and no state"
	case hasState:
		kind = "a URL with a state and no code"
	default:
		kind = "a URL with no code and no state"
	}
	scheme := "another scheme"
	if u.Scheme == "http" || u.Scheme == "https" {
		scheme = u.Scheme
	}
	host := "another host"
	if ip := net.ParseIP(u.Hostname()); (ip != nil && ip.IsLoopback()) || u.Hostname() == "localhost" {
		host = "a loopback host"
	}
	path := "another path"
	if u.Path == "/callback" {
		path = "path /callback"
	}
	return fmt.Sprintf("%s (scheme %s, %s, %s)", kind, scheme, host, path)
}

// pasteFinding is the one FINDING sign-in line for a Grok browser sign-in
// run with pasteFromFile, from the record and LoginBrowserOAuth's error, and
// whether the run passed: the sign-in completed by the paste. The line holds
// the shape and the error, never the paste.
func pasteFinding(rec *pasteRecord, err error) (line string, passed bool) {
	returned, shape := rec.get()
	switch {
	case err == nil && returned:
		return "FINDING sign-in: completed by a paste of " + shape, true
	case err == nil:
		return "FINDING sign-in: completed by the loopback, not a paste; the browser reached the host, " +
			"so this run decides nothing", false
	case returned && pasteRefusedLocally(err):
		return fmt.Sprintf("FINDING sign-in: the paste was refused before any request: %s: %v", shape, err), false
	case returned:
		return fmt.Sprintf("FINDING sign-in: xAI refused the pasted %s: %v", shape, err), false
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		return "FINDING sign-in: nothing was pasted before the deadline", false
	default:
		return fmt.Sprintf("FINDING sign-in: the sign-in ended before any paste: %v", err), false
	}
}

// pasteRefusedLocally reports whether err is the SDK refusing a paste before
// it sent a request: a state mismatch, a missing or empty code, or a pasted
// denial.
func pasteRefusedLocally(err error) bool {
	msg := err.Error()
	return errors.Is(err, auth.ErrSignInDenied) || strings.Contains(msg, "callback state mismatch") ||
		strings.Contains(msg, "missing authorization code") || strings.Contains(msg, "empty authorization code")
}
