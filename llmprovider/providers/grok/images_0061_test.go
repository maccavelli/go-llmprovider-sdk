package grok

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// inputLayer is a caller layer that states a model's input (0061-MADR D5).
type inputLayer []catalog.Modality

func (inputLayer) Name() string { return "test" }

func (l inputLayer) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	return catalog.ModelInfo{ID: id, Input: l}.Known(catalog.FieldInput), true
}

// imageCheck is wirecase.Images' check of a case whose Images is support.
func imageCheck(support llmprovider.Support) wirecase.ImageCheck {
	return wirecase.ImageCheck{Support: support,
		TextOnly:    catalog.WithModelLayer(inputLayer{catalog.ModalityText}),
		TakesImages: catalog.WithModelLayer(inputLayer{catalog.ModalityText, catalog.ModalityImage})}
}

// TestImages_0061 (0061-MADR D5): Responses: Supported, each case; GIF and WebP refused (Q10).
func TestImages_0061(t *testing.T) {
	for _, c := range wireCases {
		wirecase.Images(t, c, imageCheck(llmprovider.Supported))
	}
}

// TestImages_MediaTypes_0061 (0061-MADR D5, Q10): xAI takes JPEG and PNG
// only, so a GIF or a WebP is refused, by Generate and by Stream, before the
// network; a JPEG, and an image by URL with no media type, are sent.
func TestImages_MediaTypes_0061(t *testing.T) {
	c := wireCases[0]
	request := func(img llmprovider.ImageItem) *llmprovider.Request {
		return &llmprovider.Request{Input: []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "look"}, img}}
	}
	for _, mediaType := range []string{"image/gif", "image/webp"} {
		srv := wiretest.NewServer(t, c.Reply)
		p, err := c.Build(srv.URL, llmprovider.WithoutModelMetadata())
		if err != nil {
			t.Fatal(err)
		}
		req := request(llmprovider.ImageItem{MediaType: mediaType, Data: []byte("img")})
		if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) ||
			!strings.Contains(err.Error(), "Input[1] is "+mediaType) {
			t.Errorf("%s: Generate = %v; want ErrUnsupported naming Input[1]", mediaType, err)
		}
		for _, err := range llmprovider.Stream(context.Background(), p, req) {
			if !errors.Is(err, llmprovider.ErrUnsupported) {
				t.Errorf("%s: Stream = %v; want ErrUnsupported", mediaType, err)
			}
			break
		}
		if n := len(srv.Requests()); n != 0 {
			t.Errorf("%s: %d request(s) sent; want none", mediaType, n)
		}
	}
	for _, img := range []llmprovider.ImageItem{{MediaType: "image/jpeg", Data: []byte("img")}, {URL: wirecase.ImageURL}} {
		srv := wiretest.NewServer(t, c.Reply)
		p, err := c.Build(srv.URL, llmprovider.WithoutModelMetadata())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Generate(context.Background(), request(img)); err != nil {
			t.Errorf("%+v: Generate = %v; want it sent", img, err)
		}
	}
}
