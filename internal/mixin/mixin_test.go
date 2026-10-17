package mixin

import "testing"

func TestLoadAll(t *testing.T) {
	cat, err := LoadAll("../../testdata/mixins")
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

func TestLoadAllAndValidateFullSelection(t *testing.T) {
	cat, err := LoadAll("../../testdata/mixins")
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
