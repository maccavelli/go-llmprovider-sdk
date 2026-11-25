---
status: accepted
date: 2026-11-25
decision-makers: repository owner
consulted: "the owner's answer of 2026-11-25 to 0092-MADR's Q-B2 and its order ruling, and the owner's answers of 2026-11-25 to this record's OQ1 to OQ3, each relayed by the coordinator and quoted below; a missing-case review of the first draft (2026-11-25, relayed by the coordinator); this repository at 78393b8, read only, and a scratch clone of it at the same commit for the measurements below (no file of the tree changed)"
informed: "0092-MADR, whose ST27 this gate checks from the day it lands; every later record whose changelog entry goes under `[Unreleased]`"
---

<!-- markdownlint-disable MD013 MD024 -->
<!-- claims: self=78393b8 -->

# Refuse an Edit to a Released CHANGELOG Section: `changelog-check` Compares Each Released Section With Its Text at Release

Associated PLAN: [0094-PLAN-refuse-an-edit-to-a-released-changelog-section.md](0094-PLAN-refuse-an-edit-to-a-released-changelog-section.md)

*(2026-11-25, first draft. The tree is clean at `78393b8` on `main`; `python3 -B scripts/check_records.py --next` printed `0094` there. One question was open (OQ1).)*

*(2026-11-25, revision 1, still a draft outside the tree: the owner answered OQ1 and two questions the coordinator put alongside it, OQ2 and OQ3, quoted in "Owner rulings"; a missing-case review returned five findings and eight false claims. Changes: RS13 decided (an accepted list of tag to reviewed commit, covering the body and the compare link); RS28 (release notes, an owner step), RS29 and RS30 (a `[YANKED]` heading) new; RS11 widened to a clone holding only some tags; RS33 new, the release window, and every case that tags or adds a section closes the window first; a section is carried by its version alone (RS2), and a released section with no baseline is a problem unless its tag is the newest (RS4); RS31 (a second compare link) and RS32 (a tag such as `v1.34.0-rc.1`) new; RS10's case dated so the date check passes; RS17's plants redesigned; RS18 gains a case; RS7's CRLF case dropped; the cost driver restated; D2 made to match the prototype, re-run; RS15's plant placed on RS1's case; D1 records where R53 cannot hold. 33 rows: 31 decided, 2 out of scope, 0 open.)*

*(2026-11-25, accepted. Owner ruling, quoted exactly, given for this MADR and its PLAN together: "Accept; approve; build now (Recommended)".)*

## Context and Problem Statement

[0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md](0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md) ST27 puts its change under `CHANGELOG.md`'s `[Unreleased]`, never into a released section. Its named check, `make changelog-check`, cannot catch that plant: `scripts/check_changelog.py` reads the section headings (their form, order, dates, and a tag with no section) and never a section's body (E1, E2). On a scratch clone, 0092's two **Fixed** entries moved under `## [v1.33.0]` passed it (E21). The owner ruled on 0092's Q-B2 for a gate, and for its order: 0092 first, this gate next (E19, E20). Until it lands, ST27 is checked by 0092's claims rows E130now to E132now, with a dated note that this gate replaces them (E3, E22).

### Owner rulings (2026-11-25, relayed by the coordinator; labels quoted exactly)

| Question | The owner's answer | Effect here |
| :--- | :--- | :--- |
| 0092-MADR Q-B2 | "Build a released-section gate" | this record |
| 0092-MADR Q-B2, the order | "Commit 0092 now, gate next (Recommended)" | 0092 committed at `78393b8`; this gate replaces ST27's claims rows (RS22) |
| OQ1, a needed fix to a released section | "Accepted list: tag → reviewed commit (Recommended)" | RS13: each entry maps a released tag to the commit whose text becomes that section's new baseline, body and compare link; each entry names its accepting record; a second edit fails until accepted again |
| OQ2, the GitHub Release of an accepted section | "Yes, as an owner step (Recommended)" | RS28: the accepting record lists `gh release edit` for each affected tag, run by the owner over the network; the gate does not |
| OQ3, a yanked release | "Allow a [YANKED] marker" | RS29, RS30: a released heading may gain a `[YANKED]` suffix, which is not an edit; the rest of the section stays frozen |

The owner's rule for this gate, as relayed: "cheap, no network, git reads only" (RS16).

### What `changelog-check` does today (read at 78393b8)

* It holds `CHANGELOG.md` to the `v*` tags and `authcontract/CHANGELOG.md` to the `authcontract/v*` tags, a section named for the version alone (E4, E5, E6). A tag counts only in the form `vX.Y.Z` (E45, E46).
* A heading must match `RELEASE`, `## [vX.Y.Z] - YYYY-MM-DD` exactly (E44). A section with no tag passes when it is newer than every tag: the release being prepared, between the date step and its tag (E7, E8).
* It runs one `git log` per tag for the dates (E9), and prints a count of sections and problems (E10). Measured: 0.77 s on this Mac (MS6).
* `--section TAG` prints a section's body for the release job's notes; that reader drops trailing lines that start with `[`, the compare links after the last section (E11). It is not a gate and this record does not change its output (RS19).
* Both workflows run it, with every tag checked out (`fetch-depth: 0`): `ci.yml` on code changes and every tag, `docs.yml` on a Markdown-only change, which `ci.yml` skips (E12 to E16). The self-test's workflow contract holds it in `MARKDOWN_GATES` (E17, E18).
* The self-test copies the tree into a shared clone, which carries the `v*` tags and drops the `authcontract/v*` ones (E23, E24), and tags a copy's commit with `tag_tree` when a case needs its own tag (E25).

### What a released section's text at its tag is (measured 2026-11-25, read only, at 78393b8)

Scratch scripts (`<scratch>/s0094/measure.py`, `<scratch>/s0094/prototype.py`) read each released section at HEAD and at the tags. The prototype was re-run after revision 1 with D2's body rule and the version-alone match (`<scratch>/s0094/proto-batch-2.txt`); the findings did not change.

| Id | Finding |
| :--- | :--- |
| MS1 | 47 release sections (46 root, 1 nested), each with its tag. 22 root sections and the nested one equal their text at their own tag. |
| MS2 | 23 root sections, v1.0.0 to v1.15.0, have no text at their own tag: `CHANGELOG.md` did not exist there. It was written in 0048-PLAN Phase 5 and first shipped at v1.16.0, whose file carries each of them, equal to HEAD's. |
| MS3 | v1.24.0's tag holds the file with that release's entries still under `[Unreleased]`, no `[v1.24.0]` section; v1.24.1's file carries it, equal to HEAD's. |
| MS4 | With D1's baseline, all 47 are compared, 0 differ, and each released tag's compare link equals its line at that baseline (46 compared). No file read, at HEAD or at any tag, holds a second compare link for a label. |
| MS5 | On a scratch clone, the comparison catches 0092 ST27's plant (`v1.33.0 differs from its text at v1.33.0`), an edit to v1.1.0 (named against v1.16.0) and an edited `[v1.32.0]` compare link; it passes trailing spaces on a released line, an edited `[Unreleased]` link, an entry added under `[Unreleased]`, and the date step to an untagged `[v1.34.0]` (`<scratch>/s0094/plants-2.txt`). |
| MS6 | Cost: today's `changelog-check` 0.77 s; the comparison of both files, reading every tag's file with one `git cat-file --batch` process, 0.07 s; with one `git show` per tag, 0.68 s. One existing self-test case (copy, plant, run) 2.1 s. |

So the premise relayed by the coordinator, "compare each section with `git show <tag>:CHANGELOG.md`", holds for 23 of 47 sections; the other 24 need a baseline at a later tag (RS2).

## Decision Drivers

* **ST27 checked by a gate**, not by claims rows pinned to line numbers that every release moves.
* **Released text does not change**: a release's notes are its section (R53, E26); an entry belongs under `[Unreleased]`.
* **No false failure on the release path**: the date step before the tag, the tag's own CI run, and the day after a release (0093-MADR's A3 lesson, E27, E28).
* **"cheap, no network, git reads only"**, the owner's rule as relayed. Today's check already runs one `git log` per tag (E9); the comparison adds two `git cat-file` processes per file, however many tags there are, and no fetch.
* **One place**: both workflows, the local gate and the self-test already run `changelog-check` (E12 to E18).

