package llmprovider_test

// The checks the person-gated live tests run after their sign-in, untagged so
// that an offline twin (live_twins_test.go) runs each against a fake and a
// planted break of a check is caught with no sign-in (0046-MADR D3). The live
// tests call them with the real session; the twins with a fake's.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
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
// removes the file (removePasteFile, which waits out a writer that still holds
// it open), records the line's shape in rec and returns the line. It
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
					if err := removePasteFile(ctx, path); err != nil {
						return "", err
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
			if err := removePasteFile(ctx, path); err != nil {
				return "", err
			}
			line = strings.TrimSuffix(line, "\r")
			rec.set(pasteShape(line))
			return line, nil
		}
	}
}

// errSharingViolation is Windows' ERROR_SHARING_VIOLATION (32), "The process
// cannot access the file because it is being used by another process": what
// DeleteFile reports while another handle to the file is open without
// FILE_SHARE_DELETE, as a writer's is between its last write and its close.
// syscall.Errno is on every platform; the constant is not, so the number is
// named here and held to Windows by fileHeldOpen.
const errSharingViolation syscall.Errno = 32

// fileHeldOpen reports whether err is a removal refused because another
// handle holds the file open. Only Windows refuses so; elsewhere an open file
// can be removed, and no error is this one.
func fileHeldOpen(err error) bool {
	var errno syscall.Errno
	return runtime.GOOS == "windows" && errors.As(err, &errno) && errno == errSharingViolation
}

