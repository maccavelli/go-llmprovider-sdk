package llmprovider

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// issuerKeys are generated once per test binary: RSA key generation is slow.
var issuerKeys = sync.OnceValue(func() struct {
	rsa *rsa.PrivateKey
	ec  *ecdsa.PrivateKey
	ed  ed25519.PrivateKey
} {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return struct {
		rsa *rsa.PrivateKey
		ec  *ecdsa.PrivateKey
		ed  ed25519.PrivateKey
	}{rsaKey, ecKey, edKey}
})

// testIssuer is a fake OIDC issuer's signing side for login tests: it serves
// its keys at /jwks, adds jwks_uri to discovery documents, and signs
// id_tokens. Keys are named "rsa-1", "ec-1" and "ed-1".
type testIssuer struct {
	base     string
	jwksHits atomic.Int32
	mu       sync.Mutex
	nonce    string
	// hideKid names a key left out of /jwks, to test a key rotation.
	hideKid string
}

// newTestIssuer registers /jwks on mux for an issuer at base.
func newTestIssuer(mux *http.ServeMux, base string) *testIssuer {
	ti := &testIssuer{base: base}
	mux.HandleFunc("/jwks", ti.serveJWKS)
	return ti
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func (ti *testIssuer) serveJWKS(w http.ResponseWriter, _ *http.Request) {
	ti.jwksHits.Add(1)
	keys := issuerKeys()
	var set []map[string]string
	add := func(kid string, key map[string]string) {
		ti.mu.Lock()
		hidden := ti.hideKid == kid
		ti.mu.Unlock()
		if !hidden {
			key["kid"] = kid
			set = append(set, key)
		}
	}
	add("rsa-1", map[string]string{"kty": "RSA", "use": "sig", "n": b64(keys.rsa.N.Bytes()),
		"e": b64(big.NewInt(int64(keys.rsa.E)).Bytes())})
	add("ec-1", map[string]string{"kty": "EC", "crv": "P-256", "x": b64(keys.ec.X.FillBytes(make([]byte, 32))),
		"y": b64(keys.ec.Y.FillBytes(make([]byte, 32)))})
	add("ed-1", map[string]string{"kty": "OKP", "crv": "Ed25519", "x": b64(keys.ed.Public().(ed25519.PublicKey))})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": set})
}

// discovery returns a discovery document for the issuer: fields, plus its
// jwks_uri.
func (ti *testIssuer) discovery(t *testing.T, fields map[string]string) string {
	t.Helper()
	doc := map[string]any{"issuer": ti.base, "jwks_uri": ti.base + "/jwks"}
	for k, v := range fields {
		doc[k] = v
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// captureNonce records the nonce of an authorize URL, for the next id_token.
func (ti *testIssuer) captureNonce(rawURL string) {
	if u, err := url.Parse(rawURL); err == nil {
		ti.mu.Lock()
		ti.nonce = u.Query().Get("nonce")
		ti.mu.Unlock()
	}
}

// tokenResponse is a token-endpoint body carrying a valid id_token for
// clientID, signed with alg, echoing any captured nonce.
func (ti *testIssuer) tokenResponse(t *testing.T, alg, clientID, access, refresh string) string {
	t.Helper()
	ti.mu.Lock()
	nonce := ti.nonce
	ti.mu.Unlock()
	claims := map[string]any{"iss": ti.base, "aud": clientID, "sub": "user-1",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix()}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	raw, err := json.Marshal(map[string]any{"access_token": access, "refresh_token": refresh, "expires_in": 3600,
		"id_token": signTestJWT(t, alg, kidFor(alg), claims)})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func kidFor(alg string) string {
	switch alg {
	case "ES256":
		return "ec-1"
	case "EdDSA":
		return "ed-1"
	}
	return "rsa-1"
}

// signTestJWT signs claims with the issuer key alg names; alg "none" and
// "HS256" produce tokens verification must refuse.
func signTestJWT(t *testing.T, alg, kid string, claims map[string]any) string {
	t.Helper()
	header := map[string]string{"alg": alg, "typ": "JWT"}
	if kid != "" {
		header["kid"] = kid
	}
	h, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	c, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := b64(h) + "." + b64(c)
	digest := sha256.Sum256([]byte(input))
	keys := issuerKeys()
	var sig []byte
	switch alg {
	case "RS256":
		sig, err = rsa.SignPKCS1v15(rand.Reader, keys.rsa, crypto.SHA256, digest[:])
	case "PS256":
		sig, err = rsa.SignPSS(rand.Reader, keys.rsa, crypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	case "ES256":
		var r, s *big.Int
		r, s, err = ecdsa.Sign(rand.Reader, keys.ec, digest[:])
		if err == nil {
			sig = append(r.FillBytes(make([]byte, 32)), s.FillBytes(make([]byte, 32))...)
		}
	case "EdDSA":
		sig = ed25519.Sign(keys.ed, []byte(input))
	case "HS256":
		sig = digest[:]
	case "none":
		sig = nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + b64(sig)
}
