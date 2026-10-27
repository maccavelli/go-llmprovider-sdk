---
status: accepted
date: 2026-10-27
decision-makers: repository owner
consulted: "this repository at 876b81e (the Code claims); Anthropic's Authentication and Workspaces documentation, read 2026-10-25; the owner's probes of three Anthropic keys on 2026-10-25 (the Evidence below); the owner's runs of TestLive_Workspace_0064 and the same documentation, re-read, on 2026-10-27 (the amendment of that date)"
informed: every consumer of llmprovider and the wizard; prepare-commit-msg and gobble-cli
---

<!-- markdownlint-disable MD013 -->

# Send Anthropic's Workspace Header for Keys Not Scoped to a Workspace

## Context and Problem Statement

Anthropic issues personal and service account keys either for one workspace
or for every workspace the key's identity may use. A key of the second kind
must name a workspace on every request, in the `anthropic-workspace-id`
header; without it every request fails with HTTP 400
`invalid_request_error`:

> This API key is not scoped to a workspace, so this request must include
> the anthropic-workspace-id header with the ID of the workspace to use. Add
> the header, or use an API key that is scoped to a workspace.

The Claude provider and the catalog's Claude listing cannot send that header.
On 2026-10-25 the owner configured prepare-commit-msg (SDK v1.22.2) with such
a key. The wizard warned `live model listing for Claude (Anthropic) is
unavailable (invalid request, HTTP 400); search covers the built-in catalog
only`, offered the static catalog, and saved the key. Every commit message
generated with it would then fail with the same 400. A key of a supported,
documented type does not work in this SDK at all, and the wizard reports it
as an outage.

This record decides how the SDK carries a workspace for the Claude provider,
how the wizard detects that a key needs one and asks for it, and how the
refusal is classified.

### Evidence, measured 2026-10-25

Three keys of the owner's, probed by a scratch script against
`https://api.anthropic.com` with `anthropic-version: 2023-06-01`. The
workspace ID is redacted to `wrkspc_REDACTED`; it is the same workspace in
every row.

| Request | Key A: personal, one workspace (`sk-ant-usr-`) | Key B: legacy workspace key (`sk-ant-api03-`) | Key C: personal, not scoped to a workspace (`sk-ant-usr-`) |
| --- | --- | --- | --- |
| `GET /v1/models`, no workspace header, `x-api-key` | 200 | 200 | 400, the message above |
| the same with `Authorization: Bearer` | 200 | 200 | 400, the message above |
| `POST /v1/messages/count_tokens`, no header | 200 | 200 | 400, the message above |
| `POST /v1/messages`, no header | 200 | 200 | not sent |
| `GET /v1/organizations/workspaces?include_default=true` | 401 `authentication_error`, "The Admin API requires an Admin API key or an organization-scoped API key." | 401, the same | 403 `permission_error`, "Missing permissions. …" |
| models, count_tokens and messages, header `wrkspc_REDACTED` | 200 | 200 | 200 |
| models, header `not-a-workspace` | 400, "anthropic-workspace-id header must be a valid workspace ID." | the same | the same |
| models, header well-formed but invented | 400, the same message | the same | the same |

Every 200 carried the response header `anthropic-workspace-id:
wrkspc_REDACTED`.

From the table:

* **E1.** The key prefix does not tell the scope: keys A and C are both
  `sk-ant-usr-`.
* **E2.** Key C works on every endpoint the SDK calls once the header names a
  workspace its identity may use. Nothing else changes.
* **E3.** A key scoped to a workspace accepts the header naming that same
  workspace. Sending the header is harmless when it matches.
* **E4.** Key C could not list the organization's workspaces: its user lacks
  the Admin API permission. A wizard cannot rely on discovering the
  workspaces.
* **E5.** The documentation says a well-formed ID of a workspace that does
  not exist, or that the identity may not use, returns 404
  `not_found_error`, "Workspace `<id>` not found.". The probe's invented ID
  failed the format check instead, so the 404 is UNCONFIRMED here, and the
  PLAN's Phase 1 measures it. *Annotated 2026-10-27: Phase 1 could not;
  see the amendment of 2026-10-27.*

