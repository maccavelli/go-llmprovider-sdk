package llmprovider

// What stays in llmprovider of the Ollama provider, which moved to
// providers/ollama (0015-PLAN S7): the descriptor and the listing use the
// base URL.

const ollamaBaseURL = "http://localhost:11434"

// wireShapesProbedOnOllama is the date the Ollama wire shapes were measured
// against a running instance (v0.31.1): GET /v1/models returns the OpenAI list
// shape, and POST /v1/chat/completions answers 200 with NO Authorization header.
const wireShapesProbedOnOllama = "2026-08-29"
