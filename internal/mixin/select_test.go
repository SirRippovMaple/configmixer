package mixin

import "testing"

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

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

func TestOrderedCategories(t *testing.T) {
	cat := testCatalog()
	order, err := cat.OrderedCategories()
	if err != nil {
		t.Fatalf("OrderedCategories: %v", err)
	}
	if len(order) != len(cat.Categories) {
		t.Fatalf("expected %d categories, got %v", len(cat.Categories), order)
	}

	envIdx := indexOf(order, "environment")
	tenantIdx := indexOf(order, "tenant")
	fbAppIdx := indexOf(order, "facebook-app")

	if envIdx > tenantIdx {
		t.Errorf("expected environment before tenant, got order %v", order)
	}
	if envIdx > fbAppIdx {
		t.Errorf("expected environment before facebook-app, got order %v", order)
	}
}

func TestOrderedCategoriesCycle(t *testing.T) {
	aReq, _ := ParseExpr("b == x")
	bReq, _ := ParseExpr("a == x")

	a := Mixin{ID: "a1", Category: "a", Tags: map[string]string{"a": "x"}, Requires: []Expr{aReq}}
	b := Mixin{ID: "b1", Category: "b", Tags: map[string]string{"b": "x"}, Requires: []Expr{bReq}}

	cat := &Catalog{Categories: map[string][]Mixin{
		"a": {a},
		"b": {b},
	}}

	if _, err := cat.OrderedCategories(); err == nil {
		t.Error("expected error for dependency cycle, got nil")
	}
}

func TestIntroducedConflicts(t *testing.T) {
	cat := testCatalog()
	dev, _ := cat.Find("environment", "dev")
	acme, _ := cat.Find("tenant", "acme")
	globex, _ := cat.Find("tenant", "globex")
	fbProd, _ := cat.Find("facebook-app", "prod")

	// globex requires env == prod, which dev definitely contradicts.
	if got := IntroducedConflicts(Selection{"environment": dev}, "tenant", globex); len(got) != 1 {
		t.Errorf("expected one definite conflict for dev + globex, got %v", got)
	}
	if got := IntroducedConflicts(Selection{"environment": dev}, "tenant", acme); len(got) != 0 {
		t.Errorf("expected no conflict for dev + acme, got %v", got)
	}

	// environment is still open, so globex's env clause can't be judged yet.
	if got := IntroducedConflicts(Selection{}, "tenant", globex); len(got) != 0 {
		t.Errorf("expected clause on an open category to be skipped, got %v", got)
	}

	// dev + prod facebook-app already conflict; that's not acme's fault.
	partial := Selection{"environment": dev, "facebook-app": fbProd}
	if got := IntroducedConflicts(partial, "tenant", acme); len(got) != 0 {
		t.Errorf("expected pre-existing conflict not to be attributed, got %v", got)
	}
}
