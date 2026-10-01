---
status: accepted
date: 2026-10-01
decision-makers: repository owner
consulted: 0002-MADR-migrate-llmprovider-from-mcplib.md, go-core-lib 0001-MADR-scaffold-shared-go-library.md
informed: pi-go, go-core-lib, mcplib, magic-cli-remote
---
# The module is released under the Apache License 2.0

## Context and Problem Statement

[0002-MADR-migrate-llmprovider-from-mcplib.md](0002-MADR-migrate-llmprovider-from-mcplib.md)
deferred a licence file: "No `LICENSE` is added. That is the owner's
choice and is not decided here." [0002-PLAN-migrate-llmprovider-from-mcplib.md](0002-PLAN-migrate-llmprovider-from-mcplib.md)
Phase 2b listed a `LICENSE` as out of scope for the same reason.
`docs/architecture.md` still lists "A licence file" under what is not
here.

The owner directed on 2026-10-01 that this repository, `pi-go`, and
`go-core-lib` are Apache-2.0. `go-core-lib` already is: its `LICENSE`
is the Apache License 2.0, 201 lines, appendix left as the unfilled
template, `cmp`-identical to `magic-cli-remote`. This module still has
none.

The problem: choose the licence this module is released under, and put
that choice in the tree.

## Decision Drivers

* Owner-stated licence Apache-2.0 for this repository, `pi-go`, and
  `go-core-lib`.
* Fleet copies already in use: `go-core-lib` and `magic-cli-remote`
  ship the same 201-line Apache-2.0 text with the appendix unfilled.
* 0002 deferred the file as an owner choice; that choice is now made.
* Copyright stays out of `LICENSE`. The fleet copies leave the
  appendix template unfilled; they do not invent a copyright header.

## Considered Options

* Apache License 2.0, the fleet `LICENSE` copy with the appendix left
  unfilled
* MIT License
* Leave the tree unlicensed

## Decision Outcome

Chosen option: "Apache License 2.0, the fleet `LICENSE` copy with the
appendix left unfilled", because the owner directed Apache-2.0 and the
file is already the fleet's canonical copy.

`LICENSE` is copied byte-for-byte from `go-core-lib` (which is
`cmp`-identical to `magic-cli-remote`). The appendix template is not
filled. A `NOTICE` file is not added: `go-core-lib` has none, and this
module has no third-party attribution that Apache §4(d) would require
to live there.

The root `README.md` gains a License section that names Apache-2.0 and
links `LICENSE`. `docs/architecture.md` lists `LICENSE` in the tree and
drops "A licence file" from what is not here.

### Consequences

* Good, because a reader and a consumer can see the licence from the
  first file they open.
* Good, because the text matches `go-core-lib` and `magic-cli-remote`,
  so the three sibling libraries share one licence file.
* Neutral, because the appendix copyright placeholders stay unfilled,
  as in those copies.
* Neutral, because 0002's deferral is closed rather than silently
  rewritten: this record decides what 0002 left open.

### Confirmation

* `cmp LICENSE` against `go-core-lib/LICENSE` exits 0.
* `LICENSE` is 201 lines and begins with `Apache License`.
* `README.md` has a License section linking `LICENSE`.
* `docs/architecture.md` does not list "A licence file" under what is
  not here.

## Pros and Cons of the Options

### Apache License 2.0, the fleet `LICENSE` copy with the appendix left unfilled

* Good, because it is the owner's choice and the fleet's existing
  copy.
* Good, because patent grant and NOTICE rules are explicit.
* Neutral, because the appendix is unfilled, matching the source copy.

### MIT License

* Good, because it is short and common for Go libraries.
* Bad, because the owner chose Apache-2.0, and `go-core-lib` already
  ships Apache-2.0.

### Leave the tree unlicensed

* Good, because it is the status 0002 recorded.
* Bad, because the owner has now chosen a licence, and consumers have
  no grant in-tree.

## More Information

* Closes the licence deferral in
  [0002-MADR-migrate-llmprovider-from-mcplib.md](0002-MADR-migrate-llmprovider-from-mcplib.md)
  (Phase 2b amendment) and
  [0002-PLAN-migrate-llmprovider-from-mcplib.md](0002-PLAN-migrate-llmprovider-from-mcplib.md)
  ("Not in Phase 2b").
* Implemented by
  [0018-PLAN-apache-2-license.md](0018-PLAN-apache-2-license.md).
* `go-core-lib` `0001-MADR-scaffold-shared-go-library.md` §7 already
  chose Apache-2.0 for that module and noted this repository had no
  licence file. That historical sentence is left as written; this
  record is the later fact.
