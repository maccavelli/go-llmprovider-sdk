---
status: proposed
date: 2026-10-19
associated-madr: "0046-MADR-v1-15-0-follow-ups.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Closing v1.15.0's Follow-ups

Associated MADR: [0046-MADR-v1-15-0-follow-ups.md](0046-MADR-v1-15-0-follow-ups.md)

## Goal

When this PLAN is complete, all of the following hold:

* The three docs tables match the code, and `scripts/check_tables.py`
  fails CI on any new drift (D4, A1).
* Every check in the five person-gated live tests runs in an offline twin,
  and is seen to fail on a planted break there (D3).
* Each consumer has an adoption issue (D2).
* Issues 2 and 3 carry an accurate closing comment, and are closed (D1).

Nothing is published before the owner asks for it, in the same turn.

## Scope

### In scope

* **Phase 1 (D4):**
  * the R2 rows, the architecture rows and `AGENTS.md`'s live paragraph;
  * `scripts/check_tables.py`, with its self-test in
    `scripts/test_gates.py`;
  * the `Makefile` target, CI's step and `tools/gate.py`.
* **Phase 2 (D3):**
  * `llmprovider/live_checks_test.go` and `llmprovider/live_twins_test.go`,
    both untagged;
  * the two live test files that call them;
  * the helpers that move out of two tagged files.
* **Phase 3 (D2):** the two drafts in Appendix B, opened in the consumer
  repositories on the owner's ask.
* **Phase 4 (D1):** the two comments in Appendix A, posted on the owner's
  ask, then both issues closed.

### Out of scope

* Any production `.go` file, exported identifier or wire golden. Phase 2
  plants into three production files on a scratch clone only.
* Any change inside gobble-cli or prepare-commit-msg.
* A release: nothing exported changes, so no tag is made.

## Preconditions and conventions

* **Approval.** 0046-MADR is accepted, with A1. The owner approves this
  PLAN, whole or by phase.
* **`<base>`** is the commit that carries 0046-MADR A1 and this PLAN.
* **Gate, at the end of each phase that changes the tree (1 and 2):**
  * `tools/gate.py`: 23 checks before Phase 1, 24 from its end;
  * `make pre-add-check` on the phase's Go files;
  * `tools/apidiff.py v1.15.0`, which must print nothing;
  * `tools/denyscan.py --changed`, `make records-check`;
  * `tools/claims.py` on this PLAN.

  The phase is then staged for the owner's `git commit --no-edit`.
* **Red first.** Every new check and assertion is seen to fail on a
  planted break, by `tools/plant.py` or `scripts/test_gates.py`, on a
  scratch clone, never in the tree. A plant caught only by a build failure
  is rewritten to keep the build, and run again.
* **Publishing** (Phases 3 and 4): the exact text is in a file, passes
  `tools/denyscan.py` on that file, and is shown to the owner. It is sent
  only on the owner's word in that turn. Each URL goes into this record.
* **Stop conditions:**
  * a fact of 0046-MADR, or a claim below, no longer holds;
  * a step would change a production file;
  * a change to a check function would change what a live test asserts
    (R46);
  * a consumer's `HEAD` has moved, so its claims fail.

## Code claims

The lines this PLAN's steps change or plant into, at `<base>`'s parent
`d32bfa8`, which they all share.

<!-- claims: self=d32bfa8 -->

