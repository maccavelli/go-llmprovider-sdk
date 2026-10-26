---
status: complete
date: 2026-10-26
associated-madr: "0065-MADR-tests-reach-no-outside-host.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Keeping the Tests Off the Network

Associated MADR: [0065-MADR-tests-reach-no-outside-host.md](0065-MADR-tests-reach-no-outside-host.md)

## Goal

The MADR's D1 to D3. When this PLAN is complete, no test run of any package
sends a request off the machine, and in the six packages the MADR names a
test that would fails its package's run, naming the host.

## Scope

### In scope

* `llmprovider/catalog/model_metadata_test.go` (`listT`),
  `llmprovider/catalog/session_client_0026_test.go`
  (`TestList_GivesASessionItsClient`);
* `llmprovider/providers/{claude,gemini,grok,openai}/helpers_test.go`
  (`build`);
* `internal/localonly/` (the package and its test);
* a `localonly_test.go` with `TestMain` in `llmprovider/catalog`,
  `llmprovider/providers/{claude,gemini,grok,openai,chatcompat}`;
* `AGENTS.md`, `docs/architecture.md`, `docs/README.md` (this pair's rows),
  `scripts/coverage-floors.txt` if `coverage-check` asks for the new
  package's floor.

### Out of scope

* The SDK's behaviour and API: no production file changes.
* The other packages' tests: measured at 0; they gain no guard here.

## Implementation Steps

### Phase 1: the call sites (D1)

1. `listT`: the document off unless `metadataOn` and the options name a URL.
2. `TestList_GivesASessionItsClient`: `WithoutModelMetadata()`.
3. The four `build`s: `WithoutModelMetadata()` when the options name no URL.
4. Every package through the logging proxy: 0. Every test passes.

### Phase 2: the guard (D2)

1. `internal/localonly`: `Main`, and the proxy it runs; its test: a request
   off the machine through `http.ProxyFromEnvironment` is refused and
   recorded, a loopback request is not, and the report names the host once
   with its count.
2. The six `TestMain`s, each in a file with `//go:build !live_gateways`.
3. Plants, on a scratch clone (`tools/plant.py`): D1's option removed from
   `claude`'s `build` (the package fails with the guard's message, naming
   `models.opencode.ai:443`); `listT`'s URL test removed (`catalog` fails
   alike); the guard's exit code dropped (`localonly`'s own test fails).

### Phase 3: docs (D3), and the close

1. `AGENTS.md`, `docs/architecture.md`; `docs/README.md` by
   `tools/records.py`.
2. The gate; this record's execution entries; the MADR and this PLAN
   `accepted` and `complete`.

## Verification

| # | Criterion | Check |
| :--- | :--- | :--- |
| V1 | No package sends a request off the machine | every package through the logging proxy |
| V2 | A leak fails its package's run, naming the host | Phase 2's plants |
| V3 | The tests' results are unchanged | `go test ./...`; the gate |
| V4 | No production file changes | `git diff --stat` lists test files, `internal/localonly`, and docs only |

## Rollout and Rollback

* **Rollout.** One commit per phase, staged by the agent and committed by
  the owner. No release of its own.
* **Rollback.** A phase reverts on its own; Phase 2 first.

## Execution record

### Approval (2026-10-25)

* The owner chose option A and asked for it to be executed.

### Phase 1: the call sites (2026-10-25)

* `listT` lists with the document off unless `metadataOn` and the options
  name a URL (`namesDocument`, by `llmprovider.ResolveOptions`);
  `TestList_GivesASessionItsClient` passes `WithoutModelMetadata()`; the
  four `build`s add it when the options name no URL. No production file
  changed.
* `go test -count=1 ./...`: every package passed.
* Every package, each alone, through the logging proxy: 0 requests; the
  six the MADR names, shown one by one: `catalog` 0, `claude` 0, `gemini`
  0, `grok` 0, `openai` 0, `chatcompat` 0 (from 24, 18, 18, 19, 20 and 0).
* `make pre-add-check` on the six files: clean.
* **Staged:** this pair, `docs/README.md`'s rows,
  `llmprovider/catalog/{model_metadata,session_client_0026}_test.go`,
  `llmprovider/providers/{claude,gemini,grok,openai}/helpers_test.go`.

