---
status: proposed
date: 2026-10-24
decision-makers: repository owner
consulted: "the owner's report of 2026-10-24; a read of every file of wizard/ at 2d6e7f3; probes P1 to P8 on a scratch clone (method below); the wizard's statement coverage; gobble-cli at 142685b, the wizard's caller; 0028-MADR-heuristics-and-performance-from-the-research-pass.md and its PLAN (H10b, D3); the Anthropic key rules of TruffleHog and gitleaks; 0013-MADR-remediate-debugging-pass-findings.md (C3) and 0020-MADR-remediate-v1-debugging-pass-findings.md (F50)"
informed: gobble-cli, and every consumer of the wizard
---

<!-- markdownlint-disable MD013 -->

# Remediate the Wizard's Debugging-Pass Findings, the Claude Key Warning First

## Context and Problem Statement

On 2026-10-24 the owner reported a bug and asked for a pass over the
wizard: "there is a bug in the claude/anthropic API key validator, it
returns the key is not the expected shape even when it is and it saves it
and it works fine. we need to ensure the wizard is thoroughly hardened,
stable, and bug free. find bugs. find gaps. look for missing or incomplete
wiring. report findings to an madr for review."

The pass read every file of `wizard/` at `2d6e7f3` (about 2,870 lines,
91.9 % of statements covered by its tests), traced the paths a consumer
takes, and ran probes P1 to P8 on a scratch clone. It found 13 findings:

* 2 of high severity, among them the reported bug (W1);
* 5 medium;
* 6 low, including a test gap and documentation.

This record lists each finding with its evidence. It decides how to
remediate them, and asks the owner the design questions the fixes depend
on.

### Method

* **Read.** Every file of `wizard/`, against the doc of each exported name,
  `Options`' fields first: is each field read where its doc says, and does
  each path do what the doc says?
* **Probed.** P1 to P8 are scratch tests in a clone of `2d6e7f3`, never in
  the tree. Each runs the wizard through its own seams, with a scripted
  `Prompter`, a stubbed `HTTPClient` or a stubbed login. Each records what
  the wizard does today. More Information gives each one's method and
  output.
* **Measured.** `go test -coverprofile` over `./wizard/`, and the
  uncovered statements, file by file.
* **Checked against.** The caller: gobble-cli at `142685b` calls
  `ResolveCredential` with `Discover: true` (G1) and `NoEndpointPrompt:
  true` (G2). Its own `Secret` reads a paste as the wizard's does: it skips
  escape sequences and trims. The Anthropic key rules of TruffleHog and
  gitleaks are fetched.

### Findings

Severity: **high** loses data or misleads the user about a credential;
**medium** is wrong behaviour on a path a user takes; **low** is a corner,
a gap in tests, or documentation. Each finding says how it is known:
**[probed]** P1 to P8, **[read]** the code at the claim, **[measured]**
coverage.

#### High

* **W1. A key the provider accepts is called the wrong shape, and the
  warning is never withdrawn** **[probed]** (P1; C1 to C4). This is the
  reported bug.
  * The Claude check is exact: `sk-ant-` then `api03` or `admin01`, then 93
    characters and `AA`, 108 in all (C1). The two scanners agree with it:
    TruffleHog's `\b(sk-ant-(?:admin01|api03)-[\w\-]{93}AA)\b` and gitleaks'
    `sk-ant-api03-[a-zA-Z0-9_\-]{93}AA`. A standard key passes, and P1 shows
    no warning for one.
  * But any other tag or length is warned about, and the listing then
    accepts it and the wizard saves it. P1 ran an `sk-ant-api04-` key of 108
    characters, and an `sk-ant-api03-` key of 109. Each got "this does not
    look like a Claude (Anthropic) API key (expected sk-ant-api03- and 95
    characters ending in AA); the model listing will check it" (C2). Each
    was then listed once, accepted, and returned as the Result's key, with
    no later notice.
  * The warning runs before the listing (C3), and nothing reports the
    listing's verdict. A user whose key works is left with a warning that
    it does not, as the owner saw.
  * The exact shapes rest on little: "No vendor publishes a full format"
    (C5). Each shape was checked against one key, the owner's (C6). The same
    false warning can meet any provider whose real keys vary, among them an
    older Together key that is not `tgp_v1_`.
  * The shape text names `sk-ant-api03-` only, though the check also
    accepts `admin01` (C1).
  * Which variant the owner's key is, is not known: the pass read no key.
    Q0 asks for its shape, which the fix does not depend on.
