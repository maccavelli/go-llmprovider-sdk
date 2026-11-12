package authcontract

import "encoding/json"

// Feature is an additive capability a method's sign-in uses. A Method lists
// its features, so that a client that cannot render one says so before it
// starts. It is an open enum: a reader keeps a value it does not know. It is
// a value, safe for concurrent use while no goroutine changes it.
type Feature string

const (
	// FeatureConsent means the sign-in asks for a consent's yes or no, in a
	// confirm step.
	FeatureConsent Feature = "consent"
	// FeatureForm means the sign-in asks for named inputs, in a form step.
	FeatureForm Feature = "form"
	// FeaturePaste means the sign-in asks for a paste, in a paste step.
	FeaturePaste Feature = "paste"
)

// Kind is a method's mechanism: what the person does to sign in. It is an
// open enum: a reader lists a method of a kind it does not know and does not
// drive it. It is a value, safe for concurrent use while no goroutine
// changes it.
type Kind string

const (
	// KindAPIKey is a method that takes an API key the person enters.
	KindAPIKey Kind = "api_key"
	// KindTokenPaste is a method that takes a token the person pastes,
	// issued elsewhere.
	KindTokenPaste Kind = "token_paste"
	// KindDeviceCode is a device sign-in: a code is shown, entered on
	// another device, and the sign-in waits for it.
	KindDeviceCode Kind = "device_code"
	// KindBrowser is a browser sign-in: a URL is opened, and the host's
	// loopback listener or a pasted URL or code completes it.
	KindBrowser Kind = "browser"
	// KindOAuth is an OAuth sign-in whose mechanism is learned at start. It
	// goes with CompletionAtStart, and its first steps say whether it is a
	// device or a browser sign-in.
	KindOAuth Kind = "oauth"
	// KindVendorImport is a method that imports a credential another
	// program has already stored.
	KindVendorImport Kind = "vendor_import"
	// KindHostOnly is a method that can be finished only on the host. It
	// has no steps; its Completion says how it finishes.
	KindHostOnly Kind = "host_only"
)

// Completion is how a method's sign-in completes. It is an open enum: a
// reader lists a method whose completion it does not know and does not drive
// it. It is a value, safe for concurrent use while no goroutine changes it.
type Completion string

const (
	// CompletionValue completes on a value the person enters: a key or a
	// token.
	CompletionValue Completion = "value"
	// CompletionDevice completes when the code shown is entered on another
	// device.
	CompletionDevice Completion = "device"
	// CompletionBrowserPaste completes a browser sign-in by a paste, or by
	// the host's loopback listener where it can be reached.
	CompletionBrowserPaste Completion = "browser_paste"
	// CompletionBrowserLoopback completes a browser sign-in by the host's
	// loopback listener only.
	CompletionBrowserLoopback Completion = "browser_loopback"
	// CompletionAtStart is a completion learned when the sign-in starts,
	// from its first steps. It goes with KindOAuth.
	CompletionAtStart Completion = "at_start"
	// CompletionLocalFile completes by reading a credential file another
	// program stored.
	CompletionLocalFile Completion = "local_file"
	// CompletionHostCommand completes by a command a person runs on the
	// host, which Method.Command shows.
	CompletionHostCommand Completion = "host_command"
)

// Paste is what a paste may be. It is an open enum: a reader keeps a value
// it does not know. It is a value, safe for concurrent use while no
// goroutine changes it.
type Paste string

const (
	// PasteNone is no paste.
	PasteNone Paste = ""
	// PasteURLOrCode is the full callback URL or its code alone.
	PasteURLOrCode Paste = "url_or_code"
	// PasteURLOnly is the full callback URL only.
	PasteURLOnly Paste = "url_only"
	// PasteCode is a code only.
	PasteCode Paste = "code"
)

// Yields is what a sign-in leaves behind. It is an open enum: a reader keeps
// a value it does not know. It is a value, safe for concurrent use while no
// goroutine changes it.
type Yields string

const (
	// YieldsAPIKey is an API key.
	YieldsAPIKey Yields = "api_key"
	// YieldsOAuthSession is an OAuth session that refreshes itself.
	YieldsOAuthSession Yields = "oauth_session"
	// YieldsAccessOnly is an access token alone, with no refresh token.
	YieldsAccessOnly Yields = "access_only"
	// YieldsDeviceKey is a key a device sign-in issues.
	YieldsDeviceKey Yields = "device_key"
	// YieldsReadThrough is a credential left in another program's store,
	// read from there when it is needed.
	YieldsReadThrough Yields = "read_through"
	// YieldsUnknown is not known before the sign-in. It is valid only on a
	// method whose completion is CompletionAtStart; what such a sign-in
	// yielded is its done step's Yields.
	YieldsUnknown Yields = "unknown"
)

