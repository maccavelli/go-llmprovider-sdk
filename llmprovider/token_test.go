package llmprovider

import (
	"context"
	"testing"
)

// TestStaticToken_ReturnsAPIKey: a static key is a TokenAPIKey with no Header
// of its own, so it goes in the service's own header (0016-MADR A6). Before
// A6 it reported TokenBearer and Authorization.
func TestStaticToken_ReturnsAPIKey(t *testing.T) {
	tok, err := NewStaticToken("sk-test").Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok.Type != TokenAPIKey {
		t.Errorf("Type = %q, want %q", tok.Type, TokenAPIKey)
	}
	if tok.Header != "" {
		t.Errorf("Header = %q, want none", tok.Header)
	}
	if tok.Value != "sk-test" {
		t.Errorf("Value = %q, want %q", tok.Value, "sk-test")
	}
	if !tok.Expiry.IsZero() {
		t.Errorf("Expiry = %v, want zero", tok.Expiry)
	}
}

func TestStaticToken_EmptyValueStillReturnsToken(t *testing.T) {
	_, err := NewStaticToken("").Token(context.Background())
	if err != nil {
		t.Errorf("Token() with empty value returned error %v; want nil", err)
	}
}

func TestStaticToken_CustomHeader(t *testing.T) {
	s := &StaticToken{Value: "x", Header: "X-Custom"}
	tok, err := s.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok.Header != "X-Custom" {
		t.Errorf("Header = %q, want %q", tok.Header, "X-Custom")
	}
}
