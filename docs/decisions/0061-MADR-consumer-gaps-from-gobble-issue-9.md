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
