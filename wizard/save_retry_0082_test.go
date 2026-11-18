package wizard

// 0082-PLAN Phase 1's third pass (the second amendment of 2026-11-17: R-6, the
// unreadable client record; R-7, the wait for the lock; N-3, the store given
// to the session before the lock). Every issuer is in process.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// scriptedLock0082 is a TokenStore and RefreshLocker that behaves as
// FileTokenStore does when another process holds the lock: a try with a
// context already done fails with an error matching ErrProviderUnavailable and
// the context's own; a wait ends with the context, or, at its bound, with an
// error matching ErrProviderUnavailable alone. held is the other holder;
// bounds are the bounds of the waits in turn (a bound that ends frees the lock
// when freesAtBound is set, as the holder's refresh ending would); a wait past
// the last is unbounded.
type scriptedLock0082 struct {
	mu            sync.Mutex
	events        []string
	held          bool
	bounds        []time.Duration
	freesAtBound  bool
	tries, waits  int
	rec           *auth.OAuthSession
	onFirstTry    func()
	ignoreContext bool // a custom locker that never looks at ctx
	hooked        bool
	// plainOnCancel makes a wait that the context ends report only
	// ErrProviderUnavailable, as a locker that does not wrap the context does.
	plainOnCancel bool
}

func (s *scriptedLock0082) log(e string) {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
}

func (s *scriptedLock0082) got() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.events...)
}

func (s *scriptedLock0082) free() {
	s.mu.Lock()
	s.held = false
	s.mu.Unlock()
}

func (s *scriptedLock0082) Load(context.Context, llmprovider.ProviderID) (*auth.OAuthSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rec, nil
}

func (s *scriptedLock0082) Save(_ context.Context, _ llmprovider.ProviderID, session *auth.OAuthSession) error {
	s.log("save")
	s.mu.Lock()
	s.rec = session
	s.mu.Unlock()
	return nil
}

func (s *scriptedLock0082) Delete(context.Context, llmprovider.ProviderID) error { return nil }

func (s *scriptedLock0082) LockRefresh(ctx context.Context, _ llmprovider.ProviderID) (func(), error) {
	s.mu.Lock()
	first := !s.hooked
	s.hooked = true
	hook := s.onFirstTry
	s.mu.Unlock()
	if first && hook != nil {
		hook()
	}
	taken := func() (func(), error) {
		s.log("lock")
		return func() { s.log("unlock") }, nil
	}
	s.mu.Lock()
	if !s.held {
		s.mu.Unlock()
		return taken()
	}
	if ctx.Err() != nil && !s.ignoreContext { // a try: the context is already done
		s.tries++
		s.mu.Unlock()
		s.log("try")
		return nil, fmt.Errorf("%w: another process is refreshing: %w", llmprovider.ErrProviderUnavailable, ctx.Err())
	}
	var bound time.Duration
	if s.waits < len(s.bounds) {
		bound = s.bounds[s.waits]
	}
	s.waits++
	s.mu.Unlock()
	s.log("wait")
	var deadline <-chan time.Time
	if bound > 0 {
		deadline = time.After(bound)
	}
	tick := time.NewTicker(2 * time.Millisecond)
	defer tick.Stop()
	for {
		s.mu.Lock()
		held := s.held
		s.mu.Unlock()
		if !held {
			return taken()
		}
		var done <-chan struct{}
		if !s.ignoreContext {
			done = ctx.Done()
		}
		select {
		case <-done:
			if s.plainOnCancel {
				return nil, fmt.Errorf("%w: another process is refreshing", llmprovider.ErrProviderUnavailable)
			}
			return nil, ctx.Err()
		case <-deadline:
			if s.freesAtBound {
				s.free()
			}
			return nil, fmt.Errorf("%w: another process is refreshing", llmprovider.ErrProviderUnavailable)
		case <-tick.C:
		}
	}
}

// answers0082 is a Prompter that answers Confirm from a script, runs onConfirm
// at each, and records every notice and question. With no answer left, Confirm
// fails, as a prompter with no terminal does.
type answers0082 struct {
	mu        sync.Mutex
	confirms  []bool
	onConfirm func()
	asked     []string
	notices   []string
}

