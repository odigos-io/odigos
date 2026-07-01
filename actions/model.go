package actions

// Action describes the static, YAML-defined catalog entry for a single action
// type. It mirrors the structure used by the destinations catalog
// (odigos/destinations) so that the frontend can dynamically render the list of
// available actions and (where provided) their form fields, instead of relying
// on hard-coded options baked into the UI.
type Action struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	// Type must match the GraphQL ActionType enum value (e.g. "K8sAttributesResolver").
	Type        string `yaml:"type"`
	DisplayName string `yaml:"displayName"`
}

type Spec struct {
	// Description is a short, human-readable summary shown in the action picker.
	Description string `yaml:"description"`
	// Signals declares which telemetry signals this action can process. Mirrors
	// the destinations catalog (odigos/destinations) so the YAML authoring format
	// is identical across catalogs. The UI resolves the action icon locally from
	// the action type, so no icon is described here.
	Signals struct {
		Traces struct {
			Supported bool `yaml:"supported"`
		}
		Metrics struct {
			Supported bool `yaml:"supported"`
		}
		Logs struct {
			Supported bool `yaml:"supported"`
		}
		Profiles struct {
			Supported bool `yaml:"supported"`
		}
	} `yaml:"signals"`
	// DocsURL is the full documentation URL (e.g. https://docs.odigos.io/...).
	DocsURL string `yaml:"docsUrl"`
	// Fields is an optional dynamic-form description. Actions describe their form
	// fields here so the UI can render them generically (including nested/tabular
	// fields); complex actions may still ship a bespoke form in the UI.
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
