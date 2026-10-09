// Package transport holds the HTTP plumbing every package in this module
// shares (0015-MADR D2, amendment "S7b's import graph"): the default client,
// the client identity and its User-Agent, Retry-After parsing, and the listing
// probe. It imports only the standard library, so llmprovider, auth, catalog
// and the provider packages can all import it. Turning a response into an
// *llmprovider.APIError is the error model's job, in llmprovider. Each package
// closes its own response bodies, so that bodyclose can see them closed.
package transport

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math"
	"net"
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// dialer bounds a connection attempt at 30 s, net/http's own default, so a
// blackholed host fails in 30 s rather than at the OS limit (0021-MADR T12).
var dialer = &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}

// DefaultClient returns an http.Client whose every phase is bounded, so a hung
// or non-responsive LLM endpoint can never block a caller indefinitely. The
// stdlib http.DefaultClient has no timeout and must not be used here. Its
// bounds follow the ChatGPT client (0021-MADR D2, amending MADR 0012 §1.3):
//
//   - a connection is made within 30 s;
//   - a generation may take 300 s to its first byte;
//   - there is no total timeout: a reply's body may take as long as it keeps
//     sending, and wire.Post ends it after 300 s with no data
//     (StreamIdleTimeout). Callers wanting less set a context deadline.
//
// Listings and auth requests keep their own bounds (10 s and 30 s). It
// honours HTTP_PROXY, HTTPS_PROXY and NO_PROXY, as net/http's own default
// transport does (0016-MADR D8). Setting a dialer turns off net/http's
// automatic HTTP/2, so ForceAttemptHTTP2 keeps it.
//
// Each call returns a new client, one per provider instance, over the one
// transport the process keeps for the default configuration, so instances
// share connections (0028-MADR D-A2). The client's transport does not
// implement CloseIdleConnections: one instance closing its idle connections
// does not close those the others would reuse.
func DefaultClient() *http.Client {
	return &http.Client{Transport: scoped{base: defaultManager.transport(defaultConfig)}}
}

// Config is one default transport's configuration. Two equal Configs share a
// transport; one that differs gets its own, and never shares connections
// with the others (0028-MADR D-A2).
type Config struct {
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	IdleConnTimeout       time.Duration
	MaxIdleConnsPerHost   int
	ForceAttemptHTTP2     bool
	// rootCAs, when set, replaces the system's roots: tests trust their own
	// server with it.
	rootCAs *x509.CertPool
}

// defaultConfig is DefaultClient's configuration. Tests set rootCAs.
var defaultConfig = Config{
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 300 * time.Second,
	IdleConnTimeout:       90 * time.Second,
	MaxIdleConnsPerHost:   16,
	ForceAttemptHTTP2:     true,
}

// manager keeps the process's default transports, one per Config, each
// created on first use and kept for the process.
type manager struct {
	mu         sync.Mutex
	transports map[Config]*http.Transport
}

// defaultManager is the process's manager.
var defaultManager = &manager{transports: map[Config]*http.Transport{}}

// transport returns cfg's transport, creating it on first use.
func (m *manager) transport(cfg Config) *http.Transport {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.transports[cfg]; ok {
		return t
	}
	t := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     cfg.ForceAttemptHTTP2,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
	}
	if cfg.rootCAs != nil {
		t.TLSClientConfig = &tls.Config{RootCAs: cfg.rootCAs, MinVersion: tls.VersionTLS12}
	}
	m.transports[cfg] = t
	return t
}

// drop closes cfg's transport's idle connections and forgets it; tests use it.
func (m *manager) drop(cfg Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.transports[cfg]; ok {
		t.CloseIdleConnections()
		delete(m.transports, cfg)
	}
}

// scoped is one client's view of a shared transport: it round-trips, and has
// no CloseIdleConnections, so http.Client.CloseIdleConnections leaves the
// shared connections open.
type scoped struct{ base *http.Transport }

// RoundTrip sends req on the shared transport.
func (s scoped) RoundTrip(req *http.Request) (*http.Response, error) { return s.base.RoundTrip(req) }

// Base is the shared transport, for tests that check its settings
// (0028-PLAN D15).
func (s scoped) Base() *http.Transport { return s.base }

// Client identification (MADR 0012 §1.4): every request names this module and
// the consuming application honestly, and never a reference client.
const (
	// DefaultClientName is the client name when the caller names none.
	DefaultClientName = "go-llmprovider-sdk"
	// DevelVersion is the version of a build with no module version.
	DevelVersion    = "(devel)"
	sdkModulePath   = "github.com/maccavelli/go-llmprovider-sdk"
	headerUserAgent = "User-Agent"
)

