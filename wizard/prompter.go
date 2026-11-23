// Package wizard provides the canonical LLM provider configuration flow for
// applications that have a setup UI.
//
// It owns the DATA and the FLOW; consumers own the RENDERING. That split is
// deliberate: consumers disagree about UI toolkits, and a library that imported
// one would impose an interactive-terminal dependency on every importer,
// including headless ones that never draw a menu.
//
// Consumers implement Prompter over whatever they already use. TextPrompter is
// provided for those with no toolkit at all.
//
// With Options.Discover the flow checks a credential by listing the provider's
// models with it, and a credential the provider refuses is asked for again or
// ends the run. Kilo's public listing answers every credential, so for a Kilo
// credential used with no organization the check is a read of Kilo's profile
// first, whose 401 is that refusal and whose other failures are a warning.
//
// A terminal program with no toolkit runs the flow over its standard input and
// output, and keeps the Result in its own configuration:
//
//	res, err := wizard.ConfigureLLM(ctx, &wizard.TextPrompter{In: os.Stdin, Out: os.Stdout},
//		wizard.Options{AllowEnv: true, LookupEnv: os.Getenv, Discover: true})
//	if err != nil {
//		return err
//	}
//	save(res.Provider, res.Model, res.APIKey)
package wizard

import (
	"context"
	"fmt"
)

// Level classifies a Notify message. It is an integer, safe for
// concurrent use.
type Level int

// Notification levels.
const (
	LevelInfo Level = iota
	LevelWarn
	LevelError
)

// String implements fmt.Stringer.
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return fmt.Sprintf("level(%d)", int(l))
	}
}

// Choice is one selectable option. Detail is an optional qualifier shown
// alongside the label, e.g. a provider's pricing model. It is safe for
// concurrent use only while no goroutine changes it.
type Choice struct {
	Label  string
	Detail string
}

// Prompter is the rendering seam. A consumer implements it over its own UI
// toolkit; this package never imports one.
//
// The interface is deliberately small and stable: every method mirrors
// something all three existing wizards already do. It is public API in a
// library twelve repositories consume, so changing it later is a breaking
// change for all of them.
//
// An implementation must allow one concurrent use: during a browser sign-in,
// ConfigureLLM may call Notify while an Input still waits for the pasted code.
type Prompter interface {
	// Select asks the user to choose one option, returning its index.
	// defaultIdx is chosen when the user accepts the default. Besides its
	// menus, the wizard asks with Select, before a browser sign-in, where the
	// browser is: "Where is the browser you will sign in with?", with two
	// choices, "On this machine", at index 0 and the default, and "On another
	// device: I will paste the address back". The default is the answer to give
	// when nobody can choose: a Prompter that cannot ask returns it. The
	// question is not asked under Options.NoLoopback, NoBrowserQuestion,
	// NoPastePrompt or Driver (0083-MADR D3). The interface itself is unchanged.
	Select(title string, choices []Choice, defaultIdx int) (int, error)

	// MultiSelect asks the user to choose zero or more options, returning
	// their indices. preselected may be nil.
	MultiSelect(title string, choices []Choice, preselected []int) ([]int, error)

	// Confirm asks a yes/no question, returning def when the user accepts
	// the default.
	Confirm(question string, def bool) (bool, error)

	// Input reads a line of visible text, returning def when the user
	// accepts the default.
	Input(prompt, def string) (string, error)

	// Secret reads a credential without echoing it in full. Implementations
	// must never write the value to a log. The wizard trims the value, and
	// does not take one that is empty once trimmed as a key: it gives a
	// notice and asks again, up to three asks in all for each prompt, and the
	// third blank ends the run with an error. An implementation signals a
	// cancel with an error, not with an empty value, which is asked for again
	// (0085-MADR D3). An implementation that reads the entry in raw mode,
	// where Enter can arrive as CR then LF, must not leave that LF for the
	// next prompt (0087-MADR A6).
	Secret(prompt string) (string, error)

	// Notify reports progress or a problem. It must not be used to display a
	// credential; callers mask first.
	Notify(level Level, format string, args ...any)
}

// InputerContext is implemented by a Prompter that can give up a read. The
// wizard calls InputContext in place of Input for a browser sign-in's paste
// prompt, the one read the loopback can make moot, and ends ctx once the
// sign-in has finished, so that the sign-in returns without the Enter it
// otherwise asks for (0057-MADR D3). Every other prompt still calls Input.
//
// InputContext is Input that gives up when ctx ends: it returns soon after,
// with an error, and leaves no read running that could take the user's next
// answer, so that a line typed later goes to the next prompt. A Prompter that
// cannot promise this must not implement it: TextPrompter does not, since a
// read of stdin cannot be given up. A Prompter that wraps another must
// forward it, or the wizard asks for the Enter.
//
// The wizard calls it on a goroutine of its own, while another goroutine may
// call Notify, as it calls Input; an implementation must allow that one
// concurrent use. If it has not returned shortly after ctx ends, the wizard
// asks for the Enter and waits for it.
type InputerContext interface {
	InputContext(ctx context.Context, prompt, def string) (string, error)
}
