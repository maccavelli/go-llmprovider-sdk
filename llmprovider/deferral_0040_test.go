package llmprovider

import (
	"errors"
	"strings"
	"testing"
)

// 0040-MADR D1 and A5: Tool.Deferred, Tool.Search and ToolLoadItem, and the
// requests refused before the network.

// deferralTools is a direct tool, a search tool and two deferred tools.
func deferralTools() []Tool {
	return []Tool{
		{Name: "read_file"},
		{Name: "tool_search", Search: true},
		{Name: "get_weather", Deferred: true},
		{Name: "lookup_stock", Deferred: true},
	}
}

// loadingHistory is a search call and its result loading get_weather.
func loadingHistory() []Item {
	return []Item{
		MessageItem{Role: RoleUser, Text: "weather in Paris"},
		FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"query":"weather"}`},
		FunctionCallOutputItem{CallID: "c1", Output: "Loaded get_weather."},
		ToolLoadItem{CallID: "c1", Name: "get_weather"},
	}
}

// TestRequest_Deferral_0040: each D1 refusal is ErrInvalidRequest naming what
// is wrong, and the valid shapes pass.
func TestRequest_Deferral_0040(t *testing.T) {
	c := Capabilities{Tools: Supported, ForcedToolChoice: Supported}
	for name, req := range map[string]*Request{
		"no deferral":                {Tools: []Tool{{Name: "read_file"}}},
		"deferred, unloaded":         {Tools: deferralTools()},
		"deferred, loaded":           {Tools: deferralTools(), Input: loadingHistory()},
		"forcing a loaded":           {Tools: deferralTools(), Input: loadingHistory(), ToolChoice: ForceTool("get_weather")},
		"forcing a direct":           {Tools: deferralTools(), ToolChoice: ForceTool("read_file")},
		"a search, nothing deferred": {Tools: []Tool{{Name: "read_file"}, {Name: "tool_search", Search: true}}},
	} {
		if err := c.Check(req); err != nil {
			t.Errorf("%s: %v; want it accepted", name, err)
		}
	}

	twoSearches := deferralTools()
	twoSearches[0].Search = true
	deferredSearch := deferralTools()
	deferredSearch[1].Deferred = true
	allDeferred := []Tool{{Name: "a", Deferred: true}, {Name: "b", Deferred: true}}
	unknownLoad := loadingHistory()
	unknownLoad[3] = ToolLoadItem{CallID: "c1", Name: "send_email"}
	directLoad := loadingHistory()
	directLoad[3] = ToolLoadItem{CallID: "c1", Name: "read_file"}
	// The load before its result, and for a call with none.
	earlyLoad := loadingHistory()
	earlyLoad[2], earlyLoad[3] = earlyLoad[3], earlyLoad[2]
	orphanLoad := loadingHistory()
	orphanLoad[3] = ToolLoadItem{CallID: "c9", Name: "get_weather"}
	for name, tc := range map[string]struct {
		req  *Request
		want string
	}{
		"two search tools":         {&Request{Tools: twoSearches}, `"read_file" and "tool_search"`},
		"a deferred search":        {&Request{Tools: deferredSearch}, `search tool "tool_search" is deferred`},
		"every tool deferred":      {&Request{Tools: allDeferred}, "every tool is deferred"},
		"loading an unknown":       {&Request{Tools: deferralTools(), Input: unknownLoad}, `loads "send_email"`},
		"loading a direct tool":    {&Request{Tools: deferralTools(), Input: directLoad}, `loads "read_file"`},
		"a load before its result": {&Request{Tools: deferralTools(), Input: earlyLoad}, `for call "c1", which has no earlier result`},
		"a load with no result":    {&Request{Tools: deferralTools(), Input: orphanLoad}, `for call "c9", which has no earlier result`},
		"forcing an unloaded":      {&Request{Tools: deferralTools(), ToolChoice: ForceTool("get_weather")}, `forces "get_weather", which is deferred and not loaded`},
	} {
		err := c.Check(tc.req)
		if !errors.Is(err, ErrInvalidRequest) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v; want ErrInvalidRequest containing %q", name, err, tc.want)
		}
	}
}
