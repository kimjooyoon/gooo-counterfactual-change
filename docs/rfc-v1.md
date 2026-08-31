# Counterfactual change protocol v1

## Purpose

A language implementation may propose a self-change without establishing that
the change is safe. This protocol makes the proposal a counterfactual
experiment: the candidate is generated as a caller-owned artifact, executed in
an ephemeral CI copy, and judged against exact before/after observations.

## Lifecycle

The 15 fixed activities are grouped into five phases:

1. `BASELINE`: bind, digest, and freeze the immutable baseline.
2. `CLAIM`: declare one explicit change claim and its exact expected outcome.
3. `CANDIDATE`: generate and bind a candidate transformation without editing
   the source repository.
4. `EXECUTION`: allocate an ephemeral CI copy and record comparable before and
   after build, test, and resource observations.
5. `DECISION`: resolve guardrails and record a human adoption decision.

The `.gooo` source and JSON contract must agree cell-for-cell, including
ordinal, phase, input, output, and semantic edge. The compiler emits a
semantic IR with source and contract digests. The checked-in generated
evaluator evaluates the exact claimed result; the parent evaluator then
checks bindings, guardrails, observations, and decisions.

## State semantics

Every claim resolves to exactly one of `CLOSED`, `UNKNOWN`, or `REFUTED`.
Resolution precedence is `REFUTED > UNKNOWN > CLOSED`. A known regression,
failed build, failed test, open guardrail, or exact-claim contradiction is
`REFUTED`. If no known contradiction exists but a comparable after observation
is missing, the claim is `UNKNOWN` and must carry the six-field frontier tuple.
Only exact closure with every fixed guardrail closed yields human decision
`ADOPTABLE`; `UNKNOWN` yields `HOLD_UNKNOWN`, and `REFUTED` yields `REJECTED`.

## Authority and boundaries

The evaluator records `repository_writes=0`, `local_test_executions=0`, and
`cross_project_required_gates=0`. These values are checked against both the
source declaration and command-line observations. Generated output must be in
an empty directory outside the source repository. The conformance workflow
creates an ephemeral copy from the checked-out revision before invoking the
chain and checks that the original checkout remains unchanged.

## Replay

The same immutable inputs, subject revision, Go version, and inventory are
used for two executions. Machine artifacts and the human report must be byte
identical. Replay confirms determinism; replay alone cannot close a claim.

## Adoption boundary

The protocol records a human decision but has no operation that writes the
candidate to a core language repository. A positive decision is a review
result, not an authorization to mutate the protected baseline.
