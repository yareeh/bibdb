// Package fixrules holds the registered rules used by `bibdb fix`. Each rule
// inspects a single Entry and either auto-fixes it in place (Severity AutoFix)
// or surfaces a problem that needs an external repair (Severity Report).
//
// Discipline: when bibdb gains a feature or fix that changes how entries
// should look, add a new Rule here with Since=<next release>. Rule IDs are an
// API once shipped — deprecate, never repurpose.
package fixrules

import (
	"sort"

	"github.com/yareeh/bibdb/internal"
)

// Severity describes whether a rule mutates the entry or only reports an
// issue that needs an external (human / LLM) repair.
type Severity int

const (
	// AutoFix rules mechanically rewrite the entry to satisfy the rule.
	AutoFix Severity = iota
	// Report rules describe a problem without mutating; the caller surfaces
	// the message so the user can repair it (e.g. via skyebot's /bib redo).
	Report
)

func (s Severity) String() string {
	switch s {
	case AutoFix:
		return "auto-fix"
	case Report:
		return "report"
	default:
		return "unknown"
	}
}

// Result is what a single rule returns after running against one entry.
type Result struct {
	// Changed is true when an AutoFix rule mutated the entry.
	Changed bool
	// Messages describes what was changed or detected. One per issue.
	Messages []string
	// NeedsExternal lists problems a Report rule surfaced. Empty for AutoFix.
	NeedsExternal []string
}

// Rule is a single check / fix unit.
type Rule struct {
	// ID is a stable slug used by --rule and in Rules.md.
	ID string
	// Since is the bibdb release that introduced this rule. Entries whose
	// bibdbversion is >= Since are skipped by default.
	Since string
	// Severity controls whether the rule mutates the entry or reports only.
	Severity Severity
	// Description is a single sentence used by --list-rules and Rules.md.
	Description string
	// Apply runs the rule against one entry, returning what changed.
	Apply func(e *internal.Entry) Result
}

var registry []Rule

// Register adds r to the registry. Rules call this from init() in their own
// files. Duplicate IDs panic — this is a programmer error.
func Register(r Rule) {
	for _, existing := range registry {
		if existing.ID == r.ID {
			panic("fixrules: duplicate rule ID " + r.ID)
		}
	}
	registry = append(registry, r)
}

// All returns a copy of the registry sorted by ID for stable iteration.
func All() []Rule {
	out := make([]Rule, len(registry))
	copy(out, registry)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ByID returns the rule with the given ID and whether it was found.
func ByID(id string) (Rule, bool) {
	for _, r := range registry {
		if r.ID == id {
			return r, true
		}
	}
	return Rule{}, false
}
