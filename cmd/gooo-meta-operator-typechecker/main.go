package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kimjooyoon/gooo-meta-operator-typechecker/internal/generated"
	"github.com/kimjooyoon/gooo-meta-operator-typechecker/internal/meta"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "compile":
		err = compileCommand(os.Args[2:])
	case "typecheck":
		err = typecheckCommand(os.Args[2:])
	case "execute":
		err = executeCommand(os.Args[2:])
	case "conformance":
		err = conformanceCommand(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func compileCommand(arguments []string) error {
	flags := flag.NewFlagSet("compile", flag.ContinueOnError)
	sourcePath := flags.String("source", "", "path to authoritative .gooo source")
	contractPath := flags.String("contract", "", "path to transport contract")
	outputIR := flags.String("output-ir", "", "absolute output path for typed IR")
	outputGo := flags.String("output-go", "", "absolute output path for generated executor")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *sourcePath == "" || *contractPath == "" || *outputIR == "" || *outputGo == "" {
		return fmt.Errorf("compile requires -source, -contract, -output-ir, and -output-go")
	}
	_, err := meta.WriteCompileOutputs(*sourcePath, *contractPath, *outputIR, *outputGo)
	return err
}

func typecheckCommand(arguments []string) error {
	flags := flag.NewFlagSet("typecheck", flag.ContinueOnError)
	sourcePath := flags.String("source", "", "path to authoritative .gooo source")
	contractPath := flags.String("contract", "", "path to transport contract")
	caseID := flags.String("case", "", "optional fixed case id")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	source, err := meta.ParseSource(*sourcePath)
	if err != nil {
		return err
	}
	contract, err := meta.LoadContract(*contractPath)
	if err != nil {
		return err
	}
	ir, err := meta.BuildIR(source, contract)
	if err != nil {
		return err
	}
	if *caseID != "" {
		for _, proof := range ir.Proofs {
			if proof.CaseID == *caseID {
				return printJSON(proof)
			}
		}
		return fmt.Errorf("unknown case %q", *caseID)
	}
	return printJSON(ir.Proofs)
}

func executeCommand(arguments []string) error {
	flags := flag.NewFlagSet("execute", flag.ContinueOnError)
	root := flags.String("root", ".", "repository root")
	caseID := flags.String("case", "", "fixed case id")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *caseID == "" {
		return fmt.Errorf("execute requires -case")
	}
	absoluteRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	source, err := meta.ParseSource(filepath.Join(absoluteRoot, "examples/meta-operator-typechecker-v1/main.gooo"))
	if err != nil {
		return err
	}
	contract, err := meta.LoadContract(filepath.Join(absoluteRoot, "contracts/denominator-v1.json"))
	if err != nil {
		return err
	}
	ir, err := meta.LoadIR(filepath.Join(absoluteRoot, "internal/generated/semantic-ir.json"))
	if err != nil {
		return err
	}
	if err := meta.VerifyIRAgainstSource(ir, source, contract); err != nil {
		return err
	}
	for index, caseDecl := range source.Cases {
		if caseDecl.ID != *caseID {
			continue
		}
		fixture := source.Fixtures[0]
		raw, readErr := os.ReadFile(filepath.Join(absoluteRoot, fixture.Path))
		if readErr != nil {
			return readErr
		}
		result, executeErr := generated.Execute(caseDecl.ID, meta.FixtureInput{FixtureID: fixture.ID, Path: fixture.Path, Raw: string(raw)})
		if executeErr != nil {
			return executeErr
		}
		generatedProof, proofOK := generated.ProofFor(caseDecl.ID)
		if !proofOK {
			return fmt.Errorf("generated proof is missing for %s", caseDecl.ID)
		}
		if err := meta.VerifyProofBinding(generatedProof, ir); err != nil {
			return err
		}
		verification := meta.VerifyExecution(ir.Proofs[index], result, fixture)
		return printJSON(struct {
			Execution    meta.ExecutionResult    `json:"execution"`
			Verification meta.VerificationResult `json:"verification"`
		}{result, verification})
	}
	return fmt.Errorf("unknown case %q", *caseID)
}

func conformanceCommand(arguments []string) error {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	root := flags.String("root", ".", "repository root")
	fixtures := flags.String("fixtures", "fixtures/compiler", "fixture directory")
	outputDir := flags.String("output-dir", "", "absolute caller-owned output directory")
	testsTotal := flags.Int("tests-total", 7, "CI-reported test total")
	testsSelected := flags.Int("tests-selected", 7, "CI-reported selected tests")
	testsExecuted := flags.Int("tests-executed", 7, "CI-reported executed tests")
	testsReused := flags.Int("tests-reused", 0, "CI-reported reused tests")
	testsFailed := flags.Int("tests-failed", 0, "CI-reported failed tests")
	testsUnknown := flags.Int("tests-unknown", 0, "CI-reported unknown tests")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	absoluteRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	if *outputDir == "" {
		return fmt.Errorf("conformance requires -output-dir")
	}
	if !filepath.IsAbs(*outputDir) {
		return fmt.Errorf("-output-dir must be absolute")
	}
	if _, err := os.Stat(filepath.Join(absoluteRoot, *fixtures)); err != nil {
		return fmt.Errorf("fixture directory: %w", err)
	}
	sourcePath := filepath.Join(absoluteRoot, "examples/meta-operator-typechecker-v1/main.gooo")
	contractPath := filepath.Join(absoluteRoot, "contracts/denominator-v1.json")
	irPath := filepath.Join(absoluteRoot, "internal/generated/semantic-ir.json")
	generatedPath := filepath.Join(absoluteRoot, "internal/generated/meta.gooo.go")
	var source meta.SourceDecl
	var contract meta.Contract
	var ir meta.IR
	var generatedRaw []byte
	var errStage error
	stages := map[string]meta.StageMetric{}
	stages["parse"] = measure(func() error {
		source, errStage = meta.ParseSource(sourcePath)
		if errStage != nil {
			return errStage
		}
		contract, errStage = meta.LoadContract(contractPath)
		return errStage
	})
	if errStage != nil {
		return errStage
	}
	stages["lower"] = measure(func() error { ir, errStage = meta.BuildIR(source, contract); return errStage })
	if errStage != nil {
		return errStage
	}
	stages["typecheck"] = measure(func() error {
		for _, caseDecl := range source.Cases {
			if _, errStage = meta.TypecheckCase(source, caseDecl); errStage != nil {
				return errStage
			}
		}
		return nil
	})
	if errStage != nil {
		return errStage
	}
	stages["generate"] = measure(func() error {
		var generated []byte
		generated, errStage = meta.GenerateGoSource(ir)
		if errStage != nil {
			return errStage
		}
		generatedRaw, errStage = os.ReadFile(generatedPath)
		if errStage != nil {
			return errStage
		}
		if string(generated) != string(generatedRaw) {
			return fmt.Errorf("checked-in generated executor differs from source lowering")
		}
		return nil
	})
	if errStage != nil {
		return errStage
	}
	var loadedIR meta.IR
	if loadedIR, err = meta.LoadIR(irPath); err != nil {
		return err
	}
	ir = loadedIR
	stages["verify"] = measure(func() error { return meta.VerifyIRAgainstSource(ir, source, contract) })
	if errStage != nil {
		return errStage
	}
	tests := meta.TestMetrics{Total: *testsTotal, Selected: *testsSelected, Executed: *testsExecuted, Reused: *testsReused, Failed: *testsFailed, Unknown: *testsUnknown}
	_ = generatedRaw
	_, err = meta.RunConformance(source, contract, ir, meta.ConformanceOptions{Root: absoluteRoot, OutputDir: *outputDir, StageMetrics: stages, Tests: tests, Executor: generated.Execute, ProofProvider: generated.ProofFor})
	return err
}

func measure(operation func() error) meta.StageMetric {
	started := time.Now()
	before := rssKiB()
	err := operation()
	after := rssKiB()
	peak := before
	if after > peak {
		peak = after
	}
	if err != nil {
		return meta.StageMetric{WallMS: time.Since(started).Milliseconds(), PeakRSSKiB: peak}
	}
	return meta.StageMetric{WallMS: time.Since(started).Milliseconds(), PeakRSSKiB: peak}
}

func rssKiB() int64 {
	if raw, err := os.ReadFile("/proc/self/statm"); err == nil {
		var size, resident int64
		if _, scanErr := fmt.Sscanf(string(raw), "%d %d", &size, &resident); scanErr == nil {
			return resident * int64(os.Getpagesize()) / 1024
		}
	}
	return 0
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gooo-meta-operator-typechecker <compile|typecheck|execute|conformance> ...")
}
