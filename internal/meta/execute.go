package meta

import (
	"fmt"
	"strings"
)

func ExecuteProof(proof Proof, input FixtureInput) (ExecutionResult, error) {
	if proof.Schema != GeneratedSchema || proof.Decision != "CLOSED" {
		return ExecutionResult{Schema: ReceiptSchema, CaseID: proof.CaseID, Decision: proof.Decision, Reason: proof.TerminalReason}, nil
	}
	if proof.SourceDigest == "" || proof.ContractDigest == "" || proof.IRDigest == "" || proof.CaseID == "" {
		return ExecutionResult{}, fmt.Errorf("proof is missing an immutable binding")
	}
	if proof.TargetStage < proof.Typed.Stage || proof.Typed.InputType == "" || proof.Typed.OutputType == "" || len(proof.Typed.Preconditions) == 0 || len(proof.Typed.Postconditions) == 0 {
		return ExecutionResult{}, fmt.Errorf("proof does not carry stage, type, precondition, and postcondition facts")
	}
	compiled, err := CompileFixture(input.Raw)
	if err != nil {
		return ExecutionResult{}, err
	}
	result := ExecutionResult{Schema: ReceiptSchema, CaseID: proof.CaseID, Decision: "CLOSED", Reason: proof.TerminalReason, Output: compiled}
	previousDigest := DigestBytes([]byte(compiled))
	postcondition := "fixture_valid"
	readSeen := false
	for _, step := range proof.Typed.Plan {
		switch step.Operation {
		case "identity":
			result.EffectTrace = append(result.EffectTrace, "identity:∅")
			postcondition = "artifact_stable"
		case "read_fixture":
			result.Effects = appendUnique(result.Effects, "fixture.read")
			result.EffectTrace = append(result.EffectTrace, "read_fixture:fixture.read")
			readSeen = true
			postcondition = "fixture_read"
		case "normalize":
			if !readSeen {
				return ExecutionResult{}, fmt.Errorf("normalize precondition fixture_read was not established")
			}
			result.Effects = appendUnique(result.Effects, "artifact.normalize")
			result.EffectTrace = append(result.EffectTrace, "normalize:artifact.normalize")
			postcondition = "fixture_normalized"
		case "generic_cap":
			result.Effects = appendUnique(result.Effects, "fixture.read")
			result.EffectTrace = append(result.EffectTrace, "generic_cap:fixture.read")
			readSeen = true
			postcondition = "fixture_read"
		case "replay":
			currentDigest := DigestBytes([]byte(result.Output))
			if currentDigest != previousDigest {
				return ExecutionResult{}, fmt.Errorf("replay digest mismatch")
			}
			result.Effects = appendUnique(result.Effects, "replay.verify")
			result.EffectTrace = append(result.EffectTrace, "replay:replay.verify")
		default:
			return ExecutionResult{}, fmt.Errorf("generated proof contains unknown executor step %q", step.Operation)
		}
	}
	if postcondition != proof.ExpectedPost {
		return ExecutionResult{}, fmt.Errorf("postcondition mismatch: got %s, want %s", postcondition, proof.ExpectedPost)
	}
	result.ArtifactDigest = DigestBytes([]byte(result.Output))
	return result, nil
}

func CompileFixture(raw string) (string, error) {
	module, term := "", ""
	for _, rawLine := range strings.Split(raw, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return "", fmt.Errorf("compiler fixture line must have two fields")
		}
		switch fields[0] {
		case "module":
			if module != "" {
				return "", fmt.Errorf("compiler fixture has duplicate module")
			}
			module = fields[1]
		case "term":
			if term != "" {
				return "", fmt.Errorf("compiler fixture has duplicate term")
			}
			term = fields[1]
		default:
			return "", fmt.Errorf("compiler fixture has unknown declaration %q", fields[0])
		}
	}
	if module == "" || term == "" {
		return "", fmt.Errorf("compiler fixture is incomplete")
	}
	return "module:" + module + ";term:" + term, nil
}

func appendUnique(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}