| # | What the step touches | Where | Text |
| :--- | :--- | :--- | :--- |
| K1 | R2's citation, which gains 0046 (Phase 1 step 2) | `docs/guides/api-standards.md:27` | `` F62; `internal/listlimit`, 0038 A3; `internal/chatgpt`, 0041 D9) `` |
| K2 | R2's row for the catalogs command | `docs/guides/api-standards.md:39` | `` \| `llmprovider`, `catalog`, `internal/catalogtool` \| `` |
| K3 | R2's row for the wire formats | `docs/guides/api-standards.md:46` | `` \| `llmprovider/internal/wire/<format>` \| one wire format \| `llmprovider`, `internal/wire`, `internal/transport`, `internal/redact` \| `` |
| K4 | R2's row for `wizard` | `docs/guides/api-standards.md:48` | `` \| the above, `golang.org/x/term` \| `` |
| K5 | the architecture row for `wizard` | `docs/architecture.md:91` | `` \| `llmprovider`, `auth`, `catalog`, `providers`, `internal/redact`, `golang.org/x/term` \| `` |
| K6 | the architecture row for `kilo` | `docs/architecture.md:102` | `` `internal/kiloendpoint`, `internal/wire`, `internal/wire/deferral` \| `` |
| K7 | the architecture row for the catalogs command | `docs/architecture.md:122` | `` \| `llmprovider`, `catalog`, `internal/catalogtool` \| `` |
| K8 | `AGENTS.md`'s live paragraph, the line that names `DEVICE_LOGIN` | `AGENTS.md:180` | `` Grok browser logins), `LLMPROVIDER_LIVE_DEVICE_LOGIN` (the Grok device-code `` |
| K9 | `AGENTS.md`'s list of the CI checks | `AGENTS.md:146` | `` generate-check records-check gate-selftest`; run them before asking for a push. `` |
| K10 | the `Makefile`'s `.PHONY` line | `Makefile:14` | `.PHONY: all help test test-sum fmt vet lint tidy vuln pre-add-check parity-check dep-check coverage-check api-check generate-check records-check gate-selftest tools-test` |
| K11 | the `Makefile`'s `records-check` target, after which `tables-check` goes | `Makefile:83` | `records-check: ## Checks the decision records and their index` |
| K12 | CI's step | `.github/workflows/ci.yml:60` | `run: make parity-check dep-check coverage-check api-check generate-check records-check gate-selftest` |
| K13 | the gate's list | `tools/gate.py:98` | `make("records-check"),` |
| K14 | the self-test's gates | `scripts/test_gates.py:37` | `"records-check": "scripts/check_records.py",` |
| K15 | the helper that moves from a tagged file (Phase 2 step 1) | `llmprovider/live_chatgpt_test.go:20` | `func liveOpenAI(t *testing.T, src llmprovider.TokenSource, model string, opts ...llmprovider.Option) llmprovider.Provider {` |
| K16 | the second | `llmprovider/live_chatgpt_test.go:29` | `func userText(text string) *llmprovider.Request {` |
| K17 | the third | `llmprovider/live_grok_test.go:22` | `func liveGrok(t *testing.T, src llmprovider.TokenSource, model string, opts ...llmprovider.Option) llmprovider.Provider {` |
| K18 | the live suite's transient skip, which a check takes as a function | `llmprovider/live_gateways_test.go:52` | `func skipIfTransient(t *testing.T, err error) {` |
| K19 | the ChatGPT login's check | `llmprovider/live_oauth_login_test.go:74` | `out, err := llmprovider.GenerateText(ctx, liveOpenAI(t, session, "gpt-6-astra"), userText("Reply with only the word ALPHA"))` |
| K20 | the Grok browser login's check | `llmprovider/live_oauth_login_test.go:97` | `out, err := llmprovider.GenerateText(ctx, liveGrok(t, session, "grok-4.6"), userText("Reply with only the word ALPHA"))` |
| K21 | the Grok device login's check | `llmprovider/live_oauth_login_test.go:123` | `out, err := llmprovider.GenerateText(ctx, liveGrok(t, session, "grok-4.6"), userText("Reply with only the word ALPHA"))` |
| K22 | Kilo's profile, which the check passes options to | `llmprovider/live_oauth_login_test.go:153` | `account, err := auth.KiloProfile(ctx, session.Access)` |
| K23 | the plan's three steps, which move | `llmprovider/live_chatgpt_plan_test.go:67` | `func planSignIn(ctx context.Context, t *testing.T) (*auth.OAuthSession, auth.TokenStore) {` |
| K24 | the second | `llmprovider/live_chatgpt_plan_test.go:105` | `func planRequests(ctx context.Context, t *testing.T, session *auth.OAuthSession) {` |
| K25 | the third | `llmprovider/live_chatgpt_plan_test.go:155` | `func planRefresh(ctx context.Context, t *testing.T, session *auth.OAuthSession, store auth.TokenStore) {` |
| K26 | the profile's route, which the twin's fake answers by suffix | `llmprovider/auth/kilo_device.go:185` | `req, err := http.NewRequestWithContext(ctx, http.MethodGet, kiloendpoint.Route(base, "profile"), http.NoBody)` |
| K27 | the stream event the twin's fake sends for text | `llmprovider/internal/wire/responses/responses.go:659` | `"response.output_text.delta":            llmprovider.EventTextDelta,` |
| K28 | plant P6's line | `llmprovider/auth/tokenstore_file.go:170` | `HostID:    s.HostID,` |
| K29 | plant P12's line | `llmprovider/auth/oauth_session.go:264` | `TokenURL: s.TokenURL, Resource: s.Resource, Scope: s.Scope, IDToken: s.IDToken, HostID: s.HostID,` |
| K30 | plant P13's line, where the plan revocation reads its endpoints (once in the file) | `llmprovider/auth/oauth_revoke.go:135` | `endpoints, err := planOAuthEndpoints(ctx, client, issuer)` |
| K31 | plant P8's line | `llmprovider/catalog/raw.go:113` | `plan := id == llmprovider.ProviderOpenAI && chatgpt.IsPlanSession(src)` |

