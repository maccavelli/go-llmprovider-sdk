---
status: accepted
date: 2026-10-13
decision-makers: repository owner
consulted: the SDK's catalog, provider and wizard code at `21186d0`; gobble-cli and prepare-commit-msg, each a local checkout (below)
informed: every caller of `catalog.List`, a provider's `ListModels`, or `wizard.ConfigureLLM`
---

<!-- markdownlint-disable MD013 -->

# List the Full Curated Catalog by Default, and Six Only on Request

## Context and Problem Statement

A listing today recommends at most six models per provider. The owner's
request (2026-10-13):

> i need to make a way for consumers to get the full, curated model catalogs
> per provider rather than the 6 model limit. the 6 model limit should be
> opt-in, with the default being all curated models in a catalog. gooble-cli
> needs to get the full provider catalog since it is an agent harness,
> whereas prepare-commit-msg should only get the top 6 models per provider.

### 1. Where the six is enforced

`catalog.MaxListed` is `6` (`llmprovider/catalog/models_catalog.go:26-28`,
"the hard cap for configure-time model menus"). Every path that recommends
models stops at it:

* **The open catalogs' ranking.** `rankRecommended`
  (`llmprovider/catalog/model_ranking.go:248-284`) takes the eligible
  candidates in profile order, at most `maxPerRankGroup` (2) per group, then
  the fill order, and breaks at `MaxListed` in both loops. Kilo
  (`discovery.go:978-993`), OpenCode Zen and Go, Hugging Face and Together
  (`model_metadata.go:683-703`) rank through it.
* **The vendor catalogs' curation.** `curateFromCatalog`
  (`models_catalog.go:423-481`) takes the static list's ids that the listing
  serves, then backfills from the rest by rank, and stops at `MaxListed`.
  Gemini, OpenAI, Claude and Grok curate through it, as do the open
  catalogs' fallbacks.
* **Ollama.** `ollamaCatalog` recommends the first `MaxListed` installed
  models (`discovery.go:460-467`).
* **The probes.** Each probing provider's `ListModels` (OpenAI with an API
  key, Claude, Gemini, Grok, Ollama) passes `catalog.MaxListed` as the probe
  limit (for example `providers/gemini/gemini.go:298`). Every probe is a
  billed generation (`llmprovider/options.go:65-73`).
* **The wizard.** A ChatGPT session's listing and a third party's
  `ListModels` are cut to `catalog.MaxListed` recommended
  (`wizard/configure.go:481-484`, `:509-512`). A built-in provider's comes
  from `catalog.List`, already cut.

What a listing returns beyond the six is `Catalog.Usable`
(`discovery.go:58-77`): every id the usability filters admit, in the
listing's order, unranked, and including the ids the ranking excludes. A
provider's `ListModels` (`llmprovider/contract.go:24-30`) returns
`Recommended` only, so a caller that uses a provider rather than `catalog`
cannot reach the rest at all.

The static lists (`catalog.Static`, `Descriptor.StaticModels`,
`llmprovider/registry.go:31-32`) hold at most six each. They are the
fallback when a listing fails, and 0033-MADR-public-model-metadata-and-current-catalogs.md
keeps the ranked ones equal to a recorded snapshot's six.

### 2. How the two consumers use the list

**gobble-cli** (a local checkout at `ad6ea57`; it requires the SDK at
`v1.2.1`) is an agent harness:

* It never calls `catalog` or `wizard`. `llm/provider/ambient.go:71-78`
  (`Models`) returns `slices.Clone(d.StaticModels)`, and `Choose` takes
  `d.StaticModels[0]` (`:28-35`).
* `acpserver/config.go:41-42` publishes that list, uncut, as the ACP `model`
  select option. `:70` refuses any model outside it ("unknown model"). The
  picker, `/model` and `set_config_option` are therefore limited to the
  static six.
* Its own records already plan to list through the SDK:
  `docs/reports/0014-REPORT-llmprovider-sdk-integration.md:317-321` says that
  configuring and `gobble models` "go through `wizard` and `catalog` via
  `llm/provider`", and `:393` says "Do not implement a gobble-owned model
  catalog".

**prepare-commit-msg** (a local checkout at `f1c3523`; it requires the SDK at
`v1.3.2`) writes commit messages:

* The interactive path is `wizard.ConfigureLLM`, with `Discover: true` and
  `NeedFallbacks: true` (`internal/ui/setup.go:231-243`). The SDK's menu is
  what the user sees.
