#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
	printf 'usage: conformance.sh PATH_TO_BINARY GO_VERSION\n' >&2
	exit 64
fi

bin=$(realpath "$1")
go_version=$2
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
subject_sha=${SUBJECT_SHA:?SUBJECT_SHA is required}
work=${RUNNER_TEMP:?RUNNER_TEMP is required}/gooo-counterfactual-change-work
isolated="$work/isolated"
first="$work/first"
second="$work/second"
mkdir -p "$isolated" "$first" "$second"

git -C "$root" archive --format=tar "$subject_sha" | tar -x -C "$isolated"
copy="$isolated"

count_lines() {
	local total=0
	local path
	while IFS= read -r path; do
		total=$((total + $(wc -l < "$path")))
	done
	echo "$total"
}

file_list=$(mktemp)
find "$copy" -type f -not -path "$copy/README.md" -print | sort > "$file_list"
files=$(wc -l < "$file_list" | tr -d ' ')
directories=$(find "$copy" -type d -print | wc -l | tr -d ' ')
physical_lines=$(count_lines < "$file_list")
go_file_list=$(mktemp)
find "$copy" -type f -name '*.go' -print | sort > "$go_file_list"
go_files=$(wc -l < "$go_file_list" | tr -d ' ')
go_lines=$(count_lines < "$go_file_list")
gooo_file_list=$(mktemp)
find "$copy" -type f -name '*.gooo' -print | sort > "$gooo_file_list"
gooo_files=$(wc -l < "$gooo_file_list" | tr -d ' ')
gooo_lines=$(count_lines < "$gooo_file_list")

source="$copy/examples/counterfactual-change/main.gooo"
contract="$copy/contracts/counterfactual-denominator-v1.json"
cases="$copy/fixtures/scenarios.json"
baseline="$copy/fixtures/baseline-identity.json"
generated="$copy/generated/evaluator.go"
evaluator="$copy/scripts/conformance.sh"

before_status=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')

run_one() {
	local destination=$1
	mkdir -p "$destination/artifacts"
	"$bin" compile --source "$source" --contract "$contract" --ir "$destination/semantic-ir.json"
	"$bin" evaluate \
		--source "$source" \
		--contract "$contract" \
		--ir "$destination/semantic-ir.json" \
		--cases "$cases" \
		--baseline-fixture "$baseline" \
		--generated-go "$generated" \
		--evaluator "$evaluator" \
		--artifact-dir "$destination/artifacts" \
		--execution-mode EPHEMERAL_CI_COPY \
		--subject-sha "$subject_sha" \
		--go-version "$go_version" \
		--directories "$directories" \
		--files "$files" \
		--physical-lines "$physical_lines" \
		--go-files "$go_files" \
		--go-lines "$go_lines" \
		--gooo-files "$gooo_files" \
		--gooo-lines "$gooo_lines" \
		--repository-writes 0 \
		--local-test-executions 0 \
		--cross-project-required-gates 0
}

run_one "$first"
after_first_status=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before_status" = "$after_first_status"

run_one "$second"
after_second_status=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before_status" = "$after_second_status"

for name in $(find "$first/artifacts" -maxdepth 1 -type f -printf '%f\n' | sort); do
	test -f "$second/artifacts/$name"
	cmp -s "$first/artifacts/$name" "$second/artifacts/$name"
done
test "$(find "$first/artifacts" -maxdepth 1 -type f | wc -l | tr -d ' ')" = 14
test "$(wc -l < "$first/artifacts/claims.ndjson" | tr -d ' ')" = 12
test "$(wc -l < "$first/artifacts/before-observations.ndjson" | tr -d ' ')" = 12
test "$(wc -l < "$first/artifacts/after-observations.ndjson" | tr -d ' ')" = 12
test "$(wc -l < "$first/artifacts/guardrail-outcomes.ndjson" | tr -d ' ')" = 12

