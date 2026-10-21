package cmd

import (
	"fmt"
	"strings"

	"configmixer/internal/mixin"

	"github.com/spf13/cobra"
)

// categoryFlagNames returns the derived flag names for a category, used both
// to register flags and to check for name collisions up front.
func categoryFlagNames(category string) [5]string {
	return [5]string{"select-" + category, "no-auto-" + category, "no-default-" + category, "no-script-" + category, "prompt-" + category}
}

// checkFlagNameCollisions verifies that every per-category flag name
// (--select-<category>, --no-auto-<category>, --no-default-<category>,
// --no-script-<category>, --prompt-<category>) is
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

// registerNoScriptFlags registers one bool flag per category to opt out of
// a configured selector script for that category.
func registerNoScriptFlags(c *cobra.Command, catalog *mixin.Catalog) map[string]*bool {
	noScriptFlags := map[string]*bool{}
	for _, category := range catalog.CategoryNames() {
		v := c.Flags().Bool("no-script-"+category, false, fmt.Sprintf("disable the configured selector script for category %q", category))
		noScriptFlags[category] = v
	}
	return noScriptFlags
}

// registerPromptFlags registers one bool flag per category that forces an
// interactive prompt for that category, bypassing any embedded selection or
// configured default.
func registerPromptFlags(c *cobra.Command, catalog *mixin.Catalog) map[string]*bool {
	promptFlags := map[string]*bool{}
	for _, category := range catalog.CategoryNames() {
		v := c.Flags().Bool("prompt-"+category, false, fmt.Sprintf("force an interactive prompt for category %q", category))
		promptFlags[category] = v
	}
	return promptFlags
}

// categoryBoolFlag names a bool-typed per-category flag set for use in
// checkCategoryFlagExclusivity's mutual-exclusivity check.
type categoryBoolFlag struct {
	namePrefix string
	flags      map[string]*bool
}

// checkCategoryFlagExclusivity errors if more than one of --select-<category>
// and the given bool-typed per-category flag sets (e.g. --no-auto-<category>,
// --no-default-<category>, --prompt-<category>) is set for any category.
func checkCategoryFlagExclusivity(catalog *mixin.Catalog, categoryFlags map[string]*string, boolFlagSets ...categoryBoolFlag) error {
	for _, category := range catalog.CategoryNames() {
		var set []string
		if *categoryFlags[category] != "" {
			set = append(set, "--select-"+category)
		}
		for _, bf := range boolFlagSets {
			if *bf.flags[category] {
				set = append(set, "--"+bf.namePrefix+category)
			}
		}
		if len(set) > 1 {
			return fmt.Errorf("%s are mutually exclusive", strings.Join(set, ", "))
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
