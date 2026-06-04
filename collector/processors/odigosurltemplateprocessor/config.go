package odigosurltemplateprocessor

import (
	"fmt"
	"regexp"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type CustomIdConfig struct {

	// A regexp string which will be matched against all path segments.
	// If the regexp matches, the segment will be templatized.
	Regexp string `mapstructure:"regexp"`

	// If the regexp matches, this is the name that will be used in the span name and attributes.
	// e.g. if name is "userId" then route like this will be produced "/users/{userId}".
	// default value (if empty) is "id".
	TemplateName string `mapstructure:"template_name"`
}

type TemplatizationConfig struct {

	// This option allows fine-tuning for specific paths to customize what to templatize and what not.
	// The rule format supports:
	// 1. Static string: "/v1/users" will match "/v1/users" but not "/v1/admins".
	// 2. Templated segment: "/v1/{name}" matches any single segment and produces "/v1/{name}". Template name defaults to "id" if omitted.
	// 3. Wildcard: "/v1/*" will match "/v1/users" or "/v1/admins" (one segment) but not "/v1/a/b".
	TemplatizationRules []string `mapstructure:"templatization_rules"`

	// CustomIds is a list of additional regex patterns that will be used to match and templated matching path segment.
	// It allows users to define their own regex patterns for custom id formats used/observed in their applications.
	// Note that this regexp should catch ids, but avoid catching other static unrelated strings.
	// For example, if you have ids in the system like "ap123" then a regexp that matches "^ap\d+" would be good,
	// but regexp like "^ap" is too permissive and will also catch "/api".
	// compatible with golang regexp module https://pkg.go.dev/regexp
	// for performance reasons, avoid using compute-intensive expressions or adding too many values here.
	CustomIds []CustomIdConfig `mapstructure:"custom_ids"`
}

type Config struct {
	// TemplatizationConfig is a list of rules that will be used in addition to the default rules
	// to apply templatization to the path.
	// It is optional and defaults to empty.
	TemplatizationConfig `mapstructure:",squash"`

	// OdigosConfigExtension is the default for Odigos: per-workload rules from the extension cache (e.g. odigos_config_k8s).
	// Must implement OdigosConfigExtension. If omitted from YAML, only TemplatizationConfig above is used (legacy).
	OdigosConfigExtension *component.ID `mapstructure:"odigos_config_extension"`
}

var _ xconfmap.Validator = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (c Config) Validate() error {
	for _, rule := range c.TemplatizationRules {
		if _, err := parseUserInputRuleString(rule); err != nil {
			return err
		}
	}

	for _, r := range c.CustomIds {
		if _, err := regexp.Compile(r.Regexp); err != nil {
			return fmt.Errorf("invalid custom id regexp: %w", err)
		}
	}

	// When set, the extension component type must be a valid OTel type string.
	if c.OdigosConfigExtension != nil {
		typeStr := c.OdigosConfigExtension.Type().String()
		if _, err := component.NewType(typeStr); err != nil {
			return fmt.Errorf("invalid odigos_config_extension type %q: %w", typeStr, err)
		}
	}

	return nil
}
