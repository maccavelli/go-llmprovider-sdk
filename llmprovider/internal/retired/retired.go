// Package retired holds the ids each provider's deprecations page, or its
// listing, has marked retired, scheduled for shutdown, routed to another
// model, or open only to past users, each with why (0033-MADR D4, and its
// amendment "sources that lag").
//
// It is a leaf: catalog's tests and the maintainer tool both read it
// (0033-PLAN-public-model-metadata.md Phase 4).
package retired

import "github.com/maccavelli/go-llmprovider-sdk/llmprovider"

// IDs are the retired ids by provider, each with why, as the deprecations
// pages said on 2026-10-11 (0033-PLAN-refresh-the-curated-catalogs.md
// Phase 1).
var IDs = map[llmprovider.ProviderID]map[string]string{
	llmprovider.ProviderOpenAI: {
		"gpt-4.1-nano": "shuts down 2026-10-23",
		"o4-mini":      "shuts down 2026-10-23",
	},
	llmprovider.ProviderGemini: {
		"gemini-3.7-flash":      "routed to gemini-3.8-flash",
		"gemini-3.5-flash":      "routed to gemini-3.6-flash",
		"gemini-2.5-flash":      "open only to past users",
		"gemini-2.5-flash-lite": "open only to past users",
		"gemini-3.1-flash-lite": "shuts down 2027-05-07",
	},
	llmprovider.ProviderClaude: {
		"claude-haiku-4-5":           "retirement floor 2026-10-15",
		"claude-sonnet-4-5-20250929": "retires 2026-11-30",
	},
	llmprovider.ProviderGrok: {
		"grok-4-fast-reasoning": "retired 2026-05-15",
		"grok-4":                "not listed; grok-4-0709 retired 2026-05-15",
		"grok-3-mini":           "not listed",
		"grok-3-mini-fast":      "not listed",
		"grok-3":                "retired 2026-05-15",
	},
}