// removePasteFile removes path. While the file is held open by its writer,
// which on Windows is exactly when the writer has written the newline and not
// yet closed, it tries again every pasteInterval until the removal succeeds or
// ctx ends, and returns ctx's error then. Any other error is returned at
// once. The error holds the path, never the paste.
func removePasteFile(ctx context.Context, path string) error {
	ticker := time.NewTicker(pasteInterval)
	defer ticker.Stop()
	for {
		err := os.Remove(path)
		if err == nil {
			return nil
		}
		if !fileHeldOpen(err) {
			return fmt.Errorf("paste file: %w", err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("paste file: still held open: %w", ctx.Err())
		case <-ticker.C:
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

// checkDeviceChatGPT checks an OpenAI device login's session (0074-PLAN
// Phase 7 step 1): a ChatGPT session holding an access and a refresh token,
// then one short generation through the ChatGPT backend on model, then its
// revocation, after which a refresh is refused. A session that a failed check
// leaves behind is revoked when the test ends. It logs no token.
func checkDeviceChatGPT(ctx context.Context, t *testing.T, session *auth.OAuthSession, model string,
	skip transientSkip, opts ...llmprovider.Option) {
	t.Helper()
	if !session.ChatGPT() || session.Access == "" || session.Refresh == "" {
		t.Fatalf("FINDING sign-in: ChatGPT session %t, access set %t, refresh set %t; want a ChatGPT session with both tokens",
			session.ChatGPT(), session.Access != "", session.Refresh != "")
	}
	t.Log("FINDING sign-in: OpenAI device login approved; a ChatGPT session with an access and a refresh token")
	revoked := false
	t.Cleanup(func() {
		if revoked {
			return
		}
		if err := auth.RevokeOAuthSession(context.Background(), session); err != nil {
			t.Errorf("RevokeOAuthSession when the test ends: %v", err)
		}
	})
	checkWord(ctx, t, liveOpenAI(t, session, model, opts...), "ALPHA", skip)
	t.Log("FINDING use: one short generation through the ChatGPT backend returned the requested word")
	checkIdentityRevoked(ctx, t, session)
	revoked = true
}

// stepRunner is the part of x/signin's SignIn that driveSteps uses, so that a
// twin can script it.
type stepRunner interface {
	Steps() <-chan authcontract.Step
	Answer(context.Context, authcontract.Answer) error
}

// stepHooks is what a person does at each step that asks or shows. A nil
// paste makes a paste step an error: the sign-in was not meant to ask one.
type stepHooks struct {
	confirm func(authcontract.Step) bool
	openURL func(authcontract.Step)
	paste   func(context.Context) (string, error)
}

// driveSteps answers a sign-in's steps until its channel closes, and returns
// the kinds of the steps it saw, in order. It answers a confirm step with
// h.confirm, shows an open_url step through h.openURL, and answers a paste
// step with the line h.paste returns. It returns an error when h.paste does,
// or when ctx ends; the sign-in's own outcome is its Wait's.
func driveSteps(ctx context.Context, run stepRunner, h stepHooks) ([]authcontract.StepKind, error) {
	var kinds []authcontract.StepKind
	for {
		var step authcontract.Step
		var open bool
		select {
		case step, open = <-run.Steps():
		case <-ctx.Done():
			return kinds, ctx.Err()
		}
		if !open {
			return kinds, nil
		}
		kinds = append(kinds, step.Kind)
		var answer *authcontract.Answer
		switch step.Kind {
		case authcontract.StepConfirm:
			yes := h.confirm != nil && h.confirm(step)
			answer = &authcontract.Answer{Seq: step.Seq, Confirm: &yes}
		case authcontract.StepOpenURL:
			if h.openURL != nil {
				h.openURL(step)
			}
		case authcontract.StepPaste:
			if h.paste == nil {
				return kinds, fmt.Errorf("the sign-in asked for a paste (step %d), which this run does not take", step.Seq)
			}
			line, err := h.paste(ctx)
			if err != nil {
				return kinds, err
			}
			answer = &authcontract.Answer{Seq: step.Seq, Paste: line}
		}
		if answer != nil {
			if err := run.Answer(ctx, *answer); err != nil {
				return kinds, fmt.Errorf("answer to step %d: %w", step.Seq, err)
			}
		}
	}
}

// stepKinds is kinds as text, in order.
func stepKinds(kinds []authcontract.StepKind) string {
	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = string(k)
	}
	return strings.Join(names, ", ")
}

// signInFinding is the one FINDING sign-in line of a sign-in that only a
// paste completes (no listener bound), from the paste record, the steps seen
// and the sign-in's error, and whether the run passed: it completed by a
// paste. The line holds the paste's shape and the error, never the paste.
func signInFinding(vendor string, rec *pasteRecord, kinds []authcontract.StepKind, err error) (line string, passed bool) {
	returned, shape := rec.get()
	steps := stepKinds(kinds)
	switch {
	case err == nil && returned:
		return fmt.Sprintf("FINDING sign-in: %s sign-in completed by a paste of %s; steps: %s", vendor, shape, steps), true
	case err == nil:
		return fmt.Sprintf("FINDING sign-in: %s sign-in completed with no paste, which a sign-in with no listener cannot do; steps: %s",
			vendor, steps), false
	case returned:
		return fmt.Sprintf("FINDING sign-in: %s did not complete by the pasted %s: %v; steps: %s", vendor, shape, err, steps), false
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		return fmt.Sprintf("FINDING sign-in: nothing was pasted before the deadline; steps: %s", steps), false
	default:
		return fmt.Sprintf("FINDING sign-in: the sign-in ended before any paste: %v; steps: %s", err, steps), false
	}
}

// loopbackFinding is the one FINDING sign-in line of a sign-in that the
// loopback on port completes, and whether it passed.
func loopbackFinding(vendor string, port int, kinds []authcontract.StepKind, err error) (line string, passed bool) {
	if err != nil {
		return fmt.Sprintf("FINDING sign-in: the %s sign-in by the loopback on port %d failed: %v; steps: %s",
			vendor, port, err, stepKinds(kinds)), false
	}
	return fmt.Sprintf("FINDING sign-in: the %s sign-in completed by the loopback, which redirected to port %d; steps: %s",
		vendor, port, stepKinds(kinds)), true
}

// redirectPort is the port of the authorize URL raw's redirect_uri.
func redirectPort(raw string) (int, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return 0, fmt.Errorf("authorize URL: %w", err)
	}
	redirect, err := url.Parse(u.Query().Get("redirect_uri"))
	if err != nil {
		return 0, fmt.Errorf("authorize URL: redirect_uri: %w", err)
	}
	if redirect.Port() == "" {
		return 0, errors.New("authorize URL: no redirect_uri with a port")
	}
	return strconv.Atoi(redirect.Port())
}

// holdLoopbackPort takes port on the loopback addresses until the test ends,
// so that a sign-in finds it busy, and fails the test when it cannot.
func holdLoopbackPort(t *testing.T, port int) {
	t.Helper()
	text := strconv.Itoa(port)
	v4, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", text))
	if err != nil {
		t.Fatalf("hold port %d: %v; free it and run again", port, err)
	}
	held := []net.Listener{}
	held = append(held, v4)
	if v6, err := net.Listen("tcp6", net.JoinHostPort("::1", text)); err == nil {
		held = append(held, v6)
	}
	t.Cleanup(func() {
		for _, l := range held {
			_ = l.Close()
		}
	})
}

// checkRefreshOnce binds session to a scratch FileTokenStore in t.TempDir,
// forces one refresh, and checks that the token rotated and that the store
// holds the rotation.
func checkRefreshOnce(ctx context.Context, t *testing.T, session *auth.OAuthSession) {
	t.Helper()
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, session.Provider, session); err != nil {
		t.Fatalf("Save: %v", err)
	}
	session.Store = store
	first, err := session.Token(ctx)
	if err != nil {
		t.Fatalf("FINDING refresh: Token: %v", err)
	}
	session.Invalidate()
	next, err := session.Token(ctx)
	if err != nil || next.Value == first.Value || !next.Expiry.After(time.Now()) {
		t.Fatalf("FINDING refresh: forced refresh: err %v, token rotated %t", err, err == nil && next.Value != first.Value)
	}
	if held, err := store.Load(ctx, session.Provider); err != nil || held == nil || held.Access != next.Value {
		t.Fatalf("FINDING refresh: the store does not hold the rotated token: %v", err)
	}
	t.Log("FINDING refresh: the token rotated once and the scratch store holds it")
}

// sameClient reports whether two plan sessions are under one issued client,
// which a sign-in again reuses (0075-MADR D7), and under one host id.
func sameClient(a, b *auth.OAuthSession) error {
	if a.ClientID == "" || a.ClientID != b.ClientID {
		return fmt.Errorf("the second sign-in's client is not the first's (first %d chars, second %d chars)",
			len(a.ClientID), len(b.ClientID))
	}
	if a.HostID == "" || a.HostID != b.HostID {
		return errors.New("the second sign-in's host id is not the first's")
	}
	return nil
}

// planClientStore is the store a plan sign-in keeps its session and its
// issued client in: FileTokenStore.
type planClientStore interface {
	auth.TokenStore
	auth.ClientStore
}

// reuseAuthorizeError checks the authorize URL of a plan sign-in that reuses
// the client record rec with no plan session (0075-MADR D7): it names the
// kept client and the kept host id, and sends no agent_name_hint and no
// id_token_hint. It is nil when it does.
func reuseAuthorizeError(raw string, rec auth.ClientRecord) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("authorize URL: %w", err)
	}
	query := u.Query()
	if got := query["client_id"]; len(got) != 1 || got[0] != rec.ClientID {
		return fmt.Errorf("authorize URL has %d client_id value(s), the kept client among them %t; want exactly the kept client",
			len(got), slices.Contains(got, rec.ClientID))
	}
	for _, name := range []string{"agent_name_hint", "id_token_hint"} {
		if query.Has(name) {
			return fmt.Errorf("authorize URL carries %s; a sign-in that reuses a client with no session sends none", name)
		}
	}
	if got := query["ext_agent_host_id"]; len(got) != 1 || got[0] != rec.HostID {
		return fmt.Errorf("authorize URL has %d ext_agent_host_id value(s), the kept host id among them %t; want exactly the kept one",
			len(got), slices.Contains(got, rec.HostID))
	}
	return nil
}

// reusedClientError reports whether the plan session is under the client
// and the host id the record holds, which a reuse keeps unchanged. It names
// lengths and no id.
func reusedClientError(s *auth.OAuthSession, rec auth.ClientRecord) error {
	switch {
	case rec.ClientID == "" || rec.HostID == "":
		return errors.New("the client record holds no client or no host id")
	case s.ClientID != rec.ClientID:
		return fmt.Errorf("the session's client is not the kept one (kept %d chars, session %d chars)",
			len(rec.ClientID), len(s.ClientID))
	case s.HostID != rec.HostID:
		return errors.New("the session's host id is not the kept one")
	}
	return nil
}

