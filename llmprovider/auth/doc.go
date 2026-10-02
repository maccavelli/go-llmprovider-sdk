// Package auth holds the credentials that sign in and refresh: OAuth
// sessions, the browser and device login flows (OpenAI's, Grok's and Kilo's),
// revocation and id_token checks, vendor CLI sessions, and the stores that
// keep sessions (0015-MADR D2, amendment "what `auth` holds"). A session is a
// llmprovider.TokenSource; pass it to a provider with
// llmprovider.WithTokenSource.
//
// The default OAuth client ids are the vendor CLIs' own public clients:
// Codex's for OpenAI and the Grok CLI's for xAI. Subscription OAuth works only
// with them, so they are borrowed, and a vendor may restrict them at any time
// (0016-MADR D7 and Consequences). OAuthFlowOptions.ClientID overrides them.
package auth
