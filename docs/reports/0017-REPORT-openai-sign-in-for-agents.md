---
status: observation
date: 2026-10-14
---
# OpenAI's Sign in with ChatGPT for Open-Source Agents: The Flow, Its Terms, and What `llmprovider` Lacks

## Purpose

[0017-MADR-together-provider-and-auth-extensions.md](../decisions/0017-MADR-together-provider-and-auth-extensions.md)
D4 records a public-client OpenAI flow with dynamic client registration, seen
in pi's source, and leaves it to a later amendment to decide "whether it
becomes an OpenAI auth method". Its evidence was "one day old and comes from
one client", and "No OpenAI document was read".

On 2026-10-14 the owner of gobble-cli, this module's consumer, asked whether
that flow keeps the ChatGPT subscription login gobble's users would expect
while following OpenAI's policy, and said it should be adopted if so. The
question arose because this module's ChatGPT logins sign in with the Codex
CLI's client id. gobble-cli recorded the terms check behind that question in
gobble-cli `docs/decisions/0005-PLAN-v1-feature-scope.md`, entry "F4-3b made
executable".

This report records what OpenAI's documents, pi's source, this module's
source and two live probes show. It decides nothing. The amendment to
0017-MADR D4 decides.

## Method

- **OpenAI's documents.** Every page under
  `https://developers.openai.com/siwc/token-sharing-open-source/`, and the
  `siwc`, `siwc/quickstart`, `siwc/ui-ux-guidelines` and
  `siwc/request-client-id` pages, were fetched directly on 2026-10-14 and
  read as text in full. None shows a date. The Sign in with ChatGPT Terms
  answer 403 to a direct fetch; they were read from the Wayback Machine's
  snapshot `20261007132710` of
  `https://openai.com/policies/sign-in-with-chatgpt-terms/`, which is dated
  29 September 2026.
- **Source.** pi at `312184edb`; this module at `v1.13.0` (`c5bc023`), the
  latest tag, whose five later commits on `main` are catalog work and one plan update.
- **Probes.**
  - The client-id probe was run by gobble-cli's owner on 2026-10-14. It sent
    this module's device and browser requests with a client id gobble chose
    in each vendor's shape, and with an unregistered control. No vendor
    CLI's id was sent and no user signed in.
  - This module's own probe, `live_openai_signin_probe_test.go` (0017-PLAN
    U4), has not been run.
- **Marks.** **(V)** is a claim read directly in the cited source or page.
  **(O)** is a probe's observed output. A quotation is the source's text,
  unchanged.

## F1 — The flow (V)

- **Registration.** The first sign-in for an account sends
  `client_id=dynamic_agent_client`, the app's name as `agent_name_hint`, and
  a host identifier, `ext_agent_host_id`. "After the user names and
  authorizes the agent, the callback returns its issued client_id", such as
  `oaiapp_…`, saved per ChatGPT account. "dynamic_agent_client is the
  first-time registration entrypoint, not the client ID to save or use for
  token exchange." It needs "neither a client secret nor a partner API key"
  (`siwc/token-sharing-open-source/sign-in`).
- **Each issued client is bound** "to the authenticated user and the
  workspace selected during registration". A later sign-in reuses it, with
  `id_token_hint` and optionally `login_hint`, and omits `agent_name_hint`.
  A callback that supplies a different client id is rejected
  (`sign-in`, `profiles-and-sessions`).
- **The host identifier** is persisted before the first sign-in and kept for
  the host's life: `urn:ietf:params:oauth:jwk-thumbprint:…` (recommended),
  `urn:uuid:…` or `did:key:…`. It is "not an authentication credential", and
  must not be an email or user id (`token-sharing-open-source`).
- **Authorization request.**
  - Endpoint: `https://auth.openai.com/api/accounts/authorize`.
  - Scopes: `openid profile email offline_access resource.invoke chatgpt.tokens.use.direct`.
  - `resource=https://api.openai.com/v1`, with PKCE `S256`, `state` and `nonce`.
  - `redirect_uri` is an HTTP loopback on `127.0.0.1`, path `/auth/callback`;
    "only the port may vary", and "Do not substitute with localhost".
- **Exchange.** A form `authorization_code` grant to
  `https://auth.openai.com/api/accounts/oauth/token`, with the issued client
  id, the same `redirect_uri` and the same `resource`.
