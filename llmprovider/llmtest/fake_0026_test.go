package llmtest

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestFake_ReplyNilRefused (0026-MADR F58): Reply(nil) is a mistake in the
// test that scripts it, so it panics there, saying so, not later inside
// Generate with a nil dereference.
func TestFake_ReplyNilRefused(t *testing.T) {
	f := NewFake("fake", llmprovider.Capabilities{})
	defer func() {
		got, _ := recover().(string)
		if !strings.Contains(got, "Reply(nil)") {
			t.Fatalf("Reply(nil) panicked with %q; want a message naming Reply(nil)", got)
		}
	}()
	f.Reply(nil)
	t.Fatal("Reply(nil) returned; want it to panic at the call")
}

// TestFake_RequestsCopiesSchema (0026-MADR F58): Requests returns copies, a
// tool's Schema included, so a caller that changes its schema afterwards does
// not change the recorded request.
func TestFake_RequestsCopiesSchema(t *testing.T) {
	f := NewFake("fake", llmprovider.Capabilities{Tools: llmprovider.Supported}).ReplyText("ok")
	schema := map[string]any{"type": "object", "properties": map[string]any{}}
	req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Text: "hi"}},
		Tools: []llmprovider.Tool{{Name: "t", Schema: schema}}}
	if _, err := f.Generate(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	schema["type"] = "CHANGED"
	got := f.Requests()[0].Tools[0].Schema
	want := map[string]any{"type": "object", "properties": map[string]any{}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("recorded schema = %v; want the schema as it was sent, %v", got, want)
	}
}
