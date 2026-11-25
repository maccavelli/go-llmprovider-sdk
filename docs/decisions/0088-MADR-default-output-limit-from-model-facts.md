---
status: accepted
date: 2026-11-25
decision-makers: repository owner
consulted: "the owner's answers of 2026-11-21 to OQ1 to OQ8, relayed by the coordinator; the coordinator's brief of 2026-11-21, which relays a gobble-cli report and the owner's choice among three options (its label quoted below), with facts the coordinator verified; record review 1 of this pair (2026-11-21), each finding re-verified against the code before it was written in; this repository at 55a5dca, read only; gobble-cli at 4c3e397, read only; the read-only upstream checkouts kilocode at 58072bd696, opencode at 4ce137b39e, codex at 57e64682f8 and grok-build at 2bdd1d6a; Anthropic's API reference as bundled with the drafting agent's tools, read 2026-11-21"
informed: every caller of Generate and Stream that sends a request with no MaxOutputTokens and no WithMaxTokens; gobble-cli first, whose agent turns name no limit; prepare-commit-msg and magic-cli-remote
---

<!-- markdownlint-disable MD013 MD024 -->

# Send the Model's Own Output Limit When a Request Names None, So Reasoning Does Not Spend the Whole 8192-Token Default

Associated PLAN: [0088-PLAN-default-output-limit-from-model-facts.md](0088-PLAN-default-output-limit-from-model-facts.md)

*(2026-11-22: reopened by the research amendment, whose open rows ST63, ST65, ST66, ST67, ST71 and ST74 make the record `proposed` by the records rule; the owner's 2026-11-21 answers stand. It was `accepted` on 2026-11-21. See "Amendment (2026-11-22)" at the end.)* *(2026-11-22, second: the owner answered OQ10 to OQ12, deciding ST65, ST66, ST67 and ST71; OQ9 is reworked from evidence, with ST63, ST74 and ST85 to ST89 open; still `proposed`. See "Amendment (2026-11-22, second)".)* *(2026-11-22, third: after record review 2, OQ9 is split into OQ9, OQ14, OQ15 and OQ17, and the new export is OQ16; ST63 is decided from xAI's schema; still `proposed`. See "Amendment (2026-11-22, third)".)* *(2026-11-22, fourth: the owner answered OQ14 to OQ16 and replied on OQ9 by asking for model output-limit discovery, now 0089-MADR; OQ9's rows wait for 0089's OQ1, and the release is OQ18; still `proposed`. See "Amendment (2026-11-22, fourth)".)* *(2026-11-22, fifth: the owner answered OQ18 and 0089-MADR's OQ1 to OQ4, which decide every open row; `accepted`. See "Amendment (2026-11-22, fifth)".)* *(2026-11-25, fifteenth: the owner ruled R1 to R3, now rows ST107 to ST109; R1's ruling leaves one case open, the streamed Interaction (ST110, OQ19), so by the records rule the record is `proposed` again until OQ19 is answered. See "Amendment (2026-11-25, fifteenth)".)* *(2026-11-25, sixteenth: OQ19 answered, "Decide by what arrived"; ST110 decided; 0 rows open; `accepted`. See "Amendment (2026-11-25, sixteenth)".)*

## Context and Problem Statement

When a request names no output limit (no `Request.MaxOutputTokens`, no
`WithMaxTokens`), every built-in provider sends the SDK's default, 8192
tokens (C1, C2), cut to the model's own limit when the model facts state a
smaller one, and never raised (C6 to C16; 0041-MADR D2 R7, the owner's Q3
ruling "cut only", C46, C47). On a wire where reasoning counts against that
limit, a model reasoning at a high effort over a large context can spend all
of it on reasoning and stop with nothing to show.

### The report (2026-11-21, facts verified by the coordinator)

* gobble-cli ran a large multi-file turn on Kilo's `kilo-auto/balanced`
  with thinking `high`. It failed with
  `llm: api status 0: llm: incomplete response: llmprovider: empty answer: length: chat completions: the answer has no content`.
* gobble's agent request sets no output limit (C55): its `llm.Request` has
  no `MaxTokens`, so the SDK request's `MaxOutputTokens` is 0 (C56) and the
  SDK sends its default, 8192 (C1). Kilo cuts that default to the model's
  limit when its facts state one, and never raises it (C6). gobble's own
  catalog states `max-out 65536` for that model.
* On Chat Completions, reasoning tokens count against `max_tokens`. At high
  effort with a big context, the model spent all 8,192 on reasoning and
  finished `length` with no text, which the SDK reports as `ErrEmptyAnswer`
  (C30, C31; 0026-MADR F7; 0061-MADR D1). Earlier turns in the same session
  used 504 and 2,518 output tokens.

### The owner's choice

Of three options put by the coordinator, the owner chose **"Fix the default
in the SDK instead"**. The other two were a gobble-cli record for the budget
and a clear message, and a gobble-cli message-only fix. The option's stated
cost: "Fixes every SDK caller, but it's another record in the SDK's tag
scope, which you set to five; gobble's message stays as it is." This record
decides how the SDK fixes the default; which option, and the details below,
are the owner's: the owner answered all eight questions on 2026-11-21
(Owner rulings, below).

### How the limit is chosen and sent today (read at 55a5dca)

* `newSettings` sets `MaxTokens: 8192` (C1); `WithMaxTokens(n)` overwrites
  it (C4); `Settings.MaxTokens` returns one number, so a provider cannot
  tell a caller's `WithMaxTokens(8192)` from the default (C2). The
  precedents for telling a given option apart are `Settings.HTTPClientGiven`
  (C3; 0026-MADR F37), derived from the value because the default is nil
  (C110), and `Settings.LoggerGiven` (C111). The default output limit is
  not zero, so a flag for it must be set by `WithMaxTokens` itself.
* Each provider's body function takes `req.MaxOutputTokens` when it is
  positive, as given, neither cut nor raised; otherwise the configured value,
  cut to `facts.MaxOutputTokens` when the facts state a positive one (C6,
  C8 to C16). One function per provider serves both `Generate` and
  `Stream`; `Stream` adds `stream: true` (C42).
* The facts are read once per request from what is already cached, never
  fetched and never waited for (C85; 0041-MADR D1): a caller's layers, a
  listing a `catalog.List`, `ListModels` or `catalog.CachedModel` call
  already read, and the models.dev document. `catalog.CachedFacts` reads a
  cached listing whatever `WithoutModelMetadata` says, which drops only the
  models.dev document (C75, C76). Kilo's request reads its facts this way
  (C84). `ModelInfo.SourceOf` names the layer that set a field (C103).
* Only three listings state an output limit: Gemini's, Claude's and Kilo's
  (C104, C105, C106; Kilo's field is `top_provider.max_completion_tokens`,
  C33). Together, Hugging Face, Ollama and chatcompat get one only from a
  caller's layer or from models.dev; chatcompat reads models.dev by the
  model's canonical id, with quantisation and format tokens dropped (C102),
  so its fact is the hosted model's, not the local server's.
