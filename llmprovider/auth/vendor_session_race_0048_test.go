package auth

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestVendorCLISession_ConcurrentUse_0048 shows that one VendorCLISession is
// safe for concurrent use: 16 goroutines calling Token and Account at once all
// succeed, and -race finds nothing (0048-MADR D3).
func TestVendorCLISession_ConcurrentUse_0048(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	content, access := codexAuthJSON(t, time.Now().Add(time.Hour), "acct_0048")
	writeVendorFile(t, path, content)
	s := &VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: path}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 20 {
				tok, err := s.Token(context.Background())
				if err != nil || tok.Value != access {
					t.Errorf("Token = %q, %v; want the file's token", tok.Value, err)
					return
				}
				if id, _ := s.Account(); id != "acct_0048" {
					t.Errorf("Account = %q, want acct_0048", id)
					return
				}
			}
		})
	}
	wg.Wait()
}
