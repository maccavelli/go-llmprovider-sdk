// Package signinflow is the step engine under llmprovider/x/signin and the
// wizard (0074-MADR D3, D4, D6). It runs one sign-in, taking the login
// functions of llmprovider/auth as values: it hands the person's side of the
// sign-in out as numbered contract steps, takes the answers to the steps that
// ask, checks a pasted callback before it reaches auth, and returns the
// result. It saves nothing and reads no environment variable.
//
// A Run is built by Start and ended by its own goroutine: the last step is
// done or failed, after which the Steps channel is closed.
package signinflow

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// stepBuffer holds every step a sign-in can emit (at most an open_url, three
// pastes, and an end), so that the engine never blocks on a slow reader.
const stepBuffer = 16

// maxPastes is the pastes one sign-in takes: the first two bad ones are asked
// again, and a third ends the sign-in (0074-MADR D4).
const maxPastes = 3

// Mode is which login a Spec runs.
type Mode int

// The modes. Each names the auth function that does the work.
const (
	// ModeDevice runs a device-code sign-in: show_code, waiting, then the end.
	ModeDevice Mode = iota + 1
	// ModeBrowser runs LoginBrowserOAuth: open_url, a paste step when the
	// method takes a paste, then the end.
	ModeBrowser
	// ModePlan runs LoginChatGPTPlan, as ModeBrowser does.
	ModePlan
	// ModeImport runs Spec.Import and is done at once.
	ModeImport
)

// DeviceFlow is a device sign-in that has its code: what to show, and how to
// wait for the approval.
type DeviceFlow struct {
	// URL is where the person enters Code.
	URL string
	// Code is the code the person enters.
	Code string
	// Expiry is when Code stops being accepted.
	Expiry time.Time
	// Wait polls until the approval, the expiry or ctx ending.
	Wait func(ctx context.Context) (*auth.OAuthSession, error)
	// Cancel stops Wait; nil is no-op.
	Cancel func()
}

// Logins are the login functions a Run calls. A nil field takes auth's own,
// so a test supplies a fake for the one it exercises. A Logins is safe for
// concurrent use while no goroutine changes it.
type Logins struct {
	// Browser is auth.LoginBrowserOAuth.
	Browser func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (*auth.OAuthSession, error)
	// Plan is auth.LoginChatGPTPlan.
	Plan func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error)
	// Device is auth.StartDeviceOAuth, as a DeviceFlow.
	Device func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (DeviceFlow, error)
}

// withDefaults returns l with auth's functions in the nil fields.
func (l Logins) withDefaults() Logins {
	if l.Browser == nil {
		l.Browser = auth.LoginBrowserOAuth
	}
	if l.Plan == nil {
		l.Plan = auth.LoginChatGPTPlan
	}
	if l.Device == nil {
		l.Device = startDevice
	}
	return l
}

// startDevice is auth.StartDeviceOAuth as a DeviceFlow.
func startDevice(ctx context.Context, p llmprovider.ProviderID, opts auth.OAuthFlowOptions) (DeviceFlow, error) {
	login, err := auth.StartDeviceOAuth(ctx, p, opts)
	if err != nil {
		return DeviceFlow{}, err
	}
	return DeviceFlow{URL: login.VerificationURI, Code: login.UserCode, Expiry: login.Expiry,
		Wait: login.Wait, Cancel: login.Cancel}, nil
}

