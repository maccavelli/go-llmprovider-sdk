package wizard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// 0080-MADR D1 to D6: the credential step verifies a Kilo credential used
// with no organization against Kilo's profile, under Options.Discover. Each
// test runs against kilo0080, an in-process stand-in for Kilo (0065-MADR: no
// outside host), through Options.HTTPClient, so the real auth.KiloProfile
// and the real listing answer the wizard.
//
// Every credential below is a fixed fake value.

const (
	// kiloGood0080 and kiloGood2nd0080 are fake keys the stand-in accepts;
	// kiloBad0080 and kiloBad2nd0080 are fake keys it refuses.
	kiloGood0080    = "fake-kilo-good-key-0080"
	kiloGood2nd0080 = "fake-kilo-good-key-second-0080"
	kiloBad0080     = "fake-kilo-bad-key-0080"
	kiloBad2nd0080  = "fake-kilo-bad-key-second-0080"

	// kiloServiceText0080 is text the stand-in puts in every failure body, so
	// that a test sees whether the wizard shows the service's words (0056-MADR
	// D3).
	kiloServiceText0080 = "SERVICE-MESSAGE-0080"

	kiloOrganization0080 = "org-kept-0080"
)

// kiloRead0080 is one request the stand-in saw.
type kiloRead0080 struct {
	token string // the bearer, empty when none was sent
	agent string // the User-Agent
	host  string
	path  string
	org   string // the organization header
}

// kilo0080 answers as Kilo does (0080-MADR, "Live probes of 2026-11-16", and
// the owner's run of 2026-11-17 in 0080-PLAN, "Phase 1, the owner's run"):
//
//   - the public listing answers 200 whatever the credential (L1 to L6);
//   - an organization's listing answers 401 to a credential it does not know
//     (L7 to L12), and 200 to one it does;
//   - /api/profile answers 401 "Unauthorized" with no credential and 401
//     "Invalid token" with a bad one (L13 to L16), and 200 with a profile
//     for a good one (the owner's two live runs);
//   - anything else is a 404.
//
// profile, when set, replaces the profile's answer: an outage.
type kilo0080 struct {
	mu       sync.Mutex
	accepts  map[string]bool
	profile  func(*http.Request) (*http.Response, error)
	profiles []kiloRead0080
	orgLists []kiloRead0080
	public   []kiloRead0080
	others   []kiloRead0080
}

// newKilo0080 is a stand-in that accepts exactly the fake credentials given.
func newKilo0080(accepted ...string) *kilo0080 {
	k := &kilo0080{accepts: map[string]bool{}}
	for _, token := range accepted {
		k.accepts[token] = true
	}
	return k
}

func respond0080(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status),
		Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

// outage makes the profile answer status with a body that carries the
// service's text.
func (k *kilo0080) outage(status int) {
	k.profile = func(r *http.Request) (*http.Response, error) {
		return respond0080(r, status, `{"error":"`+kiloServiceText0080+`"}`), nil
	}
}

func (k *kilo0080) RoundTrip(r *http.Request) (*http.Response, error) {
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	seen := kiloRead0080{token: token, agent: r.Header.Get("User-Agent"), host: r.URL.Host, path: r.URL.Path,
		org: r.Header.Get("X-KILOCODE-ORGANIZATIONID")}
	k.mu.Lock()
	defer k.mu.Unlock()
	switch {
	case strings.HasSuffix(r.URL.Path, "/api/profile"):
		k.profiles = append(k.profiles, seen)
		switch {
		case k.profile != nil:
			k.mu.Unlock()
			resp, err := k.profile(r)
			k.mu.Lock()
			return resp, err
		case token == "":
			return respond0080(r, http.StatusUnauthorized, `{"error":"Unauthorized"}`), nil
		case !k.accepts[token]:
			return respond0080(r, http.StatusUnauthorized, `{"error":"Invalid token (req-0080)"}`), nil
		}
		return respond0080(r, http.StatusOK, `{"organizations":[],"hasPersonalAccount":true}`), nil
	case strings.Contains(r.URL.Path, "/organizations/") && strings.HasSuffix(r.URL.Path, "/models"):
		k.orgLists = append(k.orgLists, seen)
		if !k.accepts[token] {
			return respond0080(r, http.StatusUnauthorized, `{"error":"User not authenticated - no user to set on context"}`), nil
		}
		return respond0080(r, http.StatusOK, `{"data":[{"id":"kilo-auto/free"}]}`), nil
	case strings.HasSuffix(r.URL.Path, "/models"):
		k.public = append(k.public, seen)
		return respond0080(r, http.StatusOK, `{"data":[{"id":"kilo-auto/free"}]}`), nil
	}
	k.others = append(k.others, seen)
	return respond0080(r, http.StatusNotFound, `{}`), nil
}

