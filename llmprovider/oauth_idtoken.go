package llmprovider

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// id_token verification (0016-MADR D7 as the owner decided it, and its
// amendment A3). Every login verifies the id_token's signature against the
// issuer's published keys before any claim is used, and any failure fails the
// login.

const (
	// jwksCacheTTL is how long an issuer's keys are reused.
	jwksCacheTTL = time.Hour
	// jwksLimit bounds a JWKS response.
	jwksLimit = 1 << 20
	// idTokenLeeway tolerates clock skew on exp.
	idTokenLeeway = time.Minute
	// minRSABits refuses weak RSA keys.
	minRSABits = 2048
)

// The algorithms the built-in issuers sign with (probed 2026-09-30).
const (
	algRS256 = "RS256"
	algES256 = "ES256"
	algES384 = "ES384"
)

// verifiableAlgorithms are the JWS algorithms the standard library verifies.
// none and every HMAC algorithm are refused.
var verifiableAlgorithms = []string{
	algRS256, "RS384", "RS512", "PS256", "PS384", "PS512", algES256, algES384, "EdDSA",
}

// errIDToken prefixes every verification failure.
var errIDToken = errors.New("oauth: id_token rejected")

// idTokenCheck is what one login's id_token must satisfy.
type idTokenCheck struct {
	issuer, clientID, nonce string
	jwksURL                 string
	// advertised is the discovery document's
	// id_token_signing_alg_values_supported; empty when it lists none.
	advertised []string
}

// verifyIDToken verifies raw's signature and claims, or fails.
func verifyIDToken(ctx context.Context, client *http.Client, raw string, check idTokenCheck, now time.Time) error {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return fmt.Errorf("%w: not a signed JWT", errIDToken)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return fmt.Errorf("%w: header: %w", errIDToken, err)
	}
	if !algorithmAllowed(header.Alg, check.advertised) {
		return fmt.Errorf("%w: algorithm %q is not allowed", errIDToken, header.Alg)
	}
	if header.Kid == "" {
		return fmt.Errorf("%w: no key id", errIDToken)
	}
	if check.jwksURL == "" {
		return fmt.Errorf("%w: the issuer publishes no jwks_uri", errIDToken)
	}
	key, err := issuerKey(ctx, client, check.jwksURL, header.Kid)
	if err != nil {
		return fmt.Errorf("%w: %w", errIDToken, err)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("%w: signature encoding: %w", errIDToken, err)
	}
	if err := verifyJWS(header.Alg, key, []byte(parts[0]+"."+parts[1]), signature); err != nil {
		return fmt.Errorf("%w: %w", errIDToken, err)
	}
	var claims struct {
		Iss   string          `json:"iss"`
		Aud   json.RawMessage `json:"aud"`
		Exp   *float64        `json:"exp"`
		Nonce string          `json:"nonce"`
	}
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		return fmt.Errorf("%w: claims: %w", errIDToken, err)
	}
	switch {
	case strings.TrimRight(claims.Iss, "/") != strings.TrimRight(check.issuer, "/"):
		return fmt.Errorf("%w: issuer %q, want %q", errIDToken, claims.Iss, check.issuer)
	case !audienceContains(claims.Aud, check.clientID):
		return fmt.Errorf("%w: audience does not name client %q", errIDToken, check.clientID)
	case claims.Exp == nil:
		return fmt.Errorf("%w: no exp", errIDToken)
	case !now.Before(time.Unix(int64(*claims.Exp), 0).Add(idTokenLeeway)):
		return fmt.Errorf("%w: expired", errIDToken)
	case check.nonce != "" && claims.Nonce != check.nonce:
		return fmt.Errorf("%w: nonce does not match the one sent", errIDToken)
	}
	return nil
}

func decodeJWTPart(part string, into any) error {
	raw, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, into)
}

// algorithmAllowed reports whether alg is verifiable and, when the issuer
// advertises its algorithms, one of them.
func algorithmAllowed(alg string, advertised []string) bool {
	if !slices.Contains(verifiableAlgorithms, alg) {
		return false
	}
	return len(advertised) == 0 || slices.Contains(advertised, alg)
}

// audienceContains reads aud as a string or an array.
func audienceContains(aud json.RawMessage, clientID string) bool {
	var one string
	if json.Unmarshal(aud, &one) == nil {
		return one == clientID
	}
	var many []string
	return json.Unmarshal(aud, &many) == nil && slices.Contains(many, clientID)
}

// jwk is one JSON Web Key, as far as verification reads it.
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

// jwksCache holds each issuer's keys by JWKS URL.
var jwksCache = struct {
	sync.Mutex
	entries map[string]jwksEntry
}{entries: map[string]jwksEntry{}}

type jwksEntry struct {
	keys    map[string]jwk
	fetched time.Time
}