## Considered Options

* A. Compare each released section with its text at the first tag that carries it, inside `changelog-check`
* B. Compare only with the section's own tag
* C. Refuse a diff to a released section against the previous commit or the merge base
* D. Keep ST27's claims rows as the only check

## Decision Outcome

Chosen option: "A. Compare each released section with its text at the first tag that carries it, inside `changelog-check`", because it covers all 47 released sections today (MS4), needs only git reads of tags already checked out, passes the release path untouched, and adds no target, workflow line or contract entry.

### D1. A released section, and its baseline

A tag is a release only in the form `vX.Y.Z` (or `authcontract/vX.Y.Z` for the nested file); `v1.34.0-rc.1` is neither a release nor a baseline (RS32). A section `## [vX.Y.Z] - date` is released when its tag exists in the clone. A file at a tag carries a section when one of its level-two headings starts `## [vX.Y.Z]`, by the version alone, whatever its date or suffix (RS2). The baseline is that section's body, and the tag's compare-link line, in the file of the first tag, by version, at or after `vX.Y.Z` in the same namespace, that carries it: its own tag where the file had it, v1.16.0 for v1.0.0 to v1.15.0, and v1.24.1 for v1.24.0 (MS2, MS3). An `ACCEPTED` entry for the tag replaces that baseline with the file at its reviewed commit (RS13).

A released section that no file carries is a problem, unless its tag is the newest of its namespace: a tag made before its date step, which the next tag's file will carry (RS4). `[Unreleased]`, the preamble, and one section newer than every tag are never compared (RS5, RS20); two or more such sections mean the clone lacks tags (RS11).

