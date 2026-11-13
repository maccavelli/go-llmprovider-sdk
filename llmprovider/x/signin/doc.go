// Package signin runs one sign-in through one handle, and describes the
// sign-in methods a descriptor offers in the words of the auth contract
// module (github.com/maccavelli/go-llmprovider-sdk/authcontract).
//
// Methods returns a descriptor's methods as the contract's Method values. Start
// begins a sign-in and returns a SignIn: its Steps are what the person is
// shown or asked, Answer gives the answer to a step that asks, and Wait
// returns the Credential the sign-in yields. The sign-in saves nothing: the
// caller saves, as the wizard does after its listing. ErrorKindOf names an
// error in the contract's one vocabulary of failures.
//
// The methods Start drives are device_code (OpenAI, Grok and Kilo),
// browser_oauth (OpenAI and Grok), chatgpt_plan and import_vendor_cli. A
// value method (api_key, token_stdin) is not driven: the caller reads the
// value itself.
//
// # Experimental
//
// This package is experimental (R4 of the API standards): it is outside the
// v1 promise and "make api-check", and may change in any release. A later
// record promotes Start, SignIn, Options, Credential, Methods and ErrorKindOf
// into llmprovider/auth, in a minor release, once gobble-cli and
// magic-cli-remote have each shipped a release that drives a sign-in through
// it (0074-MADR D9).
//
// A minimal device sign-in prints each step and returns the credential:
//
//	run, err := signin.Start(ctx, llmprovider.ProviderKilo, llmprovider.AuthDeviceCode, signin.Options{})
//	if err != nil {
//		return err
//	}
//	go func() {
//		for step := range run.Steps() {
//			fmt.Println(step.Kind, step.URL, step.Code)
//		}
//	}()
//	credential, err := run.Wait(ctx)
package signin
