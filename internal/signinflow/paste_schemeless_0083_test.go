package signinflow

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestPasteSchemelessCallback_0083 (0083-MADR D11, ST76 to ST85): the address bar
// shows the host and port without the scheme until it is tapped, so a paste
// can lack http://. One that begins with the callback's exact host and port is
// read as the address it is, with every check an address meets; no other
// prefix is.
func TestPasteSchemelessCallback_0083(t *testing.T) {
	const callback = "127.0.0.1:1455/auth/callback"
	authorize := "https://issuer.test/authorize?redirect_uri=" + url.QueryEscape("http://"+callback) + "&state=" + testState
	good := callback + "?code=c&state=" + testState
	for _, c := range []struct {
		name    string
		accepts authcontract.Paste
		paste   string
		bad     string // a phrase of the reason, "" when the paste passes
	}{
		{"a code or a URL: read as the address, its state checked", authcontract.PasteURLOrCode, callback + "?code=c&state=other",
			"not from this sign-in"},
		{"a code or a URL: no state", authcontract.PasteURLOrCode, callback + "?code=c", "not from this sign-in"},
		{"a code or a URL: no code", authcontract.PasteURLOrCode, callback + "?state=" + testState, "carries no code"},
		{"a code or a URL: this state and a code", authcontract.PasteURLOrCode, good, ""},
		{"a first plan sign-in with a client", authcontract.PasteURLOnly, callback + "?code=c&client_id=app&state=" + testState, ""},
		{"a first plan sign-in without a client", authcontract.PasteURLOnly, good, "carries no client_id"},
		{"a code only: refused as a URL is", authcontract.PasteCode, good, "That is a URL, not a code"},
		{"spaces and a line break around it", authcontract.PasteURLOrCode, "  " + callback + "?code=c&state=other \n", "not from this sign-in"},
		{"a denial is passed on", authcontract.PasteURLOrCode, callback + "?error=access_denied&state=" + testState, ""},
		{"a retry that does not parse", authcontract.PasteURLOrCode, "127.0.0.1:1455/%zz?code=c&state=" + testState, "not a usable address"},
		{"a port that begins with the callback's and is none", authcontract.PasteURLOrCode, "127.0.0.1:1455x/auth/callback?code=c&state=other", ""},
		{"the host and port alone", authcontract.PasteURLOrCode, "127.0.0.1:1455", "cut short"},
		{"the host and port and a slash", authcontract.PasteURLOrCode, "127.0.0.1:1455/", "cut short"},
		{"the host and port alone, a first plan sign-in", authcontract.PasteURLOnly, "127.0.0.1:1455", "cut short"},
		{"the host and port alone, a code only", authcontract.PasteCode, "127.0.0.1:1455/", "cut short"},
		{"the host and port alone, with spaces", authcontract.PasteURLOrCode, "  127.0.0.1:1455/ \n", "cut short"},
		{"a longer port that begins with the callback's", authcontract.PasteURLOrCode, "127.0.0.1:14550/auth/callback?code=c&state=other", ""},
		{"another port", authcontract.PasteURLOrCode, "127.0.0.1:1457/auth/callback?code=c&state=other", ""},
		{"the callback's host with no port", authcontract.PasteURLOrCode, "127.0.0.1/auth/callback?code=c&state=other", ""},
		{"localhost as it is read today", authcontract.PasteURLOrCode, "localhost:1455/auth/callback?code=c&state=other", "not from this sign-in"},
		{"[::1] is a bare code", authcontract.PasteURLOrCode, "[::1]:1455/auth/callback?code=c&state=other", ""},
		{"a colon and a slash in a bare code", authcontract.PasteURLOrCode, "12:34/x", ""},
		{"a bare code with a slash", authcontract.PasteURLOrCode, "abc/def", ""},
		{"a bare code for a first plan sign-in", authcontract.PasteURLOnly, "abc/def", "That is not a URL"},
	} {
		reason := newPasteCheck(authorize, c.accepts).check(c.paste)
		if (c.bad == "") != (reason == "") || (c.bad != "" && !strings.Contains(reason, c.bad)) {
			t.Errorf("%s: check = %q; want %q", c.name, reason, c.bad)
		}
	}
}

