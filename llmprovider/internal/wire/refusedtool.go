package wire

import (
	"errors"
	"iter"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/toolschema"
)

// Held is what a request's body held of the caller's tools, as the provider
// encoded it after deferral (0084-MADR D4). It is the only place a refusal's
// name or index is resolved, so a refusal never names a tool the body did not
// hold.
type Held struct {
	// Listed names the entries of the body's tools list, in order, as sent: a
	// service that counts by position counts these. An entry that is not a
	// caller's tool, such as the SDK's own use_tool wrapper, is "". A Search
	// tool sent as a native tool_search entry, which carries no name, is
	// listed under its own.
	Listed []string
	// Inside names the caller's tools the body held outside that list: the
	// definitions in each tool_search_output item of a Responses input.
	Inside []string
}

// has reports whether name is a caller's tool the body held.
func (h Held) has(name string) bool {
	return name != "" && (slices.Contains(h.Listed, name) || slices.Contains(h.Inside, name))
}

// NameRefusedTool sets Tool on the *APIError of kind ErrToolSchemaRefused that
// err is or wraps, to the one tool the service's refusal names among those
// held, and returns err, the error it was given. It leaves Tool empty when the
// refusal names none, names several, names a tool the body did not hold, or
// is in no form toolschema reads; and it leaves any other error, of any other
// kind, a cancelled context included, as it is (0084-MADR D2, D4).
//
// The *APIError is found by errors.As, so a refusal that Reauth or a caller
// wrapped is still named. Its Message is the service's text, redacted and
// bounded to 512 bytes, so a name the redaction removed or the bound cut is
// not read: a name is only ever taken from the list the body held, never from
// the text.
func NameRefusedTool(err error, held Held) error {
	if err == nil {
		return nil
	}
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok || !errors.Is(apiErr.Kind, llmprovider.ErrToolSchemaRefused) {
		return err
	}
	apiErr.Tool = refusedTool(apiErr, held)
	return err
}

// refusedTool is the name of the one tool held that the refusal in apiErr
// names, by name or by position, or "".
func refusedTool(apiErr *llmprovider.APIError, held Held) string {
	form := toolschema.Read([]string{apiErr.Code}, apiErr.Message)
	refs := toolschema.References(form, apiErr.Message)
	found := ""
	resolve := func(name string) bool {
		if !held.has(name) || (found != "" && found != name) {
			return false
		}
		found = name
		return true
	}
	for _, name := range refs.Names {
		if !resolve(name) {
			return ""
		}
	}
	for _, i := range refs.Indexes {
		if i < 0 || i >= len(held.Listed) || !resolve(held.Listed[i]) {
			return ""
		}
	}
	return found
}

// NameRefusedStream is seq with NameRefusedTool applied to each error it
// yields: a stream's refusal arrives before any event, from the send that
// opened it, and it is named where the provider returns it.
func NameRefusedStream(seq iter.Seq2[llmprovider.Event, error], held Held) iter.Seq2[llmprovider.Event, error] {
	return func(yield func(llmprovider.Event, error) bool) {
		for ev, err := range seq {
			if err != nil {
				err = NameRefusedTool(err, held)
			}
			if !yield(ev, err) {
				return
			}
		}
	}
}
