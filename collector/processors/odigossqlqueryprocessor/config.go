package odigossqlqueryprocessor

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	// EnhanceAttributes controls whether attributes such as db.operation.name
	// and table names are extracted from the SQL query and set on the span.
	EnhanceAttributes bool `mapstructure:"enhance_attributes"`

	// Obfuscate controls whether literal values in the SQL query are replaced
	// with placeholders on db.query.text / db.statement.
	// Example: "SELECT * FROM users WHERE age > 18" -> "SELECT * FROM users WHERE age > ?"
	Obfuscate bool `mapstructure:"obfuscate"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	return nil
}
