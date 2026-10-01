package llmprovider

// The dates the gateways' wire shapes were measured on. The raw live probes
// in live_gateways_test.go check them, and TestWireShapesProbedOn keeps them
// valid dates. The providers are in providers/, and the listing in catalog
// (0015-PLAN S7, S8).
// Re-validate with: go test -tags live_gateways ./llmprovider/ -run Live

// wireShapesProbedOnKilo is the date Kilo's wire shapes were measured against
// the live gateway: the message.reasoning spelling (not reasoning_content),
// supported_parameters as a per-model capability list, pricing.completion as
// a string with "-1" for variable-priced tiers, and /chat/completions
// answering free models with no credential.
const wireShapesProbedOnKilo = "2026-08-29"

// wireShapesProbedOnHuggingFace is the date the router's wire shapes were
// measured: the /v1/models metadata field names that drive curation
// (throughput, first_token_latency_ms, supports_tools,
// architecture.output_modalities), and the /v1/responses
// HTTP-200-with-status:"failed" behaviour that is the reason the provider does
// not use it.
const wireShapesProbedOnHuggingFace = "2026-08-29"

// wireShapesProbedOnOllama is the date the Ollama wire shapes were measured
// against a running instance (v0.31.1): GET /v1/models returns the OpenAI list
// shape, and POST /v1/chat/completions answers 200 with NO Authorization header.
const wireShapesProbedOnOllama = "2026-08-29"
