package meta

import (
	"fmt"
	"reflect"
)

func VerifyIRAgainstSource(ir IR, source SourceDecl, contract Contract) error {
	if err := ValidateIR(ir); err != nil {
		return err
	}
	expected, err := BuildIR(source, contract)
	if err != nil {
		return err
	}
	if expected.IRDigest != ir.IRDigest || expected.SourceDigest != ir.SourceDigest || expected.ContractDigest != ir.ContractDigest {
		return fmt.Errorf("typed IR is stale or not source-bound")
	}
	if !reflect.DeepEqual(expected.Proofs, ir.Proofs) {
		return fmt.Errorf("proof-carrying IR differs from source lowering")
	}
	return nil
}

func VerifyProofBinding(proof Proof, ir IR) error {
	if proof.Schema != GeneratedSchema || proof.CaseID == "" || proof.SourceDigest != ir.SourceDigest || proof.ContractDigest != ir.ContractDigest || proof.IRDigest != ir.IRDigest {
		return fmt.Errorf("generated proof binding is invalid")
	}
	for _, candidate := range ir.Proofs {
		if candidate.CaseID == proof.CaseID && reflect.DeepEqual(candidate, proof) {
			return nil
		}
	}
	return fmt.Errorf("generated proof does not match typed IR for %s", proof.CaseID)
}

func VerifyExecution(proof Proof, result ExecutionResult, fixture FixtureDecl) VerificationResult {
	verified := true
	reason := "EXECUTION_RESULT_AND_EFFECT_TRACE_MATCH"
	if proof.Decision != "CLOSED" {
		if proof.Decision == "UNKNOWN" && !proof.Unknown.Complete() {
			return VerificationResult{CaseID: proof.CaseID, Decision: result.Decision, Reason: "UNKNOWN_TUPLE_INCOMPLETE", Verified: false}
		}
		verified = result.Decision == proof.Decision && result.Reason == proof.TerminalReason
		if !verified {
			reason = "NON_CLOSED_TERMINAL_DECISION_MISMATCH"
		}
		return VerificationResult{CaseID: proof.CaseID, Decision: result.Decision, Reason: reason, Verified: verified}
	}
	if result.CaseID != proof.CaseID || result.Decision != "CLOSED" || result.Reason != proof.TerminalReason {
		verified = false
		reason = "TERMINAL_REASON_MISMATCH"
	}
	if !sameStrings(result.Effects, proof.ExpectedEffects) {
		verified = false
		reason = "EFFECT_ROW_MISMATCH"
	}
	if !sameStrings(result.EffectTrace, proof.ExpectedEffectTrace) {
		verified = false
		reason = "EFFECT_TRACE_MISMATCH"
	}
	if result.Output != fixture.ExpectedOutput {
		verified = false
		reason = "COMPILER_FIXTURE_OUTPUT_MISMATCH"
	}
	if result.ArtifactDigest == "" || result.ArtifactDigest != DigestBytes([]byte(result.Output)) {
		verified = false
		reason = "ARTIFACT_DIGEST_MISMATCH"
	}
	return VerificationResult{CaseID: proof.CaseID, Decision: result.Decision, Reason: reason, Verified: verified}
}
