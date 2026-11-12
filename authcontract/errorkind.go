package authcontract

// ErrorKind is a failure's class: the one word every consumer uses for it.
// The constants are listed in the order a consumer that maps an error tests
// them, the most specific first. Each is named Failure and its value in
// camel case, after the type that carries the kind. It is an open enum: a
// reader keeps a value it does not know. It is a value, safe for
// concurrent use while no goroutine changes it.
type ErrorKind string

const (
	// FailureCanceled is a sign-in or a call the caller or the user stopped.
	FailureCanceled ErrorKind = "canceled"
	// FailureTimeout is a deadline that passed.
	FailureTimeout ErrorKind = "timeout"
	// FailureDenied is a sign-in the user or the identity provider refused.
	FailureDenied ErrorKind = "denied"
	// FailureExpired is a code or a sign-in window that expired.
	FailureExpired ErrorKind = "expired"
	// FailureAuth is a credential that was refused.
	FailureAuth ErrorKind = "auth"
	// FailureQuota is a quota, credit or balance exhausted.
	FailureQuota ErrorKind = "quota"
	// FailureRateLimited is a call that was throttled.
	FailureRateLimited ErrorKind = "rate_limited"
	// FailureNotPermitted is something the account may not do.
	FailureNotPermitted ErrorKind = "not_permitted"
	// FailureWorkspace is a workspace that is required or was not found.
	FailureWorkspace ErrorKind = "workspace"
	// FailureContextOverflow is input longer than the model's window.
	FailureContextOverflow ErrorKind = "context_overflow"
	// FailureEmptyAnswer is an answer with nothing usable.
	FailureEmptyAnswer ErrorKind = "empty_answer"
	// FailureIncomplete is a response cut short.
	FailureIncomplete ErrorKind = "incomplete"
	// FailureInvalidRequest is a request the service refuses as malformed.
	FailureInvalidRequest ErrorKind = "invalid_request"
	// FailureInvalidProvider is an unknown or unsafe provider id.
	FailureInvalidProvider ErrorKind = "invalid_provider"
	// FailureOffline is no network.
	FailureOffline ErrorKind = "offline"
	// FailureUnavailable is a service that failed.
	FailureUnavailable ErrorKind = "unavailable"
	// FailureUnsupported is a capability the provider lacks.
	FailureUnsupported ErrorKind = "unsupported"
	// FailureOther is none of the above.
	FailureOther ErrorKind = "other"
)

// Failure is why a sign-in failed: a failed step's Error. It is a value, safe
// for concurrent use while no goroutine changes it.
type Failure struct {
	Kind      ErrorKind `json:"kind"`
	Message   string    `json:"message"`
	Retryable bool      `json:"retryable"`
}