// TestPasteSchemelessHandedOnWithScheme_0083 (ST76, ST81, ST84): the engine hands
// the login the address with http:// in front, for a paste that began with
// the callback's host and port and no scheme, and a bare code as it was.
func TestPasteSchemelessHandedOnWithScheme_0083(t *testing.T) {
	const callback = "127.0.0.1:1455/auth/callback"
	authorize := "https://issuer.test/authorize?redirect_uri=" + url.QueryEscape("http://"+callback) + "&state=" + testState
	for _, c := range []struct{ name, paste, want string }{
		{"a schemeless address", callback + "?code=c&state=" + testState, "http://" + callback + "?code=c&state=" + testState},
		{"padded", "  " + callback + "?code=c&state=" + testState + "\n", "http://" + callback + "?code=c&state=" + testState},
		{"an address with its scheme", "http://" + callback + "?code=c&state=" + testState, "http://" + callback + "?code=c&state=" + testState},
		{"a bare code", "code-only", "code-only"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := make(chan string, 1)
			logins := Logins{Browser: func(ctx context.Context, provider llmprovider.ProviderID, o auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
				_ = o.OpenURL(authorize)
				text, err := o.InputCode(ctx)
				got <- text
				return &auth.OAuthSession{Provider: provider, Access: "access-0123456789"}, err
			}}
			run, err := Start(background(), Spec{Provider: llmprovider.ProviderOpenAI, Method: pastable(), Mode: ModeBrowser,
				NoLoopback: true, Logins: logins})
			if err != nil {
				t.Fatalf("Start = %v", err)
			}
			open := next(t, run)
			ask := next(t, run)
			if open.Kind != authcontract.StepOpenURL || ask.Kind != authcontract.StepPaste {
				t.Fatalf("steps %v, %v; want open_url, paste", open.Kind, ask.Kind)
			}
			if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: c.paste}); err != nil {
				t.Fatalf("Answer = %v", err)
			}
			if _, err := run.Wait(bounded(t)); err != nil {
				t.Fatalf("Wait = %v", err)
			}
			if text := <-got; text != c.want {
				t.Errorf("the login was handed %q; want %q", text, c.want)
			}
		})
	}
}

// TestPasteCutShortAskedAgain_0083 (ST89): a paste of the callback's host and
// port alone, with or without a trailing slash, is refused with its own words
// and asked again, counted among the three; nothing reaches the login until a
// whole address does.
func TestPasteCutShortAskedAgain_0083(t *testing.T) {
	const callback = "127.0.0.1:1455/auth/callback"
	authorize := "https://issuer.test/authorize?redirect_uri=" + url.QueryEscape("http://"+callback) + "&state=" + testState
	got := make(chan string, 2)
	logins := Logins{Browser: func(ctx context.Context, provider llmprovider.ProviderID, o auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		_ = o.OpenURL(authorize)
		text, err := o.InputCode(ctx)
		got <- text
		return &auth.OAuthSession{Provider: provider, Access: "access-0123456789"}, err
	}}
	run, err := Start(background(), Spec{Provider: llmprovider.ProviderOpenAI, Method: pastable(), Mode: ModeBrowser,
		NoLoopback: true, Logins: logins})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	next(t, run) // open_url
	ask := next(t, run)
	for _, cut := range []string{"127.0.0.1:1455", "127.0.0.1:1455/"} {
		if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: cut}); err != nil {
			t.Fatalf("Answer = %v", err)
		}
		ask = next(t, run)
		if ask.Kind != authcontract.StepPaste || !strings.Contains(ask.Text, "cut short") || strings.Contains(ask.Text, "state does not match") {
			t.Fatalf("after %q the next step is %v %q; want a paste step that says the address is cut short", cut, ask.Kind, ask.Text)
		}
	}
	select {
	case text := <-got:
		t.Fatalf("the login was handed %q before a whole address was pasted", text)
	default:
	}
	whole := callback + "?code=c&state=" + testState
	if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: whole}); err != nil {
		t.Fatalf("Answer = %v", err)
	}
	if _, err := run.Wait(bounded(t)); err != nil {
		t.Fatalf("Wait = %v", err)
	}
	if text := <-got; text != "http://"+whole {
		t.Errorf("the login was handed %q; want the whole address with http:// in front", text)
	}
}