### Phase 2: the guard (2026-10-25)

* Phase 1 is committed as `539affa`.
* `internal/localonly`: `Main`, `Start`, `Guard.Stop`, `Report`,
  `ExitCode`. Its tests: `TestGuard` runs `TestGuardChild` in a process of
  its own, since `net/http` reads the proxy variables once a process, at its
  first proxied request (a second guard in one process was not seen: the
  first run of `-count=5 -shuffle=on` failed `TestGuard` with `proxyconnect
  tcp: dial tcp 127.0.0.1:…: connect: connection refused`, the first
  guard's closed port); there, a loopback request is served and not
  recorded, two plain requests and one TLS request off the machine are
  refused and recorded, and `Stop` gives `off-machine.invalid (2)`,
  `off-machine.invalid:443 (1)`. `TestMain_Runs` drives `Main` with a
  stand-in runner; `TestExitCode_Report` the code and the message. Coverage
  85.1 %; `-count=5 -shuffle=on` and `-race -count=2` pass.
* The six `localonly_test.go`, each `//go:build !live_gateways`, each
  `TestMain` calling `localonly.Main`. `go vet ./...` and `go vet -tags
  live_gateways ./...` clean; `go test -count=1 ./...`: every package
  passed, and the guard reported nothing.
* **Plants** (`tools/plant.py`, on a scratch clone; all 3 caught): `claude`'s
  `build` leaving the document on, and `listT`'s document test removed,
  each fail their package; reproduced with the full output, each test
  passes and the guard fails the run: `--- PASS: TestListModels` then
  `localonly: the tests sent requests off the machine, to
  models.opencode.ai:443 (1); …`, `FAIL …/providers/claude`; the same for
  `TestList_Cached_0041` and `catalog`. The guard's exit code dropped fails
  `TestMain_Runs` and `TestExitCode_Report`.
* **Deviation 1 (2026-10-25): the package's rows in Phase 2.** Found:
  `make tables-check` failed with `internal/localonly: no row` in
  `docs/architecture.md` and in `docs/guides/api-standards.md`: a package's
  rows come with it (R43). The architecture rows move here from Phase 3, and
  `docs/guides/api-standards.md` (R2's table), which the Scope did not name,
  joins it. The one resolution that leaves the tree correct, so it was taken
  and recorded rather than offered.
* **Deviation 2 (2026-10-25): `Main`'s parameter.** Found: with `Main(m
  *testing.M)`, the package measured 73.3 %, under the 80 % floor, as no
  test can make a `*testing.M`. `Main` takes `interface{ Run() int }`,
  which `*testing.M` is; the MADR's D2 says so. No caller changes.
* `golangci-lint`'s errcheck: `Serve`'s and `Shutdown`'s errors are
  reported to stderr, `Serve`'s normal close aside.
* **Checks:** `make pre-add-check` on the 8 Go files: clean; `make
  tables-check`, `dep-check`, `parity-check`: 0 problems; `tools/gate.py`: 28 of 28 passed.
* **Staged:** this record and its MADR,
  `internal/localonly/localonly{,_test}.go`, the six
  `localonly_test.go`, `docs/architecture.md`,
  `docs/guides/api-standards.md`.

### Phase 3: docs, and the close (2026-10-26)

* Phase 2 is committed as `b9fc408`.
* `AGENTS.md`: the rule (a test serves its own replies and its own
  document, or turns the document off) and the guard, after the test-cache
  paragraph. `docs/architecture.md`'s rows landed in Phase 2 (deviation 1);
  `docs/README.md`'s rows in Phase 1.
* **Verification:** V1, every package through the logging proxy, 0
  (Phase 1, unchanged since: Phases 2 and 3 add no request); V2, Phase 2's
  plants; V3, `go test ./...` and the gate, every package passing; V4, no
  production file changed: the three commits change test files,
  `internal/localonly` (imported only by tests) and docs.
* **Checks:** `make tables-check`, `records-check`, `docs-check`;
  `recordlint --changed`, `denyscan --changed`, `linkcheck`: each 0 problems; `tools/gate.py`: 28 of 28 passed.
* **Staged:** `AGENTS.md`, this record (`complete`), and `docs/README.md`'s
  row.
