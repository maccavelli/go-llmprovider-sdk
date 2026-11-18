package wizard

// 0082-PLAN Phase 1's fifth pass (the fourth amendment of 2026-11-18: R-12, only
// an undecodable client record is none; R-13, "another process" only when the
// lock was held). Every issuer is in process.

import (
	"context"
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// countingClients is a FileTokenStore whose client record reads are counted, and
// whose reads can be made to fail first with err.
type countingClients struct {
	*auth.FileTokenStore
	mu    sync.Mutex
	loads int
	fail  []error // the errors of the next reads, in turn
}

func (c *countingClients) LoadClient(ctx context.Context, id llmprovider.ProviderID) (auth.ClientRecord, error) {
	c.mu.Lock()
	c.loads++
	var err error
	if len(c.fail) > 0 {
		err, c.fail = c.fail[0], c.fail[1:]
	}
	c.mu.Unlock()
	if err != nil {
		return auth.ClientRecord{}, err
	}
	return c.FileTokenStore.LoadClient(ctx, id)
}

func (c *countingClients) reads() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loads
}

// planSignIn0082 runs the wizard's plan sign-in against store, with the plan
// login stubbed, and returns the login's calls and options, the fake prompter,
// and the sign-in's error.
func planSignIn0082(t *testing.T, store auth.TokenStore) (calls *int, given *auth.ChatGPTPlanOptions, f *fakePrompter, err error) {
	t.Helper()
	given = new(auth.ChatGPTPlanOptions)
	calls = stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		*given = opts
		return planSessionFor("oaiapp_mine"), nil
	})
	f = newFake(t, fakePrompter{selects: []int{planMethodIdx(t), 0}, allowLeftover: true})
	_, _, err = ResolveCredential(context.Background(), f, Options{TokenStore: store, AgentName: planTestApp, NoPastePrompt: true},
		llmprovider.ProviderOpenAI)
	return calls, given, f, err
}

// clientNotices0082 counts the notices that speak of the client record.
func clientNotices0082(f *fakePrompter) int {
	n := 0
	for _, s := range f.seenNotify {
		if strings.Contains(s, "client record") {
			n++
		}
	}
	return n
}

// R-12 (0082-MADR, fourth amendment): a client record that exists but cannot be
// read for a reason other than its content, here a permission refused, fails
// the sign-in with an error that names the file and none of its content. The
// error is injected, as the operating system's own is, so that every platform
// runs it; the real file with mode 0 is in client_read_unix_0082_test.go. The
// login does not run, so no client is registered; the record and the saved
// session are as they were; no notice says the record is replaced. Red while
// any read error is an unreadable record.
func TestClientRecordPermissionErrorFailsTheSignIn_0082(t *testing.T) {
	ctx := context.Background()
	fs := newClientStore(t)
	denied := &iofs.PathError{Op: "open", Path: filepath.Join(fs.Dir, "openai.client"), Err: iofs.ErrPermission}
	store := &countingClients{FileTokenStore: fs, fail: []error{denied}}
	mine := auth.ClientRecord{ClientID: "oaiapp_mine", HostID: "urn:uuid:SECRET-HOST"}
	if err := store.SaveClient(ctx, llmprovider.ProviderOpenAI, mine); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, planSessionFor("oaiapp_mine")); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(fs.Dir, "openai.client"))
	if err != nil {
		t.Fatal(err)
	}
	calls, _, f, err := planSignIn0082(t, store)
	if !errors.Is(err, iofs.ErrPermission) || !strings.Contains(err.Error(), "openai.client") {
		t.Fatalf("err = %v; want the sign-in failed with the permission error, naming openai.client", err)
	}
	for _, bad := range []string{"SECRET-HOST", "oaiapp_mine", "urn:uuid"} {
		if strings.Contains(err.Error(), bad) {
			t.Errorf("the error %q carries %q from the record", err, bad)
		}
	}
	if *calls != 0 || clientNotices0082(f) != 0 {
		t.Errorf("the login ran %d time(s) and notices were %q; want neither: no client is registered and the record is not replaced", *calls, f.seenNotify)
	}
	after, err := os.ReadFile(filepath.Join(fs.Dir, "openai.client"))
	if err != nil || string(after) != string(before) {
		t.Errorf("the record after = %q, %v; want it as it was", after, err)
	}
	if s, err := fs.Load(ctx, llmprovider.ProviderOpenAI); err != nil || s == nil || s.ClientID != "oaiapp_mine" {
		t.Errorf("the saved session = %v, %v; want it untouched", s, err)
	}
}

