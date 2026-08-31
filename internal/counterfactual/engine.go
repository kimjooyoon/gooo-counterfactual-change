package counterfactual

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-counterfactual-change/generated"
)

var artifactNames = []string{
	"experiment-manifest.json",
	"semantic-ir.json",
	"baseline-binding-receipt.json",
	"candidate.patch.json",
	"candidate-transformations.ndjson",
	"before-observations.ndjson",
	"after-observations.ndjson",
	"guardrail-outcomes.ndjson",
	"claims.ndjson",
	"adoption-decisions.ndjson",
	"execution-receipt.json",
	"resource-build-test-observations.ndjson",
	"replay-receipt.json",
	"counterfactual-report.md",
}

type Compilation struct {
	IR       SemanticIR
	IRDigest string
}

type caseEvaluation struct {
	Scenario       Scenario
	Before         ObservationRecord
	After          ObservationRecord
	Guardrail      GuardrailRecord
	Claim          ClaimRecord
	Adoption       AdoptionDecision
	Transformation CandidateTransformation
}

func Compile(sourcePath, contractPath, irPath string) (Compilation, error) {
	source, err := ParseSource(sourcePath)
	if err != nil {
		return Compilation{}, err
	}
	contract, err := LoadContract(contractPath)
	if err != nil {
		return Compilation{}, err
	}
	if err := ValidateDeclarations(source, contract); err != nil {
		return Compilation{}, err
	}
	if err := ensureExternalPath(irPath, sourcePath); err != nil {
		return Compilation{}, err
	}
	ir := SemanticIR{
		Schema: IRSchema, Version: "v1", ExperimentID: source.ExperimentID,
		BaselineID: source.BaselineID, ClaimID: source.ClaimID, CandidateID: source.CandidateID,
		DenominatorID: source.DenominatorID, CellCount: source.CellCount, CaseCount: source.CaseCount,
		Authority: source.Authority, Precedence: append([]string(nil), source.Precedence...),
		UnknownFields: append([]string(nil), source.UnknownFields...), Claim: source.Claim,
		Candidate: source.Candidate, Guardrails: append([]string(nil), source.Guardrails...),
		Activities: append([]Activity(nil), source.Activities...), SourceDigest: source.SourceDigest,
	}
	ir.ContractDigest, err = DigestValue(contract)
	if err != nil {
		return Compilation{}, err
	}
	ir.IRDigest, err = unsignedIRDigest(ir)
	if err != nil {
		return Compilation{}, err
	}
	if err := os.MkdirAll(filepath.Dir(irPath), 0o755); err != nil {
		return Compilation{}, err
	}
	if err := writeJSON(irPath, ir); err != nil {
		return Compilation{}, err
	}
	return Compilation{IR: ir, IRDigest: ir.IRDigest}, nil
}

func unsignedIRDigest(ir SemanticIR) (string, error) {
	ir.IRDigest = ""
	return DigestValue(ir)
}

func VerifyIR(path, sourcePath string, expected SemanticIR) (SemanticIR, error) {
	var ir SemanticIR
	if err := readJSON(path, &ir); err != nil {
		return SemanticIR{}, err
	}
	sourceDigest, err := DigestFile(sourcePath)
	if err != nil {
		return SemanticIR{}, err
	}
	if ir.Schema != IRSchema || ir.SourceDigest != sourceDigest || ir.IRDigest == "" {
		return SemanticIR{}, fmt.Errorf("semantic IR does not bind the source")
	}
	computed, err := unsignedIRDigest(ir)
	if err != nil {
		return SemanticIR{}, err
	}
	if computed != ir.IRDigest {
		return SemanticIR{}, fmt.Errorf("semantic IR digest mismatch")
	}
	if ir.IRDigest != expected.IRDigest || ir.ContractDigest != expected.ContractDigest {
		return SemanticIR{}, fmt.Errorf("semantic IR is not the compiled IR")
	}
	return ir, nil
}

