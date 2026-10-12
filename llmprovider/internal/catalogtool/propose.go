// Package catalogtool proposes the next curated model lists, for a person to
// review (0033-MADR D4). It changes nothing: internal/cmd/catalogs prints its
// proposals.
package catalogtool

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/retired"
)

// Sources are what a proposal is made from: the models.dev document, each
// provider's listed usable ids, for the providers whose listing was read, and
// each ranked list's utility six, where it was read.
type Sources struct {
	Metadata catalog.Metadata
	Listed   map[llmprovider.ProviderID][]string
	Ranked   map[llmprovider.ProviderID][]string
}

// Kind is what a proposal does to a list.
type Kind string

// The kinds of proposal. A review line is for a person to check against the
// provider's official page; it is never applied as it stands.
const (
	KindAdd    Kind = "add"
	KindRemove Kind = "remove"
	KindReview Kind = "review"
)

// Proposal is one change to one provider's static list.
type Proposal struct {
	Provider llmprovider.ProviderID
	ID       string
	Kind     Kind
	// Reason says why: for a removal, the retirement, the missing listing or
	// the missing rank; for an addition or a review, the model's rank where
	// it has one, then its status, release date and input price.
	Reason string
}

// String is the proposal as one tab-separated line.
func (p Proposal) String() string {
	return strings.Join([]string{string(p.Provider), string(p.Kind), p.ID, p.Reason}, "\t")
}

// rankedLists are the providers whose static list is the ranker's utility
// six (0009-MADR §3, §4, §7): the five gateways (0033-MADR's amendment "the
// second refresh, and what the command reads", D4b as refined; Q17). The
// vendor lists are rebuilt from the official pages (D4).
var rankedLists = map[llmprovider.ProviderID]bool{
	llmprovider.ProviderOpencodeZen: true, llmprovider.ProviderOpencodeGo: true, llmprovider.ProviderKilo: true,
	llmprovider.ProviderHuggingFace: true, llmprovider.ProviderTogether: true,
}

// Ranked reports whether provider's static list is the ranker's utility six.
func Ranked(provider llmprovider.ProviderID) bool { return rankedLists[provider] }

// Propose compares each static list with the sources, and returns the ids to
// remove, to add and to review, sorted by provider, then by id. Each list is
// proposed by its own rule (0033-MADR's amendment "the second refresh, and
// what the command reads").
//
//   - Removed, from any list: an id the deprecations pages retired
//     (retired.IDs), one models.dev marks deprecated or retired, or, where the
//     listing was read, one it does not list. From a ranked list, where its
//     six was read: one the six leaves out.
//   - Added, to a ranked list: each id of its six that the list lacks, with
//     its rank.
//   - Reviewed, for a vendor list: a model of the listing, where it was read,
//     or else of models.dev, that the list lacks; that no source retires;
//     that is not known to lack tools; and that is released no earlier than
//     the list's newest model. A person checks it against the official page.
func Propose(static map[llmprovider.ProviderID][]string, src Sources) []Proposal {
	var out []Proposal
	for provider, ids := range static {
		out = append(out, removals(provider, ids, src)...)
		if Ranked(provider) {
			out = append(out, rankedAdditions(provider, ids, src)...)
		} else {
			out = append(out, reviews(provider, ids, src)...)
		}
	}
	slices.SortFunc(out, func(a, b Proposal) int {
		return strings.Compare(string(a.Provider)+"\x00"+a.ID, string(b.Provider)+"\x00"+b.ID)
	})
	return out
}

// removals are the static ids to remove, each with its reason.
func removals(provider llmprovider.ProviderID, ids []string, src Sources) []Proposal {
	listed, read := src.Listed[provider]
	six, ranked := src.Ranked[provider]
	ranked = ranked && Ranked(provider)
	var out []Proposal
	for _, id := range ids {
		reason := ""
		info, known := src.Metadata.Model(provider, id)
		switch {
		case retired.IDs[provider][id] != "":
			reason = "retired: " + retired.IDs[provider][id]
		case known && stale(info):
			reason = "models.dev: " + string(info.Status)
		case read && !slices.Contains(listed, id):
			reason = "not in the listing"
		case ranked && !slices.Contains(six, id):
			reason = "not in today's utility six"
		}
		if reason != "" {
			out = append(out, Proposal{Provider: provider, ID: id, Kind: KindRemove, Reason: reason})
		}
	}
	return out
}

// rankedAdditions are the ids of a ranked list's six that the list lacks,
// each with its rank, status, release and price. A list whose six was not
// read gets none.
func rankedAdditions(provider llmprovider.ProviderID, ids []string, src Sources) []Proposal {
	six := src.Ranked[provider]
	var out []Proposal
	for i, id := range six {
		if slices.Contains(ids, id) {
			continue
		}
		info, _ := src.Metadata.Model(provider, id)
		reason := fmt.Sprintf("ranked %d of %d in today's utility six; %s", i+1, len(six), describe(info))
		out = append(out, Proposal{Provider: provider, ID: id, Kind: KindAdd, Reason: reason})
	}
	return out
}

// reviews are a vendor list's newer models, each with its status, release
// and price, for a person to check against the official page.
func reviews(provider llmprovider.ProviderID, ids []string, src Sources) []Proposal {
	newest, dated := newestRelease(provider, ids, src.Metadata)
	candidates, read := src.Listed[provider]
	if !read {
		for _, info := range src.Metadata.Models(provider) {
			candidates = append(candidates, info.ID)
		}
	}
	var out []Proposal
	for _, id := range candidates {
		info, known := src.Metadata.Model(provider, id)
		switch {
		case slices.Contains(ids, id), !known, retired.IDs[provider][id] != "", stale(info),
			info.Has(catalog.FieldTools) && !info.Tools:
			continue
		case dated && (!info.Has(catalog.FieldReleased) || info.Released.Before(newest)):
			continue
		}
		out = append(out, Proposal{Provider: provider, ID: id, Kind: KindReview,
			Reason: "review against the official page; " + describe(info)})
	}
	return out
}

// stale reports whether models.dev marks the model deprecated or retired.
func stale(info catalog.ModelInfo) bool {
	return info.Status == catalog.StatusDeprecated || info.Status == catalog.StatusRetired
}

// newestRelease is the latest release date among the list's models, and
// whether any is known.
func newestRelease(provider llmprovider.ProviderID, ids []string, m catalog.Metadata) (time.Time, bool) {
	var newest time.Time
	dated := false
	for _, id := range ids {
		if info, ok := m.Model(provider, id); ok && info.Has(catalog.FieldReleased) {
			if !dated || info.Released.After(newest) {
				newest, dated = info.Released, true
			}
		}
	}
	return newest, dated
}

// describe is an addition's reason: its status, release date and input price.
func describe(info catalog.ModelInfo) string {
	status := "status unknown"
	if info.Has(catalog.FieldStatus) {
		status = "status " + string(info.Status)
	}
	released := "release date unknown"
	if info.Has(catalog.FieldReleased) {
		released = "released " + info.Released.Format(time.DateOnly)
	}
	price := "input price unknown"
	if info.Has(catalog.FieldCostInput) {
		price = fmt.Sprintf("input $%.2f per million tokens", info.Cost.Input)
	}
	return status + ", " + released + ", " + price
}