// Spec describes one sign-in. It is safe for concurrent use while no
// goroutine changes it; Start reads it once.
type Spec struct {
	// Provider is the provider the login signs in to.
	Provider llmprovider.ProviderID
	// Method is the method's facts. Its Paste and FirstPaste decide the
	// browser steps; its Yields is what the done step carries.
	Method authcontract.Method
	// Mode is the login to run.
	Mode Mode
	// Flow is the options of a ModeBrowser or ModeDevice login. The engine
	// sets OpenURL, InputCode and NoLoopback itself.
	Flow auth.OAuthFlowOptions
	// Plan is the options of a ModePlan login, with the same exceptions.
	Plan auth.ChatGPTPlanOptions
	// First reports a first sign-in, for which the method's FirstPaste, when
	// set, says what a paste may be.
	First bool
	// ConsentText maps a consent id to the app's own wording, which the
	// confirm step shows in place of the method's text (0075-MADR D4).
	ConsentText map[string]string
	// ClientStore, when set, keeps a plan sign-in's issued client and host id
	// apart from the session (0075-MADR D7). Before a plan sign-in with no
	// Plan.Previous, the engine loads the record and signs in to its client
	// with its host id, unless Plan.HostID names another; once the login
	// returns a session, it saves the client and host id the session holds, at
	// once, since the registration happened at OpenAI whatever the caller then
	// does with the session.
	ClientStore auth.ClientStore
	// NoPaste leaves out the paste step: the loopback alone completes a
	// browser sign-in.
	NoPaste bool
	// NoLoopback binds no listener: a paste alone completes a browser
	// sign-in. A device sign-in ignores it.
	NoLoopback bool
	// Import is a ModeImport sign-in's work: it checks the vendor CLI's file
	// and returns the session that reads it.
	Import func(context.Context) (*auth.VendorCLISession, error)
	// Logins is the login functions; nil fields take auth's.
	Logins Logins
	// Logger, when set, receives each accepted answer at debug level, with
	// the answer's secrets masked by its own LogValue. Nil logs nothing.
	Logger *slog.Logger
}

// Result is what a finished sign-in produced: Session for a browser, plan
// or device sign-in, Import for a vendor import.
type Result struct {
	Session *auth.OAuthSession
	Import  *auth.VendorCLISession
	// ClientSaveErr is the error of saving the plan's issued client, which
	// does not fail the sign-in: the next one registers another client.
	ClientSaveErr error
}

// ask is a step that asks, and the answer's way back to the login.
type ask struct {
	answers  chan authcontract.Answer
	secret   bool
	answered bool
}

// Run is a sign-in in progress. It is safe for concurrent use: Answer,
// Cancel and Wait may run at once.
type Run struct {
	method  authcontract.Method
	steps   chan authcontract.Step
	cancel  context.CancelFunc
	done    chan struct{}
	logger  *slog.Logger
	logCtx  context.Context
	mu      sync.Mutex
	seq     int
	asks    map[int]*ask
	ended   bool
	cancels bool
	stop    func()
	result  Result
	err     error
}

// Start checks spec and starts the sign-in on its own goroutine. An option
// that nothing could complete is refused before any request: a browser
// method whose Paste is empty with NoLoopback gives an error matching
// llmprovider.ErrUnsupported, and NoLoopback with NoPaste one matching
// llmprovider.ErrInvalidRequest, as does an unknown Mode or a ModeImport
// without Import. ctx bounds the sign-in; Cancel ends it sooner.
func Start(ctx context.Context, spec Spec) (*Run, error) {
	if err := check(spec); err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	r := &Run{
		method: spec.Method,
		steps:  make(chan authcontract.Step, stepBuffer),
		cancel: cancel,
		done:   make(chan struct{}),
		logger: spec.Logger,
		logCtx: context.WithoutCancel(ctx),
		asks:   map[int]*ask{},
	}
	spec.Logins = spec.Logins.withDefaults()
	go func() {
		defer cancel()
		result, err := r.run(runCtx, spec)
		r.finish(result, err)
	}()
	return r, nil
}

// check refuses what no login could complete or run.
func check(spec Spec) error {
	switch spec.Mode {
	case ModeDevice:
		return nil
	case ModeImport:
		if spec.Import == nil {
			return fmt.Errorf("%w: signinflow: an import sign-in needs Import", llmprovider.ErrInvalidRequest)
		}
		return nil
	case ModeBrowser, ModePlan:
		if !spec.NoLoopback {
			return nil
		}
		if spec.Method.Paste == "" {
			return fmt.Errorf("%w: signinflow: method %q takes no paste, so NoLoopback leaves nothing to complete it",
				llmprovider.ErrUnsupported, spec.Method.ID)
		}
		if spec.NoPaste {
			return fmt.Errorf("%w: signinflow: NoLoopback with NoPaste leaves nothing to complete the sign-in",
				llmprovider.ErrInvalidRequest)
		}
		return nil
	default:
		return fmt.Errorf("%w: signinflow: unknown mode %d", llmprovider.ErrInvalidRequest, spec.Mode)
	}
}