- **Validation.**
  - The ID token is checked against OpenAI's JWKS, with the audience set to
    the issued client id, and its `nonce` against the one sent.
  - The granted scopes must include `chatgpt.tokens.use.direct`: "A valid ID
    token alone does not authorize ChatGPT plan usage."
- **There is no device flow.** For a remote VM the guide is to "Complete
  OAuth locally" and transfer the credential file "over a secure channel such
  as SSH" (`self-hosted-vms`).
- **Tokens** (`token-reference`):
  - Access tokens are "valid for one hour"; refresh tokens "for 30 days", and
    each refresh returns a replacement.
  - The access token's audience is `https://api.openai.com/v1`, and its
    `client_id` claim is the issued id.
- **Refresh** sends the issued client id (not `dynamic_agent_client`), the
  refresh token and `resource`, and omits `scope`. It must be serialised
  "so two processes do not race a rotating token" (`profiles-and-sessions`).
- **Sign-out** revokes the refresh token at the discovery document's
  `revocation_endpoint`, with the issued client id. "An empty HTTP 200 is the
  revocation success response, including for an already-invalid token."
- **Storage.** One record per issued client id and verified identity, written
  "atomically with owner-only permissions (0600 on Unix)". The documented
  record holds `client_id`, `ext_agent_host_id`, `id_token`, `access_token`,
  `refresh_token`, `expires_in`, `scopes` and `saved_at` (`sign-in`, step 5).

## F2 — Where the token is used, and what a request may carry (V)

- **Only the public Responses API.** "Use the public Responses API endpoint
  above for this ChatGPT plan usage flow; do not point it at ChatGPT's
  backend-api endpoints" (`models-and-inference`).
  `subscription_sharing_route_not_supported` (403) answers another route
  (`errors-and-recovery`).
- **Models.** `GET https://api.openai.com/v1/models` with the access token
  returns "the selected ChatGPT account's model catalog" as a `models`
  array, each with `slug`, `display_name` and `visibility`. A picker shows
  `visibility: "list"`. This is not the platform's `data` array.
- **Request rules** (`preview-limitations`):
  - "Set store: false and stream: true."
  - System prompts go in `instructions`, since "explicit {type: "message",
    role: "system"} items are rejected".
  - Fifteen fields are unsupported and omitted, `max_output_tokens`,
    `prompt_cache_retention` and `temperature` among them; `prompt_cache_key`
    is not listed. "Omit previous_response_id over HTTP and send the required
    history in input."
  - "Supported tools: Group function/custom tools in namespaces or supply
    them through additional_tools input items."
  - "Unsupported tools: … hosted MCP/connectors, and Responses tool_search."
  - Text, images and files are accepted when the model takes them.
- **Errors.**
  - Before a stream opens, admission may answer 401, 403 or 503 with a body
    such as `{"detail":"..."}`, not the API's error object.
  - After it, `subscription_sharing_*` codes, among them:

    | Code | Status |
    | :--- | :--- |
    | `subscription_sharing_usage_limit_exceeded` | 429 |
    | `subscription_sharing_unsupported_capability` | 400, with `error.param` |
    | `subscription_sharing_user_not_eligible` | 403 |

  - A usage-limit error "can arrive as response.failed … after streaming has
    begun". "OpenAI does not silently switch the request to another billing
    path."
- **Usage.** "For ChatGPT Plus users, the five-hour usage limit is shared
  across all apps where they use their ChatGPT plan"; it does not apply to
  Pro (`profiles-and-sessions`).

## F3 — The terms (V)

The Sign in with ChatGPT Terms, 29 September 2026, snapshot of 2026-10-07:

- **§1, identity.**
  - "Use your app's own name during sign-in and activation. Do not
    impersonate OpenAI, another application, or another open-source
    project."
  - "Obtain and use SIWC access and refresh tokens ('Authentication
    Tokens') only through OpenAI's supported sign-in flow and as authorized
    by the user."
  - "Any persistent storage of Authentication Tokens must be local and under
    the user's control, not in a remote or managed environment."
