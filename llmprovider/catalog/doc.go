// Package catalog knows the models each provider offers, and what is known of
// each one.
//
// It holds the curated static catalogs (Static, Label, Rank), the live
// listing curated against them (List, Catalog), model search (Search), the
// ranking profiles (Profile), and model metadata.
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
