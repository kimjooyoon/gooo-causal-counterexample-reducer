package reducer

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Invariant struct {
	Ordinal int
	Group   string
	ID      string
	Name    string
}

type Vector struct {
	Name       string
	Indicators []string
}

type DeletionRule struct {
	Ordinal int
	Kind    string
	Target  string
}

type StateTransition struct {
	Ordinal int
	From    string
	To      string
	Evidence string
}

type Scenario struct {
	Ordinal  int
	ID       string
	Fixture  string
	Expected Decision
}

type OracleContract struct {
	Name         string
	Required     bool
	ContractID   string
	ReplayCount  int
}

type Contract struct {
	ID                    string
	Version               string
	ToolchainDigest       string
	DecisionPrecedence    []Decision
	UnknownFields         []string
	DeletionOrder         []DeletionRule
	TieBreak              string
	GraphBound            int
	Oracle                OracleContract
	ReplayReceiptFields   []string
	GlobalMinimumClaim    bool
	PredicateMonotone     bool
	RuntimeRepositoryWrites int
	RuntimeLocalTests     int
	RuntimeCrossProjectGates int
	Invariants            []Invariant
	Vectors               []Vector
	Transitions           []StateTransition
	PreservationFields    []string
	Scenarios             []Scenario
}

func LoadContract(path string) (Contract, error) {
	file, err := os.Open(path)
	if err != nil {
		return Contract{}, err
	}
	defer file.Close()

	var contract Contract
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "gooo" {
			if len(fields) < 3 {
				return Contract{}, fmt.Errorf("line %d: invalid header", lineNumber)
			}
			contract.Version = fields[2]
			continue
		}
		if fields[0] == "precedence" {
			if len(fields) != 2 {
				return Contract{}, fmt.Errorf("line %d: precedence must be a single ordered value", lineNumber)
			}
			contract.DecisionPrecedence = parseDecisions(fields[1])
			continue
		}
		if fields[0] == "unknown_fields" {
			if len(fields) != 2 {
				return Contract{}, fmt.Errorf("line %d: unknown_fields must be a single ordered value", lineNumber)
			}
			contract.UnknownFields = splitList(fields[1])
			continue
		}
		attrs, parseErr := parseAttributes(fields[1:])
		if parseErr != nil {
			return Contract{}, fmt.Errorf("line %d: %w", lineNumber, parseErr)
		}
		get := func(key string) string { return attrs[key] }
		integer := func(key string) (int, error) {
			value, parseErr := strconv.Atoi(get(key))
			if parseErr != nil {
				return 0, fmt.Errorf("line %d: %s must be an integer", lineNumber, key)
			}
			return value, nil
		}
		switch fields[0] {
		case "denominator":
			contract.ID = get("id")
		case "decision":
			continue
		case "toolchain":
			contract.ToolchainDigest = get("digest")
		case "deletion_order":
			ordinal, parseErr := integer("ordinal")
			if parseErr != nil { return Contract{}, parseErr }
			contract.DeletionOrder = append(contract.DeletionOrder, DeletionRule{Ordinal: ordinal, Kind: get("kind"), Target: get("target")})
		case "tie_break":
			contract.TieBreak = get("kind")
		case "graph_bound":
			bound, parseErr := integer("max_items")
			if parseErr != nil { return Contract{}, parseErr }
			contract.GraphBound = bound
		case "oracle":
			replays, parseErr := integer("replay_count")
			if parseErr != nil { return Contract{}, parseErr }
			required, parseErr := strconv.ParseBool(get("required"))
			if parseErr != nil { return Contract{}, fmt.Errorf("line %d: required must be boolean", lineNumber) }
			contract.Oracle = OracleContract{Name: get("name"), Required: required, ContractID: get("contract"), ReplayCount: replays}
		case "replay_receipt":
			if len(fields) > 1 { contract.ReplayReceiptFields = splitList(fields[1]) }
		case "claim":
			global, parseErr := strconv.ParseBool(get("global_minimum"))
			if parseErr != nil { return Contract{}, fmt.Errorf("line %d: global_minimum must be boolean", lineNumber) }
			monotone, parseErr := strconv.ParseBool(get("predicate"))
			if parseErr == nil {
				contract.PredicateMonotone = monotone
			} else {
				contract.PredicateMonotone = get("predicate") == "monotone"
			}
			contract.GlobalMinimumClaim = global
		case "preservation":
			if len(fields) > 1 { contract.PreservationFields = splitList(fields[1]) }
		case "runtime":
			var parseErr error
			contract.RuntimeRepositoryWrites, parseErr = integer("repository_writes")
			if parseErr != nil { return Contract{}, parseErr }
			contract.RuntimeLocalTests, parseErr = integer("local_test_executions")
			if parseErr != nil { return Contract{}, parseErr }
			contract.RuntimeCrossProjectGates, parseErr = integer("cross_project_required_gates")
			if parseErr != nil { return Contract{}, parseErr }
		case "state_transition":
			ordinal, parseErr := integer("ordinal")
			if parseErr != nil { return Contract{}, parseErr }
			contract.Transitions = append(contract.Transitions, StateTransition{Ordinal: ordinal, From: get("from"), To: get("to"), Evidence: get("evidence")})
		case "invariant":
			ordinal, parseErr := integer("ordinal")
			if parseErr != nil { return Contract{}, parseErr }
			contract.Invariants = append(contract.Invariants, Invariant{Ordinal: ordinal, Group: get("group"), ID: get("id"), Name: get("name")})
		case "vector":
			contract.Vectors = append(contract.Vectors, Vector{Name: get("name"), Indicators: splitList(get("indicators"))})
		case "scenario":
			ordinal, parseErr := integer("ordinal")
			if parseErr != nil { return Contract{}, parseErr }
			contract.Scenarios = append(contract.Scenarios, Scenario{Ordinal: ordinal, ID: get("id"), Fixture: get("fixture"), Expected: Decision(get("expected"))})
		case "authority", "source_policy", "improvement_pair":
			continue
		default:
			return Contract{}, fmt.Errorf("line %d: unknown directive %q", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return Contract{}, err
	}
	if err := contract.Validate(); err != nil {
		return Contract{}, err
	}
	return contract, nil
}

func parseAttributes(fields []string) (map[string]string, error) {
	attrs := make(map[string]string, len(fields))
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid attribute %q", field)
		}
		attrs[parts[0]] = strings.Trim(parts[1], "\"")
	}
	return attrs, nil
}

