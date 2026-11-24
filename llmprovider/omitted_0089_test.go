package llmprovider

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// This file holds what 0089-PLAN's "OQ1 (1)'s measurement" shares between
// TestLive_OmittedLimit_0089 (live_0089_omitted_test.go, built with
// live_gateways) and its offline twin, TestOmittedLimit_Twin_0089: the
// request that sends no output limit, the stream reader, and the word that
// classifies a finish against the stated limit (0089-MADR DL65). Every error
// text here is fixed; none carries a body, a message or a key.

// omittedPrompt asks for far more output than any stated limit the
// measurement meets (a model's limit is at most 131,072 tokens; counting to
// 400,000 takes several times that), so a model that fills its limit ends in
// "length".
const omittedPrompt = "Write the integers from 1 upward, one per line, as 'N is a number', " +
	"continuing without stopping or summarizing until you are cut off. Do not stop before 400000."

// omittedResult is what one streamed request returns. Out and Reasoning are
// -1 when the stream carried no usage block, or no reasoning count.
type omittedResult struct {
	Status        int
	Finish        string
	Out           int
	Reasoning     int
	StreamErrType string
	StreamErrCode int
}

var omittedErrType = regexp.MustCompile(`^[a-z_]{1,32}$`)

// omittedChunk is the part of one stream event the measurement reads.
type omittedChunk struct {
	Error   json.RawMessage `json:"error"`
	Choices []struct {
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		CompletionTokens int `json:"completion_tokens"`
		Details          *struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage"`
}

// streamOmitted sends one streamed Chat Completions request to base with NO
// output limit, and reads the stream to its end. It sends exactly one request.
func streamOmitted(ctx context.Context, client *http.Client, base, key, model string) (omittedResult, error) {
	res := omittedResult{Out: -1, Reasoning: -1}
	body, err := json.Marshal(map[string]any{
		"model":          model,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
		"messages":       []map[string]any{{"role": "user", "content": omittedPrompt}},
	})
	if err != nil {
		return res, errors.New("omitted limit: the request could not be encoded")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return res, errors.New("omitted limit: the request could not be built")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		return res, errors.New("omitted limit: the request failed in transport")
	}
	defer closeResponseBody(resp)
	res.Status = resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		return res, nil
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		data = strings.TrimSpace(data)
		if data == "[DONE]" {
			continue
		}
		var c omittedChunk
		if json.Unmarshal([]byte(data), &c) != nil {
			continue
		}
		if len(c.Error) > 0 && string(c.Error) != "null" {
			var e struct {
				Type string          `json:"type"`
				Code json.RawMessage `json:"code"`
			}
			res.StreamErrType = "error"
			if json.Unmarshal(c.Error, &e) == nil {
				if omittedErrType.MatchString(e.Type) {
					res.StreamErrType = e.Type
				}
				var code int
				if json.Unmarshal(e.Code, &code) == nil {
					res.StreamErrCode = code
				}
			}
		}
		for _, ch := range c.Choices {
			if ch.FinishReason != nil && *ch.FinishReason != "" {
				res.Finish = *ch.FinishReason
			}
		}
		if c.Usage != nil {
			res.Out = c.Usage.CompletionTokens
			if c.Usage.Details != nil {
				res.Reasoning = c.Usage.Details.ReasoningTokens
			}
		}
	}
	if sc.Err() != nil {
		return res, errors.New("omitted limit: the stream could not be read to its end")
	}
	return res, nil
}

// omittedVerdict classifies one run against the model's stated limit
// (0089-PLAN, "OQ1 (1)'s measurement"): only a "length" finish decides.
// "below": Kilo handed out less than the stated limit (it keeps sending the
// discovered limit); "equal" and "above": it handed out the stated limit or
// more (it omits the field); "undecided": a "stop", no finish, or no usage.
func omittedVerdict(finish string, out, stated int) string {
	if finish != "length" || out < 0 || stated <= 0 {
		return "undecided"
	}
	switch {
	case out < stated:
		return "below"
	case out == stated:
		return "equal"
	default:
		return "above"
	}
}

// omittedUndecided says why a run decided nothing: the finish was not
// "length", or it was and the stream carried no output count (out is -1), or
// the model states no limit, so the two could not be compared.
func omittedUndecided(finish string, out, stated int) string {
	cause := fmt.Sprintf("the finish was %q, not length", finish)
	switch {
	case finish == "length" && out < 0:
		cause = "the length finish came with no usage count, so the output could not be compared"
	case finish == "length":
		cause = "the length finish could not be compared: no stated limit"
	}
	return fmt.Sprintf("undecided, %s (output tokens %d, stated %d): only a length finish with a count decides", cause, out, stated)
}

