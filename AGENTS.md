# AGENTS.md

Instructions for AI coding agents working in this repository. All agents read
this file. A repository-local `CLAUDE.md` / `.claude/rules/` / `.grok/rules/` /
`.opencode/rules.md` wins only where it is more specific than this file.

`go-llmprovider-sdk` is a shared Go library
(`github.com/maccavelli/go-llmprovider-sdk`). It is a library only: no packaged
binary. LLM providers live in `llmprovider/`; the configuration wizard lives in
`wizard/`; secret redaction for both lives in `internal/redact/`. Requires Go
1.27.2.

## API work

Every change to an exported API follows
[`docs/guides/api-standards.md`](docs/guides/api-standards.md). The guide is
normative: a change that breaks a rule is wrong, and the rule changes only by
amending the decision it cites. In the same change, update
[`docs/guides/migrating-from-mcplib.md`](docs/guides/migrating-from-mcplib.md)
for anything that replaces an `mcplib` identifier, and
[`docs/architecture.md`](docs/architecture.md) for the layout (R43). A new
provider follows
[`docs/guides/adding-a-provider.md`](docs/guides/adding-a-provider.md), after
its MADR.

## Dependencies

The standard library and `golang.org/x/term` (with its indirect
`golang.org/x/sys`) only. Any other module needs a MADR in this repository
first. Never import `github.com/maccavelli/mcplib` or the MCP go-sdk: this
module exists so that callers can use the providers without either.

`go.mod` and `go.sum` change with the code that needs them: a requirement is
added in the commit that adds its first import, and removed in the commit that
removes its last. `go mod tidy -diff` is clean at every commit.

## MADR and PLAN before mutating work

**Whenever the user asks for an MADR and a plan, load the
`madr-and-plan-writing` skill first** and follow it for authoring, naming and
review. This applies both to writing a fresh pair and to amending an existing
one.

The name is exact — it is the `name:` field of the skill. A mistyped call
returns `Unknown skill`, and an agent that proceeds without the skill writes
something shaped like a MADR while missing the required headings and the
directory rule below. Verify against the filesystem rather than memory:

```bash
ls -d ~/.claude/skills/*madr* && grep '^name:' ~/.claude/skills/*madr*/SKILL.md
```

**Read-only investigation is allowed with no pair.** Reading, searching,
`git log` / `git show` / `git diff`, and existing tests or diagnostics that
do not write the tree do not need a MADR.

**Mutating work is not.** Before the first write, name the
`docs/decisions/NNNN-MADR-*` / `docs/decisions/NNNN-PLAN-*` pair being
executed, or stop and write one.

Mutating means: creating, editing, or deleting files; staging or committing
(except the bootstrap exception below); dependency or lockfile changes;
CI / config / hook changes; builds or installers that write the tree,
`$HOME`, or a live service; generating committed artifacts.

Order:

1. Investigate (read-only).
2. Write or amend the MADR (`status: proposed` unless the owner already
   decided). Present it. Do not implement.
3. Write or amend the PLAN. Present it.
4. Mutate **only after** the owner explicitly approves execution
   (`proceed`, `execute the plan`, `do phase N`). Stay inside that PLAN.
5. Anything discovered mid-execution that is out of scope waits: amend the
   pair, re-approve, then continue. Completing a phase is not permission to
   invent the next unwritten one.

Follow-up vs greenfield:

- **Same topic** (debug, leftover phase, bug found in that plan's live
  run): amend that number. Add a PLAN phase or an amendment in the MADR. Do
  not silently rewrite historical rationale.
- **Greenfield**: next unused `NNNN`, new MADR, new PLAN, same slug. No
  mutation until that PLAN is approved.

Bootstrap exception: authoring `docs/decisions/NNNN-MADR-*`,
`docs/decisions/NNNN-PLAN-*`, this file, and the per-agent pointers
(`.claude/rules/`, `.grok/rules/`, `.opencode/rules.md`) does not require a
*prior* pair. Putting source, tests, CI, or product config in that same commit
is a violation.

`git push` and tags still need an explicit ask in the same turn.

## Records

```text
docs/decisions/NNNN-MADR-short-slug.md
docs/decisions/NNNN-PLAN-short-slug.md
docs/reports/NNNN-REPORT-short-slug.md
docs/reports/NNNN-GATES-short-slug.md
```

