---
status: accepted
date: 2026-11-24
decision-makers: repository owner
consulted: "the owner's reply of 2026-11-22 on 0088-MADR OQ9, the owner's question on omitting the limit and the owner's design note, each relayed by the coordinator and quoted or assessed below; 0088-MADR-default-output-limit-from-model-facts.md and its report; record reviews 1 to 3 of 2026-11-22 (review 3 of this record's first draft), each finding re-verified against the code before it was written in; research rounds 1 to 3 of 2026-11-22, each claim this record rests on re-read at its source; this repository at 751e970 and gobble-cli at dfaa1ac, read only; the public pages and source files named under More Information"
informed: every caller of Generate and Stream that names no output limit; gobble-cli first, whose 0066-MADR uses this record's API to set each session's model up; prepare-commit-msg and magic-cli-remote
---

<!-- markdownlint-disable MD013 MD024 -->

# Discover Each Model's Output Limit, Window and Their Evidence, and Send a Limit Only Where It Is Needed

Associated PLAN: [0089-PLAN-model-output-limit-discovery.md](0089-PLAN-model-output-limit-discovery.md)

*(2026-11-22, second draft, after record review 3: a smaller API (R55), learned windows dropped and passed-back caps expired, the validator's model match by resolved and canonical id, the one-model and server reads stored apart from the listing, a limits-only models.dev extract embedded in the SDK, the refusal forms mapped to their wires, the owner's design note assessed, and the questions regrouped: the cold request with the unknown Messages case, and the ceiling with gobble-cli's override.)*

*(2026-11-22, the owner's answers: the owner answered OQ1 to OQ4, relayed by the coordinator ("Owner rulings" below), and 0088's OQ18, which puts this record in the release after the next tag, together with 0088. Every row is decided or out of scope but DL65, which awaits live evidence; by the records rule the record is `accepted`. Its PLAN stays `proposed` until the owner approves it.)*

## Context and Problem Statement

[0088-MADR-default-output-limit-from-model-facts.md](0088-MADR-default-output-limit-from-model-facts.md)
decided that a request naming no output limit is sent the model's stated limit.
Its open question OQ9 asked what is sent when nothing states one. The owner's
reply of 2026-11-22, verbatim:

> "option 1 wording is confusing. stated limit or nothing, yet, nothing is not nothing, nothing is forcing an 8192 limit on anthropic models which all can support 64k or higher, so nothing means nothing. we need to address this as a real gap. my proposal is to implement a model output token discovery feature in llmprovider-sdk that provides an API surface that clients (gobble) can call to determine the model output token values acccurately to set them correctly in sessions. evaluate this and dig into research/web research similar use cases and solutions other related projects are using to resolve this ambiguity and ensure accurate model setup for sessions. be thorough. be detailed. write a new pair in the sdk repo for work there, and a pair in the gobble-cli repo for the adjoining work there to get this in place in a forward-thinking, robust, and effective manner."

While this pair was drafted the owner asked, verbatim, a question and not yet a
ruling:

> "on the maxoutput tokens issue, one thing, most providers and models to NOT require that to even be set. anthropic does, and we need to work with that, but why not simply omit setting any max output token value at all to any provider that does not require it?"

OQ1 puts that as its recommended option, with the costs the evidence shows.
The owner also supplied a design note for assessment; "The owner's design
note" below says what this record takes from it.

This record is the SDK pair. It is also the record 0088's OQ12 called for
("New record, all three (Recommended)": the limit sent and its source, a
per-request ceiling that is not "given", a refused cap reported and
remembered).

### The gap, at 751e970

* **A request knows only what is already cached.** Each provider reads its
  facts by `catalog.CachedFacts` (C10, C11), which starts no fetch (C5); "No
  request waits, and no request lists" (0041-MADR D1, C29). Before a listing
  is cached, and offline, a request knows only what a caller's layer says:
  the models.dev document is fetched, not embedded (C44).
* **An alias is not in Claude's listing for dated generations.** The 4.5
  models are listed under dated ids (`claude-sonnet-4-5-20250929`, C24), so
  the alias `claude-sonnet-4-5` is in no entry; the 4.6 and later models are
  listed under undated ids. Anthropic's one-model read resolves an alias
  (W2).
* **An unknown limit is a number.** The default is 8192 (C14) on every wire,
  including the optional ones; Messages requires a value. That is the owner's
  "nothing is not nothing".
* **A fact carries its source's name only.** `ModelInfo` holds one merged
  value per field and `SourceOf` names the layer that set it (C1, C2). The
  losing sources' values and when each was read are not kept.
* **A caller's models.dev copy outranks the service.** Every caller layer
  ranks above the provider's listing (C3), and a models.dev document handed in
  as a layer keeps the name "models.dev" (C7, C8). gobble-cli hands its
  snapshot that way (gobble-cli 0065-MADR D14).
* **The listings carry facts the SDK drops** (the SDK's own fixtures):
  Together's `config.max_output_length` (C21; its struct reads no such field,
  C20), Hugging Face's per-provider `context_length` (C23), the ChatGPT
  backend's `context_window` (C22), Kilo's `per_request_limits` (null in every
  fixture entry, C25). Claude and Gemini state their window as an input limit
  (C16, C18).
* **A refusal that states the limit is read for a yes or no only.** The
  overflow forms parse the window and the split and keep neither (C27, C28).
  No code parses Anthropic's `max_tokens: N > M` form; the SDK's only copy is
  a test's invented string (C31).
* **Local servers.** Only Ollama's dialect is recorded (C26); its window is the
  trained context from `/api/show`, not the context the server serves, which
  `/api/ps` reports for a loaded model (W6).
* **A full listing refresh replaces the models it stored** (C45), and a
  request reads stored models of any age (C46).

### What others do (research round 3; the sources under More Information)

* **Live first-party discovery** exists where the service states the limit:
  Anthropic's Models API (`max_tokens`, "Maximum value for the `max_tokens`
  parameter when using this model", W1), Gemini's `outputTokenLimit`,
  OpenRouter's per-model and per-endpoint `max_completion_tokens` (W4), and
  Kilo's OpenRouter-shaped `top_provider.max_completion_tokens` (C20b). OpenAI,
  xAI, Together's documented schema, Hugging Face, Mistral, Cerebras and
  Bedrock state no output limit in any listing.
* **Clients with an unknown model**: llm-anthropic uses 64,000, "the lowest
  output limit of any current model, so requests are never rejected" (W9).
  Vercel's AI SDK marks it `knownModel: false` and warns when its fallback is
  used (W10). Pydantic AI keeps the limit `None` and sends 16,384 (W11).
  LiteLLM's bundled map gave Claude Sonnet 4 an output limit of 1,000,000, its
  window (W12). Cline carries `source: "unknown"`.
* **Learning from refusals**: Hermes parses the stated cap; its open issue
  #150519 (W3) records an output-cap refusal read as a context window,
  persisted, so that "every later agent init fails ... until the cache row is
  deleted by hand"; its reporter's fix: "An output-cap error never becomes a
  persisted context window."
* No library sends a deliberately oversized limit to read the cap up front.

## Decision Drivers

* The owner's words: a limit comes from discovered evidence, never from a
  picked number; "nothing means nothing"; and the question whether a limit is
  needed at all where the wire does not require one.
* "Unknown" is a state a caller can see, never a silent number.
* Every fact carries where it came from and when it was read; disagreement is
  shown, not hidden.
* A learned value is safe: validated before use, never a window, never
  persisted without an expiry, never above a stated source (W3).
* A caller's limit is handled as 0088 ST15 to ST18 decide.
* No regression: a request that succeeds today still succeeds.
* R55: each export has a named call site (C33).

## Considered Options

* A. A discovery API over the catalog: each fact with its evidence and time,
  unknown as a state, an embedded registry extract below the live sources,
  validated refusals as a bound; the wire default reads it
* B. Probing: a request sent to learn the cap (an oversized limit, or
  descending tiers)
* C. Learning from refusals alone
* D. A bundled registry as the SDK's only source
* E. Leave it to callers: the SDK keeps sending what it sends; a caller states
  limits in layers
* F. Escalation: send a guess, and resend larger after a cut answer

## Decision Outcome

Chosen option: "A. A discovery API over the catalog", because it uses every
source that states a limit, keeps each source's value and time so that a
conflict is visible and resolved by rank, represents "unknown" as a state, and
serves a caller setting a session up (`LookupLimits`) and a request
(`CachedLimits`) from one rule; B, C, D and F are parts of A or rejected below.

### The surface (exported; names re-read at Phase 1's start)

The smallest surface with a named call site (R55; gobble-cli's 0066-MADR names
each). The catalog's pair names follow `LookupModel` and `CachedFacts`.

In `llmprovider`:

```go
// SentLimit is what a request carried as its output limit (D7).
type SentLimit struct {
    Tokens int        // the value on the wire, after AddThinkingFor; 0 when Sent is false
    Sent   bool
    Basis  LimitBasis
    Source string     // the chosen evidence's source name, or the refusal form's
}

// LimitBasis is why a request carried its output limit. A closed set (R21).
type LimitBasis string

const (
    LimitBasisRequest  LimitBasis = "request"  // Request.MaxOutputTokens
    LimitBasisOption   LimitBasis = "option"   // WithMaxTokens, or a listing probe
    LimitBasisFact     LimitBasis = "fact"     // the chosen fact, held to the window and the room
    LimitBasisRefusal  LimitBasis = "refusal"  // resent at a validated refusal's stated cap
    LimitBasisFallback LimitBasis = "fallback" // OQ2's answer: no source states the limit on Messages
    LimitBasisNone     LimitBasis = "none"     // nothing sent
)

// Response gains: OutputLimit SentLimit

// LimitObservation is an output cap a service stated when it refused a
// limit (D5). It is a value, safe for concurrent use while no goroutine
// changes it.
type LimitObservation struct {
    Model   string    // as the request named it
    BaseURL string    // the provider's base URL; empty for its default
    Scope   string    // the Kilo organization or Claude workspace; empty for none
    Tokens  int
    Seen    time.Time
}

// WithLimitObservations passes back observations a WarningLimitLearned
// reported (D5). New refuses, with ErrInvalidRequest, one with no model,
// Tokens below 1, or a zero Seen.
func WithLimitObservations(obs ...LimitObservation) Option

const WarningLimitLearned WarningKind = "limit_learned" // Warning.Observed holds it

// Warning gains: Observed LimitObservation `json:",omitzero"`
```

In `catalog`:

```go
// Limits is what is known of one model's limits on one provider, with the
// evidence for each (D1). A value, safe for concurrent use while no
// goroutine changes it.
type Limits struct {
    Provider llmprovider.ProviderID
    Model    string
    Output   Fact      // the most the output-limit field may carry
    StreamedOutput Fact // (2026-11-23, DL76) Output for a streamed request
    Window   Fact      // input plus output, where a source states it
    Input    Fact      // the input limit, where a source states it apart (Claude, Gemini)
    Required bool      // the wire requires an output limit (Messages)
    Checked  time.Time // when the sources were read
}

// Fact is one limit: the chosen value and every source's.
type Fact struct {
    Tokens      int        // the chosen value; 0 when not Known
    Known       bool
    Conflicting bool       // two sources state different values
    Evidence    []Evidence // every source that stated it, the chosen first, then by rank
}

// Evidence is one source's statement of a fact.
type Evidence struct {
    Source string    // "listing:claude", "server:ollama", "models.dev", "models.dev (canonical id)",
                     // "models.dev (family)", "models.dev (bundled 2026-11-20)", a layer's Name,
                     // "remembered"; and (2026-11-23, deviation 7's design note) "observed",
                     // a cap the memo keeps (D5), which bounds the chosen value (D1)
    Tokens int
    Seen   time.Time // when it was read or observed; zero for a caller's layer
    Stale  bool      // read longer ago than its source's freshness (D2)
}

// LookupLimits reads every source of model's limits on provider, bounded as
// LookupModel is, warms the caches a request reads, and returns what they
// state (D2). A model no source knows is not an error. A failing source is
// reported in the error, joined, beside what the others gave. Errors:
// ErrInvalidProvider for an id LookupModel refuses; the closed-cache error
// LookupModel returns; ctx's error; each source's own (*APIError, or a plain
// error for a transport failure).
func LookupLimits(ctx context.Context, provider llmprovider.ProviderID, src llmprovider.TokenSource,
    model string, opts ...llmprovider.Option) (Limits, error)

// CachedLimits is LookupLimits without a fetch, as CachedFacts is: it starts
// none, unlike CachedModel. It is what a request reads.
func CachedLimits(ctx context.Context, provider llmprovider.ProviderID, model string,
    opts ...llmprovider.Option) Limits
```

The evidence's kind (caller, service, registry, remembered) is derived from
its source name, inside the SDK. `ModelInfo`, its merge and `LookupModel` are
unchanged; a provider's output limit is read from `Limits`. Conditional
exports: `Request.OutputCeiling` under OQ4 (2) or (3); `WithLimitLearning` or
`WithoutLimitLearning` as OQ3 answers; `WarningOutputLimitUnknown` under OQ2
(3).

### Owner rulings (2026-11-22)

Each answer is the owner's, relayed by the coordinator, its label quoted as
given.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ1 | "Only where required (Recommended)" | (1): a limit only where the wire requires one; nothing on optional wires; Kilo sends the discovered limit until measured | DL46 to DL48, DL69; DL65's interim; 0088 ST74, ST85, ST86, ST90, ST92, ST98, and ST9 changed |
| OQ2 | "Never wait; route's smallest (Recommended)" | (1): no request waits; on Messages with nothing known, the route's smallest current limit, labelled fallback | DL50, DL51 |
| OQ3 | "By default, can turn off (Recommended)" | (1): caps remembered and reported by default; `WithoutLimitLearning()` turns that off | DL49, DL73; 0088 ST89 |
| OQ4, asked with gobble-cli's 0066 | first a question, verbatim: "how about option 1 and option 3? pros vs cons. explain."; the coordinator explained that under the omit rule a models.json `maxTokens` reaches only Messages routes and caps no optional wire, that option 3 adds a global `outputLimit` as a new SDK ceiling and an ACP option, with their costs, and that option 3 can be added later with nothing to undo; then the answer: "models.json only, for now (Recommended)" | (1): models.json `maxTokens` only; no SDK ceiling | DL52; gobble-cli 0066 S32 to S34 |
| 0088 OQ18 | "Five now; 0088+0089 next (Recommended)" | (1): the next tag carries 0081, 0083, 0084, 0085 and 0087, with the untagged 0080 and 0082 entries already in `[Unreleased]`; this record and 0088 go together in the release after; it changes the owner's 0088 OQ8 answer, "Join this tag" | 0088 ST43 |
| How `Limits` shows the memo's streamed and plain caps (2026-11-23, merged phase layer B) | "Add a stream dimension" (not the recommended option) | `Limits` reports a separate streamed output limit, `StreamedOutput`, beside `Output`: each held to the cap kept for its own call (D5's key has stream or plain) | DL76 |
| `TestThinkingWire_Claude`'s budget case, which DL55 changes (2026-11-23, merged phase layer B, finding B1) | first "Name it; add a caller layer (Recommended)"; then, corrected, "Explicit WithMaxTokens(8192) (Recommended)": the first answer's question had wrongly described a caller layer, and the build's probe on a scratch copy showed a caller layer stating 8,192 sends `max_tokens 8192 thinking map[budget_tokens:8191 type:enabled]` (0088 ST30 stops the raise at the facts' limit), while `WithMaxTokens(8192)` sends `max_tokens 13096 thinking map[budget_tokens:9000 type:enabled]` | the case gets the caller's `WithMaxTokens(8192)`, so `max_tokens` 13,096 and `budget_tokens` 9,000 stay pinned, and 0088 ST16 is exercised | 0088 ST101; DL55's annotation |
| Layer B's goldens (2026-11-23, finding B2) | "Merge layers B and C" | layers B and C of the merged phase are built as one layer, so ST100's goldens change in the layer that changes them; a merged layer is not a build pass | none (0089-PLAN deviation 8) |
| OpenCode's messages-route fallback (2026-11-23, merged phase layer B) | "Models on opencode's route table (Recommended)" | the smallest limit among the models the shared route table (`internal/opencoderoute`) lists for the gateway's messages route; the extract's whole section, which holds models the gateway no longer routes (`claude-3-5-haiku`, 8,192), is not read for it | DL77 |
| DL51 (2026-11-22, after OQ2) | "Refuse, say how to fix (Recommended)" | a Messages-format request for a model no source states any limit for is refused before sending, `ErrInvalidRequest`, its message saying to state the model's limit (for example models.json `maxTokens`, or a caller's `catalog.WithModelLayer`); no invented number | DL51 |

