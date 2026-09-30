package llmprovider

import (
	"testing"
)

func systemConversation() []Item {
	return []Item{
		MessageItem{Role: "system", Text: "Always answer in French."},
		MessageItem{Role: "system", Text: "Be brief."},
		MessageItem{Role: jsonRoleUser, Text: "Say hello."},
	}
}

func TestOpencodeMessages_SystemMessageIsTopLevel(t *testing.T) {
	p, err := NewOpencode(ProviderOpencodeGo, "k", "qwen3.8-flash", WithOpencodeRoute(OpencodeRouteMessages))
	if err != nil {
		t.Fatalf("NewOpencode: %v", err)
	}
	body := p.messagesBody(systemConversation(), nil, false)
	if body["system"] != "Always answer in French.\n\nBe brief." {
		t.Errorf("system = %#v, want the two system items joined", body["system"])
	}
	mustEqualJSON(t, body[jsonKeyMessages], `[{"role":"user","content":"Say hello."}]`)
}
