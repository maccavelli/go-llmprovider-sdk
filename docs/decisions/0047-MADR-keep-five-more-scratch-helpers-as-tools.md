---
status: accepted
date: 2026-10-19
decision-makers: repository owner
consulted: docs/decisions/0035-MADR-maintainer-tools-in-python.md; a read of the 103 helper scripts left in the agent's session scratchpad from the work on records 0017 to 0046, on 2026-10-19; a count of account-data-shaped strings in the committed test data at ab704db
informed: maintainers and agents working in this repository
---

<!-- markdownlint-disable MD013 -->

# Keep Five More Scratch Helpers: a Fixture Gate, Appends for `edit.py` and `plant.py`, Record Status, Record Lint, and Claims Plants

## Context and Problem Statement

[0035-MADR](0035-MADR-maintainer-tools-in-python.md) moved the helpers that
had lived in a session scratchpad into `tools/`, because the scratchpad is
pruned and a lost helper could not be rebuilt without its transcript. The
work on records 0017 to 0046 left 103 more scratch scripts.
On 2026-10-19 each one was read in full and sorted **[read]**:

| Class | Scripts | What they are |
| :--- | ---: | :--- |
| A. Assert-once replace, sometimes with an append, on records and docs | 29 | the `sub` / `apply(path, edits, append=None)` helper, copied |
| B. The same helper, on Go, Python, the Makefile or CI | 22 | as A, on code; some create a file |
| C. A phase's execution record, appended | 21 | as A, with an "already there?" guard |
| D. A record's status, and `docs/README.md`'s row and count | 13, inside A and C | a status edit plus two index edits |
| E. Plant-spec generators | 6 | Python lists written as JSON for `plant.py` |
| F. Scratch-clone experiments | 5 | change, run, report |
| Candidates, five already covered by a tool | 19 | probes and checks |
| One-off | 1 | |

The classes, and what each needs that no tool does:

1. **`tools/edit.py` cannot append or read its text from a file.** Its spec
   is a list of `{"file", "old", "new"}` objects (`tools/edit.py:5`), and
   `plan` only replaces (`tools/edit.py:27-42`) **[read]**. So 72 scripts
   (A, B, C) carried their own copy of the same helper. They needed two
   things `edit.py` lacks: an append to a file's end, for every execution
   record; and multi-line Go and Markdown without JSON escaping, which is
   also why the six plant generators (E) exist.
2. **No tool keeps a record's status and the index in step.**
   `scripts/check_records.py` checks the index's rows, statuses and its
   "N records." line (`scripts/check_records.py:45-47`, `:129-165`), and
   changes nothing **[read]**. Thirteen scripts made the same three edits
   by hand: the front matter, the row, and the count.
3. **No tool scans a recorded fixture for account data.** `denyscan.py`
   finds this machine's account, its host and a real home path
   (`tools/denyscan.py:2`) **[read]**. A listing or a stream recorded from
   a live service can hold other people's data, or the owner's account
   ids: an e-mail address, a UUID, an `org-…` / `user-…` id, a JWT. Two
   scratch scripts looked for those before 0041's listings were committed
   (`p8/inspect_chatgpt.py`, `p8/idhits.py`). A count over the 250
   committed `testdata` files at `ab704db`, by kind, never by value
   **[probed]**:

   | File | UUID-shaped | Account-id-shaped |
   | :--- | ---: | ---: |
   | `llmprovider/catalog/testdata/ranking-2026-10-12/together.json` | 134 | 15 |
   | `llmprovider/catalog/testdata/rawlistings-2026-10-15/chatgpt.json` | 0 | 21 |
   | `llmprovider/providers/openai/testdata/chatgpt-reasoning.sse` | 3 | 3 |
   | `llmprovider/providers/openai/testdata/chatgpt-text.sse` | 3 | 3 |
   | `llmprovider/providers/openai/testdata/chatgpt-tool.sse` | 3 | 3 |

   No e-mail address, JWT or home path was found. Each of the 188 was
   then placed by its context, never its value: the JSON key or stream
   field it sits under, its prefix, its length and its character variety,
   with counts of exact placeholder strings **[probed]**:

   | Hits | Where | What they are |
   | ---: | :--- | :--- |
   | 18 | the three `chatgpt-*.sse` files: `response.prompt_cache_key`, `response.safety_identifier` | placeholders: every UUID is the nil UUID, every id is `user-REDACTED`, 3 of 3 in each file; the capture "redacted account identifiers" (`0012-PLAN-chatgpt-backend.md:437`) |
   | 36 | `together.json`'s `config.chat_template`; `chatgpt.json`'s `model_messages` policy text | words in prose (`user_…`): no match holds a digit or a capital letter |
   | 134 | `together.json`'s `uuid`, in model entries | Together's model ids: each entry's `id` is a public `owner/name` path, of 33 owners (`Qwen`, `google`, `meta-llama`, …), from its public listing (`e2a0f1e`) |

   So none of the 188 is account data **[derived]**. The 54 placeholder and
   prose hits are what a loose pattern finds.
