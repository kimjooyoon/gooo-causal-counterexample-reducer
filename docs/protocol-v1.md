# Causal counterexample reduction protocol v1

## Boundary

The caller supplies an immutable `gooo.causal-counterexample-reducer/input/v1` envelope. It contains five source portions: a decision report, causal graph, evidence digests, cell dependencies, and original state. The report also carries the scenario/source/contract/fixture/toolchain/runner provenance tuple.

The reducer clones those portions into a caller-owned output slice. It never writes to the repository, mutates the input, commits, pushes, merges, tags, or releases.

## Decision preservation

The decision lattice is `REFUTED > UNKNOWN > CLOSED`.

For `REFUTED`, the contradiction witness is part of the preservation oracle. Every required witness node, edge, evidence digest, cell, and state key must remain present. A candidate that changes the decision or loses the witness is rejected and recorded; the final result remains `REFUTED`.

For `UNKNOWN`, the six fields `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by` are copied exactly. The direct cause and required IDs for the blocked frontier are also preserved. A dependency-blocked frontier is represented by named `blocked_by` entries such as `cell:cell-provenance` and `evidence:ev-provenance`.

If the final slice cannot preserve the original decision, priority, provenance, required UNKNOWN fields, direct cause, blocked frontier, or contradiction witness, the reducer fails closed as `REFUTED`.

## Deterministic deletion

The `.gooo` contract declares the only deletion order: nodes, edges, evidence digests, cell dependencies, then original-state keys. IDs are sorted lexically within each category. Each candidate is evaluated by the declared two-replay oracle. A candidate is committed only when both replay receipts agree and the preservation predicate holds.

The preservation predicate is monotone for the declared fixtures. The one-pass fixed-order algorithm therefore returns a 1-minimal slice in the contract's order: no remaining single deletion accepted by the same oracle can be made. This is not a global-minimum claim. The report explicitly carries `global_minimum_claim=false`.

## Staged UNKNOWN outcomes

Oracle absence, stale evidence, an unbounded graph, ambiguous tie-breaks, and unstable replay cannot be safely reduced. Each produces `UNKNOWN` at the stage that detected it, with all six required fields and a next operation. The reducer does not pretend that an unevaluable slice is minimal.

Malformed JSON, missing required structures, duplicate IDs, dangling graph references, invalid precedence, or missing provenance fails closed as `REFUTED`.

## Evidence

Reports include integer before/after vectors for nodes, edges, evidence, cell dependencies, and original-state keys. Oracle calls, wall time, and peak RSS are independent guardrail metrics. Reduction is reported separately from correctness preservation. The denominator has exactly twelve named invariants, partitioned 4/4/4 into `FOUNDATION`, `COHERENCE`, and `REGRESSION`; the indicator distribution has 4/4/4 entries in `DRIVER`, `OUTCOME`, and `GUARDRAIL`.

An improvement claim is `UNKNOWN` with `vectors=null` unless a paired baseline has exactly the same scenario, source, contract, fixture, toolchain, and runner digests. The reducer's own before/after slice vectors are not silently promoted to a cross-run improvement claim.
