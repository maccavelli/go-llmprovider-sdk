# Maintainer tools

The tools a maintainer, or an agent, runs while working in this repository,
as [0035-MADR](../docs/decisions/0035-MADR-maintainer-tools-in-python.md)
decides. The gates CI runs are in [`scripts/`](../scripts/); these are not run
by CI.

Each tool is Python 3 on the standard library. Each finds the repository root
from its own file, so it runs from any working directory. None names a host,
an account or a machine path. Anything outside the repository is an argument.

Exit codes, for every tool: **0** clean, **1** a finding, **2** a tool error.

| Tool | What it does | Usage |
| :--- | :--- | :--- |
| `gate.py` | the local gate: 25 checks on a scratch clone of the working tree, staged there, started once the load average is below the CPU count | `tools/gate.py [--src DIR] [--out DIR] [--max-load N] [--timeout S]` |
| `plant.py` | runs plants from a JSON spec on a scratch clone; fails if any plant is not caught; a plant's text may be read from a file (`old_file`, `new_file`) | `tools/plant.py SPEC.json` |
| `linkcheck.py` | every relative link in every Markdown file resolves | `tools/linkcheck.py [--detail FILE]` |
| `denyscan.py` | no file names this machine's account, its host, or a real home path; it prints the kind of hit, never the identifier | `tools/denyscan.py FILE...` or `--changed` |
| `apidiff.py` | every exported API change between two refs, or a ref and the tree, at `scripts/check_api.py`'s pin | `tools/apidiff.py OLD [NEW]` |
| `edit.py` | exact replacements, appends and new files from a JSON spec, each asserted, all applied or none; any text may be read from a file beside the spec (`old_file`, `new_file`, `append_file`, `create_file`) | `tools/edit.py SPEC.json` |
| `records.py` | a record's status and date, and its row in `docs/README.md`, changed together; `add` indexes a record and updates the count; `scripts/check_records.py`'s checks run after | `tools/records.py add RECORD...` or `status RECORD STATUS [--date YYYY-MM-DD]` |
| `recordlint.py` | the markdownlint findings an edit adds to a record: each rule counted at `HEAD` and in the tree, on copies, with the records' own conventions (MD004 `consistent`, MD010 `code_blocks: false`) | `tools/recordlint.py RECORD...` or `--changed` |
| `liveskips.py` | sorts a `go test -v` run into passes, skips with their reasons, and failures | `tools/liveskips.py RUN.txt` |
| `consumer.py` | after a tag: a consumer of every public package built for six platforms, and optionally a dependent repository moved to the version | `tools/consumer.py VERSION [--dependent PATH]` |
| `claims.py` | each row of a record's evidence index holds its text, on its line, at the revision the record's `<!-- claims: … -->` marker names; finds other repositories beside this one; `--json` for agents, `--fixes` writes an `edit.py` spec for rows found on another line; `--plants SPEC.json` runs plants of false claims on a copy of a record, each of which must fail. Run by hand before a record is staged, not by the gate | `tools/claims.py RECORD...` or `--changed`, `[--repo NAME=DIR] [--rev NAME=REV] [--json] [--fixes SPEC.json]`, or `--plants SPEC.json` |
| `catalogcheck.py` | at a refresh: records a snapshot with `internal/cmd/catalogs -snapshot`, or checks one given with `--dir`, and reports each rule that differs from the facts (the `TestAgreement0041_` tests, run with `LLMPROVIDER_SNAPSHOT_DIR`) and how many of each open catalog's six changed under each profile, against the newest committed snapshot. It reports and never edits (0041-MADR D7); a finding is for a person to review, not a gate | `tools/catalogcheck.py [--dir DIR \| --out DIR] [--baseline DIR] [--doc FILE]` |
| `offline.py` | after a tag: a program reads a `ModelInfo` from a metadata document with no request, built for six platforms; two built-in plants must each fail it | `tools/offline.py VERSION [--metadata FILE] [--provider ID] [--model ID]` |

`_common.py` holds what they share:

- the repository root;
- a runner that never uses a shell;
- the load wait;
- the scratch clone.

## The gate's checks

In order:

1. `pre-add-check`.
2. `go vet` for darwin, linux and windows, with and without
   `live_gateways`.
3. `go test -race -cover`, then `go test -shuffle=on`.
4. `go mod tidy -diff`, `lint`, `parity-check`, `dep-check`,
   `coverage-check`, `api-check`, `generate-check`, `records-check`,
   `tables-check`, `fixtures-check` and `gate-selftest`.
5. markdownlint, on every Markdown file but the records.
6. `TestWireGoldens` three times.
7. The link check, and the identifier scan of the changed files.
8. `tools-test`.

Each check's output is in `--out`, with the exit codes in `summary.out`.

## Plant specs

```json
{"packages": ["./llmprovider/catalog/"], "run": "_0033$",
 "plants": [{"name": "the decoder drops limit.output", "file": "llmprovider/catalog/model_metadata.go",
             "old": "…", "new": "…"}]}
```

`"command"` replaces the default `go test -count=1 -run RUN PACKAGES`, so a
spec can plant in `tools/` and run `make tools-test`'s command.

## Tests

`make tools-test` runs [`test_tools.py`](test_tools.py). Each tool there fails
on a planted input and passes a clean one, in temporary directories. Run it
after changing a tool. The gate runs it too.

## The one tool in Go

`llmprovider/internal/cmd/sdkaudit` is a maintainer tool written in Go, not
Python, as 0048-MADR D8 decides: it parses the module with `go/ast`, which no
regular expression matches. It lives beside `internal/cmd/catalogs`, as a
package of the module, so the module's checks hold it: its rows in the
package tables, its tests, and R47's 80 % coverage.

```bash
go run ./llmprovider/internal/cmd/sdkaudit            # the measures, as JSON
go run ./llmprovider/internal/cmd/sdkaudit -check     # R50–R52's violations
```

With `-check` it exits 1 on a violation. `-rules` picks the rules, and
`-codedoc` the packages whose comment must hold a code block.