func splitList(value string) []string {
	if value == "" { return nil }
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item != "" { result = append(result, item) }
	}
	return result
}

func parseDecisions(value string) []Decision {
	parts := strings.Split(value, ">")
	result := make([]Decision, 0, len(parts))
	for _, item := range parts {
		if item != "" { result = append(result, Decision(item)) }
	}
	return result
}

func (c Contract) Validate() error {
	if c.ID == "" || c.Version == "" { return fmt.Errorf("contract identity is required") }
	if len(c.DecisionPrecedence) != 3 || c.DecisionPrecedence[0] != Refuted || c.DecisionPrecedence[1] != Unknown || c.DecisionPrecedence[2] != Closed {
		return fmt.Errorf("decision precedence must be REFUTED>UNKNOWN>CLOSED")
	}
	wantUnknown := []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}
	if !sameStrings(c.UnknownFields, wantUnknown) { return fmt.Errorf("unknown field contract is not the six-field contract") }
	if len(c.Invariants) != 12 { return fmt.Errorf("denominator must contain exactly twelve invariants") }
	counts := map[string]int{}
	for index, invariant := range c.Invariants {
		if invariant.Ordinal != index+1 || invariant.ID == "" || invariant.Name == "" { return fmt.Errorf("invalid invariant ordinal or identity") }
		counts[invariant.Group]++
	}
	if counts["FOUNDATION"] != 4 || counts["COHERENCE"] != 4 || counts["REGRESSION"] != 4 { return fmt.Errorf("invariant proof partition must be 4/4/4") }
	if len(c.Vectors) != 3 { return fmt.Errorf("indicator distribution must have three vectors") }
	vectorCounts := map[string]int{}
	for _, vector := range c.Vectors { vectorCounts[vector.Name] = len(vector.Indicators) }
	if vectorCounts["DRIVER"] != 4 || vectorCounts["OUTCOME"] != 4 || vectorCounts["GUARDRAIL"] != 4 { return fmt.Errorf("indicator distribution must be 4/4/4") }
	if len(c.DeletionOrder) != 5 || c.TieBreak != "lexical_id" { return fmt.Errorf("deletion order and lexical tie-break are required") }
	for index, rule := range c.DeletionOrder {
		if rule.Ordinal != index+1 || rule.Kind == "" { return fmt.Errorf("deletion order must be ordinal and complete") }
	}
	if c.GraphBound <= 0 || c.Oracle.Name == "" || c.Oracle.ContractID == "" || c.Oracle.ReplayCount != 2 || len(c.ReplayReceiptFields) != 6 {
		return fmt.Errorf("oracle, graph bound, and replay receipt contract are incomplete")
	}
	if c.GlobalMinimumClaim || !c.PredicateMonotone { return fmt.Errorf("contract must disclaim global minimum and declare monotone predicate") }
	if c.RuntimeRepositoryWrites != 0 || c.RuntimeLocalTests != 0 || c.RuntimeCrossProjectGates != 0 { return fmt.Errorf("runtime zero gates are mandatory") }
	if len(c.Scenarios) != 7 { return fmt.Errorf("denominator must contain exactly seven canonical scenarios") }
	seen := map[string]bool{}
	for index, scenario := range c.Scenarios {
		if scenario.Ordinal != index+1 || scenario.ID == "" || scenario.Fixture == "" || !validDecision(scenario.Expected) { return fmt.Errorf("invalid scenario declaration") }
		if seen[scenario.ID] { return fmt.Errorf("duplicate scenario %q", scenario.ID) }
		seen[scenario.ID] = true
	}
	return nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) { return false }
	for index := range left { if left[index] != right[index] { return false } }
	return true
}
