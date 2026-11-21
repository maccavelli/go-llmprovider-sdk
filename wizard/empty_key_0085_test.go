package wizard

// 0085-MADR: a typed or pasted key that is empty after trimming is asked for
// again, up to three times, and then the run ends with an error. Every
// credential here is a made-up string or blank; every request is answered in
// process; no outside host (0065-MADR).

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// blankInputs0085 are values a person or a caller's Prompter may return that
// are empty once trimmed: empty, spaces, a tab, CR+LF, a non-breaking space,
// a mix, a lone CR and a lone LF (MADR ST1 to ST4).
var blankInputs0085 = []struct{ name, value string }{
	{"empty", ""},
	{"spaces", "   "},
	{"tab", "\t"},
	{"crlf", "\r\n"},
	{"nbsp", "\u00a0"},
	{"mixed", " \t  \n"},
	{"cr", "\r"},
	{"lf", "\n"},
}

// keyProviders0085 are the built-in providers whose descriptor says a
// credential must be collected.
func keyProviders0085(t *testing.T) []llmprovider.Descriptor {
	t.Helper()
	var need []llmprovider.Descriptor
	for _, d := range providers.Default().Descriptors() {
		if d.RequiresAPIKey {
			need = append(need, d)
		}
	}
	if len(need) != 9 {
		t.Fatalf("%d built-in providers require a key; want 9", len(need))
	}
	return need
}

// apiKeyMenu0085 is the selects that reach d's API-key method under o: the
// menu's index when the menu is shown, none when it is not.
func apiKeyMenu0085(t *testing.T, d llmprovider.Descriptor, o Options) []int {
	t.Helper()
	methods := offeredAuthMethods(d, o)
	if len(methods) == 0 || (len(methods) == 1 && methods[0].ID == llmprovider.AuthAPIKey) {
		return nil
	}
	for i, m := range methods {
		if m.ID == llmprovider.AuthAPIKey {
			return []int{i}
		}
	}
	t.Fatalf("%s offers no API-key method", d.ID)
	return nil
}

// pasteIdx0085 is the index of d's pasted-credential method under o.
func pasteIdx0085(t *testing.T, d llmprovider.Descriptor, o Options) int {
	t.Helper()
	for i, m := range offeredAuthMethods(d, o) {
		if m.ID == llmprovider.AuthTokenStdin {
			return i
		}
	}
	t.Fatalf("%s offers no pasted-credential method", d.ID)
	return -1
}

// zero0085 reports whether res is the zero Result: a run that ended with an
// error returns none.
func zero0085(res Result) bool { return reflect.DeepEqual(res, Result{}) }

func options0085(t *testing.T) Options {
	return Options{NoEndpointPrompt: true, ProviderOptions: []llmprovider.Option{testCache(t)}}
}

// resolveTyped0085 runs ResolveCredential for d under o with secrets scripted
// to the Prompter, reaching the API-key method.
func resolveTyped0085(t *testing.T, d llmprovider.Descriptor, o Options, secrets []string) (Result, *fakePrompter, error) {
	t.Helper()
	f := newFake(t, fakePrompter{selects: apiKeyMenu0085(t, d, o), secrets: secrets, allowLeftover: true})
	res, _, err := ResolveCredential(context.Background(), f, o, d.ID)
	return res, f, err
}

func noKeyNotice0085(d llmprovider.Descriptor) string {
	return fmt.Sprintf("no key was entered; %s needs one", d.Label)
}

func noKeyError0085(d llmprovider.Descriptor) string {
	return fmt.Sprintf("wizard: no %s key entered after 3 tries", d.Label)
}

