// Package vocab loads and queries the controlled keyword vocabulary — the
// SKOS-shaped concept scheme stored as taxonomy.yaml at the data-repo root.
//
// It is the single source of truth for keyword categorization. bibdb's fix
// rules, `bibdb vocab` command, and markdown export all read it here instead
// of hardcoding category lists; skyebot and the skye skills query it through
// `bibdb vocab`.
//
// The package is a leaf: it imports only the YAML library and the standard
// library, so any other bibdb package may depend on it without cycles.
package vocab

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileName is the taxonomy file, resolved relative to the data-repo root.
const FileName = "taxonomy.yaml"

// Concept is one term in the vocabulary (a skos:Concept).
type Concept struct {
	ID        string   `yaml:"id"`        // kebab slug, matches the Obsidian tag slug
	PrefLabel string   `yaml:"prefLabel"` // the one canonical spelling
	Facet     string   `yaml:"facet"`     // top|topic|place|person|org|genre|time
	Broader   string   `yaml:"broader"`   // parent concept id (optional)
	AltLabel  []string `yaml:"altLabel"`  // variants that normalize to PrefLabel
	Dewey     string   `yaml:"dewey"`     // Dewey mapping (top-level spine)
	ACM       string   `yaml:"acm"`       // ACM CCS path (CS/AI subtree)
}

// Scheme is the concept-scheme metadata block.
type Scheme struct {
	ID              string   `yaml:"id"`
	Title           string   `yaml:"title"`
	Version         string   `yaml:"version"`
	Facets          []string `yaml:"facets"`
	DefaultFacet    string   `yaml:"defaultFacet"`
	RequireTopLevel bool     `yaml:"requireTopLevel"`
}

type file struct {
	Scheme   Scheme    `yaml:"scheme"`
	Concepts []Concept `yaml:"concepts"`
}

// Vocabulary is a parsed, indexed taxonomy.
type Vocabulary struct {
	Scheme   Scheme
	concepts []Concept
	byID     map[string]*Concept
	byLabel  map[string]*Concept // lowercased prefLabel + altLabels → concept
}

// Load reads taxonomy.yaml from root. A missing file is not an error: it
// returns (nil, nil) so vocab-aware features degrade gracefully to their
// pre-taxonomy behavior.
func Load(root string) (*Vocabulary, error) {
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return Parse(data)
}

// Parse builds a Vocabulary from taxonomy.yaml bytes.
func Parse(data []byte) (*Vocabulary, error) {
	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	v := &Vocabulary{
		Scheme:   f.Scheme,
		concepts: f.Concepts,
		byID:     make(map[string]*Concept, len(f.Concepts)),
		byLabel:  make(map[string]*Concept, len(f.Concepts)*2),
	}
	if v.Scheme.DefaultFacet == "" {
		v.Scheme.DefaultFacet = "topic"
	}
	for i := range v.concepts {
		c := &v.concepts[i]
		v.byID[c.ID] = c
		v.byLabel[strings.ToLower(c.PrefLabel)] = c
		for _, a := range c.AltLabel {
			key := strings.ToLower(strings.TrimSpace(a))
			if key == "" {
				continue
			}
			// prefLabels win over altLabels on collision; first altLabel wins.
			if _, exists := v.byLabel[key]; !exists {
				v.byLabel[key] = c
			}
		}
	}
	return v, nil
}

// Concepts returns a copy of all concepts (safe to sort/mutate by the caller
// without disturbing the internal id/label indexes).
func (v *Vocabulary) Concepts() []Concept {
	out := make([]Concept, len(v.concepts))
	copy(out, v.concepts)
	return out
}

// Lookup resolves a raw term (case-insensitive, trimmed) to its concept.
func (v *Vocabulary) Lookup(term string) (*Concept, bool) {
	c, ok := v.byLabel[strings.ToLower(strings.TrimSpace(term))]
	return c, ok
}

// ByID returns the concept with the given id.
func (v *Vocabulary) ByID(id string) (*Concept, bool) {
	c, ok := v.byID[id]
	return c, ok
}

// Canonical returns the canonical prefLabel for term and whether it was known.
// Unknown terms are returned trimmed and unchanged.
func (v *Vocabulary) Canonical(term string) (string, bool) {
	if c, ok := v.Lookup(term); ok {
		return c.PrefLabel, true
	}
	return strings.TrimSpace(term), false
}

