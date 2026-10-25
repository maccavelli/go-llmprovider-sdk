---
status: in-progress
date: 2026-10-24
associated-madr: "0061-MADR-consumer-gaps-from-gobble-issue-9.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement gobble-cli's Consumer Gaps: Empty-Answer and Offline Errors, Max Effort, a Chat Completions Constructor, and Image Input

Associated MADR: [0061-MADR-consumer-gaps-from-gobble-issue-9.md](0061-MADR-consumer-gaps-from-gobble-issue-9.md)

## Goal

The MADR's D1 to D11, under the owner's answers to Q1 to Q8 and, once given,
Q9 to Q11: the exports of D10 and no other, each wire value as D3's and D5's
tables say, every new test seen to fail first, and one release, `v1.23.0`.

## Scope

### In scope

* `llmprovider`: `api_error.go`, `constants.go`, `contract.go`,
  `capabilities.go`, `item.go`, `warning.go`, and the `Provider` interface's
  doc.
* `llmprovider/internal/transport`: a new `offline.go`.
* `llmprovider/internal/wire`: `finish.go`, `post.go`, a new `effort.go` and
  `images.go`; the wire packages `messages`, `chatcompletions`, `responses`,
  `generatecontent`.
* `llmprovider/catalog`: the send sites of the MADR's M4 in `discovery.go`,
  `raw.go`, `listing_metadata.go`, `model_metadata.go`.
* `llmprovider/auth`: `oauth_session.go` and `request_bound.go`'s send sites.
* Every provider package under `llmprovider/providers/`, and the new
  `llmprovider/providers/chatcompat`.
* `llmprovider/llmtest`: the images need.
* `llmprovider/internal/wirecase`: the scenarios `effort-max` and `image`,
  and a `chatcompat` case; their goldens.
* Docs: `CHANGELOG.md`, `README.md`'s Status, `docs/architecture.md`,
  `docs/guides/api-standards.md` (R22, R38), each changed package's doc.
* Records: this pair, and its index rows.

### Out of scope

* Any export not in the MADR's D10 (a stop condition).
* `wizard`. Headers (D6, beyond one doc sentence), the 1-hour rate (D7),
  argument deltas (D8).
* gobble-cli: its adoption is its own record.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase, once Q9 to
  Q11 are answered. A phase starts after the owner has committed the last.
* **Red first.** Every new test is seen to fail on `d9b1910`'s code, with any
  new name declared in a shim test file in the scratch clone so that it
  builds, or on a planted break by `tools/plant.py` on a scratch clone, never
  in the tree. The phase's record holds each failure line.
* **The drift check.** After a phase's code and before its new tests,
  `go test -race -count=1 -shuffle=on` on the phase's packages passes with
  every existing test unchanged, those named in the phase aside.
* **No key, token or real image in a test, a log or a record.** Keys are
  built to a shape; an image is a 1×1 PNG built in the test from bytes.
* **The checks, before staging,** each exit status captured before any
  filter: `make pre-add-check FILES="…"` on the phase's Go files;
  `python3 -B tools/gate.py`; `make api-check` (from Phase 1 on, the
  additions listed in the record); `tools/claims.py` on the MADR;
  `make records-check`; `tools/recordlint.py --changed`;
  `tools/denyscan.py --changed`; `tools/linkcheck.py`.
* **Stage, then the owner commits** with `git commit --no-edit`. A push and a
  tag happen on the owner's word, after the disclosure guard.
