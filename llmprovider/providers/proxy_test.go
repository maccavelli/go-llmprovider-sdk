package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// proxyChildEnv marks the child process TestDefaultClient_HonoursProxy runs:
// it names the marker file the parent wrote.
const proxyChildEnv = "LLMPROVIDER_TEST_PROXY_CHILD"

// proxyMark is what the parent writes to the marker proxyChildEnv names,
// which proves the parent launched the child (0090-MADR T13).
const proxyMark = "launched by TestDefaultClient_HonoursProxy"

// proxyLaunched reports whether path is a marker the parent wrote: a value
// left set in a shell names none, so it is ignored.
func proxyLaunched(path string) bool {
	if path == "" {
		return false
	}
	raw, err := os.ReadFile(filepath.Clean(path))
	return err == nil && string(raw) == proxyMark
}

// proxyVariables are cleared from the child's environment before the test's
// own proxy is set, so the host's settings cannot leak in.
var proxyVariables = []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY", "REQUEST_METHOD"}

// TestDefaultClient_HonoursProxy (0016-PLAN T1 step 2): every provider in
// Default, built without WithHTTPClient, reaches its service through
// HTTP_PROXY. It was llmprovider's test over its G-wire cases, which lost each
// provider as 0015-PLAN S7 moved it; over Default it covers every built-in.
// net/http reads the proxy variables once per process, so the providers run
// in a child process that starts with them set. Each provider's base URL is a
// non-loopback host that does not resolve, because loopback is never proxied:
// only a proxied request can succeed. The proxy answers with G-wire's canned
// reply for the request's path, which serves every wire format.
func TestDefaultClient_HonoursProxy(t *testing.T) {
	if proxyLaunched(os.Getenv(proxyChildEnv)) {
		proxyChild(t)
		return
	}
	var mu sync.Mutex
	var hosts []string
	proxy := httptest.NewServer(wiretest.StreamReplies(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hosts = append(hosts, r.Host)
		mu.Unlock()
		reply := wirecase.Case{}.Reply(r)
		w.WriteHeader(max(reply.Status, http.StatusOK))
		_, _ = w.Write([]byte(reply.Body))
	})))
	t.Cleanup(proxy.Close)

	marker := filepath.Join(t.TempDir(), "parent-0090")
	if err := os.WriteFile(marker, []byte(proxyMark), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDefaultClient_HonoursProxy$", "-test.count=1")
	cmd.Env = append(withoutProxyVariables(os.Environ()), proxyChildEnv+"="+marker, "HTTP_PROXY="+proxy.URL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("providers did not reach the proxy: %v\n%s", err, out)
	}
	mu.Lock()
	defer mu.Unlock()
	descs := Default().Descriptors()
	if len(descs) == 0 {
		t.Fatal("Default holds no provider")
	}
	for _, d := range descs {
		if !slices.Contains(hosts, string(d.ID)+".invalid") {
			t.Errorf("%s: no request reached the proxy (saw %v)", d.ID, hosts)
		}
	}
}

// TestDefaultClient_HonoursProxy_StrayValue_0090 (0090-MADR T13):
// proxyChildEnv left set in a shell, to a value no parent wrote, does not
// send TestDefaultClient_HonoursProxy down its child part: the test runs as
// the parent, through a proxy of its own, and the proxy the shell's
// HTTP_PROXY names gets no request.
func TestDefaultClient_HonoursProxy_StrayValue_0090(t *testing.T) {
	var mu sync.Mutex
	var reached int
	// It answers as the test's own proxy does, so that a child part it
	// serves passes quickly and only the count tells.
	stray := httptest.NewServer(wiretest.StreamReplies(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reached++
		mu.Unlock()
		reply := wirecase.Case{}.Reply(r)
		w.WriteHeader(max(reply.Status, http.StatusOK))
		_, _ = w.Write([]byte(reply.Body))
	})))
	t.Cleanup(stray.Close)
	cmd := exec.Command(os.Args[0], "-test.run=^TestDefaultClient_HonoursProxy$", "-test.count=1")
	cmd.Env = append(withoutProxyVariables(os.Environ()), proxyChildEnv+"=1", "HTTP_PROXY="+stray.URL)
	out, err := cmd.CombinedOutput()
	mu.Lock()
	defer mu.Unlock()
	if reached != 0 {
		t.Errorf("the stray value ran the child part: %d request(s) reached the shell's proxy; output:\n%s", reached, out)
	}
	if err != nil {
		t.Errorf("TestDefaultClient_HonoursProxy under a stray value: %v; want it to pass as the parent; output:\n%s", err, out)
	}
}

// proxyChild generates once through every provider in Default, each aimed at
// <id>.invalid.
func proxyChild(t *testing.T) {
	for _, d := range Default().Descriptors() {
		model := "test-model"
		if len(d.StaticModels) > 0 {
			model = d.StaticModels[0]
		}
		p, err := New(d.ID, llmprovider.WithAPIKey("proxy-key"), llmprovider.WithModel(model),
			llmprovider.WithBaseURL("http://"+string(d.ID)+".invalid"))
		if err != nil {
			t.Fatalf("%s: %v", d.ID, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		got, err := llmprovider.GenerateText(ctx, p, &llmprovider.Request{Input: []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: wirecase.Prompt}}})
		cancel()
		if err != nil || got != "hello" {
			t.Errorf("%s: GenerateText = %q, %v; want \"hello\" through the proxy", d.ID, got, err)
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
