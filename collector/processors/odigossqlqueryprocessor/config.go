package odigossqlqueryprocessor

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	// InferAttributes controls whether attributes such as db.operation.name
	// and table names are extracted from the SQL query and set on the span.
	InferAttributes bool `mapstructure:"infer_attributes"`

	// RedactLiterals controls whether literal values in the SQL query are replaced
	// with placeholders on db.query.text / db.statement.
	// Example: "SELECT * FROM users WHERE age > 18" -> "SELECT * FROM users WHERE age > ?"
	RedactLiterals bool `mapstructure:"redact_literals"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	return nil
}
