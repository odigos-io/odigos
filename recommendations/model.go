package recommendations

import (
	"github.com/odigos-io/odigos/common"
	"gopkg.in/yaml.v3"
)

// Recommendation is the flat catalog entry for a single recommendation,
// parsed from the nested manifests under manifests/.
type Recommendation struct {

	// Type uniquely identifies this recommendation (e.g. "InferDBAttributes").
	Type common.RecommendationType `yaml:"type"`

	// K8sObjectName is the metadata.name of the Recommendation CR created in the cluster.
	K8sObjectName string `yaml:"k8sObjectName"`

	// OSS is true when the recommendation is available in the open-source edition.
	OSS bool `yaml:"oss"`

	// RequireOdigosDeployment is true when enabling the recommendation changes the Odigos deployment itself.
	RequireOdigosDeployment bool `yaml:"requireOdigosDeployment"`

	// Conditions are prerequisites that should hold before the recommendation is considered relevant.
	Conditions []Condition `yaml:"conditions"`

	// AppliedWhen lists checks that must all hold for the recommendation to be considered applied.
	// Each entry is typed; currently EffectiveConfig compares a path in OdigosConfiguration to a value.
	AppliedWhen []AppliedWhenCheck `yaml:"appliedWhen"`

	// Categories are UI grouping labels for this recommendation
	// (e.g. "Enrichment", "Cost Optimization", "Usability").
	Categories []string `yaml:"categories"`

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

	// Remediations are optional choices the user can pick when applying the recommendation.
	Remediations []Remediation `yaml:"remediations"`
}

type Condition struct {
	Type string `yaml:"type"`
}

const (
	ConditionTypeGoEnterpriseSources = "GoEnterpriseSources"
)

const (
	AppliedWhenTypeEffectiveConfig = "EffectiveConfig"
	AppliedWhenTypeActionExists    = "ActionExists"
)

// AppliedWhenCheck is one criterion for whether a recommendation is currently applied.
// Type selects the check kind; additional fields depend on the type.
type AppliedWhenCheck struct {
	// Type selects how this check is evaluated (e.g. EffectiveConfig, ActionExists).
	Type string `yaml:"type"`

	// Expression is a JMESPath expression evaluated against the effective-config YAML.
	// It must return a boolean. Used when Type is EffectiveConfig.
	// Examples: "goAutoOffsetsMode != 'off'", "sampling.k8sHealthProbesSampling.enabled == `true`"
	Expression string `yaml:"expression,omitempty"`

	// ActionType is the Action config type to look for (e.g. InferDbAttributes).
	// Used when Type is ActionExists. The recommendation is applied when at least one
	// non-disabled Action of this type exists in the cluster.
	ActionType string `yaml:"actionType,omitempty"`
}

type Remediation struct {
	// Type uniquely identifies this remediation within the recommendation (e.g. "DropAllHealthProbes").
	Type       string            `yaml:"type"`
	ButtonText string            `yaml:"buttonText"`
	Tooltip    string            `yaml:"tooltip"`
	Steps      []RemediationStep `yaml:"steps"`
	// ApplyExamples are copy-paste snippets for applying this remediation outside the UI
	// (e.g. Helm values or an Action CR).
	ApplyExamples []ApplyExample `yaml:"applyExamples"`
}

const (
	RemediationStepTypeEditConfig       = "EditConfig"
	RemediationStepTypeApplyOdigosAction = "ApplyOdigosAction"
)

const (
	ApplyExampleTypeHelmValues   = "HelmValues"
	ApplyExampleTypeOdigosAction = "OdigosAction"
)

// ApplyExample is a declarative snippet showing how to apply a remediation outside the UI.
// Type selects the snippet kind; Content is the raw text (typically YAML).
type ApplyExample struct {
	// Type selects the example kind (e.g. HelmValues, OdigosAction).
	Type string `yaml:"type"`

	// Content is the example body to show or copy (usually a YAML fragment or full manifest).
	Content string `yaml:"content"`
}

// RemediationStep is one operation to perform when the user chooses a remediation.
// Type selects the operation; additional fields depend on the type.
type RemediationStep struct {
	// Type selects how this step is applied (e.g. EditConfig, ApplyOdigosAction).
	Type string `yaml:"type"`

	// Path is a dotted path into OdigosConfiguration (same shape as appliedWhen EffectiveConfig paths).
	// Used when Type is EditConfig.
	// Examples: "sampling.k8sHealthProbesSampling.enabled", "goAutoOffsetsMode"
	Path string `yaml:"path,omitempty"`

	// Value is the value to set at Path. Used when Type is EditConfig.
	// Parsed as a typed YAML value (bool, number, string, map, list).
	Value any `yaml:"value,omitempty"`
}

// OdigosActionExample returns the first ApplyExample of type OdigosAction, if any.
func (r Remediation) OdigosActionExample() (ApplyExample, bool) {
	for _, example := range r.ApplyExamples {
		if example.Type == ApplyExampleTypeOdigosAction {
			return example, true
		}
	}
	return ApplyExample{}, false
}

func (r *Recommendation) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Spec struct {
			Type                    common.RecommendationType `yaml:"type"`
			K8sObjectName           string                   `yaml:"k8sObjectName"`
			OSS                     bool                      `yaml:"oss"`
			RequireOdigosDeployment bool                      `yaml:"requireOdigosDeployment"`
			Conditions              []Condition               `yaml:"conditions"`
			AppliedWhen             []AppliedWhenCheck        `yaml:"appliedWhen"`
			Categories              []string                  `yaml:"categories"`
			Title                   string                    `yaml:"title"`
			Summary                 string                    `yaml:"summary"`
			Description             string                    `yaml:"description"`
			Docs                    string                    `yaml:"docs"`
			Pros                    []string                  `yaml:"pros"`
			Cons                    []string                  `yaml:"cons"`
			Remediations            []Remediation             `yaml:"remediations"`
		} `yaml:"spec"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}

	r.Type = raw.Spec.Type
	r.K8sObjectName = raw.Spec.K8sObjectName
	r.OSS = raw.Spec.OSS
	r.RequireOdigosDeployment = raw.Spec.RequireOdigosDeployment
	r.Conditions = raw.Spec.Conditions
	r.AppliedWhen = raw.Spec.AppliedWhen
	r.Categories = raw.Spec.Categories
	r.Title = raw.Spec.Title
	r.Summary = raw.Spec.Summary
	r.Description = raw.Spec.Description
	r.Docs = raw.Spec.Docs
	r.Pros = raw.Spec.Pros
	r.Cons = raw.Spec.Cons
	r.Remediations = raw.Spec.Remediations
	return nil
}
