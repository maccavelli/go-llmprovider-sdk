package localonly

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// TestGuard (0065-MADR D2) runs TestGuardChild in a process of its own:
// net/http reads the proxy variables once a process, at its first proxied
// request, so only a fresh process sees a guard's.
func TestGuard(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestGuardChild$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), "LOCALONLY_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "--- PASS: TestGuardChild") {
		t.Fatalf("TestGuardChild: %v\n%s", err, out)
	}
}

// TestGuardChild, in TestGuard's process: through net/http's environment
// proxy, a request to a host off the machine, plain or TLS, is refused and
// recorded; one to a loopback server is served and not recorded; Stop names
// each host once, with its count.
func TestGuardChild(t *testing.T) {
	if os.Getenv("LOCALONLY_CHILD") != "1" {
		t.Skip("run by TestGuard, in a process of its own")
	}
	g, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) }))
	defer local.Close()
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	resp, err := client.Get(local.URL)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("loopback: %v, %v; want it served", resp, err)
	}
	_ = resp.Body.Close()
	for range 2 {
		resp, err := client.Get("http://off-machine.invalid/x")
		if err != nil || resp.StatusCode != http.StatusBadGateway {
			t.Fatalf("plain request off the machine: %v, %v; want the guard's 502", resp, err)
		}
		_ = resp.Body.Close()
	}
	if resp, err := client.Get("https://off-machine.invalid/x"); err == nil {
		_ = resp.Body.Close()
		t.Fatal("TLS request off the machine: a response; want the CONNECT refused")
	}
	hosts := g.Stop()
	if want := []string{"off-machine.invalid (2)", "off-machine.invalid:443 (1)"}; !slices.Equal(hosts, want) {
		t.Errorf("Stop = %q, want %q", hosts, want)
	}
}

// run is a stand-in for *testing.M: its Run calls fn and returns code.
type run struct {
	fn   func()
	code int
}

func (r run) Run() int {
	r.fn()
	return r.code
}

// TestMain_Runs (0065-MADR D2): Main returns a run's code when it reached no
// host off the machine, and 1 for a passing run that reached one. net/http
// may have read the proxy variables already in this process, so the run
// here names Main's own proxy, from the variable Main's Start set.
func TestMain_Runs(t *testing.T) {
	if got := Main(run{fn: func() {}, code: 3}); got != 3 {
		t.Errorf("a run that reached nothing: %d, want 3", got)
	}
	leak := func() {
		proxy, err := url.Parse(os.Getenv("HTTPS_PROXY"))
		if err != nil {
			t.Fatal(err)
		}
		client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxy)}}
		if resp, err := client.Get("http://off-machine.invalid/"); err == nil {
			_ = resp.Body.Close()
		}
	}
	if got := Main(run{fn: leak, code: 0}); got != 1 {
		t.Errorf("a passing run that reached a host: %d, want 1", got)
	}
}

// TestExitCode_Report: a passing run that reached a host fails; a failing run
// keeps its code; a run that reached none keeps its code; the report names
// each host and the record.
func TestExitCode_Report(t *testing.T) {
	hosts := []string{"models.opencode.ai:443 (3)"}
	for _, c := range []struct {
		code  int
		hosts []string
		want  int
	}{{0, hosts, 1}, {2, hosts, 2}, {0, nil, 0}, {1, nil, 1}} {
		if got := ExitCode(c.code, c.hosts); got != c.want {
			t.Errorf("ExitCode(%d, %q) = %d, want %d", c.code, c.hosts, got, c.want)
		}
	}
	if r := Report(hosts); !strings.Contains(r, "models.opencode.ai:443 (3)") || !strings.Contains(r, "0065-MADR") {
		t.Errorf("Report = %q, want the host and the record", r)
	}
}
