// Package pub is a planted package with one of each defect sdkaudit counts.
package pub

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// Doer is an exported interface whose doc says nothing of concurrency.
type Doer interface{ Do() }

// ErrX is a sentinel.
var ErrX = errors.New("x")

// Shared is exported mutable state.
var Shared = map[string]int{}

var now = time.Now

var pattern = regexp.MustCompile("x")

var mu sync.Mutex

var names = []string{"a"}

var other = 1

type plantError struct{}

func (*plantError) Error() string { return "planted" }

var errPlanted error = &plantError{}

// Holder keeps a context in a struct. It is safe for concurrent use.
type Holder struct{ ctx context.Context }

func Undocumented() {}

// Wrong starts with another word.
func Misnamed() {}

// Late takes its context second.
func Late(n int, ctx context.Context) {}

// Make returns an interface.
func Make() Doer { return nil }

// Old is retired. Deprecated: use Make.
func Old() {}

// Quiet returns something.
func Quiet() error { return nil }

// Fail panics, makes a request with no context, and returns an error that
// drops its cause's wrapping.
func Fail(err error) error {
	_, _ = http.Get("http://example.invalid")
	_ = time.Now()
	if err == nil {
		panic("no")
	}
	return fmt.Errorf("lost: %v", err)
}