// options is the wizard's options for a check under Discover through k, with
// a listing cache of this test's own (0058-MADR).
func (k *kilo0080) options(t *testing.T) Options {
	t.Helper()
	return Options{Discover: true, HTTPClient: &http.Client{Transport: k},
		ProviderOptions: []llmprovider.Option{testCache(t)}}
}

// profileTokens is the bearer of each profile request, in order.
func (k *kilo0080) profileTokens() []string {
	k.mu.Lock()
	defer k.mu.Unlock()
	var out []string
	for _, seen := range k.profiles {
		out = append(out, seen.token)
	}
	return out
}

func (k *kilo0080) profileReads() []kiloRead0080 {
	k.mu.Lock()
	defer k.mu.Unlock()
	return slices.Clone(k.profiles)
}

// requests is the number of requests of every kind the stand-in saw.
func (k *kilo0080) requests() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.profiles) + len(k.orgLists) + len(k.public) + len(k.others)
}

// stubKiloDeviceLogin0080 replaces only the device login, as stubKilo does,
// and leaves the profile to the stand-in. The token it issues is
// kiloWizardToken, a fixed fake.
func stubKiloDeviceLogin0080(t *testing.T) {
	t.Helper()
	original := deviceLogin
	t.Cleanup(func() { deviceLogin = original })
	deviceLogin = func(_ context.Context, provider llmprovider.ProviderID, _ auth.OAuthFlowOptions, show func(DeviceCode) error) (*auth.OAuthSession, error) {
		if err := show(DeviceCode{Provider: provider, URL: "https://app.kilo.ai/device", Code: "KILO-0080"}); err != nil {
			return nil, err
		}
		return &auth.OAuthSession{Provider: llmprovider.ProviderKilo, Access: kiloWizardToken}, nil
	}
}

// keptKilo0080 is a store holding a Kilo session of the fake token access.
func keptKilo0080(access string) *store0059 {
	store := newStore0059()
	store.held[llmprovider.ProviderKilo] = &auth.OAuthSession{Provider: llmprovider.ProviderKilo, Access: access}
	return store
}

// wantRefused0080 fails unless err is Kilo's 401 as the wizard's refusal: it
// matches ErrAuthFailure, not ErrNotPermitted, carries status 401, and its
// text holds wording.
func wantRefused0080(t *testing.T, err error, wording string) {
	t.Helper()
	if err == nil {
		t.Errorf("error nil; want a refusal saying %q", wording)
		return
	}
	apiErr, isAPI := errors.AsType[*llmprovider.APIError](err)
	if !errors.Is(err, llmprovider.ErrAuthFailure) || errors.Is(err, llmprovider.ErrNotPermitted) || !isAPI || apiErr.Status != http.StatusUnauthorized {
		t.Errorf("error %v; want Kilo's 401, matching ErrAuthFailure and not ErrNotPermitted", err)
	}
	if !strings.Contains(err.Error(), wording) {
		t.Errorf("error %v; want it to say %q", err, wording)
	}
}

// assertNoKiloServiceText0080 fails when anything the prompter showed holds
// the stand-in's message, or a fake credential.
func assertNoKiloServiceText0080(t *testing.T, f *fakePrompter) {
	t.Helper()
	for _, shown := range f.allText {
		for _, forbidden := range []string{kiloServiceText0080, kiloGood0080, kiloBad0080, kiloWizardToken} {
			if strings.Contains(shown, forbidden) {
				t.Errorf("shown %q; want neither the service's message nor a credential", shown)
			}
		}
	}
}