// TestTypedBlankKeyIsRefused_0085 (ST1 to ST4, ST23, ST26, ST27, ST29, ST52,
// ST63, ST64, ST65, ST68): a Prompter that returns a blank value, then a key,
// is asked twice for every provider that requires a key, Kilo, OpenCode Zen
// and OpenCode Go among them, and the Result carries the second value, after
// one notice. Kilo is run without a TokenStore, where it shows no method menu,
// and with one, where it does; Grok and OpenAI show the menu either way.
func TestTypedBlankKeyIsRefused_0085(t *testing.T) {
	for _, d := range keyProviders0085(t) {
		stores := []bool{false}
		if d.ID == llmprovider.ProviderKilo {
			stores = []bool{false, true}
		}
		for _, withStore := range stores {
			wantMenu := d.ID == llmprovider.ProviderGrok || d.ID == llmprovider.ProviderOpenAI ||
				(d.ID == llmprovider.ProviderKilo && withStore)
			for _, in := range blankInputs0085 {
				name := fmt.Sprintf("%s/store=%v/%s", d.ID, withStore, in.name)
				t.Run(name, func(t *testing.T) {
					o := options0085(t)
					if withStore {
						o.TokenStore = newMemoryTokenStore()
					}
					res, f, err := resolveTyped0085(t, d, o, []string{in.value, "made-up-key-1234"})
					if err != nil || res.Kind != CredAPIKey || res.APIKey != "made-up-key-1234" || len(f.seenSecret) != 2 ||
						countContaining(f.seenNotify, noKeyNotice0085(d)) != 1 {
						t.Errorf("err %v, kind %q, key %q, Secret asked %d time(s), notices %q; want the second key after two asks and one notice",
							err, res.Kind, res.APIKey, len(f.seenSecret), f.seenNotify)
					}
					if n := len(f.seenSelect); wantMenu && n != 1 || !wantMenu && n != 0 {
						t.Errorf("method menu asked %d time(s); want it asked: %v", len(f.seenSelect), wantMenu)
					}
				})
			}
		}
	}

	// ST68: a caller's descriptor that requires a key, in a Registry of the
	// caller's own, is refused alike; nothing here names a provider id.
	t.Run("registered descriptor", func(t *testing.T) {
		const id llmprovider.ProviderID = "acme-0085"
		reg := llmprovider.NewRegistry()
		d := llmprovider.Descriptor{ID: id, Label: "Acme", RequiresAPIKey: true}
		if err := reg.Register(d, func(...llmprovider.Option) (llmprovider.Provider, error) {
			return nil, errors.New("not built by this test")
		}); err != nil {
			t.Fatal(err)
		}
		o := options0085(t)
		o.Registry = reg
		f := newFake(t, fakePrompter{secrets: []string{"  ", "made-up-key-1234"}})
		res, _, err := ResolveCredential(context.Background(), f, o, id)
		if err != nil || res.APIKey != "made-up-key-1234" || len(f.seenSecret) != 2 ||
			countContaining(f.seenNotify, noKeyNotice0085(d)) != 1 {
			t.Errorf("err %v, key %q, Secret asked %d time(s), notices %q; want the second key after two asks and one notice",
				err, res.APIKey, len(f.seenSecret), f.seenNotify)
		}
	})
}

// TestConfigureBlankKeyIsRefused_0085 (ST26, ST58): ConfigureLLM shares the
// credential step, so a blank typed key is asked for again there too.
func TestConfigureBlankKeyIsRefused_0085(t *testing.T) {
	f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderTogether), 0},
		secrets: []string{"   ", "made-up-key-1234"}})
	res, err := ConfigureLLM(context.Background(), f, options0085(t))
	if err != nil || res.APIKey != "made-up-key-1234" || len(f.seenSecret) != 2 {
		t.Errorf("err %v, key %q, Secret asked %d time(s); want the second key after two asks", err, res.APIKey, len(f.seenSecret))
	}
}

// TestPastedBlankKeyIsRefused_0085 (ST32, ST33, ST64): Grok's pasted key, and
// OpenAI's pasted credential, ask again for a blank paste. OpenAI's was an
// access-token error at once.
func TestPastedBlankKeyIsRefused_0085(t *testing.T) {
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderGrok, llmprovider.ProviderOpenAI} {
		t.Run(string(id), func(t *testing.T) {
			d, _ := providers.Default().Descriptor(id)
			o := options0085(t)
			o.TokenStore = newMemoryTokenStore()
			f := newFake(t, fakePrompter{selects: []int{pasteIdx0085(t, d, o)}, secrets: []string{"  \t ", "sk-made-up-key-1234"}})
			res, _, err := ResolveCredential(context.Background(), f, o, d.ID)
			if err != nil || res.Kind != CredAPIKey || res.APIKey != "sk-made-up-key-1234" || len(f.seenSecret) != 2 ||
				countContaining(f.seenNotify, noKeyNotice0085(d)) != 1 {
				t.Errorf("err %v, kind %q, key %q, Secret asked %d time(s), notices %q; want the second paste after two asks",
					err, res.Kind, res.APIKey, len(f.seenSecret), f.seenNotify)
			}
		})
	}
}

