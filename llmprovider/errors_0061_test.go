package llmprovider

import (
	"errors"
	"fmt"
	"testing"
)

// TestErrEmptyAnswer_Chain_0061 (0061-MADR D1): ErrEmptyAnswer matches
// ErrIncomplete and ErrInvalidRequest, and an answer of its kind is not
// retried.
func TestErrEmptyAnswer_Chain_0061(t *testing.T) {
	if !errors.Is(ErrEmptyAnswer, ErrIncomplete) || !errors.Is(ErrEmptyAnswer, ErrInvalidRequest) {
		t.Errorf("ErrEmptyAnswer matches ErrIncomplete %t, ErrInvalidRequest %t; want both",
			errors.Is(ErrEmptyAnswer, ErrIncomplete), errors.Is(ErrEmptyAnswer, ErrInvalidRequest))
	}
	if got := ErrEmptyAnswer.Error(); got != "llmprovider: empty answer" {
		t.Errorf("ErrEmptyAnswer = %q; want %q", got, "llmprovider: empty answer")
	}
	err := &APIError{Kind: ErrEmptyAnswer, Reason: "content_filter"}
	if !errors.Is(err, ErrEmptyAnswer) || !errors.Is(err, ErrIncomplete) {
		t.Errorf("an APIError of kind ErrEmptyAnswer: matches it %t, ErrIncomplete %t; want both",
			errors.Is(err, ErrEmptyAnswer), errors.Is(err, ErrIncomplete))
	}
	if err.Retryable() || retryable(err) {
		t.Errorf("Retryable %t, retryable %t; want neither", err.Retryable(), retryable(err))
	}
}

// TestErrOffline_Chain_0061 (0061-MADR D2): ErrOffline matches
// ErrProviderUnavailable and is retried.
func TestErrOffline_Chain_0061(t *testing.T) {
	if !errors.Is(ErrOffline, ErrProviderUnavailable) {
		t.Error("ErrOffline does not match ErrProviderUnavailable")
	}
	if got := ErrOffline.Error(); got != "llmprovider: offline" {
		t.Errorf("ErrOffline = %q; want %q", got, "llmprovider: offline")
	}
	wrapped := fmt.Errorf("%w: dial tcp: connection refused", ErrOffline)
	if !retryable(wrapped) {
		t.Error("an offline failure is not retried")
	}
}
