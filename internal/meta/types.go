package meta

const (
	SourceSchema      = "gooo/meta-operator-typechecker/source/v1"
	IRSchema          = "gooo/meta-operator-typechecker/ir/v1"
	GeneratedSchema   = "gooo/meta-operator-typechecker/generated/v1"
	ReportSchema      = "gooo/meta-operator-typechecker/conformance-report/v1"
	ReceiptSchema     = "gooo/meta-operator-typechecker/execution-receipt/v1"
	ContractSchema    = "gooo/meta-operator-typechecker/denominator/v1"
	FixedCaseCount    = 7
	UnknownFieldCount = 6
)

type Authority struct {
	RepositoryWrites          int `json:"repository_writes"`
	LocalTestExecutions       int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
	AutomaticCommit           int `json:"automatic_commit"`
	AutomaticPush             int `json:"automatic_push"`
	AutomaticMerge            int `json:"automatic_merge"`
	AutomaticRelease          int `json:"automatic_release"`
}

type DenominatorDecl struct {
	ID        string `json:"id"`
	CaseCount int    `json:"case_count"`
	Fixed     bool   `json:"fixed"`
}

type KindDecl struct {
	ID string `json:"id"`
}

type TypeDecl struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type EffectDecl struct {
	ID string `json:"id"`
}

type StageDecl struct {
	ID    string `json:"id"`
	Index int    `json:"index"`
}

type OriginDecl struct {
	ID       string `json:"id"`
	Identity string `json:"identity"`
}

type CapabilityDecl struct {
	ID string `json:"id"`
}

type OperationDecl struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Input    string   `json:"input"`
	Output   string   `json:"output"`
	Stage    string   `json:"stage"`
	Origin   string   `json:"origin"`
	Requires []string `json:"requires"`
	Effects  []string `json:"effects"`
	Pre      string   `json:"pre"`
	Post     string   `json:"post"`
}

type RuleDecl struct {
	ID         string            `json:"id"`
	Operator   string            `json:"operator"`
	Properties map[string]string `json:"properties"`
}

type FixtureDecl struct {
	ID             string `json:"id"`
	Path           string `json:"path"`
	ExpectedOutput string `json:"expected_output"`
}

type CaseDecl struct {
	Ordinal         int      `json:"ordinal"`
	ID              string   `json:"id"`
	Expression      string   `json:"expression"`
	Fixture         string   `json:"fixture"`
	TargetStage     string   `json:"target_stage"`
	BoundCaps       []string `json:"bound_capabilities"`
	Expected        string   `json:"expected"`
	ExpectedReason  string   `json:"expected_reason"`
	ExpectedEffects []string `json:"expected_effects"`
	ExpectedPost    string   `json:"expected_post"`
	ExpectedTrace   []string `json:"expected_trace"`
	UnknownClass    string   `json:"unknown_class"`
	NextOperation   string   `json:"next_operation"`
	BlockedBy       []string `json:"blocked_by"`
}

type SourceDecl struct {
	Schema        string           `json:"schema"`
	Version       string           `json:"version"`
	Denominator   DenominatorDecl  `json:"denominator"`
	Authority     Authority        `json:"authority"`
	Precedence    []string         `json:"precedence"`
	UnknownFields []string         `json:"unknown_fields"`
	Kinds         []KindDecl       `json:"kinds"`
	Types         []TypeDecl       `json:"types"`
	Effects       []EffectDecl     `json:"effects"`
	Stages        []StageDecl      `json:"stages"`
	Origins       []OriginDecl     `json:"origins"`
	Capabilities  []CapabilityDecl `json:"capabilities"`
	Operations    []OperationDecl  `json:"operations"`
	Rules         []RuleDecl       `json:"rules"`
	Fixtures      []FixtureDecl    `json:"fixtures"`
	Cases         []CaseDecl       `json:"cases"`
	SourceDigest  string           `json:"source_digest"`
}

type Contract struct {
	Schema            string   `json:"schema"`
	ID                string   `json:"id"`
	Version           string   `json:"version"`
	CaseCount         int      `json:"case_count"`
	Fixed             bool     `json:"fixed"`
	CaseIDs           []string `json:"case_ids"`
	ContractRole      string   `json:"contract_role"`
	SemanticAuthority string   `json:"semantic_authority"`
}

type Expr struct {
	Name string `json:"name"`
	Args []Expr `json:"args,omitempty"`
}

type PlanStep struct {
	Operation  string `json:"operation"`
	Capability string `json:"capability,omitempty"`
}

type TypedExpression struct {
	Expression           Expr       `json:"expression"`
	InputType            string     `json:"input_type"`
	OutputType           string     `json:"output_type"`
	Stage                int        `json:"stage"`
	Origin               string     `json:"origin"`
	RequiredCapabilities []string   `json:"required_capabilities"`
	Effects              []string   `json:"effects"`
	Preconditions        []string   `json:"preconditions"`
	Postconditions       []string   `json:"postconditions"`
	Plan                 []PlanStep `json:"plan"`
	Replay               bool       `json:"replay"`
}

type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

func (u Unknown) Complete() bool {
	return u.Stage != "" && u.Step != "" && u.Reason != "" && u.UnknownClass != "" && u.NextOperation != "" && len(u.BlockedBy) > 0
}