// logoutKeepsClientError saves session to store and deletes it, as
// wizard.Logout does, and reports whether the client record is the same
// afterwards (0075-MADR D7): a sign-out does not register another client.
func logoutKeepsClientError(ctx context.Context, store planClientStore, session *auth.OAuthSession) error {
	before, err := store.LoadClient(ctx, llmprovider.ProviderOpenAI)
	if err != nil {
		return fmt.Errorf("LoadClient before: %w", err)
	}
	if before.ClientID == "" {
		return errors.New("the store holds no client record before the sign-out")
	}
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, session); err != nil {
		return fmt.Errorf("Save: %w", err)
	}
	if err := store.Delete(ctx, llmprovider.ProviderOpenAI); err != nil {
		return fmt.Errorf("Delete: %w", err)
	}
	after, err := store.LoadClient(ctx, llmprovider.ProviderOpenAI)
	if err != nil {
		return fmt.Errorf("LoadClient after: %w", err)
	}
	if after != before {
		return errors.New("the client record changed when the session was deleted")
	}
	return nil
}

// countingTransport counts the requests that reach base.
type countingTransport struct {
	base http.RoundTripper
	n    *atomic.Int64
}

func (c countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.n.Add(1)
	return c.base.RoundTrip(r)
}

// checkPlanConsent is the consent end to end (0075-MADR D5; 0075-PLAN Phase
// 6 step 2) on a plan session: bound to store, it declines the background-use
// consent through SetConsent, and a request marked llmprovider.WithBackground
// must be refused (ErrNotPermitted) with no request sent; it grants the
// consent, and one short marked generation must go through and send one.
// client is the transport the requests are counted on; opts reach the
// provider.
func checkPlanConsent(ctx context.Context, t *testing.T, session *auth.OAuthSession, store auth.TokenStore, model string,
	client *http.Client, opts ...llmprovider.Option) {
	t.Helper()
	provider := llmprovider.ProviderOpenAI
	if err := store.Save(ctx, provider, session); err != nil {
		t.Fatalf("Save: %v", err)
	}
	session.UseStore(store)
	var sent atomic.Int64
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	counted := &http.Client{Transport: countingTransport{base: base, n: &sent}, Timeout: client.Timeout}
	p := liveOpenAI(t, session, model, append(opts, llmprovider.WithHTTPClient(counted))...)
	marked := llmprovider.WithBackground(ctx)

	granted := false
	if err := auth.SetConsent(ctx, store, provider, openai.ConsentBackgroundUse, granted); err != nil {
		t.Fatalf("SetConsent (decline): %v", err)
	}
	_, err := llmprovider.GenerateText(marked, p, userText("Reply with only the word ALPHA"))
	if !errors.Is(err, llmprovider.ErrNotPermitted) || sent.Load() != 0 {
		t.Fatalf("FINDING consent: declined, a marked request gave %v and sent %d request(s); want ErrNotPermitted and none",
			err, sent.Load())
	}
	t.Log("FINDING consent: declined, a marked request is refused with no request sent")

	granted = true
	if err := auth.SetConsent(ctx, store, provider, openai.ConsentBackgroundUse, granted); err != nil {
		t.Fatalf("SetConsent (grant): %v", err)
	}
	checkWord(marked, t, p, "ALPHA", noSkip)
	if sent.Load() == 0 {
		t.Fatal("FINDING consent: granted, a marked generation sent no request")
	}
	t.Log("FINDING consent: granted, a marked generation went through")
}

// nameRecord is what nameRecorder keeps of one request: the value of its
// originator header, when it sent one, and the first product name of its
// User-Agent. It keeps no URL, token, code or body.
type nameRecord struct {
	originator    string
	hasOriginator bool
	product       string
}

// nameRecorder is a transport that records each request's nameRecord, and
// sends it on (0079-PLAN Phase 5 step 2).
type nameRecorder struct {
	mu   sync.Mutex
	recs []nameRecord
}

func (r *nameRecorder) wrap(base http.RoundTripper) http.RoundTripper {
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		rec := nameRecord{product: firstProduct(req.Header.Get("User-Agent"))}
		if values := req.Header.Values("Originator"); len(values) > 0 {
			rec.originator, rec.hasOriginator = values[0], true
		}
		r.mu.Lock()
		r.recs = append(r.recs, rec)
		r.mu.Unlock()
		return base.RoundTrip(req)
	})
}

// mark is how many requests are recorded now; since(mark) is the rest.
func (r *nameRecorder) mark() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.recs)
}

func (r *nameRecorder) since(mark int) []nameRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]nameRecord(nil), r.recs[mark:]...)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// firstProduct is the product name a User-Agent leads with: its text up to
// the first "/" or space.
func firstProduct(ua string) string {
	if i := strings.IndexAny(ua, "/ "); i >= 0 {
		return ua[:i]
	}
	return ua
}

// namedRequestsError checks the recorded requests: there are at least min,
// each led its User-Agent with name, and each that sent an originator sent
// name; with wantOriginator at least one sent one.
func namedRequestsError(recs []nameRecord, name string, min int, wantOriginator bool) error {
	if len(recs) < min {
		return fmt.Errorf("%d request(s) recorded, want at least %d", len(recs), min)
	}
	sawOriginator := false
	for i, r := range recs {
		if r.product != name {
			return fmt.Errorf("request %d led its User-Agent with %q, want %q", i+1, r.product, name)
		}
		if r.hasOriginator {
			sawOriginator = true
			if r.originator != name {
				return fmt.Errorf("request %d sent the originator %q, want %q", i+1, r.originator, name)
			}
		}
	}
	if wantOriginator && !sawOriginator {
		return errors.New("no request sent an originator")
	}
	return nil
}

// nameProbe is what one half of TestLive_0079Name uses after its sign-in.
type nameProbe struct {
	name   string
	rec    *nameRecorder
	client *http.Client // records every request, the sign-in's included
	opts   []llmprovider.Option
	store  *auth.FileTokenStore
}

func (n nameProbe) named() []llmprovider.Option {
	return append(append([]llmprovider.Option(nil), n.opts...), llmprovider.WithClientInfo(n.name, ""), llmprovider.WithHTTPClient(n.client))
}

// control is named() without the name: the default originator and
// User-Agent.
func (n nameProbe) control() []llmprovider.Option {
	return append(append([]llmprovider.Option(nil), n.opts...), llmprovider.WithHTTPClient(n.client))
}

// listAndGenerate lists the ChatGPT backend's models, which must hold one,
// and generates one short word with the first (0079-PLAN Phase 5).
func listAndGenerate(ctx context.Context, session *auth.OAuthSession, opts []llmprovider.Option) error {
	lister, err := openai.New(append([]llmprovider.Option{llmprovider.WithTokenSource(session)}, opts...)...)
	if err != nil {
		return fmt.Errorf("openai.New: %w", err)
	}
	listing, ok := lister.(llmprovider.ModelLister)
	if !ok {
		return errors.New("the openai provider lists no models")
	}
	models, err := listing.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("ListModels: %w", err)
	}
	if len(models) == 0 {
		return errors.New("ListModels returned no model")
	}
	p, err := openai.New(append([]llmprovider.Option{llmprovider.WithTokenSource(session), llmprovider.WithModel(models[0])}, opts...)...)
	if err != nil {
		return fmt.Errorf("openai.New: %w", err)
	}
	out, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word ALPHA"))
	if err != nil {
		return fmt.Errorf("Generate: %w", err)
	}
	if !strings.Contains(strings.ToUpper(out), "ALPHA") {
		return fmt.Errorf("Generate = %q", out)
	}
	return nil
}

