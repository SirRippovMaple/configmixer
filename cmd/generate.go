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
	var noAuto bool

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a .env file by selecting one mixin per category",
		// DisableFlagParsing lets us peek --mixins-dir, load the catalog, register
		// per-category flags, and then parse the full flag set in one pass.
		DisableFlagParsing: true,
		RunE: func(c *cobra.Command, args []string) error {
			c.Flags().StringVar(&mixinsDir, "mixins-dir", ".configmixer", "directory containing mixin categories")
			c.Flags().StringVar(&outPath, "out", ".env", "path to write the generated .env file")
			c.Flags().BoolVar(&noAuto, "no-auto", false, "disable auto-selecting the only compatible mixin for every category")

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

			categoryFlags := registerCategoryIDFlags(c, catalog)
			noAutoFlags := map[string]*bool{}
			for _, category := range catalog.CategoryNames() {
				na := c.Flags().Bool("no-auto-"+category, false, fmt.Sprintf("disable auto-selecting the only compatible mixin for category %q", category))
				noAutoFlags[category] = na
			}

			if helpRequested {
				return c.Help()
			}

			if err := c.Flags().Parse(args); err != nil {
				return err
			}

			return runGenerate(catalog, categoryFlags, noAuto, noAutoFlags, outPath)
		},
	}
	return cmd
}

func runGenerate(catalog *mixin.Catalog, categoryFlags map[string]*string, noAuto bool, noAutoFlags map[string]*bool, outPath string) error {
	sel, err := selectionFromFlags(catalog, categoryFlags)
	if err != nil {
		return err
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
		if len(choices) == 1 && !noAuto && !*noAutoFlags[category] {
			sel[category] = choices[0]
			fmt.Printf("auto-selected %s: %s\n", category, choices[0].ID)
			continue
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

	if err := writeEnvFile(outPath, sel, kv); err != nil {
		return err
	}

	fmt.Printf("wrote %s\n", outPath)
	return nil
}

// writeEnvFile writes kv as a .env file to outPath, preceded by a marker
// comment embedding sel so a later `regenerate` can reproduce it without
// prompting.
func writeEnvFile(outPath string, sel mixin.Selection, kv map[string]string) error {
	comment, err := mixin.EncodeSelectionComment(sel)
	if err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating %q: %w", outPath, err)
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, comment); err != nil {
		return fmt.Errorf("writing %q: %w", outPath, err)
	}
	if err := dotenv.Write(f, kv); err != nil {
		return fmt.Errorf("writing %q: %w", outPath, err)
	}
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
