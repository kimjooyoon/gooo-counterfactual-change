// Code generated from examples/counterfactual-change/main.gooo; DO NOT EDIT.
// The parent evaluator in internal/counterfactual independently checks the
// generated decision and all protocol guardrails.
package generated

const GeneratedVersion = "gooo-counterfactual-generated-evaluator-v1"

const (
	Closed  = "CLOSED"
	Unknown = "UNKNOWN"
	Refuted = "REFUTED"
)

type Decision struct {
	State  string
	Reason string
}

func EvaluateAfter(comparable bool, outcome string, buildStatus string, testStatus string) Decision {
	if outcome == "REGRESSION" || buildStatus == "FAIL" || testStatus == "FAIL" {
		return Decision{State: Refuted, Reason: "KNOWN_REGRESSION"}
	}
	if !comparable {
		return Decision{State: Unknown, Reason: "MISSING_COMPARABLE_OBSERVATION"}
	}
	if outcome != "CLAIMED_SUCCESS" || buildStatus != "PASS" || testStatus != "PASS" {
		return Decision{State: Refuted, Reason: "EXACT_CLAIM_CONTRADICTED"}
	}
	return Decision{State: Closed, Reason: "EXACT_CLAIM_AND_EXECUTION_CLOSED"}
}