type Proof struct {
	Schema              string          `json:"schema"`
	CaseID              string          `json:"case_id"`
	Decision            string          `json:"decision"`
	TerminalReason      string          `json:"terminal_reason"`
	SourceDigest        string          `json:"source_digest"`
	ContractDigest      string          `json:"contract_digest"`
	IRDigest            string          `json:"ir_digest"`
	TargetStage         int             `json:"target_stage"`
	Typed               TypedExpression `json:"typed_expression"`
	ExpectedEffects     []string        `json:"expected_effects"`
	ExpectedPost        string          `json:"expected_post"`
	ExpectedEffectTrace []string        `json:"expected_effect_trace"`
	Unknown             Unknown         `json:"unknown,omitempty"`
	Replay              bool            `json:"replay"`
}

type IR struct {
	Schema         string           `json:"schema"`
	Version        string           `json:"version"`
	SourceDigest   string           `json:"source_digest"`
	ContractDigest string           `json:"contract_digest"`
	Denominator    DenominatorDecl  `json:"denominator"`
	Authority      Authority        `json:"authority"`
	Precedence     []string         `json:"precedence"`
	UnknownFields  []string         `json:"unknown_fields"`
	Kinds          []KindDecl       `json:"kinds"`
	Types          []TypeDecl       `json:"types"`
	Effects        []EffectDecl     `json:"effects"`
	Stages         []StageDecl      `json:"stages"`
	Origins        []OriginDecl     `json:"origins"`
	Capabilities   []CapabilityDecl `json:"capabilities"`
	Operations     []OperationDecl  `json:"operations"`
	Rules          []RuleDecl       `json:"rules"`
	Fixtures       []FixtureDecl    `json:"fixtures"`
	Cases          []CaseDecl       `json:"cases"`
	Proofs         []Proof          `json:"proofs"`
	IRDigest       string           `json:"ir_digest"`
}

type TypecheckResult struct {
	CaseID   string          `json:"case_id"`
	Decision string          `json:"decision"`
	Reason   string          `json:"reason"`
	Typed    TypedExpression `json:"typed_expression"`
	Proof    Proof           `json:"proof"`
	Unknown  Unknown         `json:"unknown,omitempty"`
}

type FixtureInput struct {
	FixtureID string `json:"fixture_id"`
	Path      string `json:"path"`
	Raw       string `json:"raw"`
}

type ExecutionResult struct {
	Schema         string   `json:"schema"`
	CaseID         string   `json:"case_id"`
	Decision       string   `json:"decision"`
	Reason         string   `json:"reason"`
	Output         string   `json:"output"`
	ArtifactDigest string   `json:"artifact_digest"`
	Effects        []string `json:"effects"`
	EffectTrace    []string `json:"effect_trace"`
}

type VerificationResult struct {
	CaseID   string `json:"case_id"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
	Verified bool   `json:"verified"`
}

type CaseVector struct {
	Ordinal  int      `json:"ordinal"`
	CaseID   string   `json:"case_id"`
	Expected string   `json:"expected"`
	Observed string   `json:"observed"`
	Reason   string   `json:"reason"`
	Effects  []string `json:"effects"`
	Trace    []string `json:"effect_trace"`
	Unknown  Unknown  `json:"unknown"`
}

type StageMetric struct {
	WallMS     int64 `json:"wall_ms"`
	PeakRSSKiB int64 `json:"peak_rss_kib"`
}

type Inventory struct {
	RootReadmeExcluded int   `json:"root_readme_excluded"`
	GoFiles            int   `json:"go_files"`
	GoPhysicalLines    int   `json:"go_physical_lines"`
	GoooFiles          int   `json:"gooo_files"`
	GoooPhysicalLines  int   `json:"gooo_physical_lines"`
	DescendantDirs     int   `json:"descendant_dirs"`
	RegularFiles       int   `json:"regular_files"`
	GeneratedFiles     int   `json:"generated_files"`
	GeneratedBytes     int64 `json:"generated_bytes"`
}

type TestMetrics struct {
	Total    int `json:"total"`
	Selected int `json:"selected"`
	Executed int `json:"executed"`
	Reused   int `json:"reused"`
	Failed   int `json:"failed"`
	Unknown  int `json:"unknown"`
}

type ConformanceReport struct {
	Schema                     string                 `json:"schema"`
	Decision                   string                 `json:"decision"`
	Reason                     string                 `json:"reason"`
	Precedence                 []string               `json:"precedence"`
	SourceDigest               string                 `json:"source_digest"`
	ContractDigest             string                 `json:"contract_digest"`
	IRDigest                   string                 `json:"ir_digest"`
	Inventory                  Inventory              `json:"inventory"`
	Stages                     map[string]StageMetric `json:"stages"`
	Tests                      TestMetrics            `json:"tests"`
	Cases                      []CaseVector           `json:"cases"`
	GeneratedFiles             []string               `json:"generated_files"`
	GeneratedBytes             int64                  `json:"generated_bytes"`
	Authority                  Authority              `json:"authority"`
	LocalIntegrationExecutions int                    `json:"local_integration_executions"`
}

type ExecutionReceipt struct {
	Schema           string             `json:"schema"`
	CaseID           string             `json:"case_id"`
	Typecheck        TypecheckResult    `json:"typecheck"`
	Execution        ExecutionResult    `json:"execution"`
	Verification     VerificationResult `json:"verification"`
	RepositoryWrites int                `json:"repository_writes"`
}