* **W2. A fresh sign-in replaces the saved session before the listing
  checks it, and a refusal leaves the refused one** **[probed]** (P8; C9,
  C10).
  * `saveOAuthCredential` saves a browser or device sign-in's session to
    `Options.TokenStore` at once (C9). Kilo's device sign-in does the same
    (C10). The listing that checks it runs after.
  * P8 put a good OpenAI session in the store, signed in again through a
    stubbed browser login, and had the listing refuse with HTTP 401. The
    run ended with "OpenAI refused the sign-in; sign in again". The store
    then held the refused session, and the good one was gone.
  * The store is the session's only copy (0016-MADR A10), so the consumer's
    saved configuration now names a session that does not work.

#### Medium

* **W3. The warning promises a check that does not run** **[probed]** (P2;
  C2, C4). Without `Options.Discover` the wizard lists nothing (C4), yet
  the warning says "the model listing will check it". P2: no listing call,
  and the key returned.
* **W4. A split escape sequence is typed into the key** **[probed]** (P3;
  C7).
  * `TextPrompter`'s masked entry takes an ESC with nothing buffered after
    it for the Escape key, and drops it alone (C7). When a sequence's bytes
    arrive in two reads, the rest of it is then typed into the entry.
  * P3: a paste read as `"\x1b[200~sk-ant-key"`, `"AA\x1b"`, `"[201~\r"`
    gave `"sk-ant-keyAA[201~"`; one read as `"\x1b"`, `"[200~…"` gave
    `"[200~sk-ant-keyAA"`.
  * It needs a terminal that sends bracketed-paste markers, or a key
    sequence that crosses a read. The key is then wrong, and the shape
    warning (W1) is the only sign of it.
* **W5. The paste prompt spins on a `Prompter` that answers at once**
  **[probed]** (P4; C8). The browser sign-in asks again while the answer is
  empty (0057-MADR D4), with no pause and no limit (C8). P4: a `Prompter`
  whose `Input` returns `""` with no error was asked 517,770 times in
  100 ms, until the login's context ended. `TextPrompter` at the end of its
  input answers once with the default, then with an error, so it does not
  spin. A consumer's `Prompter` that keeps answering the default would spin
  for the login's 10 minutes.
* **W6. A saved session that cannot be used is offered, then ends the
  run** **[probed]** (P7; C11, C12). The wizard asks "Keep the existing
  session (••••••••abcd)?" (C12) before it checks the session. P7: with an
  expired session and no refresh token, the user said yes, and the run
  ended with "the saved OpenAI session cannot be kept (oauth: no refresh
  token); sign in again" (C11). No sign-in menu was shown. The user must
  start the wizard again.
* **W7. A saved fallback the menu does not offer is dropped** **[probed]**
  (P5; C13). The fallback round preselects the saved fallbacks it offers
  (0020-MADR F50). One that is neither recommended nor among a search's
  matches is offered nowhere, and the Result leaves it out. P5: the saved
  `claude-legacy-fallback`, a blank round, and `Fallbacks = []`. The model
  has this case covered, as "current" (`selectRecommended`). This is what
  is left of 0013-MADR C3.

#### Low

* **W8. `ConfigureLLM` asks questions on an ended context** **[probed]**
  (P6; C14, C15). `ResolveCredential` checks `ctx` first (C15).
  `ConfigureLLM` does not (C14). P6: with `ctx` already cancelled, it asked
  for the provider and the API key, then failed at the model, "select
  model: context canceled".
* **W9. `NoEndpointPrompt` still asks about a plain-http endpoint**
  **[read]** (C16, C17). Under `NoEndpointPrompt` a remote URL "is checked
  as a typed one is" (C16). For plain http to a remote host, that check is
  a `Confirm` (C17). gobble passes `NoEndpointPrompt` (G2) to sign in with
  no terminal, where that question cannot be answered. The URL is one the
  caller already holds.
* **W10. Raw-mode masked entry has no test** **[measured]** (C18).
  `secretOnce`, the path `Secret` takes on a terminal, is 26.7 % covered:
  `MakeRaw`, its fallback, and the restore are run by no test. `readMasked`
  is tested apart from the terminal. Other uncovered statements are each
  an error return, listed in More Information.
* **W11. A pasted ChatGPT token's session does not keep the caller's
  client** **[read]** (C19, C20). `Options.HTTPClient`'s doc says "a
  session a sign-in creates keeps it" (C20). The session built from a
  pasted access token is given none (C19). It has no refresh, so no request
  depends on it today. The doc is wrong for it.
