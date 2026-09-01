package meta

import (
	"fmt"
	"strings"
)

type typedWork struct {
	typed    TypedExpression
	decision string
	reason   string
	unknown  Unknown
}

func TypecheckCase(source SourceDecl, caseDecl CaseDecl) (TypecheckResult, error) {
	env := newEnvironment(source)
	expression, err := ParseExpression(caseDecl.Expression)
	if err != nil {
		return TypecheckResult{}, err
	}
	targetStage, ok := env.stages[caseDecl.TargetStage]
	if !ok {
		return TypecheckResult{}, fmt.Errorf("unknown target stage %q", caseDecl.TargetStage)
	}
	bound := make(map[string]bool, len(caseDecl.BoundCaps))
	for _, capability := range caseDecl.BoundCaps {
		if !env.capabilities[capability] {
			return TypecheckResult{}, fmt.Errorf("case binds undeclared capability %q", capability)
		}
		bound[capability] = true
	}
	work := checkExpression(env, expression, bound)
	if work.decision == "CLOSED" && work.typed.Stage > targetStage {
		work.decision = "REFUTED"
		work.reason = "STAGE_ESCAPE"
	}
	if work.decision != caseDecl.Expected {
		return TypecheckResult{}, fmt.Errorf("expected %s, typechecker produced %s", caseDecl.Expected, work.decision)
	}
	// The terminal reason is a source-declared semantic label. The typechecker
	// derives the decision and the proof facts; the executor and verifier must
	// still reproduce this label before a CLOSED case can be accepted.
	work.reason = caseDecl.ExpectedReason
	if work.decision == "CLOSED" {
		if !sameStrings(work.typed.Effects, caseDecl.ExpectedEffects) {
			return TypecheckResult{}, fmt.Errorf("expected effects %v, typechecker produced %v", caseDecl.ExpectedEffects, work.typed.Effects)
		}
		if work.typed.Postconditions != nil && len(work.typed.Postconditions) > 0 && work.typed.Postconditions[len(work.typed.Postconditions)-1] != caseDecl.ExpectedPost {
			return TypecheckResult{}, fmt.Errorf("expected postcondition %s, typechecker produced %s", caseDecl.ExpectedPost, work.typed.Postconditions[len(work.typed.Postconditions)-1])
		}
	}
	if work.decision == "UNKNOWN" {
		if !work.unknown.Complete() {
			return TypecheckResult{}, fmt.Errorf("UNKNOWN result is missing one of the six required fields")
		}
		if work.unknown.UnknownClass != caseDecl.UnknownClass || work.unknown.NextOperation != caseDecl.NextOperation || !sameStrings(work.unknown.BlockedBy, caseDecl.BlockedBy) {
			return TypecheckResult{}, fmt.Errorf("UNKNOWN tuple does not match source declaration")
		}
	}
	proof := Proof{
		Schema: GeneratedSchema, CaseID: caseDecl.ID, Decision: work.decision, TerminalReason: work.reason,
		SourceDigest: source.SourceDigest, Typed: work.typed, ExpectedEffects: append([]string(nil), caseDecl.ExpectedEffects...),
		ExpectedPost: caseDecl.ExpectedPost, ExpectedEffectTrace: append([]string(nil), caseDecl.ExpectedTrace...), Unknown: work.unknown,
		Replay: work.typed.Replay, TargetStage: targetStage,
	}
	return TypecheckResult{CaseID: caseDecl.ID, Decision: work.decision, Reason: work.reason, Typed: work.typed, Proof: proof, Unknown: work.unknown}, nil
}

func ParseExpression(raw string) (Expr, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Expr{}, fmt.Errorf("empty meta expression")
	}
	open := strings.IndexByte(raw, '(')
	if open < 0 {
		if strings.ContainsAny(raw, "){},") {
			return Expr{}, fmt.Errorf("invalid expression %q", raw)
		}
		return Expr{Name: raw}, nil
	}
	if !strings.HasSuffix(raw, ")") || open == 0 {
		return Expr{}, fmt.Errorf("invalid expression %q", raw)
	}
	name := strings.TrimSpace(raw[:open])
	if name == "" {
		return Expr{}, fmt.Errorf("expression function name is empty")
	}
	inside := raw[open+1 : len(raw)-1]
	parts, err := splitExpressionArgs(inside)
	if err != nil {
		return Expr{}, err
	}
	expression := Expr{Name: name, Args: make([]Expr, 0, len(parts))}
	for _, part := range parts {
		child, childErr := ParseExpression(part)
		if childErr != nil {
			return Expr{}, childErr
		}
		expression.Args = append(expression.Args, child)
	}
	return expression, nil
}

