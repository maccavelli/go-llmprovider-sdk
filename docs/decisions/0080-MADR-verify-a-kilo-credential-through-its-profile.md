---
status: accepted
date: 2026-11-16
decision-makers: repository owner
consulted: "the owner's ruling of 2026-11-16, relayed by the coordinator (its label quoted below), and the owner's answers to Q1 to Q7 and approval of the same day (below, verbatim); this repository at ceb489e (v1.31.0 and two records commits); the kilocode checkout at 58072bd696; gobble-cli at 1305a61 (branch p3-acpext-v0.5.0), and its 0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md, D4's notes of 2026-11-16, at cecc892 on the same branch; live probes of Kilo's endpoints with made-up tokens, 2026-11-16"
informed: gobble-cli, which checks a new credential by this module's rule on its terminal path (wizard.ResolveCredential) and on its ACP path (its own checkListing); every caller of wizard.ConfigureLLM, wizard.ResolveCredential and auth.KiloProfile
---

<!-- markdownlint-disable MD013 MD024 -->

# Verify a Kilo Credential Against Kilo's Profile at Sign-In When No Organization Is Set, Because Kilo's Public Listing Refuses No Key

## Context and Problem Statement

The wizard checks a new credential by listing the provider's models with it,
and saves it only once the listing has not refused it
([0059-MADR-wizard-debugging-pass-findings.md](0059-MADR-wizard-debugging-pass-findings.md)
W2, Q2 (a), C38). A typed key the listing refuses is asked for again, up to
three times; a refused key from anywhere else ends the run naming its source
([0028-MADR-heuristics-and-performance-from-the-research-pass.md](0028-MADR-heuristics-and-performance-from-the-research-pass.md)
D-H10, C40). "Refused" is `refusedCredential`: `ErrAuthFailure` and not
`ErrNotPermitted` (D-H2; C12, C39). gobble-cli copies that rule for its ACP
sign-ins (G1 to G3) and gets it from the wizard on its terminal path (G4, G5).

For Kilo with no organization that check is blind. The listing the SDK reads
with no organization, `https://api.kilo.ai/api/gateway/models` (C1, C16),
does not look at the `Authorization` header: it answers 200 with the same
body to no credential, to a made-up bearer token and to a made-up expired
token shaped like a JWT (probes L1 to L6). The code already says so: "The
public endpoint answers with no credential" (C3). So, with no organization, a
wrong Kilo key passes the wizard's check, is saved, and fails on first use.
Kilo's profile, `https://api.kilo.ai/api/profile`, does look at it: it
answers 401 to no credential and to each made-up token (L13 to L15).

### The owner's ruling of 2026-11-16

Relayed by the coordinator; its label, verbatim: **"In the SDK, with its next
record"**.

As it was put to the owner: a go-llmprovider-sdk decision record and plan;
for Kilo with no organization the wizard verifies the credential against the
profile endpoint (a 401 refuses; an outage does not fail the sign-in); the
plan's first step confirms which key kinds the profile accepts; it shares one
SDK release with a typed "tool schema refused" error that is queued for a
record of its own; gobble then bumps its pin. Options shown and not chosen:
in gobble now and the SDK later; document it only.

**What the ruling settles** (D1, D11): where the check goes (this module,
this record), its direction (the profile, for Kilo with no organization, in
the wizard), that a 401 refuses, that an outage does not fail a sign-in, that
the PLAN's first phase settles the key kinds, the shared release, and that
gobble adopts by a pin bump. **What it leaves** is every other choice below;
each is an open question (Q1 to Q7), with a recommendation, and is
`proposed` until the owner answers.

### The owner's answers and approval of 2026-11-16

*(Added 2026-11-16.)* The owner answered Q1 to Q7 the same day, each as the
draft recommended. Each answer is recorded under its question in "Open
questions for the owner", with the question as it was put and the options
not chosen. The record is `accepted` from that date.

The approval, the owner's ruling, relayed by the coordinator: **"Approve and
run through phase 4"**. As it was put: the answers are recorded, the pair is
set to accepted and committed under the standing approval; after the owner
has run Phase 1's live steps, the tests, the implementation and the gates
run without stopping, each reviewed and committed by the coordinator; the
release still waits for the owner's word. It was not the option the
coordinator recommended. Options shown and not chosen: approve and run
Phase 0 only, then stop; record the answers and do not approve yet.

### Every way a Kilo credential reaches a store through this module

Read at `ceb489e`. "No organization" is the public listing (C1); "with an
organization" is the organization's listing (C2), which authenticates (L7
to L12).

| # | Path | Check today | Refuses a wrong credential? |
| :--- | :--- | :--- | :--- |
| P1 | Wizard, API key typed (C31) | the listing; an API key never has an organization: `credentialResult` takes the credential's, which a key does not set (C15) | no |
| P2 | Wizard, API key from `KILO_API_KEY` under `AllowEnv` (C33) | the same | no |
| P3 | Wizard, API key kept from `Options.Existing` | the same | no |
| P4 | Wizard, device sign-in as a session (C32) | the profile is read for the organization menu (C22); its failure, a 401 included, is a warning and the personal account (C23); then the listing | with an organization chosen: yes. With the personal account: no |
| P5 | Wizard, device sign-in as a key (`Options.KiloDeviceKey`) | no profile is read (C24, C37); the listing, with no organization | no |
| P6 | Wizard, a kept session (`Existing.Kind` OAuth) | `validateKept` checks only that it is Kilo's (C27); the listing, with `Existing.Organization` (C26) | with an organization: yes. Without: no |
| P7 | `signin.Start(…, ProviderKilo, AuthDeviceCode, …)` and `Wait`, a `DeviceKey` (C28) | none: the handle saves nothing and checks nothing; the caller does (C29, C42) | the caller's rule: gobble lists with the key (G1, G2), so no |
| P8 | A caller that builds `kilo.New` or calls `catalog.List` with a key itself | not a sign-in; the first generation | (at first use) |

A fresh device token (P4, P5, P7) was issued by Kilo moments before: it is
wrong only if Kilo is. The real exposure is a key a person types or pastes
(P1), and a key or session kept or read from the environment (P2, P3, P6).

### `auth.KiloProfile` as a verifier

Read at `ceb489e`:

* It builds `{origin}{prefix}/api/profile` from the configured base URL, or
  from a token that embeds a base URL, as the gateway's endpoints are built
  (C6, C7; Kilo's `getKiloUrlFromToken`, K9).
* It sends the token as a bearer with the caller's client and User-Agent,
  bounded by 30 s (C9), with no retry.
* A non-2xx is `llmprovider.ClassifyHTTPError("kilo", …)` (C8). A 401 is an
  `*APIError` of kind `ErrAuthFailure` (C10), so `refusedCredential` holds
  and `signin.ErrorKindOf` gives `auth`. Any Kilo 403 is `ErrNotPermitted`
  (C11), so `refusedCredential` does not hold. A 5xx is
  `ErrProviderUnavailable`; a transport failure `ErrOffline`.
* A live device token has been read through it: 0046-PLAN's run of
  `TestLive_KiloDeviceLogin` on 2026-10-19 passed "the login, the token's
  shape and the profile" (C34, C35).

### What Kilo's own client does (read, the `kilocode` checkout at `58072bd696`)

* Its `fetchProfile` reads `{KILO_API_BASE}/api/profile`, the API origin
  (`KILO_API_URL` or `https://api.kilo.ai`), not a URL a token embeds (K1,
  K3). It reads 401 and 403 alike as "Invalid token" (K2).
* Its terminal device sign-in reads the profile right after the token and
  fails the sign-in when that throws (K5); its TUI sign-in ignores a failure
  (K6).
* It reads the profile only for a device session: its server's profile
  handler refuses a credential stored as an API key (K10), so Kilo's client
  never shows whether the profile accepts a dashboard API key.
* Its listing retries without the token on a 401 with no organization (K7);
  an organization lists at `{gateway}/organizations/{id}/models` (K8), on
  its default gateway `https://ai-gateway.kilo.ai/api/v1` (K4).
* Kilo's documentation: API keys "are JWT tokens tied to your Kilo account"
  (K11), copied from "Your Profile" on the personal account (K14);
  "Organization tokens are scoped with a 15-minute expiry" (K12); the
  gateway serves free models to anonymous requests (K13).

### Live probes of 2026-11-16 (made-up tokens only)

No real credential was read or sent. "made-up" is a random bearer token;
"JWT-shaped" is a made-up token in JWT form, its signature random and its
`exp` in the past. Bodies were compared by hash and then deleted.

| Id | Request | No credential | Made-up | JWT-shaped | Mark |
| :--- | :--- | :--- | :--- | :--- | :--- |
| L1–L3 | `GET https://api.kilo.ai/api/gateway/models` | 200 | 200 | 200, all three bodies byte-identical | verified by the coordinator, re-run by me |
| L4–L6 | `GET https://ai-gateway.kilo.ai/api/v1/models` | 200 | 200 | 200, identical to L1 | verified by the coordinator, re-run by me |
| L7–L8 | `GET https://api.kilo.ai/api/organizations/{made-up id}/models` with the organization header, the URL this module lists an organization at (C2) | 401 | 401 | – | new: the coordinator probed L9's URL, not this one |
| L9–L10 | `GET https://api.kilo.ai/api/gateway/organizations/{made-up id}/models` | 401 | 401 | – | verified by the coordinator, re-run by me |
| L11–L12 | `GET https://ai-gateway.kilo.ai/api/v1/organizations/{made-up id}/models`, Kilo's client's URL (K8) | 401 | 401 | – | new |
| L13–L15 | `GET https://api.kilo.ai/api/profile` | 401, body error "Unauthorized" | 401, body error "Invalid token (…)" with a request id | 401, "Invalid token (…)" | L13–L14 verified by the coordinator, re-run by me; L15 and the two bodies new |
| L16 | `GET /api/profile` with the organization header, made-up token | – | 401, "Invalid token (…)" | – | new |
| L17–L18 | `GET https://api.kilo.ai/api/profile/balance` | 401 "Unauthorized" | 401 "Invalid token (…)" | – | new |
| L19–L20 | `GET https://api.kilo.ai/api/defaults` | 200 | 200, identical | – | new: does not authenticate |
| L21–L22 | `POST https://api.kilo.ai/api/gateway/chat/completions` with an existing paid model and an empty message list | 401, code `PAID_MODEL_AUTH_REQUIRED` | 401, code `INVALID_TOKEN`, `WWW-Authenticate: Bearer error="invalid_token"` | – | new |
| L23–L24 | the same with an existing free model | 400, "Input required" | 401, code `INVALID_TOKEN` | – | new: the bearer is checked before the body |
| L25 | the same with an empty body, at this URL and at `https://ai-gateway.kilo.ai/api/v1/chat/completions` | 404 "Model not found" | 404 "Model not found" | – | new: the model is checked first |

Every response came from Vercel. None carried `Retry-After` or a rate-limit
header. What a real key gets is not known from these probes: a dashboard API
key, an organization token and a revoked key could each answer the profile
differently. The PLAN's Phase 1 finds out (ruled, D1).

### Evidence index

<!-- claims: self=ceb489e kilocode=58072bd696 gobble-cli=1305a61 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The Kilo base the listing uses with no organization | `llmprovider/internal/kiloendpoint/kiloendpoint.go:15` | `const BaseURL = "https://api.kilo.ai/api/gateway"` |
| C2 | An organization lists at `/api/organizations/{id}/models` | `llmprovider/internal/kiloendpoint/kiloendpoint.go:52` | `e.Models = Route(u, "organizations/"+url.PathEscape(org)) + "/models"` |
| C3 | The listing's own comment: the public endpoint needs no credential | `llmprovider/catalog/discovery.go:1410` | `The public endpoint answers with no` |
| C4 | The listing sends a key it is given | `llmprovider/catalog/discovery.go:1420` | `token.Apply(req, headerAuthorization, bearerScheme)` |
| C5 | `KiloProfile` is exported | `llmprovider/auth/kilo_device.go:179` | `func KiloProfile(ctx context.Context, token string, opts ...llmprovider.Option) (KiloAccount, error) {` |
| C6 | It derives its origin from the base URL or the token | `llmprovider/auth/kilo_device.go:184` | `base, err := url.Parse(kiloendpoint.Resolve(st.BaseURL(), token, "").Gateway)` |
| C7 | and reads `/api/profile` there | `llmprovider/auth/kilo_device.go:188` | `kiloendpoint.Route(base, "profile")` |
| C8 | A non-2xx is classified as every provider error is | `llmprovider/auth/kilo_device.go:198` | `llmprovider.ClassifyHTTPError(string(llmprovider.ProviderKilo), resp)` |
| C9 | An auth request is bounded by 30 s | `llmprovider/auth/request_bound.go:17` | `var authRequestTimeout = 30 * time.Second` |
| C10 | A 401 is `ErrAuthFailure` | `llmprovider/api_error.go:225` | `case status == http.StatusUnauthorized \|\| status == http.StatusForbidden:` |
| C11 | Any Kilo 403 is `ErrNotPermitted` | `llmprovider/api_error.go:552` | `service == string(ProviderKilo) && (status == http.StatusForbidden` |
| C12 | The wizard's refusal predicate | `wizard/configure.go:668` | `return errors.Is(err, llmprovider.ErrAuthFailure) && !errors.Is(err, llmprovider.ErrNotPermitted)` |
| C13 | The credential step lists with the new credential | `wizard/configure.go:455` | `cat, listErr := discoverModels(ctx, p, d, res, credential.source, o)` |
| C14 | and saves it after the listing | `wizard/configure.go:514` | `if err := credential.save(ctx, d); err != nil {` |
| C15 | A Result's organization is the credential's | `wizard/configure.go:524` | `Kind: c.kind, APIKey: c.apiKey, VendorAuthPath: c.vendorPath, Organization: c.organization}` |
| C16 | The listing takes the Result's organization | `wizard/configure.go:844` | `llmprovider.For(llmprovider.ProviderKilo, catalog.WithKiloOrganization(res.Organization)),` |
| C17 | Without Discover nothing is listed | `wizard/configure.go:827` | `if !o.Discover {` |
| C18 | ResolveCredential checks only with Discover | `wizard/configure.go:333` | `checked only with Options.Discover.` |
| C19 | and says a public listing says nothing of the key | `wizard/configure.go:342` | `that needs no credential, such as Hugging Face's, says nothing of the key.` |
| C20 | A refused device key's message | `wizard/configure.go:719` | `return fmt.Errorf("wizard: %s refused the device sign-in; sign in again: %w", d.Label, err)` |
| C21 | The listing's default bound | `wizard/configure.go:23` | `const defaultDiscoverLimit = 10 * time.Second` |
| C22 | The device session reads the profile | `wizard/auth.go:401` | `account, err := kiloProfile(ctx, session.Access, opts...)` |
| C23 | and warns on any failure | `wizard/auth.go:403` | `could not read the Kilo account's organizations` |
| C24 | The device key reads no profile | `wizard/auth.go:437` | `key, under Options.KiloDeviceKey: nothing is saved and no profile is read,` |
| C25 | The device key is an API key | `wizard/auth.go:448` | `cred := staticCredential(CredAPIKey, session.Access)` |
| C26 | A kept Kilo session keeps its organization | `wizard/auth.go:203` | `cred.organization = o.Existing.Organization` |
| C27 | A kept Kilo session is checked only for its provider | `wizard/auth.go:514` | `if session.Provider != llmprovider.ProviderKilo {` |
| C28 | The handle's Wait | `llmprovider/x/signin/signin.go:254` | `func (s *SignIn) Wait(ctx context.Context) (Credential, error) {` |
| C29 | saves nothing; the caller does | `llmprovider/x/signin/doc.go:8` | `returns the Credential the sign-in yields. The sign-in saves nothing: the` |
| C30 | `x/signin` is experimental | `llmprovider/x/signin/doc.go:19` | `This package is experimental (R4 of the API standards): it is outside the` |
| C31 | Kilo offers an API key | `llmprovider/providers/kilo/descriptor.go:26` | `ID:             llmprovider.AuthAPIKey,` |
| C32 | and a device sign-in | `llmprovider/providers/kilo/descriptor.go:37` | `ID:             llmprovider.AuthDeviceCode,` |
| C33 | Kilo's key variable | `llmprovider/provider.go:106` | `ProviderKilo:        "KILO_API_KEY",` |
| C34 | The live device check reads the profile | `llmprovider/live_checks_test.go:88` | `account, err := auth.KiloProfile(ctx, session.Access, opts...)` |
| C35 | and passed on 2026-10-19 | `docs/decisions/0046-PLAN-v1-15-0-follow-ups.md:693` | `token's shape and the profile had passed.` |
| C36 | A test holds that the device key reads no profile | `wizard/options_0056_test.go:78` | `if store.saves != 0 \|\| *profiles != 0` |
| C37 | 0056-MADR D2 decided it | `docs/decisions/0056-MADR-wizard-credential-only-entry-point.md:174` | `* it saves no session, reads no profile and asks no organization question,` |
| C38 | 0059-MADR's save rule | `docs/decisions/0059-MADR-wizard-debugging-pass-findings.md:325` | `* **(a)** Save the new session only once the listing has accepted it;` |
| C39 | 0028-MADR's 403 rule | `docs/decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md:370` | `` * `statusSentinel`'s 403 becomes `ErrNotPermitted`; a 403 whose body `` |
| C40 | 0028-MADR's rule for a key not typed | `docs/decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md:422` | `* A credential the user did not type (an environment variable, a vendor` |
| C41 | 0017-MADR D2: the profile offers the organizations | `docs/decisions/0017-MADR-together-provider-and-auth-extensions.md:133` | `` * After login, `/api/profile` offers the organizations, for Kilo's `` |
| C42 | 0074-MADR D3: the caller saves | `docs/decisions/0074-MADR-one-sign-in-handle-and-method-facts.md:403` | `the caller saves, as the wizard does after its listing (0059-MADR W2).` |
| C43 | 0074-MADR D9: `x/signin` outside the API check | `docs/decisions/0074-MADR-one-sign-in-handle-and-method-facts.md:755` | `` * `llmprovider/x/signin` is outside `make api-check` (R4, R48; C37, C38): it `` |
| C44 | R48 | `docs/guides/api-standards.md:317` | `` - **R48. From `v1.0.0`, every non-internal package outside `llmprovider/x/` is `` |
| C45 | R55 | `docs/guides/api-standards.md:367` | `- **R55. A new export is justified.** A PLAN that adds an exported name` |
| K1 | Kilo reads the profile at its API origin | `kilocode:packages/kilo-gateway/src/api/profile.ts:9` | `KILO_API_BASE}/api/profile` |
| K2 | and reads 401 and 403 alike | `kilocode:packages/kilo-gateway/src/api/profile.ts:17` | `if (response.status === 401 \|\| response.status === 403) {` |
| K3 | The API origin | `kilocode:packages/kilo-gateway/src/api/constants.ts:13` | `export const KILO_API_BASE = process.env[ENV_KILO_API_URL] \|\| DEFAULT_KILO_API_URL` |
| K4 | The default gateway | `kilocode:packages/kilo-gateway/src/api/constants.ts:19` | `export const DEFAULT_KILO_AI_GATEWAY_URL = "https://ai-gateway.kilo.ai/api/v1"` |
| K5 | The terminal device sign-in reads the profile | `kilocode:packages/kilo-gateway/src/auth/device-auth.ts:117` | `const profileData = await getKiloProfile(token)` |
| K6 | The TUI sign-in ignores its failure | `kilocode:packages/kilo-gateway/src/auth/device-auth-tui.ts:81` | `const profile = await getKiloProfile(token).catch(() => undefined)` |
| K7 | The listing's anonymous retry | `kilocode:packages/kilo-gateway/src/api/models.ts:242` | `if (response.status === 401 && token && !organizationId && !finalBaseURL.includes("/organizations/")) {` |
| K8 | An organization's listing | `kilocode:packages/kilo-gateway/src/api/models.ts:217` | `? resolveKiloGatewayUrl(` |
| K9 | A token may embed a base URL | `kilocode:packages/kilo-gateway/src/auth/token.ts:9` | `const match = token.match(` |
| K10 | The profile handler takes only a device session | `kilocode:packages/kilo-gateway/src/server/handlers.ts:62` | `if (!info \|\| info.type !== "oauth") throw new UnauthorizedError("Not authenticated with Kilo Gateway")` |
| K11 | API keys are JWTs | `kilocode:packages/kilo-docs/pages/gateway/authentication.md:18` | `API keys are JWT tokens tied to your Kilo account.` |
| K12 | Organization tokens expire in 15 minutes | `kilocode:packages/kilo-docs/pages/gateway/authentication.md:67` | `Organization tokens are scoped with a 15-minute expiry` |
| K13 | Anonymous access to free models | `kilocode:packages/kilo-docs/pages/gateway/authentication.md:71` | `The gateway allows unauthenticated access for free models only.` |
| K14 | The dashboard key is on the personal account | `kilocode:packages/kilo-docs/pages/getting-started/setup-authentication.md:50` | `2. Go to **Your Profile** on your **personal account** (not in an organization)` |
| G1 | gobble's ACP check | `gobble-cli:llm/provider/signinrun.go:209` | `func checkListing(` |
| G2 | skips a public listing only without a key | `gobble-cli:llm/provider/signinrun.go:210` | `if PublicListing(id) && key == "" {` |
| G3 | and copies the wizard's predicate | `gobble-cli:llm/provider/signinrun.go:198` | `func refusedCredential(err error) bool {` |
| G4 | gobble's terminal path takes the device key | `gobble-cli:llm/provider/signin.go:354` | `KiloDeviceKey:    true,` |
| G5 | through the wizard | `gobble-cli:llm/provider/signin.go:95` | `var resolveCredential = wizard.ResolveCredential` |
| G6 | gobble names Kilo's listing public | `gobble-cli:llm/provider/list.go:44` | `var publicListings = []string{"opencode-zen", "opencode-go", "huggingface", "kilo"}` |
| G7 | and tells its user a typed key there is not checked | `gobble-cli:internal/cli/login.go:117` | `lists its models without a key, so the key was not checked.` |
| G8 | gobble's pin | `gobble-cli:go.mod:17` | `github.com/maccavelli/go-llmprovider-sdk v1.31.0` |

## Decision Drivers

* **A wrong key is not saved in silence.** That is what W2 and D-H10 are
  for; with no organization they do nothing for Kilo today.
* **A working key is never refused.** The check must not refuse a valid key
  of any kind Kilo issues. Only a device token is known to read the profile
  (C35); a dashboard key and an organization token are not (D1, Phase 1).
* **An outage never fails a sign-in** (the ruling; D-H10's "A listing that
  fails for another reason keeps today's warning").
* **One rule for the wizard and its callers.** gobble's terminal and ACP
  paths are to agree (gobble-cli's
  `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  D4, its note "Keep the wizard's rule: a 401 refuses").
* **The existing classification keeps working:** `refusedCredential`,
  `signin.ErrorKindOf` and gobble's copy see a refusal as they see a
  listing's.
* **The least new API** (R55) **and the fewest requests:** one request per
  sign-in at most, none where the listing already authenticates.
* **API stability** (R4, R48; C43, C44).

## Considered Options

* **V1.** The wizard verifies through `auth.KiloProfile`, as part of its
  credential check; no new export. A caller copies the rule.
* **V2.** V1, with the check in `internal/signinflow`, which the wizard
  calls, and an experimental `signin.VerifyKilo` in `llmprovider/x/signin`
  over it, for a caller such as gobble's ACP path.
* **V3.** V1, with a stable exported verifier in `llmprovider/auth`, under
  the v1 promise.
* **V4.** Verify inside the sign-in engine: the device sign-ins, the
  wizard's and the handle's, read the profile before they return.
* **V5.** Verify inside `catalog.List`: a Kilo listing with a key and no
  organization also reads the profile, and fails with its 401.
* **V6.** Verify by another authenticated request: a chat completion with an
  existing model and an empty message list (L21 to L24), or
  `/api/profile/balance` (L17, L18).
* **V7.** Do nothing in code; document that a Kilo key with no organization
  is not checked.

Two narrower choices sit across them:

* **On a 401: refuse (R1) or warn (R2).** The ruling chose R1.
* **With an organization: rely on the organization's listing (O1) or read
  the profile too (O2).**

## Decision Outcome

Chosen option: **"V2", with R1 and O1**, because it closes P1 to P6 in the
wizard with the classification callers already use, gives a caller the same
rule without copying it, and adds nothing under the v1 promise until the
handle is promoted (0074-MADR D9). D1 and D11 are the first ruling's; D2 and
D5 to D10 are the owner's answers to Q1 to Q7 of 2026-11-16, each the option
the draft recommended, so their text stands as drafted, now decided; D3, D4
and D12 follow from them. *(2026-11-16: this paragraph said "proposed" and
"the recommendation of the question named beside it" until the owner's
answers.)*

### D1. Where, and which way (ruled)

* The wizard verifies a Kilo credential against Kilo's profile when the
  credential is to be used with no organization. The profile's 401 is a
  refusal, handled as the listing's refusal is: the store is unchanged
  until the check passes (W2), a typed key is asked for again up to three
  times, and a key from anywhere else ends the run naming its source
  (D-H10).
* A profile that cannot answer (unreachable, slow, a 5xx) does not fail the
  sign-in.
* The PLAN's first phase settles which key kinds the profile accepts,
  before any code: a valid kind it refuses would make this record refuse a
  working key, and the PLAN stops there for the owner.
* The change ships in one release with the typed "tool schema refused"
  error, which is queued for a record of its own; gobble then bumps its pin
  (D11, D12).

*(Note, 2026-11-20: [0081-MADR-a-cancelled-sign-in-saves-nothing.md](0081-MADR-a-cancelled-sign-in-saves-nothing.md) D5: "the store is unchanged until the
check passes (W2)" holds for `ResolveCredential` as written; for
`ConfigureLLM` it holds until the run ends, since the session is saved as its
last step, after the model and fallback questions.)*

### D2. Which credentials (Q1, answered)

Every Kilo credential the credential step checks, whatever its source: a
typed, environment or saved key (P1 to P3), a device session (P4), a device
key (P5) and a kept session (P6). This amends
[0056-MADR-wizard-credential-only-entry-point.md](0056-MADR-wizard-credential-only-entry-point.md)
D2's "reads no profile" (C37): in key mode the profile is read once, to
verify; no organization is asked and nothing is saved, as before.

### D3. How the check joins the credential step

* It is part of the step's check, beside the listing (C13), and only under
  `Options.Discover`, as every check is (C17, C18). Without Discover nothing
  changes.
* A refusal enters the loop the listing's refusal enters, so each source
  keeps its present wording: "refused this key; enter another" for a typed
  key; the source named for the environment and a saved key; "refused the
  device sign-in; sign in again" for the device key (C20); the sign-in menu,
  once, for a kept session (0066-MADR V7).
* The device session's profile read, made for the organization menu (C22),
  is its check: no second request. Under Discover its 401 is a refusal, not
  today's warning (C23); without Discover the warning stays.
* The profile request carries what the wizard's other requests carry: the
  endpoint in use, the caller's client, `Options.Identity` first and
  `ProviderOptions` after (0020-MADR F20; 0079-MADR D5).
* No other provider's check changes.

### D4. With an organization, the listing is the check (O1)

The organization's listing answers 401 to no credential and to a made-up one
at the URL this module uses and at Kilo's (L7 to L12), so it already refuses.
The profile is not read for it. A device session reads the profile anyway,
before its organization is chosen (D3).

*(Added 2026-11-17, the owner's ruling, relayed by the coordinator: "Treat it
as having an organisation: no profile read". As put: a token whose embedded
URL names an organization is checked by that organization's listing alone, as
D4 intends, so a valid organization token cannot be refused by an unverified
check; a small shared helper does it, because of Go's internal-package rule.
Options not chosen: accept it as built and name the risk; skip the profile
check for any token that embeds a URL. Evidence: for a token of the form
`https://<host>/api/organizations/<id>:<secret>` with no organization entered,
the listing is already that organization's, and the profile was read as well,
and refused on its 401. "Has an organization" is now one rule, in
`internal/kiloendpoint` `Organization`, the rule the listing's URL is built by.
It moved from `llmprovider/internal` to the module's own `internal`, so that
`internal/signinflow`, and through it the wizard and `llmprovider/x/signin`,
may import it. It decides in one place, `signinflow.VerifyKilo`, for the wizard
and for `signin.VerifyKilo`. That ruling covers the explicit organization and
a token's embedded URL, and nothing more.)*

*(Added 2026-11-17, a second ruling the same day, the owner's, relayed by the
coordinator: "Count it; record the unprobed listing as not verified
(Recommended)". As put: a base URL whose own path names an organization, such
as `https://<host>/api/organizations/<id>`, counts as having an organization,
as the code was built: the profile is not read, and that organization's
listing is the check. Evidence: `kiloendpoint.Resolve` lists a base URL of
that form at `{base}/models`, an organization's listing, and sends no
`X-KILOCODE-ORGANIZATIONID` header for it, since the header is set only when the
resolved organization is non-empty (`llmprovider/catalog/discovery.go:1413` to
`:1424`). `Organization` returns the base URL's organization as its third
case. What is not verified is whether that listing, sent without the header,
refuses a wrong key: see "Not verified".)*

*(Added 2026-11-17, a third ruling the same day, the owner's, relayed by the
coordinator: "Share the type internally and fix it now (Recommended)". As put:
the rule is that a Kilo credential with an organisation reads no profile; a
test showed the profile is still read when the calling program supplies the
organisation through the wizard's provider options; the fix needs one internal
type shared across three packages. The option chosen read:
"Three files, no exported name, the listing behaves as before. The rule then holds for every way an organisation can be given, with a test and a planted break. One more short pass and re-check before the commit."
This is a fourth source of the
organization, beside the explicit one, a token's embedded URL and the base
URL's path. The option's value is `kiloendpoint.OrganizationOption`, which
`catalog.WithKiloOrganization` carries and the check reads from the resolved
options.)*

*(Corrected 2026-11-17, after the third independent review found the wording
above and in the first version of this addition false, in place: it said the
explicit organization starts from the Result's and is replaced by the last
option, and that a test held the rule against the listing's actual request.
The listing's real order is the following, and the check now follows it by
construction.)*

*The precedence, as the listing applies it.* The explicit organization is the
last `catalog.WithKiloOrganization` applied, an empty one included, replacing
an earlier one. `llmprovider.ResolveOptions` applies the options outside a `For`
first, in order, then those inside `For(Kilo, ...)`, in order. The wizard's own
`For`, which carries the Result's organization (a kept session's, or a device
session's choice), stands among the listing's options before any `For` the
caller passed in `Options.ProviderOptions`: so an option the caller gives
inside a `For` replaces the Result's, and one given outside a `For` is replaced
by it, even when the Result's is empty. With no explicit organization, a
token's embedded URL names one; failing that, the base URL's own path does. The
check is given the listing's options, built in one place (`listingOptions`,
`wizard/configure.go`), and `signinflow.VerifyKilo` reads the organization from
them; for `signin.VerifyKilo`, a non-empty `organization` argument is one more
such option, put where the wizard puts the Result's: inside a `For` for Kilo,
before every `For` of the caller's and after every option outside a `For`.
Tests hold the check against the listing's actual request for each form: direct
and `For` options, in each order, a token's URL, a base path, and the Result's
organization.

*(Added 2026-11-17, the coordinator's decision, not the owner's ruling. The
option the owner chose read "Three files, no exported name, the listing behaves
as before...". The third review's blocker could only be closed by also building
the options in one place in the wizard (`listingOptions`, `wizard/configure.go`,
used by `discoverModels` and by the check in `wizard/kilo_verify.go`), so five
non-test files changed for it, not three. The coordinator judged that inside the
ruling's purpose, that the check decides by the listing's own rule, on the
evidence that no exported name changed and that the fourth review found the
listing's requests identical to `b109cea` in 48 of 48 rows (four providers by
twelve caller-option sets). The owner is told.)*

### D5. A 403 is not a refusal (Q4, answered)

A 403 from the profile is told as an outage is (D6), as every Kilo 403 is
`ErrNotPermitted` (C11; D-H2). *(2026-11-16, by the answer to Q4: this
sentence went on "unless Phase 1 measures a 403 that Kilo gives a bad
credential with a code", which was Q4's option (c), not chosen. A 403
Phase 1 measures for a bad credential is put to the owner; it does not
change this rule by itself.)*

### D6. An outage: a warning, within the listing's bound (Q5, answered)

The check runs within `Options.DiscoverLimit`, the listing's bound (10 s by
default, C21), not `auth`'s 30 s (C9). An outage, a timeout, a 5xx, a 403, an
unexpected status or a body that does not decode gives one warning, by class
and status, never the service's text (0056-MADR D3): "could not verify the
Kilo key (…); it is saved unchecked". The sign-in goes on.

