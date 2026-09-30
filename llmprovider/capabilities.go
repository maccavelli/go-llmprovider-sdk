package llmprovider

// Support is how far a provider honours one capability (0015-MADR D4).
type Support int

const (
	// Unsupported means a request needing the capability fails before any
	// network call, with an error matching ErrUnsupported.
	Unsupported Support = iota
	// BestEffort means the request is sent, and the provider degrades as its
	// package doc lists; for example, a tool is offered but not forced.
	BestEffort
	// Supported means the capability is honoured as requested.
	Supported
)

// String returns the constant's name.
func (s Support) String() string {
	switch s {
	case Unsupported:
		return "Unsupported"
	case BestEffort:
		return "BestEffort"
	case Supported:
		return "Supported"
	default:
		return "Support(invalid)"
	}
}

// Capabilities declares what a provider supports (0015-MADR D4). The zero
// value supports nothing.
type Capabilities struct {
	// Tools is calling functions the request offers.
	Tools Support
	// ForcedToolChoice is ToolChoiceRequired and ForceTool.
	ForcedToolChoice Support
	// Reasoning is Request.Reasoning.
	Reasoning Support
	// Continuation is Request.PreviousResponseID.
	Continuation Support
	// NativeStreaming is a Streamer implementation. Without one, Stream
	// emits Generate's result as events.
	NativeStreaming Support
}