* **W12. The refused-sign-in error repeats a separator** **[probed]**
  *(Withdrawn 2026-10-24: the probe's artifact. See the amendment.)*
  (P8). P8's error reads `oauth: refresh failed: : {"error":…}`. It is in
  `llmprovider/auth`, not the wizard, and is routed there.
* **W13. A refused key from the environment or the saved config ends the
  run** **[read]** (C21). A typed key the listing refuses is asked for
  again, up to three (C21). A key from `ANTHROPIC_API_KEY` or from
  `Options.Existing` that it refuses ends the run, naming its source. This
  is 0028-MADR D-H10, as decided; it is listed because a user with a stale
  environment variable cannot type a key in the same run, and Q6 asks
  whether to keep it.

## Decision Drivers

* A notice about a credential must be true, and must not outlive the fact
  that disproves it (W1, W3).
* Nothing the wizard does may lose a working credential (W2).
* A path a user takes must not end where it could go on (W6, W7).
* The wizard must not spin or hang on a `Prompter` that keeps its contract
  (W5).
* No change to an exported name's signature. A behaviour that changes is
  documented in the same change (R52).
* Each fix is a test seen to fail first, on today's code or on a planted
  break.

## Considered Options

* **R1.** Remediate in a 0059 PLAN, by severity, after the design
  questions below.
* **R2.** Fix W1 alone now, and leave the rest to a later pass.
* **R3.** Fold each finding into the record that owns its area: 0028 for
  W1 and W3, 0057 for W5, 0020 for W7.

For W1, the shape check itself:

* **S1.** Warn before the listing only for a key that is plainly another
  vendor's, by `foreignKeyVendor`, for every provider. Drop the exact-shape
  warning. When the listing refuses a key that also lacks its provider's
  shape, say so in the refusal notice. Without `Discover`, claim no check.
* **S2.** Keep the warning before the listing, with looser shapes: a
  prefix family such as `sk-ant-<tag><nn>-` and a length range. Word it by
  whether a listing runs, and add a notice when the listing accepts the
  key.
* **S3.** Remove the shape warning, and keep only `foreignKeyVendor` for
  the OpenAI paste, as before 0028.

## Decision Outcome

Proposed: **R1 with S1**, because S1 removes W1's false warning for every
provider, not only Claude, and puts the shape where it explains something:
in the notice that a key was refused. It keeps 0028's useful part, naming
another vendor's key, and stops relying on lengths no vendor publishes.
R1 keeps one record for one pass, as 0020 did.

### D1. Phases for the PLAN, once this record is accepted

1. **W1 and W3:** the shape check, by Q1's answer.
2. **W2:** the session kept until the new one is checked, by Q2's answer.
3. **W4 and W5:** the masked entry's escape state kept across reads (an
   ESC at the end of the buffer waits for its next byte), and the paste
   prompt's empty answers paced, by Q4's answer.
4. **W6 and W7:** an unusable saved session not offered, by Q3's answer;
   saved fallbacks offered, by Q5's answer.
5. **W8, W9, W10, W11:** the context checked first; `NoEndpointPrompt` by
   Q7's answer; a seam for raw mode, so that `secretOnce` is tested without
   a terminal; `HTTPClient` given to the pasted session.
6. *(Withdrawn 2026-10-24, with W12.)* **W12** is routed to
   `llmprovider/auth` in the same PLAN, as its own
   step. **W13** changes only if Q6 says so.

Each phase is tests first: P1 to P8 become tests that fail on today's
code. Each phase updates the docs of what it changes, and `CHANGELOG.md`.
The release is a patch or a minor one, by whether a behaviour a consumer
sees changes (W1, W2, W6).

### Consequences

* Good, because the reported bug, and the class it belongs to, stops: no
  key that works is called the wrong shape.
* Good, because a sign-in cannot lose the session that worked (W2).
* Good, because the paths a user takes go on where they used to end (W6,
  W7), and the masked entry types only what was pasted (W4).
* Neutral, because a mistyped key of the right vendor gets no warning
  before the listing. The listing then refuses it, and the refusal names
  the shape (S1).
* Bad, because with `Discover` off, a wrong key gets no warning at all,
  where today a wrongly shaped one does.
* Bad, because the pass is 13 findings in 6 phases, which is more work
  than W1 alone (R2).

### Confirmation