So the exports are the 18 lines of the PLAN's list and `WithoutLimitLearning`;
neither `WarningOutputLimitUnknown` nor `Request.OutputCeiling` is added.

### The decisions

* **D1. The evidence and its rank.** Each source is read apart and kept, and
  ranked: the caller's layers, then the service (its listing, the one-model
  read, a local server's details), then the registry (the fetched models.dev
  document, then the embedded extract, D3), then a remembered observation,
  which fills a gap only (D5). A layer whose name is the models.dev document's
  (C7) is registry evidence wherever it came from. Within the service, a
  server's served context ranks above the trained one; within the registry,
  an exact id above a canonical-id match above a family match (D3). Two
  sources stating different values make the fact conflicting; the highest
  rank is chosen, every value kept. An observed cap (D4) is not a rank but a
  bound: the service refuses above it, so a chosen output limit above it is
  held to it. The output fact is never above the stated window (0088 ST91). A
  stated 0 is not a limit (0041-MADR A1).
* **D2. Discovery and storage.** `LookupLimits` reads what `LookupModel`
  reads, bounded by the same 10 s (C6), sharing one fetch per listing (C12),
  plus: for Claude, a model absent from the listing by `GET
  /v1/models/{model_id}`, which takes an id or an alias (W2); for Ollama and
  its dialect, `/api/ps`'s `context_length` of a loaded model (W6); and the
  fields D3 names. The one-model and `/api/ps` reads are stored in their own
  map in the cache, keyed as the listing is plus the model, so a listing
  refresh, which replaces its models (C45), does not drop them; a 404 is kept
  as "none" for the same freshness. Freshness is the listing's 10 minutes
  (C47); a request reads stored facts of any age, as it reads a listing's
  (C46), each marked `Stale` once older than its freshness, `Seen` giving the
  age. A closed cache is refused (C13), and with no network the caches, the
  layers, the extract and the remembered observations answer, the failures in
  the error.