* **Stop conditions:**
  * a fact of the MADR no longer holds;
  * an existing test other than those a phase names would change;
  * an exported name not in D10 would be added, or one of D10 would change;
  * a doc would state a behaviour no test shows;
  * a live test contradicts a table of D3 or D5.

  On any of these, stop, record a dated deviation here (with an amendment
  in the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. The MADR records the owner's answers to Q9 to Q11, and Decision Outcome
   names them; `tools/records.py status`: the MADR `accepted`, this PLAN
   `in-progress`.
2. Stop; Phase 1 starts after the owner commits.

### Phase 1: `ErrEmptyAnswer` (D1)

1. **`llmprovider/api_error.go`:** in the sentinel `var` block after
   `ErrIncomplete`, `ErrEmptyAnswer`, as D1 writes it, with a doc comment
   saying it matches `ErrIncomplete` and `ErrInvalidRequest`, that a
   refusal and a cut answer are it too, and that `APIError.Reason` tells
   them apart.
2. **`llmprovider/internal/wire/finish.go`:** `EmptyAnswer`'s `Kind` is
   `llmprovider.ErrEmptyAnswer`; `Message` and `Reason` unchanged; its doc
   says so.
3. **`llmprovider/contract.go`:** the `Provider` interface's `Generate` doc
   gains: "An answer with no content gives an `*APIError` matching
   `ErrEmptyAnswer`."
4. **The drift check** on `./llmprovider/...`.
5. **Tests:**
   * `llmprovider/errors_0061_test.go`, `TestErrEmptyAnswer_Chain_0061`:
     `errors.Is` with `ErrEmptyAnswer`, `ErrIncomplete`, `ErrInvalidRequest`;
     its message is `"llmprovider: empty answer"`; an `*APIError` of that kind
     is not `Retryable()`.
   * `llmprovider/internal/wire/finish_0061_test.go`,
     `TestEmptyAnswer_Kind_0061`: `EmptyAnswer("x", "content_filter")`
     matches `ErrEmptyAnswer` and `ErrIncomplete`, its `Message` ends ": the
     answer has no content", its `Reason` is `content_filter`.
   * In each wire package, `TestDecode_EmptyIsErrEmptyAnswer_0061`: the
     no-content reply each of the nine call sites answers (messages 313, 334;
     chatcompletions 450, 474, 511; responses 350; generatecontent 216, 224,
     265; gemini's interactions 223) decodes to an error matching
     `ErrEmptyAnswer`.
6. **Plants:** `Kind` back to `ErrIncomplete` (the wire tests fail);
   `ErrEmptyAnswer`'s parent `ErrInvalidRequest` (the chain test fails on
   `ErrIncomplete`).
7. **Docs:** `CHANGELOG.md` `[Unreleased]`, Added.

### Phase 2: `ErrOffline` (D2)

1. **`llmprovider/api_error.go`:** `ErrOffline`, as D2 writes it, with its
   doc: what counts as offline (D2's rule), that it matches
   `ErrProviderUnavailable` and is retried.
2. **`llmprovider/internal/transport/offline.go` (new):**
   * `func Offline(err error) bool`: D2's rule exactly. The errno list is
     chosen by `runtime.GOOS`, `windows` giving the Winsock codes as
     `syscall.Errno(10061)` and the others, as gobble-cli's `unreachable` does
     (G3, G4).
   * `func WrapOffline(ctx context.Context, err, offline error) error`: `err`
     when `err` is nil, `ctx.Err()` is not nil, or `Offline(err)` is false;
     else `fmt.Errorf("%w: %w", offline, err)`.
3. **The send sites,** each `err` from the `Do` or `RoundTrip` of the MADR's
   M4 passed through `transport.WrapOffline(ctx, err, llmprovider.ErrOffline)`
   before it is returned or wrapped further:
   * `internal/wire/post.go:144`: after the `Unsendable` and `wrote` checks,
     in place of the bare `return nil, err`;
   * `catalog/discovery.go` lines 709, 810, 898, 956, 1002, 1095, 1197, 1294
     (at 898 and 956 the existing `ErrProviderUnavailable` wrap is kept around
     it);
   * `catalog/raw.go` 142, 167, 202; `catalog/listing_metadata.go:369`;
     `catalog/model_metadata.go:535`;
   * `providers/openai/chatgpt_listing.go:86`;
   * `auth/oauth_session.go:587` (the `ErrProviderUnavailable` wrap kept
     around it); `auth/request_bound.go:21`.
4. **The drift check** on `./llmprovider/...`.
5. **Tests:**
   * `llmprovider/internal/transport/offline_0061_test.go`,
     `TestOffline_Rule_0061`, a table: offline for `nettest.Refusing()`'s
     error (C15), a `*url.Error` around a `*net.DNSError`, a dial
     `*net.OpError` that timed out, and one each for `ENETDOWN`,
     `ENETUNREACH`, `EHOSTUNREACH` (Unix) or their Winsock codes (Windows);
     not offline for a plain error, a `*url.Error` around a TLS error, an
     `OpError` with `Op` `"proxyconnect"` or `"read"`, and an `OpError` dial
     without a listed cause or a timeout.
   * `TestWrapOffline_Context_0061`: a refused dial wraps under a live
     context, and not under an ended one (`errors.Is(err, ErrOffline)` false,
     the context's error kept).
   * `llmprovider/errors_0061_test.go`, `TestErrOffline_Chain_0061`: matches
     `ErrProviderUnavailable`; the package's `retryable` is true for it.
   * `llmprovider/internal/wire/post_0061_test.go`,
     `TestPost_RefusedDialIsOffline_0061`: `wire.Post` with
     `nettest.Refusing()` returns an error matching `ErrOffline` and
     `ErrProviderUnavailable`.
   * `llmprovider/catalog/offline_0061_test.go`,
     `TestList_RefusedDialIsOffline_0061`: `catalog.List` for `openai` and
     `ollama`, each with `nettest.Refusing()` and its own cache.
   * `llmprovider/auth/offline_0061_test.go`,
     `TestRefresh_RefusedDialIsOffline_0061` and
     `TestSignInRequest_RefusedDialIsOffline_0061`.
   * `llmprovider/providers/openai/offline_0061_test.go`,
     `TestChatGPTListing_RefusedDialIsOffline_0061`.
6. **Plants:** `Offline` returning false (every site test fails); the
   context check dropped (`TestWrapOffline_Context_0061` fails); one site's
   wrap removed, `post.go`'s (its test fails).
7. **Docs:** `CHANGELOG.md` `[Unreleased]`, Added (`ErrOffline`) and Changed
   (D2's behaviour change); `docs/architecture.md`'s error section names
   `ErrOffline` and `ErrEmptyAnswer`.

### Phase 3: `EffortMax` (D3)

1. **`llmprovider`:** `constants.go` gains `effortMax = "max"`;
   `contract.go`'s `EffortMax Effort = effortMax` after `EffortXHigh`, and
   `valid()` accepts it; `warning.go`'s `WarningEffortDegraded WarningKind =
   "effort_degraded"`, documented as D3 says.
2. **`llmprovider/internal/wire/effort.go` (new):**
   * `func ResolveMax(efforts []llmprovider.Effort, known bool) (llmprovider.Effort, bool)`:
     D3's rules 1 to 3; it returns the effort to send and whether it was
     degraded. Rule 2's target follows the owner's Q9 answer.
   * `func EffortWarning(model string, sent llmprovider.Effort) llmprovider.Warning`.
3. **Each provider,** where its effort is chosen, when the request's effort
   is `EffortMax`: `ResolveMax` with its facts, then its own wire mapping,
   appending `EffortWarning` to the `Response`'s `Warnings` when degraded or
   when its table row says "warned":
   * `claude`: the effort passed to `messages.AddThinkingFor`;
   * `openai`: `effort()`, for the Platform, ChatGPT and plan paths;
   * `grok`: `effortOrder` gains `effortMax` after `effortXHigh`, so
     `clampReasoningEffortFor` sends `xhigh` for a model whose menu stops
     there, warned;
   * `gemini`: `thinkingLevelFor` sends `high` for `max`, warned;
   * `ollama`: `max` sent as `"max"`;
   * `huggingface`, `together`, `kilo` (both fields of `reasoningFields`),
     `opencode` (each route by its family's rule): `"max"` after
     `ResolveMax`.
4. **The drift check** on `./llmprovider/...`.
5. **Tests:**
   * `llmprovider/contract_0061_test.go`, `TestEffortMax_Valid_0061`.
   * `llmprovider/internal/wire/effort_0061_test.go`,
     `TestResolveMax_Rules_0061`: facts with `max`; facts
     `['low','medium','high','xhigh']`; `['low','medium','high']`;
     `['low','high','max']`; no facts.
   * In each provider package, `TestEffortMax_Mapping_0061`: with facts
     listing `max`, without, and with none, the request body's effort field
     and the `Response.Warnings`, as D3's table says.
   * `llmprovider/internal/wirecase`: the scenario `effort-max`
     (`Reasoning{Effort: EffortMax}`); its 20 goldens recorded with `-update`,
     each diffed against D3's table before it is committed.
6. **Plants:** `valid()` without `EffortMax` (its test fails); `ResolveMax`
   always returning `max` (the rules test and Grok's and Gemini's mappings
   fail); Grok's order without `effortMax` (its mapping fails).
7. **Docs:** each provider's package doc, under Degradations; `CHANGELOG.md`
   Added.

### Phase 4: `providers/chatcompat` (D4)

1. **`llmprovider/providers/chatcompat/` (new):**
   * `chatcompat.go`: the package doc (what it is, the optional key, no
     facts, Capabilities, Degradations); `New`, `WithID`; the provider:
     `ID`, `Capabilities`, `Generate`, `Stream`, `ListModels`, through
     `internal/wire/chatcompletions` and `wire.Post`, as `huggingface` does;
   * `descriptor.go`: `Descriptor(id, label)`;
   * `New`'s refusals as D4 lists, its doc naming each kind (R52).
2. **`docs/guides/api-standards.md`:** R22 and R38 as D4 amends them, citing
   this record.
3. **`docs/architecture.md`:** the package table's row (its "Depends on" as
   `go list` gives it, for `tables-check`), and the providers' list.
4. **Tests:**
   * `chatcompat_test.go`: `TestNew_Refusals_0061` (no id, empty id, each
     built-in id in another case, no base URL); `TestNoKey_NoAuthorization_0061`;
     `TestKey_Bearer_0061`; `TestListModels_0061` (`GET {base}/models`);
   * `llmtest_test.go`: `llmtest.Run` (R44);
   * `example_test.go`: an `Example` over `httptest` with `// Output:` (R50);
   * `helpers_test.go`: `testCache(t)` in its `build` (0058-MADR D1);
   * `wirecase`: a `chatcompat` case; its goldens for every scenario.
5. **Plants:** the built-in check removed (`TestNew_Refusals_0061` fails);
   the bearer header always set (`TestNoKey_NoAuthorization_0061` fails).
6. **Docs:** `CHANGELOG.md` Added.

### Phase 5: `ImageItem` and `Capabilities.Images` in the contract (D5, first half)

1. **`llmprovider`:** `constants.go`'s `itemTypeImage = "image"`;
   `item.go`'s `ImageItem`, as D5 writes it, and the sealed doc's list;
   `capabilities.go`'s `Images Support`, after `NativeStreaming`;
   `contract.go`: `validate` accepts `ImageItem` with D5's checks, and its
   error text's list names it; `Check`'s needs gain
   `{hasImage(req.Input), c.Images, "image input"}`; `warning.go`'s
   `WarningImageDropped`.
2. **Every provider** still declares no `Images`, so `Unsupported`: a request
   with an image is refused before the network until Phase 6.
3. **`llmprovider/llmtest`:** the needs table gains
   `{"images", caps.Images, imageRequest(), false}` (C39).
4. **The drift check** on `./...`.
5. **Tests:** `llmprovider/item_0061_test.go`, `TestImageItem_Validate_0061`
   (each of D5's refusals, and an accepted `Data` and `URL` image);
   `TestCheck_ImagesUnsupported_0061`; `llmtest`'s own test of the new need.
6. **Plants:** `Check` without the images need (its test and `llmtest`'s
   fail); the https check dropped (the validate test fails).

### Phase 6: the five encoders and each provider's images (D5, second half)

1. **`llmprovider/internal/wire/images.go` (new),** on the imports R2
   allows `internal/wire` (no `catalog`):
   * `func DataURL(img llmprovider.ImageItem) string`:
     `data:<MediaType>;base64,<Data>`;
   * `func RefuseImages(model string, input []llmprovider.Item, known, takesImages bool) error`:
     nil when `input` holds no `ImageItem` or `known` is false; else, when
     `takesImages` is false, an error matching `ErrUnsupported` naming the
     model. The provider passes `known` as
     `facts.Has(catalog.FieldInput)` and `takesImages` as
     `slices.Contains(facts.Input, catalog.ModalityImage)`.
2. **The encoders,** each joining as D5's table says:
   * `messages`: `Block` gains `Source *ImageSource`
     (`type`, `media_type`, `data`, `url`); `appendBlock` turns a user text
     message into blocks when an image joins it, as it does for the
     assistant today (C26, C27);
   * `chatcompletions`: a user message's `Content` becomes parts when an image
     joins; `Opts` gains `ImageURLString bool`, which writes `image_url` as a
     string (Ollama, E9);
   * `responses`: an `InputItem`'s content becomes parts (`input_text`,
     `input_image`);
   * `generatecontent`: `Part` gains `InlineData`, appended with merge;
   * `gemini`'s `interactionsInput`: the image block joins the `user_input`
     step's content.
3. **Each provider's `Capabilities.Images`** as D5's table, with Q10 and
   Q11 as answered; each calls `RefuseImages` with its facts before its
   request; Ollama and OpenCode drop as the table says, with
   `WarningImageDropped`; Grok refuses GIF and WebP per Q10.
4. **The drift check** on `./...`: the existing goldens unchanged.
5. **Tests:**
   * each wire package, `TestImage_Encode_0061`: a user text then a `Data`
     image then a `URL` image, the JSON as D5's table;
   * each provider, `TestImages_0061`: its capability, its refusal of a
     model whose facts lack image input, and its drops or refusals;
   * `wirecase`: the scenario `image` (a user text, a 1×1 PNG by `Data`, an
     https `URL`); its goldens recorded with `-update`, each diffed against
     D5's table before it is committed.
6. **Plants:** the Messages user-text conversion removed (its encode test
   fails); Ollama's string form dropped (its test fails); `RefuseImages`
   always nil (the providers' refusal tests fail).
7. **Docs:** each provider's package doc (images, drops, refusals);
   `docs/architecture.md`'s G-wire bullet with the counts `go test` then
   reports (F16); `CHANGELOG.md` Added.

### Phase 7: live tests, run by the owner

1. `llmprovider/providers/<id>/live_0061_test.go` behind `live_gateways`,
   switched by `LLMPROVIDER_LIVE_0061` and each provider's key variable:
   an image by `Data` and one by `URL`, and an `EffortMax` request, to
   Anthropic, OpenAI, xAI, Gemini, Hugging Face, Together, Kilo and
   OpenCode; Ollama with `LLMPROVIDER_LIVE_0061_OLLAMA` (a local server);
   the ChatGPT session under Q11 (a), with `LLMPROVIDER_LIVE_CHATGPT`.
2. Each checks a reply with no error, and records the status and the
   `Warnings`, never the reply's text.
3. The owner runs them; each result goes in this record. A result that
   contradicts D3's or D5's table is a stop condition; Q11 (a)'s result
   decides its amendment.

### Phase 8: release `v1.23.0`

1. `CHANGELOG.md`'s `[v1.23.0]`, dated by the commit the tag names; the
   README's Status; the comparison links.
2. Before the push: the gate; `tools/apidiff.py v1.22.2`, listing exactly
   D10's additions and no incompatible change; the notes from
   `scripts/check_changelog.py --section v1.23.0` on a scratch clone with a
   local tag, scanned; the disclosure guard.
3. The tag on the owner's word; its CI; the Release's body equal to the
   notes.
4. After the tag, on the owner's word, `gh release delete v1.20.1 --yes`, the
   Release that falls to sixth (R53); `tools/consumer.py v1.23.0` and
   `tools/offline.py v1.23.0`.
5. A comment on issue #9, on the owner's word, naming `v1.23.0` and each
   work item's outcome; the issue closed on the owner's word.
6. The MADR's Confirmation, item by item; this PLAN `complete`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | The empty answer matches `ErrEmptyAnswer` on every wire, and still `ErrIncomplete` | 1 | Phase 1's tests |
| V2 | A dial with no network matches `ErrOffline` on every send site, and a 5xx, a TLS failure or an ended context does not | 2 | Phase 2's tests |
| V3 | `EffortMax` is sent per D3's table, warned where degraded | 3 | the mapping tests; the `effort-max` goldens |
| V4 | `chatcompat` passes `llmtest`, refuses as D4 says, and sends no key when given none | 4 | Phase 4's tests |
| V5 | An image is refused, sent or dropped per D5's tables | 5, 6 | the validate, encode and provider tests; the `image` goldens |
| V6 | The exports are exactly D10's | 1–6, 8 | `make api-check`; `tools/apidiff.py v1.22.2` |
| V7 | Every UNCONFIRMED point has a live result | 7 | the record |
| V8 | Every new test failed first | 1–6 | the records |

## Rollout and Rollback

* **Rollout.** Commits on `main`, one per phase, staged by the agent and
  committed by the owner; one tag, `v1.23.0`, after Phase 7.
* **Rollback.** Before the tag, a phase reverts on its own, newest first.
  After it, a fix is a patch release; an export is never removed in `v1`
  (R48).

## Execution record

### Phase 0: records (2026-10-24)

* **Approval.** The owner committed the pair as `f86fc3f`, answered Q9 (a),
  Q10 (a) and Q11 (a), and approved the PLAN ("q9: a, q10: a, q11: a.
  committed, proceed").
* **Step 1.** The MADR's Decision Outcome, D3's rule 2, D5's provider table
  and its Owner questions record the answers; `tools/records.py status`: the
  MADR `accepted`, this PLAN `in-progress`, with their index rows.
* **Not done here:** Phase 1 starts after the owner has committed this
  phase.
* **Committed** by the owner as `a0ca500` ("committed, proceed").

### Phase 1: `ErrEmptyAnswer` (2026-10-24)

* **Step 1.** `llmprovider/api_error.go`: `ErrEmptyAnswer`, a
  `sentinelError` with parent `ErrIncomplete`, after it in the block, its
  doc as the step says.
* **Step 2.** `wire.EmptyAnswer`'s `Kind` is `llmprovider.ErrEmptyAnswer`;
  `Message` and `Reason` unchanged; its doc says both.
* **Step 3.** The `Provider` interface's `Generate` doc names
  `ErrEmptyAnswer`.
* **Step 4, the drift check:** `go test -race -count=1 -shuffle=on
  ./llmprovider/...`: ok, every existing test unchanged.
* **Step 5, the tests:** `TestErrEmptyAnswer_Chain_0061`,
  `TestEmptyAnswer_Kind_0061`, and `TestDecode_EmptyIsErrEmptyAnswer_0061`
  in `messages`, `chatcompletions`, `responses`, `generatecontent` and
  `gemini`. Run alone with `-coverprofile`, the last executes each call
  site's line: "executed: 10 of 10".
* **Correction (2026-10-24): ten call sites, not nine.** Step 5 says "the
  nine call sites" and lists ten lines (messages 2, chatcompletions 3,
  responses 1, generatecontent 3, gemini 1); the count was wrong, the list
  right. No decision changes.
* **Red first,** on `d9b1910`'s code in a scratch clone, `ErrEmptyAnswer`
  declared in a shim so that the tests build: every wire case failed, ten
  of ten, for example

  ```text
  --- FAIL: TestDecode_EmptyIsErrEmptyAnswer_0061
      empty_0061_test.go:20: no content: llmprovider: incomplete response: stop: messages: the answer has no content; want an error matching ErrEmptyAnswer and ErrIncomplete
  --- FAIL: TestEmptyAnswer_Kind_0061
      finish_0061_test.go:18: EmptyAnswer = llmprovider: incomplete response: content_filter: x: the answer has no content; want an *APIError matching ErrEmptyAnswer and ErrIncomplete
  ```

  `TestErrEmptyAnswer_Chain_0061` passed there, the shim being the sentinel
  itself; its red is the second plant.
* **Step 6, the plants** (`tools/plant.py`), both caught:
  * `Kind` back to `ErrIncomplete`: "finish_0061_test.go:18: EmptyAnswer =
    llmprovider: incomplete response: …; want an *APIError matching
    ErrEmptyAnswer and ErrIncomplete", and every wire case;
  * the parent `ErrInvalidRequest`: "errors_0061_test.go:13: ErrEmptyAnswer
    matches ErrIncomplete false, ErrInvalidRequest true; want both".
* **Step 7.** `CHANGELOG.md` `[Unreleased]`, Added.
* **The exports.** `make api-check` reports incompatible changes only:
  "against v1.22.2, 0 incompatible change(s)". The additions are
  `tools/apidiff.py v1.22.2`'s: "Compatible changes: -
  ./llmprovider.ErrEmptyAnswer: added", D10's first row and nothing else.
  The rule's "`make api-check` (… the additions listed in the record)"
  takes them from `tools/apidiff.py` from here on.
* **The checks:** `make pre-add-check` on the 10 Go files, "10 file(s)
  clean"; `scripts/check_changelog.py`, 0 problems; `python3 -B
  tools/gate.py`: "gate: 28 of 28 passed".
* **Not done here:** Phases 2 to 8.