- **§2, plan usage.**
  - "Requests must originate from the user's local runtime or a remote
    runtime only that user controls."
  - "Requests must be for the authenticated user and arise from their
    activity or expressly authorized automations or background processes.
    Obtain express consent before background use."
  - "Another user's activity must not trigger requests to the authenticated
    user's account."
  - "Connected application only. Use the user's plan only for the application
    they connected. Do not provide general-purpose API access for other tools
    or unrelated requests."
  - "No charge."
- **§4, prohibited.**
  - "Pooling, transferring, reselling, gifting, or sharing ChatGPT plan usage
    or Authentication Tokens, except for token handling expressly permitted by
    these Terms."
  - "Using one user's subscription to fulfill another user's requests."
  - "Alter, reverse engineer, or otherwise amend the SIWC software."
- **§5, branding.** "Use OpenAI names, logos, and buttons only as authorized."
- **Availability** (`siwc/quickstart`): "ChatGPT plan usage is available to
  all open-source partners and selected private clients." For a paid or
  remotely hosted app, an interest form applies (`token-sharing-open-source`).
- **UI guidance** (`siwc/ui-ux-guidelines`):
  - a one-time "You're using your ChatGPT plan" confirmation;
  - "Using ChatGPT plan" near the composer or model selector, with a "Manage
    usage" link;
  - on a usage-limit error, "Manage usage" as the primary action.

## F4 — A client id cannot be chosen (O)

The client-id probe sent this module's requests with an id in each vendor's
shape: `app_` and 24 characters for OpenAI, a UUIDv5 for xAI. The control was
`not-a-registered-client`.

- **OpenAI.**
  - Both ids, on the device request and the authorize request, were answered
    `400` `invalid_client`: "This app is unavailable. Contact the app owner
    and try again later."
  - The discovery document lists no `registration_endpoint`. Its
    `authorization_endpoint` is `https://auth.openai.com/api/accounts/authorize`,
    F1's endpoint, and its grant types are `authorization_code` and
    `refresh_token`.