* **D3. What each source states.** Claude `max_tokens` and `max_input_tokens`;
  Gemini `outputTokenLimit` and `inputTokenLimit`; Kilo
  `top_provider.max_completion_tokens` and, when set,
  `per_request_limits.completion_tokens` (the smaller), and `context_length`;
  Grok `context_length`; Together `context_length` and
  `config.max_output_length`; Hugging Face's per-provider `context_length`
  (the suffixed provider's when the id names one, else the smallest among live
  providers stating one); the ChatGPT backend's `context_window`; Ollama's
  served and trained context. The registry is matched by stages, each a
  source name of its own: the exact id; models.dev's `canonical_model_id`,
  which the catalog already reads (C48), compared by its canonical tokens
  (C49); a family match with a dated suffix (`-YYYYMMDD`) or a routing suffix
  after `:` removed, used only when nothing exact or canonical states the
  fact. An embedded, limits-only extract of models.dev (output, window and
  input per provider section and model), generated from a public models.dev
  document at each release by a maintainer tool, as OpenCode's route snapshot
  is embedded (C50), is the registry's offline tier: `Stale` always, its date
  in its source name. `Limits.Required` is true on the Messages routes (Claude
  and OpenCode's messages route). Reasoning accounting is documented per wire
  in the package docs (the table below), not exported.
* **D4. A refusal is evidence only once validated.** One recorded form per
  wire, mapped as DL68 decides. Output-cap forms: Anthropic's `max_tokens: N >
  M, which is the maximum allowed number of output tokens` (W7); OpenAI's
  `This model supports at most M completion tokens, whereas you provided N`
  (W7, a Chat Completions text); Gemini's `value of N but the supported range
  is from 1 (inclusive) to M (exclusive)`, read as M − 1 (W7, generateContent).
  Window forms: OpenRouter's as Kilo passes it, Kilo's documented one, OpenAI's
  split and Together's (0088 C38, C39, C97). The validator: (V1) a text
  matching an output-cap form and a window form is rejected; an output-cap
  form never sets the window and a window form never the output limit (W3);
  (V2) a stated N must equal the limit on the wire, after `AddThinkingFor`;
  (V3) M below N; (V4) M at least the wire's documented minimum (16 on
  Responses, W8; 1 elsewhere); (V5) a window above the input it states; (V6) a
  named model must be the request's model, the id the listing or one-model
  read resolved it to, or share its canonical tokens with the provider prefix
  dropped; otherwise the value serves this request's resend only and is not
  kept. *(2026-11-23, from 0089-PLAN's Phase 2 analysis: the resolved id reaches a provider through the internal `cacheref` value `catalog.WithCache` carries; no new export.)* A rejected refusal is returned as today. *(2026-11-26, from 0088-MADR ST113, the owner's "Drop the minimum for now": V4's 16 on Responses is not applied; `ResponsesLimits.Minimum` is 1, and the record that first makes a Responses route send a limit the SDK chose re-adds it. The Responses wire reads no refusal form, so V4 is not reached there either way.)*
* **D5. What is kept, and for how long.** A validated output cap is kept in a
  memo per `catalog.Cache` (the process's without one), keyed by provider,
  base URL, Kilo organization or Claude workspace, model and whether the
  request streamed (Gemini states different caps for a streamed and a plain
  call, 0088 ST95), at most 256 models, oldest dropped, never written to disk;
  so a wrong cap lasts at most one process. It is dropped when a response for
  its key reports more output than it. A window is never kept: an observed
  window serves the one request's resend only (a gateway's per-endpoint
  window, kept, would shrink every later turn). Whether the memo and its
  report run by default or on an option is OQ3. A `WarningLimitLearned`
  reports a kept cap once per key, with its base URL and scope; a caller may
  pass it back with `WithLimitObservations`, a common option that
  `LookupLimits` also reads. *(2026-11-23, from 0089-PLAN's Phase 2 analysis: a shared option that `catalog` reads through `Settings.Values()`, as it reads `WithModelLayer`'s layers.)* A passed-back cap is used only where its base URL
  and scope match, only while its `Seen` is within 24 hours (the SDK's window
  for serving a service's stale statement, C19), and only where no caller,
  service or registry source states the output limit; otherwise it is kept as
  evidence and the stated source chosen. So no learned value outranks a stated
  one, and none persists unexpired. *(2026-11-23, decided by the coordinator from evidence: the memo's per-model notes, the id a one-model read resolved (V6) and the route's fallback (DL51, DL77), stay internal to `internal/limitmemo` and `internal/cacheref`; no export.)*
* **D6. The rule a request follows.** A request's own `MaxOutputTokens` is
  sent as given; a `WithMaxTokens` value is cut to the stated limit and never
  raised (0088 ST15 to ST18); none on the ChatGPT routes (0088 ST4).
  Otherwise OQ1 decides where a fact adds the field at all. Where it does, the
  chosen output fact is sent, held to the window (0088 ST91), to an observed
  cap (D5) and, once 0088's OQ14 (3) is built, to the room its estimate
  leaves; where the wire requires a value and no source states one, OQ2's
  answer. A refused SDK-chosen value is sent once more at the validated stated
  number as OQ3 answers, sticky and outside 0062's bound (0088 A4, A5, OQ6).
* **D7. The report.** Every `Response`, and the stream's final one, carries
  `OutputLimit`: the value on the wire (after `AddThinkingFor` raises it, so
  the reported value is the sent one), whether one was sent, its basis and its
  source. This is 0088 OQ12's "the limit sent and its source".
* **D8. The ceiling.** 0088 OQ12's per-request ceiling had one named caller,
  gobble-cli's per-request fit, which the owner's OQ14 answer replaced; OQ4
  asks it together with gobble-cli's override setting, its only possible
  caller.
* **D9. Docs and the standard.** `docs/guides/api-standards.md` gains R57: "No
  fixed number decides an output limit a caller did not give: the model's
  evidence does (`catalog.Limits`), and a limit is sent only as the record of
  the default rule decides; a number with no source is listed with its
  provenance in the record that keeps it." R6's sentence on the output limit
  ("cuts the default and never raises it", C51) and its source order, and R7's
  `Response` fields (`OutputLimit`) change with it. Every doc line 0088 A8
  names says what the code does.
* **D10. Relation to 0088.** This record decides 0088's OQ12 rows (ST67,
  ST71). 0088's ST74, ST85, ST86, ST90, ST92 and ST98 wait for OQ1, and its
  ST89 for OQ3. 0088's OQ14 (3) estimate is built on `Limits.Window`, or
  `Limits.Input` where only that is stated, after this record's phases. Which
  release carries both is 0088's OQ18. *(2026-11-22: answered "Five now;
  0088+0089 next (Recommended)": the next tag carries 0081, 0083, 0084, 0085
  and 0087, with the untagged 0080 and 0082 entries; this record and 0088 go
  together in the release after, built once 0087 is finished.)*

### Every provider and route

| Provider, route | Field | Output evidence | Window or input evidence | When nothing is sent | Reasoning in the limit | Facts that add the field (OQ1 (2); under (1) only the required rows, and Kilo until DL65) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Claude | `max_tokens` | listing `max_tokens` (C15); one-model read for an absent id (W2); registry | input: `max_input_tokens` (C16) | required (Anthropic) | counted (thinking is inside `max_tokens`) | every kind |
| OpenCode messages | `max_tokens` | registry | registry | treated as required: the Messages wire requires it; the gateway's own rule is not documented | counted | every kind |
| OpenAI with a key, OpenCode responses | `max_output_tokens` | registry (the listing states none) | registry | optional; OpenAI documents no default; its own client sends none (0088 E16) | counted (OpenAI's 25,000 advice, 0088 WS2) | OpenAI: caller only; OpenCode: every kind |
| ChatGPT backend and plan | none | not sent | window: `context_window` (C22) | never sent: OpenAI asks it omitted, though M5 measured it accepted (C52, 0043-MADR D5) | counted | none |
| Grok | `max_output_tokens` | registry | window: `context_length` | optional; 128,000 visible tokens (0088 WS3) | not counted (xAI's schema) | caller only |
| Gemini | `generation_config.max_output_tokens` | listing `outputTokenLimit` (C17) | input: `inputTokenLimit` (C18) | optional; not documented for Interactions (the provider's only API, C53); generateContent's default is the model's own limit (0088 WS7) | counted (W5) | every kind |
| OpenCode google | `maxOutputTokens` | registry | registry | optional; not documented through the gateway | counted | every kind |
| Kilo | `max_tokens` | listing `top_provider.max_completion_tokens` (C20b), `per_request_limits` when set (C25) | window: `context_length` | optional; not documented (DL65) | counted | every kind |
| Together | `max_tokens` | listing `config.max_output_length` (C21); registry | window: `context_length` | optional; not documented | counted | every kind |
| Hugging Face | `max_tokens` | registry | window: per-provider `context_length` (C23) | optional; not documented | counted | every kind |
| OpenCode chat | `max_tokens` | registry | registry | optional; not documented | counted | every kind |
| Ollama, and chatcompat's Ollama dialect | `max_tokens` | none from the server | window: `/api/ps` served (W6), else `/api/show` trained | optional; unbounded (`num_predict` -1, 0088 WS9) | not established | caller only (0088 OQ7) |
| chatcompat, other servers | `max_tokens` | caller layers; registry by canonical id | caller layers; registry | optional; not established | not established | caller only (0088 OQ7) |

### The owner's design note

The owner supplied a design note for assessment (relayed 2026-11-22). What
this record takes from it, and what it does not:

* Taken: the provider matrix (D3; with Together's unread field, and Kilo's
  listing stating a limit); unknown as a state, never an invented ceiling
  (D1, `Fact.Known`); an embedded snapshot as the offline tier below the live
  sources (D3, DL70); staged id matching with its own provenance (D3, DL71);
  "requires a limit" as a field (`Limits.Required`); separating the discovered
  limit from the request's field and omitting it where optional (OQ1 (1)); no
  flat global ceiling (OQ1 (1) and (2)); a fit to the room left, as 0088's
  OQ14 (3), with its margin named as an unmeasured constant (the numbers
  table); a self-healing resend on a refusal that states the ceiling, with a
  validator (D4, D5), on by default or opt-in as OQ3 asks.
* Already so: the thinking budget below `max_tokens` (`AddThinkingFor` raises
  `max_tokens` above the budget, C54; DL55); a four-tier order with provenance
  (the catalog's layers and `SourceOf`, C1 to C3, which `Limits` extends).
* Not taken: "Grok counts reasoning in output" (xAI's schema: the limit counts
  visible output only, 0088 WS3); a public output-parameter name (each wire's
  field stays internal); a 12 to 24 hour catalog freshness and a "safety
  margin" constant (numbers with no evidence; the SDK keeps its 10 minutes and
  24 hours, C47, C19); discovery that blocks when the catalog is configured
  (against 0041-MADR D1; the caller awaits `LookupLimits` where it chooses, and
  OQ2 asks whether a cold request waits). The note's models.dev `catalog.json`
  endpoint is published (research round 3, models.dev's README) but not
  needed: the SDK reads the document it already fetches.

### Consequences

* Good, because a session set up through `LookupLimits` carries its model's
  own limit from the first request wherever one is sent, Claude aliases
  included, and the response says which source chose it or that none was
  sent.
* Good, because "unknown" is a state, and offline the embedded extract still
  answers for every model models.dev knows.
* Good, because a stale or wrong registry value no longer outranks a live
  listing, and every disagreement is shown with both values.
* Good, because a learned cap is validated, kept for one process, passed back
  only with a 24-hour expiry, never above a stated source, and a window is
  never learned (W3's failure cannot recur).
* Neutral, because `ModelInfo` and every existing lookup keep their merge.
* Bad, because 19 names are added in `apidiff`'s form (the PLAN's list and
  `WithoutLimitLearning`, by OQ3's answer), and removing any after the tag
  needs a record.
* Bad, because discovery reads two endpoints no request read before (Claude's
  one-model read, Ollama's `/api/ps`), each bounded as a listing is, and the
  embedded extract adds a release step and a file to the module.
* Bad, because three refusal forms rest on forum copies of the services' text
  (W7); a variant they do not show is returned unread.
* Bad, because OQ1 (1) or (2) replaces owner answers of 2026-11-21 in 0088
  (OQ2's ceilings, OQ4's 8192; under (1) also OQ7's raise from a caller's
  layer on local servers), each named in OQ1's costs.

### Confirmation

* The PLAN's tests, one per decided row, red at the base where a row changes
  behaviour, each with a plant caught on a scratch clone.
* `python3 -B tools/apidiff.py` against the newest tag at Phase 1's base lists
  the PLAN's list in its form (a new type once, no fields of a new type)
  beside those `[Unreleased]` carries.
* `python3 -B tools/claims.py` passes on this record, and each planted false
  row is caught.
* An independent review of the last build phase finds no blocker.

## State Table

"Decided" names what decides the row; each decided row's test and plant are
in the PLAN, one to one. "OQ" names this record's questions; "0088 OQ" that
record's. *(Second draft: rows marked "(revised)" changed after review 3;
DL66 to DL73 are new.)*

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| DL1 | Claude listing entry | decided | output from `max_tokens`, input from `max_input_tokens`; window unknown | C15, C16; W1 | `TestDiscover_ListingFacts_0089`; plant: `max_input_tokens` read as the window |
| DL2 | Claude model absent from the listing (the 4.5 aliases; a newer alias) (revised) | decided | one `GET /v1/models/{id}`, its `max_tokens` as service evidence under the request's name and its resolved id kept for V6; a 404 stored as none (DL66) | W2; C24 | `TestDiscover_ClaudeAlias_0089`; plant: the one-model read skipped: unknown |
| DL3 | Gemini listing entry | decided | output `outputTokenLimit`, input `inputTokenLimit` | C17, C18 | `TestDiscover_ListingFacts_0089`; plant: input read as window |
| DL4 | Kilo entry with both caps | decided | the smaller | C20b, C25; W4 | `TestDiscover_UnreadFields_0089`; plant: the larger chosen |
| DL5 | Kilo auto-routing entry with a null cap | decided | output unknown; window from `context_length` | 0088 C34; C20b | `TestDiscover_ListingFacts_0089`; plant: the null read as 0 and known |
| DL6 | Grok entry | decided | window from `context_length`; output registry only; not required | the 2026-10-14 fixture (no output field on 14 models); 0088 WS3 | `TestDiscover_WireFacts_0089`; plant: Grok marked required |
| DL7 | OpenAI with a key | decided | registry only; not required | the 2026-10-14 fixture; 0088 E16 | `TestDiscover_WireFacts_0089`; plant: marked required |
| DL8 | ChatGPT backend and plan (revised) | decided | window from `context_window`; never sent, as 0043-MADR D5 decided though M5 measured it accepted | C22, C52 | `TestDiscover_UnreadFields_0089` and `TestOutputLimit_Default_0089`'s plan case; plant: a field sent |
| DL9 | Together entry with `config.max_output_length` | decided | service evidence; without it, registry | C20, C21 | `TestDiscover_UnreadFields_0089`; plant: the field unread |
| DL10 | Hugging Face entry, providers stating `context_length` | decided | the suffixed provider's, else the smallest among live providers | C23 | `TestDiscover_UnreadFields_0089`; plant: the largest chosen |
| DL11 | OpenCode Zen and Go, every route (revised) | decided | registry only; the messages route required (the Messages wire), the gateway's own rule not documented | their listing carries ids only | `TestDiscover_WireFacts_0089`; plant: the messages route not required |
| DL12 | Ollama, a loaded model | decided | window: served above trained, conflicting when they differ; output unknown | W6; C26 | `TestDiscover_OllamaServed_0089`; plant: trained ranked first |
| DL13 | Ollama, a model not loaded | decided | the trained context only | W6 | `TestDiscover_OllamaServed_0089`; plant: the model reported unknown |
| DL14 | chatcompat at a server with no recorded dialect (llama.cpp, vLLM, LM Studio, OpenRouter) | out of scope | caller layers and the registry by canonical id, as today; a dialect needs a recorded reply (0061-MADR E2); the schemas are named in W13 | | |
| DL15 | A caller layer named "models.dev" | decided | registry evidence, below the service | D1; C7, C8 | `TestDiscover_EvidenceKinds_0089`; plant: classified as caller |
| DL16 | A caller layer of any other name | decided | caller evidence, above the service | D1; 0033-MADR D2 | `TestDiscover_EvidenceKinds_0089`; plant: classified as registry |
| DL17 | Two sources state different values (revised) | decided | conflicting; the higher rank chosen (caller, service, registry, remembered); every value kept | D1 | `TestDiscover_Conflict_0089`; plant: the lower rank chosen |
| DL18 | Two caller layers state different values | decided | the later layer chosen (C3), conflicting | C3 | `TestDiscover_Conflict_0089`; plant: the earlier chosen |
| DL19 | Stated output above the stated window | decided | held to the window, conflicting | 0088 ST91 | `TestDiscover_OutputAboveWindow_0089`; plant: kept above |
| DL20 | Stated 0 | decided | unknown; the 0 kept as evidence | 0041-MADR A1 | `TestDiscover_StatedZero_0089`; plant: 0 known |
| DL21 | No source knows the model | decided | every fact unknown; no error | D2 | `TestDiscover_Unknown_0089`; plant: an error returned |
| DL22 | A source fails | decided | the others' facts returned; the failure in the joined error | D2 | `TestDiscover_SourceFailure_0089`; plant: the error dropped |
| DL23 | A fact older than its source's freshness (revised) | decided | used as today (a request reads stored facts of any age, C46), marked `Stale`, `Seen` its age; this covers a listing served after a failure within 24 h (C19) | D2 | `TestDiscover_Stale_0089`; plant: `Stale` left false |
| DL24 | Offline, cold cache (revised) | decided | the layers, the embedded extract and the remembered observations; the rest unknown; the error names each failure | D2, D3 | `TestDiscover_Offline_0089`, an empty document served; plant: an empty document read as a fact *(2026-11-22, the owner's ruling "Split the test across phases (Recommended)": Phase 1 pins the layers, the extract and the empty document; Phase 2 adds the passed-back cap's case to the same test, so the row is fully pinned after Phase 2)* |
| DL25 | A closed `catalog.Cache` | decided | refused; no request reaches any server | C13 | `TestDiscover_Closed_0089`, every server counting requests; plant: the check removed, so the one-model or `/api/ps` server sees one |
| DL26 | ctx cancelled during discovery | decided | ctx's error, with what was read before | D2 | `TestDiscover_Cancelled_0089`; plant: the error swallowed |
| DL27 | Concurrent `LookupLimits` for one listing | decided | one fetch | C12 | `TestDiscover_Concurrent_0089`; plant: a fetch per call |
| DL28 | An id `LookupModel` refuses | decided | `ErrInvalidProvider` | D2 | `TestDiscover_InvalidProvider_0089`; plant: nil error |
| DL29 | `WithoutModelMetadata` (revised) | decided | the fetched document left out; the embedded extract and a "models.dev" layer still registry | C7; D3 | `TestDiscover_EvidenceKinds_0089`; plant: the extract dropped with the document |
| DL30 | The wire table (revised) | decided | `Required` true on Claude and OpenCode's messages route only; reasoning accounting in the package docs | the table's sources | `TestDiscover_WireFacts_0089`; plant: `Required` on Kilo |
| DL31 | An output-cap refusal passing V1 to V6 | decided | an observed bound; the resend per OQ3 | D4 | `TestLimitRefusal_Validator_0089`, case 1; plant: the form not recognised |
| DL32 | A text matching both forms | decided | rejected | V1; W3 | case 2; plant: read as a window |
| DL33 | A stated N other than the wire's value | decided | rejected | V2 | case 3; plant: N not compared |
| DL34 | M at or above N, or below the wire's minimum | decided | rejected | V3, V4 | cases 4 and 5; plant: either check removed |
| DL35 | A window form (revised) | decided | the window and the input stated or derived serve this request's resend only; never kept, passed back or shown in `Limits` | D4, D5 | `TestLimitRefusal_WindowForms_0089`; plant: the window kept in the memo |
| DL36 | A refusal naming a model (revised) | decided | kept when it is the request's model, its resolved id, or shares its canonical tokens with the provider prefix dropped; otherwise this request only | V6 | `TestLimitRefusal_NamedModel_0089`; plants: an exact-only compare (the alias case not kept); any name kept (the routed case kept) |
| DL37 | A request's own `MaxOutputTokens` refused with a stated cap (revised) | decided | kept in the memo (as OQ3 answers), `CachedLimits` shows it; not resent (0088 ST17, ST18); no warning, since an error carries no response | D4; 0088 ST17, ST18 | `TestLimitRefusal_CallerLimit_0089`; plant: resent *(2026-11-23, decided by the coordinator from evidence: a refused `WithMaxTokens` value is kept, not resent, exactly as a request's own value, since a caller's value is given (0088 ST15 to ST18); the test's option case pins it)* |
| DL38 | The memo (revised) | decided | per cache; keyed by provider, base URL, scope, model, stream or plain; 256, oldest dropped; never on disk; a process's life | D5 | `TestLimitMemo_Scope_0089`, `TestLimitMemo_Bound_0089`; plants: one memo for every cache; stream and plain merged; no bound |
| DL39 | A response reporting more output than a kept cap | decided | dropped | D5 | `TestLimitMemo_SelfHeal_0089`; plant: kept |
| DL40 | A passed-back cap (revised) | decided | used only where base URL and scope match, `Seen` within 24 h, and no caller, service or registry source states the output | D5; C19 | `TestLimitObservations_PassBack_0089`; plants: no expiry; another scope used; used above a registry fact *(2026-11-23, decided by the coordinator from evidence: the provider's default base URL is read as empty on both sides, the observation's and the lookup's, so that a cap passed back with no base URL matches a provider's own request, which names its default; the test's default-base case, plant: the default not read as empty)* |
| DL41 | A passed-back cap where a source states the output | decided | kept as evidence; the stated source chosen | D5 | as DL40, case 2; plant: the remembered value chosen |
| DL42 | An invalid passed-back observation | decided | `New` refuses it, `ErrInvalidRequest` | D5 | `TestLimitObservations_Invalid_0089`; plant: accepted |
| DL43 | Concurrent requests on one memo | decided | under a mutex | D5 | `TestLimitMemo_Concurrent_0089` under `-race`; plant: the lock removed |
| DL44 | A caller's value (revised) | decided | the request's own value sent as given; `WithMaxTokens` cut to the stated limit, never raised; basis request or option | 0088 ST15 to ST18 | `TestOutputLimit_Reported_0089`; plant: basis fact for a caller's value |
| DL45 | The ChatGPT routes | decided | nothing sent; basis none | 0088 ST4 | `TestOutputLimit_Default_0089`; plant: a field *(2026-11-23: in `llmprovider/providers/openai/output_limit_0089_test.go`, package `openai`, whose test helpers sign in the ChatGPT and plan sessions; the plant is `max_output_tokens` 8192 given on those routes, caught by the basis)* |
| DL46 | No caller value, an output fact known | decided *(2026-11-22: the owner's answer)* | on a Messages route (Claude, OpenCode's messages route) the chosen output fact, held to the window and to an observed cap, no fixed ceiling, basis fact; on an optional wire, nothing (DL47) | OQ1 "Only where required (Recommended)" | `TestOutputLimit_Omitted_0089`; plant: Claude omitted *(2026-11-23: `TestOutputLimit_Omitted_0089` reads only the wire, so that it runs red at the base; the basis, fact on Claude and none on an optional wire, is pinned by `TestOutputLimit_Reported_0089`)* |
| DL47 | No caller value, nothing known, optional field | decided *(2026-11-22: the owner's answer)* | nothing sent, basis none; the service's own default applies; and on an optional wire nothing is sent even where a fact is known | OQ1 "Only where required (Recommended)" | `TestOutputLimit_Omitted_0089`; plant: a fact sent on Gemini |
| DL48 | Grok, OpenAI with a key, Ollama, chatcompat with registry or caller-layer facts | decided *(2026-11-22: the owner's answer)* | nothing sent, whatever a registry or a caller's layer states; a caller's own value still sent; 0088 OQ7's raise from a caller's layer on local servers (ST9) ends | OQ1 "Only where required (Recommended)" | `TestOutputLimit_Omitted_0089`; plant: a caller layer sent on Ollama |
| DL49 | A refused SDK-chosen value, the refusal validated | decided *(2026-11-22: the owner's answer)* | resent once at the validated stated cap, basis refusal, sticky and outside 0062's bound; with `WithoutLimitLearning()` too | OQ3 "By default, can turn off (Recommended)" | `TestLimitLearning_Default_0089`; plant: the resend dropped under the option |
| DL50 | A request whose model's facts are not cached | decided *(2026-11-22: the owner's answer)* | it never waits: it reads the cache and the embedded extract; 0041-MADR D1 stands | OQ2 "Never wait; route's smallest (Recommended)" | `TestOutputLimit_Fallback_0089`; plant: a wait for the listing |
| DL51 | No source states the limit on a Messages route; a refusal stating no number | decided *(2026-11-22: the owner's answers to OQ2 and to DL51)* | the smallest limit any current model on the route is stated to take (64,000 on Claude in the 2026-10-14 listing), basis fallback; a refusal stating the model's cap resent at it; a refusal stating no number returned. Where no source states any limit for the model or the route's current models, the request is refused before sending, `ErrInvalidRequest`, its message saying to state the model's limit (for example models.json `maxTokens`, or a caller's `catalog.WithModelLayer`); no number is invented *(2026-11-22: this refusal was the drafter's; the owner then ruled it, "Refuse, say how to fix (Recommended)")* | OQ2 "Never wait; route's smallest (Recommended)"; DL51 "Refuse, say how to fix (Recommended)" | `TestOutputLimit_Fallback_0089`; plants: the largest current limit chosen; retired models counted; a number sent with no source; the refusal's message without the fix |
| DL52 | A per-request ceiling, and gobble-cli's override | decided *(2026-11-22: the owner's answer)* | no `Request.OutputCeiling`; gobble-cli's override is models.json `maxTokens` only (its 0066 S32 to S34); a ceiling can be added by a later record with nothing to undo | OQ4 "models.json only, for now (Recommended)" | `make api-check`, `tools/apidiff.py`; plant: the field added on a scratch clone, reported |
| DL53 | A listing probe | decided | its 8192 counts as given; basis option | 0088 ST26 | `TestOutputLimit_Reported_0089`, the probe case; plant: basis fact |
| DL54 | `OutputLimit` on a stream | decided | on the final response | D7 | `TestOutputLimit_Reported_0089`, stream case; plant: zero on a stream |
| DL55 | The budget thinking shape (revised) | decided | `AddThinkingFor` starts from the chosen value and may raise it (C54); `OutputLimit.Tokens` is the raised value | D7; 0088 ST2, ST30 | `TestOutputLimit_BudgetReported_0089`; plant: the pre-raise value reported *(2026-11-23: the existing `TestThinkingWire_Claude` budget case, which pinned 13,096 from today's 8192 default, would send the chosen fact; by the owner's rulings it gets the caller's `WithMaxTokens(8192)`, a planned change named by 0088 ST101)* |
| DL56 | 0088's OQ14 (3) estimate (revised) | decided | built after this record, on `Limits.Window`, else `Limits.Input` | the owner's OQ14 answer | `TestOutputLimit_EstimateReadsLimits_0089` (in 0088's Phase 1c); plant: `Limits.Input` ignored when the window is unknown *(2026-11-23, 0088-MADR's twelfth amendment: the estimate counts the request's content as pi does; the test's layer states output 1,000,000, above the 895,904 the hold leaves, and input 1,000,000, no window, on a Claude model in no other source)* |
| DL57 | A provider outside this module (revised) | decided | it may call `CachedLimits` for a chatcompat id and read `SentLimit` | R17 | `TestOutputLimit_OutsideProvider_0089`; plant: the chatcompat id refused |
| DL58 | The exported API (revised) | decided | the PLAN's list in `apidiff`'s form, each with its call site | R55 (C33) | `make api-check`, `tools/apidiff.py`; plant: one more name on a scratch clone |
| DL59 | R57, R6, R7 and the docs (revised) | decided | D9 | D9 | a `tools/claims.py` row per new sentence, planted false |
| DL60 | The changelog | decided | **Added** for each name; **Changed** for the default and the fields read | R53 | `make changelog-check`, a claims row per entry |
| DL61 | OpenRouter's per-endpoint data | out of scope | no provider speaks to OpenRouter directly; Kilo documents no such route; a chatcompat dialect needs a recorded reply | | |
| DL62 | A token-counting endpoint | out of scope | 0088 ST69 | | |
| DL63 | A limit in a response header | out of scope | none found (0088 ST70; round 3) | | |
| DL64 | Probing by an oversized request | out of scope | option B, rejected | | |
| DL65 | Kilo, no limit sent, for a model whose upstream requires one | open: awaits live evidence | until `TestLive_OmittedLimit_0089`'s verdict, Kilo sends the discovered limit (decided by OQ1 "Only where required (Recommended)"); the verdict decides, by a dated row, whether Kilo then omits it | OQ1 "Only where required (Recommended)"; the measurement | `TestOutputLimit_Omitted_0089`'s Kilo case pins the interim (plant: Kilo omitted before the verdict); `TestLive_OmittedLimit_0089` gives the verdict |
| DL66 | Where the one-model and `/api/ps` reads are kept (new) | decided | their own map per listing key and model; 10-minute freshness; a 404 kept as none for the same; a listing refresh does not drop them | D2; C45 | `TestDiscover_ReadsSurviveRefresh_0089`; plants: stored in the listing's models (gone after a refresh); the 404 not kept (a second read) |
| DL67 | What V2 compares (new) | decided | the stated N against the wire's value after `AddThinkingFor` | V2; C54 | `TestLimitRefusal_Validator_0089`, the budget case; plant: N compared with the pre-raise value |
| DL68 | Which wires read which refusal form (new) | decided | Anthropic's: Claude, OpenCode messages, and Kilo (an upstream's text, through V6); OpenAI's "supports at most": the Chat Completions wires (Kilo, Together, Hugging Face, OpenCode chat, chatcompat), not Responses (its text is not established); Gemini's range: OpenCode google (generateContent), not Interactions; the window forms: as 0088 ST51 and C38, C39 *(2026-11-23, confirmed against the code in layer B: `llmprovider/internal/wire/limit_refusal.go` gives Kilo OpenRouter's form (0088 C39, `providers/kilo/error_in_200_test.go:25`) and Kilo's documented form (0088 C97, ST51), and not the split; the split (0088 C38, Together's `providers/together/reply_0028_test.go:23`) is read on the Chat Completions wires other than Kilo; Messages, Responses, Interactions and generateContent read no window form. The row stands; this states its window column by wire.)* | D4; W7 | `TestLimitRefusal_FormsByWire_0089`; plant: OpenAI's form read on Responses |
| DL69 | chatcompat at a gateway fronting Claude (OpenRouter) (new) | decided *(2026-11-22: the owner's answer)* | nothing sent unless the caller gives its own limit; not measured, since a chatcompat base URL is the caller's, not a service this SDK names | OQ1 "Only where required (Recommended)" | `TestOutputLimit_Omitted_0089`, the chatcompat case; plant: a caller layer sent |
| DL70 | The embedded extract (new) | decided | limits only, per section and model; registry below the fetched document; `Stale`, dated; regenerated at each release from a public models.dev document by `tools/limitsnapshot.py`, which keeps only `limit` fields | D3; C50 | `TestDiscover_EmbeddedExtract_0089`; plants: the extract ranked above the fetched document; a non-limit field kept by the tool (its test) |
| DL71 | Staged registry matching (new) | decided | exact, then canonical id, then family, each its own source name; a family match only where nothing exact or canonical states the fact | D3; C48, C49 | `TestDiscover_MatchStages_0089`; plants: a family match chosen over an exact one; a family match named as exact |
| DL72 | `LookupLimits` and the request's cache key (new) | decided | the provider's request reads what `LookupLimits` stored when given the same options and credential | C47 | `TestDiscover_RequestReadsDiscovery_0089` *(2026-11-23: in `llmprovider/providers/claude/limits_discovery_0089_test.go`)* *(2026-11-22: in `llmprovider/providers/claude`, since a `catalog` test cannot build a provider that imports `catalog`; decided by the coordinator from `catalog/evidence_0041_test.go`'s precedent)*; plant: a key without the scope (Claude workspace): the request misses |
| DL73 | The memo and report: by default or on an option (new) | decided *(2026-11-22: the owner's answer)* | on by default: a validated cap kept in the process memo and reported in `WarningLimitLearned`; `WithoutLimitLearning()` turns the memory and the report off, the resend staying | OQ3 "By default, can turn off (Recommended)" | `TestLimitLearning_Default_0089`; plant: the option ignored |
| DL74 | `Limits.Required` on an OpenCode route with no models.dev document (new, 2026-11-22, a missing case) | decided *(2026-11-22: the owner's answer)* | the route is read from OpenCode's route snapshot and table, moved to an internal package `catalog` and `providers/opencode` both read; so a Zen or Go model on the messages route is `Required` with or without the document, and OpenCode's routing is unchanged | "Share the route table (Recommended)" | `TestDiscover_RequiredWithoutDocument_0089`; plant: `Required` read from the document only (the Zen Claude model under `WithoutModelMetadata` reports false) |
| DL76 | `Limits` and the memo's streamed and plain caps (new, 2026-11-23) | decided *(2026-11-23: the owner's ruling)* | `Limits.StreamedOutput` beside `Output`: each the output fact held to the window and then to the cap the memo keeps for its own call, streamed or plain, the cap's evidence named "observed"; with no cap kept for either, the two are equal; a request reads the one for its call | "Add a stream dimension" | `TestLimitMemo_StreamedOutput_0089` (layer B); plant: the streamed cap read into `Output` (stream and plain merged in `Limits`) |
| DL77 | OpenCode's messages-route fallback (new, 2026-11-23) | decided *(2026-11-23: the owner's ruling)* | on an OpenCode gateway's messages route, where no source states the model's output limit, the fallback (DL51) is the smallest output limit among the models the shared route table (`llmprovider/internal/opencoderoute`) routes to messages on that gateway, each read from the fetched document where it is fetched, else the embedded extract; a model the document marks deprecated or retired is not counted; at the 2026-10-12 extract, 64,000 on Zen and 131,072 on Go | "Models on opencode's route table (Recommended)" | `TestOutputLimit_Fallback_0089`, its OpenCode case; plant: the extract's whole section read (Zen's `claude-3-5-haiku`, 8,192, chosen) |
| DL75 | The Claude workspace `LookupLimits` reads (new, 2026-11-23) | decided *(2026-11-23: by the coordinator, from precedent)* | the catalog reads only `catalog.WithClaudeWorkspace` (`llmprovider/catalog/config.go:170-178`), and the Claude provider converts its own `WithWorkspace` into that option for its listing (`llmprovider/providers/claude/claude.go:270`); `LookupLimits` follows the same rule, and its doc comment tells a caller to pass `catalog.WithClaudeWorkspace` | the precedent cited | `TestDiscover_RequestReadsDiscovery_0089`; plant: the stored key without the scope (the request misses) |

Counts: 67 decided, 5 out of scope (DL14, DL61 to DL64), 0 open, 1 awaiting
live evidence (DL65); 73 rows. *(2026-11-22: DL74 added, decided by the owner's
ruling "Share the route table (Recommended)": 68 decided, 74 rows.
2026-11-23: DL75 added, decided by the coordinator from precedent: 69
decided, 75 rows. 2026-11-23: DL76 added, decided by the owner's ruling "Add a
stream dimension": 70 decided, 76 rows. 2026-11-23: DL77 added, decided by the owner's ruling "Models on opencode's route table (Recommended)": 71 decided, 77 rows.)* *(Before the owner's answers: 58 decided, 5
out of scope, 10 open. First draft: 52 decided, 5 out of scope, 8 open, 65
rows.)*

## Owner Questions

*(2026-11-22: all four answered; see "Owner rulings".)* Four questions,
three options each, the recommended first. "Optional" means
the wire lets a request send no limit; Messages (Claude, OpenCode's messages
route) requires one.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ1. When a request names no limit, should the SDK send one at all? | (1) **Only where the wire requires one**: on Messages the discovered limit, held to the window; on every optional wire nothing, so the service's own default applies. Kilo keeps sending the discovered limit until one live request per model shows its own default is not lower. A caller's own value is still sent. (2) **Wherever a fact is known**: the best evidence, held to the window, no fixed ceiling; on Grok, OpenAI with a key, Ollama and chatcompat only a caller's layer counts (a "models.dev" layer counts as models.dev). (3) **Keep the 2026-11-21 rule**, with discovery informing it: at most 32,000 on a stream, 8192 when unknown, half the window. | (1), per wire: Grok: 128,000 visible tokens (documented). Gemini: not documented for its Interactions API. OpenAI with a key: not documented (its own client sends none); no rate-limit charge for an unused limit. Ollama: unbounded; loses today's 8192 runaway guard, and a caller's layer no longer raises there or on chatcompat (reverses 0088 OQ7, ST9). Together, Hugging Face, OpenCode's chat, responses and google routes, chatcompat: not documented. chatcompat at OpenRouter fronting Claude: OpenRouter's fill not documented; its credit hold on an unpublished cap (relayed, not re-read) can refuse a low balance with 402; a caller there passes its own limit. Kilo: the measurement bills up to 64,000 plus 65,536 output tokens. A long plain `Generate` is already streamed (0088 OQ15). Replaces 0088 OQ2's ceilings and OQ4's 8192. (2) Kilo's stated limits, up to 1,800,000, are sent as stated, held to the window; OpenAI charges a layer-stated limit against the key's rate limit; Ollama and optional wires with no fact lose 8192; replaces the same 2026-11-21 answers. (3) numbers with no source stay; Claude 4.6 and later held to 32,000; 8192 for a model no source knows. | DL46 to DL48, DL65, DL69; 0088 ST74, ST85, ST86, ST90, ST92, ST98 |
| OQ2. A request reads only what is cached, and never waits (0041-MADR D1). What happens on a cold request, and on Messages when no source at all knows the model? | (1) **Never wait; on Messages use the route's smallest current limit**: a cold request reads the cache and the embedded extract; on Messages with nothing known, the smallest limit any current model on that route is stated to take (64,000 on Claude in the 2026-10-14 listing), labelled fallback, as llm-anthropic does; a refusal stating the model's cap is then resent at it. (2) **Wait once per model for discovery** (at most 10 s; a failure not retried for 1 minute, as the models.dev document's is), then as (1). (3) **Never wait; on Messages send 8192, labelled fallback, with a warning.** | (1) a Claude model newer than the extract and not yet cached gets less than its own limit until its listing is cached; an older model with a smaller cap is refused once, then resent; 0041 D1 stands. (2) amends 0041 D1; the first request per model per process can wait up to 10 s; on a failing listing, a cold request waits up to 10 s once a minute. (3) a number with no source, shown in `OutputLimit` and a warning; the reported failure can recur on such a model at high effort; one more exported warning kind. | DL50, DL51 |
| OQ3. When a service refuses a limit the SDK chose and states its cap, the SDK resends once at that cap. Should it also remember the cap and report it, by default or only when asked? | (1) **By default**, with `WithoutLimitLearning()` to turn the memory and report off (the resend stays). (2) **Only when asked**, with `WithLimitLearning()`: without it, the resend at the stated cap still happens on each such request, and nothing is remembered or reported; gobble-cli asks for it. (3) **Only when asked, resend included**: without the option, the refusal is returned. | (1) later requests in a process may carry a smaller limit than the first did (reported); one option. (2) a request on a model whose facts overstate its cap pays one refused send each time, unless its caller opts in; one option; matches the note's "opt-in for determinism". (3) reverses 0088 OQ3 and OQ6's resend (owner answers of 2026-11-21): a turn that would pass at the cap fails; one option. | DL49, DL73; 0088 ST89 |
| OQ4. Asked with gobble-cli's 0066 (its override). How may a user cap gobble's output limit, and does the SDK add a ceiling for it? | (1) **models.json `maxTokens` only; no SDK ceiling yet.** (2) **models.json and a gobble setting `outputLimit`, sent as a new SDK ceiling** (`Request.OutputCeiling`), which the SDK still fits and resends below, honoured down to 1,024 as 0088 OQ12 answered. (3) **(2) and an ACP option per session for the setting.** | (1) no new surface; one edit per model; under OQ1 (1) a stated `maxTokens` drives the Messages routes only, the service's default applying elsewhere; OQ12's ceiling not built. (2) one SDK field and one gobble setting; a number the user picks caps every model; under OQ1 (1) the ceiling is sent as the limit on an optional wire. (3) as (2), and an option every ACP client shows. | DL52; gobble-cli 0066 S32 to S34 |

## Pros and Cons of the Options

### A. A discovery API over the catalog

* Good, because it uses every source a service offers and keeps each one's
  value and time, so conflicts and staleness are visible.
* Good, because unknown is a state, and a refusal is used only through a
  validator, as a bound, kept for one process.
* Bad, because it adds names and a release step (the extract).

### B. Probing

* Good, because a refusal states the cap on Anthropic.
* Bad, because every probe is a billed or rate-counted request; no library
  does it (research round 3); OpenAI, xAI and the gateways state no cap in a
  known form.

### C. Learning from refusals alone

* Good, because the service's own word is the best evidence of what it takes.
* Bad, because every model pays a refused send first, only three services
  state a cap, and W3 shows a wrong value persisting without a validator. It
  is part of A (D4, D5).

### D. A bundled registry as the only source

* Good, because it works offline.
* Bad, because it is stale at release and wrong in known ways (W12; the
  registry's outputs above their window, 0088 report). A keeps one, as the
  lowest registry tier only (D3).

### E. Leave it to callers

* Good, because no new API.
* Bad, because every caller rebuilds the same sources and the gap the owner
  named stays.

### F. Escalation after a cut answer

* Good, because a cut answer proves the limit was too small.
* Bad, because the cut answer is billed first, a stream has already shown it,
  and the start value is still a guess.

## More Information

### Relation to other records

* 0088-MADR-default-output-limit-from-model-facts.md: D10; its fourth
  amendment; its OQ18 decides the release.
* 0041-MADR-decide-per-model-behaviour-from-metadata.md: D1 (no request waits)
  stands unless OQ2 is answered (2).
* 0033-MADR's caller-layer rank stands for `ModelInfo`; `Limits` ranks a
  models.dev document as registry (DL15).
* 0062-MADR-learn-thinking-parameters-from-refusals.md: the memo, warning and
  pass-back pattern (D5), with scope, transport and an expiry added to its key
  and its pass-back (C35, C36).
* gobble-cli `docs/decisions/0066-MADR-set-each-session-model-up-from-discovered-limits.md`:
  the call sites (R55) and the session setup.

### What could not be established

* What Kilo, OpenRouter, Together, Hugging Face, OpenAI and Gemini's
  Interactions API do when no output limit is sent (none documents it;
  OQ1 (1)'s measurement covers Kilo).
* Whether Anthropic's cap refusal names the dated id when the request named
  an alias (likely from forum copies; V6 accepts either).
* Whether every current Claude model's `max_tokens` is set in the live Models
  API (14 of 14 in the 2026-10-14 fixture).
* The current text of OpenAI's and Gemini's cap refusals on Responses and
  Interactions.
* Whether Kilo's `per_request_limits` is ever set; whether Ollama counts
  thinking in `num_predict`; whether a refused request is billed.

### Numbers this record keeps

| Number | Where | Provenance |
| :--- | :--- | :--- |
| 10 s | discovery's bound | the listing's existing bound (C6) |
| 10 minutes | the one-model and `/api/ps` reads' freshness | the listing's (C47) |
| 24 h | a passed-back cap's expiry | the SDK's window for a service's stale statement (C19) |
| 1 minute | OQ2 (2)'s failure interval | the models.dev document's retry (C55) |
| 256 | the memo | 0062-MADR's memory bound (C35) |
| 16 | V4 on Responses | OpenAI's spec minimum (W8) *(2026-11-26: not applied; 1 until a record re-adds it, 0088-MADR ST113)* |
| 1,024 | OQ4 (2)'s floor | 0088 OQ12's answer |
| 64,000 | OQ2 (1)'s fallback on Claude today | the smallest current `max_tokens` in Claude's listing; a rule, not a constant |
| 4,096 and 3.5 characters a token | 0088 OQ14 (3)'s margin and estimate | pi's, unmeasured (0088's numbers table) |

### Evidence index

<!-- claims: self=751e970 gobble-cli=dfaa1ac -->

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C1 | the three limit fields | `llmprovider/catalog/model_info.go:30` | `ContextWindow, MaxInputTokens, MaxOutputTokens int` |
| C2 | a fact names its source | `llmprovider/catalog/model_info.go:233` | `// SourceOf names the layer that set f` |
| C3 | caller layers rank above the listing | `llmprovider/catalog/lookup.go:29` | `every caller layer ranks` |
| C5 | a request's read starts no fetch | `llmprovider/catalog/lookup.go:160` | `It starts no` |
| C6 | the bound | `llmprovider/catalog/discovery.go:119` | `const modelListingTimeout = 10 * time.Second` |
| C7 | a models.dev document's source name | `llmprovider/catalog/metadata_document.go:70` | `func (m Metadata) Name() string { return sourceModelsDev }` |
| C8 | a layer's facts take its name | `llmprovider/catalog/lookup.go:220` | `overlay(&info, v, l.Name())` |
| C10 | Claude's request reads cached facts | `llmprovider/providers/claude/claude.go:430` | `catalog.CachedFacts(` |
| C11 | so does Kilo's | `llmprovider/providers/kilo/kilo.go:361` | `catalog.CachedFacts(` |
| C12 | one fetch per listing at a time | `llmprovider/catalog/listing_cache.go:201` | `func (c *store) claimListing(key listingKey, fetch func(call *listingCall)) (*listingCall, error) {` |
| C13 | a closed cache is refused | `llmprovider/catalog/lookup.go:93` | `if err := cfg.store.refuseClosed(); err != nil {` |
| C14 | the default | `llmprovider/settings.go:88` | `return settings{cfg: providerConfig{MaxTokens: 8192}}` |
| C15 | Claude's output limit *(2026-11-23: moved at `6ace750`; see C15now)* | `llmprovider/catalog/listing_metadata.go:156` | `b.info.MaxOutputTokens = *m.MaxTokens` |
| C16 | Claude's input limit *(2026-11-23: moved at `6ace750`; see C16now)* | `llmprovider/catalog/listing_metadata.go:152` | `b.info.MaxInputTokens = *m.MaxInputTokens` |
| C17 | Gemini's output limit *(2026-11-23: moved at `6ace750`; see C17now)* | `llmprovider/catalog/listing_metadata.go:135` | `b.info.MaxOutputTokens = *m.OutputTokenLimit` |
| C18 | Gemini's input limit *(2026-11-23: moved at `6ace750`; see C18now)* | `llmprovider/catalog/listing_metadata.go:131` | `b.info.MaxInputTokens = *m.InputTokenLimit` |
| C19 | the last good listing's age | `llmprovider/catalog/listing_cache.go:56` | `const lastGoodWindow = 24 * time.Hour` |
| C20 | Together's listing struct | `llmprovider/catalog/listing_metadata.go:75` | `type togetherModel struct {` |
| C20b | Kilo's output limit | `llmprovider/catalog/listing_metadata.go:298` | `b.info.MaxOutputTokens = *e.TopProvider.MaxCompletionTokens` |
| C21 | Together's fixture states an output limit | `llmprovider/catalog/testdata/ranking-2026-10-12/together.json:1` | `"max_output_length":40000` |
| C22 | the ChatGPT backend states a window | `llmprovider/catalog/testdata/rawlistings-2026-10-15/chatgpt.json:1` | `"context_window":272000` |
| C23 | Hugging Face states a window per provider | `llmprovider/catalog/testdata/ranking-2026-10-12/hf.json:1` | `"provider":"novita","status":"live","context_length":1000000` |
| C24 | Claude's listing names the 4.5 models by dated ids | `llmprovider/catalog/testdata/firstparty-2026-10-14/claude.json:1` | `"id":"claude-sonnet-4-5-20250929"` |
| C25 | Kilo's per-request limits, null | `llmprovider/catalog/testdata/ranking-2026-10-12/kilo.json:1` | `"per_request_limits":null` |
| C26 | the one recorded dialect | `llmprovider/internal/compatdialect/compatdialect.go:49` | `Name: "ollama", Label: "Ollama",` |
| C27 | the split is parsed | `llmprovider/context_overflow.go:64` | `contextSplitPattern = regexp.MustCompile(` |
| C28 | and only a reason kept | `llmprovider/context_overflow.go:69` | `max_tokens alone fills the context window: lower it` |
| C29 | no request waits | `docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md:323` | `No request waits, and no request lists.` |
| C31 | the SDK's only copy of Anthropic's cap form, a test's invented string | `llmprovider/providers/claude/forced_learn_0034_test.go:100` | `max_tokens: 999999 > 128000` |
| C33 | R55 *(2026-11-23: moved at `6ace750`; see C33now)* | `docs/guides/api-standards.md:370` | `R55. A new export is justified.` |
| C35 | 0062's memo bound | `llmprovider/internal/wire/learn.go:14` | `const learnedLimit = 256` |
| C36 | 0062's memo key, without a scope | `llmprovider/internal/wire/learn.go:54` | `func LearnedKey(provider, baseURL, model string) string {` |
| C37 | the pass-back option | `llmprovider/refusal.go:47` | `func WithReasoningRefusals(refusals ...ReasoningRefusal) Option {` |
| C38 | the learned warning | `llmprovider/warning.go:34` | `WarningReasoningLearned WarningKind = "reasoning_learned"` |
| C39 | the response type | `llmprovider/item.go:114` | `type Response struct {` |
| C40 | gobble's provider has no document | `gobble-cli:llm/provider/provider.go:215` | `llmprovider.WithoutModelMetadata(),` |
| C41 | gobble's lookup | `gobble-cli:llm/provider/model.go:169` | `info, _ := catalog.CachedModel(ctx, llmprovider.ProviderID(id), src, model, opts...)` |
| C42 | gobble's models.json layer name | `gobble-cli:llm/provider/model.go:177` | `func (modelsJSONLayer) Name() string { return "models.json" }` |
| C44 | the document is fetched from a URL | `llmprovider/catalog/model_metadata.go:28` | `models.opencode.ai/api.json` |
| C45 | a listing refresh replaces its models | `llmprovider/catalog/listing_cache.go:75` | `if !merge` |
| C46 | stored models are returned at any age | `llmprovider/catalog/listing_cache.go:86` | `cachedListing returns a listing's metadata, and whether it is fresh. Stale` |
| C47 | the listing's freshness | `llmprovider/catalog/listing_cache.go:20` | `const listingMetadataTTL = 10 * time.Minute` |
| C48 | the catalog reads models.dev's canonical id | `llmprovider/catalog/model_info.go:21` | `CanonicalID is models.dev's canonical_model_id` |
| C49 | the canonical-id source name | `llmprovider/catalog/canonical.go:20` | `const sourceCanonical = "models.dev (canonical id)"` |
| C50 | an embedded snapshot precedent *(2026-11-23: moved at `6ace750`; see C50now)* | `llmprovider/providers/opencode/route.go:71` | `//go:embed routes_snapshot.json` |
| C51 | R6's output-limit sentence *(2026-11-23: moved at `6ace750`; see C51now)* | `docs/guides/api-standards.md:103` | `the output limit, which cuts the default and never raises it.` |
| C52 | the plan omits the limit, though accepted | `llmprovider/providers/openai/openai.go:498` | `max_output_tokens, which it says to omit though M5 measured it` |
| C53 | Gemini's provider speaks Interactions | `llmprovider/providers/gemini/gemini.go:2` | `Interactions API with an API key` |
| C54 | the budget raises `max_tokens` | `llmprovider/internal/wire/messages/messages.go:646` | `if maxTokens <= budget {` |
| C55 | the document's failure interval | `llmprovider/catalog/model_metadata.go:34` | `modelMetadataRetryAfter = time.Minute` |

### Re-grounding at `6ace750` (2026-11-23)

`python3 -B tools/claims.py docs/decisions/0089-MADR-model-output-limit-discovery.md --rev self=6ace750`, at the merged phase's base (0089-PLAN deviation 6): 42 of 49 pass, exit 1. The seven rows that moved, none gone, each at its line at `6ace750`; C50's text moved to another file with OpenCode's route snapshot (0089-PLAN deviation 4). The ids end in "now" so that `tools/claims.py` does not read them against this record's `self=751e970`; with plain ids and `self=6ace750`, on a scratch copy, 7 of 7 pass, exit 0.

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C15now | Claude's output limit | line | `llmprovider/catalog/listing_metadata.go:186` | `b.info.MaxOutputTokens = *m.MaxTokens` |
| C16now | Claude's input limit | line | `llmprovider/catalog/listing_metadata.go:182` | `b.info.MaxInputTokens = *m.MaxInputTokens` |
| C17now | Gemini's output limit | line | `llmprovider/catalog/listing_metadata.go:165` | `b.info.MaxOutputTokens = *m.OutputTokenLimit` |
| C18now | Gemini's input limit | line | `llmprovider/catalog/listing_metadata.go:161` | `b.info.MaxInputTokens = *m.InputTokenLimit` |
| C33now | R55 | line | `docs/guides/api-standards.md:371` | `R55. A new export is justified.` |
| C50now | an embedded snapshot precedent | file | `llmprovider/internal/opencoderoute/opencoderoute.go:30` | `//go:embed routes_snapshot.json` |
| C51now | R6's output-limit sentence | line | `docs/guides/api-standards.md:104` | `the output limit, which cuts the default and never raises it.` |

### Web and source evidence

| Id | Source | Read | Status |
| :--- | :--- | :--- | :--- |
| W1 | Anthropic Python SDK, `src/anthropic/types/model_info.py` (main): `max_tokens` "Maximum value for the `max_tokens` parameter when using this model."; `max_input_tokens` "Maximum input context window size in tokens for this model." | 2026-11-22 | verified by the drafting step (fetched) |
| W2 | Anthropic Python SDK, `src/anthropic/resources/models.py`: `retrieve(model_id)`, "Model identifier or alias", path `/v1/models/{model_id}` | 2026-11-22 | verbatim copy read (round 3) |
| W3 | hermes-agent issue #150519, open, 2026-11-07 | 2026-11-22 | verified by the drafting step (fetched); names #133606 (load-balanced aliases) |
| W4 | OpenRouter `openapi.json`: `PublicEndpoint.max_completion_tokens` and `TopProviderInfo.max_completion_tokens`, nullable | 2026-11-22 | verbatim copy read by the drafting step |
| W5 | Gemini thinking page (Interactions): the output limit includes thought tokens | 2026-11-22 | relayed (round 3) |
| W6 | Ollama `api/types.go` (main): `ProcessModelResponse.ContextLength` (`/api/ps`); `/api/show`'s `<arch>.context_length`, the trained maximum | 2026-11-22 | verbatim copy read by the drafting step |
| W7 | refusal texts stating a cap: Anthropic (forum copy), OpenAI ("supports at most N completion tokens", forum, a Chat Completions model), Gemini ("supported range is from 1 (inclusive) to 65537 (exclusive)", forum, generateContent) | 2026-11-22 | secondary; the SDK's C31 is a test string, not a recorded reply |
| W8 | OpenAI OpenAPI spec: Responses `max_output_tokens` `minimum: 16` *(2026-11-26: the SDK does not apply it; `ResponsesLimits.Minimum` is 1 until a record makes a Responses route send a limit the SDK chose and re-adds this floor, 0088-MADR ST113)* | 2026-11-22 | 0088 report WS11 |
| W9 | llm-anthropic `llm_anthropic.py` at 30d4d939c4: `UNKNOWN_MODEL_MAX_TOKENS` 64000, "the lowest output limit of any current model, so requests are never rejected" | 2026-11-22 | verbatim copy read (round 3) |
| W10 | Vercel AI SDK at 64a2866982: `knownModel: false` and a warning when the fallback is used | 2026-11-22 | verbatim copy read (round 3) |
| W11 | Pydantic AI at 463a6d519b: `anthropic_max_output_tokens: int \| None`, "Default: `None` (unknown)", and 16,384 sent when it is unknown | 2026-11-22 | verbatim copy read (round 3) |
| W12 | litellm issue #14876 (closed 2025-09-26): Sonnet 4's `max_output_tokens` set to 1,000,000 "when it should remain 64,000" | 2026-11-22 | relayed from the issue's text (round 3) |
| W13 | local servers for a later dialect record: llama.cpp `meta.n_ctx` and `/props`; vLLM `max_model_len`; LM Studio `loaded_instances[].config.context_length`; TGI `/info` `max_total_tokens` | 2026-11-22 | verbatim copies read (round 3); no reply recorded |

## Amendment (2026-11-24): the joint gates' claims rows, and v1.33.0

Additions only; no row's decision changes. The owner's ruling of 2026-11-24, relayed by the coordinator, quoted as given: "Yes, re-scope v1.33.0 (Recommended)": v1.33.0 is cut from what `33e6f09` holds, with the caller-limit refusal report the owner scoped in on 2026-11-24 (its own amendment of this record, to follow), after the joint gates and a short release review whose findings are blockers only (PLAN entry of 2026-11-24). DL65 stays open, and its interim ships (Kilo sends its discovered limit); DL70's regeneration of the extract before the tag is put to the owner (the PLAN entry). *(2026-11-24: both answered; see the annotations below.)*

### Annotations (2026-11-24)

Annotations of the State Table's rows, made here rather than in the rows' own lines; no row's decision changes.

* **DL70 (annotated 2026-11-24).** The owner's ruling, relayed by the coordinator, quoted as given: "One fetch, then ship (Recommended)". Before v1.33.0's tag, an agent fetches models.dev's public `api.json` once, with no key, and regenerates the extract, which the goldens and the gates check before the tag (PLAN entry of 2026-11-24).
* **DL65 (annotated 2026-11-24).** The owner's ruling, relayed by the coordinator, quoted as given: "Run before the tag". An agent makes a few short live calls on Kilo before v1.33.0's tag, environment keys only, allowlisted output; the verdict becomes a dated row here. Until then the interim stands.

### Evidence added (2026-11-24)

DL59's rows, PLAN Phase 4's check: one per line of R6's output-limit sentences, R7's `OutputLimit` sentence, R57 and its Checks row in `docs/guides/api-standards.md`, and one per package doc's output-limit line (the first line of its 0089 paragraph; `catalog`'s limits paragraph). DL60's rows, one per changelog entry, are 0088-MADR's C149now to C160now (its fourteenth amendment). The ids end in "now", so that `tools/claims.py` does not read them against this record's `self=751e970`: each was checked with a plain id against a scratch clone's commit of the tree, 47 of 47, and planted false there, 47 of 47 caught.

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C56now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:116` | `` The output limit follows R57. `MaxOutputTokens` is sent as given; a `` |
| C57now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:117` | `` `WithMaxTokens` value is cut to the model's stated output limit and never `` |
| C58now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:118` | `raised; the ChatGPT backend and plan routes send none, whatever is given` |
| C59now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:119` | `(0088 ST4). With neither, a Messages route (Claude, OpenCode's messages` |
| C60now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:120` | `route), whose wire requires a limit, sends the model's output limit as` |
| C61now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:121` | `` `catalog.CachedLimits` states it: the caller's layers, then the service, `` |
| C62now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:122` | `then the registry (the fetched models.dev document, then the SDK's` |
| C63now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:123` | `` embedded extract), then a cap passed back with `WithLimitObservations`, `` |
| C64now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:124` | `held to the stated window and to a cap the service stated; where none is` |
| C65now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:125` | `stated, the smallest limit of the route's current models, and where none` |
| C66now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:126` | `` of those is either, `ErrInvalidRequest` before sending. Kilo sends its `` |
| C67now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:127` | `stated output limit (0089 DL65, measured 2026-11-24: with none sent, Kilo cut one model at 128,000 of 131,072 stated); every other` |
| C68now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:128` | `wire sends none, so the service's own default applies. A limit the SDK` |
| C69now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:129` | `chose, the stated limit or the route's smallest, is held to the room the` |
| C70now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:130` | `stated window (else the stated input limit) leaves the request's input,` |
| C71now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:131` | `estimated from its content as pi does, never below the wire's minimum,` |
| C72now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:132` | `before a thinking budget raises it; a caller's value is not held. A` |
| C73now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:133` | `` `Generate` that names no limit is read over the service's stream and `` |
| C74now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:134` | `returned whole. (0089 D1 to D7, DL46 to DL51, DL56; 0088 ST15 to ST18,` |
| C75now | R6's output-limit sentences (0089 DL59) | new | `docs/guides/api-standards.md:135` | `ST87, ST88, ST105, ST106)` |
| C76now | R7's `OutputLimit` (0089 DL59) | new | `docs/guides/api-standards.md:139` | `` (0034 D7), and `OutputLimit`, a `SentLimit`: the output limit the request `` |
| C77now | R7's `OutputLimit` (0089 DL59) | new | `docs/guides/api-standards.md:140` | `carried, after a thinking budget raised it, whether one was sent, its` |
| C78now | R7's `OutputLimit` (0089 DL59) | new | `docs/guides/api-standards.md:141` | `` `LimitBasis` and its source; a stream's final response carries it too `` |
| C79now | R7's `OutputLimit` (0089 DL59) | new | `docs/guides/api-standards.md:142` | `(0089 D7, DL54, DL55).` |
| C80now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:427` | `**R57. No fixed number decides an output limit a caller did not give:**` |
| C81now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:428` | `` the model's evidence does (`catalog.Limits`), and a limit is sent only as `` |
| C82now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:429` | `R6's default rule decides. A number with no source is listed with its` |
| C83now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:430` | `provenance in the record that keeps it: a listing probe's 8192, which is` |
| C84now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:431` | `sent as given (0088 ST26; 0089 DL53); the estimate's 3.5 characters a` |
| C85now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:432` | `token, 4,800 characters an image and 4,096 spare, pi's and unmeasured` |
| C86now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:433` | `(0088 ST87). A refusal that states a cap or a` |
| C87now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:434` | `window resends a limit the SDK chose once at the stated value; a stated` |
| C88now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:435` | `` cap is kept for the process and reported in a `WarningLimitLearned` `` |
| C89now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:436` | `` unless `WithoutLimitLearning` is given, and a caller's own limit is never `` |
| C90now | R57 (0089 DL59) | new | `docs/guides/api-standards.md:437` | `resent. (0089 D4 to D6, D9; 0088 ST99)` |
| C91now | R57's Checks row (0089 DL59) | new | `docs/guides/api-standards.md:456` | `` R57 \| `git grep -n -w` of 32000, 16384 and 8192 in the module's non-test Go code `` |
| C92now | `catalog`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/catalog/doc.go:44` | `A model's limits, with every source's statement of each, are Limits:` |
| C93now | `chatcompat`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/providers/chatcompat/chatcompat.go:92` | `optional on this wire (0089-MADR D6, DL47, DL48): a request's own` |
| C94now | `claude`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/providers/claude/claude.go:116` | `The output limit (0089-MADR D6, D7): a request's own MaxOutputTokens is` |
| C95now | `gemini`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/providers/gemini/gemini.go:78` | `The output limit is optional on this wire (0089-MADR D6, DL47, DL48): a` |
| C96now | `grok`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/providers/grok/grok.go:93` | `The output limit is optional on this wire (0089-MADR D6, DL47, DL48): a` |
| C97now | `huggingface`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/providers/huggingface/huggingface.go:80` | `The output limit is optional on this wire (0089-MADR D6, DL47, DL48): a` |
| C98now | `kilo`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/providers/kilo/kilo.go:88` | `The output limit (0089-MADR D6, D7, DL65): a request's own` |
| C99now | `ollama`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/providers/ollama/ollama.go:86` | `The output limit is optional on this wire (0089-MADR D6, DL47, DL48): a` |
| C100now | `openai`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/providers/openai/openai.go:109` | `The output limit is optional on this wire (0089-MADR D6, DL47, DL48): a` |
| C101now | `opencode`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/providers/opencode/opencode.go:116` | `- The output limit (0089-MADR D6, D7): a request's own MaxOutputTokens` |
| C102now | `together`'s package doc, its output-limit line (0089 DL59) | new | `llmprovider/providers/together/together.go:77` | `The output limit is optional on this wire (0089-MADR D6, DL47, DL48): a` |

