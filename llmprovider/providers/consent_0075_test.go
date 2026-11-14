package providers

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDescriptors_PlanConsent_0075 (0075-MADR D1): the plan's three consent
// fields, word for word; no other method has a consent.
func TestDescriptors_PlanConsent_0075(t *testing.T) {
	const (
		id   = "chatgpt_plan.background_use"
		text = "You're using your ChatGPT plan: this app's requests count toward its usage, " +
			"which you can manage on the usage page. " +
			"Allow this app to also use your plan for background requests it starts on its own?"
		url = "https://chatgpt.com/settings/usage"
	)
	seen := false
	for _, d := range Default().Descriptors() {
		for _, m := range d.AuthMethods {
			if d.ID == llmprovider.ProviderOpenAI && m.ID == llmprovider.AuthChatGPTPlan {
				seen = true
				if m.Consent != id || m.ConsentText != text || m.ConsentURL != url {
					t.Errorf("the plan's consent = %q, %q, %q; want %q, %q, %q", m.Consent, m.ConsentText, m.ConsentURL, id, text, url)
				}
				continue
			}
			if m.Consent != "" || m.ConsentText != "" || m.ConsentURL != "" {
				t.Errorf("%s %s has a consent (%q, %q, %q); want none", d.ID, m.ID, m.Consent, m.ConsentText, m.ConsentURL)
			}
		}
	}
	if !seen {
		t.Error("the OpenAI descriptor has no ChatGPT plan method")
	}
}
