package llmtest

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

func TestFake_RepliesInOrderAndRecordsRequests(t *testing.T) {
	boom := errors.New("boom")
	f := NewFake("fake", llmprovider.Capabilities{Tools: llmprovider.Supported}).
		ReplyText("first").
		Fail(boom).
		Handle(func(_ context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
			return &llmprovider.Response{Model: req.Model}, nil
		})
	if f.ID() != "fake" || f.Capabilities().Tools != llmprovider.Supported {
		t.Fatalf("ID %q, capabilities %+v", f.ID(), f.Capabilities())
	}
	ctx := context.Background()
	if text, err := llmprovider.GenerateText(ctx, f, &llmprovider.Request{Model: "m1"}); err != nil || text != "first" {
		t.Fatalf("first reply = %q, %v", text, err)
	}
	if _, err := f.Generate(ctx, &llmprovider.Request{Model: "m2"}); !errors.Is(err, boom) {
		t.Fatalf("second reply err = %v, want boom", err)
	}
	if resp, err := f.Generate(ctx, &llmprovider.Request{Model: "m3"}); err != nil || resp.Model != "m3" {
		t.Fatalf("third reply = %+v, %v", resp, err)
	}
	if _, err := f.Generate(ctx, &llmprovider.Request{Model: "m4"}); !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Fatalf("exhausted script err = %v, want ErrProviderUnavailable", err)
	}
	var models []string
	for _, req := range f.Requests() {
		models = append(models, req.Model)
	}
	if len(models) != 4 || models[0] != "m1" || models[3] != "m4" {
		t.Fatalf("recorded models %q, want m1..m4", models)
	}
}

func TestFake_RefusesWhatItCannotDo(t *testing.T) {
	f := NewFake("fake", llmprovider.Capabilities{}).ReplyText("never")
	req := &llmprovider.Request{Tools: []llmprovider.Tool{{Name: "t"}}}
	if _, err := f.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) {
		t.Fatalf("tools on a Fake without them: err = %v, want ErrUnsupported", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.Generate(ctx, &llmprovider.Request{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context: err = %v, want context.Canceled", err)
	}
	if n := len(f.Requests()); n != 0 {
		t.Fatalf("%d requests recorded, want none: refused requests are not accepted", n)
	}
}

func TestFake_RecordsCopies(t *testing.T) {
	reply := &llmprovider.Response{Output: []llmprovider.Item{llmprovider.MessageItem{Text: "a"}}}
	f := NewFake("fake", llmprovider.Capabilities{Tools: llmprovider.Supported, Reasoning: llmprovider.Supported}).Reply(reply)
	req := &llmprovider.Request{
		Input:     []llmprovider.Item{llmprovider.MessageItem{Text: "in"}},
		Tools:     []llmprovider.Tool{{Name: "t"}},
		Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortLow},
	}
	resp, err := f.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Output[0] = llmprovider.MessageItem{Text: "changed"}
	req.Input[0] = llmprovider.MessageItem{Text: "changed"}
	req.Tools[0].Name = "changed"
	req.Reasoning.Effort = llmprovider.EffortHigh
	got := f.Requests()[0]
	if got.Input[0].(llmprovider.MessageItem).Text != "in" || got.Tools[0].Name != "t" || got.Reasoning.Effort != llmprovider.EffortLow {
		t.Fatalf("a caller's change reached the recorded request: %+v", got)
	}
	if reply.Output[0].(llmprovider.MessageItem).Text != "a" {
		t.Fatal("a caller's change to the response reached the scripted reply")
	}
}

func TestFake_ConcurrentUse(t *testing.T) {
	f := NewFake("fake", llmprovider.Capabilities{})
	const n = 16
	for range n {
		f.ReplyText("x")
	}
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			if _, err := f.Generate(context.Background(), &llmprovider.Request{}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if got := len(f.Requests()); got != n {
		t.Fatalf("%d requests recorded, want %d", got, n)
	}
}
