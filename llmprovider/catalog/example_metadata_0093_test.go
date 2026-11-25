package catalog_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// ExampleWithMetadata reads a models.dev document, as a caller would after
// downloading https://models.dev/api.json, and hands it in: the lookup reads
// the model's window from it, with no request. The SDK fetches no document
// itself (0093-MADR D1, D6; R50).
func ExampleWithMetadata() {
	doc, err := catalog.ParseMetadata(strings.NewReader(
		`{"anthropic":{"models":{"claude-x":{"name":"Claude X","limit":{"context":200000,"output":64000}}}}}`))
	if err != nil {
		fmt.Println(err)
		return
	}
	var cache catalog.Cache
	info, err := catalog.LookupModel(context.Background(), llmprovider.ProviderClaude, nil, "claude-x",
		catalog.WithMetadata(doc), catalog.WithCache(&cache))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(info.ContextWindow, info.SourceOf(catalog.FieldContextWindow))
	// Output: 200000 models.dev
}
