---
status: accepted
date: 2026-10-22
decision-makers: repository owner
consulted: "GitHub issues 6, 7 and 8 (maccavelli/go-llmprovider-sdk), and gobble-cli at ea04387 as they cite; 0008-MADR-repair-oauth-loopback-and-session-wiring.md (D3, its amendment of 2026-09-27); 0043-MADR-openai-chatgpt-plan-sign-in.md (A3); 0056-MADR-wizard-credential-only-entry-point.md (D4, D5); the standards guide (R40, R48, R51, R52, R55); database/sql/driver's optional interfaces; this repository at ca111b1"
informed: gobble-cli, and every consumer of the wizard
---

<!-- markdownlint-disable MD013 -->

# Give the Wizard's Device Code to the Caller as Data, Let a Browser Sign-In Run Without a Paste Prompt, and End the Paste Prompt Once the Loopback Wins

## Context and Problem Statement

[Issues 6](https://github.com/maccavelli/go-llmprovider-sdk/issues/6),
[7](https://github.com/maccavelli/go-llmprovider-sdk/issues/7) and
[8](https://github.com/maccavelli/go-llmprovider-sdk/issues/8), opened
2026-10-23 UTC by the owner for gobble-cli, ask for three changes at
`v1.21.0`. gobble wants every sign-in to go through
`wizard.ResolveCredential`: its interactive `/login`, its first run,
`gobble setup`, and `gobble auth login`, which also runs with no terminal
and with `--json`.

* **Issue 6.** A device sign-in's code reaches the caller only as a
  notice's text, and its expiry does not reach it at all.
* **Issue 7.** A browser sign-in always starts a paste prompt, which a
  caller with no terminal can neither answer nor leave. A side request: an
  empty paste ends the sign-in.
* **Issue 8.** Once the loopback has finished a browser sign-in, the wizard
  waits for Enter, to end the paste prompt it started.

The three share one cause. The wizard's only channels to its caller are
the `Prompter`'s methods, and `Input` cannot be abandoned.

### Facts (observed 2026-10-22 at `ca111b1`)

* **F1. The device code is a notice.** `oauthFlowOptions` turns
  `NotifyDevice` into the notice "Open %s and enter code %s" (C1).
  `auth.LoginDeviceOAuth` calls `NotifyDevice` with the URL and the code
  only (C2). A caller that wants the code as data must parse the notice,
  which is no contract **[read]**.
* **F2. The expiry exists and is never zero.** `auth.DeviceLogin`, which
  `auth.StartDeviceOAuth` returns, has `Expiry` (C3). Its source differs by
  provider **[read]**:
  * Grok: the server's `expires_in`, else 300 s (C4);
  * Kilo: the server's `expires_in`, else Kilo's default life (C6);
  * OpenAI: the module's own fixed wait, not a value from the server (C5).
* **F3. Three device sign-ins, one seam.** The wizard runs Grok's and
  OpenAI's device sign-ins, Kilo's, and Kilo's as a key, each through the
  package variable `loginDeviceOAuth` (C7). Three test files replace it:
  `auth_test.go`, `kilo_device_test.go` and `remediation_test.go`
  **[read]**.
* **F4. The paste prompt is always started.** Every browser sign-in, the
  ChatGPT plan's included, sets `InputCode` to the paste prompt (C8), which
  calls `Prompter.Input` (C9). The login ends on an error from `InputCode`
  (C10). So a `Prompter` that cannot answer ends the sign-in, and one that
  blocks holds it **[read]**.
* **F5. The loopback alone completes every browser sign-in, the ChatGPT
  plan's first included.** The loopback's callback carries the URL's
  `client_id` (C11). A first plan sign-in is refused only when the callback
  names no issued client (C12). "A first sign-in needs the whole URL" (C13)
  is about the *pasted* fallback, where a bare code carries no client id.
  It does not apply to the loopback **[read]**. *(This corrects the
  assessment given to the owner on 2026-10-22, which said that a plan's
  first sign-in "needs the pasted URL". Q1 was answered on that summary. Q3
  asks it again.)*
* **F6. An empty paste ends the sign-in.** The paste parser refuses it as
  "oauth: empty authorization code" (C14) **[read]**.
* **F7. The wizard drops a context that already says when to stop.** Both
  logins give `InputCode` a context of its own, and cancel it when they
  return (C15 to C17). The wizard's paste prompt receives that context, but
  `Prompter.Input` takes none (C18), so the read goes on after the context
  ends **[read]**.
* **F8. Why the wizard waits for Enter.** `drain` notifies "Browser sign-in
  finished; press Enter to continue" and waits for the pending read (C19).
  0008-MADR's amendment of 2026-09-27 requires it (C20). `TextPrompter`
  keeps one `bufio.Reader` across prompts (C21), so a read left running
  races the next prompt for the user's next line. A test holds the
  behaviour (C25) **[read]**.
* **F9. `TextPrompter` cannot abandon a read of stdin.** These are probes
  of a scratch program on this host (darwin/arm64, Go 1.27.3), run on a
  pipe and on a pseudo-terminal **[probed]**:
  * **P1.** `os.Stdin.SetReadDeadline` returns "file type does not support
    deadline", which matches `os.ErrNoDeadline`, on both.
  * **P2.** A read already blocked on `os.Stdin` is still blocked a second
    after a deadline is asked for, on both.
  * **P3.** The only route to a deadline is to set `O_NONBLOCK` on fd 0
    and wrap it in a new `*os.File`. A read then returns "i/o timeout" after
    200 ms. But the flag is on the open file description, which the parent
    shell shares: after the program exited, a new process on that terminal
    saw `O_NONBLOCK` set on fd 0. The terminal has been changed for every
    program that uses it afterwards.

  `TextPrompter`'s `Secret` also puts that terminal in raw mode for its
  masked entry (C22). A read left running from the paste prompt could
  receive the keys typed at the next `Secret` prompt.
* **F10. gobble already reads stdin in a way that can be abandoned**
  (`ea04387`, pinned at `v1.21.0`, G1) **[read]**:
  * `editor.Reader` "owns stdin": one goroutine reads it for the life of
    the process (G2);
  * its `ReadContext` returns when `ctx` ends (G3), and a chunk that
    arrives later "is served to the next call" (G7), so no input is lost to
    an abandoned read;
  * its `auth login` paste reader is the hazard F8 names: a goroutine with
    its own `bufio.Reader` on stdin, abandoned when `ctx` ends (G4). On an
    empty paste it waits for the browser instead of failing (G5);
  * its `--json` device event carries the provider, the URL, the code and
    the expiry (G6).

## Decision Drivers

* A caller with no terminal must be able to run every sign-in that can
  finish without one, through the wizard.
* A finished sign-in should return at once, whenever the `Prompter` can
  promise to stop reading.
* 0008-MADR's amendment stands: two reads never share the input, and the
  user's next line goes to the next prompt.
* Only additions to the API (R48), each with a call site (R55).
  `ConfigureLLM`'s and `ResolveCredential`'s behaviour is unchanged when no
  new option is set.
* No change to the terminal's state that outlives the program (F9 P3).

## Considered Options

The device code (issue 6):

* **DC1.** `Options.ShowDeviceCode func(DeviceCode) error`, with a new
  struct `wizard.DeviceCode` holding the provider, the URL, the code and the
  expiry.
* **DC2.** The issue's shape, `Options.ShowDeviceCode func(url, code string,
  expires time.Time)`.
* **DC3.** A new field in `auth.OAuthFlowOptions`, which
  `auth.LoginDeviceOAuth` would call with the expiry.

The paste prompt for a caller with no terminal (issue 7):

* **NP1.** `Options.NoPastePrompt bool`. No paste prompt is started, and
  the loopback alone completes the sign-in.
* **NP2.** No option. The caller's `Prompter.Input` blocks until `ctx`
  ends.

Ending the paste prompt when the loopback wins (issue 8):

* **IC1.** An optional interface, `wizard.InputerContext`, whose method
  `InputContext(ctx, prompt, def)` the wizard calls in place of `Input` for
  the paste prompt, and cancels once the login returns.
* **IC2.** A context on `Prompter.Input` itself.
* **IC3.** `Options.CancelInput func()`, which the wizard calls when the
  loopback wins.
* **IC4.** Abandon the read: no drain.
* **IC5.** `TextPrompter` abandons its own reads, by a deadline, by
  `O_NONBLOCK`, by a reader that owns stdin, or by `poll(2)`.

## Decision Outcome

Chosen options: **DC1, NP1 and IC1**. The owner answered Q1 and Q2 below on
2026-10-22, then Q3 to Q5 the same day, each with the recommended answer.
Q3 (a) replaces Q1's answer, which rested on the wrong summary that F5
corrects. The PLAN's Phase 0 marked this record `accepted`.

### D1. `Options.ShowDeviceCode` and `wizard.DeviceCode`

```go
// DeviceCode is a device sign-in's code, for a caller that shows it itself.
type DeviceCode struct {
	Provider llmprovider.ProviderID
	URL      string    // where the user enters the code; it may already carry it
	Code     string
	Expiry   time.Time // when the code stops being accepted; never zero
}

// in Options:
ShowDeviceCode func(DeviceCode) error
```

* **When set,** it is called once per device sign-in, in place of the
  notice, with the code the device authorization returned. This covers
  Grok's and OpenAI's sign-ins, Kilo's, and Kilo's as a key. It runs on the
  flow's goroutine, before the wait for approval.
* **An error from it** cancels the device login and is returned as it is,
  as a `Prompter`'s error is: a code the user never saw cannot be approved.
  *(Amended 2026-10-22: it ends the sign-in before any poll, and no
  `Cancel` is called. See the amendment below.)*
* **When nil,** the notice is today's, word for word.
* **The expiry** is `DeviceLogin.Expiry`, as F2 describes. The doc says it
  is never zero, and that for OpenAI it is the module's own wait.
* **How.** The wizard calls `auth.StartDeviceOAuth`, then the hook or the
  notice, then `DeviceLogin.Wait`, in place of `auth.LoginDeviceOAuth`.
  Nothing in `llmprovider/auth` changes. The test seam `loginDeviceOAuth`
  becomes a wizard-level `deviceLogin`, which returns plain values, since
  only `auth` can build a `DeviceLogin` that waits. F3's three test files
  move to it, and keep what they assert (R46).
* **The struct (DC1), the owner's answer to Q4,** not the issue's three
  arguments (DC2). A function type cannot gain a parameter under R48, and
  a struct can gain a field. gobble's event needs the provider (G6), and
  with an empty `id` only the wizard knows which one the user chose.

### D2. `Options.NoPastePrompt`

When true:

* a browser sign-in starts no paste prompt and shows no paste instruction.
  It still shows the authorize URL, and still calls `Options.OpenURL`. Only
  the loopback completes the sign-in, within the login's own limit;
* nothing is left to drain, so the sign-in returns as soon as the login
  does;
* **the ChatGPT plan sign-in is offered,** as every other browser sign-in,
  with no paste prompt: the owner's answer to Q3, which replaces Q1's.
  `offeredAuthMethods` is unchanged (C23);
* every other method is unchanged.

When false, the sign-in is as today, apart from D4's empty paste.

**Why the plan sign-in stays offered.** F5 shows that the loopback
completes it, a first sign-in included, as it completes Grok's and
OpenAI's browser sign-ins. Hiding it would cost gobble its only scripted
route to the ChatGPT plan, and protect nothing: a machine whose browser
cannot reach the loopback fails every browser sign-in alike, by the login's
time limit. *(Q1's answer, "hide it", was given on 2026-10-22 on the
assessment's wrong summary, and is kept below as given.)*

### D3. `wizard.InputerContext`

```go
// InputerContext is implemented by a Prompter that can give up a read.
type InputerContext interface {
	InputContext(ctx context.Context, prompt, def string) (string, error)
}
```

* **The contract,** in its doc. `InputContext` is `Input` that gives up
  when `ctx` ends: it returns soon after, with an error. It leaves no read
  running that could take the user's next answer, so a line typed later
  goes to the next prompt. A `Prompter` that cannot promise this must not
  implement it. A `Prompter` that wraps another must forward it.
  Concurrency (R51): the wizard calls it on a goroutine of its own, while
  another goroutine may call `Notify`, as it calls `Input` today.
* **Where it is used:** the paste prompt only, the one read that something
  else can make moot. Every other prompt keeps calling `Input`.
* **How.** The paste prompt derives its own context from the one the login
  gives it, and `drain` cancels it. The wizard therefore does not rely on
  the login's own cancel (F7), which is not documented in `auth`.
* **`drain`,** for a `Prompter` that implements it: cancel, then wait for
  the read to return, with no notice. If the read has not returned within
  a short grace (`drainGrace`, 2 s), the wizard shows today's "press Enter"
  notice and keeps waiting. So a `Prompter` that breaks the contract gets
  today's behaviour, not a silent hang.
* **For one that does not implement it,** nothing changes: the notice, then
  the wait (0008-MADR's amendment).
* **`TextPrompter` does not implement it,** the owner's answer to Q2. F9
  shows why it cannot meet the contract. Its doc says that it keeps the
  Enter.
* **The name** follows `database/sql/driver`, where a `Conn` may implement
  `QueryerContext` for `QueryContext`. The method keeps the issue's name.
* **gobble's call site:** a `Prompter` over its `editor.Reader`, whose
  `ReadContext` already meets the contract (F10).

### D4. An empty paste asks again

The paste prompt asks again on an empty answer, instead of passing it to the
login (F6). Each time it first checks that `drain` has not run, so the Enter
that `drain` asks for ends the read and is not taken as a paste. At the end
of the input (`TextPrompter` on a closed pipe) the second read is an error,
which ends the sign-in as before.

**Ask again, the owner's answer to Q5,** not gobble's "wait for the browser"
(G5). A stray Enter would leave a user whose browser cannot reach the
loopback no way to paste until the login's limit, ten minutes (C26).

### D5. Tests, docs, release

* **Tests,** each first seen failing on a planted break:
  * the hook receives the provider, URL, code and expiry, and no notice is
    shown; an error from it ends the sign-in, and the login is cancelled
    *(amended: and no poll is made)*;
    without it, today's notice;
  * the expiry through the real `auth.StartDeviceOAuth`, over a client
    whose transport answers Grok's endpoints with `expires_in`;
  * `NoPastePrompt`: no `Input`, no paste instruction, the URL shown, the
    login's `InputCode` nil, and the plan sign-in offered, its login's
    `InputCode` nil too (Q3);
  * a `Prompter` implementing `InputerContext` whose read the loopback
    overtakes: no "press Enter" notice, the next prompt starts after the
    read returned, and the read's context was cancelled;
  * a `Prompter` implementing it that ignores `ctx`: the notice after the
    grace, then today's wait;
  * an empty paste asked again; the Enter after `drain` not asked again;
  * `TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt` and every other
    existing test pass, F3's three files moved to the new seam.
* **Docs:** `docs/architecture.md`'s wizard section; `CHANGELOG.md`'s Added
  and Changed (D4).
* **Release:** `v1.22.0`, a minor release, since it adds exported names.
  After the tag, `v1.18.0`'s GitHub Release is deleted, keeping five (R53).
* **The new names,** for R55, each with gobble as the caller:
  `DeviceCode`, `Options.ShowDeviceCode`, `Options.NoPastePrompt`,
  `InputerContext`.

### Consequences

* Good, because `gobble auth login --json` can run every device sign-in and
  every loopback-reachable browser sign-in through the wizard, and report
  the device code as data.
* Good, because a finished browser sign-in returns at once for a
  `Prompter` that can give up a read, and 0008-MADR's guarantee holds for
  every `Prompter`.
* Good, because no terminal state is changed (F9 P3), and `TextPrompter` is
  unchanged.
* Neutral, because `TextPrompter` users still press Enter after a browser
  sign-in (Q2).
* Bad, because an optional interface is lost by a wrapper that does not
  forward it. Such a wrapper falls back to the Enter, which is safe; the doc
  says to forward it.
* Bad, because an empty paste no longer ends the sign-in at once (D4). A
  script feeding `TextPrompter` from a closed pipe still ends, on the
  second read's error, which wraps `io.EOF`, in place of "empty
  authorization code".
* Bad, because four names are added to the API, and stay (R48).

### Confirmation

* Each test of D5, seen failing first, and `make api-check` listing exactly
  the four additions.
* `make docs-check`: the new names' docs say whether they are safe for
  concurrent use (R51) and name their error kinds (R52).
* Each issue's request, item by item, in the PLAN's record.

## Pros and Cons of the Options

### DC1. A hook taking a `DeviceCode`

* Good, because it can gain a field without breaking (R48).
* Good, because it carries the provider, which gobble's event needs.
* Bad, because it is two names, not one.

### DC2. The issue's three arguments

* Good, because it is one name.
* Bad, because it has no provider, and can never gain one.

### DC3. A field in `auth.OAuthFlowOptions`

* Good, because a caller of `auth` gains the expiry too.
* Bad, because `auth.StartDeviceOAuth` already gives the expiry. The field
  would be a second route to it, with no caller but the wizard (R55).

### NP1. `Options.NoPastePrompt`

* Good, because a caller with no terminal says so once, and the wizard
  asks it nothing it cannot answer.
* Bad, because a sign-in whose browser cannot reach the loopback then fails
  by the login's time limit, not at once.

### NP2. No option

* Bad, because the caller's `Prompter` must block an `Input` it cannot
  answer, and with a `Prompter` that cannot give up a read, `drain` then
  waits for it forever.

### IC1. `InputerContext`

* Good, because only a `Prompter` that can keep the contract opts in, and
  every other one keeps 0008-MADR's behaviour.
* Good, because it is an addition (R48), and it has a precedent in the
  standard library.
* Bad, because a wrapper hides it unless it forwards it.

### IC2. A context on `Prompter.Input`

* Bad, because it breaks every `Prompter` implementation (R48).

### IC3. `Options.CancelInput`

* Good, because a wrapper cannot hide it.
* Bad, because it splits one contract between `Options` and the `Prompter`,
  and names no particular read.

### IC4. Abandon the read

* Bad, because it brings back the race 0008-MADR's amendment closed, for
  `TextPrompter` and for any `Prompter` like it.

### IC5. `TextPrompter` abandons its own reads

* Bad, because a deadline is refused on stdin, and a blocked read cannot
  be stopped (F9 P1, P2).
* Bad, because `O_NONBLOCK` changes the terminal for every program after
  this one (F9 P3).
* Bad, because a reader that owns stdin, like gobble's, takes `In` from the
  caller for the life of the process, and keeps a read running across a
  `Secret` prompt's raw mode (F9).
* Bad, because `poll(2)` with a cancel pipe is Unix only. A Windows console
  needs its own reader, which this record has not measured, and
  `golang.org/x/sys` would become a direct requirement.
* The owner chose to keep the Enter for `TextPrompter` (Q2). A later record
  may revisit it.

## Owner questions

* **Q1 (D2). The ChatGPT plan sign-in under `NoPastePrompt`.**
  * **(a)** Hide it.
  * **(b)** Refuse it, as an error, when chosen.

  **Answered 2026-10-22: (a).**
* **Q2 (D3). `TextPrompter`.**
  * **(a)** It keeps the Enter, and does not implement `InputerContext`.
  * **(b)** It implements it, by one of IC5's routes.

  **Answered 2026-10-22: (a).**
* **Q3 (D2). Q1 again, given F5.**
  * **(a)** Offer it, as every other browser sign-in (recommended).
  * **(b)** Hide it, as answered.

  **Answered 2026-10-22: (a).** It replaces Q1's answer.
* **Q4 (D1). The hook's shape.**
  * **(a)** `func(DeviceCode) error` (recommended).
  * **(b)** The issue's `func(url, code string, expires time.Time)`.

  **Answered 2026-10-22: (a).**
* **Q5 (D4). An empty paste.**
  * **(a)** Ask again (recommended).
  * **(b)** Stop asking, and wait for the loopback, as gobble does.
  * **(c)** Unchanged: it ends the sign-in.

  **Answered 2026-10-22: (a).**

## More Information

* The issues:
  [#6](https://github.com/maccavelli/go-llmprovider-sdk/issues/6),
  [#7](https://github.com/maccavelli/go-llmprovider-sdk/issues/7),
  [#8](https://github.com/maccavelli/go-llmprovider-sdk/issues/8).
* F9's probe is a scratch program built outside the tree, with its build
  cache in the session's scratch directory. It is not committed. Its three
  modes are `deadline`, `blocked` and `nonblock`; the pseudo-terminal came
  from `script(1)`.
* `database/sql/driver` documents `QueryerContext`, `ExecerContext` and
  `ConnPrepareContext` as interfaces a driver may implement, which
  `database/sql` then uses.
* Related:
  [0008-MADR-repair-oauth-loopback-and-session-wiring.md](0008-MADR-repair-oauth-loopback-and-session-wiring.md)
  (D3 and its amendment of 2026-09-27),
  [0043-MADR-openai-chatgpt-plan-sign-in.md](0043-MADR-openai-chatgpt-plan-sign-in.md)
  (A3),
  [0056-MADR-wizard-credential-only-entry-point.md](0056-MADR-wizard-credential-only-entry-point.md)
  (D4, D5).

### Code claims

<!-- claims: self=ca111b1 gobble-cli=ea04387 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The device code becomes a notice | `wizard/auth.go:656` | `p.Notify(LevelInfo, "Open %s and enter code %s", verificationURL, userCode)` |
| C2 | The login passes no expiry | `llmprovider/auth/oauth_device.go:199` | `opts.NotifyDevice(login.VerificationURI, login.UserCode)` |
| C3 | The handle has the expiry | `llmprovider/auth/oauth_device.go:90` | `Expiry time.Time` |
| C4 | Grok's: the server's, else 300 s | `llmprovider/auth/oauth_device.go:323` | `expires := durationFromSeconds(float64(device.ExpiresIn), defaultDeviceExpiry, false)` |
| C5 | OpenAI's: the module's own wait | `llmprovider/auth/oauth_device.go:227` | `deadline := config.now().Add(openAIDeviceTimeout)` |
| C6 | Kilo's: the server's, else its default | `llmprovider/auth/kilo_device.go:78` | `deadline := config.now().Add(durationFromSeconds(float64(device.ExpiresIn), kiloDeviceDefaultLife, false))` |
| C7 | The device seam | `wizard/auth.go:41` | `loginDeviceOAuth  = auth.LoginDeviceOAuth` |
| C8 | The paste prompt is the login's InputCode | `wizard/auth.go:691` | `flow.InputCode = paste.input` |
| C9 | It reads through Input | `wizard/auth.go:736` | `return s.p.Input(s.prompt, "")` |
| C10 | An InputCode error ends the login | `llmprovider/auth/oauth_loopback.go:102` | `// and an error from opts.InputCode ends the login, matching that error.` |
| C11 | The loopback carries the client id | `llmprovider/auth/oauth_loopback.go:540` | `case result <- oauthCallbackResult{code: code, clientID: callbackClientID(query)}:` |
| C12 | A first plan sign-in needs a client from the callback | `llmprovider/auth/oauth_plan.go:282` | `return "", errors.New("oauth: the callback carries no issued client_id: registration did not complete")` |
| C13 | 0043-MADR A3 on the paste | `docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md:385` | `a first sign-in needs the whole URL` |
| C14 | An empty paste is refused | `llmprovider/auth/oauth_loopback.go:609` | `return oauthCallbackResult{err: errors.New("oauth: empty authorization code")}` |
| C15 | The paste gets a context of its own | `llmprovider/auth/oauth_loopback.go:156` | `inputCtx, inputCancel = context.WithCancel(flowCtx)` |
| C16 | Cancelled when the browser login returns | `llmprovider/auth/oauth_loopback.go:168` | `defer inputCancel()` |
| C17 | And when the plan login returns | `llmprovider/auth/oauth_plan.go:161` | `defer inputCancel()` |
| C18 | Input takes no context | `wizard/prompter.go:83` | `Input(prompt, def string) (string, error)` |
| C19 | The drain's notice | `wizard/auth.go:750` | `s.p.Notify(LevelInfo, "Browser sign-in finished; press Enter to continue")` |
| C20 | 0008-MADR's amendment requires it | `docs/decisions/0008-MADR-repair-oauth-loopback-and-session-wiring.md:605` | `asked for as "Browser sign-in finished; press Enter to continue", before` |
| C21 | TextPrompter keeps one reader | `wizard/text_prompter.go:48` | `reader *bufio.Reader` |
| C22 | Secret's raw mode | `wizard/text_prompter.go:382` | `state, err := term.MakeRaw(fd)` |
| C23 | The plan sign-in's existing filter | `wizard/auth.go:72` | `without the consumer's app name, so is the ChatGPT plan sign-in` |
| C24 | The paste instruction | `wizard/auth.go:717` | `s.p.Notify(LevelInfo, "%s", s.instruction)` |
| C25 | The test that holds the drain | `wizard/auth_test.go:455` | `countContaining(p.seenNotify, "press Enter to continue") != 1` |
| C26 | The browser login's limit | `llmprovider/auth/oauth_loopback.go:25` | `10 * time.Minute` |
| G1 | gobble pins v1.21.0 | `gobble-cli:go.mod:16` | `github.com/maccavelli/go-llmprovider-sdk v1.21.0` |
| G2 | gobble's reader owns stdin | `gobble-cli:internal/cli/editor/reader.go:26` | `// Reader owns stdin. Start launches the one goroutine that reads it, in` |
| G3 | Its read takes a context | `gobble-cli:internal/cli/editor/reader.go:115` | `func (r *Reader) ReadContext(ctx context.Context, p []byte) (int, error) {` |
| G7 | A late chunk goes to the next call | `gobble-cli:internal/cli/editor/reader.go:112` | `served by then is served to the next call` |
| G4 | Its paste reader, abandoned on ctx | `gobble-cli:internal/cli/auth.go:273` | `s, _ := bufio.NewReader(e.stdin).ReadString('\n')` |
| G5 | An empty paste waits for the browser | `gobble-cli:internal/cli/auth.go:279` | `<-ctx.Done() // nothing pasted: the browser's callback may still arrive` |
| G6 | Its device event | `gobble-cli:internal/cli/auth.go:232` | `c := loginEvent{Event: "device_code", Provider: id, URL: d.URL, Code: d.Code}` |
| C27 | A device login polls only in Wait | `llmprovider/auth/oauth_device.go:164` | `session, err := d.poll(pollCtx)` |
| C28 | Cancel | `llmprovider/auth/oauth_device.go:179` | `func (d *DeviceLogin) Cancel() {` |
| C29 | Cancel sets a flag | `llmprovider/auth/oauth_device.go:181` | `d.canceled = true` |

## Amendment 2026-10-22: an error from `ShowDeviceCode` needs no `Cancel`

* **Found** at the PLAN's Phase 1, before any code. D1 says an error from
  the hook "cancels the device login", and the PLAN's step 1 calls
  `login.Cancel()`. But a `DeviceLogin` makes no request until `Wait`
  (C27). `Cancel` sets a flag (C28, C29) and stops a running `Wait`. With
  no `Wait` it changes nothing that can be observed, so the PLAN's plant of
  its removal could fail no test.
* **The owner chose (2026-10-22),** "resolution 1": no `Cancel`. An error
  from the hook ends the sign-in before `Wait`, so before any poll, and is
  returned as it is. Keeping the call as a defence, a line no test can
  check, was put to the owner and not chosen. D1 stands otherwise.
