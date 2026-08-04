package recommendations

import "gopkg.in/yaml.v3"

type RecommendationType string

const (
	RecommendationTypeInferDBAttributes   RecommendationType = "InferDBAttributes"
	RecommendationTypeAutoGoOffsetUpdater RecommendationType = "AutoGoOffsetUpdater"
	RecommendationTypeEnableOwnMetrics    RecommendationType = "EnableOwnMetrics"
	RecommendationTypeSampleHealthProbes  RecommendationType = "SampleHealthProbes"
	RecommendationTypeUrlTemplatization   RecommendationType = "UrlTemplatization"
)

// Recommendation is the flat catalog entry for a single recommendation,
// parsed from the nested manifests under manifests/.
type Recommendation struct {

	// Type uniquely identifies this recommendation (e.g. "InferDBAttributes").
	Type RecommendationType `yaml:"type"`

	// OSS is true when the recommendation is available in the open-source edition.
	OSS bool `yaml:"oss"`

	// RequireOdigosDeployment is true when enabling the recommendation changes the Odigos deployment itself.
	RequireOdigosDeployment bool `yaml:"requireOdigosDeployment"`

	// Conditions are prerequisites that should hold before the recommendation is considered relevant.
	Conditions []Condition `yaml:"conditions"`

	// Title is the short display name shown in the UI.
	Title string `yaml:"title"`

	// Summary is a one-line explanation of what the recommendation does.
	Summary string `yaml:"summary"`

	// Description is the longer body text explaining the recommendation.
	Description string `yaml:"description"`

	// Docs is an optional URL to the full documentation page.
	Docs string `yaml:"docs"`

	// Pros lists benefits of enabling the recommendation.
	Pros []string `yaml:"pros"`

	// Cons lists trade-offs or costs of enabling the recommendation.
	Cons []string `yaml:"cons"`

	// Actions are optional choices the user can pick when applying the recommendation.
	Actions []Action `yaml:"actions"`
}

type Condition struct {
	Type string `yaml:"type"`
}

type Action struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

func (r *Recommendation) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Spec struct {
			Type                    RecommendationType `yaml:"type"`
			OSS                     bool               `yaml:"oss"`
			RequireOdigosDeployment bool               `yaml:"requireOdigosDeployment"`
			Conditions              []Condition        `yaml:"conditions"`
			Title                   string             `yaml:"title"`
			Summary                 string             `yaml:"summary"`
			Description             string             `yaml:"description"`
			Docs                    string             `yaml:"docs"`
			Pros                    []string           `yaml:"pros"`
			Cons                    []string           `yaml:"cons"`
			Actions                 []Action           `yaml:"actions"`
		} `yaml:"spec"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}

	r.Type = raw.Spec.Type
	r.OSS = raw.Spec.OSS
	r.RequireOdigosDeployment = raw.Spec.RequireOdigosDeployment
	r.Conditions = raw.Spec.Conditions
	r.Title = raw.Spec.Title
	r.Summary = raw.Spec.Summary
	r.Description = raw.Spec.Description
	r.Docs = raw.Spec.Docs
	r.Pros = raw.Spec.Pros
	r.Cons = raw.Spec.Cons
	r.Actions = raw.Spec.Actions
	return nil
}
