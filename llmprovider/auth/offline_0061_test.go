package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestRefresh_RefusedDialIsOffline_0061 (0061-MADR D2): a refresh whose dial
// is refused matches ErrOffline, and still ErrProviderUnavailable.
func TestRefresh_RefusedDialIsOffline_0061(t *testing.T) {
	srv, _ := refreshServer(t, func(_ int32, _ http.ResponseWriter, _ *http.Request) {})
	session := expiredSession(srv, DefaultOpenAIIssuer)
	session.TokenURL, session.HTTPClient = nettest.RefusedURL, nettest.Refusing()
	_, err := session.Token(context.Background())
	if !errors.Is(err, llmprovider.ErrOffline) || !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("Token = %v; want ErrOffline and ErrProviderUnavailable", err)
	}
}

// TestSignInRequest_RefusedDialIsOffline_0061 (0061-MADR D2): a sign-in
// request whose dial is refused matches ErrOffline.
func TestSignInRequest_RefusedDialIsOffline_0061(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, nettest.RefusedURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := doBounded(nettest.Refusing(), req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if !errors.Is(err, llmprovider.ErrOffline) {
		t.Errorf("doBounded = %v; want ErrOffline", err)
	}
}
