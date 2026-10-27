package ollama

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

// TestImages_0061 (0061-MADR D5): BestEffort: an image's bytes as a data URL, an image by URL dropped.
func TestImages_0061(t *testing.T) {
	for _, c := range wireCases {
		wirecase.Images(t, c, imageCheck(llmprovider.BestEffort))
	}
}

// TestImages_Drops_0061 (0061-MADR D5): Ollama takes no image URL: the image by URL is dropped,
// with one WarningImageDropped naming Input[2], and the request carries the
// image's bytes, its image_url a data URL string.
func TestImages_Drops_0061(t *testing.T) {
	c := wireCases[0]
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
	if strings.Contains(string(raw), wirecase.ImageURL) || !strings.Contains(string(raw), `"image_url":"data:image/png;base64,`+wirecase.PNGBase64+`"`) {
		t.Errorf("the request = %s; want the image's bytes, its image_url a data URL string, and no image URL", raw)
	}
}
