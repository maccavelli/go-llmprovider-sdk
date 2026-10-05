package responses

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The Responses wire at its own level (0015-PLAN S7b); the provider packages
// test it through Generate.

func TestInput(t *testing.T) {
	got, err := json.Marshal(Input([]llmprovider.Item{
		llmprovider.MessageItem{Role: "user", Text: "weather?"},
		llmprovider.FunctionCallItem{CallID: "c1", Name: "get_weather", Arguments: `{"city":"Paris"}`},
		llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "sunny"},
		llmprovider.ReasoningItem{Text: "not sent"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"content":"weather?","role":"user"},` +
		`{"arguments":"{\"city\":\"Paris\"}","call_id":"c1","name":"get_weather","type":"function_call"},` +
		`{"call_id":"c1","output":"sunny","type":"function_call_output"}]`
	if string(got) != want {
		t.Errorf("Input = %s\nwant    %s", got, want)
	}
}

// TestDecode: each output type becomes its item; output_text and text both
// count as text, an empty message is dropped, and a bad body is an error.
func TestDecode(t *testing.T) {
	body := `{"id":"r1","status":"completed","output":[
		{"type":"reasoning","summary":[{"type":"summary_text","text":"think"},{"type":"text","text":"ing"},{"type":"other","text":"no"}]},
		{"type":"message","content":[{"type":"output_text","text":"hel"},{"type":"text","text":"lo"},{"type":"refusal","text":"no"}]},
		{"type":"message","content":[]},
		{"type":"function_call","call_id":"c1","name":"get_weather","arguments":"{}"},
		{"type":"web_search_call"}]}`
	res, err := Decode(strings.NewReader(body))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := []llmprovider.Item{
		llmprovider.ReasoningItem{Text: "thinking", Format: "responses"},
		llmprovider.MessageItem{Role: "assistant", Text: "hello"},
		llmprovider.FunctionCallItem{CallID: "c1", Name: "get_weather", Arguments: "{}"},
	}
	if res.ID != "r1" || len(res.Output) != len(want) {
		t.Fatalf("Decode = %+v, want id r1 and %v", res, want)
	}
	for i := range want {
		if res.Output[i] != want[i] {
			t.Errorf("output[%d] = %#v, want %#v", i, res.Output[i], want[i])
		}
	}
	if _, err := Decode(strings.NewReader("not json")); err == nil {
		t.Error("a bad body decoded")
	}
	var incompleteErr *llmprovider.APIError
	if _, err := Decode(strings.NewReader(`{"status":"incomplete"}`)); !errors.As(err, &incompleteErr) ||
		!errors.Is(incompleteErr.Kind, llmprovider.ErrIncomplete) || incompleteErr.Reason != "unspecified" {
		t.Errorf("incomplete with no reason: %v, want reason unspecified", err)
	}
}

// sse frames events as the ChatGPT backend does.
func sse(events ...string) string {
	var b strings.Builder
	for _, e := range events {
		b.WriteString("event: x\ndata: " + e + "\n\n")
	}
	return b.String()
}

// TestReadStream: items arrive in output_item.done, the id in created and
// completed; failed, incomplete and an early end are errors.
func TestReadStream(t *testing.T) {
	t.Run("completed", func(t *testing.T) {
		res, err := ReadStream("p", strings.NewReader(sse(
			`{"type":"response.created","response":{"id":"r0"}}`,
			`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"hi"}]}}`,
			`{"type":"response.in_progress"}`,
			`{"type":"response.completed","response":{"id":"r1"}}`,
		)))
		if err != nil || res.ID != "r1" || res.OutputText() != "hi" {
			t.Fatalf("ReadStream = %+v, %v; want r1 saying hi", res, err)
		}
	})
	t.Run("completed with no id keeps created's", func(t *testing.T) {
		res, err := ReadStream("p", strings.NewReader(sse(
			`{"type":"response.created","response":{"id":"r0"}}`, `{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"ok"}]}}`,
			`{"type":"response.completed","response":{}}`)))
		if err != nil || res.ID != "r0" {
			t.Fatalf("ReadStream = %+v, %v; want r0", res, err)
		}
	})
	t.Run("data split over lines, and a final event with no blank line", func(t *testing.T) {
		res, err := ReadStream("p", strings.NewReader(sse(`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"ok"}]}}`)+"data: {\"type\":\"response.completed\",\ndata: \"response\":{\"id\":\"r2\"}}"))
		if err != nil || res.ID != "r2" {
			t.Fatalf("ReadStream = %+v, %v; want r2", res, err)
		}
	})
	t.Run("[DONE] is skipped", func(t *testing.T) {
		if _, err := ReadStream("p", strings.NewReader("data: [DONE]\n\n")); !errors.Is(err, llmprovider.ErrProviderUnavailable) {
			t.Fatalf("err = %v, want the early-end error", err)
		}
	})
	for _, tc := range []struct {
		name, event string
		want        error
	}{
		{"failed with an error", `{"type":"response.failed","response":{"error":{"code":"context_length_exceeded","message":"too long"}}}`, llmprovider.ErrContextOverflow},
		{"failed without one", `{"type":"response.failed","response":{}}`, llmprovider.ErrProviderUnavailable},
		{"incomplete", `{"type":"response.incomplete","response":{"incomplete_details":{"reason":"max_output_tokens"}}}`, llmprovider.ErrIncomplete},
		{"not JSON", `{not json`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadStream("p", strings.NewReader(sse(tc.event)))
			if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
	t.Run("ended early", func(t *testing.T) {
		_, err := ReadStream("p", strings.NewReader(sse(`{"type":"response.created","response":{"id":"r0"}}`)))
		if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !strings.Contains(err.Error(), "before response.completed") {
			t.Fatalf("err = %v, want a retryable early end", err)
		}
	})
	t.Run("a read error", func(t *testing.T) {
		_, err := ReadStream("p", errReader{})
		if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !strings.Contains(err.Error(), "read stream") {
			t.Fatalf("err = %v, want a retryable read error", err)
		}
	})
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }
