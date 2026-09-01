package meta

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ConformanceOptions struct {
	Root          string
	OutputDir     string
	StageMetrics  map[string]StageMetric
	Tests         TestMetrics
	Executor      func(string, FixtureInput) (ExecutionResult, error)
	ProofProvider func(string) (Proof, bool)
}

func RunConformance(source SourceDecl, contract Contract, ir IR, options ConformanceOptions) (ConformanceReport, error) {
	if err := ValidateSource(source, contract); err != nil {
		return ConformanceReport{}, err
	}
	if err := VerifyIRAgainstSource(ir, source, contract); err != nil {
		return ConformanceReport{}, err
	}
	if options.Executor == nil {
		return ConformanceReport{}, errors.New("generated executor is required")
	}
	if options.ProofProvider == nil {
		return ConformanceReport{}, errors.New("generated proof provider is required")
	}
	if err := ensureCallerOutput(options.Root, options.OutputDir); err != nil {
		return ConformanceReport{}, err
	}
	inventory, generatedFiles, err := CollectInventory(options.Root)
	if err != nil {
		return ConformanceReport{}, err
	}
	report := ConformanceReport{
		Schema: ReportSchema, Decision: "CONFORMANCE_CLOSED", Reason: "ALL_FIXED_CASES_VERIFIED", Precedence: append([]string(nil), source.Precedence...),
		SourceDigest: source.SourceDigest, ContractDigest: ir.ContractDigest, IRDigest: ir.IRDigest, Inventory: inventory,
		Stages: options.StageMetrics, Tests: options.Tests, Cases: make([]CaseVector, 0, len(source.Cases)), GeneratedFiles: generatedFiles,
		GeneratedBytes: inventory.GeneratedBytes, Authority: source.Authority, LocalIntegrationExecutions: 0,
	}
	if report.Stages == nil {
		report.Stages = map[string]StageMetric{}
	}
	executeStarted := time.Now()
	executeBefore := observedRSSKiB()
	for index, caseDecl := range source.Cases {
		proof := ir.Proofs[index]
		generatedProof, proofOK := options.ProofProvider(caseDecl.ID)
		if !proofOK {
			return ConformanceReport{}, fmt.Errorf("generated proof is missing for %s", caseDecl.ID)
		}
		if err := VerifyProofBinding(generatedProof, ir); err != nil {
			return ConformanceReport{}, err
		}
		if generatedProof.CaseID != proof.CaseID {
			return ConformanceReport{}, fmt.Errorf("generated proof order mismatch for %s", caseDecl.ID)
		}
		fixture := fixtureFor(source, caseDecl.Fixture)
		raw, readErr := os.ReadFile(filepath.Join(options.Root, fixture.Path))
		if readErr != nil {
			return ConformanceReport{}, readErr
		}
		input := FixtureInput{FixtureID: fixture.ID, Path: fixture.Path, Raw: string(raw)}
		result, executeErr := options.Executor(caseDecl.ID, input)
		if executeErr != nil {
			return ConformanceReport{}, fmt.Errorf("case %s execution: %w", caseDecl.ID, executeErr)
		}
		if err := VerifyProofBinding(proof, ir); err != nil {
			return ConformanceReport{}, err
		}
		verification := VerifyExecution(proof, result, fixture)
		if !verification.Verified {
			return ConformanceReport{}, fmt.Errorf("case %s verification failed: %s", caseDecl.ID, verification.Reason)
		}
		report.Cases = append(report.Cases, CaseVector{Ordinal: caseDecl.Ordinal, CaseID: caseDecl.ID, Expected: caseDecl.Expected, Observed: result.Decision, Reason: result.Reason, Effects: append([]string(nil), result.Effects...), Trace: append([]string(nil), result.EffectTrace...), Unknown: proof.Unknown})
	}
	executeAfter := observedRSSKiB()
	executePeak := executeBefore
	if executeAfter > executePeak {
		executePeak = executeAfter
	}
	report.Stages["execute"] = StageMetric{WallMS: time.Since(executeStarted).Milliseconds(), PeakRSSKiB: executePeak}
	if err := writeJSON(filepath.Join(options.OutputDir, "conformance-report.json"), report); err != nil {
		return ConformanceReport{}, err
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "inventory.json"), inventory); err != nil {
		return ConformanceReport{}, err
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "stage-metrics.json"), options.StageMetrics); err != nil {
		return ConformanceReport{}, err
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "test-metrics.json"), options.Tests); err != nil {
		return ConformanceReport{}, err
	}
	if err := os.WriteFile(filepath.Join(options.OutputDir, "ci-summary.md"), []byte(RenderSummary(report)), 0o644); err != nil {
		return ConformanceReport{}, err
	}
	return report, nil
}

