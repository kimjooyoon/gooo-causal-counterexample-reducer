package reducer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"
)

type deletionUnit struct {
	Kind string
	ID   string
}

func (u deletionUnit) Key() string { return u.Kind + ":" + u.ID }

func Reduce(input Input, contract Contract) Result {
	started := time.Now()
	base := cloneSlice(input)
	result := Result{
		Schema:         "gooo.causal-counterexample-reducer/result/v1",
		Scenario:       input.DecisionReport.Scenario,
		Decision:       input.DecisionReport.Decision,
		Priority:       input.DecisionReport.Priority,
		Slice:          base,
		Provenance:     provenanceOf(input.DecisionReport),
		ReplayReceipts: []ReplayReceipt{},
	}
	result.Metrics = metricsFor(base, base, 0, started)

	if err := validateInput(input, contract); err != nil {
		result.Decision = Refuted
		result.Priority = Refuted.Priority()
		result.PreservationFailures = []PreservationFailure{{Unit: "input", Reason: "malformed input: " + err.Error()}}
		result.Minimality = Minimality{Status: "REFUTED", Scope: "fixed-order-deletion", GlobalMinimumClaim: contract.GlobalMinimumClaim, AuditPassed: false}
		result.Correctness = Correctness{ProvenancePreserved: false, ReplayStable: false}
		return finalizeResult(result, input, contract, base, 0, started)
	}

	if blocked := preflightUnknown(input, contract); blocked != nil {
		result.Decision = Unknown
		result.Priority = Unknown.Priority()
		result.Unknown = blocked
		result.Minimality = Minimality{Status: "UNKNOWN", Scope: "fixed-order-deletion", GlobalMinimumClaim: contract.GlobalMinimumClaim, AuditPassed: false}
		result.Correctness = correctnessForBlock(input, result)
		return finalizeResult(result, input, contract, base, 0, started)
	}

	current := base
	oracleCalls := 0
	for _, unit := range orderedUnits(base, contract) {
		if !current.has(unit.Kind, unit.ID) {
			continue
		}
		candidate := current.without(unit.Kind, unit.ID)
		receipts := replayCandidate(input, candidate, unit, contract)
		oracleCalls += len(receipts)
		result.ReplayReceipts = append(result.ReplayReceipts, receipts...)
		if preservesCandidate(input, candidate, unit, receipts) {
			current = candidate
		} else {
			result.PreservationFailures = append(result.PreservationFailures, failureFor(input, unit, receipts))
		}
	}

	minimalityPassed := true
	for _, unit := range orderedUnits(current, contract) {
		if !current.has(unit.Kind, unit.ID) {
			continue
		}
		candidate := current.without(unit.Kind, unit.ID)
		receipts := replayCandidate(input, candidate, unit, contract)
		oracleCalls += len(receipts)
		result.ReplayReceipts = append(result.ReplayReceipts, receipts...)
		if preservesCandidate(input, candidate, unit, receipts) {
			minimalityPassed = false
			result.PreservationFailures = append(result.PreservationFailures, PreservationFailure{Unit: unit.Key(), Reason: "final slice still permits a single deletion"})
		}
	}

	result.Slice = current
	result.Unknown = cloneUnknown(input.DecisionReport.Unknown)
	result.Witness = cloneWitness(input.DecisionReport.Witness)
	result.Minimality = Minimality{Status: "PROVEN", Scope: "fixed-order-deletion-1-minimal", GlobalMinimumClaim: false, AuditPassed: minimalityPassed}
	result.Decision = input.DecisionReport.Decision
	result.Priority = input.DecisionReport.Priority
	result.Correctness = correctness(input, result)
	if !minimalityPassed {
		result.Decision = Refuted
		result.Priority = Refuted.Priority()
	}
	if !allRequiredCorrectness(result.Correctness) {
		result.Decision = Refuted
		result.Priority = Refuted.Priority()
		result.PreservationFailures = append(result.PreservationFailures, PreservationFailure{Unit: "final-slice", Reason: "required decision, priority, provenance, UNKNOWN fields, direct cause, frontier, or witness was not preserved"})
	}
	result.Metrics = metricsFor(base, current, oracleCalls, started)
	return finalizeResult(result, input, contract, current, oracleCalls, started)
}

