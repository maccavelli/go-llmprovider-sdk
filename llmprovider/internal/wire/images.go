package wire

import (
	"encoding/base64"
	"fmt"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// DataURL is img's bytes as a data URL, data:<MediaType>;base64,<Data>
// (0061-MADR D5).
func DataURL(img llmprovider.ImageItem) string {
	return "data:" + img.MediaType + ";base64," + base64.StdEncoding.EncodeToString(img.Data)
}

// RefuseImages refuses input for a model that does not take images
// (0061-MADR D5): nil when input holds no ImageItem or known is false, the
// facts not stating the model's input; else, when takesImages is false, an
// error matching ErrUnsupported naming the model. A provider passes known as
// facts.Has(catalog.FieldInput) and takesImages as
// slices.Contains(facts.Input, catalog.ModalityImage).
func RefuseImages(model string, input []llmprovider.Item, known, takesImages bool) error {
	if !known || takesImages || !slices.ContainsFunc(input, isImage) {
		return nil
	}
	return fmt.Errorf("%w: model %q takes no image input, by its facts", llmprovider.ErrUnsupported, model)
}

// DropImageURLs is input without its images by URL, with a
// WarningImageDropped for each, naming its index in input and why, never the
// URL, which may carry a token (0061-MADR D5; the amendment "D5 made exact
// before Phase 6", item 4). Input is returned as it is when it holds none.
func DropImageURLs(model string, input []llmprovider.Item, why string) ([]llmprovider.Item, []llmprovider.Warning) {
	var (
		kept     []llmprovider.Item
		warnings []llmprovider.Warning
	)
	for i, item := range input {
		if img, ok := item.(llmprovider.ImageItem); ok && img.URL != "" {
			if kept == nil {
				kept = append(make([]llmprovider.Item, 0, len(input)), input[:i]...)
			}
			warnings = append(warnings, llmprovider.Warning{Kind: llmprovider.WarningImageDropped, Model: model,
				Detail: fmt.Sprintf("Input[%d], an image by URL, was not sent: %s", i, why)})
			continue
		}
		if kept != nil {
			kept = append(kept, item)
		}
	}
	if warnings == nil {
		return input, nil
	}
	return kept, warnings
}

// isImage reports whether item is an ImageItem.
func isImage(item llmprovider.Item) bool {
	_, ok := item.(llmprovider.ImageItem)
	return ok
}
