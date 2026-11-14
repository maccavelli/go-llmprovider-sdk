package signin

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

const (
	planConsentID0075   = "chatgpt_plan.background_use"
	planConsentURL0075  = "https://chatgpt.com/settings/usage"
	planConsentText0075 = "You're using your ChatGPT plan: this app's requests count toward its usage, " +
		"which you can manage on the usage page. " +
		"Allow this app to also use your plan for background requests it starts on its own?"
)

// answerConsent reads the plan's confirm step, which comes first, and
// answers yes, so that the sign-in goes on to its open_url as step 2.
func answerConsent(t *testing.T, run *SignIn) {
	t.Helper()
	step := nextStep(t, run)
	wantStep(t, step, authcontract.StepConfirm, 1)
	yes := true
	if err := run.Answer(background(), authcontract.Answer{Seq: step.Seq, Confirm: &yes}); err != nil {
		t.Fatalf("Answer to the consent = %v", err)
	}
}

// TestMethods_Consent_0075 (0075-MADR D1): the plan's method has its consent,
// required, with the usage link and the consent feature; no other method has
// one.
func TestMethods_Consent_0075(t *testing.T) {
	usage := authcontract.Link{Rel: authcontract.LinkUsage, URL: planConsentURL0075}
	for _, d := range builtinDescriptors() {
		for _, m := range Methods(d) {
			plan := d.ID == llmprovider.ProviderOpenAI && m.ID == string(llmprovider.AuthChatGPTPlan)
			if !plan {
				if len(m.Consents) != 0 || len(m.Links) != 0 {
					t.Errorf("%s %s has consents %+v and links %+v; want none", d.ID, m.ID, m.Consents, m.Links)
				}
				continue
			}
			if len(m.Consents) != 1 || m.Consents[0].ID != planConsentID0075 || !m.Consents[0].Required ||
				m.Consents[0].Text != planConsentText0075 || len(m.Consents[0].Links) != 1 || m.Consents[0].Links[0] != usage {
				t.Errorf("the plan's consents = %+v; want %s, required, with its text and the usage link", m.Consents, planConsentID0075)
			}
			if len(m.Links) != 1 || m.Links[0] != usage {
				t.Errorf("the plan's links = %+v; want the usage link", m.Links)
			}
			want := []authcontract.Feature{authcontract.FeaturePaste, authcontract.FeatureConsent}
			if len(m.Features) != 2 || m.Features[0] != want[0] || m.Features[1] != want[1] {
				t.Errorf("the plan's features = %v; want %v", m.Features, want)
			}
		}
	}
}

// TestPlanConfirmStep_0075 (0075-MADR D4): a first sign-in's steps begin
// confirm, then open_url; the step's id, text, link and default (false); the
// app's text when given.
func TestPlanConfirmStep_0075(t *testing.T) {
	for _, c := range []struct {
		name string
		text map[string]string
		want string
	}{
		{"the method's text", nil, planConsentText0075},
		{"the app's text", map[string]string{planConsentID0075: "My App may also work in the background."}, "My App may also work in the background."},
	} {
		t.Run(c.name, func(t *testing.T) {
			probe := newLoginProbe()
			run, err := startWith(background(), llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
				Options{AgentName: "My App", ConsentText: c.text}, probe.logins())
			if err != nil {
				t.Fatalf("start = %v", err)
			}
			confirm := nextStep(t, run)
			wantStep(t, confirm, authcontract.StepConfirm, 1)
			if confirm.ConfirmID != planConsentID0075 || confirm.Text != c.want || confirm.Default == nil || *confirm.Default ||
				len(confirm.Links) != 1 || confirm.Links[0].Rel != authcontract.LinkUsage || confirm.Links[0].URL != planConsentURL0075 {
				t.Errorf("confirm step = %+v; want id %s, text %q, default false and the usage link", confirm, planConsentID0075, c.want)
			}
			yes := true
			if err := run.Answer(background(), authcontract.Answer{Seq: confirm.Seq, Confirm: &yes}); err != nil {
				t.Fatalf("Answer = %v", err)
			}
			wantStep(t, nextStep(t, run), authcontract.StepOpenURL, 2)
			close(probe.release)
			if _, err := run.Wait(bounded(t)); err != nil {
				t.Fatalf("Wait = %v", err)
			}
		})
	}
}
