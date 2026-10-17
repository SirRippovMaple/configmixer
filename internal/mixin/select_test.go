package mixin

import "testing"

func testCatalog() *Catalog {
	dev := Mixin{ID: "dev", Category: "environment", Tags: map[string]string{"env": "dev"}}
	prod := Mixin{ID: "prod", Category: "environment", Tags: map[string]string{"env": "prod"}}

	devReq, _ := ParseExpr("env == dev")
	prodReq, _ := ParseExpr("env == prod")

	acme := Mixin{ID: "acme", Category: "tenant", Tags: map[string]string{"tenant": "acme"}, Requires: []Expr{devReq}}
	globex := Mixin{ID: "globex", Category: "tenant", Tags: map[string]string{"tenant": "globex"}, Requires: []Expr{prodReq}}

	fbDev := Mixin{ID: "dev", Category: "facebook-app", Tags: map[string]string{"fb-app": "dev"}, Requires: []Expr{devReq}}
	fbProd := Mixin{ID: "prod", Category: "facebook-app", Tags: map[string]string{"fb-app": "prod"}, Requires: []Expr{prodReq}}

	fbAccount := Mixin{ID: "main", Category: "facebook-account", Tags: map[string]string{"fb-account": "main"}}

	return &Catalog{Categories: map[string][]Mixin{
		"environment":      {dev, prod},
		"tenant":           {acme, globex},
		"facebook-app":     {fbDev, fbProd},
		"facebook-account": {fbAccount},
	}}
}

func TestValidate(t *testing.T) {
	cat := testCatalog()
	dev, _ := cat.Find("environment", "dev")
	prod, _ := cat.Find("environment", "prod")
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
		t.Errorf("expected valid selection, got error: %v", err)
	}

	invalid := Selection{
		"environment":      dev,
		"facebook-app":     fbProd,
		"facebook-account": account,
	}
	if err := Validate(invalid); err == nil {
		t.Error("expected error for dev environment + prod facebook-app, got nil")
	}

	invalidTenant := Selection{
		"environment": prod,
		"tenant":      acme,
	}
	if err := Validate(invalidTenant); err == nil {
		t.Error("expected error for prod environment + acme tenant, got nil")
	}
}

func TestCompatibleChoices(t *testing.T) {
	cat := testCatalog()
	dev, _ := cat.Find("environment", "dev")

	partial := Selection{"environment": dev}
	choices := CompatibleChoices(cat, "facebook-app", partial)
	if len(choices) != 1 || choices[0].ID != "dev" {
		t.Errorf("expected only dev facebook-app to be compatible, got %v", choices)
	}

	tenantChoices := CompatibleChoices(cat, "tenant", partial)
	if len(tenantChoices) != 1 || tenantChoices[0].ID != "acme" {
		t.Errorf("expected only acme tenant to be compatible, got %v", tenantChoices)
	}

	accountChoices := CompatibleChoices(cat, "facebook-account", partial)
	if len(accountChoices) != 1 {
		t.Errorf("expected facebook-account to always be compatible, got %v", accountChoices)
	}
}
