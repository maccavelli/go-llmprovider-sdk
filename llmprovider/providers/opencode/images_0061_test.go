package opencode

import (
	"context"
	"encoding/json"
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

// TestImages_0061 (0061-MADR D5): BestEffort, each route's form; the google route drops an image by URL.
func TestImages_0061(t *testing.T) {
	for _, c := range wireCases {
		wirecase.Images(t, c, imageCheck(llmprovider.BestEffort))
	}
}

// TestImages_Drops_0061 (0061-MADR D5): the google route takes no image URL: the image by URL is dropped,
// with one WarningImageDropped naming Input[2], and the request carries the
// image's bytes, as inline_data.
func TestImages_Drops_0061(t *testing.T) {
	c := wireCases[googleCase(t)]
	srv := wiretest.NewServer(t, c.Reply)
	p, err := c.Build(srv.URL, llmprovider.WithoutModelMetadata())
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Generate(context.Background(), &llmprovider.Request{Input: wirecase.ImageItems()})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Warnings) != 1 || resp.Warnings[0].Kind != llmprovider.WarningImageDropped ||
		!strings.HasPrefix(resp.Warnings[0].Detail, "Input[2], an image by URL, was not sent: ") {
		t.Errorf("Warnings = %+v; want one WarningImageDropped for Input[2]", resp.Warnings)
	}
	raw, err := json.Marshal(srv.Requests())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), wirecase.ImageURL) || !strings.Contains(string(raw), `"inline_data":{"data":"`+wirecase.PNGBase64+`","mime_type":"image/png"}`) {
		t.Errorf("the request = %s; want the image's bytes, as inline_data, and no image URL", raw)
	}
}

// googleCase is the index of the google route's case.
func googleCase(t *testing.T) int {
	t.Helper()
	for i, c := range wireCases {
		if c.Name == "opencode-zen-google" {
			return i
		}
	}
	t.Fatal("no opencode-zen-google case")
	return -1
}
