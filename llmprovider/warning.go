package llmprovider

// WarningKind names what a Warning reports (0034-MADR D7). It is a string,
// safe for concurrent use.
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
	// WarningDeferralEmulated reports that the request's deferred tools were
	// not sent in the service's own form: they were appended to the tools as
	// they were loaded, or dispatched through a wrapper, as the provider's
	// package doc lists. Its Detail names which. An appended load changes the
	// tools, so the prompt cache breaks there once (0040-MADR D6, D7).
	WarningDeferralEmulated WarningKind = "deferral_emulated"
	// WarningEffortDegraded reports that a request at EffortMax was sent at a
	// lower effort: the model's facts list efforts without max, or the
	// service has none. Its Detail names the effort sent (0061-MADR D3).
	WarningEffortDegraded WarningKind = "effort_degraded"
)

// Warning is a non-fatal note on a Response: the request was answered, but
// not exactly as asked. It is advisory; an error says when a request failed
// (0034-MADR D7). It is safe for concurrent use only while no goroutine
// changes it.
type Warning struct {
	// Kind says what happened.
	Kind WarningKind
	// Model is the model the warning is about.
	Model string
	// Detail is a sentence for a person, not for parsing.
	Detail string
}