func validateInput(input Input, contract Contract) error {
	if input.Schema != "gooo.causal-counterexample-reducer/input/v1" {
		return fmt.Errorf("unexpected input schema")
	}
	report := input.DecisionReport
	if report.Schema != "gooo.causal-counterexample-reducer/decision-report/v1" {
		return fmt.Errorf("unexpected decision report schema")
	}
	if report.Scenario == "" || report.ContractDigest != contract.ID || report.SourceDigest == "" || report.FixtureDigest == "" || report.ToolchainDigest != contract.ToolchainDigest || report.RunnerDigest == "" {
		return fmt.Errorf("immutable provenance tuple is incomplete or mismatched")
	}
	if !validDecision(report.Decision) || report.Priority != report.Decision.Priority() {
		return fmt.Errorf("decision priority is invalid")
	}
	if report.Decision == Unknown && !report.Unknown.Valid() {
		return fmt.Errorf("UNKNOWN must contain six required fields")
	}
	if report.Decision == Refuted && !report.Witness.Valid() {
		return fmt.Errorf("REFUTED must contain a contradiction witness")
	}
	if input.CausalGraph.Schema != "gooo.causal-counterexample-reducer/causal-graph/v1" {
		return fmt.Errorf("unexpected causal graph schema")
	}
	if input.OriginalState == nil {
		return fmt.Errorf("original state is required")
	}
	if err := validateIDs(input); err != nil {
		return err
	}
	return nil
}

func validateIDs(input Input) error {
	nodeIDs := map[string]bool{}
	for _, node := range input.CausalGraph.Nodes {
		if node.ID == "" || nodeIDs[node.ID] {
			return fmt.Errorf("node IDs must be non-empty and unique")
		}
		nodeIDs[node.ID] = true
	}
	edgeIDs := map[string]bool{}
	for _, edge := range input.CausalGraph.Edges {
		if edge.ID == "" || edgeIDs[edge.ID] {
			return fmt.Errorf("edge IDs must be non-empty and unique")
		}
		if !nodeIDs[edge.From] || !nodeIDs[edge.To] {
			return fmt.Errorf("edge %q has a dangling endpoint", edge.ID)
		}
		edgeIDs[edge.ID] = true
	}
	evidenceIDs := map[string]bool{}
	for _, evidence := range input.EvidenceDigests {
		if evidence.ID == "" || evidence.Digest == "" || evidence.ObservedDigest == "" || evidenceIDs[evidence.ID] {
			return fmt.Errorf("evidence digests must be complete and unique")
		}
		evidenceIDs[evidence.ID] = true
	}
	cellIDs := map[string]bool{}
	for _, cell := range input.CellDependencies {
		if cell.ID == "" || cell.Cell == "" || cellIDs[cell.ID] {
			return fmt.Errorf("cell dependency IDs must be complete and unique")
		}
		cellIDs[cell.ID] = true
	}
	return nil
}

