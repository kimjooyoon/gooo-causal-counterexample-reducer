package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-causal-counterexample-reducer/reducer"
)

func main() {
	mode := flag.String("mode", "reduce", "reduce or conformance")
	contractPath := flag.String("contract", ".gooo/causal-counterexample-reducer.gooo", "path to the .gooo contract")
	inputPath := flag.String("input", "", "immutable input JSON")
	fixturesRoot := flag.String("fixtures", ".", "repository root containing canonical fixtures")
	outputPath := flag.String("output", "", "caller-owned output directory")
	flag.Parse()

	if *outputPath == "" {
		fatal(errors.New("-output is required"))
	}
	contract, err := reducer.LoadContract(*contractPath)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*outputPath, 0o755); err != nil {
		fatal(err)
	}

	switch *mode {
	case "reduce":
		if *inputPath == "" {
			fatal(errors.New("-input is required in reduce mode"))
		}
		result := reduceFile(*inputPath, contract)
		if err := writeJSON(filepath.Join(*outputPath, "reduction-report.json"), result); err != nil {
			fatal(err)
		}
	case "conformance":
		report, metrics, proofs, err := runConformance(*fixturesRoot, contract, *outputPath)
		if err != nil {
			fatal(err)
		}
		if err := writeJSON(filepath.Join(*outputPath, "conformance-report.json"), report); err != nil {
			fatal(err)
		}
		if err := writeJSON(filepath.Join(*outputPath, "metrics.json"), metrics); err != nil {
			fatal(err)
		}
		if err := writeJSON(filepath.Join(*outputPath, "proof-vectors.json"), proofs); err != nil {
			fatal(err)
		}
		if !report.Pass {
			os.Exit(1)
		}
	default:
		fatal(fmt.Errorf("unsupported mode %q", *mode))
	}
}

func reduceFile(path string, contract reducer.Contract) reducer.Result {
	data, err := os.ReadFile(path)
	if err != nil {
		return reducer.FailClosedResult(contract, filepath.Base(path), err.Error())
	}
	var input reducer.Input
	if err := json.Unmarshal(data, &input); err != nil {
		return reducer.FailClosedResult(contract, filepath.Base(path), err.Error())
	}
	return reducer.Reduce(input, contract)
}

