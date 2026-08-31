package counterfactual

import (
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-counterfactual-change/generated"
)

func TestFixedSourceAndContract(t *testing.T) {
	sourcePath := filepath.Join("..", "..", "examples", "counterfactual-change", "main.gooo")
	contractPath := filepath.Join("..", "..", "contracts", "counterfactual-denominator-v1.json")
	source, err := ParseSource(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := LoadContract(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDeclarations(source, contract); err != nil {
		t.Fatal(err)
	}
	if len(source.Activities) != ActivityCount || source.CaseCount != CaseCount {
		t.Fatalf("unexpected fixed declaration: activities=%d cases=%d", len(source.Activities), source.CaseCount)
	}
}

func TestCandidatePatchIsUsefulAndBound(t *testing.T) {
	sourcePath := filepath.Join("..", "..", "examples", "counterfactual-change", "main.gooo")
	contractPath := filepath.Join("..", "..", "contracts", "counterfactual-denominator-v1.json")
	compilation, err := Compile(sourcePath, contractPath, filepath.Join(t.TempDir(), "semantic-ir.json"))
	if err != nil {
		t.Fatal(err)
	}
	patch, err := GenerateCandidate(compilation.IR)
	if err != nil {
		t.Fatal(err)
	}
	if patch.CandidateID != "candidate-explicit-result" || patch.BaselineSourceDigest != compilation.IR.SourceDigest || patch.PatchDigest == "" {
		t.Fatalf("candidate patch is not bound: %+v", patch)
	}
	if patch.PatchText == "" || patch.ExecutionBoundary != "CALLER_OWNED_EPHEMERAL_CI_COPY" {
		t.Fatalf("candidate patch lacks a useful isolated transformation")
	}
}

func TestDecisionPrecedence(t *testing.T) {
	unknown := generated.EvaluateAfter(false, "OBSERVATION_UNAVAILABLE", "MISSING", "MISSING")
	if unknown.State != StateUnknown {
		t.Fatalf("expected UNKNOWN, got %s", unknown.State)
	}
	refuted := generated.EvaluateAfter(false, "REGRESSION", "MISSING", "MISSING")
	if refuted.State != StateRefuted {
		t.Fatalf("known regression must be REFUTED, got %s", refuted.State)
	}
}