func splitExpressionArgs(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}
	depth := 0
	start := 0
	parts := []string{}
	for index, character := range raw {
		switch character {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced expression parentheses")
			}
		case ',':
			if depth == 0 {
				part := strings.TrimSpace(raw[start:index])
				if part == "" {
					return nil, fmt.Errorf("empty expression argument")
				}
				parts = append(parts, part)
				start = index + 1
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced expression parentheses")
	}
	part := strings.TrimSpace(raw[start:])
	if part == "" {
		return nil, fmt.Errorf("empty expression argument")
	}
	return append(parts, part), nil
}

func checkExpression(env typecheckEnvironment, expression Expr, capabilities map[string]bool) typedWork {
	switch expression.Name {
	case "compose":
		if len(expression.Args) != 2 {
			return refutedWork("COMPOSITION_ARITY")
		}
		left := checkExpression(env, expression.Args[0], capabilities)
		right := checkExpression(env, expression.Args[1], capabilities)
		if left.decision != "CLOSED" || right.decision != "CLOSED" {
			return mergeNonClosed(env, left, right)
		}
		if left.typed.OutputType != right.typed.InputType {
			return refutedWork("TYPE_MISMATCH")
		}
		if env.rules["compose_effects"].Properties["effects"] != "disjoint_union" {
			return refutedWork("COMPOSITION_RULE_NOT_DECLARED")
		}
		if overlaps(left.typed.Effects, right.typed.Effects) {
			return refutedWork("EFFECT_ROW_OVERLAP")
		}
		if env.rules["stage_escape"].Properties["stage"] != "must_not_escape" {
			return refutedWork("STAGE_RULE_NOT_DECLARED")
		}
		return typedWork{decision: "CLOSED", reason: "COMPOSITION_TYPED", typed: TypedExpression{
			Expression: expression, InputType: left.typed.InputType, OutputType: right.typed.OutputType,
			Stage: maxInt(left.typed.Stage, right.typed.Stage), Origin: right.typed.Origin,
			RequiredCapabilities: union(left.typed.RequiredCapabilities, right.typed.RequiredCapabilities),
			Effects:              union(left.typed.Effects, right.typed.Effects),
			Preconditions:        append(append([]string{}, left.typed.Preconditions...), right.typed.Preconditions...),
			Postconditions:       append(append([]string{}, left.typed.Postconditions...), right.typed.Postconditions...),
			Plan:                 append(append([]PlanStep{}, left.typed.Plan...), right.typed.Plan...), Replay: left.typed.Replay || right.typed.Replay,
		}}
	case "instantiate":
		if len(expression.Args) != 2 || expression.Args[0].Name == "" || len(expression.Args[0].Args) != 0 || expression.Args[1].Name == "" || len(expression.Args[1].Args) != 0 {
			return refutedWork("INSTANTIATION_SHAPE")
		}
		operation, ok := env.operations[expression.Args[0].Name]
		if !ok {
			return refutedWork("UNKNOWN_OPERATION")
		}
		capability := expression.Args[1].Name
		if env.rules["capability_binding"].Properties["missing"] != "UNKNOWN" {
			return refutedWork("CAPABILITY_RULE_NOT_DECLARED")
		}
		if !env.capabilities[capability] || !contains(capabilities, capability) {
			return capabilityUnknown(capability)
		}
		if !containsSlice(operation.Requires, "$cap") {
			return refutedWork("NON_POLYMORPHIC_INSTANTIATION")
		}
		return leafWork(env, expression, operation, []string{capability}, capabilities)
	case "stage_escape":
		if len(expression.Args) != 2 || expression.Args[0].Name == "" || expression.Args[1].Name == "" {
			return refutedWork("STAGE_ESCAPE_SHAPE")
		}
		operation, ok := env.operations[expression.Args[0].Name]
		if !ok {
			return refutedWork("UNKNOWN_OPERATION")
		}
		work := leafWork(env, expression.Args[0], operation, nil, capabilities)
		if work.decision != "CLOSED" {
			return work
		}
		target, targetOK := env.stages[expression.Args[1].Name]
		if !targetOK {
			return refutedWork("UNKNOWN_STAGE")
		}
		if env.rules["stage_escape"].Properties["stage"] == "must_not_escape" && work.typed.Stage > target {
			return refutedWork("STAGE_ESCAPE")
		}
		return work
	case "origin_capture":
		if len(expression.Args) != 2 || expression.Args[0].Name == "" || expression.Args[1].Name == "" {
			return refutedWork("ORIGIN_CAPTURE_SHAPE")
		}
		operation, ok := env.operations[expression.Args[0].Name]
		if !ok {
			return refutedWork("UNKNOWN_OPERATION")
		}
		work := leafWork(env, expression.Args[0], operation, nil, capabilities)
		if work.decision != "CLOSED" {
			return work
		}
		captured, capturedOK := env.origins[expression.Args[1].Name]
		if !capturedOK {
			return refutedWork("UNKNOWN_ORIGIN")
		}
		if env.rules["origin_hygiene"].Properties["origin"] == "must_be_hygienic" && operation.Origin != captured.ID {
			return refutedWork("ORIGIN_CAPTURE")
		}
		return work
	case "unbound":
		if len(expression.Args) != 1 {
			return refutedWork("UNBOUND_SHAPE")
		}
		return checkExpression(env, expression.Args[0], map[string]bool{})
	case "replay":
		if len(expression.Args) != 1 {
			return refutedWork("REPLAY_SHAPE")
		}
		if !contains(capabilities, "replay.fixture") {
			return capabilityUnknown("replay.fixture")
		}
		work := checkExpression(env, expression.Args[0], capabilities)
		if work.decision != "CLOSED" {
			return work
		}
		if env.rules["replay"].Properties["digest"] != "exact" {
			return refutedWork("REPLAY_RULE_NOT_DECLARED")
		}
		work.typed.Expression = expression
		work.typed.Effects = union(work.typed.Effects, []string{"replay.verify"})
		work.typed.RequiredCapabilities = union(work.typed.RequiredCapabilities, []string{"replay.fixture"})
		work.typed.Plan = append(work.typed.Plan, PlanStep{Operation: "replay", Capability: "replay.fixture"})
		work.typed.Replay = true
		work.reason = "REPLAY_TYPED"
		return work
	default:
		operation, ok := env.operations[expression.Name]
		if !ok {
			return refutedWork("UNKNOWN_OPERATION")
		}
		return leafWork(env, expression, operation, nil, capabilities)
	}
}