// foreverBlank0085 is a Prompter whose Secret always answers blank: a
// scripted or non-interactive caller, or one that says "cancelled" with "".
type foreverBlank0085 struct {
	*fakePrompter
	asks int
}

func (p *foreverBlank0085) Secret(string) (string, error) {
	p.asks++
	if p.asks > 50 {
		return "", errors.New("stopped by the test after 50 asks")
	}
	return "  ", nil
}

// TestForeverBlankPrompterEnds_0085 (ST11 to ST14, ST19, ST65): a Prompter
// that never returns a key is asked exactly three times, gives exactly two
// notices, and ends the run with the no-key error and no Result, for each of
// the nine providers. The error matches no sentinel.
func TestForeverBlankPrompterEnds_0085(t *testing.T) {
	for _, d := range keyProviders0085(t) {
		t.Run(string(d.ID), func(t *testing.T) {
			o := options0085(t)
			p := &foreverBlank0085{fakePrompter: newFake(t, fakePrompter{selects: apiKeyMenu0085(t, d, o)})}
			res, _, err := ResolveCredential(context.Background(), p, o, d.ID)
			wantNotes := []string{noKeyNotice0085(d), noKeyNotice0085(d)}
			if err == nil || err.Error() != noKeyError0085(d) || !zero0085(res) || p.asks != maxTypedKeys ||
				fmt.Sprint(p.seenNotify) != fmt.Sprint(wantNotes) {
				t.Errorf("err %v, result %+v, asks %d, notices %q; want %q, no result, %d asks, notices %q",
					err, res, p.asks, p.seenNotify, noKeyError0085(d), maxTypedKeys, wantNotes)
			}
			for name, target := range map[string]error{"ErrAuthFailure": llmprovider.ErrAuthFailure,
				"ErrInvalidRequest": llmprovider.ErrInvalidRequest, "context.Canceled": context.Canceled, "io.EOF": io.EOF} {
				if errors.Is(err, target) {
					t.Errorf("err %v matches %s; want it to match no sentinel", err, name)
				}
			}
		})
	}
}

// TestForeverBlankPasteEnds_0085 (ST13, ST32, ST33): a Prompter that never
// pastes a credential is asked exactly three times at the paste prompt, Grok's
// and OpenAI's, gives exactly two notices, and ends the run with the same
// no-key error as the typed prompt, text and all, no "enter credential: "
// prefix, no Result, and no OpenAI access-token error.
func TestForeverBlankPasteEnds_0085(t *testing.T) {
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderGrok, llmprovider.ProviderOpenAI} {
		t.Run(string(id), func(t *testing.T) {
			d, _ := providers.Default().Descriptor(id)
			o := options0085(t)
			o.TokenStore = newMemoryTokenStore()
			p := &foreverBlank0085{fakePrompter: newFake(t, fakePrompter{selects: []int{pasteIdx0085(t, d, o)}})}
			res, _, err := ResolveCredential(context.Background(), p, o, d.ID)
			wantNotes := []string{noKeyNotice0085(d), noKeyNotice0085(d)}
			if err == nil || err.Error() != noKeyError0085(d) || !zero0085(res) || p.asks != maxTypedKeys ||
				fmt.Sprint(p.seenNotify) != fmt.Sprint(wantNotes) {
				t.Errorf("err %v, result %+v, asks %d, notices %q; want %q, no result, %d asks, notices %q",
					err, res, p.asks, p.seenNotify, noKeyError0085(d), maxTypedKeys, wantNotes)
			}
			for name, target := range map[string]error{"ErrAuthFailure": llmprovider.ErrAuthFailure,
				"ErrInvalidRequest": llmprovider.ErrInvalidRequest, "context.Canceled": context.Canceled, "io.EOF": io.EOF} {
				if errors.Is(err, target) {
					t.Errorf("err %v matches %s; want it to match no sentinel", err, name)
				}
			}
		})
	}
}

