# Gooo Counterfactual Change

Gooo Counterfactual Change is an independent protocol for testing a proposed
language change before a human decides whether it is adoptable. A proposal is
not evidence by itself: the protocol binds the proposal to an immutable
baseline, runs the candidate in an ephemeral CI copy, compares exact integer
observations, evaluates fixed guardrails, and records the human decision.

The source-to-evidence chain is:

```text
.gooo source → semantic IR → generated Go evaluator → machine receipts → human report
```

The released example contains a fixed 15-cell activity denominator and an
executable corpus of exactly 12 cases: 4 `CLOSED`, 4 `UNKNOWN`, and 4
`REFUTED`. The four fixed guardrails are baseline immutability, candidate
isolation, zero local authority, and a fixed resource budget.

Resolution is `REFUTED > UNKNOWN > CLOSED`. A candidate is `ADOPTABLE` only
when the exact claimed outcome closes and all fixed guardrails are `CLOSED`.
Missing comparable before/after observations produce `UNKNOWN` with
`stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`.
A known contradiction or regression is `REFUTED`, including when another
observation is missing.

The command writes 14 machine and human artifacts only beneath an empty,
caller-owned output directory. It rejects output inside the source
repository. It never edits or applies a candidate to the source repository;
the candidate artifact is a useful Gooo-fragment proposal for human review.

## Example

The user-facing declaration is
[`main.gooo`](/Users/alice/meta-go/gooo-counterfactual-change/examples/counterfactual-change/main.gooo).
The CI workflow compiles it to semantic IR, generates the candidate patch, and
evaluates the normal, unknown, and refuted fixtures from an ephemeral copy.

The report always records exact counts and integer build, test, resource, and
authority observations. Repository writes, local test executions, and
cross-project required gates are all fixed at zero during evaluation. Other
projects are not required inputs, and no result is adopted into a core
repository.

## Validation

GitHub Actions is the validation authority and uses Go 1.27 for formatting,
vetting, testing, building, conformance, deterministic replay, and artifact
upload. The project-root `README.md` is explicitly excluded from the
repository inventory used by the conformance receipt.