// R-12: a read that fails once for a transient reason fails that sign-in the
// same way, with the record untouched and the login never run; the next sign-in
// reads again and goes on. Red while the error is none.
func TestTransientClientReadFailsOnlyThatSignIn_0082(t *testing.T) {
	ctx := context.Background()
	store := &countingClients{FileTokenStore: newClientStore(t), fail: []error{errors.New("read openai.client: i/o timeout")}}
	mine := auth.ClientRecord{ClientID: "oaiapp_mine", HostID: "urn:uuid:mine"}
	if err := store.SaveClient(ctx, llmprovider.ProviderOpenAI, mine); err != nil {
		t.Fatal(err)
	}
	calls, _, f, err := planSignIn0082(t, store)
	if err == nil || !strings.Contains(err.Error(), "i/o timeout") {
		t.Fatalf("err = %v; want the read's error", err)
	}
	if *calls != 0 || clientNotices0082(f) != 0 {
		t.Errorf("the login ran %d time(s) and notices were %q; want neither", *calls, f.seenNotify)
	}
	if rec, rerr := store.FileTokenStore.LoadClient(ctx, llmprovider.ProviderOpenAI); rerr != nil || rec != mine {
		t.Errorf("the record reads %+v, %v; want it untouched", rec, rerr)
	}
	if got := store.reads(); got != 1 {
		t.Errorf("%d reads for the failed sign-in, want one", got)
	}
	calls, _, _, err = planSignIn0082(t, store)
	if err != nil || *calls != 1 {
		t.Errorf("the next sign-in: err %v after %d login(s); want it to go on", err, *calls)
	}
	if got := store.reads(); got != 2 {
		t.Errorf("%d reads for the two sign-ins, want two", got)
	}
}

