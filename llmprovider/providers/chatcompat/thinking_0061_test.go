package chatcompat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// thinkingRefuser is a server that refuses each chat request carrying a
// reasoning_effort with Ollama's recorded refusal of thinking (0061-MADR,
// amendment "Ollama's refusal of thinking, learned"), answers every other
// with wirecase's canned reply, and keeps each chat request's
// reasoning_effort, "" for none.
func thinkingRefuser(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "internal", "wire", "testdata", "refusals-0061", "chatcompletions",
		"ollama-gemma3-12b-max.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	var (
		mu   sync.Mutex
		sent []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			reply := wirecase.Case{}.Reply(r)
			_, _ = io.WriteString(w, reply.Body)
			return
		}
		var body struct {
			Effort string `json:"reasoning_effort"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		sent = append(sent, body.Effort)
		mu.Unlock()
		if body.Effort != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(fixture.Status)
			_, _ = io.WriteString(w, fixture.Body)
			return
		}
		_, _ = io.WriteString(w, wirecase.ChatReply)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), sent...)
	}
}

// TestLearn_NoThinking_0061 (0061-MADR, amendment "Ollama's refusal of
// thinking, learned"): pointed at an Ollama server, chatcompat learns a
// model's refusal of thinking on the first request, which is sent again with
// no effort, warned, and every later request to it sends none, in one send:
// the matcher is the Chat Completions wire's, which chatcompat shares.
func TestLearn_NoThinking_0061(t *testing.T) {
	srv, sent := thinkingRefuser(t)
	p := build(t, srv.URL, llmprovider.WithModel("gemma3:12b"))
	ask := func(effort llmprovider.Effort) *llmprovider.Response {
		t.Helper()
		resp, err := p.Generate(context.Background(), &llmprovider.Request{Input: text("hi").Input,
			Reasoning: &llmprovider.Reasoning{Effort: effort}})
		if err != nil {
			t.Fatalf("at %s: %v", effort, err)
		}
		return resp
	}
	first := ask(llmprovider.EffortMax)
	if got := strings.Join(sent(), ","); got != "max," {
		t.Fatalf("efforts sent = %q; want max, then none", got)
	}
	var learned, degraded int
	for _, w := range first.Warnings {
		switch {
		case w.Kind == llmprovider.WarningReasoningLearned &&
			w.Refused == (llmprovider.ReasoningRefusal{Model: "gemma3:12b", Effort: llmprovider.EffortMax}):
			learned++
		case w.Kind == llmprovider.WarningEffortDegraded && strings.Contains(w.Detail, "sent none"):
			degraded++
		}
	}
	if learned != 1 || degraded != 1 || len(first.Warnings) != 2 {
		t.Errorf("Warnings = %+v; want the refusal learned and the effort sent as none", first.Warnings)
	}
	second := ask(llmprovider.EffortHigh)
	if got := strings.Join(sent(), ","); got != "max,," {
		t.Errorf("efforts sent = %q; want the second request in one send, with none", got)
	}
	if len(second.Warnings) != 1 || second.Warnings[0].Kind != llmprovider.WarningEffortDegraded {
		t.Errorf("the second request's Warnings = %+v; want one WarningEffortDegraded", second.Warnings)
	}
}
