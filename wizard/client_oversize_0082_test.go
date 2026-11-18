package wizard

// 0082-PLAN Phase 1's sixth pass (the fifth amendment of 2026-11-18: R-14, an
// over-size client record is undecodable; the fifth review's null row and S14).
// Every issuer is in process.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// R-14 (0082-MADR, fifth amendment; the review's row "valid JSON over 64 KiB"):
// a client record file over 64 KiB is corruption, not a permissions problem. It
// is replaced with the same single notice as any other corrupt record, with a
// plan session of the app's own stored and with nothing stored: the sign-in goes
// on, no plan session is handed over, the new client replaces the file, and
// nothing of the file's content, or of the size error's text, appears in any
// prompt or notice. Red while an over-size file is a read error that fails every
// sign-in.
func TestOverSizeClientRecordIsReplacedWithOneNotice_0082(t *testing.T) {
	big := append(append([]byte(`{"client_id":"oaiapp_big","host_id":"OVERSIZE-SECRET`), bytes.Repeat([]byte("x"), 70<<10)...), `"}`...)
	for _, stored := range []bool{true, false} {
		t.Run(fmt.Sprintf("session stored %v", stored), func(t *testing.T) {
			ctx := context.Background()
			store := newClientStore(t)
			if stored {
				if err := store.Save(ctx, llmprovider.ProviderOpenAI, planSessionFor("oaiapp_mine")); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(store.Dir, "openai.client")
			if err := os.WriteFile(path, big, 0o600); err != nil {
				t.Fatal(err)
			}
			calls, given, f, err := planSignIn0082(t, store)
			if err != nil || *calls != 1 {
				t.Fatalf("the sign-in: err %v after %d login(s); want it to go on", err, *calls)
			}
			if given.Previous != nil || given.ClientID != "" {
				t.Errorf("the login was given Previous %v, client %q; want neither", given.Previous, given.ClientID)
			}
			if n := clientNotices0082(f); n != 1 {
				t.Errorf("notices %q; want exactly one about the client record", f.seenNotify)
			}
			rec, rerr := store.LoadClient(ctx, llmprovider.ProviderOpenAI)
			if rerr != nil || rec.ClientID != "oaiapp_mine" {
				t.Errorf("the record after = %+v, %v; want the new sign-in's in place of the file", rec, rerr)
			}
			for _, text := range f.allText {
				for _, bad := range []string{"OVERSIZE-SECRET", "oaiapp_big", "larger than", "65536"} {
					if strings.Contains(text, bad) {
						t.Errorf("a notice or prompt %q carries %q, from the file or its size error", text, bad)
					}
				}
			}
		})
	}
}

// The review's row "null": a client record file holding the JSON null decodes to
// the zero record, which is no record, as an absent file is: no notice, with a
// plan session of the app's own stored that session is signed in to again, with
// nothing stored a new client is registered. It is the intended reading of "no
// record" and stands, as before; this pins it. Green on `5f2c2ab`, a guard.
func TestNullClientRecordIsNoRecord_0082(t *testing.T) {
	t.Run("a plan session stored", func(t *testing.T) {
		ctx := context.Background()
		store := newClientStore(t)
		if err := store.Save(ctx, llmprovider.ProviderOpenAI, planSessionFor("oaiapp_mine")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(store.Dir, "openai.client"), []byte("null"), 0o600); err != nil {
			t.Fatal(err)
		}
		calls, given, f, err := planSignIn0082(t, store)
		if err != nil || *calls != 1 || given.Previous == nil || given.Previous.ClientID != "oaiapp_mine" || len(f.seenNotify) != 0 {
			t.Errorf("err %v, %d login(s), Previous %v, notices %q; want the stored plan session signed in to again and no notice", err, *calls, given.Previous, f.seenNotify)
		}
	})
	t.Run("nothing stored", func(t *testing.T) {
		store := newClientStore(t)
		if err := os.WriteFile(filepath.Join(store.Dir, "openai.client"), []byte("null"), 0o600); err != nil {
			t.Fatal(err)
		}
		calls, given, f, err := planSignIn0082(t, store)
		if err != nil || *calls != 1 || given.Previous != nil || given.ClientID != "" || len(f.seenNotify) != 0 {
			t.Errorf("err %v, %d login(s), Previous %v, client %q, notices %q; want a new client registered and no notice", err, *calls, given.Previous, given.ClientID, f.seenNotify)
		}
	})
}

// S14 (the fifth review's survivor, caught only by its own test): the error the
// neutral question shows passes through redact.String, so a token-shaped string
// in a locker's error does not reach the person's terminal. Green on `5f2c2ab`;
// red under the plant that drops the redaction.
func TestNeutralQuestionRedactsASecretInTheLockError_0082(t *testing.T) {
	const token = "sk-proj-abcdefghijklmnopqrstuvwxyz0123456789ABCD"
	store := &failFirst0082{failures: 1, err: errors.New("lock refused for " + token + " at /x/openai.lock")}
	p := &answers0082{confirms: []bool{true}}
	if err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t)); err != nil {
		t.Fatal(err)
	}
	asked, notices := p.seen()
	if len(asked) != 1 || len(notices) != 0 {
		t.Fatalf("questions %q, notices %q; want one question and no notice", asked, notices)
	}
	if strings.Contains(asked[0], token) || strings.Contains(asked[0], "abcdefghijklmnopqrstuvwxyz0123456789") {
		t.Errorf("the question %q carries the token", asked[0])
	}
	if !strings.Contains(asked[0], "/x/openai.lock") {
		t.Errorf("the question %q does not show the rest of the error", asked[0])
	}
}

// R-14, N-b (the review's notes, stated and pinned, no change): the neutral
// question turns newlines into spaces and removes control characters, and shows
// a long path whole; what is left of an escape sequence is harmless text. The
// error returned on "no" is the store's own, whole, with its control characters.
func TestNeutralQuestionStatedLimits_0082(t *testing.T) {
	long := "/home/" + strings.Repeat("d", 300) + "/openai.oslock"
	e := errors.New("lock \x1b[31mFAILED\x1b[0m\nline two\r\n at " + long)
	store := &failFirst0082{failures: 5, err: e}
	p := &answers0082{confirms: []bool{false}}
	err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t))
	asked, _ := p.seen()
	if len(asked) != 1 || strings.ContainsAny(asked[0], "\x1b\n\r") {
		t.Fatalf("questions %q; want one, with no escape or line break", asked)
	}
	if !strings.Contains(asked[0], "[31mFAILED[0m line two") || !strings.Contains(asked[0], long) {
		t.Errorf("question %q; want the escape's remains as text, the newlines as spaces, and the long path whole", asked[0])
	}
	if !errors.Is(err, e) || !strings.Contains(err.Error(), "\x1b[31m") {
		t.Errorf("no: err %v; want the store's own error, whole", err)
	}
}
