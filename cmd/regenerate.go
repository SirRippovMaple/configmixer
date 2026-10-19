package cmd

import (
	"fmt"
	"os"
	"strings"

	"configmixer/internal/mixin"

	"github.com/spf13/cobra"
)

func newRegenerateCmd() *cobra.Command {
	var mixinsDir string
	var outPath string

	cmd := &cobra.Command{
		Use:   "regenerate",
		Short: "Regenerate a .env file from its embedded mixin selection, without prompting",
		// DisableFlagParsing mirrors generate's approach: peek --mixins-dir, load
		// the catalog, register per-category override flags, then parse once.
		DisableFlagParsing: true,
		RunE: func(c *cobra.Command, args []string) error {
			c.Flags().StringVar(&mixinsDir, "mixins-dir", ".configmixer", "directory containing mixin categories")
			c.Flags().StringVar(&outPath, "out", ".env", "path to the .env file to regenerate (read from and overwritten)")

			mixinsDir = peekStringFlag(args, "mixins-dir", mixinsDir)
			helpRequested := hasHelpFlag(args)

			catalog, err := mixin.LoadAll(mixinsDir)
			if err != nil {
				if helpRequested {
					return c.Help()
				}
				return err
			}

			categoryFlags := registerCategoryIDFlags(c, catalog)

			if helpRequested {
				return c.Help()
			}

			if err := c.Flags().Parse(args); err != nil {
				return err
			}

			return runRegenerate(catalog, categoryFlags, outPath)
		},
	}
	return cmd
}

func runRegenerate(catalog *mixin.Catalog, categoryFlags map[string]*string, outPath string) error {
	embedded, err := readEmbeddedSelection(outPath)
	if err != nil {
		return err
	}

	sel, err := selectionFromFlags(catalog, categoryFlags)
	if err != nil {
		return err
	}

	for category, id := range embedded {
		if _, ok := sel[category]; ok {
			continue
		}
		if _, exists := catalog.Categories[category]; !exists {
			continue
		}
		m, ok := catalog.Find(category, id)
		if !ok {
			return fmt.Errorf("embedded selection for category %q references unknown mixin %q; pass --%s to override", category, id, category)
		}
		sel[category] = m
	}

	order, err := catalog.OrderedCategories()
	if err != nil {
		return err
	}

	for _, category := range order {
		if _, ok := sel[category]; ok {
			continue
		}
		choices := mixin.CompatibleChoices(catalog, category, sel)
		switch len(choices) {
		case 0:
			return fmt.Errorf("no compatible mixin available for category %q given current selection", category)
		case 1:
			sel[category] = choices[0]
			fmt.Printf("auto-selected %s: %s\n", category, choices[0].ID)
		default:
			return fmt.Errorf("category %q has no embedded or flag-provided selection and %d compatible mixins; pass --%s to disambiguate (regenerate does not prompt)", category, len(choices), category)
		}
	}

	if err := mixin.Validate(sel); err != nil {
		return err
	}

	kv, err := mixin.Merge(sel)
	if err != nil {
		return err
	}

	if err := writeEnvFile(outPath, sel, kv); err != nil {
		return err
	}

	fmt.Printf("wrote %s\n", outPath)
	return nil
}

// readEmbeddedSelection reads the marker comment line written by `generate`
// (see mixin.EncodeSelectionComment) back out of an existing .env file.
func readEmbeddedSelection(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", path, err)
	}
	return mixin.DecodeSelectionComment(strings.Split(string(data), "\n"))
}