R53 says a release's notes are its section. For v1.0.0 to v1.15.0 and v1.24.0 that cannot hold at their own tags, which carry no such section (MS2, MS3): the text this gate freezes for them is what v1.16.0 and v1.24.1 first shipped. R53's text says so (RS23).

### D2. What is compared

A section's body is its lines after its heading, to the next line that starts with `##` and a space (any level-two heading, `[Unreleased]` included); the last level-two section ends at the first compare-link line after its heading (`[Unreleased]: URL` or `[vX.Y.Z]: URL`), or the end of the file. Each line's trailing white space and the body's leading and trailing blank lines are dropped; nothing else is normalised: an interior blank line, an indent, a re-wrap and every character count (RS7, RS8). The working file is read with `read_text`, whose universal newlines turn CRLF into LF, and each tag's file is split with `splitlines`, so a CRLF line ending is no difference by the reading itself, not by this rule; no case covers it. The heading line is not compared; its date, form and order are the existing checks' (RS29). A released tag's compare-link line is compared with the baseline's, where the baseline file has one; a second definition of a released label is a problem (RS9, RS31). The prototype implements exactly this rule (`<scratch>/s0094/prototype.py`, `bodies`).

### D3. Where it runs

Inside `changelog-check`, by default, for both files: `make changelog-check` with no argument, so `ci.yml`, `docs.yml`, `tools/gate.py` and the self-test's clean run need no change, and `MARKDOWN_GATES` stays as it is (RS14). A separate target was weighed: it would need a `Makefile` target, a line in each workflow, a `MARKDOWN_GATES` entry and a `GATES` entry, for a check that reads the same file and the same tags.

### D4. Missing history is a tool error, never a pass

No `git fetch`, no network. A shallow clone, or a tag or accepted commit the clone does not hold, exits 2 naming the cause and that CI checks out with `fetch-depth: 0`. Two or more release sections untagged and newer than every tag of their namespace exit 2: at most one can be the release being prepared, so the clone lacks tags (none at all, or only some). A tag's file that is not UTF-8 exits 2 (RS11, RS12, RS21).

### D5. A yanked release

`RELEASE` gains an optional suffix, `## [vX.Y.Z] - YYYY-MM-DD [YANKED]`, its version and date groups unchanged, so the date check still compares the date with its tag's commit, the order check is unchanged, the summary counts the section, and `--section` finds it. The heading is not part of the body, so adding the marker, or removing it, is not an edit; the body and compare link stay frozen. A section with no tag may not carry the marker (RS29, RS30).

### D6. Output

A problem names the file (the nested one by its name, as today), the heading's line, the section, the baseline (tag, or accepted commit), and the first line that differs, by its line number in the working file; a link problem names the link's line and the baseline. An accepted section prints `accepted (<record>): <tag> at <commit>`. The summary line adds the count compared, the count accepted and the count with no released text. Exit 0 clean, 1 a problem, 2 a tool error (RS15).

### Consequences

* Good, because ST27, and the same rule for every later record, is checked by a gate that fails on its plant (MS5).
* Good, because the 24 sections whose own tag lacks them are guarded too (MS2, MS3).
* Good, because the release path passes: the date step (one untagged section), the tag's run (equal to itself) and later runs (RS5, RS6, RS33).
* Good, because a needed fix has a reviewed way through, a commit anyone can `git show`, named by a record (RS13).
* Neutral, because `changelog-check` rises by about 0.07 s (MS6), and `gate-selftest`, which `docs.yml` also runs, by about 2 s per new case, near 70 s for the cases below.
* Bad, because a clone without all its tags, which passes today, now exits 2 (RS11).
* Bad, because an accepted fix to a Release's notes is a separate owner step over the network (RS28).

### Confirmation

* The PLAN's self-test cases, one per decided row, each red at the base on its assertion or taking its red from its plant, and each plant caught on a scratch clone (`tools/plant.py`).
* The new cases pass on a scratch clone in the release window (date step committed, untagged) and on one where that step is tagged (RS17, RS33).
* 0092 ST27's plant, re-run at the build's base, is caught by `make changelog-check` (RS22).
* `tools/claims.py` passes this record's evidence index, its plants caught.

## Owner Questions (answered)

| Question | Options (recommended first, as put) | The owner's answer |
| :--- | :--- | :--- |
| OQ1. After this gate, any edit to a past release's notes fails. Should a needed fix (a typo, a link to a moved record, a home path `disclosure-check` finds) have a way through? | (1) a recorded exception list, as `api-check`'s `ACCEPTED` is; (2) no way through; (3) dated notes only | "Accepted list: tag → reviewed commit (Recommended)": (1), each entry a tag and a reviewed commit (RS13) |
| OQ2. Does an accepted fix also update that tag's GitHub Release? | put by the coordinator | "Yes, as an owner step (Recommended)" (RS28) |
| OQ3. May a released section be marked yanked? | put by the coordinator | "Allow a [YANKED] marker" (RS29, RS30) |

## State Table