*(Added 2026-11-17, the coordinator's reading after an independent review, not
the owner's: D3 keeps the organization menu's warning "without Discover" only,
so under Discover a device session's failure that is no refusal is told by this
warning, once, and not by the menu's "could not read the Kilo account's
organizations (…); using the personal account". A ctx that has ended gives
neither: the check returns ctx's error and nothing is saved. Superseded the
same day by the owner's ruling below, as to the menu's warning; the ended-ctx
sentence stands.)*

*(Amended 2026-11-17, the owner's ruling, relayed by the coordinator: "Two
notices". As put: the staged code showed only this warning, so a person whose
account has organizations was put on the personal account without being told,
which affects whose usage and billing apply; the record asks for one warning
but does not settle this; the option chosen reads "The decision's warning and
the old organisation-menu warning both show. Says everything, but contradicts
the decision's "one warning"." So this decision departs from "one warning" for
one case, and only that one: under Discover, a Kilo device session whose
profile read ends in anything that is not a refusal and not the caller's
context ending (a 403, a 5xx, a body that does not decode, a refused
connection, a stall) shows two notices, each once: "could not read the Kilo
account's organizations (…); using the personal account" first, then this
warning; the session is saved. The menu's comes first because it reports what
happened first, the account's organizations could not be read and the personal
account is used, and this one then reports the conclusion for the key. Every
other credential shows this warning alone. Unchanged: a 401 is a refusal with
no menu warning (D3); an ended context shows nothing and saves nothing;
without Discover the menu's warning stands alone.)*

*(Note, 2026-11-20: [0081-MADR-a-cancelled-sign-in-saves-nothing.md](0081-MADR-a-cancelled-sign-in-saves-nothing.md) S76, the owner's ruling "...; going on
without verifying it" (Recommended): the warning's words are now "could not
verify the Kilo key (...); going on without verifying it", in place of "...;
it is saved unchecked", which claimed a save the run may never make: a key is
never saved, a session is saved only when the run ends well, and
`ConfigureLLM` may still be abandoned at a model question. The decision
stands: one warning, by class and status, and the sign-in goes on. A ctx that
has ended shows neither this warning nor the organization menu's, with
`Discover` or without.)*

### D7. The profile's URL stays `KiloProfile`'s (Q6, answered)

The check reads the profile where `KiloProfile` does: the origin of the base
URL in use, or of a token that embeds one (C6). Kilo's client always reads
its API origin (K1), but a token minted by another Kilo backend would then be
checked against the wrong one and refused. A custom endpoint without
`/api/profile` gives a 404, which is not a refusal (D6).

### D8. No option turns it off (Q7, answered)

`Options.Discover` false already turns every check off (C17). No field is
added.

### D9. The refusal's error (Q3, answered)

The refusal is the profile's `*APIError` (status 401, kind
`ErrAuthFailure`), wrapped by the wizard's existing messages. No new
sentinel: `refusedCredential`, `errors.Is(err, llmprovider.ErrAuthFailure)`
and `signin.ErrorKindOf` (`auth`) see it as they see the listing's.

