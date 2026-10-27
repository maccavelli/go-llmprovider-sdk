package wizard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// errorClasses are the module's kind sentinels, each child before its
// parent, so that describe names the most specific one an error matches
// (0056-MADR D3).
var errorClasses = []error{
	llmprovider.ErrQuotaExhausted,
	llmprovider.ErrRateLimited,
	llmprovider.ErrNotPermitted,
	llmprovider.ErrAuthFailure,
	llmprovider.ErrContextOverflow,
	llmprovider.ErrIncomplete,
	llmprovider.ErrWorkspaceRequired,
	llmprovider.ErrWorkspaceNotFound,
	llmprovider.ErrInvalidRequest,
	llmprovider.ErrProviderUnavailable,
	llmprovider.ErrUnsupported,
}

// describe words err for the user from its class and HTTP status, never its
// text: a service's message can echo a key it has masked itself, which
// redaction leaves as it is (0056-MADR F7, D3). The error returned to the
// caller keeps its text.
func describe(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timed out"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	}
	class := "the request failed"
	for _, sentinel := range errorClasses {
		if errors.Is(err, sentinel) {
			class = strings.TrimPrefix(sentinel.Error(), "llmprovider: ")
			break
		}
	}
	if apiErr, ok := errors.AsType[*llmprovider.APIError](err); ok && apiErr.Status != 0 {
		return fmt.Sprintf("%s, HTTP %d", class, apiErr.Status)
	}
	return class
}
