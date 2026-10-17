package mixin

import "testing"

func TestMerge(t *testing.T) {
	sel := Selection{
		"environment": {ID: "dev", Env: map[string]string{"APP_ENV": "development"}},
		"tenant":      {ID: "acme", Env: map[string]string{"TENANT_ID": "acme"}},
	}
	kv, err := Merge(sel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv["APP_ENV"] != "development" || kv["TENANT_ID"] != "acme" {
		t.Errorf("unexpected merged env: %v", kv)
	}
}

func TestMergeDuplicateKey(t *testing.T) {
	sel := Selection{
		"environment": {ID: "dev", Env: map[string]string{"SHARED_KEY": "a"}},
		"tenant":      {ID: "acme", Env: map[string]string{"SHARED_KEY": "b"}},
	}
	if _, err := Merge(sel); err == nil {
		t.Error("expected error for duplicate key, got nil")
	}
}
