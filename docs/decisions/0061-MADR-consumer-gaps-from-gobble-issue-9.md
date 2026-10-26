---
status: accepted
date: 2026-10-24
decision-makers: repository owner
consulted: "GitHub issue #9 (2026-10-24); gobble-cli at 142685b (its llm/provider, llm, llm/catalog and tool/builtin packages; its 0004, 0005 and 0031 MADRs as the issue cites them); this repository at d9b1910; the recorded fixtures under llmprovider/catalog/testdata (models.dev snapshots of 2026-09-26 and 2026-10-12, first-party listings of 2026-10-14); the vendors' documentation fetched on 2026-10-24 (Anthropic, OpenAI, xAI, Google Gemini, Ollama, Hugging Face, Together, Kilo and OpenRouter, OpenCode), quoted below with URLs; the owner's answers of 2026-10-24 to Q1 to Q8"
informed: gobble-cli, and every consumer of llmprovider and its providers
---

<!-- markdownlint-disable MD013 -->

# Close gobble-cli's Consumer Gaps: Empty-Answer and Offline Errors, Max Effort, a Chat Completions Constructor, and Image Input

## Context and Problem Statement

Issue #9 lists what gobble-cli still copies around this module because the
public contract lacks it: an empty-answer sentinel, an offline sentinel,
`EffortMax`, a generic Chat Completions constructor, a per-request header
hook, a 1-hour cache-write rate, image input, and, as its Q1, streamed
tool-call arguments. gobble-cli's rule is to wait for this library rather
than grow a second wire (the issue, citing gobble's 0004-MADR). The issue
asks a MADR here to decide names (R55) and keep `apidiff` compatible (R48).

Every fact below was read in this repository at `d9b1910` or in gobble-cli
at `142685b` (the Code claims, run by `tools/claims.py`), measured from the
recorded fixtures (Measurements), or quoted from a vendor's page fetched on
2026-10-24 (External evidence). Where a vendor's page does not say a thing,
this record says so (UNCONFIRMED) and the PLAN pins it with a live test or
leaves it out.

### Findings, verified

**The issue's facts that hold (each with its claim):**

* **F1.** `Item` is sealed and has five types; `Request.validate` refuses any
  other (C1, C2). `Capabilities` has five fields, none for images (C3), and
  `Support`'s zero value is `Unsupported` (C4), so a field added to it is
  `Unsupported` for every provider, a third party's included, until set.
  `Capabilities.Check` refuses an `Unsupported` need from a table (C5).
* **F2.** The empty answer is an `*APIError` of kind `ErrIncomplete` whose
  message ends ": the answer has no content" (C8), and gobble-cli matches that
  suffix as text (G1). Rewording it would silently break gobble-cli.
* **F3.** A failure to reach the service is returned unwrapped by
  `wire.Post` (C12, C13), so `errors.Is(err, ErrProviderUnavailable)` is false
  for a refused dial, and gobble-cli classifies the dial itself (G2 to G5).
* **F4.** `Effort` stops at `xhigh` (C18); gobble-cli sends its `max` level as
  `EffortXHigh` (G6).
* **F5.** gobble-cli wraps the client in its own `headerTransport` (G7); its
  `read` tool says it cannot send images (G8); its content already carries
  `MediaType` and `Data` (G9); it skips a `models.json` provider this module
  does not build (G10). It pins this module at `v1.22.0` (G11).
* **F6.** `ModelRates` has one `CacheWrite` (C42).

**Where the issue's premises are wrong or incomplete (corrected here):**

* **F7. Anthropic has a `max` effort.** The issue says "if the service has
  no `max` effort … do not invent a fourth Anthropic effort". Anthropic's
  documentation lists "`low`, `medium`, `high`, `xhigh`, and `max`" (E3), its
  own listing reports `effort.max` per model (C20; Measurements M1), and the
  catalog already parses it into a model's facts (C19, C20). It also says
  "Not every model that supports `max` supports `xhigh`" (E3): in the
  recorded listing, `claude-sonnet-4-6` and `claude-opus-4-6` list `max` and
  not `xhigh`, and `claude-opus-4-5-20251101` lists only `low` to `high` (M1).
* **F8. `max` is not one wire value everywhere.** OpenAI's Responses and Chat
  Completions enums include `max` (E5); xAI's stop at `xhigh` (E4); Gemini's
  `thinking_level` stops at `high` (E8); Ollama resolves `xhigh` to `max` and
  this module already sends `EffortXHigh` there as `"max"` (C23; E9). The
  module already maps the top effort per provider: Grok clamps by an order
  (C21), Gemini sends `xhigh` as `high` (C22), Ollama sends `xhigh` as `max`
  (C23), and the Messages wire passes the effort as given (C24).
* **F9. Same-role text does not merge on the Messages wire.** The issue says
  "the Messages encoder already merges same-role blocks". A `MessageItem`
  opens a message of its own (C26); only tool blocks join the previous
  message. The block type has no `source` field (C27). Each of the five
  encoders sends a message's text as one string or one text part (C26, C28 to
  C31), so an image needs a defined joining rule per wire (D5).
* **F10. The decoder test the issue cites is about output.** Chat
  Completions' `TestChatCompletions_ContentParts` drops an `image_url` part of
  the model's *answer*; it says nothing of input. Image output is out of the
  issue's scope.
* **F11. R2 needs no change for a new provider package.** R2's
  `llmprovider/providers/<id>` row already covers "one provider or gateway
  family" importing `llmprovider`, `auth`, `catalog` and internal packages
  (C38). A new package needs a row in `docs/architecture.md`'s package table,
  which `tables-check` enforces.
* **F12. The send sites are more than `internal/transport`.** Fifteen
  non-test calls send a request (Measurements M4) *(corrected 2026-10-24:
  seventeen; see the amendment "Phase 2's corrections")*: `wire.Post`; catalog
  discovery (eight), `raw.go` (three), `listing_metadata.go` and
  `model_metadata.go`; the ChatGPT listing; the OAuth refresh and the sign-in
  requests. Ollama's two discovery calls and the OAuth refresh already wrap
  `ErrProviderUnavailable` (C16, C17). `llmprovider` imports
  `internal/transport` (C14), so `transport` cannot name `ErrOffline`.
* **F13. Additions are a minor release.** The issue's "one patch release is
  enough" is wrong for new exported names; this repository ships additions as
  minor versions: `wizard.Options.ShowDeviceCode`, an addition, is in
  `v1.22.0`'s Added section (C46, C47).
* **F14. The 1-hour rate has no source.** The recorded models.dev documents
  carry `cache_write`, `context_over_200k` and `tiers`, and no 1-hour price
  (C43; Measurements M3).
* **F15. R28 forbids a request hook; R22 and R38 do not fit a generic
  constructor.** R28: "There are no hooks inside providers" (C36). R22: ids
  are models.dev keys (C35). R38: adding a provider is an entry in
  `providers.Default()` (C37).
* **F16. A stale count in the architecture.** `docs/architecture.md` says
  G-wire drives "18 provider and gateway-route cases through seven scenarios
  … 114 golden files" (C41); the code has 20 cases, 12 scenarios and 227
  golden files (C40; Measurements M5). 0060's docs pass missed it; this
  record's image and effort scenarios change the count again, so D11
  corrects it.

## Decision Drivers

* gobble-cli drops each copy only when the library gives the same behaviour
  through the public contract, and never a second wire (the issue).
* Additions only: `make api-check` against `v1.22.2` lists additions and no
  incompatible change (R48).
