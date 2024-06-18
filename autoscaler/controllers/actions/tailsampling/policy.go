package sampling

type PolicyDetails interface {
	Validate() error
}

type LatencyConfig struct {
	MinimumLatencyThreshold int  `json:"threshold_ms"`
	MaximumLatencyThreshold *int `json:"upper_threshold_ms,omitempty"`
}

type PolicyValidationError struct {
	Reason string
}

func (m *PolicyValidationError) Error() string {
	return m.Reason
}

func (lc *LatencyConfig) Validate() error {
	if lc.MaximumLatencyThreshold != nil {
		if *lc.MaximumLatencyThreshold < 0 {
			return &PolicyValidationError{Reason: "upper latency threshold must be positive"}
		}
	}
	if lc.MinimumLatencyThreshold < 0 {
		return &PolicyValidationError{Reason: "minimum latency threshold must be positive"}
	}
	return nil
}

type ProbabilisticConfig struct {
	Value float64 `json:"sampling_percentage"`
}

func (pc *ProbabilisticConfig) Validate() error {
	if pc.Value < 0 {
		return &PolicyValidationError{Reason: "sampling_percentage cannot be negative"}
	}
	return nil
}
