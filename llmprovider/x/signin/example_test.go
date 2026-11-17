package signin_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
)

// ExampleStart signs in to Kilo by a device code, against a test server that
// approves at once: it prints the kind of each step, then what the sign-in
// yields. A real caller shows the show_code step's URL and code to the
// person, and saves the credential.
func ExampleStart() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/device-auth/codes":
			fmt.Fprint(w, `{"code":"KILO-1234","verificationUrl":"https://example.com/device","expiresIn":600}`)
		case strings.HasPrefix(r.URL.Path, "/api/device-auth/codes/"):
			fmt.Fprint(w, `{"token":"kilo-device-token-0123456789"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	run, err := signin.Start(ctx, llmprovider.ProviderKilo, llmprovider.AuthDeviceCode,
		signin.Options{HTTPClient: server.Client(), Issuer: server.URL})
	if err != nil {
		fmt.Println(err)
		return
	}
	credential, err := run.Wait(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	for step := range run.Steps() {
		fmt.Println("step:", step.Kind)
	}
	fmt.Println("yields:", credential.Yields())
	// Output:
	// step: show_code
	// step: waiting
	// step: done
	// yields: device_key
}

// ExampleVerifyKilo checks two Kilo keys against a test server that stands in
// for Kilo's profile: it accepts one key and answers 401 to the other, which
// ErrorKindOf names "auth". A key checked with an organization is not read at
// all, since the organization's listing is the check there.
func ExampleVerifyKilo() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/profile" || r.Header.Get("Authorization") != "Bearer example-good-key" {
			http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{"organizations":[],"hasPersonalAccount":true}`)
	}))
	defer server.Close()

	ctx := context.Background()
	opts := []llmprovider.Option{
		llmprovider.WithHTTPClient(server.Client()),
		llmprovider.WithBaseURL(server.URL + "/api/gateway"),
	}
	fmt.Println("good key:", signin.VerifyKilo(ctx, "example-good-key", "", opts...))
	err := signin.VerifyKilo(ctx, "example-wrong-key", "", opts...)
	fmt.Println("wrong key:", signin.ErrorKindOf(err), errors.Is(err, llmprovider.ErrAuthFailure))
	fmt.Println("with an organization:", signin.VerifyKilo(ctx, "example-wrong-key", "org-1", opts...))
	// Output:
	// good key: <nil>
	// wrong key: auth true
	// with an organization: <nil>
}

// ExampleMethods lists the methods a descriptor offers, in the contract's
// words.
func ExampleMethods() {
	d := llmprovider.Descriptor{
		ID:             "example",
		Label:          "Example",
		RequiresAPIKey: true,
	}
	for _, m := range signin.Methods(d) {
		fmt.Println(m.ID, m.Kind, m.Yields)
	}
	// Output: api_key api_key api_key
}

// ExampleErrorKindOf names a failure in the contract's vocabulary.
func ExampleErrorKindOf() {
	fmt.Println(signin.ErrorKindOf(fmt.Errorf("sign-in: %w", llmprovider.ErrRateLimited)))
	// Output: rate_limited
}
