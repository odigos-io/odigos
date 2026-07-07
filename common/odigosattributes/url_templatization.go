package odigosattributes

const (
	// Span attribute indicating how URL templatization was applied. Type: string.
	// Not set when templatization was skipped (no path, default disabled, skip policy).
	UrlTemplatizationMethodAttribute = "odigos.url_templatization.method"
)

// UrlTemplatizationMethod describes which class of templatization was applied to a span.
// nil means templatization was skipped.
type UrlTemplatizationMethod string

func (m UrlTemplatizationMethod) Ptr() *UrlTemplatizationMethod {
	return &m
}

const (
	// A custom templatization rule matched the path.
	UrlTemplatizationMethodCustomRule UrlTemplatizationMethod = "custom_rule"
	// Default heuristic templatization replaced one or more path segments.
	UrlTemplatizationMethodDefaultHeuristic UrlTemplatizationMethod = "default_heuristic"
	// The path was normalized without segment templatization (e.g. "//" or empty path → "/").
	UrlTemplatizationMethodPathNormalization UrlTemplatizationMethod = "path_normalization"
	// Templatization was evaluated but the path was left unchanged.
	UrlTemplatizationMethodUnchanged UrlTemplatizationMethod = "unchanged"
)