// listRawModels is catalog.ListRaw for Grok, which must hold a model.
func listRawModels(ctx context.Context, session *auth.OAuthSession, opts []llmprovider.Option) error {
	raw, err := catalog.ListRaw(ctx, llmprovider.ProviderGrok, session, opts...)
	if err != nil {
		return fmt.Errorf("ListRaw: %w", err)
	}
	if len(raw.Models) == 0 {
		return errors.New("ListRaw returned no model")
	}
	return nil
}

// nameVerdict is the outcome of a call that is run under the name and, when
// it fails, once without it.
type nameVerdict string

const (
	namePassed  nameVerdict = "passed"  // under the name
	nameRefused nameVerdict = "refused" // failed under the name, passed without
	nameFailed  nameVerdict = "failed"  // failed under both: not this record's
)

// probeWithControl runs run under named, and, when that fails, once under
// control (0079-PLAN Phase 5 step 1, "a control for a refusal"). It returns
// the verdict and the error under the name.
func probeWithControl(run func([]llmprovider.Option) error, named, control []llmprovider.Option) (nameVerdict, error) {
	err := run(named)
	if err == nil {
		return namePassed, nil
	}
	if run(control) == nil {
		return nameRefused, err
	}
	return nameFailed, err
}

// unnamedCopy is a copy of a session's stored fields with no identity and no
// client of its own, so it is revoked or refreshed under this module's name.
func unnamedCopy(s *auth.OAuthSession) *auth.OAuthSession {
	return &auth.OAuthSession{Provider: s.Provider, Access: s.Access, Refresh: s.Refresh, Expiry: s.Expiry,
		Issuer: s.Issuer, ClientID: s.ClientID, AccountID: s.AccountID, FedRAMP: s.FedRAMP, TokenURL: s.TokenURL,
		Resource: s.Resource, Scope: s.Scope, IDToken: s.IDToken, HostID: s.HostID, Kind: s.Kind, Consents: s.Consents,
		HTTPClient: s.HTTPClient}
}

// reloadedSession saves signed to the probe's scratch store and loads it
// back: a session with no identity and no client of its own, so that what
// names it afterwards is the hand-off of the provider or listing it is given
// to (0079-MADR D4), and what carries it to the issuer is the client that
// hand-off gives. It is revoked when the test ends unless the returned
// function was called first.
func reloadedSession(ctx context.Context, t *testing.T, n nameProbe, signed *auth.OAuthSession) (*auth.OAuthSession, func()) {
	t.Helper()
	if err := n.store.Save(ctx, signed.Provider, signed); err != nil {
		t.Fatalf("Save: %v", err)
	}
	session, err := n.store.Load(ctx, signed.Provider)
	if err != nil || session == nil {
		t.Fatalf("Load = %v, %v", session, err)
	}
	revoked := false
	t.Cleanup(func() {
		if revoked {
			return
		}
		if err := auth.RevokeOAuthSession(context.Background(), session); err != nil {
			t.Logf("FINDING refused: revocation under the name when the test ends: %v", err)
			if err := auth.RevokeOAuthSession(context.Background(), unnamedCopy(session)); err != nil {
				t.Errorf("RevokeOAuthSession of an unnamed copy when the test ends: %v", err)
			}
		}
	})
	return session, func() { revoked = true }
}

// checkNamedRefresh forces one refresh, which must succeed and rotate the
// token, and checks that every request it sent led its User-Agent with the
// name. A refresh that fails under the name is tried once on an unnamed copy,
// and the verdict logged; the copy is revoked, so nothing stays alive.
func checkNamedRefresh(ctx context.Context, t *testing.T, n nameProbe, session *auth.OAuthSession) {
	t.Helper()
	mark := n.rec.mark()
	if err := forceRefresh(ctx, session); err != nil {
		cp := unnamedCopy(session)
		if cerr := forceRefresh(ctx, cp); cerr != nil {
			t.Fatalf("FINDING refresh: failed under the name (%v) and without it (%v); not this record's", err, cerr)
		}
		_ = auth.RevokeOAuthSession(ctx, cp)
		t.Fatalf("FINDING refused: a refresh under the name failed (%v) and passed without it", err)
	}
	if err := namedRequestsError(n.rec.since(mark), n.name, 1, false); err != nil {
		t.Fatalf("FINDING refresh: %v", err)
	}
	t.Logf("FINDING refresh: a forced refresh succeeded and led its User-Agent with %q", n.name)
}

// forceRefresh invalidates session's token and asks for another, which must
// differ.
func forceRefresh(ctx context.Context, session *auth.OAuthSession) error {
	first, err := session.Token(ctx)
	if err != nil {
		return fmt.Errorf("Token: %w", err)
	}
	session.Invalidate()
	next, err := session.Token(ctx)
	if err != nil {
		return fmt.Errorf("refresh: %w", err)
	}
	if next.Value == first.Value {
		return errors.New("the token did not rotate")
	}
	return nil
}

// checkNamedRevocation revokes session under the name, checks that every
// request of it, the discovery included (min requests in all), led its
// User-Agent with the name, and that a refresh afterwards is refused. A
// revocation the issuer refuses is logged, and an unnamed copy is revoked
// instead, so nothing stays alive; the test then fails.
func checkNamedRevocation(ctx context.Context, t *testing.T, n nameProbe, session *auth.OAuthSession, min int) {
	t.Helper()
	mark := n.rec.mark()
	if err := auth.RevokeOAuthSession(ctx, session); err != nil {
		if cerr := auth.RevokeOAuthSession(ctx, unnamedCopy(session)); cerr != nil {
			t.Fatalf("FINDING revocation: refused under the name (%v) and without it (%v)", err, cerr)
		}
		t.Fatalf("FINDING refused: revocation under the name failed (%v) and passed without it", err)
	}
	if err := namedRequestsError(n.rec.since(mark), n.name, min, false); err != nil {
		t.Fatalf("FINDING revocation: %v", err)
	}
	session.Invalidate()
	_, err := session.Token(ctx)
	if err == nil {
		t.Fatal("FINDING revocation: a refresh after revocation succeeded")
	}
	t.Logf("FINDING revocation: revoked under the name, led with it; a refresh afterwards is refused, as ErrAuthFailure %t",
		errors.Is(err, llmprovider.ErrAuthFailure))
}

