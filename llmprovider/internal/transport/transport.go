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
	"fmt"
	"math"
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultClient returns an http.Client with bounded timeouts so a hung or
// non-responsive LLM endpoint can never block a caller indefinitely. The stdlib
// http.DefaultClient has no timeout and must not be used here. A generation may
// take 300 s to its first byte, as the reference clients allow (MADR 0012
// §1.3); callers wanting less set a context deadline. Listings keep their own
// 10 s bound. It honours HTTP_PROXY, HTTPS_PROXY and NO_PROXY, as net/http's
// own default transport does (0016-MADR D8).
func DefaultClient() *http.Client {
	return &http.Client{
		Timeout: 330 * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 300 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConnsPerHost:   4,
		},
	}
}

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
		return time.Duration(secs * float64(time.Second))
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
		return time.Duration(ms * float64(time.Millisecond))
	}
	return ParseRetryAfter(h.Get("Retry-After"))
}
