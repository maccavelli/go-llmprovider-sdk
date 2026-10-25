// Package compatid is a chatcompat provider's id as an option's value: what
// chatcompat.WithID sets and catalog reads, so that the catalog lists a
// chatcompat provider's server and neither package imports the other
// (0061-MADR, amendment "chatcompat, provider-agnostic", E1).
package compatid

import "strings"

// Value is chatcompat.WithID's value: the provider's id.
type Value string

// Of is the id of the last Value among values, and whether there is one.
func Of(values []any) (string, bool) {
	id, found := "", false
	for _, v := range values {
		if got, ok := v.(Value); ok {
			id, found = string(got), true
		}
	}
	return id, found
}

// For reports whether values carry a Value naming id, compared without case,
// as provider ids are.
func For(values []any, id string) bool {
	got, ok := Of(values)
	return ok && strings.EqualFold(got, id)
}