func preflightUnknown(input Input, contract Contract) *UnknownDetails {
	if contract.Oracle.Required && !input.Oracle.Available {
		return stagedUnknown(input, "preflight", "check-oracle-capability", "declared preservation oracle is unavailable", "oracle-unavailable", "provide the immutable preservation oracle", []string{"oracle:" + contract.Oracle.ContractID})
	}
	stale := make([]string, 0)
	for _, evidence := range input.EvidenceDigests {
		if !evidence.Fresh || evidence.Digest != evidence.ObservedDigest {
			stale = append(stale, "evidence:"+evidence.ID)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		return stagedUnknown(input, "preflight", "check-evidence-digests", "evidence digest is stale", "stale-evidence", "refresh and rebind the stale evidence digest", stale)
	}
	itemCount := len(input.CausalGraph.Nodes) + len(input.CausalGraph.Edges) + len(input.EvidenceDigests) + len(input.CellDependencies) + len(input.OriginalState)
	if !input.Oracle.GraphBounded || itemCount > contract.GraphBound {
		return stagedUnknown(input, "preflight", "check-graph-bound", "causal graph exceeds the declared finite bound", "unbounded-graph", "supply a bounded causal graph or a new bounded contract", []string{"graph:bound"})
	}
	if !input.Oracle.TieBreakUnique {
		return stagedUnknown(input, "preflight", "check-tie-break", "deletion candidates do not have a unique declared tie-break", "ambiguous-tie-break", "declare a lexical candidate tie-break", []string{"tie-break:ambiguous"})
	}
	if !input.Oracle.ReplayStable {
		return stagedUnknown(input, "preflight", "check-replay-stability", "preservation replay is not stable", "unstable-replay", "replay the immutable oracle until receipts agree", []string{"oracle:replay"})
	}
	return nil
}

func stagedUnknown(input Input, stage, step, reason, class, next string, blockedBy []string) *UnknownDetails {
	directCause := input.DecisionReport.DirectCause
	if directCause == "" && input.DecisionReport.Unknown != nil {
		directCause = input.DecisionReport.Unknown.DirectCause
	}
	return &UnknownDetails{Stage: stage, Step: step, Reason: reason, UnknownClass: class, NextOperation: next, BlockedBy: cloneStrings(blockedBy), DirectCause: directCause}
}

func orderedUnits(slice CausalSlice, contract Contract) []deletionUnit {
	units := make([]deletionUnit, 0)
	for _, rule := range contract.DeletionOrder {
		ids := make([]string, 0)
		switch rule.Kind {
		case "node":
			for _, item := range slice.Nodes {
				ids = append(ids, item.ID)
			}
		case "edge":
			for _, item := range slice.Edges {
				ids = append(ids, item.ID)
			}
		case "evidence":
			for _, item := range slice.EvidenceDigests {
				ids = append(ids, item.ID)
			}
		case "cell_dependency":
			for _, item := range slice.CellDependencies {
				ids = append(ids, item.ID)
			}
		case "original_state":
			for key := range slice.OriginalState {
				ids = append(ids, key)
			}
		}
		sort.Strings(ids)
		for _, id := range ids {
			units = append(units, deletionUnit{Kind: rule.Kind, ID: id})
		}
	}
	return units
}

func replayCandidate(input Input, candidate CausalSlice, unit deletionUnit, contract Contract) []ReplayReceipt {
	digest := sliceDigest(candidate)
	receipts := make([]ReplayReceipt, 0, contract.Oracle.ReplayCount)
	for run := 1; run <= contract.Oracle.ReplayCount; run++ {
		observed := input.DecisionReport.Decision
		reason := "preservation-oracle"
		if containsString(input.Oracle.DecisionChangeOnDelete, unit.Key()) {
			observed = Closed
			reason = "decision-changed-after-deletion"
		}
		if containsString(input.Oracle.WitnessLossOnDelete, unit.Key()) {
			reason = "contradiction-witness-lost-after-deletion"
		}
		receipts = append(receipts, ReplayReceipt{Run: run, InputDigest: input.DecisionReport.SourceDigest, CandidateDigest: digest, ObservedDecision: observed, Stable: true, Reason: reason})
	}
	return receipts
}

func preservesCandidate(input Input, candidate CausalSlice, unit deletionUnit, receipts []ReplayReceipt) bool {
	for _, receipt := range receipts {
		if receipt.ObservedDecision != input.DecisionReport.Decision || !receipt.Stable {
			return false
		}
	}
	if input.DecisionReport.Decision == Unknown {
		unknown := input.DecisionReport.Unknown
		if !unknown.Valid() || !requiredIDsPresent(candidate, unknown.RequiredNodeIDs, unknown.RequiredEdgeIDs, unknown.RequiredEvidenceIDs, unknown.RequiredCellIDs, unknown.RequiredStateKeys) {
			return false
		}
		if !blockedFrontierPresent(candidate, unknown.BlockedBy) {
			return false
		}
	}
	if input.DecisionReport.Decision == Refuted {
		witness := input.DecisionReport.Witness
		if !witness.Valid() || !requiredIDsPresent(candidate, witness.RequiredNodeIDs, witness.RequiredEdgeIDs, witness.RequiredEvidenceIDs, witness.RequiredCellIDs, witness.RequiredStateKeys) {
			return false
		}
	}
	return true
}

func requiredIDsPresent(slice CausalSlice, nodes, edges, evidence, cells, state []string) bool {
	for _, id := range nodes {
		if !slice.has("node", id) {
			return false
		}
	}
	for _, id := range edges {
		if !slice.has("edge", id) {
			return false
		}
	}
	for _, id := range evidence {
		if !slice.has("evidence", id) {
			return false
		}
	}
	for _, id := range cells {
		if !slice.has("cell_dependency", id) {
			return false
		}
	}
	for _, id := range state {
		if !slice.has("original_state", id) {
			return false
		}
	}
	return true
}

func blockedFrontierPresent(slice CausalSlice, frontier []string) bool {
	for _, entry := range frontier {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}
		kind := parts[0]
		switch kind {
		case "node", "edge", "evidence", "original_state":
		case "cell":
			kind = "cell_dependency"
		default:
			continue
		}
		if !slice.has(kind, parts[1]) {
			return false
		}
	}
	return true
}