// issuerKey returns the key named kid, from the cache or a fetch. An unknown
// kid refetches the keys once, for an issuer that has rotated them.
func issuerKey(ctx context.Context, client *http.Client, jwksURL, kid string) (jwk, error) {
	jwksCache.Lock()
	entry, cached := jwksCache.entries[jwksURL]
	jwksCache.Unlock()
	fresh := cached && time.Since(entry.fetched) < jwksCacheTTL
	if fresh {
		if key, ok := entry.keys[kid]; ok {
			return key, nil
		}
	}
	keys, err := fetchJWKS(ctx, client, jwksURL)
	if err != nil {
		return jwk{}, err
	}
	jwksCache.Lock()
	jwksCache.entries[jwksURL] = jwksEntry{keys: keys, fetched: time.Now()}
	jwksCache.Unlock()
	key, ok := keys[kid]
	if !ok {
		return jwk{}, fmt.Errorf("no key %q in the issuer's keys", kid)
	}
	return key, nil
}

func fetchJWKS(ctx context.Context, client *http.Client, jwksURL string) (map[string]jwk, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("keys request: %w", err)
	}
	identityOf(ProviderConfig{}).setUserAgent(req)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch the issuer's keys: %w", err)
	}
	defer closeResponseBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch the issuer's keys: HTTP %d", resp.StatusCode)
	}
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, jwksLimit)).Decode(&set); err != nil {
		return nil, fmt.Errorf("decode the issuer's keys: %w", err)
	}
	keys := make(map[string]jwk, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kid != "" && (k.Use == "" || k.Use == "sig") {
			keys[k.Kid] = k
		}
	}
	return keys, nil
}

// verifyJWS checks signature over input with key, for alg.
func verifyJWS(alg string, key jwk, input, signature []byte) error {
	if key.Alg != "" && key.Alg != alg {
		return fmt.Errorf("key %q is for %s, not %s", key.Kid, key.Alg, alg)
	}
	switch alg {
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512":
		pub, err := rsaKey(key)
		if err != nil {
			return err
		}
		hash, digest := jwsDigest(alg[2:], input)
		if alg[0] == 'P' {
			return rsa.VerifyPSS(pub, hash, digest, signature, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
		}
		return rsa.VerifyPKCS1v15(pub, hash, digest, signature)
	case algES256, algES384:
		pub, err := ecKey(key, alg)
		if err != nil {
			return err
		}
		size := (pub.Curve.Params().BitSize + 7) / 8
		if len(signature) != 2*size {
			return errors.New("bad ECDSA signature length")
		}
		_, digest := jwsDigest(alg[2:], input)
		r, s := new(big.Int).SetBytes(signature[:size]), new(big.Int).SetBytes(signature[size:])
		if !ecdsa.Verify(pub, digest, r, s) {
			return errors.New("bad signature")
		}
		return nil
	case "EdDSA":
		if key.Kty != "OKP" || key.Crv != "Ed25519" {
			return fmt.Errorf("key %q is not an Ed25519 key", key.Kid)
		}
		pub, err := base64.RawURLEncoding.DecodeString(key.X)
		if err != nil || len(pub) != ed25519.PublicKeySize {
			return errors.New("bad Ed25519 key")
		}
		if !ed25519.Verify(pub, input, signature) {
			return errors.New("bad signature")
		}
		return nil
	}
	return fmt.Errorf("algorithm %q is not allowed", alg)
}

// jwsDigest hashes input for a "256", "384" or "512" algorithm suffix.
func jwsDigest(bits string, input []byte) (crypto.Hash, []byte) {
	switch bits {
	case "384":
		sum := sha512.Sum384(input)
		return crypto.SHA384, sum[:]
	case "512":
		sum := sha512.Sum512(input)
		return crypto.SHA512, sum[:]
	default:
		sum := sha256.Sum256(input)
		return crypto.SHA256, sum[:]
	}
}

func rsaKey(key jwk) (*rsa.PublicKey, error) {
	if key.Kty != "RSA" {
		return nil, fmt.Errorf("key %q is not an RSA key", key.Kid)
	}
	n, errN := base64.RawURLEncoding.DecodeString(key.N)
	e, errE := base64.RawURLEncoding.DecodeString(key.E)
	if errN != nil || errE != nil || len(e) == 0 || len(e) > 4 {
		return nil, fmt.Errorf("bad RSA key %q", key.Kid)
	}
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	if pub.N.BitLen() < minRSABits {
		return nil, fmt.Errorf("RSA key %q is under %d bits", key.Kid, minRSABits)
	}
	return pub, nil
}

// ecKey decodes an EC key on the curve alg requires, validating the point.
func ecKey(key jwk, alg string) (*ecdsa.PublicKey, error) {
	want, curve := "P-256", elliptic.P256()
	if alg == algES384 {
		want, curve = "P-384", elliptic.P384()
	}
	if key.Kty != "EC" || key.Crv != want {
		return nil, fmt.Errorf("key %q is not a %s key", key.Kid, want)
	}
	x, errX := base64.RawURLEncoding.DecodeString(key.X)
	y, errY := base64.RawURLEncoding.DecodeString(key.Y)
	size := (curve.Params().BitSize + 7) / 8
	if errX != nil || errY != nil || len(x) != size || len(y) != size {
		return nil, fmt.Errorf("bad EC key %q", key.Kid)
	}
	pub, err := ecdsa.ParseUncompressedPublicKey(curve, append(append([]byte{4}, x...), y...))
	if err != nil {
		return nil, fmt.Errorf("EC key %q is not on its curve", key.Kid)
	}
	return pub, nil
}