## Amendment (2026-11-24, second): a learned cap reaches a caller whose call failed

Additions only, except DL37, which is annotated, not rewritten. The owner's word of 2026-11-24, relayed by the coordinator, quoted as given: "scope this into v1.33.0: It also found something for the SDK release. When a service refuses a limit the caller set, the SDK learns the cap but never reports it, since that warning only travels on a successful reply." The owner then answered OQ5 below (2026-11-24, relayed by the coordinator), quoted as given: "On the error (Recommended)". So `APIError` gains `Observed LimitObservation`, and DL78 to DL94 are decided for that answer.

### What `33e6f09` does (verified on a scratch clone, 2026-11-24)

`wire.LimitRetry.Check` keeps a validated, kept cap for every basis (C161now) and builds the `WarningLimitLearned` there (C162now), before it decides that a caller's value is not resent (C163now); the warning reaches the caller only through `Finish`, which a provider calls on success (C164now, C166now), and `FinishStream`, which reads only `EventDone` (C165now). DL37 recorded the consequence as a decision ("no warning, since an error carries no response"); the owner's word reverses it. A scratch test on a fake server (Claude, `Request.MaxOutputTokens` 200,000, refused "max_tokens: 200000 > 128000, which is the maximum allowed number of output tokens for claude-caller-0089"):