### D10. The shared surface (Q2, answered)

* The check lives in `internal/signinflow`, which both the wizard and
  `x/signin` already import, so it is written once.
* `llmprovider/x/signin` gains one experimental function,
  `VerifyKilo(ctx, key, organization string, opts ...llmprovider.Option)
  error`: nil when the profile answers, when an organization is given (its
  listing checks), or when the profile cannot say (D6); an error matching
  `llmprovider.ErrAuthFailure`, the profile's `*APIError`, when Kilo refuses
  the key; ctx's error when ctx ends. Its doc names each kind (R52).
* It is outside `make api-check` (R4, R48; C30, C43) and moves into `auth`
  with the rest of the handle when a later record promotes it (0074-MADR
  D9). Nothing is added under the v1 promise.
* Its call site (R55) is gobble's ACP check (G1), by a gobble-cli record.

*(Added 2026-11-17, the owner's ruling, relayed by the coordinator: "Its own 10
s bound; a stall is "cannot say"". As put: `VerifyKilo` bounds its profile read
by 10 s itself and returns nil when that passes, as for any outage; it returns
the context's error only when the caller's own context ended; one rule for the
wizard and for callers such as gobble, matching Q5. Options not chosen:
document it and let the caller handle its deadline; a bound the caller can set,
defaulting to 10 s. Evidence: with a 200 ms caller deadline and a stalled
profile it returned "context deadline exceeded", kind "timeout". The 10 s is
`signinflow.DefaultCheckLimit`, the constant the wizard's default discovery
limit is too. A caller's mistake in the options, an option for another
provider or a base URL that does not parse, is not "cannot say": it is an error
matching `ErrInvalidRequest`, and no request, as the nil of this decision is
for a profile that cannot say. "nil" means verified or cannot say; a caller
that must tell them apart reads the profile with `auth.KiloProfile`.)*

