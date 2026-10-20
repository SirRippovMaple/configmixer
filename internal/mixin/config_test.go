package mixin

import (
	"os"
	"path/filepath"
	"testing"
)

func testConfigCatalog() *Catalog {
	return &Catalog{Categories: map[string][]Mixin{
		"environment": {{ID: "dev", Category: "environment"}, {ID: "prod", Category: "environment"}},
	}}
}

func writeConfigFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing config.yaml: %v", err)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir, testConfigCatalog())
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Defaults) != 0 {
		t.Errorf("Defaults = %v, want empty", cfg.Defaults)
	}
}

func TestLoadConfigValid(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, "defaults:\n  environment: dev\n")

	cfg, err := LoadConfig(dir, testConfigCatalog())
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Defaults["environment"] != "dev" {
		t.Errorf("Defaults[environment] = %q, want dev", cfg.Defaults["environment"])
	}
}

func TestLoadConfigUnknownCategory(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, "defaults:\n  bogus: dev\n")

	if _, err := LoadConfig(dir, testConfigCatalog()); err == nil {
		t.Error("expected error for unknown category, got nil")
	}
}

func TestLoadConfigUnknownMixinID(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, "defaults:\n  environment: staging\n")

	if _, err := LoadConfig(dir, testConfigCatalog()); err == nil {
		t.Error("expected error for unknown mixin id, got nil")
	}
}
