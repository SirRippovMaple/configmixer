package mixin

import (
	"fmt"
	"maps"
	"sort"
	"strings"
)

// Selection maps a category name to the mixin chosen for it.
type Selection map[string]Mixin

// AggregateTags unions the tags contributed by every mixin in the selection.
// Later categories (in sorted order) win on key collisions.
func AggregateTags(sel Selection) map[string]string {
	cats := make([]string, 0, len(sel))
	for c := range sel {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	tags := map[string]string{}
	for _, c := range cats {
		maps.Copy(tags, sel[c].Tags)
	}
	return tags
}

// Validate checks every selected mixin's requires/excludes clauses against the
// tags contributed by the OTHER mixins in the selection. It returns a single
// error describing every violation found.
func Validate(sel Selection) error {
	var problems []string
	for category, m := range sel {
		others := Selection{}
		maps.Copy(others, sel)
		delete(others, category)
		tags := AggregateTags(others)

		for _, req := range m.Requires {
			if !req.Eval(tags) {
				problems = append(problems, fmt.Sprintf("%s/%s requires %s", category, m.ID, req))
			}
		}
		for _, exc := range m.Excludes {
			if exc.Eval(tags) {
				problems = append(problems, fmt.Sprintf("%s/%s excludes %s", category, m.ID, exc))
			}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("incompatible selection:\n  - %s", strings.Join(problems, "\n  - "))
}

// CompatibleChoices filters a category's mixins down to those that produce a
// valid selection when combined with the given partial selection (which must
// not already contain that category).
func CompatibleChoices(cat *Catalog, category string, partial Selection) []Mixin {
	var out []Mixin
	for _, m := range cat.Categories[category] {
		trial := Selection{}
		maps.Copy(trial, partial)
		trial[category] = m
		if Validate(trial) == nil {
			out = append(out, m)
		}
	}
	return out
}