func failureFor(input Input, unit deletionUnit, receipts []ReplayReceipt) PreservationFailure {
	for _, receipt := range receipts {
		if receipt.ObservedDecision != input.DecisionReport.Decision {
			return PreservationFailure{Unit: unit.Key(), Reason: "oracle observed a decision change"}
		}
	}
	if input.DecisionReport.Decision == Refuted && containsString(input.Oracle.WitnessLossOnDelete, unit.Key()) {
		return PreservationFailure{Unit: unit.Key(), Reason: "contradiction witness would be lost"}
	}
	if input.DecisionReport.Decision == Unknown {
		return PreservationFailure{Unit: unit.Key(), Reason: "direct cause or blocked_by frontier would be lost"}
	}
	return PreservationFailure{Unit: unit.Key(), Reason: "preservation predicate rejected the deletion"}
}

func correctness(input Input, result Result) Correctness {
	unknownFields := true
	directCause := true
	blockedBy := true
	if input.DecisionReport.Decision == Unknown {
		unknownFields = sameUnknownFields(input.DecisionReport.Unknown, result.Unknown)
		directCause = input.DecisionReport.Unknown != nil && result.Unknown != nil && input.DecisionReport.Unknown.DirectCause == result.Unknown.DirectCause
		blockedBy = input.DecisionReport.Unknown != nil && result.Unknown != nil && reflect.DeepEqual(input.DecisionReport.Unknown.BlockedBy, result.Unknown.BlockedBy)
	}
	witness := true
	if input.DecisionReport.Decision == Refuted {
		witness = input.DecisionReport.Witness.Valid() && result.Witness.Valid() && requiredIDsPresent(result.Slice, result.Witness.RequiredNodeIDs, result.Witness.RequiredEdgeIDs, result.Witness.RequiredEvidenceIDs, result.Witness.RequiredCellIDs, result.Witness.RequiredStateKeys)
	}
	return Correctness{
		DecisionPreserved:      result.Decision == input.DecisionReport.Decision,
		PriorityPreserved:      result.Priority == input.DecisionReport.Priority,
		UnknownFieldsPreserved: unknownFields,
		DirectCausePreserved:   directCause,
		BlockedByPreserved:     blockedBy,
		WitnessPreserved:       witness,
		ProvenancePreserved:    provenanceComplete(result.Provenance),
		ReplayStable:           result.Minimality.Status != "UNKNOWN" || len(result.ReplayReceipts) == 0,
	}
}

func correctnessForBlock(input Input, result Result) Correctness {
	correct := correctness(input, result)
	correct.ReplayStable = false
	return correct
}

func sameUnknownFields(left, right *UnknownDetails) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Stage == right.Stage && left.Step == right.Step && left.Reason == right.Reason && left.UnknownClass == right.UnknownClass && left.NextOperation == right.NextOperation && reflect.DeepEqual(left.BlockedBy, right.BlockedBy)
}

func allRequiredCorrectness(correct Correctness) bool {
	return correct.DecisionPreserved && correct.PriorityPreserved && correct.UnknownFieldsPreserved && correct.DirectCausePreserved && correct.BlockedByPreserved && correct.WitnessPreserved && correct.ProvenancePreserved
}

func provenanceOf(report DecisionReport) Provenance {
	return Provenance{Scenario: report.Scenario, SourceDigest: report.SourceDigest, ContractDigest: report.ContractDigest, FixtureDigest: report.FixtureDigest, ToolchainDigest: report.ToolchainDigest, RunnerDigest: report.RunnerDigest}
}

func provenanceComplete(provenance Provenance) bool {
	return provenance.Scenario != "" && provenance.SourceDigest != "" && provenance.ContractDigest != "" && provenance.FixtureDigest != "" && provenance.ToolchainDigest != "" && provenance.RunnerDigest != ""
}

func metricsFor(before, after CausalSlice, oracleCalls int, started time.Time) Metrics {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	return Metrics{
		Nodes:                     IntegerVector{Before: len(before.Nodes), After: len(after.Nodes)},
		Edges:                     IntegerVector{Before: len(before.Edges), After: len(after.Edges)},
		Evidence:                  IntegerVector{Before: len(before.EvidenceDigests), After: len(after.EvidenceDigests)},
		CellDependencies:          IntegerVector{Before: len(before.CellDependencies), After: len(after.CellDependencies)},
		OriginalStateKeys:         IntegerVector{Before: len(before.OriginalState), After: len(after.OriginalState)},
		OracleCalls:               oracleCalls,
		WallMS:                    int(time.Since(started).Milliseconds()),
		PeakRSSKiB:                int(memory.Sys / 1024),
		RepositoryWrites:          0,
		LocalTestExecutions:       0,
		CrossProjectRequiredGates: 0,
	}
}