* The non-interactive `configure --yes` path calls `catalog.List`
  (`setup.go:154`). It takes the static catalog when `Recommended` is empty
  (`:161-164`, its 0009-MADR D3), then `models[0]` as the primary
  (`:347-353`), and at most `config.MaxFallbacks` (3) of the rest as
  fallbacks (`:176-187`).
* `internal/ui/setup_test.go:70-71` asserts the menu: "6 models, then 7
  Other".

No other repository on this host requires this module.

### 3. The owner's choices (2026-10-13)

Asked with the evidence above, the owner chose:

* **Source:** the live listing. The static lists stay short. gobble-cli moves
  to `catalog.List` in its own repository.
* **Membership:** the curated list, all of it. The full list is the
  ranking with no cap: every known model that works, and none that the
  ranking leaves out. In review the owner replaced the first answer, "every
  usable model", with: "the full list being the curate list, instead of all
  models, the full live catalog of known, working models, but all of them"
  (2026-10-13).
* **The wizard:** it follows the option. Its menu shows the full list by
  default. A caller that passes the limit gets six.
* **Probes:** only the first six are probed. The rest are returned unprobed.

## Decision Drivers

* **The default is the full catalog;** the six is opt-in (the owner).
* **The six a caller opts into is exactly today's six,** in content and in
  order, empty case included. prepare-commit-msg's `--yes` fallback
  (0009-MADR D3 there) and 0021-MADR C13's "a live listing that curates to
  nothing keeps `Recommended` empty" depend on it. So do the snapshot tests of
  0033-PLAN-refresh-the-curated-catalogs.md.
* **The short list is a prefix of the full one,** so a caller that cuts the
  full list itself gets the same six.
* **One option reaches every path:** `catalog.List`, each provider's
  `ListModels`, and the wizard. It is carried the way `catalog.WithProfile`
  is already carried (`catalog/config.go:66-81`).
* **No new billed requests:** a full list must not mean a probe per model.
* **The compatibility promise** (R48): additions only. No signature changes.
* **The static lists and their snapshot rule stay as decided** by
  0033-MADR-public-model-metadata-and-current-catalogs.md.

## Considered Options

* **A.** `Recommended` holds the full curated catalog by default, and
  `catalog.WithLimit(n)` cuts it.
* **B.** `Recommended` stays at six, and a new `Catalog` field carries the
  full ranked list.
* **C.** A second listing function, such as `catalog.ListAll`, returns the
  full list, and `List` is unchanged.
* **D.** As A, and each static list also grows to the provider's full ranked
  list, recorded from each snapshot.

## Decision Outcome

Chosen option: "A", because it alone makes the full catalog the default, as
the owner asked. It reaches a provider's `ListModels` and the wizard through
one option. It adds one exported function and changes no signature.

### D1. The default: the full curated catalog

With no limit, a live listing's `Recommended` is the provider's ranking with
no cap: the curated list, all of it.

* **The open catalogs:** `rankRecommended` without its cap. First come the
  eligible candidates in profile order, at most `maxPerRankGroup` per group.
  Then comes the fill order: the rest of the usable ids, in the curation's
  order.
* **The vendor catalogs:** `curateFromCatalog` without its cap. First come
  the static list's served ids, in static order, then the rest of the usable
  ids by rank.
* **Ollama:** the installed models, in listing order.

**What the full list leaves out.** It holds every *known, working* model:

* **Known:** the listing serves the id and the usability filters admit it.
  On Together, models.dev must list it too (`listedOnly`, 0009-MADR §3 item
  10).
* **Working:** the id is not measured to fail at request
  (`excludedAtRequest`: `-contributor` ids, `opencodeGoRegionGated`,
  `togetherUnserved`).
* **The profile:** under the utility profile, the profile's exclusions
  (`utilityExcluded`: `kilo-auto/*`, dense models of 14B or less) are left
  out too, as from today's six. Under `ProfileCapable` there are none.

The ids left out stay in `Usable`, so search still finds them. `Usable`
itself does not change.

**The fallback curation applies "working" too.** Today `curateFromCatalog`
does not apply `excludedAtRequest`. It is used by the vendor catalogs, and
by the open catalogs when the metadata fetch is off or fails. Within six,
the static ids come first, so an unservable id rarely appears. With no cap,
every usable id would. So `excludedAtRequest` applies on every path. On
Together, with no metadata, "known" falls back to the listing's own
usability filter, as the ranking already does.

