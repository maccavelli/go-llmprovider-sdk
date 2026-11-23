---
status: accepted
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

*(Added 2026-11-20.)* Two decisions join this outcome, each recorded in full
in "Amendment of 2026-11-20, second pass": **D9**, a provider's constructor
treats a whitespace-only key as no key (the owner's ruling on ST48, "Pull it
into 0085"), which widens this record to the public constructors; and
**D10**, a repeat ask checks the run's context first (the owner's ruling on
ST53). D8's "Not changing" list stands for the wizard; D9 names what it
changes outside it.

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

* **Ruling, 2026-11-17: accepted.** Accepted 2026-11-17 by the owner's ruling,
  relayed by the coordinator: "Accept all five and run each plan up to its
  live step or release (Recommended)". The option's description, as put with
  it by the coordinator (the coordinator's words, not the owner's own): "Each
  is marked accepted and built in dependency order, one at a time in the SDK
  tree, each with its independent review; 0081 and 0084 stop at the probes
  only you can run; nothing is tagged without your word. They are committed
  and pushed under your standing approval as each phase passes."

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

## Amendment of 2026-11-20: the state table, before any code

Added after the record was accepted and before any of its code. Nothing
above is rewritten or removed; the decision, the status and every owner
answer stand. The facts are those of `24756a6`, the tip of `main` on
2026-11-20; the record above was written at `b109cea`.

### Where the record stands at `24756a6`

* **No code is built.** The PLAN's execution record holds the drafting, the
  amendment of the drafts and the approval, and no phase. No commit after
  the records' own (`a3002aa`, `8f133db`) names 0085, no `wizard/*0085*`
  file exists, and no function `secretKey` exists.
* **The lines this record changes have not changed.** The bodies of
  `promptAPIKey`, `resolveAPIKey`, `keepExistingKey`, `resolveTokenStdin`,
  `warnForeignKey`, `notifyShape`, `checkAccessToken`, `maxTypedKeys`,
  `TextPrompter.Secret`, `TextPrompter.readLine` and the test's
  `fakePrompter.Secret` are identical at `b109cea` and `24756a6` (compared
  function by function, read only). 0080's code (`00757ed`) and 0082's
  (`8051b00`, `04433ae`, `5f2c2ab`, `1545172`, `fd986c4`) moved them down
  their files; "The evidence index re-pointed at `24756a6`", below, gives
  each line.
* **Build order.** The coordinator's order of 2026-11-20 builds the three
  records that share the wizard's files as 0081, then this record, then
  0083; 0084 is built apart. This record's code therefore starts on the
  commit that carries 0081's code (the PLAN's "Prerequisites", annotated
  there), and 0081's code waits on the owner's live run (0081-PLAN Phase
  1b).
* **Two "Not verified" items are now read at a commit.** `verifyKilo`
  returns without reading Kilo's profile for an empty token (C64now), and
  its comment says this record's gap is "found, and not changed here"
  (C65now), which becomes untrue when this record's code lands (ST62).
  `signin.VerifyKilo` has no empty-key test of its own (C66now). gobble-cli's
  `Secret` still turns an exactly empty answer, compared untrimmed, into a
  cancel at that repository's head `b1994a0` (G2now; read only).

### The boundary with a consumer's own input layer

A consumer CLI has a known first-run defect: the line feed left over from a
CR+LF answer to a menu is read by its next prompt as an empty API key. That
consumer fixes its input layer under its own record, and this record is not
widened to cover it. **This record decides only what the SDK does with the
value a `Prompter`'s `Secret` returns.** Everything before that value, how
the consumer reads its terminal, which bytes it leaves between two prompts,
and whether it turns an empty answer into a cancel, is the `Prompter`
implementation's. Rows ST23 to ST25 state each side. The SDK's own
`TextPrompter` is unchanged (D2): its line mode reads to the line feed and
trims (C18); its raw mode drops a line feed that arrives with the carriage
return and leaves a later one to the next prompt, by design (C70).

### The state table

Terms. **Blank**: empty after `strings.TrimSpace` (D4). **Asked again**: one
warning notice, then the same prompt again. **The no-key error**: the plain
error D3 names, returned after the third blank at one prompt, with no
Result. The **typed prompt** is `promptAPIKey`'s and the **paste prompt**
`resolveTokenStdin`'s; both call D2's one helper. A test marked
"(proposed)" is added by the PLAN's entry of 2026-11-20, with its plant; a
row's phase is the PLAN's Phase 1 unless it says otherwise. "Today" is the
code at `24756a6`.

