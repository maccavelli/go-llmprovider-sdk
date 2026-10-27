package huggingface

import (
	"testing"

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

// TestImages_0061 (0061-MADR D5): the Chat Completions object form: Supported.
func TestImages_0061(t *testing.T) {
	for _, c := range wireCases {
		wirecase.Images(t, c, imageCheck(llmprovider.Supported))
	}
}
