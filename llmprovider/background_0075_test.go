package llmprovider

import (
	"context"
	"testing"
)

// unrelatedKey is another package's context key.
type unrelatedKey struct{}

// plainKey is a key of the built-in string type, which only a collision with
// another package's would make the mark.
func plainKey() any { return "background" }

// TestWithBackground_0075 (0075-MADR D5): the mark is read back; an unmarked
// context reads false; a child context keeps the mark; a value under the
// string "background" does not mark.
func TestWithBackground_0075(t *testing.T) {
	ctx := context.Background()
	if IsBackground(ctx) {
		t.Error("an unmarked context reads as background")
	}
	marked := WithBackground(ctx)
	if !IsBackground(marked) {
		t.Error("a marked context does not read as background")
	}
	if IsBackground(ctx) {
		t.Error("marking a context marked its parent")
	}
	child, cancel := context.WithCancel(marked)
	defer cancel()
	if !IsBackground(context.WithValue(child, unrelatedKey{}, 1)) {
		t.Error("a context derived from a marked one lost the mark")
	}
	if IsBackground(context.WithValue(ctx, plainKey(), true)) {
		t.Error(`a value under the string "background" marks a context`)
	}
}