// runChatGPTName is the ChatGPT half of TestLive_0079Name after its sign-in:
// the sign-in's session is saved and loaded back, so it has no identity and
// no client, and the openai providers built with the name give it both. It
// lists the models and generates once, under the name and, on a failure, once
// without it; every request that reached the backend carried the name as its
// originator and led its User-Agent with it; a forced refresh; the
// revocation.
func runChatGPTName(ctx context.Context, t *testing.T, n nameProbe, signed *auth.OAuthSession) {
	t.Helper()
	session, revoked := reloadedSession(ctx, t, n, signed)
	mark := n.rec.mark()
	verdict, err := probeWithControl(func(opts []llmprovider.Option) error { return listAndGenerate(ctx, session, opts) },
		n.named(), n.control())
	switch verdict {
	case nameRefused:
		t.Fatalf("FINDING refused: the listing or the generation failed under the name (%v) and passed without it", err)
	case nameFailed:
		t.Fatalf("FINDING use: the listing or the generation failed under the name (%v) and without it; not this record's", err)
	}
	if err := namedRequestsError(n.rec.since(mark), n.name, 2, true); err != nil {
		t.Fatalf("FINDING use: %v", err)
	}
	t.Logf("FINDING use: the listing and one generation passed under %q; the backend's requests sent it as their originator", n.name)
	checkNamedRefresh(ctx, t, n, session)
	checkNamedRevocation(ctx, t, n, session, 1)
	revoked()
}

// runGrokName is the Grok half after its sign-in: the session is saved and
// loaded back, so catalog.ListRaw's hand-off gives it the name and the
// client; the raw listing must hold a model, under the name and, on a
// failure, once without it; a forced refresh; the revocation, with its
// discovery.
func runGrokName(ctx context.Context, t *testing.T, n nameProbe, signed *auth.OAuthSession) {
	t.Helper()
	session, revoked := reloadedSession(ctx, t, n, signed)
	verdict, err := probeWithControl(func(opts []llmprovider.Option) error { return listRawModels(ctx, session, opts) },
		n.named(), n.control())
	switch verdict {
	case nameRefused:
		t.Fatalf("FINDING refused: the listing failed under the name (%v) and passed without it", err)
	case nameFailed:
		t.Fatalf("FINDING use: the listing failed under the name (%v) and without it; not this record's", err)
	}
	t.Logf("FINDING use: the raw listing held a model under %q", n.name)
	checkNamedRefresh(ctx, t, n, session)
	checkNamedRevocation(ctx, t, n, session, 2)
	revoked()
}

// authorizeWant says what an authorize URL carries: each parameter in equals
// once with that value, each in present once with some value, none of absent
// at all. It is what a probe asserts of the URL it shows before anyone signs
// in, written from the code that builds the URL and run offline against it
// (0074-PLAN Phase 7, deviation 2026-11-14).
type authorizeWant struct {
	equals  map[string]string
	present []string
	absent  []string
}

// authorizeError is nil when the authorize URL raw meets want. It names the
// parameter that does not and never a value.
func authorizeError(raw string, want authorizeWant) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	query := u.Query()
	for _, name := range slices.Sorted(maps.Keys(want.equals)) {
		if got := query[name]; len(got) != 1 || got[0] != want.equals[name] {
			return fmt.Errorf("parameter %s has %d value(s), the expected one among them %t; want exactly the expected value",
				name, len(got), slices.Contains(got, want.equals[name]))
		}
	}
	for _, name := range want.present {
		if got := query[name]; len(got) != 1 || got[0] == "" {
			return fmt.Errorf("parameter %s has %d value(s); want exactly one, not empty", name, len(got))
		}
	}
	for _, name := range want.absent {
		if query.Has(name) {
			return fmt.Errorf("parameter %s is sent; this method's URL sends none", name)
		}
	}
	return nil
}

// browserAuthorizeWant is a browser_oauth URL: it names this module, or the
// application an Identity names, in param (OpenAI's originator, Grok's
// referrer; 0076-MADR D3).
func browserAuthorizeWant(param, name string) authorizeWant {
	return authorizeWant{equals: map[string]string{param: name}}
}

// planFirstAuthorizeWant is the plan's first sign-in, read from
// planAuthorizeURL: it names the app in agent_name_hint, and has a client id
// and a host id; it sends no originator and no id_token_hint.
func planFirstAuthorizeWant(agent string) authorizeWant {
	return authorizeWant{equals: map[string]string{"agent_name_hint": agent},
		present: []string{"client_id", "ext_agent_host_id"}, absent: []string{"originator", "id_token_hint"}}
}

// planAgainAuthorizeWant is the plan's sign-in again with Previous, read from
// planAuthorizeURL: Previous's client, its ID token as id_token_hint when it
// has one, and no agent_name_hint and no originator.
func planAgainAuthorizeWant(previous *auth.OAuthSession) authorizeWant {
	want := authorizeWant{equals: map[string]string{"client_id": previous.ClientID},
		present: []string{"ext_agent_host_id"}, absent: []string{"agent_name_hint", "originator"}}
	if previous.IDToken != "" {
		want.equals["id_token_hint"] = previous.IDToken
	} else {
		want.absent = append(want.absent, "id_token_hint")
	}
	return want
}

// The earlier-grant probe's checks (0081-PLAN Phase 1a, MADR D12): after a
// person signs in to a provider a second time, does the first session still
// work? TestLive_EarlierGrant_0081 runs them with a real sign-in, a real
// browser and the terminal device; TestEarlierGrantReport_0081 runs the same
// code against in-process stand-ins, and proves the output holds no token,
// code, URL, id or error text. Every sign-in lives in memory: no store is
// opened and nothing is written to disk.

// The four sign-ins the probe measures, named by LLMPROVIDER_LIVE_0081_PROVIDER.
const (
	grantChatGPT = "chatgpt"
	grantPlan    = "plan"
	grantGrok    = "grok"
	grantKilo    = "kilo"
)

var grantProviders = []string{grantChatGPT, grantPlan, grantGrok, grantKilo}

// The verdicts one run reports.
const (
	grantKept          = "kept"
	grantEnded         = "ended"
	grantIndeterminate = "indeterminate"
)

