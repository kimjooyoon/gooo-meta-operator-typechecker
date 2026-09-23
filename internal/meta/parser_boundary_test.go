package meta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSourcePreservesQuotedCommentMarkers(t *testing.T) {
	source, err := ParseSourceBytes([]byte("gooo meta_operator_typechecker v1\norigin id=literal identity=\"literal#and//markers\" # comment\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(source.Origins) != 1 || source.Origins[0].Identity != "literal#and//markers" {
		t.Fatalf("quoted identity was altered: %#v", source.Origins)
	}
}

func TestLoadContractRejectsUnknownAndTrailingJSON(t *testing.T) {
	cases := map[string]string{
		"unknown":  `{"unexpected":true}`,
		"trailing": `{} {"extra":true}`,
	}
	for name, payload := range cases {
		path := filepath.Join(t.TempDir(), name+".json")
		if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadContract(path); err == nil {
			t.Fatalf("%s contract was accepted", name)
		}
	}
}