- `NNNN` is a zero-padded 4-digit number, one sequence across
  `docs/decisions/` and `docs/reports/`. A MADR and its PLAN share the same
  number and the same slug.
- **Next number** is the highest `NNNN` among all four kinds anywhere under
  `docs/`, plus one: `python3 -B scripts/check_records.py --next` prints it.
  Never reuse a number, never renumber an existing record, never leave a gap
  deliberately.
- Records moved here from `mcplib` were renumbered once, by
  `docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md`; that record
  maps every old number to its new one.
- Cite records by full filename, never by number alone. Cite another
  repository's record by repository and filename; a relative link cannot reach
  it.
- `docs/README.md` indexes every record. Update it in the same change.
- `make records-check` checks these rules: names and directories, statuses
  by kind, MADR/PLAN pairs, and the index's rows, statuses and count
  (`docs/decisions/0025-MADR-records-consistency-check.md`).

## Pre-add checks

Before staging Go files, run:

```bash
make pre-add-check                 # every tracked Go file
make pre-add-check FILES="a.go b.go"
```

It runs `scripts/go-precheck.sh`: `gofmt` on the files;
`golangci-lint run -c .golangci.yml --build-tags live_gateways ./...` for the
host and again with `GOOS=windows`, the same commands as `make lint` and CI,
which also lint the live-tagged tests and the `_windows.go` files; `go vet`
and `go test` on the packages the files belong to, a deleted file's
included; and `govulncheck ./...` at CI's pinned version, through `go run`,
as `make vuln` runs it (`GO_PRECHECK_SKIP_VULN=1` skips it offline). `golint` is
not used: its checks are `revive`'s `exported`, `package-comments` and
`var-naming` rules in `.golangci.yml`. A file that fails is not committed.

The machine-wide agent gate runs the same script before every agent
`git commit` that stages Go files, and denies the commit when it fails. There
is no `git add` hook on every host; do not rely on one.

`make lint` and `make vuln` must be clean before a release-shaped change.
CI also runs `make parity-check dep-check coverage-check api-check
generate-check records-check gate-selftest`; run them before asking for a push. A coverage floor is changed only in
`scripts/coverage-floors.txt`, by a record.

## Live tests

Live tests call real services and are opt-in:

```bash
go test -tags live_gateways ./llmprovider/... -run Live
```

`./llmprovider/...` includes the live tests in `auth` and `catalog`. A
provider's live tests run when its key variable is set (`ProviderEnvVars()`),
and skip without it. These suites are switched on by their own variable:
`LLMPROVIDER_LIVE_CHATGPT`, `LLMPROVIDER_LIVE_BROWSER_LOGIN` (the ChatGPT and
Grok browser logins), `LLMPROVIDER_LIVE_DEVICE_LOGIN` (the Grok device-code
login), `LLMPROVIDER_LIVE_OPENAI_SIGNIN` (the OpenAI sign-in probe),
`LLMPROVIDER_LIVE_GROK_CLI`, `LLMPROVIDER_LIVE_TOGETHER` (with
`TOGETHER_API_KEY`). The login tests need a person to sign in. CI only vets
the live-tagged files (`go vet -tags live_gateways ./...`); it never runs them.

## Identifiers

Nothing committed carries a hostname, account name, org-internal path, or a
real-machine absolute path. Use placeholders (`<user>`, `/home/<user>/...`).

- When recording an identifier scan in a record or a commit, describe what was
  scanned for ("the local account name", "the hostname domain"); never quote
  it.
- Pushes to GitHub pass through a global pre-push disclosure guard. Before
  asking the owner to push, run the guard itself over the outgoing commits:

  ```bash
  echo "refs/heads/main $(git rev-parse HEAD) refs/heads/main $(git rev-parse origin/main)" |
    python3 ~/.global-git-hooks/github-disclosure.py pre-push origin "$(git remote get-url origin)"
  ```

- Never bypass it with `--no-verify`.

## Commits

`git commit --no-edit`. Never pass `-m` / `--message` / `-F`. The global
`prepare-commit-msg` hook writes the message from the staged diff. This
repository sets a local `user.name` / `user.email`; do not override it.
