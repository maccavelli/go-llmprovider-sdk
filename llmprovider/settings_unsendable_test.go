package llmprovider

import (
	"errors"
	"net/url"
	"syscall"
	"testing"
)

// TestResolveOptions_RefusesUnsendable (0026-MADR F11, F19): a value no
// request could carry is refused when the provider is built, with
// ErrInvalidRequest (R23), not sent, retried and returned with no kind.
func TestResolveOptions_RefusesUnsendable(t *testing.T) {
	for name, opt := range map[string]Option{
		"base URL without a scheme":  WithBaseURL("localhost:11434"),
		"base URL without a host":    WithBaseURL("http://"),
		"base URL with a bad escape": WithBaseURL("http://host/%zz"),
		"key with a newline":         WithAPIKey("sk-test\n"),
		"source token with a CR":     WithTokenSource(NewStaticToken("sk-test\r")),
		"client info with CRLF":      WithClientInfo("app\r\nX-Injected: 1", "1.0"),
		"client version with a NUL":  WithClientInfo("app", "1.0\x00"),
		"session id with a newline":  WithSessionID("s-1\n"),
		"unknown reasoning effort":   WithReasoning(&Reasoning{Effort: "extreme"}),
		"negative reasoning budget":  WithReasoning(&Reasoning{Budget: -1}),
	} {
		if _, err := ResolveOptions(ProviderOpenAI, []Option{opt}); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: ResolveOptions = %v, want ErrInvalidRequest", name, err)
		}
	}
	for name, opt := range map[string]Option{
		"loopback base URL with a slash": WithBaseURL("http://localhost:11434/"),
		"https base URL with a path":     WithBaseURL("https://api.example.test/v1"),
		"no base URL":                    WithBaseURL(""),
		"plain key":                      WithAPIKey("sk-test"),
		"plain client info":              WithClientInfo("my-app", "1.2.3"),
		"high effort":                    WithReasoning(&Reasoning{Effort: "high", Budget: 1024}),
	} {
		if _, err := ResolveOptions(ProviderOpenAI, []Option{opt}); err != nil {
			t.Errorf("%s: ResolveOptions = %v, want it accepted", name, err)
		}
	}
}

// TestRetryable_UnsendableIsTerminal (0026-MADR F11): a failure to build or
// send a request that no retry can mend is not retried; a failure to reach
// the service still is.
func TestRetryable_UnsendableIsTerminal(t *testing.T) {
	for name, err := range map[string]error{
		"bad header value": &url.Error{Op: "Post", URL: "https://x.test", Err: errors.New(`net/http: invalid header field value for "Authorization"`)},
		"URL parse":        &url.Error{Op: "parse", URL: "http://host/%zz", Err: errors.New(`invalid URL escape "%zz"`)},
		"no scheme":        &url.Error{Op: "Post", URL: "localhost:11434/v1", Err: errors.New(`unsupported protocol scheme "localhost"`)},
	} {
		if retryable(err) {
			t.Errorf("%s: retryable = true, want false", name)
		}
	}
	refused := &url.Error{Op: "Post", URL: "https://x.test", Err: syscall.ECONNREFUSED}
	if !retryable(refused) {
		t.Error("connection refused: retryable = false, want true")
	}
}
