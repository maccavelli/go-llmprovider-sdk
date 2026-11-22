---
status: accepted
date: 2026-11-17
decision-makers: repository owner
consulted: "the owner's request of 2026-11-17 and the owner's answers to Q1 to Q7 and, later the same day, to N1 to N7 and Q8, relayed by the coordinator; this repository at b109cea (v1.31.0 and five commits, none in the code read here); gobble-cli at 2004980 (branch p3-acpext-v0.5.0; the draft first read cecc892 and gobble's staged working tree, which 2004980 committed the same day); prepare-commit-msg at 2715f2d; magic-cli-remote at e7de8568; the codex checkout at 57e64682f8, the opencode checkout at 4ce137b39e, the kilocode checkout at 58072bd696 and the grok-build checkout at 2bdd1d6a, read only; a read-only pass of 2026-11-17 over who can tell the wizard that a browser is remote; the independent reviewer's port measurements of gobble-cli's 0052-PLAN P4; two runs of this module's tests made for this record, on a scratch extraction of b109cea"
informed: gobble-cli, whose two terminal plan sign-in tests wait on the option by the owner's ruling; prepare-commit-msg, the other Go caller of the wizard found; every caller of wizard.ConfigureLLM and wizard.ResolveCredential
---

<!-- markdownlint-disable MD013 MD024 -->

# Let the Wizard Run a Browser Sign-In With No Callback Listener, Completed by a Paste, When the Caller Says So or the Person Says the Browser Is Elsewhere

## Context and Problem Statement

The wizard's browser sign-ins (OpenAI's `browser_oauth`, Grok's
`browser_oauth` and OpenAI's `chatgpt_plan`) always bind a callback
listener. OpenAI's two bind one of OpenAI's registered ports, 1455 or
1457, on both loopback addresses. `wizard.Options` has no way to skip it
(C40 to C45), although every layer beneath the wizard already can: the two
`auth` option structs have had a v1 `NoLoopback` field since 0074-MADR D4
(C9, C19, R1), the step engine `internal/signinflow` passes its
`Spec.NoLoopback` to them (C29, C34), and the experimental
`signin.Options.NoLoopback` sets it (C36, C37). The wizard sets
`Spec.NoPaste` from its own option but never `Spec.NoLoopback` (C41).

A sign-in with no listener has been verified live for all three methods,
through `x/signin`. OpenAI's browser sign-in and the plan's each completed
by a callback URL pasted from a phone, with the redirect naming port 1455,
where nothing listened (R5, R7). Grok's completed by a pasted code (R8,
R9).

### The owner's request of 2026-11-17

Relayed by the coordinator, in the owner's words: "analyze, research,
assess and evaluate these: - Whether to draft SDK records for the three
other items on its next release: interfaces in place of the concrete
session-type checks, a wizard option to skip the callback listener, and the
typed "tool schema refused" error. and draft the detailed, facts-grounded
and complete madr docs for review." This record is the second item. The
other two are records of their own, not yet numbered.

### The owner's answers of 2026-11-17

Relayed by the coordinator. Each label is quoted as the owner chose it;
what was put to the owner is in this record's words beside it.

* **Q1, the means:** "One option, `wizard.Options.NoLoopback` (Recommended)".
  Put: one boolean the caller sets, passed to the engine's existing field,
  rather than an injectable listener or an automatic fallback.
* **Q2, which sign-ins:** "Every browser sign-in the wizard runs (Recommended)".
  Put: OpenAI's and Grok's `browser_oauth` and `chatgpt_plan`, with device
  sign-ins unaffected, rather than OpenAI's alone or the plan's alone.
