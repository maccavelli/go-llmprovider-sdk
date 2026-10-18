package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type pkceCodes struct {
	verifier  string
	challenge string
}

func newPKCE() (pkceCodes, error) {
	verifier, err := randomBase64URL()
	if err != nil {
		return pkceCodes{}, fmt.Errorf("oauth: generate PKCE verifier: %w", err)
	}
	digest := sha256.Sum256([]byte(verifier))
	return pkceCodes{
		verifier:  verifier,
		challenge: base64.RawURLEncoding.EncodeToString(digest[:]),
	}, nil
}

// randomBase64URL is 32 random bytes, base64url-encoded without padding: a
// PKCE verifier, a state or a nonce.
func randomBase64URL() (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(random), nil
}
