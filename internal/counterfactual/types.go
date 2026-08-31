package counterfactual

const (
	SourceSchema   = "gooo/counterfactual-change/source/v1"
	ContractSchema = "gooo/counterfactual-change/denominator/v1"
	IRSchema       = "gooo/counterfactual-change/semantic-ir/v1"
	CaseSchema     = "gooo/counterfactual-change/cases/v1"
	ActivityCount  = 15
	CaseCount      = 12
	GuardrailCount = 4

	StateClosed  = "CLOSED"
	StateUnknown = "UNKNOWN"
	StateRefuted = "REFUTED"
)

type Authority struct {
	RepositoryWrites          int `json:"repository_writes"`
	LocalTestExecutions       int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}

type Activity struct {
	Ordinal int    `json:"ordinal"`
	ID      string `json:"id"`
	Phase   string `json:"phase"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Edge    string `json:"edge"`
}

type ChangeClaim struct {
	ID            string `json:"id"`
	ExpectedBefore string `json:"expected_before"`
	ExpectedAfter  string `json:"expected_after"`
	Exact         bool   `json:"exact"`
}

type CandidateDecl struct {
	ID          string `json:"id"`
	Operation   string `json:"operation"`
	Target      string `json:"target"`
	PatchFormat string `json:"patch_format"`
}

type SourceDecl struct {
	Schema        string       `json:"schema"`
	Version       string       `json:"version"`
	ExperimentID  string       `json:"experiment_id"`
	BaselineID    string       `json:"baseline_id"`
	ClaimID       string       `json:"claim_id"`
	CandidateID   string       `json:"candidate_id"`
	DenominatorID string       `json:"denominator_id"`
	CellCount     int          `json:"cell_count"`
	CaseCount     int          `json:"case_count"`
	Authority     Authority    `json:"authority"`
	Precedence    []string     `json:"precedence"`
	UnknownFields []string     `json:"unknown_fields"`
	Claim         ChangeClaim  `json:"claim"`
	Candidate     CandidateDecl `json:"candidate"`
	Guardrails    []string     `json:"guardrails"`
	Activities    []Activity   `json:"activities"`
	SourceDigest  string       `json:"source_digest"`
}

type Contract struct {
	Schema     string     `json:"schema"`
	ID         string     `json:"id"`
	Version    string     `json:"version"`
	CellCount  int        `json:"cell_count"`
	CaseCount  int        `json:"case_count"`
	Fixed      bool       `json:"fixed"`
	Guardrails []string   `json:"guardrails"`
	Activities []Activity `json:"activities"`
}

type SemanticIR struct {
	Schema        string        `json:"schema"`
	Version       string        `json:"version"`
	ExperimentID  string        `json:"experiment_id"`
	BaselineID    string        `json:"baseline_id"`
	ClaimID       string        `json:"claim_id"`
	CandidateID   string        `json:"candidate_id"`
	DenominatorID string        `json:"denominator_id"`
	CellCount     int           `json:"cell_count"`
	CaseCount     int           `json:"case_count"`
	Authority     Authority     `json:"authority"`
	Precedence    []string      `json:"precedence"`
	UnknownFields []string      `json:"unknown_fields"`
	Claim         ChangeClaim   `json:"claim"`
	Candidate     CandidateDecl `json:"candidate"`
	Guardrails    []string      `json:"guardrails"`
	Activities    []Activity    `json:"activities"`
	SourceDigest  string        `json:"source_digest"`
	ContractDigest string       `json:"contract_digest"`
	IRDigest      string        `json:"ir_digest,omitempty"`
}

type BaselineFixture struct {
	Schema         string `json:"schema"`
	BaselineID     string `json:"baseline_id"`
	SourceFile     string `json:"source_file"`
	SourceDigest   string `json:"source_digest"`
	Revision       string `json:"revision"`
	Immutable      bool   `json:"immutable"`
	FixtureDigest  string `json:"fixture_digest"`
}

type BuildObservation struct {
	Status       string `json:"status"`
	ExitCode     int    `json:"exit_code"`
	DurationMS   int    `json:"duration_ms"`
	ArtifactBytes int   `json:"artifact_bytes"`
}

type TestObservation struct {
	Status     string `json:"status"`
	ExitCode   int    `json:"exit_code"`
	Executed   int    `json:"executed"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	DurationMS int    `json:"duration_ms"`
}

type ResourceObservation struct {
	CPUTimeMS  int `json:"cpu_time_ms"`
	PeakRSSKiB int `json:"peak_rss_kib"`
	WallMS     int `json:"wall_ms"`
}

