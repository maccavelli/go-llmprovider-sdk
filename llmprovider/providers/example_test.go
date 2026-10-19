package providers_test

import (
	"fmt"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// ExampleNew builds a built-in provider by its id, as a configuration names
// it. Building one sends no request.
func ExampleNew() {
	p, err := providers.New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("your-api-key"))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(p.ID())
	// Output: claude
}

// ExampleDefaultWith builds a Registry whose built-in providers share one
// cache, given once rather than to each New.
func ExampleDefaultWith() {
	var cache catalog.Cache
	reg := providers.DefaultWith(catalog.WithCache(&cache))
	p, err := reg.New(llmprovider.ProviderGemini, llmprovider.WithAPIKey("your-api-key"))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(p.ID())
	// Output: gemini
}