func loadCorpus(path string, ir SemanticIR) (ScenarioCorpus, string, error) {
	var corpus ScenarioCorpus
	if err := readJSON(path, &corpus); err != nil {
		return ScenarioCorpus{}, "", err
	}
	if corpus.Schema != CaseSchema || corpus.DenominatorID != ir.DenominatorID || len(corpus.Cases) != CaseCount {
		return ScenarioCorpus{}, "", fmt.Errorf("scenario corpus must contain exactly %d cases for the fixed denominator", CaseCount)
	}
	seen := make(map[string]bool, len(corpus.Cases))
	for _, scenario := range corpus.Cases {
		if scenario.CaseID == "" || seen[scenario.CaseID] {
			return ScenarioCorpus{}, "", fmt.Errorf("scenario case identity is not unique")
		}
		seen[scenario.CaseID] = true
		if scenario.ClaimID != ir.ClaimID || scenario.CandidateID != ir.CandidateID {
			return ScenarioCorpus{}, "", fmt.Errorf("scenario %s is not bound to the declared claim and candidate", scenario.CaseID)
		}
		if scenario.ExpectedState != StateClosed && scenario.ExpectedState != StateUnknown && scenario.ExpectedState != StateRefuted {
			return ScenarioCorpus{}, "", fmt.Errorf("scenario %s has unsupported expected state", scenario.CaseID)
		}
	}
	digest, err := DigestValue(corpus)
	if err != nil {
		return ScenarioCorpus{}, "", err
	}
	return corpus, digest, nil
}

func verifyBaseline(path, sourcePath, expectedID string) (BaselineFixture, string, error) {
	var fixture BaselineFixture
	if err := readJSON(path, &fixture); err != nil {
		return BaselineFixture{}, "", err
	}
	if fixture.Schema != "gooo/counterfactual-change/baseline/v1" || fixture.BaselineID != expectedID ||
		fixture.SourceFile == "" || fixture.Revision == "" || !fixture.Immutable || fixture.FixtureDigest == "" {
		return BaselineFixture{}, "", fmt.Errorf("baseline fixture is incomplete or mutable")
	}
	var raw map[string]any
	if err := readJSON(path, &raw); err != nil {
		return BaselineFixture{}, "", err
	}
	declared := fixture.FixtureDigest
	raw["fixture_digest"] = ""
	actual, err := DigestValue(raw)
	if err != nil {
		return BaselineFixture{}, "", err
	}
	if actual != declared {
		return BaselineFixture{}, "", fmt.Errorf("baseline fixture digest mismatch")
	}
	sourceDigest, err := DigestFile(sourcePath)
	if err != nil {
		return BaselineFixture{}, "", err
	}
	if fixture.SourceDigest != sourceDigest || !strings.HasSuffix(filepath.ToSlash(sourcePath), filepath.ToSlash(fixture.SourceFile)) {
		return BaselineFixture{}, "", fmt.Errorf("baseline fixture does not bind the immutable source")
	}
	return fixture, actual, nil
}

func GenerateCandidate(ir SemanticIR) (CandidatePatch, error) {
	patch := CandidatePatch{
		Schema:               "gooo/counterfactual-change/candidate-patch/v1",
		CandidateID:          ir.Candidate.ID,
		ClaimID:              ir.Claim.ID,
		BaselineID:           ir.BaselineID,
		BaselineSourceDigest: ir.SourceDigest,
		Operation:            ir.Candidate.Operation,
		Target:               ir.Candidate.Target,
		PatchFormat:          ir.Candidate.PatchFormat,
		PatchText:            "require exact_claimed_outcome=CLAIMED_SUCCESS\nrequire every_fixed_guardrail=CLOSED\notherwise=REFUTED_or_UNKNOWN_by_evidence",
		ExecutionBoundary:    "CALLER_OWNED_EPHEMERAL_CI_COPY",
	}
	digest, err := DigestValue(patch)
	if err != nil {
		return CandidatePatch{}, err
	}
	patch.PatchDigest = digest
	return patch, nil
}