**A. The value the typed prompt receives** (every provider that requires a
key; `ResolveCredential` and `ConfigureLLM` alike, ST26)

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST1 | `""` | decided | asked again; a key on the second ask is the Result's `APIKey`, `Kind` `api_key`, after two `Secret` calls and one warning notice | D1 (the ruling, "A record of its own: refuse an empty key at the prompt, every provider (Recommended)"), D3, D4 | `TestTypedBlankKeyIsRefused_0085` |
| ST2 | whitespace only: spaces, a tab, a non-breaking space, a mix | decided | as ST1 | D4 (Q1, "Empty after trimming ordinary whitespace (Recommended)") | `TestTypedBlankKeyIsRefused_0085` (four of its six values) |
| ST3 | a lone CR, `"\r"`, or a lone LF, `"\n"` | decided | as ST1: `TrimSpace` removes either | D4 (Q1) | `TestTypedBlankKeyIsRefused_0085`, two values added (proposed) |
| ST4 | CR+LF, `"\r\n"` | decided | as ST1 | D4 (Q1) | `TestTypedBlankKeyIsRefused_0085` (its CRLF value) |
| ST5 | a key padded with spaces or a non-breaking space, or with a trailing `"\n"` or `"\r\n"` | decided | trimmed and used: one ask, no notice, `APIKey` the key alone | D4 ("no value that is trimmed today becomes a key, and nothing new is trimmed"); 0020-MADR F18 | `TestKeyWhitespaceKinds_0085`; a trailing `"\r\n"` added (proposed) |
| ST6 | a quoted key, `"\"made-up-key\""` or `"'made-up-key'"`, and a value of two quote characters alone | decided | returned unchanged, quotes kept; neither is blank, so neither is asked again | D4 ("nothing new is trimmed"); Q1's text, "if nothing is left, it is refused"; PLAN "Out of scope" (no check of a key's shape) | `TestKeyWhitespaceKinds_0085`, three values added (proposed) |
| ST7 | an interior space, line feed, NUL or ESC (`"made up key"`, `"made-up\nkey"`, `"made-up\x00key"`, `"made-up\x1bkey"`) | decided | returned unchanged; under `Discover` the listing is its only check, without it the Result carries it | D4 ("Inner whitespace, other control characters [...] are not tested; the provider's listing or request is left to refuse them"); Consequences, last bullet | `TestKeyWhitespaceKinds_0085` (inner spaces); three control-character values added (proposed) |
| ST8 | only characters Go does not count as space, such as a lone zero-width space | decided | returned unchanged | D4 ("a lone zero-width space is not" blank) | `TestKeyWhitespaceKinds_0085` |
| ST9 | a very long key (1 MiB of letters) | decided | returned unchanged; no length bound. A bound would be a check of the key's shape, which is out of scope | D4; PLAN "Out of scope" | `TestKeyWhitespaceKinds_0085`, one value added (proposed) |
| ST10 | not valid UTF-8: `"\xff"`, or a lone `"\xa0"` byte (a non-breaking space in a legacy single-byte code page) | decided | returned unchanged: `TrimSpace` removes no invalid byte, so neither is blank. Validating the encoding is out of scope | D4 (the rule is `TrimSpace`'s; "nothing new is trimmed"); PLAN "Out of scope" | `TestKeyWhitespaceKinds_0085`, two values added (proposed) |

**B. The bound, the notice and the error**

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST11 | three blanks in a row at one prompt | decided | the no-key error after three `Secret` calls; no Result; no listing request | D3 (Q3, "Up to three asks per prompt, then an error (Recommended)") | `TestForeverBlankPrompterEnds_0085` |
| ST12 | whether the third blank also gives the warning notice before the error | open | today's draft (PLAN step 4) gives two notices and none with the error; D3 says "a blank value gives a warning notice [...] and the prompt again", which the third cannot have | none | none until answered |
| ST13 | the words of the notice and of the error | open | D3's proposals, "no key was entered; <Label> needs one" and "wizard: no <Label> key entered after 3 tries"; D3 says they "may change before the PLAN is approved" and "Not verified" says the owner has not seen them; the approval of 2026-11-17 does not name them. The same words would follow a blank paste prompt, "Paste your <Label> credential" | the kind only: D3, D6 (Q4) | `TestTypedBlankKeyIsRefused_0085` and `TestForeverBlankPrompterEnds_0085` once answered |
| ST14 | the no-key error's type, and `errors.Is` | decided | a plain `fmt.Errorf` value; `errors.Is` is false for `llmprovider.ErrAuthFailure`, `llmprovider.ErrInvalidRequest`, `context.Canceled` and `io.EOF`; no exported name is added; a caller can tell it apart only by its text | D3 ("It matches no sentinel and is not an `ErrAuthFailure`"), D6 (Q4, "A plain wizard error, no new name (Recommended)") | `TestForeverBlankPrompterEnds_0085`, four `errors.Is` assertions added (proposed) |
| ST15 | blanks across the refusal loop: a typed key the listing refuses is asked for by a new call of the typed prompt; does its blank count restart? | open | the PLAN's helper restarts it at each call: up to three asks for each key, so at most nine `Secret` calls for the three keys of 0028-MADR D-H10. D3's "for each prompt, counted apart from refused keys" and Q3's "per prompt" can be read as each call or as each of the two prompts in a run | none | none until answered |

**C. The `Prompter`'s other answers, and runs with no person**

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST16 | `Secret` returns an error on the first ask | decided | returned at once, wrapped "enter API key: " (typed) or "enter credential: " (paste); `errors.Is` matches the `Prompter`'s error; no second ask | D3 ("A `Prompter`'s own error is wrapped as now [...] and returned at once") | `TestKeyPrompterErrorIsReturned_0085` (typed and paste) |
| ST17 | `Secret` returns an error after one blank | decided | one notice, then the error returned at once, wrapped as ST16; no third ask | D3 | `TestKeyPrompterErrorIsReturned_0085`, a blank-then-error case added (proposed) |
| ST18 | `Secret` returns a value and an error together | decided | the error is returned, wrapped as ST16; the value is not used | D3 | `TestKeyPrompterErrorIsReturned_0085`, a value-with-error case added (proposed) |
| ST19 | a caller's `Prompter` that says "cancelled" with `""` | decided | asked twice more, then the no-key error, which does not match `context.Canceled` | D3 ("the owner chose that bound over ending at the first blank"); Consequences, Neutral | `TestForeverBlankPrompterEnds_0085` |
| ST20 | non-interactive with the SDK's `TextPrompter`: a blank line, then end of input | decided | `TextPrompter` warns once and returns "wizard: no value entered: EOF" (C22); the wizard wraps it as ST16 and returns it at once; `errors.Is(err, io.EOF)` is true; the wizard's own bound is never reached | D2 (`TextPrompter` unchanged), D3 | `TestTextPrompterRefusesBlank_0085`, an end-of-input case added (proposed) |
| ST21 | interactive `TextPrompter`, blank entries, line or raw mode | decided | `TextPrompter` asks again itself ("empty value; please try again", C20), without a bound of its own, and never returns a blank; the wizard sees the key | D2; R2 | `TestTextPrompterRefusesBlank_0085` |
| ST22 | Ctrl-C at `TextPrompter`'s raw-mode prompt | decided | "wizard: cancelled: context canceled", wrapped as ST16, returned at once; `errors.Is(err, context.Canceled)` is true | D2, D3 | `TestKeyPrompterErrorIsReturned_0085` (the same `%w` wrap, with a sentinel) |

**D. The consumer boundary**

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST23 | a consumer's `Secret` returns the line feed left from a CR+LF menu answer, `"\n"`, or `""` | decided (SDK side) | asked again, as ST1; if the consumer's next read returns the person's key, the run goes on with it | D3, D4 | `TestTypedBlankKeyIsRefused_0085` with `"\n"` (ST3, proposed) |
| ST24 | a consumer's `Secret` turns an empty answer into a cancel error before the SDK sees it (G1) | decided (SDK side) | the error is returned at once, as ST16 | D3 | `TestKeyPrompterErrorIsReturned_0085` |
| ST25 | why the consumer left the line feed, and how its terminal layer reads | out of scope | the consumer's own record; the SDK's contract starts at the value `Secret` returns | the ruling's reach ("at the prompt"); D2 | none |

**E. Providers, credential kinds and entry points**

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST26 | the two entry points, `ResolveCredential` and `ConfigureLLM` | decided | the same outcome through either: they share the credential step (C33) | D2 | `TestTypedBlankKeyIsRefused_0085`; `TestConfigureBlankKeyIsRefused_0085` |
| ST27 | the API key as a provider's only method (`signIn`'s first branch, C12) and as a menu choice (C13; Grok, Kilo, OpenAI) | decided | the same refusal on both branches | D2 ("every route into the two key prompts") | `TestTypedBlankKeyIsRefused_0085`, which reaches both; Phase 1 names in it which providers take the menu (proposed) |
| ST28 | the typed prompt asked again after a listing refusal (C6) | decided | the same helper and the same refusal | D2 | `TestCompatBlankKeyIsNotListed_0085` (the same call site) |
| ST29 | each of the nine providers that require a key, Kilo, OpenCode Zen and OpenCode Go among them | decided | refused alike; no code names those three; nothing is built for the later keyless option | D2; D7 (Q6, "A later record adds an option that lets an empty key through") | `TestTypedBlankKeyIsRefused_0085` (nine by six) |
| ST30 | a provider whose key is optional (a Chat Completions server, one found by `LocalServers` that wants a key), asked after its listing refuses no key | decided | asked again; the server never sees a blank `Authorization` | D2 | `TestCompatBlankKeyIsNotListed_0085` |
| ST31 | a provider that takes no key: Ollama, a keyless Chat Completions server | decided | no key prompt, unchanged | D2 ("Ollama and a keyless server see no change") | `TestCompatNoKeyNeeded_0085`; the existing `TestOllama_RefusedNoKeyAsked_0066` |
| ST32 | the paste prompt, Grok, blank | decided | asked again, the same bound and error as the typed prompt | D2, D8 | `TestPastedBlankKeyIsRefused_0085` |
| ST33 | the paste prompt, OpenAI, blank | decided | asked again where today it is the access-token error at once; a non-blank paste still meets the foreign-key and access-token checks unchanged (C17) | D8 | `TestPastedBlankKeyIsRefused_0085`, an OpenAI case added (proposed) |
| ST34 | a browser sign-in's pasted URL or code (`Input`, `InputContext` or `Options.Driver`) | out of scope | unchanged: not a key, and not read by `Secret` | D2 (the two prompts only) | none |
| ST35 | the device sign-ins, the ChatGPT plan, the vendor CLI import, a kept session | out of scope | unchanged: no typed key; each checks its own session (an empty kept session is not offered; an empty Kilo token is an error, C24) | D2 | none |
| ST36 | Kilo's device token as a key (`Options.KiloDeviceKey`) | out of scope | unchanged: an empty token is already an error (C24); a token of spaces is the service's, not typed | "Every route", its Kilo row | none |