* **What the caller sees.** One send; `*llmprovider.APIError`, `Status` 400, kind `ErrInvalidRequest` (not `ErrContextOverflow`), `Code` "invalid_request_error", `Limit` nil, text `llmprovider: invalid request: claude HTTP 400 invalid_request_error: max_tokens: 200000 > 128000, which is the maximum allowed number of output tokens for claude-caller-0089`; no warning, no field holds 128,000. The same on `Stream` (the error as its one event), under `WithRetry` and under `WithStreamRetry` (one send each: the kind is not retried).
* **The memo holds it.** 128,000 under the call's own key: plain for `Generate`, streamed for `Stream`; `catalog.CachedLimits` shows it as `Output` (or `StreamedOutput`) with the one evidence "observed" (C172now). Under `WithoutLimitLearning`, nothing.
* **A later call.** With no limit, a `Generate` (read over the stream, 0088 ST88) after a refused plain `Generate` sent the route's fallback, 64,000, basis fallback (the cap is under the plain key); after a refused `Stream` it sent 128,000, basis fact, source "observed". The same caller value again: sent as given and refused again (0088 ST17).
* **A first payload inside a 200 (0088 A5)**, on Kilo's `Stream`: the error is a `*transport.afterReply` wrapping an `*APIError` (`Status` 0, `Code` "400"), and the streamed key holds 128,000.
* **The SDK's own value, its resend failed.** Claude's 128,000 refused at 64,000, the resend answered 503: the 503 `*APIError` is returned, the memo holds 64,000, nothing reports it. Under `WithRetry`, refused, then 503, then answered: three sends (128,000, 64,000, 64,000), `OutputLimit` basis fact, source "observed", and no `WarningLimitLearned` (C169now).
* Two plants on the scratch test were caught: the cap not kept ("memo holds 0; want 128000"); a caller's value resent, which put the warning on the response ("a WarningLimitLearned reached the caller").

