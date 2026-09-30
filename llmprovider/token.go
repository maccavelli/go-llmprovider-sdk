package llmprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// TokenType identifies how a token authenticates a provider request.
type TokenType string

const (
	// TokenAPIKey identifies a provider API key.
	TokenAPIKey TokenType = "api_key"
	// TokenBearer identifies a bearer token.
	TokenBearer TokenType = "bearer"
)

// redactedSecret replaces a secret in every formatted form of a
// secret-bearing value (0016-MADR D5). It reveals nothing, not even a suffix:
// these forms reach logs, where redact.MaskSecret's partial reveal is not
// allowed.
const redactedSecret = "[redacted]"

// secretText is what a formatted value shows for a secret: nothing when it
// is empty, else redactedSecret.
func secretText(secret string) string {
	if secret == "" {
		return ""
	}
	return redactedSecret
}

// expiryText renders an expiry, or "none" for the zero time.
func expiryText(t time.Time) string {
	if t.IsZero() {
		return "none"
	}
	return t.UTC().Format(time.RFC3339)
}

// Token is an authentication value returned by a TokenSource. Its String,
// GoString and LogValue never show Value.
type Token struct {
	Value  string
	Type   TokenType
	Expiry time.Time
	Header string
}

// String renders the token without its value.
func (t Token) String() string {
	return fmt.Sprintf("Token{Type:%s Header:%s Expiry:%s Value:%s}", t.Type, t.Header, expiryText(t.Expiry), secretText(t.Value))
}

// GoString renders the token for %#v, without its value.
func (t Token) GoString() string { return "llmprovider." + t.String() }

// MarshalJSON encodes the token without its value, so a struct holding a
// Token, logged through slog's JSON handler or encoded by a caller, never
// carries the secret (0016-MADR D5 and its amendment of 2026-09-30). The
// encoding does not round-trip: code that needs the value reads the field.
func (t Token) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type   TokenType `json:"type"`
		Header string    `json:"header"`
		Expiry string    `json:"expiry"`
		Value  string    `json:"value"`
	}{t.Type, t.Header, expiryText(t.Expiry), secretText(t.Value)})
}

// LogValue renders the token for slog, without its value.
func (t Token) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(t.Type)),
		slog.String("header", t.Header),
		slog.String("expiry", expiryText(t.Expiry)),
		slog.String("value", secretText(t.Value)),
	)
}

// TokenSource supplies authentication for a provider request.
type TokenSource interface {
	Token(ctx context.Context) (Token, error)
}

// StaticToken returns one fixed bearer token without expiry. Its String,
// GoString and LogValue never show Value.
type StaticToken struct {
	Value  string
	Header string
}

// NewStaticToken constructs a static bearer-token source.
func NewStaticToken(value string) *StaticToken {
	return &StaticToken{Value: value}
}

// Token returns the static value and its configured authorization header.
func (s *StaticToken) Token(ctx context.Context) (Token, error) {
	header := s.Header
	if header == "" {
		header = "Authorization"
	}
	return Token{
		Value:  s.Value,
		Type:   TokenBearer,
		Header: header,
	}, nil
}

// String renders the source without its value.
func (s StaticToken) String() string {
	return fmt.Sprintf("StaticToken{Header:%s Value:%s}", s.Header, secretText(s.Value))
}

// GoString renders the source for %#v, without its value.
func (s StaticToken) GoString() string { return "llmprovider." + s.String() }

// MarshalJSON encodes the source without its value; see Token.MarshalJSON.
func (s StaticToken) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Header string `json:"header"`
		Value  string `json:"value"`
	}{s.Header, secretText(s.Value)})
}

// LogValue renders the source for slog, without its value.
func (s StaticToken) LogValue() slog.Value {
	return slog.GroupValue(slog.String("header", s.Header), slog.String("value", secretText(s.Value)))
}
