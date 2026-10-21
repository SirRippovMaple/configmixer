package mixin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// selectorScriptInput is the JSON document written to a selector script's
// stdin.
type selectorScriptInput struct {
	// Category is the category the script is selecting a mixin for.
	Category string `json:"category"`
	// Selection maps each already-fixed category to its selected mixin ID.
	Selection map[string]string `json:"selection"`
	// Mixins lists every mixin ID in Category.
	Mixins []string `json:"mixins"`
	// Valid lists the mixin IDs in Category that don't definitely conflict
	// with Selection (see ValidSelectorChoices).
	Valid []string `json:"valid"`
}

// ValidSelectorChoices returns the mixins in category that introduce no
// definite conflict with partial (see IntroducedConflicts). Unlike
// CompatibleChoices, a mixin whose clauses depend on a still-open category
// is kept, since it can't be ruled out yet.
func ValidSelectorChoices(cat *Catalog, category string, partial Selection) []Mixin {
	var out []Mixin
	for _, m := range cat.Categories[category] {
		if len(IntroducedConflicts(partial, category, m)) == 0 {
			out = append(out, m)
		}
	}
	return out
}

// RunSelectorScript runs spec.Path to select a mixin ID for category. The
// script receives a JSON object on stdin describing the category, the
// current partial selection, every mixin ID in the category, and the subset
// of those that are valid given the selection. It prints the chosen mixin ID
// on stdout; empty output (after trimming whitespace) means the script
// declines to choose, and is returned as "". A non-zero exit is an error.
func RunSelectorScript(spec SelectorScriptSpec, cat *Catalog, category string, sel Selection) (string, error) {
	input := selectorScriptInput{
		Category:  category,
		Selection: make(map[string]string, len(sel)),
		Mixins:    []string{},
		Valid:     []string{},
	}
	for c, m := range sel {
		input.Selection[c] = m.ID
	}
	for _, m := range cat.Categories[category] {
		input.Mixins = append(input.Mixins, m.ID)
	}
	for _, m := range ValidSelectorChoices(cat, category, sel) {
		input.Valid = append(input.Valid, m.ID)
	}

	stdin, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("encoding input for selector script %q: %w", spec.Path, err)
	}

	cmd := exec.Command(spec.Path)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running selector script %q for category %q: %w (stderr: %s)", spec.Path, category, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}