### D2. The opt-in: `catalog.WithLimit(n int) llmprovider.Option`

* **Scope:** every built-in provider, as `WithProfile` is
  (`llmprovider.ScopedOptionFor(builtinIDs, …)`). One option list carrying
  it is valid for every provider, and the providers that never rank still
  honor its cut.
* **With `n > 0`,** `Recommended` is D1's list cut at `n`.
  `WithLimit(catalog.MaxListed)` is today's `Recommended`, with one
  exception: on the fallback curation, the six no longer include an id
  measured not to answer (D1). Without a cap the list is empty in exactly
  the cases it is empty today, so 0021-MADR C13 and prepare-commit-msg's
  fallback hold with or without the limit.
* **With `n ≤ 0`,** there is no limit, as with no option: the zero value is
  the default, as `WithProfile`'s is.
* **The prefix:** a limited `Recommended` is always a prefix of the
  unlimited one. The ranking gives the same order with or without the cap,
  because the cap only stops its loops early.

### D3. The static lists are unchanged

`catalog.Static`, `Descriptor.StaticModels` and the fallback after a failed
listing still hold at most `MaxListed`, as 0033-MADR's rule keeps them. The
limit cuts a static fallback like any other list. With `Live` false,
`Usable` still equals `Recommended`. Option D is not taken.

### D4. A provider's `ListModels` returns the full list, and probes six

* Every built-in provider's `ListModels` returns its listing's
  `Recommended` under the caller's options. So it is the full list by
  default, and cut when the caller passed `catalog.WithLimit` to `New`. It
  reaches the listing through the options the provider already carries.
* **Probes** (OpenAI with an API key, Claude, Gemini, Grok, Ollama) send at
  most `MaxListed` billed requests per listing, as today, for the first
  `min(MaxListed, len(list))` ids. A probed id that fails is dropped. Every
  later id is returned unprobed, and the list keeps its order. When every
  probe fails, the list is returned unchanged, as today. A caller that wants
  no probes passes `WithModelProbes(false)`, as gobble-cli does.

### D5. The wizard follows the option

* The wizard's listing already carries `Options.ProviderOptions` into
  `catalog.List` (`wizard/configure.go:418-424`), so a built-in provider's
  menu is the full list by default, and six when the caller passes
  `catalog.WithLimit(catalog.MaxListed)`.
* A ChatGPT session's listing and a third party's `ListModels` follow the
  same option. Today they are always cut at `MaxListed`
  (`configure.go:483`, `:511`).
* Search still covers `Usable`.
* The wizard reads the limit without a new exported accessor. How is a
  PLAN detail.

### D6. `MaxListed` keeps its name and value

Its doc changes from "the hard cap for configure-time model menus" to what
it now is:

* the length of the short list (`WithLimit(MaxListed)`);
* the bound on the static lists;
* the bound on probes per listing.

### D7. The release

* **Version:** an exported addition (`WithLimit`) and a changed default, so
  `v1.11.0`.
* **Release notes:** they name the default change first, and give the
  opt-in a caller adds to keep today's six.
* **Consumers, in their own repositories:** prepare-commit-msg adds
  `catalog.WithLimit(catalog.MaxListed)` to its listing options and its
  wizard `ProviderOptions` when it bumps. gobble-cli moves from
  `StaticModels` to `catalog.List` (or `ListModels`) with no limit. Neither
  change is part of this record's PLAN.

### Consequences

* Good, because an agent harness reaches every known, working model of a
  provider, ranked, through the function or method it already uses.
* Good, because `WithLimit(MaxListed)` is today's behavior, apart from D1's
  fallback exclusion, and a test can hold it to that.
* Good, because an empty `Recommended` still means "nothing curates", with
  or without the limit. So 0021-MADR C13, the wizard's "no %s model meets
  the profile" notice (`wizard/model_select.go:50-56`) and
  prepare-commit-msg's fallback to the static catalog keep their meaning.
* Good, because the short list is a prefix of the full one, so the six best
  stay first in every menu.
* Good, because no probe is added: a listing is billed as today.
* Neutral, because the static lists, `Usable`, `Search` and the ranker's
  rules do not change.
* Bad, because the default changes under every caller that reads
  `Recommended` as a short list. Until it opts in, it gets a long one. For
  Kilo, that is the 285 ids the 2026-10-12 snapshot counts as usable, less
  the ranking's exclusions. A wizard caller's menu grows to match. The only
  external callers are the owner's two, and the release notes say so.