func Evaluate(options EvaluateOptions, compiled SemanticIR) error {
	if options.ExecutionMode != "EPHEMERAL_CI_COPY" {
		return fmt.Errorf("execution must be isolated in an ephemeral CI copy")
	}
	if options.Authority != (Authority{}) {
		return fmt.Errorf("evaluation authority must record repository_writes=0, local_test_executions=0, cross_project_required_gates=0")
	}
	ir, err := VerifyIR(options.IR, options.Source, compiled)
	if err != nil {
		return err
	}
	contract, err := LoadContract(options.Contract)
	if err != nil {
		return err
	}
	source, err := ParseSource(options.Source)
	if err != nil {
		return err
	}
	if err := ValidateDeclarations(source, contract); err != nil {
		return err
	}
	corpus, corpusDigest, err := loadCorpus(options.Cases, ir)
	if err != nil {
		return err
	}
	baseline, baselineDigest, err := verifyBaseline(options.BaselineFixture, options.Source, ir.BaselineID)
	if err != nil {
		return err
	}
	generatedDigest, err := DigestFile(options.GeneratedGo)
	if err != nil {
		return err
	}
	evaluatorDigest, err := DigestFile(options.Evaluator)
	if err != nil {
		return err
	}
	patch, err := GenerateCandidate(ir)
	if err != nil {
		return err
	}

	evaluations := make([]caseEvaluation, 0, len(corpus.Cases))
	for _, scenario := range corpus.Cases {
		result, err := evaluateCase(scenario, ir, patch)
		if err != nil {
			return err
		}
		evaluations = append(evaluations, result)
	}
	if err := validateCaseCounts(evaluations); err != nil {
		return err
	}
	if err := ensureEmptyOutput(options.ArtifactDir, options.Source); err != nil {
		return err
	}

	beforeRecords := make([]ObservationRecord, 0, CaseCount)
	afterRecords := make([]ObservationRecord, 0, CaseCount)
	guardrailRecords := make([]GuardrailRecord, 0, CaseCount)
	claimRecords := make([]ClaimRecord, 0, CaseCount)
	adoptionRecords := make([]AdoptionDecision, 0, CaseCount)
	transformationRecords := make([]CandidateTransformation, 0, CaseCount)
	resourceRecords := make([]map[string]any, 0, CaseCount)
	for _, evaluation := range evaluations {
		beforeRecords = append(beforeRecords, evaluation.Before)
		afterRecords = append(afterRecords, evaluation.After)
		guardrailRecords = append(guardrailRecords, evaluation.Guardrail)
		claimRecords = append(claimRecords, evaluation.Claim)
		adoptionRecords = append(adoptionRecords, evaluation.Adoption)
		transformationRecords = append(transformationRecords, evaluation.Transformation)
		resourceRecords = append(resourceRecords, map[string]any{
			"case_id": evaluation.Scenario.CaseID,
			"before": map[string]any{
				"build":     evaluation.Scenario.Before.Build,
				"tests":     evaluation.Scenario.Before.Tests,
				"resources": evaluation.Scenario.Before.Resources,
			},
			"after": map[string]any{
				"build":     evaluation.Scenario.After.Build,
				"tests":     evaluation.Scenario.After.Tests,
				"resources": evaluation.Scenario.After.Resources,
			},
		})
	}

	provenance := map[string]any{
		"source_digest":           ir.SourceDigest,
		"contract_digest":         ir.ContractDigest,
		"semantic_ir_digest":      ir.IRDigest,
		"baseline_fixture_digest": baselineDigest,
		"generated_go_digest":     generatedDigest,
		"evaluator_digest":        evaluatorDigest,
		"scenario_corpus_digest":  corpusDigest,
		"candidate_patch_digest":  patch.PatchDigest,
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "semantic-ir.json"), ir); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "baseline-binding-receipt.json"), map[string]any{
		"schema":                    "gooo/counterfactual-change/baseline-binding-receipt/v1",
		"baseline_id":               baseline.BaselineID,
		"revision":                  baseline.Revision,
		"immutable":                 baseline.Immutable,
		"source_file":               baseline.SourceFile,
		"source_digest":             baseline.SourceDigest,
		"fixture_digest":            baselineDigest,
		"source_repository_changed": false,
	}); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "candidate.patch.json"), patch); err != nil {
		return err
	}
	if err := writeNDJSON(filepath.Join(options.ArtifactDir, "candidate-transformations.ndjson"), transformationRecords); err != nil {
		return err
	}
	if err := writeNDJSON(filepath.Join(options.ArtifactDir, "before-observations.ndjson"), beforeRecords); err != nil {
		return err
	}
	if err := writeNDJSON(filepath.Join(options.ArtifactDir, "after-observations.ndjson"), afterRecords); err != nil {
		return err
	}
	if err := writeNDJSON(filepath.Join(options.ArtifactDir, "guardrail-outcomes.ndjson"), guardrailRecords); err != nil {
		return err
	}
	if err := writeNDJSON(filepath.Join(options.ArtifactDir, "claims.ndjson"), claimRecords); err != nil {
		return err
	}
	if err := writeNDJSON(filepath.Join(options.ArtifactDir, "adoption-decisions.ndjson"), adoptionRecords); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "execution-receipt.json"), map[string]any{
		"schema":                                 "gooo/counterfactual-change/execution-receipt/v1",
		"execution_mode":                         options.ExecutionMode,
		"candidate_execution_location":           "CALLER_OWNED_EPHEMERAL_CI_COPY",
		"source_repository_writes":               0,
		"local_test_executions":                  0,
		"cross_project_required_gates":            0,
		"candidate_applied_to_source_repository": false,
		"case_count":                             CaseCount,
		"provenance":                             provenance,
	}); err != nil {
		return err
	}
	if err := writeNDJSON(filepath.Join(options.ArtifactDir, "resource-build-test-observations.ndjson"), resourceRecords); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.ArtifactDir, "replay-receipt.json"), map[string]any{
		"schema":                        "gooo/counterfactual-change/replay-receipt/v1",
		"replay_identity":               provenance,
		"deterministic_replay_required": true,
		"second_run_byte_identical":     true,
		"replay_alone_can_close":        false,
	}); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(options.ArtifactDir, "counterfactual-report.md"), []byte(renderReport(evaluations, provenance)), 0o644); err != nil {
		return err
	}

	artifactDigests := make(map[string]string, len(artifactNames)-1)
	for _, name := range artifactNames {
		if name == "experiment-manifest.json" {
			continue
		}
		digest, err := DigestFile(filepath.Join(options.ArtifactDir, name))
		if err != nil {
			return err
		}
		artifactDigests[name] = digest
	}
	manifest := map[string]any{
		"schema":         "gooo/counterfactual-change/experiment-manifest/v1",
		"subject_sha":    options.SubjectSHA,
		"go_version":     options.GoVersion,
		"experiment_id":  ir.ExperimentID,
		"baseline_id":    ir.BaselineID,
		"claim_id":       ir.ClaimID,
		"candidate_id":   ir.CandidateID,
		"execution_mode": options.ExecutionMode,
		"contracts": map[string]any{
			"activity_cells":       ActivityCount,
			"activity_mapping":     "1:1",
			"executable_cases":     CaseCount,
			"fixed_guardrails":     GuardrailCount,
			"root_readme_excluded": true,
		},
		"case_states": map[string]int{
			StateClosed:  countStates(evaluations, StateClosed),
			StateUnknown: countStates(evaluations, StateUnknown),
			StateRefuted: countStates(evaluations, StateRefuted),
		},
		"adoption_decisions": map[string]int{
			"ADOPTABLE":    countAdoptions(evaluations, "ADOPTABLE"),
			"HOLD_UNKNOWN": countAdoptions(evaluations, "HOLD_UNKNOWN"),
			"REJECTED":     countAdoptions(evaluations, "REJECTED"),
		},
		"precedence":       []string{StateRefuted, StateUnknown, StateClosed},
		"provenance":       provenance,
		"artifacts":        artifactNames,
		"artifact_digests": artifactDigests,
		"authority":        options.Authority,
		"inventory": map[string]any{
			"root_readme_excluded": true,
			"directories":          options.Metrics.Directories,
			"files":                options.Metrics.Files,
			"physical_lines":       options.Metrics.PhysicalLines,
			"go_files":             options.Metrics.GoFiles,
			"go_lines":             options.Metrics.GoLines,
			"gooo_files":           options.Metrics.GoooFiles,
			"gooo_lines":           options.Metrics.GoooLines,
		},
		"core_adoption": map[string]any{
			"performed":          false,
			"decision_authority": "HUMAN_REVIEW_ONLY",
		},
		"semantic_close_rule": "ADOPTABLE requires exact claimed outcome CLOSED and every fixed guardrail CLOSED",
		"unknown_rule":        "missing comparable observations produce UNKNOWN with the six-field frontier tuple",
	}
	return writeJSON(filepath.Join(options.ArtifactDir, "experiment-manifest.json"), manifest)
}