// TestKiloVerify_TypedKeyRefused_0080 (D1, D3): a typed Kilo key the profile
// refuses is asked for again, as a refused key is for every provider; the
// second is returned. The public listing answers 200 to both, so only the
// profile can refuse the first.
func TestKiloVerify_TypedKeyRefused_0080(t *testing.T) {
	k := newKilo0080(kiloGood0080)
	d := descriptor0056(t, llmprovider.ProviderKilo)
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, secrets: []string{kiloBad0080, kiloGood0080}})
	res, _, err := ResolveCredential(context.Background(), f, k.options(t), llmprovider.ProviderKilo)
	if err != nil {
		t.Fatalf("ResolveCredential: %v", err)
	}
	want := d.Label + " refused this key (authentication failed, HTTP 401); enter another"
	if n := countContaining(f.seenNotify, want); n != 1 {
		t.Errorf("notices %q; want %q once", f.seenNotify, want)
	}
	if res.Kind != CredAPIKey || res.APIKey != kiloGood0080 {
		t.Errorf("Result kind %q, key %q; want the second key", res.Kind, res.APIKey)
	}
	if len(f.seenSecret) != 2 {
		t.Errorf("%d keys asked; want 2", len(f.seenSecret))
	}
	if got := k.profileTokens(); !slices.Equal(got, []string{kiloBad0080, kiloGood0080}) {
		t.Errorf("profile read with %q; want the first key, then the second", got)
	}
	assertNoKiloServiceText0080(t, f)
}

// TestKiloVerify_EnvironmentAndSavedKeyRefused_0080 (D1, D3): a key from the
// environment, then a saved one, that the profile refuses ends the run naming
// its source, matching ErrAuthFailure; no key is asked again (0028-MADR
// D-H10).
func TestKiloVerify_EnvironmentAndSavedKeyRefused_0080(t *testing.T) {
	d := descriptor0056(t, llmprovider.ProviderKilo)
	t.Run("environment", func(t *testing.T) {
		k := newKilo0080()
		o := k.options(t)
		o.AllowEnv = true
		o.LookupEnv = func(name string) string {
			if name == d.EnvVar {
				return kiloBad0080
			}
			return ""
		}
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, confirms: []bool{true}})
		_, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
		wantRefused0080(t, err, d.Label+" refused the key in "+d.EnvVar)
		if len(f.seenSecret) != 0 || len(k.profileTokens()) != 1 {
			t.Errorf("%d keys asked, %d profile reads; want none asked, one read", len(f.seenSecret), len(k.profileTokens()))
		}
	})
	t.Run("saved", func(t *testing.T) {
		k := newKilo0080()
		o := k.options(t)
		o.Existing = Result{Provider: llmprovider.ProviderKilo, Kind: CredAPIKey, APIKey: kiloBad0080}
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, confirms: []bool{true}})
		_, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
		wantRefused0080(t, err, d.Label+" refused the saved key")
		if len(f.seenSecret) != 0 || len(k.profileTokens()) != 1 {
			t.Errorf("%d keys asked, %d profile reads; want none asked, one read", len(f.seenSecret), len(k.profileTokens()))
		}
	})
}

// TestKiloVerify_DeviceKeyRefused_0080 (D2, D3; Q1): under
// Options.KiloDeviceKey, a device token the profile refuses ends the run,
// saying to sign in again, matching ErrAuthFailure. The profile is read once,
// no organization is asked and nothing is saved (0056-MADR D2, amended).
func TestKiloVerify_DeviceKeyRefused_0080(t *testing.T) {
	stubKiloDeviceLogin0080(t)
	k := newKilo0080()
	store := newStore0059()
	o := k.options(t)
	o.TokenStore, o.KiloDeviceKey = store, true
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
	_, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
	wantRefused0080(t, err, "refused the device sign-in; sign in again")
	if reads := len(k.profileTokens()); reads != 1 {
		t.Errorf("%d profile reads; want one", reads)
	}
	if slices.Contains(f.seenSelect, "Use Kilo as:") || len(store.recorded()) != 0 {
		t.Errorf("selects %q, saves %+v; want no organization question and no save", f.seenSelect, store.recorded())
	}
}