func fixtureFor(source SourceDecl, id string) FixtureDecl {
	for _, fixture := range source.Fixtures {
		if fixture.ID == id {
			return fixture
		}
	}
	return FixtureDecl{ID: id}
}

func ensureCallerOutput(root, output string) error {
	if output == "" || !filepath.IsAbs(output) {
		return errors.New("output directory must be an absolute caller-owned path")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absoluteOutput, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	if isWithin(absoluteRoot, absoluteOutput) {
		return errors.New("output directory must be outside the repository")
	}
	if info, statErr := os.Stat(absoluteOutput); statErr == nil {
		if !info.IsDir() {
			return errors.New("output path must be a directory")
		}
		entries, readErr := os.ReadDir(absoluteOutput)
		if readErr != nil {
			return readErr
		}
		if len(entries) != 0 {
			return errors.New("output directory must be empty")
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	return os.MkdirAll(absoluteOutput, 0o755)
}

func isWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func observedRSSKiB() int64 {
	raw, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 2 {
		return 0
	}
	var resident int64
	if _, err := fmt.Sscan(fields[1], &resident); err != nil {
		return 0
	}
	return resident * int64(os.Getpagesize()) / 1024
}

func RenderSummary(report ConformanceReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Meta-operator typechecker conformance\n\nDecision: `%s`\n\nReason: `%s`\n\n", report.Decision, report.Reason)
	b.WriteString("Precedence: `REFUTED > UNKNOWN > CLOSED`\n\n")
	b.WriteString("## Exact inventory and CI observations\n\n")
	fmt.Fprintf(&b, "- root_readme_excluded: %d\n- go_files: %d\n- go_physical_lines: %d\n- gooo_files: %d\n- gooo_physical_lines: %d\n- descendant_dirs: %d\n- regular_files: %d\n- generated_files: %d\n- generated_bytes: %d\n", report.Inventory.RootReadmeExcluded, report.Inventory.GoFiles, report.Inventory.GoPhysicalLines, report.Inventory.GoooFiles, report.Inventory.GoooPhysicalLines, report.Inventory.DescendantDirs, report.Inventory.RegularFiles, report.Inventory.GeneratedFiles, report.Inventory.GeneratedBytes)
	b.WriteString("\n\n### Stage metrics\n\n| stage | wall_ms | peak_rss_kib |\n|---|---:|---:|\n")
	stageNames := []string{"parse", "lower", "typecheck", "generate", "execute", "verify"}
	for _, stage := range stageNames {
		metric := report.Stages[stage]
		fmt.Fprintf(&b, "| %s | %d | %d |\n", stage, metric.WallMS, metric.PeakRSSKiB)
	}
	fmt.Fprintf(&b, "\n### Test metrics\n\n- total: %d\n- selected: %d\n- executed: %d\n- reused: %d\n- failed: %d\n- unknown: %d\n\n", report.Tests.Total, report.Tests.Selected, report.Tests.Executed, report.Tests.Reused, report.Tests.Failed, report.Tests.Unknown)
	b.WriteString("### Fixed-case vector\n\n| ordinal | case | expected | observed | reason | effects | effect trace |\n|---:|---|---|---|---|---|---|\n")
	for _, vector := range report.Cases {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s | %s |\n", vector.Ordinal, vector.CaseID, vector.Expected, vector.Observed, vector.Reason, strings.Join(vector.Effects, ","), strings.Join(vector.Trace, ","))
	}
	b.WriteString("\n### UNKNOWN tuple\n\n")
	for _, vector := range report.Cases {
		if vector.Observed == "UNKNOWN" {
			fmt.Fprintf(&b, "- stage: `%s`; step: `%s`; reason: `%s`; unknown_class: `%s`; next_operation: `%s`; blocked_by: `%s`\n", vector.Unknown.Stage, vector.Unknown.Step, vector.Unknown.Reason, vector.Unknown.UnknownClass, vector.Unknown.NextOperation, strings.Join(vector.Unknown.BlockedBy, ","))
		}
	}
	b.WriteString("\nAuthority: repository_writes=0, local_test_executions=0, local_integration_executions=0, cross_project_required_gates=0, automatic_commit=0, automatic_push=0, automatic_merge=0, automatic_release=0.\n")
	return b.String()
}
