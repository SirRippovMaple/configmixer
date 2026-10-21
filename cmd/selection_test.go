package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"configmixer/internal/mixin"
)

func testSelectionCatalog(t *testing.T) *mixin.Catalog {
	t.Helper()
	devReq, err := mixin.ParseExpr("env == dev")
	if err != nil {
		t.Fatal(err)
	}
	prodReq, err := mixin.ParseExpr("env == prod")
	if err != nil {
		t.Fatal(err)
	}

	dev := mixin.Mixin{ID: "dev", Category: "environment", Tags: map[string]string{"env": "dev"}}
	prod := mixin.Mixin{ID: "prod", Category: "environment", Tags: map[string]string{"env": "prod"}}

	fbDev := mixin.Mixin{ID: "dev", Category: "facebook-app", Tags: map[string]string{"fb-app": "dev"}, Requires: []mixin.Expr{devReq}}
	fbProd := mixin.Mixin{ID: "prod", Category: "facebook-app", Tags: map[string]string{"fb-app": "prod"}, Requires: []mixin.Expr{prodReq}}

	return &mixin.Catalog{Categories: map[string][]mixin.Mixin{
		"environment":  {dev, prod},
		"facebook-app": {fbDev, fbProd},
	}}
}

func boolFlags(categories ...string) map[string]*bool {
	out := map[string]*bool{}
	for _, c := range categories {
		v := false
		out[c] = &v
	}
	return out
}

