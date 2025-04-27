package odigosurltemplateprocessor

import (
	"fmt"
	"regexp"

	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type K8sWorkload struct {
	Namespace string `mapstructure:"namespace"`
	Kind      string `mapstructure:"kind"`
	Name      string `mapstructure:"name"`
}

// MatchProperties is similar to how the collector "attributes" and "resource"
// processors allows to match on handy attributes and resources.
// at the moment, we will just use the workload properties to match on
// but in the future this can be extended and shared with other processors
// that need some filtering capabilities.
type MatchProperties struct {
	// describes a list of k8s workloads to match on
	// A match occurs if the resource matches a workload in the list.
	// The workload is defined by the namespace, kind and name.
	// Kind is "deployment", "statefulset" or "daemonset".
	// This is optional field, and defaults to empty.
	K8sWorkloads []K8sWorkload `mapstructure:"k8s_workloads"`
}

// Inspired by the "attributes" and "resource" processors config structure which will allow us
// to be more streamlined with common existing components and extend/reuse in the future.
// This matcher is adapted to the odigos use case.
// basically, given a telemetry item, it will either match or not match an item,
// based on the optional properties that are defined in the config.
type MatchConfig struct {
	// If set, a span that matches the filter properties will be excluded from processing.
	// if a span matches both include and exclude, it will be excluded (exclude takes precedence).
	Exclude *MatchProperties `mapstructure:"exclude"`

	// If set, the span must match at least one of the properties to be processed.
	Include *MatchProperties `mapstructure:"include"`

	// If neither Include nor Exclude are specified, the processor will match all otel resources.
}

type TemplatizationConfig struct {

	// This option allows fine-tuning for specific paths to customize what to templatize and what not.
	// The rule looks like this: "/v1/{foo:regex}/bar/{baz}".
	// Each segment part in "{}" denote templatization, and all other segments should match the text exactly.
	// Inside the "{}" you can optionally set the template name and matching regex.
	// The template name is the name that will be used in the span name and attributes (e.g. "/users/{userId}").
	// The regex is optional, and if provided, it will be used to match the segment.
	// If the regex does not match, the rule will be skipped and other rules and templatization will be evaluated.
	// Example: "/v1/{foo:\d+}" will match "/v1/123" producing "/v1/{foo}", but not with "/v1/abc".
	// compatible with golang regexp module https://pkg.go.dev/regexp
	// for performance reasons, avoid using compute-intensive expressions or adding too many values here.
	TemplatizationRules []string `mapstructure:"templatization_rules"`

	// CustomIdsRegexp is a list of additional regex patterns that will be used to match and templated matching path segment.
	// It allows users to define their own regex patterns for custom id formats used/observed in their applications.
	// Note that this regexp should catch ids, but avoid catching other static unrelated strings.
	// For example, if you have ids in the system like "ap123" then a regexp that matches "^ap\d+" would be good,
	// but regexp like "^ap" is too permissive and will also catch "/api".
	// compatible with golang regexp module https://pkg.go.dev/regexp
	// for performance reasons, avoid using compute-intensive expressions or adding too many values here.
	CustomIdsRegexp []string `mapstructure:"custom_ids_regexp"`
}

type Config struct {
	MatchConfig `mapstructure:",squash"`

	// TemplatizationConfig is a list of rules that will be used in addition to the default rules
	// to apply templatization to the path.
	// It is optional and defaults to empty.
	TemplatizationConfig `mapstructure:",squash"`
}

var _ xconfmap.Validator = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (c Config) Validate() error {
	for _, rule := range c.TemplatizationRules {
		if _, err := parseUserInputRuleString(rule); err != nil {
			return err
		}
	}

	for _, r := range c.CustomIdsRegexp {
		if _, err := regexp.Compile(r); err != nil {
			return fmt.Errorf("invalid custom id regexp: %w", err)
		}
	}
	return nil
}
