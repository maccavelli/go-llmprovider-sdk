---
status: proposed
date: 2026-11-17
decision-makers: repository owner
consulted: "the owner's ruling of 2026-11-17, relayed by the coordinator (its label quoted below), and the owner's rulings of the same day on the draft's six questions (their labels quoted in 'Decided'); an independent reviewer's findings on 0080, relayed by the coordinator; this repository at b109cea (five commits after v1.31.0, the commit before 0080's code) and 0080's staged tree as read on 2026-11-17; gobble-cli at cecc892 (branch p3-acpext-v0.5.0); the codex, opencode and kilocode checkouts, read only, at 57e6468, 4ce137b and 58072bd"
informed: gobble-cli, which signs in through wizard.ResolveCredential on its terminal path; every caller that implements wizard.Prompter, and every caller of wizard.ConfigureLLM and wizard.ResolveCredential
---

<!-- markdownlint-disable MD013 MD024 -->

# Refuse a Typed API Key That Is Empty After Trimming: the Wizard Asks Again, up to Three Times, and Then Ends the Run

## Context and Problem Statement

The wizard returns the key a person types at its prompts in `Result.APIKey`,
with `Kind` set to `CredAPIKey`. Its two key prompts, the typed key
(`promptAPIKey`) and the pasted credential of a provider's paste method
(`resolveTokenStdin`), trim what the `Prompter` returns and use what is left,
whatever it is (C1 to C4, C14 to C16). A value that is empty after trimming
therefore comes back as a successful key: no error, no question, and, for a
provider whose listing does not refuse it, no check. `ConfigureLLM` and
`ResolveCredential` share the step that does this (C33), so the gap is one.

It predates
[0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md)
and is every provider's, not Kilo's. An independent review of 0080 reported
it, and 0080 commits as it is and names it. On 0080's staged tree, as read
on 2026-11-17 and not pinned to a commit, the two Kilo checks differ on it:
`signin.VerifyKilo` sends the key as given to Kilo's profile, and 0080's own
test expects `llmprovider.ErrAuthFailure` for an empty key against its stub,
while the wizard's own check, `verifyKilo`, returns without reading the
profile for a credential whose token is empty.

### The owner's ruling of 2026-11-17

Relayed by the coordinator; its label, verbatim: **"A record of its own:
refuse an empty key at the prompt, every provider (Recommended)"**.

As it was put to the owner: an independent review confirmed a gap that
predates 0080 and affects every provider, a typed key made only of spaces is
trimmed to empty and the wizard returns it as a valid key with no check and
no error; the option chosen fixes it once for all providers that require a
key, outside 0080, as one more small record on the SDK's next release, with
0080 committing as it is and naming the gap.

**What the ruling settles** (D1): that this is a record of its own, outside
0080; its direction, that an empty key is refused at the prompt; its reach,
every provider that requires a key; and that it ships in the SDK's next
release. **What it left** was every other choice: which values count as
empty, which other routes carry the same value, what the person sees, how
often the question is asked again, the error, the release's wording, and
keyless use of three providers. Those were six questions (Q1 to Q6); the
owner answered them the same day, and "Decided" below records each answer
with its label. No question is open. The record stays `proposed` until the
owner accepts it. The tag and the GitHub Release are each the owner's word.

### The evidence, item by item

Each item says how it was established: "re-run here" (a scratch test against
a scratch copy of the tree at `b109cea`, made with `git archive`, never in
the working tree), "read", or "not re-run". The tests use the wizard's own
scripted `Prompter` (`fakePrompter`), its own `TextPrompter` over a pipe, and
in-process `RoundTripper`s and one loopback `httptest` server for the
optional-key case; no outside host and no real credential. The test files
are kept for the PLAN (Phase 1).

The reviewer's claims, each checked:

* **R1. A typed key made of blanks is returned as a key.** Reported by the
  reviewer. **Re-run here**, and widened: for each of the nine built-in
  providers that require a key, and each of six blank values (empty, spaces,
  a tab, CRLF, a non-breaking space, a mix), `ResolveCredential` returns
  `Kind` `api_key`, `APIKey` empty, no error, after one `Secret` question:
  54 of 54 cases (`TestTypedBlankKeyIsRefused_0085`, red; the table is in
  "Every route" below). `ConfigureLLM` does the same
  (`TestConfigureBlankKeyIsRefused_0085`, red).
* **R2. "`TextPrompter.Secret` refuses an empty entry but not one of
  spaces."** **Not confirmed.** `TextPrompter` trims a line before it tests
  it (C18) and in raw mode trims the entry (C21), so `Secret` sees an empty
  value for spaces, a tab and a non-breaking space alike, warns "empty value;
  please try again", and asks again (C19, C20). Re-run through a pipe and
  through the raw-mode seam: three blank entries then a key give three
  warnings and the key (`TestTextPrompterRefusesBlank_0085`, green, both
  modes); end of input after a blank line is an error, not a loop (C22). The
  package's own prompter never returned the gap to the wizard. The gap is a
  caller's own `Prompter`'s.
* **R3. A caller's `Prompter` may return empty.** Confirmed: nothing in the
  interface forbids it (C23), and the wizard's test prompter returns what it
  is scripted with (C58). gobble-cli's own `Secret` turns an empty answer
  into a cancel before the wizard sees it (G1, read), so gobble's terminal
  path is not exposed to R1 at the typed prompt.
* **R4. The environment path skips an empty value.** Confirmed, and one case
  wider: only an empty string is skipped (C8). A value of spaces is offered
  ("Use ... from the environment (...)?") and, when kept, returned as it is,
  with no trim (C9). **Re-run:** environment `"   "` gives key `"   "`;
  `"made-up-key\n"` gives the key with its newline
  (`TestBlankEnvAndSavedKey_0085`, logged).
