package reducer

import "sort"

// Decision is the terminal decision in the declared lattice.
type Decision string

const (
	Closed  Decision = "CLOSED"
	Unknown Decision = "UNKNOWN"
	Refuted Decision = "REFUTED"
)

func (d Decision) Priority() int {
	switch d {
	case Refuted:
		return 3
	case Unknown:
		return 2
	case Closed:
		return 1
	default:
		return 0
	}
}

func validDecision(d Decision) bool {
	return d == Closed || d == Unknown || d == Refuted
}

type DecisionReport struct {
	Schema         string                `json:"schema"`
	Scenario       string                `json:"scenario"`
	SourceDigest   string                `json:"source_digest"`
	ContractDigest string                `json:"contract_digest"`
	FixtureDigest  string                `json:"fixture_digest"`
	ToolchainDigest string               `json:"toolchain_digest"`
	RunnerDigest   string                `json:"runner_digest"`
	Decision       Decision              `json:"decision"`
	Priority       int                   `json:"priority"`
	DirectCause    string                `json:"direct_cause,omitempty"`
	Unknown        *UnknownDetails       `json:"unknown,omitempty"`
	Witness        *ContradictionWitness `json:"witness,omitempty"`
}

type UnknownDetails struct {
	Stage            string   `json:"stage"`
	Step             string   `json:"step"`
	Reason           string   `json:"reason"`
	UnknownClass     string   `json:"unknown_class"`
	NextOperation    string   `json:"next_operation"`
	BlockedBy        []string `json:"blocked_by"`
	DirectCause      string   `json:"direct_cause,omitempty"`
	RequiredNodeIDs  []string `json:"required_node_ids,omitempty"`
	RequiredEdgeIDs  []string `json:"required_edge_ids,omitempty"`
	RequiredEvidenceIDs []string `json:"required_evidence_ids,omitempty"`
	RequiredCellIDs  []string `json:"required_cell_ids,omitempty"`
	RequiredStateKeys []string `json:"required_state_keys,omitempty"`
}

func (u *UnknownDetails) Valid() bool {
	return u != nil && u.Stage != "" && u.Step != "" && u.Reason != "" &&
		u.UnknownClass != "" && u.NextOperation != "" && len(u.BlockedBy) > 0
}

type ContradictionWitness struct {
	Claim            string   `json:"claim"`
	Contradiction    string   `json:"contradiction"`
	RequiredNodeIDs  []string `json:"required_node_ids"`
	RequiredEdgeIDs  []string `json:"required_edge_ids"`
	RequiredEvidenceIDs []string `json:"required_evidence_ids"`
	RequiredCellIDs  []string `json:"required_cell_ids"`
	RequiredStateKeys []string `json:"required_state_keys"`
}

func (w *ContradictionWitness) Valid() bool {
	return w != nil && w.Claim != "" && w.Contradiction != ""
}

type Node struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type Edge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

type EvidenceDigest struct {
	ID             string `json:"id"`
	Digest         string `json:"digest"`
	ObservedDigest string `json:"observed_digest"`
	Fresh          bool   `json:"fresh"`
}

type CellDependency struct {
	ID        string   `json:"id"`
	Cell      string   `json:"cell"`
	DependsOn []string `json:"depends_on"`
}

type CausalGraph struct {
	Schema string `json:"schema"`
	Nodes  []Node `json:"nodes"`
	Edges  []Edge `json:"edges"`
}

type OracleObservation struct {
	Available            bool     `json:"available"`
	GraphBounded         bool     `json:"graph_bounded"`
	TieBreakUnique       bool     `json:"tie_break_unique"`
	ReplayStable         bool     `json:"replay_stable"`
	DecisionChangeOnDelete []string `json:"decision_change_on_delete,omitempty"`
	WitnessLossOnDelete  []string `json:"witness_loss_on_delete,omitempty"`
}

type Input struct {
	Schema            string              `json:"schema"`
	DecisionReport    DecisionReport      `json:"decision_report"`
	CausalGraph       CausalGraph         `json:"causal_graph"`
	EvidenceDigests   []EvidenceDigest    `json:"evidence_digests"`
	CellDependencies  []CellDependency   `json:"cell_dependencies"`
	OriginalState     map[string]string   `json:"original_state"`
	Oracle            OracleObservation   `json:"oracle"`
}

type CausalSlice struct {
	Nodes            []Node            `json:"nodes"`
	Edges            []Edge            `json:"edges"`
	EvidenceDigests  []EvidenceDigest `json:"evidence_digests"`
	CellDependencies []CellDependency `json:"cell_dependencies"`
	OriginalState    map[string]string `json:"original_state"`
}