// R-12 (the single read): one sign-in reads the client record once, the wizard's
// read handed to the engine, with no record, with a record a stored plan session
// signs in to again, with another application's session stored, and with an
// undecodable record. Red while the engine reads the record again.
func TestOneClientRecordReadPerSignIn_0082(t *testing.T) {
	cases := []struct {
		name    string
		record  *auth.ClientRecord
		stored  *auth.OAuthSession
		corrupt bool
	}{
		{"nothing saved", nil, nil, false},
		{"a record and its plan session", &auth.ClientRecord{ClientID: "oaiapp_mine", HostID: "urn:uuid:oaiapp_mine"}, planSessionFor("oaiapp_mine"), false},
		{"a record and another application's session", &auth.ClientRecord{ClientID: "oaiapp_mine", HostID: "urn:uuid:oaiapp_mine"}, planSessionFor("oaiapp_other"), false},
		{"a record and no session", &auth.ClientRecord{ClientID: "oaiapp_mine", HostID: "urn:uuid:oaiapp_mine"}, nil, false},
		{"an undecodable record", nil, planSessionFor("oaiapp_mine"), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			store := &countingClients{FileTokenStore: newClientStore(t)}
			if c.record != nil {
				if err := store.SaveClient(ctx, llmprovider.ProviderOpenAI, *c.record); err != nil {
					t.Fatal(err)
				}
			}
			if c.stored != nil {
				if err := store.Save(ctx, llmprovider.ProviderOpenAI, c.stored); err != nil {
					t.Fatal(err)
				}
			}
			if c.corrupt {
				if err := os.WriteFile(filepath.Join(store.Dir, "openai.client"), []byte("{{ junk"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			calls, _, _, err := planSignIn0082(t, store)
			if err != nil || *calls != 1 {
				t.Fatalf("the sign-in: err %v after %d login(s)", err, *calls)
			}
			if got := store.reads(); got != 1 {
				t.Errorf("%d reads of the client record in one sign-in, want one", got)
			}
		})
	}
}

// R-12: an undecodable record is still replaced, with the one notice, whatever
// the wizard hands to the engine: the file written by an older build, junk, or
// a value of the wrong type for a field.
func TestUndecodableRecordsAreReplacedWithOneNotice_0082(t *testing.T) {
	for name, content := range map[string]string{
		"junk":            "{{ junk",
		"a wrong type":    `{"client_id": 17}`,
		"an empty file":   "",
		"a truncated one": `{"client_id": "oaiapp_x"`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := &countingClients{FileTokenStore: newClientStore(t)}
			if err := os.WriteFile(filepath.Join(store.Dir, "openai.client"), []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			calls, _, f, err := planSignIn0082(t, store)
			if err != nil || *calls != 1 {
				t.Fatalf("the sign-in: err %v after %d login(s); want it to go on", err, *calls)
			}
			if n := clientNotices0082(f); n != 1 {
				t.Errorf("notices %q; want exactly one about the client record", f.seenNotify)
			}
			rec, rerr := store.FileTokenStore.LoadClient(ctx, llmprovider.ProviderOpenAI)
			if rerr != nil || rec.ClientID != "oaiapp_mine" {
				t.Errorf("the record after = %+v, %v; want the new sign-in's", rec, rerr)
			}
		})
	}
}

// failFirst0082 is a locker whose first failures tries fail with err, and which
// then behaves as scriptedLock0082 does. It counts the calls made with a live
// context, which are waits, and every call.
type failFirst0082 struct {
	scriptedLock0082
	failures int
	err      error
	calls    int
	live     int
}

func (s *failFirst0082) LockRefresh(ctx context.Context, id llmprovider.ProviderID) (func(), error) {
	s.mu.Lock()
	s.calls++
	if ctx.Err() == nil {
		s.live++
	}
	fail := s.calls <= s.failures
	s.mu.Unlock()
	if fail {
		return nil, s.err
	}
	return s.scriptedLock0082.LockRefresh(ctx, id)
}

func (s *failFirst0082) counts() (calls, live int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls, s.live
}

// neutralQuestion0082 is the question R-13 asks when a try to take the lock
// fails and the lock was not seen held.
func neutralQuestion0082(label, why string) string {
	return fmt.Sprintf("The lock on the saved %s session could not be taken (%s). Try saving the new sign-in again?", label, why)
}

// R-13 (0082-MADR, fourth amendment): a try that fails with an error that does
// not wrap the cancelled context is not "another process holds the lock". There
// is no waiting notice, no wait, and no question or notice that says another
// process is refreshing; one question asks, in neutral words, whether to try
// again, and shows the real error. Yes repeats the try, which saves; no returns
// the error whole and saves nothing. Red while any failed try is waited for and
// called another process.
func TestFailedTryThatIsNotContentionAsksInNeutralWords_0082(t *testing.T) {
	const reason = "open /x/openai.lock: permission denied"
	t.Run("yes", func(t *testing.T) {
		store := &failFirst0082{failures: 1, err: errors.New(reason)}
		p := &answers0082{confirms: []bool{true}}
		d := descriptor0082(t)
		if err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, d); err != nil {
			t.Fatalf("save = %v; want the second try to save", err)
		}
		asked, notices := p.seen()
		if len(asked) != 1 || asked[0] != neutralQuestion0082(d.Label, reason) {
			t.Errorf("questions %q; want the one neutral question %q", asked, neutralQuestion0082(d.Label, reason))
		}
		if len(notices) != 0 {
			t.Errorf("notices %q; want none", notices)
		}
		if calls, live := store.counts(); calls != 2 || live != 0 {
			t.Errorf("%d lock calls, %d of them waits; want two tries and no wait", calls, live)
		}
		if n := saves0082(&store.scriptedLock0082); n != 1 {
			t.Errorf("%d saves, want one", n)
		}
	})
	t.Run("no", func(t *testing.T) {
		store := &failFirst0082{failures: 5, err: errors.New(reason)}
		p := &answers0082{confirms: []bool{false}}
		err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t))
		if err == nil || !strings.Contains(err.Error(), reason) {
			t.Errorf("err = %v; want the locker's error", err)
		}
		if n := saves0082(&store.scriptedLock0082); n != 0 {
			t.Errorf("%d saves, want none", n)
		}
	})
	t.Run("yes twice, then no", func(t *testing.T) {
		store := &failFirst0082{failures: 5, err: errors.New(reason)}
		p := &answers0082{confirms: []bool{true, true, false}}
		err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t))
		if err == nil || !strings.Contains(err.Error(), reason) {
			t.Errorf("err = %v; want the locker's error", err)
		}
		asked, notices := p.seen()
		if len(asked) != 3 || len(notices) != 0 {
			t.Errorf("questions %q, notices %q; want three questions and no notice", asked, notices)
		}
		if calls, live := store.counts(); calls != 3 || live != 0 {
			t.Errorf("%d lock calls, %d waits; want three tries and no wait", calls, live)
		}
	})
	t.Run("nothing says another process", func(t *testing.T) {
		store := &failFirst0082{failures: 1, err: errors.New(reason)}
		p := &answers0082{confirms: []bool{true}}
		_ = (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t))
		asked, notices := p.seen()
		for _, text := range append(asked, notices...) {
			if s := strings.ToLower(text); strings.Contains(s, "another process") || strings.Contains(s, "waiting") || strings.Contains(s, "still refreshing") {
				t.Errorf("%q says the lock is held by another process", text)
			}
		}
	})
	t.Run("no prompter", func(t *testing.T) {
		store := &failFirst0082{failures: 5, err: errors.New(reason)}
		err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), nil, descriptor0082(t))
		if err == nil || !strings.Contains(err.Error(), reason) {
			t.Errorf("err = %v; want the locker's error", err)
		}
	})
}

