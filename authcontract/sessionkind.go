package authcontract

// SessionKind is the kind of a stored sign-in session, with the strings of
// the SDK's auth.SessionKind. It is a value, safe for concurrent use while
// no goroutine changes it.
//
// It is an open enum (rule 2): a reader keeps a value it does not know and
// shows it as it came, with no label; a producer emits only the values
// below. Each value's label, the text a client shows for it, is in the JSON
// Schema, as this type's "x-enum-labels", not in Go.
type SessionKind string

const (
	// SessionChatGPT names a session from a ChatGPT sign-in through OpenAI's
	// browser or device sign-in.
	SessionChatGPT SessionKind = "chatgpt"
	// SessionChatGPTPlan names a session from the ChatGPT plan sign-in,
	// under the signing-in application's own client.
	SessionChatGPTPlan SessionKind = "chatgpt_plan"
	// SessionGrok names a session from xAI's Grok browser or device sign-in.
	SessionGrok SessionKind = "grok"
	// SessionKiloDevice names a session from Kilo's device sign-in.
	SessionKiloDevice SessionKind = "kilo_device"
)