* **R5. A saved key is offered only when not empty.** Confirmed, with the
  same case: `Options.Existing.APIKey` of spaces is offered and kept
  untrimmed (C10, C11; same test).
* **R6. The same happens for OpenCode Zen.** Confirmed, and for all nine
  (R1). It also happens for OpenCode Go, which the review did not name.
* **R7. "Kilo's free models accept anonymous calls, so an empty key is not
  always wrong."** Partly. The provider constructors for Kilo, OpenCode Zen
  and OpenCode Go accept an empty key and send the service's anonymous or
  public token (C45, C46; `providers.New` with an empty key returned no
  error for those three and "api key is required" for the other six,
  `TestBlankKeyBuildsProvider_0085`, logged), and the README says so (C48).
  But the wizard offers no way to choose "no key" for them: their
  descriptors say `RequiresAPIKey` (C36, C37, C59), the key prompt has no
  "none" entry, and the SDK's own `TextPrompter` has never let an empty
  entry through (R2). An empty key reaches the Result only through a
  caller's `Prompter`, so no documented way of choosing anonymous use
  through the wizard exists to preserve.

What else was found:

* **R8. The pasted credential has its own prompt and its own gap.** Grok's
  pasted key (`AuthTokenStdin`) takes a blank paste as an API key, `Kind`
  `api_key`, empty, no error (C14 to C16; re-run, `TestPastedBlankToken_0085`,
  logged). OpenAI's paste is checked as an access token (C17), so a blank one
  is already an error, "this is not a ChatGPT access token (a JWT) or an
  OpenAI API key (sk-...)", with no second question.
* **R9. What a blank key does under `Options.Discover`.** Re-run with a
  stubbed listing (`TestBlankKeyUnderDiscover_0085`, logged): Claude's
  listing, stubbed to refuse a blank `x-api-key`, refuses it, and the
  existing loop says "refused this key" and asks again (C27, C28), costing
  one of the three typed keys. A listing that does not authenticate accepts
  a blank key: Kilo's is public (0080's probes, K7 and K13, C57), and
  OpenCode Zen's and Go's, stubbed to answer any key, accepted it. Whether
  the real Zen and Go listings are public was not probed here. The other
  providers' real listings were not modelled; whether each refuses a blank
  key is **not verified**.
* **R10. An optional-key server.** A Chat Completions server is asked for a
  key only after its listing refuses the request without one (C29). A blank
  answer there is listed with, refused again, and costs one typed key
  (`TestCompatBlankKeyIsNotListed_0085`: the listing saw `""`, `""`, then
  `Bearer lm-key`). A server that takes no key is never asked
  (`TestCompatNoKeyNeeded_0085`, green).
* **R11. The prototype.** A fix of the size D2 to D6 describe, on a scratch
  copy only: one helper of 15 lines used by both prompts, and a trim in two
  conditions (C8, C10). All the new tests then pass, and
  `go test -count=1 ./wizard/` passes whole (no existing test changes). Each
  of eight planted breaks (the trim removed, the bound removed, a blank
  returned after the bound, the paste prompt unchecked, the environment
  untrimmed, the saved key untrimmed, the notice removed, the error's wrap
  dropped) was caught by at least one new test. A prototype is evidence of size, not
  the implementation; the PLAN writes it.

### Every route a key takes to the Result

"Blank" below means empty after `strings.TrimSpace`. "Re-run" is the
reproduction at `b109cea`.

| Route | Reached when | A blank value today | How it is known |
| :--- | :--- | :--- | :--- |
| Typed key, first ask | the provider requires a key and the method is the API key (`resolveAPIKey`, C5) | returned as `CredAPIKey` with an empty key, no error | re-run, 54 of 54; R1 |
| Typed key, after a refused one | the listing refused a typed key, an optional key, or no key (C6, C7) | the same, and listed with | re-run (optional-key case, R10); read |
| Pasted credential | Grok's pasted key; OpenAI's paste (C14 to C17) | Grok: returned as above. OpenAI: an access-token error | re-run, R8 |
| Environment | `Options.AllowEnv`, the descriptor's variable set (C8) | `""` is skipped; blanks are offered and kept untrimmed | re-run, R4 |
| Saved key | `Options.Existing` names the provider (C10) | `""` is not offered; blanks are offered and kept untrimmed | re-run, R5 |
| Kilo's device key | `Options.KiloDeviceKey` | an empty token is already an error (C24) | read |
| A caller's `Prompter` | any of the above | what the caller returns is used as above | R3 |
| The SDK's `TextPrompter` | any typed or pasted route | never returns blank: it asks again | re-run, R2 |

### Other whitespace

Re-run (`TestBlankKeyWhitespaceKinds_0085`, logged), through a typed key:

| Entered | Returned | In scope of "empty after trimming" |
| :--- | :--- | :--- |
| `"  made-up-key  "` | `"made-up-key"` | no: trimmed already |
| `"made-up-key\n"` | `"made-up-key"` | no: trimmed already |
| a non-breaking space either side | trimmed | no: `TrimSpace` strips it |
| `"made up key"` (inner spaces) | `"made up key"`, unchanged | no: not empty |
| a lone zero-width space, `"\u200b"` | `"\u200b"`, unchanged | no: Go does not count it as space (D4) |
| `"made-up\u200bkey"` | unchanged | no |
| spaces, tab, CRLF, non-breaking space, a mix | `""` | **yes** |

### Which providers require a key

