package signin_test

import (
	"context"
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
