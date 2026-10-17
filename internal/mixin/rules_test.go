package mixin

import "testing"

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

func TestParseExprErrors(t *testing.T) {
	bad := []string{"", "env ==", "env == dev &&", "(env == dev", "env === dev"}
	for _, expr := range bad {
		if _, err := ParseExpr(expr); err == nil {
			t.Errorf("ParseExpr(%q) expected error, got nil", expr)
		}
	}
}
