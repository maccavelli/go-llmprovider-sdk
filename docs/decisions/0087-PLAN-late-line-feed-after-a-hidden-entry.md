---
status: in-progress
date: 2026-11-22
associated-madr: "0087-MADR-late-line-feed-after-a-hidden-entry.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Settling a Hidden Entry's Enter, So Its Late Line Feed Does Not Answer the Next Prompt

Associated MADR: [0087-MADR-late-line-feed-after-a-hidden-entry.md](0087-MADR-late-line-feed-after-a-hidden-entry.md)

## Goal

After this PLAN, an Enter that reaches `TextPrompter` as CR then LF in two
reads, the LF within 50 ms, no longer answers the question after a hidden
entry with its default, on Unix, and on Windows if Phase 1 sees the split
there; the person's next Enter at that question still does (0020-MADR F17);
type-ahead, pipes and every line-mode read behave as before; no exported
name changes; and the docs say so.

The MADR is `accepted` (option A, the owner's rulings OR1 to OR7). This PLAN
is `proposed`: it awaits the owner's approval to execute. Nothing below
starts before the owner says `proceed`, `execute the plan` or `do phase N`.
*(Annotated 2026-11-22: the owner approved this PLAN and the start of Phase
1; it is `in-progress`. See the execution record's entry of that date.)*

## Scope

* `wizard/text_prompter.go`: `secretOnce` stores its raw-mode file in an
  unexported field, `rawIn`, and clears it before the restore (MADR A1a,
  OR8); `readMasked(prompt)` keeps its signature and settles only while
  `rawIn` is set; the reader keeps a peek's error for the next read;
  `TextPrompter`'s doc comment gains a paragraph and its first line changes
  (MADR A6a).
* `wizard/settle_unix.go` (new, `//go:build unix`): the readiness check
  through `golang.org/x/sys/unix.Select`. *(Fourth pass.)*
  `settleWait = 50 * time.Millisecond` is declared once, in
  `wizard/text_prompter.go`, for every platform, since that shared file
  passes it to `pendingWithin`.
* `wizard/settle_other.go` (new, `//go:build !unix`): no wait, until Phase
  1 maps to "seen"; then `wizard/settle_windows.go` (new, `//go:build
  windows`) takes Windows and `settle_other.go` becomes `!unix && !windows`.
* `wizard/prompter.go`: one sentence in `Prompter.Secret`'s doc comment (MADR
  A6, OR5).
* `go.mod`: `golang.org/x/sys` from indirect to direct; `go.sum` as
  `go mod tidy` leaves it.
* Tests: `wizard/late_lf_0087_test.go`, `wizard/late_lf_0087_unix_test.go`,
  and, if "seen" in a pseudo console, `wizard/late_lf_0087_windows_test.go`,
  or, if "seen" on a real console only, `wizard/live_settle_0087_windows_test.go`
  (`live_gateways`); Phase 1's probe, `llmprovider/live_raw_enter_0087_windows_test.go`
  (OR9), and its offline twin, `llmprovider/raw_enter_format_0087_test.go`.
* Docs: `README.md` ("The configuration wizard"), `docs/architecture.md`
  (the `TextPrompter` bullet, the `wizard` row of the package table, and
  lines 11, 199 and 1388), `docs/guides/api-standards.md` (R2's text at
  lines 31 to 32 and its `wizard` row), `AGENTS.md` (line 32, and "Live
  tests": the variable `LLMPROVIDER_LIVE_0087` and who runs it),
  `CHANGELOG.md` (`[Unreleased]`, **Fixed**), and this pair (MADR A6a, ST41,
  ST42).
* Out of scope: every other prompt and flow; mcplib's copy of `TextPrompter`;
  gobble-cli's input layer; any consumer's code; the tag itself, which the
  release's own step makes.

## Relation to Sibling Records

Read in the committed text at `24756a6`. The owner ruled this record into
the same tag as 0081, 0083, 0084 and 0085, built last, after 0083 (OR6).

* **0081-PLAN-a-cancelled-sign-in-saves-nothing.md** changes
  `wizard/configure.go`, `wizard/auth.go`, `wizard/kilo_verify.go`,
  `wizard/device.go` and their tests; neither prompter file. Shared:
  `CHANGELOG.md`.
* **0084-PLAN-typed-error-for-a-refused-tool-schema.md** changes
  `wizard/describe.go` and no prompter file. Shared: `CHANGELOG.md`,
  `README.md`, `docs/architecture.md`.
* **0085-PLAN-refuse-an-empty-typed-key.md** edits `wizard/prompter.go`: the
  package comment and `Prompter.Secret`'s doc comment (`wizard/prompter.go:94`
  to `:95` at `24756a6`; its claims cite it as `wizard/prompter.go:88` at
  `b109cea`). Its claims cite `wizard/text_prompter.go` lines 156 (C18), 402
  (C19), 408 (C22), 410 (C20) and 496 (C21), at `b109cea`, where they hold
  the same text as at `24756a6`. It adds `TestTextPrompterRefusesBlank_0085`,
  which this PLAN keeps green (MADR ST16). Its row ST69 points here.
* **0083-PLAN-let-the-wizard-sign-in-without-a-callback-listener.md** edits
  `wizard/prompter.go`: the package doc and `Prompter.Select`'s doc comment
  (`:78` at `24756a6`; cited as `:72` at `b109cea`, P1). Its claims cite
  `wizard/prompter.go` `:66` (C55), `:74` (C56), `:82` (C57) and `:114` (C58)
  at `b109cea`, which are `:72`, `:80`, `:88` and `:120` at `24756a6`, and
  `wizard/text_prompter.go` lines 244 (C59, P19) and 251 (C60). Its tests
  run `TextPrompter.Select`.
* **What this record edits, so the others' re-grounding knows it lands
  after them:** in `wizard/text_prompter.go`, the `TextPrompter` doc comment
  (lines 32 to 46 at `24756a6`; a paragraph added there moves every line
  after it, so all seven lines above), `secretOnce`'s call at 441, and
  `readMasked`'s CR case at 484 to 496 (0085's C21 is line 496); in
  `wizard/prompter.go`, `Prompter.Secret`'s doc comment, after 0085's edit
  of it. Built last, it moves nothing an unbuilt sibling still cites.