### D11. The release (ruled; tag on the owner's word)

One minor release after `v1.31.0` (`v1.32.0` unless another release comes
first), shared with the typed "tool schema refused" error's record. The tag
and the GitHub Release are each the owner's word, never a standing approval.

### D12. gobble afterwards

* **Terminal path** (G4, G5): the pin bump alone gives it D2 to D9 through
  `wizard.ResolveCredential`. Its notice that Kilo's typed key "was not
  checked" (G7) becomes untrue for Kilo and is gobble's to change.
* **ACP path** (G1 to G3): a call of `signin.VerifyKilo` for Kilo, in
  `checkListing`; nothing is removed, since `checkListing` and its predicate
  serve every other provider.
* Each by a gobble-cli record of its own.

### Consequences

* Good, because a wrong Kilo key typed, pasted, read from the environment or
  kept is refused at sign-in, with no organization, as every other
  provider's is.
* Good, because a refusal flows through the loop, wording and predicates
  that exist, and gobble's copy of the predicate sees it unchanged.
* Good, because the device session's existing profile read becomes the
  check at no extra request, and the organization paths add none.
* Good, because nothing enters the v1 promise.
* Neutral, because the check costs one request per Kilo sign-in with no
  organization (P1 to P3, P5, P6), within the listing's bound.
