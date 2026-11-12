---
status: in-progress
date: 2026-11-12
associated-madr: "0078-MADR-cached-metadata-refusal-test-uses-a-refused-option.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Refused Option and the Wait in CachedMetadataWith's Refusal Test

Associated MADR: [0078-MADR-cached-metadata-refusal-test-uses-a-refused-option.md](0078-MADR-cached-metadata-refusal-test-uses-a-refused-option.md)

## Goal

The MADR's D1 and D2:

* `TestCachedMetadataWith_Unavailable`'s second call passes an option
  `CachedMetadataWith` refuses, and the test waits for every fetch before it
  counts the requests, so a call that fetches fails it on every run;
* the change is seen red before and green after, and its check is seen to
  fail on three plants, each on a scratch clone, never in the tree;
* `ci.yml` on `main` then passes in full.

The MADR is `proposed`, and so is this PLAN: nothing below Phase 0 starts
before the owner accepts the one and approves the other.

## Scope

### In scope

* `llmprovider/catalog/model_metadata_0028_test.go`, lines 63 to 67 (P1 to
  P3): the edit of MADR D1.
* This pair and its two rows in `docs/README.md`.

### Out of scope

* **Any non-test file.** No exported name, golden or behaviour changes;
  `tools/apidiff.py` is not needed.
* **`For`'s semantics** (MADR option C).
* **The background work of MADR D3** (O1, O2), observed and not decided.
* **`CHANGELOG.md`.** A test-only change has no entry, as `e5eea9d` had
  none.
* **[0073-PLAN-publish-an-auth-contract-module.md](0073-PLAN-publish-an-auth-contract-module.md).**
  Its Phase 4 is closed by the green run of Phase 4 below, recorded there
  when it happens, not by this PLAN.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts after the owner has committed the last.
* **The new check is seen to fail first** (Phase 2), on scratch clones,
  never in the tree.
* **Commands.** Each exit status is captured before any filter; long output
  goes to a scratch file. This Mac is shared: one test run at a time.
* **Stage, then the owner commits** with `git commit --no-edit`. The push
  happens on the owner's word, after the disclosure guard has run over the
  outgoing commits.
