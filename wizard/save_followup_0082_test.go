package wizard

// 0082-PLAN Phase 1's fourth pass (the third amendment of 2026-11-17: R-9, an
// unreadable client record end to end; R-10, the retry with any store). Every
// issuer is in process.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// R-9 (the review's F-1): an unreadable openai.client, end to end, whatever
// the store holds. In none does the person get stuck; in none is a plan session
// handed over as Previous, since its client cannot be checked against the
// record; the other application's client is never adopted; the one notice
// says the record could not be read and shows no part of the error's text; and
// the new sign-in's client record replaces the unreadable one. Red when the
// engine fails on the record, when Previous is handed over, or when the notice
// carries the error.
func TestUnreadableClientRecordEndToEnd_0082(t *testing.T) {
	chatGPT := &auth.OAuthSession{TokenURL: "https://issuer.invalid/never", Provider: llmprovider.ProviderOpenAI,
		Kind: auth.SessionChatGPT, Access: "a", Refresh: "r", Issuer: auth.DefaultOpenAIIssuer,
		ClientID: auth.DefaultOpenAIClientID, Expiry: time.Now().Add(time.Hour)}
	cases := []struct {
		name   string
		stored *auth.OAuthSession
	}{
		{"a plan session stored", planSessionFor("oaiapp_mine")},
		{"nothing stored", nil},
		{"a ChatGPT login stored", chatGPT},
		{"another application's plan session stored", planSessionFor("oaiapp_other")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			store := newClientStore(t)
			if c.stored != nil {
				if err := store.Save(ctx, llmprovider.ProviderOpenAI, c.stored); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(store.Dir, "openai.client"), []byte("{{ junk-SECRET"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, readErr := store.LoadClient(ctx, llmprovider.ProviderOpenAI)
			if readErr == nil {
				t.Fatal("the planted record reads")
			}
			var given auth.ChatGPTPlanOptions
			stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
				given = opts
				return planSessionFor("oaiapp_mine"), nil
			})
			f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t), 0}, allowLeftover: true})
			_, _, err := ResolveCredential(ctx, f, Options{TokenStore: store, AgentName: planTestApp, NoPastePrompt: true},
				llmprovider.ProviderOpenAI)
			if err != nil {
				t.Fatalf("the person is stuck: %v", err)
			}
			if given.Previous != nil || given.ClientID != "" {
				t.Errorf("the login was given Previous %v, client %q; want neither: the record cannot say whose a plan session is",
					given.Previous, given.ClientID)
			}
			rec, recErr := store.LoadClient(ctx, llmprovider.ProviderOpenAI)
			if recErr != nil || rec.ClientID != "oaiapp_mine" || rec.HostID != "urn:uuid:oaiapp_mine" {
				t.Errorf("the record after the sign-in = %+v, %v; want the new sign-in's, replacing the unreadable one", rec, recErr)
			}
			told := 0
			for _, n := range f.seenNotify {
				if strings.Contains(n, "client record") && strings.Contains(n, "could not be read") {
					told++
				}
			}
			if told != 1 {
				t.Errorf("notices %q; want exactly one saying the client record could not be read", f.seenNotify)
			}
			for _, text := range f.allText {
				for _, bad := range []string{readErr.Error(), "invalid character", "junk", "SECRET", "looking for"} {
					if strings.Contains(text, bad) {
						t.Errorf("a notice or prompt %q carries %q, from the file or its read error", text, bad)
					}
				}
			}
		})
	}
}

// heldErr0082 is gobble's filelock.HeldError.
type heldErr0082 struct{ pid int }

func (e *heldErr0082) Error() string { return fmt.Sprintf("locked by pid %d", e.pid) }

// gobbleLock0082 models gobble's Logins.LockRefresh over filelock.Lock as read
// from internal/auth/logins.go and internal/filelock/lock.go: it tries the lock;
// while another process holds it, it waits with ctx, returning ctx's error
// the moment ctx is done, even for a first try; at its deadline it returns a
// *HeldError, wrapped in "auth.json: <id> refresh lock: %w". It never wraps
// llmprovider.ErrProviderUnavailable.
type gobbleLock0082 struct {
	scriptedLock0082
	deadline time.Duration
}

func (s *gobbleLock0082) LockRefresh(ctx context.Context, id llmprovider.ProviderID) (func(), error) {
	start := time.Now()
	for {
		s.mu.Lock()
		held := s.held
		s.mu.Unlock()
		if !held {
			s.log("lock")
			return func() { s.log("unlock") }, nil
		}
		if time.Since(start) >= s.deadline {
			return nil, fmt.Errorf("auth.json: %s refresh lock: %w", id, &heldErr0082{pid: 4242})
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("auth.json: %s refresh lock: %w", id, ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// R-10 (the review's F-2): a locker that reports the limit of its wait with its
// own error, as gobble's does, still gets the question: the wizard saw the lock
// held at its first try, so any failed wait but its own context's ending leads
// to the retry. Yes saves once the lock is free; no returns the locker's own
// error, whole, and saves nothing. Red when the question needs
// ErrProviderUnavailable.
func TestGobbleLikeLockerAtItsLimitOffersTheRetry_0082(t *testing.T) {
	t.Run("yes", func(t *testing.T) {
		store := &gobbleLock0082{scriptedLock0082: scriptedLock0082{held: true}, deadline: 40 * time.Millisecond}
		p := &answers0082{confirms: []bool{true}, onConfirm: store.free}
		if err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t)); err != nil {
			t.Fatalf("save = %v; want the retry to save once the lock is free", err)
		}
		if asked, notices := p.seen(); len(asked) != 1 || waitingNotice0082(notices) != 1 {
			t.Errorf("questions %q, notices %q; want one question and one waiting notice", asked, notices)
		}
		if n := saves0082(&store.scriptedLock0082); n != 1 {
			t.Errorf("%d saves, want one", n)
		}
	})
	t.Run("no", func(t *testing.T) {
		store := &gobbleLock0082{scriptedLock0082: scriptedLock0082{held: true}, deadline: 40 * time.Millisecond}
		p := &answers0082{confirms: []bool{false}}
		err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t))
		var held *heldErr0082
		if !errors.As(err, &held) || held.pid != 4242 {
			t.Errorf("err = %v; want the locker's own error, whole", err)
		}
		if n := saves0082(&store.scriptedLock0082); n != 0 {
			t.Errorf("%d saves, want none", n)
		}
	})
}

