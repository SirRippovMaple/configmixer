package mixin

import (
	"slices"
	"sort"
	"testing"
)

func TestParseExprEval(t *testing.T) {
	cases := []struct {
		expr string
		tags map[string]string
		want bool
	}{
		{"env == dev", map[string]string{"env": "dev"}, true},
		{"env == dev", map[string]string{"env": "prod"}, false},
		{"env != dev", map[string]string{"env": "prod"}, true},
		{"env != dev", map[string]string{}, true},
		{"env == dev && tenant == acme", map[string]string{"env": "dev", "tenant": "acme"}, true},
		{"env == dev && tenant == acme", map[string]string{"env": "dev", "tenant": "globex"}, false},
		{"env == dev || env == prod", map[string]string{"env": "prod"}, true},
		{"(env == dev || env == prod) && tenant == acme", map[string]string{"env": "prod", "tenant": "acme"}, true},
		{"(env == dev || env == prod) && tenant == acme", map[string]string{"env": "prod", "tenant": "globex"}, false},
	}
	for _, c := range cases {
		e, err := ParseExpr(c.expr)
		if err != nil {
			t.Fatalf("ParseExpr(%q): %v", c.expr, err)
		}
		if got := e.Eval(c.tags); got != c.want {
			t.Errorf("ParseExpr(%q).Eval(%v) = %v, want %v", c.expr, c.tags, got, c.want)
		}
	}
}

func TestExprKeys(t *testing.T) {
	cases := []struct {
		expr string
		want []string
	}{
		{"env == dev", []string{"env"}},
		{"env != dev", []string{"env"}},
		{"env == dev && tenant == acme", []string{"env", "tenant"}},
		{"env == dev || env == prod", []string{"env", "env"}},
		{"(env == dev || tenant == acme) && region == eu", []string{"env", "tenant", "region"}},
	}
	for _, c := range cases {
		e, err := ParseExpr(c.expr)
		if err != nil {
			t.Fatalf("ParseExpr(%q): %v", c.expr, err)
		}
		got := e.Keys()
		sort.Strings(got)
		want := slices.Clone(c.want)
		sort.Strings(want)
		if !slices.Equal(got, want) {
			t.Errorf("ParseExpr(%q).Keys() = %v, want %v", c.expr, got, want)
		}
	}
}

func TestParseExprErrors(t *testing.T) {
	bad := []string{"", "env ==", "env == dev &&", "(env == dev", "env === dev"}
	for _, expr := range bad {
		if _, err := ParseExpr(expr); err == nil {
			t.Errorf("ParseExpr(%q) expected error, got nil", expr)
		}
	}
}
