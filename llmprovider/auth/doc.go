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
//
// LoginChatGPTPlan is OpenAI's Sign in with ChatGPT for agents (0043-MADR):
// the user's ChatGPT plan, under the consumer's own app name, with a client
// that OpenAI registers for the app at the first sign-in, so no id is
// borrowed. Its session is a plan session (OAuthSession.PlanUsage), which
// the openai provider sends to the platform's Responses API.
//
// OpenAI's terms for plan usage ask things of the app that this package
// cannot do for it, and no option claims to (0043-MADR D8). The consumer
// sends requests only from the user's own runtime, or one that only that
// user controls, and only for that user's activity. It obtains the user's
// express consent before any background use. It uses the plan only for the
// application the user connected, never as general-purpose API access for
// other tools. It does not pool, share, transfer or resell plan usage or the
// tokens, and it charges nothing for this use.
package auth