// omittedLine is the one allowlisted line the measurement keeps per model.
func omittedLine(model string, stated int, r omittedResult, verdict string) string {
	return fmt.Sprintf("model=%s status=%d finish=%q out=%d reasoning=%d stated=%d verdict=%s",
		model, r.Status, r.Finish, r.Out, r.Reasoning, stated, verdict)
}

// TestOmittedLimit_Twin_0089 runs the measurement's request, reader and
// verdict word against a fake Kilo: the request carries no limit field under
// any name, the stream's finish and counts are read, an in-stream error is
// read as its type and numeric code, and the verdict follows the rule.
func TestOmittedLimit_Twin_0089(t *testing.T) {
	var sentBody map[string]any
	serve := func(stream string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			sentBody = nil
			_ = json.Unmarshal(raw, &sentBody)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, stream)
		}))
	}
	const head = "data: {\"choices\":[{\"delta\":{\"content\":\"1\"},\"finish_reason\":null}]}\n\n"
	cases := []struct {
		name    string
		stream  string
		stated  int
		finish  string
		out     int
		reason  int
		errType string
		errCode int
		verdict string
	}{
		{"length below", head + "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}]}\n\n" +
			"data: {\"choices\":[],\"usage\":{\"completion_tokens\":32000,\"completion_tokens_details\":{\"reasoning_tokens\":5}}}\n\ndata: [DONE]\n\n",
			64000, "length", 32000, 5, "", 0, "below"},
		{"length equal", head + "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}]}\n\n" +
			"data: {\"choices\":[],\"usage\":{\"completion_tokens\":64000}}\n\n",
			64000, "length", 64000, -1, "", 0, "equal"},
		{"length above", head + "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}]}\n\n" +
			"data: {\"choices\":[],\"usage\":{\"completion_tokens\":64001}}\n\n",
			64000, "length", 64001, -1, "", 0, "above"},
		{"stop decides nothing", head + "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
			"data: {\"choices\":[],\"usage\":{\"completion_tokens\":3}}\n\n",
			64000, "stop", 3, -1, "", 0, "undecided"},
		{"length without usage", head + "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}]}\n\n",
			64000, "length", -1, -1, "", 0, "undecided"},
		{"in-stream error", head + "data: {\"error\":{\"type\":\"timeout\",\"code\":503,\"message\":\"private text\"}}\n\n",
			64000, "", -1, -1, "timeout", 503, "undecided"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := serve(c.stream)
			defer srv.Close()
			res, err := streamOmitted(context.Background(), srv.Client(), srv.URL, "key", "m/x")
			if err != nil {
				t.Fatalf("streamOmitted: %v", err)
			}
			for _, k := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens"} {
				if _, found := sentBody[k]; found {
					t.Errorf("the request carried %q, want no limit field", k)
				}
			}
			if res.Status != http.StatusOK || res.Finish != c.finish || res.Out != c.out || res.Reasoning != c.reason ||
				res.StreamErrType != c.errType || res.StreamErrCode != c.errCode {
				t.Errorf("result %+v, want status 200, finish %q, out %d, reasoning %d, error %q %d",
					res, c.finish, c.out, c.reason, c.errType, c.errCode)
			}
			if got := omittedVerdict(res.Finish, res.Out, c.stated); got != c.verdict {
				t.Errorf("verdict %q, want %q", got, c.verdict)
			}
			if line := omittedLine("m/x", c.stated, res, c.verdict); strings.Contains(line, "private") {
				t.Errorf("the kept line carries message text: %s", line)
			}
		})
	}
	t.Run("undecided says why", func(t *testing.T) {
		const tail = "): only a length finish with a count decides"
		for _, c := range []struct {
			finish      string
			out, stated int
			want        string
		}{
			{"stop", 27726, 64000, `undecided, the finish was "stop", not length (output tokens 27726, stated 64000` + tail},
			{"length", -1, 64000, `undecided, the length finish came with no usage count, so the output could not be compared (output tokens -1, stated 64000` + tail},
			{"length", 5, 0, `undecided, the length finish could not be compared: no stated limit (output tokens 5, stated 0` + tail},
			{"", -1, 64000, `undecided, the finish was "", not length (output tokens -1, stated 64000` + tail},
		} {
			if got := omittedUndecided(c.finish, c.out, c.stated); got != c.want {
				t.Errorf("omittedUndecided(%q, %d, %d) = %q, want %q", c.finish, c.out, c.stated, got, c.want)
			}
		}
	})
	t.Run("not ok status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPaymentRequired)
		}))
		defer srv.Close()
		res, err := streamOmitted(context.Background(), srv.Client(), srv.URL, "key", "m/x")
		if err != nil || res.Status != http.StatusPaymentRequired || res.Finish != "" {
			t.Errorf("result %+v, err %v; want status 402, no finish, no error", res, err)
		}
	})
}
