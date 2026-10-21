package cmd

import (
	"fmt"

	"configmixer/internal/mixin"
)

// resolveDefaults adds the configured default mixin into sel for every
// category that doesn't already have a fixed selection (from a flag or an
// embedded selection) and isn't opted out via --no-default-<category> or
// forced to prompt via --prompt-<category>. forcedPrompt may be nil for
// commands without a prompt flag.
//
// Defaults are treated exactly like flag selections: once resolved they are
// fixed, so the interactive/auto resolution of the remaining open categories
// filters against them and cannot produce a combination that invalidates a
// default. Compatibility is deliberately NOT checked here — a default may
// legitimately depend on a category that is still open, so validating it in
// isolation would reject valid selections. Genuine conflicts (mutually
// incompatible defaults, or a default that contradicts a flag) are caught by
// the final mixin.Validate once every category is concrete.
func resolveDefaults(catalog *mixin.Catalog, cfg *mixin.Config, order []string, sel mixin.Selection, noDefaultFlags map[string]*bool, forcedPrompt map[string]*bool) error {
	for _, category := range order {
		if _, ok := sel[category]; ok {
			continue
		}
		if forcedPrompt != nil && forcedPrompt[category] != nil && *forcedPrompt[category] {
			continue
		}
		defaultID, ok := cfg.Defaults[category]
		if !ok || (noDefaultFlags[category] != nil && *noDefaultFlags[category]) {
			continue
		}
		m, ok := catalog.Find(category, defaultID)
		if !ok {
			return fmt.Errorf("default mixin %q for category %q no longer exists", defaultID, category)
		}
		sel[category] = m
		fmt.Printf("default %s: %s\n", category, m.ID)
	}
	return nil
}
