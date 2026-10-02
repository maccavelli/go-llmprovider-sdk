package llmprovider

import (
	"net/http"
	"testing"
	"time"
)

// TestResolveOptions_DefaultTimeout is the #5 regression: the default client
// must have a timeout (the old http.DefaultClient had none). MADR 0012 §1.3
// set it to 330 s.
func TestResolveOptions_DefaultTimeout(t *testing.T) {
	st, err := ResolveOptions(ProviderOpenAI, nil)
	if err != nil {
		t.Fatal(err)
	}
	if st.HTTPClient() == nil {
		t.Fatal("default HTTPClient is nil")
	}
	if st.HTTPClient().Timeout != 330*time.Second {
		t.Errorf("default timeout: got %v want 330s", st.HTTPClient().Timeout)
	}
}

func TestResolveOptions_WithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	st, err := ResolveOptions(ProviderOpenAI, []Option{WithHTTPClient(custom)})
	if err != nil || st.HTTPClient() != custom {
		t.Errorf("WithHTTPClient override not honored (err %v)", err)
	}
}