### Findings in the code, verified at 876b81e

* **F1.** Generation sets `x-api-key` and `anthropic-version` and no other
  Anthropic header: `llmprovider/providers/claude/claude.go:336-337`. No
  option of the provider names a workspace.
* **F2.** The listing does the same: `fetchClaudePage`,
  `llmprovider/catalog/discovery.go:801-809`.
* **F3.** The listing cache's key carries the Kilo organization and no other
  scope (`llmprovider/catalog/listing_cache.go:26-37`). A Claude listing per
  workspace would share one entry.
* **F4.** A 400 from Anthropic is classified by status alone:
  `classifyAPIError`'s default, `statusSentinel`, gives `ErrInvalidRequest`
  (`llmprovider/api_error.go:396-453`). The precedent for reading a 400's
  message is xAI's refused key (`api_error.go:414-419`, 0028-MADR H11).
* **F5.** `refusedCredential` (`wizard/configure.go:470`) does not match
  `ErrInvalidRequest`, so `credentialStep` (`configure.go:326`) does not ask
  again, `discoverModels` warns it as an outage (`configure.go:630`), and the
  credential is saved.
* **F6.** The wizard words every such notice from the error's class and HTTP
  status, never the service's text
  (0056-MADR-wizard-credential-only-entry-point.md D3, F7;
  `wizard/describe.go`). A new refusal must be a class, not a message.
* **F7.** The Kilo organization is the precedent for a per-provider routing
  value: `kilo.WithOrganization` for generation,
  `catalog.WithKiloOrganization` for the listing, `wizard.Result.Organization`
  for the consumer to persist (0017-MADR D2; `wizard/configure.go:54`).

## Decision Drivers

* A documented Anthropic key type must work: generation, streaming, the
  listing and the probes.
* The wizard must not save a credential that cannot generate, and must say
  what is wrong in words the user can act on.
* Notices stay worded from classes (0056-MADR D3).
* New exports are few and each has a call site (R55); the shape follows the
  Kilo organization's, so a consumer who wired one can wire the other.
* No behaviour changes for keys that already work (E3 makes the header
  harmless, but it is still sent only when set).

## Considered Options

* A. A workspace option on the provider and the listing, a classified
  refusal, and a wizard prompt
* B. A generic extra-headers option
* C. No SDK change: consumers wrap the transport, and users are told to
  create workspace-scoped keys
* D. Option A, plus discovering the workspaces through the Admin API

## Decision Outcome

Chosen option: "A. A workspace option on the provider and the listing, a
classified refusal, and a wizard prompt", because it makes the key type work
everywhere the SDK calls Anthropic with three small exports shaped like the
Kilo organization's, keeps notices worded from classes, and stops the wizard
saving a key that cannot generate. D is deferred by E4.

### D1. Generation: `claude.WithWorkspace`

`claude.WithWorkspace(id string) llmprovider.Option`, scoped to
`ProviderClaude` by `llmprovider.ScopedOption`, as `kilo.WithOrganization` is.
When `id` is not empty, every request the provider sends carries
`anthropic-workspace-id: <id>`: Generate, Stream, and the probes ListModels
runs. The provider passes it to its listing (D2), as kilo passes its
organization. Empty sends no header, which is today's request.