type Provenance struct {
	Scenario        string `json:"scenario"`
	SourceDigest    string `json:"source_digest"`
	ContractDigest  string `json:"contract_digest"`
	FixtureDigest   string `json:"fixture_digest"`
	ToolchainDigest string `json:"toolchain_digest"`
	RunnerDigest    string `json:"runner_digest"`
}

type IntegerVector struct {
	Before int `json:"before"`
	After  int `json:"after"`
}

type Metrics struct {
	Nodes                   IntegerVector `json:"nodes"`
	Edges                   IntegerVector `json:"edges"`
	Evidence                IntegerVector `json:"evidence"`
	CellDependencies        IntegerVector `json:"cell_dependencies"`
	OriginalStateKeys       IntegerVector `json:"original_state_keys"`
	OracleCalls             int           `json:"oracle_calls"`
	WallMS                  int           `json:"wall_ms"`
	PeakRSSKiB              int           `json:"peak_rss_kib"`
	RepositoryWrites        int           `json:"repository_writes"`
	LocalTestExecutions     int           `json:"local_test_executions"`
	CrossProjectRequiredGates int         `json:"cross_project_required_gates"`
}

type Correctness struct {
	DecisionPreserved       bool `json:"decision_preserved"`
	PriorityPreserved       bool `json:"priority_preserved"`
	UnknownFieldsPreserved  bool `json:"unknown_fields_preserved"`
	DirectCausePreserved    bool `json:"direct_cause_preserved"`
	BlockedByPreserved      bool `json:"blocked_by_preserved"`
	WitnessPreserved        bool `json:"witness_preserved"`
	ProvenancePreserved     bool `json:"provenance_preserved"`
	ReplayStable            bool `json:"replay_stable"`
}

type ReplayReceipt struct {
	Run              int      `json:"run"`
	InputDigest      string   `json:"input_digest"`
	CandidateDigest  string   `json:"candidate_digest"`
	ObservedDecision Decision `json:"observed_decision"`
	Stable           bool     `json:"stable"`
	Reason           string   `json:"reason"`
}

type PreservationFailure struct {
	Unit   string `json:"unit"`
	Reason string `json:"reason"`
}

type Minimality struct {
	Status             string `json:"status"`
	Scope              string `json:"scope"`
	GlobalMinimumClaim bool   `json:"global_minimum_claim"`
	AuditPassed        bool   `json:"audit_passed"`
}

type ImprovementReport struct {
	Status           Decision     `json:"status"`
	Reason           string       `json:"reason"`
	ProvenanceTuple  *Provenance  `json:"provenance_tuple"`
	Vectors          any          `json:"vectors"`
}