func (a *answers0082) Select(string, []Choice, int) (int, error) { return 0, errors.New("no select") }
func (a *answers0082) MultiSelect(string, []Choice, []int) ([]int, error) {
	return nil, errors.New("no multiselect")
}
func (a *answers0082) Input(string, string) (string, error) { return "", errors.New("no input") }
func (a *answers0082) Secret(string) (string, error)        { return "", errors.New("no secret") }

func (a *answers0082) Confirm(q string, _ bool) (bool, error) {
	a.mu.Lock()
	a.asked = append(a.asked, q)
	hook := a.onConfirm
	if len(a.confirms) == 0 {
		a.mu.Unlock()
		return false, errors.New("no terminal: no answer to give")
	}
	answer := a.confirms[0]
	a.confirms = a.confirms[1:]
	a.mu.Unlock()
	if hook != nil {
		hook()
	}
	return answer, nil
}

func (a *answers0082) Notify(_ Level, format string, args ...any) {
	a.mu.Lock()
	a.notices = append(a.notices, fmt.Sprintf(format, args...))
	a.mu.Unlock()
}

func (a *answers0082) seen() (asked, notices []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.asked...), append([]string(nil), a.notices...)
}

func saves0082(s *scriptedLock0082) int {
	n := 0
	for _, e := range s.got() {
		if e == "save" {
			n++
		}
	}
	return n
}

func waitingNotice0082(notices []string) int {
	n := 0
	for _, m := range notices {
		if strings.Contains(m, "waiting for another process") {
			n++
		}
	}
	return n
}

// R-7: the wizard tries the lock once; only if it is held does it say it is
// waiting. With the lock free there is no notice.
func TestSaveSaysItIsWaitingOnlyWhenTheLockIsHeld_0082(t *testing.T) {
	t.Run("free", func(t *testing.T) {
		store := &scriptedLock0082{}
		p := &answers0082{}
		if err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t)); err != nil {
			t.Fatal(err)
		}
		if _, notices := p.seen(); len(notices) != 0 {
			t.Errorf("notices %q; want none with the lock free", notices)
		}
		if got := store.got(); strings.Join(got, ",") != "lock,save,unlock" {
			t.Errorf("events %q; want lock, save, unlock", got)
		}
	})
	t.Run("held, then freed during the wait", func(t *testing.T) {
		store := &scriptedLock0082{held: true}
		p := &answers0082{}
		go func() {
			time.Sleep(60 * time.Millisecond)
			store.free()
		}()
		if err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t)); err != nil {
			t.Fatal(err)
		}
		if asked, notices := p.seen(); waitingNotice0082(notices) != 1 || len(asked) != 0 {
			t.Errorf("notices %q, questions %q; want one waiting notice and no question", notices, asked)
		}
		if got := store.got(); strings.Join(got, ",") != "try,wait,lock,save,unlock" {
			t.Errorf("events %q; want one try, one wait, then lock, save, unlock", got)
		}
	})
}

// R-7: at the store's bound the session is kept and the person is asked
// whether to try the save again; yes repeats the wait, and the save lands once
// the lock is free.
func TestSaveRetriesAfterTheBoundOnRequest_0082(t *testing.T) {
	store := &scriptedLock0082{held: true, bounds: []time.Duration{30 * time.Millisecond}, freesAtBound: true}
	p := &answers0082{confirms: []bool{true}}
	session := newSession0082()
	if err := (resolvedCredential{session: session, saveTo: store}).save(context.Background(), p, descriptor0082(t)); err != nil {
		t.Fatal(err)
	}
	asked, notices := p.seen()
	if len(asked) != 1 || waitingNotice0082(notices) != 1 {
		t.Errorf("questions %q, notices %q; want one question and one waiting notice", asked, notices)
	}
	if n := saves0082(store); n != 1 {
		t.Errorf("%d saves, want one after the retry", n)
	}
	if got := store.got(); strings.Join(got, ",") != "try,wait,lock,save,unlock" {
		t.Errorf("events %q; want the retry to take the lock before it saves", got)
	}
	if store.rec != session {
		t.Errorf("the store holds %v; want the completed sign-in, kept and saved, not asked for again", store.rec)
	}
}