// TestResolveDefaultsFiltersOpenCategories covers the bug this change fixes.
// environment is open (no default, resolved interactively/auto) and is first
// in topological order; facebook-app's configured default ("prod") requires
// env==prod. Because the default is pre-resolved into the selection, the open
// environment category must filter down to only "prod" — picking "dev" can no
// longer be offered and then blow up when facebook-app's default is applied.
func TestResolveDefaultsFiltersOpenCategories(t *testing.T) {
	cat := testSelectionCatalog(t)
	cfg := &mixin.Config{Defaults: map[string]string{"facebook-app": "prod"}}
	order := []string{"environment", "facebook-app"}
	noDefault := boolFlags("environment", "facebook-app")

	sel := mixin.Selection{}
	if err := resolveConfigured(cat, cfg, order, sel, nil, noDefault, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := sel["facebook-app"]; !ok {
		t.Fatal("expected facebook-app default to be pre-resolved into the selection")
	}

	choices := mixin.CompatibleChoices(cat, "environment", sel)
	if len(choices) != 1 || choices[0].ID != "prod" {
		t.Fatalf("expected only %q to remain compatible with the facebook-app default, got %v", "prod", choices)
	}
}

func TestResolveDefaultsSkipsNoDefaultAndForcedPrompt(t *testing.T) {
	cat := testSelectionCatalog(t)
	cfg := &mixin.Config{Defaults: map[string]string{"facebook-app": "prod"}}
	order := []string{"environment", "facebook-app"}

	noDefault := boolFlags("environment", "facebook-app")
	*noDefault["facebook-app"] = true
	sel := mixin.Selection{}
	if err := resolveConfigured(cat, cfg, order, sel, nil, noDefault, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := sel["facebook-app"]; ok {
		t.Fatal("expected facebook-app default to be skipped due to --no-default-facebook-app")
	}

	forcedPrompt := boolFlags("environment", "facebook-app")
	*forcedPrompt["facebook-app"] = true
	sel2 := mixin.Selection{}
	if err := resolveConfigured(cat, cfg, order, sel2, nil, boolFlags("environment", "facebook-app"), forcedPrompt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := sel2["facebook-app"]; ok {
		t.Fatal("expected facebook-app default to be skipped due to --prompt-facebook-app")
	}
}

func TestResolveDefaultsDoesNotOverrideExistingSelection(t *testing.T) {
	cat := testSelectionCatalog(t)
	fbDev, _ := cat.Find("facebook-app", "dev")
	cfg := &mixin.Config{Defaults: map[string]string{"facebook-app": "prod"}}
	order := []string{"environment", "facebook-app"}

	sel := mixin.Selection{"facebook-app": fbDev} // e.g. set by a flag
	if err := resolveConfigured(cat, cfg, order, sel, nil, boolFlags("environment", "facebook-app"), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := sel["facebook-app"]; got.ID != "dev" {
		t.Fatalf("expected existing selection %q to be preserved, got %q", "dev", got.ID)
	}
}

func TestResolveDefaultsErrorsOnMissingMixin(t *testing.T) {
	cat := testSelectionCatalog(t)
	cfg := &mixin.Config{Defaults: map[string]string{"facebook-app": "ghost"}}
	order := []string{"environment", "facebook-app"}

	err := resolveConfigured(cat, cfg, order, mixin.Selection{}, nil, boolFlags("environment", "facebook-app"), nil)
	if err == nil {
		t.Fatal("expected an error for a default referencing a non-existent mixin")
	}
}

func writeSelectorScript(t *testing.T, body string) mixin.SelectorScriptSpec {
	t.Helper()
	path := filepath.Join(t.TempDir(), "select.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatalf("writing selector script: %v", err)
	}
	return mixin.SelectorScriptSpec{Path: path}
}

// TestResolveConfiguredSelectorScript covers how a selector script's outcome
// interacts with the facebook-app default ("prod").
func TestResolveConfiguredSelectorScript(t *testing.T) {
	cat := testSelectionCatalog(t)
	dev, _ := cat.Find("environment", "dev")
	prod, _ := cat.Find("environment", "prod")
	order := []string{"environment", "facebook-app"}

	tests := []struct {
		name     string
		script   string
		sel      mixin.Selection
		noScript bool
		want     string
	}{
		{"script beats default", "echo dev", mixin.Selection{"environment": dev}, false, "dev"},
		{"decline falls back to default", "true", mixin.Selection{}, false, "prod"},
		{"unknown id falls back to default", "echo ghost", mixin.Selection{}, false, "prod"},
		{"definite conflict falls back to default", "echo dev", mixin.Selection{"environment": prod}, false, "prod"},
		{"undecidable pick is accepted", "echo dev", mixin.Selection{}, false, "dev"},
		{"--no-script falls back to default", "echo dev", mixin.Selection{"environment": dev}, true, "prod"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &mixin.Config{
				Defaults:        map[string]string{"facebook-app": "prod"},
				SelectorScripts: map[string]mixin.SelectorScriptSpec{"facebook-app": writeSelectorScript(t, tt.script)},
			}
			noScript := boolFlags("environment", "facebook-app")
			*noScript["facebook-app"] = tt.noScript

			if err := resolveConfigured(cat, cfg, order, tt.sel, noScript, boolFlags("environment", "facebook-app"), nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := tt.sel["facebook-app"].ID; got != tt.want {
				t.Errorf("facebook-app = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveConfiguredSelectorScriptSkippedWhenSelectedOrPrompted(t *testing.T) {
	cat := testSelectionCatalog(t)
	fbProd, _ := cat.Find("facebook-app", "prod")
	order := []string{"environment", "facebook-app"}
	// Running the script would fail the test, proving it wasn't run.
	cfg := &mixin.Config{SelectorScripts: map[string]mixin.SelectorScriptSpec{"facebook-app": writeSelectorScript(t, "exit 1")}}

	sel := mixin.Selection{"facebook-app": fbProd} // e.g. set by a flag
	if err := resolveConfigured(cat, cfg, order, sel, nil, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := sel["facebook-app"].ID; got != "prod" {
		t.Errorf("expected existing selection to be preserved, got %q", got)
	}

	forcedPrompt := boolFlags("environment", "facebook-app")
	*forcedPrompt["facebook-app"] = true
	sel2 := mixin.Selection{}
	if err := resolveConfigured(cat, cfg, order, sel2, nil, nil, forcedPrompt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := sel2["facebook-app"]; ok {
		t.Error("expected facebook-app to be left open due to --prompt-facebook-app")
	}
}

func TestResolveConfiguredSelectorScriptError(t *testing.T) {
	cat := testSelectionCatalog(t)
	order := []string{"environment", "facebook-app"}
	cfg := &mixin.Config{
		Defaults:        map[string]string{"facebook-app": "prod"},
		SelectorScripts: map[string]mixin.SelectorScriptSpec{"facebook-app": writeSelectorScript(t, "exit 1")},
	}

	if err := resolveConfigured(cat, cfg, order, mixin.Selection{}, nil, nil, nil); err == nil {
		t.Fatal("expected a failing selector script to return an error")
	}
}
