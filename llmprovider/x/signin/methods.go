package signin

import (
	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Methods returns d's sign-in methods as the contract describes them, in
// menu order. Each AuthMethod maps field for field, with the contract's
// schema version set, and the paste feature when its Paste fact is set. A
// descriptor that lists no method but requires a key gets one api_key
// method; one that requires no key gets none. Methods never sets a command
// or the contract's oauth and at_start values, which no built-in method has
// (0074-MADR D2). It reads d only and is safe for concurrent use.
func Methods(d llmprovider.Descriptor) []authcontract.Method {
	if len(d.AuthMethods) == 0 {
		if !d.RequiresAPIKey {
			return nil
		}
		return []authcontract.Method{methodOf(implicitKey(d))}
	}
	methods := make([]authcontract.Method, 0, len(d.AuthMethods))
	for _, m := range d.AuthMethods {
		methods = append(methods, methodOf(m))
	}
	return methods
}

// implicitKey is the api_key method a descriptor that requires a key and
// lists no method has by implication, so that no descriptor changes (0074-MADR
// D2).
func implicitKey(d llmprovider.Descriptor) llmprovider.AuthMethod {
	label := "API key"
	if d.Label != "" {
		label = d.Label + " API key"
	}
	return llmprovider.AuthMethod{
		ID:             llmprovider.AuthAPIKey,
		Label:          label,
		Kind:           string(authcontract.KindAPIKey),
		Completion:     string(authcontract.CompletionValue),
		Yields:         string(authcontract.YieldsAPIKey),
		ClientIdentity: string(authcontract.ClientNone),
	}
}

// methodOf is m as the contract's Method.
func methodOf(m llmprovider.AuthMethod) authcontract.Method {
	method := authcontract.Method{
		SchemaVersion:  authcontract.SchemaVersion,
		ID:             string(m.ID),
		Kind:           authcontract.Kind(m.Kind),
		Label:          m.Label,
		Detail:         m.Detail,
		Completion:     authcontract.Completion(m.Completion),
		Paste:          authcontract.Paste(m.Paste),
		FirstPaste:     authcontract.Paste(m.FirstPaste),
		Yields:         authcontract.Yields(m.Yields),
		ClientIdentity: authcontract.ClientIdentity(m.ClientIdentity),
		Refreshable:    m.Refreshable,
		Revocable:      m.Revocable,
		Slot:           m.Slot,
	}
	if m.Paste != "" {
		method.Features = []authcontract.Feature{authcontract.FeaturePaste}
	}
	return method
}
