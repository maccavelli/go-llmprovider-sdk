---
status: complete
date: 2026-10-01
associated-madr: "0018-MADR-apache-2-license.md"
---
# Implement the Apache License 2.0 for go-llmprovider-sdk

Associated MADR: [0018-MADR-apache-2-license.md](0018-MADR-apache-2-license.md)

## Goal

The repository is licensed under the Apache License 2.0. The licence
file is the fleet copy. The index, architecture, README, and 0002
records describe that fact.

## Scope

In:

* `LICENSE`
* `README.md` License section
* `docs/architecture.md` tree and "What is not here"
* [0002-MADR-migrate-llmprovider-from-mcplib.md](0002-MADR-migrate-llmprovider-from-mcplib.md)
  and [0002-PLAN-migrate-llmprovider-from-mcplib.md](0002-PLAN-migrate-llmprovider-from-mcplib.md)
  amendments
* `docs/README.md` record index and "I want to…" row

Out:

* A `NOTICE` file. 0018-MADR does not add one.
* Filling the Apache appendix copyright placeholders.
* Source-file copyright headers.
* Any change to `go-core-lib`. That module already has this `LICENSE`.

## Implementation Steps

1. Copy `LICENSE` from `go-core-lib` byte for byte. `cmp` against that
   file and against `magic-cli-remote/LICENSE`. Both exit 0.
2. Append a License section to `README.md`: "Licensed under the Apache
   License, Version 2.0. See [LICENSE](LICENSE)." The wording matches
   `go-core-lib`.
3. In `docs/architecture.md`, add `LICENSE` to the tree listing as
   Apache License 2.0, and remove the "A licence file." bullet from
   "What is not here".
4. Amend 0002-MADR with a dated note that the licence deferral is
   closed by this pair. Strike 0002-PLAN Phase 2b's "A `LICENSE`"
   out-of-scope bullet and point here.
5. Index this pair in `docs/README.md`. Update the record count. Add
   an "I want to…" row for the licence.

## Verification

* `cmp LICENSE` against `go-core-lib/LICENSE` exits 0.
* `wc -l LICENSE` is 201.
* `README.md` contains a License heading and a relative link to
  `LICENSE`.
* `docs/architecture.md` does not contain the sentence "A licence
  file."
* `docs/README.md` lists 0018-MADR and 0018-PLAN.

## Rollout and Rollback

The file is documentation. Removing `LICENSE` would return the tree to
0002's deferred state; do not do that once this plan is complete.

## Execution record

**2026-10-01.** Owner directed Apache-2.0 for this repository, `pi-go`,
and `go-core-lib`. All five steps ran in this change.

* `LICENSE` copied from `go-core-lib`. `cmp` against
  `go-core-lib/LICENSE` exit 0. `wc -l` is 201.
* First-fail of that `cmp` (scratch copy with one extra byte): exit 1,
  `cmp: EOF on '.../go-core-lib/LICENSE' after byte 11357, line 201`.
* `README.md` License section matches `go-core-lib`.
* `docs/architecture.md` lists `LICENSE` in the tree. The "A licence
  file." bullet is gone.
* 0002-MADR gained Amendment 2026-10-01. 0002-PLAN Phase 2b's LICENSE
  out-of-scope bullet is struck.
* `docs/README.md` is 41 records and has the licence "I want to…" row.

No `NOTICE` was added, as scoped. The Apache appendix was not filled.