* Every unsupported or degraded request is refused before the network or
  reported, never silently changed (R11, R12; 0034's Warning).
* No vendor behaviour is assumed: a wire value is sent only where the
  vendor's documentation, its own listing, or a live test shows it.
* The rules hold: R9 (sealed `Item`), R21 (named constants), R23, R25 to
  R27 (errors), R28, R44, R45, R50 to R52, R55.

## Considered Options

The owner chose among three options per question on 2026-10-24; the chosen
one is first in each.

* **Headers (work item 6):** close it, a caller's `RoundTripper` through
  `WithHTTPClient` being the supported way (chosen); add `WithHeaderFunc` and
  amend R28; defer.
* **Chat Completions constructor (work item 5):** a generic
  `providers/chatcompat` outside `Default()`, amending R22 and R38 (chosen);
  named built-ins instead; defer.
* **Offline (work item 3):** `ErrOffline` unwrapping to
  `ErrProviderUnavailable` on every send path (chosen); a kind of its own;
  generation only.
* **1-hour cache-write rate (work item 7):** defer until models.dev
  publishes it (chosen); derive it for Anthropic; add an empty field.
* **`EffortMax` (work item 4):** degrade with a warning where the model does
  not list `max` (chosen); always send `max`; refuse.
* **Image input (work item 1):** a sealed `ImageItem`, `Capabilities.Images`,
  refusal before the network, a dropped image warned (chosen); always fail;
  defer.
* **Argument deltas (the issue's Q1):** keep 0031-MADR Q4 closed (chosen); in
  this record; a later record.
* **Records and release:** one MADR and one release, `v1.23.0` (chosen); one
  MADR with phased releases; one MADR per item.

## Decision Outcome

Chosen options as above, because each closes a copy gobble-cli keeps
without a rule broken that the record does not amend, and the evidence
supports each wire value. Three points the owner's first answers left open,
found while proving this record's facts, are Q9 to Q11; the owner answered
each (a) on 2026-10-24, and the PLAN's Phase 0 marked this record
`accepted`.

### D1. `ErrEmptyAnswer` (work item 2)

* In `llmprovider/api_error.go`'s sentinel block, beside `ErrIncomplete`:
  `ErrEmptyAnswer error = &sentinelError{msg: "llmprovider: empty answer", parent: ErrIncomplete}`,
  the pattern its neighbours use (C6, C7). It matches `ErrIncomplete` and,
  through it, `ErrInvalidRequest`, so every existing `errors.Is` still
  matches, and `Retryable()` is unchanged (not retried).
* `wire.EmptyAnswer` sets `Kind: llmprovider.ErrEmptyAnswer`. Its `Message`
  and `Reason` are unchanged, so gobble-cli's suffix match keeps working until
  it moves to the sentinel.
* Its doc says a refusal (`Reason` `content_filter`) and a cut answer
  (`length`) are this sentinel too, told apart by `APIError.Reason`.

### D2. `ErrOffline` (work item 3; Q3 (a))

* `ErrOffline error = &sentinelError{msg: "llmprovider: offline", parent: ErrProviderUnavailable}`
  in `api_error.go`'s block. It is retried, as `ErrProviderUnavailable` is
  (C10).
* **The rule,** gobble-cli's own (G2 to G4), moved here: an error is offline
  when its chain holds a `*url.Error` and either a `*net.DNSError`, or a
  `*net.OpError` whose `Op` is `"dial"` and that timed out or whose cause is
  connection refused, network down, network unreachable or host unreachable
  (`ECONNREFUSED`, `ENETDOWN`, `ENETUNREACH`, `EHOSTUNREACH`; on Windows the
  Winsock codes 10061, 10050, 10051, 10065). One condition is added: the
  request's context has not ended. A dial cut short by the caller's own
  deadline or cancel is not offline (it keeps the context's error, and
  `WithRetry` does not resend it). A TLS failure, a proxy's `proxyconnect`
  failure, and any HTTP response, a 5xx included, are not offline.
* **Where:** `internal/transport` gains `Offline(err error) bool`, the rule,
  on the standard library only (R2's row for `transport`), and
  `WrapOffline(ctx context.Context, err, offline error) error`, which returns
  `fmt.Errorf("%w: %w", offline, err)` when the context is live and the rule
  holds, else `err`; it takes the sentinel because `transport` cannot import
  `llmprovider` (C14). Every send site of M4 calls it with
  `llmprovider.ErrOffline`. In `wire.Post` it runs after the `Unsendable` and
  after-reply checks, which keep precedence (C13).
* **The behaviour change,** for the changelog: a refused or unresolved dial
  now matches `ErrProviderUnavailable` and `ErrOffline`, where today it
  matches neither (F3). It is still retried by `WithRetry`, as a `*url.Error`
  is today (C11).

### D3. `EffortMax` (work item 4; Q5 (a), Q9)

* `EffortMax Effort = "max"`, after `EffortXHigh`, and `valid()` accepts it
  (C18). Its doc: the highest tier a service names, above `xhigh` where the
  service has both.
* **The rule, every provider:** for a request whose effort is `EffortMax`,
  the provider reads the model's facts (each already does, per request):
  1. the facts list efforts and include `max`: the wire's `max` form is sent;
  2. the facts list efforts and not `max`: the highest listed effort below
     it in the order `low`, `medium`, `high`, `xhigh` is sent (`xhigh` where
     listed, Q5 (a); otherwise the highest listed below it, Q9 (a)), and
     the `Response` carries a
     `Warning{Kind: WarningEffortDegraded}` naming what was sent;
  3. the facts list no efforts: the wire's `max` form is sent, except on a
     wire with no `max`, as the table says, with the warning.
* **Per provider,** from each wire's documentation (External evidence):

  | Provider, route | `max` sent as | Without `max` in the facts | Evidence |
  | :--- | :--- | :--- | :--- |
  | Claude; Kilo's and OpenCode's messages routes | `output_config.effort: "max"` on an effort-only model; a budget model sends no effort, as for `xhigh` today | rule 2 | E3; C24 |
  | OpenAI (Platform, ChatGPT, plan); OpenCode's responses route | `reasoning.effort: "max"` | rule 2 | E5 |
  | Grok | xAI has no `max`: `xhigh`, by its clamp with `max` above `xhigh` in its order, warned | rule 2 | E4; C21 |
  | Gemini; OpenCode's google route | Gemini has nothing above `high`: `high`, as `xhigh` is today, warned | rule 2 | E8; C22 |
  | Ollama | `"max"`, as `xhigh` already is | rule 2 | E9; C23 |
  | Hugging Face, Together, Kilo's chat route, OpenCode's chat route | `reasoning_effort` or Kilo's `reasoning.effort`: `"max"` | rule 2 | E10, E11, E12; M2 |
  | `chatcompat` | `reasoning_effort: "max"` | rule 3 (an unknown id has no facts) | E5 |

* `WarningEffortDegraded WarningKind = "effort_degraded"` joins
  `warning.go` (C25). A provider's package doc lists its mapping under
  Degradations (R12).
* Out of this decision: an `xhigh` request to a model that lists `max` and not
  `xhigh` (F7) is sent as today.

### D4. `providers/chatcompat` (work item 5; Q2 (a))

* **The package** `llmprovider/providers/chatcompat` speaks Chat Completions
  through `internal/wire/chatcompletions`, as Hugging Face and Together do.
  R2's provider row covers it (F11); `docs/architecture.md` gains its row.
* **Exports:**
  * `New(opts ...llmprovider.Option) (llmprovider.Provider, error)`: one
    constructor, options only, no network call (R14, R40);
  * `WithID(id llmprovider.ProviderID) llmprovider.Option`, required;
  * `Descriptor(id llmprovider.ProviderID, label string) llmprovider.Descriptor`,
    so a caller registers it: `SupportsBaseURL` true, `RequiresAPIKey` false,
    no `AuthMethods` beyond the API key, no `StaticModels`.
* **`New` refuses,** with `ErrInvalidProvider` (as `Registry.Register` does,
  C32): no `WithID`, an empty id, or a built-in's id compared without case
  (the ten `Provider*` constants); and, with `ErrInvalidRequest` (R23), no
  `WithBaseURL`.
* **The key is optional.** With `WithAPIKey` or `WithTokenSource` the token
  goes in `Authorization: Bearer`; with neither, no `Authorization` header is
  sent. The issue names LM Studio, llama.cpp and vLLM, local servers a caller
  runs without a key; a remote server that needs one answers 401, which is
  `ErrAuthFailure`. This differs from the built-ins, which require a token
  where their service does, and the package doc says so.
* **Requests:** `POST {base}/chat/completions`; `ListModels` is
  `GET {base}/models` (`ModelLister`), probes off unless
  `WithModelProbes(true)`. `catalog.List` refuses an unlisted id with
  `ErrUnsupported` (C33, C34), the path on which a caller such as `wizard`
  asks the provider's own `ListModels`.
* **Capabilities:** Tools `Supported`, ForcedToolChoice `BestEffort`,
  Reasoning `BestEffort`, Continuation `Unsupported`, NativeStreaming
  `Supported`, Images `Supported` (D5). A server's refusal is its own error.
  Model facts: none (an unknown id); the package doc says what that leaves
  undone.
* **`providers.Default()` does not hold it.** **R22** is amended: "Built-in
  provider ids are the models.dev registry keys already in use; a provider a
  caller builds with `chatcompat` takes the caller's id." **R38** is amended:
  "… A provider outside this module needs no change here: its caller
  registers it in a `Registry`; so does a `chatcompat` provider, whose package
  is here but which `providers.Default()` does not list."

### D5. `ImageItem` and `Capabilities.Images` (work item 1; Q6 (a), Q10, Q11)

* **The item,** in `llmprovider/item.go` (R9):

  ```go
  // ImageItem is an image in a user turn of Request.Input. Only a request
  // holds one. Exactly one of Data and URL is set.
  type ImageItem struct {
      Role      Role   // RoleUser; empty is the user's
      MediaType string // image/png, image/jpeg, image/gif or image/webp
      Data      []byte // the image's bytes, sent base64-encoded
      URL       string // an https URL the service fetches
  }
  ```

  Its concurrency sentence is `MessageItem`'s (R51). `itemKind` returns a new
  `itemTypeImage`.
* **Validation,** in `Request.validate` (C2), each an `ErrInvalidRequest`
  before the network: a role other than `RoleUser` or empty; both `Data` and
  `URL`, or neither; a `URL` that is not an absolute `https` URL; `Data` with
  a `MediaType` other than the four. A `URL` may leave `MediaType` empty.
  Size is not checked: the services' limits differ (E1, E6, E7, E13) and
  each refuses its own.
* **`Capabilities.Images Support`,** the sixth field. `Check` refuses a
  request holding an `ImageItem` when it is `Unsupported`, with
  `ErrUnsupported` (C5). `llmtest.Run`'s needs table gains it (C39).
* **A model that does not take images** is refused before the generation
  request, with `ErrUnsupported`, when its facts state its input and image is
  not among it (C44): Anthropic's listing's `image_input`, Ollama's
  `/api/show` `vision`, models.dev's `modalities.input`.
* **Joining a turn,** per wire. The image keeps its place among the user's
  items:

  | Wire | An image is sent as | It joins |
  | :--- | :--- | :--- |
  | Messages | `{"type":"image","source":{"type":"base64","media_type":M,"data":B64}}`, or `{"type":"image","source":{"type":"url","url":U}}` | the user message just before it, whose text becomes a text block; else a user message of its own |
  | Chat Completions | `{"type":"image_url","image_url":{"url":"data:M;base64,B64"}}`, or `{"url":U}` | the user message just before it, whose text becomes a `text` part; else its own |
  | Responses | `{"type":"input_image","image_url":"data:M;base64,B64"}`, or `"image_url":U` | the user input item just before it, whose text becomes an `input_text` part; else its own |
  | generateContent | `{"inline_data":{"mime_type":M,"data":B64}}` | the user content just before it, as a part |
  | Interactions | `{"type":"image","data":B64,"mime_type":M}`, or `{"type":"image","uri":U}` | the `user_input` step just before it, as a content block |

  Evidence: E1, E2 (Messages); E6 (Chat Completions, Responses); E7
  (generateContent's inline data); E14 (Interactions). The G-wire goldens for
  these change, an R45 exception this record names (D11).
* **Per provider:**

  | Provider | `Images` | What it does |
  | :--- | :--- | :--- |
  | Claude | `Supported` | Messages; JPEG, PNG, GIF, WebP (E1) |
  | OpenAI, Platform | `Supported` | Responses; data or https URL (E6) |
  | OpenAI, ChatGPT and plan sessions | `Unsupported` (Q11 (a)) | UNCONFIRMED: no documentation of images on the Codex backend; a live test run by the owner decides an amendment |
  | Grok | `Supported` | Responses' `input_image`, its `image_url` a string (E4b); xAI documents JPEG and PNG only, so GIF and WebP are refused with `ErrUnsupported` before the network (Q10) |
  | Gemini | `Supported` | Interactions (E14) |
  | Hugging Face | `Supported` | Chat Completions' object form (E10); data and https URLs UNCONFIRMED, pinned by a live test |
  | Together | `Supported` | data and https URLs (E11) |
  | Kilo | `Supported` | its chat route's object form, base64 shown (E12); its messages route as Claude's |
  | Ollama | `BestEffort` | a data URL, with `image_url` a bare string as Ollama documents it; an https URL is dropped, warned, since Ollama marks image URLs unsupported (E9) |
  | OpenCode Zen and Go | `BestEffort` | each route's form; on the google route an https URL is dropped, warned (generateContent's `file_data` taking a web URL is UNCONFIRMED, E7); OpenCode documents images on no route (E15) |
  | `chatcompat` | `Supported` | the Chat Completions object form; the server decides |

* `WarningImageDropped WarningKind = "image_dropped"`; its `Detail` says which
  image and why. A provider's package doc lists what it drops (R12).
* **Not here:** decoding, resizing or EXIF (gobble-cli's); a Files API id;
  image output.

### D6. Headers (work item 6; Q1 (a))

No new API. A caller that sets headers per request does it in its own
`http.RoundTripper`, given through `WithHTTPClient`, as gobble-cli does
(G7); the caller tags each request's context with its model itself. R28 and
R32 (C45) are unchanged. The package doc of `llmprovider` says so in a
sentence under `WithHTTPClient`.

### D7. The 1-hour cache-write rate (work item 7; Q4 (a))

Deferred: `ModelRates` is unchanged until models.dev publishes a 1-hour
cache-write price (F14). A later record adds the field and its source.

### D8. Argument deltas (the issue's Q1; Q7 (a))

0031-MADR Q4 stands: a tool call arrives whole, as one event.

### D9. One release (Q8 as answered)

Everything here ships together as `v1.23.0`, after the last phase. The
phases land on `main` as they pass; only the tag waits. gobble-cli adopts
the tag in its own record.

### D10. The exports and their call sites (R55)

| Export | Package | Call site that needs it |
| :--- | :--- | :--- |
| `ErrEmptyAnswer` | `llmprovider` | `wire.EmptyAnswer` (C8); gobble-cli's empty-answer match (G1) |
| `ErrOffline` | `llmprovider` | the 16 send sites of M4 that call `Do` (the amendment "Phase 2's corrections"); gobble-cli's `offline` and its wrap (G2, G5) |
| `EffortMax` | `llmprovider` | each provider's effort mapping (D3); gobble-cli's `reasoningFor` (G6) |
| `WarningEffortDegraded` | `llmprovider` | the providers' rule 2 and wire mappings (D3) |
| `ImageItem` | `llmprovider` | the five encoders (D5); gobble-cli's image content and its `read` tool (G8, G9) |
| `Capabilities.Images` | `llmprovider` | `Check` and `llmtest.Run` (C5, C39); gobble-cli's ACP `image` capability |
| `WarningImageDropped` | `llmprovider` | Ollama's and OpenCode's drops (D5) |
| `New`, `WithID`, `Descriptor` | `providers/chatcompat` | gobble-cli's `models.json` providers it skips today (G10) |
| `SharedOption` | `llmprovider` | `catalog.WithCache` and `chatcompat.WithID` (the amendment "Phase 4's options") |
| `Discover`, `Server` | `providers/chatcompat` | a caller's configure step: gobble-cli's, the wizard's (the amendment "`chatcompat`, provider-agnostic", E5) |
| `Catalog.Search`, `Match.Why` | `catalog` | the wizard's `searchCatalog` and its rows' `Detail` (the amendment "Model search, rebuilt", S1, S4) |

No other name is exported. `make api-check` against `v1.22.2` lists exactly
these.

### D11. Docs and tests

* `CHANGELOG.md` `[Unreleased]`: Added (the exports) and Changed (D2's
  behaviour change); then `[v1.23.0]`.
* `docs/architecture.md`: `chatcompat`'s package row and provider entry;
  "The wizard" unchanged; the G-wire bullet corrected to the counts after
  this record's scenarios (F16).
* `docs/guides/api-standards.md`: R22 and R38 as amended (D4).
* G-wire gains two scenarios, `image` (a user text, a PNG by `Data` and an
  https `URL`) and `effort-max`; their goldens are recorded with `-update`,
  citing this record (R45).
* R50: an example in `providers/chatcompat` that runs offline over
  `httptest`.

### Consequences

* Good, because gobble-cli can drop its empty-answer suffix match, its dial
  classifier, its `max`-to-`xhigh` map, its skip of unbuilt providers, and
  its "cannot send images" note.
* Good, because each wire's `max` and image form rests on its vendor's page,
  and every UNCONFIRMED point is either left out or pinned by a live test.
* Neutral, because headers stay a caller's transport, and the 1-hour rate
  waits for data.
* Bad, because one release waits for the largest item: gobble-cli keeps its
  copies until `v1.23.0`.
* Bad, because a dial failure now matches `ErrProviderUnavailable`, which a
  caller's existing branch may treat differently (D2).

### Confirmation

* `make api-check` against `v1.22.2`: additions only, exactly D10's.
* `llmtest.Run` passes for every built-in and for `chatcompat`, with the
  images need.
* The G-wire goldens of `image` and `effort-max` match D3's and D5's tables
  for every case.
* `errors.Is` tests: `ErrEmptyAnswer` for each wire's no-content path, still
  matching `ErrIncomplete`; `ErrOffline` for a refused dial
  (`nettest.Refusing`, C15), an unresolved name and a dial timeout, and not
  for a 503, a TLS failure or a dial the caller's context ended.
* Each new test fails first on `d9b1910`'s code or on a planted break.
* The live tests the PLAN names, run by the owner, for the UNCONFIRMED
  points.

## Pros and Cons of the Options

### Headers: close, add a hook, defer

* Close: Good, because no rule changes and gobble-cli's transport already
  works. Bad, because the transport stays in gobble-cli.
* Hook: Good, because gobble-cli drops a transport. Bad, because R28 changes
  for one consumer.
* Defer: Neutral; the item stays open.

### Chat Completions: generic, named built-ins, defer

* Generic: Good, because any server, local ones included, is reachable. Bad,
  because R22 and R38 are amended, and no model facts exist for it.
* Named: Good, because ids stay models.dev keys. Bad, because each server is
  a package, and a local server has none.
* Defer: Bad, because gobble-cli keeps skipping providers.

### Offline: unwrap to unavailable, own kind, generation only

* Unwrap on all paths: Good, because R25 maps onto an existing kind and
  listing and generation agree. Bad, because existing branches see a new
  match.
* Own kind: Good, because no branch changes. Bad, because R25 prefers an
  existing kind.
* Generation only: Bad, because listing and generation disagree.

### 1-hour rate: defer, derive, empty field

* Defer: Good, because nothing is invented. Bad, because gobble-cli prices as
  today.
* Derive: Bad, because a price rule is assumed.
* Empty field: Bad, because a field that is always zero misleads.

### `EffortMax`: degrade and warn, always send, refuse

* Degrade: Good, because a request is answered and the caller is told. Bad,
  because the facts must be read, which every provider already does.
* Always send: Bad, because a service that lacks `max` refuses it (E4, E8).
* Refuse: Bad, because gobble-cli retries at a lower level itself.

### Images: as proposed, always fail, defer

* As proposed: Good, because each provider sends what its wire takes and says
  what it drops. Bad, because every wire and its goldens change.
* Always fail: Bad, because a mixed gateway refuses outright.
* Defer: Bad, because gobble-cli's 1.0 gate waits on it (the issue).

### Argument deltas: closed, here, later

* Closed: Good, because the event set is unchanged. Bad, because ACP's
  `tool_call_update` does not stream arguments.

### Release: one, phased, per item

* One release: Good, because one tag. Bad, because the small items wait.

## Owner questions

* **Q1 (D6) to Q8 (D9).** Answered 2026-10-24 by the owner through two
  prompts of three options each: Q1 (a) headers closed; Q2 (a) `chatcompat`
  generic outside `Default()`; Q3 (a) `ErrOffline` unwraps, all paths; Q4
  (a) the 1-hour rate deferred; Q5 (a) `EffortMax` degrades with a warning;
  Q6 (a) images as the issue proposes; Q7 (a) argument deltas closed; Q8
  one MADR and one release.
* **Q9 (D3 rule 2). `max` on a model whose facts list neither `max` nor
  `xhigh`** (M1, M2: `claude-opus-4-5-20251101` lists `low`, `medium`,
  `high`; many models list `low`, `high`, `max`).
  * **(a)** Send the highest listed effort below `max`, warned: the same
    nearest-lower clamp Grok already applies (C21) (recommended).
  * **(b)** Send `xhigh` whatever the facts list, warned, and let the service
    refuse or map it.
  * **(c)** Refuse with `ErrUnsupported` before the network.

  **Answered 2026-10-24: (a).**
* **Q10 (D5). Grok and GIF or WebP.** xAI's guide documents JPEG and PNG
  only; its OpenAPI's Chat Completions schema mentions WebP (E4b).
  * **(a)** Refuse GIF and WebP for Grok with `ErrUnsupported` before the
    network, until xAI documents them (recommended).
  * **(b)** Send them and let xAI refuse.
  * **(c)** Drop them with `WarningImageDropped`.

  **Answered 2026-10-24: (a).**
* **Q11 (D5). Images on a ChatGPT or plan session.** Nothing documents image
  input on the Codex backend; `openai.New` already sets other capabilities
  per session kind.
  * **(a)** `Unsupported` for those sessions until a live test, run by the
    owner, shows them accepted; an amendment then enables it (recommended).
  * **(b)** `Supported` now, as on the Platform.
  * **(c)** `BestEffort`, dropping images with a warning.

  **Answered 2026-10-24: (a)** ("q9: a, q10: a, q11: a. committed,
  proceed").

## More Information

### External evidence (fetched 2026-10-24)

Each quote is as this record's research copied it from the fetched page:
E5 and E6 from the pages downloaded and stripped to text; the others through
a fetch tool's extraction, which gives the page's words but not its raw
markup. Three on which a decision rests were checked again against the raw
page on 2026-10-24: E3's "Not every model that supports" (Anthropic's effort
page), E5's enum sentence (both OpenAI references), and E9's "Image URL"
item, unchecked (`disabled:true`, not `checked`) where "Base64 encoded
image" is listed. The PLAN's Phase 7 pins each wire value with a live test.

| Id | Vendor, point | URL | Quote |
| :--- | :--- | :--- | :--- |
| E1 | Anthropic: image block and media types | <https://platform.claude.com/docs/en/build-with-claude/vision> | "provide images to Claude as `image` content blocks using one of three source types: 1. A base64-encoded image … 2. A URL reference to an image hosted online 3. A `file_id`"; "Claude supports JPEG, PNG, GIF, and WebP images" |
| E2 | Anthropic: text and images in one message | same | "Images placed after text or interpolated with text still perform well" |
| E3 | Anthropic: effort values | <https://platform.claude.com/docs/en/build-with-claude/effort> | "Values are the level names (`low`, `medium`, `high`, `xhigh`, and `max`)."; "Not every model that supports `max` supports `xhigh`." |
| E4 | xAI: effort values | <https://docs.x.ai/developers/model-capabilities/text/reasoning> | the table lists `low`, `medium`, `high`, `xhigh`; "On models that do not support it, such as `grok-4.5`, requests with `\"xhigh\"` are treated as `\"high\"`." No `max` on the page or in <https://docs.x.ai/openapi.json>. |
| E4b | xAI: image input | <https://docs.x.ai/developers/model-capabilities/images/understanding> | `{"type":"input_image","image_url":"data:image/jpeg;base64,…"}`; "The `image_url` value can also be a public URL on the Internet instead of a base64 data URL."; "Supported image file types: `jpg/jpeg` or `png`." |
| E5 | OpenAI: effort values | <https://developers.openai.com/api/reference/resources/responses/methods/create> | "Currently supported values are none , minimal , low , medium , high , xhigh , and max ." (the Chat Completions reference reads the same) |
| E6 | OpenAI: image parts | <https://developers.openai.com/api/docs/guides/images-vision>; the Chat Completions and Responses references | Chat: "url : string Either a URL of the image or the base64 encoded image data."; Responses: "image_url … A fully qualified URL or base64 encoded image in a data URL." |
| E7 | Gemini generateContent: inline data | <https://ai.google.dev/api/generate-content> | "data string ( bytes format) Raw bytes for media formats. A base64-encoded string."; `file_data` with a web URL: UNCONFIRMED (the reference's examples use Files API URIs only) |
| E8 | Gemini: thinking levels | <https://ai.google.dev/api/interactions-api> | "thinking_level … Possible values minimal … low … medium … high" |
| E9 | Ollama: OpenAI compatibility | <https://docs.ollama.com/api/openai-compatibility> | "[x] Base64 encoded image", "[ ] Image URL"; examples write `"image_url": "data:image/png;base64,…"` as a string; "`\"xhigh\"` or `\"ultra\"` map to `\"max\"`" |
| E10 | Hugging Face router | <https://huggingface.co/docs/inference-providers/tasks/chat-completion> | `image_url*` object with `url*` string; "reasoning_effort … Common values: none, minimal, low, medium, high, xhigh." |
| E11 | Together | <https://docs.together.ai/docs/openai-api-compatibility>; <https://docs.together.ai/docs/inference/chat/reasoning> | "Vision models accept `image_url` with both remote URLs and base64 data URIs."; "DeepSeek V4 Pro 0813 accepts only `\"high\"` and `\"max\"` for `reasoning_effort`." |
| E12 | Kilo gateway; OpenRouter | <https://kilo.ai/docs/gateway/api-reference>; <https://openrouter.ai/docs/guides/best-practices/reasoning-tokens> | Kilo: `{ type: "image_url"; image_url: { url: string; detail?: string } }`, "The gateway forwards each request to OpenRouter"; OpenRouter: effort "Can be \"max\", \"xhigh\", \"high\", \"medium\", \"low\", \"minimal\" or \"none\"" |
| E13 | Gemini: inline size | <https://ai.google.dev/gemini-api/docs/image-understanding> | "Inline image data limits your total request size … to 20MB." |
| E14 | Gemini Interactions: image content | <https://ai.google.dev/api/interactions-api> | "ImageContent An image content block. type … Always set to \"image\" . data string (optional) … uri string (optional) … mime_type" |
| E15 | OpenCode Zen and Go | <https://opencode.ai/docs/zen/>; <https://opencode.ai/docs/go/> | the routes' endpoints (Chat Completions, Responses, Messages, Google); images and reasoning effort are not documented on either page |

### Measurements

* **M1.** The first-party listing `llmprovider/catalog/testdata/firstparty-2026-10-14/claude.json`,
  read by a scratch script: of 14 Claude models, 9 list `low` to `max`;
  `claude-sonnet-4-6` and `claude-opus-4-6` list `low`, `medium`, `high`,
  `max`; `claude-opus-4-5-20251101` lists `low`, `medium`, `high`; two list
  none; all 14 report `image_input` supported.
* **M2.** The models.dev snapshot of 2026-10-12, its `reasoning_options`
  effort values per section: `max` appears in `anthropic`, `openai`,
  `huggingface`, `kilo`, `opencode`, `opencode-go` and `togetherai`; not in
  `xai`. Common lists include `['low','high','max']` and
  `['low','medium','high']`.
* **M3.** The keys under every model's `cost` in both recorded models.dev
  snapshots: `input`, `output`, `cache_read`, `cache_write`, `reasoning`,
  `input_audio`, `output_audio`, `context_over_200k.*`, `tiers[].*`. None is
  a 1-hour write.
* **M4.** `.Do(` and `.RoundTrip(` in non-test Go files: `internal/wire/post.go:144`;
  `catalog/discovery.go` lines 709, 810, 898, 956, 1002, 1095, 1197, 1294;
  `catalog/raw.go` 142, 167, 202; `catalog/listing_metadata.go:369`;
  `catalog/model_metadata.go:535`; `providers/openai/chatgpt_listing.go:86`;
  `auth/oauth_session.go:587`; `auth/request_bound.go:21`. Not counted:
  `internal/cmd/catalogs` (a maintainer tool) and `llmtest` (the harness).
* **M5.** G-wire: 20 case directories under the providers' `testdata/wire`,
  12 scenarios in `wirecase.go`, 227 golden files.

### Related

* [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md):
  the contract, D3 to D11.
* [0034-MADR-degrade-forced-tool-choice-per-model.md](0034-MADR-degrade-forced-tool-choice-per-model.md):
  the `Warning` pattern.
* [0041-MADR-decide-per-model-behaviour-from-metadata.md](0041-MADR-decide-per-model-behaviour-from-metadata.md):
  per-model facts.
* [0058-MADR-test-cache-isolation-for-reused-ports.md](0058-MADR-test-cache-isolation-for-reused-ports.md):
  the tests' own caches.
* gobble-cli's `0031-MADR-sdk-current-releases-and-adoption.md` (D9) and
  `0005-MADR-v1-feature-scope.md`, as issue #9 cites them.

### Code claims

<!-- claims: self=d9b1910 gobble-cli=142685b -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | Five Item types, sealed | `llmprovider/item.go:13` | `FunctionCallOutputItem, ReasoningItem, and ToolLoadItem, which only a` |
| C2 | validate refuses any other Item type | `llmprovider/contract.go:277` | `"FunctionCallOutputItem, ReasoningItem or ToolLoadItem value", ErrInvalidRequest, i, item)` |
| C3 | Capabilities' last field | `llmprovider/capabilities.go:46` | `NativeStreaming Support` |
| C4 | Support's zero value is Unsupported | `llmprovider/capabilities.go:10` | `Unsupported Support = iota` |
| C5 | Check's needs table | `llmprovider/contract.go:239` | `{req.PreviousResponseID != "", c.Continuation, "continuation"},` |
| C6 | The sentinel-with-parent type | `llmprovider/api_error.go:23` | `type sentinelError struct {` |
| C7 | ErrIncomplete matches ErrInvalidRequest | `llmprovider/api_error.go:47` | `ErrIncomplete error = &sentinelError{msg: "llmprovider: incomplete response", parent: ErrInvalidRequest}` |
| C8 | The empty answer's message | `llmprovider/internal/wire/finish.go:34` | `Message: where + ": the answer has no content"}` |
| C9 | ErrProviderUnavailable | `llmprovider/provider.go:62` | `ErrProviderUnavailable = errors.New("llmprovider: provider unavailable")` |
| C10 | Unavailable is retried | `llmprovider/retry.go:329` | `case errors.Is(err, ErrRateLimited), errors.Is(err, ErrProviderUnavailable):` |
| C11 | A kindless send failure is retried | `llmprovider/retry.go:339` | `return errors.As(err, &urlErr) && !transport.Unsendable(err)` |
| C12 | wire.Post sends | `llmprovider/internal/wire/post.go:144` | `resp, err := c.Client.Do(req)` |
| C13 | A failure to reach the service keeps its own error | `llmprovider/internal/wire/post.go:150` | `// request was written, such as a timeout awaiting the reply's` |
| C14 | llmprovider imports transport | `llmprovider/api_error.go:18` | `"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"` |
| C15 | The refused-dial test client | `internal/nettest/nettest.go:35` | `return nil, &net.OpError{Op: "dial", Net: network, Err: syscall.ECONNREFUSED}` |
| C16 | Ollama's listing wraps Unavailable | `llmprovider/catalog/discovery.go:902` | `return nil, fmt.Errorf("%w: could not reach Ollama at %s: %w", llmprovider.ErrProviderUnavailable, baseURL, err)` |
| C17 | The OAuth refresh wraps Unavailable | `llmprovider/auth/oauth_session.go:589` | `return nil, llmprovider.Token{}, true, fmt.Errorf("%w: oauth: refresh request: %w", llmprovider.ErrProviderUnavailable, err)` |
| C18 | The effort set | `llmprovider/contract.go:80` | `case EffortLow, EffortMedium, EffortHigh, EffortXHigh:` |
| C19 | Facts may name max | `llmprovider/catalog/model_info.go:60` | `and with its names: "none", "minimal" and "max" among them, which` |
| C20 | Anthropic's listing gives max | `llmprovider/catalog/listing_metadata.go:190` | `{"xhigh", e.XHigh}, {"max", e.Max}} {` |
| C21 | Grok's effort order | `llmprovider/providers/grok/reasoning.go:29` | `var effortOrder = []string{effortLow, effortMedium, effortHigh, effortXHigh}` |
| C22 | Gemini sends xhigh as high | `llmprovider/providers/gemini/interactions.go:104` | `// xhigh sent as high; anything else is omitted, leaving the model's default.` |
| C23 | Ollama sends xhigh as max | `llmprovider/providers/ollama/ollama.go:87` | `maxEffort = "max"` |
| C24 | Messages sends the effort as given | `llmprovider/internal/wire/messages/messages.go:621` | `fields["output_config"] = map[string]any{keyEffort: effort}` |
| C25 | The Warning kinds | `llmprovider/warning.go:12` | `WarningForcedChoiceDegraded WarningKind = "forced_tool_choice_degraded"` |
| C26 | Messages: a MessageItem opens its own message | `llmprovider/internal/wire/messages/messages.go:235` | `messages = append(messages, Message{Role: role, Content: Content{text: arena.Keep(v.Text)}})` |
| C27 | Messages: Block has no source | `llmprovider/internal/wire/messages/messages.go:84` | `// redacted_thinking, tool_use, tool_result and tool_reference. A field a` |
| C28 | Chat Completions: a message's content is text | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:280` | `messages = append(messages, chatMessage{Role: role, Content: v.Text})` |
| C29 | Responses: a message's content is text | `llmprovider/internal/wire/responses/responses.go:105` | `input = append(input, InputItem{Role: role, Content: arena.Keep(v.Text)})` |
| C30 | generateContent: a message is one text part | `llmprovider/internal/wire/generatecontent/generatecontent.go:132` | `appendPart(role, Part{Text: arena.Keep(v.Text)}, false)` |
| C31 | Interactions: a message is one text step | `llmprovider/providers/gemini/interactions.go:73` | `steps = append(steps, interactionTextStep(interactionStepUserInput, v.Text))` |
| C32 | Register refuses a duplicate | `llmprovider/registry.go:76` | `return fmt.Errorf("llmprovider: provider %q is already registered: %w", d.ID, ErrInvalidProvider)` |
| C33 | catalog.List refuses an unlisted id | `llmprovider/catalog/discovery.go:210` | `return Catalog{}, unlistedProvider(id)` |
| C34 | as ErrUnsupported | `llmprovider/catalog/discovery.go:643` | `return fmt.Errorf("model listing: unsupported provider %q: %w", id, llmprovider.ErrUnsupported)` |
| C35 | R22 | `docs/guides/api-standards.md:184` | `- **R22. Provider ids are the models.dev registry keys** already in use.` |
| C36 | R28: no hooks | `docs/guides/api-standards.md:213` | `same shape. There are no hooks inside providers. (0015 D8; 0044 D1; 0051` |
| C37 | R38 | `docs/guides/api-standards.md:250` | `` - **R38. Adding a provider is one package, one entry in `providers.Default()`'s `` |
| C38 | R2's provider row | `docs/guides/api-standards.md:47` | `` \| `llmprovider/providers/<id>` \| one provider or gateway family: `New`, its options, its `Descriptor` \| `llmprovider`, `auth`, `catalog`, internal packages \| `` |
| C39 | llmtest's needs loop | `llmprovider/llmtest/llmtest.go:332` | `{"continuation", caps.Continuation, withPrevious(textRequest()), false},` |
| C40 | G-wire's scenarios | `llmprovider/internal/wirecase/wirecase.go:258` | `var scenarios = []scenario{` |
| C41 | The architecture's G-wire count | `docs/architecture.md:974` | `` listing): 114 golden files. It runs them against `internal/wiretest`, and `` |
| C42 | ModelRates' one cache write | `llmprovider/catalog/model_info.go:79` | `CacheRead, CacheWrite float64` |
| C43 | The cost document's fields | `llmprovider/catalog/model_metadata.go:63` | `` CacheWrite      *float64        `json:"cache_write"` `` |
| C44 | Image input in the facts | `llmprovider/catalog/model_info.go:124` | `ModalityImage Modality = "image"` |
| C45 | WithHTTPClient's single client | `docs/guides/api-standards.md:230` | `- **R32. One default HTTP client per provider instance,** serving requests,` |
| C46 | The v1.22.0 section | `CHANGELOG.md:89` | `## [v1.22.0] - 2026-10-22` |
| C47 | ShowDeviceCode added there | `CHANGELOG.md:93` | `` - `wizard.Options.ShowDeviceCode` and `wizard.DeviceCode`: a device `` |
| G1 | gobble matches the empty answer's text | `gobble-cli:llm/provider/provider.go:523` | `const sdkEmptyAnswer = ": the answer has no content"` |
| G2 | gobble classifies offline | `gobble-cli:llm/provider/provider.go:548` | `func offline(err error) bool {` |
| G3 | its Winsock codes | `gobble-cli:llm/provider/provider.go:570` | `const wsaENETDOWN, wsaENETUNREACH, wsaECONNREFUSED, wsaEHOSTUNREACH = 10050, 10051, 10061, 10065` |
| G4 | its Unix codes | `gobble-cli:llm/provider/provider.go:573` | `return []syscall.Errno{syscall.ECONNREFUSED, syscall.ENETDOWN, syscall.ENETUNREACH, syscall.EHOSTUNREACH}` |
| G5 | gobble wraps its ErrOffline | `gobble-cli:llm/provider/provider.go:633` | `wrapped = fmt.Errorf("%w: %w", llm.ErrOffline, err)` |
| G6 | gobble sends max as xhigh | `gobble-cli:llm/provider/provider.go:499` | `"high": llmprovider.EffortHigh, "xhigh": llmprovider.EffortXHigh, "max": llmprovider.EffortXHigh,` |
| G7 | gobble's headers transport | `gobble-cli:llm/provider/provider.go:157` | `out.Transport = headerTransport{base: base, headers: headers}` |
| G8 | gobble cannot send images | `gobble-cli:tool/builtin/read.go:31` | `const imageNotSent = "[The image was not sent: gobble cannot send images to models yet.]"` |
| G9 | gobble's image content | `gobble-cli:llm/llm.go:125` | `` MediaType    string         `json:"mediaType,omitzero"` `` |
| G10 | gobble skips an unbuilt provider | `gobble-cli:llm/catalog/modelsjson.go:196` | `return fmt.Sprintf("not a provider go-llmprovider-sdk %s builds (one of %s)", sdk, strings.Join(ProviderIDs(), ", "))` |
| G11 | gobble's SDK pin | `gobble-cli:go.mod:16` | `github.com/maccavelli/go-llmprovider-sdk v1.22.0` |

## Amendment 2026-10-24: Phase 2's corrections

Found while implementing D2; no decision changes.

* **Seventeen send calls, not fifteen.** M4 lists seventeen lines:
  `wire/post.go` 1, `catalog/discovery.go` 8, `catalog/raw.go` 3,
  `catalog/listing_metadata.go` 1, `catalog/model_metadata.go` 1,
  `providers/openai/chatgpt_listing.go` 1, `auth/oauth_session.go` 1,
  `auth/request_bound.go` 1. F12 said fifteen and D10 "the 15 send sites";
  the list was right, the count wrong.
* **Sixteen are wrapped; `raw.go:202` is not.** It is
  `rawRecorder.RoundTrip`, a `RoundTripper` that `listRawOnce` puts inside
  the client its ChatGPT, plan, Ollama and listing fetches send through
  (`raw.go:106-126`), each of which is a wrapped `Do` site. A `RoundTrip`'s
  error carries no `*url.Error` until `Client.Do` adds one, so D2's rule
  could never hold there: a wrap would be dead code. `ListRaw` with a
  refused dial matches `ErrOffline` (the PLAN's Phase 2 test).
* **The classification runs before the derived context is cancelled.**
  `wire.Post` and `auth`'s `doBounded` cancel their own context before
  returning a send failure, so `WrapOffline` runs first; `wire.Post`'s
  `Unsendable` and after-write checks still take precedence.
* **Windows' names beside Winsock's.** gobble-cli's rule counts only the
  Winsock codes on Windows (G3). The syscall package's `ECONNREFUSED`,
  `ENETDOWN`, `ENETUNREACH` and `EHOSTUNREACH` are values of their own on
  Windows, and `internal/nettest.Refusing` builds its refused dial from
  `syscall.ECONNREFUSED` (C15), so the rule counts both there: the four
  names and the four codes. `GOOS=windows go build ./...` and `go vet` pass.

## Amendment 2026-10-24: Phase 3's corrections

Found while implementing D3; no decision changes.

* **Kilo has no messages route.** D3's first row names "Kilo's and
  OpenCode's messages routes". Kilo sends only `POST
  {base}/chat/completions` (its package doc: "Only POST
  {base}/chat/completions is used"), so its `EffortMax` is the chat row's:
  `"max"` in `reasoning.effort` or `reasoning_effort`, rule 2.
* **Grok follows rule 1 where its facts list `max`.** With `max` above
  `xhigh` in `effortOrder`, `clampReasoningEffortFor` builds a known
  model's menu from the efforts its facts list, in that order, so a model
  whose facts list `max` is sent `max`, unwarned. xAI's listing lists no
  `max` (E4), so the row's "xAI has no `max`" holds for every model it
  lists. The row's `xhigh` is the clamp's highest: a model whose menu stops
  at `high`, such as `grok-4.5`, is sent `high`, warned.
* **Gemini's `high`.** Where `ResolveMax` returns `max` (facts that list
  it, or none), Gemini and OpenCode's google route send `high`. On the
  google route with no facts, `high` is the dynamic thinking budget, as
  `xhigh` is today; the warning names `high`.
* **A warning names an effort sent.** Where the provider omits the effort,
  as Grok does for a model off every menu with no facts and Gemini for a
  level the model does not list, no warning is added, as for any other
  effort.
* **Rule 3's edge.** Facts that list efforts but none of `low`, `medium`,
  `high` and `xhigh` (only `none` or `minimal`) send `max`, unwarned, as
  facts that list none.

## Amendment 2026-10-24: D3 reworked by 0062

The owner re-assessed the thinking design on 2026-10-24: parameters are to
be decided from facts and from the services' refusals, not from rules per
provider or model. That decision is
[0062-MADR-learn-thinking-parameters-from-refusals.md](0062-MADR-learn-thinking-parameters-from-refusals.md)
(its D10; the owner's Q2 (b) there).

* `EffortMax` and `WarningEffortDegraded` stand as D3 and D10 define them.
* Rules 1 and 2, which read the facts, become 0062's effort ladder.
* Rule 3's per-wire rows (Grok's clamp; Gemini's and OpenCode's google
  route's `high`) and the table's "`max` sent as" column are superseded by
  0062's D2 to D4. They are kept only where 0062's D8 keeps a rule.
* The PLAN's Phase 3 was committed on 2026-10-24 as `dd41faf`, rule 3's
  rows included. The owner had first chosen to hold it (0062's Q2 (b)), then
  committed it and kept the commit (0062-MADR's amendment "0061's Phase 3
  committed"). 0062-PLAN's Phase 2 reworks its code in place, in that
  phase's own commit. Until then, `main` sends `max` as rule 3's rows say.
* The PLAN's Phases 4 to 8 follow 0062-PLAN, and the release of Phase 8
  carries 0062.

## Amendment 2026-10-25: Phase 4's options

`chatcompat`'s provider id is the caller's, and `llmprovider.ResolveOptions`
decides, by that id, which provider-specific options a `New` takes and which
`For(id, …)` overlays apply (`llmprovider/settings.go:178`, `:273`). The
PLAN's Phase 4 stopped on two consequences D4 did not foresee:

* **`catalog.WithCache` would be refused.** It is scoped to the built-in
  ids (`catalog/cache.go:43`), as 0048-MADR D1 decided ("scoped as R17
  says"). R49 holds a provider's probes and learned refusals in the
  `catalog.Cache` that `WithCache` gives its `New`, and 0058-MADR D1 gives
  each test that probes its own; `chatcompat` would have neither.
* **`WithID` cannot be scoped.** A scoped option names its ids when it is
  made, and `chatcompat`'s is only known from `WithID` itself.

The owner chose, on 2026-10-25, a shared option (option "2a"), for
`WithCache` and `WithID` only:

* **D10 gains `llmprovider.SharedOption(name string, value any) Option`:**
  an option every provider's `New` takes, whatever its id; its value joins
  `Settings.Values` in order, for a provider that reads it, and one that
  does not ignores it. A `For(id, …)` holding it applies it to `id` alone,
  as any option. Call sites (R55): `catalog.WithCache` and
  `chatcompat.WithID`.
* **`catalog.WithCache` is a shared option.** Every `New` takes it: the
  built-ins and `chatcompat` read it; a provider outside this module cannot
  (`internal/cacheref`) and ignores it, where it refused it. 0048-MADR D1's
  "scoped as R17 says" no longer holds for it; R17 and R18 name the shared
  option. `catalog.WithLimit`, `WithProfile`, `WithWorkload` and
  `WithModelLayer` stay scoped to the built-ins, as 0038-MADR D2 and the
  wizard's `Options.ModelLimit` rely on.
* **`chatcompat.WithID` is a shared option, and `chatcompat.New` resolves
  the options twice:** first under the scope `"chatcompat"`, to read the last
  `WithID` and refuse as D4 says; then under that id, so that `For(id, …)`
  applies, and the provider is built from the second. A `WithID` inside
  `For(id, …)` is not read: it is given bare, as the factory a caller
  registers adds it.
* **Also settled for `New`, under rules D4 did not name:**
  * by R16, an `auth.OAuthSession` or `auth.VendorCLISession` is refused
    with `ErrUnsupported`, as Hugging Face's `New` refuses one: its token is
    a vendor's, and is never sent to another server;
  * probes are off unless `WithModelProbes(true)`: `New` puts
    `WithModelProbes(false)` before the caller's options, as `Settings`
    cannot tell its default from a given `true`;
  * an empty key is no key: no `Authorization` header is sent, as with no
    source (`Token.Apply` would send the scheme `Bearer` and an empty value).
* **Offered and not chosen:** one fixed scope, `"chatcompat"`, which
  `For(id, …)` would not reach; no catalog option, which breaks R49; an
  internal bridge in place of the export ("2b"); and `catalog.WithLimit`
  shared too, which would reverse 0038-MADR D2 and its A2's wizard path.

## Amendment 2026-10-25: `chatcompat`, provider-agnostic (proposed)

**Status: accepted (owner, 2026-10-25): all five, E1 to E5,** with the
amendment "Phase 4's options", which E1 builds on. D10's table gains
`SharedOption`, `Discover` and `Server`. The owner asked, before Phase
4 runs, that `chatcompat` serve any OpenAI-compatible server, not only one
shaped as Hugging Face's router, and that its caches, listings, facts and
search be robust and fast. This amendment evaluates D4 against that, and
proposes five enhancements, E1 to E5. Each is additive: no exported name
changes, and only E5 adds names.

### What D4 leaves undone

* **No catalog path.** `catalog.List`, `LookupModel`, `CachedModel` and
  `CachedFacts` refuse an id that is not built in (`catalog/lookup.go:46`,
  `lookupProvider`), so a `chatcompat` listing is fetched by its own code,
  uncached, unshared between callers, with no last good listing, no `Menu`
  and no facts. The built-ins have all of these: a 10-minute listing cache
  keyed by provider, base URL and credential fingerprint
  (`catalog/listing_cache.go:19`, `:36`), one fetch in flight per key
  (`claimListing`, 0053-MADR), a last good listing for 24 hours after a
  transient failure (`:53`), and Ollama's per-model details cached by digest.
* **No facts.** Every request of a `chatcompat` provider sends the default
  output limit, no replay field, and its effort unchecked against a list,
  though the servers it targets publish context windows and capabilities
  (External evidence below).
* **One request shape.** A server that refuses an optional field the
  Chat Completions wire sends (`stream_options`, a tool's `strict`,
  `reasoning_effort`) fails every request; 0062's learner reads only the
  refusal forms of Hugging Face and Kilo.
* **Search for curated listings.** `catalog.Search`'s tiers
  (`catalog/model_matcher.go:18-31`) suit a provider's tidy ids; a local
  server's ids carry tags and quantisations (measured: the owner's Ollama
  lists `granite3-dense:8b-instruct-q5_K_M` beside `granite3-dense:8b`), and
  a typo finds nothing until the subsequence tier.

### External evidence (searched 2026-10-25)

Measured in this session: the owner's local Ollama answers `GET /v1/models`
with `object` and `data`, each entry holding only `id`, `object`, `created`
and `owned_by`, `owned_by` being `"library"`, and no `Server` header.

The rest is reported by documentation or by clients that read it, and is
**UNCONFIRMED here until a phase records it** (each phase below records
before it relies):

* **llama.cpp's server:** `GET /v1/models` with `owned_by: "llamacpp"` and a
  `meta` block (`n_ctx`, `n_ctx_train`) on a single-model server; `GET
  /props` with the per-slot `n_ctx` and the vision modality
  ([simplified.guide](https://www.simplified.guide/llama-cpp/server-start),
  [pi-llamacpp-provider](https://pi.dev/packages/pi-llamacpp-provider?page=34));
  Zed reads `/props` for the context length
  ([Zed](https://zed.dev/docs/ai/use-a-local-model)).
* **LM Studio:** `GET /api/v0/models`, each entry's `type` (`llm`, `vlm`,
  `embeddings`), `state` (`loaded`, `not-loaded`) and `max_context_length`
  ([LM Studio](https://lmstudio.ai/docs/api/rest-api)); Zed fills its
  `max_tokens` from it
  ([Zed commit](https://git.secluded.site/zed/commit/52fa7ababb7feff1a3434e03e94d7a26b34b3d24)).
* **Ollama:** `POST /api/show`'s `capabilities` (`completion`, `tools`,
  `thinking`, `vision`) and `model_info`'s `*.context_length`
  ([Zed commit](https://git.secluded.site/zed/commit/3d5ddcccf08ea4d421b11474cf1190fc6bb1b4a1)),
  which this module's catalog already reads for Ollama.
* **OpenRouter-shaped listings:** `context_length`, `supported_parameters`
  and `architecture.input_modalities` in `GET /models`
  ([OpenRouter](https://openrouter.ai/docs/overview/models)).
* **vLLM** serves at `http://localhost:8000/v1` by its docs' example
  ([vLLM](https://docs.vllm.ai/en/v0.8.5/serving/openai_compatible_server.html));
  that its `/v1/models` carries `max_model_len` was not found in a primary
  source.
* **Text Generation Inference:** an `/info` endpoint with
  `max_input_length` and `max_total_tokens`
  ([router source](https://huggingface.co/spaces/nateraw/jupyterlab-inference-dev/blob/8e222be4de35a42260e57972be2953cc7bd3e02c/router/src/lib.rs)).
* **Similar projects drop what a server refuses.** LiteLLM raises on a
  parameter a model does not take unless `drop_params` drops it
  ([LiteLLM](https://docs.litellm.ai/docs/completion/drop_params)), from a
  table that can be wrong
  ([issue 26444](https://github.com/BerriAI/litellm/issues/26444)); the
  Vercel AI SDK's OpenAI-compatible provider makes stream usage an opt-in,
  `includeUsage`
  ([example](https://antseed.com/integrations/vercel-ai-sdk)). This module
  learns from the refusal itself (0062-MADR), which needs no table.

### E1. One listing path for `chatcompat`, inside `catalog`

* **What.** `catalog.List`, `LookupModel`, `CachedModel` and `CachedFacts`
  accept a provider id that is not built in when the options carry
  `chatcompat.WithID` of that id and a base URL. The listing is `GET
  {base}/models`, through the existing listing cache: one fetch in flight
  per key, fresh for 10 minutes, the last good one served for 24 hours after
  a transient failure, keyed as every listing is. `chatcompat.ListModels` is
  that listing's `Recommended`; each request reads `CachedFacts`, never
  fetching (0041-MADR D1), as the built-ins do. `catalog.Search` and
  `Catalog.Menu` work on it unchanged.
* **Why.** The caches, the in-flight sharing and the failure path exist and
  are tested; a second implementation in `chatcompat` would drift from
  them.
* **API.** No name added. `catalog.List`'s and `LookupModel`'s docs gain
  the case; an unknown id without `WithID` is refused as today. The
  `WithID` value is a type in a new internal package both read
  (`llmprovider/internal/compatid`), so neither imports the other.
* **Depends on** the amendment "Phase 4's options": `WithCache` and `WithID`
  shared.

### E2. Facts for any server: its dialect's details, then models.dev by canonical id

* **What.** The E1 listing scores the reply against a table of dialects,
  each a set of features of `/models` (`owned_by` values, a `meta` block,
  `max_model_len`, `context_length`, `architecture`), and, for one that
  scores above its threshold and is recorded (below), reads its one details
  endpoint once per listing: llama.cpp `/props`, LM Studio
  `/api/v0/models`, Ollama `/api/show` (the catalog's digest-cached reader),
  TGI `/info`; an OpenRouter-shaped listing needs none. The details map to
  `ModelInfo`: the context window caps the output limit, `vlm` or `vision`
  marks image input, `embeddings` a caveat that it writes no text, `tools`
  and `thinking` the toggles, `reasoning_content` the replay field.
* **Then models.dev, by canonical id,** as the lowest layer, for what the
  server does not say: a model id is canonicalised (lower case; the
  organisation prefix dropped; an Ollama tag split, `latest` dropped;
  quantisation and format tokens dropped: `q4_k_m`, `iq*`, `awq`, `gptq`,
  `fp8`, `bf16`, `int4`, `gguf`, `mlx`, `*bit`), and matched to models.dev's
  ids canonicalised the same way, across its providers. Only an exact
  canonical match counts, never a fuzzy one; where providers disagree on a
  field, the most common value wins, and on a tie the smaller limit.
  `SourceOf` names the layer "models.dev (canonical id)".
* **Order of authority,** as 0062-MADR D2's: a refusal learned, the caller's
  layers, the server's details, its listing, models.dev by canonical id.
* **Recorded first.** A dialect is enabled only with a recorded reply of
  its server in `llmprovider/catalog/testdata/compat/<dialect>/`; the local
  Ollama is recordable now, the others when the owner runs one. An
  unrecorded dialect is a plain listing.

### E3. Optional-field learning for Chat Completions servers

* **What.** `chatcompat`'s learner gains a field matcher: a 400 whose
  message names a field the request sent, from a closed list of optional
  fields (`reasoning_effort`, `stream_options`, a tool's `strict`), beside
  a recorded phrase of rejection, is that field's refusal. `reasoning_effort`
  takes 0062's last rung (none, with `WarningReasoningLearned`);
  `stream_options` and `strict` are dropped, the drop kept in the memo per
  base URL and model, and the request sent again within `MaxSends`. A
  stream without `stream_options` carries no usage; the package doc says
  so.
* **Recorded first,** as 0062-MADR D4 requires: each phrase is one a
  recorded server sent; with none recorded for a field, the field is never
  dropped.
* **API.** No name added; the memo gains a set of dropped fields.

### E4. Search and the menu for uncurated listings

* **What, in `catalog`, for every provider:**
  * **A typo tier:** when no tier above the subsequence one matches, a
    query token within an edit distance of 1 (2 for tokens of 8 runes or
    more; Damerau's transposition counted as one edit, bounded and stopping
    early) of an id token matches, scored above the subsequence tier;
  * **An index per listing:** each id's tokens, compact form and label are
    computed once per listing and cached beside it, not per query, so a
    search over a listing of a thousand ids allocates nothing per id; a
    benchmark pins it;
  * **Variants grouped:** ids with one canonical id (E2's) sit together in
    the menu, the plainest first (`granite3-dense:8b` before its
    `-instruct-q5_K_M`);
  * **A default rank for a listing with no ranking:** text models first
    (an embedding, reranker, speech or image-only model, by its facts or
    its id's tokens, gets the existing "writes no text" caveat), loaded
    models first where the server says, then the larger context window,
    then the newer `created`, the listing's order breaking ties.
* **API.** None: `Search`'s signature and its tiers' order above the
  subsequence one are unchanged; the existing Search tests pin it.

### E5. Local server discovery

* **What.** `chatcompat.Discover(ctx context.Context, opts
  ...llmprovider.Option) ([]Server, error)` asks the documented default
  ports of local servers on loopback only (`127.0.0.1`), each with one
  `GET /v1/models`, at once, each bounded by 300 ms, sending no key, and
  returns those that answer: `type Server struct { BaseURL, Label string;
  Models []string }`, the label the dialect's (E2) or the host and port. The
  ports, each confirmed in the record before it ships: LM Studio 1234,
  Ollama 11434 (measured), llama.cpp and LocalAI 8080, vLLM 8000, Jan 1337,
  KoboldCpp 5001.
* **Why.** A configure step, gobble-cli's or the wizard's, offers what is
  running without asking for a URL.
* **API.** Two names added to D10 (R55: a caller's configure step).

### Cost and order

E1 is the base of E2 and E4's menu; E3 and E5 stand alone. In the PLAN they
are phases 4a to 4e, after Phase 4 and before Phase 5, each with its
recordings, tests, plants and checks.

E4's menu half (variant grouping, the default rank) runs in Phase 4c, after
E2, whose canonical id and dialect details it reads; its search half is
Phase 4b's, as the amendment "Model search, rebuilt" replaces it (PLAN,
deviation 2, 2026-10-25).

## Amendment 2026-10-25: Model search, rebuilt (proposed)

**Status: accepted (owner, 2026-10-25),** with its tier key (S2) over
Reciprocal Rank Fusion and over BM25F alone, and `TextPrompter`'s aligned
column on every menu (S4). The owner asked, before Phase 4b runs, for fuzzy
matching, BM25 and a clearer search prompt and results. This replaces E4's search half (its typo tier and per-listing
index); E4's variant grouping and default rank stand.

### Search as it is

* `catalog.Search(provider, models, query)` scores ids and labels by tier
  (`catalog/model_matcher.go:18-31`): exact, prefix, substring, compact,
  label, token prefix, and a subsequence only when nothing else matched.
  Within a tier every match scores the same, so the listing's order
  decides; a typo finds nothing until the subsequence tier, and often
  nothing then.
* A model's facts are not searched, though `Catalog.Menu` holds them: no
  query finds "vision", "128k", "reasoning" or "free".
* The wizard (`wizard/model_select.go:40-91`) shows each match's label alone,
  at most 20, with a notice when it drops more; no id, no facts, no reason
  for the match, no way to see the rest.

### Evidence (searched 2026-10-25)

* **Fuzzy matching:** fzf scores a subsequence with a bonus at word
  boundaries and for consecutive runs, the first character's doubled, and a
  gap penalty that cancels a boundary bonus past about 8 characters
  ([fzf algo.go](https://fossies.org/linux/fzf/src/algo/algo.go)).
* **BM25F:** BM25 over weighted fields; `k1` 1.2 is the common default, a
  short field such as a title needs little or no length normalisation
  (`b` 0) and a weight of 2 to 3
  ([Robertson et al.](https://arxiv.org/pdf/1709.03260),
  [Elasticsearch](https://books.didispace.com/elasticsearch-definitive-guide-cn/170_Relevance/70_Pluggable_similarities.html)).
* **Fusion:** Reciprocal Rank Fusion, `k` 60, merges ranked lists by rank
  alone ([ParadeDB](https://www.paradedb.com/learn/search-concepts/reciprocal-rank-fusion)).
* **Model pickers** show, per row, the context window, prices and
  capabilities, and filter as one types
  ([pi-model-costs](https://pi.dev/packages/pi-model-costs),
  [modelsdev](https://docs.rs/modelsdev/0.6.0)).

### S1. One engine, for ids alone and for a menu with facts

* **`catalog.Search` keeps its signature** and searches ids and labels with
  the engine below.
* **`func (c Catalog) Search(query string) []Match` is added:** it searches
  `c.Menu`, each entry's id, label and facts. The wizard calls it (R55).
* **`Match` gains `Why string`**, a short reason ("exact id", "id prefix",
  "all words", "near sonnet", "vision"), for a picker to show. An added
  field is compatible (R48).

### S2. The ranking: tiers first, then BM25F, then the fuzzy score

Each model is a document of fields, each tokenised in lower case on `-`,
`_`, `/`, `.`, `:`, `~`, spaces and letter-digit boundaries (`gpt4o` is
`gpt`, `4`, `o`; `qwen3:8b` is `qwen`, `3`, `8b`, the size kept whole):

| Field | Weight | `b` |
| :--- | ---: | ---: |
| id | 3 | 0 |
| label | 2 | 0 |
| family | 1.5 | 0 |
| facets (S3) | 1 | 0.5 |

A query word matches a document word exactly (1), as a prefix of it (0.8,
two runes or more), or within a bounded edit distance (Damerau's
transposition one edit; 1 for words of 4 to 7 runes, 0.6; 2 from 8 runes,
0.4; none under 4). BM25F sums, per query word, its best match's weight
times the field weight, with `k1` 1.2 and the IDF over the listing.

The order is a key, compared in turn, so that a stronger kind of match is
never outranked by a weaker one with more words:

1. **The tier:** the exact id; an id prefix; every query word matched (an
   exact, prefix or fuzzy match in some field); some words matched, more
   first; a subsequence of the compact id only (fzf's scoring: boundary and
   consecutive bonuses, the first character doubled, gap penalties).
2. **BM25F's score.**
3. **The fuzzy subsequence score,** fzf's.
4. **The listing's order.**

Reciprocal Rank Fusion was weighed and not chosen: it lets a model second
in every list outrank one first in one, which a picker cannot explain; the
key keeps today's tiers (an exact id first, as the existing Search tests
pin), and adds BM25F and fuzziness inside and below them.

### S3. Facets: a model's facts as words

From each menu entry's `ModelInfo`, as the facts say and only then:
`vision` (image input), `audio`, `pdf`, `tools`, `reasoning` and `thinking`,
`open` (open weights), `free` (both prices zero), the context window as
words (`8k` to `1m`, the bucket at or below it, `128k` matching
`128k`-and-above), the release year, and the family. A query of
`claude vision 200k` finds the Claude models with image input and a window
of at least 200k, and `Why` names the facets matched.

### S4. The picker, with no change to `Prompter`

`Prompter` and `Choice` are unchanged (R48; their doc calls a change
breaking). The wizard and `TextPrompter` change:

* **The prompt:** "Search models: a name, a family, or a feature such as
  vision, tools, 128k (blank for the recommended)".
* **Each row:** `Label` the model's name or id; `Detail` its facts, in one
  order, those it has: the context window (`200k ctx`), the prices (`$3 /
  $15 per M`), the facets (`vision · tools · reasoning`), and for a search
  result `Why` after them (`· near sonnet`).
* **The title** counts the matches: `Choose a Claude model (12 matches for
  "sonet"):`.
* **Paging,** not a cut: ten rows, then "More (10 of 32 shown)" as a choice,
  which shows the next ten; "Search again" and "Other" stay last.
* **No match:** the nearest three ids by the fuzzy score are offered as "Did
  you mean", before "Use … as the model id" and "Search again".
* **An exact id** typed is the default row.
* **`TextPrompter`** aligns `Detail` in a column, the labels padded to the
  longest, at most 40 runes, on every menu it draws.

### S5. Speed

The index (each document's fields' words, the listing's vocabulary and its
document frequencies) is built once per `Catalog`, when `List` builds it, and
once per `catalog.Search` call; fuzzy matching compares a query word with the
vocabulary's words of a length within its bound, and stops a distance once
it passes the bound. A benchmark of a thousand ids pins the time and
allocations per search, recorded before and after.

### Scope

`wizard` joins this record's scope for S4: `wizard/model_select.go` and
`wizard/text_prompter.go`, and the tests that pin their output, each listed
in the PLAN before it changes. D10 gains `Catalog.Search` and `Match.Why`.

### S2 to S4, made exact before the code (2026-10-25)

Reading the tests that pin Search (`catalog/model_matcher_test.go`,
`matcher_coverage_test.go`, `kilo_catalog_test.go`) before writing the code
made three points exact, within what the owner approved:

* **S2's tiers are today's, kept whole, and three fallbacks.** Today's tiers
  keep their scores, their tokeniser and their reach: the exact id (6000),
  an id prefix (5000), an id substring (4000), the compact id (3500), the
  label's display name (3000), a token prefix (2000). Beside them: every
  query word matched exactly or as a prefix in some field, facets included
  (1800). Only when none of these matches a model are the fallbacks tried,
  in turn, each only when the one before matched nothing: every word
  matched with one at least within the edit bound (1500); some words, more
  first (1200 and the count); letters in order (1000 and today's bonus).
  Inside a score the order is BM25F's, then fzf's, then the listing's;
  `Match.Score` stays the tier's value, so a picker that reads it reads what
  it did. A typo never adds a row to a query that matched.
* **S3 has no `open` facet:** `ModelInfo` holds no open-weights fact. The
  facets are `vision`, `audio`, `video` and `pdf` (input modalities),
  `tools`, `reasoning` and `thinking`, `free`, the context buckets, the
  release year, and the family, each only where `Has` reports its field.
* **S4's facts go on the search results' rows;** the recommended menu's
  rows are as they are.

### S1 amended in execution: Usable, with Menu's facts (2026-10-25)

S1 said `Catalog.Search` searches `c.Menu`. Executing it found two facts S1
did not weigh:

* `List` cuts `Menu` to `WithLimit(n)` (`catalog/menu.go`, `cutMenu` in both
  builders) and leaves `Usable` uncut, and the wizard passes
  `catalog.WithLimit(o.ModelLimit)` (`wizard/configure.go`). Searching
  `Menu` would hide every model past the limit, where the wizard searches
  them all today.
* Equal scores keep `Usable`'s order, cheapest first for Kilo and fastest
  first for Hugging Face (0026-MADR F54); `Menu`'s order is the band, then
  the newest.

**The owner chose (2026-10-25):** `Catalog.Search` searches `Usable`'s ids,
in `Usable`'s order, each with the facts of `Menu`'s entry of the same id,
compared without case, where `Menu` holds it; when `Usable` is empty, it
searches `Menu`'s ids. A model past the limit is found by its id and label,
without facets. Searching `Menu` as written, and leaving `Menu` uncut by
`List` (a change to 0038-MADR D1 and A5), were weighed and not chosen.

### S4 amended in execution: no "Did you mean" (2026-10-25)

S4's "Did you mean", the nearest three ids within the edit bound when a
search finds nothing, is empty by construction: the engine returns every
model one of whose words is within the bound of a query word (S2's
fallbacks), so a search that finds nothing has no id within it. **The owner
chose (2026-10-25)** to drop it: the no-match menu is as it was, and the
near tiers' rows, each with its `Why`, are the corrections. A suggestion one
edit looser, as an exported `Catalog.Suggest` or inside the wizard, was
weighed and not chosen. S4's exact id as the default row needs no code: the
exact id is always the first row.

### S2's facet length, made exact in execution (2026-10-25)

S2 gives the facets `b` 0.5, so a longer facets field scores lower. The
context buckets are cumulative (S3: `128k` matches a window of 128k and
above), so a larger window adds more words, and measured on the words alone
a 2M-context model scored below an 8k one on `vision`. The facets' length
counts the buckets as one word; `TestBM25F_0061` pins it, and a plant that
counts them whole is caught. No decision changes: S2 left the length
unstated.

## Amendment 2026-10-25: E2 and E4's menu, made exact before Phase 4c

**Status: accepted (owner, 2026-10-25)** for its four decisions; the rest
makes exact what E2 and E4 left unstated, within them. Found by reading
`catalog`'s listing, its Ollama reader and its menu builders, and by probing
the local Ollama (0.33.2) and the recorded models.dev document
(`catalog/testdata/ranking-2026-10-12/metadata.json`) before any code.

### What was found

* **E4's example contradicts E2's rules.** E2 drops only quantisation,
  format and the tag `latest`, so `granite3-dense:8b` and
  `granite3-dense:8b-instruct-q5_K_M` canonicalise to `granite3-dense-8b`
  and `granite3-dense-8b-instruct`: E4's own example would not group.
* **"For every provider".** The built-in providers' menus are a ranked
  band, then newest first (0041-MADR D3, A4); grouping inside them would
  reorder ranked rows.
* **No dialect rule.** E2 names a score and a threshold, and sets neither.
  Ollama's `/v1/models` gives each entry `id`, `object`, `created` and
  `owned_by` (`library` for a pulled model), and every id carries a `:`
  tag (`llama3.2:latest`).
* **models.dev is off for a compat listing** (`metadataOff`, 0061-MADR
  E1), and the built-in Ollama never fetches it.
* **Exact matches are rare, as intended:** of the local Ollama's eight
  models, only `gpt-oss:20b` has an exact canonical match in the recorded
  document (`huggingface/openai/gpt-oss-20b`, `kilo/openai/gpt-oss-20b`).
* **`/api/show` carries the local account:** its `modelfile` names the blob
  by its absolute path under the user's home.

### The owner's decisions (2026-10-25)

1. **Variants group by a looser key:** the canonical id less the role
   tokens `instruct`, `chat`, `it` and `base`. The models.dev match keeps
   E2's canonical id, so an instruct model never takes a base model's
   facts. A role token in the canonical id itself was not chosen (it would
   merge `qwen3-235b-a22b-instruct-2507` with `qwen3-235b-a22b-2507`), nor
   was keeping E2's id and amending the example.
2. **Grouping and the default rank apply to unranked listings only:** a
   `chatcompat` listing and the built-in Ollama's menu. The ranked
   providers' menus are unchanged. For Ollama this replaces 0041-MADR A4's
   `/api/tags` order. chatcompat alone, and every provider after its band,
   were not chosen.
3. **A listing is Ollama's when every entry has a `:`-tagged id, an integer
   `created` and an `owned_by`.** The score is the share of these marks met
   across the entries; the threshold is all of them. A confirming `GET
   /api/version`, and `owned_by` `library` alone, were not chosen.
4. **A compat listing fetches models.dev as a built-in's does:** alongside
   the listing, cached and shared, off with `WithoutModelMetadata`; a
   failed fetch leaves its layer empty. Reading only a document already
   cached, and an opt-in option, were not chosen.

### Made exact within E2 and E4

* **X1. The canonical id.** Lower case; everything up to the last `/`
  dropped; an Ollama tag split at the last `:`, `latest` dropped; the rest
  split on `-`, `.`, `:` and spaces; a token that is a quantisation or a
  format dropped (`q` or `iq` and a digit, then any letters, digits and
  `_`, as `q4_k_m` or `iq3_xxs`; `awq`, `gptq`, `fp8`, `fp16`, `bf16`,
  `int4`, `int8`, `gguf`, `mlx`, and a number followed by `bit`); the
  others joined by `-`, a `_` within one becoming `-`. models.dev's ids are
  canonicalised the same way. Only equal canonical ids match.
* **X2. The variant key and its order.** The canonical id less the tokens
  `instruct`, `chat`, `it` and `base` (decision 1). A group sits at its
  best-ranked member's place, its members the plainest first: the fewest
  tokens in the lower-case id, split on `-`, `_`, `.`, `:`, `/`, then the
  listing's order.
* **X3. The default rank** (decision 2): the uncaveated before the
  caveated; then a model the dialect says is loaded (Ollama says none);
  then the larger context window, a known one first; then the newer
  `created` (a compat listing's `created`; the built-in Ollama's
  `/api/tags` `modified_at`, which Ollama's `/v1/models` reports as
  `created`); then the listing's order. Then X2 groups the variants,
  within the uncaveated and within the caveated, so that grouping never
  lifts a caveated model. On a `chatcompat` listing only, a model whose
  facts do not state its output, and whose id has the token `embed`,
  `embedding`, `embeddings`, `rerank`, `reranker`, `whisper` or `tts`, gets
  `CaveatNoTextOutput`; the built-in Ollama keeps 0042-MADR D1, which never
  reads a name (owner, 2026-10-25, PLAN deviation 8). `Recommended` is the ranked order less
  the caveated, as it is for each listing today (Ollama's keeps a model
  whose only caveat is `CaveatNoTools`, 0042-MADR D2).
* **X4. The details.** The Ollama dialect's details are `/api/tags` and
  `/api/show` at the server's root (the base URL less a trailing `/v1`),
  through the catalog's digest-cached reader, as the built-in Ollama reads
  them; they are the listing's facts, `SourceOf` `listing:<id>`.
* **X5. Disagreement.** Where several models.dev entries share a canonical
  id, each field takes the value most of them give, the providers read in
  name order; on a tie, a limit (`ContextWindow`, `MaxInputTokens`,
  `MaxOutputTokens`) takes the smaller, and any other field stays unknown.
  `SourceOf` names `models.dev (canonical id)`.
* **X6. The order of authority** within `catalog`: the caller's layers,
  the server's details, models.dev by canonical id. A learned refusal
  outranks them all in `chatcompat`'s learner, outside `catalog`, as
  `TestNew_Learn_Effort_0061` already pins.
* **X7. The fixtures** keep each `/api/show` reply's `capabilities`,
  `details`, `model_info` and `modified_at`, and drop its `modelfile`,
  `license`, `template`, `parameters` and `tensors`: the `modelfile` holds a
  path under the local account, and the rest is unread bulk.
* **X8. `catalog.WithModelLayer` is a shared option** (owner, 2026-10-25,
  PLAN deviation 9), so that X6's first layer reaches a compat listing: as
  a scoped option, `List` and `chatcompat.New` refused it for a compat id.
  The built-ins and `chatcompat` read it; a provider outside this module
  ignores it, where it refused it. This replaces, for `WithModelLayer`
  alone, the amendment "Phase 4's options"' "stay scoped to the built-ins";
  `WithLimit`, `WithProfile` and `WithWorkload` stay scoped. Keeping it
  scoped (no caller layer for a compat listing), and a
  `chatcompat.WithModelLayer`, were weighed and not chosen.
* **X9. The canonical match searches the sections the SDK keeps** (owner,
  2026-10-25, PLAN deviation 10): `decodeModelMetadata` keeps nine of
  models.dev's sections (`opencode`, `opencode-go`, `huggingface`,
  `togetherai`, `openai`, `anthropic`, `google`, `xai`, `kilo`; 0028-MADR
  D-A1), so "across its providers" is across these. They hold the gateways
  that list open models. Keeping every section, and adding named sections
  of local models, were weighed and not chosen.

## Amendment 2026-10-26: E3 waits for a recorded refusal

**Status: accepted (owner, 2026-10-26).** E3 learns an optional field's
refusal only from a phrase a recorded server sent. Phase 4d's first step
probed the local Ollama (0.33.2), `POST /v1/chat/completions` with
`qwen3:8b`, each of E3's fields with a valid value and an odd one:

| Field sent | Status |
| :--- | :--- |
| none | 200 |
| `reasoning_effort` `high`, `max` | 200 |
| `reasoning_effort` `bogus` | 400 `invalid reasoning value: "bogus" (must be "minimal", "low", "medium", "high", "xhigh", "ultra", "max", or "none")` |
| `stream_options` `{"include_usage": true}`, with `stream` and without | 200 |
| `stream_options` `{"bogus": 1}`, with `stream` | 200 |
| a tool's `strict` `true`, `"yes"` | 200 |

Its one refusal is of an effort the SDK never sends: `Ladder` sends only the
efforts it names. No phrase is recorded for any of E3's fields, so E3 as
written would send every field always: its matcher would match nothing.
**The owner chose** to record the probe and build nothing: E3 runs as
written when a server the owner runs (LM Studio, vLLM, llama.cpp) is
recorded refusing one of its fields. Building the matcher with no phrase,
and taking phrases from servers' documentation and source in place of a
recording (which E3 and 0062-MADR D4 forbid), were weighed and not chosen.

## Amendment 2026-10-26: E5's ports and a keyed server, confirmed before Phase 4e

**Status: accepted (owner, 2026-10-26).** E5 asks that each default port be
confirmed from its server's documentation before it ships. Read on
2026-10-26, official pages only:

| Server | Port | The documentation's words | Source |
| :--- | ---: | :--- | :--- |
| Ollama | 11434 | measured: the local Ollama 0.33.2 answers there | 0041-MADR A4; this record's Phase 4c |
| llama.cpp `llama-server` | 9931 | "`--port PORT` \| port to listen (default: 9931)"; "by default listens on `127.0.0.1:9931`" | <https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md> |
| LocalAI | 8080 | "By default the LocalAI WebUI should be accessible from `http://localhost:8080`." | <https://localai.io/docs/basics/try/> |
| vLLM | 8000 | "By default, it starts the server at `http://localhost:8000`." | <https://docs.vllm.ai/en/latest/getting_started/quickstart.html> |
| Jan | 1337 | "With the default settings, the API base URL is `http://127.0.0.1:1337/v1`." | <https://docs.jan.ai/desktop/api-server> |
| KoboldCpp | 5001 | "The default is localhost:5001; a custom `--port` changes it." | <https://github.com/LostRuins/koboldcpp/blob/concedo/README.md> |
| LM Studio | 1234 | no default stated: "`--port` … If not provided, uses the last used port"; "The following examples assume the server port is `1234`" | <https://lmstudio.ai/docs/cli/server-start>, <https://lmstudio.ai/docs/app/api/endpoints/openai> |

Two of E5's entries were wrong: llama.cpp's default is 9931, not 8080 (8080
appears in its Docker examples only, which pass `--port 8080`); and LM
Studio states no default. On keys: Jan, "If a key is configured, requests
must include it", answering "401 Unauthorized"; LM Studio, "By default, LM
Studio does not require authentication for API requests"
(<https://lmstudio.ai/docs/developer/core/authentication>).

**The owner chose (2026-10-26):**

1. **The ports:** 11434, 9931, 8080, 8000, 1337, 5001 and 1234, in that
   order; 1234 is recorded as the port LM Studio's documentation and
   examples use, not a stated default. Probing only stated defaults (no LM
   Studio), and E5's list with its attributions as written, were not
   chosen.
2. **A keyed server is returned, flagged:** a server that answers `GET
   /v1/models` with 401 or 403 is in `Discover`'s result with no `Models`
   and `Server.NeedsKey` true, so a configure step can ask for its key.
   `Server` gains `NeedsKey bool`; it is a struct this phase adds, so D10
   gains no name. Leaving it out, and returning it with no flag, were not
   chosen.

**And, as the code needs:** a server's label is its dialect's (E2), so the
dialect table's marks and score move from `catalog` into an internal
package, `llmprovider/internal/compatdialect`, that `catalog` and
`chatcompat` both read, as `internal/compatid` serves `WithID`; `catalog`
keeps each dialect's details reader. No exported name is added for it.