## Rules for Every Phase

* **Red, then green.** Each test marked red is run at the phase's base with
  `tools/redgreen.py SPEC.json --expect fail --rev <base>` and must fail on
  its assertion; each guard marked green there `--expect pass`. After the
  change, all `--expect pass`.
* **Plants.** Each plant is the single production change named, made on a
  scratch copy of the fixed code, never in the tree; each must turn its test
  red. The execution record quotes each plant's failing line.
* **Heavy runs.** `python3 -B tools/gate.py` and `make pre-add-check` run
  through `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`,
  or on another host through `run-on-host`, one at a time per host. A
  single `go test ./wizard` may stay on the session host. A Windows run goes
  through `run-on-host --need windows`, with `sh -c` as its first word.
* **Live runs.** Phase 1's probe is run by an agent on a pseudo console
  (OR7); only the two real-console steps are the owner's.
* **Stop conditions,** each a stop and a prompt to the owner: an existing
  test, golden or fixture needs a change; `apidiff` reports any change; a red
  test cannot be red at the base without a production change; a row's
  outcome cannot be met as written; an observation the MADR's mapping does
  not name; a third build pass of one phase.

## Implementation Steps

### Phase 0: records

1. This pair and its index rows; the pointer line in 0085-MADR beside its
   ST69. Done on 2026-11-20 (MADR `accepted`, PLAN `proposed`).
2. The owner approves this PLAN; its status goes to `in-progress`
   (`tools/records.py status`).

Exit criterion: no blocker, every row tested, plants caught (here: every
decided row of the MADR names its test and plant, and none is `open`).

### Phase 1: the Windows probe

It changes no production file, so it may run before the siblings are built.

1. **Re-ground** at the phase's start commit: `golang.org/x/term`'s
   `term_windows.go` `makeRaw`, Go's `internal/poll/fd_windows.go`
   `readConsole`, and `golang.org/x/sys/windows`'s `CreatePseudoConsole`,
   `WaitForSingleObject` and `NewLazySystemDLL`, at the versions `go.mod`
   names; and that `PeekConsoleInputW` is still unwrapped.
2. **Write the probe** (an agent): `TestLive_RawEnterBytes_0087` in
   `llmprovider/live_raw_enter_0087_windows_test.go` (OR9), as the MADR's
   "Windows: the probe" describes: the standard library only (kernel32
   through `syscall.NewLazyDLL`); `CONIN$` and `CONOUT$` opened by the probe,
   never the test binary's standard input or output; `tmp/live-0087/`
   created by the probe; modes `pty` (P1 to P3) and `console` (R1 to R3);
   only allowlisted lines; no key, sign-in, browser or network. Its offline
   twin, `TestRawEnterBytesFormat_0087` in
   `llmprovider/raw_enter_format_0087_test.go` (no tags), runs the line
   formatting over an injected reader and checks that only the mode, the
   console kind, the case name, the result word and line-ending bytes are
   printed; its plant (the formatter prints a letter it read) must turn it
   red. `AGENTS.md` "Live tests" names `LLMPROVIDER_LIVE_0087`; `make
   tables-check` passes, ~~and fails with that name removed~~ *(annotated
   2026-11-22, OR12: that plant is not caught, since `tables-check` admits a
   variable its own record names; `make tables-check` fails when the probe
   reads a variable no document names, and the MADR's claims row C47now,
   planted false by `tools/claims.py --plants`, pins `AGENTS.md`'s name;
   MADR "Amendments of 2026-11-22", A-1)* (MADR ST42).
   `GOOS=windows go vet -tags live_gateways ./llmprovider` is clean, and
   `make dep-check` is unchanged.
3. **Run it** (an agent):
   `~/.agents/skills/available-dev-hosts/scripts/run-on-host --need windows -- sh -c 'LLMPROVIDER_LIVE_0087=pty go test -tags live_gateways ./llmprovider -run "^TestLive_RawEnterBytes_0087$" -count=1 -v'`,
   and read `tmp/live-0087/pty.txt` back with the run's output. Quote both
   in the MADR as a dated amendment. If it yields nothing (the MADR's last
   mapping row), fix the probe and rerun once; nothing a second time is a
   stop.
4. **The owner's two steps** (the MADR's "What a pseudo console cannot
   show"), with no redirection: the classic console window, then a Windows
   Terminal tab. The probe shows its requests on the console and writes
   `tmp/live-0087/conhost.txt` and `tmp/live-0087/terminal.txt`; the agent
   reads the two files and quotes them.
5. **Map** each observation by the MADR's table: ST20 and ST21 become
   `decided` by dated rows, and the Windows branch of Phase 2 is fixed
   ("seen" or "not seen"). An observation the table does not name is a stop.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 2: the settle

Starts on the commit that carries 0081's, 0085's and 0083's code (the base).

1. **Re-ground.** Run
   `python3 -B tools/claims.py docs/decisions/0087-MADR-late-line-feed-after-a-hidden-entry.md --rev self=<base>`;
   annotate each moved row in the MADR, dated, with its new line. Re-read at
   the base the lines "Relation to Sibling Records" names, in their new
   places: the `TextPrompter` doc comment, `secretOnce`, `readMasked`'s CR
   case, and `Prompter.Secret`'s doc comment as 0085 left it. Re-check that
   `Secret` is called only from 0085 D2's `secretKey` helper (C46), and
   `secretKey` only from the two places C21 and C22 named before 0085 moved
   them, and that no hidden entry precedes 0083's question (ST28, ST15). Record all of it in the
   execution record.