// ownErrLock0082 answers a try with a done context with an error of its own,
// unwrapped, as a locker may that does not return ctx.Err(); a wait with a
// live context blocks until the lock is free.
type ownErrLock0082 struct{ scriptedLock0082 }

func (s *ownErrLock0082) LockRefresh(ctx context.Context, _ llmprovider.ProviderID) (func(), error) {
	for {
		s.mu.Lock()
		held := s.held
		s.mu.Unlock()
		if !held {
			s.log("lock")
			return func() { s.log("unlock") }, nil
		}
		if ctx.Err() != nil {
			return nil, errors.New("lock busy") // its own error, wrapping nothing
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// R-10 (the review's N-a, W10): a try that fails for any reason while the
// sign-in's own context is alive is the lock being held, so the wizard waits,
// as it did for any locker before the try was added. A locker whose answer to a
// done context is its own error is waited for and saves; the same locker with
// the sign-in's own context ended returns that context's error, with no
// notice and no question. Red when only an error that wraps the cancelled
// context counts as held.
func TestAnyFailedTryWithALiveContextIsHeld_0082(t *testing.T) {
	t.Run("waits and saves", func(t *testing.T) {
		store := &ownErrLock0082{scriptedLock0082{held: true}}
		p := &answers0082{}
		go func() {
			time.Sleep(60 * time.Millisecond)
			store.free()
		}()
		if err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t)); err != nil {
			t.Fatalf("save = %v; want the wait to end with the save", err)
		}
		if asked, notices := p.seen(); waitingNotice0082(notices) != 1 || len(asked) != 0 {
			t.Errorf("notices %q, questions %q; want one waiting notice, no question", notices, asked)
		}
	})
	t.Run("the sign-in's own context ended", func(t *testing.T) {
		store := &ownErrLock0082{scriptedLock0082{held: true}}
		p := &answers0082{confirms: []bool{true}}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(ctx, p, descriptor0082(t))
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v; want the context's error", err)
		}
		if asked, notices := p.seen(); len(notices) != 0 || len(asked) != 0 {
			t.Errorf("notices %q, questions %q; want none", notices, asked)
		}
	})
}

// alwaysFails0082 is a locker that fails every try and every wait with an error
// of its own at once, as a store whose lock file cannot be opened does.
type alwaysFails0082 struct{ scriptedLock0082 }

func (s *alwaysFails0082) LockRefresh(context.Context, llmprovider.ProviderID) (func(), error) {
	return nil, errors.New("lock file unreadable")
}

// R-10: a lock that fails for good is reported as the wait's failure: the
// waiting notice, then the question, then the error if the person declines. It
// is the stated cost of treating any failed try as held; nothing is saved.
func TestLockThatFailsForGoodEndsWithTheQuestionThenTheError_0082(t *testing.T) {
	store := &alwaysFails0082{}
	p := &answers0082{confirms: []bool{false}}
	err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t))
	if err == nil || !strings.Contains(err.Error(), "lock file unreadable") {
		t.Errorf("err = %v; want the locker's error", err)
	}
	if asked, notices := p.seen(); len(asked) != 1 || waitingNotice0082(notices) != 1 {
		t.Errorf("questions %q, notices %q; want one of each", asked, notices)
	}
	if n := saves0082(&store.scriptedLock0082); n != 0 {
		t.Errorf("%d saves, want none", n)
	}
}

// failingSave0082 is a store whose lock is free and whose Save fails.
type failingSave0082 struct{ scriptedLock0082 }

func (s *failingSave0082) Save(context.Context, llmprovider.ProviderID, *auth.OAuthSession) error {
	s.log("save")
	return errors.New("disk full")
}

// R-10: the question is for a lock that was seen held and failed to come free,
// not for a save that fails under a lock the wizard holds: that error is
// returned, with no question and no second attempt. Red when any error leads
// to the question.
func TestSaveFailingUnderTheLockAsksNothing_0082(t *testing.T) {
	store := &failingSave0082{}
	p := &answers0082{confirms: []bool{true}}
	err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t))
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Errorf("err = %v; want the save's error", err)
	}
	if asked, notices := p.seen(); len(asked) != 0 || len(notices) != 0 {
		t.Errorf("questions %q, notices %q; want none", asked, notices)
	}
	if n := saves0082(&store.scriptedLock0082); n != 1 {
		t.Errorf("%d save attempts, want one", n)
	}
}
