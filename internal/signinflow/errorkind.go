package signinflow

import (
	"context"
	"errors"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// kindRows are 0073-MADR D3's kinds, each with the sentinel it maps from, in
// that table's order: a consumer tests the most specific first. Several
// sentinels match more than one row (ErrQuotaExhausted also matches
// ErrRateLimited, ErrEmptyAnswer also matches ErrIncomplete and
// ErrInvalidRequest, ErrOffline also matches ErrProviderUnavailable, and
// both sign-in sentinels match ErrAuthFailure), so the order is the rule.
var kindRows = []struct {
	kind authcontract.ErrorKind
	is   []error
}{
	{authcontract.FailureCanceled, []error{context.Canceled}},
	{authcontract.FailureTimeout, []error{context.DeadlineExceeded}},
	{authcontract.FailureDenied, []error{auth.ErrSignInDenied}},
	{authcontract.FailureExpired, []error{auth.ErrSignInExpired}},
	{authcontract.FailureAuth, []error{llmprovider.ErrAuthFailure}},
	{authcontract.FailureQuota, []error{llmprovider.ErrQuotaExhausted}},
	{authcontract.FailureRateLimited, []error{llmprovider.ErrRateLimited}},
	{authcontract.FailureNotPermitted, []error{llmprovider.ErrNotPermitted}},
	{authcontract.FailureWorkspace, []error{llmprovider.ErrWorkspaceRequired, llmprovider.ErrWorkspaceNotFound}},
	{authcontract.FailureContextOverflow, []error{llmprovider.ErrContextOverflow}},
	{authcontract.FailureEmptyAnswer, []error{llmprovider.ErrEmptyAnswer}},
	{authcontract.FailureIncomplete, []error{llmprovider.ErrIncomplete}},
	{authcontract.FailureInvalidRequest, []error{llmprovider.ErrInvalidRequest}},
	{authcontract.FailureInvalidProvider, []error{llmprovider.ErrInvalidProvider}},
	{authcontract.FailureOffline, []error{llmprovider.ErrOffline}},
	{authcontract.FailureUnavailable, []error{llmprovider.ErrProviderUnavailable}},
	{authcontract.FailureUnsupported, []error{llmprovider.ErrUnsupported, errors.ErrUnsupported}},
}

// ErrorKindOf maps err to 0073-MADR D3's kinds, most specific first, in that
// table's order. An error that matches none is "other", and a nil one is the
// empty kind.
func ErrorKindOf(err error) authcontract.ErrorKind {
	if err == nil {
		return ""
	}
	for _, row := range kindRows {
		for _, sentinel := range row.is {
			if errors.Is(err, sentinel) {
				return row.kind
			}
		}
	}
	return authcontract.FailureOther
}
