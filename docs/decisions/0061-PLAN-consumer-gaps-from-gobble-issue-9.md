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
   * *(As built, a third: `MaxEffort`; see Phase 3's record.)*
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
     and the `Response.Warnings`, as D3's table says. *(As built: one table
     across the providers, and OpenCode's chat route in its own package; see
     Phase 3's record.)*
   * `llmprovider/internal/wirecase`: the scenario `effort-max`
     (`Reasoning{Effort: EffortMax}`); its 20 goldens recorded with `-update`,
     each diffed against D3's table before it is committed.
6. **Plants:** `valid()` without `EffortMax` (its test fails); `ResolveMax`
   always returning `max` (the rules test and Grok's and Gemini's mappings
   fail); Grok's order without `effortMax` (its mapping fails). *(As
   built: five; see Phase 3's record.)*
7. **Docs:** each provider's package doc, under Degradations; `CHANGELOG.md`
   Added.

### Phase 4: `providers/chatcompat` (D4)

1. **`llmprovider/providers/chatcompat/` (new):**
   * `chatcompat.go`: the package doc (what it is, the optional key, no
     facts, Capabilities, Degradations); `New`, `WithID`; the provider:
     `ID`, `Capabilities`, `Generate`, `Stream`, `ListModels`, through
     `internal/wire/chatcompletions` and ~~`wire.Post`~~ `wire.Send` and
     `wire.StreamEventsRetry` with a per-request `wire.Learner` and the Chat
     Completions matcher, as `huggingface` does (amended 2026-10-25, the
     deviation in this phase's record);
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

### Phases 4a to 4e: `chatcompat`, provider-agnostic (approved 2026-10-25)

The MADR's amendment "`chatcompat`, provider-agnostic" decides E1 to E5; the
owner chose all five. Each phase starts after the owner has committed the
last; each records before it relies, and each new test is seen to fail
first.

* **Phase 4a, E1, the catalog path.**
  1. `llmprovider/internal/compatid`: the `WithID` value's type.
  2. `catalog`: `lookupProvider` and `configFor` admit an id carried by a
     `compatid` value with a base URL; `fetchListing`'s case for it, `GET
     {base}/models` by `fetchDataIDs`; the listing cache, flight and last
     good listing unchanged.
  3. `chatcompat.ListModels` is `catalog.List(...).Recommended`, probed
     when asked; each request reads `CachedFacts`.
  4. Tests: `TestCompatListing_Cached_0061` (two `ListModels` within 10
     minutes make one `GET`; concurrent ones share it);
     `TestCompatListing_LastGood_0061` (a 503 after a good listing serves it);
     `TestCompatListing_UnknownRefused_0061` (an unknown id without `WithID`
     is refused as today). Plants: the cache bypassed; the flight bypassed.
* **Phase 4b, E4, search and the menu** ~~(the typo tier and the
  per-listing index)~~ as the MADR's amendment "Model search, rebuilt"
  (proposed 2026-10-25) replaces E4's search half:
  1. **Before any change,** the record lists every existing test that pins
     Search's results or the wizard's and `TextPrompter`'s output (a search
     of every `_test.go` for `Search(`, `searchModelsPrompt`,
     `maxSearchResults`, `showing %d of %d`, `matchChoices` and
     `renderChoices`'s lines), and `BenchmarkSearch_1000_0061`'s figures on
     the code as it stands.
  2. `catalog`: `search.go`, the tokeniser, the index, BM25F, the bounded
     Damerau distance, fzf's subsequence score, the key (S2); facets (S3);
     `Search` on the engine; `Catalog.Search`; `Match.Why`; the index built
     with each `Catalog` (S5). ~~`menu.go`: variant grouping; the default
     rank.~~ Moved to Phase 4c (deviation 2, 2026-10-25).
  3. `wizard`: the prompt, the rows' `Detail`, the counted title, paging,
     ~~"Did you mean"~~ (dropped, deviation 3), the exact id's default (S4);
     `TextPrompter`'s aligned column.
  4. Tests: `TestTokenise_0061`; `TestDistance_0061` (the bound, the
     transposition, the early stop); `TestBM25F_0061` (a field's weight, the
     IDF); `TestSearch_Key_0061` (each tier outranks the next, BM25F inside
     a tier); `TestSearch_Typo_0061` (`sonet`, `gemnii`, `qwen3:8`);
     `TestSearch_Facets_0061` (`claude vision 200k`, `free`, `reasoning`);
     `TestSearch_ExistingTiers_0061` (every existing Search test's result);
     `BenchmarkSearch_1000_0061`; ~~`TestMenu_Variants_0061`;
     `TestMenu_DefaultRank_0061`~~ (to Phase 4c); the wizard's
     `TestSearch_Paging_0061`,
     ~~`TestSearch_DidYouMean_0061`~~ (deviation 3),
     `TestSearch_Detail_0061`; and
     `TextPrompter`'s `TestRenderChoices_Aligned_0061`.
  5. Plants: the tier key dropped (BM25F alone orders); a fuzzy bound of 3;
     a facet read where the facts are silent; paging lost; the index built
     per query (the benchmark's allocations rise).
  6. Docs: `catalog`'s and `wizard`'s package docs, the CHANGELOG, the
     architecture rows; D10's names (`Catalog.Search`, `Match.Why`).
* **Phase 4c, E2, facts.**
  1. Record: the local Ollama's `/v1/models` and `/api/show` for two models,
     now; each other dialect when the owner runs its server, by
     `LLMPROVIDER_LIVE_0061_COMPAT` naming its base URL. Fixtures in
     `llmprovider/catalog/testdata/compat/<dialect>/`, `make fixtures-check`.
  2. `catalog`: the dialect table and score; each recorded dialect's
     details reader; the canonical id; the models.dev layer by canonical id.
     Then E4's menu half, moved here from Phase 4b (deviation 2,
     2026-10-25): `menu.go`'s variant grouping, by the canonical id, the
     plainest first; and the default rank for a listing with no ranking,
     text models first, loaded models first where the dialect says, then
     the larger context window, then the newer `created`, the listing's
     order breaking ties.
  3. Tests: `TestDialect_Score_0061` over every fixture (each recorded reply
     scores its own dialect, and no other above threshold);
     `TestCanonicalID_0061` (a table, each quantisation and tag form);
     `TestFacts_Order_0061` (a refusal, the caller's layer, the server's
     details, models.dev by canonical id, in that order);
     `TestCanonicalMatch_ExactOnly_0061` (a near id gets nothing);
     `TestMenu_Variants_0061`; `TestMenu_DefaultRank_0061`. Plants: the
     threshold removed; a fuzzy canonical match; variants not grouped.
* **Phase 4d, E3, optional fields.**
  1. Record: each server reachable (the local Ollama now) sent each optional
     field it may refuse; only a recorded refusal's phrase enters the
     matcher.
  2. `chatcompat`: the field matcher; the memo's dropped fields; the
     package doc.
  3. Tests: `TestFieldRefusal_Fixtures_0061` (each recorded refusal read, no
     other 400); `TestLearn_FieldDrop_0061` (dropped, resent, the next
     request right the first time). Plants: the matcher reads an unsent
     field; the drop not remembered.
* **Phase 4e, E5, discovery.**
  1. `chatcompat`: `Discover` and `Server` (D10's two names, R55).
  2. Tests: `TestDiscover_Loopback_0061` (only `127.0.0.1`; local
     `httptest` servers on the ports' stand-ins, each answering or not;
     the bound kept); `TestDiscover_NoKey_0061`. Plants: a host other than
     loopback; the timeout removed.
  3. Each default port confirmed from its server's documentation, quoted
     with its URL in the record, before it ships.

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
* **Committed** by the owner as `c59cd7a` ("committed, proceed").

### Phase 2: `ErrOffline` (2026-10-24)

* **Step 1.** `ErrOffline`, a `sentinelError` with parent
  `ErrProviderUnavailable`, after `ErrEmptyAnswer`, its doc as D2.
* **Step 2.** `internal/transport/offline.go`: `Offline` and `WrapOffline`,
  D2's rule.
* **Step 3, the sites.** Sixteen wrapped: `wire/post.go` 1,
  `catalog/discovery.go` 8 (the two Ollama sites inside their
  `ErrProviderUnavailable` wrap), `catalog/raw.go` 2 (142, 167),
  `listing_metadata.go` 1 (inside its wrap), `model_metadata.go` 1,
  `chatgpt_listing.go` 1 (with `req.Context()`), `oauth_session.go` 1
  (inside its wrap), `request_bound.go` 1. `grep -c transport.WrapOffline`
  per file: 1, 8, 2, 1, 1, 1, 1, 1.
* **Deviation (2026-10-24): four corrections,** recorded in the MADR's
  amendment "Phase 2's corrections"; no decision changes:
  * seventeen send calls, not fifteen (the MADR's F12 and D10 said 15);
  * `raw.go:202`, `rawRecorder.RoundTrip`, is not wrapped: a `RoundTrip`
    error has no `*url.Error` until `Client.Do` adds it, and every call
    through that recorder is a wrapped `Do` site; `ListRaw`'s test shows
    the result;
  * `wire.Post` and `doBounded` classify before cancelling their derived
    context, which would otherwise always read as ended; step 3's "after
    the `Unsendable` and `wrote` checks" holds for precedence;
  * on Windows the rule counts the syscall names and the Winsock codes;
    `GOOS=windows go build ./...` and `GOOS=windows go vet` pass.
* **An addition:** `providers/openai` now imports `internal/transport`, so
  `docs/architecture.md`'s row for it names it; `make tables-check` failed
  ("llmprovider/providers/openai: imports llmprovider/internal/transport,
  which its row does not name") until it did, then "38 packages, 0
  problem(s)". R2's provider row already allows internal packages.
* **Step 4, the drift check:** `go test -race -count=1 -shuffle=on ./...`:
  ok, every existing test unchanged.
* **Step 5, the tests:** `TestOffline_Rule_0061` (a refused dial from
  `nettest.Refusing()`, a DNS failure, a dial timeout, network down,
  network unreachable, host unreachable, and on Windows Winsock's refused;
  not a plain error, a TLS failure, a proxy's failure, a reset read, a dial
  with another cause, or a refused dial outside a `*url.Error`);
  `TestWrapOffline_Context_0061`; `TestErrOffline_Chain_0061`;
  `TestPost_RefusedDialIsOffline_0061` (and a 503, not offline);
  `TestList_RefusedDialIsOffline_0061` (OpenAI, Ollama, `ListRaw`);
  `TestRefresh_RefusedDialIsOffline_0061`;
  `TestSignInRequest_RefusedDialIsOffline_0061`;
  `TestChatGPTListing_RefusedDialIsOffline_0061`. Against `d9b1910`'s code,
  in a scratch clone, with the names declared in shims with no behaviour,
  every case but the chain failed, for example:

  ```text
  offline_0061_test.go:65: a refused dial, as the OS gives it: Offline = false; want true (Get "http://127.0.0.1:1": dial tcp: connection refused)
  post_0061_test.go:24: refused dial: Post "http://127.0.0.1:1": dial tcp: connection refused; want ErrOffline and ErrProviderUnavailable
  offline_0061_test.go:23: List: err <nil>, Catalog.Err Get "http://127.0.0.1:1/models": dial tcp: connection refused; want ErrOffline in one
  offline_0061_test.go:31: ListRaw: model listing: openai page 1: Get "http://127.0.0.1:1/models": dial tcp: connection refused; want ErrOffline
  offline_0061_test.go:21: Token = llmprovider: provider unavailable: oauth: refresh request: Post "http://127.0.0.1:1": …; want ErrOffline and ErrProviderUnavailable
  offline_0061_test.go:37: doBounded = Post "http://127.0.0.1:1": dial tcp: connection refused; want ErrOffline
  offline_0061_test.go:23: fetchCatalog = model listing: chatgpt models: Get "http://127.0.0.1:1": dial tcp: connection refused; want ErrOffline
  ```

* **Step 6, the plants,** all four caught: `Offline` never true; the
  context check dropped ("ended context: offline: Get …; want the error as
  it was"); `wire.Post`'s wrap removed ("refused dial: Post …; want
  ErrOffline and ErrProviderUnavailable"); the parent `ErrInvalidRequest`
  ("ErrOffline does not match ErrProviderUnavailable", "an offline failure
  is not retried").
* **Step 7, docs:** `CHANGELOG.md` Added (`ErrOffline`) and Changed (D2's
  behaviour change); `docs/architecture.md`: "Answers" names
  `ErrEmptyAnswer`, "Errors" names both sentinels; OpenAI's package row.
* **The exports,** `tools/apidiff.py v1.22.2`: "./llmprovider.ErrEmptyAnswer:
  added", "./llmprovider.ErrOffline: added"; nothing else.
* **The checks:** `make pre-add-check` on the 16 Go files, "16 file(s)
  clean"; `scripts/check_changelog.py`, 0 problems; `make tables-check`, 0
  problems; `python3 -B tools/gate.py`, "gate: 28 of 28 passed";
  `tools/claims.py` on the MADR, 58 of 58.
* **Committed** by the owner as `ea33b32` and `1b46940`. `ea33b32` holds
  part of the phase and does not build: `catalog/discovery.go` calls
  `transport.WrapOffline` eight times and `listing_metadata.go` once, and
  neither imports `internal/transport`; `1b46940` adds the imports, the
  remaining sites, the tests and the docs. The owner chose to keep both
  ("option 1").
* **Not done here:** Phases 3 to 8.

### Phase 3: `EffortMax` (2026-10-24)

* **Step 1.** `effortMax = "max"` in `constants.go`; `EffortMax` after
  `EffortXHigh` in `contract.go`, which `valid()` accepts; and
  `WarningEffortDegraded` in `warning.go`, each documented as D3 says.
  `catalog.ModelReasoning.Efforts`' doc, which called `max` a name without a
  constant, names `EffortMax`.
* **Step 2.** `llmprovider/internal/wire/effort.go`: `ResolveMax` and
  `EffortWarning`, as planned, and a third, `MaxEffort`, which returns an
  effort unchanged unless it is `EffortMax`, and otherwise `ResolveMax`'s,
  with the warning when degraded; seven providers call it, so the rule is
  written once. The warning's `Detail` is "reasoning effort max is not
  offered for MODEL; sent EFFORT".
* **Step 3, the providers.** Each provider's `body` returns its warnings
  beside the body, and its `call` appends them to `deferral.Warnings`'s in
  `wire.Call.Warnings`; no test calls `body`. `claude`'s `addReasoning`
  takes the model's facts and returns the warnings; `openai`, `huggingface`
  and `together` call `MaxEffort` in `body`; `ollama`'s `effort` and
  `kilo`'s `reasoningFields` take the model and its facts and call it,
  `ollama`'s after its `xhigh` to `"max"`; `grok`'s `effort` returns the
  warning when the clamp sends below `max`; `gemini`'s `body` resolves `max`
  before `thinkingLevelFor`, whose signature its tests use, and warns with
  the level sent; `opencode`'s `resolve` sets the call's effort and
  warnings by `maxOnRoute`, and `wireCall` appends them.
* **Deviation (2026-10-24): four corrections and one clarification,**
  recorded in the MADR's amendment "Phase 3's corrections"; no decision
  changes:
  * Kilo has no messages route; D3's first row named one;
  * on Grok, facts that list `max` send `max` (rule 1), because the clamp's
    menu is the facts' efforts in `effortOrder`, which now holds `max`; and
    Grok's "`xhigh`" is the clamp's highest, `high` for `grok-4.5`;
  * Gemini and OpenCode's google route send `high` where `ResolveMax`
    returns `max`; on the google route with no facts that is the dynamic
    thinking budget, as for `xhigh`;
  * a warning names an effort sent: an omitted effort carries none;
  * rule 3 covers facts that list none of `low`, `medium`, `high` and
    `xhigh`.
* **Step 4, the drift check:** `go test -race -count=1 -shuffle=on ./...`:
  ok, every existing test unchanged; `make pre-add-check` runs the
  packages' tests again.
* **Step 5, the tests.** `TestEffortMax_Valid_0061`;
  `TestResolveMax_Rules_0061` (the five planned cases and four more: `low`
  only, another order, facts known and empty, facts of `none` and
  `minimal`); `TestMaxEffort_Warning_0061`. **A deviation in layout:** in
  place of a test in each provider package, `TestEffortMax_Mapping_0061` in
  `llmprovider/providers/effortmax_0061_test.go` is one table of eleven
  provider and route cases, each at rule 1, rule 2 and no facts, through
  `Generate`, with the facts from a caller's layer
  (`catalog.WithModelLayer`), as `facts_0041_test.go` does; it checks the
  body's effort and `Response.Warnings`. OpenCode's chat route reads its
  efforts from the models.dev document, not the facts' layers, so it has
  its own, `TestEffortMax_ChatRoute_0061`, in its package. The G-wire
  scenario `effort-max` records the response and its warnings; its 20
  goldens, with no facts, were checked against D3's table by a script that
  prints each body's effort and warnings: `max` for Claude, OpenAI's four,
  Hugging Face, Kilo, Ollama, Together, OpenCode's responses routes and
  Zen's messages route; Gemini `high`, warned; Grok's two `high` (its case
  model is `grok-4.5`), warned; OpenCode's google route a dynamic budget,
  warned `high`; OpenCode Go's messages route none (`qwen3.8-flash` takes a
  budget, as its `thinking` golden shows); the chat routes none, with no
  document. G-wire now has 13 scenarios and 247 goldens (Phase 6 corrects
  `docs/architecture.md`'s count, F16). Red first, on a scratch clone of
  `1b46940` with the tests copied in: with the new names declared in shims
  of no behaviour, every test failed, for example:

  ```text
  contract_0061_test.go:17: a request at EffortMax: llmprovider: invalid request: unknown reasoning effort "max"; want valid
  effort_0061_test.go:39: facts to high (rule 2, Q9): ResolveMax([low medium high], true) = "max", false; want "high", true
  effortmax_0061_test.go:105: claude/claude-sonnet-5, rule 1: llmprovider: invalid request: unknown reasoning effort "max"
  effortmax_0061_test.go:41: Generate: llmprovider: invalid request: unknown reasoning effort "max"
  ```

  With steps 1 and 2 in place and no provider changed, the two `llmprovider`
  tests passed and every provider's mapping failed, for example:

  ```text
  effortmax_0061_test.go:108: claude/claude-sonnet-5, rule 2: sent "max"; want "high"
  effortmax_0061_test.go:108: grok/grok-4.6, rule 1: sent "xhigh"; want "max"
  effortmax_0061_test.go:121: grok/grok-4.6, no facts: warnings []; want one naming xhigh
  effortmax_0061_test.go:108: gemini/gemini-3.8-flash, rule 1: sent "max"; want "high"
  effortmax_0061_test.go:121: opencode-zen/gemini-3.8-flash, no facts: warnings []; want one naming high
  effortmax_0061_test.go:44: reasoning_effort = ""; want "high"
  ```

* **Step 6, the plants,** all five caught (`tools/plant.py`), the planned
  three and two more: `valid()` without `EffortMax`
  (`unknown reasoning effort "max"; want valid`); `ResolveMax` always `max`
  (`claude/claude-sonnet-5, rule 2: sent "max"; want "high"`); Grok's order
  without `effortMax` (`grok/grok-4.6, rule 1: sent "xhigh"; want "max"`);
  OpenAI's `call` dropping the effort warning
  (`openai/gpt-6-luna, rule 2: warnings []; want one naming high`);
  OpenCode's google route sending `max`
  (`opencode-zen/gemini-3.8-flash, rule 1: sent "max"; want "high"`).
* **Step 7, docs:** each of the nine provider packages' doc, under
  Degradations; `CHANGELOG.md` Added; `docs/architecture.md`: the
  `internal/wire` row names the three functions, and "Response.Warnings"
  names `WarningEffortDegraded`.
* **The exports,** `tools/apidiff.py v1.22.2`: "./llmprovider.EffortMax:
  added", "./llmprovider.WarningEffortDegraded: added", beside Phases 1 and
  2's; nothing else.
* **The checks:** `make pre-add-check` on the 20 Go files, "20 file(s)
  clean (gofmt, golangci-lint, go vet, go test, govulncheck)"; `tools/recordlint.py
  --changed`, no rule rose; `tools/denyscan.py --changed`, 0 hits;
  `tools/claims.py` on the MADR, 58 of 58; `python3 -B tools/gate.py`,
  "gate: 28 of 28 passed".
* **Committed as `dd41faf` (2026-10-24).** After the phase was staged, the
  owner re-assessed the thinking design and first chose to hold it (Q2 (b)
  of
  [0062-MADR-learn-thinking-parameters-from-refusals.md](0062-MADR-learn-thinking-parameters-from-refusals.md)).
  The owner then committed it, all 44 staged paths, and chose to keep the
  commit (that MADR's amendment "0061's Phase 3 committed").
  * 0062-PLAN's Phase 2 reworks its code in place, replacing D3's rule-3
    rows with 0062's ladder, in that phase's own commit.
  * Until then, `main` sends `max` as rule 3's rows say. Nothing of it is
    pushed or released.
* **Reworked by 0062-PLAN Phase 2 (2026-10-25),** in that phase's commit:
  D3's rules 1 and 2 are 0062's ladder; its rule-3 rows (Grok's clamp, and
  `high` on Gemini and OpenCode's google route) are removed; `ResolveMax`,
  `MaxEffort` and `EffortWarning` stay for Ollama alone, until 0062-PLAN's
  Phase 5. `TestEffortMax_Mapping_0061`'s Grok, Gemini and google-route rows
  and their `effort-max` goldens follow.
* **Not done here:** Phases 4 to 8. They follow 0062-PLAN (the MADR's
  amendment "D3 reworked by 0062").
* **Reworked by 0062-PLAN Phase 5 (2026-10-25),** committed as `dc7e3b0`:
  Ollama takes its effort from 0062's ladder too, and `xhigh` is sent as
  asked; `ResolveMax`, `MaxEffort` and `EffortWarning` are removed, and with
  them `TestResolveMax_Rules_0061` and `TestMaxEffort_Warning_0061`.
  `TestEffortMax_Mapping_0061`'s Ollama row and Ollama's `effort-max` golden
  did not change. Facts listing efforts, none of `low` to `max`, which rule 3
  sent `max`, now send none, warned
  ([0062-MADR-learn-thinking-parameters-from-refusals.md](0062-MADR-learn-thinking-parameters-from-refusals.md),
  amendment "Phase 5's Ollama cases").
* **Resumes at Phase 4 (2026-10-25):** 0062-PLAN is complete with its
  Phase 6.

### Phase 4: `providers/chatcompat` (2026-10-25)

Done; staged for the owner's commit. 0062-PLAN is committed through its
Phase 6 as `876b81e`.

* **Deviation 2026-10-25, step 1: the send path.** Step 1 said
  `chatcompat` sends through `wire.Post`, "as `huggingface` does". Since
  [0062-MADR-learn-thinking-parameters-from-refusals.md](0062-MADR-learn-thinking-parameters-from-refusals.md)
  (accepted), Hugging Face and every other provider send through
  `wire.Send` and `wire.StreamEventsRetry` with a per-request
  `wire.Learner` (its D5), and its D1 names `chatcompat` among the Chat
  Completions providers; no provider calls `wire.Post` (`git grep`).
  Resolutions offered: as Hugging Face now, with the Chat Completions
  matcher (recommended); a `Learner` with no matcher, as Ollama; `wire.Post`
  as written. **The owner chose as Hugging Face now.** The MADR's D4 says
  "as Hugging Face and Together do", which holds; it is not amended.
  * Step 4 gains `TestLearn_Effort_0061` in `chatcompat`: `max` refused in
    Hugging Face's recorded form is learned and sent again lower, and a
    refusal passed back with `WithReasoningRefusals` is never sent.
    `providers`' `TestLearn_Effort_0062` cannot build `chatcompat`, which
    `providers.New` does not hold.
* **Stop 2026-10-25, step 1: the options `chatcompat.New` can take.**
  `catalog.WithCache` is scoped to the built-in ids (`catalog/cache.go:43`),
  so `chatcompat.New` would refuse it, against R49 and this phase's own
  step 4 (`testCache(t)`, 0058-MADR D1); and `WithID` cannot be scoped to an
  id it alone carries. Resolutions offered: one fixed scope (recommended);
  `WithCache` for any provider, expanded at the owner's request into a
  `SharedOption` export (2a) or an internal bridge (2b); no catalog option.
  **The owner chose 2a,** first with `catalog.WithLimit` shared too; shown
  that this reverses 0038-MADR D2 and the wizard's path on it (its doc, and
  `TestWithLimit_Scope` and `TestWizard_ProviderOptionsLimitRefused`, which
  pin it), **the owner chose `WithCache` only.** The MADR's amendment
  "Phase 4's options" records it.
* **The steps, amended (2026-10-25), before any change:**
  1. `llmprovider/settings.go`: `SharedOption`; an `Option` gains its kind,
     and `take` collects a shared option's value as a scoped one's, for any
     id. Its doc says every `New` takes it.
  2. `catalog/cache.go`: `WithCache` is `SharedOption("catalog.WithCache",
     …)`; its doc says every provider's `New` takes it, and which use it.
  3. `providers/chatcompat`, as step 1 says, with: `WithID` a shared
     option; `New` resolving twice; R16's refusal of an OAuth or vendor CLI
     session; probes off unless asked; no `Authorization` header for an
     empty key; and the cache's memo and probe store from `WithCache`.
  4. `docs/guides/api-standards.md`: R17 names the shared option; R18 says
     a shared option is foreign to no provider; R22 and R38 as D4 amends
     them.
  5. Tests, besides step 4's: `TestSharedOption_0061` (`llmprovider`: every
     id resolves it, its value in `Values`, in order; a scoped option still
     refused; inside `For(id, …)`, applied to `id` alone);
     `TestWithCache_AnyProvider_0061` (a third-party id resolves
     `catalog.WithCache`); `TestNew_ForID_0061` (`For(id, WithBaseURL(…))`
     reaches the provider); `TestCache_Shared_0061` (two `chatcompat`
     providers given one `catalog.Cache` share a learned refusal, and two
     caches do not); `TestNew_Refusals_0061` gains the OAuth and vendor CLI
     sessions.
  6. Plants, besides step 5's: a shared option refused by a third party's
     `New`; `New` resolving once, so `For(id, …)` is lost; `chatcompat`
     ignoring `WithCache`.
  7. `make api-check` lists `SharedOption` and `chatcompat`'s three.
     `CHANGELOG.md`: Added, `SharedOption` and `chatcompat`; Changed,
     `catalog.WithCache` taken by every `New`.
  8. No existing test is expected to change: `TestWithLimit_Scope` and the
     wizard's tests keep `WithLimit`'s scope. A test that does is a stop.
* **The owner's request (2026-10-25), before Phase 4 runs:** evaluate
  `chatcompat` for any server, not Hugging Face's shape alone, and propose
  enhancements to its caches, listings, facts and search. The MADR's
  amendment "`chatcompat`, provider-agnostic" proposes E1 to E5, from a
  search of similar projects and a reading of the owner's local Ollama
  (`GET /v1/models`, read-only), and Phases 4a to 4e lay them out. Phase 4
  waits on the owner's approval of the amendment "Phase 4's options" and on
  which of E1 to E5 are chosen.
* **Approved (owner, 2026-10-25):** the amendment "Phase 4's options", and
  E1 to E5, all five. Phase 4 runs by its amended steps; Phases 4a to 4e
  follow it, one at a time. `chatcompat.ListModels` in Phase 4 is its own
  `GET {base}/models`, which Phase 4a moves into `catalog` (E1).
* **Amended steps 1 and 2:** `llmprovider/settings.go`, `SharedOption`, an
  `Option`'s `shared` kind, which `take` collects for any id;
  `catalog/cache.go`, `WithCache` a shared option, its doc naming who reads
  it. The packages `llmprovider`, `catalog`, `providers` and `wizard` passed
  unchanged after it, `TestWithLimit_Scope` and the wizard's
  `TestWizard_ProviderOptionsLimitRefused` among them (step 8).
* **Step 1 and amended step 3,** `llmprovider/providers/chatcompat`:
  `chatcompat.go` (the package doc; `WithID`, a shared option; `New`, which
  puts `WithModelProbes(false)` first, resolves under the scope
  `"chatcompat"` to read the last `WithID`, refuses as D4 says and an OAuth
  or vendor CLI session by R16, then resolves under the id; `Generate` and
  `Stream` through `wire.Send` and `StreamEventsRetry` with a `Learner` and
  `chatcompletions.MatchRefusal`; no `Authorization` header for an empty
  key; `ListModels`, its own `GET {base}/models`, bounded by
  `wire.ReplyLimit`, an offline failure wrapped as `ErrOffline`, probed only
  when asked, through `wire.ProbeHealth` and the cache's probe store);
  `descriptor.go`, `Descriptor(id, label)`. The ten built-in ids are listed
  in the package, and `TestNew_BuiltinIDs_0061` pins them to
  `providers.Default()`. `revive`'s `confusing-naming` found a `listModels`
  beside `ListModels`; it is `fetchModels`.
* **Step 2 and amended step 4,** `docs/guides/api-standards.md`: R17 names the
  shared option; R18 says a shared option is foreign to no provider; R22 and
  R38 as D4 amends them. `docs/architecture.md`: the tree's line, the
  package row (its imports as `go list` gives them), the options paragraph;
  and the `providers` row names its ten imports: `make tables-check` read its
  "the provider packages" as every package under `llmprovider/providers/`,
  `chatcompat` included, which `providers` does not import.
* **Step 4 and amended step 5, the tests,** each passing:
  `TestNew_Refusals_0061`, `TestNoKey_NoAuthorization_0061`,
  `TestKey_Bearer_0061`, `TestListModels_0061`, `TestNew_ForID_0061`,
  `TestLearn_Effort_0061`, `TestCache_Shared_0061`, `TestDescriptor_0061`,
  `TestNew_BuiltinIDs_0061`, `TestRegistry_0061`, `TestConformance`
  (`llmtest.Run`, R44), `ExampleNew` (R50), `TestWireGoldens`;
  `TestSharedOption_0061` (`llmprovider`); `TestWithCache_AnyProvider_0061`
  (`catalog`). The twelve G-wire goldens, written by `-update`, were each
  compared field by field with Hugging Face's of the same scenario: they
  differ only in the model, the key and the listed ids.
* **Step 5 and amended step 6, the plants,** by `tools/plant.py` on scratch
  clones: 9, all caught. The PLAN's two: the built-in check removed
  (`chatcompat_test.go:47: a built-in id: <nil>; want llmprovider: invalid
  provider id`), the bearer header always set (`chatcompat_test.go:76: POST
  /chat/completions: Authorization "Bearer"; want none`); and a shared
  option refused by another id; `New` resolving once, so `For(id, …)` is
  lost; `chatcompat` ignoring `WithCache` (`efforts sent [max high high
  high]`); `WithCache` scoped to the built-ins again; probes on by default;
  an OAuth session taken. A first plant of the sessions' case was caught
  only by its build, and was planted again as one that builds.
* **Step 6, docs:** `CHANGELOG.md`, Added (`chatcompat`, `SharedOption`) and
  Changed (`WithCache`).
* **Checks,** each exit 0: `go test -race -count=1 -shuffle=on ./...`; `make
  pre-add-check` on the 12 Go files, "12 file(s) clean"; `go vet -tags
  live_gateways ./...`; `make api-check`, "0 incompatible change(s)";
  `tools/apidiff.py v1.22.2`, `SharedOption` and the package `chatcompat`
  added beside 0061's and 0062's; `make tables-check docs-check
  changelog-check parity-check dep-check fixtures-check records-check`;
  `tools/claims.py`; `tools/recordlint.py --changed`; `tools/denyscan.py
  --changed`; `tools/linkcheck.py`.
* **The gate:** "gate: 28 of 28 passed"; `chatcompat` covered at 91.3 %, against the 80 % floor.
* **Not this phase's:** `docs/decisions/0064-MADR-anthropic-workspace-for-unscoped-keys.md`,
  its PLAN, and their two rows in `docs/README.md`, written in the tree by
  another session during this phase; they are not staged.
* **Staged:** this phase's paths.

### Phase 4a: E1, the catalog path (2026-10-25)

Done; staged for the owner's commit. Phase 4 is committed as `b663b71`.

* **Step 1:** `llmprovider/internal/compatid`, `Value`, `Of` and `For`, with
  the standard library alone; R2's table gains its row, and `catalog`'s may
  import it.
* **Step 2,** `catalog`:
  * `config.go`: `config.compat`, set for an id no built-in has that the
    options' `chatcompat.WithID` names, compared without case, which also
    turns the models.dev document off for it; `listingConfig`, which lists
    such an id given a base URL, refuses one without it (ErrInvalidRequest),
    and gives any other unknown id `unlistedProvider`'s ErrUnsupported, as
    before, whatever its options.
  * `discovery.go`: `List` resolves through `listingConfig`; `listOnce`
    admits a compat listing, reads no document for it, and uses the listing
    cache, the flight and the last good listing unchanged;
    `fetchCompatUsable`, `fetchDataIDs` on `{base}/models`, no
    `Authorization` header for an empty key, empty ids dropped;
    `compatCurate`, the server's order cut to the limit, as
    `buildListing`'s curator for it. With no dates, `menuOf` orders the
    models past the first six by id; Phase 4b's default rank (E4) orders
    them.
  * `lookup.go`: `lookupConfig` and `hasListingLayer` replace
    `lookupProvider`, so `LookupModel`, `CachedModel` and `CachedFacts` take
    a compat id named by `WithID`, its listing its one layer besides the
    caller's; any other unknown id is ErrInvalidProvider, as before.
  * **Found and fixed on the way:** `make pre-add-check` on `config.go`
    reported `unparam` on `tokenHeader`, every one of its four callers
    passing `"Authorization"` and `"Bearer"`, which this phase's fifth call
    made no different; it is `bearerHeader(tok)`, with both inside, and no
    request changes.
* **Step 3,** `chatcompat`: `WithID`'s value is `compatid.Value`;
  `ListModels` is `catalog.List`'s `Recommended`, a listing that is not live
  returning its failure (`listingError`: as it came with a kind, else
  ErrIncomplete), probed when asked as before; each request reads
  `catalog.CachedFacts`, for the output limit, the efforts and the replay
  field, as Hugging Face's does. Its package doc and `ListModels`' say so.
  Its twelve G-wire goldens did not change: the listing request through
  `catalog` is the one it sent itself.
* **Step 4, the tests,** each passing, the catalog's under `-race`:
  `TestCompatListing_Cached_0061` (one request for two Lists within ten
  minutes, one for five at once, one more after eleven minutes; the first
  six in the server's order; empty and repeated ids dropped),
  `TestCompatListing_LastGood_0061`, `TestCompatListing_UnknownRefused_0061`,
  `TestCompatListing_NoKey_0061`; `TestOfAndFor_0061` (`compatid`);
  `TestListModels_Cached_0061` (`chatcompat`, two providers of one cache,
  one request). Every existing test of `catalog`, `providers`, `chatcompat`
  and `wizard` passed unchanged.
* **Step 4's plants,** by `tools/plant.py` on scratch clones: 8, all caught.
  The PLAN's two: the cache bypassed (`compat_0061_test.go:53: a second List
  within ten minutes: <nil>, 2 request(s); want 1`) and the flight bypassed
  (`five Lists at once: 5 requests; want 1`); and the last good listing not
  served; an unknown id listed without `WithID`; an empty key sent as a
  bearer token; empty ids kept; a failed listing returned as an empty one;
  an unreadable listing kept without a kind. The unknown-id plant was first
  caught only by its build, and was planted again as one that builds.
* **Docs:** `docs/guides/api-standards.md` R2 (the row and the citation);
  `docs/architecture.md`, the rows of `catalog`, `chatcompat` and
  `internal/compatid`; `catalog.List`'s and `LookupModel`'s docs;
  `CHANGELOG.md`, `chatcompat`'s entry.
* **Checks,** each exit 0: `go test -race -count=1 -shuffle=on ./...`; `make
  pre-add-check` on the 8 Go files, "8 file(s) clean"; `go vet -tags
  live_gateways ./...`; `make api-check`, "0 incompatible change(s)";
  `tools/apidiff.py v1.22.2`, no addition beside Phase 4's; `make
  tables-check docs-check changelog-check dep-check fixtures-check
  records-check`; `tools/claims.py`; `tools/recordlint.py --changed`;
  `tools/denyscan.py --changed`; `tools/linkcheck.py`.
* **The gate:** "gate: 28 of 28 passed"; covered: `catalog` 95.9 %, `internal/compatid` 100 %, `chatcompat` 91.7 %, each against its 80 % floor.
* **Not this phase's:** the 0064 pair and its rows in `docs/README.md`,
  another session's; not staged.
* **Staged:** this phase's paths.

### Phase 4b: the owner's request (2026-10-25)

* Phase 4a is committed as `590b994`. Before Phase 4b ran, the owner asked
  for fuzzy matching, BM25, and a clearer search prompt and results. The
  MADR's amendment "Model search, rebuilt" proposes it, from a reading of
  `catalog/model_matcher.go` and `wizard/model_select.go` and a search of
  fzf's algorithm, BM25F's tuning, rank fusion and model pickers; Phase 4b's
  steps are revised to it, and `wizard` joins the scope. Nothing is changed
  until the owner approves.
* **Approved (owner, 2026-10-25):** the amendment, with the tier key over
  Reciprocal Rank Fusion and over BM25F alone, and the aligned column on
  every `TextPrompter` menu. Phase 4b runs by its revised steps.
* **Step 1, before any change:**
  * The tests that pin Search's results or the wizard's and `TextPrompter`'s
    output, found by searching every `_test.go` for `Search(`, the prompt
    constants, `maxSearchResults`, the cap's notice, `matchChoices`,
    `renderChoices`'s row format and the no-match menu: 21 Search tests in
    `catalog`, and the wizard's `model_select_test.go`,
    `select_ctx_0026_test.go`, `inputctx_0057_test.go`, `configure_test.go`
    and `fake_prompter_test.go`. **Only these change:**
    * `catalog/model_matcher_test.go`'s `TestSearchModels_Subsequence`: its
      query, `sonet`, is within one edit of `sonnet`, so the fuzzy tier
      (1500) answers it, where the subsequence tier (1035) did; the test
      takes `cdsnt`, a subsequence of `claude-sonnet-5` and of no word, and
      `sonet` goes to `TestSearch_Typo_0061`.
    * `wizard/model_select_test.go`'s `TestConfigureLLM_SearchResultsCapped`,
      which pins twenty rows and the cap's notice: paging shows ten and
      "More", and no notice; it is restated, and renamed
      `TestConfigureLLM_SearchResultsPaged`.
    Every other is expected to pass unchanged; one that does not is a stop.
  * **`BenchmarkSearch_1000_0061`'s baseline,** today's `Search` over a
    thousand ids (`search_bench_0061_test.go`, written first), `-benchtime
    200x -count 3`, the median of each: `sonnet` 1.12 ms, 533 KB, 4514
    allocations; `qwen 32b` 1.28 ms, 612 KB, 6012; `qwen3:8b` 1.45 ms, 622
    KB, 5183; `gema` 1.38 ms, 616 KB, 5186; `zzzz` 1.34 ms, 604 KB, 5178.
* **Step 2, so far:** `catalog/search.go` (the engine), `Search` on it,
  `Match.Why`, `Catalog.Search`, the index built in `List`. The existing
  tests: only `TestSearchModels_Subsequence` failed (`scores = [1500], want
  [1035]`), as step 1 listed; it takes `cdsnt` and scores 1010 (two token
  starts, no consecutive letters). Every other test in `./llmprovider/...`
  and `./wizard` passed unchanged.
* **The benchmark after,** `-benchtime 200x -count 3`, the median:
  * `BenchmarkSearch_1000_0061`, `Search` over a thousand ids, which indexes
    them on every call: `sonnet` 2.16 ms, 937 KB, 1151 allocations; `qwen
    32b` 2.35 ms, 951 KB, 1384; `qwen3:8b` 2.50 ms, 997 KB, 1869; `gema` 2.31
    ms, 930 KB, 1198; `zzzz` 2.26 ms, 931 KB, 1251. Against the baseline:
    about a quarter of the allocations, about 1.6 times the bytes and
    about 1.8 times the time, the index's cost. Two changes made in the
    measuring brought it there from 3.3 ms, 2.5 MB and 28 842 allocations:
    the index read through an accessor, not a `MenuEntry` per id, and every
    document's words in one backing array, sized at 32 words a model.
  * `BenchmarkCatalogSearch_1000_0061` (added), `Catalog.Search` on the
    index built once, as `List` builds it: `sonnet` 0.16 ms, 34 KB, 102
    allocations; `qwen 32b` 0.31 ms, 48 KB, 335; `qwen3:8b` 0.46 ms, 94 KB,
    820; `gema` 0.31 ms, 26 KB, 149; `zzzz` 0.24 ms, 28 KB, 202: from three
    to seven times faster than the baseline, with a tenth of its
    allocations or fewer.
* **Deviation 1 (2026-10-25): `Catalog.Search`'s documents.** Found: the
  MADR's S1 says `Catalog.Search` searches `c.Menu`, but `List` cuts `Menu`
  to `WithLimit(n)` and not `Usable`, the wizard passes `WithLimit`, and
  `Usable`'s order breaks ties (0026-MADR F54). Decision (owner): it searches
  `Usable`, in its order, with `Menu`'s facts by id; `Menu`'s ids when
  `Usable` is empty (the MADR's "S1 amended in execution"). No file joins
  the phase's scope.
* **Deviation 2 (2026-10-25): E4's menu half moves to Phase 4c.** Found:
  step 2 put `menu.go`'s variant grouping and default rank in Phase 4b, but
  the MADR's E4 groups variants by "E2's" canonical id and ranks loaded
  models first "where the server says", E2's dialect details; E2 is Phase
  4c. Decision (owner): both move to Phase 4c, after E2's canonical id and
  dialects, with `TestMenu_Variants_0061` and `TestMenu_DefaultRank_0061`
  and a plant (variants not grouped); Phase 4b is search and the picker.
  Steps 2 and 4 are struck through above, and Phase 4c's steps 2 and 3
  gain them. No file joins either phase's scope beyond `menu.go`, already
  E4's.
* **Deviation 3 (2026-10-25): "Did you mean" dropped.** Found: S4 offers
  the nearest three ids within the edit bound when a search finds nothing,
  but the engine returns every model one of whose words is within that
  bound of a query word (`scoreNearAll` for one word, `scorePartial` for
  several), so when nothing matches no id is within the bound and the list
  is empty by construction. Decision (owner): dropped; the no-match menu
  stays as it is ("Use … as the model id", "Search again"), and the near
  tiers' rows, each with its `Why` ("near sonnet"), are the corrections.
  `TestSearch_DidYouMean_0061` is struck. S4's other point found empty: the
  exact id is always the first row (`scoreExact`, the highest tier), so
  the default row, the first, is it with no code; `TestSearch_Paging_0061`
  asserts it.
* **Steps 2 to 6, as built:**
  * `catalog`: `search.go`, the engine (S2, S3, S5); `model_matcher.go`'s
    tiers on the index's precomputed forms (`tierScore`, `tokensPrefixed`),
    `Match.Why`, a glob's `Why` (`id pattern`, `name pattern`), and `Search`
    on the engine; `Catalog.Search`, over `Usable` with `Menu`'s facts
    (deviation 1); `Catalog`'s unexported `provider` and `index`, set by
    `List` (`discovery.go`); the package doc.
  * The facets' BM25F length counts the context buckets as one word (the
    MADR's "S2's facet length, made exact in execution"): measured on the
    words alone, a 2M-context model ranked below an 8k one on `vision`.
  * `wizard`: the prompt; each search row's `Detail`, the facts known then
    `Why` (`matchDetail`, `tokenCount`, `price`); the counted title
    (`searchResultsTitle`, `matchCount`); paging (`pageMatches`, ten a page,
    `moreMatchesLabel`); `searchCatalog`, `Catalog.Search` on a listing
    `List` built (its `Source` set), else `catalog.Search` over `Usable`,
    the excluded left out for the fallbacks; `menuFacts`. The fallbacks'
    `MultiSelect` keeps its cut of twenty and its notice, its rows gaining
    `Detail`: S4's paging is a menu's More row, which a `MultiSelect`, whose
    answer is its picks, has no turn for. `searchFallbackPrompt` is
    unchanged. `TextPrompter.renderChoices` right-aligns the numbers and pads
    the labels of rows with a detail to the longest, at most 40 runes
    (`maxLabelPad`).
  * Lint: gocritic's `argOrder` (`isSizeUnit`, now a `switch`) and
    `initClause` (`fzfScore`'s two scans, now `continue` and the step before
    the test), fixed before the checks below.
* **Tests added:** `catalog/search_0061_test.go`: `TestTokenise_0061`,
  `TestDistance_0061` (`editsWithin` against the whole OSA table, every
  pair at bounds 0 to 3, and `editBound`), `TestBM25F_0061` (the id over
  the family, a rare word over a common one, the buckets' length),
  `TestSearch_Key_0061`, `TestSearch_Typo_0061` (`sonet`, `gemnii`,
  `qwen3:8`, and a partial match), `TestSearch_Facets_0061`,
  `TestCatalogSearch_Usable_0061`, `TestCatalogSearch_Index_0061` (`List`'s
  index, its provider, a replaced `Usable`, at most 300 allocations a query
  on a built index), `TestSearch_ExistingTiers_0061` (each tier's score and
  `Why`), `TestSearch_Glob_Why_0061`; `search_bench_0061_test.go`'s
  `BenchmarkCatalogSearch_1000_0061`; `wizard/search_0061_test.go`:
  `TestSearch_Paging_0061`, `TestSearch_Detail_0061`,
  `TestRenderChoices_Aligned_0061`. Restated: `TestSearchModels_Subsequence`
  (`cdsnt`, 1010) and `TestConfigureLLM_SearchResultsPaged` (three pages,
  the counted title, no notice), as step 1 listed.
* **The benchmark, final,** `-benchtime 200x -count 3`, the median, after
  the gate:
  * `BenchmarkSearch_1000_0061`: `sonnet` 2.31 ms, 970 KB, 1151
    allocations; `qwen 32b` 2.50 ms, 984 KB, 1384; `qwen3:8b` 2.62 ms, 1030
    KB, 1869; `gema` 2.56 ms, 962 KB, 1198; `zzzz` 2.58 ms, 964 KB, 1251:
    from 1.8 to 2.1 times the baseline's time, 1.6 to 1.8 times its bytes,
    and a quarter of its allocations or fewer; the index, built per call.
  * `BenchmarkCatalogSearch_1000_0061`: `sonnet` 0.16 ms, 34 KB, 102;
    `qwen 32b` 0.32 ms, 48 KB, 335; `qwen3:8b` 0.47 ms, 94 KB, 820; `gema`
    0.31 ms, 26 KB, 149; `zzzz` 0.23 ms, 28 KB, 202: from three to seven
    times faster than the baseline.
* **Plants,** `tools/plant.py` on a scratch clone, `./llmprovider/catalog/`
  and `./wizard/`, `-run '_0061|Search'`, run again after the lint fixes;
  all 7 caught:
  * the tier key dropped (BM25F alone orders): `TestSearch_Key_0061` (`zeta
    = [zeta-pro zeta], want the exact id before the prefix`) and
    `TestSearchModels_Tiers`;
  * a fuzzy bound of 3: `TestSearchModels_NoSubsequenceOverLabels` (`hspd =
    [fireworks/llama-ash …], want no matches`), `TestSearch_Typo_0061`,
    `TestSearch_Facets_0061` and two more;
  * a facet read where the facts are silent (`vision` from `Input` without
    `Has`): `TestSearch_Facets_0061` (`"vision" = [claude-sonnet-5 gpt-x
    silent]`);
  * the buckets counted whole in the facets' length: `TestBM25F_0061`
    (`vision = [small big], want big, small`);
  * the index built per query: `TestCatalogSearch_Index_0061` (`1151
    allocations a query, want at most 300`);
  * paging lost: `TestConfigureLLM_SearchResultsPaged` and
    `TestSearch_Paging_0061`;
  * the labels not padded: `TestRenderChoices_Aligned_0061` (`line 2 = "
    1) a — x"`).
* **Checks:** `make pre-add-check` on the 11 Go files: clean (gofmt,
  golangci-lint for darwin and windows, go vet, go test, govulncheck);
  `go test -race -count=1 -shuffle=on ./...`, `go vet -tags live_gateways
  ./...`, `make api-check` (0 incompatible changes against `v1.22.2`),
  `tools/apidiff.py v1.22.2`, `tables-check`, `fixtures-check`,
  `changelog-check`, `docs-check`, `records-check`, `recordlint --changed`,
  `denyscan --changed` (0 hits), `linkcheck`: each exit 0;
  `tools/gate.py`: 28 of 28 passed.
* **Not done in this phase:** E4's menu half (deviation 2, to Phase 4c);
  "Did you mean" (deviation 3, dropped).
* **Staged:** `CHANGELOG.md`, `docs/architecture.md`, this record and its
  MADR, `llmprovider/catalog/{discovery,doc,model_matcher,search}.go`,
  `llmprovider/catalog/{model_matcher,search_0061,search_bench_0061}_test.go`,
  `wizard/{model_select,text_prompter}.go`,
  `wizard/{model_select,search_0061}_test.go`.

### Phase 4c: before any change (2026-10-25)

* Phase 4b is committed as `fca2133`. Reading the listing, the Ollama
  reader and the menu builders, and probing the local Ollama 0.33.2 and the
  recorded models.dev document, found four points the record left open or
  contradicted; the owner decided each (the MADR's amendment "E2 and E4's
  menu, made exact before Phase 4c"):
  * **Deviation 4: the variant key.** E4's example does not group under
    E2's canonical id. Decision: a looser key, the canonical id less the
    role tokens.
  * **Deviation 5: the scope of grouping and the default rank.** "For every
    provider" would reorder ranked bands. Decision: unranked listings only,
    `chatcompat`'s and the built-in Ollama's; Ollama's menu leaves
    0041-MADR A4's `/api/tags` order. `llmprovider/catalog/menu.go`'s
    `ollamaMenu` and `discovery.go`'s `ollamaTag` (its `modified_at`) join
    the phase's scope.
  * **Deviation 6: the dialect rule.** None was set. Decision: every entry
    a `:`-tagged id, an integer `created`, an `owned_by`; the threshold all.
  * **Deviation 7: models.dev for a compat listing.** It was off. Decision:
    fetched as a built-in's is.
* **The revised steps:**
  1. **Record** (`LLMPROVIDER_LIVE_0061_COMPAT` naming a base URL, in a
     live-tagged test that writes the fixtures): the local Ollama's
     `/v1/models`, `/api/tags`, and `/api/show` for `qwen3:8b` and
     `gemma3:12b`, trimmed as X7 says, into
     `llmprovider/catalog/testdata/compat/ollama/`; `make fixtures-check`.
  2. `catalog`: `canonical.go` (X1, X2, X5's merge); `dialect.go` (the
     dialect table and score, decision 3, and the Ollama dialect's details,
     X4); the compat fetch keeps `created` and runs the dialect; the compat
     listing fetches models.dev (decision 4) and reads it by canonical id
     as its lowest layer, beside `mergeLayers`; `menu.go`'s default rank
     and variant grouping (X2, X3) for the compat and Ollama menus.
  3. **Before the tests change,** every existing test is run on the new
     code; a failure is kept only if it pins the built-in Ollama's menu
     order across models whose window or `modified_at` differ (deviation
     5), each listed here with its output before it is restated. Any other
     failure is a stop.
  4. Tests: `TestDialect_Score_0061`, `TestCanonicalID_0061`,
     `TestFacts_Order_0061`, `TestCanonicalMatch_ExactOnly_0061`,
     `TestCanonicalMerge_0061` (X5's majority and ties),
     `TestMenu_Variants_0061`, `TestMenu_DefaultRank_0061` (compat and
     Ollama). Plants: the threshold removed; a fuzzy canonical match;
     variants not grouped; the role tokens in the canonical id; a tie
     resolved to the larger limit.
  5. Docs: `catalog`'s package doc, `Catalog.Menu`'s doc, the CHANGELOG
     (Changed: Ollama's menu order; Added: a compat listing's facts),
     `docs/architecture.md`'s Ollama and listing paragraphs, `AGENTS.md`'s
     live variables if `scripts/check_tables.py` asks.
* **Step 1, done:** `TestLive_RecordCompat_0061`
  (`live_0061_compat_record_test.go`), with `LLMPROVIDER_LIVE_0061_COMPAT`
  `http://127.0.0.1:11434/v1`, `LLMPROVIDER_LIVE_0061_COMPAT_DIALECT`
  `ollama` and `LLMPROVIDER_LIVE_0061_COMPAT_SHOW` `qwen3:8b,gemma3:12b`,
  wrote `models.json` (700 bytes), `tags.json`
  (3341), `show-qwen3_8b.json` (1371) and `show-gemma3_12b.json` (1775)
  into `llmprovider/catalog/testdata/compat/ollama/`. A search of them for a
  home path and the local account name found none; `make fixtures-check`:
  353 files, 0 problems. `/v1/models`' `created` of `llama3.2:latest`,
  1783879777, is 2026-07-12T18:09:37Z, the instant of its `/api/tags`
  `modified_at` (`13:09:37-05:00`), as X3 says.
* **Step 3, the first run** (`go test -count=1 ./...`, on step 2's code):
  one failure, `TestOllamaUsable_0042`, cases S4 and S5:
  `granite-embedding:30m: caveats [no-text-output]; want []` and the same
  for `nomic-embed-text:latest`, each then out of `Recommended`. It is not
  the kind step 3 admits: no order test failed; X3's id-token caveat met
  0042-MADR D1, which keeps a model whose output Ollama does not report.
* **Deviation 8 (2026-10-25): the id-token caveat for chatcompat only.**
  Decision (owner): only a `chatcompat` listing reads an id's tokens; the
  built-in Ollama keeps 0042-MADR D1, and S4 and S5 stand unchanged. The
  MADR's X3 says so. Applying it to both, amending 0042-MADR D1, and
  dropping the id-token rule were weighed and not chosen.
* **Step 3, the second run** (`go test -count=1 ./...`, after deviation 8):
  every package passed. No existing test pins the built-in Ollama's order
  across models whose window or `modified_at` differ.
* **Step 3, the outside network:** decision 4 starts a models.dev fetch
  for a compat listing, so a test that lists one without turning the
  document off reaches `models.opencode.ai`. Measured by running the tests
  through a logging proxy that refuses every request
  (`HTTPS_PROXY`, each package alone): on this phase's code, `catalog` 30
  `CONNECT models.opencode.ai:443`, `chatcompat` 4, `wizard` 0; on a scratch
  clone of `fca2133`, `catalog` 24, `chatcompat` 0, `wizard` 0. The 24 are
  older than this phase and outside it, and are reported, not fixed. The
  10 new ones were `compat_0061_test.go`'s `compatList` and
  `TestCompatListing_NoKey_0061`, and `chatcompat`'s `build`: each now
  passes `llmprovider.WithoutModelMetadata()`. After: `catalog` 24,
  `chatcompat` 0, as before the phase.
* **Deviation 9 (2026-10-25): `catalog.WithModelLayer` shared.** Found:
  `TestFacts_Order_0061` failed with `model listing: unsupported provider
  "acme"`: `WithModelLayer` is `ScopedOptionFor(builtinIDs)`, so a compat
  id refuses it, and X6's caller layer never reaches a compat listing.
  Decision (owner): a shared option (the MADR's X8), as `WithCache` is;
  `llmprovider/catalog/lookup.go`, `lookup_scope_0033_test.go`'s
  `TestWithModelLayer_Scoped_0033` (its other id now takes the option,
  ignored, where it refused it; restated) and `docs/guides/api-standards.md`'s
  R17 join the phase's scope.
* **Deviation 10 (2026-10-25): the kept sections.** Found: a test document
  whose one section was `acme-cloud` gave no canonical facts:
  `decodeModelMetadata` keeps only the nine sections the SDK reads. Decision
  (owner): the match searches those nine (the MADR's X9); the test's
  document is a `huggingface` section. No file joins the phase's scope.
* **As built:**
  * `catalog/canonical.go`: `canonicalTokens`, `canonicalID` (X1),
    `variantKey`, `plainness` (X2), `Metadata.modelFor`, which reads a
    compat model by canonical id and any other by its section, and
    `Metadata.canonicalModel` on an index built once per document
    (`modelMetadataDoc.canonical`), its groups merged by `mergeCanonical`
    and `majority` (X5).
  * `catalog/dialect.go`: `compatEntry`, the table `dialects` (Ollama's
    three marks), `dialectScore`, `dialectOf` at `dialectThreshold` 1
    (decision 3), `ollamaDialectDetails` (X4), `fetchCompatEntries`.
  * `discovery.go`: `fetchCompat` (ids, created times, the dialect's
    details) in place of `fetchCompatUsable` and `compatCurate`; the
    models.dev fetch started for a compat listing (decision 4);
    `listingResult.created` and `fetchedListing.created`; `ollamaTag`'s
    `modified_at`, parsed leniently, as the built-in Ollama's created
    times; `compatMenu` for a compat listing.
  * `menu.go`: `defaultRank` (X3), `groupVariants` (X2) within the
    uncaveated and within the caveated, `compatMenu` (its band the first six
    uncaveated, by place), `compatCaveats` and `noTextTokens` (a compat
    listing only, deviation 8), `ollamaMenu` on the default rank.
  * `config.go`: a compat listing no longer turns the document off;
    `lookup.go`: `modelFor` in `LookupModel`, `CachedModel` and
    `CachedFacts`, and `WithModelLayer` shared (X8); `menu.go`'s
    `factsOf` reads `modelFor`.
  * `models_catalog.go`: `denyWhisper` beside `denyEmbed` and `denyTTS`,
    for goconst, which counted `menu.go`'s `noTextTokens`' literals with
    its own; it joins the phase's scope for that alone.
  * `chatcompat.go`'s package doc, `catalog`'s package doc, `List`'s and
    `Catalog.Menu`'s docs, `docs/architecture.md`, `docs/guides/api-standards.md`'s
    R17, the CHANGELOG's Changed.
* **Tests:** added `compat_facts_0061_test.go`: `TestDialect_Score_0061`
  (each recorded dialect's reply, every other recorded `{"data": [...]}`
  listing, one untagged id, one entry without `created`),
  `TestCanonicalID_0061`, `TestCanonicalMatch_ExactOnly_0061` (on the
  recorded models.dev document: `gpt-oss:20b` matches; `qwen3:8b`,
  `gpt-oss:21b`, `gpt-oss`, `deepseek-r1:7b`, `llama3.2:latest` do not),
  `TestCanonicalMerge_0061`, `TestFacts_Order_0061` (the recorded Ollama
  served by a fake, a caller's layer, a models.dev document: each field's
  value and source), `TestMenu_Variants_0061`, `TestMenu_DefaultRank_0061`
  (a compat menu, Ollama's, a plain server whose order is not its created
  order, and the recorded Ollama end to end); `live_0061_compat_record_test.go`.
  Restated: `TestWithModelLayer_Scoped_0033` (deviation 9). Changed, for
  the outside network only: `compatList`, `TestCompatListing_NoKey_0061`,
  `chatcompat`'s `build`.
* **Plants,** `tools/plant.py` on a scratch clone, `./llmprovider/catalog/`,
  `-run '_0061|_0042'`, run again on the final code; all 9 caught:
  * the threshold removed: `TestDialect_Score_0061` (`one untagged id:
    dialect "ollama", want none`);
  * a fuzzy canonical match: `TestCanonicalMatch_ExactOnly_0061`
    (`qwen3:8b matched {… Qwen3 VL 30B A3B Thinking …}`);
  * variants not grouped: `TestMenu_Variants_0061`;
  * the role tokens in the canonical id: `TestCanonicalID_0061` (`an
    instruct model and its base share the canonical id
    "qwen3-235b-a22b-2507"`);
  * a tie resolved to the larger limit: `TestCanonicalMerge_0061`;
  * the id-token caveat on the built-in Ollama: `TestMenu_DefaultRank_0061`
    and `TestOllamaUsable_0042` S4 and S5;
  * the server's details dropped, and the details read under `/v1`, not
    the root: `TestFacts_Order_0061` and `TestMenu_DefaultRank_0061`;
  * the created times not read: `TestMenu_DefaultRank_0061` (`a plain
    server's menu = [a b c d]`), caught only once its plain-server case
    was added: the recorded Ollama lists the newest first, so its order
    passed for created's, and the first run of the plants did not catch it.
* **Checks:** `make pre-add-check` on the 15 Go files: clean (after
  goconst's `denyWhisper`); `go test -race -count=1 -shuffle=on ./...`,
  `go vet -tags live_gateways ./...`, `make api-check` (0 incompatible
  changes against `v1.22.2`), `tools/apidiff.py v1.22.2`, `tables-check`
  (after the live variables were named in full in step 1's record),
  `fixtures-check`, `changelog-check`, `docs-check`, `records-check`,
  `recordlint --changed`, `denyscan --changed`, `linkcheck`: each exit 0;
  `tools/gate.py`: 28 of 28 passed. The outside network, through the logging proxy, on the final
  code: `catalog` 24, `chatcompat` 0, `wizard` 0, as before the phase.
* **Not done in this phase:** a dialect besides Ollama's (none recorded:
  LM Studio, llama.cpp, vLLM and TGI wait for the owner's servers and step
  1's test); "loaded first" (no recorded dialect says it); the 24 requests
  to `models.opencode.ai` older tests send, which the owner then asked to
  have fixed: [0065-MADR-tests-reach-no-outside-host.md](0065-MADR-tests-reach-no-outside-host.md),
  measured across the module at 99.
* **Staged:** `CHANGELOG.md`, `docs/architecture.md`,
  `docs/guides/api-standards.md`, this record and its MADR,
  `llmprovider/catalog/{canonical,config,dialect,discovery,doc,lookup,menu,model_metadata,models_catalog}.go`,
  `llmprovider/catalog/{compat_0061,compat_facts_0061,live_0061_compat_record,lookup_scope_0033}_test.go`,
  `llmprovider/catalog/testdata/compat/ollama/`,
  `llmprovider/providers/chatcompat/chatcompat.go`,
  `llmprovider/providers/chatcompat/helpers_test.go`.
