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
	problems := violations(sel, func(Expr, map[string]string) bool { return true })
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("incompatible selection:\n  - %s", strings.Join(problems, "\n  - "))
}

// IntroducedConflicts reports the requires/excludes violations that adding m
// for category to partial definitely introduces. Since partial may still be
// missing categories, a clause is only judged when every tag key it
// references is present among the other selected mixins' tags; a clause that
// depends on a still-open category can't be decided yet and is skipped
// (Validate catches it once the selection is complete). Violations already
// present in partial without m are not attributed to m.
func IntroducedConflicts(partial Selection, category string, m Mixin) []string {
	judgeable := func(e Expr, tags map[string]string) bool {
		for _, k := range e.Keys() {
			if _, ok := tags[k]; !ok {
				return false
			}
		}
		return true
	}

	existing := map[string]bool{}
	for _, p := range violations(partial, judgeable) {
		existing[p] = true
	}

	trial := Selection{}
	maps.Copy(trial, partial)
	trial[category] = m

	var out []string
	for _, p := range violations(trial, judgeable) {
		if !existing[p] {
			out = append(out, p)
		}
	}
	return out
}

// violations checks every selected mixin's requires/excludes clauses against
// the tags contributed by the OTHER mixins in the selection, skipping clauses
// for which judgeable returns false. It returns a sorted description of each
// violation found.
func violations(sel Selection, judgeable func(Expr, map[string]string) bool) []string {
	var problems []string
	for category, m := range sel {
		others := Selection{}
		maps.Copy(others, sel)
		delete(others, category)
		tags := AggregateTags(others)

		for _, req := range m.Requires {
			if judgeable(req, tags) && !req.Eval(tags) {
				problems = append(problems, fmt.Sprintf("%s/%s requires %s", category, m.ID, req))
			}
		}
		for _, exc := range m.Excludes {
			if judgeable(exc, tags) && exc.Eval(tags) {
				problems = append(problems, fmt.Sprintf("%s/%s excludes %s", category, m.ID, exc))
			}
		}
	}
	sort.Strings(problems)
	return problems
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
