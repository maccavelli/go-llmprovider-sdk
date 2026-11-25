---
status: in-progress
date: 2026-11-25
associated-madr: "0094-MADR-refuse-an-edit-to-a-released-changelog-section.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement: Refuse an Edit to a Released CHANGELOG Section

Associated MADR: [0094-MADR-refuse-an-edit-to-a-released-changelog-section.md](0094-MADR-refuse-an-edit-to-a-released-changelog-section.md)

*(2026-11-25, revision 1, a draft outside the tree: rewritten for the MADR's revision 1, the owner's answers to OQ1 to OQ3 and the missing-case review. 0 rows open.)*

*(2026-11-25, approved. Owner ruling, quoted exactly, given for the MADR and this PLAN together: "Accept; approve; build now (Recommended)". The repository's PLAN statuses (`scripts/check_records.py`) have no `approved`; an approved PLAN awaiting its build carries `in-progress`, as 0093-PLAN does.)*

## Goal

After this PLAN, `make changelog-check` compares each released section of `CHANGELOG.md` and `authcontract/CHANGELOG.md`, and each released tag's compare link, with its text in the first tag's file that carries it, or at the reviewed commit an `ACCEPTED` entry names, and fails an edit. A heading may gain a `[YANKED]` suffix. Missing tags or history exit 2. So 0092-MADR ST27, and the same rule for every later record, is checked by CI, with git reads only.

The MADR has 0 open rows and is `proposed` until the owner accepts it. Phase 0 places the records (the bootstrap rule allows it) and takes the acceptance and the approval of this PLAN. No phase after Phase 0 starts until the MADR is `accepted`, this PLAN is approved, and the owner says to proceed.

## Scope

* Phase 0: `docs/decisions/0094-MADR-*`, `docs/decisions/0094-PLAN-*`, `docs/README.md` (two index rows and the count).
* Phase 1: `scripts/check_changelog.py` (`RELEASE` with a `[YANKED]` suffix, the carry match, the reader, the guards, the comparison, the link checks, `ACCEPTED`, the output, the docstring); `scripts/test_gates.py` (the helpers `tag_dated` and `close_window`, the new `GateSelfTest` cases, a new `ChangelogReleasedTest` class); the `Makefile`'s comment and help line; `docs/architecture.md`; `docs/guides/api-standards.md` (R53 and its row); `AGENTS.md`; `CHANGELOG.md` (the preamble sentence and a **Changed** entry under `[Unreleased]`); dated notes on `docs/decisions/0092-MADR-*` and `docs/decisions/0092-PLAN-*` (RS22).
* Not in scope: `.github/workflows/*` (RS14); `--section`'s output (RS19); `tools/gate.py`, `MARKDOWN_GATES`, `tools/README.md`, `README.md` (RS14, RS23); `gh release edit`, which an accepting record carries as an owner step (RS28); tags moved or re-made (RS26); other repositories (RS27); any Go code (RS25).

## Rules for Every Phase

* **Red, then green.** A case marked red fails at the phase's base on its assertion, not on an error loading the script or a missing helper (the helpers land first, inert). A case of a function the base lacks takes its red from its plants. A case marked green is a pin and passes at the base. After the change, all pass.
* **Plants.** Each is one change to `scripts/check_changelog.py` (or, for RS17, to `scripts/test_gates.py`) on a scratch clone, never the tree, through `tools/plant.py` with `"command": ["python3", "-B", "-m", "unittest", "scripts/test_gates.py", "-k", "<case>"]`; it must turn its own row's case red; the execution record quotes the failing line.
* **Not depending on the latest tag.** No case names the newest tag or reads `[Unreleased]`'s entries; a case that tags `v99.0.0` or adds a section calls `close_window` first (RS17, RS33).
* **Heavy runs.** `make pre-add-check`, the CI `make` set (it holds `coverage-check` and `gate-selftest`), and `python3 -B tools/gate.py` go through `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`, or another host through `scripts/run-on-host`, one at a time per host. A `-k` subset of `scripts/test_gates.py`'s changelog cases runs no Go and may stay on the session host.
* **Staged only.** Each phase ends with its files staged; the owner commits.
* **Stop conditions:** an existing case of `scripts/test_gates.py` needs a change no row names; the tree differs from a baseline (MS4 no longer holds); a red case cannot be red at the base on its assertion; a row's outcome cannot be met as written; a cited line has moved with its text changed; the same step fails twice; a third build pass of one phase.

## Implementation Steps

### Phase 0: records into the tree, acceptance, approval

1. **Re-ground.** `python3 -B tools/claims.py <the MADR> --rev self=<base>`; move each row whose line moved by dated amendment with ids ending in "now". Re-run `<scratch>/s0094/prototype.py` at the base: MS4 holds.
2. **The pair and the index.** Copy the MADR and this PLAN into `docs/decisions/`; `python3 -B tools/records.py add` for both; `make records-check` (plant: this PLAN's index row removed on a scratch clone, caught); `python3 -B tools/recordlint.py --changed`; `python3 -B tools/claims.py --changed` with its plants; `python3 -B tools/denyscan.py --changed`.
3. The owner accepts the MADR and approves this PLAN (`python3 -B tools/records.py status`: `accepted`, `in-progress`).

Exit criterion: no blocker; claims pass at the base and their plants are caught; `make records-check` clean; staged.

### Phase 1: the comparison (RS1 to RS25, RS28 to RS33)

1. **Re-ground** at the phase's base: E1 to E46 with `tools/claims.py --rev self=<base>`. Record whether the base is in the release window and which tag is newest; no case may depend on either.
2. **Helpers, inert.** `tag_dated(tree, tag, date)` (as `tag_tree`, with `GIT_AUTHOR_DATE` and `GIT_COMMITTER_DATE` at `<date>T12:00:00+0000`) and `close_window(tree)` (when the copy's first release section has no tag, `tag_dated` it at its heading's date), in `scripts/test_gates.py`; `ChangelogReleasedTest` loads `scripts/check_changelog.py` by path (`importlib.util.spec_from_file_location`).
3. **Cases first.**

   | Case | Rows | Base | Plant (on the finished code) |
   | :--- | :--- | :--- | :--- |
   | `test_changelog_check_released_entry_added`: exit 1, the heading's line, `differs from its text at v1.14.0`, `first at line` | RS1, RS15 | red | the comparison skipped; the baseline tag left out of the message |
   | `test_changelog_check_released_backfilled_section`: v1.1.0 edited, names v1.16.0 | RS2 | red | the baseline read only at the section's own tag |
   | `test_changelog_check_released_late_section`: v1.24.0 edited, names v1.24.1 | RS2 | red | the same plant |
   | `test_changelog_check_released_carried_by_version`: `v99.0.0`'s file heading undated, the working body edited; exit 1 naming `v99.0.0` | RS2 | red | "carries" matched with the full `RELEASE` |
   | `test_changelog_check_released_newest_tagged`: the first release section whose tag exists gains an entry | RS3 | red | the first release section never compared |
   | `test_changelog_check_released_no_baseline`: (a) the newest tag, no file carries it, exit 0, `1 with no released text`; (b) an older such tag, exit 1 | RS4, RS15 | red ((a)'s summary; (b)) | the exception taken off ((a)); the exception given to every tag ((b)) |
   | `test_changelog_check_unreleased_and_release_commit_pass`: (a) an entry under `[Unreleased]`; (b) one untagged `[v99.0.0]`; (c) a preamble line; each exit 0 | RS5, RS20 | green (pins) | `[Unreleased]` compared with the newest tag's ((a)); an untagged section read as differing ((b)); the preamble compared ((c)) |
   | `test_changelog_check_released_after_its_tag`: `[v99.0.0]` tagged, exit 0; an entry then added, exit 1 | RS6 | red (the edit) | the baseline searched among tags below the section's version |
   | `test_changelog_check_released_whitespace_passes`: trailing spaces on a v1.14.0 line, a blank line at its end; exit 0 | RS7 | green (pin) | lines compared unnormalised |
   | `test_changelog_check_released_rewrap`: two lines of a v1.14.0 entry joined; exit 1 | RS8 | red | white space collapsed before comparing |
   | `test_changelog_check_released_link_changed`: `[v1.14.0]`'s link changed; exit 1 | RS9 | red | link lines not compared |
   | `test_changelog_check_released_last_section`: v1.0.0's last line edited; exit 1 naming v1.16.0 | RS9 | red | the last section never compared |
   | `test_changelog_check_unreleased_link_passes`: the `[Unreleased]` link changed, a `[v99.0.0]` link added; exit 0 | RS9 | green (pin) | the link block read as part of the last section |
   | `test_changelog_check_released_contract_section`: `authcontract/v0.1.0` tagged with `tag_dated` at 2026-11-12, its body edited; exit 1, `differs`, the file and full tag named, no `is dated` | RS10 | red | the nested file not compared |
   | `test_changelog_check_contract_untagged_not_compared`: the nested body edited, no tag; exit 0 | RS10 | green (pin) | RS11's guard counting one untagged section |
   | `test_changelog_check_missing_tags_is_a_tool_error`: (a) no tags; (b) tags from v1.20.0 on deleted; (c) two untagged sections above a closed window; each exit 2 | RS11, RS15 | red | the guard removed ((a), (b)); its bound off by one ((c)) |
   | `test_changelog_check_shallow_is_a_tool_error`: the copy's HEAD in `.git/shallow`; exit 2 | RS12 | red | the shallow check removed |
   | `ChangelogReleasedTest.test_missing_tag_commit_is_a_tool_error` | RS12 | from its plant | a missing commit read as a tag with no file |
   | `test_changelog_check_released_accepted_edit`: (a) to (e) as RS13 | RS13, RS15 | from its plants (the base has no `ACCEPTED` to plant into) | `ACCEPTED` ignored; the link compared with the tag's line; any text passing for a listed tag; a carrier-less commit falling back to the tag; an absent commit read as no file; the entry's form not checked |
   | `ChangelogReleasedTest.test_git_calls_per_file`: two `cat-file` calls per file, no `show`, no `fetch` | RS16 | from its plant | one `git show` per tag |
   | `test_changelog_check_released_duplicate_section`: `has a section already`, no `differs` | RS18 | green at the base (no comparison runs); red from its plant | the last occurrence compared |
   | `test_changelog_check_released_tag_not_utf8`: exit 2 naming `v99.0.0` | RS21 | red | decoded with `errors="replace"` |
   | `test_changelog_check_released_yanked`: (a) marker on v1.14.0, exit 0; (b) with the wrong date, `is dated`; (c) with the body edited, `differs`; (d) `--section v1.14.0` prints the body | RS29, RS19 | red (at the base the marked heading is a form problem, and `--section` finds no section) | `RELEASE` unchanged; the date check skipped for a yanked heading; a yanked section not compared |
   | `test_changelog_check_yanked_untagged`: exit 1, `marked [YANKED] and has no tag` | RS30 | red | the check removed |
   | `test_changelog_check_released_link_twice`: an identical `[v1.14.0]:` line added; exit 1 | RS31 | red | links read last-wins, no duplicate check |
   | `test_changelog_check_prerelease_tag_ignored`: `v99.0.0-rc.1` at body X, the working body Y; exit 0 | RS32 | green (pin) | `VERSION` loosened to allow a suffix; rc tags taken as baselines |
   | existing: the six `test_changelog_check_*` heading cases, the four `test_changelog_section_*` cases, `test_clean_copy_passes_every_gate`, `WorkflowContractTest` | RS14, RS18, RS19 | green (pins) | none: no target, workflow or contract change |

4. **The code**, in `scripts/check_changelog.py`, per MADR D1 to D6: `RELEASE` with the optional `[YANKED]` suffix group; a carry match on `## [vX.Y.Z]`; the reader (`git cat-file --batch-check` for `<tag>^{commit}` and each accepted commit, `git cat-file --batch` for each file), one process each per file; the shallow guard, the untagged-newer guard (two or more), the missing-commit and UTF-8 errors (exit 2); the body, its normalisation, the baseline search, the no-baseline rule; the link comparison and the second-link problem; `ACCEPTED: dict[str, tuple[str, str]] = {}` with its form checks; the `[YANKED]`-without-tag problem; the messages and the summary. The existing checks' outcomes and `--section`'s output do not change.
5. **The docs** (RS23, RS24): the script's docstring; the `Makefile`'s comment and help line; `docs/architecture.md`'s scripts-table line, a `changelog-check` item in its Tooling list, its `gate-selftest` item; R53 (released sections frozen; the later-file baselines of v1.0.0 to v1.15.0 and v1.24.0; `[YANKED]`; `ACCEPTED` with its record and the owner's `gh release edit` step) and its table row; `AGENTS.md`'s CI paragraph (an entry goes under `[Unreleased]`); `CHANGELOG.md`'s preamble sentence and a **Changed** entry under `[Unreleased]`. Return docs rows: `path:line — fixed` or `— open: <reason>`, and `docs: searched <paths>`.
6. **0092's notes** (RS22): dated annotations on 0092-MADR's ST27 row and an amendment there, and on 0092-PLAN's ST27 entry. Additions only.
7. `python3 -B -m unittest scripts/test_gates.py -k changelog -k ChangelogReleased -k yanked -k prerelease -k missing_tags -k shallow` on the session host: every case green. Then every plant of step 3 through `tools/plant.py`, each caught by its own case, its failing line quoted.

Exit criterion: no blocker; every case red at the base where marked (or from its plants, or green where marked) and green after; every plant caught; every docs row `fixed`; staged.

### Phase 2: gates

1. **RS17, RS33.** Two scratch clones of the working tree (`tools/_common.scratch_clone`): on the window clone, the next release's date step (`[Unreleased]` to `[v<next>] - <date>`, a new empty `[Unreleased]`, the compare links) committed with `git commit-tree`, untagged; on the tagged clone, the same step committed with `GIT_COMMITTER_DATE` at its heading's date and tagged. On each, the step 7 subset passes and `make changelog-check` exits 0. Plants, on `scripts/test_gates.py` in each clone: RS3's case taking the first release section without checking its tag (fails on the window clone); `close_window` made a no-op (RS4's, RS6's and RS21's cases fail on the window clone).
2. **RS22.** On a scratch clone at the phase's base, 0092 ST27's plant: the first `[Unreleased]` entry moved into the newest tagged section, or, with `[Unreleased]` empty, an entry planted there; `make changelog-check` exits 1 with RS1's message.
3. `make pre-add-check` through `run-heavy` (no Go file changes; it runs the module's checks).
4. The CI set, through `run-heavy`: `make parity-check dep-check coverage-check api-check contract-check generate-check records-check tables-check fixtures-check changelog-check docs-check markdownlint links-check disclosure-check gate-selftest`.
5. `python3 -B tools/gate.py` through `run-heavy`, after step 4 has finished (one heavy run on the host at a time).
6. Records: `python3 -B tools/claims.py` on this MADR (re-grounded) and on 0092-MADR, with their plants; `python3 -B tools/recordlint.py --changed`; `python3 -B tools/denyscan.py --changed`; `make records-check`.

Exit criterion: every gate clean, its command, host and exit recorded; both clones pass and RS17's plants are caught; ST27's plant caught; staged.

### Phase 3: review

1. A `code-reviewer` review of Phases 1 and 2 against the MADR's rows; a finding outside the rows is a follow-up, recorded, not built; one that needs a ruling goes to the owner as a question, three options, the recommended first.
2. Rulings recorded by dated amendment before any code; a third build pass of a phase is a stop.

Exit criterion: no blocker; the MADR and this PLAN describe what was built; this PLAN `complete` (`tools/records.py status`).

## Verification

* Every decided row maps to a case or a step: RS1 to RS13, RS15, RS16, RS18, RS20, RS21, RS29 to RS32 to Phase 1's cases; RS14 and RS19 to the existing pins and RS29's (d); RS17 and RS33 to Phase 2 step 1; RS22 to Phase 2 step 2; RS23 and RS24 to Phase 1 step 5 with claims rows; RS25 to `make api-check`; RS28 to the accepting record's review. RS26 and RS27 are out of scope.
* `make changelog-check` on the tree: exit 0, 47 compared, 0 accepted, 0 with no released text (MS4).
* `tools/claims.py` passes the MADR, re-grounded at each phase's base, with its plants caught.

## Rollout and Rollback

* Rollout: with the next push. CI runs it in both workflows from that push; no workflow change.
* Rollback: revert `scripts/check_changelog.py` and its cases; the docs rows revert with them, and 0092-MADR ST27's note gets a dated reversal pointing back to E130now to E132now.