// TestKiloVerify_DeviceSessionRefused_0080 (D3; W2): a device session whose
// profile read, made for the organization menu, is a 401 is refused, not
// warned about: the store is as it was, the error says to sign in again, and
// the profile is read once, not twice.
func TestKiloVerify_DeviceSessionRefused_0080(t *testing.T) {
	stubKiloDeviceLogin0080(t)
	k := newKilo0080()
	store := newStore0059()
	o := k.options(t)
	o.TokenStore = store
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
	_, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
	wantRefused0080(t, err, "refused the sign-in; sign in again")
	if saves, held := store.recorded(), store.held[llmprovider.ProviderKilo]; len(saves) != 0 || held != nil {
		t.Errorf("saves %+v, the store holds %+v; want none, and nothing", saves, held)
	}
	if reads := len(k.profileTokens()); reads != 1 {
		t.Errorf("%d profile reads; want one", reads)
	}
	// Not today's warning as well: the 401 is a refusal (D3, C23).
	if n := countContaining(f.seenNotify, "could not read the Kilo account's organizations"); n != 0 {
		t.Errorf("notices %q; want no organization-menu warning beside the refusal", f.seenNotify)
	}
}

// TestKiloVerify_KeptSessionRefused_0080 (D3; 0066-MADR V7): a kept Kilo
// session, with no organization, that the profile refuses leads to the notice
// and the sign-in menu, once; the new session is saved. The kept token is
// read once and the new one once.
func TestKiloVerify_KeptSessionRefused_0080(t *testing.T) {
	stubKiloDeviceLogin0080(t)
	k := newKilo0080(kiloWizardToken)
	store := keptKilo0080(keptAccess0066)
	o := k.options(t)
	o.TokenStore = store
	o.Existing = Result{Provider: llmprovider.ProviderKilo, Kind: CredOAuth}
	d := descriptor0056(t, llmprovider.ProviderKilo)
	f := newFake(t, fakePrompter{confirms: []bool{true}, inputs: []string{acceptDefault},
		selects: []int{methodIdx0059(t, o, llmprovider.ProviderKilo, llmprovider.AuthDeviceCode)}})
	res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
	if err != nil {
		t.Fatalf("ResolveCredential: %v", err)
	}
	assertKeptRefused0066(t, f, d.Label)
	if got := k.profileTokens(); !slices.Equal(got, []string{keptAccess0066, kiloWizardToken}) {
		t.Errorf("profile read with %d tokens, %q; want the kept token, then the new one", len(got), got)
	}
	if saves := store.recorded(); res.Kind != CredOAuth || len(saves) != 1 || saves[0].access != kiloWizardToken {
		t.Errorf("Kind %q, saves %+v; want oauth, and the new token saved once", res.Kind, saves)
	}
}

// assertSessionOutage0080 fails unless a device session whose profile could
// not be read, under Discover, told two notices, each once and in this order
// (0080-MADR D6, amended 2026-11-17 by the owner's ruling, relayed by the
// coordinator, "Two notices"): that the organizations could not be read and
// the personal account is used, then that the key could not be verified. The
// notices give class and status, and never the service's text.
func assertSessionOutage0080(t *testing.T, notices []string, class string) {
	t.Helper()
	menu := "could not read the Kilo account's organizations (" + class + "); using the personal account"
	check := "could not verify the Kilo key (" + class + "); going on without verifying it"
	if countContaining(notices, menu) != 1 || countContaining(notices, check) != 1 {
		t.Errorf("notices %q; want %q once and %q once", notices, menu, check)
		return
	}
	at := func(want string) int {
		return slices.IndexFunc(notices, func(n string) bool { return strings.Contains(n, want) })
	}
	if at(menu) > at(check) {
		t.Errorf("notices %q; want the organization menu's first, then the check's", notices)
	}
}

