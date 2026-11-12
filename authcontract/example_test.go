package authcontract_test

import (
	"fmt"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
)

// An answer whose step is a secret, or not known, prints its paste masked.
// Its JSON carries the paste, since the answer must reach the sign-in.
func ExampleAnswer_String() {
	a := authcontract.Answer{
		SchemaVersion: authcontract.SchemaVersion,
		Seq:           3,
		Paste:         "http://localhost:1455/auth/callback?code=ac_example",
	}
	fmt.Println(a)
	// Output:
	// Answer{SchemaVersion:1 Seq:3 Secret:<nil> Paste:[redacted]}
}