type InvariantResult struct {
	Ordinal int    `json:"ordinal"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Holds   bool   `json:"holds"`
}

type ProofVectors struct {
	Foundation []InvariantResult `json:"FOUNDATION"`
	Coherence  []InvariantResult `json:"COHERENCE"`
	Regression []InvariantResult `json:"REGRESSION"`
}

type Indicator struct {
	Name         string `json:"name"`
	IntegerValue *int   `json:"integer_value,omitempty"`
	BooleanValue *bool  `json:"boolean_value,omitempty"`
}

type IndicatorDistribution struct {
	Driver    []Indicator `json:"DRIVER"`
	Outcome   []Indicator `json:"OUTCOME"`
	Guardrail []Indicator `json:"GUARDRAIL"`
}

type Result struct {
	Schema                string                 `json:"schema"`
	Scenario              string                 `json:"scenario"`
	Decision              Decision               `json:"decision"`
	Priority              int                    `json:"priority"`
	Unknown               *UnknownDetails        `json:"unknown,omitempty"`
	Witness               *ContradictionWitness  `json:"witness,omitempty"`
	Slice                 CausalSlice            `json:"reduced_slice"`
	Provenance            Provenance             `json:"provenance"`
	Metrics               Metrics                `json:"metrics"`
	Correctness           Correctness            `json:"correctness"`
	Minimality            Minimality             `json:"minimality"`
	Improvement           ImprovementReport      `json:"improvement"`
	ReplayReceipts        []ReplayReceipt       `json:"replay_receipts"`
	PreservationFailures  []PreservationFailure `json:"preservation_failures,omitempty"`
	Proof                 ProofVectors           `json:"proof_vectors"`
	IndicatorDistribution IndicatorDistribution `json:"indicator_distribution"`
}

func cloneStrings(in []string) []string {
	if in == nil {
		return nil
	}
	out := append([]string(nil), in...)
	return out
}

func cloneUnknown(in *UnknownDetails) *UnknownDetails {
	if in == nil {
		return nil
	}
	out := *in
	out.BlockedBy = cloneStrings(in.BlockedBy)
	out.RequiredNodeIDs = cloneStrings(in.RequiredNodeIDs)
	out.RequiredEdgeIDs = cloneStrings(in.RequiredEdgeIDs)
	out.RequiredEvidenceIDs = cloneStrings(in.RequiredEvidenceIDs)
	out.RequiredCellIDs = cloneStrings(in.RequiredCellIDs)
	out.RequiredStateKeys = cloneStrings(in.RequiredStateKeys)
	return &out
}

func cloneWitness(in *ContradictionWitness) *ContradictionWitness {
	if in == nil {
		return nil
	}
	out := *in
	out.RequiredNodeIDs = cloneStrings(in.RequiredNodeIDs)
	out.RequiredEdgeIDs = cloneStrings(in.RequiredEdgeIDs)
	out.RequiredEvidenceIDs = cloneStrings(in.RequiredEvidenceIDs)
	out.RequiredCellIDs = cloneStrings(in.RequiredCellIDs)
	out.RequiredStateKeys = cloneStrings(in.RequiredStateKeys)
	return &out
}

func cloneSlice(in Input) CausalSlice {
	out := CausalSlice{
		Nodes:            append([]Node(nil), in.CausalGraph.Nodes...),
		Edges:            append([]Edge(nil), in.CausalGraph.Edges...),
		EvidenceDigests:  append([]EvidenceDigest(nil), in.EvidenceDigests...),
		CellDependencies: append([]CellDependency(nil), in.CellDependencies...),
		OriginalState:    make(map[string]string, len(in.OriginalState)),
	}
	for key, value := range in.OriginalState {
		out.OriginalState[key] = value
	}
	for index := range out.CellDependencies {
		out.CellDependencies[index].DependsOn = cloneStrings(out.CellDependencies[index].DependsOn)
	}
	return normalizeSlice(out)
}

func normalizeSlice(in CausalSlice) CausalSlice {
	out := in
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].ID < out.Nodes[j].ID })
	sort.Slice(out.Edges, func(i, j int) bool { return out.Edges[i].ID < out.Edges[j].ID })
	sort.Slice(out.EvidenceDigests, func(i, j int) bool { return out.EvidenceDigests[i].ID < out.EvidenceDigests[j].ID })
	sort.Slice(out.CellDependencies, func(i, j int) bool { return out.CellDependencies[i].ID < out.CellDependencies[j].ID })
	for index := range out.CellDependencies {
		sort.Strings(out.CellDependencies[index].DependsOn)
	}
	return out
}

func (s CausalSlice) has(kind, id string) bool {
	switch kind {
	case "node":
		for _, item := range s.Nodes {
			if item.ID == id { return true }
		}
	case "edge":
		for _, item := range s.Edges {
			if item.ID == id { return true }
		}
	case "evidence":
		for _, item := range s.EvidenceDigests {
			if item.ID == id { return true }
		}
	case "cell_dependency":
		for _, item := range s.CellDependencies {
			if item.ID == id { return true }
		}
	case "original_state":
		_, ok := s.OriginalState[id]
		return ok
	}
	return false
}

func (s CausalSlice) without(kind, id string) CausalSlice {
	out := s
	switch kind {
	case "node":
		nodes := make([]Node, 0, len(s.Nodes))
		for _, item := range s.Nodes {
			if item.ID != id { nodes = append(nodes, item) }
		}
		out.Nodes = nodes
		edges := make([]Edge, 0, len(s.Edges))
		for _, item := range s.Edges {
			if item.From != id && item.To != id { edges = append(edges, item) }
		}
		out.Edges = edges
	case "edge":
		edges := make([]Edge, 0, len(s.Edges))
		for _, item := range s.Edges {
			if item.ID != id { edges = append(edges, item) }
		}
		out.Edges = edges
	case "evidence":
		evidence := make([]EvidenceDigest, 0, len(s.EvidenceDigests))
		for _, item := range s.EvidenceDigests {
			if item.ID != id { evidence = append(evidence, item) }
		}
		out.EvidenceDigests = evidence
	case "cell_dependency":
		cells := make([]CellDependency, 0, len(s.CellDependencies))
		for _, item := range s.CellDependencies {
			if item.ID != id { cells = append(cells, item) }
		}
		out.CellDependencies = cells
	case "original_state":
		state := make(map[string]string, len(s.OriginalState))
		for key, value := range s.OriginalState {
			if key != id { state[key] = value }
		}
		out.OriginalState = state
	}
	return normalizeSlice(out)
}

func (s CausalSlice) count(kind string) int {
	switch kind {
	case "node": return len(s.Nodes)
	case "edge": return len(s.Edges)
	case "evidence": return len(s.EvidenceDigests)
	case "cell_dependency": return len(s.CellDependencies)
	case "original_state": return len(s.OriginalState)
	default: return 0
	}
}
