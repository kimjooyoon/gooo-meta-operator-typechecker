package meta

import (
	"fmt"
	"strings"
)

func BuildIR(source SourceDecl, contract Contract) (IR, error) {
	if err := ValidateSource(source, contract); err != nil {
		return IR{}, err
	}
	contractDigest, err := ContractDigest(contract)
	if err != nil {
		return IR{}, err
	}
	ir := IR{
		Schema: SourceToIRSchema(source.Schema), Version: "v1", SourceDigest: source.SourceDigest,
		ContractDigest: contractDigest, Denominator: source.Denominator, Authority: source.Authority,
		Precedence: append([]string(nil), source.Precedence...), UnknownFields: append([]string(nil), source.UnknownFields...),
		Kinds: append([]KindDecl(nil), source.Kinds...), Types: append([]TypeDecl(nil), source.Types...), Effects: append([]EffectDecl(nil), source.Effects...),
		Stages: append([]StageDecl(nil), source.Stages...), Origins: append([]OriginDecl(nil), source.Origins...),
		Capabilities: append([]CapabilityDecl(nil), source.Capabilities...), Operations: append([]OperationDecl(nil), source.Operations...),
		Rules: cloneRules(source.Rules), Fixtures: append([]FixtureDecl(nil), source.Fixtures...), Cases: append([]CaseDecl(nil), source.Cases...),
		Proofs: make([]Proof, 0, len(source.Cases)),
	}
	for _, caseDecl := range source.Cases {
		result, err := TypecheckCase(source, caseDecl)
		if err != nil {
			return IR{}, fmt.Errorf("case %s: %w", caseDecl.ID, err)
		}
		result.Proof.ContractDigest = contractDigest
		ir.Proofs = append(ir.Proofs, result.Proof)
	}
	ir.IRDigest, err = unsignedIRDigest(ir)
	if err != nil {
		return IR{}, err
	}
	for index := range ir.Proofs {
		ir.Proofs[index].IRDigest = ir.IRDigest
	}
	return ir, nil
}

func SourceToIRSchema(sourceSchema string) string {
	if sourceSchema == SourceSchema {
		return IRScheme()
	}
	return strings.Replace(sourceSchema, "/source/", "/ir/", 1)
}

func IRScheme() string { return IRSchema }

func ValidateIR(ir IR) error {
	if ir.Schema != IRSchema || ir.Version != "v1" || ir.SourceDigest == "" || ir.ContractDigest == "" || ir.IRDigest == "" {
		return fmt.Errorf("invalid typed meta IR identity")
	}
	if ir.Denominator.CaseCount != FixedCaseCount || !ir.Denominator.Fixed || len(ir.Cases) != FixedCaseCount || len(ir.Proofs) != FixedCaseCount {
		return fmt.Errorf("typed meta IR is not fixed at seven cases")
	}
	expected, err := unsignedIRDigest(ir)
	if err != nil {
		return err
	}
	if expected != ir.IRDigest {
		return fmt.Errorf("typed meta IR digest mismatch")
	}
	for index, proof := range ir.Proofs {
		if proof.IRDigest != ir.IRDigest || proof.SourceDigest != ir.SourceDigest || proof.ContractDigest != ir.ContractDigest || proof.CaseID != ir.Cases[index].ID {
			return fmt.Errorf("proof %d is not bound to typed meta IR", index+1)
		}
	}
	return nil
}

func cloneRules(rules []RuleDecl) []RuleDecl {
	result := make([]RuleDecl, len(rules))
	for index, rule := range rules {
		result[index] = rule
		result[index].Properties = make(map[string]string, len(rule.Properties))
		for key, value := range rule.Properties {
			result[index].Properties[key] = value
		}
	}
	return result
}

type typecheckEnvironment struct {
	source       SourceDecl
	operations   map[string]OperationDecl
	stages       map[string]int
	origins      map[string]OriginDecl
	capabilities map[string]bool
	effects      map[string]bool
	rules        map[string]RuleDecl
}

