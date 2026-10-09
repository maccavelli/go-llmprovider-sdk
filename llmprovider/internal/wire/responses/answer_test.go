package responses

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestInput_EmptyRoleIsTheUsers (0020-MADR F10): an empty Role is the user's
// (contract.go), so it goes out as "user", never "".
func TestInput_EmptyRoleIsTheUsers(t *testing.T) {
	got := Input([]llmprovider.Item{llmprovider.MessageItem{Text: "hi"}})
	if len(got) != 1 {
		t.Fatalf("Input = %v, want one item", got)
	}
	if r := got[0].Role; r == nil || *r != "user" { // typed (0028-PLAN D14, Phase 9b; D16)
		t.Errorf("Input = %v, want role user", got)
	}
}

// TestDecode_ReportsModelAndFinish (0020-MADR F11): the model that answered,
// and why it stopped.
func TestDecode_ReportsModelAndFinish(t *testing.T) {
	for _, c := range []struct {
		name, body string
		finish     llmprovider.FinishReason
	}{
		{"text", `{"id":"r","model":"gpt-served","status":"completed","output":[
			{"type":"message","content":[{"type":"output_text","text":"hi"}]}]}`, llmprovider.FinishStop},
		{"call", `{"id":"r","model":"gpt-served","status":"completed","output":[
			{"type":"function_call","call_id":"c","name":"f","arguments":"{}"}]}`, llmprovider.FinishToolCalls},
	} {
		res, err := Decode(strings.NewReader(c.body))
		if err != nil || res.Model != "gpt-served" || res.FinishReason != c.finish {
			t.Errorf("%s: Decode = %+v, %v; want model gpt-served, finish %q", c.name, res, err, c.finish)
		}
	}
	stream := "data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"hi\"}]}}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"r\",\"model\":\"gpt-served\",\"status\":\"completed\"}}\n\n"
	res, err := ReadStream("openai", strings.NewReader(stream))
	if err != nil || res.Model != "gpt-served" || res.FinishReason != llmprovider.FinishStop {
		t.Errorf("ReadStream = %+v, %v; want model gpt-served, finish stop", res, err)
	}
}

// TestReadStream_ErrorEventIsClassified (0020-MADR F39): a stream's top-level
// error event keeps its code, so exhausted quota is not a retryable "stream
// ended".
func TestReadStream_ErrorEventIsClassified(t *testing.T) {
	stream := "data: {\"type\":\"error\",\"code\":\"insufficient_quota\",\"message\":\"You exceeded your current quota\"}\n\n"
	_, err := ReadStream("openai", strings.NewReader(stream))
	if !errors.Is(err, llmprovider.ErrQuotaExhausted) {
		t.Errorf("ReadStream = %v, want ErrQuotaExhausted", err)
	}
}

// TestReasoning_EncryptedIsKeptAndReplayed (0020-MADR F24): an encrypted
// reasoning item decodes into Encrypted and goes back in the next input; an
// empty one is not kept.
func TestReasoning_EncryptedIsKeptAndReplayed(t *testing.T) {
	res, err := Decode(strings.NewReader(`{"id":"r","status":"completed","output":[
		{"type":"reasoning","id":"rs_1","encrypted_content":"ENC","summary":[]},
		{"type":"reasoning","summary":[]},
		{"type":"message","content":[{"type":"output_text","text":"hi"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []llmprovider.Item{llmprovider.ReasoningItem{Encrypted: "ENC", Format: "responses"},
		llmprovider.MessageItem{Role: "assistant", Text: "hi"}}
	if !reflect.DeepEqual(res.Output, want) {
		t.Errorf("Output = %#v, want %#v", res.Output, want)
	}
	input := Input(res.Output[:1])
	if len(input) != 1 {
		t.Fatalf("replayed input = %v, want one item", input)
	}
	if r := input[0]; r.Type == nil || *r.Type != "reasoning" || r.EncryptedContent == nil || *r.EncryptedContent != "ENC" { // typed (0028-PLAN D14, Phase 9b; D16)
		t.Errorf("replayed input = %v, want the reasoning item with its encrypted_content", input)
	}
}