4. **Nothing lints the records.** The gate runs markdownlint on every
   Markdown file but the records (`tools/gate.py:58-60`), whose config
   excludes them **[read]**. Many records carry findings that predate them,
   so a plain run is noise; `issues/lintdiff.py` compared the counts per
   rule at `HEAD` and in the tree, so that only an edit's new findings
   showed.
5. **Claims plants cannot run through `plant.py`.** `claims.py` finds
   another repository beside this one or beside its parent
   (`tools/claims.py:91-102`) **[read]**. `plant.py` runs on a scratch
   clone (`tools/_common.py:55`), where neither sibling exists; naming them
   with `--repo` would put a machine path in the spec. So 0046's plants of
   false claims ran from `issues/plant46.py` and `plantplan46.py`, on a
   temporary copy of the record, in the working checkout.

## Decision Drivers

* A helper that two plans needed is a tool, or it is lost when the
  scratchpad is pruned (0035-MADR).
* 0035-MADR's rules hold for every tool: Python 3 on the standard library;
  the root found from the tool's own file; no host, account or machine path;
  exit codes 0 clean, 1 a finding, 2 a tool error; a test in
  `tools/test_tools.py` that fails it on a planted input.
* A scan for account data must never print, log or store the value it
  found: the organization's rules mask RESTRICTED data in fixtures and logs,
  and an agent's tool output reaches a model.
* One tool per job. A change to an existing tool keeps every spec and
  command line that works today working.

## Considered Options

* **A. Three new tools and two extensions:** `fixturescan.py`,
  `records.py` and `recordlint.py`; appends and file text in `edit.py`;
  plants in `claims.py`.
* **B. Leave them in the scratchpad.**
* **C. One multi-command records tool** for D2–D5.
* **D. As A, with the fixture scan as a CI gate in `scripts/`.**

## Decision Outcome

Chosen option: "D", because each job gets one tool under 0035-MADR's
rules, the extensions remove the reason 72 scripts copied a helper, and the
fixture check, whose first findings have been reviewed, is a gate from the
start, so a fixture holding account data cannot be merged.

### D1. `scripts/check_fixtures.py`: a CI gate for account data in recorded fixtures

* **Kinds:** an e-mail address; a UUID; an account-, user- or
  organization-id (`user-…`, `org-…`, `acct-…`, `account-…`, eight or more
  characters after the prefix, at least one a digit or a capital letter);
  a JWT (`eyJ…` with two dot-separated parts); a home path
  (`/Users/<name>`, `/home/<name>`).
* **Never a hit:** a placeholder (`REDACTED`, a run of `X`, text in angle
  brackets such as `<user>`); the nil UUID; an `example.com`,
  `example.org`, `.test`, `.invalid` or `.example` address.
* **What it scans:** every tracked file under a `testdata` directory, or
  the files named.
* **What it prints:** `path:line:column kind fingerprint`, where the
  fingerprint is the first 12 hex digits of the value's SHA-256. Never the
  value, nor any text around it.
* **The baseline,** `scripts/fixtures-baseline.txt`: one `path kind
  fingerprint` per line, sorted, under a header that names the review that
  admitted them. A hit in the baseline passes; any other is a finding, exit
  1. A baseline line no hit matches is a finding too, so the file never
  holds stale entries. Its first content is the 134 Together model ids of
  the review above *(A1: 133 lines, one per distinct id)*.
* **`--write-baseline`** writes the current hits' lines and prints only
  their count: a person's step, after a review like the one above.
* **CI** runs it as `make fixtures-check`, in the list beside
  `tables-check`. `scripts/test_gates.py` plants a fixture with an address,
  an id and a JWT, and `gate-selftest` requires each to be caught, and the
  clean copy to pass. The local gate runs it too.
* **Recorded fixtures are redacted before they are committed,** and
  `AGENTS.md` says how, as the 2026-09-27 ChatGPT captures were: an
  account id becomes its prefix and `REDACTED` (`user-REDACTED`), a UUID
  the nil UUID, an address `user@example.com`.

### D2. `tools/edit.py`: appends, new files, and text from a file

* An entry may be `{"file", "append"}`: the text is added at the file's end,
  after one newline if the file does not end with one.
* An entry may be `{"file", "create"}`: a new file with that text. It fails
  if the file exists.
* Any text field may instead name a file, as `"old_file"`, `"new_file"`,
  `"append_file"` or `"create_file"`, read relative to the spec's own
  directory. So multi-line Go and Markdown need no escaping.
* As now, every entry is checked before any file is written; a spec that
  fails changes nothing. Every spec valid today means what it means today.
* `tools/plant.py`'s plants read `"old_file"` and `"new_file"` the same
  way, so a plant of multi-line code needs no generator.

### D3. `tools/records.py`: a record's status, and the index

* `records.py add RECORD...` adds each record's row to `docs/README.md`'s
  Records table, in number and kind order: its number, kind, a link titled
  with its first heading, and its status. It updates "N records.".
* `records.py status RECORD STATUS [--date YYYY-MM-DD]` sets the front
  matter's `status`, and its `date` (today by default), and the row's
  status.
* Statuses are checked against `scripts/check_records.py`'s lists, which
  it imports, so the two never disagree. After writing, it runs
  `check_records.py`'s checks, and exits 1 on any problem.
* It never touches the "I want to…" table, which a person writes.

### D4. `tools/recordlint.py`: an edit's new markdownlint findings

* For each record named, or `--changed`: markdownlint-cli2 on the file at
  `HEAD` (from `git show`, into a temporary directory) and in the tree.
* It prints the count per rule for both, and each finding of a rule whose
  count rose, with its line. Exit 1 when any count rose; a new record is
  compared with no findings.
* It respects a record's own `markdownlint-disable` comments, as
  markdownlint does.
* *(A2, 2026-10-19)* Both versions are linted from copies in a temporary
  directory, with the repository's rules and two for the records' own
  conventions: MD004 `consistent` and MD010 `code_blocks: false`.

### D5. `tools/claims.py --plants SPEC.json`: plants of false claims

* The spec is `{"record", "plants": [{"name", "old", "new"}]}`. Each plant
  is applied, once, to a copy of the record in a temporary directory, and
  the copy is checked as the record would be, with the same `--repo`,
  `--rev` and `--search`.
* Each plant must fail. A plant that passes, or whose `old` does not match
  exactly once, is a finding, exit 1. The tree's record is not touched.

### D6. The documents

* `tools/README.md`'s table gains the two tools and the three extensions,
  and its gate section the fixture check.
* `AGENTS.md`'s "Tools" list names them; its CI list names `fixtures-check`;
  a paragraph on recorded fixtures gives D1's redaction.
* `docs/architecture.md` follows, where it lists `tools/`
  (`docs/architecture.md:39-41`, `:873-878`).

### Consequences

* Good, because 72 copies of one helper, 13 hand-made index edits, and six
  JSON generators become calls to tested tools.
* Good, because a recorded fixture is checked for account data before it is
  committed, and the check never shows a value.
* Good, because a record's lint and its claims are checked the way its
  code is: against a planted break, and against what changed.
* Neutral, because the baseline holds 134 public model ids, and a refresh of
  Together's listing rewrites it.
* Bad, because `edit.py`'s spec grows from one entry shape to three, `tools/`
  from 11 tools to 13, and CI gains a gate.

### Confirmation

* Each new tool, and each new mode, has a test in `tools/test_tools.py`
  that fails it on a planted input and passes a clean one; `make
  tools-test` passes.
