//go:build live_gateways

package llmprovider_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// interactionTap keeps a copy of each reply body, up to 4 MiB, for the
// measurement to read its events; the SDK reads the reply as it would.
type interactionTap struct {
	mu   sync.Mutex
	body []byte
	code int
}

func (tap *interactionTap) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	_ = resp.Body.Close()
	if err != nil {
		return nil, errors.New("interactions: the reply could not be read")
	}
	tap.mu.Lock()
	tap.body, tap.code = b, resp.StatusCode
	tap.mu.Unlock()
	resp.Body = io.NopCloser(bytes.NewReader(b))
	return resp, nil
}

// interactionShape is the allowlisted shape of one Interactions stream:
// event types and counts, step types, field names, booleans. No text.
type interactionShape struct {
	events            []string // event_type values, in order, runs collapsed as type*n
	completedSteps    bool     // interaction.completed's interaction carries "steps"
	completedStepN    int
	completedTypes    []string
	completedKeys     []string // the field names of interaction.completed's interaction
	repeatsText       bool     // its model_output text equals the streamed text deltas joined
	textBeforeStart   int      // text deltas for an index no step.start had opened
	argDeltas         int      // arguments_delta deltas
	argNonEmpty       int      // of them, a piece that is not empty, "null" or "{}"
	argEmptyObject    int      // of them, exactly "{}"
	deltaTypes        []string // step.delta delta types seen
	stepStartTypes    []string
	streamedTextBytes int
}

func (s interactionShape) String() string {
	return fmt.Sprintf("events=%s step_start=%s delta_types=%s completed_keys=%s completed_has_steps=%t completed_steps=%d "+
		"completed_step_types=%s completed_repeats_text=%t text_before_start=%d arguments_delta=%d non_empty=%d empty_object=%d streamed_text_bytes=%d",
		strings.Join(s.events, ","), strings.Join(s.stepStartTypes, ","), strings.Join(s.deltaTypes, ","),
		strings.Join(s.completedKeys, ","), s.completedSteps, s.completedStepN, strings.Join(s.completedTypes, ","),
		s.repeatsText, s.textBeforeStart, s.argDeltas, s.argNonEmpty, s.argEmptyObject, s.streamedTextBytes)
}

// readInteractionShape reads body's data payloads into their shape.
func readInteractionShape(body []byte) interactionShape {
	var s interactionShape
	var streamed strings.Builder
	opened := map[int]bool{}
	deltaTypes, startTypes := map[string]bool{}, map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for sc.Scan() {
		data, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "data:")
		if !ok {
			continue
		}
		var ev struct {
			EventType string `json:"event_type"`
			Index     int    `json:"index"`
			Step      struct {
				Type string `json:"type"`
			} `json:"step"`
			Delta struct {
				Type      string          `json:"type"`
				Text      string          `json:"text"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"delta"`
			Interaction json.RawMessage `json:"interaction"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(data)), &ev) != nil {
			s.events = append(s.events, "(undecodable)")
			continue
		}
		if n := len(s.events); n > 0 && strings.Split(s.events[n-1], "*")[0] == ev.EventType {
			var count int
			_, _ = fmt.Sscanf(strings.TrimPrefix(s.events[n-1], ev.EventType+"*"), "%d", &count)
			s.events[n-1] = fmt.Sprintf("%s*%d", ev.EventType, max(count, 1)+1)
		} else {
			s.events = append(s.events, ev.EventType)
		}
		switch ev.EventType {
		case "step.start":
			opened[ev.Index] = true
			startTypes[ev.Step.Type] = true
		case "step.delta":
			deltaTypes[ev.Delta.Type] = true
			switch ev.Delta.Type {
			case "text":
				if !opened[ev.Index] {
					s.textBeforeStart++
				}
				streamed.WriteString(ev.Delta.Text)
			case "arguments_delta":
				s.argDeltas++
				var piece string
				if json.Unmarshal(ev.Delta.Arguments, &piece) != nil {
					piece = string(ev.Delta.Arguments)
				}
				switch strings.TrimSpace(piece) {
				case "", "null":
				case "{}":
					s.argEmptyObject++
				default:
					s.argNonEmpty++
				}
			}
		case "interaction.completed":
			var fields map[string]json.RawMessage
			_ = json.Unmarshal(ev.Interaction, &fields)
			for k := range fields {
				s.completedKeys = append(s.completedKeys, k)
			}
			sort.Strings(s.completedKeys)
			raw, has := fields["steps"]
			s.completedSteps = has
			var steps []struct {
				Type    string `json:"type"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			}
			_ = json.Unmarshal(raw, &steps)
			s.completedStepN = len(steps)
			var final strings.Builder
			for _, st := range steps {
				s.completedTypes = append(s.completedTypes, st.Type)
				if st.Type == "model_output" {
					for _, c := range st.Content {
						final.WriteString(c.Text)
					}
				}
			}
			s.repeatsText = final.Len() > 0 && final.String() == streamed.String()
		}
	}
	for k := range deltaTypes {
		s.deltaTypes = append(s.deltaTypes, k)
	}
	for k := range startTypes {
		s.stepStartTypes = append(s.stepStartTypes, k)
	}
	slices.Sort(s.deltaTypes)
	slices.Sort(s.stepStartTypes)
	s.streamedTextBytes = streamed.Len()
	return s
}