* **Q3, who decides:** "The wizard asks the person". Put: before a browser
  sign-in the wizard asks whether the browser is on this machine, when the
  caller allows the question; every Go caller gets it without a flag of
  its own, and every browser sign-in costs one more prompt. Not chosen: the
  caller alone, by the option (the draft's recommendation); the wizard
  detecting a remote session by itself. Before choosing, the owner asked
  whether codex, grok, opencode and kilo, as well as gobble, would pass the
  wizard what it needs. The answer, re-checked for this record, is "Who can
  tell the wizard" below.
* **Q4, the name:** "`NoLoopback` (Recommended)". Put: the name `auth`, the
  engine and `x/signin` already use, rather than one of its own.
* **Q5, with the no-paste option:** "An invalid-request error before any question is asked (Recommended)".
  Put: `NoLoopback` with `NoPastePrompt` fails at once, rather than at the
  sign-in or by overriding `NoPastePrompt`.
* **Q6, the wording:** "The draft's two instructions, word for word (Recommended)".
  Put: the two instructions now in D5, rather than today's words or the
  caller's.
* **Q7, ports busy without the option:** "No automatic fallback; correct the advice per method (Recommended)".
  Put: the error stays, with its prefix, and its advice names what the
  person can do for each method.
* **Q8, the release:** answered later the same day, with N1 to N7; see
  "Decided".

Q1 and Q3 together mean both exist: a caller's option that binds no
listener, and a question the wizard puts to the person. How the two fit
was put to the owner as N1 to N7, answered the same day ("Decided"); D1
to D10 state the result.

### The need behind it: gobble-cli's terminal tests

gobble-cli signs a person in two ways. Over ACP it uses `x/signin`, where
it sets `NoLoopback` for a remote client (G5; gobble-cli
`0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D3's
note, G6). In the terminal it uses `wizard.ResolveCredential` (G2, G3),
where its `Remote` option is ignored (G1).

In gobble-cli's 0052-PLAN P4 a reviewer required a test that completes the
plan sign-in in the terminal. The two tests added,
`TestSignInChatGPTPlanTerminal` and `TestPlanAfterLogoutReusesClientTerminal`
(G16, G17), drive the real wizard against an issuer in process. The paste they
write completes the sign-in before any browser could. The wizard still
binds its listener: 127.0.0.1:1455 and [::1]:1455, falling back to 1457.
The reviewer measured (RUN3, RUN4):

* 20 runs alone pass, with only the test binary listening on 1455;
* with 1455 held on both families, the tests fall back to 1457 and pass;
* with 1455 and 1457 held, they fail at once.

Earlier, for gobble's ACP tests, the owner had ruled "Assert what gobble
passes; use no real ports" (G7). For these two tests the owner ruled, on
2026-11-17, relayed by the coordinator: **"Keep them until the SDK offers a
no-listener option"** (G19, G20). Options not chosen, as the coordinator
relayed them: replace them with seam tests that use no real ports; keep
them permanently on real ports. So the tests wait on this module.

### Who can tell the wizard that the browser is elsewhere

Re-checked for this record, read only:

* **Only Go programs reach the wizard.** The two Go callers found:
  * gobble-cli's terminal path (G2, G3) has no remote signal. Its `Remote`
    option is ignored there (G1).
  * prepare-commit-msg calls `wizard.ConfigureLLM` with a `TextPrompter`
    and an `OpenURL` (P1 to P3), at SDK `v1.29.0` (P4). It has no remote
    signal either.
* **magic-cli-remote** imports this module but not the wizard. Its ACP
  `authenticate` sends a method id only (M1), and it pins gobble's `acpext`
  at `v0.4.0` (M2), which predates `AuthenticateMeta` *(the version claim
  is from the 2026-11-17 pass, not re-read here)*.
* **gobble's ACP remote signal goes to `x/signin`, not the wizard.** It is
  `_meta.gobble.remote` (G9), carried to `NoLoopback` (G5). No client sends
  it yet (M1). ACP has no standard field for it *(from the 2026-11-17
  pass)*.
* **The other agents cannot reach the wizard at all.**
  * Codex is Rust, and its app server's login types are its own (U19).
  * opencode is TypeScript, and its ACP `authenticate` accepts its own
    method only (U20).
  * Grok's CLI is Rust and carries its own `headless` flag on ACP
    `authenticate` (U21).
  * kilocode is TypeScript, with its own headless method (U17).
* **So with no question, no person using any caller found could tell the
  wizard the browser is elsewhere.** That is what Q3's answer gives them.

### Who else needs it

* **This module's own tests do not need the option.**
  * The wizard's tests replace the two login functions through package
    seams (C38, C39, C48, C49; 0074-MADR D6, R2).
  * `auth`'s tests swap the registered ports for ports `internal/nettest`
    hands out (C46, C47; 0045-MADR D2, R16).
  * With 1455 and 1457 held on both families, the `wizard`,
    `internal/signinflow` and `x/signin` tests all pass: 517 PASS lines in
    all (tests and subtests), none skipped (RUN1). A control shows the hold
    does stop a real bind (RUN2).
  * Neither seam is exported, and a caller outside the module can reach
    neither.
  * The wizard's test fake refuses a `Select` it was not scripted with
    (C61). A new question in every browser sign-in meets every test that
    runs one through the fake. A prototype of the question on scratch
    copies (RUN5, RUN6) found 30 such tests in 13 files.
  * *(Corrected 2026-11-17.)* N1 was put with "44 places". That was a
    `grep` count of the login stubs in 15 files, which counts helper
    definitions and tests that run with a `Driver` or `NoPastePrompt`. The
    tests that actually meet the question are the 30 RUN5 and RUN6 list.
* **A person whose browser is on another machine** (SSH, a container, a
  remote desktop). Today the wizard already races a paste against the
  listener (0008-MADR D3, R10), so such a person can finish by pasting.
  But the listener is still bound on the remote host, the instruction says
  "If the browser does not return here" (C42), which it never can, and the
  bind can fail there (C5). *(Inferred: no report from such a user is on
  record.)*
* **A machine whose registered ports are taken.** With 1455 and 1457 both
  held, the browser and plan sign-ins fail before the URL is shown and
  before any paste is asked (C8, C5), with "registered callback ports
  unavailable; use device-code login" (RUN2's text). For the plan that
  advice is wrong: it has no device flow (C20). One holder on 1455 is
  harmless: the wizard falls back to 1457 (C2; RUN3).
* **Another agent on the same host.**
  * Codex binds 127.0.0.1:1455, sends `/cancel` to whatever holds it, and
    after about two seconds falls back to 1457 (U1 to U7).
  * opencode does the same (U11 to U13).
  * kilocode's newer OpenAI plugin binds `localhost:1455` with no fallback
    (U15, U16), so while the wizard holds 1455 that login fails *(inferred
    from the code; not run)*.
  * The wizard's listener answers `/cancel` with a 404: it serves only its
    callback path (C18).

### The wizard's browser sign-ins today

Read at `b109cea`.

| Method | Listener | Redirect URI | Fallback | When it cannot bind |
| :--- | :--- | :--- | :--- | :--- |
| OpenAI `browser_oauth` | 127.0.0.1 and ::1 on 1455, else 1457 (C1 to C4) | `http://127.0.0.1:{port}/auth/callback` (C17) | to 1457 (C2) | error C5, before the URL is shown (C8) |
| OpenAI `chatgpt_plan` | the same function (C21) | the same | the same | the same, after the consent question and the issuer's discovery (C35, C22) |
| Grok `browser_oauth` | 127.0.0.1 only, an ephemeral port (C6) | `http://127.0.0.1:{port}/callback` | none needed | only if the system has no free ephemeral port *(inferred)* |
| any device sign-in | none | none | – | – |

* **How long:** the listener lives for the sign-in, at most 10 minutes
  (C7), and is shut down when it returns.
* **Why both families:** the listener was made dual-stack when OpenAI's
  redirect named `localhost`, which a browser may resolve to ::1
  (0008-MADR, R11). The redirect has named 127.0.0.1 since 0012-MADR §5.3,
  and the ::1 listener was kept as "harmless" (R12, R13). *(Inferred: a
  browser following `http://127.0.0.1:…` never reaches the ::1 listener;
  that listener only holds the port.)*
* **The paste:**
  * Every built-in browser method's `Paste` fact is `url_or_code`, and the
    plan's first sign-in's `FirstPaste` is `url_only` (C25 to C28). A first
    plan sign-in learns its issued client from the callback URL (0043-MADR
    A3, R14).
  * The wizard sets these facts itself for the methods it runs (C40, C24).
  * The engine checks a pasted URL's `state` and, for `url_only`, its
    `client_id` (C31, C32). It takes three pastes in all (C30).
  * `auth` checks the state again (C54); a bare code skips it (C16).
* **Options that bear on it:**
  * `OpenURL` is called with the authorize URL right after the URL and the
    instruction are shown (C63, C64).
  * `NoPastePrompt` leaves the loopback alone to complete the sign-in
    (C45; 0057-MADR D2, R15).
  * `Driver` takes the sign-in's steps, while "the menus, confirmations and
    every other question still use the Prompter" (C43, C62).
  * None removes the listener.
* **Where a browser sign-in starts:**
  * The person picks the method from the menu (C67).
  * A plan sign-in then asks its consent through the engine, before the
    login runs (C35). The wizard renders it as a `Select` of three choices
    (C65).
  * A kept session that the listing refuses leads to the sign-in menu,
    once (C68).
* **The `Prompter` is a stable interface.** Its doc says that changing it
  "is a breaking change" for every consumer (C55). It has `Select` and
  `Confirm` (C56, C57). An optional capability is added as a separate
  interface the wizard checks for, as `InputerContext` is (C58).
  `TextPrompter`'s `Select` returns the default on an empty line and an
  error once the input is exhausted (C59, C60). Any `Prompter` error ends
  the run (C66).

### With no listener

* **The redirect URI** is OpenAI's first registered port and path,
  `http://127.0.0.1:1455/auth/callback`, for the browser and the plan
  sign-ins (C10). For Grok it is `http://127.0.0.1:{p}/callback`, with `p`
  drawn from 49152 to 65535 and never bound (C11).
* **The provider requires a registered port for OpenAI** *(inferred from
  Codex's comment; not probed)*. Codex, with the same client id, keeps its
  two ports "in sync with the Codex CLI Hydra redirect URI allow-list" (U2).
  Grok takes any loopback port: its listener is ephemeral (C6), and its
  no-listener redirect is a random port (C11).
* **What the person sees:** the browser is sent to the loopback address
  and fails to load. In the live run the phone's address bar "showed only
  the host and port until tapped; the full address, with the code and
  state, was copied from it then" (R6). Grok's page showed a code, which
  was pasted (R8, R9).
* **What it needs:** something that reads the paste.
  * `auth` refuses `NoLoopback` with no `InputCode` (C12).
  * The engine refuses it with `NoPaste`, and with a browser method that
    has no `Paste` fact (C33).
  * No built-in browser method lacks one (C25 to C28).
* **No provider flow at `b109cea` is known that a paste cannot complete.**
  A first plan sign-in needs the whole URL (R14), which the person copies
  from the address bar (R6).

### The security properties a design must keep

* **`state`:** generated per sign-in. It is checked by the listener (C53)
  and, on a pasted URL, by the engine and by `auth` (C31, C54). A pasted
  bare code carries none and is taken as it is (C16). Nothing in this
  record changes any of it.
* **PKCE:** the challenge is S256 (C13). The verifier is sent only in the
  token exchange (C14) and never leaves the process.
* **`redirect_uri` equality:** one string, built once (C8), is sent in the
  authorize URL and again in the exchange (C15, C14; the plan's, C23). With
  no listener it is still that string.
* **The person's answer** to the question chooses between two flows that
  each keep these properties. It is not a credential and is not stored.

### Evidence index

<!-- claims: self=b109cea gobble-cli=2004980 codex=57e64682f8 opencode=4ce137b39e kilocode=58072bd696 prepare-commit-msg=2715f2d magic-cli-remote=e7de8568 grok-build=2bdd1d6a -->

Each row was read at the revision the marker names.

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | OpenAI's registered ports | `llmprovider/auth/oauth_loopback.go:42` | `var openaiLoopbackPorts = []int{1455, 1457}` |
| C2 | The browser listener tries them in order | `llmprovider/auth/oauth_loopback.go:469` | `listeners, port, err := listenOpenAILoopback(openaiLoopbackPorts)` |
| C3 | on 127.0.0.1 | `llmprovider/auth/oauth_loopback.go:504` | `v4, err4 := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", portText))` |
| C4 | and on ::1 | `llmprovider/auth/oauth_loopback.go:505` | `v6, err6 := net.Listen("tcp6", net.JoinHostPort("::1", portText))` |
| C5 | Both taken is this error | `llmprovider/auth/oauth_loopback.go:499` | `registered callback ports unavailable; use device-code login` |
| C6 | Grok's listener is ephemeral, IPv4 | `llmprovider/auth/oauth_loopback.go:477` | `listener, err := net.Listen("tcp", "127.0.0.1:0")` |
| C7 | A browser sign-in lasts at most 10 minutes | `llmprovider/auth/oauth_loopback.go:27` | `oauthBrowserTimeout         = 10 * time.Minute` |
| C8 | The listener is bound first, and the redirect URI built once | `llmprovider/auth/oauth_loopback.go:204` | `listeners, redirectURI, callbackPath, err := browserEndpoint(config.provider, config.noLoopback)` |
| C9 | `OAuthFlowOptions.NoLoopback` (v1) | `llmprovider/auth/oauth_loopback.go:69` | `NoLoopback bool` |
| C10 | With no listener, OpenAI's redirect names the first port | `llmprovider/auth/oauth_loopback.go:462` | `return nil, fmt.Sprintf("http://127.0.0.1:%d%s", openaiLoopbackPorts[0], openAICallbackPath), openAICallbackPath, nil` |
| C11 | and Grok's a port never bound | `llmprovider/auth/oauth_loopback.go:460` | `return nil, fmt.Sprintf("http://127.0.0.1:%d/callback", port), "/callback", nil` |
| C12 | `NoLoopback` needs an `InputCode` | `llmprovider/auth/oauth_loopback.go:268` | `NoLoopback binds no listener, so the sign-in needs an InputCode to complete it` |
| C13 | PKCE is S256 | `llmprovider/auth/oauth_loopback.go:584` | `query.Set("code_challenge_method", "S256")` |
| C14 | The exchange sends the verifier | `llmprovider/auth/oauth_loopback.go:805` | `"code_verifier":     {verifier},` |
| C15 | and the same redirect URI | `llmprovider/auth/oauth_loopback.go:256` | `return login.exchange(flowCtx, callback, redirectURI)` |
| C16 | A pasted bare code is taken as it is | `llmprovider/auth/oauth_loopback.go:784` | `return oauthCallbackResult{code: input}` |
| C17 | OpenAI's redirect host is 127.0.0.1, as Codex's | `llmprovider/auth/oauth_loopback.go:473` | `// 127.0.0.1, as Codex redirects with the same client id since` |
| C18 | The listener serves only its callback path | `llmprovider/auth/oauth_loopback.go:619` | `mux.HandleFunc(path, func(w http.ResponseWriter, request *http.Request) {` |
| C19 | `ChatGPTPlanOptions.NoLoopback` (v1) | `llmprovider/auth/oauth_plan.go:83` | `NoLoopback bool` |
| C20 | The plan has no device flow | `llmprovider/auth/oauth_plan.go:101` | `matching llmprovider.ErrNotPermitted. There is no device flow.` |
| C21 | The plan runs the same browser function | `llmprovider/auth/oauth_plan.go:139` | `return runBrowserLogin(ctx, browserLogin{` |
| C22 | after the issuer's discovery | `llmprovider/auth/oauth_plan.go:119` | `endpoints, err := planEndpointsFor(ctx, config)` |
| C23 | The plan's exchange sends the same redirect URI | `llmprovider/auth/oauth_plan.go:285` | `"redirect_uri":      {redirectURI},` |
| C24 | The plan's paste needs the URL on a first sign-in | `llmprovider/auth/oauth_plan.go:70` | `// loopback listener; a first sign-in needs the whole URL, which names the` |
| C25 | Grok's browser method takes a paste | `llmprovider/providers/grok/descriptor.go:43` | `Paste:          "url_or_code",` |
| C26 | so does OpenAI's | `llmprovider/providers/openai/descriptor.go:45` | `Paste:          "url_or_code",` |
| C27 | and the plan's | `llmprovider/providers/openai/descriptor.go:96` | `Paste:          "url_or_code",` |
| C28 | whose first sign-in takes the URL only | `llmprovider/providers/openai/descriptor.go:97` | `FirstPaste:     "url_only",` |
| C29 | The engine's `Spec.NoLoopback` | `internal/signinflow/signinflow.go:136` | `NoLoopback bool` |
| C30 | Three pastes in all | `internal/signinflow/signinflow.go:33` | `const maxPastes = 3` |
| C31 | A pasted URL's state is checked | `internal/signinflow/paste.go:65` | `case c.state != "" && query.Get("state") != c.state:` |
| C32 | and, for `url_only`, its client | `internal/signinflow/paste.go:71` | `case c.accepts == authcontract.PasteURLOnly && query.Get("client_id") == "":` |
| C33 | The engine refuses `NoLoopback` with `NoPaste` | `internal/signinflow/signinflow.go:231` | `NoLoopback with NoPaste leaves nothing to complete the sign-in` |
| C34 | and passes it to the plan login | `internal/signinflow/signinflow.go:432` | `opts.OpenURL, opts.InputCode, opts.NoLoopback = openURL, inputCode, spec.NoLoopback` |
| C35 | The plan's consent is asked before the login runs | `internal/signinflow/signinflow.go:403` | `if answered, err = r.askConsents(ctx, spec); err != nil {` |
| C36 | `signin.Options.NoLoopback` (experimental) | `llmprovider/x/signin/signin.go:60` | `NoLoopback bool` |
| C37 | is passed to the engine | `llmprovider/x/signin/signin.go:193` | `NoLoopback:  o.NoLoopback,` |
| C38 | The wizard's browser login seam | `wizard/auth.go:42` | `loginBrowserOAuth = auth.LoginBrowserOAuth` |
| C39 | and plan login seam | `wizard/auth.go:43` | `loginChatGPTPlan  = auth.LoginChatGPTPlan` |
| C40 | The wizard states the browser method's paste itself | `wizard/auth.go:240` | `Method: authcontract.Method{ID: string(method), Paste: authcontract.PasteURLOrCode,` |
| C41 | The wizard sets `NoPaste`, never `NoLoopback` | `wizard/auth.go:824` | `spec.NoPaste = o.NoPastePrompt` |
| C42 | The paste instruction | `wizard/auth.go:763` | `pasteCodeInstruction = "If the browser does not return here, paste the redirected URL or the authorization code"` |
| C43 | `Options.Driver` | `wizard/configure.go:175` | `Driver StepDriver` |
| C44 | `Options.OpenURL` | `wizard/configure.go:131` | `OpenURL func(string) error` |
| C45 | `Options.NoPastePrompt` | `wizard/configure.go:205` | `NoPastePrompt bool` |
| C46 | `auth`'s tests swap the registered ports | `llmprovider/auth/no_loopback_0074_test.go:29` | `ports := []int{nettest.Port(t), nettest.Port(t)}` |
| C47 | `nettest.Port` | `internal/nettest/nettest.go:65` | `func Port(t testing.TB) int {` |
| C48 | The wizard's tests stub the plan login | `wizard/plan_0043_test.go:43` | `func stubPlanLogin(t *testing.T, login func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error)) *int {` |
| C49 | and the browser login | `wizard/auth_test.go:506` | `func stubBrowserLogin(t *testing.T, login func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (*auth.OAuthSession, error)) {` |
| C50 | One test binds the real ports, and skips when they are busy | `llmprovider/auth/oauth_redirect_test.go:16` | `t.Skipf("OpenAI loopback ports busy: %v", err)` |
| C51 | The open_url rendering | `wizard/auth.go:873` | `s.p.Notify(LevelInfo, "Open %s in your browser", authorizeURL)` |
| C52 | A failed sign-in records no consent answer | `internal/signinflow/signinflow.go:467` | `login returns, and nowhere before: a sign-in that fails records none` |
| C53 | The listener checks the callback's state | `llmprovider/auth/oauth_loopback.go:636` | `if !callbackStateMatches(query.Get("state"), state) {` |
| C54 | and `auth` checks a pasted URL's | `llmprovider/auth/oauth_loopback.go:773` | `if !callbackStateMatches(query.Get("state"), expectedState) {` |
| C55 | Changing `Prompter` breaks its consumers | `wizard/prompter.go:66` | `library twelve repositories consume, so changing it later is a breaking` |
| C56 | `Prompter.Select` | `wizard/prompter.go:74` | `Select(title string, choices []Choice, defaultIdx int) (int, error)` |
| C57 | `Prompter.Confirm` | `wizard/prompter.go:82` | `Confirm(question string, def bool) (bool, error)` |
| C58 | An optional capability is its own interface | `wizard/prompter.go:114` | `type InputerContext interface {` |
| C59 | `TextPrompter.Select` takes the default on an empty line | `wizard/text_prompter.go:244` | `return defaultIdx, p.flushErr()` |
| C60 | and errors once the input is exhausted | `wizard/text_prompter.go:251` | `return 0, fmt.Errorf("wizard: input exhausted while selecting")` |
| C61 | The wizard's test fake refuses an unscripted `Select` | `wizard/fake_prompter_test.go:106` | `return 0, fmt.Errorf("fakePrompter: unexpected Select(%q)", title)` |
| C62 | With a `Driver`, every other question uses the `Prompter` | `wizard/configure.go:171` | `// NoPastePrompt still decides whether a paste is asked, and the menus,` |
| C63 | `OpenURL`'s doc | `wizard/configure.go:128` | `// OpenURL opens an OAuth authorization URL. Nil reports the URL through` |
| C64 | The URL and instruction are shown, then `OpenURL` is called | `wizard/auth.go:797` | `paste.show(authorizeURL)` |
| C65 | The consent is a `Select` of three | `wizard/driver.go:230` | `choice, err := r.p.Select(title, consentChoices, 1)` |
| C66 | A `Prompter` error ends the run | `wizard/configure.go:259` | `// A Prompter's error ends the flow and is returned, wrapped or as it is, so` |
| C67 | The method is chosen from the menu | `wizard/auth.go:218` | `method := methods[idx].ID` |
| C68 | A refused kept session leads to the sign-in menu once | `wizard/configure.go:250` | `sign-in menu, once, without it; the store keeps it until a new session is` |
| C69 | v1.31.0 changed a wizard question in a minor release | `CHANGELOG.md:93` | `- The wizard's plan notice is one question with three choices, the notice and` |
| R1 | 0074-MADR D4: the two `auth` fields are v1 | `docs/decisions/0074-MADR-one-sign-in-handle-and-method-facts.md:442` | `` `OAuthFlowOptions.NoLoopback` and `ChatGPTPlanOptions.NoLoopback` are part `` |
| R2 | 0074-MADR D6: the wizard keeps its own seams | `docs/decisions/0074-MADR-one-sign-in-handle-and-method-facts.md:644` | `wizard gives it its own test seams, so every wizard test keeps its seams and` |
| R3 | 0074-MADR D4: two consumers no longer contend | `docs/decisions/0074-MADR-one-sign-in-handle-and-method-facts.md:447` | `longer contend for ports 1455 and 1457 (F14) when neither needs the` |
| R4 | 0074-MADR D4: with it, only a paste completes a sign-in | `docs/decisions/0074-MADR-one-sign-in-handle-and-method-facts.md:446` | `port, and only a paste completes the sign-in. Two consumers on one host no` |
| R5 | 0074-PLAN Phase 7 step 2: the browser sign-in by paste, no listener | `docs/decisions/0074-PLAN-one-sign-in-handle-and-method-facts.md:2740` | `FINDING redirect: the authorize URL's redirect_uri names port 1455, where nothing listens` |
| R6 | The address bar | `docs/decisions/0074-PLAN-one-sign-in-handle-and-method-facts.md:2752` | `showed only the host and port until tapped; the full address, with the` |
| R7 | Step 3: the plan by paste | `docs/decisions/0074-PLAN-one-sign-in-handle-and-method-facts.md:2776` | `FINDING sign-in: OpenAI plan sign-in completed by a paste` |
| R8 | Phase 3a: Grok by a pasted code | `docs/decisions/0074-PLAN-one-sign-in-handle-and-method-facts.md:1639` | `FINDING sign-in: completed by a paste of a bare code` |
| R9 | whose page showed the code | `docs/decisions/0074-PLAN-one-sign-in-handle-and-method-facts.md:1650` | `in chrome, asked for permission, then provided the code.` |
| R10 | 0008-MADR D3: the wizard races a paste | `docs/decisions/0008-MADR-repair-oauth-loopback-and-session-wiring.md:249` | `3. **D3 — Wizard always races loopback against paste-code.**` |
| R11 | 0008-MADR: an IPv6 miss | `docs/decisions/0008-MADR-repair-oauth-loopback-and-session-wiring.md:147` | `Consequence: a Windows IPv6 miss` |
| R12 | 0012-MADR §5.3: the redirect names 127.0.0.1 | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:804` | `` `http://127.0.0.1:{port}/auth/callback`. Evidence: `` |
| R13 | and the dual-stack listener stays, harmless | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:812` | `` dual-stack `localhost` listener stays; it remains harmless afterwards. `` |
| R14 | 0043-MADR A3: a first plan sign-in needs the URL | `docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md:385` | `completes a re-sign-in only, and a first sign-in needs the whole URL.` |
| R15 | 0057-MADR D2 | `docs/decisions/0057-MADR-wizard-device-code-and-paste-prompt.md:198` | `` ### D2. `Options.NoPastePrompt` `` |
| R16 | 0045-MADR D2: test ports | `docs/decisions/0045-MADR-tests-never-reuse-a-freed-port.md:105` | `` `Port` returns a loopback port in `[20000, 30000)`: `` |
| R17 | R48: `wizard` is under the v1 promise | `docs/guides/api-standards.md:317` | `` - **R48. From `v1.0.0`, every non-internal package outside `llmprovider/x/` is `` |
| R18 | R55: a new export names its call site | `docs/guides/api-standards.md:367` | `- **R55. A new export is justified.** A PLAN that adds an exported name` |
| R19 | R4: experimental API only under `x/` | `docs/guides/api-standards.md:77` | `` - **R4. Experimental API lives under `llmprovider/x/`** and nowhere else. It is `` |
| R20 | `x/signin` is experimental | `llmprovider/x/signin/doc.go:19` | `This package is experimental (R4 of the API standards): it is outside the` |
| R21 | R46: a moved test keeps its meaning | `docs/guides/api-standards.md:307` | `- **R46. A moved or ported test keeps its meaning.** Its call syntax may change;` |
| G1 | gobble's `SignIn` ignores `Remote` | `gobble-cli:llm/provider/signin.go:90` | `Remote bool` |
| G2 | gobble's terminal sign-in is the wizard | `gobble-cli:llm/provider/signin.go:95` | `var resolveCredential = wizard.ResolveCredential` |
| G3 | called with these options | `gobble-cli:llm/provider/signin.go:345` | `res, _, err := resolveCredential(ctx, w, wizard.Options{` |
| G4 | including `NoPastePrompt` with its `Steps` | `gobble-cli:llm/provider/signin.go:358` | `NoPastePrompt:    o.NoPaste \|\| o.Steps != nil,` |
| G5 | gobble's ACP path sets `NoLoopback` | `gobble-cli:llm/provider/signinrun.go:107` | `NoLoopback:  o.Remote,` |
| G6 | 0051-MADR D3's note: the ports stay free | `gobble-cli:docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md:399` | `Ports 1455 and 1457 stay free for other` |
| G7 | the ruling "Assert what gobble passes; use no real ports" | `gobble-cli:docs/decisions/0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md:653` | `"Assert what gobble passes; use no real ports"` |
| G8 | gobble's pin | `gobble-cli:go.mod:17` | `github.com/maccavelli/go-llmprovider-sdk v1.31.0` |
| G9 | gobble's ACP remote field | `gobble-cli:acpext/auth.go:48` | `` Remote bool   `json:"remote,omitzero"` `` |
| G10 | gobble's wizard prompter passes a `Select` to its own `Choose` | `gobble-cli:llm/provider/signin.go:416` | `func (w *wizardPrompter) Select(title string, choices []wizard.Choice, def int) (int, error) {` |
| G11 | gobble's `--json` path asks nothing | `gobble-cli:llm/provider/signin.go:74` | `// which are then not called. No paste is asked, and a step that asks` |
| G12 | gobble's terminal test helper | `gobble-cli:internal/cli/plansignin_test.go:255` | `func terminalPlan(t *testing.T, e *runEnv, ti *loginIssuer, answers ...string)` |
| G13 | replaces `openURL` | `gobble-cli:internal/cli/plansignin_test.go:282` | `replace(t, &openURL, func(_ context.Context, u string) error {` |
| G14 | which writes the callback the person would paste | `gobble-cli:internal/cli/plansignin_test.go:292` | `cb := redirect + "?code=c1&state=" + url.QueryEscape(state)` |
| G15 | and replaces `signInWith`, where the HTTP client and cache are set | `gobble-cli:internal/cli/plansignin_test.go:296` | `replace(t, &signInWith, func(ctx context.Context, id string, p provider.Prompter, o provider.SignInOptions` |
| G16 | The first terminal test | `gobble-cli:internal/cli/plansignin_test.go:313` | `func TestSignInChatGPTPlanTerminal(t *testing.T) {` |
| G17 | The second | `gobble-cli:internal/cli/plansignin_test.go:359` | `func TestPlanAfterLogoutReusesClientTerminal(t *testing.T) {` |
| G18 | `signInFailed` names the collision by this text | `gobble-cli:internal/cli/plansignin_test.go:400` | `if strings.Contains(e.Error(), "registered callback ports unavailable") {` |
| G19 | 0052-PLAN P4 step 1c | `gobble-cli:docs/decisions/0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md:453` | `**Step 1c, the terminal tests keep the SDK's callback ports.** "Keep them until the SDK offers a no-listener option"` |
| G20 | the owner's second ruling of the second review | `gobble-cli:docs/decisions/0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md:1540` | `2. "Keep them until the SDK offers a no-listener option"` |
| G21 | R2-B3 | `gobble-cli:docs/decisions/0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md:1559` | `**R2-B3, the terminal tests keep the SDK's ports.**` |
| G22 | The third review: the Codex claim was reasoned, not measured | `gobble-cli:docs/decisions/0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md:1648` | `1. **A measurement the reviewer did not make.**` |
| G23 | 0051-PLAN's note: the P6 ruling not applied to the two tests | `gobble-cli:docs/decisions/0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md:659` | `*(2026-11-17, later: for the two terminal plan sign-in tests of 0052-PLAN P4,` |
| G24 | the developers' guide's note | `gobble-cli:docs/guides/developing.md:89` | `keep those ports until go-llmprovider-sdk offers a wizard option to skip the listener.` |
| P1 | prepare-commit-msg calls the wizard | `prepare-commit-msg:internal/ui/setup.go:274` | `res, err := wizard.ConfigureLLM(ctx, prompter, wizard.Options{` |
| P2 | with a `TextPrompter` | `prepare-commit-msg:internal/ui/setup.go:228` | `prompter := &wizard.TextPrompter{In: in, Out: os.Stdout}` |
| P3 | and an `OpenURL` | `prepare-commit-msg:internal/ui/setup.go:285` | `OpenURL:         openBrowser,` |
| P4 | at SDK v1.29.0 | `prepare-commit-msg:go.mod:6` | `github.com/maccavelli/go-llmprovider-sdk v1.29.0` |
| M1 | magic-cli-remote's ACP `authenticate` sends a method id only | `magic-cli-remote:internal/provider/acpagent/acpagent.go:789` | `conn.Authenticate(initCtx, acp.AuthenticateRequest{MethodId: acp.AuthMethodId(id)})` |
| M2 | it pins gobble's `acpext` at v0.4.0 | `magic-cli-remote:go.mod:15` | `github.com/maccavelli/gobble-cli/acpext v0.4.0` |
| U1 | Codex's first port | `codex:codex-rs/login/src/server.rs:79` | `const DEFAULT_PORT: u16 = 1455;` |
| U2 | its ports are the issuer's allow-list | `codex:codex-rs/login/src/server.rs:80` | `// Keep in sync with the Codex CLI Hydra redirect URI allow-list.` |
| U3 | its second | `codex:codex-rs/login/src/server.rs:81` | `const FALLBACK_PORT: u16 = 1457;` |
| U4 | It binds 127.0.0.1 only | `codex:codex-rs/login/src/server.rs:642` | `let preferred_bind_address = format!("127.0.0.1:{port}");` |
| U5 | It asks the holder to cancel | `codex:codex-rs/login/src/server.rs:666` | `if let Err(cancel_err) = send_cancel_request(port) {` |
| U6 | then falls back | `codex:codex-rs/login/src/server.rs:678` | `"default login callback port is unavailable; falling back to the registered fallback port"` |
| U7 | Its own server ends on `/cancel` | `codex:codex-rs/login/src/server.rs:524` | `"/cancel" => HandledRequest::ResponseAndExit {` |
| U8 | Its port is a public option | `codex:codex-rs/login/src/server.rs:93` | `pub port: u16,` |
| U9 | which its tests set to 0 | `codex:codex-rs/login/tests/suite/login_server_e2e.rs:147` | `port: 0,` |
| U10 | A remote user is sent to the device flow | `codex:codex-rs/cli/src/login.rs:123` | `On a remote or headless machine? Use` |
| U11 | opencode's first port | `opencode:packages/core/src/plugin/provider/openai.ts:19` | `const callbackPort = 1455` |
| U12 | it cancels the holder too | `opencode:packages/core/src/plugin/provider/openai.ts:163` | `` fetch(`http://localhost:${port}/cancel`, { `` |
| U13 | and with both taken, sends the user to the device flow | `opencode:packages/core/src/plugin/provider/openai.ts:129` | `but both are already in use. Stop the processes using those ports or choose Codex device code (legacy)` |
| U14 | opencode's tests keep the fixed ports as the contract | `opencode:.opencode/skills/unit-tests/SKILL.md:68` | `The fixed ports 1455/1457 are the contract there` |
| U15 | kilocode's newer plugin: one port | `kilocode:packages/core/src/plugin/provider/openai.ts:16` | `const callbackPort = 1455` |
| U16 | bound with no fallback | `kilocode:packages/core/src/plugin/provider/openai.ts:88` | `server.listen(callbackPort, "localhost", () => resume(Effect.void))` |
| U17 | and a headless method | `kilocode:packages/core/src/plugin/provider/openai.ts:109` | `label: "ChatGPT Pro/Plus (headless)",` |
| U18 | Kilo's docs name the port | `kilocode:packages/kilo-docs/pages/ai-providers/openai-chatgpt-plus-pro.md:172` | `The direct VS Code and CLI flow uses local port` |
| U19 | Codex's app server has login types of its own, a device code among them | `codex:codex-rs/app-server-protocol/src/protocol/v2/account.rs:97` | `ChatgptDeviceCode,` |
| U20 | opencode's ACP `authenticate` accepts its own method only | `opencode:packages/cli/src/acp/service.ts:146` | `if (params.methodId !== AuthMethodID)` |
| U21 | Grok's CLI carries its own `headless` flag on ACP `authenticate` | `grok-build:crates/codegen/xai-grok-shell/src/agent/mvp_agent/mod.rs:1070` | `headless: bool,` |

**Runs.**

| Id | What | Result | Mark |
| :--- | :--- | :--- | :--- |
| RUN1 | This module's `wizard`, `internal/signinflow` and `llmprovider/x/signin` tests, `go test -count=1 -v`, one package at a time, on a scratch extraction of `b109cea`, with 1455 and 1457 held on 127.0.0.1 and ::1 for the whole run; the ports checked free first | `wizard` exit 0, 437 PASS; `signinflow` exit 0, 19 PASS; `x/signin` exit 0, 61 PASS; no FAIL, no SKIP; no line names 1455, 1457 or the ports error | ran, 2026-11-17 |
| RUN2 | Control for RUN1: `auth`'s `TestBrowserListener_OpenAIRedirectsToLoopbackIP` (C50), which binds the real ports, under the same hold | SKIP, with `oauth: registered callback ports unavailable; use device-code login: listen tcp4 127.0.0.1:1455: bind: address already in use` | ran, 2026-11-17 |
| RUN3 | The reviewer's run of gobble's two tests (0052-PLAN P4): 20 runs alone; 5 with 1455 held; 5 with 1455 and 1457 held; two binaries at once with 1455 held | alone exit 0, 40 PASS, the only listeners `cli.test` on 127.0.0.1:1455 and [::1]:1455; 1455 held exit 0, 10 PASS; both held exit 1, 10 FAIL; two at once exit 0 and 0 | read (the reviewer's output) |
| RUN4 | The reviewer's third round: 5 runs free, 5 with both held | free exit 0, 10 PASS; both held exit 1, 10 FAIL, no SKIP | read (the reviewer's output) |
| RUN5 | A prototype of the question on a scratch extraction of `b109cea`: in `browserSignIn`, a `Select` with the D3 title and choices, asked unless a `Driver` or `NoPastePrompt` is set; then `go test -count=1 -v -timeout 600s ./wizard/` | exit 1; 259 top-level PASS, 30 FAIL, in 13 files (PLAN Phase 1's table); every failure is the fake's scripted `Select` (an unexpected `Select`, a scripted answer consumed by the new question, or one left unused); none hung | ran, 2026-11-17 |
| RUN6 | The same prototype over 0080's staged files (the likely base: `b109cea` with the index's added, changed and renamed files written over it, its deleted ones removed); first without the prototype, then with it | without: exit 0, 305 top-level PASS; with: exit 1, the same 30 tests, 275 PASS | ran, 2026-11-17 |

The reviewer's statement that a real Codex login could not bind 1455
while a test held it is not in either output; gobble's third review
records it as reasoned, not measured (G22). It is true of port 1455, and
Codex then falls back to 1457 (U5, U6) *(inferred from Codex's code; not
run)*.

## Decision Drivers

* **gobble's two tests bind no real port** once the option exists, as the
  owner's ruling requires (G19, G20). They assert at gobble's own seam what
  gobble passes, as its earlier ruling does (G7).
* **What real users get** (Q3, answered): the person, who alone knows where
  the browser is, says so, and no caller found could say it for them. One
  more prompt per browser sign-in is the accepted cost.
* **No question may hang or loop** a caller with no person at a terminal.
* **The least new API** (R18) **under the v1 promise** (R17). `wizard` is
  v1, so any field added there is permanent until v2. The `Prompter`
  interface does not change (C55).
* **Security unchanged:** `state`, PKCE and `redirect_uri` equality hold
  under every answer (C13 to C15, C53, C54).
* **One meaning for one name:** the same switch already exists three times
  under one name (C9, C19, C36).
* **Effort and risk:** reuse what 0074 built and verified live (R5 to R9)
  rather than add a second mechanism.

## Considered Options

The means (Q1):

* **A. A wizard option that binds no listener** (the item as named):
  `wizard.Options.NoLoopback bool`, which the wizard passes to
  `signinflow.Spec.NoLoopback` for every browser sign-in it runs. The
  sign-in completes by a paste only. Its instruction says so.
* **B. An injectable listener.** `wizard.Options.CallbackListen func(ctx
  context.Context, network, address string) (net.Listener, error)`, passed
  down through new v1 fields `auth.OAuthFlowOptions.Listen` and
  `auth.ChatGPTPlanOptions.Listen` of the same type. The browser function
  calls it in place of `net.Listen`, and the redirect URI is built from the
  returned listener's port. A test returns an ephemeral listener;
  production is unchanged. *(B2, its narrower form, as Codex's public
  `port` (U8, U9): `wizard.Options.CallbackPorts []int` and the same two
  `auth` fields, replacing `[1455, 1457]`.)*
* **C. Automatic fallback to paste.** No option. When both registered ports
  are taken and the sign-in can read a paste, the engine reruns the login
  with `NoLoopback` and shows a notice, instead of failing. With
  `NoPastePrompt` the error stays. *(It needs the ports failure to be
  recognisable across packages: an error kind `auth` wraps, from a package
  under the root `internal/`, which `auth` may import as it imports
  `internal/redact`.)*
* **D. Nothing in the wizard; callers that need no listener use
  `x/signin`.** gobble's terminal path would replace
  `wizard.ResolveCredential` with `signin.Start` and a terminal renderer of
  its own.
* **E. Nothing.** gobble's two tests keep binding the real ports, or the
  owner reverses the ruling and they become seam tests.
* **F. Export the wizard's test seams.** `wizard.Options.Logins` (a new v1
  type holding the browser and plan login functions, as `signinflow.Logins`
  does), so a caller's test replaces the login itself.

Who decides that a sign-in has no listener (Q3):

* **The caller alone**, by A's option.
* **The person**, by a question the wizard asks before a browser sign-in,
  when the caller allows it, beside A's option.
* **The wizard**, by detecting a remote session itself (such as
  `SSH_CONNECTION` through `LookupEnv`), as Codex prefers its device flow
  when headless (U10).

## Decision Outcome

The owner has answered every question this record put (Q1 to Q8 and N1
to N7, "Decided"). D1 to D10 state what was decided. The record stays
`proposed` until the owner accepts it with its PLAN,
[0083-PLAN-let-the-wizard-sign-in-without-a-callback-listener.md](0083-PLAN-let-the-wizard-sign-in-without-a-callback-listener.md).

### D1. The option (Q1, Q4, answered)

`wizard.Options` gains, after `NoPastePrompt`:

```go
// NoLoopback binds no callback listener in a browser sign-in, the ChatGPT
// plan's included, and asks the person no question about it: the redirect
// URI names OpenAI's first registered port, or for Grok a port nothing
// listens on, and only a pasted callback URL or code completes the sign-in
// (auth.OAuthFlowOptions.NoLoopback; 0074-MADR D4). It suits a caller that
// knows the browser is elsewhere, a host whose ports 1455 and 1457 must stay
// free, and a test that binds no port. A device sign-in ignores it. With
// NoPastePrompt it is an error matching llmprovider.ErrInvalidRequest,
// before any question (0083-MADR).
NoLoopback bool
```

* `browserSignIn` sets `spec.NoLoopback` to `o.NoLoopback`, or to the
  person's answer (D3), beside `spec.NoPaste` (C41). Nothing in `auth`,
  `internal/signinflow` or `x/signin` changes for it.
* It is a v1 field (R17): the wizard has no experimental tier (R19).
* **R55's call site:** gobble-cli's `provider.SignIn`, passing its
  `SignInOptions.Remote` (G1, G3), which its two terminal tests set.

### D2. What it covers (Q2, answered)

Every browser sign-in the wizard runs: OpenAI's and Grok's `browser_oauth`
and `chatgpt_plan`, first and again. The question (D3) is asked for those
sign-ins only. A device sign-in, an API key, a token paste and a vendor
import are unchanged, and never ask it. Every method stays in the menu:
each browser method the wizard runs takes a paste (C40, C24 to C28).

### D3. The question (Q3; N1 to N5, answered)

* **When it is allowed** (N1): by default, for every caller. A new field
  turns it off:

  ```go
  // NoBrowserQuestion asks the person no question about where the browser
  // is: a browser sign-in binds its callback listener, as before
  // 0083-MADR, unless NoLoopback is set. For a caller that knows the
  // browser is on this machine, or whose tests script every question.
  NoBrowserQuestion bool
  ```

* **Which caller settings ask it** (N2, N3): the question is
  asked only when the wizard itself renders the paste. It is not asked
  under `NoLoopback` (the caller has said), `NoBrowserQuestion` (the caller
  has said the other way), `NoPastePrompt` (D4) or a `Driver`. The
  full table is D7.
* **Its form** (N4): one call of the existing `Prompter.Select`
  (C56), so the stable interface is unchanged (C55). The title is "Where is
  the browser you will sign in with?". There are two choices: "On this
  machine", the default (index 0), and "On another device: I will paste
  the address back".
* **Where and how often** (N5): once per run, after the person
  chooses a browser method (C67) and before the plan's consent (C35) and
  the browser. A second browser sign-in in the same run, after a refused
  kept session (C68), reuses the answer.
* **What the answers do:**
  * "On this machine" is today's flow: the listener, the paste race and
    today's instruction (C42).
  * "On another device" is the flow `NoLoopback` gives: no listener, D5's
    words, and `OpenURL` per D6.
* **It never hangs or loops.**
  * It is one `Select`, never repeated, and never asked where no person
    reads the `Prompter` (D7).
  * `TextPrompter` takes the default on an empty line and returns an error
    once its input is exhausted (C59, C60).
  * Any `Prompter` error ends the run, as every question's does (C66).
  * A `Prompter` that blocks forever blocks on the provider menu before
    this question, so the question adds no new way to hang.

### D4. With `NoPastePrompt` (Q5, answered)

* `NoLoopback` with `NoPastePrompt`: `ConfigureLLM` and `ResolveCredential`
  return an error matching `llmprovider.ErrInvalidRequest` before any
  question. The engine would refuse it at the sign-in anyway (C33), after
  the person had answered the menus.
* `NoPastePrompt` alone: the question is not asked. With no paste, only
  the loopback can complete the sign-in, so "On another device" could not
  be honoured. This is Q5's reasoning, applied to the question. The
  listener binds as today. gobble's `--json` path is this case: it sets
  `NoPastePrompt` with its `Steps` (G4, G11).

### D5. The words shown (Q6, answered)

Under `NoLoopback`, or after the answer "On another device", the open_url
rendering (C51) keeps "Open %s in your browser". The instruction (C42)
becomes, word for word:

* code or URL: "This sign-in does not wait for the browser to come back.
  After you sign in, paste here the code the page shows or, if the browser
  is sent to a page it cannot open, the whole address from its address
  bar" (Grok's page shows a code, R9; OpenAI's browser is sent to the
  loopback address, R6)
* URL only (a first plan sign-in): "This sign-in does not wait for the
  browser to come back. After you sign in, the browser cannot open the page
  it is sent to: copy the whole address from its address bar and paste it
  here"

The paste prompts stay as they are. With the answer "On this machine", or
with no question asked and no `NoLoopback`, the instruction is today's.

### D6. Opening the browser (N6, answered)

* **After the answer "On another device":** `Options.OpenURL` is not
  called. The URL is shown with D5's words. Opening a browser on this
  machine would at best give a warning where none can open (C63), and at
  worst open one on a screen nobody watches.
* **Under the caller's `NoLoopback`:** `OpenURL` is called as today (C64).
  The caller chose the flow, and its `OpenURL` may be how it delivers the
  URL. gobble's tests paste from it (G13, G14).

### D7. Every case

| Caller sets | Question asked | Listener | Completes by | Words |
| :--- | :--- | :--- | :--- | :--- |
| nothing | yes | "On this machine": bound; "On another device": none | the loopback or a paste; a paste | today's; D5's |
| `NoLoopback` | no | none | a paste | D5's |
| `NoLoopback` and `NoPastePrompt` | no | – | error before any question (D4) | – |
| `NoBrowserQuestion` | no | bound | the loopback or a paste | today's |
| `NoPastePrompt` | no | bound | the loopback | the URL only (0057-MADR D2) |
| `Driver`, no `NoLoopback` | no (N3) | bound | the loopback, or the Driver's paste | the Driver's |
| `Driver` and `NoLoopback` | no | none | the Driver's paste | the Driver's |

`NoLoopback` with `NoBrowserQuestion` is `NoLoopback`: the caller has said
where the browser is, so there is no question to suppress.

### D8. When both ports are taken (Q7, N7, answered)

* No fallback. The error keeps its prefix "oauth: registered callback ports
  unavailable", which gobble's `signInFailed` matches (G18).
* Its advice names what the person can do:
  * for the plan, which has no device flow (C20): "free ports 1455 and 1457
    and try again";
  * for the browser sign-in: "free ports 1455 and 1457, or use device-code
    login".
* `auth`'s port function is told which sign-in it serves. The doc comments
  of `LoginBrowserOAuth` and `LoginChatGPTPlan` name `NoLoopback` as the
  caller's way past it.
* After the answer "On this machine" (N7), the same error ends the
  run. The wizard adds nothing.

### D9. The callers after a pin bump (N1, answered)

* **gobble-cli's terminal path, with no code change:** the person is asked
  the question before each run's browser sign-in. Its `wizardPrompter`
  passes the `Select` to gobble's own `Choose` (G10). Its `--json` path
  sets `NoPastePrompt` (G4), so it is not asked there.
* **prepare-commit-msg, with no code change:** the same, through its
  `TextPrompter` (P1 to P3), when it bumps from `v1.29.0` (P4).
* **Is this a v1 break?**
  * The repository's API rules see only the API: R48's `apidiff` (R17)
    would list `NoBrowserQuestion` as an addition, and no rule names a
    question the wizard asks.
  * The `Prompter` is unchanged (C55).
  * The precedent is v1.31.0, a minor release, which changed a wizard
    question under "Changed" (C69).
  * The new prompt is recorded the same way. A caller whose scripted
    `Prompter` refuses an unexpected `Select`, as this module's own fake
    does (C61), sees its tests fail until it scripts the answer or sets
    `NoBrowserQuestion`.
* **This module's tests:** the 30 tests in 13 files that run a browser
  sign-in through the fake (C61; RUN5, RUN6) each script the answer or set
  `NoBrowserQuestion`, as the PLAN's Phase 1 lists them. Under R46 (R21)
  their call syntax changes; what they assert does not.

### D10. The release and gobble afterwards (Q8, answered)

* The change ships in the next release, with 0080 and the other drafts of
  the owner's request, on the owner's word. The PLAN must finish and pass
  its gates before the tag; if it is not ready when the owner gives the
  word, it moves to the release after.
* After the release, gobble-cli:
  * bumps its pin (G8);
  * passes `Remote` to the wizard's `NoLoopback` in `provider.SignIn`;
  * sets `o.Remote = true` in its `terminalPlan`, beside the HTTP client
    and cache (G15).
* The two tests then run with no listener and **no question**: `NoLoopback`
  skips it (D7). Their scripted lines are unchanged. The paste they already
  write completes the sign-in, as the ruling "Keep them until the SDK
  offers a no-listener option" foresees.
* gobble asserts at its own `resolveCredential` seam that `NoLoopback`
  reaches the wizard (G7's form), with a planted break (`Remote` not
  passed).
* `signInFailed`'s port branch and the developers' guide's note (G24) go.
* These steps are for gobble's records to decide; this record only makes
  them possible.

### Consequences

* Good, because gobble's two terminal tests can run with no real port and
  no question, which the owner's ruling waits on.
* Good, because a person at any Go caller of the wizard, gobble's terminal
  and prepare-commit-msg alike, can say the browser is elsewhere. Today
  none can (P1 to P4, G1).
* Good, because a person whose browser is on another machine is told to
  copy the address, not to wait for a return that cannot come.
* Good, because the whole mechanism beneath the field exists and has been
  verified live (R5 to R9). The code change is a pass-through, a question,
  a check and wording.
* Good, because `state`, PKCE and `redirect_uri` equality are unchanged,
  and with no listener the host has none a local process or page can
  reach.
* Neutral, because both fields are permanent under the v1 promise (R17):
  two booleans, one named as its three siblings are.
* Neutral, because gobble's terminal tests then cover the terminal path with
  no listener, not the listener itself, which stays this module's to test
  (C46, C50).
* Neutral, because whether gobble also offers its user a terminal flag (one
  that sets `NoLoopback` and skips the question) is gobble's decision, not
  this record's.
* Neutral, because an ACP client still tells gobble through
  `_meta.gobble.remote`, to `x/signin` (G9, G5), which this record does not
  touch. The question does not reach an ACP client.
* Bad, because every run with a browser sign-in asks one more question,
  the cost the owner accepted under Q3.
* Bad, because after a pin bump every existing caller asks it with no
  code change (N1). A caller's scripted `Prompter` that refuses an
  unexpected `Select` fails until it scripts the answer or sets
  `NoBrowserQuestion` (D9). This module's own 30 tests change (RUN5).
* Bad, because on a host where another application listens on 1455, a
  same-host browser's redirect with no listener reaches that application,
  with the code and state. Without the PKCE verifier (C14) the code cannot
  be exchanged, and the address bar still holds the URL to paste
  *(inferred; not run)*.
* Bad, because a host with both ports busy still fails after the answer
  "On this machine" (D8); only the advice improves.
* *(Added 2026-11-17.)* Neutral, because the PLAN also repairs
  `TestStart_NoLoopback_0074`, by the owner's ruling of that day, relayed by
  the coordinator: "yes, fold that into the plan". Its `grok` subtest dials
  the no-listener redirect's random port, so another program's listener on
  that port can fail it, as it once did in a race run of 0080's review. It
  is the only check that Grok's no-listener sign-in binds nothing. This record
  changes that code and adds the same assertion for the wizard, so the repair
  belongs here. The assertion moves to `auth`, which sees every bind through
  an unexported hook; nothing exported changes
  ([0083-PLAN-let-the-wizard-sign-in-without-a-callback-listener.md](0083-PLAN-let-the-wizard-sign-in-without-a-callback-listener.md),
  "Amendment 2026-11-17: the no-listener test that another program's listener can fail").

### Confirmation

* **Tests, with no real port.** Each is written first and seen red at
  `b109cea`. A test that names a new field cannot compile there, so its
  red is a planted break on a scratch clone (`tools/plant.py`). Each later
  test is seen failing on its own planted break.
  * **The option reaches the login.** Through the wizard's seams (C48,
    C49), `NoLoopback` reaches the login for OpenAI's and Grok's
    `browser_oauth` and for `chatgpt_plan`, first and again, with no
    `Select` asked. Plant: the pass-through removed.
  * **The answer reaches the login.** "On another device" gives
    `NoLoopback` at the login and D5's words. "On this machine", an empty
    answer and the default each give neither. Plant: the answer ignored.
  * **The question's place.** It comes after the method choice and before
    the plan's consent, and once per run when a refused kept session leads
    to a second sign-in. Plants: asked after the consent; asked twice.
  * **No question where none may be asked.** The `Prompter`'s `Select`
    fails the test if it is called, and blocks forever if it is allowed to
    run, under each of `NoLoopback`, `NoBrowserQuestion`, `NoPastePrompt`
    and a `Driver`. Each sign-in completes within the test's deadline.
    Plant: the question asked in one of the four.
  * **No loop on an ended input.** A `TextPrompter` whose input is empty
    gives the default and the listener. One whose input is exhausted gives
    one error that ends the run, with the question rendered once. Plant:
    the question re-asked on an error.
  * **Device and value methods ask nothing.** Plant: the question asked
    for a device sign-in.
  * **D4's error.** `NoLoopback` with `NoPastePrompt` returns
    `ErrInvalidRequest` with no `Prompter` call. Plant: the check removed.
  * **D6.** `OpenURL` is not called after "On another device", and is
    called under `NoLoopback`. Plant: the call restored after the answer.
  * **D8's advice** for the plan and for the browser sign-in, with the
    prefix, in `auth`'s own package with `nettest` ports held as C46 does.
    Plant: the old text restored.
  * **That nothing binds under `NoLoopback`** is `auth`'s existing test
    (`no_loopback_0074_test.go`, C46), unchanged.
* **`make api-check`** and `tools/apidiff.py v1.31.0` list two additions,
  `wizard.Options.NoLoopback` and `wizard.Options.NoBrowserQuestion`, and
  nothing else. `Prompter` is unchanged.
* **RUN1 again** after the change: the three packages pass with 1455 and
  1457 held.
* **A live check is the owner's, and optional:** a wizard plan sign-in in a
  terminal, answering "On another device", completed by a paste from a
  phone. The path beneath the wizard was verified live (R5 to R9); the
  wizard adds the question, the pass-through and the words.
* **`tools/claims.py`** on this record and its PLAN, and the gates the PLAN
  names.

## Pros and Cons of the Options

### A. A wizard option that binds no listener (chosen, Q1)

* Good, because it is one boolean on the v1 surface, passed to fields that
  exist (C9, C19, C29, C36).
* Good, because it serves tests and callers that know: no port, true words.
* Good, because it changes no security property, and removes a local
  listener.
* Good, because its mechanism was verified live for all three methods
  (R5 to R9).
* Neutral, because on its own it needs a caller that knows, and none found
  does (the Context's "Who can tell the wizard"); Q3's question fills that.
* Bad, because a test that sets it no longer covers the listener path,
  which is this module's to test.

### B. An injectable listener

* Good, because a caller's test exercises the real listener path, on an
  ephemeral port.
* Good, because Codex has the same seam, a public port its tests set to 0
  (U8, U9).
* Bad, because it adds at least three v1 names (one wizard field, two
  `auth` fields), each permanent, whose only callers are tests: R55's call
  site would be a test.
* Bad, because for OpenAI the redirect URI must name a registered port
  (U2), so any other listener fails against the real issuer *(inferred
  from Codex's comment; not probed)*. In production it is a trap.
* Bad, because it does nothing for a person whose browser is elsewhere.
  A listener a caller supplies could also be on a non-loopback address,
  which would expose the callback to the network unless the wizard checks
  it.

### C. Automatic fallback to paste

* Good, because a person on a host with both ports busy can still finish,
  with no option.
* Good, because it adds no API.
* Bad, because gobble's tests still bind a free port: the ruling's
  condition is not met by C alone.
* Bad, because it changes what `LoginBrowserOAuth` and `LoginChatGPTPlan`
  do when ports are busy (an error today). Otherwise it needs an error kind
  carried from `auth` to the engine through a new internal package, and a
  notice the contract's steps must carry.
* Bad, because no upstream agent does it. Codex and opencode fall back to
  the second port and then fail (U5, U6, U13); kilocode's newer plugin has
  one port (U15, U16).
* Neutral, because with Q3's question a person who knows the ports are
  busy can answer "On another device", which gives C's result by choice
  (D8).

### D. Callers use `x/signin`

* Good, because it needs no change here.
* Bad, because gobble's terminal path would lose the wizard's provider and
  method menus, its endpoint and key prompts, and the listing check that
  refuses a bad credential. It would also lose the kept-session handling,
  Kilo's device key, the paste race's drain and the consent's rendering,
  and would have to rebuild each.
* Bad, because `x/signin` is experimental (R20) and may change in any
  release.
* Bad, because prepare-commit-msg, the other Go caller (P1), would need
  the same rewrite to give its person the choice.

### E. Nothing

* Good, because nothing changes here.
* Bad, because gobble's tests keep binding 127.0.0.1 and ::1 on 1455 or
  1457 on every run, and fail when both are held (RUN3, RUN4). They can
  also take 1455 from a kilocode login, which has no fallback (U16)
  *(inferred)*. The owner rejected "keep them permanently on real ports"
  (G19, G20).
* Bad, because the plan's ports error keeps advising a device flow the
  plan does not have (C5, C20).

### F. Export the wizard's test seams

* Good, because a caller's test could replace the login entirely.
* Bad, because gobble's tests would then test a fake login, not the
  module's real flow against an issuer in process, which the reviewer
  asked for.
* Bad, because it puts a new type and a field under the v1 promise for
  tests only, and lets any caller replace a sign-in's login in production.

### The person is asked (chosen, Q3)

* Good, because the person is the only one who knows where the browser is,
  and no caller found carries that fact to the wizard (Context).
* Good, because it reaches every Go caller after a pin bump (N1), with no
  code change in any of them.
* Good, because it uses `Prompter.Select`, so no implementer breaks (C55,
  C56).
* Neutral, because a caller that knows sets `NoLoopback` or
  `NoBrowserQuestion` and skips it (D7).
* Bad, because one more prompt comes in every run with a browser sign-in.
* Bad, because existing scripted tests, this module's and callers', meet a
  new `Select` (C61, D9).

### The caller alone (not chosen)

* Good, because it asks nothing.
* Bad, because no caller found has the fact to pass (P1 to P4, G1).

### Detection (not chosen)

* Good, because it asks nothing.
* Bad, because no environment variable says reliably where the person's
  browser is: a remote desktop or a forwarded port each defeats it
  *(inferred)*. Nothing in this module reads the environment unless the
  caller passes `LookupEnv`.

## Decided

The owner's answers of 2026-11-17, relayed by the coordinator, by label.
What was put is in "The owner's answers of 2026-11-17" above.

* **Q1, the means:** "One option, `wizard.Options.NoLoopback` (Recommended)" (D1).
* **Q2, which sign-ins:** "Every browser sign-in the wizard runs (Recommended)" (D2).
* **Q3, who decides:** "The wizard asks the person" (D3). This was not the
  draft's recommendation.
* **Q4, the name:** "`NoLoopback` (Recommended)" (D1).
* **Q5, with the no-paste option:** "An invalid-request error before any question is asked (Recommended)" (D4).
* **Q6, the wording:** "The draft's two instructions, word for word (Recommended)" (D5).
* **Q7, ports busy without the option:** "No automatic fallback; correct the advice per method (Recommended)" (D8).

Answered later the same day, relayed by the coordinator. What was put is
beside each label, in this record's words; each was the recommended
option.

* **N1, the default:** "On by default; a new `NoBrowserQuestion` field turns it off (Recommended)" (D3, D9).
  Put: the question is asked unless the caller turns it off. After a pin
  bump with no code change, gobble's terminal sign-in and
  prepare-commit-msg start asking it. This module's own tests, put as 44
  places (30 tests, RUN5), and any caller's scripted prompter that refuses
  an unexpected question, must script the answer or turn it off. Not
  chosen: off by default with a field to turn it on; on by default with no
  field.
* **N2, the option's meaning:** "Set: no listener and no question; unset: ask where allowed (Recommended)" (D7).
  Put: `NoLoopback` set means no listener and no question; unset, the
  question is asked where it is allowed, else the listener binds. The
  "local, do not ask" state is `NoBrowserQuestion`. Not chosen: a type of
  three values beside `NoLoopback`; no local state.
* **N3, no person at the prompt:** "Only when the wizard itself shows the paste prompt; no answer means "this machine" (Recommended)" (D3, D7).
  Put: never under `NoLoopback`, `NoBrowserQuestion`, `NoPastePrompt` or
  a `Driver`; one `Select`, never repeated; an empty answer or an ended
  input is "On this machine"; a `Prompter` error ends the run. Not chosen:
  asked under a `Driver` too; carried as a contract step a `Driver`
  answers.
* **N4, the form:** "A two-choice menu through the existing `Select` (Recommended)" (D3).
  Put: the title and the two choices D3 gives, the first the default. Not
  chosen: `Confirm`; a new optional interface.
* **N5, where and how often:** "Once per run, after the browser method is chosen and before consent (Recommended)" (D3).
  Put: a second sign-in in the run reuses the answer. Not chosen: before
  every browser sign-in; after the consent, inside the engine's steps.
* **N6, opening the browser:** "Not after the person's answer; still under the caller's option (Recommended)" (D6).
  Put: `OpenURL` is not called after "On another device", and is called
  under the caller's `NoLoopback`. Not chosen: called in every case;
  called in neither.
* **N7, both ports busy after "On this machine":** "The ruled error only (Recommended)" (D8).
  Put: Q7's error and nothing more. Not chosen: a notice suggesting the
  other answer; asking again.
* **Q8, the release:** "All four in the next release with 0080, on my word (Recommended)" (D10).
  Put to the owner once for four of this module's drafts together. Each
  record's PLAN must finish and pass its gates before the tag. One not
  ready when the owner gives the word moves to the release after.

* **Ruling, 2026-11-17: accepted.** Accepted 2026-11-17 by the owner's ruling,
  relayed by the coordinator: "Accept all five and run each plan up to its
  live step or release (Recommended)". The option's description, as put with
  it by the coordinator (the coordinator's words, not the owner's own): "Each
  is marked accepted and built in dependency order, one at a time in the SDK
  tree, each with its independent review; 0081 and 0084 stop at the probes
  only you can run; nothing is tagged without your word. They are committed
  and pushed under your standing approval as each phase passes."

## Open questions for the owner

None. Every question this record put is answered ("Decided").

## More Information

### Records this decision amends

* [0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md)
  D4 and D6: `NoLoopback` reaches the wizard too, by the caller's option or
  the person's answer. A dated note, pointing here.
* [0057-MADR-wizard-device-code-and-paste-prompt.md](0057-MADR-wizard-device-code-and-paste-prompt.md)
  D2: `NoPastePrompt` with `NoLoopback` is an error, and `NoPastePrompt`
  asks no question (D4).
* [0008-MADR-repair-oauth-loopback-and-session-wiring.md](0008-MADR-repair-oauth-loopback-and-session-wiring.md)
  D3: with no listener the race is not run, and the paste alone completes
  the sign-in.

### Related

* gobble-cli, by repository and filename:
  * `docs/decisions/0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`:
    P4 step 1c, the "second review and fixes", R2-B3, and the third review
    (G19 to G22).
  * `docs/decisions/0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`:
    P6's ruling and its 2026-11-17 note (G7, G23).
  * `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`:
    D3's note (G6).
  * `docs/guides/developing.md` (G24).
* This module:
  * [0043-MADR-openai-chatgpt-plan-sign-in.md](0043-MADR-openai-chatgpt-plan-sign-in.md)
    A3;
  * [0045-MADR-tests-never-reuse-a-freed-port.md](0045-MADR-tests-never-reuse-a-freed-port.md)
    D2;
  * [0012-MADR-conform-providers-to-reference-clients.md](0012-MADR-conform-providers-to-reference-clients.md)
    §5.3;
  * [0020-MADR-remediate-v1-debugging-pass-findings.md](0020-MADR-remediate-v1-debugging-pass-findings.md)
    F44 and F46;
  * [0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md)
    Phase 3a and Phase 7 (the live runs);
  * [0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md)
    D4: the precedent for a wizard question asked before the browser.
* The two other items of the owner's request (interfaces in place of the
  concrete session-type checks; the typed "tool schema refused" error):
  records of their own, not yet numbered. With this one they are among the
  four drafts Q8's answer puts in 0080's release (D10).
* Upstream prior art (read only, at the marker's revisions):
  * Codex binds 127.0.0.1 on 1455, asks the holder to `/cancel`, and falls
    back to 1457. It offers no paste, and sends a remote user to
    `codex login --device-auth` (U1 to U10).
  * opencode does the same, and names a device method "Codex device code
    (legacy)" (U11 to U13).
  * kilocode's newer OpenAI plugin binds `localhost:1455` alone, with a
    headless method (U15 to U17). Its docs tell the user to check the port
    (U18).
  * Grok's CLI takes a `headless` flag from its ACP client (U21).
  * None asks the person where the browser is. None lets a caller skip the
    listener and complete the browser flow by a paste. This module's
    `NoLoopback` already does the second.

### Found beyond the question asked

* With both ports busy, the plan sign-in asks its consent and fetches the
  issuer's discovery before the bind fails (C35, C22). The answer is not
  kept: it is recorded only on a session the login returns (C52; 0075-MADR
  D4).
* The ports error advises "use device-code login" for the plan, which has
  none (C5, C20). D8 corrects it.
* The ::1 listener on OpenAI's port holds the port and, since the redirect
  names 127.0.0.1, receives nothing *(inferred)*. It was kept as harmless
  (R13). Not this record's to change.
* The reviewer's "a real Codex login … cannot bind" 1455 holds of that port;
  Codex then falls back to 1457 (U5, U6). The collisions that fail a
  person are both ports held, or kilocode's single port (U16).
* The `Prompter`'s doc says twelve repositories consume this library (C55).
  The pass of 2026-11-17 found two Go callers of the wizard (gobble-cli and
  prepare-commit-msg). The rest were not enumerated.

### Not verified

* What OpenAI's issuer does with a redirect URI naming an unregistered port
  (B's premise): inferred from Codex's comment (U2), not probed.
* That kilocode's newer plugin fails while the wizard holds 1455: read from
  its code (U16), not run.
* A wizard sign-in with no listener by a live paste: not run. The same path
  through `x/signin` was (R5 to R9).
* *(Corrected 2026-11-17.)* The Go callers of the wizard. This line read
  that the records name only gobble-cli. That is wrong. The pass of
  2026-11-17 found prepare-commit-msg too (P1 to P4). It also found that
  magic-cli-remote imports this module but not the wizard (M1, M2), and
  that the Rust and TypeScript agents cannot reach it (U19 to U21, U17).
  Programs outside the repositories that pass read were not searched, and
  the `Prompter`'s doc counts twelve consumers of the library (C55).
* That magic-cli-remote's pinned `acpext` `v0.4.0` lacks `AuthenticateMeta`,
  and that ACP has no standard field for a remote sign-in: from the
  2026-11-17 pass, not re-read for this record.
* How many of gobble's own tests, besides the two, drive the real wizard
  through a browser sign-in and would meet the question (N1): not counted.
  gobble's two set `NoLoopback` (D10).
* RUN5 and RUN6 prototyped the question per sign-in, not once per run, and
  without the two new fields. They count the tests that meet it; they do
  not test the decided design, which the PLAN's tests do.

### The PLAN

[0083-PLAN-let-the-wizard-sign-in-without-a-callback-listener.md](0083-PLAN-let-the-wizard-sign-in-without-a-callback-listener.md),
`proposed`, carries out D1 to D10. The release follows on the owner's word
(Q8). gobble-cli's pin bump, its two tests, its scripted prompters and any
terminal flag are gobble-cli's own records' to plan, and so is
prepare-commit-msg's pin bump (D9, D10).

## Amendment of 2026-11-20: the state table, before any code

*(Added 2026-11-20 by the readiness pass of the four records drafted with
0080, before any of this record's code. Nothing above is rewritten or
removed; no decision, status or owner answer changes. The facts are those of
`24756a6`, the tip of `main` on 2026-11-20.)*

*(Second pass, the same day: an independent review's findings and the owner's
rulings on the first pass's open rows are applied in place, every row id
kept. "Second pass: what changed", at the end of this amendment, lists each
change.)*

### Where the record stands

* **Phase 0 has landed; nothing else has.** The records were placed in
  `a3002aa`, the PLAN's amendment folded in by `bf8e502`, and both statuses
  set by `8f133db` (the MADR `accepted`, the PLAN `in-progress`). At
  `24756a6` no Go file names `0083`, `NoBrowserQuestion` or
  `BindsNothing_0083` (`git grep`, `git log -S`). The PLAN's execution record
  holds its drafting and its approval only.
* **Two sentences above predate the acceptance.** "The record stays
  `proposed` until the owner accepts it with its PLAN" (Decision Outcome) and
  the PLAN described as "`proposed`" (More Information, "The PLAN") were true
  when written. The ruling of 2026-11-17 in "Decided" accepted the record.
* **Open rows.** The first pass left seven open rows; the owner ruled on all
  of them the same day ("The owner's rulings of 2026-11-20", below). After
  the second pass seven rows were open (ST68, ST70 to ST72, ST83 to ST85),
  and the owner ruled on those too, the same day ("Third pass", below): no
  row is open. This pass does not change the record's status.

### Build order, and the files the two earlier records change first

The coordinator's order for the three records that share the wizard's files
is 0081, then 0085, then this record, last. At `24756a6` the PLANs' scopes
name these files
([0081-PLAN-a-cancelled-sign-in-saves-nothing.md](0081-PLAN-a-cancelled-sign-in-saves-nothing.md)
"Scope" and Phase 2;
[0085-PLAN-refuse-an-empty-typed-key.md](0085-PLAN-refuse-an-empty-typed-key.md)
"Scope" and Phase 1):

| File | 0081 changes | 0085 changes | This record changes |
| :--- | :--- | :--- | :--- |
| `wizard/configure.go` | the unsaved step and `commitCredential`; `credentialStep` as the two in order; `ResolveCredential`'s check after the step removed; `ConfigureLLM`'s check after the step and its save at its end; the docs of `credentialStep`, `ConfigureLLM`, `ResolveCredential`, `Options.TokenStore` | `secretKey`; `promptAPIKey`; `resolveAPIKey`'s and `keepExistingKey`'s trims; their docs and those of `ConfigureLLM` and `ResolveCredential` | the two fields; D4's check in `ConfigureLLM` and `ResolveCredential`; the docs of `Options.OpenURL`, `Options.NoPastePrompt`, `ConfigureLLM`, `ResolveCredential` |
| `wizard/auth.go` | the docs of `resolvedCredential.save` and `saveTo` | `resolveTokenStdin` and its doc | the question and its run-held answer; `spec.NoLoopback` in `browserSignIn`; D5's instructions; D6's `OpenURL` rule |
| `wizard/prompter.go` | – | `Prompter.Secret`'s doc | `Prompter.Select`'s doc |
| new wizard tests | `cancel_save_0081_test.go`, `save_at_end_0081_test.go` | `empty_key_0085_test.go` and its companions | `browser_question_0083_test.go`, and the 30 existing tests of the PLAN's Phase 1 table |
| `llmprovider/x/signin/` | `signin.go`: one sentence in `Wait`'s doc | – | `start_browser_0074_test.go` (the amendment of 2026-11-17) |
| `README.md` | the wizard's sign-in bullet | the wizard section, and the line on Kilo and OpenCode keys | the wizard section, and the note to scripts |
| `docs/architecture.md` | "When a session is saved", the wizard's paragraph on the save | "A refused key", "An optional key" | "The wizard", beside the `NoPastePrompt` bullet |
| `CHANGELOG.md` `[Unreleased]` | Fixed and Changed | Fixed | Added and Changed |
| `docs/guides/adding-a-provider.md` | – | the key step | – |
| `wizard/kilo_verify.go` | not in its scope | not in its scope | not in its scope |

The coordinator's brief names `wizard/kilo_verify.go` among the files all
three records change. None of the three scopes at `24756a6` does. The base's
`git diff` (PLAN, Phase 1 step 1, as amended this date) settles it.

So every citation of `wizard/auth.go`, `wizard/configure.go`,
`wizard/prompter.go`, `README.md`, `docs/architecture.md` and
`CHANGELOG.md` re-grounded below holds at `24756a6` only, and moves again
before this record's build. The PLAN's entry of this date makes Phase 1's
first step a re-grounding at the base, after 0081's and 0085's code, which
this one does not replace.

### The state table

Each row is one input or state the change can meet. **Decided** rows give
the outcome a test can check, what decides it (an owner answer by its label,
as "Decided" records it, or a D-number), and the test that pins it: the
PLAN's name and phase, or an `NT` id, a test the PLAN's entry of this date
proposes. **Out of scope** rows give the reason. **Open** rows are not
decided by this record or any owner answer; they are put below.

#### The caller's settings and the question

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST1 | Nothing set; the person picks "On this machine" | decided | the login gets `NoLoopback` false; the listener binds; today's instruction (C42); `OpenURL` called | D3; N3 "Only when the wizard itself shows the paste prompt; no answer means "this machine" (Recommended)" | `TestNoLoopbackOffByDefault_0083` (1) |
| ST2 | Nothing set; an empty line | decided | the default, index 0, "On this machine": as ST1 | N3, as ST1; N4 "A two-choice menu through the existing `Select` (Recommended)" | `TestQuestionForm_0083` (1) for the index; the empty line itself: NT1 (1) |
| ST3 | Nothing set; the input ends at the question (the first read at its end) | decided | the default, as ST1 | N3, as ST1 | `TestQuestionEndedInputIsThisMachine_0083` (1) |
| ST4 | The input already exhausted at the question, or any other `Prompter` error from it | decided | one error that ends the run, the question rendered once, never asked again | D3; N3, as ST1; C66 | `TestQuestionExhaustedInputEnds_0083` (1) |
| ST5 | Nothing set; the person picks "On another device" | decided | the login gets `NoLoopback` true; D5's words, the URL-only ones for a first plan sign-in; `OpenURL` not called | D3, D5, D6; Q6 "The draft's two instructions, word for word (Recommended)"; N6 "Not after the person's answer; still under the caller's option (Recommended)" | `TestQuestionAnswerElsewhere_0083` (1); the first plan sign-in's words after the answer: NT2 (1) |
| ST6 | `NoLoopback` | decided | no question; the login gets `NoLoopback` true for OpenAI's and Grok's `browser_oauth` and for `chatgpt_plan`, first and again; D5's words; `OpenURL` called once | D1, D6, D7; Q1 "One option, `wizard.Options.NoLoopback` (Recommended)"; N2 "Set: no listener and no question; unset: ask where allowed (Recommended)"; N6, as ST5 | `TestNoLoopbackReachesTheLogin_0083`, `TestQuestionNoLoopbackStillOpens_0083` (1) |
| ST7 | `NoLoopback` with `NoPastePrompt` | decided | an error matching `llmprovider.ErrInvalidRequest` from `ConfigureLLM` and from `ResolveCredential`, with no `Prompter` call, whatever provider or method would follow | D4; Q5 "An invalid-request error before any question is asked (Recommended)" | `TestNoLoopbackWithNoPastePromptRefused_0083` (1) |
| ST8 | `NoBrowserQuestion` | decided | no question; `NoLoopback` false at the login; today's instruction; `OpenURL` called | D3, D7; N1 "On by default; a new `NoBrowserQuestion` field turns it off (Recommended)" | `TestNoBrowserQuestionKeepsToday_0083` (1) |
| ST9 | `NoLoopback` with `NoBrowserQuestion` | decided | as ST6 | D7, its closing sentence | the question's absence: `TestQuestionNotAskedWhereItMustNot_0083` (1); `NoLoopback` at the login: NT3 (1) |
| ST10 | `NoPastePrompt` alone | decided | no question; the listener binds (`NoLoopback` false at the login); the loopback alone completes | D4, D7; N3, as ST1 | `TestQuestionNotAskedWhereItMustNot_0083`, with NT12's assertion of `NoLoopback` at the login (1) |
| ST11 | A `Driver`, no `NoLoopback` | decided | no question; the listener binds (`NoLoopback` false at the login); the loopback or the `Driver`'s paste | D7; N3, as ST1 | `TestQuestionNotAskedWhereItMustNot_0083`, with NT12's assertion of `NoLoopback` at the login (1) |
| ST12 | A `Driver` with `NoLoopback` | decided | no question; `NoLoopback` true at the login; the `Driver`'s paste completes | D7 | no test yet: NT4 (1) |
| ST13 | The question's form | decided | one `Select`, title "Where is the browser you will sign in with?", choices "On this machine" and "On another device: I will paste the address back", default index 0 | D3; N4, as ST2 | `TestQuestionForm_0083` (1) |
| ST14 | Its place in a run | decided | after the method menu; before the plan's consent; before the browser | D3; N5 "Once per run, after the browser method is chosen and before consent (Recommended)" | `TestQuestionBeforeConsent_0083` (1) |
| ST15 | A second browser sign-in in one run | out of scope at `24756a6`: no run reaches it | N5 decides that a second browser sign-in in a run reuses the answer, but no run makes two. A kept session runs no sign-in (`wizard/auth.go:199` to `:211`), so the sign-in after a refused kept session (`wizard/configure.go:535` to `:545`) is the run's first, and a refused new session ends the run (`TestKept_RefusedThenNewRefused_0066`). D3's "after a refused kept session (C68)" names that first sign-in. The PLAN's step R.3 checks at the base whether 0081's or 0085's code adds a path; if it does, the row is decided by N5 and pinned by NT13 on that path | N5 | none at `24756a6`; the reachable case is ST67 |
| ST16 | A later run with the same `Options` value | decided | asked again: the answer is held by the run and stored nowhere | "The security properties a design must keep", its last bullet; PLAN Phase 1 step 6 | no test yet: NT5 (1) |
| ST17 | `ResolveCredential`, as against `ConfigureLLM` | decided | the same question and outcomes, ST1 to ST15, through both entry points | D9 (gobble's terminal path, `ResolveCredential`, is asked) | `TestNoLoopbackWithNoPastePromptRefused_0083` covers both for ST7 only; the rest: NT2 (1) |
| ST18 | The question against the plan's store reads: the saved OpenAI session, and the client record that 0082 reads in `resolveChatGPTPlan` before `browserSignIn` | decided | the question is asked right before `browserSignIn`, after both reads: a read that fails the sign-in asks no question; the warning for an undecodable record comes before the question | the owner's "Right before the browser sign-in, after the reads (Recommended)", relayed by the coordinator (2026-11-20) | NT14 (1) |

#### Each provider and sign-in mode

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST19 | OpenAI `browser_oauth` with no listener | decided | the redirect is `http://127.0.0.1:1455/auth/callback`; no bind is attempted; a paste completes | D1, D2; C10; the PLAN's amendment of 2026-11-17, by the owner's "yes, fold that into the plan" | `TestNoLoopback_BindsNothing_0083` (`auth`), `TestNoLoopbackReachesTheLogin_0083` (1) |
| ST20 | Grok `browser_oauth` with no listener | decided | the redirect names a port in 49152 to 65535; no bind is attempted; a paste completes | D1, D2; C11; the amendment, as ST19 | `TestNoLoopback_BindsNothing_0083`; the repaired `TestStart_NoLoopback_0074` (1); its 50 rounds under `-race` with 1024 listeners held (2) |
| ST21 | `chatgpt_plan`, a first sign-in, with no listener | decided | D5's URL-only words. That only a whole URL with a `client_id` completes it is the engine's existing check (ST43), not this row's test | D2, D5; C28; R14 | `TestNoLoopbackReachesTheLogin_0083` (1); after the answer: NT2 (1) |
| ST22 | `chatgpt_plan`, a sign-in again, with no listener | decided | D5's code-or-URL words | D2, D5 | `TestNoLoopbackReachesTheLogin_0083` (1) |
| ST23 | Each device sign-in: OpenAI's, Grok's, Kilo's, and Kilo's under `KiloDeviceKey` | decided | no question; `NoLoopback` set or not, the sign-in runs as without it | D1 ("A device sign-in ignores it"), D2; Q2 "Every browser sign-in the wizard runs (Recommended)" | `TestQuestionNotAskedForOtherMethods_0083` (1) for OpenAI's and Grok's; Kilo's two and `NoLoopback` with a device sign-in: NT6 (1) |
| ST24 | An API key, a pasted token (`token_stdin`), a vendor import, a kept session, a provider with no sign-in | decided | no question | D2; Q2, as ST23 | `TestQuestionNotAskedForOtherMethods_0083` (1) for the key and the import; the pasted token and the kept session: NT6 (1) |

#### The callback listener

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST25 | It can bind (ST1, ST8, ST10, ST11) | decided | today's listener and paste race | D3 ("today's flow"), D7 | the login's options at the seam: `TestNoLoopbackOffByDefault_0083` (1); the real bind is `auth`'s, unchanged (C50) |
| ST26 | 1455 taken, 1457 free | out of scope | the fallback to 1457 (C2) is unchanged; D8 changes only the text when both fail | – | – |
| ST27 | Both taken, OpenAI `browser_oauth` | decided | an error that starts "oauth: registered callback ports unavailable;" and advises "free ports 1455 and 1457, or use device-code login" | D8; Q7 "No automatic fallback; correct the advice per method (Recommended)" | `TestPortsAdvicePerMethod_0083`, with NT15's assertion of the browser's whole advice (1); `TestListenOpenAILoopback_ErrorsWhenAllBusy`, its call syntax only (1) |
| ST28 | Both taken, `chatgpt_plan` | decided | the same prefix; "free ports 1455 and 1457 and try again"; no "device-code"; after the consent and the discovery (C35, C22). That a failed sign-in records no consent is the engine's, existing (C52), and not this row's test | D8; Q7, as ST27 | `TestPortsAdvicePerMethod_0083` (1) |
| ST29 | Both taken after "On this machine", in the wizard | decided | that error ends the run; no notice, no second question, no retry | D8; N7 "The ruled error only (Recommended)" | no test yet: NT7 (1) |
| ST30 | Both families fail on both ports for a reason other than "in use" (as `isAddrInUse` reads it, `llmprovider/auth/oauth_loopback.go:565`): a sandbox's or a policy's refusal, no loopback interface | decided | the error keeps its prefix "oauth: registered callback ports unavailable;" and, in place of "free ports", points the person to the sign-in with no listener, in ST70's words | the owner's "Own advice for failures other than in-use (Recommended)", relayed by the coordinator (2026-11-20) | NT16, through the listen hook (1) |
| ST31 | 127.0.0.1 fails for a reason other than "in use" while ::1 binds | out of scope: a stated limit | the ::1 listener alone is kept (`oauth_loopback.go:509` to `:523`) and the redirect names 127.0.0.1, so only the paste completes, under today's words | the owner's ruling on ST30 makes it a stated limit; the dual-stack listener of 0008-MADR and 0012-MADR §5.3 is unchanged | – |
| ST32 | Grok's ephemeral bind fails | decided | the error keeps its prefix "oauth: listen for Grok callback:" and points the person to the sign-in with no listener, in ST70's words | the owner's ruling on ST30 | NT16 (1) |
| ST33 | Under no listener, another program on this host listens on 1455 and a browser on this host is sent there | out of scope | the Consequences accept it: the code reaches that program without the PKCE verifier, and the address bar still holds the URL to paste *(inferred; not run)*; a test would bind a real port, which the PLAN's "Ports" rule forbids | – | – |
| ST34 | The platforms: Linux, macOS, Windows | decided | ST27's and ST28's texts on each; `isAddrInUse` matches the Windows text too (`oauth_loopback.go:565`) | D8 | `TestPortsAdvicePerMethod_0083` (1), run by CI on its three platforms (`.github/workflows/ci.yml`, the `os` matrix) |

#### The code pasted by hand

The engine's and `auth`'s checks of a paste are not changed by this record
(D1: "Nothing in `auth`, `internal/signinflow` or `x/signin` changes for
it"; D5: "The paste prompts stay as they are"; "The security properties a
design must keep": "Nothing in this record changes any of it"). With no
listener they are the only way to finish, so each case is listed, with
today's outcome and the test that already holds it.

*(Second pass, 2026-11-20: the owner's ruling on a pasted address without
its scheme changes the engine's parser for that one case: D11 and ST76 to
ST86, below. The rows of this section are otherwise as written; their
reason "D1" reads as D1 narrowed by D11.)*

| Id | Case | Status | Outcome at `24756a6` (unchanged) | Reason | Held by |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST35 | A whole URL with this sign-in's state and a code | out of scope | the sign-in completes | D1, D5 | `TestEngine_PasteChecks_0074` "url with state and code"; `TestStart_NoLoopback_0074` |
| ST36 | A bare code, where the method takes one | out of scope | taken with no state check (C16) | D1, D5 | `TestEngine_PasteChecks_0074` "bare code", "padded bare code" |
| ST37 | Empty, or whitespace only, by a person (an Enter) | out of scope | asked again, not counted among the three pastes | 0057-MADR D4; D5 | `TestPaste_EmptyAskedAgain_0057`, `TestPaste_SlowEmptyAskedAgain_0059` |
| ST38 | Empty three times within 50 ms (a `Prompter` that answers at once) | decided | with no listener (`NoLoopback`, or "On another device"): the sign-in ends at once with an error, in ST71's words, and does not wait for the login's limit. With a listener: unchanged, the notice and the wait for the loopback (0059-MADR W5) | the owner's "End at once with an error; neutral notice words (Recommended)", relayed by the coordinator (2026-11-20) | NT17 (1); with a listener, `TestPaste_QuickEmptiesStop_0059`, unchanged |
| ST39 | A URL cut short | out of scope | without its state or code: asked again (ST40, ST41); with a code cut short: the exchange is refused and the sign-in ends with the issuer's error, not asked again | D1, D5 | `TestEngine_PasteChecks_0074` "another state", "no code" |
| ST40 | A URL with another sign-in's state | out of scope | "That URL is not from this sign-in"; asked again; the third bad paste ends the sign-in with `ErrInvalidRequest` | D1; security properties | `TestEngine_PasteChecks_0074` "another state"; `TestStart_BadPaste_0074` |
| ST41 | A URL with no state | out of scope | as ST40 | D1; security properties | `TestEngine_PasteChecks_0074` "no state" |
| ST42 | The whole redirect URL instead of the code | out of scope | taken where the method takes a code or a URL; required for a first plan sign-in; D5's words ask for it | D1, D5 | `TestEngine_PasteChecks_0074` "url with state and code", "url_only with a client" |
| ST43 | A bare code for a first plan sign-in | out of scope | "That is not a URL"; asked again | D1, D5; R14 | `TestEngine_PasteChecks_0074` "url_only, a bare code" |
| ST44 | Pasted twice: a second line after the first completed the sign-in | out of scope | no paste prompt is open; a `TextPrompter` gives the line to the next question, as when the paste wins today | D5 | `TestInputerContext_PasteWins_0057` |
| ST45 | A code that expired before it was pasted | out of scope | the exchange is refused and the sign-in ends with the issuer's error | D1 | not held by a test here; `auth`'s exchange error path |
| ST46 | A pasted denial (`error=` with this state) | out of scope | passed on; `auth` ends the sign-in with it | D1 | `TestEngine_PasteChecks_0074` "a denial is passed on"; `TestStart_PastedDenial_0074` |
| ST47 | The listener and a paste both arrive ("On this machine") | decided | the first wins; a paste prompt the loopback overtook is drained before any other prompt | D3 ("the listener, the paste race"); 0008-MADR D3 | `TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt`, `TestInputerContext_PasteWins_0057` (each scripting the answer, Phase 1 table) (1) |

#### Cancel and timeout, at each stage

A cancel is the run's context ending; the run's deadline is a cancel
(0081-MADR D4, Q3 "A cancel or the run's deadline saves nothing; the
listing's own limit still saves (Recommended)").

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST48 | A cancel while the question is on screen | decided | the question waits as every menu does. Once it returns, the run goes on as today after a cancel at any menu: the steps that follow run until one reads the context. For a plan sign-in the engine emits the consent step (`internal/signinflow/signinflow.go:538`) before it reads the context (`:544`), so the wizard renders the consent `Select`. The real OpenAI login falls back to its built-in endpoints when discovery fails (`llmprovider/auth/oauth_loopback.go:340` to `:346`), then binds, unless there is no listener, and calls `OpenURL` (`:204`, `:219` to `:223`) before it sees the ended context. The run then returns an error matching the context's; the login, if it is called, is given the ended context; nothing is saved | D3 ("adds no new way to hang"); 0081-MADR D9, the row "during the sign-in, before the provider issued anything (K1)"; what runs after the question is today's | NT8, on Grok's `browser_oauth`, which has no consent step (1) |
| ST49 | A cancel during the plan's consent | out of scope | unchanged: 0075-MADR D4 and 0081-MADR D9 (K1) | 0081 | 0081's and 0075's tests |
| ST50 | A cancel, or the run's deadline, while the paste is awaited with no listener, on an `InputerContext` `Prompter` | decided | an error matching `context.Canceled` or `context.DeadlineExceeded`; nothing saved; the paste read is ended by its context, with no Enter asked | 0081-MADR D9 (K1) for the error and the store; 0057-MADR D3 for the read's end. The first pass cited 0081-MADR D9 alone, which does not decide when the run ends; a `Prompter` that is not `InputerContext` is ST69 | NT9 (1) *(Annotated 2026-11-21, build pass 2: the owner ruled "Fix it: report the deadline (Recommended)"; `stepRenderer.run` no longer asks the run to cancel when the paste read ends with the run's own context, so a deadline is reported as `context.DeadlineExceeded` every time and a cancel as `context.Canceled`; NT9's deadline case asserts `DeadlineExceeded`.)* |
| ST51 | The drain notice after a cancel or the login's limit, with no listener, when a paste read is still pending | decided | it no longer says "finished": after a cancel it says the sign-in was cancelled, after the login's limit that it ended, in ST72's words, and asks for the Enter; after the loopback won it is unchanged. Whether the same words apply with a listener is ST72 | the owner's "End at once with an error; neutral notice words (Recommended)", relayed by the coordinator (2026-11-20); for a cancel, also the ruling on ST69 | NT18 (cancel), NT19 (the login's limit) (1) |
| ST52 | The login's own limit, 10 minutes (C7), with no listener | out of scope | the sign-in ends with the login's timeout; nothing was issued, nothing saved | D1 (`auth` unchanged); 0081-MADR D9 (K1) | – |
| ST53 | A cancel after the paste: the exchange, the listing, the model questions, the save | out of scope | 0081-MADR D9 and D5 apply unchanged: nothing saved; `ConfigureLLM` saves last; a session's write checks the context under the lock (0082) | 0081 | 0081's tests, which include browser and plan sign-ins (ST57) |
| ST54 | A first plan sign-in by paste, the run cancelled later | out of scope | the issued client is kept as the login returns (0075-MADR D7), as 0081-MADR D7 leaves it | 0081 Q6 "Leave both; add one sentence to `Wait`'s doc (Recommended)" | – |

#### A session with no browser here

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST55 | A remote session (SSH, a container), a person at it, answering "On another device" | decided | as ST5 | D3, D6 | `TestQuestionAnswerElsewhere_0083` (1) |
| ST56 | The same, answering "On this machine" | decided | as ST1: the listener binds on the remote host, `OpenURL` is called, and the paste race still completes, under today's words | D3 ("today's flow") | `TestNoLoopbackOffByDefault_0083` (1) |
| ST57 | No person: the input closed or ended | decided | as ST3 or ST4 | N3, as ST1 | as ST3, ST4 |
| ST58 | No browser can open here: `OpenURL` nil, or failing | out of scope | the URL is shown (C63); an `OpenURL` error does not end the sign-in, as today | D6 changes only whether it is called | – |

#### The sibling records

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST59 | 0081's new wizard tests that run a browser or plan sign-in through the fake (its PLAN, Phase 2: `TestCancelDuringListingSavesNothing_0081`'s Grok and OpenAI browser and plan cases, and any other) meet the question at this record's base | decided | each is classified by Phase 1 step 5's rule (the save is their subject: `NoBrowserQuestion`), recorded as a deviation, the phase going on only if the rule places it without doubt | the PLAN's Phase 1 step 5 and its stop condition, approved with it ("Approval (2026-11-17)") | the census, Phase 1 step 2 (1) *(Annotated 2026-11-21, build pass 2: at the base `b9eb542` the census listed these 10 of 0081's tests and the PLAN's 30; the 10 are placed by this row's rule, `NoBrowserQuestion`, as a dated deviation of the PLAN's Phase 1 build pass 1.)* |
| ST60 | 0085's new tests | decided | none runs a browser sign-in (its PLAN's Phase 1 table: keys); any that the census lists is placed as ST59 | as ST59 | the census (1) |
| ST61 | 0082's landed wizard tests that run a plan sign-in | decided | each sets `NoPastePrompt` (`client_read_0082_test.go:63`, `save_followup_0082_test.go:64`, `save_lock_0082_test.go:253`, `:283`, `save_retry_0082_test.go:432`), so none is asked | D7; N3, as ST1 | the census (1) *(Annotated 2026-11-21, build pass 2: the census at `b9eb542` is 40 tests, the PLAN's 30 and 0081's 10 (ST59); none of 0082's is listed, as this row says.)* |
| ST62 | 0085's blank-key prompt and this record's paste prompt | out of scope | different prompts: a blank paste of a code is ST37; the pasted token is 0085's and asks no question (ST24) | 0085-MADR D2; D2 | – |

#### The two flaky tests this PLAN carries

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST63 | `TestStart_NoLoopback_0074`'s `grok` subtest dials a port another program may hold | decided | the dial goes; "binds nothing" is asserted in `auth` through the unexported listen hook | the owner's "yes, fold that into the plan" (the PLAN's amendment of 2026-11-17) | `TestNoLoopback_BindsNothing_0083` and the repaired test (1); 50 rounds, 1024 listeners held (2) |
| ST64 | `TestVerifyIDToken_UnknownKidRefetchesOnce` failed twice in one 20-round `-race` run ("0 key fetches to populate the cache, want 1") | decided: that it is fixed under this PLAN | it no longer fails when an earlier test server's keys are cached under its URL | the owner's ruling of 2026-11-18, relayed by the coordinator, as 0082's records quote it: "fix this: fold it into the 0083 plan that already carries another flaky-test fix" ([0082-PLAN-refuse-to-adopt-another-sign-in-at-refresh.md](0082-PLAN-refuse-to-adopt-another-sign-in-at-refresh.md), N-5; [0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md](0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md), "A flaky test, found and not fixed (N-5)"). This record did not carry it until now | no test yet: NT10 (1); 20 rounds of `llmprovider/auth` under `-race` (2) |
| ST65 | How it is fixed | decided | on the test side only: `keysServer`, through a helper beneath it that takes the listener, removes the key-cache entry under its own JWKS URL before it returns, so an entry an earlier server on that port left is gone; no production change | the owner's "Test side: the test server clears its own cache entry (Recommended)", relayed by the coordinator (2026-11-20). Removing the entry before the helper returns, rather than at its cleanup, follows from the ruling: the stale entry is another server's | NT10 (1) |

#### The API

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST66 | The exported API | decided | exactly two additions, `wizard.Options.NoLoopback` and `wizard.Options.NoBrowserQuestion`; `Prompter` unchanged | D1, D3, D9 | `make api-check` and `tools/apidiff.py v1.31.0` (1, 2) |

#### Rows added in the second pass

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST67 | A refused kept session, then the run's one browser sign-in (C68) | decided | the question asked once, after the method menu that follows the refusal, and never when the kept session is offered | D3 ("after the person chooses a browser method"); N5, as ST14 | `TestQuestionOncePerRun_0083`, re-scoped to this row (1) |
| ST68 | One registered port fails "in use" and the other for another reason, or one family each | decided | when any registered port failed "in use", the error gives D8's "free ports" advice; the other port's reason still shows in the joined error text | the owner's "The "free the ports" advice if any port was in use (Recommended)", relayed by the coordinator (2026-11-20) | NT23 (1) |
| ST69 | A cancel, or the run's deadline, while the paste is awaited with no listener, on a `Prompter` that is not `InputerContext` (`TextPrompter`, which prepare-commit-msg uses) | decided: a stated limit | the drain shows a notice that the sign-in was cancelled and asks for the Enter (ST72's words), then waits for the pending read; the run ends when the person presses Enter, with an error matching the context's; nothing saved | the owner's "Record it as a stated limit, with honest wording (Recommended)", relayed by the coordinator (2026-11-20); 0081-MADR D9 (K1) | NT18 (1) |
| ST70 | The words of ST30's and ST32's advice | decided | after the error's prefix: "this host does not allow the callback listener: sign in with no listener and paste the address back (in the wizard, answer that the browser is on another device; in code, set NoLoopback)" | the owner's "The full instruction (Recommended)", relayed by the coordinator (2026-11-20) | NT16 (1) |
| ST71 | The words, and the kind, of ST38's error | decided | "wizard: the prompter answered the paste with nothing, three times at once; with no callback listener only a paste can finish the sign-in", matching `llmprovider.ErrInvalidRequest`, as the third bad paste does | the owner's "Specific words, matching `ErrInvalidRequest` (Recommended)", relayed by the coordinator (2026-11-20) | NT17 (1) |
| ST72 | The words of ST51's and ST69's notices, and whether they apply with a listener too | decided | after a cancel: "Sign-in cancelled; press Enter to continue"; after the login's limit: "Sign-in timed out; press Enter to continue"; with a listener and without. After a sign-in that finished (the loopback won), unchanged: "Browser sign-in finished; press Enter to continue" | the owner's ""Sign-in cancelled / timed out; press Enter to continue", in every flow (Recommended)", relayed by the coordinator (2026-11-20) | NT18, NT19, NT24 (1) |
| ST73 | A `Prompter` that answers the question with an index out of range | decided | a plain error, through `choose` (`wizard/configure.go:1103`), and no sign-in; never read as either answer | `ConfigureLLM`'s doc: "a Prompter's choice out of range [...] plain errors" (`wizard/configure.go:308` to `:309`) | NT20 (1) |
| ST74 | A caller's own descriptor that lists `browser_oauth` for a provider `auth` does not support | out of scope | the question is asked, then `auth` returns an error matching `ErrUnsupported` (`llmprovider/auth/oauth_loopback.go:303`), as for any method it does not support; the descriptor is the caller's to correct | – | – |
| ST75 | `NoLoopback` with `NoPastePrompt` on a context that has already ended | out of scope | D4's error or the context's (0059-MADR W8) ends the run before any question, and nothing is saved; which comes first is not decided, and no caller is known to depend on it | – | – |

#### A pasted address without its scheme (D11)

The case: D5 sends the person to the address bar, which shows only the host
and port until tapped (R6), so the paste can lack `http://`. At `24756a6`,
`127.0.0.1:1455/auth/callback?code=…&state=…` does not parse:
`net/url`'s `getScheme` finds no scheme when the first character is a digit,
and `parse` then refuses a first path segment holding a colon (Go 1.27.4,
`src/net/url/url.go:369` to `:391`, `:480` to `:482`). The engine takes it as
a bare code (`internal/signinflow/paste.go:53` to `:58`), `auth` sends the
whole text as the code (`llmprovider/auth/oauth_loopback.go:784`), and the
exchange is refused; for a first plan sign-in it is "That is not a URL".
Inferred from the source, not run; NT21 shows it at the base first.

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST76 | A paste that starts with the callback's host and port (those of the `redirect_uri` the authorize URL names), with this sign-in's state and a code, where the step takes a code or a URL | decided | the parse is retried with `http://` in front; the result is read as the address, its state checked, and the sign-in completes. The same in every flow on the engine: the wizard with a listener or without, and `x/signin` | the owner's "Accept it: retry the parse with `http://` in front (Recommended)", relayed by the coordinator (2026-11-20); D11 | NT21 (`internal/signinflow`), NT22 (`x/signin`, the real `auth` code) (1) |
| ST77 | The same, for a first plan sign-in (a step that takes a URL only) | decided | read as the address, as ST76; a `client_id` is still required (C32) | as ST76 | NT21 (1) |
| ST78 | The same, for a step that takes a code only (no built-in method does) | decided | read as the address, and so refused as a URL is there: "That is a URL, not a code" | as ST76: the ruling reads such a paste "as the address" | NT21 (1) |
| ST79 | The same, with another sign-in's state, or none | decided | "That URL is not from this sign-in"; asked again, counted among the three pastes | as ST76: "the state check still applies" | NT21 (1) *(Annotated 2026-11-21, Phase 3: a paste of the callback's host and port alone, with or without a trailing `/`, is ST89's, not this row's.)* |
| ST80 | The same, with whitespace before or after it | decided | trimmed first, then as ST76 | as ST76: the retry is of the parse, which reads the trimmed paste (`internal/signinflow/paste.go:49`) | NT21 (1) |
| ST81 | The prefix tried | decided | `http://` only; never `https://` or any other | as ST76, its words | NT21 (1) |
| ST82 | A bare code that does not start with the callback's host and port, one with a colon or a slash among them | decided | as today: taken as a bare code where the step takes one | as ST76: only a paste that starts with the callback's host and port is retried | NT21 (1) |
| ST83 | A prefix other than the callback's own host and port: `localhost:<port>`, `[::1]:<port>`, the callback's host on another port, a longer port that begins with the callback's (`127.0.0.1:14550`), any `host:port/`, `localhost` in mixed case | decided | only the exact host and port of the registered callback qualify: the retried address is kept only when its host and port equal the `redirect_uri`'s. `127.0.0.1:14550` and another port do not qualify. `localhost:<port>` is read as today (an opaque URL whose scheme is "localhost", its state checked); `[::1]:<port>` as today, a bare code (ST87); any other `host:port/` as today | the owner's "Only the exact host and port of the registered callback (Recommended)", relayed by the coordinator (2026-11-20) | NT21 (1) |
| ST84 | Which parser takes the retry | decided | the engine alone: it hands `auth` the address with `http://` in front; `auth`'s `parseOAuthCallback` is unchanged, and a direct caller of the lower-level sign-in with its own input does not get the retry (ST88) | the owner's "The wizard's engine only; a failed retry is refused and asked again (Recommended)", relayed by the coordinator (2026-11-20) | NT21 (the text handed on), NT22 (1) |
| ST85 | A paste that starts with the callback's host and port whose retried parse also fails (a bad escape, a control character) | decided | refused with "That is not a usable address" and asked again, counted among the three pastes | the owner's "The wizard's engine only; a failed retry is refused and asked again (Recommended)", relayed by the coordinator (2026-11-20) | NT21 (1) |
| ST87 | `[::1]:<port>/…` pasted without its scheme | out of scope: a stated limit | taken as a bare code, as today: the exchange is refused and the sign-in ends | the owner's ruling on ST83 | – |
| ST88 | A direct caller of `auth.LoginBrowserOAuth` or `auth.LoginChatGPTPlan` with its own `InputCode` gets an address pasted without its scheme | out of scope: a stated limit | `auth` takes it as a bare code, as today; the two functions' doc comments say so | the owner's ruling on ST84 and ST85 | – |
| ST86 | An address pasted with its scheme in capitals (`HTTP://127.0.0.1:…`) | out of scope | read as today: `url.Parse` lowercases the scheme (`src/net/url/url.go:454`); the callback's host is an IP, which has no case | – | – |

Tally: ~~66 rows; 40 decided, 19 out of scope, 7 open, none awaiting live
evidence~~ (the first pass). ~~After the second pass: 86 rows; 55 decided, 24
out of scope, 7 open, none awaiting live evidence.~~ After the third pass:
88 rows; 62 decided, 26 out of scope, none open, none awaiting live evidence. (The optional live check, Confirmation's last bullet, is the
owner's and decides no row.)

### The open rows, as put to the owner

*(Second pass: the owner answered all four the same day; see "The owner's
rulings of 2026-11-20". The rows open now are under "The open rows after the
second pass".)*

Four questions, gathered for one prompt; the coordinator words the options.

1. **ST18, the question against the plan's store reads.** In a plan sign-in
   the wizard reads the saved OpenAI session and the client record before
   the engine runs; either read can fail the sign-in, and an unreadable
   record shows a warning.
2. **ST30 to ST32, a bind that fails for a reason other than a port in
   use.** D8's advice ("free ports 1455 and 1457") is ruled for ports that
   are taken; today the same error is given for any failure of both binds,
   a 127.0.0.1 failure with ::1 bound keeps a listener the redirect never
   reaches, and Grok's bind failure carries no advice.
3. **ST38 and ST51, the no-listener paste wait's words and its end.** The
   "answers at once" stop and the drain notice were written for a loopback
   that may still return; with no listener, the first waits up to the
   login's 10 minutes, and the second says "Browser sign-in finished" after
   a cancel or a timeout.
4. **ST65, how the key-cache flake is fixed.** The process-wide
   `jwksCache` (`llmprovider/auth/oauth_idtoken.go:167`) is keyed by the
   JWKS URL and kept an hour; the test counts the fetches its own server
   sees. The reviewer's reading, that a later server reusing an earlier
   port finds its keys cached, has not been reproduced; NT10 is that
   reproduction.

### Found while re-grounding

* **C60's line is the wrong branch.** Its text holds at `24756a6`
  (`wizard/text_prompter.go:251`), but that line is the error for an answer
  naming no choice on the input's last line. A read after the input ended is
  `errExhausted`, "wizard: input exhausted", wrapping `io.EOF`
  (`wizard/text_prompter.go:75`, returned at `:153`); the first read at the
  end still answers the default (`:68` to `:70`). D3's sentence holds
  through `errExhausted`, and `TestQuestionExhaustedInputEnds_0083` meets
  `errExhausted`, not C60's line.
* **0082 moved a read into the plan path.** `resolveChatGPTPlan` now reads
  the client record (`wizard/auth.go:279` to `:320`) before
  `browserSignIn`, with its own notices and errors (ST18).
* **Only one test counts the key fetches.** The test issuer's keys are made
  once per process (`llmprovider/auth/oauth_testissuer_test.go:23`), so a
  stale cache entry still verifies every other test's tokens; only
  `jwksHits`, read by `TestVerifyIDToken_UnknownKidRefetchesOnce` alone
  (`oauth_idtoken_test.go:96` to `:122`), can be thrown off by it.
* **A sibling's filename.** The PLAN's "Prerequisites" names
  `0083-MADR-a-cancelled-sign-in-saves-nothing.md`; the record meant is
  [0081-MADR-a-cancelled-sign-in-saves-nothing.md](0081-MADR-a-cancelled-sign-in-saves-nothing.md).

### The evidence index re-pointed at `24756a6`

`python3 -B tools/claims.py` on this record at its marker's revisions: 141
of 141 rows pass (exit 0). With `--rev self=HEAD`: 108 of 141 (exit 1). Every
failing row's text was found on another line; none is missing. The index
above stays as it was, pinned at `b109cea`; the rows of other repositories
are pinned to their own revisions and were not re-read. The ids below end in
"now", so that `tools/claims.py` does not read them against `b109cea`.

| Id | At `b109cea` | At `24756a6` |
| :--- | :--- | :--- |
| C29 now | `internal/signinflow/signinflow.go:136` | `:146` |
| C30 now | `internal/signinflow/signinflow.go:33` | `:35` |
| C33 now | `internal/signinflow/signinflow.go:231` | `:274` |
| C34 now | `internal/signinflow/signinflow.go:432` | `:475`, the plan's branch (`:506` is the browser's) |
| C35 now | `internal/signinflow/signinflow.go:403` | `:446` |
| C36 now | `llmprovider/x/signin/signin.go:60` | `:76` |
| C37 now | `llmprovider/x/signin/signin.go:193` | `:209` |
| C40 now | `wizard/auth.go:240` | `:245` |
| C41 now | `wizard/auth.go:824` | `:900` |
| C42 now | `wizard/auth.go:763` | `:839` |
| C43 now | `wizard/configure.go:175` | `:213` |
| C44 now | `wizard/configure.go:131` | `:165` |
| C45 now | `wizard/configure.go:205` | `:243` |
| C51 now | `wizard/auth.go:873` | `:949` |
| C52 now | `internal/signinflow/signinflow.go:467` | `:525` |
| C55 now | `wizard/prompter.go:66` | `:72` |
| C56 now | `wizard/prompter.go:74` | `:80` |
| C57 now | `wizard/prompter.go:82` | `:88` |
| C58 now | `wizard/prompter.go:114` | `:120` |
| C60 now | `wizard/text_prompter.go:251` | unchanged; see "Found while re-grounding" |
| C62 now | `wizard/configure.go:171` | `:209` |
| C63 now | `wizard/configure.go:128` | `:162` |
| C64 now | `wizard/auth.go:797` | `:873` |
| C66 now | `wizard/configure.go:259` | `:297` |
| C67 now | `wizard/auth.go:218` | `:223` |
| C68 now | `wizard/configure.go:250` | `:288` |
| C69 now | `CHANGELOG.md:93` | `:224` |
| R12 now | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:804` | `:809` |
| R13 now | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:812` | `:817` |
| R17 now | `docs/guides/api-standards.md:317` | `:318` |
| R18 now | `docs/guides/api-standards.md:367` | `:368` |
| R19 now | `docs/guides/api-standards.md:77` | `:78` |
| R20 now | `llmprovider/x/signin/doc.go:19` | `:25` |
| R21 now | `docs/guides/api-standards.md:307` | `:308` |

These lines hold at `24756a6` only. Those in `wizard/` and `CHANGELOG.md`
move again with 0081's and 0085's code; the PLAN's Phase 1 step 1 re-reads
them at the base.

### The owner's rulings of 2026-11-20

Relayed by the coordinator. Each label is quoted as the owner chose it; what
it decides is in this record's words.

* **ST18:** "Right before the browser sign-in, after the reads (Recommended)". In a plan sign-in the question comes right before
  `browserSignIn`, after the reads of the session and the client record.
* **ST30 to ST32:** "Own advice for failures other than in-use (Recommended)". A bind failure other than "in use" points the
  person to the sign-in with no listener, in place of "free ports", tested
  through the listen hook. A ::1-only listener left by a failed 127.0.0.1 bind
  is a stated limit (ST31). The words are the owner's to approve (ST70).
* **ST38 and ST51:** "End at once with an error; neutral notice words (Recommended)". With no listener, a `Prompter` that answers the
  paste empty at once ends the sign-in at once with an error; the drain notice
  no longer says "finished" after a cancel or a timeout. The words are the
  owner's to approve (ST71, ST72).
* **ST69, the review's case 2:** "Record it as a stated limit, with honest wording (Recommended)". With a `Prompter` that is not
  `InputerContext`, a cancel during the paste ends the run when the person
  presses Enter; the notice says the sign-in was cancelled and to press Enter
  to continue.
* **ST76 to ST82, the review's case 1:** "Accept it: retry the parse with `http://` in front (Recommended)". A paste that starts with the
  callback's host and port is read as the address; the state check still
  applies.
* **ST65:** "Test side: the test server clears its own cache entry (Recommended)". No production change.

### The Decision Outcome, amended 2026-11-20

Nothing in D1 to D10 is rewritten; these add to them.

* **D3, in a plan sign-in:** the question's place is ST18's.
* **D5, the paste wait with no listener:** ST38, ST51 and ST69. D5's "The
  paste prompts stay as they are" holds but for these.
* **D8, a failure other than "in use":** ST30 and ST32; ST31 is a stated
  limit.
* **D11, new: a pasted address without its scheme.** The engine's paste check
  (`internal/signinflow/paste.go`) retries, with `http://` in front, a paste
  that starts with the callback's host and port, and reads the result as the
  address, with every check an address meets (ST76 to ST82). D1's "Nothing in
  `auth`, `internal/signinflow` or `x/signin` changes for it" stays true of
  `NoLoopback`'s pass-through; this parser is the exception the ruling makes.
  Every caller of the engine gets it, `x/signin`'s included, and the CHANGELOG
  says so. Which prefixes qualify, which parser takes the retry, and a retry
  that fails are ST83 to ST85, ruled in the third pass.
* **The key-cache flake:** ST65.

### The review of 2026-11-20, verified

Each finding was checked at the lines it cites, at `24756a6`, before it was
applied.

1. **Case 1, the schemeless paste:** holds (D11's section above). Ruled:
   ST76 to ST86.
2. **Case 2, a cancel with a plain `Prompter`:** holds; `drain` notifies and
   then blocks on the pending read (`wizard/auth.go:1034` to `:1038`). Ruled:
   ST69; ST50 corrected.
3. **Case 3, `Options.Driver`'s doc:** holds (`wizard/configure.go:209` to
   `:210`). Applied in the PLAN, Phase 1 step 7's list.
4. **Case 4, an out-of-range answer:** holds; the consent's raw `Select` reads
   any index but 2 as a grant or not (`wizard/driver.go:230` to `:239`), and
   `ConfigureLLM`'s doc calls a choice out of range a plain error
   (`wizard/configure.go:308` to `:309`). ST73.
5. **Case 5, the sibling's text:** holds. Applied in the PLAN, step R.3.
6. **Case 6:** holds (`llmprovider/auth/oauth_loopback.go:303`). ST74, ST75.
7. **ST50 not decided:** holds. Corrected; ST69 added.
8. **ST48's omission:** holds (`internal/signinflow/signinflow.go:538`,
   `:544`; `llmprovider/auth/oauth_loopback.go:340` to `:346`, `:204`, `:219`
   to `:223`). Corrected; NT8 names Grok's `browser_oauth`.
9. **ST15 unreachable:** holds. Out of scope at `24756a6`, with step R's
   check; the reachable case is ST67.
10. **ST27's test:** holds. NT15.
11. **ST11's test:** holds; the PLAN's row asserts no `NoLoopback` value. NT12.
12. **ST50's test:** holds; NT9 is `InputerContext` only. NT18 adds the plain
    `Prompter`.
13. **ST28, ST21:** hold. Annotated as the engine's existing behaviour.

### The open rows after the second pass

*(Third pass: the owner ruled on all seven the same day; none is open.)*

Seven, gathered for one prompt through the coordinator: ST68 (a mixed bind
failure), ST70, ST71 and ST72 (the words the rulings leave to the owner),
ST83 (other prefixes), ST84 (which parser), ST85 (a retry that fails).

### The second pass checked against the review's three questions

Only the rows this pass added or changed.

* **Missing cases.** A mixed failure (ST68), a failed retry (ST85), a longer
  port beginning with the callback's (in ST83), and capitals in a scheme
  (ST86) were found by this check and added.
* **Decided rows not decided.** Three rest on a reading of a ruling's words,
  stated in their rows: ST65 (when the entry is removed), ST78 (a code-only
  step refuses the address), ST51 (the notice after the loopback won stays).
  None rests on what the code does today alone.
* **Tests that cannot pin their rows.** NT10's plant is a change to a test
  helper, since the ruling allows no production change. NT13 is written only
  if step R finds a path (ST15). ST31 is a stated limit with no test. Every
  other changed or added decided row has a test and a named production plant
  in the PLAN that turns it red. *(Fourth pass, 2026-11-20: true since
  three plants were corrected. NT18's (ST69) and NT21's for ST82 could not
  turn their tests red, and NT9's needed its fake's `Input` to block; the
  PLAN's "Fourth pass" says how.)*

### Second pass: what changed in this amendment

* Rows changed in place, ids kept: ST10, ST11, ST15, ST18, ST21, ST27, ST28,
  ST30 to ST32, ST38, ST48, ST50, ST51, ST65.
* Rows added: ST67 to ST86.
* Dated notes added to "Where the record stands", the paste section's
  introduction and "The open rows, as put to the owner"; the tally struck
  through and restated.
* Sections added: the owner's rulings, the Decision Outcome amended, the
  review verified, the open rows after this pass, this check.

### Third pass (2026-11-20): the owner's rulings on the last seven rows

Relayed by the coordinator. Each label is quoted as the owner chose it.

* **ST68:** "The "free the ports" advice if any port was in use (Recommended)". Any registered port failing "in use" gives D8's advice;
  the other reason still shows in the joined error text.
* **ST70:** "The full instruction (Recommended)". The advice's words are the full instruction in ST70's
  row.
* **ST71:** "Specific words, matching `ErrInvalidRequest` (Recommended)". The error's words are ST71's, and it matches
  `ErrInvalidRequest`.
* **ST72:** ""Sign-in cancelled / timed out; press Enter to continue", in every flow (Recommended)". Two notices, for a cancel and for the login's limit,
  with and without a listener. A sign-in that finished keeps "Browser sign-in
  finished; press Enter to continue": no ruling changes it.
* **ST83:** "Only the exact host and port of the registered callback (Recommended)". `localhost` and `[::1]` stay as today; `[::1]` is a
  stated limit (ST87).
* **ST84 and ST85:** "The wizard's engine only; a failed retry is refused and asked again (Recommended)". The engine alone retries and hands `auth` the
  address with `http://` in front; a direct caller of the lower-level sign-in
  with its own input does not get the retry, a stated limit said in the doc
  comments (ST88). A failed retry is "That is not a usable address", asked
  again, and counted.

Rows changed in place: ST68, ST70 to ST72, ST83 to ST85. Rows added: ST87,
ST88. Every row is now decided or out of scope.

## Amendment of 2026-11-21: a cut-short paste, after the independent review

*(Added 2026-11-21, Phase 3. No decision, status or earlier row changes but ST79's test cell, annotated above.)*

The review found that a paste of only the callback's host and port, `127.0.0.1:1455` or `127.0.0.1:1455/`, is read as the
address (D11) and then refused as "That URL is not from this sign-in (its state does not match)", which misleads a person
whose address bar showed the address cut short. The owner ruled "Change it in this record".

| Id | Case | Status | Outcome | Decided by | Test (phase) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST89 | A paste that is the callback's exact host and port alone, or with a trailing `/` and nothing after it (spaces around it trimmed first) | decided | refused with its own words, "That address is cut short: paste the whole address the browser ended on." and the step's prompt; asked again and counted among the three pastes, as ST79's are; nothing reaches `auth` or the issuer. Any other paste is as before | the owner's "Change it in this record", relayed by the coordinator (2026-11-21) | `TestPasteSchemelessCallback_0083` and `TestPasteCutShortAskedAgain_0083` (`internal/signinflow`) (1) |

One case is added to ST83's: `127.0.0.1:1455x/auth/callback?code=c&state=other`, a port that begins with the callback's and
then is not a port, is read as a bare code (the owner's "Add the one test case (Recommended)"); it pins the boundary check.
