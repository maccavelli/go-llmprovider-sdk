// Package listlimit carries catalog.WithLimit's value between the catalog
// and the built-in providers, which cut their own fallback listings with it
// (0038-MADR D2, A3). Nothing here is exported outside the module.
package listlimit

import "slices"

// Value is catalog.WithLimit's option value: the most models a listing
// recommends. Zero or less is no limit.
type Value int

// Of returns the last Value among values, since options apply in order, or
// zero, no limit, when there is none.
func Of(values []any) int {
	n := 0
	for _, v := range values {
		if l, ok := v.(Value); ok {
			n = int(l)
		}
	}
	return n
}

// Cut returns a copy of ids, cut to its first n when 0 < n < len(ids). A nil
// ids stays nil.
func Cut(ids []string, n int) []string {
	if n > 0 && n < len(ids) {
		ids = ids[:n]
	}
	return slices.Clone(ids)
}