* Neutral, because prepare-commit-msg's `--yes` path takes the same primary
  and fallbacks with or without the opt-in: they are the first four of the
  prefix. Its wizard menu is what the opt-in keeps at six.
* Bad, because a working model can be missing from the full list. It is
  missing when the utility profile excludes it, or when Together's
  models.dev section does not list it. An agent harness that wants the
  profile's exclusions back lists with `ProfileCapable`. Search over
  `Usable` still finds every id.
* Bad, because "working" covers only what was measured: ids on the
  `excludedAtRequest` lists. A model that fails for another reason, such as
  account access, can still be listed, as it can in today's six.
* Bad, because ids after the sixth are never probed. A caller that relies on
  probes gets them only for the head of the list.

### Confirmation

* **Unit tests, per provider path (open catalogs, vendors, Ollama):**
  * with `WithLimit(MaxListed)`, `Recommended` equals today's, the empty
    case included;
  * with no limit, `Recommended` is `Usable` minus exactly D1's exclusions,
    with no id twice;
  * no path, the fallback curation included, recommends an
    `excludedAtRequest` id;
  * the limited list is a prefix of the unlimited one;
  * `n ≤ 0` behaves as no option.
* **The snapshot tests** (`snapshot_20261012_test.go`) pass the limit and
  keep their meaning (R46). An unlimited replay of the snapshot pins each
  ranked list's full length, measured in the PLAN. Each is at most its
  usable count: Kilo 285, Zen 79, Go 34, Hugging Face 119, Together 172.
* **`ListModels`:**
  * returns the full list by default and is cut by the option;
  * a probe test counts at most `MaxListed` generations for a list longer
    than six;
  * the unprobed tail is kept, in order.
* **The wizard:** a menu test shows the full list by default and six with
  the option, for a built-in listing, a ChatGPT session and a third-party
  `ListModels`.
* **Each new assertion is seen failing on a plant** (`tools/plant.py`), and
  the FAIL lines are recorded.
* **`tools/apidiff.py v1.10.2`** reports only the addition of
  `catalog.WithLimit`.

## Amendment 2026-10-13: what the PLAN's research found

The owner accepted this record on 2026-10-13. Researching
0038-PLAN-full-curated-catalog-by-default.md then found two facts that this
record states wrongly, and one path it does not name. The decisions above
stand, except where these change them. Their text is not edited.

### A1. The full list is built as today's six, then the rest

D2 says the prefix holds "because the cap only stops its loops early". That
is true of `curateFromCatalog`, cut at six. It is not true of
`rankRecommended`:

* Its fill order begins with the fallback curation, which is itself cut at
  six: `append(fallback(usable), usable...)` (`model_metadata.go:702`), and
  `append(curateKilo(usable), usable...)` (`discovery.go:992`).
* Uncapped, that fallback holds every usable id. So once its first six are
  used up, an uncapped fill continues in a different order from today's.

So the full list is built as two parts:

1. today's six, computed exactly as today;
2. then the ranking with no cap, in its order, without the six's ids.

The prefix and "`WithLimit(MaxListed)` is today's six" then hold by
construction, on every path. D1's membership does not change.

### A2. The wizard takes the limit in its own field

D5 says a caller passes the limit in `Options.ProviderOptions`. That cannot
work for every provider:

* `Options.ProviderOptions` also go to the provider the wizard builds for a
  third party's listing, and for a ChatGPT session
  (`wizard/configure.go:530`).
* `catalog.WithLimit` is scoped to the built-in ids, like `WithProfile`. A
  scoped option is refused by the `New` of every provider outside its scope
  (`llmprovider/settings.go:138-145`; R18).

So the wizard takes the limit as a field, `Options.ModelLimit int`, the way
it takes `Options.Profile` (`configure.go:407-410`):

* It passes `catalog.WithLimit(o.ModelLimit)` to `catalog.List`.
* It cuts a ChatGPT session's listing, a third party's `ListModels`, and its
  static fallback to the same number.
