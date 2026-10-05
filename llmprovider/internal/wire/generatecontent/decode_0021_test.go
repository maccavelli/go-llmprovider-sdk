package generatecontent

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

func calls(res *llmprovider.Response) []llmprovider.FunctionCallItem {
	var out []llmprovider.FunctionCallItem
	for _, it := range res.Output {
		if call, ok := it.(llmprovider.FunctionCallItem); ok {
			out = append(out, call)
		}
	}
	return out
}

// TestDecode_FunctionArgsKeepPrecision (0021-MADR W1): functionCall.args come
// back exactly.
func TestDecode_FunctionArgsKeepPrecision(t *testing.T) {
	res, err := Decode(strings.NewReader(`{"candidates":[{"finishReason":"STOP","content":{"parts":[` +
		`{"functionCall":{"name":"f","args":{"z":1,"id":12345678901234567890}}}]}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := calls(res); len(got) != 1 || got[0].Arguments != `{"z":1,"id":12345678901234567890}` {
		t.Errorf("calls = %+v, want the args exactly", got)
	}
}

// TestFinish_GenerateContent (0021-MADR W3): an unknown reason is kept, and an
// empty answer keeps its reason.
func TestFinish_GenerateContent(t *testing.T) {
	res, err := Decode(strings.NewReader(`{"candidates":[{"finishReason":"OTHER","content":{"parts":[{"text":"hi"}]}}]}`))
	if err != nil || res.FinishReason != "OTHER" {
		t.Errorf("OTHER: %v, %v; want it kept", res, err)
	}
	var apiErr *llmprovider.APIError
	_, err = Decode(strings.NewReader(`{"candidates":[{"finishReason":"SAFETY","content":{}}]}`))
	if !errors.As(err, &apiErr) || !errors.Is(err, llmprovider.ErrIncomplete) || apiErr.Reason != "content_filter" {
		t.Errorf("SAFETY with no parts: %v; want ErrIncomplete, reason content_filter", err)
	}
}

// TestDecode_GeminiCallIDs (0021-MADR W10): two calls to one function have
// distinct ids: the service's, else name#index; a result for name#index is
// sent under the function's name.
func TestDecode_GeminiCallIDs(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		want []string
	}{
		"ids": {`{"candidates":[{"finishReason":"STOP","content":{"parts":[` +
			`{"functionCall":{"id":"fc_1","name":"weather","args":{"city":"Paris"}}},` +
			`{"functionCall":{"id":"fc_2","name":"weather","args":{"city":"Rome"}}}]}}]}`, []string{"fc_1", "fc_2"}},
		"no ids": {`{"candidates":[{"finishReason":"STOP","content":{"parts":[` +
			`{"functionCall":{"name":"weather","args":{"city":"Paris"}}},` +
			`{"functionCall":{"name":"weather","args":{"city":"Rome"}}}]}}]}`, []string{"weather#0", "weather#1"}},
	} {
		res, err := Decode(strings.NewReader(tc.body))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got := calls(res)
		if len(got) != 2 || got[0].CallID != tc.want[0] || got[1].CallID != tc.want[1] {
			t.Errorf("%s: calls = %+v, want ids %v", name, got, tc.want)
		}
	}
	contents, err := json.Marshal(Contents([]llmprovider.Item{
		llmprovider.FunctionCallOutputItem{CallID: "weather#1", Output: "sunny"},
	}))
	if err != nil || !strings.Contains(string(contents), `"name":"weather"`) {
		t.Errorf("a result for weather#1: %s; want it sent as weather", contents)
	}
}