func leafWork(env typecheckEnvironment, expression Expr, operation OperationDecl, instantiated []string, capabilities map[string]bool) typedWork {
	required := []string{}
	for _, capability := range operation.Requires {
		if capability == "$cap" {
			required = append(required, instantiated...)
			continue
		}
		required = append(required, capability)
		if !contains(capabilities, capability) {
			return capabilityUnknown(capability)
		}
	}
	return typedWork{decision: "CLOSED", reason: "OPERATION_TYPED", typed: TypedExpression{
		Expression: expression, InputType: operation.Input, OutputType: operation.Output, Stage: env.stages[operation.Stage], Origin: operation.Origin,
		RequiredCapabilities: unique(required), Effects: unique(operation.Effects), Preconditions: []string{operation.Pre}, Postconditions: []string{operation.Post},
		Plan: []PlanStep{{Operation: operation.ID, Capability: first(required)}},
	}}
}

func capabilityUnknown(capability string) typedWork {
	return typedWork{decision: "UNKNOWN", reason: "UNBOUND_CAPABILITY", unknown: Unknown{
		Stage: "TYPECHECK", Step: "bind_capability", Reason: "UNBOUND_CAPABILITY", UnknownClass: "UNBOUND_CAPABILITY",
		NextOperation: "BindCompilerFixture", BlockedBy: []string{capability},
	}}
}

func refutedWork(reason string) typedWork { return typedWork{decision: "REFUTED", reason: reason} }

func mergeNonClosed(env typecheckEnvironment, left, right typedWork) typedWork {
	if left.decision == "CLOSED" {
		return right
	}
	if right.decision == "CLOSED" {
		return left
	}
	if decisionBefore(env.source.Precedence, left.decision, right.decision) {
		return left
	}
	return right
}

func decisionBefore(precedence []string, left, right string) bool {
	leftIndex, rightIndex := len(precedence), len(precedence)
	for index, decision := range precedence {
		if decision == left {
			leftIndex = index
		}
		if decision == right {
			rightIndex = index
		}
	}
	return leftIndex <= rightIndex
}

func firstReason(left, right typedWork, fallback string) string {
	if left.decision == "REFUTED" {
		return left.reason
	}
	if right.decision == "REFUTED" {
		return right.reason
	}
	if left.decision == "UNKNOWN" {
		return left.reason
	}
	if right.decision == "UNKNOWN" {
		return right.reason
	}
	return fallback
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func contains(values map[string]bool, value string) bool { return values[value] }
func containsSlice(values []string, value string) bool {
	for _, current := range values {
		if current == value {
			return true
		}
	}
	return false
}
func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func union(left, right []string) []string {
	result := append([]string{}, left...)
	for _, value := range right {
		if !containsSlice(result, value) {
			result = append(result, value)
		}
	}
	return result
}

func unique(values []string) []string { return union(values, nil) }
func overlaps(left, right []string) bool {
	for _, value := range left {
		if containsSlice(right, value) {
			return true
		}
	}
	return false
}
func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
