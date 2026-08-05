package recommendations

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/yaml"
)

// ApplyStepsToConfig applies remediation steps to an OdigosConfiguration by
// patching a YAML map (dotted paths) and round-tripping back into the typed config.
func ApplyStepsToConfig(cfg *common.OdigosConfiguration, steps []RemediationStep) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	data := map[string]any{}
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("unmarshal config map: %w", err)
	}
	if data == nil {
		data = map[string]any{}
	}

	for i, step := range steps {
		if err := applyStep(data, step); err != nil {
			return fmt.Errorf("steps[%d]: %w", i, err)
		}
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal patched config: %w", err)
	}

	var updated common.OdigosConfiguration
	if err := yaml.Unmarshal(out, &updated); err != nil {
		return fmt.Errorf("patched config is not valid OdigosConfiguration: %w", err)
	}
	*cfg = updated
	return nil
}

func applyStep(data map[string]any, step RemediationStep) error {
	switch step.Type {
	case RemediationStepTypeEditConfig:
		return applyEditConfigStep(data, step)
	default:
		return fmt.Errorf("unknown remediation step type %q", step.Type)
	}
}

func applyEditConfigStep(data map[string]any, step RemediationStep) error {
	path := strings.TrimSpace(step.Path)
	if path == "" {
		return fmt.Errorf("EditConfig path is required")
	}
	return setConfigPath(data, path, step.Value)
}

// setConfigPath sets a dotted path (e.g. "sampling.k8sHealthProbesSampling.enabled") on a YAML map.
func setConfigPath(root map[string]any, path string, value any) error {
	parts := strings.Split(path, ".")
	if len(parts) == 0 || parts[0] == "" {
		return fmt.Errorf("empty path")
	}

	current := root
	for i, part := range parts {
		if part == "" {
			return fmt.Errorf("invalid path %q", path)
		}
		if i == len(parts)-1 {
			current[part] = value
			return nil
		}

		next, ok := current[part]
		if !ok || next == nil {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}

		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("path %q: %q is not an object", path, strings.Join(parts[:i+1], "."))
		}
		current = child
	}
	return nil
}
