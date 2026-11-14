package llmprovider

import "context"

// backgroundKey is the context key of the background mark, a type of its own
// so that no other package's key can collide with it.
type backgroundKey struct{}

// WithBackground returns ctx marked as carrying a request the application
// makes on its own, not for the user's direct activity, such as a session
// title or an automatic compaction. A provider that has a rule for such
// requests applies it to a marked request: the openai provider refuses one on
// a ChatGPT plan session whose user has not granted the background-use
// consent, before any request is sent (0075-MADR D5). An unmarked request is
// never affected: the application decides what is background. A marked
// context keeps its mark in every context derived from it.
func WithBackground(ctx context.Context) context.Context {
	return context.WithValue(ctx, backgroundKey{}, true)
}

// IsBackground reports whether ctx, or a context it derives from, was marked
// by WithBackground.
func IsBackground(ctx context.Context) bool {
	marked, ok := ctx.Value(backgroundKey{}).(bool)
	return ok && marked
}
