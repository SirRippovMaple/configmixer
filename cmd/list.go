package cmd

import (
	"fmt"
	"sort"

	"configmixer/internal/mixin"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var mixinsDir string

	cmd := &cobra.Command{
		Use:   "list [category]",
		Short: "List available categories and mixins",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			catalog, err := mixin.LoadAll(mixinsDir)
			if err != nil {
				return err
			}

			categories := catalog.CategoryNames()
			if len(args) == 1 {
				if _, ok := catalog.Categories[args[0]]; !ok {
					return fmt.Errorf("unknown category %q", args[0])
				}
				categories = args
			}

			for _, category := range categories {
				fmt.Println(category + ":")
				for _, m := range catalog.Categories[category] {
					fmt.Printf("  %s  tags=%s\n", m.ID, formatTags(m.Tags))
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&mixinsDir, "mixins-dir", "./mixins", "directory containing mixin categories")
	return cmd
}

func formatTags(tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	s := "{"
	for i, k := range keys {
		if i > 0 {
			s += ", "
		}
		s += k + "=" + tags[k]
	}
	return s + "}"
}