* **Stop conditions:**
  * a fact of the MADR no longer holds, P1 to P3 among them;
  * the edited test fails, or passes a plant;
  * any other test of the package fails;
  * a CI job fails.

  On any of these, stop, record a dated deviation here (with an amendment
  in the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. This pair, and its two rows in `docs/README.md` (`python3 -B
   tools/records.py add`), staged as records only. They share the change
   that sets 0074's and 0075's statuses (the owner's decision of
   2026-11-12).

Acceptance: `make records-check` and `make links-check` pass; no file
outside `docs/` changed.

### Phase 1: the edit and its regression check (MADR D1)

1. Assert P1 to P3 hold, then replace, in
   `llmprovider/catalog/model_metadata_0028_test.go`, exactly once:

   ```go
   	if _, ok := CachedMetadataWith(context.Background(), llmprovider.ProviderOpencodeZen, opt,
   		llmprovider.For(llmprovider.ProviderKilo, WithKiloOrganization("org"))); ok {
   		t.Error("an option for another provider: ok = true")
   	}
   	if n := hits.Load(); n != 0 {
   ```

   with:

   ```go
   	if _, ok := CachedMetadataWith(context.Background(), llmprovider.ProviderOpencodeZen, opt,
   		WithKiloOrganization("org")); ok {
   		t.Error("an option scoped to another provider: ok = true")
   	}
   	// A call that started a fetch is counted, however the scheduler runs it
   	// (0078-MADR).
   	waitMetadataFetches()
   	if n := hits.Load(); n != 0 {
   ```

   Assert the new text is in the file once, and `git diff --stat` shows
   that file alone, 5 insertions and 2 deletions.
2. `make pre-add-check FILES="llmprovider/catalog/model_metadata_0028_test.go"`.
3. `go test -race -count=300 -run '^TestCachedMetadataWith_Unavailable$'
   ./llmprovider/catalog/`: exit 0.
4. `go test -race -count=1 -shuffle=1794513551380402489
   ./llmprovider/catalog/`, CI's seed, then `go test -race -count=1
   -shuffle=on -v ./llmprovider/catalog/`, its seed recorded: each exit 0.

Acceptance: steps 2 to 4 exit 0; the diff is step 1's.

### Phase 2: the planted-break proof (MADR D2)

1. **Red before.** On a scratch clone of `e5eea9d`, insert only
   `waitMetadataFetches()` before line 67 and run step 3's command once: it
   fails with `requests = 1; want none yet`, exit 1 (MADR F5).
2. **Plants.** `python3 -B tools/plant.py SPEC.json` on the tree with Phase
   1's edit, with a scratch spec:

   ```json
   {"packages": ["./llmprovider/catalog/"], "run": "^TestCachedMetadataWith_Unavailable$",
    "plants": [
     {"name": "the second call given For(kilo, ...) again, which opencode-zen skips",
      "file": "llmprovider/catalog/model_metadata_0028_test.go",
      "old": "\t\tWithKiloOrganization(\"org\")); ok {\n",
      "new": "\t\tllmprovider.For(llmprovider.ProviderKilo, WithKiloOrganization(\"org\"))); ok {\n"},
     {"name": "the first call with metadata left on",
      "file": "llmprovider/catalog/model_metadata_0028_test.go",
      "old": "\t\tllmprovider.WithoutModelMetadata()); ok {\n",
      "new": "\t\tllmprovider.WithModelMetadataURL(srv.URL)); ok {\n"},
     {"name": "cachedMetadata ignores WithoutModelMetadata",
      "file": "llmprovider/catalog/model_metadata.go",
      "old": "func cachedMetadata(ctx context.Context, cfg config) (Metadata, bool) {\n\tif cfg.metadataOff {\n",
      "new": "func cachedMetadata(ctx context.Context, cfg config) (Metadata, bool) {\n\tif false {\n"}
    ]}
   ```

   Each plant must fail with `requests = 1; want none yet`; the tool must
   print "all 3 plant(s) caught", exit 0. (Run on 2026-11-12 against a
   scratch clone holding the edit: so it printed, each at line 71, MADR
   F6.)

Acceptance: step 1 exits 1 with CI's text; step 2 exits 0. The outputs are
recorded here.

### Phase 3: the gate

1. `python3 -B tools/gate.py` (the change holds a Go file, so `--markdown`
   refuses it).
2. `python3 -B tools/claims.py` on this pair; `make records-check`;
   `python3 -B tools/recordlint.py --changed`; `python3 -B
   tools/denyscan.py --changed`; `python3 -B tools/linkcheck.py`.
3. This PLAN's execution record, then stage the test file and the records.

Acceptance: every step exits 0.

### Phase 4: the push and main's CI

1. After the owner's commit, run the disclosure guard over the outgoing
   commits (AGENTS.md "Identifiers"); exit 0.
2. The push of `main`, on the owner's word.
3. `ci.yml` on `main` at that commit must pass in full: all three
   `validate` jobs, the Ubuntu race step among them; `release` skipped.
   A failure is a stop condition. A rerun of a failed job is not a pass.
4. Record the run's id and result here, and set this PLAN `complete`.
   That run is the green `main` run that closes
   [0073-PLAN-publish-an-auth-contract-module.md](0073-PLAN-publish-an-auth-contract-module.md)
   Phase 4 (its "Phase 4, the tag (2026-11-12)" entry: step 6, the proxy
   check, and its record of Phase 4 complete), and that
   [0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md)
   Phase 1 waits for. Both are recorded in those PLANs by their own
   execution, not here.

Acceptance: the run passes; its id is recorded.

## Verification

* MADR Confirmation: D2's red and green runs, the plants caught, the gate,
  and `ci.yml` on `main` green on all three runners.
* `git diff` of Phase 1 touches one test file.

## Rollout and Rollback

* One commit on `main` for Phase 1 to 3, after the records' commit. No tag
  and no release.
* Rollback is `git revert` of that commit, which brings the flake back; no
  consumer sees either.

## Code claims

<!-- claims: self=e5eea9d -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | The call Phase 1 edits | `llmprovider/catalog/model_metadata_0028_test.go:63` | `if _, ok := CachedMetadataWith(context.Background(), llmprovider.ProviderOpencodeZen, opt,` |
| P2 | its option | `llmprovider/catalog/model_metadata_0028_test.go:64` | `llmprovider.For(llmprovider.ProviderKilo, WithKiloOrganization("org"))); ok {` |
| P3 | the check the wait goes before | `llmprovider/catalog/model_metadata_0028_test.go:67` | `if n := hits.Load(); n != 0 {` |

## Execution record

### Phase 0 (2026-11-12)

The owner decided on 2026-11-12, in the coordinator session, that this pair
comes first ("Root-cause pair first"). The records agent wrote it, with the
status changes of 0074 and 0075 the owner decided the same day.

* Step 1: `python3 -B tools/records.py add` on this pair printed "records:
  written; records-check: 0 problem(s)". Each also exited 0: `make
  records-check`, "records-check: 164 records, 0 problem(s)";
  `tools/claims.py` on the MADR, "20 of 20 pass", on this PLAN, "3 of 3
  pass", on 0074-MADR "55 of 55", 0074-PLAN "6 of 6", 0075-MADR "51 of 51"
  and 0075-PLAN "8 of 8"; `tools/recordlint.py --changed`, "recordlint: 6
  record(s); no rule rose"; `tools/denyscan.py --changed`, "deny-scan: 0
  hit(s) in 7 file(s)"; `make links-check`, "links-check: 0 problem(s),
  1263 relative link(s) in 178 file(s)"; `make disclosure-check`,
  "disclosure-check: 0 hit(s) in 1585 file(s)". This entry was added after
  those runs, so they were run again before staging.
* Phase 1's block and Phase 2's spec were checked against the runs of MADR
  F6 by script: the old block occurs once in the tree, replacing it gives
  the file those runs tested, and the spec is the one `tools/plant.py` ran.
* The records were staged for the owner to commit; no file outside `docs/`
  changed.

### Phases 0 to 3 (2026-11-12)

The owner accepted the MADR and approved this PLAN on 2026-11-12, in the
coordinator session ("Approve and push"). `python3 -B tools/records.py
status` set the MADR `accepted` and this PLAN `in-progress` with `--date
2026-11-12`, each with its row in `docs/README.md`; each printed "records:
written; records-check: 0 problem(s)". The MADR's Decision Outcome records
the acceptance. No deviation.

**Phase 1.** The tree held the "before" text of step 1 once (P1 to P3).
The edit was applied by script, which asserts the old block occurs once and
the new block occurs once afterwards.

* `git diff --numstat` of the test file: `5 2`; no other non-record file
  changed. `gofmt -l llmprovider/catalog/`: no output, exit 0.
* Step 2, `make pre-add-check FILES=...`: exit 0, "go-precheck: 1 file(s)
  clean (gofmt, golangci-lint, go vet, go test, govulncheck)".
* Step 3, `go test -race -count=300 -run
  '^TestCachedMetadataWith_Unavailable$' ./llmprovider/catalog/`: exit 0,
  "ok ... 1.774s".
* Step 4, `go test -race -count=1 -shuffle=1794513551380402489
  ./llmprovider/catalog/`: exit 0, "ok ... 30.547s"; `-shuffle=on -v`: exit
  0, seed 1794515686297521000, "PASS", "ok ... 31.066s".

**Phase 2.**

* Step 1, red before: a scratch clone at `e5eea9d` with only
  `waitMetadataFetches()` inserted before line 67; step 3's command, exit 1,
  every run failing as CI did: `--- FAIL: TestCachedMetadataWith_Unavailable
  (0.00s)` and `model_metadata_0028_test.go:69: requests = 1; want none
  yet` (line 69 because of the inserted line).
* Step 2, `python3 -B tools/plant.py` with this PLAN's spec, exit 0. Each of
  the three plants was "caught (exit 1)" with `model_metadata_0028_test.go:71:
  requests = 1; want none yet` and `FAIL ... llmprovider/catalog`: the For
  option restored; the first call's metadata left on; `cachedMetadata`
  ignoring `WithoutModelMetadata`. Final line: "plant: all 3 plant(s)
  caught". The tool ran on its own scratch clone; the tree was not touched.

**Phase 3.**

* Step 1, `python3 -B tools/gate.py` (it waited for load to fall below
  10): exit 0, "gate: 30 of 30 passed". Among them: pre-add-check, the six
  vet targets, api-check, contract-check, generate-check, records-check,
  tables-check, fixtures-check, changelog-check, docs-check, gate-selftest,
  markdownlint, links-check, disclosure-check, deny-scan and tools-test, each
  `rc=0`.
* Step 2, each exit 0: `make records-check`, "164 records, 0 problem(s)";
  `tools/claims.py` on the MADR, "20 of 20 pass", and on this PLAN, "3 of 3
  pass" (both pinned to `e5eea9d`); `tools/recordlint.py --changed`, "2
  record(s); no rule rose"; `tools/denyscan.py --changed`, "0 hit(s) in 4
  file(s)"; `make links-check`, "0 problem(s), 1263 relative link(s) in 178
  file(s)"; `make disclosure-check`, "0 hit(s) in 1585 file(s)"; `make
  markdownlint`, "0 issues". Run after this entry was written.
* Step 3, the test file and the records are staged for the owner to commit.
  Phase 4 waits for the owner's commit and word to push.
