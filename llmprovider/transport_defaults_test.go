package llmprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// proxyChildEnv marks the child process TestDefaultClient_HonoursProxy runs.
const proxyChildEnv = "LLMPROVIDER_TEST_PROXY_CHILD"

// proxyVariables are cleared from the child's environment before the test's
// own proxy is set, so the host's settings cannot leak in.
var proxyVariables = []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY", "REQUEST_METHOD"}

// TestDefaultClient_HonoursProxy (0016-PLAN T1 step 2): every provider built
// without WithHTTPClient reaches its service through HTTP_PROXY. net/http reads
// the proxy variables once per process, so the providers run in a child
// process that starts with them set. Each provider's base URL is a
// non-loopback host that does not resolve, because loopback is never proxied:
// only a proxied request can succeed.
func TestDefaultClient_HonoursProxy(t *testing.T) {
	if os.Getenv(proxyChildEnv) != "" {
		proxyChild(t)
		return
	}
	var mu sync.Mutex
	var hosts []string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hosts = append(hosts, r.Host)
		mu.Unlock()
		name := strings.TrimSuffix(r.Host, ".invalid")
		for _, c := range wireCases {
			if c.name == name {
				reply := c.reply(r)
				w.WriteHeader(max(reply.Status, http.StatusOK))
				_, _ = w.Write([]byte(reply.Body))
				return
			}
		}
		http.Error(w, "unknown host "+r.Host, http.StatusBadGateway)
	}))
	t.Cleanup(proxy.Close)

	cmd := exec.Command(os.Args[0], "-test.run=^TestDefaultClient_HonoursProxy$", "-test.count=1")
	cmd.Env = append(withoutProxyVariables(os.Environ()), proxyChildEnv+"=1", "HTTP_PROXY="+proxy.URL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("providers did not reach the proxy: %v\n%s", err, out)
	}
	mu.Lock()
	defer mu.Unlock()
	for _, c := range wireCases {
		if !slices.Contains(hosts, c.name+".invalid") {
			t.Errorf("%s: no request reached the proxy (saw %v)", c.name, hosts)
		}
	}
}

// proxyChild generates once through every wire case, each aimed at
// <case>.invalid.
func proxyChild(t *testing.T) {
	for _, c := range wireCases {
		p, err := c.build("http://" + c.name + ".invalid")
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		got, err := p.Generate(ctx, wirePrompt)
		cancel()
		if err != nil || got != "hello" {
			t.Errorf("%s: Generate = %q, %v; want \"hello\" through the proxy", c.name, got, err)
		}
	}
}

func withoutProxyVariables(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if !slices.Contains(proxyVariables, strings.ToUpper(name)) {
			out = append(out, kv)
		}
	}
	return out
}

// TestShareHTTPClient_KeepsTheSessionsOwn: a session that already has a
// client keeps it; anything but an OAuth session is left alone.
func TestShareHTTPClient_KeepsTheSessionsOwn(t *testing.T) {
	own, provider := &http.Client{}, &http.Client{}
	session := &OAuthSession{HTTPClient: own}
	ShareHTTPClient(session, provider)
	if session.HTTPClient != own {
		t.Error("a session's own client was replaced")
	}
	bare := &OAuthSession{}
	ShareHTTPClient(bare, nil)
	if bare.HTTPClient != nil {
		t.Error("a nil client was shared")
	}
	ShareHTTPClient(NewStaticToken("k"), provider) // must not panic
}
