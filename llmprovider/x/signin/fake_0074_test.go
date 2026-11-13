package signin

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
)

// fakeBase is the fake issuer's origin; nothing is sent to it, since the
// fake is the client's transport.
const fakeBase = "https://issuer.test"

// fakeKilo is the fake Kilo origin.
const fakeKilo = "https://kilo.test"

var fakeKey = sync.OnceValue(func() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
})

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// fakeIssuer is an in-memory identity provider for the sign-ins of this
// package: the transport of the http.Client a test hands to Options. It
// answers OpenAI's, Grok's and Kilo's endpoints, so that a sign-in runs the
// real auth code end to end with no network and no listener. The polls of a
// device sign-in are scripted: each reply is consumed in order, and the
// last one repeats.
type fakeIssuer struct {
	mu sync.Mutex
	// openAIPoll, grokPoll and kiloPoll script the device polls. A Grok reply
	// is "ok" or an OAuth error code; the others are HTTP statuses.
	openAIPoll []int
	grokPoll   []string
	kiloPoll   []int
	nonce      string
	redirect   string
	requests   []string
	tokenCalls int
}

func (f *fakeIssuer) client() *http.Client { return &http.Client{Transport: f} }

// seen returns "METHOD path" of each request so far.
func (f *fakeIssuer) seen() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requests...)
}

// setNonce records the authorize URL's nonce, which the next id_token echoes.
func (f *fakeIssuer) setNonce(authorizeURL string) {
	if u, err := url.Parse(authorizeURL); err == nil {
		f.mu.Lock()
		f.nonce = u.Query().Get("nonce")
		f.mu.Unlock()
	}
}

func next[T any](script []T, calls int, fallback T) T {
	if len(script) == 0 {
		return fallback
	}
	if calls >= len(script) {
		return script[len(script)-1]
	}
	return script[calls]
}

func reply(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func mustJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// RoundTrip answers one request.
func (f *fakeIssuer) RoundTrip(req *http.Request) (*http.Response, error) {
	var form url.Values
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		form, _ = url.ParseQuery(string(raw))
		if len(form) == 0 {
			form = jsonForm(raw)
		}
	}
	f.mu.Lock()
	f.requests = append(f.requests, req.Method+" "+req.URL.Path)
	f.mu.Unlock()
	path := req.URL.Path
	switch {
	case strings.HasSuffix(path, "/.well-known/openid-configuration"):
		return reply(200, mustJSON(map[string]any{
			"issuer": fakeBase, "authorization_endpoint": fakeBase + "/oauth/authorize",
			"token_endpoint": fakeBase + "/oauth/token", "device_authorization_endpoint": fakeBase + "/oauth/device",
			"jwks_uri": fakeBase + "/jwks", "id_token_signing_alg_values_supported": []string{"RS256"},
		})), nil
	case path == "/jwks":
		key := fakeKey()
		return reply(200, mustJSON(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "use": "sig", "kid": "rsa-1", "n": b64(key.N.Bytes()),
			"e": b64(big.NewInt(int64(key.E)).Bytes()),
		}}})), nil
	case path == "/oauth/token":
		return f.token(form), nil
	case path == "/oauth/device":
		return reply(200, mustJSON(map[string]any{"device_code": "grok-device", "user_code": "GROK-1234",
			"verification_uri": fakeBase + "/device", "expires_in": 900, "interval": 1})), nil
	case path == "/api/accounts/deviceauth/usercode":
		return reply(200, mustJSON(map[string]any{"device_auth_id": "auth-id", "user_code": "OPEN-1234", "interval": "896"})), nil
	case path == "/api/accounts/deviceauth/token":
		f.mu.Lock()
		status := next(f.openAIPoll, f.countLocked("/api/accounts/deviceauth/token")-1, 200)
		f.mu.Unlock()
		if status != 200 {
			return reply(status, "{}"), nil
		}
		return reply(200, mustJSON(map[string]string{"authorization_code": "device-code", "code_verifier": "verifier"})), nil
	case path == "/api/device-auth/codes" && req.Method == http.MethodPost:
		return reply(200, mustJSON(map[string]any{"code": "KILO-1", "verificationUrl": fakeKilo + "/device", "expiresIn": 600})), nil
	case strings.HasPrefix(path, "/api/device-auth/codes/"):
		f.mu.Lock()
		status := next(f.kiloPoll, f.countLocked(path)-1, 200)
		f.mu.Unlock()
		if status != 200 {
			return reply(status, "{}"), nil
		}
		return reply(200, mustJSON(map[string]string{"token": "kilo-device-token-0123456789"})), nil
	}
	return reply(404, "{}"), nil
}