Each decided row names its self-test case in `scripts/test_gates.py` and its plant, one production change to `scripts/check_changelog.py` on a scratch clone unless the row says otherwise. Cases in `GateSelfTest` run on a copy of the tree (E23); cases in a new `ChangelogReleasedTest` class load `scripts/check_changelog.py` by path and read the tree only. Two new helpers: `tag_dated(tree, tag, date)`, `tag_tree` with the commit's author and committer dates set to `date`, so a heading can be dated before its tag exists; and `close_window(tree)`, which tags the copy's first release section at a commit dated by its heading when that section has no tag (RS33).

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| RS1 | An entry added to a released section (v1.14.0's body) | decided | exit 1: `line L: v1.14.0's section differs from its text at v1.14.0, first at line N`, L the heading's line and N the first line that differs | D1, D2, D6 | `test_changelog_check_released_entry_added`; plants: the comparison skipped; the baseline tag left out of the message (RS15) |
| RS2 | A released section whose own tag lacks it (v1.1.0: no file; v1.24.0: the file, no section); and a baseline file whose heading for the version is not in `RELEASE`'s form | decided | compared with the first later file that carries it, by version alone; an edit fails naming v1.16.0, v1.24.1, or the tag whose heading was malformed | D1; MS2, MS3 | `test_changelog_check_released_backfilled_section` (v1.1.0 edited, names v1.16.0); `test_changelog_check_released_late_section` (v1.24.0 edited, names v1.24.1); `test_changelog_check_released_carried_by_version`: after `close_window`, `v99.0.0` tagged with `tag_dated` at a file whose heading is `## [v99.0.0]`, no date; the working heading dated, its body edited; exit 1 naming `v99.0.0`. Plants: the baseline read only at the section's own tag (the first two pass); "carries" matched with the full `RELEASE` (the third: no baseline, newest tag, passes) |
| RS3 | 0092 ST27's shape: an entry put into the newest tagged section, found at run time | decided | exit 1, as RS1 | D1 | `test_changelog_check_released_newest_tagged`: the first release section whose tag exists gains an entry; plant: the first release section never compared |
| RS4 | A released section that no file carries: (a) its tag the newest; (b) an older tag | decided | (a) not compared, exit 0, counted as having no released text; (b) exit 1, `v99.0.0's section has no released text: no tag's file carries it` | D1 | `test_changelog_check_released_no_baseline`: after `close_window`, `v99.0.0` tagged (`tag_dated`, 2099-01-01) at a file without its section, then `[v99.0.0] - 2099-01-01` added: (a) exit 0, summary `1 with no released text`; then `[v99.0.1] - 2099-01-02` added and `[v99.0.0]` taken out, that file tagged `v99.0.1`, and `[v99.0.0]` put back: (b) exit 1. Plants: the exception taken off ((a) exits 1); the exception given to every tag ((b) passes) |
| RS5 | `[Unreleased]` changed; and one section newer than every tag, untagged | decided | neither compared; exit 0. A tag made locally and not yet pushed is compared locally, and in CI from its push on | D1; E7, E8 | `test_changelog_check_unreleased_and_release_commit_pass`, after `close_window`: (a) an entry under `[Unreleased]`; (b) one untagged `[v99.0.0] - 2099-01-01` section with an entry; each exit 0; plants: `[Unreleased]` compared with its text at the newest tag ((a)); an untagged section read as differing ((b)) |
| RS6 | After the tag | decided | the section equals its own text: exit 0; an edit after it: exit 1 naming the tag | D1 | `test_changelog_check_released_after_its_tag`: after `close_window`, `[v99.0.0] - 2099-01-01` added and tagged with `tag_dated`; exit 0; then an entry added, exit 1; plant: the baseline searched among tags below the section's version |
| RS7 | White space: trailing spaces on a released line; a blank line added at a released section's end | decided | equal: exit 0. CRLF is not a case: the reading itself turns it to LF (D2) | D2 | `test_changelog_check_released_whitespace_passes`; plant: lines compared unnormalised (exit 1) |
| RS8 | Two lines of a v1.14.0 entry joined into one | decided | differs: exit 1 | D2 | `test_changelog_check_released_rewrap`; plant: bodies compared with all white space collapsed |
| RS9 | The compare links: (a) a released tag's link changed or removed; (b) the last section, v1.0.0, edited; (c) the `[Unreleased]` link changed and a new tag's link added | decided | (a) exit 1, `line N: v1.14.0's compare link differs from its line at v1.14.0`; (b) exit 1 naming v1.16.0; (c) exit 0. A baseline file with no link line for the tag compares none | D2 | `test_changelog_check_released_link_changed`, `test_changelog_check_released_last_section`, `test_changelog_check_unreleased_link_passes`; plants: link lines not compared ((a)); the last section never compared ((b)); the link block read as part of the last section ((c)) |
| RS10 | The nested module | decided | compared by `authcontract/v*` tags only, the problem naming the file and the full tag; with no such tag (the self-test's copy, E24) its one section is not compared | D1; E5, E6 | `test_changelog_check_released_contract_section`: `authcontract/v0.1.0` tagged with `tag_dated` at 2026-11-12, its section's date, so the date check passes; the body edited; exit 1, `differs` naming `authcontract/CHANGELOG.md` and `authcontract/v0.1.0`, no `is dated` line; `test_changelog_check_contract_untagged_not_compared`: the body edited, no tag, exit 0. Plants: the nested file not compared (the first); RS11's guard counting one untagged section (the second exits 2) |
| RS11 | Two or more release sections untagged and newer than every tag of their namespace: (a) no tags; (b) the tags from v1.20.0 on deleted; (c) two untagged sections above a closed window | decided | exit 2: `changelog-check: CHANGELOG.md: N release sections are newer than every v* tag; one can be the release being prepared; this clone lacks tags (CI checks out with fetch-depth: 0)` | D4; the review's finding 1 | `test_changelog_check_missing_tags_is_a_tool_error`, three sub-cases on fresh copies; plants: the guard removed ((a), (b) exit 0); the guard's bound off by one, three or more ((c) exits 0) |
| RS12 | Missing history: (a) a shallow clone; (b) a listed tag whose commit the clone does not hold | decided | exit 2, naming the cause | D4 | (a) `test_changelog_check_shallow_is_a_tool_error`: the copy's HEAD written to its `.git/shallow`; (b) `ChangelogReleasedTest.test_missing_tag_commit_is_a_tool_error`: the reader given a tag with no object; plants: the shallow check removed; a missing commit read as a tag with no file |
| RS13 | A needed fix: `ACCEPTED`, `{full tag: (commit, record)}`, empty at build | decided | the commit's file at that tag's version is the section's baseline, body and compare link; `accepted (<record>): <tag> at <commit>`, exit 0. A further edit: exit 1 against the commit. The commit's file not carrying the section: exit 1. The commit absent: exit 2. An entry whose commit is not a full hex object name, or whose record is not `NNNN-MADR …`: exit 2 | OQ1 "Accepted list: tag → reviewed commit (Recommended)" | `test_changelog_check_released_accepted_edit`: (a) v1.14.0's body and compare link edited and committed in the copy with `git commit-tree`, no tag, the copy's `ACCEPTED` given `{"v1.14.0": (<commit>, "0094-MADR plant")}`, exit 0 with the accepted line; (b) one more edit, exit 1 naming the commit; (c) an entry naming a commit whose file has no `[v1.14.0]`, exit 1; (d) an absent commit, exit 2; (e) `main` as the commit, and `""` as the record, each exit 2. Plants: `ACCEPTED` ignored ((a)); the link compared with the tag's line ((a)); any text passing for a listed tag ((b)); a carrier-less commit falling back to the tag ((c)'s message); an absent commit read as no file ((d)); the entry's form not checked ((e)) |
| RS14 | Where it runs | decided | inside `changelog-check`, by default; `Makefile`, both workflows, `tools/gate.py`, `GATES` and `MARKDOWN_GATES` unchanged | D3; E12 to E18 | `WorkflowContractTest` and `test_clean_copy_passes_every_gate` unchanged, green with the comparison on (MS4); RS1's case runs the gate with no argument, as `make` does; no plant: no target, workflow or contract change |
| RS15 | Output and exit codes | decided | as D6 | D6 | assertions in RS1's case (the heading's line, `at v1.14.0`, `first at line`), RS4's (the summary), RS13's (the accepted line, the summary's accepted count), RS11's and RS12's (exit 2); its plant is on RS1's case: the baseline tag left out of the message |
| RS16 | Cost | decided | per file: today's `git tag --list` and one `git log` per tag, unchanged; added, one `git cat-file --batch-check` for the tags' and accepted commits, one `git cat-file --batch` for their files; no `git show` per tag, no fetch, no network | D4; MS6; "cheap, no network, git reads only" | `ChangelogReleasedTest.test_git_calls_per_file`: on the tree, `subprocess.run` counted: two `cat-file` calls per file, no `show`, no `fetch`; plant: the files read with one `git show` per tag |
| RS17 | The self-test does not depend on which tag is latest (0093-MADR A3, E27, E28) | decided | every case anchors on fixed old tags (v1.0.0, v1.1.0, v1.14.0, v1.16.0, v1.24.0, v1.24.1), finds the newest tagged section at run time (RS3), or calls `close_window` before it tags or adds a section; none names the newest tag or reads `[Unreleased]`'s entries | the 0093 lesson | the PLAN runs every changelog case on two scratch clones, one in the release window (the next release's date step committed, untagged) and one where that step is tagged with `[Unreleased]` empty; all pass on both. Plants, on the test file: RS3's case taking the first release section without checking its tag (fails on the window clone: the entry lands in the untagged section and passes); `close_window` made a no-op (RS4's, RS6's and RS21's cases fail on the window clone: the untagged `[v<next>]` is older than `v99.0.0`) |
| RS18 | A duplicated section: v1.14.0's section repeated below itself, the copy's body edited | decided | the existing problem, `v1.14.0 has a section already`; the comparison reads the first occurrence only, so no `differs` line | E7; 0048-MADR D5 | `test_changelog_check_released_duplicate_section`, and the six existing heading cases unchanged (green pins); plant: the last occurrence compared (a `differs` line appears) |
| RS19 | `--section TAG` | decided | its output unchanged; it finds a `[YANKED]` heading through `RELEASE` (D5) | E11 | the four existing `test_changelog_section_*` cases unchanged (green pins), and RS29's case (d); plant: as RS29's first |
| RS20 | The preamble above `[Unreleased]` | decided | not compared | D1 | `test_changelog_check_unreleased_and_release_commit_pass` case (c): a line added to the preamble, exit 0; plant: the preamble compared with the newest tag's |
| RS21 | A tag's file that is not UTF-8 | decided | exit 2 naming the tag | D4 | `test_changelog_check_released_tag_not_utf8`: after `close_window`, a commit whose file holds an invalid byte tagged `v99.0.0` (`tag_dated`), the working file then valid with `[v99.0.0] - 2099-01-01`; plant: decoded with `errors="replace"` |
| RS22 | 0092-MADR ST27's check | decided | from this gate's commit, ST27's check is `make changelog-check`; E130now to E132now stay as evidence; dated notes on 0092-MADR's ST27 row, an amendment there, and 0092-PLAN's ST27 entry (E3, E22) | the owner's order ruling; E19 to E22 | 0092 ST27's plant re-run at the build's base: the first `[Unreleased]` entry moved into the newest tagged section (or, with `[Unreleased]` empty, an entry planted there), caught with RS1's message |
| RS23 | Docs | decided | `scripts/check_changelog.py`'s docstring; the `Makefile`'s comment and help line (E29, E30); `docs/architecture.md`'s scripts table, a `changelog-check` item in its Tooling list, and its `gate-selftest` item (E31, E32, E33); R53 and its table row (E26, E34), R53 also saying a released section is frozen, that the baseline for v1.0.0 to v1.15.0 and v1.24.0 is a later file, that `[YANKED]` is allowed, and that a fix goes through `ACCEPTED`; `AGENTS.md`'s CI paragraph (E35); `CHANGELOG.md`'s preamble sentence (E37). `tools/README.md` (E36) and `README.md` name the check or the file, not its rules, and stay | the records rule | a claims row per changed doc line in the PLAN's execution record; `make markdownlint links-check`; plant: the architecture's scripts-table line left as today (its row fails) |
| RS24 | The changelog entry | decided | a **Changed** entry under `[Unreleased]`: CI's `changelog-check` fails an edit to a released section; the module's code is unchanged (as 0063's and 0052's CI entries, E38) | 0048-MADR D5 | `make changelog-check` on the tree passes with it; its claims row; plant: the entry put into the newest tagged section |
| RS25 | The exported API | decided | nothing: scripts and docs only | R48 | `make api-check`; no plant: no Go code changes |
| RS26 | A tag moved or re-made at another commit | out of scope | tags are not re-made here; a moved tag's file is the baseline it then holds | none | none: out of scope |
| RS27 | Other repositories' changelogs | out of scope | this record is the SDK's gate | none | none: out of scope |
| RS28 | The GitHub Release of an accepted section | decided | the accepting record's PLAN lists `gh release edit <tag> --notes-file <the section at the accepted commit>` for each affected tag that still has a Release, an owner step over the network; the gate neither calls nor checks it | OQ2 "Yes, as an owner step (Recommended)" | no gate case: the accepting record's review checks the step is listed; no plant: no code |
| RS29 | A released heading marked `[YANKED]`: (a) the marker added to v1.14.0; (b) with the wrong date; (c) with its body edited; (d) `--section v1.14.0` | decided | (a) exit 0; (b) exit 1, the existing `is dated` problem; (c) exit 1, `differs`; (d) the body printed | OQ3 "Allow a [YANKED] marker"; D5 | `test_changelog_check_released_yanked`; plants: `RELEASE` unchanged ((a) fails on the heading's form, (d) finds no section); the date check skipped for a yanked heading ((b)); a yanked section not compared ((c)) |
| RS30 | `[YANKED]` on a section with no tag | decided | exit 1, `v99.0.0 is marked [YANKED] and has no tag` | D5 | `test_changelog_check_yanked_untagged`: after `close_window`, `[v99.0.0] - 2099-01-01 [YANKED]` added; plant: the check removed |
| RS31 | A second compare-link definition for a released label (an identical `[v1.14.0]: …` line added) | decided | exit 1, `line N: a second compare link for v1.14.0` | D2; the review's finding 4 | `test_changelog_check_released_link_twice`; plant: link lines read last-wins with no duplicate check |
| RS32 | A tag not of the form `vX.Y.Z` (`v99.0.0-rc.1`) | decided | neither a release nor a baseline: an untagged `[v99.0.0]` above a closed window stays uncompared, exit 0 | D1; E45, E46 | `test_changelog_check_prerelease_tag_ignored`: after `close_window`, `[v99.0.0] - 2099-01-01` with body X committed and tagged `v99.0.0-rc.1`, then body Y; exit 0. Plants: `VERSION` loosened to allow a suffix (the rc tag has no section, exit 1); rc tags taken as baselines for their version (Y differs, exit 1) |
| RS33 | The release window: the date step committed, its tag not yet made, in CI or locally | decided | the one untagged newest section passes (RS5); every case that tags `v99.0.0` or adds a section first calls `close_window`, so it holds in the window (the review's finding 2) | D1; E7 | RS17's window clone: every case passes there, and `make changelog-check` exits 0; plant: as RS17's second |

Counts: 33 rows: 31 decided, 2 out of scope (RS26, RS27), 0 open.

## Pros and Cons of the Options

### A. Compare each released section with its text at the first tag that carries it, inside `changelog-check`

* Good, because it covers all 47 released sections and catches ST27's plant (MS4, MS5).
* Good, because it reads only tags the workflows already fetch, with two added git processes per file (MS6).
* Bad, because a clone without all its tags now exits 2 (RS11).

### B. Compare only with the section's own tag

* Good, because it is the simplest rule.
* Bad, because 24 of 47 sections, v1.0.0 to v1.15.0 and v1.24.0, have no text at their own tag and go unchecked (MS2, MS3).

### C. Refuse a diff to a released section against the previous commit or the merge base

* Good, because it needs no tag reads.
* Bad, because what it compares depends on the push: a push of several commits, a rebase, or a local run on a dirty tree each have another base; CI's push event names no stable base without the API, which the gate does not call.
* Bad, because an edit is judged by what came before it, not by what was released.

### D. Keep ST27's claims rows as the only check

* Good, because nothing changes.
* Bad, because the rows are pinned to line numbers every release moves, guard only 0092's entry, and are run by hand, not by CI: the owner's ruling chose the gate (E19).

## More Information

### Relation to other records

* **0092-MADR** ST27, Q-B2 and its fifth amendment: this gate is ST27's check from its commit (RS22).
* **0048-MADR** D5: `changelog-check` and the release job; this record adds to its gate, widens `RELEASE` for `[YANKED]`, and leaves `--section`'s output as it is.
* **0093-MADR** A3: the self-test lesson (RS17, RS33).
* **0052-MADR**, **0063-MADR**: the workflows and the Markdown gates' contract, unchanged (RS14).

### What could not be established

* That `actions/checkout` at its pinned commit fetches every tag on a pull request from a fork: the workflows' comments say every tag is checked out (E14, E16), and `api-check` relies on it in CI today; no fork pull request was observed.

### Evidence index

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E1 | what `changelog-check` checks | `scripts/check_changelog.py:2` | `"""changelog-check: CHANGELOG.md has a section for every release tag.` |
| E2 | its reader reads the level-two headings only | `scripts/check_changelog.py:77` | `if not line.startswith("## "):` |
| E3 | 0092's ST27, its check until 0094 lands | `docs/decisions/0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md:354` | `0094, a gate that compares each released section with its tag's text, replaces that check when it lands.` |
| E4 | the tags of a namespace | `scripts/check_changelog.py:96` | `tags = [t.removeprefix(prefix) for t in git("tag", "--list", f"{prefix}v*").split()` |
| E5 | the nested file and its prefix | `scripts/check_changelog.py:49` | `NESTED = (ROOT / "authcontract/CHANGELOG.md", "authcontract/")` |
| E6 | the nested file's one release section | `authcontract/CHANGELOG.md:12` | `## [v0.1.0] - 2026-11-12` |
| E7 | an untagged section newer than every tag passes | `scripts/check_changelog.py:106` | `if newest is not None and version(tag) <= version(newest):` |
| E8 | the docstring: the release being prepared | `scripts/check_changelog.py:14` | `being prepared, which its tag will name;` |
| E9 | one `git log` per tag | `scripts/check_changelog.py:109` | `want = git("log", "-1", "--format=%cs", prefix + tag).strip()` |
| E10 | the summary line | `scripts/check_changelog.py:190` | `print(f"changelog-check: {sum(1 for line in lines if RELEASE.match(line))} release section(s), "` |
| E11 | `--section` drops trailing `[` lines | `scripts/check_changelog.py:128` | `while body and (not body[-1].strip() or body[-1].startswith("[")):` |
| E12 | `ci.yml` runs it | `.github/workflows/ci.yml:75` | `run: make parity-check dep-check coverage-check api-check contract-check generate-check records-check tables-check fixtures-check changelog-check docs-check markdownlint links-check disclosure-check gate-selftest` |
| E13 | `ci.yml` skips a Markdown-only change | `.github/workflows/ci.yml:12` | `paths-ignore: ['**.md']` |
| E14 | `ci.yml` checks out every tag | `.github/workflows/ci.yml:41` | `fetch-depth: 0` |
| E15 | `docs.yml` runs it | `.github/workflows/docs.yml:47` | `run: make parity-check records-check tables-check changelog-check markdownlint links-check disclosure-check gate-selftest` |
| E16 | `docs.yml` checks out every tag | `.github/workflows/docs.yml:33` | `fetch-depth: 0` |
| E17 | the contract's Markdown gates | `scripts/test_gates.py:913` | `MARKDOWN_GATES = ("parity-check", "records-check", "tables-check", "changelog-check", "markdownlint", "links-check",` |
| E18 | the local gate reads that tuple | `tools/gate.py:48` | `from test_gates import MARKDOWN_GATES` |
| E19 | the owner's Q-B2 answer | `docs/decisions/0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md:619` | `\| Q-B2 \| "Build a released-section gate" \|` |
| E20 | the order ruling | `docs/decisions/0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md:620` | `\| Q-B2, the order \| "Commit 0092 now, gate next (Recommended)" \|` |
| E21 | ST27's plant passes `changelog-check` | `docs/decisions/0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md:599` | `` `make changelog-check` exits 0 (`47 release section(s), 0 problem(s)`) `` |
| E22 | 0092-PLAN's ST27 entry | `docs/decisions/0092-PLAN-fill-a-completed-responses-stream-from-its-terminal-output.md:206` | `0094's released-section gate replaces that check when it lands.` |
| E23 | each self-test case copies the tree | `scripts/test_gates.py:186` | `copy_tree(self.tree)` |
| E24 | the copy drops the `authcontract/v*` tags | `scripts/test_gates.py:73` | `tags = subprocess.run(["git", "tag", "-l", "authcontract/v*"], cwd=dest, capture_output=True, text=True,` |
| E25 | a case can tag its copy | `scripts/test_gates.py:652` | `def tag_tree(tree: Path, tag: str) -> None:` |
| E26 | R53 | `docs/guides/api-standards.md:409` | `- **R53. Every release has a changelog section, and the six newest tags a` |
| E27 | 0093's A3: a test that compared with the latest tag | `docs/decisions/0093-MADR-take-model-metadata-from-the-caller.md:45` | `which kept the tag's release job from running.` |
| E28 | its fix names the base | `scripts/test_gates.py:336` | `proc = run_gate(self.tree, "api-check", "--against", "v1.32.0")` |
| E29 | the `Makefile`'s comment | `Makefile:105` | `# 0048-PLAN Phase 5: CHANGELOG.md has a section for every release tag (0048-MADR D5).` |
| E30 | its help line | `Makefile:106` | `changelog-check: ## Checks CHANGELOG.md against the release tags` |
| E31 | the architecture's scripts table | `docs/architecture.md:42` | `scripts/check_changelog.py  changelog-check: CHANGELOG.md has a section for every release tag` |
| E32 | the architecture's `gate-selftest` item | `docs/architecture.md:1629` | `tables-check, fixtures-check, docs-check, changelog-check, links-check,` |
| E33 | the architecture's CI item: the full history for the tags | `docs/architecture.md:1693` | `` It checks out the full history, so that `api-check` and `changelog-check` `` |
| E34 | R53's table row | `docs/guides/api-standards.md:471` | `` \| R53 \| `make changelog-check` \| 0048-PLAN Phase 5 \| `` |
| E35 | `AGENTS.md`'s CI paragraph | `AGENTS.md:163` | `changelog-check docs-check markdownlint links-check disclosure-check` |
| E36 | `tools/README.md`'s check list | `tools/README.md:63` | `` `tables-check`, `fixtures-check`, `changelog-check`, `docs-check` and `` |
| E37 | the changelog's preamble on the gate | `CHANGELOG.md:7` | `` `make api-check` enforces (the standards guide's R48). `make changelog-check` `` |
| E38 | a CI change's changelog entry | `CHANGELOG.md:928` | `` - Both CI workflows run three more documentation gates: `markdownlint`, `` |
| E39 | `api-check`'s exception list | `scripts/check_api.py:60` | `ACCEPTED: dict[str, dict[str, str]] = {` |
| E40 | the compare links begin | `CHANGELOG.md:1904` | `[Unreleased]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.33.0...HEAD` |
| E41 | the last section | `CHANGELOG.md:1890` | `## [v1.0.0] - 2026-10-02` |
| E42 | v1.24.0's section | `CHANGELOG.md:989` | `## [v1.24.0] - 2026-10-27` |
| E43 | v1.14.0's section, the existing cases' anchor | `CHANGELOG.md:1515` | `## [v1.14.0] - 2026-10-16` |
| E44 | `RELEASE`, the heading's one form | `scripts/check_changelog.py:51` | `RELEASE = re.compile(r"^## \[(v(\d+)\.(\d+)\.(\d+))\] - (\d{4}-\d{2}-\d{2})$")` |
| E45 | `VERSION`, a tag's form | `scripts/check_changelog.py:52` | `VERSION = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")` |
| E46 | only tags of that form count | `scripts/check_changelog.py:97` | `if t.startswith(prefix) and VERSION.match(t.removeprefix(prefix))]` |
