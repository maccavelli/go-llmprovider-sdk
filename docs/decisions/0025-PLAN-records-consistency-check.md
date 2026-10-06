---
status: in-progress
date: 2026-10-06
associated-madr: "0025-MADR-records-consistency-check.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement the Records Consistency Check, and Correct the Two Stale PLAN Statuses

Associated MADR: [0025-MADR-records-consistency-check.md](0025-MADR-records-consistency-check.md)

## Goal

`make records-check` runs `scripts/check_records.py` locally and in CI; it is
self-tested; and the records pass it, with
`0004-PLAN-add-gateway-llm-providers.md` and
`0005-PLAN-canonicalize-llm-provider-configuration.md` `complete` on recorded
evidence. Done when:

* the checker fails on today's tree, naming exactly the two PLANs, and on
  each planted breach in step 1.2;
* after phase 2, `make records-check` exits 0 and `--next` prints the next
  free number;
* `make gate-selftest` passes with the new breach;
* the gate is clean at the end of each phase, and the CI change is seen
  working on the owner's next push.

## Scope

* **Added:** `scripts/check_records.py`.
* **Changed:**
  * `Makefile`: the `records-check` target;
  * `.github/workflows/ci.yml`: the gate step runs it;
  * `scripts/test_gates.py`: one breach, and the clean-copy loop's gate
    list;
  * `AGENTS.md` ("Records"; "Pre-add checks"'s CI list);
  * `docs/architecture.md`: the script list and the `make` targets;
  * `docs/decisions/0004-PLAN-add-gateway-llm-providers.md` and
    `docs/decisions/0005-PLAN-canonicalize-llm-provider-configuration.md`:
    their `status`, and a dated note each (phase 2);
  * `docs/README.md`: their two rows, and this pair's.
* **Not changed:** any Go file, `go.mod`, any other record's content, and
  the index's titles and hand-written tables.

## Implementation Steps

### Phase 1: the checker

#### 1.1 `scripts/check_records.py`

Python, the standard library only, in the style of the other `scripts/`: a
module docstring citing this PLAN, `main() -> int`, and run as `python3 -B
scripts/check_records.py`. It reads the tree from the repository root, which
it finds from its own path.

* **Default:** run the checks the MADR's Decision Outcome lists. Print each
  problem as `path: problem`, sorted. Print a closing line, `records-check:
  N records, M problem(s)`. Exit 1 if `M > 0`, else 0.
* **`--next`:** print only the next number, four digits, and exit 0. It
  runs no checks, so a broken index does not block numbering.
* **Front matter:** the YAML block between the first two `---` lines. Only
  `status`, `associated-madr` and `parent-madr` are read, as plain
  `key: value` lines, unquoted, so no YAML library is needed.
* **The index:** rows of `docs/README.md` that match `| NNNN | KIND |
  [title](decisions/... or reports/...) | status |`, and the line `N
  records.`.

#### 1.2 Seen to fail

On a scratch copy of the tree, each planted alone. Each must exit 1 and
name its file and problem:

| Breach | Expected problem |
| :--- | :--- |
| none: today's tree | the two PLANs' status `accepted`, and nothing else |
| a file `docs/decisions/0099-PLAN-orphan.md` with `status: proposed` | no MADR numbered 0099; not indexed |
| `docs/decisions/0024-MADR-...`'s row removed from the index | not indexed; the count is off by one |
| a row's status changed to `complete` for a `proposed` record | the index status differs from the file's |
| `docs/reports/notes.md` | a name outside the pattern |
| a second MADR numbered 0023 | two MADRs for one number |

Then `--next` on the scratch copy prints `0026`.

#### 1.3 Wiring

* `Makefile`, beside `generate-check`:

  ```make
  # 0025-PLAN: records' names, statuses, pairs and index (0025-MADR).
  records-check: ## Checks the decision records and their index
  	python3 -B scripts/check_records.py
  ```

  and `records-check` in `.PHONY` (`Makefile:12`).
* `ci.yml`'s gate step runs `make parity-check dep-check coverage-check
  api-check generate-check records-check gate-selftest`, and its name gains
  "records".
* `scripts/test_gates.py`:
  * `test_records_check_plan_status`: plant `status: accepted` in a
    `complete` PLAN on the copy, and assert exit 1 and the PLAN's filename
    in the output;
  * `records-check` joins the `GATES` table (`test_gates.py:30`), so the
    clean-copy test runs it.

  The clean-copy test passes only after phase 2. So phase 1's self-test run
  uses a copy with phase 2's status change applied, and the record says so.
* `AGENTS.md`:
  * "Records": **Next number** is `python3 -B scripts/check_records.py
    --next`. The rule it applies stays stated;
  * `make records-check` checks the rules above;
  * the CI list in "Pre-add checks" gains `records-check`.
* `docs/architecture.md`: `scripts/check_records.py` in the script list; and
  `records-check` in the `make` targets and the CI description, with one
  line on what it checks.

#### 1.4 Gate and record

* G1–G11 as `0021-PLAN-harden-and-tune-after-the-v1-1-review.md` defines
  them, with its phase 6 additions. G7 gains `records-check`, which is
  expected to fail on the two PLANs until phase 2; the record shows that
  output.
* The execution record below; stage. Phase 1 and phase 2 may be committed
  together: CI is red on `records-check` between them.

### Phase 2: the two statuses

#### 2.1 Evidence

Read-only, before any edit:

* in `mcplib`'s history, the merges `50ac165` (its PLAN 0003, here 0004),
  and `29ef039` and `5786e6d` (its PLAN 0004, here 0005). Record each
  commit's subject;
* each PLAN's deviation log: its last entry's phase is the plan's last
  phase.

**Stop and prompt** if a merge is missing, or either log ends before the
plan's last phase.

#### 2.2 The edit

In each PLAN:

* `status: accepted` → `status: complete`;
* a section appended at the end:

  ```markdown
  ## Amendment 2026-10-06: status

  The status was `accepted` when this plan was migrated from `mcplib`, and
  `0002-MADR-migrate-llmprovider-from-mcplib.md` did not set it. It is
  `complete` (`0025-MADR-records-consistency-check.md`): ...
  ```

  The "..." is filled with step 2.1's evidence: the merge commits and the
  deviation log's last entry. Add that the acceptance criteria describe
  `mcplib`'s tree of 2026-08-29, and were not re-run here.

`docs/README.md`: both rows read `complete`.

#### 2.3 Gate, close-out

* `make records-check` exits 0, and `make gate-selftest` passes, the
  clean-copy test included.
* The gate, as in 1.4.
* This PLAN `complete`, and its row. The MADR's status is the owner's.
* Stage; the owner commits and pushes. The CI change is seen working on
  that push, and recorded here when the owner reports it.

## Verification

* **V1:** step 1.2's six failures and `--next`.
* **V2:** `make records-check` fails before phase 2, naming the two PLANs,
  and passes after.
* **V3:** the self-test's new breach fails with exit 1, and `make
  gate-selftest` passes.
* **V4:** the gate is clean at the end of each phase.
* **V5:** CI green on the owner's push.

## Rollout and Rollback

* **Rollout:**
  * staged by the agent, and committed by the owner, with phases 1 and 2
    together or in order;
  * no release: nothing here is in the module's build.
* **Rollback:**
  * revert the commit;
  * the two PLANs' amendments are history, and can stay if only the tooling
    is reverted.

## Execution Record

### Phases 1 and 2 (2026-10-06)

Approved by the owner on 2026-10-06 ("proceed", in answer to "proceed to
0025"), after the records were committed in `759592c`. Both phases were run
before staging, as step 1.4 allows. No deviation: the four differences from
the steps' text below change no decision and no scope, and each is named.

**1.1, `scripts/check_records.py`.**
* **Built** as the MADR's Decision Outcome lists. It reads front matter as
  plain `key: value` lines, and parses the index only inside its `## Records`
  section: the "I want to…" table also links records.
* **Built beyond the step's list:** the "N records." line is compared with
  the number of rows as well as with the number of records. Step 1.2's
  "count is off by one" for a removed row needs it, since the count of
  records does not change.
* **On `HEAD`'s records** (`759592c`, with the approval's statuses): exit 1,
  exactly the two PLANs:

  ```text
  docs/decisions/0004-PLAN-add-gateway-llm-providers.md: PLAN status 'accepted', want one of proposed, in-progress, complete, superseded
  docs/decisions/0005-PLAN-canonicalize-llm-provider-configuration.md: PLAN status 'accepted', want one of proposed, in-progress, complete, superseded
  records-check: 55 records, 2 problem(s)
  ```

  A first run also reported "no 'N records.' line", because the pattern
  required the line to end after "records."; the index's line goes on with
  "Cite them by full filename.". The pattern was fixed before any of the
  runs below.