* P1 to P8, made tests, each failing on `2d6e7f3` and passing after its
  phase.
* For W1: a key of `sk-ant-api04-`, of 109 characters, and the standard
  one, each accepted by a stubbed listing, with no warning; a refused key
  without the shape, with the shape named in the refusal notice.
* `make pre-add-check`, `python3 -B tools/gate.py`, and CI on the three
  platforms.
* `secretOnce` covered by a test (W10).

## Pros and Cons of the Options

### R1. A 0059 PLAN, by severity

* Good, because one record holds the pass, its evidence and its fixes.
* Bad, because W1, which the owner reported, waits for the design
  questions. Phase 1 is W1, so it is first.

### R2. W1 alone now

* Good, because the reported bug is fixed soonest.
* Bad, because W2, which loses a working session, waits for another pass.

### R3. Fold into the owning records

* Good, because each fix sits with its decision.
* Bad, because five records reopen at once, and the pass's evidence is
  split among them.

### S1. Name another vendor's key; the shape only in a refusal

* Good, because no working key is warned about, for any provider.
* Good, because the shape still explains a refusal, where it helps.
* Bad, because a mistyped key gets no warning until the listing answers.

### S2. Looser shapes, before the listing

* Good, because a malformed key is still warned about early.
* Bad, because the lengths are still guesses (C5): the next unseen variant
  is warned about again.
* Bad, because the early warning and the later acceptance still disagree,
  until a second notice says so.

### S3. No shape check

* Good, because it is the simplest.
* Bad, because a key pasted for the wrong provider, such as `sk-ant-` for
  Grok, gets no hint until the listing refuses it.

## Owner questions

* **Q0 (W1). The shape of your key, not the key.** It confirms which
  variant P1 reproduces. This prints only the tag, the length, the last two
  characters' class and whether the body is base64url, never the key:

  ```bash
  python3 -c 'import os,re;k=os.environ["ANTHROPIC_API_KEY"].strip();m=re.match(r"sk-ant-[a-z]+[0-9]+-",k);t=m.group(0) if m else "no sk-ant-<tag>NN- prefix";b=k[len(t):] if m else k;print(t,len(k),"ends AA" if k.endswith("AA") else "ends otherwise","body base64url" if re.fullmatch(r"[A-Za-z0-9_-]+",b) else "body has other characters")'
  ```

  The fix does not depend on the answer. *(Open on 2026-10-24: not
  answered, and not needed.)*
* **Q1 (W1, W3). The shape check.**
  * **(a)** S1 (recommended).
  * **(b)** S2.
  * **(c)** S3.

  **Answered 2026-10-24: (a).**
* **Q2 (W2). A fresh sign-in and the saved session.**
  * **(a)** Save the new session only once the listing has accepted it;
    until then the store keeps the old one (recommended). A refresh during
    the check is kept in memory and saved with it.
  * **(b)** Save it at once, as today, and put the old one back when the
    listing refuses the new one.

  **Answered 2026-10-24: (a).**
* **Q3 (W6). A saved session that cannot be used.**
  * **(a)** Check it before asking. One that cannot be used is not offered:
    a notice says why, and the sign-in menu follows (recommended).
  * **(b)** Ask as today; when it cannot be kept, a notice says why, and
    the sign-in menu follows.

  **Answered 2026-10-24: (a).**
* **Q4 (W5). Empty paste answers that come at once.**
  * **(a)** After three empty answers in a row, each returned within
    50 ms, stop asking and wait for the browser, with a notice
    (recommended).
  * **(b)** Wait a short, growing pause between empty answers, and keep
    asking.

  **Answered 2026-10-24: (a).**
* **Q5 (W7). A saved fallback the menu does not offer.**
  * **(a)** Offer it in the blank round, after the recommended rows,
    marked "saved" and preselected, as the model is (recommended).
  * **(b)** Keep it in the Result unless the user clears the selection.

  **Answered 2026-10-24: (a).**
* **Q6 (W13). A refused key from the environment or the saved config.**
  * **(a)** Keep 0028-MADR D-H10: it ends the run, naming its source
    (recommended; no change).
  * **(b)** Ask for a typed key instead, as for a refused typed one.

  **Answered 2026-10-24: (a).**
* **Q7 (W9). `NoEndpointPrompt` and plain http.**
  * **(a)** Use the URL without asking, since the caller holds it, with a
    warning notice (recommended).
  * **(b)** Refuse it with an error, which the caller fixes.

  **Answered 2026-10-24: (a).**