* Each is seen to fail first, on a scratch clone (`tools/plant.py`), as
  0035-MADR asks.
* `make fixtures-check` on the tree exits 0; `gate-selftest`'s planted
  address, id and JWT each exit 1, and the output holds the kind and the
  fingerprint, never the value.
* `records.py status` on a copy of the repository's records produces what
  `check_records.py --check-all` accepts.

## Pros and Cons of the Options

### A. Three new tools and two extensions

* Good, because each tool does one job, under the rules every tool keeps.
* Good, because the extensions keep every existing spec and command valid.
* Bad, because there are more tools to keep tested.

### B. Leave them in the scratchpad

* Good, because nothing changes.
* Bad, because 0035-MADR's reason recurs: these were rebuilt by hand, plan
  after plan, and the next pruning loses them.

### C. One multi-command records tool

* Good, because there is one entry point.
* Bad, because a fixture scan, a lint delta, a status edit and a plant
  runner share no code or input, and one tool would mix four exit-code
  meanings.

### D. As A, with the fixture scan as a CI gate

* Good, because a fixture with account data cannot be merged.
* Good, because the first findings were reviewed before the gate starts:
  the baseline holds 134 public ids, and no placeholder or prose word.
* Bad, because a ranking refresh that changes Together's listing changes
  the baseline in the same commit.

## Owner questions

Answered 2026-10-19; each answer is in the decisions above.

* **Q1.** Where does the fixture scan run? **The owner: in CI from the
  start.** So it is `scripts/check_fixtures.py`, a gate (D1), and the
  chosen option is D.
* **Q2.** Who reviews the 188 strings already committed? **The owner: the
  agent, from the project's facts and history, with recommendations.** The
  review is in the Context, by context and never by value. The owner then
  chose: tighten the patterns so the 54 placeholder and prose hits never
  fire; admit the 134 Together model ids by fingerprint; a gate in
  `scripts/`; and the redaction rule in `AGENTS.md`.
* **Q3.** Should `plant.py` read `"old_file"` and `"new_file"` too? **The
  owner: yes** (D2).

## More Information

* The classification's evidence is the scratch scripts themselves, which
  are not committed; this record names them by their scratch paths only
  where it cites them.
* Related: [0035-MADR](0035-MADR-maintainer-tools-in-python.md) (the
  tools' rules), [0046-MADR](0046-MADR-v1-15-0-follow-ups.md) (the claims
  plants it ran by hand), and AGENTS.md's "Identifiers" and "Tools".

### A1 (2026-10-19): 133 baseline lines for the 134 hits

* **Found** in 0047-PLAN Phase 1 step 2: `--write-baseline` wrote 133 lines
  for 134 hits. Two of Together's model entries carry one UUID:
  `mistralai/Mistral-7B-Instruct-v0.1` and `mistralai/Mistral-7B-Instruct-v0.3`
  (fingerprint `2dd3fb6ddc44`) **[probed]**. The Context's own probe had said
  "134 hit(s), 133 distinct"; D1 carried 134 into the baseline.
* **The owner chose (2026-10-19):** a baseline line admits a value wherever
  it occurs in its file, so the baseline is 133 lines for the 134 hits. D1
  is annotated; no decision changes.

### A2 (2026-10-19): the records are linted with their own conventions

* **Found** in 0047-PLAN Phase 4, before the tool was written **[probed]**:
  under the repository's markdownlint config, the 101 records hold 18,871
  MD004 findings and 8,873 MD010, against about 500 of every other rule
  together. The config pins list markers to `-`; 95 records use `*`, as the
  records' own convention asks. MD010's are gofmt tabs in quoted Go. Almost
  every record edit adds one or the other, so D4 as written would flag
  almost every edit.
* A `--config` override does not take: markdownlint-cli2 still reads the
  repository's config from each record's directory, and its MD004 wins
  **[probed]**.
* On copies in a temporary directory, with MD004 `consistent` (one marker
  style per file) and MD010 `code_blocks: false`, the records hold about
  830 findings in 51 of 101, each a real one, such as mixed markers or a
  list without blank lines around it **[probed]**.
* **The owner chose (2026-10-19):** D4 lints both versions from such copies,
  with those two rules. No other decision changes.
