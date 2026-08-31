# Counterfactual change protocol

This repository is an independent, append-only counterfactual experiment
laboratory.

- The protected baseline is immutable input. The evaluator never edits it and
  never applies a candidate patch to the source repository.
- Candidate patches and all generated receipts are caller-owned outputs. The
  conformance workflow executes from an ephemeral CI copy and writes only to
  the runner temporary directory.
- GitHub Actions is the validation authority. Local Go build, test, vet,
  formatting, compiler, harness, and conformance commands are intentionally
  not run during development.
- Claims have exactly three states: `CLOSED`, `UNKNOWN`, and `REFUTED`, with
  `REFUTED > UNKNOWN > CLOSED` precedence. An `UNKNOWN` claim carries the six
  required frontier fields: `stage`, `step`, `reason`, `unknown_class`,
  `next_operation`, and `blocked_by`.
- The fixed denominator is explicit. Counts are integers and observations are
  exact; there are no percentages, scores, inferred improvement values, or
  cache-hit proofs.
- No result is adopted into a core language repository by this project.
