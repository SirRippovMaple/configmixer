package cmd

import (
	"fmt"

	"configmixer/internal/mixin"

	"github.com/spf13/cobra"
)

// categoryFlagNames returns the derived flag names for a category, used both
// to register flags and to check for name collisions up front.
func categoryFlagNames(category string) [3]string {
	return [3]string{"select-" + category, "no-auto-" + category, "no-default-" + category}
}

// checkFlagNameCollisions verifies that every per-category flag name
// (--select-<category>, --no-auto-<category>, --no-default-<category>) is
// unique and doesn't collide with any of the fixed flags already registered
// on c, returning a clear error instead of letting pflag panic on a
// duplicate flag registration.
func checkFlagNameCollisions(catalog *mixin.Catalog, fixedNames []string) error {
	seen := map[string]string{}
	for _, name := range fixedNames {
		seen[name] = "a built-in flag"
	}
	for _, category := range catalog.CategoryNames() {
		for _, name := range categoryFlagNames(category) {
			if owner, ok := seen[name]; ok {
				return fmt.Errorf("category %q produces flag --%s, which collides with %s; rename the category directory", category, name, owner)
			}
			seen[name] = fmt.Sprintf("category %q", category)
		}
	}
	return nil
}

// registerCategoryIDFlags registers one string flag per category (a mixin id
// override for that category) and returns the map of flag value pointers.
func registerCategoryIDFlags(c *cobra.Command, catalog *mixin.Catalog) map[string]*string {
	categoryFlags := map[string]*string{}
	for _, category := range catalog.CategoryNames() {
		v := c.Flags().String("select-"+category, "", fmt.Sprintf("mixin id to select for category %q", category))
		categoryFlags[category] = v
	}
	return categoryFlags
}

// registerNoDefaultFlags registers one bool flag per category to opt out of
// a configured default mixin for that category.
func registerNoDefaultFlags(c *cobra.Command, catalog *mixin.Catalog) map[string]*bool {
	noDefaultFlags := map[string]*bool{}
	for _, category := range catalog.CategoryNames() {
		v := c.Flags().Bool("no-default-"+category, false, fmt.Sprintf("disable the configured default mixin for category %q", category))
		noDefaultFlags[category] = v
	}
	return noDefaultFlags
}

// checkCategoryFlagExclusivity errors if more than one of
// --select-<category>, --no-auto-<category>, and --no-default-<category> is
// set for any category.
func checkCategoryFlagExclusivity(catalog *mixin.Catalog, categoryFlags map[string]*string, noAutoFlags, noDefaultFlags map[string]*bool) error {
	for _, category := range catalog.CategoryNames() {
		set := 0
		if *categoryFlags[category] != "" {
			set++
		}
		if *noAutoFlags[category] {
			set++
		}
		if *noDefaultFlags[category] {
			set++
		}
		if set > 1 {
			return fmt.Errorf("--select-%s, --no-auto-%s, and --no-default-%s are mutually exclusive", category, category, category)
		}
	}
	return nil
}

// selectionFromFlags builds a partial Selection from whichever category
// flags were populated, erroring if a flag names a mixin id that doesn't
// exist in its category.
func selectionFromFlags(catalog *mixin.Catalog, categoryFlags map[string]*string) (mixin.Selection, error) {
	sel := mixin.Selection{}
	for _, category := range catalog.CategoryNames() {
		id := *categoryFlags[category]
		if id == "" {
			continue
		}
		m, ok := catalog.Find(category, id)
		if !ok {
			return nil, fmt.Errorf("unknown mixin %q for category %q", id, category)
		}
		sel[category] = m
	}
	return sel, nil
}
