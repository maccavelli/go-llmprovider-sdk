package auth

import (
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// chatGPTLoginSession is the session a ChatGPT login yields for an id token
// with these auth claims.
func chatGPTLoginSession(t *testing.T, auth map[string]any) *OAuthSession {
	t.Helper()
	session, err := oauthSessionFromResponse(
		oauthFlowConfig{provider: llmprovider.ProviderOpenAI, issuer: DefaultOpenAIIssuer, clientID: DefaultOpenAIClientID, now: time.Now},
		DefaultOpenAIIssuer+"/oauth/token",
		oauthTokenResponse{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 3600,
			IDToken: openAITestJWT(t, map[string]any{"https://api.openai.com/auth": auth})})
	if err != nil {
		t.Fatal(err)
	}
	return session
}

// TestChatGPTLogin_FedRAMPClaim: an id token with chatgpt_account_is_fedramp
// makes a FedRAMP session, and without it none. It is the login half of the
// old TestOpenAIChatGPT_FedRAMPHeader; the openai package keeps the header
// half (0015-PLAN S7).
func TestChatGPTLogin_FedRAMPClaim(t *testing.T) {
	fedramp := chatGPTLoginSession(t, map[string]any{"chatgpt_account_id": "acct", "chatgpt_account_is_fedramp": true})
	if _, flag := fedramp.Account(); !fedramp.FedRAMP || !flag {
		t.Fatalf("FedRAMP = %t, Account's flag = %t; want both true", fedramp.FedRAMP, flag)
	}
	plain := chatGPTLoginSession(t, map[string]any{"chatgpt_account_id": "acct"})
	if _, flag := plain.Account(); plain.FedRAMP || flag {
		t.Fatalf("FedRAMP = %t, Account's flag = %t; want both false", plain.FedRAMP, flag)
	}
}