* The fields, per wire: Chat Completions `max_tokens` (C18: Kilo, Together,
  Hugging Face, Ollama, chatcompat and OpenCode's chat route); Responses
  `max_output_tokens` (C20, C21, C23: OpenAI with a key, Grok, OpenCode's
  responses route), and none at all on the ChatGPT backend and the ChatGPT
  plan (C17; 0043-MADR D5); Messages `max_tokens`, which the wire requires
  (C19, C24: Claude and OpenCode's messages route); Gemini's Interactions
  `generation_config.max_output_tokens` (C22); generateContent
  `maxOutputTokens` (C25: OpenCode's google route).
* On Messages with the budget thinking shape, `AddThinkingFor` raises
  `max_tokens` to the budget plus 4096 when it is not above the budget
  (C26 to C28), and stops at the facts' limit. Adaptive thinking carries no
  budget and leaves `max_tokens` as given.
* Kilo's reasoning object is `{effort}` with an effort, else
  `{enabled: true}` (C32); no reasoning budget is sent on Kilo.
* A listing probe runs through the provider's own `Generate`, on a copy whose
  configured limit is set to the probe's fixed 8192 (C48, C107), "kept so
  the probe's request is unchanged" (C108; 0015-MADR D1). A probe reads the
  model's facts as any request does, and runs just after the listing that
  warms them.
* A refused request is built again and resent only through 0062-MADR D5's
  path, at most seven sends in all (C86, C87; 0062-MADR Q5 (a), an owner
  ruling): one per effort rung and one for the other thinking shape, so the
  worst case already uses all seven. A stream is resent only when its
  opening fails (C88); an error inside a 200, as Kilo sends an overflow
  (C39), is classified while the stream is read (C89), after the opening.
  `Generate`'s `WithRetry` retries a timeout awaiting the reply's headers
  once (C92; 0028-MADR D-H1, C93).

### gobble's case under a raise from the facts

The SDK's 2026-10-12 Kilo snapshot states `max_completion_tokens` 65,536 for
`kilo-auto/balanced` and `kilo-auto/efficient`, 128,000 for
`kilo-auto/frontier`, and 32,768 for `kilo-auto/free` and `kilo-auto/small`
(C77, C78; the coordinator's reading and a scratch script's at 55a5dca);
the 2026-09-26 snapshot states none. gobble's provider has no models.dev
document (C57), but gobble passes its run cache on every path (C79, C80),
and its model lookup calls `catalog.CachedModel` (C81), which starts the
Kilo listing fetch (C82). Once that listing is cached under the key the
provider's request reads (C83: provider, base URL, organization, credential
fingerprint), the request's facts state 65,536, and a raise from the facts
reaches gobble's model. The first request before the fetch lands still
sends 8192: no request waits (C85). Whether gobble's lookup and its provider
compute the same key is gobble-cli's to show (ST58).

### What a raised limit costs, as far as it was established

* **Billing.** Kilo bills tokens used, at the upstream's price (C65). No
  local source shows any service charging for an unused part of a limit.
  Anthropic's reference bills thinking whether or not it is shown.
* **Rate limits.** Anthropic counts `max_tokens` toward its output tokens per
  minute limit when a request starts, and adjusts it when the request ends
  (stated in the coordinator's brief; not re-read in a local source). Kilo's
  gateway does not rate-limit paid requests itself; upstreams may (C66).
  Whether OpenAI, xAI, Google, Together, Hugging Face or OpenCode count a
  limit rather than tokens used was not established.
* **A limit the service will not take.** Anthropic refuses one in the form
  the SDK's own fixture records, `max_tokens: 999999 > 128000` (C40); the
  SDK classifies it as `ErrInvalidRequest` with no overflow reason, so it is
  not an overflow and nothing resends it. A raise from the facts exceeds the
  service's limit only when the facts overstate it: a stale listing, a
  caller's layer, or a gateway alias whose routed upstream takes less.
* **The context window.** OpenAI-compatible services refuse input plus
  `max_tokens` past the window: Together's form names the split
  "8000 in the messages, 1000 in the completion" (C38), Kilo passes
  OpenRouter's "maximum context length ... requested about" (C39). The SDK
  classifies such a refusal as `ErrContextOverflow` (C37), or, when the
  split shows the completion alone fills the window, as `ErrInvalidRequest`
  with the reason "max_tokens alone fills the context window: lower it"
  (C35; 0028-MADR D-H3). Kilo's gateway documents another form, "This
  request exceeds the model's context window of 200000 tokens. Your request
  contains approximately 250000 tokens." (C97), which no overflow form
  matches today (checked by a scratch script against every form at 55a5dca):
  it is `ErrInvalidRequest`. Several forms name the input alone over the
  window ("prompt is too long", xAI's "maximum prompt length is N",
  Together's "input (N tokens) is longer than the model's context length",
  C99 to C101); 8192 would be refused there too. opencode says "strict ones
  reject input plus output past the window" (C70); kilocode caps its limit
  to the room left for that reason (C63). On Claude 4.5 and later,
  Anthropic's reference says a request that reaches the window stops with
  `model_context_window_exceeded`, which the SDK maps to `FinishLength`
  (C29).
* **The window the facts leave.** In the 2026-10-12 Kilo snapshot, eight
  models whose stated limit is held to 32,000 would leave under 16,384
  tokens of window for input: `rekaai/reka-edge` and `microsoft/phi-4`
  (window 16,384, limit 14,745, leaving 1,639) and six with a 32,768 window
  leaving 3,277 to 6,554. models.dev states `google/gemini-2.5-flash-image`
  at 32,768 for both window and limit, leaving 768. Kilo's stated limits
  reach 1,800,000 (C124); 303 of the 383 stated are above 32,000, and 82
  equal 90% of their window, such as 943,718 of 1,048,576 (C125).
* **Streaming or not.** The default transport waits 300 s for a reply's
  headers (C41). A non-streaming `Generate` gets its headers only when the
  answer is done. 32,000 tokens take more than 300 s at any rate below about
  107 tokens per second, and 16,384 below about 55. The timeout is
  `ErrProviderUnavailable`, which `Generate`'s `WithRetry` retries once (C92),
  so a long answer can fail after two waits; whether a service bills an
  answer whose client stopped waiting was not established. `Stream` gets its
  headers at once.
* **A balance refusal.** Kilo returns 402 when the balance reaches zero
  (C95), which the SDK classifies as `ErrQuotaExhausted` (C94); an
  upstream's 402 reaches the client as 503 (C96), `ErrProviderUnavailable`
  (C98). No local source shows a refusal that depends on the size of the
  limit.

### What the reference clients do

* **kilocode** (Kilo's own client): the default is the model's limit capped
  at 32,000 (C59, C60, C64). For Claude on first-party routes it asks for the
  model's whole limit, because "the shared 32k default can be used up by
  thinking before any text or tool call is written" (C61, C62). It caps the
  limit to the room the prompt leaves in the window (C63). On OpenRouter it
  can send a reasoning budget, `reasoning: {max_tokens}` (C67).
* **opencode**: the catalog limit, 32,000 when the catalog has none, never
  above 256,000, fitted to the room the prompt leaves (C68 to C70); on
  OpenRouter the reasoning budget is held below the output limit (C71).
* **codex**: its Responses request has no output-limit field at all (C72).
* **grok-build**: an agent turn sends no output limit; side tasks set small
  ones, such as a session summary's 100 (C73, C74).

## Decision Drivers

* A request that names no limit must not fail because of a limit the SDK
  chose for it.
* A limit the caller gives, by `WithMaxTokens` or `MaxOutputTokens`, is never
  changed.
* No regression: a request that succeeds at 8192 today does not start to
  fail (a window refusal, a limit refusal, a timeout, a starved resend
  ladder).
* Per-model decisions come from the model facts, not from ids (0041-MADR D1).
* Costs stay bounded and stated: rate-limit reservations, latency, the
  non-streaming header wait, extra sends.
* The smallest exported change that lets a provider apply the rule
  (docs/guides/api-standards.md R55, C109).

## Considered Options

* A. When the caller names no limit, send the model's output limit where the
  facts state it, else today's default
* B. Raise the default only when reasoning is requested, by the effort asked
* C. Raise the global default, to 32,000
* D. Keep the default, and reserve room for the answer with a reasoning
  budget below the limit, where the wire takes one
* E. Send no limit when the caller names none, where the wire allows it

## Decision Outcome

Chosen option: "A. When the caller names no limit, send the model's output
limit where the facts state it, else today's default",
because it takes the number from the model's own facts, as 0041 decided for
the cut; it fixes the empty answer wherever the facts know the limit, on
every wire and at every effort, gobble's model included once its listing is
cached; and it leaves a caller's limit and an unknown model exactly as
today. The owner chose it, and answered every detail below, on 2026-11-21
(Owner rulings); no state row is open. *(2026-11-22: six rows are open again, added by the research amendment; the owner's answers stand.)*

### Owner rulings (2026-11-21)

Each answer is the owner's, relayed by the coordinator, quoted as given.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ1 | "The model's stated limit (Recommended)" | (1): option A | ST1 to ST3, ST5 to ST8, ST10, ST11, ST25, ST28, ST29, ST31, ST32, ST33 |
| OQ2 | "32k streaming, 16k plain (Recommended)" | (1): 32,000 on `Stream`, 16,384 on `Generate` | ST1, ST3, ST5 to ST8, ST10, ST24, ST25, ST32, ST33, ST38 |
| OQ3 | "Fit to window, resend once (Recommended)" | (1): fit to W/2 and resend once on the recognised forms | ST21, ST22, ST23, ST47 to ST52 |
| OQ4 | "Keep 8,192 (Recommended)" | (1): 8192 when the facts state no limit or are not yet cached | ST12, ST34, ST60 |
| OQ5 | "Export MaxTokensGiven" | (2), not the recommended one: exported `Settings.MaxTokensGiven() bool`; R55's cost, an export whose call sites are all in this module with no outside call site named, is accepted by this answer | ST35, ST36, ST37 |
| OQ6 | "Streams too, one extra send (Recommended)" | (1): the stream resend, the sticky undo, and the undoing send outside the bound; amends 0062-MADR Q5 (a) for a raised request: eight sends | ST55, ST56, ST57 |
| OQ7 | "Only from a caller layer (Recommended)" | (1): on chatcompat and Ollama, raise only from a caller's layer | ST9 |
| OQ8 | "Join this tag" | (2), not the recommended one: this tag's scope becomes six (0081, 0083, 0084, 0085, 0087, 0088); 0088 is built last, after 0087, and the tag waits for 0088's phases and review | ST43 |

The answers were checked against each other: OQ5 (2) changes only how a
provider reads the flag, which OQ3, OQ6 and OQ7 do not depend on; OQ8 (2)
puts `MaxTokensGiven` in a tag that already adds exported names (C123).

### What option A is, as the owner decided it

* **A1. When it applies.** Only when the request's `MaxOutputTokens` is 0 and
  no `WithMaxTokens` was given (`Settings.MaxTokensGiven`, OQ5). A caller's
  value is sent as today (ST15 to ST18). A listing probe's 8192 counts as
  given (ST26).
* **A2. The value R.** With L the facts' stated, positive `MaxOutputTokens`
  and W the facts' stated window: R is min(L, the ceiling, W/2 where W is
  stated), and never below today's value min(8192, L). The ceiling is
  32,000 on `Stream` and 16,384 on `Generate` (OQ2). R is sent only when it
  is above today's value; otherwise today's value is sent, so a stated L of
  8192 or less behaves as today (ST14), and a stated 0 is not a limit
  (ST13). On chatcompat and Ollama, L counts only when a caller's layer
  states it (OQ7).
* **A3. Unknown limit.** 8192, as today, with reasoning on or off, and
  facts not yet cached are unknown (OQ4; ST54). The docs say a caller
  supplies the fact through `catalog.WithModelLayer`.
* **A4. Refusals the raise can cause (OQ3).** A raised request refused as an
  overflow whose message does not show the input alone over the window, in
  Anthropic's limit form (C40), or in Kilo's documented window form, which
  becomes an overflow form (C97), is built again with today's value and sent
  once more. Nothing else is resent for the limit: not a tool-schema
  refusal (0084), a rate limit, an outage, or a request whose limit was not
  raised. A refusal of the resend is returned as today. Nothing is
  remembered between requests.
* **A5. How the resend runs (OQ6).** Before any event is yielded, on
  `Generate` and on `Stream`, including a stream whose first payload is an
  error inside a 200. Once a limit or overflow refusal has undone the raise,
  every later build of that request uses today's value; an effort or shape
  refusal keeps the raise. The send that undoes the raise sits outside
  0062-MADR's bound, so a raised request may make eight sends.
* **A6. Every wire that sends a limit.** Claude, OpenAI with a key, Grok,
  Gemini, Kilo, Together, Hugging Face, Ollama, chatcompat and every OpenCode
  route; the ChatGPT backend and plan still send none (ST4). On Messages with
  the budget shape, R is the `max_tokens` `AddThinkingFor` starts from,
  unchanged otherwise (ST2, ST30).
* **A7. The signal (OQ5).** Exported: `Settings.MaxTokensGiven() bool`,
  after `HTTPClientGiven` and `LoggerGiven` (C3, C111). `WithMaxTokens` sets
  a flag beside `MaxTokens` in `providerConfig` in its own closure, since
  the default is not zero and cannot be derived as `clientGiven` is (C110).
  Its call sites are the ten built-in providers' constructors; a provider
  outside this module may read it. R55's cost is accepted by the owner's
  answer: no outside call site is named.
* **A8. Docs.** Every line that says the default is never raised: the eight
  package docs (C7 and the seven at `claude.go:117`, `gemini.go:79`,
  `grok.go:94`, `huggingface.go:78`, `ollama.go:80`, `openai.go:110`,
  `together.go:71`), the ten inline comments (C6, C8 to C16),
  `docs/guides/api-standards.md:103` (C112), the doc comment of
  `TestOutputLimit_Facts_0041` (C113), `README.md:199` (C49),
  `docs/architecture.md:361` (C50), and the doc comments of
  `Settings.MaxTokens`, `WithMaxTokens` and `Request.MaxOutputTokens`, say
  what the code does (ST44); the Claude package doc states the reservation
  (ST38) and the `Generate` header wait (ST24).
* **A9. What it supersedes.** 0041-MADR D2 R7's "the default is never
  raised", and the owner's Q3 ruling "cut only", for the case A2 covers
  (ST31). The cut stays. By OQ6's answer, 0062-MADR Q5 (a)'s bound of seven
  is amended for a raised request (ST57). Both records carry a dated
  annotation of 2026-11-21 at those lines.
* **A10. The release (OQ8).** This tag, built last, after 0087; its scope
  becomes six records (ST43).

### Consequences

* Good, because the reported failure cannot recur on a model whose facts
  state its limit: at high effort a stream has up to 32,000 tokens, four
  times today's; gobble's model is raised once its cache holds Kilo's
  listing (ST33).
* Good, because no request is sent more than its model's stated limit, and a
  caller's limit is never touched.
* Good, because with A2's window fit and A4's resend, a turn that fits at
  8192 still succeeds on a strict service, in every refusal form the SDK
  recognises.
* Neutral, because billing follows tokens used where it is documented (C65).
* Bad, because on Claude every streamed request that names no limit reserves
  up to 32,000 of the output-per-minute limit instead of 8192, so a busy
  caller meets 429 sooner.
* Bad, because a `Generate` answer longer than 300 s of generation fails at
  the header wait where 8192 would have cut it short; at 16,384 that needs a
  rate below about 55 tokens per second.
* Bad, because a refusal in a form the SDK does not recognise (a routed
  upstream's, a local server's) is returned where 8192 would have been
  accepted (ST50), and a model whose facts overstate its limit pays one
  refused send on every raised request.
* Bad, because the first request before Kilo's listing is cached still sends
  8192 (ST54), and a model the caches know nothing of keeps today's failure
  (ST12).
* Bad, because the change touches a shared stream path (ST55), adds two
  refusal forms (ST49, ST51), and amends 0062's bound (ST57).
* Bad, because an existing test that pins 8192 for a 128,000-token model
  changes (C43), with the record that reverses the owner's earlier ruling.
* Bad, because one exported name is added whose only call sites are in this
  module (R55's cost, accepted by OQ5's answer), and removing it after the
  tag would take a new record.
* Bad, because this tag's scope becomes six and the tag waits for this
  record's phases and review (OQ8's answer).

### Confirmation

* The PLAN's tests, one per decided row, red at the base where a row
  changes behaviour and green after, each with a plant caught on a scratch
  clone.
* `python3 -B tools/apidiff.py v1.31.0`, the newest tag at Phase 1's base,
  lists `Settings.MaxTokensGiven` beside the names `CHANGELOG.md`'s
  `[Unreleased]` already carries, such as `ErrToolSchemaRefused`,
  `APIError.Tool`, the new `wizard.Options` fields and `signin.VerifyKilo`
  (C123), and nothing else.
* The independent review of the PLAN's Phase 3 finds no blocker.

## Owner Questions

Eight questions, each with three options, the recommended one first, as
they were put to the owner. Each names its cost and the rows its answer
decides. All eight were answered on 2026-11-21; the answers are under
"Owner rulings" above, and the state table records them.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ1. Which option? | (1) **A**: the facts' limit when the caller names none, else 8192. (2) **B**: raise only when reasoning is requested, by effort, cut to the facts' limit: low 8192, medium 16,384, high 32,768, xhigh and max 65,536. (3) **C**: a global default of 32,000, cut to the facts' limit. | (1) changes the body of every provider whose facts state a limit above 8192, and needs OQ2 to OQ7; reverses 0041 Q3 for that case; changes `TestOutputLimit_Facts_0041`'s 128,000 case (C43). (2) a table of numbers no measurement supports; reasoning-off requests unchanged; an unknown limit with reasoning on can be refused (C40); same test change. (3) every request on a model whose limit is unknown and below 32,000 is refused (C40); four times the Claude output reservation on every request; OQ3's window refusals for every model; the default-value tests change (C44, C45). | ST1 to ST3, ST5 to ST8, ST10, ST11, ST25, ST28, ST29, ST31, ST32, ST33 |
| OQ2. The ceiling, on `Stream` and `Generate` | (1) **32,000 on `Stream`, 16,384 on `Generate`**. (2) **32,000 on both**, kilocode's default (C59, C64). (3) **No ceiling**: the facts' limit as stated. | (1) two values to document and test; 16,384 is not measured, only the header-wait arithmetic supports it (more than 300 s below about 55 tokens per second); a `Generate` at high effort has half a stream's room; up to four times today's Claude reservation on streams. (2) 32,000 tokens take more than 300 s below about 107 tokens per second (C41); the timeout is `ErrProviderUnavailable`, which `WithRetry` retries once (C92; 0028-MADR D-H1), so a long `Generate` answer can fail after two waits, each possibly billed. (3) Kilo's stated limits reach 1,800,000 (C124) and 82 of 383 equal 90% of their window (C125): without OQ3 (1)'s fit, most raised Kilo requests exceed the window; up to sixteen times today's Claude reservation; the header-wait risk largest. | ST1, ST3, ST5 to ST8, ST10, ST24, ST25, ST32, ST33, ST38 |
| OQ3. Refusals the raise can cause: the window, and a limit the service will not take | (1) **Fit and resend once**: when the facts state a window W, R is held to W/2, never below today's value; a raised request refused as an overflow that does not show the input alone over the window, in Anthropic's limit form (C40), or in Kilo's documented window form (C97, added as an overflow form) is sent once more at today's value. (2) **Resend once, no fit**: the same resend on the same forms, R not held to the window. (3) **Return the refusal**: no fit, no resend; the docs tell a caller to pass `WithMaxTokens`. | (1) a window rule in the shared helper; one more send on a raised request refused in a recognised form; two forms parsed, so Kilo's documented form becomes `ErrContextOverflow` for every Kilo request, raised or not, where it is `ErrInvalidRequest` today; a refusal in an unrecorded form (a routed upstream's, a local server's) is returned; nothing is remembered, so overstated facts cost one refused send per raised request. (2) on the eight Kilo models whose stated limit leaves under 16,384 tokens of window (two leave 1,639) and models.dev's `google/gemini-2.5-flash-image` (768 left), most turns pay a refused send first; otherwise as (1). (3) a regression on strict services: a turn that fits at 8192 is refused (C38, C39, C97), a caller that compacts on `ErrContextOverflow` compacts for nothing, and overstated facts are refused as `ErrInvalidRequest`. | ST21, ST22, ST23, ST47 to ST52 |
| OQ4. The facts state no limit, or are not yet cached | (1) **8192, as today**; the docs point to `catalog.WithModelLayer`. (2) **On `Generate` only, one resend at the ceiling after an empty `length` answer** of a request whose 8192 the SDK chose. (3) **The ceiling when reasoning is requested.** | (1) a model the caches know nothing of keeps today's failure at high effort; gobble's model does not rest on this answer once its listing is cached (ST33). (2) the first 8192 tokens are billed before the resend; impossible on `Stream`, whose reasoning deltas were already yielded, so gobble, which streams (C91), gains nothing; the resend can be refused by a model whose unknown limit is lower (then returned, or resent under OQ3's forms). (3) a model whose unknown limit is lower is refused on every reasoning request (C40) unless OQ3's resend catches the form; a 32,000 Claude reservation on every reasoning request with unknown facts. | ST12, ST34, ST60 |
| OQ5. How a provider tells a caller's `WithMaxTokens` from the default | (1) **Unexported**: `WithMaxTokens` sets a flag beside `MaxTokens` in `providerConfig`, read by the providers through a new internal package that `llmprovider` fills. (2) **Exported `Settings.MaxTokensGiven() bool`**, as `HTTPClientGiven` and `LoggerGiven` (C3, C111). (3) **`MaxTokens()` returns 0 when none was given.** | (1) no API change; one internal package and the code that fills it (no such bridge exists today: every `Settings` accessor at `settings.go:303-377` is exported); a provider outside this module cannot tell, and keeps 8192; meets R55 (C109), since every call site is in this module and none outside is named (the owner is the only user). (2) one exported name and a minor version; R55 asks for the call site that needs it, and the ten built-in providers are served by (1). (3) changes a documented value (C2) every provider and caller reads; the default tests change (C44, C45). | ST35, ST36, ST37 |
| OQ6. How the resend runs (moot under OQ3 (3)) | (1) **Streams too, sticky, one send beyond the bound**: `StreamEventsRetry` also resends when a stream's first payload is an error and nothing was yielded; once a limit or overflow refusal undoes the raise, every later build of that request uses today's value, while an effort or shape refusal keeps the raise; the undoing send sits outside 0062's bound of seven. (2) **Streams too, sticky, inside the bound of seven.** (3) **No stream change, sticky, inside the bound**: the resend runs on `Generate` and when a stream fails to open. | (1) a change to a wire path every streaming provider shares; amends 0062-MADR Q5 (a), an owner ruling, to eight sends for a raised request. (2) the shared stream change; a raised request refused for its limit whose effort ladder then needs all seven rungs fails where it succeeds today. (3) a stream whose overflow arrives inside a 200 (C39's form, C89) gets no resend, and gobble streams through `WithStreamRetry` (C90, C91), so a strict service's refusal stays a regression for it; and (2)'s starved ladder. | ST55, ST56, ST57 |
| OQ7. Local servers: chatcompat and Ollama | (1) **Raise only from a caller's layer**, never from models.dev's canonical facts (by `SourceOf`, C103). (2) **Any fact**, as on every other provider. (3) **Never raise on a local server.** | (1) neither local listing states an output limit (C104 to C106 are the only ones), so in practice only a caller's layer raises; one source check. (2) models.dev's hosted limit, by canonical id with quantisation dropped (C102), is sent to a server whose own context may be far smaller; whether a local server refuses, cuts or silently shrinks the input is not established, and a silent shrink gives OQ3's resend nothing to act on. (3) a caller who states the limit in a layer still gets 8192; one more exception to document. | ST9 |
| OQ8. Which release? A fact first: `[Unreleased]` at 55a5dca also carries 0080's and 0082's entries (C120, C121); no commit of either is in `v1.31.0`, the newest tag (C122), so both are untagged. The owner's scope of five (0081, 0083, 0084, 0085, 0087) is not changed by this record. | (1) **Its own release right after this tag.** (2) **Join this tag**, built last, after 0087. (3) **The next release whenever it comes**, with other records. | (1) gobble waits for a second tag; this tag keeps the owner's five records. (2) the tag's scope becomes six and the tag waits for this record's phases and review; its changelog entry joins the others under `[Unreleased]`. (3) gobble's failure stays until that release. | ST43 |

## State Table

"Decided" means decided by an existing record, by code that no option
changes, by every option a question offered, or by the owner's answer of
2026-11-21, quoted by its label, named in the row. R is A2's value. Each
decided row names the test that pins it and the plant that must turn it
red, from the PLAN.

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST1 | Claude, Messages, no thinking or adaptive thinking; no caller limit; the facts state L above 8192 | decided | `max_tokens` R: min(L, 32,000) on `Stream`, min(L, 16,384) on `Generate`, held to W/2 where W is stated, never below today's value | OQ1 "The model's stated limit (Recommended)"; OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_Facts_0088`; plant: the raise removed, so L 20,000 sends 8192 |
| ST2 | Claude and OpenCode's messages route, the budget thinking shape; same | decided | `AddThinkingFor` starts from R; the budget and its raise rule unchanged (ST30) | OQ1 "The model's stated limit (Recommended)" | `TestDefaultLimit_BudgetShape_0088`; plants: today's 8192 passed to `AddThinkingFor` (budget 4096 sends 8192); `max_tokens` set to R after `AddThinkingFor` (budget 40,000 sends 32,000) |
| ST3 | OpenAI with an API key, Responses; same | decided | `max_output_tokens` R | OQ1 "The model's stated limit (Recommended)"; OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_Facts_0088`; plant: the `Stream` ceiling removed, so L 128,000 is sent |
| ST4 | OpenAI's ChatGPT backend and the ChatGPT plan | decided | no `max_output_tokens`, whatever the facts or the caller say, as today | `openai.go` (C17); 0043-MADR D5 | `TestOpenAI_ChatGPTOmitsMaxOutputTokens`, `TestPlan_Request_0043` (existing); plant: the plan branch sets `max_output_tokens` |
| ST5 | Grok, Responses; same | decided | as ST3 | OQ1 "The model's stated limit (Recommended)"; OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_Facts_0088`; plant: as ST3 |
| ST6 | Gemini, Interactions; same | decided | `generation_config.max_output_tokens` R | OQ1 "The model's stated limit (Recommended)"; OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_Facts_0088`; plant: `Generate` given the `Stream` ceiling, so 32,000 is sent, not 16,384 |
| ST7 | Kilo, Chat Completions (the reported case); same | decided | `max_tokens` R; with no stated L, ST12 | OQ1 "The model's stated limit (Recommended)"; OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_Facts_0088`; plant: Kilo's body keeps the old cut, so L 20,000 sends 8192 |
| ST8 | Together and Hugging Face, Chat Completions; L from a caller's layer or models.dev (their listings state none) | decided | as ST7 | OQ1 "The model's stated limit (Recommended)"; OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_Facts_0088`; plant: `Stream` given the `Generate` ceiling, so 16,384 is sent, not 32,000 |
| ST9 | Ollama and chatcompat: L from a caller's layer, or from models.dev's facts of the canonical id (C102) | decided | R from a caller's layer; today's value when L's source is models.dev | OQ7 "Only from a caller layer (Recommended)" | `TestDefaultLimit_LocalFacts_0088`; plant: the source check removed, so the models.dev case sends 32,000 |
| ST10 | OpenCode Zen and Go: the chat, responses, messages and google routes | decided | each route's field R | OQ1 "The model's stated limit (Recommended)"; OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_Facts_0088`, the OpenCode Zen and Go pairs; plant: OpenCode's `resolve` keeps the old cut |
| ST11 | The facts state no limit; reasoning off | decided | 8192, as today | OQ1 "The model's stated limit (Recommended)" | `TestDefaultLimit_UnknownLimit_0088`; plant: an unknown limit raised to the ceiling |
| ST12 | The facts state no limit; reasoning on, at any level | decided | 8192, as today | OQ4 "Keep 8,192 (Recommended)" | `TestDefaultLimit_UnknownLimit_0088`, at high; plant: as ST11 |
| ST13 | The facts state a limit of 0 (an image or video model, 0041-MADR A1) | decided | not a limit: the request is sent as if none were stated | 0041-MADR A1; the `limit > 0` guard (C6) | `TestDefaultLimit_StatedZero_0088`; plant: the guard dropped, so 0 is sent |
| ST14 | The facts state L of 8192 or less; no caller limit | decided | min(8192, L), as today; not counted as raised | 0041-MADR D2 R7 (the cut stays) | `TestOutputLimit_Facts_0041`, the 4096 case (existing); `TestDefaultLimit_Facts_0088`, the 8192 case; plant: one provider's cut removed, so 8192 is sent for 4096 |
| ST15 | `WithMaxTokens(n)`, the request's `MaxOutputTokens` 0 | decided | n, cut to L when L is smaller, never raised, as today | `WithMaxTokens`'s doc (C4); 0041-MADR D2; every option's definition | `TestDefaultLimit_CallerGiven_0088`; plant: the raise applied whatever the caller gave |
| ST16 | `WithMaxTokens(8192)`, the default's own number | decided | 8192, not raised: a given value is given | ST15 | `TestDefaultLimit_CallerGiven_0088`, case 8192; plant: "given" read as "differs from 8192" |
| ST17 | The request's `MaxOutputTokens` above 0 | decided | sent as given, neither cut nor raised, as today | `Request.MaxOutputTokens` (C5); each body function (C6) | `TestDefaultLimit_RequestGiven_0088`; plant: the request's value cut to L |
| ST18 | The request's `MaxOutputTokens` above the model's limit | decided | sent; the service's refusal (C40) is `ErrInvalidRequest`, not resent, as today | ST17 | `TestDefaultLimit_RequestGiven_0088`, case above L; plant: as ST17 |
| ST19 | The request's `MaxOutputTokens` negative | decided | `ErrInvalidRequest` before any send, as today | `Request.validate` (C51) | `TestCapabilitiesCheck_RefusesInvalidValues` (existing); plant: the check removed |
| ST20 | `WithMaxTokens(0)` or a negative value | decided | `New` refuses it with `ErrInvalidRequest`, as today | `settings.finish` (C52); 0020-MADR F29 | `TestResolveOptions_RefusesANonPositiveMaxTokens` (existing); plant: the check removed |
| ST21 | R plus the input over the window, on a strict service (C38, C39, C97) | decided | one resend with today's value; a refusal of the resend returned as today | OQ3 "Fit to window, resend once (Recommended)" | `TestDefaultLimit_OverflowResend_0088`, cases 1 and 2; plants: the resend removed; a second resend allowed |
| ST22 | R alone at or above the window | decided | prevented by the W/2 fit where W is stated; otherwise the "fills the context window" refusal (C35) is resent once | OQ3 "Fit to window, resend once (Recommended)" | `TestDefaultLimit_WindowFit_0088` and `TestDefaultLimit_OverflowResend_0088`'s completion-fills form; plants: the W/2 hold removed; the resend removed |
| ST23 | Claude: the input plus R over the window | decided | on Claude 4.5 and later the service stops at the window with `model_context_window_exceeded`, which is `FinishLength` (C29), and no resend runs; Claude's "prompt is too long" names the input alone and is not resent (ST48) | OQ3 "Fit to window, resend once (Recommended)" | `TestDefaultLimit_ClaudeWindowStop_0088`; plants: a resend on `FinishLength`; the input-alone check removed |
| ST24 | `Generate`, a non-streaming request | decided | R with a ceiling of 16,384; the 300 s header wait (C41) and `WithRetry`'s one retry of it (C92) stated in the docs | OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_Facts_0088`, the `Generate` cases; plant: `Generate` given the `Stream` ceiling |
| ST25 | `Stream` | decided | R with a ceiling of 32,000 | OQ1 "The model's stated limit (Recommended)"; OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_Facts_0088`, the `Stream` cases; plant: `Stream` given the `Generate` ceiling |
| ST26 | A listing probe | decided | the probe's 8192 counts as given: sent as today, never raised, whatever the facts state | 0015-MADR D1, "kept so the probe's request is unchanged" (C108); the probe runs through `Generate` on a copy (C107) | `TestDefaultLimit_ListingProbe_0088`; plant: the probe copy's limit not marked given, so 32,000 is sent |
| ST27 | The answer still ends `length` with no text | decided | `ErrEmptyAnswer`, its `Reason` the finish reason, unchanged | 0026-MADR F7; 0061-MADR D1; C30, C31 | `TestDecode_EmptyIsErrEmptyAnswer_0061` on each wire (existing); plant: the Chat Completions empty check removed |
| ST28 | Reasoning off | decided | the same value as with reasoning on | OQ1 "The model's stated limit (Recommended)" | `TestDefaultLimit_Facts_0088`, reasoning off; plant: the raise only when reasoning is requested |
| ST29 | Reasoning on, at low, medium, high, xhigh or max | decided | the same value at every level | OQ1 "The model's stated limit (Recommended)" | `TestDefaultLimit_Facts_0088`, each level; plant: the raise only at high and above, so the low case sends 8192 |
| ST30 | Messages, the budget shape: the raise of `max_tokens` above the budget | decided | unchanged: raised to the budget plus 4096 only when not above it, and never past the facts' limit | 0041-MADR D2 R3, R7; 0062-MADR D3; C26, C27 | `TestAddThinkingFor_0041` (existing); plant: the raise removed |
| ST31 | 0041-MADR D2 R7 ("the default is never raised") and the owner's Q3 ruling "cut only" | decided | superseded for A2's case by this record; 0041-MADR carries a dated annotation beside R7 and beside its Q3 ruling (2026-11-21); the cut stays | OQ1 "The model's stated limit (Recommended)" | a `tools/claims.py` row on each annotation once committed (PLAN Phase 1 step 6), planted false by `tools/claims.py --plants` |
| ST32 | The existing tests that pin 8192: a 128,000-token model's request (C43) and the default's value (C44, C45) | decided | the 128,000 case, sent by `Generate`, expects 16,384, a change planned and approved with this record; `Settings.MaxTokens()` still returns 8192, so C44 and C45 stand | OQ1 "The model's stated limit (Recommended)"; OQ2 "32k streaming, 16k plain (Recommended)" | `TestOutputLimit_Facts_0041`, changed; plant: the raise removed, so 8192 is sent |
| ST33 | gobble's case: Kilo, `kilo-auto/balanced`, metadata off, the provider's cache holding Kilo's listing (C77 to C82) | decided | the listing's 65,536 is L, so a stream sends 32,000 | OQ1 "The model's stated limit (Recommended)"; OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_KiloListing_0088`, the warm case; plant: the helper raises only when `SourceOf` names a caller's layer |
| ST34 | A Kilo auto-routing model whose listing entry carries no output cap (as in the 2026-09-26 snapshot) | decided | 8192 when the facts state none | OQ4 "Keep 8,192 (Recommended)" | `TestDefaultLimit_UnknownLimit_0088`; plant: as ST11 |
| ST35 | How a provider tells a caller's `WithMaxTokens` from the default | decided | exported `Settings.MaxTokensGiven() bool`, true when `WithMaxTokens` was given; the flag sits beside `MaxTokens` in `providerConfig`, set by `WithMaxTokens`'s own closure, so a `For(id, WithMaxTokens(n))` sets it for id only | OQ5 "Export MaxTokensGiven" | `TestSettings_MaxTokensGiven_0088`; plants: `WithMaxTokens` does not set the flag; the flag derived from `MaxTokens != 8192` |
| ST36 | A provider outside this module, built on `Settings` | decided | it may read `MaxTokensGiven` and apply the rule; until it does, it sends what it sent | OQ5 "Export MaxTokensGiven" | `TestSettings_MaxTokensGiven_0088`, in the external package `llmprovider_test`, as an outside provider reads it; plant: as ST35 |
| ST37 | The exported API, and `apidiff` against the newest tag at Phase 1's base | decided | `apidiff` against `v1.31.0`, the newest tag at Phase 1's base, lists `Settings.MaxTokensGiven` beside the names `[Unreleased]` already carries (C123); nothing removed or changed. R55's cost, an export whose only call sites are in this module with no outside call site named, is accepted by the owner's answer | OQ5 "Export MaxTokensGiven" | `make api-check` and `python3 -B tools/apidiff.py v1.31.0` (PLAN Phase 1 step 7); plant: a second exported name added on a scratch clone, which `apidiff` reports beyond the expected list |
| ST38 | Anthropic's output-per-minute reservation | decided | at most 32,000 per streamed request and 16,384 per `Generate` that names no limit, stated in the Claude package doc | OQ2 "32k streaming, 16k plain (Recommended)" | `TestDefaultLimit_Facts_0088` (the values) and a `tools/claims.py` row on the doc sentence, planted false; plant: the `Stream` ceiling removed |
| ST39 | Billing | out of scope | the service's, on tokens used where documented (C65); no local source shows a charge for an unused limit; stated as a cost | | |
| ST40 | The other services' rate limits | out of scope | not established in any local source; stated under "What could not be established"; Kilo's gateway does not rate-limit paid requests (C66) | | |
| ST41 | 0084: a refusal's error body | decided | read up to 1 MiB and classified as today; any form OQ3 adds is matched within the same read | 0084-MADR ST86 (C54) | `TestClassify_ErrorBodyLimit_0084` (existing); plant: 0084's own, the body limit back to 64 KiB |
| ST42 | 0081, 0083, 0085 and 0087 | out of scope | they change the wizard, the sign-in, and the key checks; none changes a line that computes or sends an output limit; re-checked at the PLAN's Phase 1 start | | |
| ST43 | The changelog entry and the release | decided | a **Changed** entry (the default, Kilo's window form) and an **Added** entry (`Settings.MaxTokensGiven`) under `[Unreleased]`, released in this tag, whose scope becomes six (0081, 0083, 0084, 0085, 0087, 0088); 0088 is built last, after 0087, and the tag waits for 0088's phases and review | OQ8 "Join this tag" | `make changelog-check`, and a `tools/claims.py` row on each entry, planted false |
| ST44 | The doc lines A8 names | decided | each says what the code does, in the same change | AGENTS.md "API work"; docs/guides/api-standards.md | a `tools/claims.py` row on each new sentence, planted false by `tools/claims.py --plants`; `make docs-check` |
| ST45 | Option D on Kilo: a reasoning budget, `reasoning: {max_tokens}`, below the limit (C32, C67) | decided | not sent; Kilo's reasoning object unchanged | no OQ1 option sends a reasoning budget; C32 | `TestGenerate_Reasoning` (C115) and `TestKilo_ReasoningEffortConfigured` (C116), existing, which compare the whole object; plant: Kilo's reasoning object given a `max_tokens` |
| ST46 | Concurrent requests on one provider, some refused and resent | decided | every value the raise and any resend set lives in the request's own build, never on the provider; each request's sends are its own | 0048-MADR R20: a provider is immutable and safe for concurrent use after `New` (C114) | `TestDefaultLimit_Concurrent_0088`, under `-race`; plant: the undone raise stored on the provider |
| ST47 | The facts state a window W, and min(L, ceiling) is above W/2 | decided | R held to W/2, never below today's value | OQ3 "Fit to window, resend once (Recommended)" | `TestDefaultLimit_WindowFit_0088`; plants: the W/2 hold removed; the floor at today's value removed |
| ST48 | An overflow whose message shows the input alone over the window: "prompt is too long", xAI's and Together's input forms (C99 to C101), or a parsed split whose input part is at or above the window | decided | not resent; returned as today | OQ3 "Fit to window, resend once (Recommended)" | `TestDefaultLimit_OverflowResend_0088`, cases 5 and 6; plant: the input-alone check removed |
| ST49 | The facts state a limit above what the service takes (stale facts, a caller's layer), refused in Anthropic's form `max_tokens: N > M` (C40) | decided | one resend with today's value | OQ3 "Fit to window, resend once (Recommended)" | `TestDefaultLimit_LimitRefusal_0088`; plant: the limit form not recognised |
| ST50 | A gateway alias whose routed upstream takes less than the alias's stated limit | decided | a refusal in a recognised form is resent once; a cut answer (`FinishLength` with text) is returned as today; a refusal in an unrecorded form is returned; which Kilo does was not established | OQ3 "Fit to window, resend once (Recommended)" | `TestDefaultLimit_LimitRefusal_0088`, the Kilo `length` case; plant: a resend on `FinishLength` |
| ST51 | Kilo's documented window form (C97), today `ErrInvalidRequest` | decided | added as an overflow form, so it is `ErrContextOverflow` for every Kilo request, and a raised request so refused is resent once | OQ3 "Fit to window, resend once (Recommended)" | `TestClassify_KiloWindowForm_0088`; plant: the form removed *(2026-11-23, ninth amendment: with two planned changes, a sample in `overflowSamples` and the form in the frozen reference, by the owner's ruling "Name the two test changes (Recommended)")* |
| ST52 | The resend's value, with the budget shape | decided | the resend starts from today's configured value; `AddThinkingFor` applies after it, unchanged, so a budget at or above 8192 still raises `max_tokens` above it (C26) | OQ3 "Fit to window, resend once (Recommended)" | `TestDefaultLimit_ResendBudgetShape_0088`; plants: the resend's `max_tokens` set to 8192 after `AddThinkingFor`; the resend built without the thinking fields |
| ST53 | A balance or credit refusal of a raised request | out of scope | Kilo's 402 at a zero balance (C95) is `ErrQuotaExhausted` (C94), not resent; an upstream's 402 reaches the client as 503 (C96), `ErrProviderUnavailable` (C98), which `WithRetry` may retry and the resend does not; a refusal that depends on the limit's size was not found in any local source; stated as a cost | | |
| ST54 | The facts are not yet cached when the request is built (a cold cache, a listing fetch still running) | decided | today's value, 8192; the request never waits and never fetches | 0041-MADR D1, "No request waits, and no request lists" (C85) | `TestDefaultLimit_KiloListing_0088`, the cold case; plant: the provider reads its facts by `catalog.LookupModel`, which fetches, so 32,000 is sent and the listing is read during the request |
| ST55 | A raised `Stream` whose refusal arrives inside a 200, as its first payload | decided | resent once at today's value when nothing was yielded; after a yielded event, the error is yielded as today | OQ6 "Streams too, one extra send (Recommended)" | `TestStreamEventsRetry_FirstPayloadError_0088`; plants: the first-payload resend removed; the resend allowed after a yielded event |
| ST56 | A later build of a request whose raise a limit or overflow refusal undid | decided | today's value for every later build of that request; an effort or shape refusal before it keeps R | OQ6 "Streams too, one extra send (Recommended)" | `TestDefaultLimit_ResendLadder_0088`, cases 1 and 2; plants: the undo not sticky; the raise undone after any refusal |
| ST57 | The bound of seven sends (0062-MADR Q5 (a)) | decided | the send that undoes the raise is not counted, so a raised request makes at most eight; 0062-MADR carries a dated annotation beside Q5 (2026-11-21) | OQ6 "Streams too, one extra send (Recommended)" | `TestDefaultLimit_ResendLadder_0088`, case 3; plant: the undoing send counted against the bound |
| ST58 | gobble's lookup and its provider resolving to the same listing key (C83) | out of scope | gobble-cli's to show in its own records; stated under "What could not be established" | | |
| ST59 | A raised request refused for anything but A4's forms: a 0084 tool-schema refusal, any other `ErrInvalidRequest`, a rate limit, an outage | decided | not resent for the limit; returned, or retried, as today | no OQ3 option resends it: each names its forms; 0084-MADR D1's kind | `TestDefaultLimit_OverflowResend_0088`, the negative cases; plant: the resend runs on any error of a raised request |
| ST60 | `Generate` with no stated limit ends `length` with no text | decided | `ErrEmptyAnswer`, one send, as today (ST27) | OQ4 "Keep 8,192 (Recommended)" | `TestDefaultLimit_UnknownLimit_0088`, the empty case; plant: a resend at the ceiling after an empty `length` answer |
| ST61 | An overflow of a request whose limit was not raised (unknown limit, a caller's limit, a stated L of 8192 or less) | decided | one send, as today | no OQ3 option resends a request the raise did not set | `TestKilo_ErrorIn200IsNotRetried` (C118) and `TestTogether_TypedErrorInside200SentOnce` (C119), existing, which stand because their models have no stated limit; `TestDefaultLimit_OverflowResend_0088`, the caller and 8192 cases; plant: the resend runs on any overflow, raised or not |

Counts: 56 decided, 5 out of scope, 0 open, 0 awaiting live evidence.
*(2026-11-22: with the research amendment's ST62 to ST74, 60 decided, 8 out of scope, 6 open, 74 rows.)* *(2026-11-22, second: with ST75 to ST89 and the owner's answers to OQ10 to OQ12, 74 decided, 8 out of scope, 7 open, 89 rows.)* *(2026-11-22, third: 84 decided, 8 out of scope, 10 open, 1 replaced, 103 rows.)*

## Pros and Cons of the Options

### A. The facts' limit when the caller names none

* Good, because the number is the model's own, so it is refused only where
  the facts overstate it (the 0041 cut, used the other way).
* Good, because it fixes every wire and every effort where the facts know the
  limit, and matches what Kilo's and opencode's clients do (C60, C68).
* Good, because an unknown model and a caller's limit behave as today.
* Neutral, because the fix reaches a model only through its facts: a listing,
  models.dev, or a caller's layer.
* Bad, because it raises the Claude output reservation and the non-streaming
  header-wait risk (OQ2 bounds both), and needs a rule for refusals the
  raise causes (OQ3, OQ6).
* Bad, because it reverses the owner's 0041 Q3 ruling for this case, and
  changes one existing test.

### B. Raise only with reasoning, by effort

* Good, because requests without reasoning keep today's limit and reservation.
* Good, because a higher effort, which spends more reasoning, gets more room.
* Bad, because the per-level numbers have no measurement behind them, and no
  wire documents how much reasoning an effort spends.
* Bad, because with no stated limit, a raise can exceed the model's limit and
  be refused (C40); with one, it is A with extra numbers.
* Bad, because a request with reasoning off can still be cut short on a large
  answer, as today.

### C. A global default of 32,000

* Good, because it is one number, matching kilocode's default (C59) and
  opencode's fallback (C68).
* Bad, because a model whose limit is below 32,000 and unknown is refused on
  every request that names none, where today's 8192 is accepted.
* Bad, because every Claude request reserves 32,000 whether or not it reasons.
* Bad, because the window refusals of OQ3 happen for every model, not only
  those whose facts are known, and no window fit is possible without facts.

### D. A reasoning budget below the limit

* Good, because the answer keeps room however long the reasoning runs.
* Bad, because few wires take a budget: Messages takes one only in the budget
  shape, which the newest Claude models refuse (Anthropic's reference); the
  Responses and Interactions wires take an effort only; Chat Completions has
  no standard budget, and Kilo's `reasoning: {max_tokens}` (C67) replaces
  the effort the SDK sends today (C32).
* Bad, because inside 8192 a budget only splits too little room: at high
  effort, reasoning cut to a budget is the quality the caller asked not to
  lose.

### E. Send no limit when the caller names none

* Good, because the service's own default applies, as codex and grok-build
  do on Responses (C72, C73).
* Bad, because Messages requires `max_tokens` (C19), so Claude needs A anyway.
* Bad, because what Kilo, OpenRouter's upstreams, Together, Hugging Face,
  Ollama or a local server default to when none is sent was not established;
  the behaviour would differ by service, unseen by the SDK.
* Bad, because 0043-MADR D5 shows the opposite case already: the ChatGPT
  routes take none, and the other routes were never measured without one.

## More Information

### The costs, by wire

| Wire | Providers | Field | Reasoning inside the limit | A limit above the service's | Input plus limit over the window |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Chat Completions | Kilo, Together, Hugging Face, Ollama, chatcompat, OpenCode chat | `max_tokens` (C18) | yes: the reported failure; 0026-MADR F7 (C30) | not recorded for these services | refused by OpenAI-compatible services (C38, C39); Kilo documents a form the SDK does not yet match (C97); local servers not established |
| Responses | OpenAI with a key, Grok, OpenCode responses | `max_output_tokens` (C20, C21, C23); none on the ChatGPT routes (C17) | taken as so; not measured for this record | not established | not established; xAI names the input alone (C100) |
| Messages | Claude, OpenCode messages | `max_tokens`, required (C19, C24) | yes, thinking is output; a budget must be below `max_tokens` (C26, C27) | refused, `max_tokens: N > M` in the SDK's fixture (C40) | Claude 4.5 and later stop at the window, `model_context_window_exceeded` (C29); "prompt is too long" names the input alone (C99) |
| Interactions | Gemini | `generation_config.max_output_tokens` (C22) | not established | not established | not established |
| generateContent | OpenCode google | `maxOutputTokens` (C25) | thoughts are counted as output in its usage | not established | not established |

### What could not be established

* Whether gobble's model lookup and its provider resolve to the same Kilo
  listing key (C83: base URL, organization, credential fingerprint), so that
  the provider's request finds what the lookup's fetch cached. Looked in:
  gobble-cli's provider construction (C79, C80, C57) and model lookup (C81);
  both take gobble's SDK options and its credential, but the end-to-end key
  was not traced. gobble-cli's records own it (ST58).
* Whether OpenRouter, or any upstream behind Kilo, refuses a request whose
  limit the account's balance cannot cover. Looked in: kilocode's gateway
  docs, which state a 402 at a zero balance (C95) and an upstream 402 turned
  into 503 (C96), and no refusal that depends on the limit; the kilocode,
  opencode and codex checkouts, searched for "402", "afford", "more credits"
  and "insufficient credits". ST53 is out of scope and stated as a cost.
* Whether Kilo's gateway sends OpenRouter's overflow form (C39), its
  documented form (C97), or both, and whether a Kilo stream carries an
  overflow inside a 200 as its first payload. Looked in: the SDK's fixtures
  and kilocode's docs. ST51 and ST55 are written for both.
* Whether a gateway alias such as `kilo-auto/balanced` routes to an upstream
  whose limit is below the alias's stated one, and whether the gateway then
  cuts or refuses (ST50). Looked in: kilocode's docs and listing code.
* Whether Anthropic's service sends exactly the fixture's limit form
  `max_tokens: N > M` (C40). Looked in: the SDK's fixtures and records, and
  the bundled API reference, which does not quote the form.
* Whether OpenAI, xAI, Google, Together, Hugging Face or OpenCode count a
  request's limit, rather than tokens used, toward a rate limit. Looked in:
  the codex, opencode, kilocode, grok-build and claude-code checkouts (no
  statement found; opencode's tests carry a "Tokens per minute limit
  exceeded" message only), this repository's records and transport
  (`transport.go` reads Anthropic's output-token reset header only).
* Anthropic's counting of `max_tokens` toward its output-per-minute limit
  rests on the coordinator's brief; the API reference bundled with the
  drafting agent's tools names the limit but not the counting rule.
* Whether a service bills an answer whose client gave up at the header wait.
  Looked in: the checkouts' billing docs (Kilo bills tokens used, C65).
* Whether a local server (Ollama, llama.cpp, LM Studio) refuses, cuts, or
  silently shrinks the input when input plus `max_tokens` passes its
  context. Looked in: this repository's fixtures and records, which record
  llama.cpp's and LM Studio's overflow forms only.
* Whether Claude models before 4.5 refuse input plus `max_tokens` past the
  window. Looked in: the bundled API reference (it names the 4.5 stop reason
  only) and the claude-code changelog.

### Evidence index

<!-- claims: self=55a5dca gobble-cli=4c3e397 kilocode=58072bd696 opencode=4ce137b39e codex=57e64682f8 grok-build=2bdd1d6a -->

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C1 | the default output limit is 8192 | `llmprovider/settings.go:88` | `return settings{cfg: providerConfig{MaxTokens: 8192}}` |
| C2 | one number, given or default | `llmprovider/settings.go:320` | `MaxTokens is the output limit from WithMaxTokens, or 8192.` |
| C3 | the precedent for a given option | `llmprovider/settings.go:315` | `func (st *Settings) HTTPClientGiven() bool { return st.s.clientGiven }` |
| C4 | `WithMaxTokens` overwrites the default | `llmprovider/options.go:79` | `cfg.MaxTokens = n` |
| C5 | a request's zero is the provider's default | `llmprovider/contract.go:222` | `MaxOutputTokens bounds the output. Zero is the provider's default.` |
| C6 | Kilo cuts and never raises | `llmprovider/providers/kilo/kilo.go:377` | `The default, cut to the model's limit, never raised (0041-MADR D2).` |
| C7 | Kilo's package doc says so | `llmprovider/providers/kilo/kilo.go:83` | `model's own when that is smaller; it is never raised.` |
| C8 | Claude cuts and never raises | `llmprovider/providers/claude/claude.go:464` | `The default, cut to the model's limit, never raised (0041-MADR D2).` |
| C9 | Gemini likewise | `llmprovider/providers/gemini/gemini.go:304` | `The default, cut to the model's limit, never raised (0041-MADR D2).` |
| C10 | Grok likewise | `llmprovider/providers/grok/grok.go:366` | `The default, cut to the model's limit, never raised (0041-MADR D2).` |
| C11 | OpenAI likewise | `llmprovider/providers/openai/openai.go:510` | `The default, cut to the model's limit, never raised (0041-MADR D2).` |
| C12 | chatcompat likewise | `llmprovider/providers/chatcompat/chatcompat.go:351` | `The default, cut to the model's limit, never raised (0041-MADR D2).` |
| C13 | Hugging Face likewise | `llmprovider/providers/huggingface/huggingface.go:304` | `The default, cut to the model's limit, never raised (0041-MADR D2).` |
| C14 | Together likewise | `llmprovider/providers/together/together.go:301` | `The default, cut to the model's limit, never raised (0041-MADR D2).` |
| C15 | Ollama likewise | `llmprovider/providers/ollama/ollama.go:314` | `The default, cut to the model's limit, never raised (0041-MADR D2).` |
| C16 | OpenCode likewise | `llmprovider/providers/opencode/opencode.go:457` | `The default, cut to the model's limit, never raised (0041-MADR D2).` |
| C17 | the ChatGPT plan sends no limit | `llmprovider/providers/openai/openai.go:498` | `max_output_tokens, which it says to omit though M5 measured it` |
| C18 | Chat Completions sends `max_tokens` | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:420` | `keyMaxTokens: maxTokens,` |
| C19 | Claude sends `max_tokens` | `llmprovider/providers/claude/claude.go:475` | `"max_tokens": maxTokens,` |
| C20 | OpenAI with a key sends `max_output_tokens` | `llmprovider/providers/openai/openai.go:513` | `body["max_output_tokens"] = maxTokens` |
| C21 | Grok sends `max_output_tokens` | `llmprovider/providers/grok/grok.go:383` | `"max_output_tokens": maxTokens,` |
| C22 | Gemini sends it in its generation config | `llmprovider/providers/gemini/gemini.go:307` | `gen := map[string]any{"max_output_tokens": maxTokens}` |
| C23 | OpenCode's responses route | `llmprovider/providers/opencode/opencode.go:683` | `"max_output_tokens": c.maxTokens,` |
| C24 | OpenCode's messages route | `llmprovider/providers/opencode/opencode.go:742` | `body[jsonKeyMaxTokens] = maxTokens` |
| C25 | OpenCode's google route | `llmprovider/providers/opencode/opencode.go:785` | `genCfg := map[string]any{"maxOutputTokens": c.maxTokens}` |
| C26 | the budget shape raises `max_tokens` when needed | `llmprovider/internal/wire/messages/messages.go:646` | `if maxTokens <= budget {` |
| C27 | to the budget plus the default budget | `llmprovider/internal/wire/messages/messages.go:647` | `maxTokens = budget + defaultThinkingBudget` |
| C28 | the default budget | `llmprovider/internal/wire/wire.go:53` | `const DefaultThinkingBudget = 4096` |
| C29 | reaching the window is a cut answer on Messages | `llmprovider/internal/wire/messages/messages.go:39` | `"model_context_window_exceeded": llmprovider.FinishLength,` |
| C30 | an answer cut while reasoning is empty | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:544` | `if finish == llmprovider.FinishLength && text == "" {` |
| C31 | the empty answer's error | `llmprovider/internal/wire/finish.go:34` | `func EmptyAnswer(where string, reason llmprovider.FinishReason) error {` |
| C32 | Kilo's reasoning object | `llmprovider/providers/kilo/kilo.go:415` | `{effort} with an effort, else {enabled: true}` |
| C33 | Kilo's listing states an output cap *(2026-11-23: moved at `6ace750`; see C33now)* | `llmprovider/catalog/listing_metadata.go:297` | `if e.TopProvider.MaxCompletionTokens != nil {` |
| C34 | Kilo's auto-routing models *(2026-11-23: moved at `6ace750`; see C34now)* | `llmprovider/catalog/listing_metadata.go:287` | `variable price of its auto-routing models` |
| C35 | the completion alone fills the window | `llmprovider/context_overflow.go:69` | `max_tokens alone fills the context window: lower it` |
| C36 | the split is parsed from the message | `llmprovider/context_overflow.go:64` | `contextSplitPattern = regexp.MustCompile(` |
| C37 | the overflow kind | `llmprovider/api_error.go:44` | `ErrContextOverflow error = &sentinelError{msg: "llmprovider: context window exceeded"` |
| C38 | Together's overflow names the split | `llmprovider/providers/together/reply_0028_test.go:23` | `(8000 in the messages, 1000 in the completion)` |
| C39 | Kilo passes OpenRouter's form | `llmprovider/providers/kilo/error_in_200_test.go:25` | `This endpoint's maximum context length is 8192 tokens. However, you requested about 9000 tokens.` |
| C40 | the SDK's fixture of Anthropic's refusal of a limit above the model's | `llmprovider/providers/claude/forced_learn_0034_test.go:100` | `max_tokens: 999999 > 128000` |
| C41 | the reply-header wait | `llmprovider/internal/transport/transport.go:82` | `ResponseHeaderTimeout: 300 * time.Second,` |
| C42 | Stream adds stream to the same body | `llmprovider/providers/kilo/kilo.go:326` | `body["stream"] = true` |
| C43 | the test that pins 8192 for 128,000 | `llmprovider/providers/facts_0041_test.go:84` | `for _, limits := range [][2]int{{4096, 4096}, {128000, 8192}} {` |
| C44 | the default's value is tested | `llmprovider/settings_test.go:83` | `st.MaxTokens() != 8192` |
| C45 | and under concurrency | `llmprovider/concurrency_0048_test.go:83` | `want := 8192` |
| C46 | 0041's rule | `docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md:358` | `default is never raised (Q3).` |
| C47 | the owner's Q3 ruling *(2026-11-23: moved at `6ace750`; see C47now)* | `docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md:1113` | `**Q3. The default output limit:** cut only (D2).` |
| C48 | a probe's own limit | `llmprovider/providers/claude/claude.go:161` | `probeMaxTokens = 8192` |
| C49 | the README's line | `README.md:199` | `or levels it takes, and an output limit that cuts the default.` |
| C50 | architecture's line *(2026-11-23: moved at `6ace750`; see C50now)* | `docs/architecture.md:361` | `provider reads it once per request for its output limit and the efforts` |
| C51 | a negative request limit is refused | `llmprovider/contract.go:262` | `if req.MaxOutputTokens < 0 {` |
| C52 | a non-positive `WithMaxTokens` is refused | `llmprovider/settings.go:233` | `if s.cfg.MaxTokens <= 0 {` |
| C53 | 0062's resend | `docs/decisions/0062-MADR-learn-thinking-parameters-from-refusals.md:215` | `sends it once more, through` |
| C54 | 0084's 1 MiB read | `docs/decisions/0084-MADR-typed-error-for-a-refused-tool-schema.md:1377` | `the one limit on how much of an error body` |
| C55 | gobble's agent request names no limit | `gobble-cli:agent/agent.go:685` | `req := &llm.Request{Model: t.a.cfg.Model, Messages: msgs, Tools: t.specs, Thinking: t.a.cfg.Thinking` |
| C56 | gobble passes its limit, 0, through | `gobble-cli:llm/provider/provider.go:599` | `MaxOutputTokens: req.MaxTokens}` |
| C57 | gobble's provider has no model metadata | `gobble-cli:llm/provider/provider.go:215` | `llmprovider.WithoutModelMetadata(),` |
| C58 | gobble requires `v1.31.0` | `gobble-cli:go.mod:16` | `go-llmprovider-sdk v1.31.0` |
| C59 | kilocode's default ceiling | `kilocode:packages/opencode/src/provider/transform.ts:22` | `export const OUTPUT_TOKEN_MAX = 32_000` |
| C60 | the model's limit, capped | `kilocode:packages/opencode/src/provider/transform.ts:1796` | `return Math.min(model.limit.output, outputTokenMax)` |
| C61 | thinking can spend the 32k default | `kilocode:packages/opencode/src/kilocode/session/llm.ts:48` | `32k default can be used up by thinking before any text or tool call is` |
| C62 | so Claude gets its whole limit | `kilocode:packages/opencode/src/kilocode/session/llm.ts:71` | `return Math.max(base, input.model.limit.output)` |
| C63 | kilocode fits the limit to the window | `kilocode:packages/opencode/src/kilocode/session/llm.ts:93` | `This prevents the provider from rejecting the request with a context` |
| C64 | Kilo's docs state the ceiling | `kilocode:packages/kilo-docs/pages/code-with-ai/agents/custom-models.md:123` | `Capped at 32,000 by default.` |
| C65 | Kilo bills tokens used | `kilocode:packages/kilo-docs/pages/gateway/usage-and-billing.md:21` | `Costs are determined by the upstream provider's pricing based on token usage:` |
| C66 | Kilo's gateway does not rate-limit paid requests | `kilocode:packages/kilo-docs/pages/gateway/usage-and-billing.md:122` | `Paid model requests are not rate-limited by the gateway itself` |
| C67 | OpenRouter's reasoning budget | `kilocode:packages/opencode/src/provider/transform.ts:1462` | `return { reasoning: { max_tokens: budget } }` |
| C68 | opencode's fallback | `opencode:packages/core/src/session/model-request.ts:54` | `const OUTPUT_TOKEN_FALLBACK = 32_000` |
| C69 | opencode's ceiling | `opencode:packages/core/src/session/model-request.ts:56` | `const OUTPUT_TOKEN_MAX = 256_000` |
| C70 | strict services refuse past the window | `opencode:packages/core/src/session/model-request.ts:60` | `strict ones reject input plus output past the window` |
| C71 | OpenRouter's budget is the upstream's | `opencode:packages/ai/src/providers/openrouter.ts:161` | `as the upstream thinking budget` |
| C72 | codex's Responses request | `codex:codex-rs/codex-api/src/common.rs:287` | `pub struct ResponsesApiRequest {` |
| C73 | grok-build's limit is optional | `grok-build:crates/codegen/xai-grok-sampling-types/src/conversation.rs:638` | `pub max_output_tokens: Option<u32>,` |
| C74 | and set only for side tasks | `grok-build:crates/codegen/xai-grok-shell/src/session/helpers/session_summary.rs:230` | `.with_max_output_tokens(100)` |
| C75 | `CachedFacts` reads a cached listing | `llmprovider/catalog/lookup.go:181` | `models, _ := cfg.store.cachedListing(listingKeyFor(id, cfg, token))` |
| C76 | `WithoutModelMetadata` drops only the document | `llmprovider/catalog/lookup.go:162` | `WithoutModelMetadata leaves the document out.` |
| C77 | the 2026-10-12 snapshot lists `kilo-auto/balanced` | `llmprovider/catalog/testdata/ranking-2026-10-12/kilo.json:1` | `"id":"kilo-auto/balanced","name":"Auto Balanced"` |
| C78 | with a 1,000,000 window and a 65,536 cap (as `kilo-auto/efficient`) | `llmprovider/catalog/testdata/ranking-2026-10-12/kilo.json:1` | `"context_length":1000000,"max_completion_tokens":65536}` |
| C79 | gobble passes its cache on every path | `gobble-cli:internal/cli/agent.go:38` | `Cache: &e.cache, Sessions: &e.sessions, // on every path (0046-MADR D7, D8)` |
| C80 | gobble's SDK options carry it | `gobble-cli:llm/provider/provider.go:162` | `opts = append(opts, o.Cache.option())` |
| C81 | gobble's model lookup | `gobble-cli:llm/provider/model.go:169` | `info, _ := catalog.CachedModel(ctx, llmprovider.ProviderID(id), src, model, opts...)` |
| C82 | `CachedModel` starts the listing fetch | `llmprovider/catalog/lookup.go:147` | `cfg.store.startListingFetch(ctx, id, src, model, cfg, key)` |
| C83 | the listing's cache key | `llmprovider/catalog/listing_cache.go:40` | `return listingKey{provider: id, baseURL: listingBaseURL(id, cfg.BaseURL), scope: cmp.Or(cfg.KiloOrganization, cfg.ClaudeWorkspace),` |
| C84 | Kilo's request reads the cached facts | `llmprovider/providers/kilo/kilo.go:361` | `info, _ := catalog.CachedFacts(ctx, llmprovider.ProviderKilo, cmp.Or(req.Model, p.model),` |
| C85 | no request waits | `docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md:323` | `No request waits, and no request lists.` |
| C86 | the bound on sends | `llmprovider/internal/wire/retry.go:15` | `const MaxSends = 7` |
| C87 | 0062's Q5 (a) | `docs/decisions/0062-MADR-learn-thinking-parameters-from-refusals.md:221` | `The bound on sends per request is 7 (Q5 (a)):` |
| C88 | a stream is resent only when its opening fails | `llmprovider/internal/wire/sse.go:139` | `failure after the stream opened is never resent` |
| C89 | an error inside a 200 is classified while the stream is read | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:660` | `error inside the 200, in a chunk or its choice, is classified as Decode` |
| C90 | gobble wraps its provider in `WithStreamRetry` | `gobble-cli:llm/provider/provider.go:260` | `w = llmprovider.WithStreamRetry(p, policyOf(r), notice)` |
| C91 | gobble streams | `gobble-cli:llm/provider/provider.go:393` | `for ev, err := range llmprovider.Stream(ctx, a.p, sreq) {` |
| C92 | a header timeout is retried once by `Generate`'s retry | `llmprovider/retry.go:325` | `after the request was written, such as a timeout awaiting the reply's` |
| C93 | 0028's D-H1 | `docs/decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md:359` | `### D-H1. A failure after the request was written is retried at most once` |
| C94 | a 402 is `ErrQuotaExhausted` | `llmprovider/api_error.go:592` | `status == http.StatusPaymentRequired:` |
| C95 | Kilo's 402 is a zero balance | `kilocode:packages/kilo-docs/pages/gateway/usage-and-billing.md:41` | `When your balance reaches zero, requests to paid models will return HTTP 402` |
| C96 | an upstream's 402 reaches the client as 503 | `kilocode:packages/kilo-docs/pages/gateway/api-reference.md:382` | `it returns 503 to the client to avoid exposing internal billing details.` |
| C97 | Kilo's documented window form | `kilocode:packages/kilo-docs/pages/gateway/api-reference.md:392` | `This request exceeds the model's context window of 200000 tokens. Your request contains approximately 250000 tokens.` |
| C98 | a 5xx is unavailable | `llmprovider/api_error.go:637` | `return false, ErrProviderUnavailable` |
| C99 | the input-alone form of Anthropic, z.ai and Ollama | `llmprovider/context_overflow.go:37` | `"Anthropic, z.ai and Ollama (pi :11, :30, :35, :38)", "too long"` |
| C100 | xAI's input-alone form | `llmprovider/context_overflow.go:42` | `"xAI (pi :17, :44)", "maximum prompt length"` |
| C101 | Together's input-alone form | `llmprovider/context_overflow.go:46` | `"Together AI (pi :21, :48)", "context length"` |
| C102 | chatcompat reads models.dev by canonical id | `llmprovider/providers/chatcompat/chatcompat.go:84` | `then models.dev's facts of the model's canonical id, from the cached` |
| C103 | a fact names its source | `llmprovider/catalog/model_info.go:236` | `func (m ModelInfo) SourceOf(f Field) string {` |
| C104 | Gemini's listing states an output limit *(2026-11-23: moved at `6ace750`; see C104now)* | `llmprovider/catalog/listing_metadata.go:135` | `b.info.MaxOutputTokens = *m.OutputTokenLimit` |
| C105 | Claude's listing states one *(2026-11-23: moved at `6ace750`; see C105now)* | `llmprovider/catalog/listing_metadata.go:156` | `b.info.MaxOutputTokens = *m.MaxTokens` |
| C106 | Kilo's listing states one *(2026-11-23: moved at `6ace750`; see C106now)* | `llmprovider/catalog/listing_metadata.go:298` | `b.info.MaxOutputTokens = *e.TopProvider.MaxCompletionTokens` |
| C107 | a probe is a copy run through `Generate` | `llmprovider/providers/claude/claude.go:545` | `probe.model, probe.maxTokens, probe.reasoning = model, probeMaxTokens, nil` |
| C108 | so that the probe's request is unchanged | `llmprovider/providers/claude/claude.go:160` | `unchanged (0015-MADR D1).` |
| C109 | R55 *(2026-11-23: moved at `6ace750`; see C109now)* | `docs/guides/api-standards.md:370` | `R55. A new export is justified.` |
| C110 | `clientGiven` is derived from the value | `llmprovider/settings.go:228` | `s.clientGiven = s.cfg.HTTPClient != nil` |
| C111 | `LoggerGiven`, a second precedent | `llmprovider/settings.go:333` | `func (st *Settings) LoggerGiven() bool { return st.s.logger != nil }` |
| C112 | the standards guide says the default is never raised *(2026-11-23: moved at `6ace750`; see C112now)* | `docs/guides/api-standards.md:103` | `the output limit, which cuts the default and never raises it.` |
| C113 | and so does the 0041 test's doc comment | `llmprovider/providers/facts_0041_test.go:65` | `8192 default, which is never raised.` |
| C114 | a provider is immutable after `New` | `docs/decisions/0048-MADR-adopt-go-sdk-best-practices.md:108` | `(R20, C15); every provider runs` |
| C115 | Kilo's reasoning-object test | `llmprovider/providers/kilo/request_test.go:123` | `func TestGenerate_Reasoning(t *testing.T) {` |
| C116 | and its configured-effort test | `llmprovider/providers/kilo/kilo_test.go:119` | `func TestKilo_ReasoningEffortConfigured(t *testing.T) {` |
| C117 | Ollama's probe test | `llmprovider/providers/ollama/listing_test.go:110` | `func TestListModels_ProbeIsTheOldRequest(t *testing.T) {` |
| C118 | Kilo's overflow inside a 200 is sent once | `llmprovider/providers/kilo/error_in_200_test.go:20` | `func TestKilo_ErrorIn200IsNotRetried(t *testing.T) {` |
| C119 | Together's likewise | `llmprovider/providers/together/reply_0028_test.go:17` | `func TestTogether_TypedErrorInside200SentOnce(t *testing.T) {` |
| C120 | `[Unreleased]` carries 0080 *(2026-11-23: moved at `6ace750`; see C120now)* | `CHANGELOG.md:55` | `([0080-MADR](docs/decisions/0080-MADR-verify-a-kilo-credential-through-its-profile.md)` |
| C121 | and 0082 *(2026-11-23: moved at `6ace750`; see C121now)* | `CHANGELOG.md:280` | `([0082-MADR](docs/decisions/0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md)` |
| C122 | the newest tag's section follows *(2026-11-23: moved at `6ace750`; see C122now)* | `CHANGELOG.md:283` | `## [v1.31.0] - 2026-11-14` |
| C123 | `[Unreleased]` already adds exported names, such as this error *(2026-11-23: moved at `6ace750`; see C123now)* | `CHANGELOG.md:23` | `llmprovider.ErrToolSchemaRefused` |
| C124 | Kilo's largest stated limit | `llmprovider/catalog/testdata/ranking-2026-10-12/kilo.json:1` | `"context_length":2000000,"max_completion_tokens":1800000}` |
| C125 | a limit at 90% of the window | `llmprovider/catalog/testdata/ranking-2026-10-12/kilo.json:1` | `"context_length":1048576,"max_completion_tokens":943718}` |

C72's struct holds no output-limit field in its body (lines 290 to 313 at
57e64682f8); C73's field is set only by `with_max_output_tokens`, found
called for a session summary (C74) and an image description, not for an
agent turn. C78 holds for both `kilo-auto/balanced` and
`kilo-auto/efficient`; the per-entry values, the 2026-09-26 snapshot's
absence of any cap, and the counts in "The window the facts leave" were read
by a scratch script over both snapshots at 55a5dca. `git log v1.31.0` holds
no commit that names 0080 or 0082; `git log v1.31.0..55a5dca` holds both
(0080's 41160af to 4605777, 0082's 8051b00).

### Relation to other records

* 0041-MADR-decide-per-model-behaviour-from-metadata.md: D2 R7's cut stays;
  its "never raised", and the owner's Q3 ruling, are reversed for A2's case
  by OQ1's answer (ST31), with a dated annotation of 2026-11-21 beside R7 and
  beside Q3. Its PLAN's test changes (ST32).
  Its D1 (no request waits) decides ST54.
* 0015-MADR-canonical-sdk-api-and-module-layout.md D1: a probe's request is
  unchanged (ST26).
* 0028-MADR-heuristics-and-performance-from-the-research-pass.md D-H1 and
  D-H3: `Generate`'s one retry of a header timeout (ST24) and the overflow
  classification A4 reads; OQ3 (1) adds Kilo's documented form beside them.
* 0026-MADR F7 and 0061-MADR D1: the empty answer stays as it is (ST27,
  ST60).
* 0062-MADR-learn-thinking-parameters-from-refusals.md D5: A4's resend uses
  its path; OQ6's answer amends its Q5 (a) bound for a raised request
  (ST57), with a dated annotation of 2026-11-21 beside Q5.
* 0048-MADR-adopt-go-sdk-best-practices.md R20: per-request state (ST46);
  docs/guides/api-standards.md R55 (OQ5).
* 0043-MADR D5: the ChatGPT plan sends no limit (ST4).
* 0084-MADR-typed-error-for-a-refused-tool-schema.md: error bodies are read
  as its ST86 decided (ST41), and its tool-schema kind is never resent for
  the limit (ST59).
* 0081, 0083, 0085 and 0087: no shared lines (ST42); by OQ8's answer this
  record joins their tag, built last, after 0087 (ST43).
* gobble-cli: whether its lookup and its provider share a listing key, so
  that its model's limit reaches the request, is for its own records (ST58).

## Amendment (2026-11-22): what the cross-agent and vendor research adds

Second draft, after record review 1 of 2026-11-22. Additions and annotations
only: the owner's answers of 2026-11-21, every row ST1 to ST61 and every claim
C1 to C125 stand as written above. Where the research contradicts a statement
above, the statement is annotated here; where it bears on a decision, the
decision is a new row, decided by the evidence when the evidence settles it,
otherwise open with a question. The evidence is
[0088-REPORT-output-limits-across-agents-and-vendors.md](../reports/0088-REPORT-output-limits-across-agents-and-vendors.md)
(three retrieval passes over kilocode, opencode, codex, grok-build and the
vendors' pages, 2026-11-22; each vendor page a row rests on was fetched again
by the synthesis step, marked "verified" there).

**Status: reopened 2026-11-22 by the research amendment; the owner's
2026-11-21 answers stand.** This amendment adds six open rows (ST63, ST65,
ST66, ST67, ST71, ST74) and four questions (OQ9 to OQ12). A record with an
open row is `proposed` (the records rule), so the front matter reads
`proposed` until they are answered; it returns to `accepted` with the
answers.

### Statements above that the research corrects

* **Anthropic's output-per-minute limit does not count `max_tokens`.**
  Anthropic's rate-limit page, read 2026-11-22: "OTPM rate limits are
  evaluated in real time as output tokens are produced, counting only the
  actual tokens generated. The `max_tokens` parameter does not factor into
  OTPM rate limit calculations, so there is no rate limit downside to setting
  a higher `max_tokens` value." (report WS1, verified). This contradicts the
  "Rate limits" bullet of "What a raised limit costs", the fifth Consequence
  ("reserves up to 32,000 of the output-per-minute limit"), the reservation
  costs in OQ1's and OQ2's cells, the sentence ST38 plans for the Claude
  package doc, and the "What could not be established" bullet that rested the
  claim on the coordinator's brief. The vendor's page settles the sentence:
  ST62 decides it. OQ2's ceilings stand; OQ9 asks whether they should move,
  now that this cost is gone.
* **grok-build's agent turn sends a limit.** "What the reference clients do"
  says "an agent turn sends no output limit", and C73's note found
  `with_max_output_tokens` only on side tasks. The agent turn sets
  `max_output_tokens` to the model's `max_completion_tokens` (C126), filled
  from the server's model catalogue, a per-model setting, or a global value
  that only fills a gap (C127). Option E's first "Good" ("as codex and
  grok-build do on Responses") holds for codex only (ST64).
* **On xAI's Responses wire the limit bounds visible output only**, and the
  service's default when it is unset is 128,000 (report WS3, verified
  through the fetch tool's relay of xAI's reference). Reasoning cannot spend
  it, so the reported failure cannot occur on Grok, and ST5's R (at most
  32,000) cuts a Grok answer below what the service would allow. OQ1's
  ground, reasoning spending the limit, does not hold there (ST63, OQ9).
* **Claude models before 4.5 refuse input plus `max_tokens` past the
  window.** "On earlier models, the API returns a validation error instead"
  (report WS4, verified). Anthropic's stop-reasons page names the opt-in
  header, `model-context-window-exceeded-2025-08-26`, "for models earlier
  than Sonnet 4.5" (read 2026-11-22, verified). This answers the last "What
  could not be established" bullet; the error's text is still not
  established (ST65, OQ10).
* **OpenAI recommends reserving at least 25,000 tokens for reasoning and
  output** (report WS2, verified), above the 16,384 `Generate` ceiling
  (ST73).
* **A `length` finish is reported three ways** across the wires (C128 to
  C130, C132): kept text as `FinishLength`; an `ErrIncomplete` that drops
  every tool call, complete ones included; and, on Responses, an
  `ErrIncomplete` with reason `max_output_tokens` for every cut answer, text,
  reasoning or tool call alike. A Responses stream yields each completed item
  before that error (C134), so a caller there already receives a cut
  answer's complete tool calls (ST66, OQ11).
* **ST53 and a size-dependent credit refusal.** OpenRouter documents a 402
  before any provider is reached when the credit `max_tokens` reserves does
  not fit (report, relayed). Through chatcompat it reaches the caller as 402;
  through Kilo, an upstream's 402 reaches the client as 503 (C96), which
  ST53 already records (ST72).

### The ownership split

The split of responsibilities between this SDK and its harness caller is
stated, with each reason, in gobble-cli's
`docs/decisions/0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md`,
section "The ownership split". In brief: this SDK owns the model facts'
sources and merge, the default when a caller names none (this record), the
field on each wire, reasoning's accounting inside the limit, the static
window rule (W/2), the one resend of a limit it chose, and a faithful report
of the finish and the usage; the harness owns which facts it adds as
`catalog.WithModelLayer` layers, each call kind's limit, the room its
conversation leaves, compaction, the response to a `length` finish, user
settings, notices and telemetry. ST66, ST67 and ST71 are the contract
between them that this SDK does not yet offer.

### State table additions (2026-11-22)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST62 | The Claude package doc's sentence on Anthropic's output-per-minute limit, which ST38 plans | decided | the sentence says Anthropic does not count `max_tokens` toward that limit; it replaces the sentence ST38 planned, whose ceilings stand; the cost statements above are annotated by this amendment | report WS1 (Anthropic's page) | a `tools/claims.py` row on the sentence (PLAN Phase 1 step 6), planted to state a reservation: caught |
| ST63 | Grok, Responses, a request that names no limit | open *(2026-11-22, fourth: decided by the third amendment's ST63 row; restated in the fourth)* | today's R bounds visible output below xAI's own default of 128,000 (report WS3); OQ9 decides | OQ9 | |
| ST64 | grok-build's agent turn, as "What the reference clients do" describes it | decided | it sends the model's `max_completion_tokens` (C126, C127); the statement above and C73's note are annotated here; no outcome changes | C126, C127 | `tools/claims.py` on C126 and C127, planted false by `tools/claims.py --plants`: caught |
| ST65 | Claude before 4.5: the input plus a raised R over the window | open *(2026-11-22, fourth: decided by OQ10's answer; restated in the fourth amendment)* | refused with a validation error (report WS4) whose text is not established; no overflow form matches the one Kilo's docs quote (C131), so under ST50 the refusal is returned where 8192 would have been accepted: on a 200,000 window and R 32,000, any input from 168,000 to 191,808 tokens | OQ10 | |
| ST66 | A `length` finish with text, with reasoning only, with complete and with incomplete tool calls, on each wire | open *(2026-11-22, fourth: decided by OQ11's answer; restated in the fourth amendment)* | today as the corrections above state; ST27 unchanged | OQ11 | |
| ST67 | What a caller can see and steer of the default: the limit sent and its source, and a per-request ceiling that does not count as given | open *(2026-11-22, fourth: decided by OQ12's answer; restated in the fourth amendment)* | neither exists: a caller's value is "given" and turns off A4's resend (ST17, ST61); gobble-cli's 0065-MADR waits on the answer for its window fit (its R35) and its telemetry (its R31) | OQ12 | |
| ST68 | A thinking budget near the limit (opencode holds a budget to half the limit, report E13) | decided | ST30 unchanged: `AddThinkingFor` raises `max_tokens` above the budget rather than cutting the budget; option D's rejection stands | ST30; 0041-MADR D2 R3 | `TestAddThinkingFor_0041` (existing), its plant as ST30's |
| ST69 | Exact input counts from a token-counting endpoint, to fit R to the room left | out of scope | an extra request before each send; 0041-MADR D1, "No request waits, and no request lists" (C85); a caller that counts can bound its own request (ST67) | | |
| ST70 | A limit pushed by the service in a response header (grok-build's proxy, report E22, E23) | out of scope | no service this SDK speaks to is documented to send one; whether xAI's public API does was not established | | |
| ST71 | A refusal that states the service's cap (Anthropic's `max_tokens: N > M`, C40), remembered beyond its request | open *(2026-11-22, fourth: decided by OQ12's answer; restated in the fourth amendment)* | A4 remembers nothing, so facts that overstate the cap cost one refused send on every raised request; a cap reported to the caller and passed back, as 0062-MADR's refusals are, would cost one | OQ12 | |
| ST72 | A gateway that reserves credit from `max_tokens` and refuses before reaching a provider (OpenRouter) | out of scope | through chatcompat at a caller's base URL, where only a caller's layer raises (OQ7), the 402 is `ErrQuotaExhausted`, not resent; through Kilo an upstream's 402 reaches the client as 503 (C96), `ErrProviderUnavailable`, as ST53 records | | |
| ST73 | `Generate` on a reasoning model: OpenAI's recommended 25,000 against the 16,384 ceiling | decided | the ceiling stands, as OQ2's answer set it for the 300 s header wait (ST24), which is arithmetic; the `Generate` doc says a request that reasons should stream, citing OpenAI's recommendation | OQ2 "32k streaming, 16k plain (Recommended)"; report WS2 | a `tools/claims.py` row on the doc sentence (Phase 1 step 6), planted to drop the advice: caught |
| ST74 | A streamed request whose model states a limit above 32,000 (Kilo's stated limits reach 1,800,000, C124) | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | today's ceiling holds it at 32,000; no caller can raise it without giving its own value, which turns off the half-window rule and A4's resend; OQ9 decides whether the stream takes the model's own limit, held to the window | OQ9 | |

Counts after this amendment: 60 decided, 8 out of scope, 6 open, 0 awaiting
live evidence (74 rows).

### Owner questions (2026-11-22)

Four questions, each with three options, the recommended one first, each
with its cost and the rows it decides. A reader needs no code: "R" is the
limit this record sends when a caller names none: the model's stated limit,
capped at 32,000 on a streamed request and 16,384 on a plain one, and at half
the window, never below 8192. "Raised" means R is above 8192.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ9. How far above 8192 may a streamed request that names no limit go, and does Grok get one at all? (Anthropic no longer counts the limit against its output rate, so that cost is gone; xAI counts only visible output and defaults to 128,000.) | (1) **32,000 on a stream on every wire but Grok's; a Grok stream that names no limit sends none**, so xAI's own default applies. Plain requests keep 16,384 everywhere. (2) **32,000 on a stream on every wire, Grok included**: the 2026-11-21 answer as it stands. (3) **A stream sends the model's own stated limit, held to half the window, with no fixed ceiling**, Grok included (kilocode gives Claude its whole limit; opencode allows up to 256,000). Plain requests keep 16,384. | (1) one rule by provider, not by model name (0041-MADR D1 holds); a Grok stream may write up to 128,000 visible tokens, as the service intends; Grok's streamed test cases send no field. (2) a Grok answer is cut at 32,000 visible tokens, or its stated limit, where the service allows 128,000; nothing new to build. (3) a model whose facts overstate its cap is refused, then resent at 8192, on every raised stream; Kilo's stated limits reach 1,800,000, so the half-window rule decides most raised Kilo streams (up to 1,000,000 on a 2,000,000 window), and a model whose window is unstated is sent its whole limit; a long stream can run for many minutes; Phase 1's streamed expectations change (ST1, ST3, ST5 to ST8, ST10, ST25). | ST63, ST74 |
| OQ10. On Claude models before 4.5, a raised request whose input nearly fills the window is refused, in a form the SDK does not recognise, where 8192 would pass. | (1) **Measure, then recognise**: a live-tagged test, run under the repository's live-run rules, records Anthropic's real refusal on a model before 4.5; the resend check of a raised request then recognises that form, as ST49 recognises the limit form, and if the text states input and limit against the window, the split is parsed as C35's is. (2) **Send Anthropic's opt-in header `model-context-window-exceeded-2025-08-26` on every raised Claude request**, so an older model stops at the window with a cut answer instead of refusing; that answer always has at least the room 8192 would have had. (3) **Leave it**: ST50 returns the refusal. | (1) one live request of about 192,000 input tokens on a model before 4.5 (whether a refused request is billed was not established) and a recorded fixture before Phase 1; one form; a variant of the text that the record does not show is still returned. (2) one request header on raised Claude requests; Anthropic's page says newer models return the stop reason without the header, not whether they accept it, so a newer model refusing it is not ruled out; a cut answer where (1) would give a whole one. (3) on a 200,000 window at R 32,000, every input from 168,000 to 191,808 tokens fails; gobble-cli compacts at the window less 16,384 (C133), so a 15,616-token band of its turns fails on those models. | ST65 |
| OQ11. A cut answer is reported three ways (kept text; an error that drops complete tool calls; an error for every cut Responses answer). Should the SDK report it one way, keeping what is usable? | (1) **A new SDK record after this tag**: every wire returns a cut answer as a response finishing `length`, with its text and its complete tool calls, the cut call dropped and named in a warning; nothing usable stays `ErrEmptyAnswer`. (2) **The same, inside this record now.** (3) **Leave it as today.** | (1) changes 0012's rule (§1.5) and rows of 0020, 0026 and 0061 in a record of its own; a caller that tested for `ErrIncomplete` on a cut answer sees a response; gobble-cli's salvage of complete tool calls on Chat Completions, Messages and generateContent waits for that tag. (2) this tag's scope grows again and Phase 1 takes three wires' decoders. (3) a harness cannot run a cut answer's complete calls except on a Responses stream, and must read each wire's form apart. | ST66 |
| OQ12. Today a caller cannot see the limit the SDK chose, cannot give it a ceiling without switching off the resend, and pays one refused send per raised request when facts overstate a cap. | (1) **A new SDK record after this tag, three additions**: the response states the limit sent and where it came from; a per-request ceiling that R respects without counting as the caller's own limit, honoured down to 1,024 even below 8192; a refused cap reported as a warning and taken back by an option, as 0062-MADR does for reasoning refusals. (2) **The same record with the first two only**: the limit sent, and the ceiling. (3) **Only the report of the limit sent.** | (1) the most new exported names, each with a call site in gobble-cli's adapter (its R31, R35, D10), so R55 is met; one more record and tag. (2) as (1) less one option; facts that overstate a cap still cost a refused send on every raised request. (3) the smallest export; a harness has no window fit beyond the half-window rule and its own compaction (gobble-cli's Q1 option (1) then relies on the resend for good), and the refused send repeats. | ST67, ST71 |

### Evidence added (2026-11-22)

Read at the marker's revisions above; `llmprovider/` and the gobble-cli
files cited are unchanged from those revisions to 18e8e72 and 0589454.

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C126 | grok-build's agent turn sends the model's limit | `grok-build:crates/codegen/xai-chat-state/src/actor/request_builder.rs:86` | `max_output_tokens: self.state.sampling_config.max_completion_tokens,` |
| C127 | a global value only fills a gap | `grok-build:crates/codegen/xai-grok-shell/src/agent/config.rs:3089` | `if let Some(v) = models.max_completion_tokens {` |
| C128 | a cut Chat Completions answer with a tool call is an error | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:533` | `if finish == llmprovider.FinishLength && len(msg.ToolCalls) > 0 {` |
| C129 | and on Messages | `llmprovider/internal/wire/messages/messages.go:349` | `if finish == llmprovider.FinishLength && hasCall {` |
| C130 | every incomplete Responses answer is an error | `llmprovider/internal/wire/responses/responses.go:413` | `return nil, incomplete(raw.IncompleteDetails.Reason)` |
| C131 | a Claude window refusal quoted in Kilo's docs | `kilocode:packages/kilo-docs/pages/code-with-ai/agents/model-selection.md:112` | `input length and max tokens exceed context limit` |
| C132 | and on generateContent | `llmprovider/internal/wire/generatecontent/generatecontent.go:241` | `if finish == llmprovider.FinishLength && hasCall {` |
| C133 | gobble-cli's compaction reserve | `gobble-cli:compaction/compaction.go:47` | `var DefaultSettings = Settings{ReserveTokens: 16384, KeepRecentTokens: 20000}` |
| C134 | a Responses stream yields each completed item before an incomplete | `llmprovider/internal/wire/responses/responses.go:700` | `case "response.output_item.done":` |

## Amendment (2026-11-22, second): the owner's answers to OQ10 to OQ12, the reply on OQ9, and a limit set from evidence

Additions only: the first amendment, its rows ST62 to ST74 and its questions
stand as written; where this amendment decides one of its open rows, the row
is decided here by a new row or a status line below, citing the answer. The
evidence for this amendment is the report's section "Round 2 (2026-11-22)"
in
[0088-REPORT-output-limits-across-agents-and-vendors.md](../reports/0088-REPORT-output-limits-across-agents-and-vendors.md).
The record stays `proposed`: OQ9 is open, reworked below.

### Owner rulings (2026-11-22)

Relayed by the coordinator; labels quoted as given.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ10 | "Measure, then recognise (Recommended)" | (1): a live-tagged test records Anthropic's real refusal on a model before 4.5; the resend check of a raised request then recognises that form | ST65 |
| OQ11 | "Inside 0088, this tag" | (2), not the recommended one: every wire reports a cut answer one way, keeping what is usable, inside this record and this tag; the record's scope and its PLAN grow | ST66, ST75 to ST84 |
| OQ12 | "New record, all three (Recommended)" | (1): a new SDK record after this tag: the limit sent and its source; a per-request ceiling that is not "given"; a refused cap reported and remembered | ST67, ST71 |
| OQ9 | not answered; the owner's reply, verbatim: "this, EXACTLY this, option 1 vs option 3 if option 3 is thoroughly researched, grounded in provider code and facts. i do not want to have to limit anything by hardcoding heuristics into my gobble app, or the sdk, if at all possible to avoid that. i want more intelligence around detecting, determining, and evaluating how to accurately set model output limits based on discovered evidence and facts, not by picking an arbitrary number. dig into this deeper, more research, dig into the opencode, kilo, grok, codex, pi, and goose sources in gitrepos to find reference, similar use cases and solutions, established best practices, and determine if this should be part of the llmprovider sdk as part of catalog information gathering. update the docs with findings. explain why you recommended option 1." | none: OQ9 stays open, reworked below | ST63, ST74, ST85 to ST89 |

Status of the first amendment's open rows after these answers: ST65 decided
by OQ10 (1); ST66 decided by OQ11 (2), its cases in ST75 to ST84; ST67 and
ST71 decided by OQ12 (1), their work in the new record described below;
ST63 and ST74 stay open under the reworked OQ9.

### Why option 1 was recommended, and what the research found since

Option 1 (32,000 on a stream on every wire but Grok's, Grok sending none) was
recommended because it changed least from the owner's 2026-11-21 answers;
its one exception, Grok, rested on xAI's own documented default; and option 3
sent every model's stated limit, which on Kilo reaches 1,800,000 and equals 90
% of the window on 82 of 383 models, so a fact that overstates a cap, or a
window, would be refused on every raised request with only the half-window
rule and one resend to absorb it. That reasoning was sound about the risk and
wrong to stop there: option 1 kept 32,000, a number with no reason in any
source. opencode introduced it in commit `469f667774` (2025-07-10), "set max
output token limit to 32_000", with no body; kilocode inherited it; opencode
itself later raised its ceiling to 256,000. Option 3's half-window rule has no
source either: it was this record's own choice.

Round 2 read pi's source, goose's upstream source, the history of every fixed
number in the four agents, the vendors' own pages and specs, and this SDK's
catalogue (the report's round 2). Its findings that bear on OQ9:

* Every output limit in opencode, kilocode, grok-build, codex and pi is
  either the model's own stated limit or a constant with no recorded reason
  (report, "Every fixed number, with its provenance").
* Three services document what happens when no limit is sent: Gemini's
  default is the model's own `outputTokenLimit` (report WS7, verified); xAI's
  is 128,000 of visible output, reasoning not counted (round 1 WS3, verified);
  Ollama's is unbounded (WS9). Anthropic requires the field. OpenAI documents
  no default, but its own client, codex, sends none (report E16), and OpenAI
  counts a sent limit against its rate limit: "Your rate limit is calculated
  as the maximum of `max_tokens` and the estimated number of tokens" (WS8,
  verified). The "Whether OpenAI ... count a limit ... toward a rate limit"
  bullet of "What could not be established" is now answered: it does, so a
  raised R spends up to R of an OpenAI key's tokens-per-minute on every
  request.
* Two services fit the window themselves: Anthropic 4.5 and later stop at it;
  Gemini 4 caps the limit to the room left. OpenAI-compatible window refusals
  state the window and the split of the request (report E40), which this SDK
  already parses and discards.
* Vendors say to stream long outputs (Anthropic's SDKs refuse a plain request
  above 21,333 tokens by a stated formula; OpenAI; xAI), which bears on the
  16,384 plain ceiling, a number this record derived from its own header wait.
* pi removed fixed limits from its summary requests because reasoning hit
  them (report E34); goose's issue 13369 shows the opposite cost, a full
  reservation with no window fit.

### Where evidence gathering belongs

* **This record (0088)** holds the rule a request follows when the caller
  names no limit: OQ9's answer, the resend, and the cut answer (OQ11).
* **The new record OQ12 calls for** (its number from
  `python3 -B scripts/check_records.py --next` when written; 0089 today) holds
  the evidence and the contract, as part of the catalogue: (1) the limit sent
  and its source on the response; (2) a per-request ceiling that the default
  respects without counting as given, honoured down to the wire's documented
  minimum (16 on OpenAI Responses, WS11), not an invented floor; (3) an
  **observed** fact source in the catalogue: a cap or window a refusal states
  (Anthropic's `max_tokens: N > M`, OpenAI's "supports at most N" and its
  window split, Gemini's range), kept per catalogue cache like 0062-MADR's
  memo, reported in a warning, passed back by an option, and ranked above the
  listing and models.dev, as 0041-MADR D1 already ranks a measured fact; (4)
  the listing fields this SDK fetches and drops: Together's
  `config.max_output_length`, Hugging Face's per-provider `context_length`,
  the ChatGPT backend's `context_window`; and one accessor for the window,
  since Claude and Gemini state it as `MaxInputTokens`.
* **Outside the catalogue**: token-counting endpoints (an extra request per
  send, against 0041-MADR D1; the harness holds the conversation and its last
  usage, as pi, opencode and kilocode use); limits pushed in a response header
  (no service this SDK speaks to documents one, ST70).
* The new record is described here, not drafted, because OQ9's answer decides
  how much of it the default uses: under OQ9 (1) the observed window and the
  stated room drive the resend; under (2) or (3) the half-window rule stays and
  the accessor only serves it.

### State table additions (2026-11-22, second)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST75 | A cut answer with text, on every wire (Chat Completions, Messages, generateContent, Interactions, Responses) | decided | a response finishing `length` with its text; on Responses this replaces today's `ErrIncomplete` (C130) | OQ11 "Inside 0088, this tag" | `TestCutAnswer_Text_0088`; plant: Responses' incomplete answer still an error |
| ST76 | A cut answer with complete tool calls, with or without text | decided | a response finishing `length` with its text and its complete tool calls; today's `ErrIncomplete` (C128, C129, C132) goes | OQ11 "Inside 0088, this tag" | `TestCutAnswer_CompleteCalls_0088`; plant: the complete calls dropped |
| ST77 | The cut tool call | decided | dropped from the response and named, by tool name and call id where the wire gave them, in a warning of kind `WarningToolCallCut` | OQ11 "Inside 0088, this tag" | `TestCutAnswer_CompleteCalls_0088`, the warning; plant: no warning |
| ST78 | How a call is known complete | decided | *(2026-11-22, third: corrected by ST94 for the last call of a cut answer.)* Chat Completions: its arguments parse as a JSON object (an empty argument string is `{}`, as today, a zero-argument call); Messages: its `tool_use` block closed and its input parsed; Responses: its `response.output_item.done` arrived (C134); generateContent and Interactions: a function-call part, whole in those formats | each wire's stream format; grok-build's rule for Chat Completions (report E20) | `TestCutAnswer_CompleteCalls_0088`, one case per wire; plant: a Chat Completions call with unparsable arguments kept |
| ST79 | A cut answer with nothing usable (only reasoning, or only a cut call) | decided | `ErrEmptyAnswer` with the service's finish reason (`length`, or `max_output_tokens` on Responses, where today it is `ErrIncomplete`), as ST27 | OQ11 "Inside 0088, this tag"; ST27 | `TestCutAnswer_NothingUsable_0088`; plant: an empty response returned |
| ST80 | `Stream` | decided | the same outcome at the stream's end: complete calls yielded as items, the cut call never yielded, and `EventDone` with the response finishing `length` in place of the error; nothing usable ends with `ErrEmptyAnswer` | OQ11 "Inside 0088, this tag" | `TestCutAnswer_Stream_0088`; plant: the stream ends with `ErrIncomplete` |
| ST81 | A Responses answer incomplete for another reason (a content filter, an interruption) | decided | as today | C130's path for other reasons; OQ11 covers the output limit only | `TestCutAnswer_OtherIncomplete_0088`; plant: a content-filter incomplete returned as a response |
| ST82 | The exported API | decided | *(2026-11-22, third: replaced by ST97, open under OQ16, until the owner answers.)* `llmprovider.WarningToolCallCut` added beside `Settings.MaxTokensGiven`; R55's call site: gobble-cli's adapter (its 0065-MADR D7); `apidiff` lists both and nothing else new from this record | OQ11 "Inside 0088, this tag"; docs/guides/api-standards.md R55 | `make api-check`, `tools/apidiff.py`; plant: a third name added on a scratch clone |
| ST83 | 0012-MADR §1.5 and 0020-MADR F3, which make a cut tool call `ErrIncomplete` | decided | each gets a dated annotation pointing here when the PLAN's Phase 0 runs again; 0026-MADR F7 and 0061-MADR D1 (the empty answer) stand | OQ11 "Inside 0088, this tag" | a `tools/claims.py` row on each annotation, planted false |
| ST84 | A caller that tests `ErrIncomplete` for a cut answer | decided | it receives a response finishing `length`; the change is a **Changed** entry in `CHANGELOG.md`; existing tests that pin the old error change with this answer, as planned, not as a deviation | OQ11 "Inside 0088, this tag" | `make changelog-check`; the changed tests named at Phase 1b step 1 |
| ST85 | OpenAI with a key, and Ollama, when no caller's layer states the limit | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | OQ9: send nothing (OpenAI's own client sends none and OpenAI charges a sent limit against its rate limit; Ollama's default is unbounded), or send R | OQ9 | |
| ST86 | No source states the limit, on a wire where the field is optional (Kilo, Together, Hugging Face, chatcompat, OpenCode's chat route, Grok, Gemini) | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | OQ9: send nothing, so the service's default applies, or 8192 as OQ4 decided | OQ9 | |
| ST87 | The window | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | OQ9: let the service fit it where it does and resend once at the room a refusal states, or hold R to half the window (OQ3) | OQ9 | |
| ST88 | A plain `Generate` whose limit the SDK raised or left out | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | OQ9: read it over the service's stream and return it whole, or hold it to 16,384 (OQ2) | OQ9 | |
| ST89 | A raised request refused in a form that states a number (Anthropic's maximum `M`; OpenAI's window and input; the pre-4.5 Claude form, once measured) | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | OQ9: resend once at the stated number (the stated maximum, or the window less the stated input), or at 8192 as OQ3 decided | OQ9 | |

Counts after this amendment: 74 decided, 8 out of scope, 7 open (ST63, ST74,
ST85 to ST89), 89 rows.

### Every number the design keeps, with its provenance

| Number | Where | Provenance | Under OQ9 (1) |
| :--- | :--- | :--- | :--- |
| 8192 | the default today (C1); the resend value of OQ3 | none: inherited from mcplib | kept only where Messages requires a value and no source states one, and as the resend when a refusal states no number: a fallback with no evidence |
| 32,000 | OQ2's streamed ceiling | none: opencode `469f667774`, no body | gone |
| 16,384 | OQ2's plain ceiling | this record's arithmetic on its own 300 s wait | gone: a raised plain request streams |
| half the window | OQ3's window hold | none: this record's choice | gone |
| 300 s | the reply-header wait (report E38) | none found | kept for every request; no longer bounds a raised answer, which streams |
| 4096 | the default thinking budget (report E39) | cited to 0062-MADR only | kept, unchanged (ST30) |
| 1024 | the low-effort thinking budget | measured: 1023 refused (2026-09-27) | kept |
| 7, and 8 for a raised request | the sends a request may make (0062-MADR Q5 (a); OQ6) | owner rulings: one send per rung of the effort ladder | kept |
| 8192 (probes) | a listing probe's limit (C48) | 0015-MADR D1: the probe's request is kept unchanged | kept; a probe's limit counts as given (ST26) |
| 16 | the new record's floor for a caller's ceiling | OpenAI's spec minimum (WS11) | the floor on that wire; each wire's documented minimum where it has one |
| 256 | the memo size the new record copies from 0062-MADR | none: 0062-MADR's choice | a fallback with no evidence, used only to bound memory |
| 128,000; the model's own; unbounded | xAI's, Gemini's and Ollama's defaults when nothing is sent | the vendors' pages (round 1 WS3; WS7; WS9) | the services' own numbers, not this SDK's |

### Owner question (2026-11-22, second)

One question, three options, the recommended one first. It replaces OQ9 of
the first amendment. "R" is the limit this record sends when a caller names
none.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ9 (reworked; *2026-11-22, third: replaced by the third amendment's OQ9, OQ14, OQ15 and OQ17*). How should a request that names no limit get one, now that no source gives a reason for 32,000 or for half the window? | (1) **From each service's evidence, with no fixed ceiling.** Where a source states the model's limit (its listing, models.dev, a caller's layer), send that limit. Send nothing on Grok, OpenAI with a key and Ollama unless a caller's layer states a limit: their documented or first-party behaviour is better evidence than any number (xAI's 128,000 of visible output; OpenAI's own client sends none, and OpenAI charges a sent limit against its rate limit; Ollama is unbounded). Where no source states it, send nothing on a wire that does not require a value; Messages, which does, gets 8192. No half-window rule: the service fits the window where it does (Claude 4.5 and later, Gemini 4); a refusal that states a number is resent once at that number (the stated maximum, or the window less the stated input), others at 8192 as now. A plain `Generate` whose limit was raised or left out is read over the service's stream, as the vendors advise for long outputs. (2) **32,000 on a stream on every wire but Grok's, Grok sending none** (the first amendment's option 1). (3) **The model's own stated limit, held to half the window, on every wire** (the first amendment's option 3). | (1) supersedes OQ2's ceilings, OQ3's half-window hold and, on optional fields, OQ4's 8192, all answers of 2026-11-21, and amends OQ7 (Ollama sends nothing rather than 8192); OpenAI's default when nothing is sent is not documented, only relied on by its own client; Kilo's stated limits are sent as stated (up to 1,800,000), so a fact that overstates a cap is refused once per request until the new record remembers the stated cap; a plain `Generate` of a raised request goes over the wire as a stream; Phase 1 grows by each wire's rule, the stream-backed `Generate` and the stated-number resend. (2) keeps the 2026-11-21 answers; 32,000 has no reason in any source; Claude 4.6 and later (128,000 stated) and most Kilo models are held to 32,000; OpenAI is charged up to 32,000 of its rate limit per request; the half-window rule and 16,384 stay. (3) one rule everywhere; half the window has no source either; OpenAI is charged up to the stated limit (128,000 on several models) against its rate limit per request; Grok's visible output is cut below its own default; 16,384 stays for plain requests. | ST63, ST74, ST85, ST86, ST87, ST88, ST89 |

## Amendment (2026-11-22, third): record review 2, and OQ9 split into the choices it bundled

Additions only, after independent record review 2 of the second amendment;
each finding was checked against the code or the saved vendor text before it
was written in. The owner's answers of 2026-11-21 and 2026-11-22 stand. Rows
of the second amendment that this one corrects carry a dated note in place
(ST78, ST82); the rest stand. The record stays `proposed`.

### What review 2 found, and what this amendment does with it

* **A models.dev document handed in as a caller's layer counts as the
  caller's statement (B1).** OQ9 (1) of the second amendment sends nothing on
  Grok and OpenAI "unless a caller's layer states one", and a caller such as
  gobble-cli hands its embedded models.dev snapshot to the provider as a
  layer, whose source name is "models.dev" (`catalog/metadata_document.go:69-70`,
  C135), the same as this SDK's own document. Every snapshot model would then
  send models.dev's number on those wires: `grok-4.3`'s stated 30,000 against
  xAI's own 128,000; `gpt-5-pro`'s stated 272,000 (models.dev states it equal
  to the model's input sub-limit) charged against an OpenAI key's rate limit.
  What counts as the caller's own statement is put to the owner (OQ17, ST90).
* **The window is a fact (review 2, "Numbers").** A limit above the model's
  stated window can never be generated; models.dev states an output above the
  window for 8 models and equal to it for 47 (report, round 2). Decided: the
  limit the SDK chooses is never above the stated window (ST91), under every
  option.
* **Strict routes do not fit the window themselves (B2).** OpenRouter's
  "maximum value is the context length minus the prompt length" and Kilo
  passes OpenRouter's refusals (C39); only Anthropic's own API (4.5 and later)
  and Gemini 4 on Google's API fit it. Without a half-window rule, every
  request on such a route whose input exceeds the window less the limit is
  refused once, every turn. The two Kilo forms state the requested total, not
  the input (C39: "requested about 9000 tokens"; C97: "contains approximately
  250000 tokens"), so the input is derived as the stated total less the limit
  sent; only OpenAI's form states the input itself ("29 in the messages, 20000
  in the completion", a real response quoted on OpenAI's forum, report round
  2). The window bound is now its
  own question (OQ14), with this cost stated, and pi's per-request estimate as
  an option.
* **A cut call with no argument bytes was counted complete (B3).** The SDK
  turns empty arguments into `{}` (`wire.CompactArguments`), so ST78 would have
  run the cut call. Corrected by ST94.
* **A plain `Generate` read over the stream had nothing decided under it
  (B4).** Its retry, reply bound, usage, error-in-200 and the existing tests
  are now rows ST88a to ST88e, in force only if OQ15 is answered (1).
* **Gemini's Interactions API reports a cut answer as `ErrIncomplete` with
  the reason `incomplete`**, no finer reason, "measured at max_output_tokens"
  (`providers/gemini/interactions.go:166-168`, C136). Decided by ST93.
* **Gemini's per-call caps** (65,536 tokens a streamed hop; 32,768 on some
  plain calls) and its continuation token: decided by ST95.
* **OpenCode's responses and google routes** were missing from ST86's list
  (ST96).
* **Ollama**: sending nothing removes today's 8192 bound on a server whose
  default is "infinite generation" on a window of 4k below 24 GiB of video
  memory, whose handling of a full window was not established; and OQ7's
  change reaches chatcompat too (ST98, in OQ9's costs).
* **OpenAI with nothing sent**: its own client's practice, not a documented
  default; whether OpenAI then charges its rate limit at the model's maximum
  was not established. Said so in OQ9's costs.
* **Grok's silence is settled by xAI's own schema** (both wires: "Defaults to
  128,000 when unset", visible output only): decided, not asked (ST63).
* **The new export** (`WarningToolCallCut`, ST82) was not named in OQ11's
  cost: put to the owner (OQ16, ST97).

### Rows

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST63 (decided 2026-11-22, third) | Grok, Responses, a request that names no limit | decided | nothing is sent unless the caller states a limit (OQ17 decides what counts); xAI's default of 128,000 visible tokens applies, reasoning outside it | xAI's schema, Chat and Responses (round 1 WS3; report R6 primary copy) | `TestDefaultLimit_GrokSilent_0088`; plant: Grok sent R |
| ST90 | What counts as the caller's own statement of a limit, for the wires that otherwise send nothing | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | OQ17 | OQ17 | |
| ST91 | The stated window W below the stated limit L, or equal to it | decided | the limit the SDK chooses is never above W, under every option of OQ9 and OQ14; a caller's own value is sent as given (ST17) | the window is the model's own fact: output cannot exceed it (models.dev: 8 entries above, 47 equal) | `TestDefaultLimit_WindowCap_0088`; plant: L sent above W *(2026-11-23, ninth amendment: a `WithMaxTokens` value is sent as given only where no source states the output limit; otherwise it is cut to the output held to W)* *(2026-11-23, thirteenth amendment: under ST87 the estimate's room, always below W, holds a chosen fact, so the first case sends 8,192 − 1 − 4,096 = 4,095; the cap is pinned on a caller's `WithMaxTokens(20000)` with L 16,384 and W 8,192: 8,192, its plant moved there; the owner's ruling "Expect 4,095; add a cap case (Recommended)")* |
| ST92 | A refusal that states the window and the requested total but not the input (Kilo's forms C39, C97) | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | OQ14: under (1), the input is the stated total less the limit sent, and the resend asks the window less that input, once | OQ14 | |
| ST93 | Gemini's Interactions answer with status `incomplete` | decided | treated as a cut at the output limit, the only cause measured (C136): its usable steps returned finishing `length`, nothing usable `ErrEmptyAnswer` with the reason `incomplete`, as ST75 to ST79 decide for the other wires | OQ11 "Inside 0088, this tag"; C136 | `TestCutAnswer_Text_0088`, `TestCutAnswer_NothingUsable_0088`, their Interactions cases; plant: Interactions left an error |
| ST94 | On a `length` finish, the last tool call with an empty argument string | decided | the cut one: dropped and named in `WarningToolCallCut` (or as OQ16 decides); an earlier call with empty arguments, followed by another call, is a complete zero-argument call; ST78 is corrected so | B3: the SDK writes empty arguments as `{}` (`wire.CompactArguments`), so a cut call before its first argument byte looks complete | `TestCutAnswer_CompleteCalls_0088`, the empty-last case; plant: the empty last call kept |
| ST95 | Gemini's per-call caps (65,536 a streamed hop; 32,768 on some plain calls) and its continuation | decided | an answer stopped at such a cap is a cut answer (ST75 to ST79); the SDK does not continue it, since Gemini continues only a stored interaction and nothing is stored by default (`gemini.go:13-14`, `WithStore`) | report R6 (Gemini thinking and long-decode pages); `gemini.go:13-14` | `TestCutAnswer_Text_0088`'s Interactions case covers the report; continuation has no test |
| ST96 | ST86's list of optional-field wires | decided | it includes OpenCode's responses and google routes | correction of ST86 | `TestDefaultLimit_PerService_0088`'s OpenCode cases |
| ST97 | `WarningToolCallCut`, the export ST82 decided | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | OQ16; ST82 stands only if OQ16 is answered (1) | OQ16 | |
| ST98 | Ollama and chatcompat when nothing is stated | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | OQ9: under (1) nothing is sent, removing today's 8192 bound on a server whose default is unbounded | OQ9 | |
| ST88a | Under OQ15 (1): a failure of a `Generate` read over the stream | decided if OQ15 is (1) | a failure before the first event is retried as `Generate`'s `WithRetry` retries today; nothing is resent after an event, as for any stream (C88) | 0028-MADR D-H1; C88 | `TestGenerateOverStream_Retry_0088`; plant: a resend after an event |
| ST88b | Under OQ15 (1): the size of the assembled answer | decided if OQ15 is (1) | bounded as a plain reply is, 16 MiB (`ReplyLimit`), on top of the per-event 16 MiB (`EventLimit`), as `ErrIncomplete` | `decode_error.go:13`, `sse.go:19` | `TestGenerateOverStream_ReplyLimit_0088`; plant: no bound on the assembled answer |
| ST88c | Under OQ15 (1): a server that ignores the stream's usage request | decided if OQ15 is (1) | usage zero, as a `Stream` reports it today | the stream readers (R7 §4.2) | `TestGenerateOverStream_NoUsage_0088`; plant: an estimated usage invented |
| ST88d | Under OQ15 (1): an error inside a 200 | decided if OQ15 is (1) | classified as `Stream` classifies it (C89), and resent once under A5 when it is the first payload | C89; A5 | `TestGenerateOverStream_ErrorIn200_0088`; plant: the error returned as an empty answer |
| ST88e | Under OQ15 (1): existing `Generate` tests whose servers answer plain JSON | decided if OQ15 is (1) | listed at Phase 1 step 1 and changed with this answer, as planned, not a deviation; a `Generate` that names its own limit stays a plain request | OQ15 (1) | the listed tests, green after the change |

Counts after this amendment: 84 decided (ST88a to ST88e counted as decided
under OQ15 (1)), 8 out of scope, 10 open (ST74, ST85 to ST90, ST92, ST97,
ST98), 1 replaced (ST82, by ST97 until OQ16 is answered), 103 rows.

### Every number, with its provenance (additions to the second amendment's table)

| Number | Where | Provenance |
| :--- | :--- | :--- |
| 65,536; 32,768 | Gemini's per-call output caps (streamed hop; some plain calls) | Gemini's pages (report R6); the service's, not this SDK's |
| 16 MiB | the reply bound (`ReplyLimit`) and the event bound (`EventLimit`) | this SDK's own bounds against an unbounded body (0020-MADR F9, 0021-MADR W6); not measured; kept |
| none (a zero margin) | OQ14 (1)'s resend at the derived room | the refusal's own numbers; OpenRouter's "about" means the resend can be refused again by a few tokens, and is then returned |
| 3.5 characters a token; 4,096 | OQ14 (3), pi's per-request fit | pi's comments, not measured (report E32, E33) |
| 4 bytes a token | an estimate of messages added since the last measured count (gobble-cli's Q1 (1)) | the practice of grok-build, codex and opencode, not measured |
| 10,666 | OQ15 (3): Anthropic's formula at this SDK's 300 s wait | Anthropic's SDK formula (128,000 tokens an hour, report WS10) applied to this SDK's wait, whose own origin was not found |

### Owner questions (2026-11-22, third)

Five questions. They replace the second amendment's reworked OQ9, which
bundled several choices. Each has three options, the recommended one first. A
reader needs no code: "the stated limit" is the model's output limit as a
listing, models.dev or the caller states it; "the window" is the model's
stated context window.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ9. When a request names no limit, what does the SDK send? (Grok already sends nothing, by xAI's own documentation.) | (1) **The stated limit, never above the window; nothing where the service's own behaviour is better evidence, and nothing where no limit is stated**: OpenAI with a key, Ollama and chatcompat send nothing unless the caller states a limit; any wire whose field is optional sends nothing when no source states a limit; Messages, which requires one, gets 8192 then. (2) **The stated limit, never above the window, on every wire; 8192 where none is stated**, as decided on 2026-11-21 for an unknown limit. (3) **The 2026-11-21 rule**: at most 32,000 on a stream and 16,384 on a plain request. | (1) OpenAI's behaviour with nothing sent is its own client's practice, not documented, and whether OpenAI then charges its rate limit at the model's maximum is unknown; Ollama loses today's 8192 bound on an unbounded default; hosts with undocumented defaults (Kilo, Together, Hugging Face, chatcompat) apply their own; supersedes OQ2's ceilings and, on optional fields, OQ4. (2) OpenAI charges each request up to the stated limit against the key's rate limit (128,000 on several models); a stated limit that overstates the service's is refused once per request until the new record remembers it; supersedes OQ2's ceilings. (3) 32,000 and 16,384 have no reason in any source; Claude 4.6 and later and most Kilo models are held to 32,000. | ST74, ST85, ST86, ST98 |
| OQ14. How is the limit kept within the room the input leaves? | (1) **The window as the only fixed bound; a refused request is sent once more at the room the refusal shows** (the window less the input it states, or less its stated total minus the limit sent), and a caller may pass its own ceiling (the new record). (2) **Hold the limit to half the window**, as decided on 2026-11-21. (3) **The SDK estimates the input from the request it builds**, as pi does (3.5 characters a token, 4,096 spare), and sends the window less that. | (1) on routes that do not fit the window themselves (Kilo and OpenRouter, Together, Hugging Face, chatcompat, Claude before 4.5), every request whose input is larger than the window less the limit is refused once, each turn, unless the caller passes a ceiling; an "about" count can make the second send fail too, and it is then returned; whether refused requests are billed was not established. (2) half the window has no source; a 200,000-token model with a 128,000 limit gets 100,000 however short the input. (3) two constants with no measurement behind them; an estimate that runs low is still refused once; the SDK reads the whole request to estimate it. | ST87, ST89, ST92 |
| OQ15. A plain (non-streaming) `Generate` waits for the whole answer before its first byte, and the SDK gives up after 300 s. When the SDK raised the limit or sent none, how is it sent? | (1) **Over the service's stream, returned whole**, as Anthropic, OpenAI and xAI advise for long outputs; a `Generate` that names its own limit stays plain. (2) **Plain, held to 16,384**, as decided on 2026-11-21. (3) **Plain, held to Anthropic's own formula at the SDK's wait**: 10,666 tokens. | (1) a change of wire form for those calls, with ST88a to ST88e decided for it; existing tests whose servers answer plain JSON change; a failure after the first event is not resent. (2) 16,384 has no measurement; under OQ9 (1) a plain call that sends nothing has no bound at all and can outlast the 300 s wait, then be retried once and billed twice. (3) a vendor's formula on a wait whose own origin is unknown; an answer longer than 10,666 tokens is cut on a plain call. | ST88 |
| OQ16. When a cut answer is returned, the cut tool call is dropped. How does the caller learn of it? | (1) **A new exported warning kind, `WarningToolCallCut`**, naming the call. (2) **A warning with the kind text "tool_call_cut" and no exported name**; callers compare the text. (3) **Not reported.** | (1) one more exported name, its one call site gobble-cli's adapter; removing it later takes a record. (2) no export, but a string callers must spell right, which the API standards discourage. (3) a caller cannot tell a cut call happened or name it to the user. | ST97 (ST82) |
| OQ17. For the wires that otherwise send nothing (OpenAI with a key, Ollama, chatcompat, Grok), what counts as the caller's own statement of a limit? | (1) **A caller's layer that is not a models.dev document**: a models.dev copy handed in as a layer (its source name is "models.dev") counts as models.dev, not as the caller. (2) **Any caller's layer.** (3) **The SDK stays as it is, and the caller withholds models.dev facts for those wires itself.** | (1) one rule on the source name, as OQ7 already reads sources; a caller that wants its copy to count must give its own layer another name. (2) gobble-cli's snapshot then sends models.dev's numbers on those wires: 30,000 on `grok-4.3` where xAI's default is 128,000, and 272,000 on `gpt-5-pro` charged against the OpenAI key's rate limit. (3) every caller must know which wires are silent and filter for them; gobble-cli would carry a provider list. | ST90 |

### Evidence added (2026-11-22, third)

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C135 | a models.dev document's source name | `llmprovider/catalog/metadata_document.go:70` | `func (m Metadata) Name() string { return sourceModelsDev }` |
| C136 | an Interactions answer gives no reason when incomplete | `llmprovider/providers/gemini/interactions.go:167` | `An Interaction gives no reason (measured at max_output_tokens).` |
| C137 | empty arguments become `{}` | `llmprovider/internal/wire/finish.go:42` | `func CompactArguments(raw json.RawMessage) (string, error) {` |
| C138 | Gemini continues only a stored interaction | `llmprovider/providers/gemini/gemini.go:13` | `Continuation is Supported only with WithStore(true): Gemini chains only from` |
| C139 | the reply bound | `llmprovider/internal/wire/decode_error.go:13` | `const ReplyLimit = 16 << 20` |
| C140 | the event bound | `llmprovider/internal/wire/sse.go:19` | `const EventLimit = 16 << 20` |

## Amendment (2026-11-22, fourth): the owner's answers to OQ14 to OQ16, the reply on OQ9, and what waits for 0089-MADR

Additions and annotations only: every earlier row, answer and claim stands as
written, except where a row below decides it, and the first amendment's
rows that later amendments decided carry a dated note in place. The record
stays `proposed`: seven rows are open, each naming its question. The
discovery work the owner asked for is
[0089-MADR-model-output-limit-discovery.md](0089-MADR-model-output-limit-discovery.md)
(`proposed`), which is also the new record OQ12's answer called for.

### Owner rulings (2026-11-22, fourth)

Relayed by the coordinator; labels and replies quoted as given.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ9 | not answered by an option; the owner's reply, verbatim: "option 1 wording is confusing. stated limit or nothing, yet, nothing is not nothing, nothing is forcing an 8192 limit on anthropic models which all can support 64k or higher, so nothing means nothing. we need to address this as a real gap. my proposal is to implement a model output token discovery feature in llmprovider-sdk that provides an API surface that clients (gobble) can call to determine the model output token values acccurately to set them correctly in sessions. evaluate this and dig into research/web research similar use cases and solutions other related projects are using to resolve this ambiguity and ensure accurate model setup for sessions. be thorough. be detailed. write a new pair in the sdk repo for work there, and a pair in the gobble-cli repo for the adjoining work there to get this in place in a forward-thinking, robust, and effective manner." | none: OQ9 is replaced by 0089-MADR's OQ1, which puts the same choice with discovery's evidence behind it | none here; ST74, ST85, ST86, ST90, ST92 and ST98 now name 0089-MADR's OQ1, and ST89 its OQ3 |
| OQ14 | "we will be going with option 3, but only after the llmprovider-sdk work related to model discovery and output limit work is complete." | (3): the SDK estimates the input from the request it builds, as pi does, and sends the window less that; built only after 0089's phases | ST87 |
| OQ15 | "Over the stream (Recommended)" | (1): a plain `Generate` whose limit the SDK raised or left out is read over the service's stream and returned whole; a `Generate` that names its own limit stays plain | ST88; ST88a to ST88e are in force |
| OQ16 | "Exported WarningToolCallCut (Recommended)" | (1): `WarningToolCallCut`, exported, naming the cut call | ST97; ST82 stands |
| OQ17 | put on 2026-11-22 (third amendment); not answered | none: under 0089-MADR OQ1 (1) no fact adds the field on the wires OQ17 named, so the question falls away; under its (2) OQ17 (1)'s rule is part of that option; so ST90 waits for 0089's OQ1 *(second draft of this amendment, after record review 3: it read "not asked, decided by 0089-MADR D1", but a `proposed` record decides no row here)* | none; ST90 names 0089-MADR's OQ1 |

### The owner's question on omitting the limit (2026-11-22)

Relayed by the coordinator while the fourth amendment was drafted; a
question, not a ruling, verbatim: "on the maxoutput tokens issue, one
thing, most providers and models to NOT require that to even be set.
anthropic does, and we need to work with that, but why not simply omit
setting any max output token value at all to any provider that does not
require it?" 0089-MADR's OQ1 puts it as its option (1), recommended, with
its costs: each optional wire then takes its service's own default, and
the discovered limit is sent only where the wire requires one, and on Kilo
until a live measurement shows its own fill.

### The table's bookkeeping

Rows the earlier amendments decided whose first row still read `open` are
annotated in place with a dated note, and restated here as decided rows, so
that every `open` row in this record names a question.

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST63 (decided; restated 2026-11-22, fourth) | Grok, a request that names no limit | decided | as the third amendment's ST63 row: nothing sent unless the caller states a limit; xAI's 128,000 visible tokens apply | xAI's schema (round 1 WS3), as the third amendment recorded | `TestDefaultLimit_GrokSilent_0088`; plant: Grok sent R |
| ST65 (decided; restated 2026-11-22, fourth) | Claude before 4.5, the input plus a raised limit over the window | decided | measured first by `TestLive_ClaudeWindowRefusal_0088`, run under the repository's live-run rules, then recognised in the resend check | OQ10 "Measure, then recognise (Recommended)" | `TestDefaultLimit_ClaudeOldWindowForm_0088`; plant: the form not recognised |
| ST66 (decided; restated 2026-11-22, fourth) | A `length` finish, each kind, on each wire | decided | one report, keeping what is usable: ST75 to ST81, ST93, ST94 | OQ11 "Inside 0088, this tag" | Phase 1b's tests, as each of ST75 to ST81 names |
| ST67 (decided; restated 2026-11-22, fourth) | What a caller sees and steers of the default | decided | built by 0089-MADR: the limit sent and its source (its D7, `Response.OutputLimit`); the ceiling is its OQ4 | OQ12 "New record, all three (Recommended)" | 0089's `TestOutputLimit_Reported_0089`; plant: basis fact for a caller's value |
| ST71 (decided; restated 2026-11-22, fourth) | A refusal stating the service's cap, remembered | decided | built by 0089-MADR D4 and D5: validated, kept per cache, reported in `WarningLimitLearned`, passed back with `WithLimitObservations` | OQ12 "New record, all three (Recommended)" | 0089's `TestLimitRefusal_Validator_0089` and `TestLimitObservations_PassBack_0089`; plants as there |

### Rows (2026-11-22, fourth)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST87 (decided 2026-11-22, fourth) | The window | decided | the SDK estimates the request's input from the body it builds (3.5 characters a token, 4,096 spare, pi's) and holds the limit to the window less that, never below the wire's minimum; it reads 0089's `Limits.Window`, or `Limits.Input` where only that is stated; built after 0089's Phase 3 (PLAN Phase 1c). Until it is built in a release, the 2026-11-21 hold to half the window (OQ3) applies *(2026-11-23, twelfth amendment: "body" reads "request", counted as pi counts it, by the owner's ruling "Count content, as pi does (Recommended)"; the amended row is there)* | OQ14 "we will be going with option 3, but only after the llmprovider-sdk work related to model discovery and output limit work is complete." | `TestDefaultLimit_EstimatedFit_0088`; plant: the 4,096 removed: 31,072 |
| ST88 (decided 2026-11-22, fourth) | A plain `Generate` whose limit the SDK chose or left out | decided | read over the service's stream and returned whole; ST88a to ST88e apply | OQ15 "Over the stream (Recommended)" | `TestGenerateOverStream_0088`; plant: the stream path skipped |
| ST90 (open, 2026-11-22, fourth) | What counts as the caller's own statement | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | 0089-MADR OQ1: under (1) moot on the wires that would otherwise send nothing; under (2) a layer whose name is the models.dev document's counts as models.dev (0089 DL15, DL16) | 0089-MADR OQ1 | |
| ST97 (decided 2026-11-22, fourth) | `WarningToolCallCut` | decided | exported; ST82 stands as written | OQ16 "Exported WarningToolCallCut (Recommended)" | as ST82 |
| ST74 (open, 2026-11-22, fourth) | A streamed request whose model states a limit above 32,000 | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | 0089-MADR OQ1 | 0089-MADR OQ1 | |
| ST85 (open, 2026-11-22, fourth) | OpenAI with a key, and Ollama, with no caller statement | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | 0089-MADR OQ1 | 0089-MADR OQ1 | |
| ST86 (open, 2026-11-22, fourth) | No source states the limit, on an optional field | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | 0089-MADR OQ1 | 0089-MADR OQ1 | |
| ST89 (open, 2026-11-22, fourth) | A raised request refused in a form that states a number | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | 0089-MADR OQ3: resent once at the validated stated cap (its D4), and whether the cap is also remembered | 0089-MADR OQ3 | |
| ST92 (open, 2026-11-22, fourth) | A refusal stating the window and the requested total, not the input | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | 0089-MADR OQ1: the input derived as the total less the limit sent, under its (1) or (2) | 0089-MADR OQ1 | |
| ST98 (open, 2026-11-22, fourth) | Ollama and chatcompat when nothing is stated | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | 0089-MADR OQ1 | 0089-MADR OQ1 | |
| ST43 (reopened 2026-11-22, fourth) | The release | open *(2026-11-22, fifth: decided; see the fourth or fifth amendment's row)* | OQ18: the owner's OQ14 answer puts part of this record after 0089, so the 2026-11-21 answer to OQ8, "Join this tag", is put again with the costs that changed | OQ18 | |

Counts after this amendment: 87 decided, 8 out of scope, 8 open (ST43, ST74,
ST85, ST86, ST89, ST90, ST92, ST98), 0 replaced, 103 rows. The restated rows above
are not new rows.

### What 0088 keeps for its release, and what waits for 0089

Kept, whatever OQ18 answers: `Settings.MaxTokensGiven` (OQ5); the resend's
path, stickiness and eighth send (OQ6); Claude's pre-4.5 window
form, measured then recognised (OQ10); the cut answer reported one way (OQ11,
Phase 1b) with `WarningToolCallCut` (OQ16); a plain `Generate` read over the
stream when the SDK chose its limit (OQ15); Grok sending nothing (ST63); the
window as a cap (ST91); Kilo's documented window form (ST51).

Waits for 0089, by the owner's words on OQ9 and OQ14: the value a request
carries when it names none (ST74, ST85, ST86, ST98, by 0089's OQ1), and with
it OQ7's raise from a caller's layer on local servers (ST9), which 0089's
OQ1 (1) would end; the value a refused request is resent at (ST89, by 0089's
OQ3 and D4; ST92, by its OQ1); what counts as the caller's statement (ST90, by
0089's OQ1); the estimate fit (ST87, after
0089's Phase 3); the limit reported and the refused cap remembered (ST67,
ST71, by 0089 itself).

Under OQ18 (3) only, this record is built before 0089 on the 2026-11-21
answers (at most 32,000 on a stream, 8192 when unknown, half the window, a
resend at 8192), and 0089 replaces them in the next release.

### Owner question (2026-11-22, fourth)

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ18. Which release carries this record, now that its default waits for 0089 by the OQ14 answer? | (1) **Release the owner's five now (0081, 0083, 0084, 0085, 0087); 0088 and 0089 together in the next release**, built 0089 first, then this record on it, then OQ14 (3)'s estimate. (2) **One release that waits for 0089**: 0081, 0083, 0084, 0085, 0087, 0088 and 0089, widening OQ8's tag from six records to seven. (3) **Release six now (0081, 0083, 0084, 0085, 0087, 0088) on the 2026-11-21 rule; 0089 follows** in the next release and replaces that rule; this keeps OQ8's answer as given. | (1) changes OQ8's answer, "Join this tag"; the five finished records reach gobble-cli now; gobble-cli's empty answer at high effort stays until the next release; nothing is built twice. (2) the five finished records wait for two record builds and their reviews; one bump for gobble-cli; the longest wait for every fix. (3) gobble-cli's reported case is fixed soonest, at 32,000 on a stream; the owner's "nothing means nothing" is not met until 0089 (8192 still sent when Claude's facts are not cached); a rule built and then removed, and two behaviour changes for every caller; Phase 1's tests written twice. | ST43 |

## Amendment (2026-11-22, fifth): the owner's answers to OQ18 and to 0089's questions

Additions only. The answers are the owner's, relayed by the coordinator, each
label quoted as given. With them no row of this record is open, and by the
records rule the record is `accepted`. Its PLAN stays `proposed` until the
owner approves it.

### Owner rulings (2026-11-22, fifth)

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ18 | "Five now; 0088+0089 next (Recommended)" | (1): the next tag carries 0081, 0083, 0084, 0085 and 0087, with the untagged 0080 and 0082 entries already in `[Unreleased]`; this record and 0089 go together in the release after. This changes the owner's OQ8 answer of 2026-11-21, "Join this tag": this record does not join the next tag | ST43 |
| 0089-MADR OQ1 | "Only where required (Recommended)" | (1): a limit only where the wire requires one (Messages); nothing on optional wires; Kilo sends the discovered limit until a live measurement decides | ST74, ST85, ST86, ST90, ST92, ST98; ST99 |
| 0089-MADR OQ2 | "Never wait; route's smallest (Recommended)" | (1) | none here (0089 DL50, DL51) |
| 0089-MADR OQ3 | "By default, can turn off (Recommended)" | (1) | ST89 |
| 0089-MADR OQ4 | after asking "how about option 1 and option 3? pros vs cons. explain." and the coordinator's explanation, "models.json only, for now (Recommended)" | (1) | none here (0089 DL52) |

### Rows (2026-11-22, fifth)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST43 (decided 2026-11-22, fifth) | The changelog entry and the release | decided | this record's **Added** and **Changed** entries go under `[Unreleased]` after the next tag, which carries 0081, 0083, 0084, 0085 and 0087 (and 0080's and 0082's entries); this record and 0089 are released together in the release after, built once 0087 is finished | OQ18 "Five now; 0088+0089 next (Recommended)" | `make changelog-check`, and a `tools/claims.py` row on each entry, planted false |
| ST74 (decided 2026-11-22, fifth) | A streamed request whose model states a limit above 32,000 | decided | on a Messages route the stated limit, held to the window and the estimate's room, with no fixed ceiling; on an optional wire nothing is sent | 0089-MADR OQ1 "Only where required (Recommended)" | 0089's `TestOutputLimit_Omitted_0089`; plant: a 32,000 ceiling kept |
| ST85 (decided 2026-11-22, fifth) | OpenAI with a key, and Ollama, with no caller statement | decided | nothing sent | 0089-MADR OQ1 "Only where required (Recommended)" | as ST74; plant: a field sent on OpenAI |
| ST86 (decided 2026-11-22, fifth) | No source states the limit, on an optional field | decided | nothing sent (and nothing where a source does state one) | 0089-MADR OQ1 "Only where required (Recommended)" | as ST74; plant: 8192 sent on Kilo |
| ST89 (decided 2026-11-22, fifth) | A raised request refused in a form that states a number | decided | resent once at the validated stated cap (0089 D4), the cap kept and reported by default, `WithoutLimitLearning()` turning that off | 0089-MADR OQ3 "By default, can turn off (Recommended)" | 0089's `TestLimitLearning_Default_0089`; plant: the resend at 8192 |
| ST90 (decided 2026-11-22, fifth) | What counts as the caller's own statement | decided | moot: no fact adds the field on the wires OQ17 named | 0089-MADR OQ1 "Only where required (Recommended)" | as ST74; plant: a caller layer sent on Grok |
| ST92 (decided 2026-11-22, fifth) | A refusal stating the window and the requested total, not the input | decided | on the routes that send a limit, the input derived as the total less the limit sent, for the one resend; the window never kept (0089 DL35) | 0089-MADR OQ1 "Only where required (Recommended)"; 0089 DL35 | 0089's `TestLimitRefusal_WindowForms_0089`; plant: the window kept |
| ST98 (decided 2026-11-22, fifth) | Ollama and chatcompat when nothing is stated | decided | nothing sent; Ollama's own default, unbounded, applies | 0089-MADR OQ1 "Only where required (Recommended)" | as ST74; plant: 8192 sent on Ollama |
| ST99 (new, decided 2026-11-22, fifth) | The 2026-11-21 answers OQ2 (ceilings), OQ4 (8192 when unknown), OQ7 (raise from a caller's layer on local servers) and OQ3's half-window hold and 8192 resend | decided | superseded where the new rule differs: on optional wires nothing is sent (ST1's and ST3's to ST10's raises apply only on Messages routes); no fixed ceiling (ST1, ST3, ST5 to ST8, ST10, ST24, ST25, ST32, ST33, ST38); unknown limits send nothing, or on Messages the route's smallest current limit (ST11, ST12, ST34, ST60); ST9's raise ends; the half-window hold gives way to OQ14 (3)'s estimate (ST21, ST22, ST47); a refusal's resend uses its stated cap (ST49, ST52). Phase 1's tests follow 0089-PLAN's chosen "Tests by answer" | 0089-MADR OQ1 "Only where required (Recommended)"; 0089-MADR OQ2; 0089-MADR OQ3 "By default, can turn off (Recommended)"; OQ14 | the tests of 0089-PLAN Phase 3 and this PLAN's Phase 1 as amended; plant: a 2026-11-21 value kept (32,000 or 8192) |

Counts after this amendment: 96 decided, 8 out of scope, 0 open, 0 awaiting
live evidence, 104 rows (ST99 added).

## Amendment (2026-11-23, sixth): the merged build, the re-grounding, and OQ10's measurement

Additions only; no row's decision changes. The owner ruled, relayed by the coordinator, that this record's PLAN Phase 1 is built with 0089-PLAN's Phases 2 and 3 as one phase ("Merge the phases"), and that OQ10's measurement runs before it ("Measure first, then build (Recommended)"); both are recorded as deviation 6 in the two PLANs, since they order the work and decide no row.

### Facts found for ST65 (2026-11-23)

The measurement OQ10 (1) decided needs a Claude model before 4.5. This record names none. The committed Claude listing of 2026-10-14 lists no model before 4.5 (C141now), and the SDK's retired list marks even the first 4.5 Haiku past its retirement floor (C142now); OpenCode Zen's listing of 2026-10-12 holds none either, while Kilo's still lists two (C143now). Whether Anthropic still serves a model before 4.5 to a request that names it was not established (no live request was sent). Put to the owner with the merged phase's stop; ST65 stands as decided until the owner rules.

### Re-grounding at `6ace750` (2026-11-23)

`python3 -B tools/claims.py docs/decisions/0088-MADR-default-output-limit-from-model-facts.md --rev self=6ace750`: 127 of 140 pass, exit 1. The 13 rows that moved, none gone, each at its line at `6ace750`, and three new rows (C141now to C143now) for the facts above. The ids end in "now" so that `tools/claims.py` does not read them against this record's `self=55a5dca`; with plain ids and `self=6ace750`, on a scratch copy, 16 of 16 pass, exit 0. C120, C121 and C123 now sit in `[v1.32.0]`'s section, released: what they claimed of `[Unreleased]` holds at `55a5dca`.

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C33now | Kilo's listing states an output cap | line | `llmprovider/catalog/listing_metadata.go:327` | `if e.TopProvider.MaxCompletionTokens != nil {` |
| C34now | Kilo's auto-routing models | line | `llmprovider/catalog/listing_metadata.go:317` | `variable price of its auto-routing models` |
| C47now | the owner's Q3 ruling | line | `docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md:1116` | `**Q3. The default output limit:** cut only (D2).` |
| C50now | architecture's line | line | `docs/architecture.md:364` | `provider reads it once per request for its output limit and the efforts` |
| C104now | Gemini's listing states an output limit | line | `llmprovider/catalog/listing_metadata.go:165` | `b.info.MaxOutputTokens = *m.OutputTokenLimit` |
| C105now | Claude's listing states one | line | `llmprovider/catalog/listing_metadata.go:186` | `b.info.MaxOutputTokens = *m.MaxTokens` |
| C106now | Kilo's listing states one | line | `llmprovider/catalog/listing_metadata.go:328` | `b.info.MaxOutputTokens = *e.TopProvider.MaxCompletionTokens` |
| C109now | R55 | line | `docs/guides/api-standards.md:371` | `R55. A new export is justified.` |
| C112now | the standards guide says the default is never raised | line | `docs/guides/api-standards.md:104` | `the output limit, which cuts the default and never raises it.` |
| C120now | `[Unreleased]` carries 0080 | line | `CHANGELOG.md:76` | `([0080-MADR](docs/decisions/0080-MADR-verify-a-kilo-credential-through-its-profile.md)` |
| C121now | and 0082 | line | `CHANGELOG.md:323` | `([0082-MADR](docs/decisions/0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md)` |
| C122now | the newest tag's section follows | line | `CHANGELOG.md:326` | `## [v1.31.0] - 2026-11-14` |
| C123now | `[Unreleased]` already adds exported names, such as this error | line | `CHANGELOG.md:44` | `llmprovider.ErrToolSchemaRefused` |
| C141now | the committed Claude listing's oldest models are 4.5 | new | `llmprovider/catalog/testdata/firstparty-2026-10-14/claude.json:1` | `"id":"claude-sonnet-4-5-20250929"` |
| C142now | the retired list | new | `llmprovider/internal/retired/retired.go:28` | `"claude-haiku-4-5":           "retirement floor 2026-10-15",` |
| C143now | Kilo still lists a Claude model before 4.5 | new | `llmprovider/catalog/testdata/ranking-2026-10-12/kilo.json:1` | `"id":"anthropic/claude-sonnet-4"` |

### OQ10's measurement and the owner's rulings (2026-11-23)

The owner's rulings of 2026-11-23 on the merged phase's stop, relayed by the coordinator: S1 "Try claude-sonnet-4 once" (one run of the measurement on `claude-sonnet-4-20250514`; a not-found, or any refusal not in the window form, is the finding that no older model is served there, and ST65 is then ruled out of scope with that and the listing evidence); S2 "Same, plus explicit-8192 goldens" (a decided row naming the G-wire goldens as a planned change, checked by a script, and one golden per wire sent with an explicit `WithMaxTokens(8192)`).

The measurement ran once, on the session host, with the environment's `ANTHROPIC_API_KEY` (checked by name only): `LLMPROVIDER_LIVE_0088=1 LLMPROVIDER_LIVE_0088_MODEL=claude-sonnet-4-20250514 go test -tags live_gateways -count=1 -run '^TestLive_ClaudeWindowRefusal_0088$' -v -timeout 10m ./llmprovider/`, exit 0, one request, 0.34 s. Its allowlisted log line: `claude-sonnet-4-20250514: HTTP 404, type "not_found_error", stop reason "", message states numbers: true`. The record, `llmprovider/providers/claude/testdata/window-0088/refusal.json`, whole (no identifier in it to redact; `make fixtures-check` exit 0):

```json
{
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 32000,
  "input_words": 184000,
  "status": 404,
  "error_type": "not_found_error",
  "message": "model: claude-sonnet-4-20250514"
}
```

Anthropic does not serve `claude-sonnet-4-20250514`, the newest Claude model before 4.5 with an output cap of at least the request's 32,000, to this key: the request was refused for its model before its size was read. With the listing evidence (C141now to C143now), no Claude model before 4.5 is served on a Messages route this SDK names. So, by the owner's ruling, ST65 is out of scope.

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST65 (2026-11-23: out of scope) | Claude before 4.5, the input plus a limit the SDK chose over the window | out of scope | no Claude model before 4.5 is served on Anthropic's API (the measurement's 404 for `claude-sonnet-4-20250514`; C141now, C142now) or OpenCode's messages route (C143now's sibling listing, `zen.json`); every served model is 4.5 or later, which stops at the window with `model_context_window_exceeded` (ST23); Kilo's older Claude models answer in the gateway's forms (ST51; 0089 DL35). `TestDefaultLimit_ClaudeOldWindowForm_0088` is not built; a request naming an unlisted older model is refused as the service refuses it (ST50) | OQ10's measurement; S1 "Try claude-sonnet-4 once" | the record above; none |
| ST100 (new, 2026-11-23) | The committed G-wire goldens, recorded with no caller limit, whose request bodies carry `"max_tokens": 8192` or `"max_output_tokens": 8192` (241 at `6ace750`) | decided | a planned change of the merged phase: regenerated with `-update`; a script asserts that each changed golden differs from `6ace750` only in the output-limit field and, where ST88 applies, the stream form, and is seen failing on a planted unrelated difference on a scratch copy; and one new golden per wire, sent with an explicit `WithMaxTokens(8192)`, pins ST16 at the wire: 8192 sent as given | S2 "Same, plus explicit-8192 goldens" | `TestWireGoldens` in each provider package; the script's planted difference; plant: `WithMaxTokens(8192)` not counted as given, so the explicit golden's 8192 goes |

Counts after this amendment: 96 decided, 9 out of scope, 0 open, 105 rows (ST65 moved to out of scope; ST100 added).

## Amendment (2026-11-23, seventh): the budget case of an existing Claude test

Additions only; no row's decision changes. Building the merged phase's layer B (0089-PLAN), the existing `TestThinkingWire_Claude` (`llmprovider/providers/claude/claude_test.go:336`), its case `claude-sonnet-5` with a budget of 9000, sent `max_tokens` 128,000 where it pinned 13,096: the 13,096 was today's 8192 default raised above the budget (ST30), and under ST2 and 0089-MADR DL55 `AddThinkingFor` starts from the chosen fact, the extract's 128,000. No row named the test.

### Owner rulings (2026-11-23, seventh)

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| B1, the budget case of `TestThinkingWire_Claude` | first "Name it; add a caller layer (Recommended)"; then, corrected, "Explicit WithMaxTokens(8192) (Recommended)" | the first answer's question had wrongly described a caller layer: the build's probe on a scratch copy, a caller layer stating 8,192, sent `max_tokens 8192 thinking map[budget_tokens:8191 type:enabled]` (ST30 stops the raise at the facts' limit), and the caller's `WithMaxTokens(8192)` sent `max_tokens 13096 thinking map[budget_tokens:9000 type:enabled]`; the case gets the caller's `WithMaxTokens(8192)` | ST101 |

### Rows (2026-11-23, seventh)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST101 (new, 2026-11-23) | `TestThinkingWire_Claude`'s budget case (`claude-sonnet-5`, budget 9000, at construction and on the request), which pinned 13,096 raised from the 8192 default | decided | a planned change of the merged phase under ST2 and 0089 DL55: the case is built with the caller's `WithMaxTokens(8192)`, so its 8192 is given (ST16), the budget raises it to 13,096 (ST30), and both values stay pinned; with no caller value the case would send the chosen fact (DL55) | B1 "Explicit WithMaxTokens(8192) (Recommended)", after "Name it; add a caller layer (Recommended)" | `TestThinkingWire_Claude`, changed; plant: `WithMaxTokens(8192)` not counted as given, so the chosen fact is sent |

Counts after this amendment: 97 decided, 9 out of scope, 0 open, 106 rows (ST101 added).

## Amendment (2026-11-23, eighth): existing tests that pin the old default on a route that now omits it

Additions only; no row's decision changes. Building the merged layer B+C (0089-PLAN, deviation 8), the existing `TestTogether_RequestShapes` (`llmprovider/providers/together/together_test.go:83`, every case, at construction and on the request) asserted `body["max_tokens"] == 8192` on a request that names no limit; under ST86 and 0089-MADR DL47 Together now sends none: `together_test.go:84: max_tokens <nil>, model openai/gpt-oss-120b`. No row named the test.

### Owner rulings (2026-11-23, eighth)

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| The stop on `TestTogether_RequestShapes` | "One class row, assert absent (Recommended)" | one row names the class, with each member found listed; a later member is added to the list by a dated annotation, with no new stop for this same cause | ST102 |

### Rows (2026-11-23, eighth)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST102 (new, 2026-11-23) | Existing tests and goldens that pin the old 8,192 default on a route that now omits it. Members: `TestTogether_RequestShapes` (`llmprovider/providers/together/together_test.go:83`) | decided | a planned change of the merged phase under ST86 and 0089-MADR DL47: each such assertion becomes "no limit sent", red at the base (where 8,192 is sent), with a plant, 8192 sent on that route; a member found later is added to this row's list by a dated annotation | "One class row, assert absent (Recommended)" | each member, changed; plant: 8192 sent on that member's route |

Counts after this amendment: 98 decided, 9 out of scope, 0 open, 107 rows (ST102 added).

## Amendment (2026-11-23, ninth): the overflow form table's guard tests, and the scope of ST91's "as given"

Additions only; no row's decision changes. Building the merged phase's layer E (0089-PLAN), ST51's code adds Kilo's documented window form (C97) to `contextOverflowMessages` (`llmprovider/context_overflow.go`), and three existing tests guard that table: with no sample for the form, `TestContextOverflow_EveryFormHasASample` (`llmprovider/context_overflow_test.go:42`, "has no sample"; `:46`, "21 samples for 22 forms") and `TestContextOverflow_PrefilterAdmitsEveryPattern` (`llmprovider/context_overflow_0028_test.go:15`) fail; with the sample, `TestContextOverflow_MatchesReference` (`llmprovider/context_overflow_0028_test.go:46`) fails, since its reference, `llmprovider/context_overflow_ref_test.go`, is generated and frozen at 0028-PLAN Phase 4 without the form. No row named the two test files.

### Owner rulings (2026-11-23, ninth)

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| ST51's changes to the overflow form table's guard tests | "Name the two test changes (Recommended)" | two planned changes of the merged phase: one sample entry for Kilo's form in `overflowSamples` (`llmprovider/context_overflow_test.go`), and Kilo's C97 pattern added to `refContextOverflowMessages` in the frozen reference (`llmprovider/context_overflow_ref_test.go`), whose generated header gains a dated line naming this row; the three guard tests pass | ST51 (annotated) |

### Annotations (2026-11-23, ninth)

* **ST51.** Its test is `TestClassify_KiloWindowForm_0088` (`llmprovider/context_overflow_0088_test.go`), and the two planned changes above are part of the row: the frozen reference is the classifier before 0028-PLAN Phase 4's prefilter, over the forms the table holds, so a form added to the table is added to it too; its header says so, dated. *(2026-11-23, the owner's ruling "Name the two test changes (Recommended)".)*
* **ST91, the scope of "as given".** A caller's `WithMaxTokens` is cut to the stated output and never raised (ST15, 0089-MADR DL44), and the stated output is held to a stated window (0089-MADR DL19). So `WithMaxTokens(20000)` with W 8,192 is sent as given, 20,000, only where no source states the model's output limit; where one does, the option is cut to the held value, 8,192. A request's own `MaxOutputTokens` is sent as given in both (ST17). `TestDefaultLimit_WindowCap_0088` pins the first case on a model no extract section names (`org/window-cap-0088`). *(2026-11-23, recorded by the builder at the coordinator's word, from layer E's run: on `openai/gpt-6-luna`, which the extract states, the same case sent 8,192.)*

Counts after this amendment: unchanged, 98 decided, 9 out of scope, 0 open, 107 rows.

## Amendment (2026-11-23, tenth): Phase 1b's stop on ST78's Messages cell, and three choices left to the review

Additions only. Building PLAN Phase 1b (the cut answer, OQ11) at `247b745`, ST78's Messages cell, "its `tool_use` block closed and its input parsed", returned a call the limit cut: on a plain Messages reply the service closes the cut block and parses its partial input, so the existing `TestDecode_StopReason` (`llmprovider/internal/wire/messages/answer_test.go:32`, input `{"subject":"fix: tru"}` on `max_tokens`) and `TestFinish_Messages` (`llmprovider/internal/wire/messages/decode_0021_test.go:57`, input `{"a":1}` on `model_context_window_exceeded`) failed: `answer_test.go:34: a tool call cut by max_tokens = <nil>, want ErrIncomplete`; `decode_0021_test.go:60: model_context_window_exceeded with a call: <nil>; want ErrIncomplete, reason length`. Anthropic's stop-reasons page (platform.claude.com, "Handling stop reasons", read 2026-11-23), on a reply cut at `max_tokens`: "Check if the last content block is an incomplete tool_use".

### Owner rulings (2026-11-23, tenth)

Relayed by the coordinator; labels quoted as given.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| ST78's Messages cell | "Last tool block is the cut call (Recommended)" | (1): when a Messages reply stops at the limit (`max_tokens` or `model_context_window_exceeded`), a `tool_use` block that is the last content block is the cut call, dropped with `WarningToolCallCut`, plain or streamed; `TestDecode_StopReason` and `TestFinish_Messages` stand unchanged | ST103 |
| The builder's three code choices, below | "Defer to the review" | built as they are, each with a test and a plant like any row's; open for PLAN Phase 3's independent review, which rules on each with the owner; not decided rows | none |

### Rows (2026-11-23, tenth)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST103 (new, 2026-11-23) | A Messages reply stopped at the limit (`max_tokens` or `model_context_window_exceeded`) whose last content block is a `tool_use` | decided | that block is the cut call, dropped and named in a `WarningToolCallCut`, by `Decode` and by `Events`, whatever its input; an earlier `tool_use` block is whole when it closed and its input parsed. Amends ST78's Messages cell | "Last tool block is the cut call (Recommended)"; Anthropic's stop-reasons page | `TestCutAnswer_CompleteCalls_0088`, its plain `{"subject":"fix: tru"}` case; `TestDecode_StopReason` and `TestFinish_Messages` unchanged; plant: the last-block rule removed, caught only by that case |

### Open for PLAN Phase 3's review (2026-11-23, tenth)

Built in Phase 1b by the owner's ruling "Defer to the review", each with a test and a plant; the reviewer rules on each with the owner. None is a decided row.

* **R1, the wire-general cut rule.** On every wire, a call of a reply the limit cut is the cut call when its arguments as returned are not a JSON object, or when it is the reply's last item and its arguments are `{}`, the form a call cut before its first argument takes once the SDK writes empty arguments as `{}` (`wire.CutCall`, applied in `wire.CutAnswer`). An earlier call with `{}` is a whole call that takes none. It reads ST94 as wire-general: on generateContent and Interactions, whose call parts ST78 calls whole, a trailing call that takes no arguments is dropped on a cut reply.
* **R2, Responses' cut call.** On the Responses wire, a cut reply's call whose item has `status` `incomplete` (plain or streamed), or that a stream added (`response.output_item.added`) and never finished (`response.output_item.done`), is the cut call, dropped and named.
* **R3, streamed calls held back.** On the Responses stream, whose items are yielded as each ends, a call whose arguments are `{}` or not whole is held, not yielded, until a later item shows it was not the last (a `{}` call is then yielded) or the stream ends (a call the finished response kept is then yielded just before `EventDone`) (`wire.HeldCalls`), so that a cut call is never yielded (ST80). On the Messages stream every `tool_use` item is held until a later block starts, as ST103 needs, and one whose input is not a whole object is then held until the stream ends, by the same rule. A call is yielded later than before on those streams: on Messages every call, on Responses one taking no arguments.

Their tests, each with plants recorded in PLAN Phase 1b's execution entry: R1 `TestCutAnswer_WireGeneral_0088` and `TestCutCall_0088`; R2 `TestCutAnswer_ResponsesCutItem_0088`; R3 `TestCutAnswer_HeldCalls_0088` and `TestHeldCalls_0088`. *(2026-11-25: R1 to R3 are ruled, no longer open: R1 "Keep, except where calls arrive whole (Recommended)" is ST107, with ST110 open under OQ19; R2 "Keep as built (Recommended)" is ST108; R3 "Keep as built (Recommended)" is ST109. See "Amendment (2026-11-25, fifteenth)". The list above keeps its text.)*

### Evidence added (2026-11-23, tenth)

The ids end in "now", as the sixth amendment's do, so that `tools/claims.py` does not read them against this record's `self=55a5dca`: the text is uncommitted when written. Each was checked with a plain id against a scratch clone's commit of the tree, and planted false there (PLAN deviation 9).

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C144now | 0012-MADR §1.5's dated annotation (ST83) | new | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:341` | `*(2026-11-23: an answer the output limit cut is no longer an error on any` |
| C145now | 0020-MADR F3's dated annotation (ST83) | new | `docs/decisions/0020-MADR-remediate-v1-debugging-pass-findings.md:90` | `` *(2026-11-23: a cut tool call is no longer `ErrIncomplete`: the answer is a response finishing `length` `` |
| C146now | the **Changed** entry (ST84) | new | `CHANGELOG.md:100` | `` - An answer the output limit cut is a response finishing `length` on every `` |
| C147now | the **Added** entry (ST82) | new | `CHANGELOG.md:52` | `` - `llmprovider.WarningToolCallCut`: the output limit cut a tool call before `` |
| C148now | ST103 in the Messages decoder | new | `llmprovider/internal/wire/messages/messages.go:365` | `b.Type == blockToolUse && (i == len(m.Content)-1 \|\| b.open \|\| err != nil)` |

Counts after this amendment: 99 decided, 9 out of scope, 0 open, 108 rows (ST103 added).

## Amendment (2026-11-23, eleventh): a Responses stream cut with no `.done` events

Additions only. A live probe (round 7, allowlisted output only, 2026-11-23, relayed by the coordinator), xAI `grok-4.20-0309-non-reasoning` with `max_output_tokens` 16: a streamed text answer the limit cut sent `response.created`, `response.in_progress`, `response.output_item.added`, `response.content_part.added` and `response.output_text.delta`, then `response.incomplete` (reason `max_output_tokens`), with no `response.output_text.done`, `response.content_part.done` or `response.output_item.done`; the final response's message item is `incomplete`. A plain text answer it cut has the top-level `status` `incomplete` and its message item's `status` `completed`. OpenAI (`gpt-4o-mini`) sends every `.done` event, the item `incomplete`. The Responses stream reader built a cut answer only from `response.output_item.done`, so on Grok a cut streamed answer had nothing usable, against ST75 and ST80. xAI did not enforce `max_output_tokens` on a forced call (1042 tokens against 160), so a Grok call cut was not observed.

### Owner rulings (2026-11-23, eleventh)

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| The Responses stream cut with no `.done` events | "Fix it now in Phase 1b (Recommended)" | fixed in Phase 1b (its pass 3, PLAN deviation 10) | ST104 |

### Rows (2026-11-23, eleventh)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST104 (new, 2026-11-23) | A Responses answer the output limit cut: streamed, with items the terminal `response.incomplete` carries that no `response.output_item.done` delivered; or plain, its top-level `status` `incomplete` (reason `max_output_tokens`) whatever its items' `status` says | decided | on a cut the stream reader also takes each item the terminal `response.incomplete` carries that no `output_item.done` delivered: its text, and its calls under the cut rules as built (ST77, ST78, R1, R2), yielded before `EventDone` when kept; a plain reply's top-level `incomplete` decides the cut, whatever an item's `status` | "Fix it now in Phase 1b (Recommended)"; the live probe above | `TestCutAnswer_XAIShapes_0088` (xAI's measured stream and plain shapes); plants: the terminal response's items ignored; the cut decided by an item's status |

Counts after this amendment: 100 decided, 9 out of scope, 0 open, 109 rows (ST104 added).

## Amendment (2026-11-23, twelfth): how the estimate counts, the fallback's hold, and the budget raise

Additions and one dated amendment of ST87. Starting PLAN Phase 1c at `f4efc5e`, its re-grounding stopped (PLAN execution entry, fifteenth): ST87 estimates "from the body it builds ... pi's", while OQ14 (3) reads "from the request it builds, as pi does", and pi counts content, not a body: the text of each message, its reasoning text, each call's name and arguments, the system text and the tools, with each image at 4,800 characters (`pi:packages/ai/src/utils/estimate.ts:15`, `:16`, `:34`, `:46` to `:68`, `:114` to `:117`). This SDK's body carries an image's bytes base64-encoded (`llmprovider/item.go:82`; `llmprovider/internal/wire/messages/messages.go:121`): a 1 MiB image is 1,398,104 characters, 399,459 tokens at 3.5, above a 200,000-token window, so a body's length would hold the limit to the wire's minimum, 1. Two more cases no row decided: whether the route's fallback (0089-MADR DL51) is held as the fact is (0089-MADR D6 names the fact only), and whether the hold applies before or after the budget shape's raise (`messages.go:731`).

### Owner rulings (2026-11-23, twelfth)

Relayed by the coordinator; labels quoted as given.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| Q1. How the estimate counts the request | "Count content, as pi does (Recommended)" | (1): the request's content, as pi counts it, not the encoded body; ST87's "body" reads "request" | ST87 (amended) |
| Q2. The route's fallback | "Hold the fallback too (Recommended)" | (1): the fallback is held to the estimate's room as the fact is | ST105 |
| Q3. Claude's budget thinking shape | "Before the raise (Recommended)" | (1): the hold applies to the chosen value, before the budget shape raises it (0089-MADR DL55) | ST106 |

### Rows (2026-11-23, twelfth)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST87 (amended 2026-11-23, twelfth) | The window | decided | the SDK estimates the request's input from the request it builds, counting its content as pi does (`pi:packages/ai/src/utils/estimate.ts`), not its encoded body: the characters (Unicode code points) of its instructions, of each message's text, of each reasoning item's text (not its signature or encrypted form), of each call's name and arguments, of each call output, of each loaded tool's name, and of each tool it sends that is not deferred (its name, description and JSON-encoded schema), with each image at 4,800 characters whatever its size; at 3.5 characters a token, rounded up. A limit the SDK chose (the fact; the fallback, ST105) is held to the window less that estimate less 4,096, never below the wire's minimum; the window is 0089's `Limits.Window`, or `Limits.Input` where only that is stated (DL56); with neither, nothing is held. A caller's value (ST17, ST91) and a refusal's resend (0089-MADR D4) are not held. A stored conversation the request continues (`PreviousResponseID`) is not counted: the SDK does not hold it | Q1 "Count content, as pi does (Recommended)"; OQ14 "we will be going with option 3, but only after the llmprovider-sdk work related to model discovery and output limit work is complete." | `TestDefaultLimit_EstimatedFit_0088`, on Claude, Kilo and OpenCode's messages route (350,000 characters, W 131,072, L 60,000: 26,976), with an image case on Claude (350,000 characters and one 1 MiB image: 354,800 characters, 101,372 tokens: 25,604); `TestEstimateInput_0088` for the count; plants: the 4,096 removed (31,072); an image counted by its base64; a reasoning signature counted; a deferred tool counted |
| ST105 (new, 2026-11-23) | No source states the output; a window or input is stated; the route's fallback is sent (0089-MADR DL51) | decided | the fallback is held to the estimate's room as the fact is (ST87), basis fallback | Q2 "Hold the fallback too (Recommended)" | `TestDefaultLimit_EstimatedFit_0088`'s fallback case: Claude, a model no source states the output for, a layer stating W 50,000, a request of 35,000 characters (10,000 tokens): 50,000 − 10,000 − 4,096 = 35,904, basis fallback (the route's fallback is 64,000 or more); plant: the fallback not held: the fallback sent |
| ST106 (new, 2026-11-23) | The budget thinking shape (0089-MADR DL55) on a limit the estimate held | decided | the hold applies to the chosen value, before `AddThinkingFor` raises it above the budget: the raise may send more than the room. Where no budget is given, the default budget (4,096) applies, so a value held to 4,096 or less is sent as 8,192 (`budget + 4,096`), the budget 4,096; the report names the sent value | Q3 "Before the raise (Recommended)" | `TestDefaultLimit_EstimatedFit_0088`'s budget case: Claude, budget-shape facts, an effort and no budget, L 128,000, W 16,000, 35,000 characters: held to 1,904, sent as 8,192 with `budget_tokens` 4,096, reported 8,192; plant: the hold applied after the raise: 1,904 sent |

### Annotations (2026-11-23, twelfth)

* **ST87's test, at `f4efc5e`.** The third amendment's PLAN row for `TestDefaultLimit_EstimatedFit_0088` reads "red: 8192"; since ST99 the base sends the fact, so the case sends 60,000 there, still a red on the assertion.
* **0089-MADR DL56's test.** `TestOutputLimit_EstimateReadsLimits_0089` needs a stated output above 895,904 for the hold to show: its layer states output 1,000,000 and input 1,000,000, no window, on a Claude model in no other source.

### Open for PLAN Phase 3's review (2026-11-23, twelfth)

* **R4, the fallback and ST91's window cap.** `wire.ChooseLimit` sends the route's fallback without holding it to a stated window (`llmprovider/internal/wire/limit_rule.go:66` to `:68` at `f4efc5e`), where ST91 says the limit the SDK chooses is never above W. It is met only where no source states the output and one states a window below the route's fallback; ST105's hold, which is below the window, covers it whenever a window is stated. Not built in Phase 1c beyond ST105; the reviewer rules with the owner.

### Every number, with its provenance (additions)

| Number | Where | Provenance |
| :--- | :--- | :--- |
| 4,800 characters an image | ST87's count | pi's `ESTIMATED_IMAGE_CHARS` (`pi:packages/ai/src/utils/estimate.ts:16`), not measured |

Counts after this amendment: 102 decided, 9 out of scope, 0 open, 111 rows (ST105 and ST106 added; ST87 amended, not a new row).

## Amendment (2026-11-23, thirteenth): ST91's test under the estimate fit

Annotations only; no row's decision changes. Building PLAN Phase 1c, the suite failed on the existing `TestDefaultLimit_WindowCap_0088` (ST91), its first case: `default_limit_0088_test.go:246: kilo, L 16,384, W 8,192: sent [4095]; want 8192`. ST87 holds a value the SDK chose to the window less the estimate less 4,096, which is always below the window, so ST91's cap on a chosen fact is no longer seen where a window is stated, and the third amendment's PLAN row ("8,192 sent, whatever OQ9 and OQ14 answer") could not be met.

### Owner rulings (2026-11-23, thirteenth)

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| ST91's first case under ST87 | "Expect 4,095; add a cap case (Recommended)" | (1): the first case expects the estimate's room, 4,095, naming ST87; a new case, a caller's `WithMaxTokens(20000)` with L 16,384 and W 8,192, sends 8,192 (ST91's ninth-amendment annotation), and the window-cap plant moves to it, caught only by it (16,384) | ST91 (annotated) |

### Open for PLAN Phase 3's review (2026-11-23, thirteenth)

* **R5, the wire's minimum of 16.** `ResponsesLimits.Minimum` is 16 (`llmprovider/internal/wire/limit_refusal.go:63`), and `LimitRetry.Choose` holds an estimate's room no lower than it; but no route on the Responses wire sends a limit the SDK chose, so only `TestFit_0088` (`llmprovider/internal/wire/estimate_0088_test.go`) reaches a minimum above 1, by a value it passes itself. Recorded by the builder; the reviewer rules with the owner.

Counts after this amendment: unchanged, 102 decided, 9 out of scope, 0 open, 111 rows.

## Amendment (2026-11-24, fourteenth): R1 to R5 move to v1.34.0

Annotations only; no row's decision changes. The owner's ruling of 2026-11-24, relayed by the coordinator, quoted as given: "Yes, re-scope v1.33.0 (Recommended)". v1.33.0 is cut from what `33e6f09` holds after the joint gates and a short release review whose findings are blockers only (PLAN amendment of 2026-11-24, seventh).

* **R1 to R5** (the tenth amendment's R1 to R3, the cut-answer choices Phase 1b deferred; the twelfth's R4; the thirteenth's R5) are not ruled before v1.33.0. Each ships as built, with the tests and plants PLAN Phases 1b and 1c record, and stays open for v1.34.0's review, where the reviewer rules on each with the owner. Where the tenth, twelfth and thirteenth amendments say "PLAN Phase 3's review", read v1.34.0's. *(2026-11-25: R1 to R3 are ruled before that review, at the owner's "Rule R1–R3 first" on 0092-MADR's Q-B1: ST107 to ST109, with ST110 open; R1's change is built in PLAN Phase 6, for v1.34.0. R4 and R5 stay open for v1.34.0's review, as this bullet says. See "Amendment (2026-11-25, fifteenth)". The bullet keeps its text.)*

### Evidence added (2026-11-24, fourteenth)

ST43's rows, one per `[Unreleased]` changelog entry this record and 0089-MADR wrote (each also 0089-MADR DL60's), at the lines `CHANGELOG.md` holds at `33e6f09`. They supersede C146now and C147now, whose lines moved when Phase 1c added its entry (C160now and C153now hold the same text). The changelog's section keeps `[Unreleased]` until the caller-limit refusal report (a 0089-MADR amendment, to follow) adds its entries; when the section is dated `[v1.33.0]`, every entry moves two lines down and these rows move with it, by dated annotation. The ids end in "now", as the sixth amendment's do: the text is uncommitted when written. Each was checked with a plain id against a scratch clone's commit of the tree, 12 of 12, and planted false there, 12 of 12 caught (PLAN execution entry of 2026-11-24, the joint gates).

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C149now | the **Added** entry on catalog.LookupLimits and catalog.CachedLimits, with catalog.Limits, catalog.Fact ... (ST43; 0089 DL60) | new | `CHANGELOG.md:16` | `` `catalog.LookupLimits` and `catalog.CachedLimits`, with `catalog.Limits`, `` |
| C150now | the **Added** entry on llmprovider.Response.OutputLimit, a llmprovider.SentLimit: the output limit ... (ST43; 0089 DL60) | new | `CHANGELOG.md:35` | `` `llmprovider.Response.OutputLimit`, a `llmprovider.SentLimit`: the output `` |
| C151now | the **Added** entry on llmprovider.WarningLimitLearned, with Warning.Observed, a llmprovider.LimitObservation: a ... (ST43; 0089 DL60) | new | `CHANGELOG.md:41` | `` `llmprovider.WarningLimitLearned`, with `Warning.Observed`, a `` |
| C152now | the **Added** entry on llmprovider.Settings.MaxTokensGiven: whether WithMaxTokens gave the limit ... (ST43; 0089 DL60) | new | `CHANGELOG.md:49` | `` `llmprovider.Settings.MaxTokensGiven`: whether `WithMaxTokens` gave the `` |
| C153now | the **Added** entry on llmprovider.WarningToolCallCut: the output limit cut a ... (ST43; 0089 DL60) | new | `CHANGELOG.md:52` | `` `llmprovider.WarningToolCallCut`: the output limit cut a tool call before `` |
| C154now | the **Changed** entry on A request that names no output ... (ST43; 0089 DL60) | new | `CHANGELOG.md:58` | `` A request that names no output limit (no `MaxOutputTokens`, no `` |
| C155now | the **Changed** entry on A limit the SDK chose (the ... (ST43; 0089 DL60) | new | `CHANGELOG.md:75` | `A limit the SDK chose (the model's stated limit, or the route's smallest)` |
| C156now | the **Changed** entry on A refusal of a limit the ... (ST43; 0089 DL60) | new | `CHANGELOG.md:82` | `A refusal of a limit the SDK chose that states the service's cap` |
| C157now | the **Changed** entry on A Generate that names no output ... (ST43; 0089 DL60) | new | `CHANGELOG.md:90` | `` A `Generate` that names no output limit is now read over the service's `` |
| C158now | the **Changed** entry on A Stream whose first payload, inside ... (ST43; 0089 DL60) | new | `CHANGELOG.md:101` | `` A `Stream` whose first payload, inside a 200, is a refusal of a limit the `` |
| C159now | the **Changed** entry on Kilo's documented window refusal ("exceeds the ... (ST43; 0089 DL60) | new | `CHANGELOG.md:104` | `Kilo's documented window refusal ("exceeds the model's context window of` |
| C160now | the **Changed** entry on An answer the output limit cut ... (ST43; 0089 DL60) | new | `CHANGELOG.md:107` | `` An answer the output limit cut is a response finishing `length` on every `` |

Counts after this amendment: unchanged, 102 decided, 9 out of scope, 0 open, 111 rows.

*(2026-11-24, note from [0093-MADR-take-model-metadata-from-the-caller.md](0093-MADR-take-model-metadata-from-the-caller.md): the F-line of 72, ST8, ST9 and ST91: "models.dev" is the document the caller hands in, not one the SDK fetched (D5). This record's "now" rows on files 0093's docs phase edited were re-checked; see the note below. The text above is unchanged.)*

*(2026-11-24, from 0093-PLAN-take-model-metadata-from-the-caller.md's Phase 5: this record's "now" rows on a file 0093's Phase 4 edited were re-run with plain ids against a scratch clone's commit of the tree (`dc96bf0ce4e6`, 0093's build, uncommitted), and against `cfc911d`, its base. 21 fail there: 3 moved by 0093's build (3 found on a new line, 0 whose text 0093 rewrote), and 18 already failing at `cfc911d` (17 found on another line, 1 on none). Each row's text is now where this note says; the rows above are unchanged.)*

* C50now (`docs/architecture.md:364`): `docs/architecture.md:362`; already failing at `cfc911d`
* C109now (`docs/guides/api-standards.md:371`): `docs/guides/api-standards.md:418`; already failing at `cfc911d`
* C112now (`docs/guides/api-standards.md:104`): on no line; already failing at `cfc911d`
* C120now (`CHANGELOG.md:76`): `CHANGELOG.md:237`, `CHANGELOG.md:351`; already failing at `cfc911d`; re-placed 2026-11-24 (the review's notes closed)
* C121now (`CHANGELOG.md:323`): `CHANGELOG.md:484`; already failing at `cfc911d`; re-placed 2026-11-24 (the review's notes closed)
* C122now (`CHANGELOG.md:326`): `CHANGELOG.md:487`; already failing at `cfc911d`; re-placed 2026-11-24 (the review's notes closed)
* C123now (`CHANGELOG.md:44`): `CHANGELOG.md:205`; already failing at `cfc911d`; re-placed 2026-11-24 (the review's notes closed)
* C146now (`CHANGELOG.md:100`): `CHANGELOG.md:130`; already failing at `cfc911d`
* C147now (`CHANGELOG.md:52`): `CHANGELOG.md:72`; already failing at `cfc911d`
* C149now (`CHANGELOG.md:16`): `CHANGELOG.md:22`; moved by 0093's build
* C150now (`CHANGELOG.md:35`): `CHANGELOG.md:52`; moved by 0093's build
* C151now (`CHANGELOG.md:41`): `CHANGELOG.md:58`; moved by 0093's build
* C152now (`CHANGELOG.md:49`): `CHANGELOG.md:69`; already failing at `cfc911d`
* C153now (`CHANGELOG.md:52`): `CHANGELOG.md:72`; already failing at `cfc911d`
* C154now (`CHANGELOG.md:58`): `CHANGELOG.md:78`; already failing at `cfc911d`
* C155now (`CHANGELOG.md:75`): `CHANGELOG.md:96`; already failing at `cfc911d`
* C156now (`CHANGELOG.md:82`): `CHANGELOG.md:103`; already failing at `cfc911d`
* C157now (`CHANGELOG.md:90`): `CHANGELOG.md:113`; already failing at `cfc911d`
* C158now (`CHANGELOG.md:101`): `CHANGELOG.md:124`; already failing at `cfc911d`
* C159now (`CHANGELOG.md:104`): `CHANGELOG.md:127`; already failing at `cfc911d`
* C160now (`CHANGELOG.md:107`): `CHANGELOG.md:130`; already failing at `cfc911d`

## Amendment (2026-11-25, fifteenth): R1 to R3 ruled, and the streamed Interaction put to the owner

Additions only; no earlier row's text changes. On [0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md](0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md)'s Q-B1 (whether 0092 builds with R1 to R3 unruled), the owner chose to rule them first, and ruled them. The tenth amendment's list and the fourteenth amendment's bullet are annotated where they stand. R4 and R5 (the twelfth and thirteenth amendments) were read again: neither is ruled here, and each stays open for v1.34.0's review, as the fourteenth amendment says.

### Owner rulings (2026-11-25, fifteenth)

Relayed by the coordinator; labels quoted exactly.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| 0092-MADR Q-B1, whether 0092 builds with R1 to R3 unruled | "Rule R1–R3 first" | (2): R1 to R3 are ruled before 0092-PLAN Phase 1; this PLAN's Phase 6 (R1's change) runs first, then 0092-PLAN Phase 1 | none here; 0092's order |
| R1, the wire-general cut rule | "Keep, except where calls arrive whole (Recommended)" | as relayed: the rule "a cut reply's last call with `{}` arguments is the cut call" applies only on the wires that stream arguments in pieces, Claude Messages, OpenAI Responses and Chat Completions; on generateContent and Interactions, whose call parts arrive whole (ST78), a last `{}` call is a whole call and runs; a call whose arguments are not a JSON object is still the cut call on every wire | ST107; ST110 left open (below) |
| R2, Responses' cut call | "Keep as built (Recommended)" | as built | ST108 |
| R3, streamed calls held back | "Keep as built (Recommended)" | as built | ST109 |

### What R1's ruling meets in the tree (read at `4fc5211`)

* **Today every wire applies the last-`{}` rule.** `wire.CutAnswer` passes "last item" to `CutCall` whatever the wire (C161now, C162now), and generateContent and Interactions finish a cut reply through it (C163now, C164now). Measured on a scratch clone at `4fc5211` (`<scratch>/r1/measure-r1.txt`, a test that ran the existing `cutRun` fixtures): a cut reply of text, `lookup {"x":1}` and a last call `now` with no arguments returns `lookup` alone with one `tool_call_cut` warning on all five wires, by `Generate` and by `Stream`, ten of ten.
* **generateContent** sends a `functionCall` part whole (C170now), plain and streamed: the ruling's ground holds.
* **An Interaction's plain reply** carries its steps whole (ST93): the ground holds.
* **An Interaction's stream does not.** It sends a call's arguments as `arguments_delta` events, which the reader assembles (C169now, C166now), and finishes through the same `response()` as the plain reply (C165now, C164now). A call whose stream is cut after its `step.start` and before its first delta has no arguments, which `CompactArguments` writes as `{}` (C167now). The SDK's own cut fixtures count the Interactions stream as one that can carry a cut call (C168now). There the ruling's words ("on generateContent and Interactions ... a last `{}` call is a whole call and runs") and its ground ("only on wires that stream arguments in pieces") disagree: under the words, a call cut before its first argument runs with none. No service was measured cutting a streamed Interaction call. That case is ST110, open, put to the owner as OQ19.
* **Messages, Responses and Chat Completions** keep R1 as built. `wire.HeldCalls` (R3) runs only on the Messages and Responses streams (C171now, C172now), both wires the rule still covers, so R1's ruling changes nothing R3 does.
* **Read with ST107:** ST94 (the empty last call) holds on Messages, Responses and Chat Completions; ST78's generateContent and Interactions cells ("a function-call part, whole in those formats") hold for generateContent and the plain Interaction; ST104's "its calls under the cut rules as built (ST77, ST78, R1, R2)" reads ST107 and ST108, and on Responses nothing changes. Their text is unchanged.

### Rows (2026-11-25, fifteenth)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST107 (new, 2026-11-25; R1) | A reply the output limit cut, carrying calls: (a) its last item a call whose arguments are `{}`; (b) a call whose arguments are not a JSON object; (c) an earlier call with `{}`; on every wire and mode except the streamed Interaction (ST110) | decided | (a) on Messages, Responses and Chat Completions, plain and streamed: the cut call, dropped and named in a `WarningToolCallCut`, as built (ST94); on generateContent, plain and streamed, and an Interaction's plain reply: a whole call that takes no arguments, returned (and on generateContent's stream yielded), with no warning. (b) the cut call on every wire, as built. (c) a whole call on every wire, as built. The wire decides, at `wire.CutAnswer`'s callers; `CutCall`'s rule and `HeldCalls` are unchanged. Amends R1 as built for generateContent and the plain Interaction | R1 "Keep, except where calls arrive whole (Recommended)"; ST78; ST93; C170now | `TestCutAnswer_WireGeneral_0088`, amended in PLAN Phase 6: its trailing-call case on all five wires by `Generate` and `Stream` (the streamed Interaction as ST110 decides): generateContent's two modes and the plain Interaction return `lookup {"x":1}` and `now {}` with no warning; Chat Completions, Messages and Responses, both modes, return `lookup {"x":1}` alone, with one warning naming `now` and `call_z`; its not-an-object case unchanged. Red at `4fc5211` on its assertion (all ten modes drop `now` there, measured above). Plant: the wire distinction removed (every wire's last `{}` call the cut one, as at `4fc5211`), caught by ST107's keep cases and by no other row's test; ST94's plant (d), the empty last call kept, re-run and still caught by `TestCutAnswer_CompleteCalls_0088` |
| ST108 (new, 2026-11-25; R2) | On the Responses wire, a cut reply's call whose item has `status` `incomplete` (plain or streamed), or that a stream added (`response.output_item.added`) and never finished (`response.output_item.done`) | decided | the cut call, dropped and named, its arguments whole or not; as built (the tenth amendment's R2) | R2 "Keep as built (Recommended)" | `TestCutAnswer_ResponsesCutItem_0088`, unchanged; plants as PLAN Phase 1b's entry records them: (a) the incomplete-status check removed, `:624: responses, an incomplete call item, Generate: calls ["lookup {\"x\":1}" "fetch {\"y\":2}"]; ...`; (b) a call added and never done not named, `:624: responses, an incomplete call item, Stream: warnings []; want one WarningToolCallCut naming fetch and call_b` |
| ST109 (new, 2026-11-25; R3) | A streamed call a cut could still drop: on the Responses stream, a call whose arguments are `{}` or not whole; on the Messages stream, every `tool_use` until a later block starts, and one whose input is not a whole object until the stream ends | decided | held, not yielded, until a later item shows it was not the last (a `{}` call is then yielded) or the stream ends (a call the finished response kept is then yielded just before `EventDone`), so a cut call is never yielded (ST80); a call is yielded later than before on those streams; as built (`wire.HeldCalls`, the tenth amendment's R3) | R3 "Keep as built (Recommended)" | `TestCutAnswer_HeldCalls_0088` and `TestHeldCalls_0088`, unchanged; plants as PLAN Phase 1b's entry records them: no call held, `:639: responses, Stream: items ["first" "lookup" "now"]; want first, then lookup, and never the last {} call`; `Later` releases nothing, `cut_0088_test.go:100: Later yielded []; want now alone, the partial call kept`; `Done` yields a dropped call, `cut_0088_test.go:109: Done yielded ["now"]; want nothing: the response dropped the call` |
| ST110 (new, 2026-11-25) | A streamed Interaction the output limit cut whose last step is a `function_call` that received no `arguments_delta`, so its arguments read `{}` (C167now) | open *(2026-11-25: decided by OQ19's answer, "Decide by what arrived"; restated in the sixteenth amendment)* | OQ19; recommended (1): the cut call, dropped and named, as on the other wires that stream arguments in pieces, while the plain Interaction keeps it (ST107) | OQ19; R1's ruling; C164now to C169now | under OQ19 (1) or (3): `TestCutAnswer_WireGeneral_0088`'s streamed Interaction case, `now` dropped and named (under (3), with no `arguments_delta`; a second case with a whole `{}` delta keeps it); plant: the stream read as the plain reply, `now {}` yielded. Under (2): `now {}` returned and yielded; plant: the stream read as a wire that streams in pieces |

### Owner question (2026-11-25, fifteenth)

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ19. R1's ruling names Interactions among the wires whose call parts arrive whole, and that holds for an Interaction's plain reply. Its stream, though, sends a call's arguments in pieces (`arguments_delta`), like Messages, Responses and Chat Completions, so a call cut after it started and before its first piece reads as `{}`, a call with no arguments. On a streamed Interaction the limit cut, what is a last call with `{}`? | (1) **The cut call, as on the other wires that stream arguments**: dropped and named; the plain Interaction keeps it. (2) **A whole call, as the ruling's words say**: kept and run on both paths. (3) **Decide by what arrived**: the cut call when no `arguments_delta` came for it; a whole call when one delivered `{}`. | (1) the same reply shape reads two ways on one wire, plain and streamed; `interaction.response()`, which both paths share, learns which path it reads (internal, no export); a zero-argument call that ends a cut streamed Interaction is dropped and named, never run, as on Messages, Responses and Chat Completions. (2) the simplest code; a call cut before its first argument runs with none, the failure R1 guards against elsewhere; no service was measured sending such a cut. (3) the reader records whether each call step received a delta (one flag a step); right whether or not Google sends a delta for a call that takes no arguments, but unmeasured: if it sends none, a zero-argument call that ends a cut stream is dropped, as under (1). | ST110 |

### Evidence added (2026-11-25, fifteenth)

Read at `4fc5211`. The ids end in "now", as the sixth amendment's do, so that `tools/claims.py` does not read them against this record's `self=55a5dca`. Each was checked with a plain id against a scratch clone at `4fc5211`, and planted false there (PLAN execution entry of 2026-11-25, nineteenth).

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C161now | `CutAnswer` gives `CutCall` "last item" on every wire | new | `llmprovider/internal/wire/finish.go:29` | `ok && CutCall(call.Arguments, i == len(r.Output)-1) {` |
| C162now | R1 as built | new | `llmprovider/internal/wire/finish.go:56` | `return !WholeObject(args) \|\| last && noArguments(args)` |
| C163now | generateContent finishes a cut reply through `CutAnswer` | new | `llmprovider/internal/wire/generatecontent/generatecontent.go:279` | `return wire.CutAnswer("generatecontent", result, finish)` |
| C164now | and so does an Interaction | new | `llmprovider/providers/gemini/interactions.go:217` | `return wire.CutAnswer("gemini interaction", result, statusIncomplete)` |
| C165now | an Interaction's stream finishes through the plain reply's `response()` | new | `llmprovider/providers/gemini/interactions.go:369` | `resp, err := s.raw.response()` |
| C166now | an Interaction's stream assembles `arguments_delta` | new | `llmprovider/providers/gemini/interactions.go:337` | `case "arguments_delta":` |
| C167now | empty arguments are written `{}` | new | `llmprovider/internal/wire/finish.go:177` | `return "{}", nil` |
| C168now | the cut fixtures: an Interaction's stream can carry a cut call | new | `llmprovider/providers/cut_answer_0088_test.go:67` | `streamCuts: true, plain: interactionCutPlain, stream: interactionCutStream},` |
| C169now | measured: an Interaction's stream sends its arguments in pieces | new | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:447` | `` `signature`, and sends its arguments as `arguments_delta`, which are `` |
| C170now | measured: generateContent sends a `functionCall` part whole | new | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:441` | `` * generateContent: text parts, and `functionCall` parts sent whole `` |
| C171now | `HeldCalls` on the Messages stream | new | `llmprovider/internal/wire/messages/messages.go:574` | `held wire.HeldCalls` |
| C172now | and on the Responses stream | new | `llmprovider/internal/wire/responses/responses.go:767` | `held wire.HeldCalls` |
| C173now | the Responses wire's cut paths, plain and streamed, through `CutAnswer` | new | `llmprovider/internal/wire/responses/responses.go:838` | `resp, err = wire.CutAnswer("responses", result, reasonOutputLimit)` |

Counts after this amendment: 105 decided, 9 out of scope, 1 open (ST110), 115 rows (ST107 to ST110 added). R4 and R5 are not rows; they stay open for v1.34.0's review. By the records rule the record is `proposed` until OQ19 is answered.

## Amendment (2026-11-25, sixteenth): OQ19 answered, the streamed Interaction decided

Additions only; ST110 is annotated where it stands. The owner's answer was given after the coordinator corrected a premise. The owner had been told earlier that Interactions calls arrive whole, but a streamed Interaction sends a call's arguments as `arguments_delta` pieces (the fifteenth amendment; C166now, C169now).

### Owner rulings (2026-11-25, sixteenth)

Relayed by the coordinator; label quoted exactly.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ19 | "Decide by what arrived" | (3): on a cut, streamed Interaction, the last call is judged by whether any `arguments_delta` came for it | ST110 |

### Rows (2026-11-25, sixteenth)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST110 (decided 2026-11-25, sixteenth) | A streamed Interaction the output limit cut whose last step is a `function_call`: (a) no `arguments_delta` came for it, so its arguments read `{}` (C167now); (b) its deltas delivered `{}` | decided | (a) the cut call: dropped and named in a `WarningToolCallCut`, never yielded; (b) a whole call that takes no arguments: returned and yielded, no warning. An Interaction's plain reply keeps ST107 (a last `{}` call is whole). The stream reader records, for each call step, whether any `arguments_delta` arrived | OQ19 "Decide by what arrived"; ST107 | `TestCutAnswer_WireGeneral_0088`'s two streamed Interaction cases (PLAN Phase 6): a last `now` step with no delta, `lookup {"x":1}` alone and one warning naming `now` and `call_z`; a last `now` step whose one delta is `{}`, `lookup {"x":1}` and `now {}` returned and yielded, no warning. Red at `4fc5211`: the `{}`-delta case (every mode drops `now` there); the no-delta case is a green pin. Plant: the delta check removed (the streamed step read as the plain reply), caught by the no-delta case |

### What is unmeasured

* Whether Google sends an `arguments_delta` for a call that takes no arguments. If it sends none, such a call that ends a cut streamed Interaction is dropped and named under (a). That is the safe direction: the call never runs with arguments it did not get, and the model asks again on the next turn. A call that is not last, or a reply that was not cut, is unaffected.
* **Follow-up, not run:** a live probe of a streamed Interaction calling a tool that takes no arguments, to see whether a delta comes (allowlisted output only: event types, delta types, counts). It runs only on the owner's word.

Counts after this amendment: 106 decided, 9 out of scope, 0 open, 115 rows. R4 and R5 are not rows; they stay open for v1.34.0's review. With no open row the record is `accepted` again.
