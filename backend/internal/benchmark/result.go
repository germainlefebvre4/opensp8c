package benchmark

import "time"

// Phase is one named interval of a run with its start and end instants.
type Phase struct {
	Name  string    `json:"name"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Timings holds the durations (in seconds) of a run. Method-A fields and
// method-B fields are never both set.
type Timings struct {
	// Method A
	HumanSec      float64 `json:"human_s,omitempty"`
	MachineSec    float64 `json:"machine_s,omitempty"`
	ExploreSec    float64 `json:"explore_s,omitempty"`
	LaunchWaitSec float64 `json:"launch_wait_s,omitempty"`
	FFSec         float64 `json:"ff_s,omitempty"`
	PoolSec       float64 `json:"pool_s,omitempty"`
	PoolActiveSec float64 `json:"pool_active_s,omitempty"`
	PoolRuns      int     `json:"pool_runs,omitempty"`
	// Method B
	AgentSec      float64 `json:"agent_s,omitempty"`
	ValidationSec float64 `json:"validation_s,omitempty"`
	// Both
	TotalSec float64 `json:"total_s"`
}

// RunConfig is the effective configuration recorded in a result.
type RunConfig struct {
	Agent      string `json:"agent"`
	Model      string `json:"model"`
	Effort     string `json:"effort"`
	MaxRetries int    `json:"max_retries"`
}

// Result is the self-contained outcome of one run. The report is generated
// from result.json files alone.
type Result struct {
	Method   string    `json:"method"`
	RunID    string    `json:"run_id"`
	StartSHA string    `json:"start_sha"`
	Config   RunConfig `json:"config"`
	Phases   []Phase   `json:"phases"`
	Timings  Timings   `json:"timings"`

	Valid          bool     `json:"valid"`
	Status         string   `json:"status"`
	InvalidReasons []string `json:"invalid_reasons,omitempty"`
	Notes          []string `json:"notes,omitempty"`

	// Retries is the number of baseline retries (method B).
	Retries int `json:"retries,omitempty"`
	// PauseReason is set when a pool run ended with a non-completed outcome.
	PauseReason string `json:"pause_reason,omitempty"`

	AcceptanceRan    bool `json:"acceptance_ran"`
	AcceptancePassed bool `json:"acceptance_passed"`
}

func (r *Result) invalidate(reason string) {
	r.Valid = false
	r.Status = statusInvalid
	r.InvalidReasons = append(r.InvalidReasons, reason)
}
