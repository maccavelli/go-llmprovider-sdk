package wire

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D7: a Call's Warnings reach the reply's Response, by Post and by
// a stream's done event, and a call with none adds nothing.

func TestCallWarnings_0040(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("data: done\n\n"))
	}))
	defer srv.Close()
	warn := llmprovider.Warning{Kind: llmprovider.WarningDeferralEmulated, Model: "m", Detail: "d"}
	call := func(ws []llmprovider.Warning) Call {
		return Call{Provider: "p", Client: srv.Client(), URL: srv.URL, Body: map[string]any{}, Warnings: ws,
			Prepare: func(*http.Request, llmprovider.Token) {}}
	}
	decode := func(io.Reader) (*llmprovider.Response, error) {
		return &llmprovider.Response{Warnings: []llmprovider.Warning{{Kind: "earlier"}}}, nil
	}
	resp, err := Post(context.Background(), call([]llmprovider.Warning{warn}), llmprovider.Token{}, decode)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Warnings) != 2 || resp.Warnings[1] != warn {
		t.Errorf("Post: warnings %+v; want the decoder's, then the call's", resp.Warnings)
	}
	resp, err = Post(context.Background(), call(nil), llmprovider.Token{}, decode)
	if err != nil || len(resp.Warnings) != 1 {
		t.Errorf("Post with no call warnings: %+v, %v; want the decoder's only", resp, err)
	}

	events := func(_ string, body io.Reader, yield func(llmprovider.Event) bool) error {
		_, _ = io.Copy(io.Discard, body)
		yield(llmprovider.Event{Type: llmprovider.EventTextDelta, Text: "x"})
		yield(llmprovider.Event{Type: llmprovider.EventDone, Response: &llmprovider.Response{}})
		return nil
	}
	var done *llmprovider.Response
	for ev, err := range StreamEvents(context.Background(), call([]llmprovider.Warning{warn}), llmprovider.NewStaticToken("k"), events) {
		if err != nil {
			t.Fatal(err)
		}
		if ev.Type == llmprovider.EventDone {
			done = ev.Response
		}
		if ev.Type == llmprovider.EventTextDelta && strings.TrimSpace(ev.Text) == "" {
			t.Error("the text event was changed")
		}
	}
	if done == nil || len(done.Warnings) != 1 || done.Warnings[0] != warn {
		t.Errorf("stream: done %+v; want the call's warning", done)
	}
}
