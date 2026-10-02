package benchmark

import "time"

// PhaseKind is a normalized phase boundary event, independent of where the
// data comes from (observer journal today, a backend metrics module later).
type PhaseKind string

const (
	ExploreStarted PhaseKind = "explore_started"
	FFStarted      PhaseKind = "ff_started"
	FFDone         PhaseKind = "ff_done"
	Launched       PhaseKind = "launched"
	PoolStarted    PhaseKind = "pool_started"
	PoolEnded      PhaseKind = "pool_ended"
	Merged         PhaseKind = "merged"
)

// OutcomeCompleted is the pool_run_end outcome of a successful pool run.
const OutcomeCompleted = "completed"

// PhaseEvent is one normalized phase boundary. Outcome and Reason are only
// set on PoolEnded.
type PhaseEvent struct {
	Kind    PhaseKind `json:"kind"`
	At      time.Time `json:"at"`
	Outcome string    `json:"outcome,omitempty"`
	Reason  string    `json:"reason,omitempty"`
}

// PhaseData is what a MetricsSource delivers to the calculator.
type PhaseData struct {
	Events []PhaseEvent
	// ObservationIncomplete is true when the observer lost its connection
	// during the run: the boundaries cannot be trusted.
	ObservationIncomplete bool
}

// MetricsSource provides normalized phase events for one method-A run.
type MetricsSource interface {
	Load() (PhaseData, error)
}

// MemorySource is an in-memory MetricsSource, used by tests.
type MemorySource struct{ Data PhaseData }

func (m MemorySource) Load() (PhaseData, error) { return m.Data, nil }
