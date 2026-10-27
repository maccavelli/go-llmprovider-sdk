package wirecase

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// ImageURL is the image scenario's image by URL.
const ImageURL = "https://example.com/cat.png"

// onePixelPNG is a 1×1 greyscale PNG, as llmtest's, so that no image is read
// from a file.
var onePixelPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x00, 0x00, 0x00, 0x00, 0x3a, 0x7e, 0x9b, 0x55, 0x00, 0x00, 0x00,
	0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0xda, 0x63, 0x60, 0x00, 0x00, 0x00,
	0x02, 0x00, 0x01, 0xe5, 0x27, 0xde, 0xfc, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

// PNGBase64 is the image scenario's PNG, base64-encoded, as each wire sends
// its bytes.
var PNGBase64 = base64.StdEncoding.EncodeToString(onePixelPNG)

// ImageItems is the image scenario's turn: the user's text, a PNG by its
// bytes, then an image by URL (0061-MADR D11).
func ImageItems() []llmprovider.Item {
	return []llmprovider.Item{
		user("What is in these pictures?"),
		llmprovider.ImageItem{MediaType: "image/png", Data: onePixelPNG},
		llmprovider.ImageItem{Role: llmprovider.RoleUser, URL: ImageURL},
	}
}

// ImageCheck is what Images checks of a case (0061-MADR D5).
type ImageCheck struct {
	// Support is the case's Capabilities.Images.
	Support llmprovider.Support
	// TextOnly and TakesImages give the case's model facts that state its
	// input: text alone, and text and image. A provider test builds each
	// from a catalog layer, which this package does not import.
	TextOnly, TakesImages llmprovider.Option
}

// Images checks c's images (0061-MADR D5): its Capabilities.Images; where
// that is Unsupported, the image turn refused before the network; else the
// turn refused, by Generate and by Stream, before the network, for a model
// whose facts take no image, and sent, with the PNG's bytes, for one whose
// facts take images and for one with no facts.
func Images(t *testing.T, c Case, check ImageCheck) {
	t.Helper()
	req := func() *llmprovider.Request { return &llmprovider.Request{Input: ImageItems()} }
	run := func(name string, body func(*testing.T, llmprovider.Provider, func() []wiretest.Request), facts ...llmprovider.Option) {
		t.Run(c.Name+"/"+name, func(t *testing.T) {
			srv := wiretest.NewServer(t, c.Reply)
			p, err := c.Build(srv.URL, append([]llmprovider.Option{llmprovider.WithoutModelMetadata()}, facts...)...)
			must(t, err)
			body(t, p, func() []wiretest.Request { return posts(srv.Requests()) })
		})
	}
	refused := func(t *testing.T, p llmprovider.Provider, requests func() []wiretest.Request, why string) {
		t.Helper()
		_, err := p.Generate(context.Background(), req())
		expect(t, errors.Is(err, llmprovider.ErrUnsupported), "Generate %s: %v; want ErrUnsupported", why, err)
		for _, err := range llmprovider.Stream(context.Background(), p, req()) {
			expect(t, errors.Is(err, llmprovider.ErrUnsupported), "Stream %s: %v; want ErrUnsupported", why, err)
			break
		}
		n := len(requests())
		expect(t, n == 0, "%s: %d generation request(s) sent; want none", why, n)
	}
	run("capability", func(t *testing.T, p llmprovider.Provider, requests func() []wiretest.Request) {
		got := p.Capabilities().Images
		expect(t, got == check.Support, "Capabilities().Images = %v; want %v", got, check.Support)
		if check.Support == llmprovider.Unsupported {
			refused(t, p, requests, "with Images Unsupported")
		}
	})
	if check.Support == llmprovider.Unsupported {
		return
	}
	run("text-only model", func(t *testing.T, p llmprovider.Provider, requests func() []wiretest.Request) {
		refused(t, p, requests, "to a model whose facts take no image")
	}, check.TextOnly)
	sent := func(t *testing.T, p llmprovider.Provider, requests func() []wiretest.Request) {
		_, err := p.Generate(context.Background(), req())
		must(t, err)
		got := requests()
		raw, err := wiretest.Encode(got)
		must(t, err)
		expect(t, len(got) == 1, "%d generation requests; want 1", len(got))
		expect(t, strings.Contains(string(raw), PNGBase64), "the request does not carry the PNG's bytes: %s", raw)
	}
	run("image model", sent, check.TakesImages)
	run("no facts", sent)
}

// expect reports a failure, format with args, when ok is false.
func expect(t *testing.T, ok bool, format string, args ...any) {
	t.Helper()
	if !ok {
		t.Errorf(format, args...)
	}
}

// must ends the test when err is set.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// posts is the generation requests among requests: the listing's are GETs.
func posts(requests []wiretest.Request) []wiretest.Request {
	var out []wiretest.Request
	for _, r := range requests {
		if r.Method == http.MethodPost {
			out = append(out, r)
		}
	}
	return out
}
