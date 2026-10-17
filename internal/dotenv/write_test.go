package dotenv

import (
	"bytes"
	"testing"
)

func TestWrite(t *testing.T) {
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

	want := "A_KEY=\"has space\"\n" +
		"B_KEY=plain\n" +
		"C_QUOTE=\"has \\\"quote\\\"\"\n" +
		"D_EMPTY=\"\"\n"
	if buf.String() != want {
		t.Errorf("Write output = %q, want %q", buf.String(), want)
	}
}