type Observation struct {
	Comparable    bool                `json:"comparable"`
	ExecutionMode string              `json:"execution_mode"`
	Subject       string              `json:"subject"`
	Outcome       string              `json:"outcome"`
	Build         BuildObservation    `json:"build"`
	Tests         TestObservation     `json:"tests"`
	Resources     ResourceObservation `json:"resources"`
}

type GuardrailObservation struct {
	ID           string `json:"id"`
	State        string `json:"state"`
	Observed     int    `json:"observed"`
	Expected     int    `json:"expected"`
	Evidence     string `json:"evidence"`
}

type Scenario struct {
	CaseID             string                `json:"case_id"`
	Kind               string                `json:"kind"`
	CandidateID        string                `json:"candidate_id"`
	ClaimID            string                `json:"claim_id"`
	Before             Observation          `json:"before"`
	After              Observation          `json:"after"`
	Guardrails         []GuardrailObservation `json:"guardrails"`
	ExpectedState      string                `json:"expected_state"`
	ExpectedAdoption   string                `json:"expected_adoption"`
}

type ScenarioCorpus struct {
	Schema        string     `json:"schema"`
	DenominatorID string     `json:"denominator_id"`
	Cases         []Scenario `json:"cases"`
}

type CandidatePatch struct {
	Schema             string `json:"schema"`
	CandidateID        string `json:"candidate_id"`
	ClaimID            string `json:"claim_id"`
	BaselineID         string `json:"baseline_id"`
	BaselineSourceDigest string `json:"baseline_source_digest"`
	Operation          string `json:"operation"`
	Target             string `json:"target"`
	PatchFormat        string `json:"patch_format"`
	PatchText          string `json:"patch_text"`
	ExecutionBoundary  string `json:"execution_boundary"`
	PatchDigest        string `json:"patch_digest,omitempty"`
}

type CandidateTransformation struct {
	CaseID       string         `json:"case_id"`
	CandidateID  string         `json:"candidate_id"`
	PatchDigest  string         `json:"patch_digest"`
	BeforeSubject string        `json:"before_subject"`
	AfterSubject  string        `json:"after_subject"`
	Transformation string       `json:"transformation"`
}

type ObservationRecord struct {
	CaseID       string      `json:"case_id"`
	Side         string      `json:"side"`
	Observation  Observation `json:"observation"`
	ObservationDigest string `json:"observation_digest"`
}

type GuardrailRecord struct {
	CaseID     string                 `json:"case_id"`
	Guardrails []GuardrailObservation `json:"guardrails"`
	AllClosed  bool                   `json:"all_closed"`
	RecordDigest string               `json:"record_digest"`
}

type Claim struct {
	State        string   `json:"state"`
	Stage        string   `json:"stage,omitempty"`
	Step         string   `json:"step,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	UnknownClass string   `json:"unknown_class,omitempty"`
	NextOperation string  `json:"next_operation,omitempty"`
	BlockedBy    []string `json:"blocked_by,omitempty"`
}

func (claim Claim) HasUnknownTuple() bool {
	return claim.State == StateUnknown && claim.Stage != "" && claim.Step != "" && claim.Reason != "" &&
		claim.UnknownClass != "" && claim.NextOperation != "" && len(claim.BlockedBy) > 0
}

type ClaimRecord struct {
	CaseID           string `json:"case_id"`
	Kind             string `json:"kind"`
	Claim            Claim  `json:"claim"`
	BeforeDigest     string `json:"before_digest"`
	AfterDigest      string `json:"after_digest"`
	GuardrailDigest  string `json:"guardrail_digest"`
	ExactComparison  string `json:"exact_comparison"`
	RecordDigest     string `json:"record_digest"`
}

type AdoptionDecision struct {
	CaseID          string `json:"case_id"`
	ClaimState      string `json:"claim_state"`
	Decision        string `json:"decision"`
	Rationale       string `json:"rationale"`
	HumanAuthority  string `json:"human_authority"`
	AppliedToCore   bool   `json:"applied_to_core"`
	RecordDigest    string `json:"record_digest"`
}

type Metrics struct {
	Directories   int `json:"directories"`
	Files         int `json:"files"`
	PhysicalLines int `json:"physical_lines"`
	GoFiles       int `json:"go_files"`
	GoLines       int `json:"go_lines"`
	GoooFiles     int `json:"gooo_files"`
	GoooLines     int `json:"gooo_lines"`
}

type EvaluateOptions struct {
	Source          string
	Contract        string
	IR              string
	Cases           string
	BaselineFixture string
	GeneratedGo     string
	Evaluator       string
	ArtifactDir     string
	ExecutionMode   string
	SubjectSHA      string
	GoVersion       string
	Metrics         Metrics
	Authority       Authority
}