**How this PLAN was proven (2026-10-19),** before it was staged:

* `tools/claims.py` passed all 31 rows above, and 0046-MADR's 39.
* Four false claims, planted in scratch copies of this PLAN, were each
  caught: CI already running `tables-check` (K12), and a wrong line for K17,
  K28 and K30.
* Phase 1 step 1's prediction was prototyped at the tree.
  * The import rule gives F7's 13 problems.
  * The live-variable rule finds 24 variables read and 6 named, and
    accepts 15 as record switches. It leaves `LLMPROVIDER_LIVE_GEMINI_429`,
    `…_429_BURST` and `…_429_MODEL` unnamed, and nothing named but unread.
* P13's insertion point is unique: K30's text occurs once in its file.
  The `formRevokeRequest` line after it occurs twice (Grok's and the
  plan's), so it is not used as the plant's target.

## Implementation Steps

### Phase 0: baseline (records only)

1. Record `<base>`, and `tools/gate.py` at `<base>`: 23 of 23.
2. Run the drift measurement of 0046-MADR F7 at `<base>`. It must print 13
   problems, the 13 of D4's table. Count the live variables (A1): 24 read,
   6 named in `AGENTS.md`.
3. Run `tools/claims.py` on 0046-MADR and on this PLAN: every row passes.
4. Record `llmprovider` coverage (`go test -count=1 -cover ./llmprovider/`).

### Phase 1: the docs tables, fixed and checked (D4, A1)

1. **`scripts/check_tables.py`,** written first, so that it can be seen to
   fail on the tree as it is:
   * **Imports.** `go list -f '{{.ImportPath}}|{{join .Imports ","}}'
     ./...`, for the host and for `GOOS=windows`, merged. Only the
     module's own packages and `golang.org/x/` are kept.
   * **A table cell's names** are its backticked tokens. A name resolves to
     the one module package that is it, or ends in `/` and it; a name that
     matches none, or more than one, is a problem. Phrases in a cell:
     * "the above" is every package of the R2 rows above;
     * "internal packages" is every package with `internal/` in its path;
     * "the provider packages" is `llmprovider/providers/<id>`;
     * "the standard library" adds nothing.
   * **R2's pattern rows.** `llmprovider/providers/<id>` and
     `llmprovider/internal/wire/<format>` cover the packages under them. A
     package with a row of its own takes that row, not the pattern.
   * **Fails (exit 1)** with one line per problem, `check-tables: <file>:
     <package or variable>: <problem>`, then `check-tables: <n>
     problem(s)`, when:
     * an architecture row's "Depends on" differs from the imports, with
       what is missing and what is stale named;
     * a package imports something its R2 row does not allow;
     * a package has no row in one of the two tables;
     * a live test reads an `LLMPROVIDER_LIVE_*` variable that `AGENTS.md`
       does not name, except a variable `LLMPROVIDER_LIVE_<NNNN>…` that a
       record numbered `NNNN`, under `docs/decisions/` or `docs/reports/`,
       names as a whole word (`\b…\b`, where `_` is part of a word);
     * `AGENTS.md` names a variable that no live test reads.

     The live tests are `llmprovider/**/live_*_test.go`, read with the
     pattern `LLMPROVIDER_LIVE_[A-Z0-9_]*[A-Z0-9]`.
   * **Exit codes:** 0 clean, 1 a problem, 2 a tool error. It runs from the
     repository root, standard library only.
   * **Seen to fail:** run at the tree before step 2. It must exit 1 with
     exactly D4's 13 problems, plus `AGENTS.md`'s missing
     `LLMPROVIDER_LIVE_GEMINI_429`, `…_429_MODEL` and `…_429_BURST`. The
     15 measurement switches must not appear (A1). The output goes in the
     record.
2. **The tables.** Each edit asserts its old text once.
   * **R2:**
     * the catalogs command's row (K2) gains `internal/transport`;
     * the wire-format row (K3) gains `internal/wire/deferral`;
     * a row is added after it: `` | `llmprovider/internal/wire/deferral` |
       how a request's deferred tools are sent, for the wire formats and
       the providers | `llmprovider`, `internal/wire/messages` | ``;
     * `wizard`'s row (K4) becomes `the above, `internal/redact`,
       `golang.org/x/term``;
     * the citation (K1) gains `; the import rows, 0046 D4`.
   * **The architecture table:**
     * `wizard` (K5) gains `providers/openai`, after `providers`;
     * `kilo` (K6) gains `internal/wire/messages`;
     * the catalogs command (K7) gains `internal/transport`;
     * `` , `internal/transport` `` is removed once from each of the rows
       of `openai`, `claude`, `gemini`, `grok`, `ollama` and
       `internal/wire/responses`.
   * **`AGENTS.md`'s live paragraph** (K8) becomes Appendix C's text.
   * **Seen to pass:** `scripts/check_tables.py` exits 0.
3. **Wired in:**
   * the `Makefile`: `tables-check` in `.PHONY` (K10), and after
     `records-check` (K11):

     ```make
     # 0046-PLAN Phase 1: the docs tables against the code (0046-MADR D4).
     tables-check: ## Checks the import tables and the live variables against the code
     	python3 -B scripts/check_tables.py
     ```

   * CI's step (K12): `tables-check` after `records-check`; the step's name
     gains `the tables` and `0046-MADR`;
   * `tools/gate.py` (K13): `make("tables-check"),` after
     `make("records-check"),`;
   * `AGENTS.md`'s CI list (K9): `tables-check` after `records-check`.
4. **The self-test** (`scripts/test_gates.py`):
   * `"tables-check": "scripts/check_tables.py",` in `GATES` (K14), so the
     clean test runs it;
   * five plants, each requiring exit 1 and the text named:

     | Test | Plant | Must report |
     | :--- | :--- | :--- |
     | `test_tables_check_architecture_stale` | `wizard`'s architecture row gains `` `internal/transport` `` | `docs/architecture.md: wizard` |
     | `test_tables_check_r2_disallowed` | the catalogs command's R2 row loses `` , `internal/transport` `` | `llmprovider/internal/cmd/catalogs` |
     | `test_tables_check_package_without_row` | a package `llmprovider/internal/zzplanted` with one `.go` file | `llmprovider/internal/zzplanted` |
     | `test_tables_check_live_variable_unnamed` | `AGENTS.md` loses `` `LLMPROVIDER_LIVE_CHATGPT_PLAN` `` | `LLMPROVIDER_LIVE_CHATGPT_PLAN` |
     | `test_tables_check_switch_without_record` | a live test file reads `LLMPROVIDER_LIVE_0045_PLANTED` | `LLMPROVIDER_LIVE_0045_PLANTED` |

5. **Docs.** `docs/architecture.md`'s tree names `scripts/check_tables.py`,
   if it lists the scripts there.
6. **Gate:** 24 of 24.

### Phase 2: the person-gated live tests' twins (D3)

1. **`llmprovider/live_checks_test.go`,** untagged, package
   `llmprovider_test`:
   * **Moved here:** `liveOpenAI` and `userText` from
     `live_chatgpt_test.go` (K15, K16), and `liveGrok` from
     `live_grok_test.go` (K17). They are removed from those files, and
     every caller in the tagged files keeps compiling, since the tagged
     build includes this file.
   * **`type transientSkip func(*testing.T, error)`,** and `func
     noSkip(*testing.T, error) {}`. A live test passes
     `llmprovider.SkipIfTransient` (K18) where it already skipped, and
     `noSkip` where it did not, so that no live test's meaning changes
     (R46).
   * **`checkWord(ctx, t, p, word, skip)`:** `GenerateText` with "Reply with
     only the word `<word>`", then `skip(t, err)`; it fails unless `err` is
     nil and the upper-cased output contains `word`. It replaces K19–K21.
   * **`checkKiloSession(ctx, t, session, model, skip, opts...)`:**
     * the token's shape;
     * `auth.KiloProfile(ctx, session.Access, opts...)` (K22);
     * `kilo.New(…, llmprovider.WithModel(model), opts...)`;
     * one generation that is not empty.

     Each is as in the live test today.
   * **The plan's steps,** from `live_chatgpt_plan_test.go` (K23–K25):
     * `checkPlanKind(t, s)`: the checks after `LoginChatGPTPlan` today;
     * `planStoreRoundTrip(ctx, t, signedIn)`: the store half of today's
       `planSignIn`;
     * `planRequests(ctx, t, session, skip, opts...)`;
     * `planRefresh(ctx, t, session, store)`;
     * `planRevoke(ctx, t, session)`: the revocation and the refused
       refresh, from the end of the live test.

     `opts` reach every `liveOpenAI` and `catalog.ListRaw` call.
2. **The live tests** call the checks:
   * `TestLive_ChatGPTBrowserLogin`: `checkWord(…, "ALPHA", noSkip)`;
   * `TestLive_GrokBrowserLogin` and `TestLive_GrokDeviceLogin`: the same;
   * `TestLive_KiloDeviceLogin`: `checkKiloSession(…, llmprovider.LiveModel(…), llmprovider.SkipIfTransient)`;
   * `TestLive_ChatGPTPlan`: the sign-in, `checkPlanKind`,
     `planStoreRoundTrip`, `planRequests(…, llmprovider.SkipIfTransient)`,
     `planRefresh`, `planRevoke`, in that order, with the cleanup that
     revokes unless `planRevoke` ran.
3. **`llmprovider/live_twins_test.go`,** untagged.
   * **`twinFake`:** an `httptest.Server` that answers by path suffix:

     | Suffix | Answer |
     | :--- | :--- |
     | `POST /responses` | If the request asks for an event stream: `response.created`; a `response.output_text.delta` with the word (K27), when `deltas`; the message item with the word; a `get_weather` `{"city":"Paris"}` function-call item when the request has tools and `tool` is set; `response.completed`. Otherwise the same as one Responses JSON reply. The word is the one the prompt asks for ("Reply with only the word X"), or `OMEGA` when `echo` is false. |
     | `POST /chat/completions` | a Chat Completions reply whose content is the word, or `""` when `chatWord` is false |
     | `GET /models` | `{"models": […]}`, one entry `{"slug": m, "visibility": "list"}` for each of `models` |
     | `GET /profile` | `profileStatus`, with `{"organizations": [], "hasPersonalAccount": true}` (K26) |
     | `GET /.well-known/openid-configuration` | the issuer and its authorize, token, revocation and JWKS endpoints, all on the fake |
     | `POST /token` | after a revocation, when `revokeKills`: 400 `{"error": "invalid_grant"}`; otherwise new tokens (`plan-access-<n>`, `plan-refresh-<n>`, 3600 s) when `rotate`, the same tokens when not |
     | `POST /revoke` | marks the session revoked; 200 |

   * **The constructor's defaults,** each a plant's target:

     ```go
     &twinFake{echo: true, chatWord: true, deltas: true, tool: true, rotate: true, revokeKills: true,
         profileStatus: http.StatusOK, models: []string{"gpt-6.1-sol", "gpt-6-luna"}}
     ```

   * **The twins,** each calling the check functions with the fake's base
     URL and client:

     | Twin | Session | Checks |
     | :--- | :--- | :--- |
     | `TestLiveTwin_ChatGPT_0046` | a ChatGPT session (OpenAI's issuer, Codex's client) | `checkWord(…, liveOpenAI(t, s, "gpt-6-astra", base, client), "ALPHA", noSkip)` |
     | `TestLiveTwin_Grok_0046` | an xAI session | `checkWord(…, liveGrok(t, s, "grok-4.6", base, client), "ALPHA", noSkip)` |
     | `TestLiveTwin_Kilo_0046` | `{Provider: kilo, Access: "twin-kilo"}` | `checkKiloSession(…, "twin-free-model", noSkip, base, client)` |
     | `TestLiveTwin_ChatGPTPlan_0046` | a plan session whose `Issuer` is the fake, with `TokenURL` its `/token`, `ClientID` `oaia_twin`, `Resource` `openai.PlatformBaseURL`, the plan scope, a host id and an ID token, and the fake's client | `checkPlanKind`, `planStoreRoundTrip`, `planRequests(…, noSkip, base, client)`, `planRefresh`, `planRevoke` |

     One twin covers both Grok live tests, which share `checkWord` and the
     provider.
4. **The plants** (`tools/plant.py`, on a scratch clone; `run`
   `^TestLiveTwin_`; package `./llmprovider/`). Each must be caught by the
   twins named, by an assertion:

   | # | Plant | File | Caught by |
   | :--- | :--- | :--- | :--- |
   | P1 | `echo: true` → `echo: false` | twins | ChatGPT, Grok, ChatGPTPlan |
   | P2 | `chatWord: true` → `chatWord: false` | twins | Kilo |
   | P3 | the Kilo session gains `Refresh: "r"` | twins | Kilo |
   | P4 | `profileStatus: http.StatusOK` → `http.StatusInternalServerError` | twins | Kilo |
   | P5 | the plan session loses `Resource` | twins | ChatGPTPlan |
   | P6 | `HostID:    s.HostID,` removed (K28) | `llmprovider/auth/tokenstore_file.go` | ChatGPTPlan |
   | P7 | `models: []string{"gpt-6.1-sol", "gpt-6-luna"}` → `models: nil` | twins | ChatGPTPlan |
   | P8 | the plan line (K31) → `plan := false` | `llmprovider/catalog/raw.go` | ChatGPTPlan |
   | P9 | `deltas: true` → `deltas: false` | twins | ChatGPTPlan |
   | P10 | `tool: true` → `tool: false` | twins | ChatGPTPlan |
   | P11 | `rotate: true` → `rotate: false` | twins | ChatGPTPlan |
   | P12 | `Scope: s.Scope, ` removed from `persistable` (K29) | `llmprovider/auth/oauth_session.go` | ChatGPTPlan |
   | P13 | after K30's line, insert `endpoints.Revocation = endpoints.Token`: the form goes to `/token`, which answers as a refresh, so nothing is revoked | `llmprovider/auth/oauth_revoke.go` | ChatGPTPlan |
   | P14 | `revokeKills: true` → `revokeKills: false` | twins | ChatGPTPlan |

5. **`AGENTS.md`** carries the rule, in Appendix C's text (Phase 1).
6. **The live runs, with the owner signing in.** One run of each suite, on
   the restructured tests:
   * `LLMPROVIDER_LIVE_BROWSER_LOGIN=1 go test -tags live_gateways -count=1
     -v -run 'TestLive_(ChatGPT|Grok)BrowserLogin$' ./llmprovider/` (two
     sign-ins);
   * `LLMPROVIDER_LIVE_DEVICE_LOGIN=1 … -run
     'TestLive_(Grok|Kilo)DeviceLogin$'` (two codes);
   * `LLMPROVIDER_LIVE_CHATGPT_PLAN=1 … -run '^TestLive_ChatGPTPlan$'` (one
     sign-in).

   A run that times out with no sign-in is run again, and both are
   recorded.
7. **Gate:** 24 of 24. `llmprovider`'s coverage is not below Phase 0's.

### Phase 3: the consumers' issues (D2)

1. Run `tools/claims.py` on 0046-MADR. If a consumer's `HEAD` has moved,
   check G1–G10 or P1–P6 at the new `HEAD`. A changed fact is a stop
   condition: the draft is corrected, and the change recorded, before
   anything is opened.
2. Write each draft in Appendix B to a file. Pass `tools/denyscan.py` on
   it, and show it to the owner.
3. **On the owner's word in that turn,** open each:
   * `gh issue create --repo maccavelli/gobble-cli --title "…" --body-file
     …`;
   * `gh issue create --repo maccavelli/prepare-commit-msg --title "…"
     --body-file …`.

   Record each URL.

### Phase 4: the issues (D1)

1. Write each comment in Appendix A to a file. Pass `tools/denyscan.py` on
   it, and show it to the owner.
2. **On the owner's word in that turn,** for issue 2 then issue 3:
   `gh issue comment <n> --body-file …`, then `gh issue close <n> --reason
   completed`. Record each comment's URL.
3. This PLAN becomes `complete`.

## Verification

| # | Criterion | Check | Phase |
| :--- | :--- | :--- | :--- |
| V1 | The tables match the code, and the live variables are named or are a record's | `scripts/check_tables.py` exits 0 | 1 |
| V2 | The check fails on each kind of drift | its five self-tests, and its exit 1 at the tree before the fix | 1 |
| V3 | CI, `make` and the gate run it | the CI run on Phase 1's commit; `tools/gate.py` 24 of 24 | 1 |
| V4 | Each person-gated live test runs its checks through a shared function | the live files call `checkWord`, `checkKiloSession` and the plan's steps | 2 |
| V5 | Every check fails on a planted break, offline | P1–P14, each caught by its twin | 2 |
| V6 | No live test asserts less than before (R46) | a reading of the diff of the two live files; the skip passed matches today's | 2 |
| V7 | Each suite passes live on the new code | the three runs' logs | 2 |
| V8 | No production file, export or golden changed | `git diff --stat`; `tools/apidiff.py v1.15.0` prints nothing | 1, 2 |
| V9 | Each consumer has its issue, with true facts | `tools/claims.py`; the URLs | 3 |
| V10 | Issues 2 and 3 carry Appendix A's comments, and are closed | the URLs; `gh issue view` | 4 |

## Rollout and Rollback

* **Phases 1 and 2** are commits the owner pushes. CI on each must pass.
  No tag: nothing exported changes.
* **Phases 3 and 4** publish text. A comment can be edited or deleted by
  the owner. A consumer issue is closed by its repository.
* **Rollback:** revert Phase 1 or 2's commit. The check, or the twins, go
  with it, and the drift or the gap returns.

## Appendix A: the issue comments (D1)

### Issue 2

> ## Shipped in v1.15.0
>
> OpenAI's Sign in with ChatGPT for agents is an OpenAI auth method as of
> `v1.15.0`. The design is
> `docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md`, and its PLAN
> is complete.
>
> * `auth.LoginChatGPTPlan(ctx, auth.ChatGPTPlanOptions{AgentName: …})`
>   signs a user in under your app's own name. The first sign-in registers
>   a client through `dynamic_agent_client`. Passing the saved session as
>   `Previous` signs in to that client again.
> * `auth.OAuthSession` gains `Resource`, `Scope`, `IDToken`, `HostID` and
>   `PlanUsage()`. A plan session refreshes and revokes with forms and
>   `resource`, at endpoints read from the issuer's discovery document and
>   required to be on its origin.
> * `openai.New` sends a plan session's requests to `api.openai.com/v1`:
>   streamed, with `store: false` and no output limit. It has no
>   continuation, and deferred tools are appended. `ListModels` reads the
>   account's `models` array and never probes.
> * `subscription_sharing_usage_limit_exceeded` is `ErrQuotaExhausted`.
>   `openai.ChatGPTUsageURL` is the "Manage usage" page.
> * The wizard offers "Use your ChatGPT plan with this app"
>   (`llmprovider.AuthChatGPTPlan`) when `wizard.Options.AgentName` and a
>   `TokenStore` are set.
>
> ## Corrections to my earlier comment
>
> * It said the session gains `Scopes`. It gained `Scope string`, the
>   scopes space-separated as the token response sends them. A slice would
>   have made `OAuthSession` incomparable, an incompatible change (0043-MADR
>   A1).
> * It said the 0043 pair was proposed. The MADR is accepted, and the PLAN
>   complete.
>
> ## Added after that comment (0043-MADR A4)
>
> * `subscription_sharing_user_not_eligible` and `…_route_not_supported`
>   are `ErrNotPermitted`, and `…_unsupported_capability` is
>   `ErrInvalidRequest`, in a reply or inside a stream. None is retried.
> * A system message in a plan session's input is refused before the
>   network (`ErrUnsupported`). Put it in `Request.Instructions`.
>
> ## What stays yours
>
> OpenAI's terms ask things of your app that the SDK cannot do for you, and
> no option claims to (0043-MADR D8):
>
> * requests only from the user's own runtime, for that user's activity;
> * the user's express consent before any background use;
> * the plan used only for the app the user connected;
> * no pooling, sharing, transfer or resale of plan usage or tokens;
> * no charge.
>
> Closing as shipped.

### Issue 3

> ## Shipped in v1.15.0
>
> `llmprovider.WithRetryNotify(p, policy, notify)` retries as `WithRetry`
> does, and calls `notify(ctx, llmprovider.RetryNotice{…})` before each
> wait. The design is `docs/decisions/0044-MADR-observe-retries.md`.
>
> * `RetryNotice` has `Provider`, `Attempt` (the try that failed, the first
>   being 1), `MaxAttempts`, `Wait` (jitter included), `ServerDirected`
>   (the wait came from `Retry-After` or a rate-limit reset header) and
>   `Err`.
> * notify runs on the goroutine of `Generate`, with its `ctx`, only when a
>   wait is about to begin. It is never called for a success, a failure
>   that is not retried, the last attempt, a wait longer than `MaxDelay` or
>   than the deadline allows, or a second failure after a reply. The wait
>   begins when notify returns, so notify must not block.
> * Through `llmprovider.Stream`, the notices come before the first event.
> * `WithRetry` is `WithRetryNotify` with no notify, and behaves as before.
>
> ## Why a function, not an `OnRetry` field
>
> A func field makes `RetryPolicy` incomparable. `api-check` reports that
> as an incompatible change against `v1.14.0` (0044-MADR E1), which would
> need a `v2`. A second constructor keeps `RetryPolicy` a plain value.
>
> Closing as shipped.

## Appendix B: the consumers' issues (D2)

### gobble-cli

**Title:** Adopt go-llmprovider-sdk v1.15.0: retry notices (SDK issue 3)
and the ChatGPT plan sign-in (F4-3d)

> go-llmprovider-sdk `v1.15.0` ships both things gobble's 0005-PLAN waits
> on. gobble is on `v1.13.0` (`go.mod:16`). `v1.14.0` came between, with
> its own notes.
>
> ## Retry notices (SDK issue 3)
>
> `llmprovider.WithRetryNotify(p, policy, notify)` is `WithRetry`, plus a
> `RetryNotice` before each wait: `Provider`, `Attempt`, `MaxAttempts`,
> `Wait`, `ServerDirected` and `Err`. It runs on the turn's goroutine, with
> its `ctx`, and must not block.
>
> * `llm/provider/provider.go:197` wraps every provider with
>   `llmprovider.WithRetry(p, policyOf(r))`. `WithRetryNotify` there,
>   with a notify that emits gobble's `Retry` event with the cause
>   `provider`, is the slot 0005-PLAN `:5698` names.
> * The terminal view words any cause already
>   (`internal/cli/view.go:185`), and `_gobble/retry`
>   (`acpserver/extensions.go:54`) carries the event to an ACP client.
> * The default policy (`llm/llm.go:42`: 3 retries, 2 s, at most 60 s)
>   decides the waits the user would see.
>
> ## The ChatGPT plan sign-in (F4-3d)
>
> `auth.LoginChatGPTPlan` signs in under the app's own name. Its session
> carries `Resource`, `Scope`, `IDToken` and `HostID`, and `PlanUsage()`
> tells it apart. gobble needs code of its own first:
>
> * its stored login (`llm/provider/login.go:18-21`) has no field for
>   those four, so a saved plan session would lose them;
> * `HasLogin` is Grok's only (`llm/provider/login.go:71`);
> * `internal/cli/auth.go:198` refuses an OpenAI login, pending this
>   release.
>
> What stays the app's under OpenAI's terms is in the SDK's `auth` package
> doc and README ("Signing in with a ChatGPT plan").

### prepare-commit-msg

**Title:** Adopt go-llmprovider-sdk v1.15.0: the ChatGPT plan sign-in in
the wizard

> go-llmprovider-sdk `v1.15.0` adds OpenAI's Sign in with ChatGPT for
> agents, and retry notices. prepare-commit-msg is on `v1.12.0`
> (`go.mod:6`); `v1.13.0` and `v1.14.0` come between, with their own notes.
>
> ## The plan sign-in: one field
>
> The wizard offers "Use your ChatGPT plan with this app" when
> `wizard.Options.AgentName` and a `TokenStore` are set.
> `internal/ui/setup.go:265` already builds `wizard.Options` with
> `TokenStore: store` (`:275`), so `AgentName: <the app's own name>`
> enables it. The session's client id is passed back as `Existing`
> (`:245`), so the wizard offers the plan method first on the next run.
> The result is `CredOAuth`, as today. What stays the app's under OpenAI's
> terms is in the SDK's README ("Signing in with a ChatGPT plan").
>
> ## Retry notices, if wanted
>
> `main.go:392` builds the policy for `WithRetry`, and the README says a
> retry is not shown (`README.md:461-463`). `llmprovider.WithRetryNotify`
> would let the hook print one line to stderr before each wait.

## Appendix C: `AGENTS.md`'s live paragraph (Phase 1 step 2)

> `./llmprovider/...` includes the live tests in `auth` and `catalog`. A
> provider's live tests run when its key variable is set
> (`ProviderEnvVars()`), and skip without it. These suites are switched on
> by their own variable: `LLMPROVIDER_LIVE_CHATGPT`,
> `LLMPROVIDER_LIVE_BROWSER_LOGIN` (the ChatGPT and Grok browser logins),
> `LLMPROVIDER_LIVE_DEVICE_LOGIN` (the Grok and Kilo device logins),
> `LLMPROVIDER_LIVE_CHATGPT_PLAN` (the ChatGPT plan sign-in),
> `LLMPROVIDER_LIVE_GROK_CLI`, `LLMPROVIDER_LIVE_TOGETHER` (with
> `TOGETHER_API_KEY`), and `LLMPROVIDER_LIVE_GEMINI_429` (with
> `LLMPROVIDER_LIVE_GEMINI_429_MODEL` and
> `LLMPROVIDER_LIVE_GEMINI_429_BURST`; it exhausts a Gemini rate limit on
> purpose). A record's own measurement test is switched on by
> `LLMPROVIDER_LIVE_<NNNN>`, with suffixes for its options, as that record
> says. The login tests need a person to sign in. Each keeps its checks
> after the sign-in in a function in `llmprovider/live_checks_test.go`,
> which an offline twin in `llmprovider/live_twins_test.go` runs against a
> fake. A planted break of one of those checks runs on its twin (0046-MADR
> D3). CI only vets the live-tagged files (`go vet -tags live_gateways
> ./...`); it never runs them. `scripts/check_tables.py` checks that every
> variable a live test reads is named here, or is a record's.
