package authcontract

import (
	"encoding/json"
	"time"
)

// StepKind is what a step shows or asks. Paste, confirm and form steps take
// an Answer; every other kind is shown, never answered. It is an open enum: a
// reader shows a step of a kind it does not know by its Text and offers
// cancel. It is a value, safe for concurrent use while no goroutine changes
// it.
type StepKind string

const (
	// StepOpenURL asks the person to open URL. With Accepts, a paste may
	// complete it, and a paste step follows.
	StepOpenURL StepKind = "open_url"
	// StepShowCode shows Code, to be entered at URL on another device.
	StepShowCode StepKind = "show_code"
	// StepPaste asks for a paste, of the form Accepts says.
	StepPaste StepKind = "paste"
	// StepConfirm asks a yes or no, the consent ConfirmID names.
	StepConfirm StepKind = "confirm"
	// StepForm asks for the named inputs in Fields.
	StepForm StepKind = "form"
	// StepWaiting says the sign-in is waiting for something outside it,
	// such as a code entered on another device.
	StepWaiting StepKind = "waiting"
	// StepDone ends a sign-in that succeeded; Yields says what it left.
	StepDone StepKind = "done"
	// StepFailed ends a sign-in that failed; Error says why.
	StepFailed StepKind = "failed"
)

// Step is one thing a sign-in shows or asks. Seq rises by one per step; the
// last is done or failed. It is a value, safe for concurrent use while no
// goroutine changes it.
type Step struct {
	SchemaVersion int                        `json:"schema_version"`
	Seq           int                        `json:"seq"`
	Kind          StepKind                   `json:"kind"`
	URL           string                     `json:"url,omitempty"`
	Code          string                     `json:"code,omitempty"`
	ExpiresAt     time.Time                  `json:"expires_at,omitzero"`
	Accepts       Paste                      `json:"accepts,omitempty"`
	Secret        bool                       `json:"secret,omitempty"` // the answer this step asks for is a secret
	ConfirmID     string                     `json:"confirm_id,omitempty"`
	Text          string                     `json:"text,omitempty"`
	Default       *bool                      `json:"default,omitempty"`
	Links         []Link                     `json:"links,omitempty"`
	Fields        []Field                    `json:"fields,omitempty"`
	Yields        Yields                     `json:"yields,omitempty"`
	Error         *Failure                   `json:"error,omitempty"`
	Meta          map[string]json.RawMessage `json:"_meta,omitempty"`
}
