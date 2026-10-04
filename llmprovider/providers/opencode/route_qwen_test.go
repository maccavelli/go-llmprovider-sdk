package opencode

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestHeuristicRoute_QwenAsTheTable (0020-MADR F41): with no table row and no
// metadata, a qwen model routes as the table's qwen rows do: -max to chat
// and -flash to messages on both gateways, another to messages on Zen and to
// chat on Go.
func TestHeuristicRoute_QwenAsTheTable(t *testing.T) {
	for gateway, rows := range routeTable {
		for model, want := range rows {
			if strings.HasPrefix(model, "qwen") {
				if got := heuristicRoute(gateway, model); got != want {
					t.Errorf("heuristicRoute(%s, %s) = %s; the table says %s", gateway, model, got, want)
				}
			}
		}
	}
	zen, goGW := llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo
	for _, c := range []struct {
		gateway llmprovider.ProviderID
		model   string
		want    Route
	}{
		{zen, "qwen9-max", RouteChatCompletions},
		{zen, "qwen9-flash", RouteMessages},
		{zen, "qwen9-plus", RouteMessages},
		{goGW, "qwen9-max", RouteChatCompletions},
		{goGW, "qwen9-flash", RouteMessages},
		{goGW, "qwen9-plus", RouteChatCompletions},
	} {
		if got := heuristicRoute(c.gateway, c.model); got != c.want {
			t.Errorf("heuristicRoute(%s, %s) = %s, want %s", c.gateway, c.model, got, c.want)
		}
	}
}
