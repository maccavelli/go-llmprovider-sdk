---
status: in-progress
date: 2026-10-25
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
