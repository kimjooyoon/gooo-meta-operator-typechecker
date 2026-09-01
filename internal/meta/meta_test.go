package meta

import (
	"path/filepath"
	"testing"
)

func TestFixedCasesTypecheck(t *testing.T) {
	root := filepath.Join("..", "..")
	source, err := ParseSource(filepath.Join(root, "examples/meta-operator-typechecker-v1/main.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	contract, err := LoadContract(filepath.Join(root, "contracts/denominator-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	ir, err := BuildIR(source, contract)
	if err != nil {
		t.Fatal(err)
	}
	if len(ir.Proofs) != FixedCaseCount {
		t.Fatalf("proof count = %d", len(ir.Proofs))
	}
	for _, caseDecl := range source.Cases {
		caseDecl := caseDecl
		t.Run(caseDecl.ID, func(t *testing.T) {
			result, err := TypecheckCase(source, caseDecl)
			if err != nil {
				t.Fatal(err)
			}
			if result.Decision != caseDecl.Expected {
				t.Fatalf("decision = %s, want %s", result.Decision, caseDecl.Expected)
			}
		})
	}
}

func TestFixtureCompiler(t *testing.T) {
	output, err := CompileFixture("module fixture\nterm identity\n")
	if err != nil {
		t.Fatal(err)
	}
	if output != "module:fixture;term:identity" {
		t.Fatalf("output = %q", output)
	}
}