func evaluateCase(scenario Scenario, ir SemanticIR, patch CandidatePatch) (caseEvaluation, error) {
	if scenario.Before.ExecutionMode != "EPHEMERAL_CI_COPY" || scenario.After.ExecutionMode != "EPHEMERAL_CI_COPY" {
		return caseEvaluation{}, fmt.Errorf("case %s is not isolated in an ephemeral CI copy", scenario.CaseID)
	}
	if !scenario.Before.Comparable || scenario.Before.Subject != "IMMUTABLE_BASELINE" || scenario.Before.Outcome != ir.Claim.ExpectedBefore {
		return caseEvaluation{}, fmt.Errorf("case %s has an invalid exact baseline observation", scenario.CaseID)
	}
	if err := validateObservation(scenario.Before); err != nil {
		return caseEvaluation{}, fmt.Errorf("case %s before observation: %w", scenario.CaseID, err)
	}
	if err := validateObservation(scenario.After); err != nil {
		return caseEvaluation{}, fmt.Errorf("case %s after observation: %w", scenario.CaseID, err)
	}
	if len(scenario.Guardrails) != GuardrailCount {
		return caseEvaluation{}, fmt.Errorf("case %s does not contain all fixed guardrails", scenario.CaseID)
	}
	seenGuardrails := make(map[string]bool, GuardrailCount)
	allClosed := true
	for _, guardrail := range scenario.Guardrails {
		if !contains(ir.Guardrails, guardrail.ID) || seenGuardrails[guardrail.ID] ||
			(guardrail.State != StateClosed && guardrail.State != "OPEN") {
			return caseEvaluation{}, fmt.Errorf("case %s has an invalid guardrail observation", scenario.CaseID)
		}
		seenGuardrails[guardrail.ID] = true
		if guardrail.State != StateClosed {
			allClosed = false
		}
	}
	if len(seenGuardrails) != GuardrailCount {
		return caseEvaluation{}, fmt.Errorf("case %s has duplicate or missing fixed guardrails", scenario.CaseID)
	}

	beforeDigest, err := DigestValue(scenario.Before)
	if err != nil {
		return caseEvaluation{}, err
	}
	afterDigest, err := DigestValue(scenario.After)
	if err != nil {
		return caseEvaluation{}, err
	}
	guardrailDigest, err := DigestValue(scenario.Guardrails)
	if err != nil {
		return caseEvaluation{}, err
	}
	guardrail := GuardrailRecord{CaseID: scenario.CaseID, Guardrails: scenario.Guardrails, AllClosed: allClosed}
	guardrail.RecordDigest, err = recordDigest(guardrail)
	if err != nil {
		return caseEvaluation{}, err
	}
	decision := generated.EvaluateAfter(scenario.After.Comparable, scenario.After.Outcome, scenario.After.Build.Status, scenario.After.Tests.Status)
	state := decision.State
	reason := decision.Reason
	if !allClosed {
		state = StateRefuted
		reason = "FIXED_GUARDRAIL_OPEN"
	}
	claim := Claim{State: state}
	comparison := "EXACT_CLAIM_CLOSED"
	if state == StateUnknown {
		claim.Stage = "EXECUTION"
		claim.Step = "compare_before_after"
		claim.Reason = reason
		claim.UnknownClass = "MISSING_COMPARABLE_OBSERVATION"
		claim.NextOperation = "RUN_CANDIDATE_IN_EPHEMERAL_CI_COPY"
		claim.BlockedBy = []string{"after.build", "after.tests", "after.resources"}
		comparison = "COMPARISON_INCOMPLETE"
	} else if state == StateRefuted {
		comparison = reason
	}
	claimRecord := ClaimRecord{
		CaseID: scenario.CaseID, Kind: scenario.Kind, Claim: claim,
		BeforeDigest: beforeDigest, AfterDigest: afterDigest,
		GuardrailDigest: guardrailDigest, ExactComparison: comparison,
	}
	claimRecord.RecordDigest, err = recordDigest(claimRecord)
	if err != nil {
		return caseEvaluation{}, err
	}
	decisionName := "REJECTED"
	if state == StateClosed {
		decisionName = "ADOPTABLE"
	} else if state == StateUnknown {
		decisionName = "HOLD_UNKNOWN"
	}
	rationale := "known contradiction or regression blocks adoption"
	if state == StateClosed {
		rationale = "exact claimed outcome closed and every fixed guardrail closed"
	} else if state == StateUnknown {
		rationale = "comparable before/after evidence is missing; human adoption is held"
	}
	adoption := AdoptionDecision{
		CaseID: scenario.CaseID, ClaimState: state, Decision: decisionName,
		Rationale: rationale, HumanAuthority: "HUMAN_REVIEW_ONLY", AppliedToCore: false,
	}
	adoption.RecordDigest, err = recordDigest(adoption)
	if err != nil {
		return caseEvaluation{}, err
	}
	if scenario.ExpectedState != state || scenario.ExpectedAdoption != decisionName {
		return caseEvaluation{}, fmt.Errorf("case %s expected %s/%s but derived %s/%s", scenario.CaseID, scenario.ExpectedState, scenario.ExpectedAdoption, state, decisionName)
	}
	return caseEvaluation{
		Scenario:  scenario,
		Before:    ObservationRecord{CaseID: scenario.CaseID, Side: "BEFORE", Observation: scenario.Before, ObservationDigest: beforeDigest},
		After:     ObservationRecord{CaseID: scenario.CaseID, Side: "AFTER", Observation: scenario.After, ObservationDigest: afterDigest},
		Guardrail: guardrail, Claim: claimRecord, Adoption: adoption,
		Transformation: CandidateTransformation{
			CaseID: scenario.CaseID, CandidateID: patch.CandidateID, PatchDigest: patch.PatchDigest,
			BeforeSubject: scenario.Before.Subject, AfterSubject: scenario.After.Subject,
			Transformation: patch.Operation,
		},
	}, nil
}

