package signinflow

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

const (
	planConsentID0075   = "chatgpt_plan.background_use"
	planConsentText0075 = "the method's default wording"
)

// planConsentMethod is the plan method with its consent (0075-MADR D1).
func planConsentMethod() authcontract.Method {
	return authcontract.Method{ID: "chatgpt_plan", Kind: authcontract.KindBrowser, Completion: authcontract.CompletionBrowserPaste,
		Paste: authcontract.PasteURLOrCode, FirstPaste: authcontract.PasteURLOnly, Yields: authcontract.YieldsOAuthSession,
		Consents: []authcontract.Consent{{ID: planConsentID0075, Required: true, Text: planConsentText0075,
			Links: []authcontract.Link{{Rel: authcontract.LinkUsage, URL: "https://usage.test/"}}}}}
}

// planLogins is a plan login that counts its calls, shows an authorize URL
// and ends when released, with the session or the error it is given.
type planLogins0075 struct {
	release chan struct{}
	calls   int
	session *auth.OAuthSession
	err     error
}

func newPlanLogins0075() *planLogins0075 {
	return &planLogins0075{release: make(chan struct{}), session: &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "access-0123456789"}}
}

func (p *planLogins0075) logins() Logins {
	return Logins{Plan: func(ctx context.Context, o auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		p.calls++
		_ = o.OpenURL("https://issuer.test/oauth/authorize?state=" + testState)
		select {
		case <-p.release:
			return p.session, p.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}
}

func startPlan0075(t *testing.T, probe *planLogins0075, previous *auth.OAuthSession) *Run {
	t.Helper()
	run, err := Start(background(), Spec{Provider: llmprovider.ProviderOpenAI, Method: planConsentMethod(), Mode: ModePlan,
		First: previous == nil, Plan: auth.ChatGPTPlanOptions{Previous: previous}, Logins: probe.logins()})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	return run
}

// answerConfirm0075 reads the confirm step and answers it.
func answerConfirm0075(t *testing.T, run *Run, answer *bool) authcontract.Step {
	t.Helper()
	step := next(t, run)
	if step.Kind != authcontract.StepConfirm || step.Seq != 1 {
		t.Fatalf("first step = %s #%d; want confirm #1", step.Kind, step.Seq)
	}
	if err := run.Answer(background(), authcontract.Answer{Seq: step.Seq, Confirm: answer}); err != nil {
		t.Fatalf("Answer = %v", err)
	}
	return step
}

// TestPlanConsentAnswer_0075 (0075-MADR D4): true gives granted on the
// session Wait returns, false and an answer with no Confirm give declined,
// and both go on to open_url.
func TestPlanConsentAnswer_0075(t *testing.T) {
	yes, no := true, false
	for _, c := range []struct {
		name   string
		answer *bool
		want   string
	}{
		{"true", &yes, planConsentID0075 + "=granted"},
		{"false", &no, planConsentID0075 + "=declined"},
		{"no Confirm", nil, planConsentID0075 + "=declined"},
	} {
		t.Run(c.name, func(t *testing.T) {
			probe := newPlanLogins0075()
			run := startPlan0075(t, probe, nil)
			answerConfirm0075(t, run, c.answer)
			if open := next(t, run); open.Kind != authcontract.StepOpenURL || open.Seq != 2 {
				t.Fatalf("step after the answer = %s #%d; want open_url #2", open.Kind, open.Seq)
			}
			close(probe.release)
			result, err := run.Wait(bounded(t))
			if err != nil || result.Session == nil || result.Session.Consents != c.want || probe.calls != 1 {
				t.Errorf("Wait = %+v, %v after %d login(s); want Consents %q and one login", result, err, probe.calls, c.want)
			}
		})
	}
}

// TestPlanConsentCarried_0075 (0075-MADR D4): a re-sign-in whose Previous has
// an answer shows no confirm and carries it; one without an answer shows
// it.
func TestPlanConsentCarried_0075(t *testing.T) {
	for _, want := range []string{planConsentID0075 + "=granted", planConsentID0075 + "=declined"} {
		probe := newPlanLogins0075()
		run := startPlan0075(t, probe, &auth.OAuthSession{ClientID: "client-0075", Consents: want})
		if open := next(t, run); open.Kind != authcontract.StepOpenURL || open.Seq != 1 {
			t.Fatalf("first step = %s #%d; want open_url #1, no confirm, for a Previous answering %q", open.Kind, open.Seq, want)
		}
		close(probe.release)
		result, err := run.Wait(bounded(t))
		if err != nil || result.Session == nil || result.Session.Consents != want {
			t.Errorf("Wait = %+v, %v; want Consents %q carried", result, err, want)
		}
	}
	probe := newPlanLogins0075()
	run := startPlan0075(t, probe, &auth.OAuthSession{ClientID: "client-0075"})
	yes := true
	answerConfirm0075(t, run, &yes)
	run.Cancel()
	_, _ = run.Wait(bounded(t))
}

// TestPlanConsentNotRecordedOnFailure_0075 (0075-MADR D4, Q4): a sign-in
// that fails after the answer puts the answer on no session.
func TestPlanConsentNotRecordedOnFailure_0075(t *testing.T) {
	probe := newPlanLogins0075()
	probe.err = errors.New("the issuer refused")
	run := startPlan0075(t, probe, nil)
	yes := true
	answerConfirm0075(t, run, &yes)
	close(probe.release)
	result, err := run.Wait(bounded(t))
	if err == nil {
		t.Fatal("Wait = nil; want the login's error")
	}
	if result.Session != nil && result.Session.Consents != "" {
		t.Errorf("a failed sign-in's session holds Consents %q; want none recorded", result.Session.Consents)
	}
}