* Bad, because a Kilo key kind the profile refuses while the gateway
  accepts it would be refused. Phase 1 exists to find one; one Phase 1
  cannot reach (an organization token, if the owner has none) stays a risk,
  named in "Not verified".
* Bad, because 0056-MADR D2's "reads no profile" changes under Q1 (a).
  `TestKiloDeviceKey_ReturnsTheKey_0056`, which holds it (C36), runs
  without Discover, so it stays as it is (D3).
* Bad, because at least six existing wizard tests list Kilo under Discover
  against fakes (the PLAN names them), and each now meets a profile read;
  the PLAN stops if one needs a change.
* Bad, because a caller who copies the old rule, and does not call
  `signin.VerifyKilo`, keeps the blind check.

*(Added 2026-11-17, after an independent review, the coordinator's wording,
not the owner's.)*

* Bad, because the SDK's redaction removes a key from an error's text by
  pattern, not by the key's value: an opaque key a service echoes in a 401
  body may remain in the returned error's text, as it may for every listing's
  refusal (0056-MADR D3). A notice never shows it, and a caller shows an
  error's class and status, not its text.
* Bad, because a key valid only for an organization, typed with no
  organization entered, is read at the profile, which Phase 1 did not try
  ("Not verified").
* Bad, because with Kilo's other gateway address,
  `https://ai-gateway.kilo.ai/api/v1`, as the endpoint, the profile is read at
  that origin (D7), which answers 404: every sign-in there warns, and a wrong
  key passes with that warning.

### Confirmation

* **Tests**, each against an in-process Kilo stand-in (`httptest` or a
  `RoundTripper`, no outside host, 0065-MADR), each seen red at `ceb489e`
  where it uses existing API, and each seen to fail on a planted break
  (`tools/redgreen.py`, `tools/plant.py`, on a scratch clone): a typed key
  refused then a good one; an environment and a saved key refused, naming
  the source; the device key refused; the device session refused with the
  store unchanged and one profile read; a kept session refused, then the
  sign-in menu once; a 403, a 503, a refused connection and a timeout each
  warn once and save; an organization reads no profile; no Discover, no
  read; no other provider reads a profile; `signin.VerifyKilo`'s kinds.
* **`make api-check`** reports no change under the v1 promise, and
  `tools/apidiff.py v1.31.0` lists no addition outside `x/`.
* **A live, owner-run probe** (PLAN Phase 1): the profile with a dashboard
  key, a device token and, if the owner has one, an organization token;
  status and counts only, no key shown or recorded.
* **`tools/claims.py`** on this record and its PLAN; the gates the PLAN
  names.

## Pros and Cons of the Options

### V1. Wizard only, with `auth.KiloProfile`

* Good, because it adds no API: `KiloProfile` already returns the 401 as
  `ErrAuthFailure` (C8, C10).
* Good, because gobble's terminal path gets it from a pin bump.
* Bad, because a caller such as gobble's ACP path must copy when to read
  the profile (Kilo, no organization) and what refuses; a later change of
  the rule does not reach the copy.

### V2. V1, with an experimental `signin.VerifyKilo` over `internal/signinflow`

* Good, because the wizard and every caller run one implementation.
* Good, because it sits where a caller already gets its credential
  (`signin.Start`, `Wait`), and is outside the v1 promise until promotion.
* Neutral, because a caller still decides when to call it, as gobble
  already decides to list.
* Bad, because it adds a name a later promotion must carry or rename.

### V3. V1, with a stable verifier in `auth`

* Good, because a caller that does not use `x/signin` gets a stable name.
* Bad, because the name and its outage semantics are fixed by R48 before
  Phase 1 and gobble's use have tested them.
* Bad, because it splits the sign-in surface between `auth` and `x/signin`
  until promotion.

### V4. Inside the sign-in engine

* Good, because no caller can forget it after a device sign-in.
* Bad, because it covers only device tokens, the ones Kilo just issued and
  least likely wrong, and none of P1 to P3 or P6.
* Bad, because the engine would start judging credentials, which 0074-MADR
  D3 leaves to the caller (C42).

### V5. Inside `catalog.List`

* Good, because every caller of the listing, gobble's two paths included,
  gets it with no code.
* Bad, because every Kilo listing with a key would cost a second request,
  at sign-in and at run time, and a listing of public models would fail
  over a profile it does not need.
* Bad, because it changes what the listing's documented behaviour is (C3)
  for callers who only want models.

### V6. Another authenticated request

* Good, because a chat completion with an empty message list is refused
  with 401 `INVALID_TOKEN` before its body is read (L22, L24).
* Bad, because it rests on the order of the gateway's checks, which nothing
  documents, and on a model id that exists (L25); if the order changed, it
  could generate and bill.
* Bad, because `/api/profile/balance` is the profile's sibling with nothing
  more to offer, and a second endpoint to keep track of.

### V7. Document only

* Good, because it costs nothing.
* Bad, because a wrong key is still saved and fails on first use; the owner
  did not choose it.

### R1. A 401 refuses (chosen by the ruling)

* Good, because it matches the listing's rule and Kilo's own client (K2).
* Bad, because a valid key the profile refuses would be refused (Phase 1).

### R2. A 401 warns

* Good, because it can never refuse a working key.
* Bad, because the key is saved anyway, which is the problem.

### O1. With an organization, the listing checks

* Good, because L7 to L12 show it authenticates; no extra request.

### O2. With an organization, read the profile too

* Bad, because it adds a request that refuses nothing the listing does not.

## Open questions for the owner

In the order to answer them: Q1 decides whether Q2 matters.
*(2026-11-16: all seven answered by the owner the same day, each as
recommended; the answers follow each question. The questions stay as
they were asked.)*

* **Q1 (D2). Which Kilo credentials are verified?**
  * **(a)** Every one the wizard's step checks: typed, environment and
    saved keys, the device session, the device key and a kept session;
    0056-MADR D2's "reads no profile" is amended (recommended: one rule,
    which gobble's two paths need to agree).
  * **(b)** All but a fresh device token, which Kilo just issued: keys and
    kept sessions only. 0056-MADR D2 stands; gobble's ACP path, device only,
    then needs nothing, and Q2 is moot.
  * **(c)** Typed keys only.

  *Answered 2026-11-16, the owner's ruling, relayed by the coordinator: **"All of them"**. As put: typed, environment and saved keys, the device session, the device key and kept sessions; amends 0056-MADR D2's "reads no profile"; the device session's existing profile read becomes the check, so it costs no extra request there. Not chosen: all but a freshly issued device token; typed keys only.*
* **Q2 (D10). What do callers get?**
  * **(a)** An experimental `signin.VerifyKilo` over `internal/signinflow`,
    outside `make api-check` (recommended).
  * **(b)** A stable `auth.VerifyKiloKey`, under the v1 promise from the
    release.
  * **(c)** No new export: the wizard alone; `KiloProfile`'s doc states the
    rule; a caller copies it.

  *Answered 2026-11-16, the owner's ruling, relayed by the coordinator: **"An experimental `signin.VerifyKilo`"**. As put: one function in the SDK's experimental sign-in package, used by the wizard and callable by gobble; nothing is added under the v1 compatibility promise; gobble adds one call on its ACP path. Not chosen: a stable `auth.VerifyKiloKey`; no export, callers copy the rule.*
* **Q3 (D9). The refusal's error.**
  * **(a)** The profile's `*APIError`, kind `ErrAuthFailure`, in the
    wizard's existing messages; no new sentinel (recommended).
  * **(b)** A new sentinel, `auth.ErrKiloKeyRefused`, matching
    `ErrAuthFailure`.
  * **(c)** A plain error naming the refusal.

  *Answered 2026-11-16, the owner's ruling, relayed by the coordinator: **"The existing auth-failure error"**. As put: no new error type; the wizard's and gobble's existing refusal predicates work unchanged, and the error's kind is `auth`. Not chosen: a new sentinel that also matches auth failure; a plain error.*
* **Q4 (D5). A 403 from the profile.**
  * **(a)** Not a refusal: a warning, as every Kilo 403 is `ErrNotPermitted`
    (D-H2) (recommended).
  * **(b)** A refusal, as Kilo's client reads 401 and 403 alike (K2).
  * **(c)** A refusal only if Phase 1 measures one Kilo gives a bad
    credential, by its code, added to D-H2's table.

  *Answered 2026-11-16, the owner's ruling, relayed by the coordinator: **"No: warn and continue"**. As put: consistent with the SDK's existing rule for every provider; a valid key that is merely not permitted something is not thrown away. Not chosen: yes, as Kilo's client treats it; only if Phase 1 measures it.*
* **Q5 (D6). An unreachable, slow or failing profile.**
  * **(a)** Within `DiscoverLimit`, one warning by class and status, and
    the credential saved (recommended).
  * **(b)** Within `auth`'s 30 s, the same warning.
  * **(c)** No warning: an outage passes in silence.

  *Answered 2026-11-16, the owner's ruling, relayed by the coordinator: **"Warn within 10 s and save"**. As put: one warning naming the class and status, bounded by the wizard's existing discovery limit (10 s), and the sign-in goes on. Not chosen: warn within 30 s and save; pass silently.*
  *(Added 2026-11-17: the same 10 s bounds `signin.VerifyKilo`, by the owner's ruling at D10.)*
* **Q6 (D7). Where the profile is read for a custom endpoint or a token
  that embeds a base URL.**
  * **(a)** Where `KiloProfile` reads it: the origin in use (recommended).
  * **(b)** Always Kilo's API origin, as Kilo's client does (K1).
  * **(c)** Skip the check unless the origin is Kilo's own.

  *Answered 2026-11-16, the owner's ruling, relayed by the coordinator: **"The origin in use, as `KiloProfile` does"**. As put: the key is verified against the same service it will be used with; no change to `KiloProfile`. Not chosen: always Kilo's own API origin; skip the check unless the origin is Kilo's own.*
* **Q7 (D8). A way to turn it off.**
  * **(a)** None beyond `Discover` false (recommended).
  * **(b)** A wizard field, such as `Options.NoKiloVerify`.
  * **(c)** An option on `signin.VerifyKilo` only.

  *Answered 2026-11-16, the owner's ruling, relayed by the coordinator: **"None beyond `Discover` false"**. As put: one existing switch governs every sign-in check. Not chosen: a wizard field of its own; an option on `VerifyKilo` only.*

## More Information

### Records this decision amends

* [0056-MADR-wizard-credential-only-entry-point.md](0056-MADR-wizard-credential-only-entry-point.md)
  D2: in key mode the profile is read once, to verify (under Q1 (a)).
* [0017-MADR-together-provider-and-auth-extensions.md](0017-MADR-together-provider-and-auth-extensions.md)
  D2: after login the profile offers the organizations (C41) and, with
  Discover, verifies the token.
* [0059-MADR-wizard-debugging-pass-findings.md](0059-MADR-wizard-debugging-pass-findings.md)
  W2 and [0028-MADR-heuristics-and-performance-from-the-research-pass.md](0028-MADR-heuristics-and-performance-from-the-research-pass.md)
  D-H10 are unchanged; they now apply to Kilo with no organization. Each gets
  a dated note pointing here.

### Related

* gobble-cli, `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`,
  D4's notes of 2026-11-16 (the ruling "Keep the wizard's rule: a 401
  refuses" at its lines 504 to 505, and the coordinator's verification
  from line 514), committed in gobble-cli at `cecc892` on its branch
  `p3-acpext-v0.5.0`; this record serves them. *(Corrected 2026-11-16: the
  draft cited them without a commit; at `1305a61`, which the evidence
  index's G rows still read, they were not yet committed.)*
* The typed "tool schema refused" error: a record of its own, not yet
  numbered, that shares D11's release.
* [0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md)
  D3 and D9: the caller saves; `x/signin` is experimental.

### Found beyond the question asked

* The coordinator's organization probe used
  `/api/gateway/organizations/{id}/models`; this module lists an
  organization at `/api/organizations/{id}/models` (C2), and Kilo's client at
  `https://ai-gateway.kilo.ai/api/v1/organizations/{id}/models` (K4, K8).
  All three answer 401 to no credential and to a made-up one (L7 to L12), so
  the conclusion holds.
* An API key in the wizard never has an organization (C15): the organization
  question is only the device session's.
* Kilo's client reads the profile only for a device session (K10), so its
  code says nothing about dashboard keys.
* `kiloendpoint`'s comments cite Kilo's `api/url.ts:9-30` and
  `api/models.ts:219`, which have moved at `58072bd696` (K8 is at 217). Not
  this record's to fix.

### Not verified

*(2026-11-16: the owner's answer to the PLAN's Q8, relayed by the
coordinator: "The dashboard key and the device sign-in" are required before
Phase 2; the organization token and the revoked key run only if the owner has
them, and each not run stays named below as "not run: no such key".)*

* What the profile answers to a real dashboard key, an organization token,
  and a revoked or rotated key: PLAN Phase 1, owner-run.
* Whether Kilo ever answers the profile with 403 for a bad credential.
* ~~That a device token still reads the profile today: last seen on
  2026-10-19 (C35); Phase 1 runs it again.~~
* The profile's rate limits: no rate-limit header was seen on any response
  probed; the profile's own 200 was not seen.

*(Added 2026-11-17, after the owner's Phase 1 run, relayed by the
coordinator and recorded in
[0080-PLAN-verify-a-kilo-credential-through-its-profile.md](0080-PLAN-verify-a-kilo-credential-through-its-profile.md),
"Phase 1, the owner's run".)*

* **Verified 2026-11-17:** a dashboard API key is accepted by the profile
  (the live `DashboardKey` subtest passed; its organization count was not
  sent, so none is recorded). A device token is accepted by the profile
  (0 organizations, a personal account), and one generation on a free model
  succeeded; this settles the struck bullet above.
* **Verified 2026-11-17, a refused key (run 3, the owner's, relayed by the
  coordinator):** the profile answered HTTP 401 to a key the owner supplied
  as invalid, and the SDK classified it `ErrAuthFailure`, not
  `ErrNotPermitted`. The 401 for a bad key was observed with a made-up token
  by the agents (L13 to L16) and with an invalid key by the owner. The
  owner's words, relayed by the coordinator: "it was a mistyped real key,
  changed the last char": a real dashboard key with its last character
  changed, the mistyped-key case this record exists for. The subtest is named
  `RevokedKey`, but no revoked key was tested.
* **Still not verified:** what the profile answers to an organization token
  ("not run: no such key", by the owner's answer to the PLAN's Q8), and to a
  key revoked after having been valid; whether Kilo answers such a key with
  401 or 403.
* Phase 1's stop condition did not arise for the two required kinds.

*(Added 2026-11-17, with the owner's ruling "Count it; record the unprobed
listing as not verified (Recommended)", relayed by the coordinator.)*

* **Not verified:** that the organization listing, sent WITHOUT the
  `X-KILOCODE-ORGANIZATIONID` header, refuses a wrong key. A base URL whose own
  path names an organization is listed at `{base}/models` with no header
  (`llmprovider/catalog/discovery.go:1413` to `:1424`: the header is set only
  when the resolved organization is non-empty). The live probes L7 and L8 sent
  the header; L9 to L12 are not recorded as sent with it, and used a made-up
  organization id and a made-up token, so their 401 does not show that a real
  organization's listing refuses a wrong key. The test stand-in checks the bearer
  token only (`wizard/kilo_verify_0080_test.go:117` to `:122`). So a wrong key
  with such a base URL may pass the check unrefused.

## Release note (2026-11-22)

`CHANGELOG.md`'s `[Unreleased]` became the `v1.32.0` section dated 2026-11-22 (R53: a section is dated by its tag's commit, so the date is re-checked by `make changelog-check` once the owner's commit exists); `[Unreleased]` is empty. `python3 -B tools/apidiff.py v1.31.0` lists additions only (`APIError.Tool`, `ErrToolSchemaRefused`, `x/signin.VerifyKilo`, `wizard.Options.NoBrowserQuestion`, `wizard.Options.NoLoopback`): a minor version. The tag is the owner's. This record's status stays as it is and goes to `complete` when the tag exists, its CI has passed, and `tools/consumer.py` and `tools/offline.py` are recorded (`tools/records.py status`). This is the release phase: the tag carries this record's entry with 0081 to 0085 and 0087 (the owner's ruling "Five now; 0088+0089 next"); 0088 and 0089 are not in it.