## More Information

### Probes

Each probe is a scratch test in package `wizard`, in a clone of `2d6e7f3`
outside the tree. It logs what the wizard did; none asserts the fix. None
reads or prints a real key: the keys are built to a shape, as
`auth_shapes_0028_test.go` builds them.

* **P1.** `ResolveCredential` for Claude, `Discover: true`, an
  `HTTPClient` whose transport answers `/v1/models` with one model. Three
  typed keys:

  ```text
  prefix sk-ant-api04- len 108: err=<nil> saved=true listing calls=1 notices=["this does not look like a Claude (Anthropic) API key (expected sk-ant-api03- and 95 characters ending in AA); the model listing will check it"]
  prefix sk-ant-api03- len 109: err=<nil> saved=true listing calls=1 notices=["this does not look like … ending in AA); the model listing will check it"]
  prefix sk-ant-api03- len 108: err=<nil> saved=true listing calls=1 notices=[]
  ```

* **P2.** As P1, without `Discover`: `err=<nil> kind=api_key listing
  calls=0`, and the same warning.
* **P3.** `TextPrompter.readMasked` over a reader that returns each given
  string as one `Read`: `entered="sk-ant-keyAA[201~"`, then
  `entered="[200~sk-ant-keyAA"`.
* **P4.** `pastePrompt.input` with a `Prompter` whose `Input` returns `""`
  and no error, and a 100 ms context: `in 100 ms: 517770 Input calls,
  err=context canceled`.
* **P5.** `ConfigureLLM` for Claude, `NeedFallbacks`, `Existing` naming
  the fallback `claude-legacy-fallback`, a blank fallback round: `err=<nil>
  fallbacks=[]`.
* **P6.** `ConfigureLLM` on a cancelled `ctx`: `err=select model: context
  canceled; asked: selects=["Choose an LLM provider:"] secrets=["Enter your
  Claude (Anthropic) API key"]`.
* **P7.** `ConfigureLLM` for OpenAI, a stored session expired and with no
  refresh token, `Existing` naming it: `err=wizard: the saved OpenAI session
  cannot be kept (oauth: no refresh token); sign in again;
  confirms=["Keep the existing session (••••••••abcd)?"]; method menus
  shown=0`.
* **P8.** `ConfigureLLM` for OpenAI with a good session in the store, the
  browser sign-in stubbed to return a new one, and an `HTTPClient` that
  answers 401: `err=wizard: OpenAI refused the sign-in; sign in again: …
  oauth: refresh failed: : {"error":…}; store now holds access
  "fresh-access-wxyz" (before: "previous-access-abcd")`.

### Coverage

`go test -coverprofile ./wizard/` at `2d6e7f3`: 91.9 %. Below 85 %:
`secretOnce` 26.7 %, `skippedEvery` 57.1 %, `accessTokenAccount` 70.0 %,
`resolveKiloDeviceKey` 75.0 %, `resolveVendorCLI` 76.5 %, `saveOAuthCredential`
77.8 %, `checkAccessToken` 78.3 %. Apart from `secretOnce` (W10), each
uncovered statement is an error return: a store that fails, a vendor CLI
file that cannot be read, a malformed token.

### The scanners' rules

* TruffleHog, `pkg/detectors/anthropic/anthropic.go`:
  `\b(sk-ant-(?:admin01|api03)-[\w\-]{93}AA)\b`.
* gitleaks, `config/gitleaks.toml`, rules `anthropic-api-key` and
  `anthropic-admin-api-key`: `\b(sk-ant-api03-[a-zA-Z0-9_\-]{93}AA)` and
  `\b(sk-ant-admin01-[a-zA-Z0-9_\-]{93}AA)`, each followed by a boundary.

Both match the wizard's Claude shape exactly. A search found a secondary
summary naming `sk-ant-api04-` keys in GitGuardian's detector examples. The
GitGuardian page itself, as fetched, shows no examples, so that is
unconfirmed. Q0 settles it for the owner's key.

### Related

* [0028-MADR-heuristics-and-performance-from-the-research-pass.md](0028-MADR-heuristics-and-performance-from-the-research-pass.md)
  (H10b, D-H10) and its PLAN (D3): the shape check and the refusal loop.
* [0057-MADR-wizard-device-code-and-paste-prompt.md](0057-MADR-wizard-device-code-and-paste-prompt.md)
  (D4): the empty paste asked again.
