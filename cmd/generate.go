package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"configmixer/internal/dotenv"
	"configmixer/internal/mixin"

	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	var mixinsDir string
	var outPath string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a .env file by selecting one mixin per category",
		// DisableFlagParsing lets us peek --mixins-dir, load the catalog, register
		// per-category flags, and then parse the full flag set in one pass.
		DisableFlagParsing: true,
		RunE: func(c *cobra.Command, args []string) error {
			c.Flags().StringVar(&mixinsDir, "mixins-dir", "./mixins", "directory containing mixin categories")
			c.Flags().StringVar(&outPath, "out", ".env", "path to write the generated .env file")

			// Per-category flags don't exist yet, so scan the raw args for
			// --mixins-dir by hand rather than doing a pflag pass (which would
			// error out on the first unrecognized flag).
			mixinsDir = peekStringFlag(args, "mixins-dir", mixinsDir)

			// cobra's automatic -h/--help handling is skipped by
			// DisableFlagParsing, so check for it manually. Register whatever
			// per-category flags we can (best-effort: the mixins dir may not
			// exist yet) before printing help.
			helpRequested := hasHelpFlag(args)

			catalog, err := mixin.LoadAll(mixinsDir)
			if err != nil {
				if helpRequested {
					return c.Help()
				}
				return err
			}

			categoryFlags := map[string]*string{}
			for _, category := range catalog.CategoryNames() {
				v := c.Flags().String(category, "", fmt.Sprintf("mixin id to select for category %q", category))
				categoryFlags[category] = v
			}

			if helpRequested {
				return c.Help()
			}

			if err := c.Flags().Parse(args); err != nil {
				return err
			}

			return runGenerate(catalog, categoryFlags, outPath)
		},
	}
	return cmd
}

func runGenerate(catalog *mixin.Catalog, categoryFlags map[string]*string, outPath string) error {
	sel := mixin.Selection{}

	for _, category := range catalog.CategoryNames() {
		id := *categoryFlags[category]
		if id == "" {
			continue
		}
		m, ok := catalog.Find(category, id)
		if !ok {
			return fmt.Errorf("unknown mixin %q for category %q", id, category)
		}
		sel[category] = m
	}

	order, err := catalog.OrderedCategories()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	for _, category := range order {
		if _, ok := sel[category]; ok {
			continue
		}
		choices := mixin.CompatibleChoices(catalog, category, sel)
		if len(choices) == 0 {
			return fmt.Errorf("no compatible mixin available for category %q given current selection", category)
		}
		chosen, err := promptChoice(reader, category, choices)
		if err != nil {
			return err
		}
		sel[category] = chosen
	}

	if err := mixin.Validate(sel); err != nil {
		return err
	}

	kv, err := mixin.Merge(sel)
	if err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating %q: %w", outPath, err)
	}
	defer f.Close()

	if err := dotenv.Write(f, kv); err != nil {
		return fmt.Errorf("writing %q: %w", outPath, err)
	}

	fmt.Printf("wrote %s\n", outPath)
	return nil
}

// peekStringFlag extracts the value of --name or --name=value from args
// without requiring the flag to be registered, falling back to def.
func peekStringFlag(args []string, name, def string) string {
	prefix := "--" + name
	for i, a := range args {
		if a == prefix && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := strings.CutPrefix(a, prefix+"="); ok {
			return v
		}
	}
	return def
}

func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

func promptChoice(reader *bufio.Reader, category string, choices []mixin.Mixin) (mixin.Mixin, error) {
	fmt.Printf("Select %s:\n", category)
	for i, m := range choices {
		fmt.Printf("  %d) %s\n", i+1, m.ID)
	}
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return mixin.Mixin{}, fmt.Errorf("reading input: %w", err)
		}
		line = strings.TrimSpace(line)
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(choices) {
			fmt.Printf("enter a number between 1 and %d\n", len(choices))
			continue
		}
		return choices[n-1], nil
	}
}