func validateObservation(observation Observation) error {
	if observation.ExecutionMode != "EPHEMERAL_CI_COPY" {
		return fmt.Errorf("execution mode is not EPHEMERAL_CI_COPY")
	}
	if observation.Subject == "" || observation.Outcome == "" {
		return fmt.Errorf("subject and outcome are required")
	}
	if observation.Build.Status != "PASS" && observation.Build.Status != "FAIL" && observation.Build.Status != "MISSING" {
		return fmt.Errorf("unsupported build status")
	}
	if observation.Tests.Status != "PASS" && observation.Tests.Status != "FAIL" && observation.Tests.Status != "MISSING" {
		return fmt.Errorf("unsupported test status")
	}
	if observation.Build.ExitCode < 0 || observation.Build.DurationMS < 0 || observation.Build.ArtifactBytes < 0 ||
		observation.Tests.ExitCode < 0 || observation.Tests.Executed < 0 || observation.Tests.Passed < 0 ||
		observation.Tests.Failed < 0 || observation.Tests.DurationMS < 0 || observation.Resources.CPUTimeMS < 0 ||
		observation.Resources.PeakRSSKiB < 0 || observation.Resources.WallMS < 0 {
		return fmt.Errorf("observations must use non-negative integer values")
	}
	if observation.Tests.Passed+observation.Tests.Failed != observation.Tests.Executed {
		return fmt.Errorf("test observations do not reconcile")
	}
	if observation.Comparable && (observation.Build.Status == "MISSING" || observation.Tests.Status == "MISSING") {
		return fmt.Errorf("comparable observation cannot have a missing build or test")
	}
	return nil
}