// BuildVersions reads the versions of this module and of the main module
// once. It is a variable so that a test can report another version.
var BuildVersions = sync.OnceValues(func() (sdk, main string) {
	return versionsOf(debug.ReadBuildInfo())
})

// versionsOf is BuildVersions' reading of info: this module's version, and
// the main module's.
func versionsOf(info *debug.BuildInfo, ok bool) (sdk, main string) {
	sdk, main = DevelVersion, DevelVersion
	if !ok {
		return sdk, main
	}
	if v := info.Main.Version; v != "" {
		main = v
	}
	if info.Main.Path == sdkModulePath {
		return main, main
	}
	for _, dep := range info.Deps {
		if dep.Path == sdkModulePath && dep.Version != "" {
			sdk = dep.Version
		}
	}
	return sdk, main
}

// Identity is who a client says it is, and the session it speaks in.
type Identity struct {
	Name, Version, Session string
}

// NewIdentity resolves an identity: an empty name is this module at its own
// version; a named application defaults to the main module's version; an
// empty session is a fresh random id.
func NewIdentity(name, version, session string) Identity {
	sdk, main := BuildVersions()
	id := Identity{Name: name, Version: version, Session: session}
	if id.Name == "" {
		id.Name = DefaultClientName
	}
	if id.Version == "" {
		id.Version = main
		if id.Name == DefaultClientName {
			id.Version = sdk
		}
	}
	if id.Session == "" {
		id.Session = rand.Text()
	}
	return id
}

// UserAgent is the identity's User-Agent: the application, the platform, and
// this module's version.
func (id Identity) UserAgent() string {
	sdk, _ := BuildVersions()
	return fmt.Sprintf("%s/%s (%s; %s) go-llmprovider-sdk/%s", id.Name, id.Version, runtime.GOOS, runtime.GOARCH, sdk)
}

// SetUserAgent names the client on req.
func (id Identity) SetUserAgent(req *http.Request) {
	req.Header.Set(headerUserAgent, id.UserAgent())
}

// ParseRetryAfter parses an HTTP Retry-After header (delta-seconds or
// HTTP-date), returning 0 when absent or unparseable.
func ParseRetryAfter(h string) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	if secs, err := strconv.ParseFloat(h, 64); err == nil && secs >= 0 && !math.IsInf(secs, 0) {
		return durationOf(secs, time.Second)
	}
	if t, err := http.ParseTime(h); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// RetryAfter reads a server-directed delay from retry-after-ms (fractional
// milliseconds), else Retry-After (MADR 0012 §1.2).
func RetryAfter(h http.Header) time.Duration {
	if ms, err := strconv.ParseFloat(strings.TrimSpace(h.Get("Retry-After-Ms")), 64); err == nil && ms >= 0 && !math.IsInf(ms, 0) {
		return durationOf(ms, time.Millisecond)
	}
	return ParseRetryAfter(h.Get("Retry-After"))
}

// Rate-limit reset headers, as the services were measured sending them
// (0028-PLAN Phase 1, T2): OpenAI's and Hugging Face's as Go durations,
// Claude's as RFC 3339 times. Gemini, Grok, Together and Kilo sent none.
var (
	resetDurationHeaders = []string{"X-Ratelimit-Reset-Requests", "X-Ratelimit-Reset-Tokens"}
	resetTimeHeaders     = []string{
		"Anthropic-Ratelimit-Requests-Reset", "Anthropic-Ratelimit-Tokens-Reset",
		"Anthropic-Ratelimit-Input-Tokens-Reset", "Anthropic-Ratelimit-Output-Tokens-Reset",
	}
)

// RateLimitReset is the longest wait the rate-limit reset headers name,
// measured from now, or 0 when none names a wait to come (0028-MADR D-H5). A
// value that does not parse, or names no time ahead, is ignored.
func RateLimitReset(h http.Header, now time.Time) time.Duration {
	var longest time.Duration
	for _, name := range resetDurationHeaders {
		if d, err := time.ParseDuration(strings.TrimSpace(h.Get(name))); err == nil && d > longest {
			longest = d
		}
	}
	for _, name := range resetTimeHeaders {
		if at, err := time.Parse(time.RFC3339, strings.TrimSpace(h.Get(name))); err == nil {
			if d := at.Sub(now); d > longest {
				longest = d
			}
		}
	}
	return longest
}

// durationOf is n units as a Duration, the longest one when n is too large.
// The conversion of a float beyond int64 is not defined: on amd64 it was
// negative, which WithRetry read as no delay (0020-MADR F31).
func durationOf(n float64, unit time.Duration) time.Duration {
	d := n * float64(unit)
	if d >= math.MaxInt64 {
		return math.MaxInt64
	}
	return time.Duration(d)
}
