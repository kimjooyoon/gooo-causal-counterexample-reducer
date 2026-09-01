package reducer

import "time"

// FailClosedResult turns an unreadable input into an explicit REFUTED result.
// It is intentionally independent of any external utility or repository state.
func FailClosedResult(contract Contract, scenario, reason string) Result {
	started := time.Now()
	result := Result{
		Schema:   "gooo.causal-counterexample-reducer/result/v1",
		Scenario: scenario,
		Decision: Refuted,
		Priority: Refuted.Priority(),
		Slice: CausalSlice{Nodes: []Node{}, Edges: []Edge{}, EvidenceDigests: []EvidenceDigest{}, CellDependencies: []CellDependency{}, OriginalState: map[string]string{}},
		Minimality: Minimality{Status: "REFUTED", Scope: "fixed-order-deletion", GlobalMinimumClaim: contract.GlobalMinimumClaim, AuditPassed: false},
		PreservationFailures: []PreservationFailure{{Unit: "input", Reason: "malformed input: " + reason}},
	}
	result.Metrics = metricsFor(result.Slice, result.Slice, 0, started)
	result.Correctness = Correctness{DecisionPreserved: false, PriorityPreserved: false, UnknownFieldsPreserved: false, DirectCausePreserved: false, BlockedByPreserved: false, WitnessPreserved: false, ProvenancePreserved: false, ReplayStable: false}
	return finalizeResult(result, Input{DecisionReport: DecisionReport{Scenario: scenario}}, contract, result.Slice, 0, started)
}
