package mixin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunSelectorScriptInput(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.json")
	path := writeScript(t, dir, "select.sh", `cat > '`+inputPath+`'; echo acme`)

	cat := testCatalog()
	dev, _ := cat.Find("environment", "dev")
	id, err := RunSelectorScript(SelectorScriptSpec{Path: path}, cat, "tenant", Selection{"environment": dev})
	if err != nil {
		t.Fatalf("RunSelectorScript: %v", err)
	}
	if id != "acme" {
		t.Errorf("id = %q, want acme", id)
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("reading captured input: %v", err)
	}
	var input selectorScriptInput
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatalf("parsing captured input %q: %v", data, err)
	}
	if input.Category != "tenant" {
		t.Errorf("category = %q, want tenant", input.Category)
	}
	if input.Selection["environment"] != "dev" || len(input.Selection) != 1 {
		t.Errorf("selection = %v, want {environment: dev}", input.Selection)
	}
	if len(input.Mixins) != 2 || indexOf(input.Mixins, "acme") < 0 || indexOf(input.Mixins, "globex") < 0 {
		t.Errorf("mixins = %v, want [acme globex]", input.Mixins)
	}
	if len(input.Valid) != 1 || input.Valid[0] != "acme" {
		t.Errorf("valid = %v, want [acme]", input.Valid)
	}
}

func TestRunSelectorScriptValidKeepsUndecidable(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.json")
	path := writeScript(t, dir, "select.sh", `cat > '`+inputPath+`'`)

	// environment is still open, so neither tenant can be ruled out yet.
	if _, err := RunSelectorScript(SelectorScriptSpec{Path: path}, testCatalog(), "tenant", Selection{}); err != nil {
		t.Fatalf("RunSelectorScript: %v", err)
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("reading captured input: %v", err)
	}
	var input selectorScriptInput
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatalf("parsing captured input %q: %v", data, err)
	}
	if len(input.Valid) != 2 {
		t.Errorf("valid = %v, want both tenants", input.Valid)
	}
}

func TestRunSelectorScriptTrimsOutput(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "select.sh", `printf '  acme \n\n'`)

	id, err := RunSelectorScript(SelectorScriptSpec{Path: path}, testCatalog(), "tenant", Selection{})
	if err != nil {
		t.Fatalf("RunSelectorScript: %v", err)
	}
	if id != "acme" {
		t.Errorf("id = %q, want acme", id)
	}
}

func TestRunSelectorScriptDecline(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "select.sh", `exit 0`)

	id, err := RunSelectorScript(SelectorScriptSpec{Path: path}, testCatalog(), "tenant", Selection{})
	if err != nil {
		t.Fatalf("RunSelectorScript: %v", err)
	}
	if id != "" {
		t.Errorf("id = %q, want empty (decline)", id)
	}
}

func TestRunSelectorScriptFailure(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "select.sh", `echo acme; echo boom >&2; exit 1`)

	if _, err := RunSelectorScript(SelectorScriptSpec{Path: path}, testCatalog(), "tenant", Selection{}); err == nil {
		t.Error("expected error for non-zero exit, got nil")
	}
}
