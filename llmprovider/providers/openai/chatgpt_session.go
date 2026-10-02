package openai

import (
	"net/http"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// ChatGPT session headers, as Codex sends them.
const (
	// headerChatGPTAccount carries a ChatGPT session's account id.
	headerChatGPTAccount = "ChatGPT-Account-Id"
	// headerOriginator names the client to the ChatGPT backend.
	headerOriginator = "originator"
	// originatorValue is this module's originator (0002-MADR §5).
	originatorValue = "go-llmprovider-sdk"
	// headerFedRAMP marks a FedRAMP account's requests, as Codex's bearer auth
	// does (model-provider/src/bearer_auth_provider.rs:43-45).
	headerFedRAMP = "X-OpenAI-Fedramp"
)

// accountSource is a session that knows its ChatGPT account:
// *auth.OAuthSession and *auth.VendorCLISession.
type accountSource interface {
	Account() (id string, fedRAMP bool)
}

// isChatGPTSession reports whether src is a ChatGPT login: an OpenAI OAuth
// session, or the Codex CLI's.
func isChatGPTSession(src llmprovider.TokenSource) bool {
	if vendor, ok := src.(*auth.VendorCLISession); ok {
		return vendor.Provider == llmprovider.ProviderOpenAI
	}
	session, ok := src.(*auth.OAuthSession)
	return ok && session.ChatGPT()
}

// setChatGPTHeaders sets the originator, and the account id and FedRAMP flag
// of src when it knows them.
func setChatGPTHeaders(req *http.Request, src llmprovider.TokenSource) {
	req.Header.Set(headerOriginator, originatorValue)
	account, ok := src.(accountSource)
	if !ok {
		return
	}
	id, fedRAMP := account.Account()
	if id != "" {
		req.Header.Set(headerChatGPTAccount, id)
	}
	if fedRAMP {
		req.Header.Set(headerFedRAMP, "true")
	}
}

// invalidate makes src fetch a new token on its next use, and reports whether
// it can.
func invalidate(src llmprovider.TokenSource) bool {
	source, ok := src.(llmprovider.InvalidatingSource)
	if ok {
		source.Invalidate()
	}
	return ok
}
