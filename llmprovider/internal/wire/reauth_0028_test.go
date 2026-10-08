package wire

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestReauth_RegionForbiddenNotRenewed (0028-MADR D-H2): OpenAI's region 403,
// as ClassifyHTTPError classifies it, says nothing about the key: Reauth
// neither renews the credential nor sends again.
func TestReauth_RegionForbiddenNotRenewed(t *testing.T) {
	body := `{"error":{"type":"request_forbidden","code":"unsupported_country_region_territory","message":"Country, region, or territory not supported"}}`
	src := &countingSource{}
	sends := 0
	_, _ = Reauth(context.Background(), "openai", src, func(llmprovider.Token) (int, error) {
		sends++
		return 0, llmprovider.ClassifyHTTPError("openai", &http.Response{StatusCode: http.StatusForbidden,
			Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))})
	})
	if sends != 1 || src.invalidated != 0 {
		t.Fatalf("sends = %d, invalidations = %d; want 1 and 0", sends, src.invalidated)
	}
}