func newEnvironment(source SourceDecl) typecheckEnvironment {
	env := typecheckEnvironment{
		source: source, operations: map[string]OperationDecl{}, stages: map[string]int{}, origins: map[string]OriginDecl{},
		capabilities: map[string]bool{}, effects: map[string]bool{}, rules: map[string]RuleDecl{},
	}
	for _, operation := range source.Operations {
		env.operations[operation.ID] = operation
	}
	for _, stage := range source.Stages {
		env.stages[stage.ID] = stage.Index
	}
	for _, origin := range source.Origins {
		env.origins[origin.ID] = origin
	}
	for _, capability := range source.Capabilities {
		env.capabilities[capability.ID] = true
	}
	for _, effect := range source.Effects {
		env.effects[effect.ID] = true
	}
	for _, rule := range source.Rules {
		env.rules[rule.ID] = rule
	}
	return env
}

func ValidateSource(source SourceDecl, contract Contract) error {
	if source.Schema != SourceSchema || source.Version != "v1" {
		return fmt.Errorf("source declaration header is invalid")
	}
	if source.Denominator.ID != contract.ID || source.Denominator.CaseCount != FixedCaseCount || contract.CaseCount != FixedCaseCount || !source.Denominator.Fixed || !contract.Fixed {
		return fmt.Errorf("fixed denominator declaration mismatch")
	}
	if contract.Schema != ContractSchema || contract.Version != "v1" || contract.ContractRole != "transport_and_ci_identity_only" || contract.SemanticAuthority != "examples/meta-operator-typechecker-v1/main.gooo" {
		return fmt.Errorf("contract is not a transport-only identity contract")
	}
	if len(source.Precedence) != 3 || strings.Join(source.Precedence, ">") != "REFUTED>UNKNOWN>CLOSED" {
		return fmt.Errorf("resolution precedence mismatch")
	}
	if len(source.UnknownFields) != UnknownFieldCount || strings.Join(source.UnknownFields, ",") != "stage,step,reason,unknown_class,next_operation,blocked_by" {
		return fmt.Errorf("UNKNOWN six-field contract mismatch")
	}
	if source.Authority != (Authority{}) {
		return fmt.Errorf("source authority must be zero")
	}
	if len(source.Cases) != FixedCaseCount || len(contract.CaseIDs) != FixedCaseCount {
		return fmt.Errorf("source and contract must contain exactly seven cases")
	}
	seen := map[string]bool{}
	for index, caseDecl := range source.Cases {
		if caseDecl.Ordinal != index+1 || caseDecl.ID == "" || seen[caseDecl.ID] || caseDecl.ID != contract.CaseIDs[index] {
			return fmt.Errorf("case %d does not match fixed denominator", index+1)
		}
		seen[caseDecl.ID] = true
		if caseDecl.Expected != "CLOSED" && caseDecl.Expected != "REFUTED" && caseDecl.Expected != "UNKNOWN" {
			return fmt.Errorf("case %s has invalid expected decision", caseDecl.ID)
		}
	}
	if err := validateUniqueIDs(source); err != nil {
		return err
	}
	if err := validateRequiredDeclarations(source); err != nil {
		return err
	}
	return nil
}

func validateUniqueIDs(source SourceDecl) error {
	groups := []struct {
		name   string
		values []string
	}{
		{"kind", idsKinds(source.Kinds)}, {"type", idsTypes(source.Types)}, {"effect", idsEffects(source.Effects)},
		{"stage", idsStages(source.Stages)}, {"origin", idsOrigins(source.Origins)}, {"capability", idsCapabilities(source.Capabilities)},
		{"operation", idsOperations(source.Operations)}, {"rule", idsRules(source.Rules)}, {"fixture", idsFixtures(source.Fixtures)},
	}
	for _, group := range groups {
		seen := map[string]bool{}
		for _, id := range group.values {
			if id == "" || seen[id] {
				return fmt.Errorf("duplicate or empty %s identity", group.name)
			}
			seen[id] = true
		}
	}
	return nil
}

