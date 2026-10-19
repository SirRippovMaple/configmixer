package cmd

import (
	"fmt"

	"configmixer/internal/mixin"

	"github.com/spf13/cobra"
)

// registerCategoryIDFlags registers one string flag per category (a mixin id
// override for that category) and returns the map of flag value pointers.
func registerCategoryIDFlags(c *cobra.Command, catalog *mixin.Catalog) map[string]*string {
	categoryFlags := map[string]*string{}
	for _, category := range catalog.CategoryNames() {
		v := c.Flags().String(category, "", fmt.Sprintf("mixin id to select for category %q", category))
		categoryFlags[category] = v
	}
	return categoryFlags
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
