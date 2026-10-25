// Package catalog knows the models each provider offers, and what is known of
// each one.
//
// It holds the curated static catalogs (Static, Label, Rank), the live
// listing curated against them (List, Catalog), the same listing as the
// provider sent it (ListRaw, RawListing), model search (Search, and
// Catalog.Search, which searches a listing's facts too), the ranking profiles
// (Profile), and model metadata.
//
// Search ranks in tiers, an exact id first, and inside a tier by BM25F over
// a model's fields (its id, its label, its family, and its facts as words
// such as vision, tools or 128k), then by fzf's subsequence score, then by
// the listing's order. A typo within an edit or two, and a query only some
// of whose words match, are fallbacks, tried only when no tier matches
// (0061-MADR, amendment "Model search, rebuilt").
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