2. **Tests first.** Each compiles at the base: they use only `TextPrompter`,
   `Choice`, `ConfigureLLM`, `Options`, the provider ids and `rawSeam0059`.
   "The line-end hook" is an `Out` that writes given bytes into the input
   pipe once, when the prompter prints `\r\n`: they arrive after the CR is
   read and before `Secret` returns, as a terminal's CR+LF does. Tests marked
   Unix are in `wizard/late_lf_0087_unix_test.go` (`//go:build unix`); the
   rest in `wizard/late_lf_0087_test.go`; the pipe's writer stays open unless
   a test closes it. Two tests set the unexported `settleWait`, which does
   not exist at the base: they are named in `tools/redgreen.py`'s `drop` list
   for the red run, and their red is their plant's (marked "dropped").

   | Test | Row | What it does | At the base | Plant (on the fixed code) |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestSecret_LateLineFeedDropped_0087` (Unix) | ST3 | Under `rawSeam0059`, on an `os.Pipe`: a key and a CR; the line-end hook writes an LF. Subtests: `Input` (default `saved-model`, answer `my-model`); `Select` (default 1 of 3, answer `3`); `Confirm` (default yes, answer `n`); `Secret` (a second hidden entry, `key2` and a CR, is returned and `Out` holds no "empty value"). | red: `Input` returns `saved-model` (the reproduction's `LF at the line end` case), `Select` 0, `Confirm` true, the second `Secret` warns | `readMasked` no longer calls the settle |
   | `TestSecret_LateLineFeedDuringWait_0087` (Unix) | ST3 | Sets `settleWait` to 2 s for the test; the line-end hook starts a goroutine that writes the LF 10 ms later; the test writes the answer only after that goroutine is done; `Input` returns the answer. The 2 s wait leaves a margin of nearly 2 s for a stalled `-race` run; W itself is pinned by ST23's claims row. | dropped | the check made once with no wait |
   | `TestSecret_NoWaitWhenNotNeeded_0087` (Unix) | ST2, ST7 | Sets `settleWait` to 5 s. Case `CR+LF together`: a key, CR and LF in one write; case `LF-ended`: a key and an LF. In each, `Secret` returns within 2 s (it would take 5 s had it waited). | dropped | the settle runs whatever the reader holds (`CR+LF together`); the settle runs after an LF (`LF-ended`) |
   | `TestSecret_NextEnterKept_0087` | ST4 | A key and a CR; after `Secret` returns, the test writes an LF; `Input` returns its default. | green | option B's rule: a leading LF dropped at the next read after a CR |
   | `TestSecret_TypeAheadKept_0087` | ST5 | The line-end hook writes `3` and an LF; `Select` returns index 2. | green | the settle discards whatever byte it peeks |
   | `TestSecret_SettleIsBounded_0087` | ST1; on Windows also ST22 ("not seen") and ST31 | Nothing follows the CR: `Secret`, on a goroutine the test waits for at most 5 s, returns within 1 s. Run on Unix and, through `run-on-host --need windows`, on Windows. | green | a blocking `Peek` in place of the bounded wait (ST1); the Windows build waits 5 s (ST22, "not seen"); a failed `PeekConsoleInputW` treated as an LF pending (ST31, "seen") |
   | `TestSecret_NoReadLeftBehind_0087` (Unix) | ST14 | After `Secret` returns, one byte is written to the pipe; 200 ms later `unix.IoctlGetInt` on the pipe's reader, with the request named per GOOS in a small test helper (*fourth pass*: Linux `unix.TIOCINQ`, `0x541b`, in `golang.org/x/sys` v0.47.0's `zerrors_linux_amd64.go:459`; darwin `0x4004667f`, `FIONREAD`, which `golang.org/x/sys` does not name, from the macOS SDK's `sys/filio.h:77` `_IOR('f', 127, int)` and `sys/ioccom.h`'s `_IOC`; the helper is two files, `_linux_test.go` and `_darwin_test.go`, so this test builds on Linux and macOS only), still counts 1. No other reader exists until the test reads, so the catch does not depend on which goroutine runs first. | green | the settle moved to a goroutine with no bound that outlives `Secret` (it reads the byte, and the count is 0) |
   | `TestSecret_RawFieldCleared_0087` (Unix) | ST36 | A hidden entry under `rawSeam0059`; then `readMasked` called directly on the same prompter, ended by a CR while the line-end hook, armed again, writes an LF; then `ans` and an LF are written; `Input` returns its default, not `ans`. | green | `secretOnce` does not clear `rawIn` (the LF is dropped and `Input` returns `ans`) |
   | `TestSecret_SecondLineEndKept_0087` (Unix) | ST6, ST37 | The line-end hook writes LF, CR, LF, then `ans` and an LF: the first `Input` returns its default, the second `ans`. Subtest `CR then CR, raw`: the hook writes a CR; a second `Secret` warns "empty value" once, then returns `key2` written after it. Subtest `CR then CR, line`: the hook writes a CR; the test writes `ans` and an LF; `Input` returns `ans`, not its default (the question waited for the next line, ST37's loss). | red: the first late LF answers the first `Input`, so the second takes its default, not `ans`; both `CR then CR` subtests green | the settle drops every CR and LF it finds; `readLine` ends a line on a CR (`CR then CR, line`) |
   | `TestSecret_LineFeedEndedEntry_0087` | ST7 | A key ended by an LF; the line-end hook writes a CR. Subtest `line mode`: `Input` returns the next answer, trimmed. Subtest `raw mode`: a second `Secret` warns "empty value" once and returns `key2`. | green | the settle also runs after an LF and drops a CR (`raw mode` sees no warning) |
   | `TestSecret_EndOfInputWhileSettling_0087` (Unix) | ST8 | The line-end hook closes the pipe's writer: `Secret` returns the key with no error; the next `Input` returns its default with no error; the one after matches `io.EOF`. | green | the settle marks the input ended when its peek meets the end |
   | `TestSecret_SettleCheckFails_0087` (Unix) | ST9 | Subtest `closed`: the line-end hook closes the pipe's reader, so its descriptor reads as negative; `Secret` returns the key with no error; the next `Input` returns an error matching `os.ErrClosed`. Subtest `descriptor 1100`: the pipe's reader duplicated to descriptor 1100 (`unix.Dup2`; the test fails, naming the limit, if the process may not open it); `Secret` returns the key with no error and no panic. | green | the settle returns its check's error as `Secret`'s: the check returns an error for a negative descriptor and for one at or above `FD_SETSIZE`, so both subtests catch it; the `FD_SETSIZE` check removed, so `FdSet.Set` panics (`descriptor 1100`) |
   | `TestSecret_PipeCRLF_0087` | ST11 | No seam; a pipe holding a key, CR, LF, `my-model`, CR, LF: `Secret` returns the key, `Input` `my-model`. | green | `readLine` stops trimming |
   | `TestSecret_AfterLineAnswer_0087` | ST13 | Under `rawSeam0059`, one write: `y`, LF, a key, CR: `Confirm` returns true, `Secret` the key, and `Out` holds no "empty value". | green | `readLine` unreads the LF it read |
   | `TestSecret_LateNulKept_0087` | ST24 | The line-end hook writes a NUL, then `ans` and an LF: `Input` returns the NUL followed by `ans`. | green | the settle drops a NUL as well as an LF |
   | `TestSecret_LongPasteSplitAtCR_0087` (Unix) | ST33 | One write of 4095 letters, a CR and an LF: the reader's first fill ends on the CR, the LF stays in the pipe; `Secret` returns the 4095 letters, and `Input` the answer written after. | red: `Input` returns its default | `readMasked` no longer calls the settle |
   | `TestConfigure_SearchAfterLateLineFeed_0087` (Unix) | ST15 | `ConfigureLLM` over a `TextPrompter` on a pipe under `rawSeam0059`, the line-end hook writing an LF; a provider that requires a key and has a static catalog, `Discover` off; the answers: the provider's number, the method if one is asked, a key and a CR, then a search for a static model that is not the first recommended, and `1`. `Result.Model` is that model. The provider and the model are chosen at the base and named in the execution record. | red: the blank search shows the recommended menu, the query is refused as a number, and `1` picks the first recommended | `readMasked` no longer calls the settle |

   If Phase 1 mapped to "seen", add in `wizard/late_lf_0087_windows_test.go`:

   | Test | Row | What it does | At the base | Plant (on the fixed code) |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestSecret_LateLineFeedDropped_Windows_0087` | ST22 ("seen") | On a pseudo console, as Phase 1's probe starts its child: a key and a CR, then an LF 10 ms later, then `my-model` and a CR; the child's `Input` returns `my-model`. | red | `readMasked` no longer calls the settle on Windows |
   | `TestSecret_SettleIsBounded_Windows_0087` | ST32 | On a pseudo console: a key and a CR only (so only the CR's key release is pending); `Secret` returns within 1 s, waited for at most 5 s. | green | any pending console event treated as a character, so the peek blocks |

   If Phase 1 mapped to "seen" on a real console only (the MADR's second
   mapping row), the Windows tests cannot be red in a pseudo console. In
   their place, `TestLive_SettleOnConsole_0087` in
   `wizard/live_settle_0087_windows_test.go` (`live_gateways`, the same
   variable set to `settle`): a `TextPrompter` on `CONIN$` and `CONOUT$`
   asks for a hidden entry, then a line-mode question with a default, and
   writes one allowlisted line, `default` or `answer`, to
   `tmp/live-0087/settle-<console kind>.txt`. The owner runs it in the
   console that split, once at the base (red: `default`) and once on the
   fixed build (green: `answer`); the base stands as its plant. These are
   the owner's only further steps, in this branch only. *(Fourth pass.)* It
   replaces only `TestSecret_LateLineFeedDropped_Windows_0087`;
   `wizard/late_lf_0087_windows_test.go` with `TestSecret_SettleIsBounded_Windows_0087`
   still runs in this branch and pins ST32, since it only has to be green on
   a pseudo console (its plant still turns it red there).

   Rows pinned by existing tests, each with its plant: ST2
   `TestTextPrompter_MaskedEntry` (plant: the buffered-LF discard removed);
   ST4 and ST12 `TestTextPrompter_EnterAfterMaskedEntryIsKept` (plant:
   option B's rule); ST10 `TestSecret_RawMode_0059`, "raw mode refused"
   (plant: the refused path's line read not trimmed); ST16
   `TestTextPrompterRefusesBlank_0085` (plant: `Secret` returns an empty
   entry instead of asking again). ST17 and ST18 are checks (step 7). ST23
   and ST35 also get a `tools/claims.py` row each in the MADR's evidence
   index once the code is written (step 6): the `settleWait` constant and the
   `Prompter.Secret` sentence; each is planted false with
   `tools/claims.py --plants` and must be caught.
3. **Red run at the base.** `tools/redgreen.py` with `--rev <base>` and the
   new files and the two "dropped" tests in its `drop` list: the four tests
   marked red `--expect fail`, each on its assertion; the guards and the
   five existing tests `--expect pass`. Quote the failing lines.
4. **The code.**
   * `wizard/settle_unix.go`: `pendingWithin(f *os.File, wait time.Duration) (bool, error)`,
     `true` when `unix.Select` reports the descriptor readable within
     `wait`; `false` with an error for a negative descriptor, one at or above
     `unix.FD_SETSIZE` (neither is given to `select`) and any `select` error,
     `EINTR` included; `false` and no error on a timeout. `readMasked`
     ignores the error (MADR A3). All unexported. `wizard/settle_other.go`:
     `false, nil`. *(Fourth pass.)* `settleWait = 50 * time.Millisecond`, an
     unexported variable, is declared once in `wizard/text_prompter.go`, which
     passes it to `pendingWithin` on every platform; no `settle_*.go` file
     declares it. The red run still builds: `tools/redgreen.py --rev <base>`
     builds the base's own production code, which has no reference to
     `settleWait`, plus the copied test files, whose only references are in
     the two dropped test bodies, which it removes; the green run builds the
     working tree, where `text_prompter.go` declares it.
   * If "seen": `wizard/settle_windows.go`, the MADR's Windows mechanism,
     with `settle_other.go`'s build line narrowed.
   * `secretOnce`, once `makeRaw` has succeeded, sets `p.rawIn = f` and
     defers clearing it, registered after the restore's defer so that it
     runs first (MADR A1a). `readMasked(prompt)` keeps its signature. In the
     CR case, after `\r\n` is printed, when no LF was dropped by today's
     buffered rule, `r.Buffered() == 0`, `p.rawIn` is not `nil` and
     `pendingWithin` is true, it peeks one byte and discards it only if it
     is an LF.
     `bufio.Reader.Peek` hands a read error over once and clears it (the
     standard library's `bufio.go`, `Peek` and `readErr`), so a peek error is
     kept on the prompter and returned first by its next read: an end of
     input there answers as `readLine` answers the first one (0020-MADR F6).
     The settle runs before `secretOnce`'s deferred restore.
   * `go.mod`: `go mod tidy` moves `golang.org/x/sys` to the direct block.
   * `GOOS=linux`, `darwin`, `freebsd` and `windows` `go vet ./wizard`.
5. **Green and plants.** `tools/redgreen.py SPEC.json --expect pass` on the
   working tree for every test above; each plant on a scratch copy, red; `go
   test ./wizard -count=1` on this Mac, and through `run-on-host --need
   windows` and `run-on-host --need linux`.
6. **Docs.**
   * `TextPrompter`'s doc comment: its first line names `golang.org/x/sys`
     beside `golang.org/x/term` (C45); on a terminal, a hidden entry ended by
     a CR waits up to 50 ms, still in raw mode, and drops an LF that arrives
     in that time, so a terminal or a paste that sends CR+LF for one Enter
     does not answer the next question; a later LF, such as the next Enter,
     answers it as before. The stated losses (MADR A4a, OR10, OR11): a CR or
     a Ctrl-J typed within the 50 ms; a Ctrl-C or Ctrl-Z typed within it
     arrives as a plain byte; bytes read ahead belong to the prompter, so a
     caller that reads the terminal itself afterwards does not see them; a
     second reader of the same terminal is not supported. For Windows, the
     sentence Phase 1's mapping gives.
   * The import's lines (MADR A6a, ST41): `docs/architecture.md`'s `wizard`
     row (C39) and lines 11, 199 and 1388 (C41 to C43);
     `docs/guides/api-standards.md` R2's lines 31 to 32 (C44) and its
     `wizard` row (C36); `AGENTS.md:32` (C40). Each names
     `golang.org/x/sys` as `wizard`'s, direct. *(Fourth pass.)* The two table
     rows (C39, C36) name the exact import paths `tables-check` compares:
     `golang.org/x/sys/unix`, and `golang.org/x/sys/windows` too on the
     Windows "seen" branch; the prose lines may say `golang.org/x/sys`. `make
     tables-check` passes, and fails with the architecture row's new name
     removed.
   * `Prompter.Secret`'s doc comment: the MADR's A6 sentence, after 0085's
     edit of the same comment.
   * `README.md`, "The configuration wizard": the `TextPrompter` bullet gains
     one sentence with the same meaning, and one naming the stated losses.
   * `docs/architecture.md`: the **`TextPrompter`** bullet gains the settle.
   * `CHANGELOG.md` `[Unreleased]`, **Fixed**: the defect, who saw it, and
     this MADR's link, beside 0081's, 0083's, 0084's and 0085's entries.
   * **Consumer adoption note,** in the changelog entry and the execution
     record: prepare-commit-msg takes the fix by bumping from `v1.29.0`, with
     no code change; gobble-cli and magic-cli-remote do not use
     `TextPrompter`; a caller that implements `Prompter` over a raw terminal
     reads the new `Prompter.Secret` sentence; mcplib's copy is unchanged.
   * The two `tools/claims.py` rows of step 2, with their plants.
   * `docs/guides/migrating-from-mcplib.md`: unchanged (no identifier
     changes).
7. **Checks.** `gofmt -l` on the changed Go files; `make pre-add-check
   FILES="..."` through `run-heavy`; `make lint` (the host and
   `GOOS=windows`); `make dep-check`; `go mod tidy -diff`; `make api-check`
   and `python3 -B tools/apidiff.py v1.31.0` (no change); `make
   records-check docs-check links-check markdownlint changelog-check
   tables-check disclosure-check`; `python3 -B tools/denyscan.py` on the
   changed files; `python3 -B tools/claims.py` on the MADR;
   `python3 -B tools/gate.py` through `run-heavy`.
8. **Records.** The execution record below: each pass numbered, the red and
   green output, each plant's failing line, the checks' exit statuses, and
   anything not done.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 3: the release

The tag's own release step (0080-PLAN's Phase 5, the release) carries this
record's changelog entry and its consumer note with the other four records'.
This PLAN goes to `complete` when that tag is made and every row is green.

Exit criterion: no blocker, every row tested, plants caught.

## Verification

* Every decided row of the MADR's state table has its test green and its
  plant caught, with the output quoted in the execution record.
* The four tests marked red are seen red at the base and green after.
* `TestTextPrompter_EnterAfterMaskedEntryIsKept`, `TestTextPrompter_MaskedEntry`
  and `TestTextPrompterRefusesBlank_0085` pass unchanged.
* `apidiff` against `v1.31.0` reports nothing; `dep-check` and `tidy -diff`
  are clean; the docs gates pass; the Windows run of `go test ./wizard`
  passes.

## Rollout and Rollback

* Rollout: internal to `TextPrompter`; a consumer takes it by bumping the
  module. No configuration and no data change.
* Rollback: revert the phase's commit; `TextPrompter` returns to the buffered
  rule of 0020's F17, and `golang.org/x/sys` to indirect. Nothing persisted
  depends on it.

## Execution Record

No phase has run: the PLAN awaits the owner's approval. *(Annotated
2026-11-22: approved; Phase 0 step 2 and Phase 1 are recorded below.)*

### 2026-11-20: records pass 4, the delta confirmation's four corrections

Applied exactly as the delta confirmation named them, nothing else:

* **ST14's request name.** `FIONREAD` does not exist in `golang.org/x/sys`
  v0.47.0. `TestSecret_NoReadLeftBehind_0087` names its request per GOOS in a
  two-file test helper: Linux `unix.TIOCINQ` (`0x541b`), found in the module
  cache at `golang.org/x/sys@v0.47.0/unix/zerrors_linux_amd64.go:459`;
  darwin `0x4004667f`, computed from the macOS SDK's `sys/filio.h:77`
  (`#define FIONREAD _IOR('f', 127, int)`) and `sys/ioccom.h` (`IOC_OUT`
  `0x40000000`, `IOCPARM_MASK` `0x1fff`, `_IOR` through `_IOC`). A search of
  `golang.org/x/sys@v0.47.0/unix` finds the name `FIONREAD` nowhere and the
  value `0x4004667f` only as `TIOCINQ` on Linux ppc, ppc64, ppc64le and
  sparc64 (for example `zerrors_linux_ppc64.go:511`), which use the same
  BSD-style encoding, so the value is corroborated; no darwin file has it.
* **The import tables.** MADR A6a and Phase 2 step 6 name
  `golang.org/x/sys/unix` (and `golang.org/x/sys/windows` on the "seen"
  branch) in the two table rows, since `tables-check` compares exact import
  paths.
* **The Windows build.** `settleWait` is declared once, in
  `wizard/text_prompter.go`; Scope and Phase 2 step 4 say so, and how the red
  run still builds.
* **ST32 in the real-console-only branch.**
  `TestSecret_SettleIsBounded_Windows_0087` still runs there.

By the approach the owner chose for 0084 ("Apply the checker's fixes
exactly; the build proves them (Recommended)", the owner, relayed by the
coordinator), there is no further review round: the red run and the plants
at build time prove these corrections.

### 2026-11-22: Phase 0 step 2, the owner's approval

The owner, relayed by the coordinator, approved this PLAN and the start of
Phase 1, by the label "Approve; start Phase 1 now (Recommended)". The status
went to `in-progress` by
`python3 -B tools/records.py status docs/decisions/0087-PLAN-late-line-feed-after-a-hidden-entry.md in-progress --date 2026-11-22`,
which wrote the front matter and the index row and exited 1 on the
records-check it runs after writing. Its output, whole:

```text
docs/README.md: says 182 records; there are 185
docs/decisions/0088-MADR-default-output-limit-from-model-facts.md: not indexed in docs/README.md
docs/decisions/0088-PLAN-default-output-limit-from-model-facts.md: not indexed in docs/README.md
docs/reports/0088-REPORT-output-limits-across-agents-and-vendors.md: not indexed in docs/README.md
records: written; records-check: 4 problem(s)
```

The four problems are another record's uncommitted, unindexed files, not
this change; the tool's own write landed (`status: in-progress`,
`date: 2026-11-22`, and the index row `in-progress`), so nothing was changed
by hand.

### 2026-11-22: Phase 1, pass 1 (stopped at step 2 on ST42's plant)

Start commit `18e8e72`, on `main`. Steps 1 and 2 were built; step 2's
`tables-check` plant was not caught, which is the stop condition "a row's
outcome cannot be met as written" (ST42), so step 3 (the pty run) and step 4
(the owner's two steps) have not run, and no MADR amendment quotes a probe.

**Step 1, re-grounding** (the module cache and `GOROOT` of Go 1.27.4, at the
versions `go.mod` names: `golang.org/x/term` v0.43.0, `golang.org/x/sys`
v0.47.0). Every fact holds:

* `golang.org/x/term@v0.43.0/term_windows.go:25` `makeRaw`: clears
  `ENABLE_ECHO_INPUT | ENABLE_PROCESSED_INPUT | ENABLE_LINE_INPUT` (`:30`) and
  sets `ENABLE_VIRTUAL_TERMINAL_INPUT` (`:31`). Holds.
* Go's `src/internal/poll/fd_windows.go:653` `readConsole`: reads at most
  `len(b)` UTF-16 units into a 10000-unit buffer, hands over up to `len(b)`
  bytes, and ends a read before a Ctrl-Z (`0x1A`). Holds.
* `golang.org/x/sys@v0.47.0/windows`: `CreatePseudoConsole`
  (`syscall_windows.go:1871`), `WaitForSingleObject`
  (`zsyscall_windows.go:3640`), `NewLazySystemDLL` (`dll_windows.go:249`).
  Holds.
* `PeekConsoleInputW` is still unwrapped: a search of the whole
  `golang.org/x/sys@v0.47.0` tree for `PeekConsoleInput` exits 1. Holds.

**Step 2, what was written.**

* `llmprovider/live_raw_enter_0087_windows_test.go` (`live_gateways`,
  Windows by its name): `TestLive_RawEnterBytes_0087`, switched on by
  `LLMPROVIDER_LIVE_0087` (`pty` or `console`; `child:<console kind>:<case>`
  is the value the probe gives its own child). Standard library only,
  kernel32 through `syscall.NewLazyDLL`.
* `llmprovider/raw_enter_format_0087_test.go` (no tags): the case runner
  `rawEnterRun0087`, the line formatter `rawEnterLine0087`, the allowlist
  `rawEnterAllowed0087`, and the twin `TestRawEnterBytesFormat_0087`.
* `AGENTS.md` "Live tests": a paragraph naming `LLMPROVIDER_LIVE_0087`, its
  two values and who runs each.

Verification of step 2 (exit statuses captured directly):

| Check | Host | Exit |
| :--- | :--- | :--- |
| `go test ./llmprovider -run '^TestRawEnterBytesFormat_0087$' -count=1 -v` | this Mac | 0 (10 subtests pass) |
| `tools/plant.py` on the twin, plant "the formatter prints a letter it read" | this Mac, scratch clone | 0 (caught) |
| `GOOS=windows go vet -tags live_gateways ./llmprovider` | this Mac | 0, no output |
| `make dep-check` | this Mac | 0 (`GOOS=host: 46 packages, 0 problem(s)`; `GOOS=windows: 46 packages, 0 problem(s)`): unchanged |
| `make api-check` | this Mac | 0 (`against v1.31.0, 0 incompatible change(s)`) |
| `make tables-check` | this Mac | 0 (`49 packages, 0 problem(s)`) |
| `tools/plant.py`, `make tables-check` with `AGENTS.md`'s name of the variable removed | this Mac, scratch clone | 1: **NOT CAUGHT** (`make tables-check` exit 0) |

The twin's plant (one change, in `rawEnterLine0087`:
`groups = append(groups, rawEnterEndings0087(r)+string(r))`), its failing
lines as `tools/plant.py` printed them:

```text
== the formatter prints a letter it read: caught (exit 1)
   --- FAIL: TestRawEnterBytesFormat_0087/split (0.00s)
   raw_enter_format_0087_test.go:271: line = "mode=pty console=pty case=lf-after-cr result=split reads=0dsk-0087-SECRETzq\r,0a\n", want "mode=pty console=pty case=lf-after-cr result=split reads=0d,0a"
   raw_enter_format_0087_test.go:274: line "mode=pty console=pty case=lf-after-cr result=split reads=0dsk-0087-SECRETzq\r,0a\n" is not allowlisted
   raw_enter_format_0087_test.go:277: line "mode=pty console=pty case=lf-after-cr result=split reads=0dsk-0087-SECRETzq\r,0a\n" prints a letter it read
plant: all 1 plant(s) caught
```

#### Deviation, 2026-11-22: ST42's `tables-check` plant cannot be caught as written (open, with the owner)

*(Annotated 2026-11-22: ruled, OR12; see "Phase 1, pass 1, continued" below.)*

* **Found.** `scripts/check_tables.py` `check_live_variables` admits a
  variable that `AGENTS.md` does not name when it matches
  `^LLMPROVIDER_LIVE_(\d{4})` and a record of that number names it (lines
  198 to 205; `SWITCH` at line 52), as `AGENTS.md` "Live tests" says ("A
  record's own measurement test is switched on by `LLMPROVIDER_LIVE_<NNNN>`
  ... as that record says"). `0087-MADR` names `LLMPROVIDER_LIVE_0087` (its
  lines 317, 348, 359, 360, 374, 375 and 494, before this date's MADR
  amendments moved them), so the plant ST42 and step 2 name,
  "`AGENTS.md`'s name of the variable removed", leaves `tables-check` at exit
  0. The check behaves as written and documented; the PLAN's plant assumed
  it did not admit a record's own variable.
* **Seen, for the owner's choice (scratch clone, nothing in the tree):** a
  plant that makes the probe read its variable's name with a suffix that
  neither `AGENTS.md` nor a 0087 record names *(reworded 2026-11-22: the
  planted name is kept out of the 0087 records, since a record of the
  number that names it admits it; see pass 1's continuation)*, is caught (`make tables-check`
  exit 2 under `tools/plant.py`, "plant: all 1 plant(s) caught").
* **Not done, pending the ruling:** step 3 (the pty run), step 4 (the
  owner's steps file), the MADR amendment, and the heavy gates
  (`make pre-add-check`, `go test ./...`, the scratch-clone `records-check`
  and `gate-selftest`).

#### Notes on the probe as built, 2026-11-22 (for the review of this pass)

*(Annotated 2026-11-22: accepted as deviations, OR13; see "Phase 1, pass 1, continued" below.)*

* **One child per case, in both modes.** Each case runs in a fresh child of
  the test binary, which exits after writing its line, so a console read
  still waiting at the end of a case is ended with its process and cannot
  take the next case's bytes. In `pty` mode each child has a pseudo console
  of its own (as the MADR says); in `console` mode the children share the
  owner's console (the MADR's "the test opens its own `CONIN$` and
  `CONOUT$`" holds for each child).
* **Two kernel32 calls beyond the MADR's list:** `FlushConsoleInputBuffer`
  (console mode only, before each request, so keys pressed between two
  requests are not the next case's) and `DeleteProcThreadAttributeList`
  (cleanup). Standard library only still holds.
* **Reading `pty.txt` back.** `run-on-host` deletes its work root when the
  command exits 0, so the file cannot be read after the run. The probe reads
  `tmp/live-0087/pty.txt` back itself after writing it and logs each line as
  `tmp/live-0087/pty.txt: <line>`, so the exact command of step 3 returns
  both the run's lines and the file's.
* **Waits.** A pty child waits 15 s for its first read and ends after 1 s
  with no read; the parent writes 200 ms after the child is ready, and P2's
  LF 10 ms after its CR. A console child waits 120 s for the person and ends
  after 1.5 s with no read; each console child is bounded at 270 s, inside
  `go test`'s default 10 m.

### 2026-11-22: Phase 1, pass 1, continued (steps 2 to 4; step 5 waits on the owner)

**The owner's rulings,** relayed by the coordinator, by their exact labels:

* OR12, on the stop at ST42's plant: "Amend the plant + claims row
  (Recommended)". Recorded in the MADR (ST42's row, the rulings table, and
  "Amendments of 2026-11-22", A-1) and in step 2's annotation above.
* OR13, on the four notes on the probe as built: "Accept all four as
  deviations (Recommended)". Recorded below, one dated deviation each, and
  in the MADR for the two kernel32 calls (A-2).

#### Deviations, 2026-11-22 (OR13)

1. **One child per case, in both modes.** Each case runs in a fresh child of
   the test binary, which exits after writing its line, so a console read
   still waiting at a case's end ends with its process and cannot take the
   next case's bytes. `pty`: a pseudo console per child, as the MADR says.
   `console`: the children share the owner's console, each opening `CONIN$`
   and `CONOUT$` itself.
2. **Two kernel32 calls beyond the MADR's list:** `FlushConsoleInputBuffer`
   (console mode, before each request) and `DeleteProcThreadAttributeList`
   (cleanup). MADR A-2.
3. **Where the formatter lives.** The case runner `rawEnterRun0087`, the
   formatter `rawEnterLine0087` and the allowlist live in the untagged twin
   file, `llmprovider/raw_enter_format_0087_test.go`, since the twin must
   build on every platform with no tag and the PLAN names only the two
   files; the twin's plant is made there.
4. **Reading `pty.txt` back.** `run-on-host` deletes its work root after a
   run that exits 0, so the probe reads `tmp/live-0087/<console kind>.txt`
   back itself and logs each line as `tmp/live-0087/<kind>.txt: <line>`;
   step 3's exact command then returns both.

#### Deviation, 2026-11-22: the planted variable's name kept out of the records

Re-running the new `tables-check` plant on the tree after the MADR's
amendment A-1 was first written: **not caught** (`tools/plant.py` exit 1,
`make tables-check` exit 0), because A-1 then quoted the planted name, and a
0087 record that names a `LLMPROVIDER_LIVE_0087…` variable admits it. A-1
and the pass's earlier "Seen" bullet were reworded to describe the name
without writing it (annotated where it changed). Re-run on the final tree:
caught, exit 2, "check-tables: AGENTS.md: <the planted name>: a live test
reads it, and AGENTS.md does not name it". The planted name lives only in
the plant's spec, outside the tree.

#### Step 2, the new pins of ST42 (OR12)

* `tools/plant.py`, `make tables-check`, plant "the probe reads a variable
  no document names", on a scratch clone of the final tree: "caught (exit
  2)", "plant: all 1 plant(s) caught", exit 0.
* The MADR's row C47now (`AGENTS.md:286`), checked with a plain id on a
  scratch commit of the working tree: "1 of 1 pass"; `tools/claims.py
  --plants`, two plants (its text changed; its line moved), "claims: all 2
  plant(s) caught", exit 0. Quoted in the MADR, A-1.

#### Step 3, the pty run

The PLAN's exact command, from this Mac, ran on the Windows laptop: exit 0,
`--- PASS: TestLive_RawEnterBytes_0087 (3.95s)`. Its three lines, and the
same three read back from `tmp/live-0087/pty.txt`, are quoted whole in the
MADR, "Amendments of 2026-11-22", A-3:

```text
mode=pty console=pty case=crlf-one-write result=split reads=0d,0a
mode=pty console=pty case=lf-after-cr result=split reads=0d,0a
mode=pty console=pty case=cooked-then-raw result=lf-first cooked=0d0a reads=0a,-,0d
```

It yielded lines on the first run, so no fix and rerun was needed. The
mapping (step 5) is not made here.

#### Step 4, the owner's two steps

Not run by the agent. The instructions were written for the owner, outside
the tree, in the session's scratch directory as `sdk-0087/OWNER-STEPS.txt`:
how to put the probe on the Windows laptop, a two-line CR+LF text to paste
(Notepad), the classic console window (`conhost.exe cmd.exe` from Win+R, so
Windows Terminal is not used even where it is the default), then a Windows
Terminal tab, each with `LLMPROVIDER_LIVE_0087=console` and the MADR's `go
test` command with no redirection; the three requests R1 to R3 as the probe
shows them; the files it writes (`tmp/live-0087/conhost.txt`,
`tmp/live-0087/terminal.txt`); and what to tell the coordinator (both
files' three lines each). Step 5 waits for those two files.

#### Checks of the pass (exit statuses captured directly)

| Check | Host | Exit |
| :--- | :--- | :--- |
| the pty probe, step 3's exact command | the Windows laptop, through `run-on-host --need windows` | 0 |
| `run-heavy -- make pre-add-check FILES="llmprovider/live_raw_enter_0087_windows_test.go llmprovider/raw_enter_format_0087_test.go"` | this Mac | 0 ("2 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)") |
| `run-heavy -- go test ./...` | this Mac | 0 (44 packages `ok`, none failed) |
| `GOOS=windows go vet -tags live_gateways ./llmprovider` | this Mac | 0, no output |
| `make dep-check` | this Mac | 0, unchanged (host and `GOOS=windows`: 46 packages, 0 problems) |
| `make api-check` | this Mac | 0 (against `v1.31.0`, 0 incompatible changes) |
| `make tables-check` | this Mac | 0 (49 packages, 0 problems) |
| `tools/plant.py`, the twin's plant | this Mac, scratch clone | 0 (caught) |
| `tools/plant.py`, ST42's new `tables-check` plant | this Mac, scratch clone | 0 (caught; `make tables-check` exit 2) |
| `tools/claims.py --plants`, C47 (plain id, scratch commit) | this Mac, scratch record | 0 (2 of 2 caught) |
| `tools/claims.py` on 0087-MADR | this Mac | 0 (46 of 46) |
| `tools/claims.py` on this PLAN | this Mac | 1: "0 of 0 pass (no evidence rows)"; this PLAN has no evidence index |
| `tools/recordlint.py --changed` | this Mac | 0 (both 0087 records: no new findings) |
| `tools/denyscan.py --changed`, and on `OWNER-STEPS.txt` | this Mac | 0, 0 hits each |
| `make records-check`, scratch clone of `18e8e72` with only this pass's files | this Mac | 0 ("182 records, 0 problem(s)") |
| `make gate-selftest`, the same clone, through `run-heavy` | this Mac | 0 ("Ran 70 tests", "OK") |

`make records-check` and `make gate-selftest` fail in the working tree while
another record's 0088 files are unindexed, so they ran on a clone of
`18e8e72` holding only `AGENTS.md`, `docs/README.md`, the two 0087 records
and the two new test files.
