package cmd

import (
	"fmt"
	"strings"

	"configmixer/internal/mixin"
)

// resolveConfigured adds a configured selection into sel for every category
// that doesn't already have a fixed selection (from a flag or an embedded
// selection) and isn't forced to prompt via --prompt-<category>. forcedPrompt
// may be nil for commands without a prompt flag.
//
// For each such category, the configured selector script runs first (unless
// opted out via --no-script-<category>). If it declines, or picks a mixin that
// is unknown or definitely conflicts with what's already selected, the
// configured default is used instead (unless opted out via
// --no-default-<category>). Categories are visited in topological order, so a
// selector script sees every earlier script/default pick in sel.
//
// Configured selections are treated exactly like flag selections: once
// resolved they are fixed, so the interactive/auto resolution of the remaining
// open categories filters against them and cannot produce a combination that
// invalidates one. Defaults are deliberately NOT checked for compatibility
// here — a default may legitimately depend on a category that is still open,
// so validating it in isolation would reject valid selections. For the same
// reason a selector script's pick is only rejected for conflicts that are
// already definite (see mixin.IntroducedConflicts). Genuine conflicts
// (mutually incompatible defaults, or a default that contradicts a flag) are
// caught by the final mixin.Validate once every category is concrete.
func resolveConfigured(catalog *mixin.Catalog, cfg *mixin.Config, order []string, sel mixin.Selection, noScriptFlags, noDefaultFlags map[string]*bool, forcedPrompt map[string]*bool) error {
	for _, category := range order {
		if _, ok := sel[category]; ok {
			continue
		}
		if forcedPrompt != nil && forcedPrompt[category] != nil && *forcedPrompt[category] {
			continue
		}

		if spec, ok := cfg.SelectorScripts[category]; ok && (noScriptFlags[category] == nil || !*noScriptFlags[category]) {
			m, ok, err := runSelectorScript(catalog, spec, category, sel)
			if err != nil {
				return err
			}
			if ok {
				sel[category] = m
				fmt.Printf("script-selected %s: %s\n", category, m.ID)
				continue
			}
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

// runSelectorScript runs category's selector script against sel and returns
// its pick. ok is false if the script declined or picked an invalid mixin, in
// which case a notice explaining why is printed.
func runSelectorScript(catalog *mixin.Catalog, spec mixin.SelectorScriptSpec, category string, sel mixin.Selection) (m mixin.Mixin, ok bool, err error) {
	id, err := mixin.RunSelectorScript(spec, catalog, category, sel)
	if err != nil {
		return mixin.Mixin{}, false, err
	}
	if id == "" {
		fmt.Printf("selector script for %s declined to select a mixin\n", category)
		return mixin.Mixin{}, false, nil
	}
	m, found := catalog.Find(category, id)
	if !found {
		fmt.Printf("selector script for %s selected unknown mixin %q; ignoring\n", category, id)
		return mixin.Mixin{}, false, nil
	}
	if conflicts := mixin.IntroducedConflicts(sel, category, m); len(conflicts) > 0 {
		fmt.Printf("selector script for %s selected incompatible mixin %q (%s); ignoring\n", category, id, strings.Join(conflicts, "; "))
		return mixin.Mixin{}, false, nil
	}
	return m, true, nil
}