// TestKiloVerify_OutagesWarn_0080 (D5, D6; Q4, Q5): a profile that cannot
// say whether the key is good warns once, by class and status and never the
// service's text, and the sign-in goes on: a 403 (every Kilo 403 is
// ErrNotPermitted, so no refusal), a 404, a 500, a 503, a body that does not
// decode, a refused connection and a stall past Options.DiscoverLimit.
func TestKiloVerify_OutagesWarn_0080(t *testing.T) {
	const limit = 300 * time.Millisecond
	verify := func(class string) string {
		return "could not verify the Kilo key (" + class + "); going on without verifying it"
	}
	typed := func(t *testing.T, k *kilo0080, o Options, class string) {
		t.Helper()
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, secrets: []string{kiloGood0080}})
		start := time.Now()
		res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
		if err != nil {
			t.Fatalf("ResolveCredential: %v", err)
		}
		if took := time.Since(start); took > 5*time.Second {
			t.Errorf("took %v; want the check bound by DiscoverLimit (%v), not auth's 30 s", took, limit)
		}
		if res.Kind != CredAPIKey || res.APIKey != kiloGood0080 {
			t.Errorf("Result kind %q, key %q; want the key returned", res.Kind, res.APIKey)
		}
		if n := countContaining(f.seenNotify, verify(class)); n != 1 {
			t.Errorf("notices %q; want %q once", f.seenNotify, verify(class))
		}
		if n := countContaining(f.seenNotify, "refused"); n != 0 || len(f.seenSecret) != 1 {
			t.Errorf("notices %q, %d keys asked; want no refusal and one key", f.seenNotify, len(f.seenSecret))
		}
		assertNoKiloServiceText0080(t, f)
	}
	// The classes are describe's words for each answer (0056-MADR D3).
	for _, tc := range []struct {
		name   string
		status int
		class  string
	}{
		{"403", http.StatusForbidden, "not permitted, HTTP 403"},
		{"404", http.StatusNotFound, "invalid request, HTTP 404"},
		{"500", http.StatusInternalServerError, "provider unavailable, HTTP 500"},
		{"503", http.StatusServiceUnavailable, "provider unavailable, HTTP 503"},
	} {
		t.Run("typed key, "+tc.name, func(t *testing.T) {
			k := newKilo0080(kiloGood0080)
			k.outage(tc.status)
			typed(t, k, k.options(t), tc.class)
			if reads := len(k.profileTokens()); reads != 1 {
				t.Errorf("%d profile reads; want one", reads)
			}
		})
	}
	t.Run("typed key, a body that does not decode", func(t *testing.T) {
		k := newKilo0080(kiloGood0080)
		k.profile = func(r *http.Request) (*http.Response, error) {
			return respond0080(r, http.StatusOK, "<html>"+kiloServiceText0080+"</html>"), nil
		}
		typed(t, k, k.options(t), "the request failed")
	})
	t.Run("typed key, a refused connection", func(t *testing.T) {
		k := newKilo0080(kiloGood0080)
		o := k.options(t)
		o.HTTPClient = nettest.Refusing()
		typed(t, k, o, "offline")
	})
	t.Run("typed key, a stall past DiscoverLimit", func(t *testing.T) {
		k := newKilo0080(kiloGood0080)
		k.profile = func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		o := k.options(t)
		o.DiscoverLimit = limit
		typed(t, k, o, "timed out")
	})
	t.Run("device key, 503", func(t *testing.T) {
		stubKiloDeviceLogin0080(t)
		k := newKilo0080()
		k.outage(http.StatusServiceUnavailable)
		o := k.options(t)
		o.TokenStore, o.KiloDeviceKey = newStore0059(), true
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
		res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
		if err != nil {
			t.Fatalf("ResolveCredential: %v", err)
		}
		if n := countContaining(f.seenNotify, verify("provider unavailable, HTTP 503")); n != 1 || res.APIKey != kiloWizardToken {
			t.Errorf("notices %q, key %q; want the warning once, and the device key returned", f.seenNotify, res.APIKey)
		}
		assertNoKiloServiceText0080(t, f)
	})
	t.Run("device session, a stall past DiscoverLimit", func(t *testing.T) {
		stubKiloDeviceLogin0080(t)
		k := newKilo0080()
		k.profile = func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		store := newStore0059()
		o := k.options(t)
		o.TokenStore, o.DiscoverLimit = store, limit
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
		start := time.Now()
		res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
		if err != nil {
			t.Fatalf("ResolveCredential: %v", err)
		}
		if took := time.Since(start); took > 5*time.Second {
			t.Errorf("took %v; want the session's read bound by DiscoverLimit (%v), not auth's 30 s", took, limit)
		}
		if res.Kind != CredOAuth {
			t.Errorf("Kind %q; want the session returned", res.Kind)
		}
		assertSessionOutage0080(t, f.seenNotify, "timed out")
	})
	for _, tc := range []struct {
		name   string
		status int
		class  string
	}{
		{"403", http.StatusForbidden, "not permitted, HTTP 403"},
		{"503", http.StatusServiceUnavailable, "provider unavailable, HTTP 503"},
	} {
		t.Run("device session, "+tc.name, func(t *testing.T) {
			stubKiloDeviceLogin0080(t)
			k := newKilo0080()
			k.outage(tc.status)
			store := newStore0059()
			o := k.options(t)
			o.TokenStore = store
			f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
			res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
			if err != nil {
				t.Fatalf("ResolveCredential: %v", err)
			}
			// The session's one profile read is its check; two notices, as
			// the owner ruled for a device session under Discover.
			if len(k.profileTokens()) != 1 {
				t.Errorf("%d profile reads; want one", len(k.profileTokens()))
			}
			assertSessionOutage0080(t, f.seenNotify, tc.class)
			if saves := store.recorded(); res.Kind != CredOAuth || len(saves) != 1 || saves[0].access != kiloWizardToken {
				t.Errorf("Kind %q, saves %+v; want oauth, saved once", res.Kind, saves)
			}
			assertNoKiloServiceText0080(t, f)
		})
	}
}

