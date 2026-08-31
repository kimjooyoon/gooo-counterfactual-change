# Execution contract

The GitHub Actions workflow is the only development validation authority for
this repository. It performs Go 1.27 formatting, vetting, tests, build, and
counterfactual conformance in CI.

The conformance script:

1. creates a source archive in the runner temporary directory;
2. invokes the binary from that ephemeral copy;
3. sends semantic IR and all generated artifacts to separate caller-owned
   temporary directories;
4. checks the source checkout before and after evaluation;
5. repeats evaluation and compares every artifact byte-for-byte;
6. checks exact activity, case, guardrail, state, decision, authority, and
   inventory counts; and
7. uploads the first machine and human evidence bundle.

No candidate patch is applied to the checked-out repository. The root README
is excluded only from inventory counts, not from the released source archive.
