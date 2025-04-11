package odigosurltemplateprocessor

import (
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type Config struct {
	// The processor by default will templatize numbers and uuids.
	// This will cover some cases, but if id is a name, pattern with letters, internal representation, etc
	// those cannot be detected deterministically.
	// The SpecificPathPatterns is a list of path templatizations rules that will be applied and taken if matched.
	// A rule is a pattern for a path that is composed of multiple path segments separated by "/".
	// each segment can be a string or a regex pattern.
	// strings are matched as is and are used in the template to replace the segment.
	// regex patterns looks like this: "/{name:regex}" and are used to match and replace the segment with the name.
	// e.g. "/v1/{foo:regex}/bar/{baz}" will match "/v1/123/bar/456" and will replace it with "/v1/:foo/bar/:baz"
	// if regex is not used, the segment will always match and replaced with the name.
	// if regex is used, and does not match, the segment will be skipped and will not take effect.
	SpecificPathPatterns []string `mapstructure:"specific_path_patterns"`
}

var _ xconfmap.Validator = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (c Config) Validate() error {
	return nil
}
