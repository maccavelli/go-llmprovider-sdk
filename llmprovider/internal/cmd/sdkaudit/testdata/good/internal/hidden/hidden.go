package hidden

// An internal package is not public: none of R50 to R52 applies to it.

type Undocumented struct{}

func Quiet() error { return nil }
