package llmprovider

// WarningKind names what a Warning reports (0034-MADR D7).
type WarningKind string

const (
	// WarningForcedChoiceDegraded reports that a forced tool choice
	// (ForceTool or ToolChoiceRequired) was not sent as asked: the provider
	// sent "auto", as its package doc lists under Degradations. The model may
	// answer without the call.
	WarningForcedChoiceDegraded WarningKind = "forced_tool_choice_degraded"
	// WarningForcedChoiceLearned reports that the model refused a forced tool
	// choice that no table or option had said it would, so the provider
	// learned it for the rest of the process. Its Model is what a caller can
	// keep, and pass back with WithModelForcedChoice on its next start.
	WarningForcedChoiceLearned WarningKind = "forced_tool_choice_learned"
)

// Warning is a non-fatal note on a Response: the request was answered, but
// not exactly as asked. It is advisory; an error says when a request failed
// (0034-MADR D7).
type Warning struct {
	// Kind says what happened.
	Kind WarningKind
	// Model is the model the warning is about.
	Model string
	// Detail is a sentence for a person, not for parsing.
	Detail string
}
