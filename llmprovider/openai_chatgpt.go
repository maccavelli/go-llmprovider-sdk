package llmprovider

import (
	"time"
)

// ChatGPT session helpers, shared by the ChatGPT listing here and the openai
// provider package. They are exported for 0015-PLAN S7 only: S7b moves them
// to llmprovider/auth, and removes these exports.
const (
	// ChatGPTAccountHeader carries a ChatGPT session's account id.
	ChatGPTAccountHeader = "ChatGPT-Account-Id"
	// ChatGPTOriginatorHeader names the client to the ChatGPT backend.
	ChatGPTOriginatorHeader = "originator"
	// ChatGPTOriginatorValue is this module's originator (0002-MADR §5).
	ChatGPTOriginatorValue = "go-llmprovider-sdk"
	// ChatGPTFedRAMPHeader marks a FedRAMP account's requests, as Codex's
	// bearer auth does (model-provider/src/bearer_auth_provider.rs:43-45).
	ChatGPTFedRAMPHeader = "X-OpenAI-Fedramp"
)

// IsChatGPTSession reports whether src is a ChatGPT login: an OpenAI
// OAuth session, or the Codex CLI's. Temporary export (0015-PLAN S7).
func IsChatGPTSession(src TokenSource) bool {
	if vendor, ok := src.(*VendorCLISession); ok {
		return vendor.Provider == ProviderOpenAI
	}
	session, ok := src.(*OAuthSession)
	return ok && session.ChatGPT()
}

// ChatGPTSessionAccountID returns a ChatGPT session's account id, or "".
// Temporary export (0015-PLAN S7).
func ChatGPTSessionAccountID(src TokenSource) string {
	if vendor, ok := src.(*VendorCLISession); ok {
		accountID, _ := vendor.vendorAccount()
		return accountID
	}
	session, ok := src.(*OAuthSession)
	if !ok {
		return ""
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.AccountID
}

// ChatGPTSessionFedRAMP reports whether src is a FedRAMP ChatGPT session.
// Temporary export (0015-PLAN S7).
func ChatGPTSessionFedRAMP(src TokenSource) bool {
	if vendor, ok := src.(*VendorCLISession); ok {
		_, fedramp := vendor.vendorAccount()
		return fedramp
	}
	session, ok := src.(*OAuthSession)
	if !ok {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.FedRAMP
}

// ExpireSession makes src fetch a new token on its next use, and reports
// whether it can: an InvalidatingSource is invalidated, an OAuth session's
// expiry is moved into the past. Temporary export (0015-PLAN S7).
func ExpireSession(src TokenSource) bool {
	if source, ok := src.(InvalidatingSource); ok {
		source.Invalidate()
		return true
	}
	session, ok := src.(*OAuthSession)
	if !ok {
		return false
	}
	session.mu.Lock()
	session.Expiry = time.Now().Add(-time.Second)
	session.mu.Unlock()
	return true
}
