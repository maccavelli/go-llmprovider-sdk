package responses

import (
	"encoding/json"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// turnWithResult is a call and its result, failed when isError is set.
func turnWithResult(isError bool) []llmprovider.Item {
	return []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"},
		llmprovider.FunctionCallItem{CallID: "call_1", Name: "record_city", Arguments: `{"city":"Paris"}`},
		llmprovider.FunctionCallOutputItem{CallID: "call_1", Output: "offline", IsError: isError},
	}
}

// asJSON is v as JSON.
func asJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestIsError_0032 (0032-MADR D3, Q3): this wire has no error field, so a
// failed result is sent exactly as one not failed.
func TestIsError_0032(t *testing.T) {
	if failedJSON, okJSON := asJSON(t, Input(turnWithResult(true))), asJSON(t, Input(turnWithResult(false))); failedJSON != okJSON {
		t.Errorf("a failed result %s differs from one not failed %s", failedJSON, okJSON)
	}
}
