package llmprovider

import "net/http"

// closeResponseBody closes a test's response body. The library has none of
// its own: no library code in this package reads a body (0015-PLAN S10).
func closeResponseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}
