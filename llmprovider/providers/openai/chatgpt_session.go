package openai

import (
	"net/http"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/chatgpt"
)

// ChatGPT session headers, as Codex sends them. They, the session check and
// the listing request live in internal/chatgpt, which catalog's ListRaw
// shares (0041-MADR D9).
const (
	// headerChatGPTAccount carries a ChatGPT session's account id.
	headerChatGPTAccount = chatgpt.HeaderAccount
	// headerOriginator names the client to the ChatGPT backend.
	headerOriginator = chatgpt.HeaderOriginator
	// originatorValue is this module's originator (0002-MADR §5).
	originatorValue = chatgpt.OriginatorValue
	// headerFedRAMP marks a FedRAMP account's requests.
	headerFedRAMP = chatgpt.HeaderFedRAMP
)

// isChatGPTSession reports whether src is a ChatGPT login: an OpenAI OAuth
// session, or the Codex CLI's.
func isChatGPTSession(src llmprovider.TokenSource) bool { return chatgpt.IsSession(src) }

// setChatGPTHeaders sets the originator, and the account id and FedRAMP flag
// of src when it knows them.
func setChatGPTHeaders(req *http.Request, src llmprovider.TokenSource) { chatgpt.SetHeaders(req, src) }