// grantSignInWait bounds one sign-in's wait for the person, grantUseLimit one
// use of a session, and grantRunLimit the run's sign-ins and uses: two
// sign-ins at their four minutes leave the run's last minute for four uses,
// which take seconds. grantRevokeLimit bounds each revocation, which runs
// after the verdict on a context of its own, so that the run's bound having
// passed does not stop it: the whole test is at most nine minutes and forty
// seconds, which the commands' -timeout of fifteen minutes holds.
const (
	grantSignInWait  = 4 * time.Minute
	grantUseLimit    = time.Minute
	grantRunLimit    = 9 * time.Minute
	grantRevokeLimit = 20 * time.Second
)

// grantByHand is the kind of a revocation line for a grant the probe does not
// revoke because it was ruled not to: it is neither a success nor a failure,
// and not Kilo's "unsupported", which says the SDK has no revocation.
const grantByHand = "by_hand"

// grantRevokes reports whether the probe revokes provider's grants at its end.
// It does not for chatgpt or grok: the owner ruled that their grants stay,
// listed for removal by hand, because nothing read says what the issuer's
// revocation ends besides the token it names, and each sign-in uses the client
// id the owner's own CLI uses, Codex's and the Grok CLI's (0081-PLAN, Phase 1b).
// No request is made for them on any path, discovery for revocation included.
// Kilo's goes through the SDK's call, which sends nothing and says unsupported.
func grantRevokes(provider string) bool { return provider != grantChatGPT && provider != grantGrok }

// grantTerminalCodes are the only OAuth error codes the probe keeps, auth's
// own terminal refresh codes (oauthTerminalRefreshCodes in auth); any other
// code is kept as "other".
var grantTerminalCodes = []string{
	"invalid_grant", "invalid_client", "refresh_token_expired", "refresh_token_reused", "refresh_token_invalidated",
}

// grantReusedCode is the code that marks a defect of the probe, never an ended
// grant: no refresh token is sent twice by the probe.
const grantReusedCode = "refresh_token_reused"

// grantRecorder is the transport every request of the probe goes through. It
// keeps, for the step under way, the status of the last response, the number
// of round trips that completed, and, for a response of 400 or more, the OAuth
// error code when it is one of grantTerminalCodes. It never reads a request,
// nor the body of a response under 400, and puts back every body it reads.
//
// The count is of completed round trips, a response or a transport error. The
// SDK sends a refresh again on a transport error, a 429 or a 5xx, up to three
// times with the same refresh token (refreshOAuthSession in auth), so a count
// above one in a use of a refreshing session says that the token was sent
// more than once. A round trip that completes in a later step than the one
// that sent it, a refresh still running after its use timed out, is counted in
// the later step.
type grantRecorder struct {
	next http.RoundTripper

	mu       sync.Mutex
	status   int
	code     string
	requests int
}

// begin starts a step: nothing has been answered yet.
func (g *grantRecorder) begin() {
	g.mu.Lock()
	g.status, g.code, g.requests = 0, "", 0
	g.mu.Unlock()
}

// last is the status of the step's last response, 0 when it has none or its
// last round trip failed, the code of a response of 400 or more, "" when
// there is none, and the number of round trips that completed.
func (g *grantRecorder) last() (status int, code string, requests int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.status, g.code, g.requests
}

func (g *grantRecorder) client() *http.Client { return &http.Client{Transport: g} }

// RoundTrip sends req on, and records the answer.
func (g *grantRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := g.next.RoundTrip(req)
	if err != nil {
		g.mu.Lock()
		g.status, g.code = 0, ""
		g.requests++
		g.mu.Unlock()
		return resp, err
	}
	code := ""
	if resp.StatusCode >= http.StatusBadRequest {
		code = grantErrorCode(resp)
	}
	g.mu.Lock()
	g.status, g.code = resp.StatusCode, code
	g.requests++
	g.mu.Unlock()
	return resp, nil
}

// grantCodeLimit is how much of a refusal's body the SDK reads for its code
// (oauthErrorBodyLimit in auth): a longer body gets the SDK no code, because the
// part it reads is no longer JSON, so the recorder reads no more.
const grantCodeLimit = 2048

// grantErrorCode reads the first grantCodeLimit bytes of a refused response's
// body for its OAuth error code (grantCodeOf), and puts them back, with the
// rest, for the caller.
func grantErrorCode(resp *http.Response) string {
	head, err := io.ReadAll(io.LimitReader(resp.Body, grantCodeLimit))
	resp.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(head), resp.Body), resp.Body}
	if err != nil {
		return ""
	}
	return grantCodeOf(head)
}

// grantCodeOf reads an OAuth error code from a body as the SDK's
// refreshErrorCode does (auth/oauth_session.go, unexported, so this is a copy;
// TestGrantCodeAgreesWithTheSDK_0081 fails if the two disagree on a table of
// bodies): {"error": "code"}, {"error": {"code": "code"}} or {"code": "code"},
// lower-cased. It keeps the code only when it is one of grantTerminalCodes, and
// otherwise says "other"; a body with no code, or none that parses, is "".
func grantCodeOf(body []byte) string {
	var top struct {
		Error json.RawMessage `json:"error"`
		Code  string          `json:"code"`
	}
	if json.Unmarshal(body, &top) != nil {
		return ""
	}
	var code string
	var inner struct {
		Code string `json:"code"`
	}
	switch {
	case json.Unmarshal(top.Error, &code) == nil && code != "":
	case json.Unmarshal(top.Error, &inner) == nil && inner.Code != "":
		code = inner.Code
	default:
		code = top.Code
	}
	code = strings.ToLower(code)
	switch {
	case code == "":
		return ""
	case slices.Contains(grantTerminalCodes, code):
		return code
	}
	return "other"
}

// grantStep is what one use of a session answered: the status of its last
// response, the error's kind, the allow-listed code, and the number of round
// trips the step made.
type grantStep struct {
	status   int
	kind     string
	code     string
	requests int
}

// grantKind names an error's kind, by errors.Is, and never its text.
func grantKind(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "canceled"
	case errors.Is(err, llmprovider.ErrUnsupported):
		return "unsupported"
	case errors.Is(err, llmprovider.ErrAuthFailure):
		return "auth"
	case errors.Is(err, llmprovider.ErrRateLimited):
		return "rate_limited"
	case errors.Is(err, llmprovider.ErrProviderUnavailable):
		return "unavailable"
	case errors.Is(err, llmprovider.ErrInvalidRequest):
		return "invalid_request"
	}
	return "other"
}

