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

// SetTokenHeader sets tok on req as Token describes it: in header, after
// scheme, unless tok names a Header of its own (R16; 0016-MADR D2, A6).
//
// Temporary export for the provider packages (0015-PLAN S7); S7b moves it to internal/transport.
func SetTokenHeader(req *http.Request, tok Token, header, scheme string) {
	name, value := tokenHeader(tok, header, scheme)
	req.Header.Set(name, value)
}
