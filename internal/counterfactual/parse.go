package counterfactual

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ParseSource(path string) (SourceDecl, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SourceDecl{}, err
	}
	decl := SourceDecl{SourceDigest: DigestBytes(data)}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "gooo":
			if len(fields) != 3 || fields[1] != "counterfactual_change" || fields[2] != "v1" {
				return SourceDecl{}, fmt.Errorf("line %d: invalid gooo header", lineNumber)
			}
			decl.Schema = SourceSchema
			decl.Version = fields[2]
		case "experiment":
			values, err := keyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.ExperimentID = values["id"]
			decl.BaselineID = values["baseline_id"]
			decl.ClaimID = values["claim_id"]
			decl.CandidateID = values["candidate_id"]
		case "denominator":
			values, err := keyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.DenominatorID = values["id"]
			decl.CellCount, err = integer(values, "cell_count")
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.CaseCount, err = integer(values, "case_count")
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
		case "authority":
			values, err := keyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Authority.RepositoryWrites, err = integer(values, "repository_writes")
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Authority.LocalTestExecutions, err = integer(values, "local_test_executions")
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Authority.CrossProjectRequiredGates, err = integer(values, "cross_project_required_gates")
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
		case "precedence":
			if len(fields) != 2 {
				return SourceDecl{}, fmt.Errorf("line %d: invalid precedence", lineNumber)
			}
			decl.Precedence = strings.Split(fields[1], ">")
		case "unknown_fields":
			if len(fields) != 2 {
				return SourceDecl{}, fmt.Errorf("line %d: invalid unknown_fields", lineNumber)
			}
			decl.UnknownFields = strings.Split(fields[1], ",")
		case "change_claim":
			values, err := keyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Claim = ChangeClaim{
				ID: values["id"], ExpectedBefore: values["expected_before"],
				ExpectedAfter: values["expected_after"], Exact: values["exact"] == "true",
			}
		case "candidate":
			values, err := keyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Candidate = CandidateDecl{
				ID: values["id"], Operation: values["operation"],
				Target: values["target"], PatchFormat: values["patch_format"],
			}
		case "guardrail":
			values, err := keyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Guardrails = append(decl.Guardrails, values["id"])
		case "activity":
			values, err := keyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			ordinal, err := integer(values, "ordinal")
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Activities = append(decl.Activities, Activity{
				Ordinal: ordinal, ID: values["id"], Phase: values["phase"],
				Input: values["input"], Output: values["output"], Edge: values["edge"],
			})
		default:
			return SourceDecl{}, fmt.Errorf("line %d: unknown declaration %q", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return SourceDecl{}, err
	}
	return decl, nil
}

func keyValues(fields []string) (map[string]string, error) {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid key/value %q", field)
		}
		values[parts[0]] = strings.Trim(parts[1], "\"")
	}
	return values, nil
}

func integer(values map[string]string, key string) (int, error) {
	value, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("missing %s", key)
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", key, value)
	}
	return number, nil
}

func LoadContract(path string) (Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, err
	}
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return Contract{}, fmt.Errorf("decode contract: %w", err)
	}
	if contract.Schema != ContractSchema {
		return Contract{}, fmt.Errorf("unexpected contract schema %q", contract.Schema)
	}
	return contract, nil
}

func ValidateDeclarations(source SourceDecl, contract Contract) error {
	if source.Schema != SourceSchema || source.Version != "v1" || contract.Version != "v1" ||
		source.DenominatorID != contract.ID || source.CellCount != ActivityCount ||
		source.CaseCount != CaseCount || contract.CellCount != ActivityCount ||
		contract.CaseCount != CaseCount || !contract.Fixed {
		return fmt.Errorf("fixed denominator declaration mismatch")
	}
	if source.ExperimentID == "" || source.BaselineID == "" || source.ClaimID == "" || source.CandidateID == "" {
		return fmt.Errorf("experiment identity is incomplete")
	}
	if source.Claim.ID != source.ClaimID || source.Claim.ExpectedBefore == "" || source.Claim.ExpectedAfter == "" || !source.Claim.Exact {
		return fmt.Errorf("exact change claim is incomplete")
	}
	if source.Candidate.ID != source.CandidateID || source.Candidate.Operation == "" || source.Candidate.Target == "" || source.Candidate.PatchFormat == "" {
		return fmt.Errorf("candidate declaration is incomplete")
	}
	if !sameStrings(source.UnknownFields, []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}) {
		return fmt.Errorf("UNKNOWN six-field contract mismatch")
	}
	if !sameStrings(source.Precedence, []string{"REFUTED", "UNKNOWN", "CLOSED"}) {
		return fmt.Errorf("resolution precedence mismatch")
	}
	if !sameStrings(source.Guardrails, contract.Guardrails) || len(source.Guardrails) != GuardrailCount {
		return fmt.Errorf("fixed guardrail set mismatch")
	}
	if source.Authority != (Authority{}) {
		return fmt.Errorf("authority declaration must be zero")
	}
	if len(source.Activities) != ActivityCount || len(contract.Activities) != ActivityCount {
		return fmt.Errorf("expected exactly %d activities", ActivityCount)
	}
	for index := 0; index < ActivityCount; index++ {
		left, right := source.Activities[index], contract.Activities[index]
		if left.Ordinal != index+1 || right.Ordinal != index+1 || left.ID == "" ||
			left.ID != right.ID || left.Phase != right.Phase || left.Input != right.Input ||
			left.Output != right.Output || left.Edge != right.Edge {
			return fmt.Errorf("activity %d does not match fixed contract", index+1)
		}
	}
	return nil
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

func findRepoRoot(start string) string {
	current, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for {
		if info, err := os.Stat(filepath.Join(current, ".git")); err == nil && info != nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func ensureExternalPath(path, sourcePath string) error {
	if path == "" {
		return fmt.Errorf("caller-owned output path is required")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	root := findRepoRoot(filepath.Dir(sourcePath))
	if root != "" && pathWithin(root, absPath) {
		return fmt.Errorf("caller-owned output must be outside the source repository")
	}
	return nil
}

func ensureEmptyOutput(path, sourcePath string) error {
	if err := ensureExternalPath(path, sourcePath); err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("caller-owned artifact directory must start empty")
	}
	return nil
}