// grantVerdict is the run's one word (0081-PLAN Phase 1a, step 6, as ruled on
// 2026-11-18). baseline is the first session's use before the second
// sign-in, first its first use after it, last its second use after it (the
// zero step when it was not made, because first failed: a refresh token the
// provider refused is not sent again), and control the second session's use.
// A step's request count above one makes the whole run indeterminate
// (the owner's ruling "The run is `indeterminate`: run it once more"); a
// baseline that failed or made more than one request ends the run with that
// verdict before the second sign-in (earlierGrant.run). Then:
//
//	baseline  control  first   last    verdict
//	not ok    any      any     any     indeterminate
//	any       not ok   any     any     indeterminate
//	ok        ok       ok      ok      kept
//	ok        ok       ok      dead    ended      (the lazy case)
//	ok        ok       ok      reused  indeterminate
//	ok        ok       ok      other   indeterminate
//	ok        ok       dead    not run ended
//	ok        ok       reused  not run indeterminate
//	ok        ok       other   not run indeterminate
//
// "dead" is a refusal as the grant gone (grantDead): kind auth with a 400 or
// 401 and a terminal code other than refresh_token_reused, or, for Kilo,
// which has no refresh, a 401. "other" is any failure that is neither.
// "kept" says nothing of an ending that comes minutes later than the run.
func grantVerdict(provider string, baseline, first, last, control grantStep) string {
	for _, s := range []grantStep{baseline, first, last, control} {
		if s.requests > 1 {
			return grantIndeterminate
		}
	}
	if baseline.kind != "ok" || control.kind != "ok" {
		return grantIndeterminate
	}
	switch {
	case first.kind == "ok" && last.kind == "ok":
		return grantKept
	case first.kind == "ok" && grantDead(provider, last):
		return grantEnded
	case first.kind != "ok" && grantDead(provider, first):
		return grantEnded
	}
	return grantIndeterminate
}

// grantDead reports whether a failed use is the provider saying the grant is
// gone.
func grantDead(provider string, s grantStep) bool {
	if s.kind != "auth" {
		return false
	}
	if provider == grantKilo {
		return s.status == http.StatusUnauthorized
	}
	if s.status != http.StatusBadRequest && s.status != http.StatusUnauthorized {
		return false
	}
	return slices.Contains(grantTerminalCodes, s.code) && s.code != grantReusedCode
}

// grantPasses reports whether a verdict passes the test: only kept does, so
// that go test itself says when a provider ended the earlier grant or the
// probe could not tell. Whether ended or indeterminate should pass is the
// owner's to rule; a ruling is this function.
func grantPasses(verdict string) bool { return verdict == grantKept }

// grantWebURL reports whether raw is an http or https URL with a host, the only
// kind the browser opener is given.
func grantWebURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Host != "" && (u.Scheme == "https" || u.Scheme == "http")
}

// grantIO is how a sign-in shows the person what it needs: the system's
// browser opener, and the terminal device opened for writing only. Neither
// writes to the test's output.
type grantIO struct {
	canOpen func() bool
	open    func(raw string) error
	tty     func() (io.WriteCloser, error)
}

// earlierGrant is one provider's run.
type earlierGrant struct {
	provider  string
	logf      func(format string, args ...any)
	rec       *grantRecorder
	io        grantIO
	needsCode bool // Kilo's user code is written to the terminal device
	wait      time.Duration
	signIn    func(ctx context.Context, show *grantShow, previous *auth.OAuthSession) (*auth.OAuthSession, error)
	use       func(ctx context.Context, session *auth.OAuthSession) error
	// revoke ends a session's grant at the provider (auth.RevokeOAuthSession in
	// the live test), each within revokeLimit.
	revoke      func(ctx context.Context, session *auth.OAuthSession) error
	revokeLimit time.Duration

	mu      sync.Mutex
	stopped string
}

// say writes one line of the test's output, through the repository's
// redaction.
func (g *earlierGrant) say(format string, args ...any) {
	g.logf("%s", redact.String(fmt.Sprintf(format, args...)))
}

// stop records why the run cannot go on; the first reason stands.
func (g *earlierGrant) stop(reason string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stopped == "" {
		g.stopped = fmt.Sprintf("%s: stopped: %s", g.provider, reason)
	}
}

func (g *earlierGrant) stoppedLine() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.stopped
}

// grantShow is what a sign-in has of the person while it runs: URL hands a
// page to the browser, Code writes Kilo's user code to the terminal device.
// A failure of either stops the run with one plain line holding neither the
// URL, the code nor the error, and cancels the sign-in's context, because the
// browser sign-ins call URL from a goroutine and ignore what it returns
// (runBrowserLogin in auth): the stop and the cancel are URL's own, not its
// caller's.
type grantShow struct {
	g      *earlierGrant
	tty    io.Writer
	cancel context.CancelFunc
}

// URL opens raw in the browser, and shows it nowhere else.
func (s *grantShow) URL(raw string) error {
	if err := s.g.io.open(raw); err != nil {
		s.g.stop("the browser could not be opened")
		s.cancel()
		return errors.New("the browser could not be opened")
	}
	return nil
}

// Code writes Kilo's user code to the terminal device, and nowhere else.
func (s *grantShow) Code(code string) {
	if s.tty != nil {
		if _, err := fmt.Fprintf(s.tty, "Kilo code: %s\n", code); err == nil {
			return
		}
	}
	s.g.stop("the terminal device could not be opened")
	s.cancel()
}

