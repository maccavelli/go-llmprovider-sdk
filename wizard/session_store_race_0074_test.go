package wizard

import (
	"context"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestSession_SaveBindsStoreWhileUsed_0074 (0074-PLAN Phase 7 step 6, the
// report's U14): the credential step binds a new session to its store while
// another goroutine uses the session. The listing here starts a goroutine of
// its own that outlives it, forces a refresh with Invalidate and Token in a
// loop, and so reads the session's Store while save sets it. Run under
// -race: save must set the store through the session's lock (UseStore), not
// by assigning the field. The store is a FileTokenStore, which reads a
// session under the session's lock as every store must, so that a report
// here is the binding's.
func TestSession_SaveBindsStoreWhileUsed_0074(t *testing.T) {
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	inner := client0059(http.StatusOK, http.StatusOK, nil)
	var session **auth.OAuthSession
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var once sync.Once
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/oauth/token") {
			once.Do(func() {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for {
						select {
						case <-stop:
							return
						default:
						}
						(*session).Invalidate() // forces a refresh, which reads Store
						_, _ = (*session).Token(context.Background())
						runtime.Gosched()
					}
				}()
			})
		}
		return inner.Transport.RoundTrip(r)
	})}
	t.Cleanup(func() { close(stop); wg.Wait() })
	session = browserGrok0059(t, client, true)
	if _, err := grok0059(t, store, client, true, 0); err != nil {
		t.Fatal(err)
	}
}
