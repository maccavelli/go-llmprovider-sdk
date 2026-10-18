package llmprovider_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
)

// The parsing behind TestLive_OpenAISigninProbe (0017-PLAN U5). It sits
// outside the live tag so that its offline test runs with the package's.

// probeListingSummary describes a sign-in token's model listing without its
// models: the top-level keys, the number of models, how many a picker shows
// (visibility "list"), and the first of those.
type probeListingSummary struct {
	Keys   []string
	Count  int
	Listed int
	First  string
}

// probeListing reads GET /v1/models as a sign-in token gets it: a "models"
// array of slugs, not the platform's "data" array (0017-MADR, amendment
// 2026-10-16).
func probeListing(body []byte) (probeListingSummary, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		return probeListingSummary{}, fmt.Errorf("the listing is not a JSON object: %w", err)
	}
	s := probeListingSummary{Keys: slices.Sorted(maps.Keys(top))}
	raw, ok := top["models"]
	if !ok {
		return s, errors.New(`the listing has no "models" array`)
	}
	var models []struct {
		Slug       string `json:"slug"`
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal(raw, &models); err != nil {
		return s, fmt.Errorf(`the listing's "models" is not an array of models: %w`, err)
	}
	s.Count = len(models)
	for _, m := range models {
		if m.Visibility == "list" && m.Slug != "" {
			if s.Listed == 0 {
				s.First = m.Slug
			}
			s.Listed++
		}
	}
	return s, nil
}

// probeRevocationEndpoint is an OpenID discovery document's
// revocation_endpoint, which must be an https URL.
func probeRevocationEndpoint(body []byte) (string, error) {
	var doc struct {
		Revocation string `json:"revocation_endpoint"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", fmt.Errorf("the discovery document is not JSON: %w", err)
	}
	if doc.Revocation == "" {
		return "", errors.New("the discovery document has no revocation_endpoint")
	}
	u, err := url.Parse(doc.Revocation)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", fmt.Errorf("the revocation_endpoint %q is not an https URL", doc.Revocation)
	}
	return doc.Revocation, nil
}

// probeRefusal describes a refused request by its status, and the code and
// redacted message of whichever error shape the body takes: the API's error
// object, OAuth's error string and description, or admission's detail.
func probeRefusal(status int, body []byte) string {
	var doc struct {
		Error       json.RawMessage `json:"error"`
		Description string          `json:"error_description"`
		Detail      any             `json:"detail"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return fmt.Sprintf("REFUSED, HTTP %d, a body of %d bytes that is not JSON", status, len(body))
	}
	var object struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Param   string `json:"param"`
	}
	var code, message string
	switch {
	case json.Unmarshal(doc.Error, &object) == nil && (object.Code != "" || object.Message != ""):
		code, message = object.Code, object.Message
		if object.Param != "" {
			message += " (param " + object.Param + ")"
		}
	case json.Unmarshal(doc.Error, &code) == nil && code != "":
		message = doc.Description
	case doc.Detail != nil:
		message = fmt.Sprint(doc.Detail)
	default:
		return fmt.Sprintf("REFUSED, HTTP %d, a JSON body of %d bytes with no error", status, len(body))
	}
	return fmt.Sprintf("REFUSED, HTTP %d, code %q: %s", status, code, redact.String(message))
}

// probeStreamOutcome describes a raw Responses request's outcome without its
// text: a refusal as probeRefusal does, or how its event stream ended and
// after how many text deltas.
func probeStreamOutcome(status int, body []byte) string {
	if status != 200 {
		return probeRefusal(status, body)
	}
	type streamError struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	deltas, events, end := 0, 0, ""
	scan := bufio.NewScanner(bytes.NewReader(body))
	scan.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for scan.Scan() {
		data, ok := strings.CutPrefix(scan.Text(), "data: ")
		if !ok || data == "[DONE]" {
			continue
		}
		var ev struct {
			Type     string `json:"type"`
			Response struct {
				Error *streamError `json:"error"`
			} `json:"response"`
			streamError
		}
		if json.Unmarshal([]byte(data), &ev) != nil {
			continue
		}
		events++
		switch ev.Type {
		case "response.output_text.delta":
			deltas++
		case "response.completed":
			end = "accepted, completed"
		case "response.incomplete":
			end = "accepted, incomplete"
		case "response.failed":
			e := ev.Response.Error
			if e == nil {
				e = &streamError{}
			}
			end = fmt.Sprintf("REFUSED in the stream, code %q: %s", e.Code, redact.String(e.Message))
		case "error":
			end = fmt.Sprintf("REFUSED in the stream, code %q: %s", ev.Code, redact.String(ev.Message))
		}
	}
	if end == "" {
		return fmt.Sprintf("HTTP 200, no terminal event in %d events", events)
	}
	return fmt.Sprintf("%s, after %d text deltas", end, deltas)
}