func sliceDigest(slice CausalSlice) string {
	data, _ := json.Marshal(normalizeSlice(slice))
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func finalizeResult(result Result, input Input, contract Contract, after CausalSlice, oracleCalls int, started time.Time) Result {
	if result.Metrics.Nodes.Before == 0 && result.Metrics.Edges.Before == 0 && result.Metrics.Evidence.Before == 0 && result.Metrics.CellDependencies.Before == 0 && result.Metrics.OriginalStateKeys.Before == 0 {
		result.Metrics = metricsFor(cloneSlice(input), after, oracleCalls, started)
	} else {
		result.Metrics.OracleCalls = oracleCalls
		result.Metrics.WallMS = int(time.Since(started).Milliseconds())
	}
	result.Proof = buildProof(result, input, contract)
	result.IndicatorDistribution = buildIndicators(result)
	result.Improvement = ImprovementReport{Status: Unknown, Reason: "a paired baseline with the exact provenance tuple is required", ProvenanceTuple: nil, Vectors: nil}
	return result
}

func buildProof(result Result, input Input, contract Contract) ProofVectors {
	holds := map[string]bool{
		"F01": input.DecisionReport.SourceDigest != "",
		"F02": result.Metrics.RepositoryWrites == 0,
		"F03": len(contract.DeletionOrder) == 5 && contract.TieBreak == "lexical_id",
		"F04": result.Decision == Refuted || result.Decision == Unknown || result.Decision == Closed,
		"C01": sameDecisions(contract.DecisionPrecedence, []Decision{Refuted, Unknown, Closed}),
		"C02": result.Decision != Unknown || result.Unknown.Valid(),
		"C03": result.Correctness.WitnessPreserved && (result.Decision != Unknown || result.Correctness.BlockedByPreserved),
		"C04": result.Minimality.Status == "UNKNOWN" || len(result.ReplayReceipts) > 0,
		"R01": provenanceComplete(result.Provenance),
		"R02": result.Metrics.Nodes.Before >= result.Metrics.Nodes.After && result.Metrics.Edges.Before >= result.Metrics.Edges.After && result.Metrics.Evidence.Before >= result.Metrics.Evidence.After && result.Metrics.CellDependencies.Before >= result.Metrics.CellDependencies.After,
		"R03": !contract.GlobalMinimumClaim,
		"R04": result.Metrics.RepositoryWrites == 0 && result.Metrics.LocalTestExecutions == 0 && result.Metrics.CrossProjectRequiredGates == 0,
	}
	proof := ProofVectors{}
	for _, invariant := range contract.Invariants {
		entry := InvariantResult{Ordinal: invariant.Ordinal, ID: invariant.ID, Name: invariant.Name, Holds: holds[invariant.ID]}
		switch invariant.Group {
		case "FOUNDATION":
			proof.Foundation = append(proof.Foundation, entry)
		case "COHERENCE":
			proof.Coherence = append(proof.Coherence, entry)
		case "REGRESSION":
			proof.Regression = append(proof.Regression, entry)
		}
	}
	return proof
}

func buildIndicators(result Result) IndicatorDistribution {
	integer := func(value int) *int { return &value }
	boolean := func(value bool) *bool { return &value }
	return IndicatorDistribution{
		Driver: []Indicator{
			{Name: "nodes_removed", IntegerValue: integer(result.Metrics.Nodes.Before - result.Metrics.Nodes.After)},
			{Name: "edges_removed", IntegerValue: integer(result.Metrics.Edges.Before - result.Metrics.Edges.After)},
			{Name: "evidence_removed", IntegerValue: integer(result.Metrics.Evidence.Before - result.Metrics.Evidence.After)},
			{Name: "cell_dependencies_removed", IntegerValue: integer(result.Metrics.CellDependencies.Before - result.Metrics.CellDependencies.After)},
		},
		Outcome: []Indicator{
			{Name: "decision_preserved", BooleanValue: boolean(result.Correctness.DecisionPreserved)},
			{Name: "priority_preserved", BooleanValue: boolean(result.Correctness.PriorityPreserved)},
			{Name: "unknown_fields_preserved", BooleanValue: boolean(result.Correctness.UnknownFieldsPreserved)},
			{Name: "witness_preserved", BooleanValue: boolean(result.Correctness.WitnessPreserved)},
		},
		Guardrail: []Indicator{
			{Name: "oracle_calls", IntegerValue: integer(result.Metrics.OracleCalls)},
			{Name: "wall_ms", IntegerValue: integer(result.Metrics.WallMS)},
			{Name: "peak_rss_kib", IntegerValue: integer(result.Metrics.PeakRSSKiB)},
			{Name: "repository_writes", IntegerValue: integer(result.Metrics.RepositoryWrites)},
		},
	}
}

func sameDecisions(left, right []Decision) bool {
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
