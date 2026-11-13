package catalog

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestListRawOriginator_0079: a ChatGPT session's raw listing sends
// WithClientInfo's name as its originator, else this module's (0079-MADR D1).
func TestListRawOriginator_0079(t *testing.T) {
	body := []byte(`{"models":[{"slug":"gpt-low","visibility":"list"}]}`)
	for name, tc := range map[string]struct {
		opts []llmprovider.Option
		want string
	}{
		"named":      {[]llmprovider.Option{llmprovider.WithClientInfo("probe-app", "1.2.3")}, "probe-app"},
		"unnamed":    {nil, "go-llmprovider-sdk"},
		"empty name": {[]llmprovider.Option{llmprovider.WithClientInfo("", "1.2.3")}, "go-llmprovider-sdk"},
	} {
		s := newPagedServer(t, body)
		opts := append([]llmprovider.Option{llmprovider.WithBaseURL(s.URL)}, tc.opts...)
		if _, err := ListRaw(context.Background(), llmprovider.ProviderOpenAI, chatGPTSession(), opts...); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		sent := s.sent()
		if len(sent) != 1 {
			t.Fatalf("%s: %d requests; want one", name, len(sent))
		}
		if got := sent[0].Header.Get("originator"); got != tc.want {
			t.Errorf("%s: originator = %q; want %q", name, got, tc.want)
		}
	}
}
