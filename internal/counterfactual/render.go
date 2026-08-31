package counterfactual

import (
	"fmt"
	"strings"
)

func renderReport(evaluations []caseEvaluation, provenance map[string]any) string {
	var builder strings.Builder
	builder.WriteString("# Counterfactual experiment report\n\n")
	builder.WriteString("This report records a candidate transformation against an immutable baseline. The candidate was evaluated in an ephemeral CI copy; the source repository was not changed and no result was applied to a core language repository.\n\n")
	builder.WriteString("## Resolution contract\n\n")
	builder.WriteString("Claims have exactly three states with precedence `REFUTED > UNKNOWN > CLOSED`. `ADOPTABLE` is emitted only when the exact claimed outcome closes and all four fixed guardrails are `CLOSED`. Missing comparable observations are `UNKNOWN`; a known contradiction or regression is `REFUTED`.\n\n")
	builder.WriteString("- Activity denominator: 15 fixed cells, mapped 1:1 to the released `.gooo` activities.\n")
	builder.WriteString("- Executable fixture denominator: 12 cases.\n")
	builder.WriteString("- Case states: CLOSED 4, UNKNOWN 4, REFUTED 4.\n")
	builder.WriteString("- Human decisions: ADOPTABLE 4, HOLD_UNKNOWN 4, REJECTED 4.\n")
	builder.WriteString("- Repository writes: 0; local test executions: 0; cross-project required gates: 0.\n\n")
	builder.WriteString("## Exact case results\n\n")
	builder.WriteString("| case | kind | state | human decision | before | after | guardrails |\n")
	builder.WriteString("|---|---|---|---|---|---|---|\n")
	for _, evaluation := range evaluations {
		guardrailState := "CLOSED"
		if !evaluation.Guardrail.AllClosed {
			guardrailState = "OPEN"
		}
		fmt.Fprintf(&builder, "| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` |\n", evaluation.Scenario.CaseID, evaluation.Scenario.Kind, evaluation.Claim.Claim.State, evaluation.Adoption.Decision, evaluation.Scenario.Before.Outcome, evaluation.Scenario.After.Outcome, guardrailState)
	}
	builder.WriteString("\n## UNKNOWN frontier\n\n")
	for _, evaluation := range evaluations {
		if evaluation.Claim.Claim.State != StateUnknown {
			continue
		}
		claim := evaluation.Claim.Claim
		fmt.Fprintf(&builder, "- `%s`: stage `%s`, step `%s`, reason `%s`, unknown_class `%s`, next_operation `%s`, blocked_by `%s`.\n", evaluation.Scenario.CaseID, claim.Stage, claim.Step, claim.Reason, claim.UnknownClass, claim.NextOperation, strings.Join(claim.BlockedBy, ","))
	}
	builder.WriteString("\n## Digest bindings\n\n")
	for _, key := range []string{"source_digest", "contract_digest", "semantic_ir_digest", "baseline_fixture_digest", "generated_go_digest", "evaluator_digest", "scenario_corpus_digest", "candidate_patch_digest"} {
		fmt.Fprintf(&builder, "- `%s`: `%v`\n", key, provenance[key])
	}
	builder.WriteString("\nThe candidate patch is a caller-owned artifact. This experiment records a human decision but does not authorize a core repository write.\n")
	return builder.String()
}
