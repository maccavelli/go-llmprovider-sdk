package llmprovider

// What the listing, the descriptors and the raw live probes need of Hugging
// Face Inference Providers. The provider is providers/huggingface (0015-PLAN
// S7), which keeps its own copy of the base URL.

// huggingFaceBaseURL is the router's OpenAI-compatible base.
const huggingFaceBaseURL = "https://router.huggingface.co/v1"

// wireShapesProbedOnHuggingFace is the date the router's wire shapes were
// measured: the /v1/models metadata field names that drive curation
// (throughput, first_token_latency_ms, supports_tools,
// architecture.output_modalities), and the /v1/responses
// HTTP-200-with-status:"failed" behaviour that is the reason the provider does
// not use it. The live probes in live_gateways_test.go check it.
// Re-validate with: go test -tags live_gateways ./llmprovider/ -run Live
const wireShapesProbedOnHuggingFace = "2026-08-29"
