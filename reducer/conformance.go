package reducer

type CaseResult struct {
	Ordinal            int      `json:"ordinal"`
	ID                 string   `json:"id"`
	Expected           Decision `json:"expected"`
	Actual             Decision `json:"actual"`
	Pass               bool     `json:"pass"`
	DecisionPreserved  bool     `json:"decision_preserved"`
	PriorityPreserved  bool     `json:"priority_preserved"`
	UnknownFieldsValid bool     `json:"unknown_fields_valid"`
	WitnessPreserved   bool     `json:"witness_preserved"`
	Metrics            Metrics  `json:"metrics"`
}

type MalformedCase struct {
	ID       string   `json:"id"`
	Expected Decision `json:"expected"`
	Actual   Decision `json:"actual"`
	Pass     bool     `json:"pass"`
}

type ConformanceReport struct {
	Schema                    string                `json:"schema"`
	DenominatorID             string                `json:"denominator_id"`
	Scenarios                 int                   `json:"scenarios"`
	Closed                    int                   `json:"closed"`
	Unknown                   int                   `json:"unknown"`
	Refuted                   int                   `json:"refuted"`
	Cases                     []CaseResult          `json:"cases"`
	Malformed                 MalformedCase         `json:"malformed_input"`
	ProofVectors              ProofVectors          `json:"proof_vectors"`
	IndicatorDistribution     IndicatorDistribution `json:"indicator_distribution"`
	RepositoryWrites          int                   `json:"repository_writes"`
	LocalTestExecutions       int                   `json:"local_test_executions"`
	CrossProjectRequiredGates int                   `json:"cross_project_required_gates"`
	Pass                      bool                  `json:"pass"`
}
