//go:build live_gateways

package auth_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestLive_KiloProfile_0080 measures which kinds of Kilo credential the
// profile endpoint accepts (0080-MADR D1; 0080-PLAN Phase 1). The owner runs
// it, never an agent. It REQUIRES LLMPROVIDER_LIVE_0080=1, and each subtest
// skips without its own variable:
//
//   - DashboardKey: KILO_API_KEY.
//   - OrganizationToken: LLMPROVIDER_LIVE_0080_ORG_KEY and
//     LLMPROVIDER_LIVE_0080_ORG.
//   - RevokedKey: LLMPROVIDER_LIVE_0080_REVOKED_KEY.
//
// Nothing here logs a key, an error's text, a response body or a field that
// identifies the account. A result is a fixed word, an HTTP status and
// counts. The subtests call kiloProfileVerdict and kiloListingVerdict, which
// the offline stand-in proof drives against a local server.
func TestLive_KiloProfile_0080(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0080") != "1" {
		t.Skip("LLMPROVIDER_LIVE_0080 unset")
	}
	t.Run("DashboardKey", func(t *testing.T) {
		key := os.Getenv("KILO_API_KEY")
		if key == "" {
			t.Skip("KILO_API_KEY unset")
		}
		ctx, cancel := kiloLiveCtx()
		defer cancel()
		v := kiloProfileVerdict(ctx, key, kiloTestOpts...)
		logProfileVerdict(t, v)
		if v.word != verdictAccepted {
			t.Errorf("the profile did not accept the dashboard key: %s, HTTP %d", v.word, v.status)
		}
	})
	t.Run("OrganizationToken", func(t *testing.T) {
		key := os.Getenv("LLMPROVIDER_LIVE_0080_ORG_KEY")
		org := os.Getenv("LLMPROVIDER_LIVE_0080_ORG")
		if key == "" || org == "" {
			t.Skip("LLMPROVIDER_LIVE_0080_ORG_KEY or LLMPROVIDER_LIVE_0080_ORG unset")
		}
		ctx, cancel := kiloLiveCtx()
		defer cancel()
		p := kiloProfileVerdict(ctx, key, kiloTestOpts...)
		logProfileVerdict(t, p)
		l := kiloListingVerdict(ctx, key, org, kiloTestOpts...)
		t.Logf("organization listing: %s, HTTP %d", l.word, l.status)
		if p.word != verdictAccepted {
			t.Errorf("the profile did not accept the organization token: %s, HTTP %d", p.word, p.status)
		}
		if l.word != verdictAccepted {
			t.Errorf("the organization listing did not accept the token: %s, HTTP %d", l.word, l.status)
		}
	})
	t.Run("RevokedKey", func(t *testing.T) {
		key := os.Getenv("LLMPROVIDER_LIVE_0080_REVOKED_KEY")
		if key == "" {
			t.Skip("LLMPROVIDER_LIVE_0080_REVOKED_KEY unset")
		}
		ctx, cancel := kiloLiveCtx()
		defer cancel()
		v := kiloProfileVerdict(ctx, key, kiloTestOpts...)
		logProfileVerdict(t, v)
		if v.word != verdictAuthFailure {
			t.Errorf("the profile did not refuse the revoked key with ErrAuthFailure (a 403 is ErrNotPermitted, not a refusal): %s, HTTP %d", v.word, v.status)
		}
	})
}

// Verdict words. They are fixed strings: the only text a result logs.
const (
	verdictAccepted     = "accepted"
	verdictAuthFailure  = "ErrAuthFailure"
	verdictNotPermitted = "ErrNotPermitted"
	verdictOtherRefusal = "other error"
)

// kiloVerdict is the whole of what a call reports: a fixed word, an HTTP
// status (0 when the error carried none) and counts. It holds no error, no
// key, no body and no account field.
type kiloVerdict struct {
	word     string
	status   int
	orgs     int
	personal bool
}

// kiloTestOpts is nil in a live run, so every call goes to the origin the SDK
// uses. An offline stand-in sets it to move the calls to a local server.
var kiloTestOpts []llmprovider.Option

func kiloLiveCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 90*time.Second)
}

// kiloProfileVerdict reads the profile with key. opts exist so a stand-in
// server can replace the origin; the live subtests pass none.
func kiloProfileVerdict(ctx context.Context, key string, opts ...llmprovider.Option) kiloVerdict {
	acct, err := auth.KiloProfile(ctx, key, opts...)
	if err != nil {
		return classifyKiloError(err)
	}
	return kiloVerdict{word: verdictAccepted, orgs: len(acct.Organizations), personal: acct.HasPersonalAccount}
}

// kiloListingVerdict lists the organization's models with the token, raw, so a
// refusal is returned and not degraded to a fallback catalog.
func kiloListingVerdict(ctx context.Context, key, org string, opts ...llmprovider.Option) kiloVerdict {
	all := append([]llmprovider.Option{catalog.WithKiloOrganization(org)}, opts...)
	if _, err := catalog.ListRaw(ctx, llmprovider.ProviderKilo, llmprovider.NewStaticToken(key), all...); err != nil {
		return classifyKiloError(err)
	}
	return kiloVerdict{word: verdictAccepted}
}

// classifyKiloError reduces err to the sentinel it matches, by name, and its
// HTTP status. It never reads err.Error(): that text can carry a URL, a
// header or a body.
func classifyKiloError(err error) kiloVerdict {
	v := kiloVerdict{word: verdictOtherRefusal}
	switch {
	// ErrNotPermitted is tested first: a 403 matches both sentinels (the
	// legacy status mapping makes it an auth failure too), and a 403 is not a
	// refusal of the credential (0080-MADR Q4), as the wizard's
	// refusedCredential also reads it.
	case errors.Is(err, llmprovider.ErrNotPermitted):
		v.word = verdictNotPermitted
	case errors.Is(err, llmprovider.ErrAuthFailure):
		v.word = verdictAuthFailure
	}
	var api *llmprovider.APIError
	if errors.As(err, &api) {
		v.status = api.Status
	}
	return v
}

// logProfileVerdict logs a profile result: the word, the status and counts.
func logProfileVerdict(t *testing.T, v kiloVerdict) {
	t.Helper()
	if v.word == verdictAccepted {
		t.Logf("profile: %s, organizations %d, personal account %t", v.word, v.orgs, v.personal)
		return
	}
	t.Logf("profile: %s, HTTP %d", v.word, v.status)
}
