package wire

import (
	"fmt"
	"math"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// MaxStreamIndex bounds an index a stream sends for a tool call, a content
// block, a step or a reasoning entry: each decoder grows a slice up to it, so
// an unchecked one could panic the caller, negative, or allocate a slot for
// every index below it, large (0053-MADR D2, Q2).
const MaxStreamIndex = 4096

// StreamIndex checks an index field of provider's stream: nil for 0 ≤ i <
// MaxStreamIndex, and otherwise an error matching llmprovider.ErrIncomplete.
// The reply was sent whole, and asking again would buy the same, as an event
// over EventLimit is.
func StreamIndex(provider, field string, i int) error {
	if i < 0 || i >= MaxStreamIndex {
		return fmt.Errorf("%w: %s: stream %s index %d out of range (0 to %d)", llmprovider.ErrIncomplete, provider, field, i,
			MaxStreamIndex-1)
	}
	return nil
}

// StreamIndexNumber is StreamIndex for an index decoded as a JSON number: one
// that is not finite, not a whole number, or out of range is refused before
// it is converted.
func StreamIndexNumber(provider, field string, f float64) (int, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) || f != math.Trunc(f) || f < 0 || f >= MaxStreamIndex {
		return 0, fmt.Errorf("%w: %s: stream %s index %v out of range (0 to %d)", llmprovider.ErrIncomplete, provider, field, f,
			MaxStreamIndex-1)
	}
	return int(f), nil
}