// ClientIdentity is whose OAuth client a method signs in as. A client shows
// it. It is an open enum: a reader keeps a value it does not know. It is a
// value, safe for concurrent use while no goroutine changes it.
type ClientIdentity string

const (
	// ClientNone is no OAuth client: the method signs in as no client.
	ClientNone ClientIdentity = "none"
	// ClientVendorBorrowed is the vendor's own client, such as its CLI's,
	// borrowed by this sign-in.
	ClientVendorBorrowed ClientIdentity = "vendor_borrowed"
	// ClientAppRegistered is a client registered for the application that
	// signs in.
	ClientAppRegistered ClientIdentity = "app_registered"
	// ClientUnknown is not known before the sign-in. It is valid only on a
	// method whose completion is CompletionAtStart.
	ClientUnknown ClientIdentity = "unknown"
)

// LinkRel is what a Link points to. It is an open enum: a reader keeps a
// value it does not know. It is a value, safe for concurrent use while no
// goroutine changes it.
type LinkRel string

const (
	// LinkTerms points to terms the sign-in is under.
	LinkTerms LinkRel = "terms"
	// LinkUsage points to where the account's usage is shown.
	LinkUsage LinkRel = "usage"
	// LinkNotice points to a notice the person should read.
	LinkNotice LinkRel = "notice"
)

// FieldType is the type of a form step's input. It is an open enum: a
// reader keeps a value it does not know. It is a value, safe for
// concurrent use while no goroutine changes it.
type FieldType string

const (
	// FieldText is a line of text.
	FieldText FieldType = "text"
	// FieldSecret is a line of text that is a secret: masked as it is
	// typed, and never logged.
	FieldSecret FieldType = "secret"
	// FieldSelect is one of the field's Options.
	FieldSelect FieldType = "select"
)

// Method is one way to sign in, as facts a client reads before it starts. It
// is a value, safe for concurrent use while no goroutine changes it.
type Method struct {
	SchemaVersion  int                        `json:"schema_version"`
	ID             string                     `json:"id"` // the provider's method id, e.g. "chatgpt_plan"
	Kind           Kind                       `json:"kind"`
	Label          string                     `json:"label"`
	Detail         string                     `json:"detail,omitempty"`
	Completion     Completion                 `json:"completion"`
	Paste          Paste                      `json:"paste,omitempty"`       // what a paste may be
	FirstPaste     Paste                      `json:"first_paste,omitempty"` // a first sign-in's, when it differs
	Yields         Yields                     `json:"yields"`
	ClientIdentity ClientIdentity             `json:"client_identity"`
	Refreshable    bool                       `json:"refreshable"`
	Revocable      bool                       `json:"revocable"`
	Slot           string                     `json:"slot,omitempty"`    // methods that share one stored credential
	Command        string                     `json:"command,omitempty"` // host_command: what to run on the host; shown, never run
	Features       []Feature                  `json:"features,omitempty"`
	Consents       []Consent                  `json:"consents,omitempty"`
	Links          []Link                     `json:"links,omitempty"`
	Meta           map[string]json.RawMessage `json:"_meta,omitempty"`
}

// Consent is a yes or no a method's terms require, under a stable id such as
// "chatgpt_plan.background_use". Text is the default wording; an app may
// show its own. It is a value, safe for concurrent use while no goroutine
// changes it.
type Consent struct {
	ID       string `json:"id"`
	Required bool   `json:"required"`
	Text     string `json:"text,omitempty"`
	Links    []Link `json:"links,omitempty"`
}

// Link is a link a method, a consent or a step shows. It is a value, safe
// for concurrent use while no goroutine changes it.
type Link struct {
	Rel   LinkRel `json:"rel"`
	URL   string  `json:"url"`
	Title string  `json:"title,omitempty"`
}

// Field is one named input of a form step. It is a value, safe for
// concurrent use while no goroutine changes it.
type Field struct {
	Key      string    `json:"key"`
	Label    string    `json:"label"`
	Type     FieldType `json:"type"`
	Options  []string  `json:"options,omitempty"`
	Required bool      `json:"required,omitempty"`
	Default  string    `json:"default,omitempty"`
}
