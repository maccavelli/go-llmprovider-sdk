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
// OpenAI's terms for plan usage ask the app to obtain the user's express
// consent before any background use, and this package carries that consent
// (0075-MADR). The plan method states that it needs the consent
// "chatgpt_plan.background_use" (llmprovider.AuthMethod.Consent), and its
// sign-in asks it before the browser opens, as one question. The answer is
// kept on the session (OAuthSession.Consents, read with Consent) and saved by
// FileTokenStore. SetConsent changes the stored answer, holding the
// provider's refresh lock, and a refresh never loses it. A request the
// consumer marks with llmprovider.WithBackground is refused by the openai
// provider, before the network, on a plan session whose stored answer
// (StoredConsent) is not granted. The consumer still decides which of its
// requests are background, and words the question.
//
// The host id OpenAI asks kept for the host's life is the consumer's
// (ChatGPTPlanOptions.HostID), and the client OpenAI issued the app
// outlives the session in a ClientStore, which FileTokenStore implements in
// <provider>.client and a sign-out leaves, so that a later sign-in signs in
// to it (ChatGPTPlanOptions.ClientID) instead of registering another.
//
// The rest of OpenAI's terms for plan usage this package cannot do for the
// consumer, and no option claims to (0043-MADR D8, as 0075-MADR amends it).
// The consumer sends requests only from the user's own runtime, or one that
// only that user controls, and only for that user's activity. It uses the
// plan only for the application the user connected, never as general-purpose
// API access for other tools. It does not pool, share, transfer or resell
// plan usage or the tokens, and it charges nothing for this use.
package auth
