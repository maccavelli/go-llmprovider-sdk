---
status: accepted
date: 2026-10-25
decision-makers: repository owner
consulted: "this repository at dccf266; the test runs through a refusing, logging proxy and the stack traces of the default document's fetches, 2026-10-25 (the Evidence below)"
informed: every contributor who writes a test that lists or looks up models
---

<!-- markdownlint-disable MD013 -->

# Keep the Tests Off the Network, and Hold Them There

## Context and Problem Statement

0061-PLAN Phase 4c measured, by running the tests through a proxy that
refuses every request and logs it, that `catalog`'s tests ask the real
models.dev document's host, `models.opencode.ai`, 24 times a run. That phase
added none, and left these to a record of their own, which the owner asked
for on 2026-10-25. Measured across the module, they are not alone.

A test that reaches the network passes or fails by the state of the network:
with it, the real document is fetched and read; without it, the fetch fails
and its layer is empty. Both runs pass today, so the result does not yet
depend on it, but nothing holds that, and a run sends the host a request it
did not ask for.

### Evidence, measured 2026-10-25 at `dccf266`

* **Every package, each alone, through the proxy** (`HTTPS_PROXY` and
  `HTTP_PROXY` naming a local proxy that answers 502 and logs the target):
  `llmprovider/catalog` 24, `providers/claude` 18, `providers/gemini` 18,
  `providers/grok` 19, `providers/openai` 20, every other package 0. Every
  request is `CONNECT models.opencode.ai:443`, the default document URL
  (`defaultModelMetadataURL`).
* **Which tests,** by a scratch clone that logs the caller's stack wherever a
  fetch of the default URL is started or made (`startModelMetadata`,
  `cachedMetadata`, `loadModelMetadata`):
  * `catalog`: tests that list through `listT` after `resetLookupCaches`,
    which calls `enableModelMetadata`, so `listT` leaves the document on
    though the test serves none: `TestList_Cached_0041`,
    `TestList_StaleOn503_0041`, `TestList_LastGood_0041`,
    `TestList_SourceAlways_0041`, `TestList_CachedRebuildsWithProfile_0041`,
    `TestList_NoStaleOn401_0041`, `TestList_StaleExpires_0041`,
    `TestList_DefaultIsFullList`, `TestWithLimit_FullList`,
    `TestWithLimit_SixIsToday`, `TestListRaw_SameTransportAsList_0041`; and
    `TestList_GivesASessionItsClient`, which calls `List` directly.
  * `claude`, `gemini`, `grok`, `openai`: their listing tests, through each
    package's `build`, which builds the provider with no document option:
    `TestListModels`, `TestListModels_CarriesTheCallersIdentity`,
    `TestListModels_ListingBounded`, `TestListModels_ProbeBodyIsTheOldProbes`,
    `TestProbeCache_ProviderProbesIntoItsCache_0049` (`_0048` on `openai`),
    and, on `openai`, `TestListModels_FullListProbesSix`,
    `TestListModels_SecondListingSendsNoProbes` and two `TestProbeCache_`
    tests; on `grok`, `TestProviderClient_SharedWithListingAndRefresh`.
* **No test in these six packages dials an outside host on purpose:** every
  request logged was the document's.

## Decision Drivers

* A test's result must not depend on the network, and a test run must send
  nothing off the machine (AGENTS.md: live tests are opt-in).
* A fix that only edits today's call sites lets the next test bring the
  leak back unseen: the leaks above were found by hand.
* A guard must fail loudly, naming the host, not refuse quietly.
* No change to the SDK's behaviour or API.

## Considered Options

* **A. The call sites, and a guard:** each test that lists or looks up
  models serves its own document or turns it off; and an internal test
  helper runs each of these packages' tests behind a proxy that refuses every
  request off the machine and fails the run, naming each host.
* **B. The call sites, and a tool:** the same fixes; and a script in `tools/`
  that runs the tests through the proxy, run by `tools/gate.py` but not by
  CI.
* **C. The call sites alone.**

## Decision Outcome

Chosen option: "A" (owner, 2026-10-25), because it is the only option that
holds the count at zero in CI, and it fails at the test run that adds a
leak, with the host named.

### D1. The call sites

* `catalog`'s `listT` turns the document off unless `enableModelMetadata`
  is on **and** the options name a document URL
  (`llmprovider.ResolveOptions(...).ModelMetadataURL()` is not empty): a test
  that resets the caches but serves no document lists with it off.
* `TestList_GivesASessionItsClient` passes `llmprovider.WithoutModelMetadata()`.
* `claude`, `gemini`, `grok` and `openai`'s test `build` adds
  `llmprovider.WithoutModelMetadata()` when the options name no document URL,
  by the same test.

### D2. The guard: `internal/localonly`

* A package `internal/localonly`, test support only, with `Main(m
  interface{ Run() int }) int`, which a `TestMain` passes its `*testing.M`
  (amended in execution, 2026-10-25: a `*testing.M` cannot be made in a
  test, and `Main` must be measured for the 80 % floor; PLAN deviation 2): it starts a proxy on `127.0.0.1` that answers every request and
  `CONNECT` with 502 and records its host; sets `HTTPS_PROXY`, `HTTP_PROXY`
  and their lower-case forms to it, and `NO_PROXY` empty; runs `m.Run()`;
  and, when any host was recorded, prints each, once, with the count, and
  returns 1 even when every test passed. Go's `ProxyFromEnvironment` never
  proxies a loopback host, so `httptest` servers are untouched.
* Each of `catalog`, `providers/claude`, `providers/gemini`,
  `providers/grok`, `providers/openai` and `providers/chatcompat` gets a
  `TestMain` that calls it, in a file built without `live_gateways`, so that
  the live tests, which reach real services on purpose, are unguarded.
* **Its limit:** a client whose transport has no `Proxy`, or a dial below
  `net/http`, is not seen. Every request measured above went through the
  SDK's clients, which take the environment's proxy.

### D3. Docs

`AGENTS.md`'s "Tools" or test paragraph names the guard and the rule (a
test serves its own document or turns it off); `docs/architecture.md`'s
package table gains `internal/localonly`.

### Consequences

* Good, because no test run sends a request off the machine in these
  packages, and a new test that would is a failed run naming the host.
* Good, because the SDK does not change.
* Neutral, because six packages gain a `TestMain`; a package added later
  opts in the same way.
* Bad, because the guard sees only requests that honour the environment's
  proxy.

### Confirmation

* Every package, through the logging proxy, 0 requests.
* A planted leak (D1's option removed from one `build`) fails that
  package's run with the guard's message, on a scratch clone.
* The tests that were leaking pass, and their results are unchanged.

## Pros and Cons of the Options

### A. The call sites, and a guard

* Good, because CI holds the count at zero.
* Bad, because of six `TestMain`s and a package.

### B. The call sites, and a tool

* Good, because no test code beyond the call sites.
* Bad, because CI does not run `tools/`; a leak lands, and is found later.

### C. The call sites alone

* Good, because smallest.
* Bad, because the next leak is found by hand, as these were.

## More Information

* [0061-PLAN-consumer-gaps-from-gobble-issue-9.md](0061-PLAN-consumer-gaps-from-gobble-issue-9.md),
  Phase 4c's "outside network" entry, where the 24 were first measured.
* [0065-PLAN-tests-reach-no-outside-host.md](0065-PLAN-tests-reach-no-outside-host.md).
