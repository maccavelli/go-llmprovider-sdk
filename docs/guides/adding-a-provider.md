# Adding a provider

How to write a provider for this module's contract. A provider can live in your
own module, which needs no change here, or in this module as a built-in. Both
start the same way, and the [last section](#a-built-in-provider) lists what a
built-in adds.

The rules cited as R*n* are in [api-standards.md](api-standards.md). A provider
that breaks one fails review here, and most of them also fail `llmtest.Run`.

## What a provider is

A value that implements `llmprovider.Provider`:

```go
type Provider interface {
    ID() ProviderID
    Capabilities() Capabilities
    Generate(ctx context.Context, req *Request) (*Response, error)
}
```

It may also implement `llmprovider.ModelLister` (`ListModels`) and
`llmprovider.Streamer` (`Stream`). Callers find both by a type assertion;
`wizard` uses `ListModels` to offer your models.
Without `Streamer`, `llmprovider.Stream` emits `Generate`'s result as events.

The packages you use, all under `github.com/maccavelli/go-llmprovider-sdk`:

| Import | For |
| :--- | :--- |
| `…/llmprovider` | the contract, options, errors, `Registry` and `Descriptor` |
| `…/llmprovider/auth` | the session types, to refuse one your service cannot take |
| `…/llmprovider/llmtest` | `Run` and `Harness`, in your tests |
| `…/llmprovider/providers` | `Default()`, the registry of the built-in providers |
| `…/wizard` | `ConfigureLLM`, `Options` and `Prompter` |

`go doc` on each gives the field types this guide does not repeat. For
example, `go doc …/llmprovider Tool` shows that `Schema` is an `any`, a JSON
Schema value you encode as it is.

## 1. Choose an id

An id is a `llmprovider.ProviderID`, a string. Use your service's
models.dev key where it has one (R22), in lower case, and export it:

```go
// ID is the Acme provider's id.
const ID llmprovider.ProviderID = "acme"
```

`Registry.Register` refuses an id the registry already holds, so check it
against `providers.Default()` if you add yours to that registry.

## 2. Write `New`

`New` takes options and nothing else. It makes no network call and takes no
context (R14, R40). It reads everything from the `Settings` that
`llmprovider.ResolveOptions` returns (R19):

```go
// New builds the Acme provider. It needs a key.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
    st, err := llmprovider.ResolveOptions(ID, opts)
    if err != nil {
        return nil, err
    }
    src := st.TokenSource()
    if src == nil {
        return nil, fmt.Errorf("%w: acme needs WithAPIKey or WithTokenSource", llmprovider.ErrInvalidRequest)
    }
    baseURL := defaultBaseURL
    if b := st.BaseURL(); b != "" {
        baseURL = strings.TrimRight(b, "/")
    }
    return &provider{
        src:       src,
        model:     st.Model(),
        baseURL:   baseURL,
        client:    st.HTTPClient(),
        userAgent: st.UserAgent(),
        logger:    st.Logger(),
        maxTokens: st.MaxTokens(),
    }, nil
}
```

- **`ResolveOptions` refuses** an option scoped to another provider, and a
  malformed `For`, with `ErrInvalidRequest` (R18). Return its error as it is.
- **What `Settings` gives you:** `Model`, `BaseURL`, `HTTPClient`,
  `TokenSource`, `UserAgent`, `Logger`, `MaxTokens`, `Reasoning`,
  `SessionID`, `ModelProbes`, `ModelMetadataURL`, `ModelMetadataDisabled`,
  and `Values` for your own options.
  - `HTTPClient` is never nil. Without `WithHTTPClient` it is the module's
    default client, which honours the proxy variables.
  - `Logger` is never nil. Without `WithLogger` it discards everything, so log
    to it freely and never to `slog`'s global functions (R31).
- **Refuse a credential your service cannot take** (R16;
  [0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md)
  D2). For example, a key-only service returns an error matching
  `ErrUnsupported` for a `*auth.OAuthSession`.
- **Keep the result immutable.** Everything `New` reads is fixed for the
  provider's life, and `Generate` may run on many goroutines at once (R20).

### Options of your own

Wrap `llmprovider.ScopedOption` around an unexported type, and read it back
from `Settings.Values` in `New`. Another provider's `New` refuses the option
unless the caller scoped it with `For(ID, …)` (R17, R18):

```go
type regionOption string

// WithRegion selects Acme's region.
func WithRegion(region string) llmprovider.Option {
    return llmprovider.ScopedOption(ID, "acme.WithRegion", regionOption(region))
}

// in New:
for _, v := range st.Values() {
    if r, ok := v.(regionOption); ok {
        p.region = string(r)
    }
}
```

## 3. Declare the capabilities

`Capabilities` gives each of `Tools`, `ForcedToolChoice`, `Reasoning`,
`Continuation` and `NativeStreaming` as `llmprovider.Unsupported`,
`BestEffort` or `Supported` (R10). The zero value is `Unsupported`.

- `BestEffort` means you send it and the service may ignore it. List every
  `BestEffort` degradation in the package doc, under "Degradations:" (R12).
- Declare `NativeStreaming` `Supported` only if you implement `Streamer`.

## 4. Write `Generate`

In this order:

1. **Check the request first:** `if err := p.caps.Check(req); err != nil {
   return nil, err }`. An unsupported need fails with `ErrUnsupported` and an
   invalid value with `ErrInvalidRequest`, before anything is sent (R11, R23).
2. **Encode it** in your service's format:
   - `req.Model`, or the model from `New` when it is empty;
   - `req.Instructions`;
   - `req.Input`, a switch on the `Item` types, with a `default` case that
     returns an error (R9);
   - `req.Tools`, and `req.ToolChoice`, whose `Tool()` says whether one tool
     is forced;
   - `req.Reasoning`, `req.MaxOutputTokens` (or `MaxTokens` from `New`), and
     `req.PreviousResponseID` when you support `Continuation`.
3. **Build the request with the context:** `http.NewRequestWithContext(ctx,
   …)`. That is what stops it when the caller cancels (R40).
4. **Set the credential and the identity.** Call `p.src.Token(ctx)`, then
   `token.Apply(httpReq, header, scheme)` with your service's own header and
   scheme: `"Authorization"` and `"Bearer"`, or `"x-api-key"` and `""` (R16;
   [0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) D2, A6).
   Set `User-Agent` to the `UserAgent` from `New`. `llmtest` checks it (R44).
5. **Send it** with the client from `New`. Close the body, and log a failed
   close to the logger from `New`.
6. **Classify a failure:** `if err := llmprovider.ClassifyHTTPError(string(ID),
   resp); err != nil { return nil, err }`. It returns nil for 200, and
   otherwise an error of the right kind (R24, R25), with a redacted,
   bounded message (R35).
7. **After a 401,** if the source is an `llmprovider.InvalidatingSource`,
   close the reply, call `Invalidate`, and send the request once more. Build
   it again, with a new body reader and a fresh `Token`: the first send has
   read the body. A built-in provider does this with
   `wire.Reauth(p.src, send)` from `llmprovider/internal/wire`, around a
   `generateOnce` that fetches its token on every call. `llmtest` checks
   it (`R16-reauth`; 0020-MADR F2).
8. **Decode the reply** into a `*llmprovider.Response`. Bound what you read,
   with `io.LimitReader`.
   - `Output` holds `MessageItem{Role: llmprovider.RoleAssistant, Text: …}`,
     `FunctionCallItem{CallID, Name, Arguments}` and `ReasoningItem` values,
     in the order the service sent them.
   - Set `ID`, `Model` and `FinishReason` when the service reports them.
   - In `Usage` a total holds its part: `InputTokens` includes
     `CachedTokens`, and `OutputTokens` includes `ReasoningTokens` (R7).
9. **Wrap your own errors with your prefix and a kind** where one applies,
   such as `fmt.Errorf("acme: decode reply: %w", err)` or
   `fmt.Errorf("acme: no output: %w", llmprovider.ErrIncomplete)` (R25, R27).

## 5. Write the descriptor

A `Descriptor` is what a configuration UI, such as `wizard`, shows for the
provider. It carries no key material (R34):

```go
// Descriptor is the Acme descriptor.
func Descriptor() llmprovider.Descriptor {
    return llmprovider.Descriptor{
        ID:              ID,
        Label:           "Acme",
        EnvVar:          "ACME_API_KEY",
        DefaultBaseURL:  defaultBaseURL,
        SupportsBaseURL: true,
        RequiresAPIKey:  true,
        StaticModels:    []string{"acme-large", "acme-small"},
    }
}
```

- **The models `wizard` offers.** With `Options.Discover`, it lists a provider
  outside this module through its own `ListModels`, when the provider is an
  `llmprovider.ModelLister`. It builds the provider from the registry with
  the credential, the base URL, `Options.HTTPClient` and
  `Options.ProviderOptions`. The first six listed ids are recommended, in
  your order, and search covers the whole listing.
- **`StaticModels` is the fallback:** without `Options.Discover`, without
  `ListModels`, or when it fails or lists nothing. In those last cases the
  wizard also warns.
- With no `AuthMethods`, or only `AuthAPIKey`, `wizard` asks for the key. Its
  other methods run the built-in providers' sign-ins, so do not list them
  for a provider of your own.

## 6. Register it

A `Registry` holds `Descriptor` and `Factory` pairs. There is no global one
(R36, R37):

```go
reg := providers.Default() // every built-in provider; llmprovider.NewRegistry() for none
if err := reg.Register(acme.Descriptor(), acme.New); err != nil {
    return err
}
p, err := reg.New(acme.ID, llmprovider.WithAPIKey(key), llmprovider.WithModel("acme-large"))
```

`providers.New(id, …)` builds only the built-in providers. Use your registry's
`New` for yours.

## 7. Offer it in the wizard

`wizard` offers what its `Options.Registry` holds (R39):

```go
res, err := wizard.ConfigureLLM(ctx, prompter, wizard.Options{
    Registry:  reg,
    Providers: []llmprovider.ProviderID{acme.ID}, // optional: show only these
})
```

`ConfigureLLM` asks through the `Prompter` you pass: `Select`, `MultiSelect`,
`Confirm`, `Input`, `Secret` and `Notify` (`go doc …/wizard Prompter`). In a
test, drive it with a `Prompter` that answers from a script and fails on a
prompt it did not expect. For a provider with the descriptor above, it asks:

1. `Select` the provider. The menu is the registry's order: `Register`
   order, after the built-ins of `providers.Default()`.
2. `Input` the endpoint, with `DefaultBaseURL` as the default, when
   `SupportsBaseURL` is set.
3. The key. With `Options.AllowEnv` and a set `EnvVar`, `Confirm` using it;
   with an `Options.Existing` key for this provider, `Confirm` keeping it;
   otherwise `Secret`.
4. `Input` a model search, where blank means the recommended list, then
   `Select` the model from it. With no `StaticModels` and no listing, it asks
   instead to `Input` a model id.
5. `MultiSelect` the fallbacks, only with `Options.NeedFallbacks`.

`Notify` reports problems, such as a listing that failed. The `Result` names
the provider, model and key. Build the provider from it with your registry's
`New`.

## 8. Pass `llmtest`

`llmtest.Run` checks a provider against the contract (R44). It starts a fake
server for each check, and builds your provider pointed at it:

```go
func TestConformance(t *testing.T) {
    llmtest.Run(t, llmtest.Harness{
        New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
            return acme.New(append([]llmprovider.Option{
                llmprovider.WithAPIKey("test-key"),
                llmprovider.WithModel("acme-large"),
                llmprovider.WithBaseURL(baseURL),
            }, opts...)...)
        },
        Text:     func(w http.ResponseWriter, r *http.Request) { /* a success with some text */ },
        ToolCall: func(w http.ResponseWriter, r *http.Request, tool string) { /* a call to tool with {} */ },
        Error:    func(w http.ResponseWriter, r *http.Request, status int) { /* an error reply */ },
    })
}
```

- `New` must apply `opts` after its own: the checks add options, such as
  `WithClientInfo`, and expect them to win.
- `Text`, `ToolCall` and `Error` write replies in your service's format.
  `ToolCall` is needed unless `Tools` is `Unsupported`.
- Run it with `go test -race`. The concurrency check means something only
  under the race detector (R20).

Each failure names the rule it breaks.

## A built-in provider

A provider in this module is all of the above, and:

- **A decision record first.** A new provider is a decision: write the MADR
  and PLAN that `AGENTS.md` describes, as
  [0017-MADR](../decisions/0017-MADR-together-provider-and-auth-extensions.md)
  did for Together AI.
- **The id** is a `ProviderID` constant in `llmprovider/provider.go`, with
  its key variable in `ProviderEnvVars()` when it takes one.
- **The package** is `llmprovider/providers/<id>`. It has `New` and
  `Descriptor()`, and a package doc that names the endpoints, the
  credentials it takes, and the degradations (R12).
- **The wire.** A format that more than one provider speaks comes from
  `llmprovider/internal/wire/<format>`, not a copy (R3).
- **The registry.** Add one entry to the `builtins` list in
  `llmprovider/providers/providers.go`, in menu order (R38).
- **The catalog.** `catalog.List` and `catalog.Static` learn the id, so the
  wizard can list its models.
- **The tests:**
  - `llmtest.Run` in the package (R44);
  - G-wire cases through `llmprovider/internal/wirecase`, whose goldens are
    recorded once with `-update` and cite the record that adds them (R45);
  - a live test behind the `live_gateways` tag and its own
    `LLMPROVIDER_LIVE_<ID>` variable (`AGENTS.md`, "Live tests");
  - 80 % coverage from the package's own tests (R47).
- **The docs:** `docs/architecture.md` (the tree, the package table and the
  provider list) in the same change (R43).
