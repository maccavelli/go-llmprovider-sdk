package wire

import (
	"bytes"
	"encoding/json"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Finish maps a service's finish reason through table (0021-MADR W3). A value
// the table does not name is kept as the service sent it, as Response's
// contract says. A reply that carries a call and would finish "stop" finishes
// "tool_calls", as every wire reports a call.
func Finish(raw string, table map[string]llmprovider.FinishReason, hasCall bool) llmprovider.FinishReason {
	finish, ok := table[raw]
	if !ok {
		finish = llmprovider.FinishReason(raw)
	}
	if hasCall && finish == llmprovider.FinishStop {
		return llmprovider.FinishToolCalls
	}
	return finish
}

// EmptyAnswer is an answer with nothing usable in it: ErrIncomplete, with the
// service's finish reason in Reason, so a refusal or a safety stop is not
// mistaken for a plain empty answer (0021-MADR W3). where names the wire.
func EmptyAnswer(where string, reason llmprovider.FinishReason) error {
	return &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: string(reason),
		Message: where + ": the answer has no content"}
}

// CompactArguments is a call's JSON arguments as a string, kept exactly: not
// decoded and re-encoded, so a large integer keeps its digits and the keys
// their order (0021-MADR W1). Empty arguments, or null, are "{}".
func CompactArguments(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return "{}", nil
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return "", err
	}
	return buf.String(), nil
}
