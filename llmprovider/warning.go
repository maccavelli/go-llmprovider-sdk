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
	// WarningEffortDegraded reports that a request's effort was not sent as
	// asked: the model's facts do not list it, or the model refused it, so
	// another effort, or none, was sent. Its Detail names the effort sent
	// (0061-MADR D3; 0062-MADR D3).
	WarningEffortDegraded WarningKind = "effort_degraded"
	// WarningReasoningLearned reports that the model refused a thinking
	// setting the provider sent, and that the provider learned it for the
	// rest of the process and sent the request again. Its Refused is what a
	// caller can keep, and pass back with WithReasoningRefusals on its next
	// start (0062-MADR D6).
	WarningReasoningLearned WarningKind = "reasoning_learned"
	// WarningImageDropped reports that an image of the request was not sent,
	// because the service cannot take it in the form given, as the provider's
	// package doc lists. Its Detail says which image and why (0061-MADR D5).
	WarningImageDropped WarningKind = "image_dropped"
	// WarningLimitLearned reports that the service refused the output limit
	// the provider sent and stated its cap, validated as 0089-MADR D4 says,
	// and that the provider keeps the cap for the rest of the process, once
	// per model, base URL, scope and stream or plain call. Its Observed is
	// what a caller can keep, and pass back with WithLimitObservations on its
	// next start (0089-MADR D5). A call that fails after such a refusal, a
	// caller's own limit refused among them, has no response to carry it: its
	// *APIError's Observed carries the same value, on every such call
	// (0089-MADR DL78, DL82).
	WarningLimitLearned WarningKind = "limit_learned"
	// WarningToolCallCut reports that the service's output limit cut a tool
	// call before its arguments were whole, so the call was dropped: the
	// response finishes FinishLength with what was usable, its text and its
	// whole calls. Its Detail names the cut call, by its tool's name and its
	// call id where the service gave them (0088-MADR ST77, OQ16).
	WarningToolCallCut WarningKind = "tool_call_cut"
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
	// Refused is the setting a WarningReasoningLearned reports the model
	// refused; zero for any other kind, and then left out of a Warning's JSON,
	// which stays as it was (0062-MADR D6).
	Refused ReasoningRefusal `json:",omitzero"`
	// Observed is the cap a WarningLimitLearned reports the service stated;
	// zero for any other kind, and then left out of a Warning's JSON
	// (0089-MADR D5).
	Observed LimitObservation `json:",omitzero"`
}