* Zero shows the whole list (D5's default).

This is a second exported addition. `tools/apidiff.py` must report exactly
two: `catalog.WithLimit` and the `wizard.Options.ModelLimit` field.

### A3. A provider reads the limit itself, for its own fallback

A provider's `ListModels` falls back to `catalog.Static` when
`catalog.List` fails or recommends nothing, outside the listing (for
example `providers/openai/openai.go:346-349`). A ChatGPT session's listing
never calls `catalog.List` (`openai.go:341-345`). So each built-in
provider's `New` reads the limit from its options, and cuts both lists.

The option's value type lives in a module-internal package,
`llmprovider/internal/listlimit`, which `catalog` and the providers share.
No accessor is exported.

## Amendment 2026-10-13 (A4): the full list's order, and the menu

0041-MADR-decide-per-model-behaviour-from-metadata.md, proposed on
2026-10-13, orders every provider's live menu by facts. It changes this
record's full list in four ways. D1 to D7 and A1 to A3 stand except where
this says otherwise. Their text is not edited. This amendment takes effect
when 0041-MADR is accepted, and is implemented by
0041-PLAN-decide-per-model-behaviour-from-metadata.md, Phase 4. The owner
accepted 0041-MADR on 2026-10-13, so A4 is in effect; until that phase
lands, the code still builds the full list as A1 describes.

### A4.1. The full list's order

D1 orders the full list as the ranking with no cap. A1 builds it as today's
six, then the rest of that ranking. After this amendment:

1. **The six** are 0041-MADR D4's recommended band, computed on its own, as
   A1 requires.
2. **The rest** are the other eligible models, newest first by release
   date (0041-MADR D1), then in listing order, then by id. They are no
   longer in the name scorers' order, or the uncapped ranking's.

The prefix property of D2 and A1 still holds by construction:
`WithLimit(MaxListed)` is the band, and `WithLimit(n)` is a prefix of the
full list.

### A4.2. What the full list leaves out

D1's "known, working" list also leaves out every model with a 0041-MADR D3
caveat:

* deprecated;
* no tools;
* no text output;
* expiring within 30 days.

A model that fails at request was already left out (D1). None of these is
dropped from the catalog. They are in the new `Catalog.Menu`, after the
full list, with their caveats, and in `Usable`, as today.

### A4.3. The menu

`Catalog` gains `Menu`, every usable model in 0041-MADR D3's order, with
its facts, rank and caveats; and `Source` and `FetchedAt` (0041-MADR D6).
`Recommended` keeps its type and is a prefix of `Menu`'s ids.

### A4.4. D3, "the static lists are unchanged"

The static lists are still at most six, and still the floor. But each
ranked list must equal the latest snapshot's utility six
(`TestStatic_EqualsLatestSnapshot`), so they change when 0041-MADR D4
changes the ranking. 0041-PLAN Phase 3 updates them.

## Pros and Cons of the Options

### A. Full by default, `catalog.WithLimit(n)` to cut

* Good, because it is the owner's default.
* Good, because one option reaches `catalog.List`, `ListModels` and the
  wizard, as `WithProfile` does.
* Good, because it adds one exported function.
* Bad, because existing callers' default changes. See Consequences.

### B. Keep `Recommended` at six; add a full field to `Catalog`

* Good, because no caller's behavior changes.
* Bad, because the default stays six, against the owner's decision.
* Bad, because `ListModels` returns `[]string`. A provider caller would
  still get six unless the interface also changed, which R48 forbids.

### C. A second listing function

* Good, because `List` is unchanged.
* Bad, because the default stays six, against the owner's decision.
* Bad, because it does not reach `ListModels` or the wizard, so each would
  need its own variant.

### D. A and full static lists

* Good, because the full list is offline, and gobble-cli's `StaticModels`
  read would need no change.
* Bad, because it commits hundreds of ids (Kilo alone has 285 usable), and
  they go stale between snapshots.
* Bad, because the live guard (`TestLive_StaticCatalogsCurrent`) would have
  to check them all.
* Bad, because the owner chose the live listing.

## More Information

* **Records this one builds on:**
  * 0009-MADR-use-case-aware-default-model-ranking.md (the ranking and its
    six);
  * 0021-MADR C13 (an empty `Recommended` on a live listing);
  * 0016-MADR A5 (probes are billed, and on by default);
  * 0033-MADR-public-model-metadata-and-current-catalogs.md (the static lists
    and the snapshots).
* **Not changed:** `Catalog`'s fields, the `ModelLister` interface,
  `Descriptor`, `Static`, `Search`, and the ranker's eligibility rules.
* **Consumers' follow-up, outside this repository:**
  * prepare-commit-msg adds the opt-in;
  * gobble-cli lists through `catalog` and widens its ACP model option to
    the listed models.

  Each needs its own record there.