// countLocked is how many requests so far were for path. f.mu is held.
func (f *fakeIssuer) countLocked(path string) int {
	n := 0
	for _, r := range f.requests {
		if strings.HasSuffix(r, " "+path) {
			n++
		}
	}
	return n
}

func jsonForm(raw []byte) url.Values {
	var m map[string]string
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	form := url.Values{}
	for k, v := range m {
		form.Set(k, v)
	}
	return form
}

// token answers the token endpoint: a code exchange, or Grok's device poll.
func (f *fakeIssuer) token(form url.Values) *http.Response {
	f.mu.Lock()
	defer f.mu.Unlock()
	if form.Get("redirect_uri") != "" {
		f.redirect = form.Get("redirect_uri")
	}
	if strings.Contains(form.Get("grant_type"), "device_code") {
		outcome := next(f.grokPoll, f.tokenCalls, "ok")
		f.tokenCalls++
		if outcome != "ok" {
			return reply(400, mustJSON(map[string]string{"error": outcome}))
		}
	}
	claims := map[string]any{"iss": fakeBase, "aud": form.Get("client_id"), "sub": "user-1",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix()}
	if f.nonce != "" {
		claims["nonce"] = f.nonce
	}
	return reply(200, mustJSON(map[string]any{"access_token": "access-token-0123456789", "refresh_token": "refresh-token-0123456789",
		"expires_in": 3600, "id_token": signJWT(claims)}))
}

// signJWT signs claims with the fake key, as RS256.
func signJWT(claims map[string]any) string {
	header := b64([]byte(mustJSON(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "rsa-1"})))
	input := header + "." + b64([]byte(mustJSON(claims)))
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, fakeKey(), crypto.SHA256, digest[:])
	if err != nil {
		panic(err)
	}
	return input + "." + b64(sig)
}

// collect returns every step still buffered once Wait has returned, and
// fails t when the channel is not closed after them.
func collect(t *testing.T, run *SignIn) []authcontract.Step {
	t.Helper()
	var steps []authcontract.Step
	for {
		select {
		case step, ok := <-run.Steps():
			if !ok {
				return steps
			}
			steps = append(steps, step)
		default:
			t.Error("the steps channel is still open after the end")
			return steps
		}
	}
}

// kinds lists the step kinds, for a failure message and a comparison.
func kinds(steps []authcontract.Step) []authcontract.StepKind {
	out := make([]authcontract.StepKind, 0, len(steps))
	for _, s := range steps {
		out = append(out, s.Kind)
	}
	return out
}

// nextStep waits for the next step, with a real-time bound, for a test that
// is not in a synctest bubble.
func nextStep(t *testing.T, run *SignIn) authcontract.Step {
	t.Helper()
	select {
	case step, ok := <-run.Steps():
		if !ok {
			t.Fatal("the steps channel closed; want another step")
		}
		return step
	case <-time.After(10 * time.Second):
		t.Fatal("no step in 10 s")
	}
	return authcontract.Step{}
}

// assertSeq fails t unless the steps are numbered 1, 2, 3, ...
func assertSeq(t *testing.T, steps []authcontract.Step) {
	t.Helper()
	for i, step := range steps {
		if step.Seq != i+1 {
			t.Errorf("step %d (%s) has Seq %d; want %d", i, step.Kind, step.Seq, i+1)
		}
		if step.SchemaVersion != authcontract.SchemaVersion {
			t.Errorf("step %d has SchemaVersion %d; want %d", i, step.SchemaVersion, authcontract.SchemaVersion)
		}
	}
}

// background is a context for a test that bounds nothing.
func background() context.Context { return context.Background() }
