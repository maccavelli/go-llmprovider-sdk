package wizard_test

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/wizard"
)

// ExampleConfigureLLM runs the configuration flow over a TextPrompter whose
// answers are scripted, one per line: the one provider offered, an API key, a
// search for a model id the catalog does not list, and the choice to use it.
// A terminal program gives os.Stdin and os.Stdout instead, and stores the
// Result in its own configuration.
func ExampleConfigureLLM() {
	answers := strings.NewReader("1\nyour-api-key\nmy-model-id\n1\n")
	p := &wizard.TextPrompter{In: answers, Out: io.Discard}
	res, err := wizard.ConfigureLLM(context.Background(), p, wizard.Options{
		Providers: []llmprovider.ProviderID{llmprovider.ProviderClaude},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(res.Provider, res.Kind, res.Model)
	// Output: claude api_key my-model-id
}