// R-13: the error shown in the question is the real one, cleaned of what a
// terminal acts on, so that a lock path or a message from a custom locker
// cannot move the cursor or hide the question.
func TestNeutralQuestionShowsTheErrorWithoutControlCharacters_0082(t *testing.T) {
	store := &failFirst0082{failures: 1, err: errors.New("bad\x1b[2J lock\nfile\r\x07")}
	p := &answers0082{confirms: []bool{true}}
	d := descriptor0082(t)
	if err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, d); err != nil {
		t.Fatal(err)
	}
	asked, _ := p.seen()
	if len(asked) != 1 {
		t.Fatalf("questions %q; want one", asked)
	}
	if strings.ContainsAny(asked[0], "\x1b\x07\n\r") {
		t.Errorf("the question %q carries a control character", asked[0])
	}
	if !strings.Contains(asked[0], "lock") || !strings.Contains(asked[0], "file") {
		t.Errorf("the question %q does not show the error", asked[0])
	}
}

// R-13: contention is the first try's error wrapping the cancelled context, and
// only that: such a try is waited for with the notice, as before, whatever else
// the error wraps, and a locker that wraps nothing else is still contention.
func TestFirstTryWrappingTheContextErrorIsContention_0082(t *testing.T) {
	for name, tryErr := range map[string]error{
		"the context's error alone":   context.Canceled,
		"wrapped in a message":        fmt.Errorf("auth.json: refresh lock: %w", context.Canceled),
		"with ErrProviderUnavailable": fmt.Errorf("%w: busy: %w", llmprovider.ErrProviderUnavailable, context.Canceled),
	} {
		t.Run(name, func(t *testing.T) {
			store := &failFirst0082{failures: 1, err: tryErr}
			p := &answers0082{}
			if err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, descriptor0082(t)); err != nil {
				t.Fatalf("save = %v; want the wait to save", err)
			}
			asked, notices := p.seen()
			if waitingNotice0082(notices) != 1 || len(asked) != 0 {
				t.Errorf("notices %q, questions %q; want one waiting notice and no question", notices, asked)
			}
			if calls, live := store.counts(); calls != 2 || live != 1 {
				t.Errorf("%d lock calls, %d waits; want a try and one wait", calls, live)
			}
		})
	}
}