The value is trimmed and not otherwise validated: Anthropic is the authority
on what a workspace ID is (E5's replies say so precisely). A value with a
control character is refused with `ErrInvalidRequest` at `New`, because it
cannot be a header value.

### D2. The listing: `catalog.WithClaudeWorkspace`

`catalog.WithClaudeWorkspace(id string) llmprovider.Option`, scoped to
`ProviderClaude`, sets the header on each page `fetchClaudePage` requests. The
listing cache's key gains the workspace: the internal field `organization`
becomes a scope that holds the Kilo organization or the Claude workspace,
whichever the listing has, so two workspaces never share an entry (F3).
*Amended 2026-10-27: the probe cache is scoped by the workspace too; see the
amendment "the probe cache".*

### D3. The refusals: two classes

Two sentinels, each a child of `ErrInvalidRequest`, so a caller that matches
`ErrInvalidRequest` today still matches, and both terminal:

* `llmprovider.ErrWorkspaceRequired`, "workspace required": Anthropic's 400
  `invalid_request_error` whose message says the key "is not scoped to a
  workspace".
* `llmprovider.ErrWorkspaceNotFound`, "workspace not found": Anthropic's 400
  `invalid_request_error` whose message says the header "must be a valid
  workspace ID", and its 404 `not_found_error` whose message begins
  "Workspace" and ends "not found." (E5).

`classifyAPIError` matches each for the Claude service only, on the error type
and a fixed phrase of the message, the way it matches xAI's refused key (F4).
The message itself is still never shown (F6): the classes' texts are the
wording. `wizard`'s `errorClasses` lists both before `ErrInvalidRequest`.

### D4. The wizard asks for the workspace

`wizard.Result` gains `Workspace string`: the Anthropic workspace a key not
scoped to one runs in, empty for every other key and provider. Its doc names
`claude.WithWorkspace` and `catalog.WithClaudeWorkspace`, as `Organization`'s
names Kilo's. `Options.Existing.Workspace` pre-fills it, and the result's
formatted view shows it.

`discoverModels` passes `catalog.WithClaudeWorkspace(res.Workspace)` with the
listing's other options. In `credentialStep`, after the listing:

1. If the failure matches `ErrWorkspaceRequired` or `ErrWorkspaceNotFound`,
   the key is not refused: the wizard notifies `Claude (Anthropic) needs a
   workspace for this key (workspace required, HTTP 400)` (or `workspace not
   found`, with its status), asks `Enter the workspace ID (wrkspc_…) from
   Console → Settings → Workspaces`, and lists again with it.
2. It asks at most `maxTypedKeys` times, as for keys (0028-MADR D-H10). Then
   it returns an error wrapping the last failure, and saves nothing.
3. The first prompt's default is `Existing.Workspace` when the existing
   provider is Claude, else the value of `ANTHROPIC_WORKSPACE_ID` when
   `AllowEnv` is set and the variable is not empty. The user confirms it
   either way.
4. A workspace that lists is kept in the Result, and the credential is saved
   as today.

*Amended 2026-10-27: "pre-fills" and D4.3 are made exact by the amendment
"D4's pre-fill".*

A key scoped to one workspace never sees the prompt. When `Discover` is off
there is no listing, so there is no check, as for keys today.

### D5. Discovery is deferred

Listing the workspaces through `GET /v1/organizations/workspaces` would let
the wizard offer names instead of asking for an ID. It answered 403 for the
owner's key C (E4), so it could not be confirmed live, and it brings the Admin
API's surface into the provider. A later record may add it, as a shortcut
before D4's prompt.

### D6. The exports and their call sites (R55)

| Export | Call site that needs it |
| --- | --- |
| `claude.WithWorkspace` | a consumer's provider construction, such as prepare-commit-msg's `main.go`, beside its `kilo.WithOrganization` |
| `catalog.WithClaudeWorkspace` | the wizard's `discoverModels`; the Claude provider's listing options |
| `llmprovider.ErrWorkspaceRequired` | the wizard's `credentialStep`; a consumer telling the user to configure a workspace |
| `llmprovider.ErrWorkspaceNotFound` | the same, for a wrong or unusable workspace |
| `wizard.Result.Workspace` | a consumer persisting the wizard's result, as it persists `Organization` |

No other exported name changes.

### D7. Docs

`docs/architecture.md` names the option and the classes; the README's Claude
section says when a workspace is needed; `CHANGELOG.md` gains the additions
under Unreleased. The migration guide from mcplib is unchanged: nothing it
maps is replaced.

### Consequences

* Good, because a key not scoped to a workspace works for generation,
  streaming, the listing and the probes (E2).
* Good, because the wizard tells the user what is missing, asks for it, and
  no longer saves a key that cannot generate.
* Good, because a consumer that matches `ErrInvalidRequest` is unaffected,
  and keys that work today send exactly the requests they send today.
* Neutral, because each consumer must persist `Result.Workspace` and pass it,
  which needs a change and a record in each consumer (prepare-commit-msg
  first).
* Bad, because the user must find the workspace ID in Console; D5 would
  remove that step only for users with the Admin API permission.
* Bad, because the classification matches English phrases of Anthropic's
  messages. A reworded message falls back to `ErrInvalidRequest`, which is
  today's behaviour; Phase 1's live test catches the change.

### Confirmation

* Offline tests, each seen to fail on a planted breach first: the header is
  sent when the option is set and absent when it is not, on Generate, Stream,
  a probe and each listing page; the cache keeps two workspaces apart; each
  recorded reply (redacted fixtures) classifies to its sentinel and still
  matches `ErrInvalidRequest`; the wizard asks for a workspace on each class,
  stops after `maxTypedKeys`, saves nothing then, and keeps the workspace
  that lists.
* A live test, `LLMPROVIDER_LIVE_0064`, with a key not scoped to a workspace
  in `ANTHROPIC_API_KEY` and its workspace in
  `LLMPROVIDER_LIVE_0064_WORKSPACE`: no header classifies to
  `ErrWorkspaceRequired`; the header lists and generates; a malformed and an
  unusable workspace classify to `ErrWorkspaceNotFound`.
* `make api-check` lists exactly D6's additions.
* The owner configures prepare-commit-msg with key C through a build that
  uses this SDK, and a commit message is generated.

## Pros and Cons of the Options

### A. A workspace option on the provider and the listing, a classified refusal, and a wizard prompt

* Good, because it covers every request the SDK sends to Anthropic.
* Good, because it mirrors the Kilo organization's options and Result field.
* Good, because notices stay worded from classes.
* Neutral, because it adds five exported names.
* Bad, because the user types an ID they must look up.

### B. A generic extra-headers option

* Good, because it serves any header a provider adds later.
* Bad, because it can override the credential and `anthropic-version`
  headers, which the providers own.
* Bad, because the listing cache cannot know which headers change a listing.
* Bad, because the wizard still misclassifies the refusal and saves the key.

### C. No SDK change

* Good, because nothing changes here: a consumer can add the header with its
  own `http.RoundTripper` through `llmprovider.WithHTTPClient`.
* Bad, because every consumer writes the same transport, and the wizard,
  which builds its own requests, still saves a key that cannot generate.
* Bad, because telling users to create a workspace-scoped key leaves a
  documented key type unsupported.

### D. Option A, plus discovering the workspaces

* Good, because a user with the Admin API permission picks a name instead of
  typing an ID.
* Bad, because the owner's key answered 403 (E4), so the path cannot be
  confirmed live now.
* Bad, because it adds the Admin API's surface for a shortcut.

## Owner questions

* **Q1 (D3).** Two sentinels, as D3 decides, or one,
  `ErrWorkspace` ("workspace required or not found"), at the cost of one
  wording for both cases?

  **Answered 2026-10-26: two sentinels, as D3 decides.** One `ErrWorkspace`,
  and `ErrWorkspaceRequired` alone (a wrong workspace left
  `ErrInvalidRequest`), were not chosen.
* **Q2 (D4.3).** Offer `ANTHROPIC_WORKSPACE_ID` as the prompt's default when
  `AllowEnv` is set? The name is the one Anthropic's SDKs read for Workload
  Identity Federation; whether they read it for API keys is UNCONFIRMED.

  **Answered 2026-10-26: as D4 writes.** `Existing.Workspace`, else
  `ANTHROPIC_WORKSPACE_ID` when `AllowEnv` is set and it is not empty, the
  user confirming either way. `Existing.Workspace` alone, and a variable of
  this module's own, were not chosen.
* **Q3 (D5).** Defer discovery, as D5 decides?

  **Answered 2026-10-26: deferred, as D5 decides.** Discovery through the
  Admin API now, falling back to the prompt, and ruling it out for good,
  were not chosen.

## More Information

### Related

* 0017-MADR D2: the Kilo organization, the shape D1, D2 and D4 follow.
* 0028-MADR H11 and D-H10: a 400 that names the credential, and the bound on
  typed answers.
* 0056-MADR-wizard-credential-only-entry-point.md D3: notices worded from
  classes.
* Anthropic, "Authentication": Select a workspace, and the 400 and 404
  replies; "Workspaces": API keys and resource scoping.

### After this record

prepare-commit-msg persists `Result.Workspace` in its per-provider config and
passes `claude.WithWorkspace` when it builds the Claude provider, beside its
`kilo.WithOrganization` (`main.go:358-359`, `internal/ui/setup.go:238`,
`:293`). That is a change and a record in that repository, after this SDK's
release.

## Amendment 2026-10-26: the findings at a878fa4

**Status: accepted with the owner's answers (2026-10-26).** The Findings in
the code were verified at `876b81e`; between that and `a878fa4`, `v1.23.0`
(0061-MADR, 0062-MADR) moved lines and changed none of them. Each holds, at:
F1, `llmprovider/providers/claude/claude.go:347-348`; F2,
`fetchClaudePage`, `llmprovider/catalog/discovery.go:866`, its one
Anthropic header `anthropic-version`; F3,
`llmprovider/catalog/listing_cache.go:26-37`, unchanged; F4,
`classifyAPIError`, `llmprovider/api_error.go:396`, xAI's refused key at
`:414`, the default `statusSentinel` at `:451-452`; F5, `credentialStep`
(`wizard/configure.go:326`), `refusedCredential` (`:470`), the outage
notice at `:630`; F7, `wizard.Result.Organization` (`configure.go:50-52`).
`make api-check` now compares against `v1.23.0`, the latest tag, so D6's
five additions are measured against it.

## Amendment 2026-10-27: Phase 1's measurement

**Status: accepted; D3 stands, E5 stays unconfirmed, and `fixtures-check`
gains workspace IDs (the owner, 2026-10-27).**

The owner ran `TestLive_Workspace_0064` (0064-PLAN Phase 1) three times on
2026-10-27, in their own terminal, against `https://api.anthropic.com`, with
a key provided by an organization and not scoped to a workspace. The first
key had expired; runs 2 and 3 used its replacement, key D. Neither key's
type nor prefix was recorded. The redacted replies are the fixtures in
`llmprovider/testdata/anthropic-workspace/`.

| Request (`GET /v1/models?limit=1`) | Run 1: the expired key | Runs 2 and 3: key D |
| --- | --- | --- |
| no workspace header | 401 `authentication_error`, "invalid x-api-key" | 400 `invalid_request_error`, the Context's message, word for word |
| header `not-a-workspace` | 400 `invalid_request_error`, "anthropic-workspace-id header must be a valid workspace ID." | the same |
| header naming the Default workspace of key D's organization | 401, as with no header | 200, and the reply carries `anthropic-workspace-id` |
| header naming a workspace of key D's organization, created for run 3 and never associated with key D | not sent | 200, the same listing byte for byte, and the reply carries the header |

* **E6.** A second key not scoped to a workspace, an organization's, gets
  the Context's 400 and the malformed header's 400 as key C did, and lists
  once the header names its workspace. E2, and D3's two 400 phrases, hold
  for it.
* **E7.** An expired key is refused as any invalid key is: 401
  `authentication_error`, "invalid x-api-key", with no header and with a
  valid workspace, which `ClassifyHTTPError` classes `ErrAuthFailure` today.
  The header's format is checked before the key: the malformed header got
  its 400 from the expired key too. D3 and D4 leave this reply alone; the
  wizard reports a refused key and does not ask for a workspace.
* **E8.** Key D was accepted in a workspace of its organization that it had
  never been associated with. Anthropic's documentation, re-read
  2026-10-27, bounds such a key by its identity, not by an association with
  the key: "The key's user or service account must have access to that
  workspace" (Workspaces, How workspaces work), and a personal key works in
  "the workspaces where your role allows API use" (Authentication, Key
  types). Key D's type and its identity's role were not recorded, so run 3
  does not show which rule admitted it. It did not test E5: the new
  workspace was one key D's identity could use.
* **E5 stays UNCONFIRMED.** Asked whether to measure it with a workspace of
  another organization, record it as unmeasured, or measure it later, the
  owner chose to record it as unmeasured. The documentation, re-read
  2026-10-27 (Authentication, Select a workspace), says: "If the workspace
  doesn't exist, or the key's user or service account doesn't have access
  to it, the API returns a 404 `not_found_error` with the message
  ``Workspace `<id>` not found.``, the same response as for any unknown
  workspace." It gives no example body. D3's 404 branch rests on that
  sentence: `not-found.json` is written from it, says so in its `source`,
  and is the fixture Phase 2 classifies. Whether the reply quotes the ID in
  backticks, as the documentation writes it, is unmeasured; D3's match, on
  "Workspace" at the start and "not found." at the end, holds either way.
  If Anthropic answers otherwise, the branch never fires and the reply stays
  `ErrInvalidRequest`, as today.
* **The fixture gate.** `scripts/check_fixtures.py`
  (0047-MADR-keep-five-more-scratch-helpers-as-tools.md D1) had no kind for
  a workspace ID, and AGENTS.md's "Recorded fixtures" named none, so
  0064-PLAN's rule that a recorded workspace ID becomes `wrkspc_REDACTED`
  and `make fixtures-check` passes was not checked. Asked to add the kind in
  Phase 1, write a separate record for it, or correct the rule's wording
  only, the owner chose to add it in Phase 1: a kind `workspace-id`,
  `wrkspc_` then letters and digits, never a hit when they are a
  placeholder (`REDACTED`, a run of `X`); `scripts/test_gates.py` plants
  one; AGENTS.md names the redaction. 0047-MADR D1 carries an annotation
  pointing here.

## Amendment 2026-10-27: the probe cache

**Status: accepted (the owner, 2026-10-27). D2 is extended; D6 is
unchanged.**

Found at `0375b46`, before Phase 3's first write:

* **F8.** The Claude provider's `ListModels` probes its listed models
  through `wire.ProbeHealth` (`llmprovider/providers/claude/claude.go:472`),
  which keeps the models that answered for 10 minutes under
  `transport.ProbeKey{Provider, BaseURL, Credential}`
  (`llmprovider/internal/transport/probe.go:101-103`, keyed at
  `llmprovider/internal/wire/probe.go:28`), and lets the probes of one key
  run once at a time (`probe.go:189-205`). D2 scoped only the listing cache,
  so two workspaces under one key would share one entry and one probe in
  flight: a model that answered in one would be listed in the other for
  10 minutes. Only a list with a model in it is kept (`probe.go:197`), so a
  run with no workspace, every probe refused, leaves nothing behind. Whether
  Anthropic lets one workspace use a model another cannot was not
  measured. Kilo does not probe: `ProbeHealth`'s callers are Gemini,
  chatcompat, Claude, Grok, Ollama and OpenAI.

Asked to key the probe cache by the workspace too, record the sharing as a
limit, or leave it to a separate record, the owner chose to key it:
`transport.ProbeKey` gains an internal `Scope`; `wire.ProbeHealth` takes
it; the Claude provider passes its workspace, and every other caller `""`,
so their keys are as they were. Nothing exported changes.

## Amendment 2026-10-27: D4's pre-fill

**Status: accepted (the owner, 2026-10-27). D4 is made exact; nothing else
changes.**

D4 read two ways: its first paragraph has `Existing.Workspace` pre-fill the
Result, while D4.3 has it as the first prompt's default, which the user
confirms, and D4.1 prompts only after a refusal. Read the second way, every
re-run with a key not scoped to a workspace would send one refused request
and show a warning before the prompt. Asked to pre-fill and ask on a
refusal, to list without a workspace first, or to confirm the existing
workspace before listing, as the existing key is, the owner chose the
first:

* When the provider is Claude and `Existing.Provider` is Claude, the first
  listing sends `Existing.Workspace`. A listing that succeeds asks nothing,
  and the Result keeps it.
* Only a workspace refusal notifies and prompts, as D4.1 says. The first
  prompt's default is D4.3's: `Existing.Workspace` when the existing
  provider is Claude, else `ANTHROPIC_WORKSPACE_ID` under `AllowEnv`. A later
  prompt has no default.
* An empty answer lists with no workspace, so a key that no longer needs
  one is not held to it.
