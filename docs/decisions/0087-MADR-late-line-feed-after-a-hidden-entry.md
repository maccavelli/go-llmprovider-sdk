---
status: accepted
date: 2026-11-20
decision-makers: repository owner
consulted: "the coordinator's brief of 2026-11-20, which relays a reviewer's finding made during the review of 0085-MADR-refuse-an-empty-typed-key.md (its claim C70); this repository at 24756a6, read only, and one scratch extraction of it for the reproduction below; the committed text of 0020-MADR and 0020-PLAN (F17 and its deviation of 2026-10-03), 0081, 0083 and 0085 at 24756a6; prepare-commit-msg at 65d2292, gobble-cli at b1994a0, magic-cli-remote and mcplib, read only; golang.org/x/term v0.43.0, golang.org/x/sys v0.47.0 and the Go 1.27.4 standard library, read only; this Mac's poll(2) manual page; the owner's rulings of 2026-11-20 on the six questions and on live runs, relayed by the coordinator (OR1 to OR7, labels quoted)"
informed: prepare-commit-msg, which runs wizard.ConfigureLLM over a TextPrompter; every caller that uses wizard.TextPrompter or implements wizard.Prompter over a terminal
---

<!-- markdownlint-disable MD013 MD024 -->

# Keep a Hidden Entry's Late Line Feed From Answering the Next Prompt: `TextPrompter` Settles Its Enter Before It Leaves Raw Mode

Associated PLAN: [0087-PLAN-late-line-feed-after-a-hidden-entry.md](0087-PLAN-late-line-feed-after-a-hidden-entry.md)

## Context and Problem Statement

`wizard.TextPrompter` reads a hidden entry (`Secret`) in raw mode on a
terminal and every other answer (`Input`, `Select`, `MultiSelect`,
`Confirm`) in line mode, all through one `bufio.Reader` it keeps across
prompts (C1). A raw entry ends on a carriage return (CR) or a line feed
(LF). When it ends on a CR, it drops an LF only if that LF is already in the
reader's buffer (C11); an LF not yet read "is not waited for: it is the
user's next Enter" (C12). A line-mode read then reads up to the next LF
(C2), trims (C3), and an empty line is the prompt's default (C4 to C7).

So an Enter that reaches the program as CR then LF, with the LF in a later
read than the CR, ends the hidden entry on the CR and leaves the LF for the
next prompt, which answers itself with its default. The reviewer of 0085
found this (C70 in that review), and 0085 keeps it out of its scope; this
record is the follow-up.

### This is the known residual of 0020's F17

0020-MADR's F17 found that raw mode "ends on `\r` and leaves the `\n`
behind, so the next prompt silently takes its default". 0020-PLAN's first
fix remembered the CR and dropped an LF at the start of the next read; review
withdrew it, because after the hidden entry the terminal is back in line
mode, where a POSIX terminal turns the person's next Enter into an LF: the
first fix swallowed that Enter, and the prompt waited for another line.
The fix kept was the buffered rule above, pinned in both directions by
`TestTextPrompter_MaskedEntry` (CR+LF in one read, C19) and
`TestTextPrompter_EnterAfterMaskedEntryIsKept` (CR, then LF in a later read,
taken as the next Enter, C18). The deviation recorded the residual: "A paste
whose `\n` arrives in a later read still answers the next prompt. Telling
that apart from an Enter would need a timer" (C29, C30).

By bytes alone, a late LF and the person's next Enter at a line-mode prompt
are the same input. What tells them apart is when the LF arrives and in which
mode: a CR+LF sent for one Enter arrives together, while the terminal is
still raw; the person's next Enter at the next prompt arrives after that
prompt is printed, after the terminal is restored.

### Reproduction (2026-11-20)

A scratch extraction of `24756a6` (`git archive HEAD`), with one new test
file, `wizard/late_lf_0087_test.go`, run by
`go test ./wizard -run TestTextPrompter_LateLineFeedAfterRawEntry_0087 -count=1 -v`.
The test uses the package's raw-mode seam, `rawSeam0059` (C20), so that
`Secret` runs its raw path on an `os.Pipe` with no terminal. Each case writes
`sk-late-lf-0087-keyAA` and a CR, reads it with `Secret`, delivers an LF in a
later write, then the person's answer, and asks a line-mode question with a
default:

* `Input`: the LF written after `Secret` returned; the answer `my-model`;
  the default `saved-model`.
* `LF at the line end`: the LF written by the prompter's `Out` at the moment
  `readMasked` prints its line end, so it is in the pipe before `Secret`
  returns, as a terminal's CR+LF is.
* `Select`: the LF written after `Secret` returned; the answer `3` of three;
  the default 1.

The output, exit status 1:

```text
=== RUN   TestTextPrompter_LateLineFeedAfterRawEntry_0087
=== RUN   TestTextPrompter_LateLineFeedAfterRawEntry_0087/Input
    late_lf_0087_test.go:64: Input after a CR+LF raw entry = "saved-model", <nil>; want "my-model": the late LF answered with the default
=== RUN   TestTextPrompter_LateLineFeedAfterRawEntry_0087/LF_at_the_line_end
    late_lf_0087_test.go:85: Input after an LF delivered at the line end = "saved-model", <nil>; want "my-model"
=== RUN   TestTextPrompter_LateLineFeedAfterRawEntry_0087/Select
    late_lf_0087_test.go:99: Select after a CR+LF raw entry = 0, <nil>; want 2: the late LF answered with the default
--- FAIL: TestTextPrompter_LateLineFeedAfterRawEntry_0087 (0.00s)
    --- FAIL: TestTextPrompter_LateLineFeedAfterRawEntry_0087/Input (0.00s)
    --- FAIL: TestTextPrompter_LateLineFeedAfterRawEntry_0087/LF_at_the_line_end (0.00s)
    --- FAIL: TestTextPrompter_LateLineFeedAfterRawEntry_0087/Select (0.00s)
FAIL
FAIL	github.com/maccavelli/go-llmprovider-sdk/wizard	0.656s
FAIL
```

The mechanism is real. Two of the three cases (`Input`, `Select`) deliver the
LF after `Secret` has returned: that is byte for byte the input of
`TestTextPrompter_EnterAfterMaskedEntryIsKept`, which 0020 decided must take
the default, and no option below except C changes it. The case that
separates the defect from the person's Enter is `LF at the line end`; the
PLAN's red test is built on it. The test file was not added to this tree.

Not measured: which real terminals deliver one Enter as CR then LF in two
reads. A terminal that sends CR+LF for Return (an emulator set to do so, or
the VT100 line-feed/new-line mode), a CR+LF paste whose two bytes are split
between reads, and a telnet or serial client are the candidates, all
inferred. A local pseudo-terminal usually delivers both bytes of one write in
one read, which the buffered rule already handles.

### What raw mode and line mode do (read in the sources)

* **POSIX.** `golang.org/x/term`'s `MakeRaw` clears `ICANON`, `ECHO`,
  `ISIG` and `ICRNL` and sets `VMIN 1`, `VTIME 0`, with `TCSETS` (Linux) or
  `TIOCSETA` (BSD, macOS), which flushes nothing. In raw mode the Enter key
  sends CR; after `Restore`, `ICRNL` turns the next Enter into LF, and a
  canonical read returns a whole line through its LF, so a line-mode answer
  leaves nothing for a following raw read.
* **Windows console** (inferred, not measured). `MakeRaw` clears
  `ENABLE_ECHO_INPUT`, `ENABLE_PROCESSED_INPUT` and `ENABLE_LINE_INPUT` and
  sets `ENABLE_VIRTUAL_TERMINAL_INPUT`; Enter is then expected to arrive as
  CR alone. In line mode the console returns a line ending in CR+LF. Go's
  console read (`internal/poll/fd_windows.go`, `readConsole`) reads up to the
  caller's buffer, 4096 bytes from `bufio`, so a cooked line and its CR+LF
  are expected in one read; it also ends a read before a Ctrl-Z. What Windows
  Terminal and the classic console send for Enter and for a pasted CR+LF in
  raw mode is not measured: the PLAN's Phase 1 probe settles it, as
  "Windows: the probe and what each observation decides" below maps (ST20,
  ST21).
* **macOS `poll(2)`.** Its manual page's BUGS: "The poll() system call
  currently does not support devices." A readiness check on a terminal there
  uses `select(2)`.

### What a person sees

The hidden entries are the typed API key (C21, asked again after a refusal,
C26) and the pasted credential (C22). What follows them:

* With `Options.Discover`, the model search question (C23, "blank for the
  recommended", C35) answers itself blank, and the recommended menu appears:
  the person never gets to search.
* A Claude key whose listing wants a workspace: the workspace question (C24)
  takes its default, the saved workspace or `ANTHROPIC_WORKSPACE_ID`'s
  value, and spends one of its three tries; when the saved workspace was the
  one just refused, it is tried again and refused again.
* A listing with no models: "No models found; enter a model id" (C25) takes
  the saved model, or, with none, ends the run with "no model available ...
  and none entered".
* A typed key asked again after a refusal (C26): the second `Secret` reads
  the LF as an empty entry and warns "empty value; please try again" (C14),
  then asks again. Visible, not silent.
* `ResolveCredential` returns with the LF still pending: the caller's next
  read of the same input takes it.

### Who is affected

* **prepare-commit-msg** runs `ConfigureLLM` over one `TextPrompter` for the
  whole run (C32), with `Discover` and `NeedFallbacks` (C34), at SDK
  `v1.29.0` (C33): the model search is the question it loses.
* **gobble-cli** (at `b1994a0`) implements its own `Prompter` and
  `InputerContext` and does not use `TextPrompter`; its own input layer has
  the same class of defect and is being fixed under its own record (a shared
  terminal input, one reader per process, CR, LF and CR+LF each ending one
  line, LF then CR counting as two). This record does not fix gobble-cli,
  and that record does not fix this one.
* **magic-cli-remote** requires the SDK and does not use `TextPrompter`.
* **mcplib**'s `wizard` carries its own older copy of `TextPrompter`, used
  by mcp-server-magicdev and mcp-server-magictools: its raw read takes one
  byte at a time from the file and ends on CR or LF, so the same late LF
  stays in the terminal for whatever reads next. It is another repository's
  code and out of this record's scope.

## Decision Drivers

* A question the person never answered must not be answered for them,
  least of all one with a consequential default (the saved model, the saved
  workspace).
* 0020's F17 decision stands unless this record changes it on purpose: the
  person's next Enter at a line-mode prompt after a hidden entry is a blank
  answer, not a byte to drop (C18).
* No read may be left running that could take a later answer
  (`InputerContext`'s contract, C17); `TextPrompter` still cannot give up a
  read (C15).
* Typed-ahead answers and scripted input on a pipe keep working.
* No exported name changes (the `Prompter` interface is a breaking change to
  touch, 0083-MADR C55); documented behaviour changes only where this record
  says so, against `v1.31.0`.
* Windows behaviour is decided from a measurement, not inferred.
* Dependencies stay within AGENTS.md "Dependencies": `wizard` may import
  `golang.org/x/sys` (C28), which `go.mod` lists as indirect today (C27).

## Considered Options

* A. Settle the Enter in raw mode: after a CR, before leaving raw mode, drop
  one LF that is pending or arrives within a short bounded wait
* B. Remember the CR: drop one LF at the start of the next read, whenever it
  arrives
* C. One shared, normalising input: every prompt read in raw mode on a
  terminal, CR, LF and CR+LF each ending one line, LF then CR two
* D. Drain pending input before a line-mode prompt that follows a hidden
  entry
* E. Leave the code, and document the limit

## Decision Outcome

Chosen option: "A. Settle the Enter in raw mode", because it is the timer
0020's deviation named (C30), confined to the one moment the defect happens:
the wait runs while the terminal is still raw, before the next prompt is
printed, so the person's next Enter, which comes after that prompt and, on
POSIX, as an LF only after the restore, is never inside it, and 0020's F17
case keeps its outcome. During the wait an LF can come only from the same
Enter (a CR+LF terminal or paste) or from Ctrl-J.

### Owner rulings (2026-11-20)

Each ruling is the owner's, relayed by the coordinator, quoted by its exact
label. The state table cites them by row.

| Ruling | Question | Label | Rows it decides |
| :--- | :--- | :--- | :--- |
| OR1 | the option | "A: wait briefly for the late line feed and drop it (Recommended)" | ST1 to ST18, ST31 to ST33 (through option A's definition) |
| OR2 | the wait | "50 ms (Recommended)" | ST23 |
| OR3 | Windows | "An agent probes it; no wait on Windows until measured (Recommended)" | ST20, ST21 (the probe and the mapping below), ST22 |
| OR4 | CR then a late NUL | "Drop only a line feed (Recommended)" | ST24 |
| OR5 | the docs | "A sentence in the `Prompter.Secret` doc comment (Recommended)" | ST35 |
| OR6 | when it is built | "In this tag, built last (Recommended)" | the PLAN's order: after 0083, in the release with 0081, 0083, 0084 and 0085 |
| OR7 | live runs (a separate ruling of the same day) | "API keys from the environment only (Recommended)", with the owner's words "script as many live runs as possible and run them, leave for me the minimum possible" | Phase 1's split: the agent runs the probe; the owner runs only the two real-console steps below |
| OR8 | `readMasked`'s signature (the review's finding 1; third pass) | "Keep the signature; the prompter carries the terminal (Recommended)" | ST36 |
| OR9 | where the probe lives (finding 3) | "Put the probe under `llmprovider/` as a `live_*_test.go` (Recommended)" | ST42; the Windows section below |
| OR10 | a CR inside the 50 ms window (finding 6) | "Accept it as a stated loss, like Ctrl-J (Recommended)" | ST6, ST37 |
| OR11 | bytes read ahead that no later prompt reads; Ctrl-C or Ctrl-Z inside the window; a second reader of the terminal (findings 7 and 10) | "Out of scope, stated in the record and docs (Recommended)" | ST38, ST39, ST40 |
| OR12 | ST42's plant, not caught by `tables-check` (Phase 1, 2026-11-22) | "Amend the plant + claims row (Recommended)" | ST42; "Amendments of 2026-11-22", A-1 |
| OR13 | the probe as built: four implementation notes (Phase 1, 2026-11-22) | "Accept all four as deviations (Recommended)" | "Amendments of 2026-11-22", A-2; the PLAN's deviations of that date |
| OR14 | ST22's pseudo-console test, green at the base (Phase 2, 2026-11-22) | "Retarget the Windows test (Recommended)" | ST22; "Amendments of 2026-11-22", A-6 |
| OR15 | ST32's plant, not caught on a pseudo console (Phase 2, 2026-11-22) | "Unit-test the event filter (Recommended)" | ST32; "Amendments of 2026-11-22", A-7 |
| OR16 | the gate self-test anchored on the `wizard` row's old text (Phase 2, 2026-11-22) | "Make the anchor robust" | A6a's amendment of this date; ST41 |
| OR17 | `select` interrupted by a signal (`EINTR`), found by ST3's flaky test (2026-11-23) | "Wait again for the time left (Recommended)" | ST9, ST43; A3's amendment; "Amendment of 2026-11-23" (A-8) |
| OR18 | when the fix of OR17 ships (2026-11-23) | "In v1.33.0 (Recommended)" | the PLAN's Phase 4: its own commit, before the 0088/0089 output-limit commit; both in `v1.33.0` |

### What option A is

* **A1. The settle.** When a raw entry ends on a CR and the reader holds
  nothing after it, `readMasked` prints its line end, then, still in raw mode
  and before the terminal is restored, waits up to W = 50 ms (OR2) for the
  input to become readable, through `select(2)` from `golang.org/x/sys/unix`
  (not `poll(2)`, which macOS does not support on a device). If it becomes
  readable, it peeks one byte through the prompter's reader: an LF is
  discarded; any other byte, a NUL included (OR4), stays in the reader for
  the next prompt.
* **A1a. The prompter carries the terminal (OR8; third pass).** `readMasked`
  keeps its signature, `readMasked(prompt string)`, and every existing call
  of it is unchanged. `secretOnce`, once `makeRaw` has succeeded on the
  `*os.File` that `inTTY` returned, stores that file in an unexported field
  of the `TextPrompter` (proposed name `rawIn`) and clears it, by a deferred
  call that runs before the terminal is restored, when the raw read returns.
  `readMasked` settles only when `rawIn` is set. When the input is not a
  terminal, when raw mode is refused, and when `readMasked` is called
  directly (as ten existing tests do), `rawIn` is `nil` and nothing waits.
* **A2. Only one LF, only after a CR.** An entry ended by an LF is not
  settled. One CR drops at most one LF. A CR already followed in the buffer
  by an LF keeps today's rule (C11) and no wait.
* **A3. A check that fails is nothing pending.** A `select` error (`EINTR`
  included, when a signal such as a window resize interrupts the wait; a late
  LF then leaks, as after W), a descriptor `select` cannot take (negative, as
  a closed file's is, or at or above `FD_SETSIZE`), or no `rawIn` means
  nothing is pending. The check returns its reason as an error, which
  `readMasked` ignores: the entry is returned as
  today. An end of input or a read error met by the peek is kept for the
  next read (`bufio.Reader.Peek` hands an error over only once), which
  answers as it does today (the first end of input answers the prompt's
  default, 0020-MADR F6; an error is returned). *(Annotated 2026-11-23,
  A-8: the Go runtime's own preemption signal interrupts the wait with no
  outside signal, often enough to fail ST3's test; the `EINTR` case is open
  with the owner.)* *(Amended 2026-11-23, OR17, "Wait again for the time
  left (Recommended)": `EINTR` is no longer nothing pending. After it, the
  check calls `select` again with the time left to a deadline fixed before
  the first call, so A5's bound holds; when no time is left, it makes no
  further call and reports nothing pending, with no error. Every other
  `select` error, and a descriptor `select` cannot take, are unchanged. The
  cases are A-8's, row ST43.)*
* **A4. Nothing else changes.** Line-mode reads, the path for an input that
  is not a terminal, and the path when raw mode is refused are unchanged;
  `Ctrl-C`, `Ctrl-U`, Backspace and escape sequences during the entry are
  unchanged.
* **A4a. Stated losses and limits (OR10, OR11; third pass).** Within the 50
  ms: a CR (a second Enter, a double tap, a bounced key) stays in the reader,
  and since a line read does not end on a CR, the next line-mode question
  waits for one more line (an accepted loss, as a Ctrl-J is); a Ctrl-C or
  Ctrl-Z arrives as a plain byte, since raw mode turns signals off. Bytes the
  settle's peek reads ahead (up to the reader's 4096) belong to the
  prompter: a caller that reads the terminal itself after the last prompt
  does not see them. A second reader of the same terminal can take the byte
  between the check and the peek, and the peek then waits for the next
  byte. Each is out of scope, and the docs say so.
* **A5. Bounded, and no read left behind.** `Secret` returns at most W after
  the CR, plus the time to read what is pending; the wait is a readiness
  check, not a read on another goroutine, and nothing reads the input after
  `Secret` returns. `TextPrompter` still does not implement `InputerContext`.
* **A6. API and docs.** No exported name is added or changed; `apidiff`
  against `v1.31.0` reports nothing. `golang.org/x/sys` becomes a direct
  requirement of `go.mod` (allowed, C28). `TextPrompter`'s doc comment says
  what the settle does, and `Prompter.Secret`'s doc comment gains one
  sentence (OR5): "An implementation that reads the entry in raw mode, where
  Enter can arrive as CR then LF, must not leave that LF for the next
  prompt."
* **A6a. What the new import makes false (the review's finding 2; third
  pass).** `wizard` importing `golang.org/x/sys/unix` (and
  `golang.org/x/sys/windows` if Windows is "seen") makes these lines
  false, and the change edits each in the same commit: `docs/architecture.md`'s
  `wizard` row of the package table (C39), which `tables-check` compares
  with `go list`; its line 11 ("indirectly, `golang.org/x/sys`", C41), line
  199 ("only `wizard` uses anything else, `golang.org/x/term`", C42) and
  line 1388 ("The standard library and `golang.org/x/term` only", C43);
  `docs/guides/api-standards.md` R2's text at lines 31 to 32 (C44) and its
  `wizard` row (C36), which `tables-check` also compares; `AGENTS.md:32`
  ("with its indirect `golang.org/x/sys`", C40); and the `TextPrompter`
  doc at `wizard/text_prompter.go:32` ("stdlib plus golang.org/x/term",
  C45). R2 itself already allows it (C28 is `dep-check`'s allowance); the
  edits make the words say what the code imports. *(Fourth pass,
  2026-11-20.)* `tables-check` compares exact import paths
  (`scripts/check_tables.py`, `resolve` and its set differences), and `go
  list` reports `golang.org/x/sys/unix`, not `golang.org/x/sys`: the two
  table rows (architecture's `wizard` row, C39, and api-standards' `wizard`
  row, C36) name `golang.org/x/sys/unix`, plus `golang.org/x/sys/windows`
  on the Windows "seen" branch. The prose lines may still say
  `golang.org/x/sys`. *(Amended 2026-11-22, OR16, "Make the anchor robust": one more
  text the edit of architecture's `wizard` row breaks is the gate self-test
  `test_tables_check_architecture_stale` in `scripts/test_gates.py`, which
  found that row by its whole old text; it now finds the row by its first
  cell, with its assertion unchanged; PLAN deviation of this date.)*
* **A7. Windows (OR3).** The Windows build does no wait until Phase 1's
  probe has run. Its outcome then follows the mapping below with no further
  ruling.

### Windows: the probe and what each observation decides (OR3, OR7, OR9)

*(Rewritten 2026-11-20, third pass, after the independent review's findings 3
to 5 and the owner's ruling OR9.)*

**Where the probe lives (OR9).** `TestLive_RawEnterBytes_0087`, in
`llmprovider/live_raw_enter_0087_windows_test.go`, package
`llmprovider_test`, built with `live_gateways` (and, by its file name,
Windows only), switched on by `LLMPROVIDER_LIVE_0087`, which `AGENTS.md`
"Live tests" names. `scripts/check_tables.py` reads the live variables of
`llmprovider/**/live_*_test.go` only (C37), so the probe is seen there. It
imports only the standard library: it calls kernel32's `SetConsoleMode`,
`GetConsoleMode`, `CreatePseudoConsole`, `ClosePseudoConsole`,
`InitializeProcThreadAttributeList`, `UpdateProcThreadAttribute` and
`CreateProcessW` through `syscall.NewLazyDLL`, and sets raw mode with the
bits `golang.org/x/term`'s `makeRaw` sets (clears `ENABLE_ECHO_INPUT`,
`ENABLE_PROCESSED_INPUT` and `ENABLE_LINE_INPUT`, sets
`ENABLE_VIRTUAL_TERMINAL_INPUT`). *(Amended 2026-11-22: it also calls
`FlushConsoleInputBuffer` and `DeleteProcThreadAttributeList`, through the
same `syscall.NewLazyDLL`; see "Amendments of 2026-11-22", A-2.)* So
`llmprovider` gains no dependency, and
R2's rows are unchanged by the probe; `dep-check` reads untagged builds only
(C38), and the probe does not lean on that. It uses no provider key, no
sign-in, no browser and no network.

**What it prints.** Only allowlisted lines: the mode, the console kind
(`pty`, `conhost`, or `terminal` when `WT_SESSION` is set), a case name, a
result word, and the observed line-ending bytes as hexadecimal, one group per
read (`mode=pty case=crlf-one-write result=together reads=0d0a`); never any
other byte it read. It creates `tmp/live-0087/` under the module root itself
(`tmp/` is ignored and absent in a fresh checkout) and writes its result
lines to `tmp/live-0087/<console kind>.txt`, and to the test's log.

**Its console.** `go test` starts the test binary with no standard input
(the null device) and sends its output to its own (the Go 1.27.4 tree's
`cmd/go/internal/test/test.go`, which sets the binary's output and not its
input), so the probe never
uses the test binary's standard input or output: it opens `CONIN$` and
`CONOUT$` itself, puts `CONIN$` in raw mode, and writes its requests to the
person on `CONOUT$`, so the person sees them on the console and only the
result lines reach the file.

* `LLMPROVIDER_LIVE_0087=pty` (the agent's run). The test creates a pseudo
  console and starts its own test binary as a child attached to it (its
  `CONIN$` is the pseudo console's input); the child puts `CONIN$` in raw
  mode and reads through a `bufio.Reader`. The parent writes into the pseudo
  console's input:
  * P1 `crlf-one-write`: a CR and an LF in one write;
  * P2 `lf-after-cr`: a CR, then an LF in a second write 10 ms later;
  * P3 `cooked-then-raw`: with the child in line mode, `y`, CR, LF, read by
    `ReadString`; then, raw, a few letters and a CR.

  Run by an agent through the fleet's host tooling:
  `~/.agents/skills/available-dev-hosts/scripts/run-on-host --need windows -- sh -c 'LLMPROVIDER_LIVE_0087=pty go test -tags live_gateways ./llmprovider -run "^TestLive_RawEnterBytes_0087$" -count=1 -v'`.
* `LLMPROVIDER_LIVE_0087=console` (the owner's run). The test opens its own
  `CONIN$` and `CONOUT$` and asks, on the console, for R1, one press of
  Enter; R2, one paste of a two-line text copied from a file with CR+LF line
  endings (any text: none of it is printed); and R3, `y` and Enter in line
  mode, then a letter and Enter in raw mode.

**What a pseudo console cannot show,** and so the owner's only steps: a real
keyboard's Enter and a real paste reach a classic console window as key
events from that window, not as bytes written to a pseudo console's input;
and Windows Terminal encodes Enter and treats a paste's line endings itself
before it writes to the pseudo console it hosts. The owner's steps, on the
Windows laptop, in a checkout of the phase's commit, with no redirection:

1. In a classic console window (Command Prompt or PowerShell started outside
   Windows Terminal): `set LLMPROVIDER_LIVE_0087=console` (or
   `$env:LLMPROVIDER_LIVE_0087="console"`), then
   `go test -tags live_gateways ./llmprovider -run "^TestLive_RawEnterBytes_0087$" -count=1 -v`,
   and answer the three requests it shows. It writes
   `tmp/live-0087/conhost.txt`.
2. The same in a Windows Terminal tab. It writes
   `tmp/live-0087/terminal.txt`.

**The mapping.** "Split" means a read that ends with `0d` followed by a read
that begins with `0a`.

| Observation | Outcome, with no further ruling |
| :--- | :--- |
| P1 split | **Late line feed seen on Windows, in a pseudo console.** ST20 is decided: Windows gets the same 50 ms wait by the mechanism below, with ST31 and ST32, and the pseudo-console tests in the PLAN's Phase 2 table |
| P1 not split, and R1 or R2 split in either console | **Late line feed seen on Windows, on a real console only.** A split measured on a real console is a measurement, and OR3's words ("no wait on Windows until measured") then give Windows the wait: ST20 is decided as in the row above. A pseudo console cannot make the Windows tests red at the base, so they are pinned by the owner's real console instead: `TestLive_SettleOnConsole_0087` (`wizard/live_settle_0087_windows_test.go`, `live_gateways`, the same variable set to `settle`), run by the owner in the console that split, once at the base (red: the question after the hidden entry takes its default) and once on the fixed build (green: it takes the answer). Those two runs are the owner's two further steps in this branch only |
| P1, R1 and R2 never split (the `0d0a` together, or `0d` alone, or no `0a` at all) | **Not seen.** ST20 is decided: no wait on Windows; `TextPrompter`'s doc comment gains the sentence "On Windows it does not wait: a Windows console was measured to deliver an Enter's CR and LF in one read." ST31 and ST32 do not arise |
| P2 shows no `0a` reaching the reader (dropped, or turned into `0d`) | an LF cannot reach the reader as a byte; with P1, R1 and R2 not split, "not seen" as above; P2 alone never makes it "seen", since its split is written by the probe |
| P3 (or R3): the raw read begins with the letters typed in raw mode | ST21 is decided: the line read took its whole line, and nothing changes |
| P3 (or R3): the raw read begins with anything else | ST21 is decided: what the raw read meets is type-ahead, as on POSIX; an LF there ends the entry empty, and `Secret` warns and asks again (C14), visibly; this record changes nothing for it, and the bytes are quoted in a dated note here |
| The pseudo console cannot be created on the laptop (P1 to P3 do not run) | R1 to R3 decide alone, by the same rows |
| The probe yields nothing: no result line, or no `tmp/live-0087/` file, in either mode | a probe fault, not an observation: the agent fixes the probe and reruns it once; if it yields nothing again, it is a stop and goes to the owner |

`bufio.Reader.ReadString('\n')` returns only after it has read an LF or met
an error, so a line read cannot leave its own LF behind; P3 checks what
follows it.

**The Windows mechanism, if seen.** `wizard/settle_windows.go`: on the
console input handle, `windows.WaitForSingleObject` in slices until 50 ms
after the CR, and after each wake `PeekConsoleInputW` (kernel32, through
`windows.NewLazySystemDLL`, since `golang.org/x/sys` v0.47.0 does not wrap
it) for the first pending key-down event that carries a character. That
character is an LF: the reader's one-byte peek runs and drops it (A1). Any
other character: nothing is dropped and the wait ends (it is type-ahead,
A1). Only events without a character (a key release, focus, mouse, a
resize): the wait goes on until 50 ms. A failed call, or a handle that is
not a console's, is nothing pending (A3). `wizard` then imports
`golang.org/x/sys/windows` too, and A6's table edits name it.

### Consequences

With option A:

* Good, because a CR+LF Enter whose LF arrives within W no longer answers
  the next question, under every flow above.
* Good, because 0020's F17 case, the person's next Enter at the next prompt,
  still takes the default, and `TestTextPrompter_EnterAfterMaskedEntryIsKept`
  is unchanged.
* Good, because a typed-ahead byte other than an LF is kept, a pipe is
  untouched, and no read is left running.
* Neutral, because every hidden entry ended by a lone CR now returns W later
  (50 ms), which a person does not notice.
* Bad, because an LF that arrives after W still answers the next prompt; the
  residual is narrower, not gone.
* Bad, because a Ctrl-J typed within W after Enter is dropped.
* Bad, because the settle is platform code (a `unix` file and a file for the
  rest), its tests run only where it exists, and Windows waits on Phase
  1's probe.

### Confirmation

* The PLAN's tests, red at the base where a row changes behaviour and green
  after, each with a plant caught on a scratch copy; the existing F17 guards
  unchanged and green.
* `python3 -B tools/apidiff.py v1.31.0` and `make api-check` report no
  change; `make dep-check` and `go mod tidy -diff` are clean with
  `golang.org/x/sys` direct.
* Phase 1's probe output quoted in this MADR as a dated amendment, ST20 and
  ST21 closed by the mapping, before Phase 2's Windows step.

## State Table

"Decided" below means decided by an existing record, or by option A's
definition above (OR1), or by another ruling named in the row. Each decided
row names what pins it and the plant that must turn that red, from the
PLAN's Phase 2 table; a plant is run on a scratch copy of the fixed code.
Rows ST31 and ST32 arise only if Phase 1 maps to "seen".

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST1 | Enter as a lone CR, nothing follows | decided | the entry is returned within W plus the time to read what is pending; the next prompt reads only what the person types next | A1, A5 (OR1) | `TestSecret_SettleIsBounded_0087`; plant: a blocking `Peek` in place of the bounded wait |
| ST2 | CR+LF in one read (a paste, a CR+LF terminal) | decided | the LF is dropped with no wait; the next prompt takes the person's answer | 0020-MADR F17 (0020-PLAN Phase 4), A2 | `TestTextPrompter_MaskedEntry` (existing; plant: the buffered-LF discard (C11) removed); and, for "no wait", `TestSecret_NoWaitWhenNotNeeded_0087`, case `CR+LF together` (plant: the settle runs whatever the reader holds) |
| ST3 | CR, then an LF in a later read, pending when the settle starts or arriving during the 50 ms (whatever sent it, Ctrl-J included) | decided | the LF is dropped; the next `Input`, `Select`, `Confirm` or `Secret` takes the person's answer, with no "empty value" warning | A1 (OR1) | `TestSecret_LateLineFeedDropped_0087` (subtests Input, Select, Confirm, Secret; plant: `readMasked` no longer calls the settle); `TestSecret_LateLineFeedDuringWait_0087` (Unix; *third pass*: it sets the wait to 2 s and writes the LF 10 ms in, so a stalled `-race` run keeps a margin of nearly 2 s; W itself is pinned by ST23's row; plant: the check made once with no wait) *(2026-11-23, A-8: this test fails 25 times in 10000 at `77563af` when the runtime's preemption signal interrupts `select`; the outcome is not met then)* |
| ST4 | CR, then an LF that arrives after W: the person's next Enter at the next prompt, or a late half that missed the wait | decided | the LF is kept and answers the next prompt with its default, as today | 0020-MADR F17 and 0020-PLAN deviation of 2026-10-03; A1 | `TestSecret_NextEnterKept_0087`, and `TestTextPrompter_EnterAfterMaskedEntryIsKept` (existing); plant: option B's rule, a leading LF dropped at the next read |
| ST5 | CR, then a byte other than LF within W (type-ahead) | decided | the byte stays in the reader and is the next prompt's | A1 (OR1) | `TestSecret_TypeAheadKept_0087`; plant: the settle discards whatever byte it peeks |
| ST6 | Two line endings after the entry, arriving after the CR is read (CR+LF then CR+LF; CR then CR) | decided | *(corrected 2026-11-20, third pass)* one LF per CR is dropped. CR+LF then CR+LF: the second ending is a whole line, and the next line-mode question takes its default. CR then CR: the kept CR does not end a line read, so the next line-mode question waits for one more line and takes it, the CR trimmed (an Enter alone then gives its default); a second hidden entry reads the CR as an empty entry and warns | A2 (OR1); OR10 | `TestSecret_SecondLineEndKept_0087` (Unix; plants: the settle drops every CR and LF it finds; `readLine` ends a line on a CR) |
| ST7 | An entry ended by an LF, alone or followed by a CR (LF then CR) | decided | no settle and no wait; a following CR stays: a line-mode prompt trims it, a raw prompt reads an empty entry and warns, as today | A2 (OR1) | `TestSecret_LineFeedEndedEntry_0087` (plant: the settle also runs after an LF and drops a CR); and, for "no wait", `TestSecret_NoWaitWhenNotNeeded_0087`, case `LF-ended` (plant: the settle runs after an LF) |
| ST8 | The input ends while settling | decided | the entry is returned with no error; the next prompt answers its default; the read after that ends the input | A3 (OR1); 0020-MADR F6 | `TestSecret_EndOfInputWhileSettling_0087`; plant: the settle marks the input ended when its peek meets the end |
| ST9 | The settle's check fails or cannot be made (a `select` error, `EINTR` included; a closed file, whose descriptor reads as negative; a descriptor at or above `FD_SETSIZE`) | decided | the entry is returned with no error; any read error is the next read's; a negative descriptor or one at or above `FD_SETSIZE` is never given to `select`; after `EINTR` a late LF leaks, as after W *(2026-11-23, A-8: the `EINTR` outcome is open with the owner, since the runtime's own signal makes it contradict ST3)* *(decided 2026-11-23, OR17: `EINTR` is no longer this row's; it is ST43's)* | A3 (OR1) | `TestSecret_SettleCheckFails_0087` (Unix; subtests "closed", "descriptor 1100"); plants: the settle returns its check's error as `Secret`'s (caught by both subtests, since the check returns an error for a negative descriptor and for one at or above `FD_SETSIZE`); the `FD_SETSIZE` check removed (the `FdSet` index panics) |
| ST10 | Raw mode refused (Git Bash, mintty) | decided | read as a line and trimmed, no settle, as today | A4 (OR1) | `TestSecret_RawMode_0059`, "raw mode refused" (existing); plant: the refused path's line read not trimmed |
| ST11 | A pipe or file, CR+LF line endings | decided | no raw mode and no settle; each line is trimmed, as today | A4 (OR1) | `TestSecret_PipeCRLF_0087`; plant: `readLine` stops trimming |
| ST12 | An injected reader that is not an `*os.File`, reaching `readMasked` directly (tests) | decided | no wait; today's buffered rule only | A3 (OR1) | `TestTextPrompter_EnterAfterMaskedEntryIsKept` (existing); plant: option B's rule |
| ST13 | A line-mode answer followed by a hidden entry (POSIX) | decided | the line read takes its LF; the raw read gets only what follows | A4 (OR1) | `TestSecret_AfterLineAnswer_0087`; plant: `readLine` unreads the LF it read |
| ST14 | `InputerContext` and an abandoned read | decided | `TextPrompter` still does not implement it; the settle is a bounded readiness check and nothing reads after `Secret` returns | A5 (OR1); `wizard/prompter.go` `InputerContext` doc (C17) | `TestSecret_NoReadLeftBehind_0087` (Unix): after `Secret` returns, one byte is written and, 200 ms later, the pipe's pending count still includes it (*fourth pass*: read by `unix.IoctlGetInt` with Linux's `unix.TIOCINQ` or darwin's `FIONREAD` value `0x4004667f`, per the PLAN), so nothing read it; plant: the settle moved to a goroutine with no bound that outlives `Secret` (it reads the byte: the count is 0, whichever goroutine runs first, since no other reader exists until the test reads) |
| ST15 | The wizard after a typed key (both hidden entries go through 0085 D2's `secretKey` helper once 0085 is built, C46): the model search question | decided | with an LF within W, the search question takes the person's query, not a blank | A1 (OR1) | `TestConfigure_SearchAfterLateLineFeed_0087`; plant: `readMasked` no longer calls the settle |
| ST16 | 0085: a blank entry at `Secret` | decided | `TextPrompter.Secret` still never returns a blank value (C14), and 0085's guard stays green | 0085-MADR D2 ("`TextPrompter` is unchanged" in 0085's scope); A4 | `TestTextPrompterRefusesBlank_0085` (built by 0085, before this record, OR6); plant: `Secret` returns an empty entry instead of asking again |
| ST17 | Exported API | decided | no exported name added or changed against `v1.31.0` | A6 (OR1) | `tools/apidiff.py v1.31.0` and `make api-check` (checks; their plant is the gate self-test's) |
| ST18 | Dependencies | decided | `golang.org/x/sys` direct in `go.mod`; nothing else | A6 (OR1); AGENTS.md "Dependencies"; C28 | `make dep-check` and `go mod tidy -diff` (checks; their plant is the gate self-test's) |
| ST19 | Ctrl-C, Ctrl-U, Backspace and escape sequences during the entry | out of scope | unchanged; the settle runs only after a CR has ended the entry | | |
| ST20 | Windows console in raw mode: the bytes of an Enter and of a pasted CR+LF, and whether they split across reads | ~~open: awaits live evidence~~ decided (2026-11-22, A-4) | *(2026-11-22: P1 split in a pseudo console, the mapping's first row: "seen"; Windows gets the 50 ms wait by the Windows mechanism, with ST31 and ST32; R1 and R2 never split on a real console)* each observation is mapped to its outcome under "Windows: the probe and what each observation decides" | OR3 (the probe and the mapping) | Phase 1's probe, P1, P2, R1, R2 |
| ST21 | Windows console: a line-mode answer followed by a hidden entry | ~~open: awaits live evidence~~ decided (2026-11-22, A-4) | *(2026-11-22: R3 began with the letters on both real consoles, the fifth row; P3 began with an LF, the sixth row; both: this record changes nothing)* mapped as ST20 is | OR3 | Phase 1's probe, P3, R3 |
| ST22 | What the Windows build does | decided | no wait until Phase 1 has run; then "seen": the Windows mechanism with W = 50 ms; "not seen": no wait and the doc sentence | OR3; A7; the mapping's second row (third pass) | "seen" in a pseudo console: `TestSecret_LateLineFeedDropped_Windows_0087` (plant: `readMasked` no longer calls the settle on Windows; *retargeted 2026-11-22, OR14, A-6: a second hidden entry after the late LF returns `key2` with no "empty value" warning, and `TestSecret_LineModeUnaffected_Windows_0087` guards that a line-mode question is unaffected*); "seen" on a real console only: `TestLive_SettleOnConsole_0087`, run by the owner in that console, red at the base and green on the fixed build, the base itself standing as its plant, since a pseudo console cannot make it red; "not seen": `TestSecret_SettleIsBounded_0087` run on Windows (plant: the Windows build waits 5 s) |
| ST23 | The length of W | decided | 50 ms | OR2 | a `tools/claims.py` row on the constant, added at Phase 2 (plant: the row's text changed, by `tools/claims.py --plants`) *(2026-11-22: C48now, A-7)* |
| ST24 | CR then NUL (a telnet client's Enter), the NUL late | decided | the NUL is not dropped: it stays in the reader and reaches the next line-mode prompt untrimmed, as today | OR4; A1 | `TestSecret_LateNulKept_0087`; plant: the settle drops a NUL as well as an LF |
| ST25 | CR-only line endings on a pipe | out of scope | not a line ending in line mode today; this record changes line reads in no way (option C would) | | |
| ST26 | Typing ahead before raw mode begins (the terminal echoes it in line mode) | out of scope | not a line-ending case; noted, not examined further | | |
| ST27 | Two goroutines reading one `TextPrompter`; `Notify` during the settle | out of scope | the type's concurrency doc is unchanged; `Notify` writes under the prompter's mutex as before | | |
| ST28 | 0083's browser question, a `Select` with a default | out of scope | no hidden entry precedes it in any flow at `24756a6` (the only `Secret` calls are C21 and C22, which 0085 D2 moves into its `secretKey` helper, C46); re-checked at the phase's start, on 0083's code, as `Secret` called only from `secretKey` and `secretKey` only from the two places C21 and C22 name | | |
| ST29 | 0081's save-last change | out of scope | changes no prompt and no read | | |
| ST30 | mcplib's own `TextPrompter` | out of scope | another repository's copy | | |
| ST31 | Windows, "seen" branch: the settle on a handle that is not a console's (a pipe under `rawSeam0059`) | decided | `PeekConsoleInputW` fails: nothing pending, the entry returned within W | A3 (OR1); OR3 | `TestSecret_SettleIsBounded_0087` run on Windows; plant: a failed `PeekConsoleInputW` treated as an LF pending, so the peek blocks |
| ST32 | Windows, "seen" branch: only events without a character pending during the wait (the CR's key release, focus, mouse, resize) | decided | no read; the wait goes on to 50 ms and returns | the Windows mechanism (OR3) | `TestSecret_SettleIsBounded_Windows_0087`, on a pseudo console with only the CR written; plant: any pending event treated as a character, so the peek blocks *(amended 2026-11-22, OR15, A-7: the plant is caught by `TestSettle_CharPending_Windows_0087`, over built console records; the pseudo-console test stays as a green bound check, since a pseudo console leaves no event without a character pending)* |
| ST33 | A paste longer than the reader's 4096-byte buffer whose CR is the last byte of one fill, the LF in the next | decided | on Unix the LF is in the descriptor, `select` sees it, and it is dropped | A1 (OR1) | `TestSecret_LongPasteSplitAtCR_0087`; plant: `readMasked` no longer calls the settle |
| ST34 | The same on Windows, "seen" branch, where Go's console reader can hold the LF in its own buffer (`readConsole` reads up to 4096 UTF-16 units and hands over 4096 bytes) | out of scope | `PeekConsoleInputW` cannot see Go's buffer, so the LF is kept, as ST4's residual; no test can fill Go's internal buffer at that boundary | | |
| ST35 | `Prompter.Secret`'s doc comment | decided | one sentence added (A6), after 0085's edit to the same comment | OR5; OR6 | a `tools/claims.py` row on the sentence, added at Phase 2; plant: the sentence removed, by `tools/claims.py --plants` *(2026-11-22: C49now, A-7)* |
| ST36 | How the settle reaches the terminal, and `readMasked`'s callers | decided | `readMasked(prompt)` is unchanged, and its ten direct test calls compile and pass unchanged; the settle runs only while `secretOnce` holds its raw-mode file in `rawIn`, which it clears before the restore | OR8; A1a | the ten existing direct callers, unchanged (a compile check); `TestSecret_RawFieldCleared_0087` (Unix): a hidden entry on a pipe under `rawSeam0059`, then `readMasked` called directly on the same prompter, ended by a CR while the line-end hook, armed again, writes an LF, then `ans` and an LF follow: the LF is not dropped, since no raw read is running, and `Input` returns its default (plant: `secretOnce` does not clear `rawIn`, so `Input` returns `ans`) |
| ST37 | A CR within the 50 ms window before a line-mode question (a second Enter, a double tap, a bounced key) | decided | kept; the question does not end on it and waits for one more line: a stated loss | OR10; A4a | `TestSecret_SecondLineEndKept_0087`, subtest `CR then CR` (plant: `readLine` ends a line on a CR) |
| ST38 | Bytes the settle's peek reads ahead that no later prompt of this `TextPrompter` reads (a hidden entry as the last prompt; `ResolveCredential` returning to a caller that reads the terminal itself) | out of scope | lost to that caller; stated in `TextPrompter`'s doc (OR11) | | |
| ST39 | Ctrl-C or Ctrl-Z typed within the 50 ms window | out of scope | arrives as a plain byte for the next prompt, since raw mode turns signals off; stated in the doc (OR11) | | |
| ST40 | A second reader of the same terminal takes the byte between the check and the peek | out of scope | the peek waits for the next byte; beside ST27; stated in the doc (OR11) | | |
| ST41 | The doc and table lines the new import makes false (A6a) | decided | each line A6a names is edited in the same change | A6a; the review's finding 2 | `make tables-check` (plant: `docs/architecture.md`'s `wizard` row left without `golang.org/x/sys`: "imports ..., which its row does not name"); `make docs-check` and Phase 2's review for the prose lines |
| ST42 | Where the probe lives, and its switch | decided | `llmprovider/live_raw_enter_0087_windows_test.go`, standard library only; `AGENTS.md` names `LLMPROVIDER_LIVE_0087` | OR9; OR12 | ~~`make tables-check` (plant: `AGENTS.md`'s name of the variable removed: "a live test reads it, and AGENTS.md does not name it")~~ *(amended 2026-11-22, OR12: that plant is not caught, since `tables-check` admits a variable its own record names; see A-1)* `make tables-check` (plant: the probe reads a variable no document names); `AGENTS.md`'s name of `LLMPROVIDER_LIVE_0087` pinned by the claims row C47now (plant: its text changed, by `tools/claims.py --plants`); `make dep-check` unchanged |
| ST43 | The settle's `select` interrupted by a signal (`EINTR`): the Go runtime's own preemption signal, a resize, any other (added 2026-11-23, A-8) | decided | `select` is called again with the time left to a deadline fixed before the first call; the cases and outcomes are A-8's table, K1 to K7 | OR17; A3's amendment | `TestPendingWithin_Interrupted_0087` (Unix; subtests per case, through the unexported seam `selectFn`); plant: the retry removed (`EINTR` returned as the check's error, as before); ST3's `TestSecret_LateLineFeedDuringWait_0087`, unchanged, 0 failures in 10000 runs |

Counts *(third pass)*: 29 decided, 11 out of scope, 0 open, 2 open awaiting
live evidence (ST20, ST21). *(2026-11-22, A-4: 31 decided, 11 out of scope,
0 open; Phase 1 mapped to "seen", in a pseudo console, so ST31 and ST32
arise.)* *(2026-11-23, A-8, OR17: 32 decided, 11 out of scope, 0 open; ST43
added.)*

## Pros and Cons of the Options

### A. Settle the Enter in raw mode

After a CR ends a raw entry, while still raw, wait up to W for an LF and drop
it (definition above).

* Good, because it separates the same Enter's LF from the next Enter by the
  mode and the moment they arrive in, which is what 0020 said would need a
  timer (C30).
* Good, because type-ahead other than an LF, pipes, the raw-refused path and
  every line-mode read are untouched.
* Good, because nothing is left reading: no goroutine, a bounded wait.
* Good, because the precedent exists: the paste prompt already tells a
  prompter's instant empty answer from a person's by 50 ms (`pasteQuick`, C31;
  0059-MADR W5).
* Neutral, because each hidden entry ended by a lone CR returns W later.
* Bad, because an LF later than W still leaks, and a Ctrl-J within W is
  dropped.
* Bad, because it is platform code (`select` on Unix; Windows waits on a
  measurement), and its tests need a `unix` build tag.

### B. Remember the CR

Set a flag when a raw entry ends on a CR; the next read of any prompt drops
one leading LF, whenever it arrives.

* Good, because it closes the split however late the LF comes, with a few
  lines of portable code.
* Bad, because on POSIX the person's next Enter at a line-mode prompt is an
  LF: it is swallowed, and the prompt waits for another line. 0020-PLAN tried
  exactly this and withdrew it; `TestTextPrompter_EnterAfterMaskedEntryIsKept`
  turns red.
* Neutral, because on Windows, where a cooked Enter is CR+LF, a lone leading
  LF cannot be the person's Enter, so a Windows-only B would be safe there;
  whether Windows needs any fix waits on ST20.
* Type-ahead: kept, except a leading LF. Pipe: untouched (no raw mode).
  Abandoned read: none. API: no names; documented behaviour changes for every
  POSIX person who accepts a default after a hidden entry.

### C. One shared, normalising input, every prompt raw

`TextPrompter` puts the terminal in raw mode for every prompt and reads all
of them through one normalising reader: CR, LF and CR+LF each end one line;
LF then CR are two; line prompts echo and edit for themselves. This is the
model the consumer CLI is adopting in its own input layer.

* Good, because in raw mode the person's Enter is always a CR, so an LF right
  after a CR is part of it whenever it arrives: the split closes completely,
  with no timer and no residual.
* Good, because one input model is shared with the consumer CLI's.
* Bad, because it rewrites every line-mode prompt: echo, Backspace, Ctrl-U
  and Ctrl-W become the prompter's; the terminal's own line editing (and the
  Windows console's history keys) is lost; Ctrl-C at a line prompt becomes an
  error instead of a signal a caller may be handling.
* Bad, because the normaliser also applies to a pipe, where a lone CR would
  start to end a line (ST25).
* Bad, because it is the largest change and the largest test surface, in a
  type twelve repositories can reach.
* Type-ahead: kept. Abandoned read: none. API: no names; documented
  behaviour of every `TextPrompter` method changes.

### D. Drain pending input before a line-mode prompt

Before a line-mode prompt that follows a hidden entry, discard what the
reader buffers and what the terminal holds (`tcflush` on Unix,
`FlushConsoleInputBuffer` on Windows).

* Good, because a late LF already present is gone.
* Bad, because it discards legitimate type-ahead and the rest of a paste.
* Bad, because an LF arriving after the drain still leaks: it narrows the
  window no better than A, and loses input A keeps.
* Bad, because it must never run on a pipe (scripted answers), so it is
  terminal-only platform code as A is.
* Abandoned read: none. API: no names; documented behaviour changes.

### E. Leave the code, and document the limit

`TextPrompter`'s doc and the README say that a terminal which delivers one
Enter as CR then LF in two reads can make the question after a hidden entry
take its default.

* Good, because nothing changes, nothing can break, and 0020's decision
  stands as written.
* Bad, because the defect stays: a question the person never saw answered
  can be answered for them, including a saved model or workspace.
* Type-ahead, pipe, abandoned read, API: unchanged.

## More Information

### Evidence index

<!-- claims: self=24756a6 prepare-commit-msg=65d2292 -->

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C1 | `TextPrompter` keeps one reader across prompts | `wizard/text_prompter.go:57` | `reader *bufio.Reader` |
| C2 | a line-mode read reads to an LF | `wizard/text_prompter.go:155` | `line, err := p.bufReader().ReadString('\n')` |
| C3 | and trims the line | `wizard/text_prompter.go:156` | `trimmed := strings.TrimSpace(line)` |
| C4 | `Input`'s empty line is its default | `wizard/text_prompter.go:359` | `return def, nil` |
| C5 | `Select`'s empty line is its default | `wizard/text_prompter.go:244` | `return defaultIdx, p.flushErr()` |
| C6 | `Confirm`'s empty line is its default | `wizard/text_prompter.go:331` | `case "":` |
| C7 | `MultiSelect`'s empty line keeps the preselection | `wizard/text_prompter.go:285` | `return preselected, p.flushErr()` |
| C8 | raw mode only on a terminal | `wizard/text_prompter.go:415` | `f, isTTY := p.inTTY()` |
| C9 | raw mode is entered | `wizard/text_prompter.go:424` | `state, err := makeRaw(fd)` |
| C10 | the hidden entry is read masked | `wizard/text_prompter.go:441` | `return p.readMasked(prompt)` |
| C11 | an LF after the CR is dropped only when buffered | `wizard/text_prompter.go:488` | `if c == '\r' && r.Buffered() > 0 {` |
| C12 | an LF not yet read is left | `wizard/text_prompter.go:487` | `read is not waited for: it is the user's next Enter.` |
| C13 | the line end is printed before return | `wizard/text_prompter.go:495` | `p.printf("\r\n")` |
| C14 | `Secret` asks again on an empty entry | `wizard/text_prompter.go:410` | `p.Notify(LevelWarn, "empty value; please try again")` |
| C15 | `TextPrompter` does not implement `InputerContext` | `wizard/text_prompter.go:44` | `It does not implement InputerContext: a read of stdin cannot be given up,` |
| C16 | the terminal functions are variables, for the test seam | `wizard/text_prompter.go:124` | `isTerminal  = term.IsTerminal` |
| C17 | `InputerContext` leaves no read running | `wizard/prompter.go:110` | `with an error, and leaves no read running that could take the user's next` |
| C18 | 0020's guard: a later LF is the next Enter | `wizard/remediation_test.go:136` | `p := &TextPrompter{In: &chunkReader{chunks: []string{"sk-abcdefgh\r", "\n", "3\n"}}, Out: io.Discard}` |
| C19 | 0020's guard: CR+LF in one read | `wizard/remediation_test.go:109` | `p := &TextPrompter{In: strings.NewReader("sk-abcdefgh\r\n3\n"), Out: io.Discard}` |
| C20 | the raw-mode seam | `wizard/edges_0059_test.go:59` | `func rawSeam0059(t *testing.T, rawErr, restoreErr error) (made, restored *int) {` |
| C21 | the typed API key is a hidden entry | `wizard/configure.go:856` | `key, err := p.Secret(fmt.Sprintf("Enter your %s API key", d.Label))` |
| C22 | the pasted credential is a hidden entry | `wizard/auth.go:604` | `value, err := p.Secret(fmt.Sprintf("Paste your %s credential", d.Label))` |
| C23 | the model search question | `wizard/model_select.go:55` | `q, err := p.Input(searchModelsPrompt, "")` |
| C24 | the workspace question, with a default | `wizard/configure.go:522` | `workspace, wsErr := p.Input(workspacePrompt, workspaceDefault(o, d, workspaces))` |
| C25 | the model id question when nothing is listed | `wizard/configure.go:340` | `manual, inputErr := p.Input("No models found; enter a model id", existingModel(o, d.ID))` |
| C26 | a refused typed key is asked for again | `wizard/configure.go:563` | `key, keyErr := promptAPIKey(p, d, o)` |
| C27 | `golang.org/x/sys` is indirect today | `go.mod:10` | `require golang.org/x/sys v0.47.0 // indirect` |
| C28 | `wizard` may import `golang.org/x/sys` | `scripts/check_deps.py:35` | `f"{MODULE}/wizard": ("golang.org/x/term", "golang.org/x/sys/"),` |
| C29 | 0020 recorded the residual | `docs/decisions/0020-PLAN-remediate-v1-debugging-pass-findings.md:614` | `arrives in a later read still answers the next` |
| C30 | and named the timer | `docs/decisions/0020-PLAN-remediate-v1-debugging-pass-findings.md:615` | `Telling that apart from an Enter would need a timer.` |
| C31 | the paste prompt's 50 ms precedent | `wizard/auth.go:939` | `50 * time.Millisecond` |
| C32 | prepare-commit-msg: one `TextPrompter` for the run | `prepare-commit-msg:internal/ui/setup.go:228` | `prompter := &wizard.TextPrompter{In: in, Out: os.Stdout}` |
| C33 | prepare-commit-msg requires `v1.29.0` | `prepare-commit-msg:go.mod:6` | `github.com/maccavelli/go-llmprovider-sdk v1.29.0` |
| C34 | prepare-commit-msg lists models | `prepare-commit-msg:internal/ui/setup.go:278` | `Discover:        true,` |
| C35 | the search question's blank is the recommended menu | `wizard/model_select.go:29` | `blank for the recommended` |
| C36 | api-standards R2's `wizard` row names only x/term | `docs/guides/api-standards.md:66` | `` `internal/redact`, `golang.org/x/term` `` |
| C37 | the table check reads live variables under `llmprovider/` only | `scripts/check_tables.py:195` | `glob("**/live_*_test.go")` |
| C38 | dep-check lists builds with no build tags | `scripts/check_deps.py:69` | `go_list(goos, "-deps", "-test", "-f"` |
| C39 | architecture's `wizard` row names only x/term | `docs/architecture.md:154` | `` `internal/redact`, `golang.org/x/term` `` |
| C40 | AGENTS.md calls x/sys indirect | `AGENTS.md:32` | `` The standard library and `golang.org/x/term` (with its indirect `` |
| C41 | architecture's opening calls x/sys indirect | `docs/architecture.md:11` | `` and, indirectly, `golang.org/x/sys` `` |
| C42 | architecture's package graph names only x/term | `docs/architecture.md:199` | `` only `wizard` uses anything else, `golang.org/x/term` `` |
| C43 | architecture's dependency rule | `docs/architecture.md:1388` | `` The standard library and `golang.org/x/term` only `` |
| C44 | R2's text | `docs/guides/api-standards.md:31` | `` Only `wizard` may leave the standard library, and then only for `` |
| C45 | the `TextPrompter` doc names only x/term | `wizard/text_prompter.go:32` | `stdlib plus golang.org/x/term` |
| C46 | 0085 D2 puts both hidden entries in one helper | `docs/decisions/0085-MADR-refuse-an-empty-typed-key.md:264` | `` One helper, proposed name `secretKey` `` |

### Relation to other records

* 0020-MADR F17 and 0020-PLAN's deviation of 2026-10-03: this record keeps
  F17's decided outcome for the person's next Enter (ST4) and adds the timer
  that deviation named (OR1).
* 0085-MADR-refuse-an-empty-typed-key.md: leaves `TextPrompter` unchanged in
  its own scope and points here (its row ST69). This record keeps
  `Secret`'s refusal of a blank entry (ST16) and is built after it (OR6).
* 0083-MADR-let-the-wizard-sign-in-without-a-callback-listener.md: its new
  question is a `Select` with a default that no hidden entry precedes (ST28).
* 0081-MADR-a-cancelled-sign-in-saves-nothing.md: no prompt or read changes
  (ST29).
* gobble-cli's own record on its terminal input fixes gobble-cli's layer, not
  this one.

### Amendments of 2026-11-22 (Phase 1, pass 1)

Each ruling is the owner's, relayed by the coordinator, by its exact label.

**A-1. ST42's plant (OR12, "Amend the plant + claims row (Recommended)").**
Found in Phase 1 step 2: `scripts/check_tables.py`'s `check_live_variables`
admits a variable `AGENTS.md` does not name when it matches
`^LLMPROVIDER_LIVE_(\d{4})` and a record of that number names it, as
`AGENTS.md` "Live tests" says. This record names `LLMPROVIDER_LIVE_0087`, so
ST42's plant ("`AGENTS.md`'s name of the variable removed") left
`make tables-check` at exit 0 under `tools/plant.py`: not caught. ST42 now
has two pins:

* `make tables-check`, with the plant "the probe reads a variable no
  document names" (the probe's `os.Getenv("LLMPROVIDER_LIVE_0087")` given
  a suffix that no document holds; the plant's spec names it, and no 0087
  record may, since a record of the number that names a variable admits
  it), which proves the check reads the probe's file;
* the claims row below, which proves `AGENTS.md` names the variable.

The row's id ends in "now", so that `tools/claims.py` does not read it
against this record's `self=24756a6`, where `AGENTS.md` does not hold it
(the convention of 0085-MADR's re-pointed rows). It is checked with a plain
id on a scratch commit of the working tree; Phase 2's re-grounding checks it
again at its base.

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C47now | AGENTS.md names the probe's switch | `AGENTS.md:286` | `` `LLMPROVIDER_LIVE_0087`: `pty` is an agent's run `` |

Seen on 2026-11-22: `tools/claims.py` on a scratch record holding this row
with the plain id C47 and `self=` a scratch commit of the working tree
(`--repo self=<scratch clone>`): "1 of 1 pass", exit 0. `tools/claims.py
--plants` on it, exit 0, both caught:

```text
== C47's text changed: caught (FAIL C47 AGENTS.md:286: line 286 is "`LLMPROVIDER_LIVE_0087`: `pty` is an agent's run, on pseudo consoles the test")
== C47's line moved: caught (FAIL C47 AGENTS.md:285: line 285 is "0087's Windows probe, `TestLive_RawEnterBytes_0087`, is switched on by")
claims: all 2 plant(s) caught
```

The `tables-check` plant, on a scratch clone of the final tree of pass 1 by
`tools/plant.py`: caught, `make tables-check` exit 2, with the line
"check-tables: AGENTS.md: <the planted name>: a live test reads it, and
AGENTS.md does not name it" (the name elided here, for the reason above).

**A-2. Two kernel32 calls beyond the list (OR13, "Accept all four as
deviations (Recommended)").** The probe also calls `FlushConsoleInputBuffer`,
in `console` mode only, before each request, so keys pressed between two
requests are not counted in the next case; and `DeleteProcThreadAttributeList`,
to free the attribute list after each pseudo console's child is started.
Both go through the same `syscall.NewLazyDLL("kernel32.dll")`: the standard
library only, and no dependency, still hold. The PLAN's deviation entries of
this date record the other notes the same ruling accepts.

**A-3. Phase 1's pty probe (PLAN Phase 1 step 3).** Run by an agent, on
the Windows laptop, with the PLAN's exact command, from this Mac:

```text
~/.agents/skills/available-dev-hosts/scripts/run-on-host --need windows -- sh -c 'LLMPROVIDER_LIVE_0087=pty go test -tags live_gateways ./llmprovider -run "^TestLive_RawEnterBytes_0087$" -count=1 -v'
```

Exit 0. The test's output, whole (the host tooling's own lines left out: the
host's probe and claim, the copy, and the queue):

```text
=== RUN   TestLive_RawEnterBytes_0087
    live_raw_enter_0087_windows_test.go:365: mode=pty console=pty case=crlf-one-write result=split reads=0d,0a
    live_raw_enter_0087_windows_test.go:365: mode=pty console=pty case=lf-after-cr result=split reads=0d,0a
    live_raw_enter_0087_windows_test.go:365: mode=pty console=pty case=cooked-then-raw result=lf-first cooked=0d0a reads=0a,-,0d
    live_raw_enter_0087_windows_test.go:365: tmp/live-0087/pty.txt: mode=pty console=pty case=crlf-one-write result=split reads=0d,0a
    live_raw_enter_0087_windows_test.go:365: tmp/live-0087/pty.txt: mode=pty console=pty case=lf-after-cr result=split reads=0d,0a
    live_raw_enter_0087_windows_test.go:365: tmp/live-0087/pty.txt: mode=pty console=pty case=cooked-then-raw result=lf-first cooked=0d0a reads=0a,-,0d
--- PASS: TestLive_RawEnterBytes_0087 (3.95s)
PASS
ok  	github.com/maccavelli/go-llmprovider-sdk/llmprovider	3.972s
```

`tmp/live-0087/pty.txt`, as the probe read it back (the host tooling
removes its work root after a run that exits 0, so the file is read back
into the log by the probe itself; PLAN deviation of 2026-11-22):

```text
mode=pty console=pty case=crlf-one-write result=split reads=0d,0a
mode=pty console=pty case=lf-after-cr result=split reads=0d,0a
mode=pty console=pty case=cooked-then-raw result=lf-first cooked=0d0a reads=0a,-,0d
```

What the lines say, read by the definitions of "What it prints" and "The
mapping" above, and nothing more: P1, a CR and an LF written in one write,
reached the reader as two reads, `0d` then `0a` ("split"); P2, the LF
written 10 ms after the CR, did too, so an LF can reach the reader as a
byte; P3's line read returned its line through `0d0a`, and the raw read that
followed began with an `0a`, before the letters (`-`) and their `0d`.

Not yet mapped: the PLAN's step 5 maps the observations when the owner's two
console files (step 4) are in, so ST20 and ST21 stay open here and the
Windows branch of Phase 2 is not yet fixed. *(Annotated 2026-11-22: mapped
in A-4.)*

**A-4. The owner's console runs, and the mapping (PLAN Phase 1 steps 4 and
5, 2026-11-22).** The owner ran the probe's `console` mode on the Windows
laptop, at the commit that carries Phase 1 (`751e970`), with a runner the
coordinator scripted: a classic console window with `WT_SESSION` cleared,
then a Windows Terminal PowerShell tab, the clipboard holding `one`, CR, LF,
`two`, CR, LF, pasted with Ctrl+V. The coordinator read the two files over
ssh and relayed them verbatim.

A first run is superseded and maps nothing: its inputs were wrong, by the
owner's own account ("i goofed those tests passed"). The second run's
files, written at 13:04:07 (`conhost.txt`) and 13:05:08 (`terminal.txt`):

```text
--- conhost.txt
mode=console console=conhost case=enter result=cr-only reads=0d
mode=console console=conhost case=paste result=cr-only reads=0d0d
mode=console console=conhost case=cooked-then-raw result=letters-first cooked=0d0a reads=-
--- terminal.txt
mode=console console=terminal case=enter result=cr-only reads=0d
mode=console console=terminal case=paste result=cr-only reads=-,0d0d
mode=console console=terminal case=cooked-then-raw result=letters-first cooked=0d0a reads=-
```

What each line says, by the probe's code (`rawEnterRun0087`,
`rawEnterLine0087`, `rawEnterResult0087`): each comma-separated group in
`reads=` is one read of the console, and holds that read's CR and LF bytes
in order; `-` is a read that held other bytes and no CR or LF; a case ends
1.5 s after its last read.

* conhost, R1 (`reads=0d`): one Enter reached the reader as one read of one
  CR. No LF.
* conhost, R2 (`reads=0d0d`): the two-line paste reached the reader in one
  read whose only line-ending bytes were two CRs: each CR+LF arrived as a
  CR. No LF, so no split.
* conhost and terminal, R3 (`cooked=0d0a reads=-`): the line-mode read
  returned its line through CR, LF; the raw read that followed began with the
  letter typed (`letters-first`) and held no CR. The owner's report, "the
  last step, type 1 letter then hit enter, when i type the letter the
  terminal ends, i do not have to press enter", fits: the case ended 1.5 s
  after the letter's read, before any Enter. The request's words ("type one
  letter and press Enter") do not match the probe's window; a follow-up for
  the probe's request and the owner's steps, which changes no mapping, since
  R3's row reads only what the raw read begins with.
* terminal, R1 (`reads=0d`): as conhost's R1.
* terminal, R2 (`reads=-,0d0d`): the paste reached the reader in two reads:
  the first held no CR or LF, the second two CRs. No LF in either: Windows
  Terminal also turned each CR+LF into a CR. No split, since "split" needs a
  read that begins with `0a`.

**The mapping,** with the pty run of A-3, by the table under "Windows: the
probe and what each observation decides":

* P1 (`crlf-one-write result=split reads=0d,0a`) is split: the table's first
  row, "Late line feed seen on Windows, in a pseudo console". ST20 is decided:
  Windows gets the same 50 ms wait by "The Windows mechanism, if seen", with
  ST31 and ST32, and the pseudo-console tests of the PLAN's Phase 2 table.
  The first row turns on P1 alone; R1 and R2 never split (a lone `0d`, and
  CR-only pastes with no `0a`), which would decide "not seen" only with P1
  not split (the third row), so it does not apply. P2 shows an `0a` reaching
  the reader, so the fourth row does not apply.
* R3 on both consoles begins with the letters: the fifth row, "the line read
  took its whole line, and nothing changes".
* P3 (`cooked-then-raw result=lf-first cooked=0d0a reads=0a,-,0d`) begins
  with an `0a`: the sixth row, "what the raw read meets is type-ahead, as on
  POSIX; an LF there ends the entry empty, and `Secret` warns and asks again
  (C14), visibly; this record changes nothing for it". The bytes quoted here,
  as that row asks: the line read returned `0d0a`, and the raw read began
  with the `0a` of the probe's one write of `y`, CR, LF, after the console
  had ended the line on the CR; then the letters (`-`) and their `0d`.
* ST21 is decided: the fifth and sixth rows give the same outcome, that this
  record changes nothing for a line-mode answer followed by a hidden entry on
  Windows. On the two real consoles a line-mode Enter left nothing behind;
  in a pseudo console an LF written after the CR stays as type-ahead, which
  `Secret` meets visibly.
* Phase 2's Windows branch is fixed: **"seen", in a pseudo console**.
  `wizard/settle_windows.go` and `wizard/late_lf_0087_windows_test.go`
  (`TestSecret_LateLineFeedDropped_Windows_0087`,
  `TestSecret_SettleIsBounded_Windows_0087`) are built; the real-console-only
  branch (`TestLive_SettleOnConsole_0087`) does not arise, and the "not seen"
  doc sentence is not written.

**A-5. The evidence index re-grounded at Phase 2's base, `1e3ea5d`
(2026-11-22).** `python3 -B tools/claims.py
docs/decisions/0087-MADR-late-line-feed-after-a-hidden-entry.md --rev
self=1e3ea5d`: 36 of 46 pass, exit 1. The ten rows that moved, and one new
(C21bnow), each at its line at `1e3ea5d`. The ids end in "now" so that `tools/claims.py` does not
read them against this record's `self=24756a6`; with plain ids and
`self=1e3ea5d`, on a scratch copy, 11 of 11 pass, exit 0.

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C17now | `InputerContext` leaves no read running | line | `wizard/prompter.go:122` | `with an error, and leaves no read running that could take the user's next` |
| C21now | the typed API key is a hidden entry, through 0085 D2's `secretKey` | text | `wizard/configure.go:1018` | `key, err := secretKey(ctx, p, d, fmt.Sprintf("Enter your %s API key", d.Label), "enter API key")` |
| C21bnow | `secretKey` is the one caller of `Secret` | new | `wizard/configure.go:1051` | `value, err := p.Secret(prompt)` |
| C22now | the pasted credential is a hidden entry, through `secretKey` | text | `wizard/auth.go:617` | `value, err := secretKey(ctx, p, d, fmt.Sprintf("Paste your %s credential", d.Label), "enter credential")` |
| C24now | the workspace question, with a default | line | `wizard/configure.go:647` | `workspace, wsErr := p.Input(workspacePrompt, workspaceDefault(o, d, workspaces))` |
| C25now | the model id question when nothing is listed | line | `wizard/configure.go:436` | `manual, inputErr := p.Input("No models found; enter a model id", existingModel(o, d.ID))` |
| C26now | a refused typed key is asked for again | text (ctx added) | `wizard/configure.go:688` | `key, keyErr := promptAPIKey(ctx, p, d, o)` |
| C31now | the paste prompt's 50 ms precedent | line | `wizard/auth.go:1096` | `50 * time.Millisecond` |
| C36now | api-standards R2's `wizard` row names only x/term | line | `docs/guides/api-standards.md:67` | `` `internal/redact`, `golang.org/x/term` `` |
| C42now | architecture's package graph names only x/term | line | `docs/architecture.md:200` | `` only `wizard` uses anything else, `golang.org/x/term` `` |
| C43now | architecture's dependency rule | line | `docs/architecture.md:1495` | `` The standard library and `golang.org/x/term` only `` |

The re-checks of "Relation to Sibling Records" and ST28, at `1e3ea5d`:
`Prompter.Secret` is called only from `secretKey` (`wizard/configure.go:1051`,
the one non-test call), and `secretKey` only from the two places C21 and C22
named (`wizard/configure.go:1018`, `wizard/auth.go:617`); 0083's question
(`askBrowserFlow`, `wizard/auth.go:921`) is asked before a browser sign-in,
which neither hidden entry precedes. The `TextPrompter` doc comment
(`wizard/text_prompter.go:32` to `:46`), `secretOnce` (`:414` to `:442`),
`readMasked`'s CR case (`:484` to `:496`) and `Prompter.Secret`'s doc comment
as 0085 left it (`wizard/prompter.go:101` to `:108`) hold the text this
record's change starts from; every `wizard/text_prompter.go` claim (C1 to
C16, C45) passes unmoved.

**A-6. Where a late LF shows on Windows, and ST22's test retargeted (OR14,
"Retarget the Windows test (Recommended)", 2026-11-22).** Found at Phase 2's red run: on a
pseudo console at `1e3ea5d`, after `Secret` ends on the CR, an LF that
arrives 10 ms later, or in the same write as the CR (P1's shape), does not
answer the line-mode question that follows: `Input` returned the answer
written after it, `my-model`, not its default. A line-mode (cooked) console
read takes a leading LF without ending a line on it. The same late LF before
a second hidden entry is read by that raw read as an empty entry, and
`Secret` warns "empty value" once (C14). Both were seen on the Windows
laptop, on scratch copies, never in the tree ("EXP one write: key=...
input=my-model"; "EXP second secret: k1ok=true k2=key2 warnings=1"). So on
Windows the late LF is not silent: its effect is the visible warning of a
second hidden entry, which a typed key refused and asked for again meets
(C26). The mapping's outcome (A-4) stands: Windows gets the settle,
`wizard/settle_windows.go`, which drops the LF before the second entry.

ST22's pseudo-console test is retargeted, by the owner's ruling:
`TestSecret_LateLineFeedDropped_Windows_0087` writes a key and a CR, an LF
10 ms later, then a second hidden entry, `key2` and a CR, and wants `key2`
and no "empty value" warning (red at the base, on the Windows laptop:
`child: "key=sk-late-lf-0087-keyAA key2=key2 warnings=1"; want "...
warnings=0"`). The former `Input` form is kept as a guard,
`TestSecret_LineModeUnaffected_Windows_0087`, green at the base and after,
that a line-mode question is unaffected on Windows.

**A-7. ST32 pinned by a unit test of the event filter (OR15, "Unit-test the event filter (Recommended)",
2026-11-22), and the Phase 2 claims rows.** Found at Phase 2's plants: with the
plant "any pending event treated as a character",
`TestSecret_SettleIsBounded_Windows_0087` stayed green on the Windows laptop:
after the raw read takes the CR, a pseudo console leaves no event without a
character pending, even when the CR is written as a win32-input-mode key-down
and key-up (a scratch experiment, never in the tree). By the owner's ruling,
the filter is an unexported pure function of `wizard/settle_windows.go`,
`charPending`, and `TestSettle_CharPending_Windows_0087` runs it over built
console records: a key release with its character, a key-down with none, and
focus, mouse, menu and resize events are false; a key-down with an LF, or
with a letter after a release, is true. The plant turns it red. The
pseudo-console test stays, green, as the bound check of ST32's outcome.

The claims rows of ST23 and ST35. The ids end in "now", so that
`tools/claims.py` does not read them against this record's `self=24756a6`;
each is checked with a plain id on a scratch commit of the working tree, and
planted false by `tools/claims.py --plants` ("2 of 2 pass", exit 0; both plants, W changed and the sentence removed, caught, exit 0).

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C48now | W is 50 ms (ST23, OR2) | `wizard/text_prompter.go:98` | `var settleWait = 50 * time.Millisecond` |
| C49now | `Prompter.Secret`'s doc names the late LF (ST35, OR5) | `wizard/prompter.go:108` | `where Enter can arrive as CR then LF, must not leave that LF for the` |

### Amendment of 2026-11-23: the Go runtime's own signal interrupts the wait (A-8; open, with the owner)

*(Decided 2026-11-23 by OR17 and OR18; see "The rulings" and "The cases the
retry decides" at the end of this amendment.)*

**What A3 asserts, and what was measured.** A3 counts a `select` error,
"`EINTR` included, when a signal such as a window resize interrupts the
wait", as nothing pending, so that a late LF leaks "as after W"; ST9 decides
the same. The assumption under it is that an interruption needs a signal
from outside, which is rare. It does not: the Go runtime's own asynchronous
preemption signal (`SIGURG`) interrupts the wait in an ordinary process, with
no outside signal, often enough to fail ST3's own test.

**The defect it caused.** `TestSecret_LateLineFeedDuringWait_0087` (ST3)
failed twice in about 23 full `go test` runs of the working tree (once in
`make coverage-check`), with
`late_lf_0087_unix_test.go:109: Input after an LF 10 ms into the wait = "saved-model", <nil>; want my-model`.

**The reproduction,** on a scratch clone of `77563af` (the `v1.32.0` code;
`wizard/` unchanged since), this Mac, idle: the wizard's test binary built
once (`go test -c ./wizard/`), then run with
`-test.run '^TestSecret_LateLineFeedDuringWait_0087$' -test.count=N`:

| Run | Failures |
| :--- | :--- |
| N = 2000 | 3 |
| N = 10000 | 25 (0.25 %) |
| N = 10000, `GODEBUG=asyncpreemptoff=1` | 0 |
| N = 5000, a second scratch clone whose `settle` prints `pendingWithin`'s result when it is not "ready" (never in the tree) | 15; 15 printed lines, each `ready=false err=interrupted system call`, between 20 µs and 650 µs into a 2 s wait |
| N = 3000 with `-race`, the Linux machine (not WSL), through `run-on-host --need linux --race` | 0 |

Every failure is an `EINTR` from `select(2)`, and switching off the
runtime's asynchronous preemption removes them all. The Linux run under
`-race` showed none in 3000; a Linux rate without `-race` was not measured.
The rate in a real terminal was not measured.

**The interleaving.** `readMasked` reads the CR with nothing buffered after
it and calls `settle` (`wizard/text_prompter.go:548` to `:549`);
`pendingWithin` enters `select(2)` with the 2 s wait
(`wizard/settle_unix.go:31`). A preemption signal the runtime requested
while the goroutine was still running Go code (`preemptM`, from sysmon's 10
ms time slice, `runtime/proc.go:6715` to `:6716` of Go 1.27.4, or from the
garbage collector) is delivered while the thread is in `select`, which
returns `EINTR` (`select` is not restarted after a handler). `pendingWithin`
returns the error (`wizard/settle_unix.go:32` to `:33`), `settle` returns as
if nothing were pending (`wizard/text_prompter.go:591`), and `Secret`
returns. The LF the test writes 10 ms after the line end then reaches the
next `Input`, a line-mode read, as an empty line: its default. This is A3's
decided path, met far more often than A3 assumed, and it contradicts ST3's
outcome ("arriving during the 50 ms ... the LF is dropped"), which ST3's
test asserts.

**What it means.** The cause is the production code, as A3 and ST9 decided
it, not the test: the test asserts ST3 exactly, and fails only when
`select` is interrupted. ST3 and ST9 cannot both hold as written once the
runtime's own signal is a routine source of `EINTR`, so ST9's `EINTR` outcome
is open again until the owner rules. Its resolutions are in the PLAN's
deviation of this date. Nothing was changed in the code or the tests.

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C50now | the readiness check returns `select`'s error, `EINTR` included | `wizard/settle_unix.go:31` | `n, err := unix.Select(fd+1, &set, nil, nil, &tv)` |
| C51now | the settle treats any error as nothing pending | `wizard/text_prompter.go:591` | `if err != nil \|\| !ready {` |

The two rows hold at `77563af`, checked with plain ids on a scratch record
(`self=77563af`): "2 of 2 pass", exit 0; `tools/claims.py --plants`, C50's
text changed and C51's line moved, "claims: all 2 plant(s) caught", exit 0.

**The rulings (2026-11-23).** Each is the owner's, relayed by the
coordinator, by its exact label; A-8 is decided by them.

* OR17: "Wait again for the time left (Recommended)": after `EINTR`,
  `pendingWithin` calls `select` again with the time left to a deadline fixed
  before the first call, so A5's bound holds; A3 and ST9 are amended (A3's
  annotation of this date; ST9's row; the new row ST43); one deterministic
  test through an unexported seam on the `select` call, whose plant removes
  the retry; accepted when the reproduction above shows 0 failures in 10000
  on this Mac, with a Linux run.
* OR18: "In v1.33.0 (Recommended)": the fix is its own commit, landing
  before the 0088/0089 output-limit commit, and both ship in `v1.33.0`.

**The cases the retry decides (ST43),** each outcome by OR17, before any
code. "The deadline" is the moment the first `select` starts plus the wait,
on the monotonic clock; "the time left" is the deadline less now.

| Case | Input or state | Outcome | Pinned by (`TestPendingWithin_Interrupted_0087` subtest, unless named) |
| :--- | :--- | :--- | :--- |
| K1 | `EINTR`, then the input readable within the time left | `true`, no error; the LF is dropped, `Secret` returns the key and the next `Input` the answer | `EINTR, then the LF` (through `Secret` and `Input`) |
| K2 | `EINTR`, then `select` again | the next call's wait is the time left, never a fresh W: at most W less the time already spent | `EINTR, then the LF` (the second call's wait is checked) |
| K3 | `EINTR` repeated until the deadline passes | `false`, no error; no call starts after the deadline | `EINTR until the deadline` |
| K4 | `EINTR` with no time left | no further `select`; `false`, no error | `EINTR with no time left` (one call) |
| K5 | any other `select` error | unchanged (A3): `false` and the error, no retry, which `readMasked` ignores | `another error` (one call); and `TestSecret_SettleCheckFails_0087`, `closed`, unchanged |
| K6 | a descriptor `select` cannot take (negative, or at or above `FD_SETSIZE`) | unchanged (A3): `false` and `errNoSelect`, never given to `select` | `TestSecret_SettleCheckFails_0087` (`closed`, `descriptor 1100`), unchanged |
| K7 | the builds that are not Unix | unaffected: `wizard/settle_windows.go` waits with `WaitForSingleObject`, which is not alertable, in slices to a deadline it already fixes, and `wizard/settle_other.go` does not wait; neither calls `select` or the seam | read in both files at `77563af`; `GOOS=windows go vet ./wizard` and `GOOS=windows go test -c ./wizard` build with the fix |

## Release note (2026-11-22)

`CHANGELOG.md`'s `[Unreleased]` became the `v1.32.0` section dated 2026-11-22 (R53: a section is dated by its tag's commit, so the date is re-checked by `make changelog-check` once the owner's commit exists); `[Unreleased]` is empty. `python3 -B tools/apidiff.py v1.31.0` lists additions only (`APIError.Tool`, `ErrToolSchemaRefused`, `x/signin.VerifyKilo`, `wizard.Options.NoBrowserQuestion`, `wizard.Options.NoLoopback`): a minor version. The tag is the owner's. This record's status stays as it is and goes to `complete` when the tag exists, its CI has passed, and `tools/consumer.py` and `tools/offline.py` are recorded (`tools/records.py status`). This record's entry is under `Fixed`, with its consumer note kept. The probe's R3 second-half request in `llmprovider/live_raw_enter_0087_windows_test.go` says "type one letter", since the case ends on the letter; the owner-steps file is outside the repository and is the coordinator's.

*(2026-11-24, note from [0093-MADR-take-model-metadata-from-the-caller.md](0093-MADR-take-model-metadata-from-the-caller.md): this record's "now" rows on files 0093's docs phase edited were re-checked; see the note below. The text above is unchanged.)*

*(2026-11-24, from 0093-PLAN-take-model-metadata-from-the-caller.md's Phase 5: this record's "now" rows on a file 0093's Phase 4 edited were re-run with plain ids against a scratch clone's commit of the tree (`dc96bf0ce4e6`, 0093's build, uncommitted), and against `cfc911d`, its base. 9 fail there: 1 moved by 0093's build (1 found on a new line, 0 whose text 0093 rewrote), and 8 already failing at `cfc911d` (7 found on another line, 1 on none). Each row's text is now where this note says; the rows above are unchanged.)*

* C21bnow (`wizard/configure.go:1051`): `wizard/configure.go:1067`; already failing at `cfc911d`
* C21now (`wizard/configure.go:1018`): `wizard/configure.go:1034`; already failing at `cfc911d`
* C24now (`wizard/configure.go:647`): `wizard/configure.go:650`; already failing at `cfc911d`
* C25now (`wizard/configure.go:436`): `wizard/configure.go:439`; already failing at `cfc911d`
* C26now (`wizard/configure.go:688`): `wizard/configure.go:691`; already failing at `cfc911d`
* C36now (`docs/guides/api-standards.md:67`): `docs/guides/api-standards.md:74`; already failing at `cfc911d`
* C42now (`docs/architecture.md:200`): `docs/architecture.md:204`; already failing at `cfc911d`
* C43now (`docs/architecture.md:1495`): on no line; already failing at `cfc911d`
* C47now (`AGENTS.md:286`): `AGENTS.md:289`; moved by 0093's build; re-placed 2026-11-24 (the review's notes closed)
