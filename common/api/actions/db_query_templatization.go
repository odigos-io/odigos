package actions

// DbQueryTemplatizationConfig is the per-container collector config for
// templatizing database query text.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type DbQueryTemplatizationConfig struct {
	// TemplatizeLiterals replaces number and string literals in SQL queries with placeholders
	// (e.g. "SELECT * FROM users WHERE id = 1" -> "SELECT * FROM users WHERE id = ?").
	//
	// what is templatized:
	// - number literals
	// - string literals
	// what is not templatized:
	// - identifiers (e.g. table names, column names, etc.)
	// - positional parameters (e.g. "SELECT * FROM users WHERE id = $1")
	// - bind parameters (e.g. "SELECT * FROM users WHERE id = :name")
	// - boolean literals
	// - NULL
	// - keywords, operators, punctuation, whitespace, comments, etc.
	TemplatizeLiterals bool `json:"templatizeLiterals,omitempty"`
}
