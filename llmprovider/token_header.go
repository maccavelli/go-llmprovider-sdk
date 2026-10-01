package llmprovider

import "net/http"

// bearerScheme is the scheme of a bearer token in an Authorization header.
const bearerScheme = "Bearer"

// tokenHeader is the header name and value that send tok (0016-MADR D2, A6).
// With no tok.Header, the value goes in header, after scheme when scheme is
// non-empty. With one, it goes in tok.Header instead: "Bearer <value>" for a
// TokenBearer, the bare value for any other Type.
func tokenHeader(tok Token, header, scheme string) (name, value string) {
	if tok.Header != "" {
		header, scheme = tok.Header, ""
		if tok.Type == TokenBearer {
			scheme = bearerScheme
		}
	}
	if scheme == "" {
		return header, tok.Value
	}
	return header, scheme + " " + tok.Value
}

// Apply sets t on req as the Token describes it (R16; 0016-MADR D2, A6): in
// header, after scheme, which are the service's own; or, when t names a Header
// of its own, there instead, "Bearer "-prefixed only for a TokenBearer. A
// provider calls it on every request, with the service's header and scheme:
// "Authorization" and "Bearer", or "x-api-key" and "".
func (t Token) Apply(req *http.Request, header, scheme string) {
	name, value := tokenHeader(t, header, scheme)
	req.Header.Set(name, value)
}
