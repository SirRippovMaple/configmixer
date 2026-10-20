package mixin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAll(t *testing.T) {
	cat, err := LoadAll("../../testdata/.configmixer")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	wantCategories := []string{"environment", "facebook-account", "facebook-app", "tenant"}
	gotCategories := cat.CategoryNames()
	if len(gotCategories) != len(wantCategories) {
		t.Fatalf("categories = %v, want %v", gotCategories, wantCategories)
	}
	for i, c := range wantCategories {
		if gotCategories[i] != c {
			t.Fatalf("categories = %v, want %v", gotCategories, wantCategories)
		}
	}

	dev, ok := cat.Find("environment", "dev")
	if !ok {
		t.Fatal("expected to find environment/dev")
	}
	if dev.Tags["env"] != "dev" {
		t.Errorf("dev.Tags[env] = %q, want dev", dev.Tags["env"])
	}
	if dev.Env["APP_ENV"] != "development" {
		t.Errorf("dev.Env[APP_ENV] = %q, want development", dev.Env["APP_ENV"])
	}

	acme, ok := cat.Find("tenant", "acme")
	if !ok {
		t.Fatal("expected to find tenant/acme")
	}
	if len(acme.Requires) != 1 {
		t.Fatalf("acme.Requires = %v, want 1 entry", acme.Requires)
	}
}

func TestLoadAllIgnoresDotDirectories(t *testing.T) {
	root := t.TempDir()

	envDir := filepath.Join(root, "environment")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "dev.yaml"), []byte("tags:\n  env: dev\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	hiddenDir := filepath.Join(root, ".scripts")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "stray.yaml"), []byte("tags:\n  bogus: true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cat, err := LoadAll(root)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	names := cat.CategoryNames()
	if len(names) != 1 || names[0] != "environment" {
		t.Fatalf("categories = %v, want only [environment]", names)
	}
}

func TestLoadAllAndValidateFullSelection(t *testing.T) {
	cat, err := LoadAll("../../testdata/.configmixer")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	dev, _ := cat.Find("environment", "dev")
	acme, _ := cat.Find("tenant", "acme")
	fbDev, _ := cat.Find("facebook-app", "dev")
	fbProd, _ := cat.Find("facebook-app", "prod")
	account, _ := cat.Find("facebook-account", "main")

	valid := Selection{
		"environment":      dev,
		"tenant":           acme,
		"facebook-app":     fbDev,
		"facebook-account": account,
	}
	if err := Validate(valid); err != nil {
		t.Errorf("expected valid selection from testdata, got: %v", err)
	}

	invalid := Selection{
		"environment":      dev,
		"tenant":           acme,
		"facebook-app":     fbProd,
		"facebook-account": account,
	}
	if err := Validate(invalid); err == nil {
		t.Error("expected error combining dev environment with prod facebook-app")
	}
}