// R-13 (the fourth review's TestR4W_LockThatCanNeverBeTaken, with FileTokenStore
// itself): the lock file's path is a directory, so the lock can never be taken
// and no other process holds it. Yes, yes, no is three tries and the store's
// error, at once; no question or notice says another process is refreshing or
// that the save is waiting; every question shows the store's own error. Red
// while the failed first try is called another process and waited for.
func TestLockFileThatCanNeverBeOpenedAsksNeutrally_0082(t *testing.T) {
	store := newClientStore(t)
	if err := os.Mkdir(filepath.Join(store.Dir, "openai.oslock"), 0o700); err != nil {
		t.Fatal(err)
	}
	f := newFake(t, fakePrompter{confirms: []bool{true, true, false}, allowLeftover: true})
	err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), f, descriptor0082(t))
	if err == nil || !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Fatalf("save = %v; want the store's error", err)
	}
	if len(f.seenConfirm) != 3 || len(f.seenNotify) != 0 {
		t.Errorf("questions %q, notices %q; want three questions and no notice", f.seenConfirm, f.seenNotify)
	}
	for _, text := range append(append([]string(nil), f.seenConfirm...), f.seenNotify...) {
		s := strings.ToLower(text)
		if strings.Contains(s, "another process") || strings.Contains(s, "waiting") || strings.Contains(s, "still refreshing") {
			t.Errorf("%q says another process holds the lock, which nothing does", text)
		}
		if !strings.Contains(s, "openai.oslock") {
			t.Errorf("%q does not show the store's error, which names the lock file", text)
		}
	}
}

// R-13 (the fourth pass's Q15, whose guard the narrowing left without a case): a
// try whose error wraps the cancelled context, while the sign-in's own context
// has ended, is that context's ending and not a held lock: no notice says the
// save is waiting, no wait follows, no question is asked, and the error is the
// context's. Red when the check of the sign-in's own context is gone, which the
// plant "a try that fails with an ended context still waits" makes.
func TestSignInContextEndedAtTheTryIsNotAHeldLock_0082(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &scriptedLock0082{held: true, onFirstTry: cancel}
	p := &answers0082{confirms: []bool{true}}
	err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(ctx, p, descriptor0082(t))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("save = %v; want the context's error", err)
	}
	if asked, notices := p.seen(); len(asked) != 0 || len(notices) != 0 {
		t.Errorf("questions %q, notices %q; want none", asked, notices)
	}
	if n := saves0082(store); n != 0 {
		t.Errorf("%d saves, want none", n)
	}
}

// R-13: a wait that fails after contention keeps the "another process" wording
// of the existing question, and the question for a failed try is not used for it.
func TestQuestionAfterContentionKeepsItsWords_0082(t *testing.T) {
	store := &scriptedLock0082{held: true, bounds: []time.Duration{30 * time.Millisecond}, freesAtBound: true}
	p := &answers0082{confirms: []bool{true}}
	d := descriptor0082(t)
	if err := (resolvedCredential{session: newSession0082(), saveTo: store}).save(context.Background(), p, d); err != nil {
		t.Fatal(err)
	}
	asked, _ := p.seen()
	want := fmt.Sprintf("Another process is still refreshing the saved %s session. Try saving the new sign-in again?", d.Label)
	if len(asked) != 1 || asked[0] != want {
		t.Errorf("questions %q; want %q", asked, want)
	}
}
