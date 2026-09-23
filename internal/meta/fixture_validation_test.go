package meta

import (
	"strings"
	"testing"
)

func TestTypecheckCaseRejectsUndeclaredFixture(t *testing.T) {
	source, err := ParseSource("../../examples/meta-operator-typechecker-v1/main.gooo")
	if err != nil {
		t.Fatal(err)
	}
	caseDecl := source.Cases[0]
	caseDecl.Fixture = "missing-fixture"

	if _, err := TypecheckCase(source, caseDecl); err == nil || !strings.Contains(err.Error(), "undeclared fixture") {
		t.Fatalf("TypecheckCase error = %v, want undeclared fixture", err)
	}
}
