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

// Sources are what a proposal is made from: the models.dev document, and each
// provider's listed usable ids, for the providers whose listing was read.
type Sources struct {
	Metadata catalog.Metadata
	Listed   map[llmprovider.ProviderID][]string
}

// Proposal is one change to one provider's static list.
type Proposal struct {
	Provider llmprovider.ProviderID
	ID       string
	// Add is true for an id to add, false for one to remove.
	Add bool
	// Reason says why: for a removal, the retirement or the missing listing;
	// for an addition, the model's status, release date and input price.
	Reason string
}

// String is the proposal as one tab-separated line.
func (p Proposal) String() string {
	action := "remove"
	if p.Add {
		action = "add"
	}
	return strings.Join([]string{string(p.Provider), action, p.ID, p.Reason}, "\t")
}

// Propose compares each static list with the sources, and returns the ids to
// remove and to add, sorted by provider, then by id.
//
//   - Removed: an id the deprecations pages retired (retired.IDs), one
//     models.dev marks deprecated or retired, or, where the listing was read,
//     one it does not list.
//   - Added: a model of the listing, where it was read, or else of models.dev,
//     that the list lacks; that no source retires; that is not known to lack
//     tools; and that is released no earlier than the list's newest model.
func Propose(static map[llmprovider.ProviderID][]string, src Sources) []Proposal {
	var out []Proposal
	for provider, ids := range static {
		out = append(out, removals(provider, ids, src)...)
		out = append(out, additions(provider, ids, src)...)
	}
	slices.SortFunc(out, func(a, b Proposal) int {
		return strings.Compare(string(a.Provider)+"\x00"+a.ID, string(b.Provider)+"\x00"+b.ID)
	})
	return out
}

// removals are the static ids to remove, each with its reason.
func removals(provider llmprovider.ProviderID, ids []string, src Sources) []Proposal {
	listed, read := src.Listed[provider]
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
		}
		if reason != "" {
			out = append(out, Proposal{Provider: provider, ID: id, Reason: reason})
		}
	}
	return out
}

// additions are the models to add, each with its status, release and price.
func additions(provider llmprovider.ProviderID, ids []string, src Sources) []Proposal {
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
		out = append(out, Proposal{Provider: provider, ID: id, Add: true, Reason: describe(info)})
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