func runConformance(root string, contract reducer.Contract, output string) (reducer.ConformanceReport, map[string]reducer.Metrics, map[string]reducer.ProofVectors, error) {
	if err := os.MkdirAll(filepath.Join(output, "cases"), 0o755); err != nil {
		return reducer.ConformanceReport{}, nil, nil, err
	}
	report := reducer.ConformanceReport{Schema: "gooo.causal-counterexample-reducer/conformance/v1", DenominatorID: contract.ID, Scenarios: len(contract.Scenarios), Cases: []reducer.CaseResult{}}
	metrics := make(map[string]reducer.Metrics, len(contract.Scenarios))
	proofs := make(map[string]reducer.ProofVectors, len(contract.Scenarios))
	allPass := true
	for _, scenario := range contract.Scenarios {
		path := filepath.Join(root, scenario.Fixture)
		data, err := os.ReadFile(path)
		var result reducer.Result
		if err != nil {
			result = reducer.FailClosedResult(contract, scenario.ID, err.Error())
		} else {
			var input reducer.Input
			if unmarshalErr := json.Unmarshal(data, &input); unmarshalErr != nil {
				result = reducer.FailClosedResult(contract, scenario.ID, unmarshalErr.Error())
			} else {
				result = reducer.Reduce(input, contract)
			}
		}
		pass := result.Decision == scenario.Expected && len(result.Proof.Foundation) == 4 && len(result.Proof.Coherence) == 4 && len(result.Proof.Regression) == 4 && len(result.IndicatorDistribution.Driver) == 4 && len(result.IndicatorDistribution.Outcome) == 4 && len(result.IndicatorDistribution.Guardrail) == 4 && result.Metrics.RepositoryWrites == 0 && result.Metrics.LocalTestExecutions == 0 && result.Metrics.CrossProjectRequiredGates == 0
		caseResult := reducer.CaseResult{Ordinal: scenario.Ordinal, ID: scenario.ID, Expected: scenario.Expected, Actual: result.Decision, Pass: pass, DecisionPreserved: result.Correctness.DecisionPreserved, PriorityPreserved: result.Correctness.PriorityPreserved, UnknownFieldsValid: result.Decision != reducer.Unknown || result.Unknown.Valid(), WitnessPreserved: result.Correctness.WitnessPreserved, Metrics: result.Metrics}
		report.Cases = append(report.Cases, caseResult)
		metrics[scenario.ID] = result.Metrics
		proofs[scenario.ID] = result.Proof
		switch result.Decision {
		case reducer.Closed:
			report.Closed++
		case reducer.Unknown:
			report.Unknown++
		case reducer.Refuted:
			report.Refuted++
		}
		if !pass {
			allPass = false
		}
		if err := writeJSON(filepath.Join(output, "cases", scenario.ID+".report.json"), result); err != nil {
			return reducer.ConformanceReport{}, nil, nil, err
		}
	}
	malformedData, err := os.ReadFile(filepath.Join(root, "fixtures", "malformed", "invalid.json"))
	if err != nil {
		return reducer.ConformanceReport{}, nil, nil, err
	}
	malformed := reducer.FailClosedResult(contract, "malformed-input", "invalid JSON")
	if json.Valid(malformedData) {
		malformed = reducer.FailClosedResult(contract, "malformed-input", "malformed fixture unexpectedly parsed")
	}
	report.Malformed = reducer.MalformedCase{ID: "malformed-input", Expected: reducer.Refuted, Actual: malformed.Decision, Pass: malformed.Decision == reducer.Refuted}
	if err := writeJSON(filepath.Join(output, "cases", "malformed-input.report.json"), malformed); err != nil {
		return reducer.ConformanceReport{}, nil, nil, err
	}
	if !report.Malformed.Pass {
		allPass = false
	}
	report.ProofVectors = proofs[contract.Scenarios[0].ID]
	report.IndicatorDistribution = indicatorDistributionFromCase(report.Cases[0], proofs[contract.Scenarios[0].ID])
	report.RepositoryWrites = 0
	report.LocalTestExecutions = 0
	report.CrossProjectRequiredGates = 0
	report.Pass = allPass
	return report, metrics, proofs, nil
}

func indicatorDistributionFromCase(caseResult reducer.CaseResult, _ reducer.ProofVectors) reducer.IndicatorDistribution {
	integer := func(value int) *int { return &value }
	boolean := func(value bool) *bool { return &value }
	return reducer.IndicatorDistribution{
		Driver: []reducer.Indicator{
			{Name: "nodes_removed", IntegerValue: integer(caseResult.Metrics.Nodes.Before - caseResult.Metrics.Nodes.After)},
			{Name: "edges_removed", IntegerValue: integer(caseResult.Metrics.Edges.Before - caseResult.Metrics.Edges.After)},
			{Name: "evidence_removed", IntegerValue: integer(caseResult.Metrics.Evidence.Before - caseResult.Metrics.Evidence.After)},
			{Name: "cell_dependencies_removed", IntegerValue: integer(caseResult.Metrics.CellDependencies.Before - caseResult.Metrics.CellDependencies.After)},
		},
		Outcome: []reducer.Indicator{
			{Name: "decision_preserved", BooleanValue: boolean(caseResult.DecisionPreserved)},
			{Name: "priority_preserved", BooleanValue: boolean(caseResult.PriorityPreserved)},
			{Name: "unknown_fields_preserved", BooleanValue: boolean(caseResult.UnknownFieldsValid)},
			{Name: "witness_preserved", BooleanValue: boolean(caseResult.WitnessPreserved)},
		},
		Guardrail: []reducer.Indicator{
			{Name: "oracle_calls", IntegerValue: integer(caseResult.Metrics.OracleCalls)},
			{Name: "wall_ms", IntegerValue: integer(caseResult.Metrics.WallMS)},
			{Name: "peak_rss_kib", IntegerValue: integer(caseResult.Metrics.PeakRSSKiB)},
			{Name: "repository_writes", IntegerValue: integer(caseResult.Metrics.RepositoryWrites)},
		},
	}
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
