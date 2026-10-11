package messages

import (
	"encoding/json"
	"strings"
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

// resultJSON is v as JSON.
func resultJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestIsError_0032 (0032-MADR D3): a failed result is sent with tool_result's is_error; one not
// failed sends today's form.
func TestIsError_0032(t *testing.T) {
	failedJSON, okJSON := resultJSON(t, FromItems(turnWithResult(true))), resultJSON(t, FromItems(turnWithResult(false)))
	if !strings.Contains(failedJSON, `"is_error":true`) {
		t.Errorf("a failed result: %s; want %s", failedJSON, `"is_error":true`)
	}
	if strings.Contains(okJSON, `"is_error"`) || !strings.Contains(okJSON, `"tool_result"`) {
		t.Errorf("a result not failed: %s; want %s and no %s", okJSON, `"tool_result"`, `"is_error"`)
	}
}