// Method returns the method's facts the Run was started with.
func (r *Run) Method() authcontract.Method { return r.method }

// Steps returns the channel of steps, in order, closed after the done or
// failed step. It is buffered for the whole sign-in.
func (r *Run) Steps() <-chan authcontract.Step { return r.steps }

// Wait returns the result, or the sign-in's error. It returns ctx's error
// when ctx ends first. It is safe to call more than once.
func (r *Run) Wait(ctx context.Context) (Result, error) {
	select {
	case <-r.done:
		return r.result, r.err
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

// Cancel ends the sign-in: its last step is failed with kind canceled, and
// Wait's error matches context.Canceled. It does nothing after the end.
func (r *Run) Cancel() {
	r.mu.Lock()
	if !r.ended {
		r.cancels = true
	}
	stop := r.stop
	r.mu.Unlock()
	r.cancel()
	if stop != nil {
		stop()
	}
}

// Answer gives the answer to the asking step whose Seq it names. A step
// that does not ask, one already answered, any other Seq, and an answer
// after the end each give an error matching llmprovider.ErrInvalidRequest.
// The engine sets the answer's Secret from its own step, whatever the
// caller sent (0073-MADR D6 rule 5). It returns ctx's error when ctx has
// ended.
func (r *Run) Answer(ctx context.Context, a authcontract.Answer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ended {
		return fmt.Errorf("%w: signinflow: the sign-in has ended", llmprovider.ErrInvalidRequest)
	}
	step, ok := r.asks[a.Seq]
	if !ok {
		return fmt.Errorf("%w: signinflow: step %d asks nothing, or does not exist", llmprovider.ErrInvalidRequest, a.Seq)
	}
	if step.answered {
		return fmt.Errorf("%w: signinflow: step %d is already answered", llmprovider.ErrInvalidRequest, a.Seq)
	}
	step.answered = true
	secret := step.secret
	a.Secret = &secret
	if r.logger != nil {
		r.logger.LogAttrs(r.logCtx, slog.LevelDebug, "signinflow: answer", slog.Any("answer", a))
	}
	step.answers <- a
	return nil
}

// emit numbers step, sends it, and, for an asking step, registers its
// answer. It returns the Seq. A step after the end is dropped, which a
// login that outlives its Run can cause only by a bug.
func (r *Run) emit(step authcontract.Step, asking bool) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ended {
		return 0
	}
	r.seq++
	step.SchemaVersion = authcontract.SchemaVersion
	step.Seq = r.seq
	if asking {
		r.asks[step.Seq] = &ask{answers: make(chan authcontract.Answer, 1), secret: step.Secret}
	}
	r.steps <- step
	return step.Seq
}

// answers returns the channel the answer to the step seq arrives on.
func (r *Run) answers(seq int) <-chan authcontract.Answer {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.asks[seq]; ok {
		return a.answers
	}
	return nil
}

// finish emits the last step, closes the channel and releases Wait.
func (r *Run) finish(result Result, err error) {
	r.mu.Lock()
	if err != nil && r.cancels {
		err = fmt.Errorf("signinflow: sign-in canceled: %w", context.Canceled)
	}
	r.seq++
	last := authcontract.Step{SchemaVersion: authcontract.SchemaVersion, Seq: r.seq}
	if err != nil {
		last.Kind = authcontract.StepFailed
		failure := Failure(err)
		last.Error = &failure
	} else {
		last.Kind = authcontract.StepDone
		last.Yields = r.method.Yields
	}
	r.steps <- last
	r.ended = true
	r.result, r.err = result, err
	close(r.steps)
	r.mu.Unlock()
	close(r.done)
}

// run does the login for spec and returns its result.
func (r *Run) run(ctx context.Context, spec Spec) (Result, error) {
	switch spec.Mode {
	case ModeDevice:
		return r.runDevice(ctx, spec)
	case ModeImport:
		session, err := spec.Import(ctx)
		return Result{Import: session}, err
	default:
		return r.runBrowser(ctx, spec)
	}
}

// runDevice shows the code, then waits for the approval.
func (r *Run) runDevice(ctx context.Context, spec Spec) (Result, error) {
	flow, err := spec.Logins.Device(ctx, spec.Provider, spec.Flow)
	if err != nil {
		return Result{}, err
	}
	r.mu.Lock()
	r.stop = flow.Cancel
	canceled := r.cancels
	r.mu.Unlock()
	if canceled && flow.Cancel != nil {
		flow.Cancel()
	}
	r.emit(authcontract.Step{Kind: authcontract.StepShowCode, URL: flow.URL, Code: flow.Code, ExpiresAt: flow.Expiry}, false)
	r.emit(authcontract.Step{Kind: authcontract.StepWaiting}, false)
	session, err := flow.Wait(ctx)
	return Result{Session: session}, err
}

// runBrowser opens the authorize URL as a step and, when the method takes a
// paste, asks for one while the login listens on the loopback.
func (r *Run) runBrowser(ctx context.Context, spec Spec) (Result, error) {
	accepts := authcontract.Paste("")
	if !spec.NoPaste {
		accepts = spec.Method.Paste
		if spec.First && spec.Method.FirstPaste != "" {
			accepts = spec.Method.FirstPaste
		}
	}
	var answered []consentAnswer
	if spec.Mode == ModePlan {
		var err error
		if answered, err = r.askConsents(ctx, spec); err != nil {
			return Result{}, err
		}
	}
	opened := make(chan string, 1)
	openURL := func(authorizeURL string) error {
		r.emit(authcontract.Step{Kind: authcontract.StepOpenURL, URL: authorizeURL, Accepts: accepts}, false)
		select {
		case opened <- authorizeURL:
		default:
		}
		return nil
	}
	var inputCode func(context.Context) (string, error)
	if accepts != "" {
		inputCode = func(ctx context.Context) (string, error) {
			select {
			case authorizeURL := <-opened:
				return r.askPaste(ctx, authorizeURL, accepts)
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}
	var session *auth.OAuthSession
	var err error
	var clientSaveErr error
	if spec.Mode == ModePlan {
		opts := spec.Plan
		opts.OpenURL, opts.InputCode, opts.NoLoopback = openURL, inputCode, spec.NoLoopback
		if spec.ClientStore != nil && opts.Previous == nil {
			record, loadErr := spec.ClientStore.LoadClient(ctx, spec.Provider)
			if loadErr != nil {
				return Result{}, fmt.Errorf("signinflow: load the issued client: %w", loadErr)
			}
			opts.ClientID = cmp.Or(opts.ClientID, record.ClientID)
			opts.HostID = cmp.Or(opts.HostID, record.HostID)
		}
		session, err = spec.Logins.Plan(ctx, opts)
		if err == nil && session != nil && spec.ClientStore != nil {
			clientSaveErr = spec.ClientStore.SaveClient(ctx, spec.Provider,
				auth.ClientRecord{ClientID: session.ClientID, HostID: session.HostID})
		}
	} else {
		opts := spec.Flow
		opts.OpenURL, opts.InputCode, opts.NoLoopback = openURL, inputCode, spec.NoLoopback
		session, err = spec.Logins.Browser(ctx, spec.Provider, opts)
	}
	if err == nil && session != nil {
		err = putConsents(ctx, spec.Provider, session, answered)
	}
	return Result{Session: session, ClientSaveErr: clientSaveErr}, err
}

// consentAnswer is the answer to one consent: asked, or carried from the
// plan session the sign-in is to.
type consentAnswer struct {
	id      string
	granted bool
}

// askConsents asks each consent of the method, as a confirm step, before the
// login starts, unless the session being signed in to again already holds an
// answer to it, which is carried instead. The answers go on the session the
// login returns, and nowhere before: a sign-in that fails records none
// (0075-MADR D4). A consent defaults to no, and an answer that gives none is
// a no.
func (r *Run) askConsents(ctx context.Context, spec Spec) ([]consentAnswer, error) {
	var answers []consentAnswer
	for _, consent := range spec.Method.Consents {
		if previous := spec.Plan.Previous; previous != nil {
			if granted, answered := previous.Consent(consent.ID); answered {
				answers = append(answers, consentAnswer{consent.ID, granted})
				continue
			}
		}
		no := false
		seq := r.emit(authcontract.Step{Kind: authcontract.StepConfirm, ConfirmID: consent.ID,
			Text: cmp.Or(spec.ConsentText[consent.ID], consent.Text), Default: &no, Links: consent.Links}, true)
		replies := r.answers(seq)
		if replies == nil {
			return nil, fmt.Errorf("%w: signinflow: the sign-in has ended", llmprovider.ErrInvalidRequest)
		}
		select {
		case reply := <-replies:
			answers = append(answers, consentAnswer{consent.ID, reply.Confirm != nil && *reply.Confirm})
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return answers, nil
}

// answerStore is the one session auth.SetConsent writes the answers into, so
// that the entries' format is auth's alone.
type answerStore struct{ consents string }

func (s *answerStore) Load(context.Context, llmprovider.ProviderID) (*auth.OAuthSession, error) {
	return &auth.OAuthSession{Consents: s.consents}, nil
}

func (s *answerStore) Save(_ context.Context, _ llmprovider.ProviderID, session *auth.OAuthSession) error {
	s.consents = session.Consents
	return nil
}

func (s *answerStore) Delete(context.Context, llmprovider.ProviderID) error { return nil }

// putConsents records answers on the session a sign-in returned.
func putConsents(ctx context.Context, provider llmprovider.ProviderID, session *auth.OAuthSession, answers []consentAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	store := &answerStore{}
	for _, answer := range answers {
		if err := auth.SetConsent(ctx, store, provider, answer.id, answer.granted); err != nil {
			return err
		}
	}
	session.Consents = store.consents
	return nil
}

// askPaste asks for the callback until one passes the checks of D4 or the
// third bad one ends the sign-in. It returns the pasted text, for auth's own
// parsing, and an error that matches llmprovider.ErrInvalidRequest after the
// third bad paste. A pasted denial passes: auth ends the sign-in with it.
func (r *Run) askPaste(ctx context.Context, authorizeURL string, accepts authcontract.Paste) (string, error) {
	want := newPasteCheck(authorizeURL, accepts)
	text := promptFor(accepts)
	for tries := 1; ; tries++ {
		seq := r.emit(authcontract.Step{Kind: authcontract.StepPaste, Accepts: accepts, Secret: true, Text: text}, true)
		answers := r.answers(seq)
		if answers == nil {
			return "", fmt.Errorf("%w: signinflow: the sign-in has ended", llmprovider.ErrInvalidRequest)
		}
		var answer authcontract.Answer
		select {
		case answer = <-answers:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		reason := want.check(answer.Paste)
		if reason == "" {
			return answer.Paste, nil
		}
		if tries >= maxPastes {
			return "", fmt.Errorf("%w: signinflow: no usable callback after %d pastes (%s)",
				llmprovider.ErrInvalidRequest, maxPastes, reason)
		}
		text = reason
	}
}

// Failure is the contract's description of err: its kind, its text without
// the characters a terminal acts on, and whether the same sign-in can
// succeed later.
func Failure(err error) authcontract.Failure {
	kind := ErrorKindOf(err)
	retryable := false
	var apiErr *llmprovider.APIError
	if errors.As(err, &apiErr) {
		retryable = apiErr.Retryable()
	} else {
		switch kind {
		case authcontract.FailureRateLimited, authcontract.FailureOffline, authcontract.FailureUnavailable:
			retryable = true
		}
	}
	return authcontract.Failure{Kind: kind, Message: redact.StripControl(redact.String(err.Error())), Retryable: retryable}
}