// TestKiloVerify_OrganizationReadsNoProfile_0080 (D4; O1): a kept session
// with Existing.Organization reads no profile: the organization's listing is
// the check, and its 401 still refuses.
func TestKiloVerify_OrganizationReadsNoProfile_0080(t *testing.T) {
	d := descriptor0056(t, llmprovider.ProviderKilo)
	t.Run("accepted", func(t *testing.T) {
		k := newKilo0080(keptAccess0066)
		o := k.options(t)
		o.TokenStore = keptKilo0080(keptAccess0066)
		o.Existing = Result{Provider: llmprovider.ProviderKilo, Kind: CredOAuth, Organization: kiloOrganization0080}
		f := newFake(t, fakePrompter{confirms: []bool{true}, inputs: []string{acceptDefault}})
		res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
		if err != nil {
			t.Fatalf("ResolveCredential: %v", err)
		}
		if reads := k.profileReads(); len(reads) != 0 || res.Organization != kiloOrganization0080 {
			t.Errorf("%d profile reads, organization %q; want none, and %q kept", len(reads), res.Organization, kiloOrganization0080)
		}
		if len(k.orgLists) != 1 || k.orgLists[0].org != kiloOrganization0080 {
			t.Errorf("organization listings %+v; want one, with the organization's header", k.orgLists)
		}
	})
	t.Run("refused", func(t *testing.T) {
		stubKiloDeviceLogin0080(t)
		k := newKilo0080(kiloWizardToken)
		store := keptKilo0080(keptAccess0066)
		o := k.options(t)
		o.TokenStore = store
		o.Existing = Result{Provider: llmprovider.ProviderKilo, Kind: CredOAuth, Organization: kiloOrganization0080}
		f := newFake(t, fakePrompter{confirms: []bool{true}, inputs: []string{acceptDefault},
			selects: []int{methodIdx0059(t, o, llmprovider.ProviderKilo, llmprovider.AuthDeviceCode)}})
		if _, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo); err != nil {
			t.Fatalf("ResolveCredential: %v", err)
		}
		assertKeptRefused0066(t, f, d.Label)
		if got := k.profileTokens(); !slices.Equal(got, []string{kiloWizardToken}) {
			t.Errorf("profile read with %q; want only the new session's token, none for the kept one", got)
		}
	})
}