jq -e '
	.schema == "gooo/counterfactual-change/experiment-manifest/v1" and
	.contracts.activity_cells == 15 and
	.contracts.executable_cases == 12 and
	.contracts.fixed_guardrails == 4 and
	.contracts.activity_mapping == "1:1" and
	.contracts.root_readme_excluded == true and
	.case_states == {"CLOSED":4,"UNKNOWN":4,"REFUTED":4} and
	.adoption_decisions == {"ADOPTABLE":4,"HOLD_UNKNOWN":4,"REJECTED":4} and
	.precedence == ["REFUTED","UNKNOWN","CLOSED"] and
	.authority == {"repository_writes":0,"local_test_executions":0,"cross_project_required_gates":0} and
	.inventory.root_readme_excluded == true and
	.core_adoption.performed == false and
	.execution_mode == "EPHEMERAL_CI_COPY"
' "$first/artifacts/experiment-manifest.json" >/dev/null

jq -e -s '
	length == 12 and
	([.[] | select(.claim.state == "UNKNOWN" and (.claim.stage|length)>0 and (.claim.step|length)>0 and (.claim.reason|length)>0 and (.claim.unknown_class|length)>0 and (.claim.next_operation|length)>0 and (.claim.blocked_by|type)=="array" and (.claim.blocked_by|length)>0)] | length) == 4 and
	([.[] | select(.claim.state == "REFUTED")] | length) == 4 and
	([.[] | select(.claim.state == "CLOSED")] | length) == 4
' "$first/artifacts/claims.ndjson" >/dev/null
jq -e -s '([.[] | select(.case_id == "refuted-04" and .claim.state == "REFUTED" and .exact_comparison == "KNOWN_REGRESSION")] | length) == 1' "$first/artifacts/claims.ndjson" >/dev/null
jq -e -s 'length == 12 and all(.[]; .applied_to_core == false and .human_authority == "HUMAN_REVIEW_ONLY")' "$first/artifacts/adoption-decisions.ndjson" >/dev/null
jq -e '.candidate_id == "candidate-explicit-result" and (.patch_digest|startswith("sha256:")) and (.patch_text|contains("require exact_claimed_outcome=CLAIMED_SUCCESS")) and .execution_boundary == "CALLER_OWNED_EPHEMERAL_CI_COPY"' "$first/artifacts/candidate.patch.json" >/dev/null
jq -e '.execution_mode == "EPHEMERAL_CI_COPY" and .candidate_applied_to_source_repository == false and .source_repository_writes == 0 and .local_test_executions == 0 and .cross_project_required_gates == 0' "$first/artifacts/execution-receipt.json" >/dev/null
jq -e '.deterministic_replay_required == true and .second_run_byte_identical == true and .replay_alone_can_close == false' "$first/artifacts/replay-receipt.json" >/dev/null

if grep -R -n -E -i 'percentage|percent|score|cache-hit|inferred improvement' "$first/artifacts"; then
	echo "forbidden aggregate or inferred-claim output" >&2
	exit 1
fi

forbidden="$root/.gooo-counterfactual-change-forbidden-output"
forbidden_ir="$work/forbidden-ir.json"
if "$bin" evaluate \
	--source "$root/examples/counterfactual-change/main.gooo" \
	--contract "$root/contracts/counterfactual-denominator-v1.json" \
	--ir "$forbidden_ir" \
	--cases "$root/fixtures/scenarios.json" \
	--baseline-fixture "$root/fixtures/baseline-identity.json" \
	--generated-go "$root/generated/evaluator.go" \
	--evaluator "$root/scripts/conformance.sh" \
	--artifact-dir "$forbidden" \
	--execution-mode EPHEMERAL_CI_COPY \
	--subject-sha "$subject_sha" \
	--go-version "$go_version" \
	--repository-writes 0 \
	--local-test-executions 0 \
	--cross-project-required-gates 0; then
	echo "repository-owned output was accepted" >&2
	exit 1
fi
test ! -e "$forbidden"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
	cat "$first/artifacts/counterfactual-report.md" >> "$GITHUB_STEP_SUMMARY"
fi

echo "counterfactual conformance passed: activities=15 cases=12 closed=4 unknown=4 refuted=4 artifacts=14 replay=byte-identical"