**1.2, seen to fail.**
* **Clean baseline:** each breach was planted alone, on a scratch copy with
  phase 2's status change applied. The baseline gives exit 0, `55 records,
  0 problem(s)`. Without it every run fails on the two PLANs anyway, which
  proves nothing about the breach.
* **The step's five breaches,** each exit 1:

  | Breach | Output |
  | :--- | :--- |
  | `0099-PLAN-orphan.md`, `status: proposed` | "no MADR numbered 0099"; "not indexed"; "says 55 records; there are 56" |
  | `0024-MADR`'s row removed | "not indexed"; "says 55 records; its Records table has 54 rows" |
  | `0025-PLAN`'s row says `complete` | "indexed as 'complete'; the file says 'in-progress'" |
  | `docs/reports/notes.md` | "name is not NNNN-KIND-slug.md" |
  | `0023-MADR-second-decision.md` | "one of 2 MADRs numbered 0023", for both; "not indexed"; the count |

  The third used `0025-PLAN`, `in-progress`: after the owner's approvals no
  record was `proposed`.
* **Six more,** so each rule is seen to fail, each exit 1 with one problem:
  * 0024's only PLAN renamed `0024-PLAN-other-slug.md`: "has slug
    'other-slug'; its MADR's is 'opencode-live-system-message-test'";
  * 0024-PLAN's `associated-madr` naming 0023's MADR: "names MADR
    '0023-MADR-huggingface-tool-choice-none.md'";
  * 0024-PLAN with no `associated-madr`: "front matter names no
    associated-madr or parent-madr";
  * 0024-MADR `status: done`, its row matching: "MADR status 'done'";
  * 0015-REPORT `status: accepted`, its row matching: "REPORT status
    'accepted', want none or observation";
  * 0024-MADR moved to `docs/reports/`: "a MADR belongs in docs/decisions/".
* **`--next`:** `0026`.

**1.3, wiring.**
* `Makefile`: `records-check`, in `.PHONY` and as a target.
* `ci.yml`: the gate step runs it, and its name says "records".
* `scripts/test_gates.py`:
  * `records-check` is in `GATES`;
  * `test_records_check_plan_status` is added.
* **The new case, seen to fail.** As first written, it required only the
  PLAN's filename in the output. On a scratch clone with the checker's PLAN
  status rule planted out, it still passed: the planted status also makes
  the index row disagree, and that message names the file too. It now
  requires `0024-PLAN-opencode-live-system-message-test.md: PLAN status
  'accepted'`. With the rule out it fails (`FAILED (failures=1)`), and on
  the tree it passes.
* `AGENTS.md`: "Next number" names `check_records.py --next`, a bullet
  describes `make records-check`, and the CI list includes it.
* `docs/architecture.md`: the script list, the `make` targets, a
  `records-check` entry, `gate-selftest`'s breaches, and the CI list.

**2.1, evidence,** read-only, in `mcplib`:

| Merge | Subject (2026-08-29) | Commits |
| :--- | :--- | :--- |
| `50ac165` | Merge branch 'feat/gateway-llm-providers' | 9 |
| `29ef039` | Merge branch 'feat/canonicalize-llm-config' | 9 |
| `5786e6d` | Merge branch 'feat/plan-0004-phases-7-8' | 3, each recording deviations (D2–D4, D5–D6, D7) |

0004-PLAN's log ends in Phase 7, of 7 (D3, D4). 0005-PLAN's ends in Phase
8, of 8 (D7). No stop condition applied.

**2.2, the edit.** Both PLANs are `complete`. Each has an "Amendment
2026-10-06: status" with step 2.1's evidence, and says the acceptance
criteria were not re-run. Their index rows say `complete`.

**2.3, the checks.**
* `make records-check`: `records-check: 55 records, 0 problem(s)`, exit 0.
* `make gate-selftest`: 8 tests OK in 56 s, and in 50 s in the gate below,
  the clean-copy test included.

**The gate,** all exit 0:

* G1: `make pre-add-check`, "396 file(s) clean (gofmt, golangci-lint, go
  vet, go test, govulncheck)";
* G2: `go vet` for darwin, linux and windows; G3: `go vet -tags
  live_gateways ./...`;
* G4: 26 packages ok with `-race -cover`, and `go test -race -shuffle=on
  ./...` exit 0; `llmprovider` 97.9% against its 95.9% floor;
* G5: `go mod tidy -diff` clean;
* G6: `make lint`, 0 issues on both builds;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: go.mod and 27 packages on both builds, 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: against `v1.2.1`, 0 incompatible changes;
  * generate-check: 1 generated file, 0 problems;
  * records-check: 55 records, 0 problems;
  * gate-selftest: 8 tests OK in 50 s;
* G8: 0 issues in the repository's 7-file markdownlint scope, which
  includes `AGENTS.md` and `docs/architecture.md`;
* G9: stable;
* G10: 0 problems, 593 relative links in 65 files;
* G11: 0 hits in 11 files.

**V-diff** against `008d956`, counted before this record was written, plus
the new `scripts/check_records.py`:

```text
.github/workflows/ci.yml                             |  4 ++--
AGENTS.md                                            | 10 +++++++---
Makefile                                             |  6 +++++-
docs/README.md                                       |  8 ++++----
docs/architecture.md                                 | 20 +++++++++++++++-----
.../decisions/0004-PLAN-add-gateway-llm-providers.md | 18 +++++++++++++++++-
...5-PLAN-canonicalize-llm-provider-configuration.md | 18 +++++++++++++++++-
.../decisions/0025-MADR-records-consistency-check.md |  6 +++++-
.../decisions/0025-PLAN-records-consistency-check.md |  2 +-
scripts/test_gates.py                                |  7 +++++++
10 files changed, 80 insertions(+), 19 deletions(-)
```

No Go file changed.

**Pending:** the CI change is seen working on the owner's next push (Goal,
V5). This PLAN stays `in-progress` until then.
