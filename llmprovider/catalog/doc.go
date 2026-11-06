// Package catalog knows the models each provider offers, and what is known of
// each one.
//
// It holds the curated static catalogs (Static, Label, LabelName, LabelNote,
// Rank, SortByRank), the live listing curated against them (List, Catalog),
// the same listing as the provider sent it (ListRaw, RawListing), model
// search (Search, and Catalog.Search, which searches a listing's facts too),
// the ranking profiles (Profile), and model metadata.
//
// Rank reads a family's two-digit minor, such as grok-4.20's, as an integer
// or as a decimal, chosen from the family's release dates, else from the
// versions its siblings fill, else as an integer: so grok-4.20 is 4.2, below
// grok-4.3, and gemini-3.10 still follows gemini-3.9. Rank reads a model
// beside its provider's static list; SortByRank reads a whole list, and the
// release dates its caller has (0070-MADR D1).
//
// Search ranks in tiers, an exact id first, and inside a tier by BM25F over
// a model's fields (its id, its label, its family, and its facts as words
// such as vision, tools or 128k), then by fzf's subsequence score, then by
// the listing's order. A typo within an edit or two, and a query only some
// of whose words match, are fallbacks, tried only when no tier matches
// (0061-MADR, amendment "Model search, rebuilt").
//
// A chatcompat provider's listing (List, with chatcompat.WithID) reads what
// the server says of its models where its /models reply is a recorded
// dialect's, Ollama's, and models.dev's facts of each model's canonical id,
// an exact match only. A listing with no ranking, a chatcompat server's or
// Ollama's, is ordered by the default rank, each model's variants together
// (0061-MADR E2, E4; its amendment "E2 and E4's menu, made exact before
// Phase 4c").
//
// ListRaw sends List's requests, through the same code, and returns every
// page byte for byte with each entry of its model array: no filter, ranking,
// caveat, static fallback or models.dev fetch, and nothing written to the
// listing cache. A failed listing is its error (0041-MADR D9).
//
// Model metadata comes from a models.dev-format document (Metadata), fetched
// and cached in-process (LookupMetadata, CachedMetadataWith), or read from one
// a caller persisted (ParseMetadata, Metadata.Encode). ModelInfo is one
// model's limits, modalities, capabilities, prices in USD per million tokens,
// and lifecycle. Each field reports whether it is known (Has) and which layer
// set it (SourceOf) (0033-MADR D1 to D3).
//
// models.dev's values are its own: the SDK corrects none of them, and some
// were found wrong when 0033-MADR was written. Examples are Together prices
// that differ from Together's docs, Hugging Face prices that are one
// provider's rather than the router's, a Kilo promotion priced as the list
// price, Zen deprecations left unmarked, and an output limit above its
// context window.
package catalog
