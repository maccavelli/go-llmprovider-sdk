package generatecontent

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

// asJSON is v as JSON.
func asJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestIsError_0032 (0032-MADR D3): a failed result is sent as functionResponse.response {"error": …}; one not
// failed sends today's form.
func TestIsError_0032(t *testing.T) {
	failedJSON, okJSON := asJSON(t, Contents(turnWithResult(true))), asJSON(t, Contents(turnWithResult(false)))
	if !strings.Contains(failedJSON, `"response":{"error":"offline"}`) {
		t.Errorf("a failed result: %s; want %s", failedJSON, `"response":{"error":"offline"}`)
	}
	if strings.Contains(okJSON, `"error"`) || !strings.Contains(okJSON, `"response":{"output":"offline"}`) {
		t.Errorf("a result not failed: %s; want %s and no %s", okJSON, `"response":{"output":"offline"}`, `"error"`)
	}
}
