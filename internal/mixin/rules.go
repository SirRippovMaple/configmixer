package mixin

import (
	"fmt"
	"strings"
)

// Expr is a boolean expression evaluated against a set of tags.
type Expr interface {
	Eval(tags map[string]string) bool
	String() string
}

type cmpExpr struct {
	key   string
	value string
	neq   bool
}

func (e cmpExpr) Eval(tags map[string]string) bool {
	v, ok := tags[e.key]
	if e.neq {
		return !ok || v != e.value
	}
	return ok && v == e.value
}

func (e cmpExpr) String() string {
	op := "=="
	if e.neq {
		op = "!="
	}
	return fmt.Sprintf("%s %s %s", e.key, op, e.value)
}

type andExpr struct{ left, right Expr }

func (e andExpr) Eval(tags map[string]string) bool { return e.left.Eval(tags) && e.right.Eval(tags) }
func (e andExpr) String() string                   { return fmt.Sprintf("(%s && %s)", e.left, e.right) }

type orExpr struct{ left, right Expr }

func (e orExpr) Eval(tags map[string]string) bool { return e.left.Eval(tags) || e.right.Eval(tags) }
func (e orExpr) String() string                   { return fmt.Sprintf("(%s || %s)", e.left, e.right) }

// ParseExpr parses a tiny boolean tag-expression grammar:
//
//	expr   := or
//	or     := and ("||" and)*
//	and    := cmp ("&&" cmp)*
//	cmp    := "(" or ")" | IDENT ("==" | "!=") IDENT
func ParseExpr(s string) (Expr, error) {
	p := &exprParser{toks: tokenize(s), src: s}
	e, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.toks) {
		return nil, fmt.Errorf("unexpected token %q in expression %q", p.toks[p.pos], s)
	}
	return e, nil
}

func tokenize(s string) []string {
	var toks []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			toks = append(toks, cur.String())
			cur.Reset()
		}
	}
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case c == ' ' || c == '\t':
			flush()
		case c == '(' || c == ')':
			flush()
			toks = append(toks, string(c))
		case c == '&' && i+1 < len(runes) && runes[i+1] == '&':
			flush()
			toks = append(toks, "&&")
			i++
		case c == '|' && i+1 < len(runes) && runes[i+1] == '|':
			flush()
			toks = append(toks, "||")
			i++
		case c == '=' && i+1 < len(runes) && runes[i+1] == '=':
			flush()
			toks = append(toks, "==")
			i++
		case c == '!' && i+1 < len(runes) && runes[i+1] == '=':
			flush()
			toks = append(toks, "!=")
			i++
		default:
			cur.WriteRune(c)
		}
	}
	flush()
	return toks
}

type exprParser struct {
	toks []string
	pos  int
	src  string
}

func (p *exprParser) peek() string {
	if p.pos >= len(p.toks) {
		return ""
	}
	return p.toks[p.pos]
}

func (p *exprParser) next() string {
	t := p.peek()
	p.pos++
	return t
}

func (p *exprParser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek() == "||" {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = orExpr{left, right}
	}
	return left, nil
}

func (p *exprParser) parseAnd() (Expr, error) {
	left, err := p.parseCmp()
	if err != nil {
		return nil, err
	}
	for p.peek() == "&&" {
		p.next()
		right, err := p.parseCmp()
		if err != nil {
			return nil, err
		}
		left = andExpr{left, right}
	}
	return left, nil
}

func (p *exprParser) parseCmp() (Expr, error) {
	if p.peek() == "(" {
		p.next()
		e, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.next() != ")" {
			return nil, fmt.Errorf("expected ')' in expression %q", p.src)
		}
		return e, nil
	}
	key := p.next()
	if key == "" {
		return nil, fmt.Errorf("unexpected end of expression %q", p.src)
	}
	op := p.next()
	if op != "==" && op != "!=" {
		return nil, fmt.Errorf("expected '==' or '!=' after %q in expression %q", key, p.src)
	}
	value := p.next()
	if value == "" {
		return nil, fmt.Errorf("expected value after %q %q in expression %q", key, op, p.src)
	}
	return cmpExpr{key: key, value: value, neq: op == "!="}, nil
}