// TestLive_InteractionsShape_0095 is 0095-MADR FU23, run on the owner's word:
// two streamed calls to the gemini provider on the Interactions API,
// gemini-3.8-flash, each capped at 1,024 output tokens. (1) A plain prompt:
// whether interaction.completed carries steps and repeats the streamed text
// (0091-MADR ST72), and whether a text delta arrives before its step.start
// (0092-MADR M11). (2) One tool that takes no arguments, forced: whether an
// arguments_delta arrives for its call, and with what (0088-MADR ST110), and
// whether the completed interaction carries the call step (0092-MADR M10).
// It runs only with LLMPROVIDER_LIVE_0095 set and GEMINI_API_KEY in the
// environment (read by name, never printed). It logs only the allowlisted
// shape: event types and counts, step and delta types, field names, counts
// and booleans, never text. One attempt per call, no retry: a failure is
// logged and fails, with a fixed text.
func TestLive_InteractionsShape_0095(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0095") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0095 to run 0095-MADR FU23's two Interactions calls")
	}
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	now := llmprovider.Tool{Name: "now", Description: "Returns the current time. Takes no arguments.",
		Schema: map[string]any{"type": "object", "properties": map[string]any{}}}
	for _, tc := range []struct {
		name string
		req  *llmprovider.Request
	}{
		{"plain", &llmprovider.Request{Input: []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Reply with one short sentence about the sea."}}}},
		{"no-argument tool, forced", &llmprovider.Request{Tools: []llmprovider.Tool{now}, ToolChoice: llmprovider.ForceTool("now"),
			Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "What time is it? Use the tool."}}}},
	} {
		tap := &interactionTap{}
		p, err := providers.New(llmprovider.ProviderGemini, llmprovider.WithAPIKey(key), llmprovider.WithModel("gemini-3.8-flash"),
			llmprovider.WithMaxTokens(1024), llmprovider.WithModelProbes(false),
			llmprovider.WithHTTPClient(&http.Client{Transport: tap, Timeout: 2 * time.Minute}))
		if err != nil {
			t.Fatalf("%s: the provider could not be built", tc.name)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		var items []string
		var streamErr error
		for ev, err := range llmprovider.Stream(ctx, p, tc.req) {
			if err != nil {
				streamErr = err
				break
			}
			if ev.Type == llmprovider.EventItem {
				items = append(items, fmt.Sprintf("%T", ev.Item))
			}
		}
		cancel()
		tap.mu.Lock()
		body, code := tap.body, tap.code
		tap.mu.Unlock()
		kind := "none"
		if streamErr != nil {
			kind = "error"
			for _, k := range []error{llmprovider.ErrEmptyAnswer, llmprovider.ErrIncomplete, llmprovider.ErrInvalidRequest,
				llmprovider.ErrRateLimited, llmprovider.ErrQuotaExhausted, llmprovider.ErrProviderUnavailable, llmprovider.ErrAuthFailure} {
				if errors.Is(streamErr, k) {
					kind = strings.TrimPrefix(k.Error(), "llmprovider: ")
					break
				}
			}
		}
		t.Logf("0095 FU23 %s: status=%d sdk_error=%s items=%s %s", tc.name, code, kind, strings.Join(items, ","), readInteractionShape(body))
		if code != http.StatusOK || streamErr != nil {
			t.Errorf("%s: HTTP status %d, SDK error kind %s; want 200 and none", tc.name, code, kind)
		}
	}
}