func validateCaseCounts(evaluations []caseEvaluation) error {
	if len(evaluations) != CaseCount || countStates(evaluations, StateClosed) != 4 ||
		countStates(evaluations, StateUnknown) != 4 || countStates(evaluations, StateRefuted) != 4 {
		return fmt.Errorf("fixed case states must be CLOSED=4, UNKNOWN=4, REFUTED=4")
	}
	if countKinds(evaluations, "NORMAL") != 4 || countKinds(evaluations, "UNKNOWN") != 4 || countKinds(evaluations, "REFUTED") != 4 {
		return fmt.Errorf("fixture kinds must be NORMAL=4, UNKNOWN=4, REFUTED=4")
	}
	return nil
}

func countStates(evaluations []caseEvaluation, state string) int {
	count := 0
	for _, evaluation := range evaluations {
		if evaluation.Claim.Claim.State == state {
			count++
		}
	}
	return count
}

func countAdoptions(evaluations []caseEvaluation, decision string) int {
	count := 0
	for _, evaluation := range evaluations {
		if evaluation.Adoption.Decision == decision {
			count++
		}
	}
	return count
}

func countKinds(evaluations []caseEvaluation, kind string) int {
	count := 0
	for _, evaluation := range evaluations {
		if evaluation.Scenario.Kind == kind {
			count++
		}
	}
	return count
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func recordDigest(value any) (string, error) {
	return DigestValue(value)
}
