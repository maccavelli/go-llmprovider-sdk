package chatcompat

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/compatdialect"
)

// Discovery (0061-MADR E5; its amendment "E5's ports and a keyed server,
// confirmed before Phase 4e").

// discoverHost is the one host Discover asks: loopback, never the network.
const discoverHost = "127.0.0.1"

// discoverPorts are the ports Discover asks, in its result's order, each
// confirmed in the record from its server's documentation: Ollama 11434,
// llama.cpp's llama-server 9931, LocalAI (and llama.cpp's Docker images)
// 8080, vLLM 8000, Jan 1337, KoboldCpp 5001, and LM Studio 1234, the port its
// documentation's examples use.
var discoverPorts = []int{11434, 9931, 8080, 8000, 1337, 5001, 1234}

// discoverBound bounds each port's ask, within the caller's ctx.
const discoverBound = 300 * time.Millisecond

// discoverReplyLimit bounds a listing Discover reads.
const discoverReplyLimit = 8 << 20

// Server is a local Chat Completions server Discover found. It is safe for
// concurrent use only while no goroutine changes it.
type Server struct {
	// BaseURL is the server's Chat Completions base, such as
	// "http://127.0.0.1:11434/v1": what WithBaseURL takes.
	BaseURL string
	// Label names the server for a person: its dialect's, such as "Ollama",
	// or its host and port, such as "127.0.0.1:1234".
	Label string
	// Models are the ids its GET /v1/models lists, in its order; none when
	// NeedsKey. An id holding a control character is left out (0066-MADR
	// V1).
	Models []string
	// NeedsKey reports a server that refused the listing, sent with no key,
	// with 401 or 403: it is there, and wants a key.
	NeedsKey bool
}

// Discover asks the documented default ports of local Chat Completions
// servers, on 127.0.0.1 only, each with one GET /v1/models, all at once, each
// bounded by 300 ms within ctx, sending no key whatever opts hold, and returns
// the servers that answer, in the ports' order: Ollama 11434, llama.cpp 9931,
// LocalAI 8080, vLLM 8000, Jan 1337, KoboldCpp 5001, LM Studio 1234. A server
// answers with a listing, its Models, or with 401 or 403, NeedsKey; any other
// reply, or none, is no server. opts give the HTTP client and the caller's
// identity. It returns an error for an option ResolveOptions refuses, and
// ctx's when ctx ends (0061-MADR E5).
func Discover(ctx context.Context, opts ...llmprovider.Option) ([]Server, error) {
	st, err := llmprovider.ResolveOptions(scope, opts)
	if err != nil {
		return nil, err
	}
	client, userAgent, logger := st.HTTPClient(), st.UserAgent(), st.Logger()
	found := make([]*Server, len(discoverPorts))
	var wg sync.WaitGroup
	for i, port := range discoverPorts {
		wg.Go(func() { found[i] = askPort(ctx, client, userAgent, logger, port) })
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []Server
	for _, s := range found {
		if s != nil {
			out = append(out, *s)
		}
	}
	return out, nil
}

// askPort asks port's GET /v1/models, with no key, within discoverBound: a
// listing is a server, labelled by its dialect or its host and port; 401 or
// 403 a server that needs a key; anything else none. A failed close is
// logged at Debug, as a listing's is.
func askPort(ctx context.Context, client *http.Client, userAgent string, logger *slog.Logger, port int) *Server {
	ctx, cancel := context.WithTimeout(ctx, discoverBound)
	defer cancel()
	host := net.JoinHostPort(discoverHost, strconv.Itoa(port))
	base := "http://" + host + "/v1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/models", http.NoBody)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Debug("llmprovider: chatcompat: close discovery body", "error", err)
		}
	}()
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &Server{BaseURL: base, Label: host, NeedsKey: true}
	case http.StatusOK:
	default:
		return nil
	}
	var reply struct {
		Data []compatdialect.Entry `json:"data"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, discoverReplyLimit)).Decode(&reply) != nil || reply.Data == nil {
		return nil
	}
	s := &Server{BaseURL: base, Label: host}
	if d, ok := compatdialect.Of(reply.Data); ok {
		s.Label = d.Label
	}
	for _, e := range reply.Data {
		if e.ID != "" && !hasControl(e.ID) {
			s.Models = append(s.Models, e.ID)
		}
	}
	return s
}

// hasControl reports whether s holds a character a terminal acts on: a C0
// control but tab, DEL, or a C1 control (U+0080 to U+009F). It is the rule of
// internal/redact, and of catalog's listings, which drop such an id
// (0066-MADR V1, Q3 (a)); no provider package imports internal/redact.
func hasControl(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool {
		return r < 0x20 && r != '\t' || r == 0x7f || r >= 0x80 && r <= 0x9f
	})
}
