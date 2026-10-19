package mixin

import "testing"

func TestEncodeDecodeSelectionComment(t *testing.T) {
	dev := Mixin{ID: "dev", Category: "environment"}
	acme := Mixin{ID: "acme", Category: "tenant"}
	sel := Selection{"environment": dev, "tenant": acme}

	line, err := EncodeSelectionComment(sel)
	if err != nil {
		t.Fatalf("EncodeSelectionComment: %v", err)
	}

	ids, err := DecodeSelectionComment([]string{"# unrelated comment", line, "FOO=bar"})
	if err != nil {
		t.Fatalf("DecodeSelectionComment: %v", err)
	}
	if ids["environment"] != "dev" || ids["tenant"] != "acme" || len(ids) != 2 {
		t.Errorf("DecodeSelectionComment = %v, want environment=dev, tenant=acme", ids)
	}
}

func TestDecodeSelectionCommentMissing(t *testing.T) {
	if _, err := DecodeSelectionComment([]string{"FOO=bar", "# just a comment"}); err == nil {
		t.Error("expected error when no marker line is present, got nil")
	}
}