// TestKiloVerify_NoDiscoverNoRead_0080 (D3; Q7): without Options.Discover,
// nothing is verified: a typed key and a device key read no profile, and a
// device session reads only the one its organization menu needs, whose 401
// is still a warning.
func TestKiloVerify_NoDiscoverNoRead_0080(t *testing.T) {
	t.Run("typed key", func(t *testing.T) {
		k := newKilo0080()
		o := k.options(t)
		o.Discover = false
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, secrets: []string{kiloBad0080}})
		res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
		if err != nil || res.APIKey != kiloBad0080 || k.requests() != 0 {
			t.Errorf("key %q, error %v, %d requests; want the key, no error, no request", res.APIKey, err, k.requests())
		}
	})
	t.Run("device key", func(t *testing.T) {
		stubKiloDeviceLogin0080(t)
		k := newKilo0080()
		o := k.options(t)
		o.Discover, o.TokenStore, o.KiloDeviceKey = false, newStore0059(), true
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
		res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
		if err != nil || res.APIKey != kiloWizardToken || k.requests() != 0 {
			t.Errorf("key %q, error %v, %d requests; want the key, no error, no request", res.APIKey, err, k.requests())
		}
	})
	t.Run("device session", func(t *testing.T) {
		stubKiloDeviceLogin0080(t)
		k := newKilo0080()
		store := newStore0059()
		o := k.options(t)
		o.Discover, o.TokenStore = false, store
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
		res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
		if err != nil || res.Kind != CredOAuth {
			t.Fatalf("Kind %q, error %v; want oauth, no error", res.Kind, err)
		}
		warning := "could not read the Kilo account's organizations (authentication failed, HTTP 401); using the personal account"
		if n := countContaining(f.seenNotify, warning); n != 1 || k.requests() != 1 || len(k.profileTokens()) != 1 {
			t.Errorf("notices %q, %d requests; want the menu's warning once, and only its one profile read", f.seenNotify, k.requests())
		}
		if saves := store.recorded(); len(saves) != 1 {
			t.Errorf("saves %+v; want the session saved once", saves)
		}
	})
}

// anyHost0080 answers every request with a listing and records its URL.
type anyHost0080 struct {
	mu   sync.Mutex
	seen []string
}

func (a *anyHost0080) RoundTrip(r *http.Request) (*http.Response, error) {
	a.mu.Lock()
	a.seen = append(a.seen, r.URL.Host+r.URL.Path)
	a.mu.Unlock()
	return respond0080(r, http.StatusOK, `{"data":[{"id":"model-0080"}]}`), nil
}

// TestKiloVerify_OtherProvidersReadNoProfile_0080 (D3: no other provider's
// check changes): OpenCode Zen and Hugging Face keys under Discover send no
// request to a profile.
func TestKiloVerify_OtherProvidersReadNoProfile_0080(t *testing.T) {
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderOpencodeZen, llmprovider.ProviderHuggingFace} {
		t.Run(string(id), func(t *testing.T) {
			hosts := &anyHost0080{}
			f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, secrets: []string{"fake-other-key-0080"}})
			if _, _, err := ResolveCredential(context.Background(), f, Options{Discover: true,
				HTTPClient:      &http.Client{Transport: hosts},
				ProviderOptions: []llmprovider.Option{testCache(t)}}, id); err != nil {
				t.Fatalf("ResolveCredential: %v", err)
			}
			hosts.mu.Lock()
			defer hosts.mu.Unlock()
			if len(hosts.seen) == 0 {
				t.Fatalf("no request was sent; want the listing")
			}
			for _, seen := range hosts.seen {
				if strings.HasSuffix(seen, "/api/profile") || strings.Contains(seen, "kilo") {
					t.Errorf("request to %q; want no Kilo profile read for %s", seen, id)
				}
			}
		})
	}
}

// TestKiloVerify_RequestNamed_0080 (D3; 0079-MADR D5; 0020-MADR F20): the
// profile request carries Options.Identity's User-Agent, the bearer, and goes
// to the endpoint in use, not Kilo's own.
func TestKiloVerify_RequestNamed_0080(t *testing.T) {
	k := newKilo0080(kiloGood0080)
	o := k.options(t)
	o.Identity = auth.Identity{Name: "probe-app", Version: "1.2.3"}
	f := newFake(t, fakePrompter{inputs: []string{"https://kilo.example.test/api/gateway"}, secrets: []string{kiloGood0080}})
	if _, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo); err != nil {
		t.Fatalf("ResolveCredential: %v", err)
	}
	reads := k.profileReads()
	if len(reads) != 1 {
		t.Fatalf("%d profile requests; want one", len(reads))
	}
	if got := reads[0]; !strings.HasPrefix(got.agent, "probe-app/1.2.3 (") || got.host != "kilo.example.test" ||
		got.path != "/api/profile" || got.token != kiloGood0080 {
		t.Errorf("profile request: User-Agent %q, host %q, path %q, bearer is the key %t; want the application's name, the endpoint's host, /api/profile, the key",
			got.agent, got.host, got.path, got.token == kiloGood0080)
	}
}