**F. The environment and `Options.Existing`, the routes this record includes**

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST37 | the environment value blank, under `AllowEnv` | decided | absent: no `Confirm`; the typed prompt follows | D5 (Q2, "Yes: a blank value counts as absent (Recommended)") | `TestBlankEnvAndSavedKeyAreAbsent_0085` |
| ST38 | the environment value `""` | decided | absent, as today (C8) | D5 | `TestBlankEnvAndSavedKeyAreAbsent_0085`, `""` among its values (proposed if absent) |
| ST39 | the environment value not blank but padded, such as `"made-up-key\n"` | decided | offered, and if kept used untrimmed: the Result's `APIKey` keeps the line feed | D5 ("offered and used as now, untrimmed"); Q2's option not chosen, "a kept value also trimmed before use" | `TestNonBlankEnvAndSavedKeyUntrimmed_0085` (proposed) |
| ST40 | `AllowEnv` false, or a descriptor with no `EnvVar` | out of scope | unchanged: the environment is not read | not a value this record meets | none |
| ST41 | `Options.Existing.APIKey` blank, for a provider that requires a key | decided | absent: no `Confirm`; the typed prompt follows | D5 (Q2) | `TestBlankEnvAndSavedKeyAreAbsent_0085` |
| ST42 | `Options.Existing.APIKey` blank, for a provider whose key is optional | decided | absent: listed with no key (the trim is `keepExistingKey`'s, which both branches call, C10) | D5 (Q2) | `TestBlankEnvAndSavedKeyAreAbsent_0085`, a Chat Completions case added (proposed) |
| ST43 | `Options.Existing.APIKey` not blank but padded | decided | offered, and if kept used untrimmed | D5 | `TestNonBlankEnvAndSavedKeyUntrimmed_0085` (proposed) |
| ST44 | a valid stored key exists (`Existing` or the environment), the person declines it, then three blanks | decided | the no-key error, no Result; a blank never falls back to the declined key; the wizard writes no configuration, so the caller's stored key is as it was (the caller decides what to do with it) | D3; `ConfigureLLM`'s doc, "It never writes configuration" | `TestBlankAfterDecliningSavedKey_0085` (proposed) |
| ST45 | a valid stored key, declined, then a blank and a new key | decided | the new key is the Result's | D3 | `TestBlankAfterDecliningSavedKey_0085` (proposed) |
| ST46 | a valid stored session in `Options.TokenStore`, declined; the API key method; three blanks | decided | the no-key error; the store's session is as it was, since a key reaches no save | D3; 0081-MADR, amendment of 2026-11-18 ("a key reaches no `save`") | `TestBlankAfterDecliningSavedKey_0085`, a session case (proposed) |

**G. The programmatic API outside the wizard, which this record leaves out**

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST47 | a provider built with `llmprovider.WithAPIKey("")` | out of scope | unchanged: six refuse it with `ErrInvalidRequest`; Kilo, OpenCode Zen and OpenCode Go send their anonymous or public token (C45 to C47) | the ruling ("at the prompt"); D7 relies on it for anonymous use (C48) | none |
| ST48 | a provider built with a key of spaces | out of scope | unchanged: each constructor compares the key with `""` untrimmed (C67now to C69now), so it is not refused there and is sent | the ruling ("at the prompt"); D2 | none |
| ST49 | `signin.VerifyKilo` with an empty key | out of scope | unchanged (C66now) | PLAN "Out of scope" (0080's checks and `llmprovider/x/signin`) | none |
| ST50 | a vendor CLI's auth file, a `TokenStore`'s file | out of scope | unchanged: no typed key; this record includes the environment and `Options.Existing` only | D5 | none |

**H. The run's state, and each platform**

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST51 | under `Discover`, a blank then a key | decided | the listing is asked once, with the key; for Kilo, the profile is read once, with the key; no "refused this key" notice | Decision Drivers ("A refusal before any request"); D2 | `TestBlankKeyNeverReachesTheListing_0085` (Claude); a Kilo case added (proposed) |
| ST52 | without `Discover`, any blank answer | decided | a blank is never the Result's `APIKey` | D1, D3 | `TestTypedBlankKeyIsRefused_0085` |
| ST53 | the run's context is cancelled, or its deadline passes, while the prompt is being asked again (a caller's `Prompter` returns blank after the cancel) | open | the PLAN's helper takes no context: it asks up to twice more, then gives the no-key error, which does not match the context's; with a key, 0081's check after the step returns the context's error. `ConfigureLLM`'s doc promises an error matching ctx's for an ended context in the cases it names | none (0081-MADR D4 makes a deadline a cancel) | none until answered |
| ST54 | two runs at once, or a `Prompter` used from two goroutines | out of scope | the helper holds no state outside its call, and the wizard calls `Secret` from one goroutine | not a case this change creates | none |
| ST55 | permission refused | out of scope | the change opens no file and calls no service; `LookupEnv` is the caller's function | not a case this change meets | none |
| ST56 | interrupted while writing | out of scope | nothing is written: a key reaches no save | 0081-MADR, amendment of 2026-11-18 | none |
| ST57 | Linux, macOS, Windows | decided | the same outcome: the helper is string handling with no platform branch; `TextPrompter`'s per-platform paths are unchanged | D2, D4 | the Phase 1 tests, in CI's three-system matrix (`.github/workflows/ci.yml`) |

**I. The sibling records**

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST58 | 0081 lands first and splits `credentialStep` into an unsaved step and `commitCredential`, moving the re-ask call site (C6) and changing the docs of `ConfigureLLM` and `ResolveCredential` | decided | this record changes `promptAPIKey`'s body, not its call sites; the doc sentences are merged into 0081's at this record's base | "Relation to 0080 and to the save-on-cancel record" (the edits "do not overlap in code; the doc comments may") | the PLAN's Phase 1 step 1, redone at 0081's commit; `TestConfigureBlankKeyIsRefused_0085` against 0081's `ConfigureLLM` |
| ST59 | 0081's "saving last": a run that ends with the no-key error | decided | nothing is saved and no Result is returned | D3; 0081-MADR D5 (one save path, after the step) | `TestBlankAfterDecliningSavedKey_0085`, its session case (proposed) |
| ST60 | 0083 lands after and changes `Prompter.Select`'s doc and the package doc in `wizard/prompter.go`, beside this record's `Prompter.Secret` doc | out of scope | its question uses `Select` and `Confirm`, never `Secret`; 0083's own re-grounding reads `wizard/prompter.go` as this record leaves it | D2 | none |
| ST61 | 0084 changes the send paths of `claude.go` and `opencode.go` | out of scope | this record cites their constructors as evidence only (C46, C47) and changes neither; the PLAN re-reads the cited lines at its base | D2 | none |
| ST62 | 0080's `verifyKilo`: its skip of an empty token, and its comment that this record's gap is "found, and not changed here" (C64now, C65now) | open | the skip stays (D8). The comment becomes untrue when this record lands; the PLAN's scope does not name `wizard/kilo_verify.go`, and its step 6 covers earlier records, not code comments | D8, for the skip only | none until answered |
| ST63 | the later record for keyless use (D7) | decided | nothing is built for it: no field, no descriptor fact, no menu entry, no code naming the three providers | D7 (Q6) | `TestTypedBlankKeyIsRefused_0085` (all nine refused alike) |

**J. What a caller observes**

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST64 | a caller whose `Prompter` returned a blank key, which today gets a `CredAPIKey` Result with an empty key | decided | a notice and the question again, then the no-key error; a blank environment value or saved key is no longer offered; a blank OpenAI paste is asked again; a blank Grok paste is asked again | D8 | the tests of ST1, ST11, ST32, ST33 and ST37 |
| ST65 | a caller that relied on a blank key for anonymous Kilo, OpenCode Zen or OpenCode Go through the wizard | decided | the no-key error after three asks; it builds the provider without the wizard's key step until the later record exists | D7; Consequences, the first Bad bullet | `TestTypedBlankKeyIsRefused_0085`; `TestForeverBlankPrompterEnds_0085` |
| ST66 | the release note | decided | a **Fixed** entry under `[Unreleased]` naming what ST64 lists | D6 (Q5, "Fixed, in the next release with 0080 and the other four (Recommended)") | `make changelog-check` (form only); Phase 1's review of the entry |

**Tally.** 66 rows: 47 decided, 14 out of scope, 5 open (ST12, ST13, ST15,
ST53, ST62), none awaiting live evidence. By the `madr-and-plan-writing`
rule, a record with an open row is `proposed`; this amendment does not
change the status, which is the owner's. The PLAN's Phase 1 does not start
until the five are answered.

### The open rows, as put to the owner

Each is put as one question in the coordinator's one prompt; the answers go
into "Decided" with their labels, and the rows become `decided`.

* **ST12, ST13.** The notice's and the error's words, and whether the third
  blank also gives the notice.
* **ST15.** Whether the blank count restarts at each call of the typed
  prompt within a run.
* **ST53.** What a run whose context ends while the prompt is asked again
  returns, and whether it asks again at all.
* **ST62.** Whether Phase 1 rewords `verifyKilo`'s comment and adds a dated
  note to 0080-PLAN's "Found, not changed".

### The evidence index re-pointed at `24756a6`

The index above stays as it was, at `b109cea`. This table is beside it, one
row per row, with the same claim, and eight rows more (C64 to C70, G2); its
ids end in "now" so that `tools/claims.py`, which reads one revision marker
per record, does not read them against `b109cea`. Each row of this
repository was checked at `24756a6` by `tools/claims.py` on a scratch copy
of this table with plain ids and its own marker (`self=24756a6`): 63 of 63
pass, and the new rows with `gobble-cli=b1994a0` added, 8 of 8. Two
planted errors in the scratch copy (C1 one line off; C50 at its old line)
were each caught. The four rows of other repositories (G1, U1 to U3) stay
pinned to the revisions of the marker above.

Tally of the 67 rows: **29 unchanged**, **33 moved** (same text, still
true), **1 moved with changed text** (C50: still true in substance; the
paragraph now also names Kilo's profile), **4 pinned**. C24, C33 and C62
match more than one line; the row names the one that carries the claim.
After 0081's code, C6, C7, C25, C27 to C29 and C33 are in the functions
0081 restructures, and are re-read at this record's base (PLAN Phase 1
step 1).

| Id | Claim | Now | Location at `24756a6` | Text |
| :--- | :--- | :--- | :--- | :--- |
| C1now | The typed key's prompt | moved | `wizard/configure.go:855` | `func promptAPIKey(p Prompter, d llmprovider.Descriptor, o Options) (string, error) {` |
| C2now | asks once | moved | `wizard/configure.go:856` | `key, err := p.Secret(fmt.Sprintf("Enter your %s API key", d.Label))` |
| C3now | trims and tests nothing | moved | `wizard/configure.go:860` | `key = strings.TrimSpace(key) // a paste's spaces (0020-MADR F18)` |
| C4now | returns whatever is left | moved | `wizard/configure.go:862` | `return key, nil` |
| C5now | `resolveAPIKey` asks it as the typed key | moved | `wizard/configure.go:834` | `key, err := promptAPIKey(p, d, o)` |
| C6now | the credential step asks it again after a refusal | moved | `wizard/configure.go:563` | `key, keyErr := promptAPIKey(p, d, o)` |
| C7now | and makes it the credential | moved | `wizard/configure.go:567` | `credential = staticCredential(CredAPIKey, key)` |
| C8now | The environment skips only an empty value | moved | `wizard/configure.go:816` | `if envVal := o.lookupEnv()(d.EnvVar); envVal != "" {` |
| C9now | and returns it untrimmed | moved | `wizard/configure.go:823` | `return envVal, keyEnvironment, nil` |
| C10now | A saved key is not offered only when empty | moved | `wizard/configure.go:841` | `if o.Existing.Provider != d.ID \|\| o.Existing.APIKey == "" {` |
| C11now | and is returned untrimmed | moved | `wizard/configure.go:849` | `return o.Existing.APIKey, true, nil` |
| C12now | A provider that requires a key is an API-key credential whatever the key | moved | `wizard/auth.go:189` | `if d.RequiresAPIKey \|\| key != "" {` |
| C13now | The menu's API-key method asks the same | moved | `wizard/auth.go:225` | `key, from, keyErr := resolveAPIKey(p, d, o)` |
| C14now | The pasted credential's own prompt | moved | `wizard/auth.go:604` | `value, err := p.Secret(fmt.Sprintf("Paste your %s credential", d.Label))` |
| C15now | trims | moved | `wizard/auth.go:610` | `value = strings.TrimSpace(value)` |
| C16now | and returns it as a key | moved | `wizard/auth.go:622` | `return staticCredential(CredAPIKey, value), nil` |
| C17now | OpenAI's paste is checked as an access token | moved | `wizard/auth.go:616` | `if err := checkAccessToken(value, time.Now()); err != nil {` |
| C18now | `TextPrompter` trims a line | unchanged | `wizard/text_prompter.go:156` | `trimmed := strings.TrimSpace(line)` |
| C19now | `Secret` refuses an empty value | unchanged | `wizard/text_prompter.go:402` | `if value != "" && !readFailed(err) {` |
| C20now | and warns | unchanged | `wizard/text_prompter.go:410` | `p.Notify(LevelWarn, "empty value; please try again")` |
| C21now | raw mode trims the entry | unchanged | `wizard/text_prompter.go:496` | `return strings.TrimSpace(string(entered)), nil` |
| C22now | the end of input is an error | unchanged | `wizard/text_prompter.go:408` | `return "", fmt.Errorf("wizard: no value entered: %w", err)` |
| C23now | `Prompter.Secret`'s doc says nothing of an empty value | moved | `wizard/prompter.go:94` | `// Secret reads a credential without echoing it in full. Implementations` |
| C24now | A Kilo device login with no token is an error | moved (also at 522) | `wizard/auth.go:450` | `errors.New("wizard: Kilo login returned no token")` |
| C25now | no model entered is a plain error | moved | `wizard/configure.go:348` | `fmt.Errorf("wizard: no model available for %s and none entered", d.Label)` |
| C26now | Typed keys are bounded at three | moved | `wizard/configure.go:726` | `const maxTypedKeys = 3` |
| C27now | the existing refusal loop | moved | `wizard/configure.go:515` | `for keys, workspaces := 1, 0; stepRefusal(listErr); {` |
| C28now | and its notice | moved | `wizard/configure.go:560` | `p.Notify(LevelWarn, "%s refused this key (%s); enter another", d.Label, describe(listErr))` |
| C29now | An optional key is asked only after a refusal | moved | `wizard/configure.go:550` | `p.Notify(LevelWarn, "%s refused the request without a key (%s); enter its key", d.Label, describe(listErr))` |
| C30now | `resolveAPIKey` branches on `RequiresAPIKey` | moved | `wizard/configure.go:802` | `if !d.RequiresAPIKey {` |
| C31now | a local provider sends none | moved | `wizard/configure.go:804` | `return "", keyOther, nil` |
| C32now | `notifyShape` already guards an empty key | moved | `wizard/auth.go:731` | `if !ok \|\| key == "" \|\| s.re.MatchString(key) {` |
| C33now | `ConfigureLLM` and `ResolveCredential` share the step | moved (also at 429) | `wizard/configure.go:332` | `res, cat, err := credentialStep(ctx, p, d, o)` |
| C34now | The descriptor's fact | unchanged | `llmprovider/registry.go:31` | `RequiresAPIKey bool` |
| C35now | and its local flag | unchanged | `llmprovider/registry.go:29` | `IsLocal bool` |
| C36now | Kilo requires a key | unchanged | `llmprovider/providers/kilo/descriptor.go:23` | `RequiresAPIKey:  true,` |
| C37now | OpenCode Zen requires a key | unchanged | `llmprovider/providers/opencode/descriptor.go:19` | `RequiresAPIKey:  true,` |
| C38now | Together requires a key | unchanged | `llmprovider/providers/together/descriptor.go:19` | `RequiresAPIKey:  true,` |
| C39now | Gemini requires a key | unchanged | `llmprovider/providers/gemini/descriptor.go:17` | `RequiresAPIKey: true,` |
| C40now | Claude requires a key | unchanged | `llmprovider/providers/claude/descriptor.go:17` | `RequiresAPIKey: true,` |
| C41now | Grok requires a key | unchanged | `llmprovider/providers/grok/descriptor.go:24` | `RequiresAPIKey: true,` |
| C42now | OpenAI requires a key | unchanged | `llmprovider/providers/openai/descriptor.go:25` | `RequiresAPIKey: true,` |
| C43now | Ollama is local and takes none | unchanged | `llmprovider/providers/ollama/descriptor.go:18` | `IsLocal:         true,` |
| C44now | a Chat Completions descriptor requires no key | unchanged | `llmprovider/providers/chatcompat/descriptor.go:7` | `// sets, and no key required. It has no default base URL, no environment` |
| C45now | Kilo's constructor with no key sends the anonymous token | unchanged | `llmprovider/providers/kilo/kilo.go:203` | `src = llmprovider.NewStaticToken(anonymousToken)` |
| C46now | OpenCode's sends the public token | unchanged | `llmprovider/providers/opencode/opencode.go:238` | `src = llmprovider.NewStaticToken(publicToken)` |
| C47now | Claude's refuses a blank key at once | unchanged | `llmprovider/providers/claude/claude.go:214` | `return nil, fmt.Errorf("%w: claude api key is required", llmprovider.ErrInvalidRequest)` |
| C48now | The README says Kilo and OpenCode accept none | unchanged | `README.md:84` | `Ollama needs no key. Kilo and OpenCode accept none: they then send the` |
| C49now | The README's wizard section | moved | `README.md:434` | `` - A sign-in saves its session to `Options.TokenStore`, once the model `` |
| C50now | The architecture's account of a refused key | moved, text changed | `docs/architecture.md:1249` | `- **A refused key.** When the listing, or for a Kilo credential used with` |
| C51now | and of an optional key | moved | `docs/architecture.md:1223` | `- **An optional key, and no default endpoint.** A remote provider with no` |
| C52now | The guide's order of the key prompts | unchanged | `docs/guides/adding-a-provider.md:282` | `` otherwise `Secret`. `` |
| C53now | The changelog's open section | unchanged | `CHANGELOG.md:12` | `## [Unreleased]` |
| C54now | The compatibility promise | moved | `docs/guides/api-standards.md:318` | `` - **R48. From `v1.0.0`, every non-internal package outside `llmprovider/x/` is `` |
| C55now | 0028's refused-key rule | unchanged | `docs/decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md:415` | `### D-H10. The wizard asks again when the service refuses the key` |
| C56now | 0066's optional key | unchanged | `docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md:101` | `* **V2. A Chat Completions server that wants a key cannot be configured,` |
| C57now | Kilo's gateway serves free models to anonymous requests | unchanged | `docs/decisions/0080-MADR-verify-a-kilo-credential-through-its-profile.md:219` | `The gateway allows unauthenticated access for free models only.` |
| C58now | The test prompter returns what it is scripted with | unchanged | `wizard/fake_prompter_test.go:162` | `func (f *fakePrompter) Secret(prompt string) (string, error) {` |
| G1now | gobble's `Secret` turns an empty answer into a cancel | pinned to its repository | `gobble-cli:internal/cli/prompt.go:71` | `if err == nil && s == "" {` |
| C59now | OpenCode Go requires a key | unchanged | `llmprovider/providers/opencode/descriptor.go:34` | `RequiresAPIKey:  true,` |
| C60now | Hugging Face requires a key | unchanged | `llmprovider/providers/huggingface/descriptor.go:19` | `RequiresAPIKey:  true,` |
| C61now | `ResolveCredential`'s doc: a listing may need no credential | moved | `wizard/configure.go:398` | `// that needs no credential, such as Hugging Face's, says nothing of the key.` |
| C62now | `resolveTokenStdin` holds the run's `Options` | moved (also at 111, 176, 390, 443, 551, 782, 857, 893) | `wizard/auth.go:600` | `o Options,` |
| C63now | `Descriptor` has no fact that a provider accepts no key | unchanged | `llmprovider/registry.go:16` | `type Descriptor struct {` |
| U1now | kilocode's key prompt refuses only a zero-length value | pinned to its repository | `kilocode:packages/opencode/src/cli/cmd/providers.ts:176` | `validate: (x) => (x && x.length > 0 ? undefined : "Required"),` |
| U2now | opencode's refuses only an empty value | pinned to its repository | `opencode:packages/cli/src/commands/handlers/auth/form.ts:65` | `validate: (value) => (!value ? "Required" : undefined)` |
| U3now | codex trims stdin and refuses an empty result | pinned to its repository | `codex:codex-rs/cli/src/login.rs:363` | `if secret.is_empty() {` |
| C64now | verifyKilo skips an empty token | new | `wizard/kilo_verify.go:94` | `if token == "" {` |
| C65now | and its comment names this record's gap as found and not changed | new | `wizard/kilo_verify.go:95` | `// A Prompter of the caller's own that returns an empty or all-space` |
| C66now | signin.VerifyKilo has no empty-key test of its own | new | `llmprovider/x/signin/verify.go:74` | `func VerifyKilo(ctx context.Context, key, organization string, opts ...llmprovider.Option) error {` |
| C67now | Claude's constructor compares the key with "" untrimmed | new | `llmprovider/providers/claude/claude.go:213` | `if s.Value == "" {` |
| C68now | Kilo's constructor likewise | new | `llmprovider/providers/kilo/kilo.go:205` | `if s.Value == "" {` |
| C69now | OpenCode's likewise | new | `llmprovider/providers/opencode/opencode.go:240` | `if s.Value == "" {` |
| C70now | TextPrompter's raw mode leaves a late line feed to the next prompt | new | `wizard/text_prompter.go:487` | `// read is not waited for: it is the user's next Enter.` |
| G2now | gobble's Secret still compares the answer untrimmed at its current head | new, at gobble-cli `b1994a0` | `gobble-cli:internal/cli/prompt.go:71` | `if err == nil && s == "" {` |

## Amendment of 2026-11-20, second pass: the owner's rulings, the review, and the constructors

Added the same day, after an independent review of the first amendment and
the owner's rulings on its open rows. Nothing above is rewritten or
removed. Where a row of the first amendment's table is decided or changed
here, the row below with the same id replaces it from this amendment on; the
first amendment's row stays as the record of what was open. Facts are of
`24756a6`, and of gobble-cli at `b1994a0`, read only.

### The owner's rulings of 2026-11-20

Each from the owner, relayed by the coordinator; the label is quoted as it
was given, and what follows it is this record's account of the option.

* **ST12, ST13, the words and the third blank:** "Approve the draft as it is
  (Recommended)". The notice is "no key was entered; <Label> needs one",
  shown after the first and the second blank; the third blank gives no
  notice and ends the run with "wizard: no <Label> key entered after 3
  tries", at the typed and the paste prompt alike (PLAN step 4).
* **ST15, the count after a refused key:** "Restart at each prompt, as
  drafted (Recommended)". Each call of the typed prompt has its own three
  asks; with the three keys of 0028-MADR D-H10, a run asks `Secret` at most
  nine times.
* **ST53, a context that ends during a repeat ask:** "Check before each
  repeat ask; return the cancel's error (Recommended)". Decision D10 below.
* **ST62, `wizard/kilo_verify.go:95`:** "Reword the comment in 0085 and note
  it in 0080's plan (Recommended)". The comment is reworded in Phase 1, the
  file joins Phase 1's scope for that one comment, and a dated note goes in
  0080-PLAN's "Found, not changed by 0080".
* **Review case 1, a refused key followed by blanks:** "Confirm D3; add the
  row and test; note it for gobble (Recommended)". Row ST67.
* **ST48, a key made only of spaces given to a provider's constructor:**
  "Pull it into 0085". This is not the recommended option. The outcome the
  owner chose: constructors treat a whitespace-only key as no key. Decision
  D9 below; it widens this record from the wizard's prompts to the public
  constructors of every provider.

### D9. A provider's constructor treats a whitespace-only key as no key (the owner's ruling on ST48)

* **The ruling.** A key that is empty after `strings.TrimSpace`, given to a
  provider through `llmprovider.WithAPIKey` or `llmprovider.NewStaticToken`,
  is no key, and is never sent to a service as a credential. A key that
  reaches a request by another path is not covered (third pass, "Not
  reached"; fourth pass). This is a new decision, added by this amendment;
  D1 to D8 stand, and D2's "the check is the prompt's" stays true of the
  wizard's prompts, beside it.
* **What it reaches.** Every built-in provider's constructor (`claude.New`,
  `gemini.New`, `grok.New`, `huggingface.New`, `kilo.New`, `openai.New`,
  `opencode.NewZen`, `opencode.NewGo`, `together.New`, `ollama.New`,
  `chatcompat.New`), through `llmprovider.WithAPIKey` and
  `llmprovider.WithTokenSource` with a `*llmprovider.StaticToken`, and
  through `providers.New` and `Registry.New`, which call the ten built-ins
  they list, and `chatcompat.New` directly. No provider
  has a key option of its own; `WithAPIKey` is the one key option.
* **What it leaves open** (rows ST73 to ST75, for the owner): what "no key"
  yields for each provider, since today's empty-key behaviour differs by
  provider (0020-MADR F52: six refuse an empty key at `New`, Kilo and the
  OpenCode gateways send their anonymous key "by design", and Ollama and a
  Chat Completions server send no `Authorization`); where the rule lives,
  which decides whether it reaches a caller-registered descriptor's
  constructor, `catalog.List`'s token source and `llmprovider/x/signin`; and
  whether a padded key that is not blank is trimmed.
* **Unchanged by it:** an empty key `""`, which is no key already (ST71); a
  key that is not blank (pending ST75); `TokenSource`s that are not a
  `*StaticToken` (sessions, a vendor CLI's login, a `CommandToken`); every
  exported name (R55: none is added).
* **Consequences.** Good, because a blank environment variable passed to
  `WithAPIKey`, as this module's own example does
  (`llmprovider/provider.go:17`, `README.md:42`), is no longer sent as a key,
  and the wizard's D5 and the constructors agree on a blank value. Bad,
  because it is a change to public constructor behaviour, a **Changed** entry
  under `[Unreleased]`, which a caller that sent a blank key on purpose
  observes; and because, under ST73's first option, a blank Kilo or OpenCode
  variable then gives anonymous use rather than a refused key.

### D10. A repeat ask checks the context first (the owner's ruling on ST53)

* Before the second and the third ask at the typed prompt and at the paste
  prompt, the helper reads the run's context. If it has ended, the prompt
  returns at once with an error that wraps the context's error, wrapped as
  the prompt's other errors are ("enter API key: " or "enter credential: "),
  so `errors.Is` matches `context.Canceled` or `context.DeadlineExceeded`
  (0081-MADR D4: a deadline is a cancel). No further `Secret` call is made.
* The first ask is not preceded by a new check: `ConfigureLLM` and
  `ResolveCredential` already check before any question (0059-MADR W8).
* *(Fourth pass.)* That check precedes only the run's first ask. D10 covers
  the blank re-asks inside one call of the prompt, and nothing else: the
  ruling was put as the context ending while the key prompt is asked again
  after a blank. The typed prompt asked after a listing refusal
  (`wizard/configure.go:563`) and the sign-in menu after a refused kept
  session (`:544`) come after a listing request and are not covered by it
  (rows ST53b, ST53c); where a cancelled run is checked is 0081's record.
* The context is threaded through the three unexported functions that lack
  it, `resolveAPIKey`, `promptAPIKey` and `secretKey`; `resolveTokenStdin`
  already takes it (as `_`). No exported signature changes.
* *(Annotated 2026-11-21, Phase 1 build pass 1.)* Two existing test files
  call `resolveAPIKey` directly and so gain `context.Background()` as the
  first argument of three calls, by the owner's ruling "Add
  context.Background() to the 3 calls (Recommended)":
  `wizard/auth_shapes_0028_test.go` (two calls) and
  `wizard/shape_0059_test.go` (one call). Nothing else in those files
  changes. No decision changes.

### The review's findings, each checked at the cited lines

* **Case 1, a refused key followed by blanks: holds.** At `24756a6` a refused
  typed key is asked for again (`wizard/configure.go:560` to `:566`), and the
  run that ends on a refusal returns `refusalError`, which wraps the
  listing's error and so matches `llmprovider.ErrAuthFailure` (`:554` to
  `:558`, `:775`). After this record, the helper's no-key error is returned
  as it is at `:563` to `:566`. gobble-cli branches on `ErrAuthFailure` to
  report the keys it asked for (`llm/provider/signin.go:372` to `:374` at
  `b1994a0`; the review cited `:366` to `:368`). Applied: row ST67.
* **Case 2, constructors and a key of spaces: holds.** `WithAPIKey` stores
  the key as given (`llmprovider/settings.go:109`); each constructor tests
  `== ""` (C67now to C69now, and the same test in Gemini, Grok, Hugging
  Face, OpenAI and Together); the package example passes
  `os.Getenv(...)` (`llmprovider/provider.go:17`); gobble's environment
  path tests `!= ""` (`llm/provider/ambient.go:31`) and passes the key to
  `WithAPIKey` (`llm/provider/provider.go:218` to `:219`). Applied by the
  owner's ruling: D9, rows ST71 to ST80.
* **Case 3, a caller-registered descriptor with `RequiresAPIKey`: holds.**
  The first amendment's rows name the nine built-ins and the Chat
  Completions server only. Applied: row ST68.
* **Case 4, `TextPrompter`'s late line feed: holds, out of scope.** A line
  feed that raw mode leaves (C70now) is read by the next line-mode prompt as
  an empty line, which takes the default without a word
  (`wizard/text_prompter.go:356` to `:358` for `Input`; `Select` likewise).
  Applied: row ST69, recorded as found, with a follow-up.
* **Question 3, tests that cannot pin their rows: each holds**, checked
  against the drafting's tests as handed over: Kilo shows no method menu
  without a `TokenStore` or `KiloDeviceKey` (`wizard/auth.go:94` to `:95`,
  `:183`), and the drafting's Options set neither; the forever-blank test
  fails only above three asks and runs Together alone; the error test uses a
  sentinel; the two guards (ST21, ST31) and test 10's case (c) had no plant
  of this record's that turns them red; ST44's environment half had no test.
  Applied: the rows marked below, and the PLAN's entry of this date.

### The state table: rows decided or changed by this amendment

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST11 | three blanks in a row at one prompt | decided | exactly three `Secret` calls and two notices, then the no-key error; no Result; no listing request | D3 (Q3); the ruling on ST12, ST13 | `TestForeverBlankPrompterEnds_0085`, asserting exactly three asks and two notices (plants P-r, P-s) |
| ST12 | the third blank and the notice | decided | the first and second blank each give one notice; the third gives none and ends the run | the owner, relayed by the coordinator: "Approve the draft as it is (Recommended)" | `TestForeverBlankPrompterEnds_0085` (two notices; plant P-s) |
| ST13 | the words | decided | notice "no key was entered; <Label> needs one"; error "wizard: no <Label> key entered after 3 tries"; the same at the paste prompt | the owner, relayed by the coordinator: "Approve the draft as it is (Recommended)" | `TestTypedBlankKeyIsRefused_0085` (the notice; plant: the drafting's "notice removed"); `TestForeverBlankPrompterEnds_0085` (the error's text; plant P-txt) |
| ST15 | blanks across the refusal loop | decided | the count restarts at each call of the typed prompt: after a refused key, three more asks are possible; at most nine `Secret` calls in a run | the owner, relayed by the coordinator: "Restart at each prompt, as drafted (Recommended)" | `TestRefusedKeyThenBlanks_0085`, its count case (plant P-cnt) |
| ST19 | a caller's `Prompter` that says "cancelled" with `""` | decided | asked exactly twice more, then the no-key error, which does not match `context.Canceled` | D3; Consequences, Neutral | `TestForeverBlankPrompterEnds_0085` (plants P-r, P-s; P-h for the non-match) |
| ST21 | interactive `TextPrompter`, blank entries, line or raw mode | decided | `TextPrompter` asks again itself, three of its own warnings "empty value; please try again", and never returns a blank; the wizard's notice is not shown | D2; R2 | `TestTextPrompterRefusesBlank_0085`, asserting three `TextPrompter` warnings and no wizard notice (plant P-v) |
| ST22 | Ctrl-C at `TextPrompter`'s raw-mode prompt | decided | "wizard: cancelled: context canceled", wrapped "enter API key: ", returned at once; `errors.Is(err, context.Canceled)` | D2, D3 | `TestKeyPrompterErrorIsReturned_0085`, a raw-mode case through `rawSeam0059` and a Ctrl-C byte (plant P-u) |
| ST27 | the API key as a provider's only method and as a menu choice | decided | the same refusal on both branches; Grok and OpenAI take the menu with no `TokenStore`, Kilo only with one (or `KiloDeviceKey`) | D2 | `TestTypedBlankKeyIsRefused_0085`, with Kilo run twice, without and with a memory `TokenStore`, asserting which runs asked the method menu (plant P-b) |
| ST31 | a provider that takes no key: Ollama, a keyless Chat Completions server | decided | no key prompt | D2 | `TestCompatNoKeyNeeded_0085` (plant P-w); `TestOllama_RefusedNoKeyAsked_0066` (plant P-x) |
| ST44 | a valid stored key (`Existing`, or the environment under `AllowEnv`), declined, then three blanks | decided | the no-key error, no Result; a blank never falls back to the declined key; the wizard writes no configuration | D3; `ConfigureLLM`'s doc | `TestBlankAfterDecliningSavedKey_0085`, cases (a) `Existing` (plant P-o) and (d) the environment (plant P-o2) |
| ST46 | a valid stored session, declined; the API key method; three blanks | decided | the no-key error, no Result; the store is not written, neither saved nor deleted | D3; 0081-MADR, amendment of 2026-11-18 | `TestBlankAfterDecliningSavedKey_0085`, case (c) (plant P-q for the error; P-y for the store) |
| ST48 | a provider built with a key of spaces | decided (by D9; replaces the first amendment's "out of scope") | see ST72 to ST75 | D9 | see ST72 |
| ST53 | the context ends, by cancel or deadline, before a repeat ask | decided | no further `Secret` call; an error wrapping the context's, matched by `errors.Is` | D10; the owner, relayed by the coordinator: "Check before each repeat ask; return the cancel's error (Recommended)" | `TestCancelBeforeRepeatAsk_0085` (plant P-ctx) |
| ST59 | 0081's saving last: a run that ends with the no-key error | decided | no Result, and the store is not written | D3; 0081-MADR D5 | `TestBlankAfterDecliningSavedKey_0085`, case (c) (plants P-q, P-y) |
| ST62 | `verifyKilo`'s skip and its comment | decided | the skip stays; the comment at `wizard/kilo_verify.go:95` to `:98` is reworded to say a prompt no longer returns a blank key (0085), and 0080-PLAN carries a dated note | D8 (the skip); the owner, relayed by the coordinator: "Reword the comment in 0085 and note it in 0080's plan (Recommended)" | a comment: Phase 1's review, and `tools/claims.py` on a row added for the new text at Phase 1 |
| ST65 | a caller relying on a blank key for anonymous Kilo, OpenCode Zen or OpenCode Go through the wizard | decided | for each of the three, three asks, then the no-key error | D7; Consequences | `TestForeverBlankPrompterEnds_0085`, run for all nine providers (plant P-t) |

**New rows**

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST67 | a key the listing refuses, then three blanks (also: an optional key refused without a key, C29; a kept session refused, then the API key method) | decided | the no-key error, which does not match `llmprovider.ErrAuthFailure` (today: `refusalError`, which does); one listing request; four `Secret` calls. gobble-cli's report of the keys asked for, read only for `ErrAuthFailure`, is not filled | D3 ("It matches no sentinel and is not an `ErrAuthFailure`"); the owner, relayed by the coordinator: "Confirm D3; add the row and test; note it for gobble (Recommended)" | `TestRefusedKeyThenBlanks_0085` (plants P-auth, P-auth2) |
| ST68 | a caller's descriptor in `Options.Registry` with `RequiresAPIKey` | decided | asked again, as ST1; the refusal names no provider | D2 ("the check is the prompt's, not the descriptor's") | `TestTypedBlankKeyIsRefused_0085`, a registered-descriptor case (plant P-reg) |
| ST69 | `TextPrompter`'s raw mode leaves a late line feed, which the next line-mode prompt takes as an empty line and its default | out of scope, found | unchanged; a `TextPrompter` defect apart from this record. Follow-up: a record of its own, for the owner to open | D2 (`TextPrompter` unchanged) | none |
| ST70 | the docs that state the gap today: `README.md:538`, `CHANGELOG.md:57` (0080's entry under `[Unreleased]`), `docs/architecture.md:1169` | decided | each sentence is corrected in Phase 1 to say a blank key is asked again | PLAN Phase 1 "Acceptance" ("the docs say what the code does") | `make docs-check`; Phase 1's review |
| ST71 | `""` given to a built-in constructor | decided | unchanged: Claude, Gemini, Grok, Hugging Face, OpenAI and Together refuse it at `New` with `ErrInvalidRequest`; Kilo sends its anonymous key, OpenCode Zen and Go their public key, each keeping a caller's `Header`; Ollama and a Chat Completions server send no `Authorization` | 0020-MADR F52; D7 relies on it; D9 ("Unchanged by it") | the existing `TestKeyedProviders_RefuseAnEmptyKey`; `TestConstructorsEmptyKeyUnchanged_0085` (proposed; plant P-k1) |
| ST72 | a whitespace-only key (spaces, a tab, a lone CR, a lone LF, CR+LF) given to each built-in constructor through `WithAPIKey` | decided | it is never sent as a credential: no request to the service carries it in any header | D9 | `TestConstructorsBlankKeyIsNoKey_0085` (proposed; plant P-k2) |
| ST73 | what "no key" then yields for each provider | open | today the providers differ (ST71); 0020-MADR F52 decides `""` only | none | none until answered |
| ST74 | where the rule lives, and so what else it reaches: a caller-registered descriptor's constructor, `WithTokenSource(NewStaticToken(key))`, a `StaticToken` with its own `Header`, `catalog.List`'s token source, `llmprovider/x/signin`'s `APIKey` and `VerifyKilo` | open | today none of them tests for blank | none | none until answered |
| ST75 | a padded key that is not blank (`" sk-x "`, `"sk-x\n"`) given to a constructor | open | today it is sent as given; a line feed fails at the transport, since Go's HTTP client refuses a header value holding one. D5 keeps the wizard's environment and saved values untrimmed, and does not decide the constructors | none | none until answered |
| ST76 | a blank key and another source of a key (the provider's environment variable, a store) | decided | no constructor falls back to another source; no constructor reads the environment today, and "no key" adds none | D9 ("no key") | `TestConstructorsBlankKeyIsNoKey_0085`, with the provider's variable set by the test (plant P-k3) |
| ST77 | the wizard and the constructors on the same blank value | decided | the wizard treats a blank environment or saved value as absent (D5), and a constructor as no key (D9) | D5, D9 | the tests of ST37 and ST72 |
| ST78 | the public API and the docs | decided | no exported name is added or removed; `make api-check` and `tools/apidiff.py v1.31.0` show no change; the doc comments of every `New` named in D9 and of `WithAPIKey`, the package example (`llmprovider/provider.go:17`), `README.md` (`:42`, `:68`, `:84`), `docs/architecture.md` and a **Changed** entry under `[Unreleased]` say what a blank key does | D9; AGENTS.md "API work" (docs in the same change) | `make api-check`; `tools/apidiff.py v1.31.0`; `make docs-check`; Phase 1b's review |
| ST79 | what gobble-cli and other consumers observe | decided | a refused key followed by blanks ends with the no-key error, not `ErrAuthFailure` (ST67); a blank environment variable passed to `WithAPIKey` is no longer sent (ST72), with ST73's outcome in its place. The changelog entries and PLAN Phase 4's adoption note say both | D3, D9; the ruling on case 1 | Phase 4 (gobble-cli's record) |
| ST80 | files other records change | decided | `llmprovider/providers/claude/claude.go` and `llmprovider/providers/opencode/opencode.go` (0084's send paths); `README.md`, `docs/architecture.md`, `CHANGELOG.md` (0081, 0083, 0084); `wizard/kilo_verify.go` (0081, by the owner's ruling, changes its notice at `:128` and when warnings are shown). Each is re-read at the start commit of the phase that edits it | "Relation to 0080 and to the save-on-cancel record" (the second to land re-reads and merges) | the first step of Phases 1 and 1b |

2026-11-20, ST69's follow-up: [0087-MADR-late-line-feed-after-a-hidden-entry.md](0087-MADR-late-line-feed-after-a-hidden-entry.md) and [0087-PLAN-late-line-feed-after-a-hidden-entry.md](0087-PLAN-late-line-feed-after-a-hidden-entry.md).

**Tally after this amendment.** 80 rows: 63 decided, 14 out of scope
(ST48 left that column for decided; ST69 joined it), 3 open (ST73, ST74,
ST75), none awaiting live evidence. The constructor phase (PLAN Phase 1b)
does not start until the three are answered; Phase 1 may start without
them.

### Self-check of the rows added or changed here

Against the review's three questions, limited to the rows above:

* **A missing case.** ST72 lists five whitespace shapes; a non-breaking
  space and a mix are D4's too, and the test's table carries the six values
  of the wizard's tests plus a lone CR and LF, so the two match. A key with
  its own `Header` and a key given as a `TokenSource` belong to ST74. No
  other case was found.
* **Decided but not decided.** ST71's "unchanged" rests on 0020-MADR F52 and
  D9's own "unchanged" list, not on today's code alone. ST76 rests on D9's
  "no key": a fall-back would be a new source, which the ruling did not
  choose. ST73 to ST75 stay open: the ruling chose the direction, not the
  outcome per provider, the place, or the trim.
* **A test that cannot pin its row.** Each decided row above names a test
  and a plant that turns it red; ST62 and ST70 are documentation, pinned by
  review and the docs checks, not by a Go test, and say so.

## Amendment of 2026-11-20, third pass: the owner's rulings on ST73 to ST75

Added the same day. Nothing above is rewritten or removed. The rows below
replace the second pass's rows of the same id; ST73 and ST74 are each split
into lettered rows. After this amendment no row is open.

### The owner's rulings

Each from the owner, relayed by the coordinator; the label as given.

* **ST73, what "no key" means for a whitespace-only key:** "Same as an empty
  key today, provider by provider (Recommended)".
* **ST74, where the rule lives:** "In `WithAPIKey` and `NewStaticToken`
  (Recommended)".
* **ST75, a padded key that is not blank:** "Sent as given (Recommended)".

### D9, its outcomes (dated addition, 2026-11-20, third pass)

D9's "What it leaves open" is answered:

* **What "no key" yields.** A whitespace-only key behaves exactly as `""`
  does today, provider by provider (0020-MADR F52): Claude, Gemini, Grok,
  Hugging Face, OpenAI and Together refuse it at `New` with an error
  matching `llmprovider.ErrInvalidRequest`, with the text `""` gives
  ("<id> api key is required"); Kilo sends its anonymous key and OpenCode
  Zen and Go their public key, keeping a caller's `Header`; Ollama and a
  Chat Completions server send no `Authorization`. No constructor falls back
  to another source.
* **Where.** In `llmprovider.NewStaticToken`, which stores a value that is
  empty after `strings.TrimSpace` as `""`, and in `llmprovider.WithAPIKey`,
  which builds its source with `NewStaticToken` (`llmprovider/settings.go:109`)
  and whose doc comment says so. It therefore reaches every constructor, a
  caller-registered one included, `catalog.List` given a `NewStaticToken`,
  and `llmprovider/x/signin`'s `APIKey` and `DeviceKey` sources, which use
  `NewStaticToken` (`x/signin/credential.go:39`, `:96`). No constructor's
  code changes. Not reached: a `StaticToken` built as a struct literal;
  `signin.VerifyKilo`, which takes the key as a string;
  `catalog.KiloModelCapabilities`, which builds `Token{Value: apiKey}`
  itself (`llmprovider/catalog/discovery.go:1530`, `:1535`); and
  `auth.KiloProfile`, which sends `"Bearer "+token` whatever the token
  (`llmprovider/auth/kilo_device.go:187`, `:199`). A blank key given to
  one of these is sent as it is today (rows ST74b, ST74d to ST74f).
* **A padded key that is not blank** is stored and sent as given, as D5
  keeps the wizard's environment and saved values.

### The state table: rows ST71 to ST80 as decided

Each of ST71 to ST77 has one test of PLAN Phase 1b; ST78 to ST80 are checks
and a step, named with them.

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST71 | `""` given to a built-in constructor | decided | unchanged, as D9's outcomes list it | 0020-MADR F52; D9 | `TestConstructorsEmptyKeyUnchanged_0085` (plant P-k1) |
| ST72 | a whitespace-only key (spaces, a tab, a non-breaking space, a mix, a lone CR, a lone LF, CR+LF) given through `WithAPIKey` to each of the eleven built-in constructors | decided | never sent as a credential: no request carries it in any header | D9 | `TestConstructorsBlankKeyIsNoKey_0085` (plant P-k2) |
| ST73a | the same key, to Claude, Gemini, Grok, Hugging Face, OpenAI or Together | decided | refused at `New` with an error matching `ErrInvalidRequest`, its text that of `""` | the owner, relayed by the coordinator: "Same as an empty key today, provider by provider (Recommended)" | `TestBlankKeyIsEmptyKeyPerProvider_0085` (plant P-k2) |
| ST73b | the same key, to Kilo, OpenCode Zen or OpenCode Go | decided | Kilo sends its anonymous key, OpenCode its public key; a caller's `Header` is kept | the same ruling | `TestBlankKeyIsEmptyKeyPerProvider_0085` (plant P-k2) |
| ST73c | the same key, to Ollama or a Chat Completions server | decided | no `Authorization` header is sent | the same ruling | `TestBlankKeyIsEmptyKeyPerProvider_0085` (plants P-k2 for the Chat Completions server, P-k6 for Ollama) |
| ST73d | the same key, to a caller-registered provider's constructor | decided | the constructor receives a `*StaticToken` whose `Value` is `""`, and does what it does with `""` | the same ruling; the owner, relayed by the coordinator: "In `WithAPIKey` and `NewStaticToken` (Recommended)" | `TestBlankKeyIsEmptyKeyPerProvider_0085` (plant P-k2) |
| ST74a | `NewStaticToken(v)` and `WithAPIKey(v)` for a blank `v` | decided | `NewStaticToken("  ").Value` is `""`; `WithAPIKey("  ")` resolves to a `*StaticToken` whose `Value` is `""` | the owner, relayed by the coordinator: "In `WithAPIKey` and `NewStaticToken` (Recommended)" | `TestStaticTokenBlankIsEmpty_0085` (plants P-k2, P-k8) |
| ST74b | a `StaticToken` built as a struct literal, `&StaticToken{Value: "  "}` | out of scope | not normalised: a literal calls neither function. `StaticToken`'s doc comment says so (PLAN Phase 1b step 4) | the same ruling ("In `WithAPIKey` and `NewStaticToken`") | none |
| ST74c | `catalog.List` given `NewStaticToken("  ")`; `x/signin`'s `APIKey{Key: "  "}` and `DeviceKey` sources | decided | reached: the token's `Value` is `""`, so no `Authorization` is sent | the same ruling | `TestStaticTokenBlankIsEmpty_0085` (plant P-k2) |
| ST74d | `signin.VerifyKilo(ctx, "  ", ...)`, which takes the key as a string | out of scope | unchanged: the string reaches neither function; PLAN "Out of scope" names `llmprovider/x/signin`'s checks | the same ruling; PLAN "Out of scope" | none |
| ST74e | `catalog.KiloModelCapabilities(ctx, "  ", model)` | out of scope (fourth pass) | unchanged: it builds `Token{Value: apiKey}` itself (`discovery.go:1535`), and its request sends a non-empty value (`:1419` to `:1420`), so the spaces are sent | the owner, relayed by the coordinator: "In `WithAPIKey` and `NewStaticToken` (Recommended)" places the rule there; a token built another way is not covered | none |
| ST74f | `auth.KiloProfile(ctx, "  ", ...)` | out of scope (fourth pass) | unchanged: it sets `Authorization` to `"Bearer "+token` whatever the token (`kilo_device.go:199`) | the same ruling, for the same reason | none |
| ST75 | a padded key that is not blank (`" made-up-key "`, `"made-up-key\n"`) given to `NewStaticToken` or `WithAPIKey` | decided | stored and sent as given | the owner, relayed by the coordinator: "Sent as given (Recommended)"; D5 | `TestConstructorsPaddedKeySentAsGiven_0085` (plant P-k7) |
| ST76 | a blank key and another source of a key | decided | no constructor falls back to the provider's environment variable or any other source | D9 | `TestBlankKeyNoFallback_0085` (plant P-k3) |
| ST77 | the wizard and the constructors on the same blank environment value | decided | the wizard does not offer it (D5); `WithAPIKey` of it gives `""`'s outcome (D9) | D5, D9 | `TestWizardAndConstructorAgreeOnBlank_0085` (plant P-k2; the first pass's "environment test untrimmed" for the wizard half) |
| ST78 | the public API and the docs | decided | no exported name added or removed; the doc comments of `WithAPIKey`, `StaticToken`, `NewStaticToken` and each constructor, the package example, `README.md`, `docs/architecture.md` and a **Changed** entry say what a blank key does, and that a struct literal is not changed | D9; AGENTS.md "API work" | `make api-check`; `tools/apidiff.py v1.31.0`; `make docs-check` |
| ST79 | what gobble-cli and other callers observe | decided | ST67's error; a blank variable passed to `WithAPIKey` refused at `New` for the six, anonymous or public use for Kilo and OpenCode | D3, D9 | Phase 4's adoption note; the **Changed** entry |
| ST80 | files other records change | decided | `claude.go` and `opencode.go`: 0084 changes their send paths and package docs; Phase 1b changes only the doc comments above `New`, `NewZen` and `NewGo`, so no common line. `README.md`, `docs/architecture.md` and `CHANGELOG.md`: changed by 0081, 0083 and 0084, and by Phases 1 and 1b. `wizard/kilo_verify.go`: 0081, by the owner's ruling, changes its notice at `:128` and when warnings are shown; Phase 1 rewords the comment at `:95` to `:98`. Each is re-read at the start commit of the phase that edits it, and the second to land merges | "Relation to 0080 and to the save-on-cancel record" | the first step of Phases 1 and 1b |

**Tally after this amendment.** 86 rows: 70 decided, 16 out of scope, 0
open, none awaiting live evidence.

## Corrections of 2026-11-20, fourth pass

Corrections found by an independent confirmation of the second and third
passes; no decision and no ruling changes. Edits to those passes' own text
(not yet committed) were made in place: D9's "never sent" now names the two
places, D9 names `chatcompat.New` apart from `providers.New`'s ten, D9's
"Not reached" list and rows ST74e and ST74f name two more key paths, D10
names what it does not cover, and ST80 carries the second pass's overlaps.
The rows below are added.

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST53b | the context ends after the listing request after which the typed prompt is asked again (`wizard/configure.go:563`) | out of scope | unchanged by this record: D10's check precedes only blank re-asks inside one call of the prompt | the owner's ruling on ST53, "Check before each repeat ask; return the cancel's error (Recommended)", was put as the context ending while the prompt is asked again after a blank; where a cancelled run is checked is 0081-MADR's (D2, D5, D9) | none |
| ST53c | the context ends during the listing request after which a refused kept session leads to the sign-in menu (`wizard/configure.go:544`) | out of scope | unchanged by this record, as ST53b | the same | none |
| ST81 | a token source that is not a `*StaticToken`, such as a `CommandToken`, returns a blank value | out of scope, found | unchanged by D9; such a value is sent as today. Found, not changed: nine providers look up model facts with `WithAPIKey(token.Value)` (for example `claude.go:397`, `kilo.go:361`; `catalog/lookup.go:164`), which after D9 gives `""`, while the listing is cached under the source's raw value (`catalog/discovery.go:337`), so the lookup misses that listing. `CachedFacts` starts no fetch (`catalog/lookup.go:160` to `:162`), so nothing more is sent | D9 ("Unchanged by it": sources that are not a `*StaticToken`) | none |

**Tally after the fourth pass.** 91 rows: 70 decided, 21 out of scope, 0
open, none awaiting live evidence.

## Corrections of 2026-11-21, Phase 1b build pass 2

Added after Phase 1b's build pass 1 and its independent review. Nothing above
is rewritten or removed: each statement named below stays as the record of
what was believed, and this section corrects it. Both items are the owner's
rulings, relayed by the coordinator.

### Three "today" facts were wrong

* **The statements.** The first amendment's row ST48 ("each constructor
  compares the key with `""` untrimmed (C67now to C69now), so it is not
  refused there and is sent"); the second pass's "Case 2, constructors and a
  key of spaces: holds" (the same claim, for every key made of whitespace);
  and the second pass's row ST75 ("a line feed fails at the transport, since
  Go's HTTP client refuses a header value holding one").
* **The truth, read at `e4835dc`.** `llmprovider/settings.go:260` to `:261`,
  in force since `faf06a8` (2026-10-07), refuses at `New` any static key that
  holds a byte below 0x20 or a DEL, with an error matching `ErrInvalidRequest`
  ("the API key holds a control character, such as a pasted newline"), for
  every provider, before any constructor's own test. So a key of only a tab, a
  lone CR, a lone LF, CR+LF, or a mix that holds one of them, was never sent:
  it was refused. A key of only spaces or a non-breaking space was the case
  the statements describe: built and sent.
* **What D9 changes for those keys, as built.** For a key of only tabs, CR, LF,
  CR+LF or a mix holding one: Kilo, OpenCode Zen and OpenCode Go go from
  refused to anonymous or public use; Ollama and a Chat Completions server go
  from refused at `New` to built, with no `Authorization`; Claude, Gemini,
  Grok, Hugging Face, OpenAI and Together keep the same kind, refused at
  `New` with `ErrInvalidRequest`, with different text ("<id> api key is
  required" for "the API key holds a control character"). A key that is not
  blank and holds a control character, such as `"key\n"`, is still refused at
  `New` by every provider.
* **The ruling.** The owner, relayed by the coordinator: **"Keep as built;
  correct the facts (Recommended)."** The behaviour stands for every blank
  value, tabs and line breaks included. This correction leaves the owner's
  ruling "Pull it into 0085" and D9 standing; the **Changed** entry says what
  changes for these keys. No decision changes.
* **Found, not changed.** The wizard keeps a non-blank environment value
  untrimmed (ST39), so `"key\n"` reaches `Result.APIKey`, and `New` then
  refuses it for the control character. This is not new and is not this
  record's: a follow-up for the owner.

### ST82: Together's listing with an empty token (the owner's ruling)

ST74c says a blank key given to the listing sends no `Authorization`. At
`e4835dc` that is true of a Chat Completions server's listing
(`llmprovider/catalog/dialect.go:79`) and not of Together's: its listing,
`fetchTogetherUsable`, calls `token.Apply` unconditionally
(`llmprovider/catalog/discovery.go:1326`), so a token whose value is `""`
sends a bare "Bearer". The owner, relayed by the coordinator: **"Fix
Together's listing."** ST74c stays as written.

| Id | Case | State | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST82 | `catalog.List` for Together given a token whose value is `""`, whether `NewStaticToken("")` or `NewStaticToken("   ")` | decided | the listing request carries no `Authorization`, as the Chat Completions listing's does; a token that is not empty is sent as before | the owner, relayed by the coordinator: "Fix Together's listing."; ST74c | `TestTogetherListingEmptyTokenSendsNoAuthorization_0085`; the tightened Together case of `TestStaticTokenBlankIsEmpty_0085` (plant: `token.Apply` restored unconditionally) |

**Tally after this correction.** 92 rows: 71 decided, 21 out of scope, 0
open, none awaiting live evidence.

*(Annotated 2026-11-21, Phase 1b CI fix.)* ST77's test,
`TestWizardAndConstructorAgreeOnBlank_0085`, moved from
`llmprovider/providers/blank_key_0085_test.go` to
`wizard/blank_key_agree_0085_test.go`, with the same assertions, by the
owner's ruling "Move that test into wizard (Recommended)": importing `wizard`
from the providers package's tests failed `make dep-check`. No decision
changes.

## Release note (2026-11-22)

`CHANGELOG.md`'s `[Unreleased]` became the `v1.32.0` section dated 2026-11-22 (R53: a section is dated by its tag's commit, so the date is re-checked by `make changelog-check` once the owner's commit exists); `[Unreleased]` is empty. `python3 -B tools/apidiff.py v1.31.0` lists additions only (`APIError.Tool`, `ErrToolSchemaRefused`, `x/signin.VerifyKilo`, `wizard.Options.NoBrowserQuestion`, `wizard.Options.NoLoopback`): a minor version. The tag is the owner's. This record's status stays as it is and goes to `complete` when the tag exists, its CI has passed, and `tools/consumer.py` and `tools/offline.py` are recorded (`tools/records.py status`). This record's entries are under `Changed` and `Fixed`.
