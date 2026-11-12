package authcontract

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
)

// redacted stands in for a secret in a formatted Answer.
const redacted = "[redacted]"

// Answer returns what a paste, confirm or form step asked for. Secret is the
// answered step's secret, set by the reader that holds that step; nil means
// not known. String, GoString and LogValue mask Paste and every Fields value
// unless Secret is a non-nil false; MarshalJSON does not mask, since the
// answer must reach the sign-in. It is a value, safe for concurrent use while
// no goroutine changes it.
type Answer struct {
	SchemaVersion int                        `json:"schema_version"`
	Seq           int                        `json:"seq"`
	Secret        *bool                      `json:"secret,omitempty"`
	Paste         string                     `json:"paste,omitempty"`
	Confirm       *bool                      `json:"confirm,omitempty"`
	Fields        map[string]string          `json:"fields,omitempty"`
	Meta          map[string]json.RawMessage `json:"_meta,omitempty"`
}

// masked reports whether a formatted answer hides Paste and the Fields
// values: always, unless Secret is a non-nil false.
func (a Answer) masked() bool { return a.Secret == nil || *a.Secret }

// shown is v as a formatted answer prints it: quoted, or redacted.
func (a Answer) shown(v string) string {
	if a.masked() {
		return redacted
	}
	return strconv.Quote(v)
}

// String renders the answer with Paste and the Fields values masked unless
// Secret is a non-nil false. SchemaVersion, Seq, Secret, Confirm, the Fields
// keys and the Meta keys are shown, never Meta's values. Paste, Confirm,
// Fields and Meta are left out when unset.
func (a Answer) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Answer{SchemaVersion:%d Seq:%d Secret:%s", a.SchemaVersion, a.Seq, boolText(a.Secret))
	if a.Paste != "" {
		fmt.Fprintf(&b, " Paste:%s", a.shown(a.Paste))
	}
	if a.Confirm != nil {
		fmt.Fprintf(&b, " Confirm:%t", *a.Confirm)
	}
	if len(a.Fields) > 0 {
		b.WriteString(" Fields:{")
		for i, k := range sortedKeys(a.Fields) {
			if i > 0 {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "%s:%s", k, a.shown(a.Fields[k]))
		}
		b.WriteString("}")
	}
	if len(a.Meta) > 0 {
		fmt.Fprintf(&b, " Meta:%v", sortedKeys(a.Meta))
	}
	b.WriteString("}")
	return b.String()
}

// GoString renders the answer for %#v, masked as String masks it.
func (a Answer) GoString() string { return "authcontract." + a.String() }

// LogValue renders the answer for slog, masked as String masks it, under
// the JSON names.
func (a Answer) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.Int("schema_version", a.SchemaVersion),
		slog.Int("seq", a.Seq),
	}
	if a.Secret != nil {
		attrs = append(attrs, slog.Bool("secret", *a.Secret))
	} else {
		attrs = append(attrs, slog.Any("secret", nil))
	}
	if a.Paste != "" {
		attrs = append(attrs, slog.String("paste", a.logShown(a.Paste)))
	}
	if a.Confirm != nil {
		attrs = append(attrs, slog.Bool("confirm", *a.Confirm))
	}
	if len(a.Fields) > 0 {
		fields := make([]slog.Attr, 0, len(a.Fields))
		for _, k := range sortedKeys(a.Fields) {
			fields = append(fields, slog.String(k, a.logShown(a.Fields[k])))
		}
		attrs = append(attrs, slog.Attr{Key: "fields", Value: slog.GroupValue(fields...)})
	}
	if len(a.Meta) > 0 {
		attrs = append(attrs, slog.Any("_meta", sortedKeys(a.Meta)))
	}
	return slog.GroupValue(attrs...)
}

// logShown is v as LogValue prints it: as it is, or redacted. A handler
// quotes it as it needs.
func (a Answer) logShown(v string) string {
	if a.masked() {
		return redacted
	}
	return v
}

// boolText is a *bool as String prints it.
func boolText(p *bool) string {
	if p == nil {
		return "<nil>"
	}
	return strconv.FormatBool(*p)
}

// sortedKeys returns m's keys in order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