// TestKeyWhitespaceKinds_0085 (ST5 to ST10) pins what is not refused: a padded
// key is trimmed, and a quoted key, inner spaces, a control character, a very
// long key, a lone zero-width space and a value that is not UTF-8 are returned
// as they are, after one ask and with no notice (MADR D4, Q1).
func TestKeyWhitespaceKinds_0085(t *testing.T) {
	d, _ := providers.Default().Descriptor(llmprovider.ProviderTogether)
	long := strings.Repeat("k", 1<<20)
	for _, c := range []struct{ name, in, want string }{
		{"padded", "  made-up-key  ", "made-up-key"},
		{"trailing newline", "made-up-key\n", "made-up-key"},
		{"trailing crlf", "made-up-key\r\n", "made-up-key"},
		{"nbsp padded", "\u00a0made-up-key\u00a0", "made-up-key"},
		{"inner space", "made up key", "made up key"},
		{"double quoted", "\"made-up-key\"", "\"made-up-key\""},
		{"single quoted", "'made-up-key'", "'made-up-key'"},
		{"two quotes", "\"\"", "\"\""},
		{"inner line feed", "made-up\nkey", "made-up\nkey"},
		{"inner nul", "made-up\x00key", "made-up\x00key"},
		{"inner escape", "made-up\x1bkey", "made-up\x1bkey"},
		{"zero-width space", "\u200b", "\u200b"},
		{"zero-width in key", "made-up\u200bkey", "made-up\u200bkey"},
		{"one mebibyte", long, long},
		{"invalid utf-8", "\xff", "\xff"},
		{"lone a0 byte", "\xa0", "\xa0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			res, f, err := resolveTyped0085(t, d, options0085(t), []string{c.in})
			if err != nil || res.APIKey != c.want || len(f.seenSecret) != 1 || len(f.seenNotify) != 0 {
				t.Errorf("err %v, key length %d, asks %d, notices %q; want the key of length %d after one ask and no notice",
					err, len(res.APIKey), len(f.seenSecret), f.seenNotify, len(c.want))
			}
		})
	}
}

// errKey0085 is a Prompter's own error.
var errKey0085 = errors.New("the prompter's own error")

// answer0085 is one reply of a scripted Secret.
type answer0085 struct {
	value string
	err   error
}

// answering0085 is a Prompter whose Secret replays replies, then fails.
type answering0085 struct {
	*fakePrompter
	replies []answer0085
	asks    int
}

func (p *answering0085) Secret(string) (string, error) {
	p.asks++
	if len(p.replies) == 0 {
		return "", errors.New("answering0085: no reply left")
	}
	r := p.replies[0]
	p.replies = p.replies[1:]
	return r.value, r.err
}

