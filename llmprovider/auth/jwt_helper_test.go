package auth

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

// openAITestJWT is an unsigned JWT carrying claims. It stayed here when
// openai_chatgpt_test.go moved to the openai package (0015-PLAN S7).
func openAITestJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal JWT claims: %v", err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}