// TestSigninProbeHelpers_0017 holds the probe's parsing to the shapes
// OpenAI's documents give (0017-MADR, amendment 2026-10-16).
func TestSigninProbeHelpers_0017(t *testing.T) {
	t.Run("listing", func(t *testing.T) {
		got, err := probeListing([]byte(`{"models":[{"slug":"a","visibility":"hide"},{"slug":"b","visibility":"list"},{"slug":"c","visibility":"list"}]}`))
		if err != nil {
			t.Fatal(err)
		}
		want := probeListingSummary{Keys: []string{"models"}, Count: 3, Listed: 2, First: "b"}
		if !slices.Equal(got.Keys, want.Keys) || got.Count != want.Count || got.Listed != want.Listed || got.First != want.First {
			t.Errorf("probeListing = %+v, want %+v", got, want)
		}
		// The platform's shape is named as such: the run's finding is this text.
		if _, err := probeListing([]byte(`{"object":"list","data":[{"id":"gpt-5"}]}`)); err == nil || !strings.Contains(err.Error(), `no "models" array`) {
			t.Errorf(`probeListing(the platform's shape) = %v, want an error naming no "models" array`, err)
		}
		for _, body := range []string{`{"models":{}}`, `not json`} {
			if _, err := probeListing([]byte(body)); err == nil {
				t.Errorf("probeListing(%s): no error", body)
			}
		}
	})
	t.Run("revocation endpoint", func(t *testing.T) {
		got, err := probeRevocationEndpoint([]byte(`{"issuer":"https://auth.example","revocation_endpoint":"https://auth.example/oauth/revoke"}`))
		if err != nil || got != "https://auth.example/oauth/revoke" {
			t.Errorf("probeRevocationEndpoint = %q, %v", got, err)
		}
		for _, body := range []string{`{"issuer":"https://auth.example"}`, `{"revocation_endpoint":"http://auth.example/revoke"}`, `{"revocation_endpoint":"https://"}`, `[]`} {
			if got, err := probeRevocationEndpoint([]byte(body)); err == nil {
				t.Errorf("probeRevocationEndpoint(%s) = %q, no error", body, got)
			}
		}
	})
	t.Run("refusal", func(t *testing.T) {
		for _, tc := range []struct {
			status int
			body   string
			want   string
		}{
			{400, `{"error":{"code":"subscription_sharing_unsupported_capability","message":"Unsupported parameter","param":"max_output_tokens"}}`,
				`REFUSED, HTTP 400, code "subscription_sharing_unsupported_capability": Unsupported parameter (param max_output_tokens)`},
			{400, `{"error":"invalid_grant","error_description":"Refresh token revoked"}`, `REFUSED, HTTP 400, code "invalid_grant": Refresh token revoked`},
			{403, `{"detail":"Not eligible"}`, `REFUSED, HTTP 403, code "": Not eligible`},
			{502, `<html>`, `REFUSED, HTTP 502, a body of 6 bytes that is not JSON`},
			{500, `{}`, `REFUSED, HTTP 500, a JSON body of 2 bytes with no error`},
		} {
			if got := probeRefusal(tc.status, []byte(tc.body)); got != tc.want {
				t.Errorf("probeRefusal(%d, %s)\n got %s\nwant %s", tc.status, tc.body, got, tc.want)
			}
		}
	})
	t.Run("stream outcome", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			status int
			body   string
			want   string
		}{
			{"completed", 200, "event: response.created\ndata: {\"type\":\"response.created\"}\n\n" +
				"data: {\"type\":\"response.output_text.delta\",\"delta\":\"AL\"}\n\n" +
				"data: {\"type\":\"response.output_text.delta\",\"delta\":\"PHA\"}\n\n" +
				"data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n",
				"accepted, completed, after 2 text deltas"},
			{"failed", 200, "data: {\"type\":\"response.created\"}\n\n" +
				"data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"code\":\"subscription_sharing_usage_limit_exceeded\",\"message\":\"Usage limit reached\"}}}\n\n",
				`REFUSED in the stream, code "subscription_sharing_usage_limit_exceeded": Usage limit reached, after 0 text deltas`},
			{"error event", 200, "data: {\"type\":\"error\",\"code\":\"server_error\",\"message\":\"Try again\"}\n\n",
				`REFUSED in the stream, code "server_error": Try again, after 0 text deltas`},
			{"cut short", 200, "data: {\"type\":\"response.created\"}\n\ndata: {\"type\":\"response.output_text.delta\"}\n\n",
				"HTTP 200, no terminal event in 2 events"},
			{"refused", 400, `{"error":{"code":"","message":"Store must be set to false"}}`,
				`REFUSED, HTTP 400, code "": Store must be set to false`},
		} {
			if got := probeStreamOutcome(tc.status, []byte(tc.body)); got != tc.want {
				t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
			}
		}
	})
}
