package instrumentationrules

// InstrumentationRule describes the static, YAML-defined catalog entry for a
// single instrumentation-rule type. It mirrors the actions catalog
// (odigos/actions) so the frontend can dynamically render the list of available
// instrumentation rules and (where provided) their form fields, instead of
// relying on hard-coded options baked into the UI.
type InstrumentationRule struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	// Type must match the GraphQL InstrumentationRuleType enum value
	// (e.g. "CodeAttributes").
	Type        string `yaml:"type"`
	DisplayName string `yaml:"displayName"`
}

type Spec struct {
	// Description is a short, human-readable summary shown in the rule picker
	// header (the new convention that replaced the old docsDescription).
	Description string `yaml:"description"`
	// SupportedLanguages declares which programming languages this rule can apply
	// to. Rules gate on language rather than telemetry signals (unlike actions).
	// Values match the kit's ProgrammingLanguages enum (e.g. "go", "java",
	// "javascript", "python").
	SupportedLanguages []string `yaml:"supportedLanguages"`
	// DocsURL is the full documentation URL (e.g. https://docs.odigos.io/...).
	DocsURL string `yaml:"docsUrl"`
	// Fields is an optional dynamic-form description. Instrumentation rules mostly
	// ship bespoke forms in the UI, so this is usually empty; it exists so future
	// rule types can be described generically (matching the actions catalog).
	Fields []Field `yaml:"fields"`
}

type Field struct {
	Name            string                 `yaml:"name"`
	DisplayName     string                 `yaml:"displayName"`
	ComponentType   string                 `yaml:"componentType"`
	ComponentProps  map[string]interface{} `yaml:"componentProps"`
	InitialValue    string                 `yaml:"initialValue"`
	RenderCondition []string               `yaml:"renderCondition"`
}