### Owner question, answered

| Question | Options (recommended first) | Cost of each | Rows | The owner's answer |
| :--- | :--- | :--- | :--- | :--- |
| OQ5. When a call fails after the service refused its output limit and stated a cap, how does the caller get that cap? | (1) **On the error (Recommended)**: `APIError` gains `Observed LimitObservation`, the field `Warning.Observed` already is, set by the provider after the send, as `APIError.Tool` is (0084 D2, C168now). (2) **A callback option** `WithLimitObserved(func(LimitObservation))`, called at every kept cap. (3) **No new name**: document that after an `ErrInvalidRequest` the caller reads `catalog.CachedLimits` for the evidence "observed" and builds the observation itself. | (1) one `apidiff` line; R24's field list grows by one (C170now); a failure that is not an `*APIError` after the SDK's own resend carries none (DL87). (2) one line; breaks R28 (C171now); a callback on the request's goroutine. (3) no line; the caller repeats the provider's options, cannot tell its call's refusal from another's, and the SDK's failed resend stays unreported. | DL78 to DL94; DL37 | "On the error (Recommended)" (2026-11-24): option (1) |

### Rows

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| DL78 | A request's own `MaxOutputTokens` refused in a cap form, validated, kept (V6), learning on; `Generate` | decided *(2026-11-24: the owner's answer)* | sent once, the refusal returned (0088 ST17, ST18), kept under the plain key, and the returned `*APIError`'s `Observed` is the kept cap: `Model` as the request named it, `BaseURL` as `WithBaseURL` gave it (empty for the default), `Scope` (Kilo organization, Claude workspace), `Tokens`, `Seen` | the owner's word; OQ5 "On the error (Recommended)"; D5 | `TestLimitRefusal_CallerReport_0089`, case generate (Claude); plant: Claude's `Generate` does not apply the report to its error |
| DL79 | As DL78, on `Stream`, refused before any event (a 4xx at opening) | decided *(2026-11-24: the owner's answer)* | the one error event's `*APIError` carries `Observed`; the streamed key holds the cap | OQ5 "On the error (Recommended)"; 0088 A5 | the same test, case stream; plant: `FinishStream` passes an error event unchanged |
| DL80 | As DL79, the refusal the stream's first payload inside a 200 (0088 A5) | decided *(2026-11-24: the owner's answer)* | as DL79: the `*APIError` is found through the transport wrapper (`errors.As`), whose type and text stay | OQ5 "On the error (Recommended)"; 0088 A5, ST55 | the same test, Kilo's case; plant: the report set only when the error itself is an `*APIError` (a type assertion) |
| DL81 | A `WithMaxTokens` value refused (basis option) | decided *(2026-11-24: the owner's answer)* | as DL78 and DL79 | OQ5; DL37's annotation of 2026-11-23 | the same test, option case; plant: the report gated on basis request |
| DL82 | The same caller value refused again in the process (the memo unchanged) | decided *(2026-11-24: the owner's answer)* | every refused call carries `Observed`; a `WarningLimitLearned` on a response stays once per key (D5) | OQ5; an error is the call's only result | the same test, second call; plant: the report gated on the memo's change |
| DL83 | `WithoutLimitLearning` | decided | nothing kept, `Observed` zero, the refusal returned as at `33e6f09` | DL73 | the same test, its case; plant: `Observed` set without the memo's keep |
| DL84 | A refusal not kept: its named model fails V6; a window form (DL35); a refusal V1 to V5 reject; a wire that reads no cap form (Responses, Interactions, DL68) | decided | `Observed` zero | D4, D5; DL35, DL36, DL68 | the same test, V6 and window cases; plant: `Observed` set from `reading.Cap` without `reading.Keep` |
| DL85 | The SDK's own value refused, resent at the cap, answered | decided | unchanged: the response's `WarningLimitLearned`, once per key | DL49, DL73 | `TestLimitLearning_Default_0089`, unchanged, re-run |
| DL86 | The SDK's own value refused, the cap kept, the resend failing with an `*APIError` of any kind | decided *(2026-11-24: the owner's answer)* | that error carries `Observed` | OQ5, applied to every kept cap | `TestLimitRefusal_ResendFails_0089`, the 503 case; plant: the report applied only to the refusal's own error |
| DL87 | As DL86, the resend failing with an error that is not an `*APIError` (a cut connection, ctx's error) | decided | not reported on the error, whose type and text stay (R24); the cap stays in the memo, `CachedLimits` shows it "observed", and the next response it bounds names it in `OutputLimit.Source` | R24 (C170now) | the same test, a closed connection; plant: the error wrapped to carry the cap (its text changes) |
| DL88 | `WithRetry`, `WithStreamRetry` | decided | a refused caller value is `ErrInvalidRequest`, never retried, returned once with `Observed`; an attempt whose `*APIError` carried `Observed` and was retried gives it to the final result: a `WarningLimitLearned` with it on the final response unless one for the same model, base URL and scope is there, else the final `*APIError`'s `Observed` when zero | 0015 D8, 0051 D2; C169now | `TestRetry_LimitObservedCarried_0089`, red at the base on its assertion (0 warnings); plants: the carry removed from `Generate`; from `Stream` |
| DL89 | Scope: model, base URL, workspace or organization, stream | decided | `Observed` is the value a `WarningLimitLearned` carries for the key, so passed back with `WithLimitObservations` it matches the same provider's lookups (DL40); stream or plain is not carried (D5's dimension stays in the memo, DL76) | D5; DL40, DL76 | `TestLimitRefusal_CallerReport_0089`, pass-back case, Claude with a workspace (`claude.WithWorkspace`, `catalog.WithClaudeWorkspace`, DL75): a new provider given the error's `Observed` shows it "remembered"; plant: `Scope` left empty (the workspace lookup misses) |
| DL90 | `Seen` and the 24-hour pass-back window | decided | `Seen` is when the refusal was read, as the warning's; a refusal counts as seen, so a passed-back cap is used for 24 h from it (DL40) | D5; C19 | the pass-back case: `Seen` within the call's span; plant: `Seen` left zero (`New` refuses the pass-back, DL42) |
| DL91 | The error's identity | decided | `Error()` does not print `Observed` (as `Limit`, `Tool`, `ResetAt`); kinds, `Retryable` and every existing golden unchanged | R24, R25, R35 | DL78's case compares the text with `33e6f09`'s; plant: `Error()` prints the cap |
| DL92 | A failure after a stream's first event | out of scope | the limit's check is never consulted after an event (`StreamEventsRetry`); a refusal of the limit arrives before any output | | |
| DL93 | The exported API | decided *(2026-11-24: the owner's answer)* | one line, `./llmprovider.APIError.Observed: added`; `tools/apidiff.py v1.32.0` lists 22 lines; call site gobble-cli 0071 V4 (R55) | OQ5 "On the error (Recommended)"; R55 | `make api-check`, `tools/apidiff.py`; plant: a second name on a scratch clone, reported |
| DL94 | R24, R57 and the docs | decided | R24 lists `Observed`; R57's "reported in a `WarningLimitLearned`" adds "or, on a call that fails, its `*APIError`'s `Observed`"; the doc comments of `APIError`, `WarningLimitLearned`, `LimitObservation`, `WithLimitObservations`, `WithoutLimitLearning`, `WithRetry`, `WithStreamRetry` and each provider's output-limit paragraph; README's limit paragraph; one `CHANGELOG.md` entry under **Added** | D9; R43, R53 | a `tools/claims.py` row per new sentence, each planted false |

*(2026-11-24, annotation of DL37: its "no warning, since an error carries no response" is superseded by DL78 to DL82, by the owner's answer "On the error (Recommended)"; the rest of the row stands.)*

Counts: DL78 to DL94 add 16 decided rows and 1 out of scope (DL92); 0 open.

### Evidence added (2026-11-24, second)

The ids continue past the highest in use in this record (C102now) and in 0088-MADR (C160now), so that none is read as another record's. Each ends in "now", and is read at `33e6f09` with a plain id on a scratch clone, as the first amendment of this date did.

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C161now | a validated cap is kept for any basis | new | `llmprovider/internal/wire/limit_rule.go:206` | `if reading.Cap > 0 && reading.Keep && r.Keep != nil && r.Keep(reading.Cap) {` |
| C162now | the warning is built at the refusal | new | `llmprovider/internal/wire/limit_rule.go:209` | `r.warning = &llmprovider.Warning{Kind: llmprovider.WarningLimitLearned, Model: r.Model,` |
| C163now | a caller's value is not resent | new | `llmprovider/internal/wire/limit_rule.go:214` | `if r.sent.Basis != llmprovider.LimitBasisFact && r.sent.Basis != llmprovider.LimitBasisFallback {` |
| C164now | the warning rides a response only | new | `llmprovider/internal/wire/limit_rule.go:247` | `if w, ok := r.Warning(); ok {` |
| C165now | a stream reports at its done event only | new | `llmprovider/internal/wire/limit_rule.go:261` | `if err == nil && ev.Type == llmprovider.EventDone {` |
| C166now | Claude's `Generate` finishes on success only | new | `llmprovider/providers/claude/claude.go:343` | `if err == nil {` |
| C167now | the structured error | new | `llmprovider/api_error.go:147` | `type APIError struct {` |
| C168now | a field a provider sets after the send | new | `llmprovider/api_error.go:186` | `Tool string` |
| C169now | `WithRetry` drops a retried attempt's error | new | `llmprovider/retry.go:161` | `if final := r.after(ctx, attempt, err, &retriedAfterReply); final != nil {` |
| C170now | R24 | new | `docs/guides/api-standards.md:251` | `` - **R24. `*APIError` is the only structured error.** It carries `Provider`, `` |
| C171now | R28's rule against hooks | new | `docs/guides/api-standards.md:274` | `There are no hooks inside providers.` |
| C172now | the memo's cap shown as "observed" | new | `llmprovider/catalog/limits.go:454` | `b.add(rankObserved, sourceObserved, tokens, at, false)` |

### Evidence added (2026-11-24, Phase 6b): DL94's rows

One row per sentence Phase 6b wrote, as DL94 says. Each ends in "now" and was checked with a plain id against a scratch clone's commit of the phase's tree (`tools/_common.scratch_clone`), 26 of 26, and planted false there, 26 of 26 caught (0089-PLAN's Phase 6b entry). C176now is the field itself (DL93).

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C173now | R24 lists `Observed` (DL94) | new | `docs/guides/api-standards.md:254` | `` is `ErrToolSchemaRefused`, and `Observed`, the output cap a refusal stated `` |
| C174now | R57: reported on a failed call's `*APIError` (DL94) | new | `docs/guides/api-standards.md:438` | `` on a call that fails, its `*APIError`'s `Observed`, unless `` |
| C175now | `APIError.Observed`'s doc comment (DL94) | new | `llmprovider/api_error.go:189` | `Observed is the output cap the service stated when it refused the` |
| C176now | the field (DL93) | new | `llmprovider/api_error.go:199` | `Observed LimitObservation` |
| C177now | `APIError`'s doc comment (DL94) | new | `llmprovider/api_error.go:143` | `(0088-MADR ST79, ST81); a call that fails after the service refused its` |
| C178now | `WarningLimitLearned`'s doc comment (DL94) | new | `llmprovider/warning.go:46` | `*APIError's Observed carries the same value, on every such call` |
| C179now | `LimitObservation`'s doc comment (DL94) | new | `llmprovider/limit_observation.go:12` | `fails, as its *APIError's Observed (DL78); a caller that keeps it passes it` |
| C180now | `WithLimitObservations`' doc comment (DL94) | new | `llmprovider/limit_observation.go:32` | `or APIError.Observed reported (0089-MADR D5, DL89). catalog.LookupLimits` |
| C181now | `WithoutLimitLearning`'s doc comment (DL94) | new | `llmprovider/sent_limit.go:52` | `Observed (DL83), and still resends once at a validated stated cap (DL49).` |
| C182now | `WithRetry`'s doc comment (DL94) | new | `llmprovider/retry.go:91` | `An output cap a provider kept on an attempt that was retried, which` |
| C183now | `WithStreamRetry`'s doc comment (DL94) | new | `llmprovider/retry.go:277` | `An output cap a retried attempt's *APIError carried as Observed goes` |
| C184now | `chatcompat`'s output-limit paragraph (DL94) | new | `llmprovider/providers/chatcompat/chatcompat.go:97` | `A refusal of a given limit in OpenAI's "supports at most M" form,` |
| C185now | `huggingface`'s output-limit paragraph (DL94) | new | `llmprovider/providers/huggingface/huggingface.go:85` | `A refusal of a given limit in OpenAI's "supports at most M" form,` |
| C186now | `ollama`'s output-limit paragraph (DL94) | new | `llmprovider/providers/ollama/ollama.go:91` | `A refusal of a given limit in OpenAI's "supports at most M" form,` |
| C187now | `together`'s output-limit paragraph (DL94) | new | `llmprovider/providers/together/together.go:82` | `A refusal of a given limit in OpenAI's "supports at most M" form,` |
| C188now | `gemini`'s output-limit paragraph (DL94) | new | `llmprovider/providers/gemini/gemini.go:83` | `A refusal of a given limit is returned as it came: this wire reads no` |
| C189now | `grok`'s output-limit paragraph (DL94) | new | `llmprovider/providers/grok/grok.go:98` | `A refusal of a given limit is returned as it came: this wire reads no` |
| C190now | `openai`'s output-limit paragraph (DL94) | new | `llmprovider/providers/openai/openai.go:114` | `A refusal of a given limit is returned as it came: this wire reads no` |
| C191now | `claude`'s output-limit paragraph (DL94) | new | `llmprovider/providers/claude/claude.go:133` | `refused so is returned, not resent, and its M kept alike; a call that` |
| C192now | `kilo`'s output-limit paragraph (DL94) | new | `llmprovider/providers/kilo/kilo.go:103` | `in a cap form is returned, not resent, and its cap kept alike; a call` |
| C193now | `opencode`'s output-limit paragraph (DL94) | new | `llmprovider/providers/opencode/opencode.go:131` | `fails, as its error's APIError.Observed (DL78). A caller's own limit` |
| C194now | README's limit paragraph (DL94) | new | `README.md:254` | `after a cap was kept, a caller's own limit refused among them, reports it` |
| C195now | the changelog's **Added** entry (DL94) | new | `CHANGELOG.md:49` | `` - `APIError.Observed`: the output cap a service stated when it refused a `` |
| C196now | architecture's errors paragraph | new | `docs/architecture.md:779` | `that fails after the service refused its output limit and stated a cap the` |
| C197now | architecture's `WithRetry` paragraph | new | `docs/architecture.md:627` | `` A cap a retried attempt's `*APIError` carried in `Observed` goes to the `` |
| C198now | architecture's `internal/wire` row | new | `docs/architecture.md:174` | `` and `Finish`, which sets a kept cap on the `*APIError` that ends a failed call `` |

*(2026-11-24, Phase 6b built: DL78 to DL91, DL93 and DL94 tested, each plant caught; DL92 out of scope. The record's counts with this amendment: 87 decided, 6 out of scope (DL14, DL61 to DL64, DL92), 0 open, 1 awaiting live evidence (DL65); 94 rows.)*

### Addition (2026-11-24, third): Kilo's range form (DL95)

Additions only. The owner's word of 2026-11-24, relayed by the coordinator, quoted as given: "Add it in v1.33.0 (Recommended)": a seventh limit-refusal form, Kilo's, built in Phase 6b. Measured 2026-11-24 by a live probe (allowlisted output; two calls on Kilo, `z-ai/glm-5.3-flashx`, whose listing states 131,072): `max_tokens` 131,073 and 200,000 were each answered HTTP 400, a body of `error` only, its keys `message`, `type` ("AI_APICallError") and `param` (an object; its values not recorded), no `code`, and the message, the same for both, `The max_tokens parameter is illegal.：限制数值范围[1,131072]` (a fullwidth colon); 131,072 was answered 200, so the range is inclusive. The message states neither the value sent nor a model.

| Question | Options (recommended first) | Cost of each | Rows | The owner's answer |
| :--- | :--- | :--- | :--- | :--- |
| OQ6. Kilo's range form states no value sent (V2) and no model (V6): how is it validated? | (1) **Read it and keep the cap (Recommended)**: V2 becomes "the stated lower bound is 1", with V3 (M below the value sent) and V4 (M at least the wire's minimum) as for every cap form; no model named is kept, as Anthropic's form naming none; then as any cap form: a limit the SDK chose resent once at M, a caller's returned, M kept and reported. (2) **Read it for the resend only**, never kept, as a cap that fails V6. (3) **Not read in v1.33.0**; the row moves to v1.34.0. | (1) nothing ties M to the request or the model, so a range another upstream states would be kept for this model, bounded by the memo's key and by DL39's drop; one internal form, the docs' "six forms" become seven. (2) one refused send per request above M, in every process; a caller's refusal carries no `Observed`. (3) a Kilo request above the upstream's range fails with `ErrInvalidRequest`, not resent. | DL95 | "Read it and keep the cap (Recommended)" (2026-11-24): option (1) |

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| DL95 | Kilo refuses a limit in its range form, "The max_tokens parameter is illegal.：限制数值范围[L,M]" (400, `type` AI_APICallError, no `code`), measured 2026-11-24 | decided *(2026-11-24: the owner's answers)* | a seventh form, `FormKiloRange`, read on Kilo's wire only; M inclusive is the cap; validated as OQ6 (1) says: L must be 1, M below the value sent and at least 1, no model to check; then as any cap form: a limit the SDK chose is resent once at M (basis refusal, source "refusal: kilo max_tokens range"); a caller's limit is returned, not resent; M is kept for the process and reported in a `WarningLimitLearned`, or on a failed call in its `*APIError`'s `Observed` | the owner's word "Add it in v1.33.0 (Recommended)"; OQ6 "Read it and keep the cap (Recommended)" | `TestLimitRefusal_KiloRangeForm_0089`, red at the base on its assertion (one send, the refusal returned): a layer stating 200,000 for `z-ai/glm-5.3-flashx`, the measured body on the send at 200,000: resent at 131,072 and answered, the cap kept and reported; a caller's 200,000: one send, the cap kept; a lower bound of 2, and an M not below the value sent: one send, nothing kept; plants: the form not in Kilo's forms; M read as exclusive; the lower bound not checked; V3 not checked |

Counts: DL95 adds 1 decided row; with it the record holds 88 decided, 6 out of scope, 0 open, 1 awaiting live evidence (DL65); 95 rows.

### Evidence added (2026-11-24, third): DL95's rows

One row per sentence DL95's build wrote, checked as DL94's rows were, with plain ids against a scratch clone's commit of the tree, 6 of 6, and planted false there, 6 of 6 caught (0089-PLAN's Phase 6b entry).

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C199now | `FormKiloRange`'s doc comment (DL95) | new | `llmprovider/internal/wire/limit_refusal.go:44` | `FormKiloRange is Kilo's range form, measured on 2026-11-24 (0089-MADR` |
| C200now | Kilo's range form's pattern (DL95) | new | `llmprovider/internal/wire/limit_refusal.go:118` | `` {FormKiloRange, regexp.MustCompile(`max_tokens parameter is illegal `` |
| C201now | Kilo's wire reads it (DL95) | new | `llmprovider/internal/wire/limit_refusal.go:69` | `FormOpenAICap \| FormKiloRange \| FormOpenRouterWindow` |
| C202now | `kilo`'s output-limit paragraph (DL95) | new | `llmprovider/providers/kilo/kilo.go:96` | `states a cap (Anthropic's or OpenAI's form, or Kilo's range form "The` |
| C203now | the changelog's resend entry (DL95) | new | `CHANGELOG.md:87` | `range, each on the wires 0089-MADR DL68 names, and on Kilo its range form,` |
| C204now | architecture's `internal/wire` row (DL95) | new | `docs/architecture.md:174` | `its seven forms, Kilo's range form the seventh (0089-MADR DL95)` |
| C205now | `opencode`'s caller-limit sentence, every route that reads a cap form (DL94) | new | `llmprovider/providers/opencode/opencode.go:132` | `refused in a cap form, Anthropic's on the messages route, OpenAI's on` |

*(2026-11-24, at the review of Phase 6b: C205now, OpenCode's caller-limit sentence (DL94), is added to the table above at the review's note, and C193now re-pointed to that paragraph's changed line; the 33 rows C173now to C205now pass against a scratch clone's commit of the tree, and each is caught planted false.)*

### Annotation (2026-11-24, fourth): DL65 measured

Additions only; the DL65 row above is annotated, not rewritten. The owner's ruling, relayed by the coordinator, quoted as given: "Write the plan's test, then run it". `TestLive_OmittedLimit_0089` (`llmprovider/live_0089_omitted_test.go`, with its request and verdict word in `llmprovider/omitted_0089_test.go` and an offline twin, `TestOmittedLimit_Twin_0089`, 4 of 4 plants caught) was run once per model on Kilo's `/chat/completions`, streamed, with NO limit field sent, `KILO_API_KEY` from the environment, a 20-minute deadline, one request each, output allowlisted:

| Model | Stated limit (listing) | Status | Finish | Output tokens | Reasoning tokens | Verdict |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `anthropic/claude-haiku-4.5` | 64,000 | 200 | `stop` | 27,726 | 0 | undecided (a `stop` decides nothing) |
| `z-ai/glm-5.3-flashx` | 131,072 | 200 | `length` | 128,000 | 173 | below the stated limit |

Run in the order the ruling named: the second model ran because the first was undecided. `anthropic/claude-opus-4.1` (32,000) was not used: its long stream died on Kilo's gateway timeout, an in-stream 503, before any finish (a scratch probe of 2026-11-24, not a run of the test).

| Row | Input or state | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| DL65 | Kilo, no limit sent, a model whose stated limit is L | decided *(2026-11-24: measured)* | a `length` finish below the stated limit means Kilo fills less, so Kilo keeps sending the discovered limit; the interim is the decision, and no code changes: `z-ai/glm-5.3-flashx` ended in `length` at 128,000 output tokens with 131,072 stated (3,072 short; the 173 reasoning tokens do not change that, whether the count includes them or not) | the measurement, under 0089-PLAN's "OQ1 (1)'s measurement" rule; OQ1 "Only where required (Recommended)" | `TestLive_OmittedLimit_0089` (live; its offline twin's plants are caught); `TestOutputLimit_Omitted_0089`'s Kilo case keeps pinning the send |

What the measurement does not show: one model ended in `length`, and the other stopped by itself, so the verdict rests on one model's one run; it shows that Kilo (or the upstream behind it) handed out less than the stated limit for a request with no limit, not why. A later record decides whether a second model's `length` finish is worth a run.

Counts: DL65 moves from awaiting live evidence to decided; with it the record holds 89 decided, 6 out of scope, 0 open, 0 awaiting live evidence; 95 rows.

*(2026-11-24, review note: the cause.)* The run does not show why the stream ended at 128,000 of 131,072 stated. "Kilo fills less" in the PLAN's rule and in this annotation means only a `length` finish below the stated limit, with no limit sent; it does not say that Kilo, or the upstream model, applies a default lower than the listing, or that the model's real cap is 128,000, below its listing. The run cannot tell these apart: it had no control at 131,072 sent. DL65's outcome, Kilo keeps sending the discovered limit, holds on every reading. The words "because Kilo's own default is lower" and "whose default is lower" that the first pass of the docs step wrote in `kilo.go`, `limit_rule.go`, `CHANGELOG.md` and `docs/guides/api-standards.md` are replaced by the measurement alone: "measured 2026-11-24: with none sent, Kilo cut one model at 128,000 of 131,072 stated". Rows C67now and C206now to C211now are re-pointed to those sentences in the tables of this record, and each is caught planted false.

*Follow-up, recorded and not run:* a control run of the same model, `z-ai/glm-5.3-flashx`, one streamed request with `max_tokens` 131,072 sent, billing up to 131,072 output tokens. A `length` finish at 128,000 would put the cut in the model, not Kilo; one at 131,072 would put it in what Kilo fills when none is sent. Either way the interim stands, so the run decides only whether the sentences may name the cause, and it needs the owner's word before it runs.

### Evidence added (2026-11-24, fourth): DL65's docs step

Additions only; no row's decision changes. The coordinator's word of 2026-11-24, "option 1", on the open docs rows of 0089-PLAN's entry "DL65 measured (executed)": every sentence that said Kilo sends its discovered limit "until DL65's measurement decides" now states DL65 as decided: Kilo keeps sending the discovered limit because, measured 2026-11-24, it cut `z-ai/glm-5.3-flashx` at 128,000 output tokens with 131,072 stated when no limit was sent. Comments and docs only; no code changes. Claims row C67now (`docs/guides/api-standards.md:127`) is re-pointed to the reworded sentence, in its own row above; C98now (`llmprovider/providers/kilo/kilo.go:88`) names DL65 as a citation only, its line is unchanged, and it still passes. Every edit keeps its file's line count, so no other row moves.

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C206now | `kilo`'s package doc cites DL65's measurement | new | `llmprovider/providers/kilo/kilo.go:94` | `// (0088-MADR ST87), as 0089-MADR DL65 decided (measured 2026-11-24: with none sent,` |
| C207now | `kilo`'s package doc, the measured cut (DL65) | new | `llmprovider/providers/kilo/kilo.go:95` | `// Kilo cut one model at 128,000 of 131,072 stated); with no fact, none is sent. A refusal of a limit the SDK chose that` |
| C208now | `limitInput`'s doc comment cites DL65's measurement | new | `llmprovider/providers/kilo/kilo.go:540` | `// Kilo sends the chosen fact, as 0089-MADR DL65 decided (measured 2026-11-24: with` |
| C209now | `limitInput`'s doc comment, the measured cut (DL65) | new | `llmprovider/providers/kilo/kilo.go:541` | `// none sent, Kilo cut one model at 128,000 of 131,072 stated), and requires none.` |
| C210now | `LimitInput.Sends`'s doc comment (DL65) | new | `llmprovider/internal/wire/limit_rule.go:26` | `// Messages wire, and Kilo (0089-MADR DL65, measured 2026-11-24; OQ1 (1)).` |
| C211now | the changelog's Kilo sentence (DL65) | new | `CHANGELOG.md:70` | `2026-11-24: with none sent, Kilo cut one model at 128,000 of 131,072 stated). Every other wire sends none, so the` |
| C212now | `TestOutputLimit_Facts_0041`'s doc comment (DL65) | new | `llmprovider/providers/facts_0041_test.go:66` | `// route, and Kilo, as DL65 decided) asks for the model's stated limit,` |
| C213now | `TestDefaultLimit_WindowCap_0088`'s doc comment (DL65) | new | `llmprovider/providers/default_limit_0088_test.go:219` | `// discovers (0089-MADR DL65, measured): a layer stating L 16,384 and W` |
| C214now | `TestOutputLimit_Omitted_0089`'s doc comment, the cited rows (DL65) | new | `llmprovider/providers/output_limit_0089_test.go:108` | `// TestOutputLimit_Omitted_0089 (OQ1 (1): DL46 to DL48, DL65's verdict,` |
| C215now | `TestOutputLimit_Omitted_0089`'s doc comment, Kilo's case (DL65) | new | `llmprovider/providers/output_limit_0089_test.go:112` | `// registry) with no fixed ceiling, and Kilo its discovered limit, as` |
| C216now | `TestOutputLimit_Omitted_0089`'s Kilo comment (DL65) | new | `llmprovider/providers/output_limit_0089_test.go:152` | `// Kilo: its discovered limit, as DL65 decided.` |
| C217now | `TestOutputLimit_Omitted_0089`'s Kilo assertion text (DL65) | new | `llmprovider/providers/output_limit_0089_test.go:163` | `want 128000 (DL65's verdict)` |

*(2026-11-24, note from [0093-MADR-take-model-metadata-from-the-caller.md](0093-MADR-take-model-metadata-from-the-caller.md): D1 to D3 (the registry is the handed-in document; the extract is gone), D5 (the memo no longer carries the route's fallback: `Limits.RouteFallback` does, per call, D7), DL23, DL24 and DL29 (no extract, no fetch age), DL50, DL51 (the refusal also names `catalog.WithMetadata`, D9), DL70 (its annotation "One fetch, then ship (Recommended)" is moot: nothing is fetched or shipped), DL71, DL74, DL77, the OQ2 ruling's "reads the cache and the embedded extract" (no extract remains), and the claims rows C62now and C63now, whose text 0093 rewrote (the note below). The text above is unchanged.)*

*(2026-11-24, from 0093-PLAN-take-model-metadata-from-the-caller.md's Phase 5: this record's "now" rows on a file 0093's Phase 4 edited were re-run with plain ids against a scratch clone's commit of the tree (`dc96bf0ce4e6`, 0093's build, uncommitted), and against `cfc911d`, its base. 74 fail there: 58 moved by 0093's build (55 found on a new line, 3 whose text 0093 rewrote), and 16 already failing at `cfc911d` (13 found on another line, 3 on none). Each row's text is now where this note says; the rows above are unchanged.)*

* C33now (`docs/guides/api-standards.md:371`): `docs/guides/api-standards.md:418`; already failing at `cfc911d`
* C51now (`docs/guides/api-standards.md:104`): on no line; already failing at `cfc911d`
* C56now (`docs/guides/api-standards.md:116`): `docs/guides/api-standards.md:117`; moved by 0093's build
* C57now (`docs/guides/api-standards.md:117`): `docs/guides/api-standards.md:118`; moved by 0093's build
* C58now (`docs/guides/api-standards.md:118`): `docs/guides/api-standards.md:119`; moved by 0093's build
* C59now (`docs/guides/api-standards.md:119`): `docs/guides/api-standards.md:120`; moved by 0093's build
* C60now (`docs/guides/api-standards.md:120`): `docs/guides/api-standards.md:121`; moved by 0093's build
* C61now (`docs/guides/api-standards.md:121`): `docs/guides/api-standards.md:122`; moved by 0093's build
* C62now (`docs/guides/api-standards.md:122`): on no line; moved by 0093's build
* C63now (`docs/guides/api-standards.md:123`): on no line; moved by 0093's build
* C64now (`docs/guides/api-standards.md:124`): `docs/guides/api-standards.md:125`; moved by 0093's build
* C65now (`docs/guides/api-standards.md:125`): `docs/guides/api-standards.md:126`; moved by 0093's build
* C66now (`docs/guides/api-standards.md:126`): `docs/guides/api-standards.md:127`; moved by 0093's build
* C67now (`docs/guides/api-standards.md:127`): on no line; moved by 0093's build
* C68now (`docs/guides/api-standards.md:128`): `docs/guides/api-standards.md:130`; moved by 0093's build
* C69now (`docs/guides/api-standards.md:129`): `docs/guides/api-standards.md:131`; moved by 0093's build
* C70now (`docs/guides/api-standards.md:130`): `docs/guides/api-standards.md:132`; moved by 0093's build
* C71now (`docs/guides/api-standards.md:131`): `docs/guides/api-standards.md:133`; moved by 0093's build
* C72now (`docs/guides/api-standards.md:132`): `docs/guides/api-standards.md:134`; moved by 0093's build
* C73now (`docs/guides/api-standards.md:133`): `docs/guides/api-standards.md:135`; moved by 0093's build
* C74now (`docs/guides/api-standards.md:134`): `docs/guides/api-standards.md:136`; moved by 0093's build
* C75now (`docs/guides/api-standards.md:135`): `docs/guides/api-standards.md:137`; moved by 0093's build
* C76now (`docs/guides/api-standards.md:139`): `docs/guides/api-standards.md:141`; moved by 0093's build
* C77now (`docs/guides/api-standards.md:140`): `docs/guides/api-standards.md:142`; moved by 0093's build
* C78now (`docs/guides/api-standards.md:141`): `docs/guides/api-standards.md:143`; moved by 0093's build
* C79now (`docs/guides/api-standards.md:142`): `docs/guides/api-standards.md:144`; moved by 0093's build
* C80now (`docs/guides/api-standards.md:427`): `docs/guides/api-standards.md:440`; already failing at `cfc911d`
* C81now (`docs/guides/api-standards.md:428`): `docs/guides/api-standards.md:441`; already failing at `cfc911d`
* C82now (`docs/guides/api-standards.md:429`): `docs/guides/api-standards.md:442`; already failing at `cfc911d`
* C83now (`docs/guides/api-standards.md:430`): `docs/guides/api-standards.md:443`; already failing at `cfc911d`
* C84now (`docs/guides/api-standards.md:431`): `docs/guides/api-standards.md:444`; already failing at `cfc911d`
* C85now (`docs/guides/api-standards.md:432`): `docs/guides/api-standards.md:445`; already failing at `cfc911d`
* C86now (`docs/guides/api-standards.md:433`): `docs/guides/api-standards.md:446`; already failing at `cfc911d`
* C87now (`docs/guides/api-standards.md:434`): `docs/guides/api-standards.md:447`; already failing at `cfc911d`
* C88now (`docs/guides/api-standards.md:435`): `docs/guides/api-standards.md:448`; already failing at `cfc911d`
* C89now (`docs/guides/api-standards.md:436`): on no line; already failing at `cfc911d`
* C90now (`docs/guides/api-standards.md:437`): on no line; already failing at `cfc911d`
* C91now (`docs/guides/api-standards.md:456`): `docs/guides/api-standards.md:470`; already failing at `cfc911d`
* C92now (`llmprovider/catalog/doc.go:44`): `llmprovider/catalog/doc.go:46`; moved by 0093's build
* C94now (`llmprovider/providers/claude/claude.go:116`): `llmprovider/providers/claude/claude.go:117`; moved by 0093's build
* C95now (`llmprovider/providers/gemini/gemini.go:78`): `llmprovider/providers/gemini/gemini.go:79`; moved by 0093's build
* C96now (`llmprovider/providers/grok/grok.go:93`): `llmprovider/providers/grok/grok.go:94`; moved by 0093's build
* C97now (`llmprovider/providers/huggingface/huggingface.go:80`): `llmprovider/providers/huggingface/huggingface.go:81`; moved by 0093's build
* C98now (`llmprovider/providers/kilo/kilo.go:88`): `llmprovider/providers/kilo/kilo.go:89`; moved by 0093's build
* C99now (`llmprovider/providers/ollama/ollama.go:86`): `llmprovider/providers/ollama/ollama.go:87`; moved by 0093's build
* C100now (`llmprovider/providers/openai/openai.go:109`): `llmprovider/providers/openai/openai.go:110`; moved by 0093's build
* C101now (`llmprovider/providers/opencode/opencode.go:116`): `llmprovider/providers/opencode/opencode.go:117`; moved by 0093's build
* C102now (`llmprovider/providers/together/together.go:77`): `llmprovider/providers/together/together.go:79`; moved by 0093's build
* C166now (`llmprovider/providers/claude/claude.go:343`): `llmprovider/providers/claude/claude.go:347`; already failing at `cfc911d`
* C170now (`docs/guides/api-standards.md:251`): `docs/guides/api-standards.md:254`; moved by 0093's build
* C171now (`docs/guides/api-standards.md:274`): `docs/guides/api-standards.md:279`; already failing at `cfc911d`
* C173now (`docs/guides/api-standards.md:254`): `docs/guides/api-standards.md:257`; moved by 0093's build
* C174now (`docs/guides/api-standards.md:438`): `docs/guides/api-standards.md:449`; moved by 0093's build
* C185now (`llmprovider/providers/huggingface/huggingface.go:85`): `llmprovider/providers/huggingface/huggingface.go:86`; moved by 0093's build
* C186now (`llmprovider/providers/ollama/ollama.go:91`): `llmprovider/providers/ollama/ollama.go:92`; moved by 0093's build
* C187now (`llmprovider/providers/together/together.go:82`): `llmprovider/providers/together/together.go:84`; moved by 0093's build
* C188now (`llmprovider/providers/gemini/gemini.go:83`): `llmprovider/providers/gemini/gemini.go:84`; moved by 0093's build
* C189now (`llmprovider/providers/grok/grok.go:98`): `llmprovider/providers/grok/grok.go:99`; moved by 0093's build
* C190now (`llmprovider/providers/openai/openai.go:114`): `llmprovider/providers/openai/openai.go:115`; moved by 0093's build
* C191now (`llmprovider/providers/claude/claude.go:133`): `llmprovider/providers/claude/claude.go:134`; moved by 0093's build
* C192now (`llmprovider/providers/kilo/kilo.go:103`): `llmprovider/providers/kilo/kilo.go:104`; moved by 0093's build
* C193now (`llmprovider/providers/opencode/opencode.go:131`): `llmprovider/providers/opencode/opencode.go:132`; moved by 0093's build
* C194now (`README.md:254`): `README.md:256`; moved by 0093's build
* C195now (`CHANGELOG.md:49`): `CHANGELOG.md:66`; moved by 0093's build
* C196now (`docs/architecture.md:779`): `docs/architecture.md:775`; moved by 0093's build
* C197now (`docs/architecture.md:627`): `docs/architecture.md:623`; moved by 0093's build
* C202now (`llmprovider/providers/kilo/kilo.go:96`): `llmprovider/providers/kilo/kilo.go:97`; moved by 0093's build
* C203now (`CHANGELOG.md:87`): `CHANGELOG.md:105`; moved by 0093's build
* C205now (`llmprovider/providers/opencode/opencode.go:132`): `llmprovider/providers/opencode/opencode.go:133`; moved by 0093's build
* C206now (`llmprovider/providers/kilo/kilo.go:94`): `llmprovider/providers/kilo/kilo.go:95`; moved by 0093's build
* C207now (`llmprovider/providers/kilo/kilo.go:95`): `llmprovider/providers/kilo/kilo.go:96`; moved by 0093's build
* C208now (`llmprovider/providers/kilo/kilo.go:540`): `llmprovider/providers/kilo/kilo.go:541`; moved by 0093's build
* C209now (`llmprovider/providers/kilo/kilo.go:541`): `llmprovider/providers/kilo/kilo.go:542`; moved by 0093's build
* C211now (`CHANGELOG.md:70`): `CHANGELOG.md:88`; moved by 0093's build

*(2026-11-27: the control run of the review note on DL65's cause ran as [0095-MADR-clear-the-sdk-follow-ups.md](0095-MADR-clear-the-sdk-follow-ups.md)'s FU24: `z-ai/glm-5.3-flashx` with `max_tokens` 131,072 sent ended `length` at 131,072 output tokens, so the cut at 128,000 with none sent is what Kilo fills when no limit is sent, not the model. DL65 stands.)*