// FacetOf returns the concept's facet, defaulting to the scheme default.
func (v *Vocabulary) FacetOf(c *Concept) string {
	if c == nil || c.Facet == "" {
		return v.Scheme.DefaultFacet
	}
	return c.Facet
}

// TopAncestor walks the broader chain from c to the first top-level concept
// (facet "top"). It returns that concept, or false if none is reachable.
func (v *Vocabulary) TopAncestor(c *Concept) (*Concept, bool) {
	seen := make(map[string]bool)
	for c != nil && !seen[c.ID] {
		if c.Facet == "top" {
			return c, true
		}
		seen[c.ID] = true
		if c.Broader == "" {
			break
		}
		c = v.byID[c.Broader]
	}
	return nil, false
}

// HasTopLevel reports whether any keyword resolves to a concept whose broader
// chain reaches a top-level category (or is itself top-level).
func (v *Vocabulary) HasTopLevel(keywords []string) bool {
	for _, k := range keywords {
		if c, ok := v.Lookup(k); ok {
			if _, ok := v.TopAncestor(c); ok {
				return true
			}
		}
	}
	return false
}

// TopLevels returns the prefLabels of every top-level (facet "top") concept.
func (v *Vocabulary) TopLevels() []string {
	var out []string
	for i := range v.concepts {
		if v.concepts[i].Facet == "top" {
			out = append(out, v.concepts[i].PrefLabel)
		}
	}
	return out
}

// TagPath returns the facet-prefixed path used to build an Obsidian tag from a
// keyword: entity facets get a "<facet>/" prefix (place/Iran, person/Trump,
// org/OpenAI, genre/poetry, time/Cold War); top and topic terms and unknown
// terms are returned bare. The caller slugifies the result into a #tag.
func (v *Vocabulary) TagPath(term string) string {
	c, ok := v.Lookup(term)
	if !ok {
		return strings.TrimSpace(term)
	}
	switch v.FacetOf(c) {
	case "top", "topic":
		return c.PrefLabel
	default:
		return v.FacetOf(c) + "/" + c.PrefLabel
	}
}

// NormalizeKeywords canonicalizes a comma-separated keywords string: each term
// is trimmed, mapped to its canonical prefLabel (altLabel → prefLabel), and
// duplicates are removed (order preserved). It returns the normalized string
// (comma-space joined), whether the canonical token sequence differs from the
// input token sequence, and human-readable change messages. Pure spacing
// differences do not count as a change. This is the shared implementation
// behind both the keywords-vocab fix rule and `bibdb vocab normalize`.
func (v *Vocabulary) NormalizeKeywords(raw string) (result string, changed bool, messages []string) {
	var trimmed, canon []string
	seen := make(map[string]bool)
	for _, p := range strings.Split(raw, ",") {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		trimmed = append(trimmed, t)
		c, known := v.Canonical(t)
		if known && c != t {
			messages = append(messages, fmt.Sprintf("keyword %q → %q", t, c))
		}
		key := strings.ToLower(c)
		if seen[key] {
			messages = append(messages, fmt.Sprintf("keyword %q deduplicated", c))
			continue
		}
		seen[key] = true
		canon = append(canon, c)
	}
	result = strings.Join(canon, ", ")
	changed = result != strings.Join(trimmed, ", ")
	return result, changed, messages
}

// --- active singleton -------------------------------------------------------
//
// Fix rules (whose Apply signature takes only the entry) and markdown export
// read the process-wide active vocabulary. A command sets it once, after
// resolving the backend, before running rules or exporting. When it is nil,
// vocab-aware features fall back to their pre-taxonomy behavior.

var active *Vocabulary

// SetActive installs v as the process-wide active vocabulary (may be nil).
func SetActive(v *Vocabulary) { active = v }

// Active returns the process-wide active vocabulary, or nil if none is loaded.
func Active() *Vocabulary { return active }

// LoadActive loads taxonomy.yaml from root and installs it as active. A
// missing file is not an error — active is set to nil and vocab-aware
// features no-op.
func LoadActive(root string) error {
	v, err := Load(root)
	if err != nil {
		return err
	}
	active = v
	return nil
}
