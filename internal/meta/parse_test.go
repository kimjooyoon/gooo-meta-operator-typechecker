package meta

import "testing"

func TestParseSourceRejectsDuplicateDeclarationKeys(t *testing.T) {
	_, err := ParseSourceBytes([]byte("gooo meta_operator_typechecker v1\ntype id=first kind=primitive id=second\n"))
	if err == nil {
		t.Fatal("expected duplicate declaration key to be rejected")
	}
}