func validateRequiredDeclarations(source SourceDecl) error {
	if !hasID(source.Kinds, "value") || !hasID(source.Kinds, "meta") || !hasID(source.Kinds, "proof") {
		return fmt.Errorf("required kinds are missing")
	}
	for _, required := range []string{"Fixture", "Artifact", "Unit", "MetaOp"} {
		found := false
		for _, declared := range source.Types {
			if declared.ID == required {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("required type %q is missing", required)
		}
	}
	for _, required := range []string{"fixture.read", "artifact.normalize", "replay.verify"} {
		if !hasEffect(source.Effects, required) {
			return fmt.Errorf("required effect %q is missing", required)
		}
	}
	if len(source.Stages) != 3 || source.Stages[0].ID != "quote" || source.Stages[0].Index != 0 || source.Stages[1].ID != "compile" || source.Stages[1].Index != 1 || source.Stages[2].ID != "run" || source.Stages[2].Index != 2 {
		return fmt.Errorf("stage indices are not the fixed quote/compile/run order")
	}
	for _, required := range []string{"literal", "fixture", "generated", "caller"} {
		if !hasOrigin(source.Origins, required) {
			return fmt.Errorf("required origin %q is missing", required)
		}
	}
	for _, required := range []string{"compiler.fixture", "replay.fixture"} {
		if !hasCapability(source.Capabilities, required) {
			return fmt.Errorf("required capability %q is missing", required)
		}
	}
	for _, required := range []string{"compose_effects", "stage_escape", "origin_hygiene", "capability_binding", "replay"} {
		if !hasRule(source.Rules, required) {
			return fmt.Errorf("required composition rule %q is missing", required)
		}
	}
	if len(source.Fixtures) != 1 || source.Fixtures[0].ID != "compiler_fixture" {
		return fmt.Errorf("compiler fixture declaration is missing")
	}
	typeNames := map[string]bool{}
	for _, declared := range source.Types {
		typeNames[declared.ID] = true
	}
	stageNames := map[string]bool{}
	for _, declared := range source.Stages {
		stageNames[declared.ID] = true
	}
	originNames := map[string]bool{}
	for _, declared := range source.Origins {
		originNames[declared.ID] = true
	}
	effectNames := map[string]bool{}
	for _, declared := range source.Effects {
		effectNames[declared.ID] = true
	}
	capabilityNames := map[string]bool{}
	for _, declared := range source.Capabilities {
		capabilityNames[declared.ID] = true
	}
	kindNames := map[string]bool{}
	for _, declared := range source.Kinds {
		kindNames[declared.ID] = true
	}
	for _, operation := range source.Operations {
		if !typeNames[operation.Kind] || !typeNames[operation.Input] || !typeNames[operation.Output] || !stageNames[operation.Stage] || !originNames[operation.Origin] || operation.Pre == "" || operation.Post == "" {
			return fmt.Errorf("operation %q has an unbound kind, type, stage, origin, or condition", operation.ID)
		}
		if !kindNames["meta"] && operation.Kind == "MetaOp" {
			return fmt.Errorf("meta kind is missing")
		}
		for _, required := range operation.Requires {
			if required != "$cap" && !capabilityNames[required] {
				return fmt.Errorf("operation %q requires undeclared capability %q", operation.ID, required)
			}
		}
		for _, effect := range operation.Effects {
			if !effectNames[effect] {
				return fmt.Errorf("operation %q uses undeclared effect %q", operation.ID, effect)
			}
		}
	}
	return nil
}

func hasID(values []KindDecl, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func hasEffect(values []EffectDecl, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func hasOrigin(values []OriginDecl, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func hasCapability(values []CapabilityDecl, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func hasRule(values []RuleDecl, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func idsKinds(values []KindDecl) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v.ID
	}
	return result
}
func idsTypes(values []TypeDecl) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v.ID
	}
	return result
}
func idsEffects(values []EffectDecl) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v.ID
	}
	return result
}
func idsStages(values []StageDecl) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v.ID
	}
	return result
}
func idsOrigins(values []OriginDecl) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v.ID
	}
	return result
}
func idsCapabilities(values []CapabilityDecl) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v.ID
	}
	return result
}
func idsOperations(values []OperationDecl) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v.ID
	}
	return result
}
func idsRules(values []RuleDecl) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v.ID
	}
	return result
}
func idsFixtures(values []FixtureDecl) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v.ID
	}
	return result
}