- **xAI.**
  - The control failed validation ("client_id is required"; "Missing or
    invalid client_id.").
  - The UUID passed validation and was refused as unregistered ("Unknown or
    disabled client"; "The client_id is not recognized.").
  - The discovery document lists no `registration_endpoint`.

An OAuth client id is a registration, not a format. For OpenAI, the issued id
from F1 is the only one an app can hold as its own.

## F5 — pi's implementation (V)

- **Login.** `packages/ai/src/auth/oauth/openai-chatgpt.ts` (`02eed88fd`,
  2026-09-29) implements F1:
  - `DYNAMIC_CLIENT_ID = "dynamic_agent_client"` and `AGENT_NAME_HINT = "Pi"`;
  - the authorize and token URLs, `RESOURCE` and the direct scope (`:15-27`);
  - the issued client id read from the callback (`:53-61`);
  - a pasted callback URL as the fallback when the listener cannot start.
- **Requests.** `packages/ai/src/api/openai-responses.ts`:
  - treats any credential sent to `https://api.openai.com/v1` that does not
    start with `sk-` as a sign-in token (`isChatGPTSignIn`, `:36-47`);
  - for one, omits `prompt_cache_retention`, `prompt_cache_options`,
    `max_output_tokens` and `temperature`, and always sends `stream: true`
    and `store: false` (`:327-346`);
  - still sends `prompt_cache_key`;
  - sends its tools as top-level Responses tools (`:352-357`), not in
    namespaces;
  - appends `chatgpt.com/settings/usage` to a
    `subscription_sharing_usage_limit_exceeded` message (`:227-228`).
- **Two OpenAI logins.** pi keeps the Codex-id login (`openai-codex.ts:22`)
  as a separate provider beside this one.

## F6 — What this module has, and what it sends (V)

- **Logins.**
  - Browser and device logins for OpenAI use `DefaultOpenAIClientID`, the
    Codex CLI's (`llmprovider/auth/oauth_constants.go:5-13`). The package doc
    says the ids are "borrowed, and a vendor may restrict them at any time"
    (`auth/doc.go:8-11`).
  - The wizard's flows pass no `ClientID`, so they take the default
    (`wizard/auth.go:549-553`, `auth/oauth_loopback.go:186-200`).
  - prepare-commit-msg, at `v1.12.0`, signs in through the wizard
    (`internal/ui/setup.go:264-277` there), so its ChatGPT login presents the
    Codex CLI's id.
- **The session.** `OAuthSession` holds `Issuer`, `ClientID`, `AccountID`,
  `FedRAMP` and `TokenURL` (`auth/tokenstore.go:21-64`). It has no `resource`,
  scopes, ID token or host id, all of which F1 needs for refresh and
  re-sign-in.
- **The refresh lock.** `OAuthSession` calls `Save` while it holds
  `LockRefresh` (`auth/oauth_session.go:740-768`). A store whose save takes
  the same lock as its refresh deadlocks.
- **Backends.** The `openai` provider has two backends. A ChatGPT session
  uses `ChatGPTBaseURL`, `https://chatgpt.com/backend-api/codex`
  (`providers/openai/openai.go:111`). An API key, or a static token, uses
  the platform (`openai.go:1-8`).
- **What the platform request sends** (`openai.go:298-345`), set against F2:

  | Sent today | F2's rule |
  | :--- | :--- |
  | `max_output_tokens`, always (`:315-321`) | unsupported |
  | `store` only with `WithStore` (`:322-328`) | `false` required |
  | `previous_response_id` when the request has one (`:342-344`) | omitted over HTTP |
  | native `tool_search` from GPT-5.4 on (0040-MADR D3) | unsupported |
  | a key's model listing, probed with "one short, billed generation" per model (`openai.go:420-422`, 0038-MADR D4) | each probe would spend the user's plan |

- **The probe** (`llmprovider/live_openai_signin_probe_test.go`) runs F1's
  browser flow under the agent name `go-llmprovider-sdk` and model
  `gpt-6-luna`. It then makes one text request and one tool call through
  `openai.New` with the access token as the API key, and logs each refusal.
  It covers neither refresh, revocation nor the model listing.

## What remains open for 0017-MADR D4's amendment

Observations only; the amendment decides.

- **Tools.** F2 says to group function tools "in namespaces or … additional_tools",
  and pi sends them top-level. Only the probe's tool call shows which the
  service accepts today.
- **The unsupported fields.** Whether the service refuses each listed field,
  or ignores some, is unmeasured. The probe sends `max_output_tokens`, so its
  text case answers for that field.
- **Where the token goes in the SDK.** Pi detects the flow by a non-`sk-`
  key. This module chooses the backend by the kind of credential, so a
  sign-in session could be a third kind, with its own request rules, listing
  and errors.
- **The session record.** F1's record has fields `OAuthSession` lacks. A
  consumer's store, such as gobble-cli's `auth.json`, keeps what the session
  carries.
- **The terms in a consumer.** §2's background-use consent, and §2's
  "general-purpose API access" line, depend on how a consumer runs. An SDK
  can carry the consent as an option and expose the usage URL; it cannot
  meet them alone.
- **The probe's runs.** Each run registers a new client under the account,
  listed in its ChatGPT settings. It runs only on the owner's request
  (0017-PLAN U4).
- **The pages carry no dates,** and the limits are titled "Preview
  limitations".

## Sources

| Claim | Source |
| :--- | :--- |
| The flow, host id, tokens, refresh, revocation, storage | `https://developers.openai.com/siwc/token-sharing-open-source` and its `sign-in`, `profiles-and-sessions`, `token-reference`, `self-hosted-vms` pages, 2026-10-14 |
| The route, models, request rules, errors, usage | the `models-and-inference`, `preview-limitations` and `errors-and-recovery` pages, 2026-10-14 |
| Availability and UI guidance | `https://developers.openai.com/siwc/quickstart`, `/siwc/ui-ux-guidelines`, 2026-10-14 |
| The terms | `https://openai.com/policies/sign-in-with-chatgpt-terms/`, 29 September 2026, Wayback snapshot `20261007132710` |
| The client-id probe | gobble-cli `docs/decisions/0005-PLAN-v1-feature-scope.md`, entry "F4-3b made executable", "The client-id probe" |
| pi's login and requests | pi `312184edb`: `packages/ai/src/auth/oauth/openai-chatgpt.ts`, `packages/ai/src/api/openai-responses.ts` |
| This module's logins, session, lock, backends and requests | `v1.13.0`: `llmprovider/auth/*.go`, `wizard/auth.go`, `llmprovider/providers/openai/openai.go` |
| prepare-commit-msg's login | prepare-commit-msg `a04b617`: `internal/ui/setup.go` |
