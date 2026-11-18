package auth

// 0082-PLAN Phase 1's fifth pass (the fourth amendment of 2026-11-18: the
// fourth review's Q16, ported). Every issuer is an in-process server, every
// foreign record names a dead token URL or the in-process issuer, and the
// package runs behind localonly's guard.

import (
	"bytes"
	"context"
	"testing"
)

// Q16 (the fourth review's TestR4_WhitespaceRefreshIsARefreshToken): a record
// whose refresh token is one space, and which carries this session's access
// token under another client, has a refresh token of its own: a record is
// "with no refresh token" only when the field is empty, not when it is blank.
// So the access-token exemption does not take it as the session's, it is judged
// by its labels, refused, nothing is sent, and it is not written over. Red when
// a blank refresh token counts as none.
func TestWhitespaceRefreshTokenIsARefreshToken_0082(t *testing.T) {
	for name, mk := range storeVariants0082() {
		t.Run(name, func(t *testing.T) {
			srv, sent := rotatingServer0082(t, nil)
			store, raw := mk()
			held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
			other := heldPlan0082(nil, srv.URL+"/other", nil) // carries the held session's access token
			other.ClientID, other.AccountID, other.Refresh = "oaiapp_other", "acct-other", " "
			raw.set(other)
			want := raw.bytes()
			held.Invalidate()
			before := fieldsOf0082(held)
			tok, err := held.Token(context.Background())
			refusedFully0082(t, held, before, tok, err)
			if got := sent(); len(got) != 0 {
				t.Errorf("refresh tokens sent %q, want none", got)
			}
			if _, saves := raw.counts(); saves != 0 || !bytes.Equal(raw.bytes(), want) {
				t.Errorf("the other sign-in's record was written over (%d saves)", saves)
			}
		})
	}
}
