package catalog

import (
	"slices"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// hasControl reports whether s holds a character a terminal acts on: a C0
// control but tab, DEL, or a C1 control (U+0080 to U+009F). It is the rule of
// internal/redact, which catalog may not import (R2). A listed id holding
// one is dropped, so that no server can move the cursor, clear the screen or
// set the window title of a terminal that shows its models (0066-MADR V1,
// Q3 (a)).
func hasControl(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool {
		return r < 0x20 && r != '\t' || r == 0x7f || r >= 0x80 && r <= 0x9f
	})
}

// withoutControl is f less each id holding a control character: from its
// usable ids, Kilo's entries and a chatcompat server's created times. The
// count is logged at Debug, never the ids (0066-MADR V1).
func (f fetchedListing) withoutControl(id llmprovider.ProviderID, cfg config) fetchedListing {
	dropped := map[string]struct{}{}
	drop := func(m string) bool {
		if !hasControl(m) {
			return false
		}
		dropped[m] = struct{}{}
		return true
	}
	f.usable = slices.DeleteFunc(f.usable, drop)
	f.kilo = slices.DeleteFunc(f.kilo, func(e kiloCatalogEntry) bool { return drop(e.ID) })
	for m := range f.created {
		if drop(m) {
			delete(f.created, m)
		}
	}
	logDropped(cfg, id, len(dropped))
	return f
}

// logDropped logs, at Debug, that n listed ids of id held a control
// character and were dropped.
func logDropped(cfg config, id llmprovider.ProviderID, n int) {
	if n > 0 && cfg.logger != nil {
		cfg.logger.Debug("catalog: listed ids holding a control character dropped", "provider", string(id), "count", n)
	}
}
