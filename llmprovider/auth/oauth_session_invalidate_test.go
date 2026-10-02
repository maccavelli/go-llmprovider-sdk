package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// An *OAuthSession is an InvalidatingSource (0015-MADR amendment "the ChatGPT
// helpers leave llmprovider").
var _ llmprovider.InvalidatingSource = (*OAuthSession)(nil)

// TestOAuthSession_Invalidate: after Invalidate, a session that was current
// must renew: with no refresh token, Token fails as for an expired session.
func TestOAuthSession_Invalidate(t *testing.T) {
	s := &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a", Expiry: time.Now().Add(time.Hour)}
	if _, err := s.Token(t.Context()); err != nil {
		t.Fatalf("before Invalidate: Token() = %v", err)
	}
	s.Invalidate()
	if _, err := s.Token(t.Context()); !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Fatalf("after Invalidate: Token() = %v, want ErrAuthFailure: no refresh token", err)
	}
}

// TestOAuthSession_Account returns the account id and FedRAMP flag.
func TestOAuthSession_Account(t *testing.T) {
	s := &OAuthSession{AccountID: "acct", FedRAMP: true}
	if id, fedRAMP := s.Account(); id != "acct" || !fedRAMP {
		t.Fatalf("Account() = %q, %t; want acct, true", id, fedRAMP)
	}
}
