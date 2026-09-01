package reducer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestContractHasExactPartitions(t *testing.T) {
	contract, err := LoadContract(filepath.Join("..", ".gooo", "causal-counterexample-reducer.gooo"))
	if err != nil { t.Fatal(err) }
	if len(contract.Invariants) != 12 { t.Fatalf("invariant count = %d", len(contract.Invariants)) }
	counts := map[string]int{}
	for _, invariant := range contract.Invariants { counts[invariant.Group]++ }
	if counts["FOUNDATION"] != 4 || counts["COHERENCE"] != 4 || counts["REGRESSION"] != 4 { t.Fatalf("unexpected invariant partition: %#v", counts) }
	vectorCounts := map[string]int{}
	for _, vector := range contract.Vectors { vectorCounts[vector.Name] = len(vector.Indicators) }
	if vectorCounts["DRIVER"] != 4 || vectorCounts["OUTCOME"] != 4 || vectorCounts["GUARDRAIL"] != 4 { t.Fatalf("unexpected vector partition: %#v", vectorCounts) }
}

func TestCanonicalFixtures(t *testing.T) {
	contract, err := LoadContract(filepath.Join("..", ".gooo", "causal-counterexample-reducer.gooo"))
	if err != nil { t.Fatal(err) }
	for _, scenario := range contract.Scenarios {
		data, readErr := os.ReadFile(filepath.Join("..", scenario.Fixture))
		if readErr != nil { t.Fatal(readErr) }
		var input Input
		if unmarshalErr := json.Unmarshal(data, &input); unmarshalErr != nil { t.Fatal(unmarshalErr) }
		result := Reduce(input, contract)
		if result.Decision != scenario.Expected { t.Fatalf("%s: got %s want %s", scenario.ID, result.Decision, scenario.Expected) }
		if len(result.Proof.Foundation) != 4 || len(result.Proof.Coherence) != 4 || len(result.Proof.Regression) != 4 { t.Fatalf("%s: proof partition is not 4/4/4", scenario.ID) }
		if len(result.IndicatorDistribution.Driver) != 4 || len(result.IndicatorDistribution.Outcome) != 4 || len(result.IndicatorDistribution.Guardrail) != 4 { t.Fatalf("%s: indicator partition is not 4/4/4", scenario.ID) }
	}
}

func TestMalformedFailsClosed(t *testing.T) {
	contract, err := LoadContract(filepath.Join("..", ".gooo", "causal-counterexample-reducer.gooo"))
	if err != nil { t.Fatal(err) }
	result := FailClosedResult(contract, "malformed-input", "invalid JSON")
	if result.Decision != Refuted { t.Fatalf("decision = %s", result.Decision) }
	if result.Metrics.RepositoryWrites != 0 || result.Metrics.LocalTestExecutions != 0 || result.Metrics.CrossProjectRequiredGates != 0 { t.Fatal("runtime zero gates were not preserved") }
}