// R-7: declining the retry saves nothing and returns the store's error, an
// error matching ErrProviderUnavailable.
func TestSaveDeclinedRetrySavesNothing_0082(t *testing.T) {
	store := &scriptedLock0082{held: true, bounds: []time.Duration{20 * time.Millisecond}}
	p := &answers0082{confirms: []bool{false}}
	err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t))
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("err = %v; want an error matching ErrProviderUnavailable", err)
	}
	if asked, _ := p.seen(); len(asked) != 1 {
		t.Errorf("questions %q; want one", asked)
	}
	if n := saves0082(store); n != 0 {
		t.Errorf("%d saves, want none", n)
	}
}

// R-7: a caller with no prompter, or one that cannot answer, gets the error as
// before: the bound's, matching ErrProviderUnavailable, and nothing is saved.
func TestSaveWithoutAnAnswerReturnsTheBoundsError_0082(t *testing.T) {
	for name, p := range map[string]Prompter{"no prompter": nil, "a prompter with no terminal": &answers0082{}} {
		t.Run(name, func(t *testing.T) {
			store := &scriptedLock0082{held: true, bounds: []time.Duration{20 * time.Millisecond}}
			err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t))
			if !errors.Is(err, llmprovider.ErrProviderUnavailable) {
				t.Errorf("err = %v; want an error matching ErrProviderUnavailable", err)
			}
			if n := saves0082(store); n != 0 {
				t.Errorf("%d saves, want none", n)
			}
		})
	}
}

// R-7: the context ending ends the wait, at the first and at the second, saving
// nothing and returning the context's error.
func TestSaveCancelledAtAnyWaitSavesNothing_0082(t *testing.T) {
	t.Run("during the first wait", func(t *testing.T) {
		store := &scriptedLock0082{held: true}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			time.Sleep(60 * time.Millisecond)
			cancel()
		}()
		err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(ctx, &answers0082{}, descriptor0082(t))
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v; want an error matching context.Canceled", err)
		}
		if n := saves0082(store); n != 0 {
			t.Errorf("%d saves, want none", n)
		}
	})
	t.Run("during the wait of the retry", func(t *testing.T) {
		store := &scriptedLock0082{held: true, bounds: []time.Duration{20 * time.Millisecond}}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		p := &answers0082{confirms: []bool{true}, onConfirm: func() {
			go func() {
				time.Sleep(60 * time.Millisecond)
				cancel()
			}()
		}}
		done := make(chan error, 1)
		go func() {
			done <- (resolvedCredential{session: newSession0082(), saveTo: store}).save(ctx, p, descriptor0082(t))
		}()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("err = %v; want an error matching context.Canceled", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the retry's wait did not end with the context")
		}
		if n := saves0082(store); n != 0 {
			t.Errorf("%d saves, want none", n)
		}
	})
	t.Run("a wait the context ended, reported by the store as unavailable only", func(t *testing.T) {
		store := &scriptedLock0082{held: true, plainOnCancel: true}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		p := &answers0082{confirms: []bool{true}}
		go func() {
			time.Sleep(60 * time.Millisecond)
			cancel()
		}()
		err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(ctx, p, descriptor0082(t))
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v; want an error matching context.Canceled", err)
		}
		if asked, _ := p.seen(); len(asked) != 0 {
			t.Errorf("questions %q; want none: a sign-in that ended is not asked to retry", asked)
		}
	})
	t.Run("a context ended while the prompt was open", func(t *testing.T) {
		store := &scriptedLock0082{held: true, bounds: []time.Duration{20 * time.Millisecond}, freesAtBound: true}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		p := &answers0082{confirms: []bool{true}, onConfirm: cancel}
		err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(ctx, p, descriptor0082(t))
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v; want an error matching context.Canceled", err)
		}
		if n := saves0082(store); n != 0 {
			t.Errorf("%d saves, want none: the answer to a prompt that outlived the sign-in saves nothing", n)
		}
	})
}

