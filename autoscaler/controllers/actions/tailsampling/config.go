package sampling

import (
	"encoding/json"
)

type TailSamplingConfig struct {
	Policies []Policy `json:"policies"`
}
type Policy struct {
	Name       string        `json:"name"`
	PolicyType string        `json:"type"`
	Details    PolicyDetails `json:"-"`
}

// Custom Marshal/Unmarshal functions support tail_sampling policies by replacing Policy Config key with PolicyType.
func (p Policy) MarshalJSON() ([]byte, error) {
	result := map[string]interface{}{
		"name": p.Name,
		"type": p.PolicyType,
	}

	configJson, err := json.Marshal(p.Details)
	if err != nil {
		return nil, err
	}

	configMap := make(map[string]interface{})
	if err := json.Unmarshal(configJson, &configMap); err != nil {
		return nil, err
	}

	result[p.PolicyType] = configMap

	return json.Marshal(result)
}
func (p *Policy) UnmarshalJSON(data []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	if err := json.Unmarshal(rawMap["name"], &p.Name); err != nil {
		return err
	}
	if err := json.Unmarshal(rawMap["type"], &p.PolicyType); err != nil {
		return err
	}

	var config PolicyDetails
	switch p.PolicyType {
	case "latency":
		var latencyConfig LatencyConfig
		if err := json.Unmarshal(rawMap["latency"], &latencyConfig); err != nil {
			return err
		}
		config = &latencyConfig
	case "probabilistic":
		var probabilisticConfig ProbabilisticConfig
		if err := json.Unmarshal(rawMap["probabilistic"], &probabilisticConfig); err != nil {
			return err
		}
		config = &probabilisticConfig
	}

	p.Details = config
	return nil
}
