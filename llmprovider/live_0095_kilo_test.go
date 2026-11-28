//go:build live_gateways

package llmprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/kiloendpoint"
)

// liveControlModel0095 and liveControlLimit0095 are 0089-MADR's control run
// (0095-MADR FU24): the model DL65 measured cut at 128,000 output tokens
// with no limit sent, and the limit its listing states, sent this time.
const (
	liveControlModel0095 = "z-ai/glm-5.3-flashx"
	liveControlLimit0095 = 131072
)

// sendLimit0095 sets max_tokens in each request's JSON body, so that
// streamOmitted's request sends a limit for the control run.
type sendLimit0095 struct{ limit int }

func (s sendLimit0095) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, errors.New("control run: the request body could not be read")
	}
	var body map[string]any
	if json.Unmarshal(raw, &body) != nil {
		return nil, errors.New("control run: the request body is not JSON")
	}
	body["max_tokens"] = s.limit
	if raw, err = json.Marshal(body); err != nil {
		return nil, errors.New("control run: the request body could not be encoded")
	}
	req = req.Clone(req.Context())
	req.Body = io.NopCloser(bytes.NewReader(raw))
	req.ContentLength = int64(len(raw))
	req.Header.Set("Content-Length", strconv.Itoa(len(raw)))
	return http.DefaultTransport.RoundTrip(req)
}

// TestLive_KiloControl_0095 is 0089-MADR's control run, which 0095-MADR FU24
// runs on the owner's word: one streamed request to Kilo for
// z-ai/glm-5.3-flashx with max_tokens 131,072 sent and the prompt DL65 used.
// A length finish at 128,000 output tokens puts the earlier cut in the
// model; one at 131,072 puts it in what Kilo fills when no limit is sent. It
// runs only with LLMPROVIDER_LIVE_0095 set and KILO_API_KEY in the
// environment (read by name, never printed), and logs only the allowlisted
// line: model, status, finish reason, output and reasoning token counts.
// Billed: up to 131,072 output tokens. One attempt; a failure is logged and
// fails, with a fixed text.
func TestLive_KiloControl_0095(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0095") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0095 to run 0095-MADR FU24's control run (it bills up to 131,072 output tokens)")
	}
	key := kiloKey(t)
	ctx, cancel := context.WithTimeout(context.Background(), liveOmittedPerModel)
	defer cancel()
	client := &http.Client{Transport: sendLimit0095{limit: liveControlLimit0095}, Timeout: liveOmittedPerModel + time.Minute}
	res, err := streamOmitted(ctx, client, kiloendpoint.BaseURL, key, liveControlModel0095)
	if err != nil {
		t.Fatalf("%s: %v", liveControlModel0095, err)
	}
	t.Logf("0095 FU24 RESULT model=%s sent=%d status=%d limit_accepted=%t finish=%q out=%d reasoning=%d stream_error=%q stream_code=%d",
		liveControlModel0095, liveControlLimit0095, res.Status, res.Status == http.StatusOK && res.StreamErrType == "",
		res.Finish, res.Out, res.Reasoning, res.StreamErrType, res.StreamErrCode)
	if res.Status != http.StatusOK || res.StreamErrType != "" {
		t.Fatalf("0095 FU24 FAILED: HTTP status %d, stream error %q, code %d; want 200 and none", res.Status,
			res.StreamErrType, res.StreamErrCode)
	}
	t.Logf("0095 FU24 OK: a %q finish at %d output tokens with %d sent", res.Finish, res.Out, liveControlLimit0095)
}