// B-2 (the second review's R2W_LockIgnoringContext): a custom locker that
// ignores ctx: a save cancelled while it waits does not return until the lock
// is free, and then saves nothing.
func TestSaveWithALockerThatIgnoresTheContext_0082(t *testing.T) {
	store := &scriptedLock0082{held: true, ignoreContext: true}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- (resolvedCredential{session: newSession0082(), saveTo: store}).save(ctx, &answers0082{}, descriptor0082(t))
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)
	store.free()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v; want an error matching context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the save did not return once the lock was free")
	}
	if n := saves0082(store); n != 0 {
		t.Errorf("%d saves, want none: the context ended while the lock was held", n)
	}
}

// R-6: an unreadable client record is no record: the wizard carries on, says the
// record could not be read, and prints nothing of the file. The second
// amendment had it hand the stored plan session over as Previous; the third
// (R-9) does not, since the record cannot say whose the session is, and
// TestUnreadableClientRecordEndToEnd_0082 holds the four situations.
func TestUnreadableClientRecordIsNoRecord_0082(t *testing.T) {
	ctx := context.Background()
	store := newClientStore(t)
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, planSessionFor("oaiapp_mine")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir, "openai.client"), []byte("{{ not json: SECRET-JUNK"), 0o600); err != nil {
		t.Fatal(err)
	}
	var given auth.ChatGPTPlanOptions
	stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		given = opts
		return planSessionFor("oaiapp_mine"), nil
	})
	f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t), 0}, allowLeftover: true})
	if _, _, err := ResolveCredential(ctx, f, Options{TokenStore: store, AgentName: planTestApp, NoPastePrompt: true},
		llmprovider.ProviderOpenAI); err != nil {
		t.Fatalf("ResolveCredential: %v; want the sign-in to carry on", err)
	}
	if given.Previous != nil {
		t.Errorf("Previous = %v; want none: with no readable record the stored plan session is not signed in to again (R-9)", given.Previous)
	}
	told := false
	for _, n := range f.seenNotify {
		if strings.Contains(n, "client record") && strings.Contains(n, "could not be read") {
			told = true
		}
	}
	if !told {
		t.Errorf("notices %q; want one saying the client record could not be read", f.seenNotify)
	}
	for _, text := range f.allText {
		if strings.Contains(text, "SECRET-JUNK") || strings.Contains(text, "not json") {
			t.Errorf("a notice or prompt %q carries the file's content", text)
		}
	}
}

// N-3 (the second review): a refresh of the new session between the moment it
// is given the store and the moment the wizard holds the lock could adopt the
// store's old record of the same labels, and the save would write the old
// tokens. The store is given to the session only under the lock, just before
// the save. Red when it is given earlier.
func TestNewSessionIsGivenTheStoreOnlyUnderTheLock_0082(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "fresh-access", "refresh_token": "fresh-refresh", "expires_in": 3600})
	}))
	defer srv.Close()
	old := newSession0082()
	old.Access, old.Refresh, old.TokenURL = "old-access", "old-refresh", srv.URL+"/token"
	store := &scriptedLock0082{rec: old}
	session := newSession0082()
	session.TokenURL, session.HTTPClient = srv.URL+"/token", srv.Client()
	// Between the wizard's first look at the lock and its taking it, another
	// goroutine of the process asks the session for a token after a 401: the
	// moment the review described.
	store.onFirstTry = func() {
		session.Invalidate()
		if _, err := session.Token(context.Background()); err != nil {
			t.Errorf("the concurrent token request: %v", err)
		}
		_ = session.Wait(context.Background())
	}
	if err := (resolvedCredential{session: session, saveTo: store}).save(context.Background(), &answers0082{}, descriptor0082(t)); err != nil {
		t.Fatal(err)
	}
	rec, _ := store.Load(context.Background(), llmprovider.ProviderOpenAI)
	if rec == nil || rec.Refresh == "old-refresh" {
		t.Errorf("the store holds %v; want the new sign-in's own tokens, not the old record's", rec)
	}
}