// run is the sequence: sign-in-1, baseline, sign-in-2, first-after, control,
// one session at a time (0081-PLAN Phase 1a, step 4). It returns the verdict,
// or, when the browser or the terminal device could not be used, the plain
// line that stops the run and no verdict.
func (g *earlierGrant) run(ctx context.Context) (verdict, stopped string) {
	if !g.io.canOpen() {
		g.stop("the browser could not be opened")
		return "", g.stoppedLine()
	}
	var tty io.Writer
	if g.needsCode {
		w, err := g.io.tty()
		if err != nil {
			g.stop("the terminal device could not be opened")
			return "", g.stoppedLine()
		}
		defer func() { _ = w.Close() }()
		tty = w
	}
	// obtained is every session a sign-in returned, in order: each holds a
	// grant at the provider, to be revoked at the end whatever else happened.
	var obtained []*auth.OAuthSession
	signIn := func(step string, previous *auth.OAuthSession) *auth.OAuthSession {
		g.rec.begin()
		sctx, cancel := context.WithTimeout(ctx, g.wait)
		defer cancel()
		session, err := g.signIn(sctx, &grantShow{g: g, tty: tty, cancel: cancel}, previous)
		if err == nil && session != nil {
			session.UseHTTPClient(g.rec.client())
			obtained = append(obtained, session)
		}
		if g.stoppedLine() != "" {
			return nil
		}
		if err == nil && session == nil {
			err = errors.New("no session")
		}
		g.say("%s %s: %s", g.provider, step, grantKind(err))
		if err != nil {
			return nil
		}
		return session
	}
	use := func(step string, session *auth.OAuthSession) grantStep {
		g.rec.begin()
		uctx, cancel := context.WithTimeout(ctx, grantUseLimit)
		defer cancel()
		err := g.use(uctx, session)
		status, code, requests := g.rec.last()
		s := grantStep{status: status, kind: grantKind(err), code: code, requests: requests}
		shown := "-"
		if code != "" {
			shown = code
		}
		g.say("%s %s: status %d, kind %s, code %s, requests %d", g.provider, step, s.status, s.kind, shown, s.requests)
		return s
	}
	first := signIn("sign-in-1", nil)
	if first == nil {
		return g.finish(ctx, grantIndeterminate, obtained)
	}
	baseline := use("baseline", first)
	if baseline.kind != "ok" || baseline.requests > 1 {
		// The first session's refresh token was spent or refused, or sent more
		// than once: it is not sent again, and the verdict cannot change, so
		// the second sign-in is not asked of the person.
		return g.finish(ctx, grantIndeterminate, obtained)
	}
	var previous *auth.OAuthSession
	if g.provider == grantPlan {
		previous = first
	}
	second := signIn("sign-in-2", previous)
	if second == nil {
		return g.finish(ctx, grantIndeterminate, obtained)
	}
	after := use("first-after", first)
	control := use("control", second)
	var last grantStep
	if after.kind == "ok" {
		last = use("last-after", first)
	}
	return g.finish(ctx, grantVerdict(g.provider, baseline, after, last, control), obtained)
}

// finish ends a run: the verdict line, or none when the run was stopped, then
// the revocation of every grant obtained, and the stop line or the verdict.
// What revocation does never reaches what finish returns.
func (g *earlierGrant) finish(ctx context.Context, verdict string, obtained []*auth.OAuthSession) (string, string) {
	line := g.stoppedLine()
	if line == "" {
		g.say("%s: %s", g.provider, verdict)
	}
	g.revokeAll(ctx, obtained)
	if line != "" {
		return "", line
	}
	return verdict, ""
}

// revokeAll revokes each session's grant, in the order they were obtained,
// each on a context of its own, detached from the run's so that the run's
// bound having passed does not stop it, and bound by revokeLimit. Its line
// is the provider, which grant, the status of the revocation's last response
// (0 when it made none: Kilo's has no revocation, so it sends nothing, and
// chatgpt's is not made) and the error's kind, or the word by_hand for a grant
// the probe does not revoke, and nothing else: no code, body or error text, and no
// request count. begin starts each from nothing, so that no revocation
// request is counted in a use step or reports one's status. It returns how
// many failed, which only the twin reads: a failure never changes the
// verdict or fails the test.
func (g *earlierGrant) revokeAll(ctx context.Context, obtained []*auth.OAuthSession) (failed int) {
	for i, session := range obtained {
		if !grantRevokes(g.provider) {
			g.say("%s revoke-%d: status 0, kind %s", g.provider, i+1, grantByHand)
			continue
		}
		g.rec.begin()
		rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), g.revokeLimit)
		err := g.revoke(rctx, session)
		cancel()
		status, _, _ := g.rec.last()
		g.say("%s revoke-%d: status %d, kind %s", g.provider, i+1, status, grantKind(err))
		if err != nil {
			failed++
		}
	}
	return failed
}

// grantOutcome is what TestLive_EarlierGrant_0081 does with a run: skip with
// a reason, fail at once with a fixed message, or fail at its end.
type grantOutcome struct {
	skip  string
	fatal string
	fail  bool
}

// grantReporter is the part of *testing.T that carries out an outcome.
type grantReporter interface {
	Skip(args ...any)
	Fatal(args ...any)
	Fail()
}

// grantApply carries out an outcome on t: the live test's last three
// statements, here so that its twin can run them on a stand-in.
func grantApply(t grantReporter, out grantOutcome) {
	switch {
	case out.skip != "":
		t.Skip(out.skip)
	case out.fatal != "":
		t.Fatal(out.fatal)
	case out.fail:
		t.Fail()
	}
}

// grantRunTest is the body of TestLive_EarlierGrant_0081, with its inputs
// given, so that its twin can run it: the switch and the provider are read
// through getenv (os.Getenv in the live test), and nothing is built, so no
// request is made and no credential is read, until both are right.
func grantRunTest(ctx context.Context, getenv func(string) string, build func(provider string) (*earlierGrant, error)) grantOutcome {
	if getenv("LLMPROVIDER_LIVE_0081") != "1" {
		return grantOutcome{skip: "LLMPROVIDER_LIVE_0081 unset: this signs in twice at one provider and needs a person"}
	}
	provider := getenv("LLMPROVIDER_LIVE_0081_PROVIDER")
	if !slices.Contains(grantProviders, provider) {
		return grantOutcome{fatal: "LLMPROVIDER_LIVE_0081_PROVIDER must be chatgpt, plan, grok or kilo"}
	}
	g, err := build(provider)
	if err != nil {
		return grantOutcome{fatal: "the probe could not be set up"}
	}
	verdict, stopped := g.run(ctx)
	if stopped != "" {
		return grantOutcome{fatal: stopped}
	}
	return grantOutcome{fail: !grantPasses(verdict)}
}

// grantUseRefresh is the use of a session that refreshes: one forced refresh,
// the request a revoked grant cannot pass. The session has no store, so
// nothing is saved, and each refresh token it holds is sent once, unless the
// SDK sends a request again (grantRecorder).
func grantUseRefresh(ctx context.Context, session *auth.OAuthSession) error {
	session.Invalidate()
	_, err := session.Token(ctx)
	return err
}

// grantUseKiloProfile is the use of a Kilo session, which does not refresh: a
// read of the account's profile with its token, which spends and rotates
// nothing.
func grantUseKiloProfile(opts ...llmprovider.Option) func(context.Context, *auth.OAuthSession) error {
	return func(ctx context.Context, session *auth.OAuthSession) error {
		_, err := auth.KiloProfile(ctx, session.Access, opts...)
		return err
	}
}
