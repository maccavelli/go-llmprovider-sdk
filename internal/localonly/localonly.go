// Package localonly keeps a package's tests on the machine (0065-MADR D2).
// Main runs them behind a proxy that refuses every request to a host off the
// machine and records it, and fails the run when any was recorded, naming
// each host, even when every test passed. Only a test's TestMain imports it.
//
// net/http's ProxyFromEnvironment never proxies a loopback host, so a test's
// httptest server is reached as before. A client whose transport has no
// Proxy, or a dial below net/http, is not seen.
package localonly

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

// Main runs m's tests behind a Guard, and returns the exit code: m's, or 1
// when the tests reached a host off the machine. m is a TestMain's
// *testing.M.
func Main(m interface{ Run() int }) int {
	g, err := Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "localonly: %v\n", err)
		return 1
	}
	code := m.Run()
	hosts := g.Stop()
	if len(hosts) > 0 {
		fmt.Fprintln(os.Stderr, Report(hosts))
	}
	return ExitCode(code, hosts)
}

// Guard is a running proxy that refuses every request with 502 and records
// its host.
type Guard struct {
	srv   *http.Server
	mu    sync.Mutex
	hosts map[string]int
}

// proxyVars are the variables net/http reads for a proxy, and its exemptions.
var proxyVars = []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"}

// Start listens on 127.0.0.1, points the process's proxy variables at the
// guard, and empties NO_PROXY. net/http reads the variables once, at its
// first proxied request, so Start must run before any.
func Start() (*Guard, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	g := &Guard{hosts: map[string]int{}}
	g.srv = &http.Server{Handler: http.HandlerFunc(g.refuse), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := g.srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "localonly: serve: %v\n", err)
		}
	}()
	url := "http://" + ln.Addr().String()
	for _, v := range proxyVars {
		if err := os.Setenv(v, url); err != nil {
			return nil, err
		}
	}
	for _, v := range []string{"NO_PROXY", "no_proxy"} {
		if err := os.Setenv(v, ""); err != nil {
			return nil, err
		}
	}
	return g, nil
}

// refuse records the request's host, a CONNECT's host:port or a plain
// request's, and answers 502.
func (g *Guard) refuse(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if r.Method != http.MethodConnect && r.URL.Host != "" {
		host = r.URL.Host
	}
	g.mu.Lock()
	g.hosts[host]++
	g.mu.Unlock()
	http.Error(w, "localonly: a test reached a host off the machine", http.StatusBadGateway)
}

// Stop closes the proxy and returns each host recorded, sorted, with its
// count: "models.opencode.ai:443 (3)".
func (g *Guard) Stop() []string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := g.srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "localonly: stop: %v\n", err)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]string, 0, len(g.hosts))
	for host, n := range g.hosts {
		out = append(out, fmt.Sprintf("%s (%d)", host, n))
	}
	slices.Sort(out)
	return out
}

// Report is the message a run that reached hosts off the machine prints.
func Report(hosts []string) string {
	return "localonly: the tests sent requests off the machine, to " + strings.Join(hosts, ", ") +
		"; a test serves its own replies, and turns the models.dev document off when it serves none (0065-MADR)"
}

// ExitCode is a run's exit code: code, or 1 when it was 0 and hosts were
// reached.
func ExitCode(code int, hosts []string) int {
	if code == 0 && len(hosts) > 0 {
		return 1
	}
	return code
}
