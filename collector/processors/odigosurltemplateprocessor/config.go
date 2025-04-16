package odigosurltemplateprocessor

import (
	"fmt"
	"regexp"

	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type Config struct {
	// The processor by default will templatize numbers and uuids.
	// This will cover some cases, but if id is a name, pattern with letters, internal representation, etc
	// those cannot be detected deterministically and might create high cardinality in span names and low cardinality attributes.
	// The TemplatizationRules is a list of path templatizations specific rules that will be applied and taken if matched.
	// A rule is a pattern for a path that is composed of multiple path segments separated by "/".
	// each segment can be a string or a template variable.
	// strings are matched as is and are used in the template to replace the segment.
	// templatization segments like this: "/{name:regex}" and are used to match and replace the segment with the name.
	// e.g. "/v1/{foo:regex}/bar/{baz}" will match "/v1/123/bar/456" and will replace it with "/v1/:foo/bar/:baz"
	// if regex is not used, the segment will always match and replaced with the name.
	// if regex is used, and does not match, the segment will be skipped and will not take effect.
	TemplatizationRules []string `mapstructure:"templatization_rules"`

	// CustomIdsRegexp is a list of regex patterns that will be used to match and templated in any path segment
	// It allows users to define their own regex patterns for ids used/observed in their applications.
	// Note that this regexp should catch ids, but avoid catching other static strings.
	// For example, if you have ids in the system like "ap123" then a regexp that matches "^ap\d+" would be good,
	// but regexp like "^ap" is too permissive and will also catch "/api".
	// compatible with golang regexp module https://pkg.go.dev/regexp
	// for performance reasons, avoid using compute-intensive expressions or adding too many values here.
	CustomIdsRegexp []string `mapstructure:"custom_ids_regexp"`
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