// TestKeyPrompterErrorIsReturned_0085 (ST16 to ST18, ST22, ST24): a Prompter's
// own error from Secret ends the run at once and errors.Is matches it, at the
// typed and the pasted prompt: on the first ask, after one blank (one notice,
// two asks), and together with a value, which is not used. A Ctrl-C at the
// raw-mode prompt matches context.Canceled.
func TestKeyPrompterErrorIsReturned_0085(t *testing.T) {
	together, _ := providers.Default().Descriptor(llmprovider.ProviderTogether)
	grok, _ := providers.Default().Descriptor(llmprovider.ProviderGrok)
	base := options0085(t)
	base.TokenStore = newMemoryTokenStore()
	prompts := []struct {
		name string
		d    llmprovider.Descriptor
		sel  []int
		wrap string
	}{
		{"typed", together, nil, "enter API key: "},
		{"pasted", grok, []int{pasteIdx0085(t, grok, base)}, "enter credential: "},
	}
	cases := []struct {
		name      string
		replies   []answer0085
		asks      int
		notices   int
		wantError bool
	}{
		{"first ask", []answer0085{{"", errKey0085}}, 1, 0, true},
		{"after a blank", []answer0085{{"  ", nil}, {"", errKey0085}}, 2, 1, true},
		{"value with error", []answer0085{{"made-up-key-1234", errKey0085}}, 1, 0, true},
	}
	for _, pr := range prompts {
		for _, c := range cases {
			t.Run(pr.name+"/"+c.name, func(t *testing.T) {
				p := &answering0085{fakePrompter: newFake(t, fakePrompter{selects: pr.sel}), replies: c.replies}
				res, _, err := ResolveCredential(context.Background(), p, base, pr.d.ID)
				if !errors.Is(err, errKey0085) || err.Error() != pr.wrap+errKey0085.Error() || !zero0085(res) ||
					p.asks != c.asks || len(p.seenNotify) != c.notices {
					t.Errorf("err %v, result %+v, asks %d, notices %q; want the prompter's error wrapped %q, no result, %d ask(s), %d notice(s)",
						err, res, p.asks, p.seenNotify, pr.wrap, c.asks, c.notices)
				}
			})
		}
	}

	t.Run("raw-mode Ctrl-C", func(t *testing.T) {
		rawSeam0059(t, nil, nil)
		p, _ := pipe0059(t, "ab\x03")
		res, _, err := ResolveCredential(context.Background(), p, options0085(t), together.ID)
		if !errors.Is(err, context.Canceled) || !strings.HasPrefix(err.Error(), "enter API key: ") || !zero0085(res) {
			t.Errorf("err %v, result %+v; want the cancel, wrapped \"enter API key: \", no result", err, res)
		}
	})
}

// cancelling0085 is a Prompter whose first Secret ends the run's context, by
// cancel or by waiting for its deadline, and returns a blank.
type cancelling0085 struct {
	*fakePrompter
	ctx    context.Context
	cancel context.CancelFunc
	asks   int
}

func (p *cancelling0085) Secret(string) (string, error) {
	p.asks++
	if p.asks > 50 {
		return "", errors.New("stopped by the test after 50 asks")
	}
	if p.cancel != nil {
		p.cancel()
	} else {
		<-p.ctx.Done()
	}
	return "  ", nil
}

// TestCancelBeforeRepeatAsk_0085 (ST53, MADR D10): when the run's context ends
// while a blank answer is being given, no second Secret call is made and the
// error wraps the context's, at the typed and the pasted prompt, by cancel and
// by a deadline that passes.
func TestCancelBeforeRepeatAsk_0085(t *testing.T) {
	together, _ := providers.Default().Descriptor(llmprovider.ProviderTogether)
	grok, _ := providers.Default().Descriptor(llmprovider.ProviderGrok)
	base := options0085(t)
	base.TokenStore = newMemoryTokenStore()
	for _, pr := range []struct {
		name string
		d    llmprovider.Descriptor
		sel  []int
		wrap string
	}{
		{"typed", together, nil, "enter API key: "},
		{"pasted", grok, []int{pasteIdx0085(t, grok, base)}, "enter credential: "},
	} {
		for _, kind := range []string{"cancel", "deadline"} {
			t.Run(pr.name+"/"+kind, func(t *testing.T) {
				var ctx context.Context
				var cancel context.CancelFunc
				p := &cancelling0085{}
				want := context.Canceled
				if kind == "cancel" {
					ctx, cancel = context.WithCancel(context.Background())
					p.cancel = cancel
				} else {
					ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
					want = context.DeadlineExceeded
				}
				defer cancel()
				p.ctx = ctx
				p.fakePrompter = newFake(t, fakePrompter{selects: pr.sel})
				res, _, err := ResolveCredential(ctx, p, base, pr.d.ID)
				if !errors.Is(err, want) || err.Error() != pr.wrap+want.Error() || !zero0085(res) || p.asks != 1 {
					t.Errorf("err %v, result %+v, asks %d; want an error wrapping %v, prefixed %q, no result, one ask",
						err, res, p.asks, want, pr.wrap)
				}
			})
		}
	}
}
