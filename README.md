# gooo-causal-counterexample-reducer

Deterministically reduces an immutable GOOO decision report to a lower-resolution causal slice when self-improvement reaches `UNKNOWN` or `REFUTED`.

The reducer makes one declared, lexical, fixed-order deletion pass. Its contract guarantees a 1-minimal slice for the monotone preservation oracle in that order. It makes no global-minimum claim.

## Contract first

The semantic owner is [.gooo/causal-counterexample-reducer.gooo](./.gooo/causal-counterexample-reducer.gooo). It fixes:

- `REFUTED > UNKNOWN > CLOSED` precedence;
- the six required `UNKNOWN` fields: `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`;
- deletion order, lexical tie-break, two replay calls, and replay receipt fields;
- twelve named invariant conditions in `FOUNDATION`, `COHERENCE`, and `REGRESSION` (4/4/4);
- `DRIVER`, `OUTCOME`, and `GUARDRAIL` indicator vectors (4/4/4); and
- `repository_writes=0`, `local_test_executions=0`, and `cross_project_required_gates=0` at runtime.

The Go implementation evaluates and generates artifacts from that contract. It never mutates the repository and writes only to a caller-owned output directory.

## Input and output

The input JSON is an immutable envelope containing a decision report, causal graph, evidence digests, cell dependencies, original state, and oracle observations. A `REFUTED` report must carry a contradiction witness. An `UNKNOWN` report must carry all six fields, a direct cause, and required IDs for its blocked frontier.

Run one reduction with:

```text
go run ./cmd/gooo-causal-counterexample-reducer \
  -mode reduce \
  -contract .gooo/causal-counterexample-reducer.gooo \
  -input fixtures/cases/normal-reduction.json \
  -output ./caller-owned-output
```

The result contains separate reduction metrics (`nodes`, `edges`, `evidence`, `cell_dependencies`, and `original_state_keys` before/after) and correctness predicates. It also contains replay receipts, the exact provenance tuple, proof vectors, and the `DRIVER`/`OUTCOME`/`GUARDRAIL` indicator distribution. It does not emit aggregate scores or percentages.

Malformed input fails closed as `REFUTED`. Missing oracle capability, stale evidence, an unbounded graph, ambiguous tie-breaks, or unstable replay are staged `UNKNOWN` outcomes with all six fields populated.

## Verification

The fixed denominator and canonical fixtures are exercised in GitHub Actions by `scripts/conformance.sh`. The local validation command count for this delivery is intentionally zero for tests, builds, vet, formatting, shell lint, action lint, jq assertions, generators, and conformance.
Deterministic fixed-order 1-minimal causal slices for UNKNOWN and REFUTED decisions