* [0013-MADR-remediate-debugging-pass-findings.md](0013-MADR-remediate-debugging-pass-findings.md)
  (C3) and
  [0020-MADR-remediate-v1-debugging-pass-findings.md](0020-MADR-remediate-v1-debugging-pass-findings.md)
  (F50): the saved fallbacks.

### Code claims

<!-- claims: self=2d6e7f3 gobble-cli=142685b -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | Claude's exact shape | `wizard/auth.go:537` | `admin01)-[A-Za-z0-9_-]{93}AA$` |
| C2 | The warning, and its promise | `wizard/auth.go:555` | `the model listing will check it` |
| C3 | It runs as the key is typed | `wizard/configure.go:525` | `warnKeyShape(p, d, key)` |
| C4 | No listing without Discover | `wizard/configure.go:553` | `if !o.Discover {` |
| C5 | No vendor publishes a format | `docs/decisions/0028-PLAN-heuristics-and-performance-from-the-research-pass.md:769` | `No vendor publishes a full format.` |
| C6 | One key measured | `docs/decisions/0028-PLAN-heuristics-and-performance-from-the-research-pass.md:779` | `` `sk-ant-api03-` + 93 + `AA`, 108 in all `` |
| C7 | A lone ESC at the buffer's end | `wizard/text_prompter.go:495` | `if r.Buffered() == 0 {` |
| C8 | Empty answers asked again | `wizard/auth.go:745` | `if err != nil \|\| strings.TrimSpace(answer) != "" {` |
| C9 | A sign-in's session saved at once | `wizard/auth.go:815` | `if err := store.Save(ctx, provider, session); err != nil {` |
| C10 | Kilo's too | `wizard/auth.go:343` | `if err := o.TokenStore.Save(ctx, d.ID, session); err != nil {` |
| C11 | An unusable kept session ends the run | `wizard/auth.go:450` | `cannot be kept (%w); sign in again` |
| C12 | Asked before it is checked | `wizard/auth.go:436` | `keep, err := p.Confirm(` |
| C13 | Only offered fallbacks are preselected | `wizard/model_select.go:185` | `indicesOf(recs, saved)` |
| C14 | ConfigureLLM starts with the menu | `wizard/configure.go:194` | `d, err := chooseProvider(p, o)` |
| C15 | ResolveCredential returns on an ended ctx | `wizard/configure.go:270` | `return res, cat, nil` |
| C16 | NoEndpointPrompt checks a remote URL | `wizard/configure.go:408` | `return validateBaseURL(p, def)` |
| C17 | which asks about plain http | `wizard/base_url.go:37` | `so the key travels unencrypted. Use it?` |
| C18 | Raw mode | `wizard/text_prompter.go:376` | `func (p *TextPrompter) secretOnce(prompt string) (string, error) {` |
| C19 | The pasted token's session | `wizard/auth.go:610` | `func accessOnlyOpenAISession(access string) *auth.OAuthSession {` |
| C20 | HTTPClient's doc | `wizard/configure.go:115` | `keeps it. Nil uses the package default.` |
| C21 | A typed key is asked again | `wizard/configure.go:340` | `for keys := 1; refusedCredential(listErr); keys++ {` |
| G1 | gobble lists | `gobble-cli:llm/provider/signin.go:176` | `Discover:         true,` |
| G2 | gobble asks no endpoint | `gobble-cli:llm/provider/signin.go:184` | `NoEndpointPrompt: true,` |

## Amendment 2026-10-24: the owner's answers, W12 withdrawn, and 0028

* **The answers.** The owner answered Q1 to Q7 on 2026-10-24, "follow
  recommendations": (a) each. Q0 is not answered; the fix does not need it.
* **W12 is withdrawn.** Its doubled separator is
  `fmt.Errorf("oauth: %s failed: %s: %s", op, …(resp.Status), …)` in
  `llmprovider/auth/oauth_session.go`, with the status line empty. P8's
  stub set a status code and no status line, which a real HTTP response
  always has. So it is the probe's artifact, not a defect. The pass has 12
  findings, 5 of them low, and the PLAN has no step for W12.
* **S1 and 0028.** S1 replaces the warning about a provider's shape that
  0028-MADR H10b added and 0028-PLAN D3 step 3 built: a typed key is no
  longer checked against `keyShapes` before the listing. The rest of D3
  stands. `foreignKeyVendor` names another vendor's key, now for every
  provider, and `keyShapes` names the expected shape in the notice of a
  refusal. 0028's records are not changed; this record is the later one.
