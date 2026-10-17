package dotenv

import (
	"bytes"
	"maps"
	"strings"
	"testing"
)

func TestParseRoundTrip(t *testing.T) {
	kv := map[string]string{
		"B_KEY":   "plain",
		"A_KEY":   "has space",
		"C_QUOTE": `has "quote"`,
		"D_EMPTY": "",
	}
	var buf bytes.Buffer
	if err := Write(&buf, kv); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Parse(&buf)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !maps.Equal(got, kv) {
		t.Errorf("Parse(Write(kv)) = %v, want %v", got, kv)
	}
}

func TestParseIgnoresBlankLinesAndComments(t *testing.T) {
	input := "# a comment\n\nKEY=value\n  \nOTHER=\"quoted\"\n"
	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := map[string]string{"KEY": "value", "OTHER": "quoted"}
	if !maps.Equal(got, want) {
		t.Errorf("Parse(%q) = %v, want %v", input, got, want)
	}
}

func TestParseMalformedLine(t *testing.T) {
	if _, err := Parse(strings.NewReader("NOT_A_KV_PAIR\n")); err == nil {
		t.Error("expected error for line missing '=', got nil")
	}
}

func TestParseUnterminatedQuote(t *testing.T) {
	if _, err := Parse(strings.NewReader(`KEY="unterminated` + "\n")); err == nil {
		t.Error("expected error for unterminated quoted value, got nil")
	}
}
