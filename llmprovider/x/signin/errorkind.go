package signin

import (
	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
)

// ErrorKindOf maps err to the contract's error kinds (0073-MADR D3), most
// specific first, in that table's order: canceled, timeout, denied, expired,
// auth, quota, rate_limited, not_permitted, workspace, context_overflow,
// empty_answer, incomplete, invalid_request, invalid_provider, offline,
// unavailable and unsupported. An error that matches none is "other"; a nil
// error is the empty kind. "denied" and "expired" are told apart by
// auth.ErrSignInDenied and auth.ErrSignInExpired (0074-MADR D8). It is safe
// for concurrent use.
func ErrorKindOf(err error) authcontract.ErrorKind { return signinflow.ErrorKindOf(err) }