The wizard knows from the descriptor: `Descriptor.RequiresAPIKey` ("whether
a credential must be collected", C34) and `IsLocal` (C35). `resolveAPIKey`
asks for a key only when `RequiresAPIKey` is set (C30); a local provider
sends none (C31); any other provider is listed with none and asked for a key
only after its listing refuses that (C29). The key prompts are therefore
reached only when a key is wanted, so the rule needs no descriptor test of
its own.

| Provider | Descriptor | Key prompt reached | A blank typed value today | `New` with a blank key |
| :--- | :--- | :--- | :--- | :--- |
| Claude | `RequiresAPIKey` (C40) | yes | returned (R1) | refused (C47) |
| Gemini | `RequiresAPIKey` (C39) | yes | returned | refused |
| Grok | `RequiresAPIKey` (C41) | yes; also the pasted-key method | returned, both | refused |
| Hugging Face | `RequiresAPIKey` (C60) | yes | returned | refused |
| Kilo | `RequiresAPIKey` (C36) | yes | returned | accepted: the anonymous token (C45) |
| OpenAI | `RequiresAPIKey` (C42) | yes; its paste is an access-token check | returned (typed); error (pasted) | refused |
| OpenCode Go | `RequiresAPIKey` (C59) | yes | returned | accepted: the public token (C46) |
| OpenCode Zen | `RequiresAPIKey` (C37) | yes | returned | accepted: the public token |
| Together | `RequiresAPIKey` (C38) | yes | returned | refused |
| Ollama | `IsLocal`, no key (C43) | never | not asked | not applicable |
| a Chat Completions server (`chatcompat.Descriptor`, C44, a caller's) | no key required | only after its listing refuses no key (R10) | listed with, refused again | not applicable |

Where an empty key is legitimate through the wizard today: Ollama and a
keyless Chat Completions server, each by choosing the provider (no key
prompt follows, and none is changed here). No provider offers "no key" as an
entry of its key prompt. Kilo, OpenCode Zen and OpenCode Go accept anonymous
use at the constructor but not through the wizard (R7).

## Decision Drivers

* A key the person never gave must not reach the caller as a success: a
  `CredAPIKey` with an empty key is one the caller then stores or passes to
  `WithAPIKey`, where six providers refuse it late and three go anonymous
  without being asked.
* One rule, in one place, for every route into the two key prompts and for
  every provider that wants a key (R1, R8).
* A scripted or non-interactive `Prompter` that answers blank for ever must
  end the run; no loop without a bound. The existing bound for typed keys is
  three (C26; 0028-MADR D-H10).
* No change to an exported name or signature (R48 of the API standards,
  C54); the SDK's own `TextPrompter` already behaves and stays as it is.
* A refusal before any request: a blank key must not reach a listing, where a
  public one would accept it.
* The error callers already handle stays what it is: this is not a provider's
  refusal, so it must not match `llmprovider.ErrAuthFailure`.
* Do not widen the decision beyond "empty after trimming" without the owner:
  what else counts as no key was Q1, and the owner kept it there (D4).
* Leave room for the later record the owner ruled for keyless use (D7): the
  refusal sits in one place that a caller-set option can later be read at,
  and nothing here removes a way to build that option.

## Considered Options

* **A.** The wizard's key prompts refuse a value that is empty after trimming
  and ask again, up to three times, then end the run with an error (the
  ruling's direction).
* **B.** `TextPrompter.Secret` alone refuses it (it already does).
* **C.** Do not refuse: let a blank key reach the provider and verify it
  there (the listing under `Options.Discover`, and 0080's Kilo check).
* **D.** Refuse after the credential is resolved, in `credentialStep`: a
  blank key from any route is an error at once, with no second question.
* **E.** Leave the code and document that a caller's `Prompter` must not
  return an empty value.

## Decision Outcome

Chosen option: **"A"**, as the ruling directs, because the two prompts are
the one place every typed or pasted key passes before it becomes a
credential, the refusal costs a person nothing the SDK's own prompter does
not already ask of them, a bounded re-ask cannot loop, and it needs no new
API. D1 is the owner's first ruling and D2 applies it; D3 to D7 are the
owner's rulings on Q1 to Q6, each with its label under "Decided"; D8 follows
from them. The record is `proposed` until the owner accepts it.

### D1. The ruling

* A record of its own, outside 0080; an empty key is refused at the prompt;
  every provider that requires a key; in the SDK's next release. 0080 commits
  as it is.

### D2. Where (the ruling, applied)

* One helper, proposed name `secretKey`, in `wizard/configure.go`, is the
  only caller of `Prompter.Secret` for a key. `promptAPIKey` (C1) calls it
  for the typed key, on the first ask and on the ask after a refused one
  (C5, C6); `resolveTokenStdin` (C14) calls it for the pasted credential.
  It trims, as both do today (C3, C15), and refuses a value that is empty.
* **The seam for a later option (D7).** The refusal is the one `value != ""`
  test in `secretKey`. Both callers already hold the descriptor and the run's
  `Options` (C1, C62), so a later record can pass `Options` to the helper, or
  have the caller choose which helper to call, and read a caller-set field
  there, for a descriptor that field names, with no other change to the two
  prompts. The helper takes the descriptor now for its notice and error
  words; it takes nothing for the option, which this record does not build.
* `TextPrompter` is unchanged (C19, C20). `Prompter`'s method set and every
  signature are unchanged.
* The check is the prompt's, not the descriptor's: the prompts are reached
  only when a key is wanted (C30), for the nine providers above and for a
  caller's provider asked for a key after a refusal. Ollama and a keyless
  server see no change.

### D3. What the person sees (Decided Q3, Q4)

* A blank value gives a warning notice, proposed words "no key was entered;
  <Label> needs one", and the prompt again, up to three asks in all
  (`maxTypedKeys`, C26) for each prompt, counted apart from refused keys.
  The third blank ends the run with a plain error, proposed words "wizard: no
  <Label> key entered after 3 tries". It matches no sentinel and is not an
  `ErrAuthFailure`. The words are the PLAN's tests', and may change before
  the PLAN is approved.
* A `Prompter`'s own error is wrapped as now ("enter API key", "enter
  credential") and returned at once.
* A scripted `Prompter` that returns blank for ever is asked three times and
  the run ends (`TestForeverBlankPrompterEnds_0085`, red at the base, which
  asks once and returns a blank key). A caller's `Prompter` that returned ""
  to say "cancelled" ends the run after two more questions, not one; the
  owner chose that bound over ending at the first blank.

### D4. What counts as empty (Decided Q1)

* Empty after `strings.TrimSpace`, the trim both prompts already apply (C3,
  C15): spaces, tabs, line breaks and the other Unicode space characters,
  the non-breaking space among them. The label says "ordinary whitespace";
  the question as put to the owner described it (see "Decided", Q1), and
  that description is exactly what this trim removes: no value that is
  trimmed today becomes a key, and nothing new is trimmed. A value of only a
  non-breaking space is blank; a lone zero-width space is not. Inner
  whitespace, other control characters and invisible characters that Go does
  not count as space are not tested; the provider's listing or request is
  left to refuse them.

### D5. The environment and a saved key (Decided Q2)

* A value that is empty after trimming is treated as an empty one is
  today: absent. It is not offered, and the typed prompt follows (C8, C10).
  A value that is not blank is offered and used as now, untrimmed.

### D6. The error and the release (Decided Q4, Q5)

* No new exported name: the error is a plain `fmt.Errorf` of the wizard, as
  "no model available ... and none entered" is (C25).
* A **Fixed** entry under `[Unreleased]` (C53), naming what a caller can
  observe (D8). It joins the one release the owner ruled for 0080 and the
  other four drafted records, on the owner's word; if this record is not
  ready when that word is given, it moves to the release after. The tag and
  the GitHub Release are each the owner's word.

### D7. Keyless use through the wizard: a later record (Decided Q6)

* The owner ruled that a later record adds an option that lets an empty key
  through for Kilo, OpenCode Zen and OpenCode Go. **This record builds
  nothing for it:** no field, no descriptor fact, no menu entry. Until that
  record exists the wizard takes no empty key for any provider, and a caller
  that wants anonymous use builds the provider without the wizard's key step
  (C48).
* What the later record has to decide, and what stays true here, is in "The
  later record for keyless use", under More Information. The seam it would
  use is D2's.

### D8. What a caller can observe, and what does not change

* Observable: a `Prompter` that returned a blank key to `ConfigureLLM` or
  `ResolveCredential` gets a question again, then an error, where it got a
  `CredAPIKey` Result with an empty key. A blank environment value and a
  blank saved key are no longer offered. OpenAI's blank paste is asked again
  where it was an error at once. A pasted Grok key that is blank is asked
  again where it was returned.
* Not changing: `TextPrompter`; the `Prompter` interface; `Result`; the
  listing refusal loop and its three keys (C27); 0080's checks; every
  descriptor; OpenAI's access-token check (C17) for a non-blank paste; and
  every run whose key is not blank.

### Consequences

* Good, because no caller receives a `CredAPIKey` with no key from the
  wizard: the nine providers and any caller's provider asked for a key.
* Good, because the check is before any request, so a public listing cannot
  accept a key nobody gave.
* Good, because it is one helper and no new API, and the SDK's own prompter
  already behaves the way the wizard now insists on.
* Neutral, because gobble-cli's own prompter already turns an empty answer
  into a cancel (G1), so its terminal path sees no change.
* Neutral, because a caller whose `Prompter` returned "" for a cancelled
  dialog is asked twice more before the run ends (D3).
* Bad, because a caller that relied on a blank key to get Kilo, OpenCode Zen
  or OpenCode Go anonymously through a custom `Prompter` loses that, with an
  error, until the later record exists. Nothing documents that route. The
  owner has ruled that such a record will follow (D7): a caller-set option
  that lets an empty key through for those three providers. Until it
  exists, a caller wanting anonymous use builds the provider without the
  wizard's key step (C48), and the wizard stays closed to an empty key.
* Neutral, because that later record reopens, under the caller's control, the
  path this record closes. This record does not foreclose it: the refusal is
  in one helper, `secretKey`, and both its callers hold the descriptor and
  `Options` (D2).
* Bad, because inner whitespace and invisible characters are still returned
  as a key (D4); under `Discover` the listing is the check.

### Confirmation

* **Tests**, in `wizard`, each against in-process stand-ins (the scripted
  `Prompter`, a `RoundTripper`, one loopback server; no outside host,
  0065-MADR), each seen red at `b109cea` and at the commit carrying 0080's
  code with `tools/redgreen.py --rev`, and each seen to fail on a planted
  break with `tools/plant.py`, on scratch clones:
  `TestTypedBlankKeyIsRefused_0085` (nine providers by six blank values: the
  second answer is used after two asks), `TestConfigureBlankKeyIsRefused_0085`,
  `TestPastedBlankKeyIsRefused_0085`, `TestForeverBlankPrompterEnds_0085`,
  `TestBlankEnvAndSavedKeyAreAbsent_0085`,
  `TestCompatBlankKeyIsNotListed_0085` and
  `TestBlankKeyNeverReachesTheListing_0085`; and, green at the base, kept as
  guards: `TestTextPrompterRefusesBlank_0085` (line and raw mode),
  `TestCompatNoKeyNeeded_0085`, `TestKeyWhitespaceKinds_0085` and
  `TestKeyPrompterErrorIsReturned_0085`.
* **No outside host in a test:**
  [0065-MADR-tests-reach-no-outside-host.md](0065-MADR-tests-reach-no-outside-host.md).
* **`make api-check`** reports no change, and `tools/apidiff.py v1.31.0`
  lists nothing for this record.
* **`tools/claims.py`** on this record and its PLAN; the gates the PLAN
  names.

## Pros and Cons of the Options

### A. The wizard's key prompts refuse and ask again, up to three times

* Good, because both prompts, the first ask and the ask after a refusal, go
  through one helper, so one rule holds for every provider and both
  entry points.
* Good, because it refuses before any request, and ends under a scripted
  `Prompter`.
* Good, because it matches how the SDK's own prompter and the typed-key
  refusal loop already behave (C19, C27).
* Neutral, because the environment and saved routes need a two-line trim
  beside it (D5).
* Bad, because a `Prompter` that signals a cancel with "" is asked three
  times, not once.

### B. `TextPrompter.Secret` alone refuses

* Good, because it is where a person types.
* Bad, because it already does (R2): the change would be nothing. The gap is
  a caller's `Prompter`'s, which this option cannot reach.

### C. Let a blank key reach the provider and verify it there

* Good, because it adds no refusal and no new words: Claude's listing
  already refuses a blank key and the loop asks again (R9).
* Bad, because it works only under `Discover`, and only where the listing
  authenticates: Kilo's listing is public (C57) and Hugging Face's needs no
  credential (C61), so a blank key is accepted there.
* Bad, because without `Discover` nothing is checked at all, and the empty
  key reaches the caller as a success.
* Bad, because for Kilo the wizard's own check skips an empty token on
  0080's tree, so verifying "there" is not what runs.

### D. Refuse after the credential is resolved, in `credentialStep`

* Good, because it is one check for every route, the environment, a saved
  key and a caller-built `Existing` included, with no change to either
  prompt.
* Good, because it is the smallest code.
* Bad, because it has no second question: a person at a custom `Prompter`
  that did not refuse is ended, where the SDK's own prompter would have asked
  again.
* Bad, because it turns a blank environment value into an error where today
  it is not an error, only an offer (C8); the person must unset the variable.
* Bad, because `credentialStep` is also where the save-on-cancel record and
  0080 are changing the wizard; a check there is the likeliest to collide.

### E. Leave it and document it

* Good, because it costs nothing now.
* Bad, because the owner ruled otherwise, and a blank key still reaches the
  caller as a success, with six providers refusing it only at `New` and three
  going anonymous unasked (R7).
* Bad, because a doc cannot stop a scripted `Prompter` from returning blank.

## Decided

The owner's rulings of 2026-11-17 on the draft's six questions, relayed by
the coordinator, in the order they were put. Each label is verbatim; what
was asked is in this record's words. The options not chosen are named so the
answer can be read against them.

* **Q1 (D4). What counts as a blank key.** Label: **"Empty after trimming
  ordinary whitespace (Recommended)"**. Asked: whether only a value that is
  empty after trimming is refused, or also one of invisible characters, or
  any key with inner whitespace or a control character. The question as put
  to the owner described the option so, in the coordinator's text, relayed
  (the description is the question's, not the owner's words): "Spaces, tabs,
  newlines and non-breaking spaces around a key are trimmed, as today; if
  nothing is left, it is refused. A key with inner spaces, or one made only
  of an invisible character such as a zero-width space, is passed on
  unchanged and left to the provider to refuse." That is `strings.TrimSpace`
  (D4). It settles what "ordinary whitespace" includes: the coordinator's
  reading of its own question, relayed, so it is not put to the owner again.
  Not chosen: refusing a value with no printable character; refusing inner
  whitespace and control characters.
* **Q2 (D5). The environment and a saved key.** Label: **"Yes: a blank value
  counts as absent (Recommended)"**. Asked: whether a blank environment value
  or a blank `Existing` key, which are offered and kept today, are treated as
  an empty one is, absent. Not chosen: the prompts only, the other routes
  left as known; absent, and a kept value also trimmed before use.
* **Q3 (D3). How often to ask again.** Label: **"Up to three asks per
  prompt, then an error (Recommended)"**. Asked: how many times a blank answer
  is followed by the question again. Not chosen: end at the first blank; ask
  without a bound.
* **Q4 (D3, D6). The error after the last blank.** Label: **"A plain wizard
  error, no new name (Recommended)"**. Asked: whether it is a plain error,
  one that wraps `llmprovider.ErrInvalidRequest`, or a new exported sentinel.
  Not chosen: the wrapped kind; `wizard.ErrNoKey`.
* **Q5 (D6). The changelog and the release.** Label: **"Fixed, in the next
  release with 0080 and the other four (Recommended)"**. As put: the entry
  joins the one release the owner ruled for the other drafts, on the owner's
  word; if this record is not ready when the word is given, it moves to the
  release after. Not chosen: a **Changed** entry; a patch release of its own.
* **Q6 (D7). Keyless use of Kilo, OpenCode Zen and OpenCode Go through the
  wizard.** Label: **"A later record adds an option that lets an empty key
  through"**. This is not the recommended option (the recommendation was no
  entry at all, a caller building the provider without a key). As put: a
  caller-set field tells the wizard that blank is acceptable for these
  providers, which reopens the path this fix closes, under the caller's
  control. It is a later record: this record builds nothing for it (D7).
  Not chosen: no such path (the recommended option); a later explicit "use
  without a key" method in the method menu.

## Open questions for the owner

None. A reading of Q1's label, drafted as a seventh question (Q7), was
settled by the question as put to the owner (see Q1 above) and was not asked.

## More Information

### Relation to 0080 and to the save-on-cancel record

* [0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md)
  commits as it is and names this gap (D1). This record changes neither its
  decisions nor its checks (D8); `verifyKilo`'s skip of an empty token stays
  true of the credential that reaches it, which after this record is never
  blank from a prompt.
* The drafted record "a cancelled sign-in saves nothing" also changes
  `wizard/configure.go`: `credentialStep`'s save and `ResolveCredential`'s
  last context check, and the doc comments of `ConfigureLLM`,
  `ResolveCredential`, `credentialStep` and `Options.TokenStore`. This record
  changes `promptAPIKey`, `resolveAPIKey`'s two conditions, `keepExistingKey`
  and `resolveTokenStdin`, and the doc comments of the same two exported
  functions. The edits do not overlap in code; the doc comments may. The
  PLAN's base is the commit that carries 0080 Phase 3's code, and it
  re-reads each line it changes at that base (the PLAN, "Relation to 0080").

### The later record for keyless use (D7)

The owner has ruled that a later record adds an option that lets an empty key
through for Kilo, OpenCode Zen and OpenCode Go, a caller-set field that tells
the wizard blank is acceptable for them. This record does not write it, and
does not bind it beyond what follows.

**What that record would have to decide:**

* **The field's name and tier.** An `Options` field is an export under the v1
  promise, which R55 asks to be justified with its call site; the other tier
  is experimental, under `llmprovider/x/` (R4), which `Options` is not.
* **Which providers it applies to, and how the wizard knows a provider
  accepts no key.** The descriptor has no such fact: `Descriptor` carries
  `RequiresAPIKey` and `IsLocal` only (C63, C34, C35), and those three
  descriptors say `RequiresAPIKey` (C36, C37, C59). That Kilo, OpenCode Zen
  and OpenCode Go take no key is code in their constructors (C45, C46), not a
  descriptor fact. The record would choose between a new descriptor fact, a
  list the caller passes, and a fixed list of provider ids in the wizard.
* **What the `Result` carries.** Today a blank key from a provider that
  requires one is `CredAPIKey` with an empty key (C12); a keyless result
  could instead be `CredNone`, which the wizard already uses for providers
  that send no key. The caller then builds the provider with no key, as
  README line 84 says (C48).
* **How it sits with this record's refusal.** The refusal stays the default
  and is read at one place (D2's seam). The option would make a blank answer
  acceptable only for a provider the field names; the bound of three asks,
  the environment and saved routes (D5) and `TextPrompter` are unaffected by
  the refusal itself.
* **How a person at the SDK's own prompter says "no key".** `TextPrompter`
  never returns a blank value (C19), so an option read only at the helper
  would be reachable from a caller's `Prompter` and not from `TextPrompter`.
  The record would need an explicit entry, such as a menu choice, a
  confirmation question, or a different prompt, for that case.
* **How it sits with 0080's checks.** On 0080's staged tree, as read and not
  pinned, `verifyKilo` already returns without reading the profile for an
  empty token and the listing is public; the record would decide whether a
  keyless Kilo credential is told as unchecked.

**What this record does so that nothing forecloses it:**

* The refusal is one test in one helper, `secretKey` (D2), and nothing else in
  the wizard refuses a blank key. A later record changes that test, or the
  caller's choice of helper, and no other.
* Both callers, `promptAPIKey` and `resolveTokenStdin`, already hold the
  descriptor and `Options` (C1, C62), so the field and the descriptor can be
  read there without a new parameter on the prompts' callers.
* No exported name, no descriptor fact and no menu entry is added or taken
  away here, so the later record starts from today's `Options` and
  descriptors.
* `Prompter.Secret`'s new doc sentence (the PLAN, step 6) says the wizard
  does not take a value that is empty once trimmed as a key, and that a
  `Prompter` signals a cancel with an error; the later record amends it for
  the providers it names.

**Until it exists,** the wizard takes no empty key for any provider, and a
caller that wants anonymous use of those three builds the provider without
the wizard's key step.

### Prior records

* [0028-MADR-heuristics-and-performance-from-the-research-pass.md](0028-MADR-heuristics-and-performance-from-the-research-pass.md)
  D-H10 and its Q4 (that record's own, not this one's): a refused typed key
  is asked again, up to three times (C55). This record asks a blank key
  again on the same bound.
* [0066-MADR-wizard-v1-24-debugging-pass-findings.md](0066-MADR-wizard-v1-24-debugging-pass-findings.md)
  V2: a remote provider with no required key is listed with none and asked
  for one when the listing refuses that (C56).
* [0020-MADR-remediate-v1-debugging-pass-findings.md](0020-MADR-remediate-v1-debugging-pass-findings.md)
  F17 and F18: a pasted key is trimmed; the trim stays.

### What the upstream agents do at an empty key (read only)

Each checkout was read, never pulled or edited, at the commit named in the
front matter. The question is only what each does at its own prompt.

* **codex** at 57e6468: its `--with-api-key` reads stdin, trims it, and exits
  with "No API key provided via stdin." when nothing is left (U3). A
  whitespace-only input is refused, an empty one too.
* **opencode** at 4ce137b: the sign-in's secret prompt validates with "not
  empty" and shows "Required" (U2). A value of spaces passes; no trim was
  found in that handler. Whether the server trims it was not checked.
* **kilocode** at 58072bd: the provider sign-in's key prompt validates with
  "present and longer than zero" and shows "Required", in two places (U1). A
  value of spaces passes; no trim was found there.

So of the three, one refuses an empty key after trimming, and two refuse
only the zero-length value. The wizard's own prompter and, after this record,
the wizard are on the first one's side.

### Not verified

* With `Options.Discover`, whether each provider's real listing refuses a
  blank key. Only Claude's was stubbed to; Kilo's public listing is from
  0080's probes (K7, K13), not run again; OpenCode Zen's and Go's listings
  were not probed against the live service.
* `signin.VerifyKilo`'s and `verifyKilo`'s handling of an empty key was read
  from 0080's staged tree on 2026-11-17, which another agent was still
  changing; it is not pinned to a commit. The PLAN re-reads it at the base.
* Whether opencode's and kilocode's servers trim a key that the prompt let
  through.
* Whether a caller outside gobble-cli returns blank from `Secret` today. No
  such caller was found; the search covered gobble-cli only.
* The words of the notice and of the error (D3) are proposals; the owner has
  not seen them. The rulings fix their kind (a warning notice, a plain
  error), not their words.
* That a later keyless option can be read at `secretKey` is a design claim
  from the code as read, not a built one: nothing was prototyped for it, and
  `Descriptor` was read as having no fact for it (C63) from its fields, not
  from a search of every use.
* Any `-race` run, any gate, any host but the session's.

### Evidence index

<!-- claims: self=b109cea gobble-cli=cecc892 codex=57e6468 opencode=4ce137b kilocode=58072bd -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The typed key's prompt | `wizard/configure.go:793` | `func promptAPIKey(p Prompter, d llmprovider.Descriptor, o Options) (string, error) {` |
| C2 | asks once | `wizard/configure.go:794` | `key, err := p.Secret(fmt.Sprintf("Enter your %s API key", d.Label))` |
| C3 | trims and tests nothing | `wizard/configure.go:798` | `key = strings.TrimSpace(key) // a paste's spaces (0020-MADR F18)` |
| C4 | returns whatever is left | `wizard/configure.go:800` | `return key, nil` |
| C5 | `resolveAPIKey` asks it as the typed key | `wizard/configure.go:772` | `key, err := promptAPIKey(p, d, o)` |
| C6 | the credential step asks it again after a refusal | `wizard/configure.go:504` | `key, keyErr := promptAPIKey(p, d, o)` |
| C7 | and makes it the credential | `wizard/configure.go:508` | `credential = staticCredential(CredAPIKey, key)` |
| C8 | The environment skips only an empty value | `wizard/configure.go:754` | `if envVal := o.lookupEnv()(d.EnvVar); envVal != "" {` |
| C9 | and returns it untrimmed | `wizard/configure.go:761` | `return envVal, keyEnvironment, nil` |
| C10 | A saved key is not offered only when empty | `wizard/configure.go:779` | `if o.Existing.Provider != d.ID \|\| o.Existing.APIKey == "" {` |
| C11 | and is returned untrimmed | `wizard/configure.go:787` | `return o.Existing.APIKey, true, nil` |
| C12 | A provider that requires a key is an API-key credential whatever the key | `wizard/auth.go:184` | `if d.RequiresAPIKey \|\| key != "" {` |
| C13 | The menu's API-key method asks the same | `wizard/auth.go:220` | `key, from, keyErr := resolveAPIKey(p, d, o)` |
| C14 | The pasted credential's own prompt | `wizard/auth.go:528` | `value, err := p.Secret(fmt.Sprintf("Paste your %s credential", d.Label))` |
| C15 | trims | `wizard/auth.go:534` | `value = strings.TrimSpace(value)` |
| C16 | and returns it as a key | `wizard/auth.go:546` | `return staticCredential(CredAPIKey, value), nil` |
| C17 | OpenAI's paste is checked as an access token | `wizard/auth.go:540` | `if err := checkAccessToken(value, time.Now()); err != nil {` |
| C18 | `TextPrompter` trims a line | `wizard/text_prompter.go:156` | `trimmed := strings.TrimSpace(line)` |
| C19 | `Secret` refuses an empty value | `wizard/text_prompter.go:402` | `if value != "" && !readFailed(err) {` |
| C20 | and warns | `wizard/text_prompter.go:410` | `p.Notify(LevelWarn, "empty value; please try again")` |
| C21 | raw mode trims the entry | `wizard/text_prompter.go:496` | `return strings.TrimSpace(string(entered)), nil` |
| C22 | the end of input is an error | `wizard/text_prompter.go:408` | `return "", fmt.Errorf("wizard: no value entered: %w", err)` |
| C23 | `Prompter.Secret`'s doc says nothing of an empty value | `wizard/prompter.go:88` | `// Secret reads a credential without echoing it in full. Implementations` |
| C24 | A Kilo device login with no token is an error | `wizard/auth.go:446` | `errors.New("wizard: Kilo login returned no token")` |
| C25 | no model entered is a plain error | `wizard/configure.go:305` | `fmt.Errorf("wizard: no model available for %s and none entered", d.Label)` |
| C26 | Typed keys are bounded at three | `wizard/configure.go:662` | `const maxTypedKeys = 3` |
| C27 | the existing refusal loop | `wizard/configure.go:456` | `for keys, workspaces := 1, 0; stepRefusal(listErr); {` |
| C28 | and its notice | `wizard/configure.go:501` | `p.Notify(LevelWarn, "%s refused this key (%s); enter another", d.Label, describe(listErr))` |
| C29 | An optional key is asked only after a refusal | `wizard/configure.go:491` | `p.Notify(LevelWarn, "%s refused the request without a key (%s); enter its key", d.Label, describe(listErr))` |
| C30 | `resolveAPIKey` branches on `RequiresAPIKey` | `wizard/configure.go:740` | `if !d.RequiresAPIKey {` |
| C31 | a local provider sends none | `wizard/configure.go:742` | `return "", keyOther, nil` |
| C32 | `notifyShape` already guards an empty key | `wizard/auth.go:655` | `if !ok \|\| key == "" \|\| s.re.MatchString(key) {` |
| C33 | `ConfigureLLM` and `ResolveCredential` share the step | `wizard/configure.go:289` | `res, cat, err := credentialStep(ctx, p, d, o)` |
| C34 | The descriptor's fact | `llmprovider/registry.go:31` | `RequiresAPIKey bool` |
| C35 | and its local flag | `llmprovider/registry.go:29` | `IsLocal bool` |
| C36 | Kilo requires a key | `llmprovider/providers/kilo/descriptor.go:23` | `RequiresAPIKey:  true,` |
| C37 | OpenCode Zen requires a key | `llmprovider/providers/opencode/descriptor.go:19` | `RequiresAPIKey:  true,` |
| C38 | Together requires a key | `llmprovider/providers/together/descriptor.go:19` | `RequiresAPIKey:  true,` |
| C39 | Gemini requires a key | `llmprovider/providers/gemini/descriptor.go:17` | `RequiresAPIKey: true,` |
| C40 | Claude requires a key | `llmprovider/providers/claude/descriptor.go:17` | `RequiresAPIKey: true,` |
| C41 | Grok requires a key | `llmprovider/providers/grok/descriptor.go:24` | `RequiresAPIKey: true,` |
| C42 | OpenAI requires a key | `llmprovider/providers/openai/descriptor.go:25` | `RequiresAPIKey: true,` |
| C43 | Ollama is local and takes none | `llmprovider/providers/ollama/descriptor.go:18` | `IsLocal:         true,` |
| C44 | a Chat Completions descriptor requires no key | `llmprovider/providers/chatcompat/descriptor.go:7` | `// sets, and no key required. It has no default base URL, no environment` |
| C45 | Kilo's constructor with no key sends the anonymous token | `llmprovider/providers/kilo/kilo.go:203` | `src = llmprovider.NewStaticToken(anonymousToken)` |
| C46 | OpenCode's sends the public token | `llmprovider/providers/opencode/opencode.go:238` | `src = llmprovider.NewStaticToken(publicToken)` |
| C47 | Claude's refuses a blank key at once | `llmprovider/providers/claude/claude.go:214` | `return nil, fmt.Errorf("%w: claude api key is required", llmprovider.ErrInvalidRequest)` |
| C48 | The README says Kilo and OpenCode accept none | `README.md:84` | `Ollama needs no key. Kilo and OpenCode accept none: they then send the` |
| C49 | The README's wizard section | `README.md:388` | `` - A sign-in saves its session to `Options.TokenStore`, once the model `` |
| C50 | The architecture's account of a refused key | `docs/architecture.md:1130` | `- **A refused key.** When the listing refuses a typed key, the wizard asks` |
| C51 | and of an optional key | `docs/architecture.md:1104` | `- **An optional key, and no default endpoint.** A remote provider with no` |
| C52 | The guide's order of the key prompts | `docs/guides/adding-a-provider.md:282` | `` otherwise `Secret`. `` |
| C53 | The changelog's open section | `CHANGELOG.md:12` | `## [Unreleased]` |
| C54 | The compatibility promise | `docs/guides/api-standards.md:317` | `` - **R48. From `v1.0.0`, every non-internal package outside `llmprovider/x/` is `` |
| C55 | 0028's refused-key rule | `docs/decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md:415` | `### D-H10. The wizard asks again when the service refuses the key` |
| C56 | 0066's optional key | `docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md:101` | `* **V2. A Chat Completions server that wants a key cannot be configured,` |
| C57 | Kilo's gateway serves free models to anonymous requests | `docs/decisions/0080-MADR-verify-a-kilo-credential-through-its-profile.md:219` | `The gateway allows unauthenticated access for free models only.` |
| C58 | The test prompter returns what it is scripted with | `wizard/fake_prompter_test.go:162` | `func (f *fakePrompter) Secret(prompt string) (string, error) {` |
| G1 | gobble's `Secret` turns an empty answer into a cancel | `gobble-cli:internal/cli/prompt.go:71` | `if err == nil && s == "" {` |
| C59 | OpenCode Go requires a key | `llmprovider/providers/opencode/descriptor.go:34` | `RequiresAPIKey:  true,` |
| C60 | Hugging Face requires a key | `llmprovider/providers/huggingface/descriptor.go:19` | `RequiresAPIKey:  true,` |
| C61 | `ResolveCredential`'s doc: a listing may need no credential | `wizard/configure.go:342` | `// that needs no credential, such as Hugging Face's, says nothing of the key.` |
| C62 | `resolveTokenStdin` holds the run's `Options` | `wizard/auth.go:524` | `o Options,` |
| C63 | `Descriptor` has no fact that a provider accepts no key | `llmprovider/registry.go:16` | `type Descriptor struct {` |
| U1 | kilocode's key prompt refuses only a zero-length value | `kilocode:packages/opencode/src/cli/cmd/providers.ts:176` | `validate: (x) => (x && x.length > 0 ? undefined : "Required"),` |
| U2 | opencode's refuses only an empty value | `opencode:packages/cli/src/commands/handlers/auth/form.ts:65` | `validate: (value) => (!value ? "Required" : undefined)` |
| U3 | codex trims stdin and refuses an empty result | `codex:codex-rs/cli/src/login.rs:363` | `if secret.is_empty() {` |
