package odigosattributes

const (
	// Span attribute indicating how URL templatization was applied. Type: string.
	// Not set when templatization was skipped (no path, default disabled, skip policy).
	UrlTemplatizationResultAttribute = "odigos.url_templatization.result"
)

// UrlTemplatizationResult describes which class of templatization was applied to a span.
// nil means templatization was skipped.
type UrlTemplatizationResult string

func (m UrlTemplatizationResult) Ptr() *UrlTemplatizationResult {
	return &m
}

const (
	// A custom templatization rule matched the path.
	UrlTemplatizationResultCustomRule UrlTemplatizationResult = "custom_rule"
	// Default heuristic templatization replaced one or more path segments.
	UrlTemplatizationResultDefaultHeuristic UrlTemplatizationResult = "default_heuristic"
	// The path was normalized without segment templatization (e.g. "//" or empty path → "/").
	UrlTemplatizationResultPathNormalization UrlTemplatizationResult = "path_normalization"
	// Templatization ran but the path had no dynamic segments, so the templatized
	// result is the static path as-is.
	UrlTemplatizationResultStaticPath UrlTemplatizationResult = "static_path"
)
