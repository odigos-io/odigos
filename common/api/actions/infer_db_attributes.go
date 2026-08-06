package actions

// InferDbAttributesConfig is the per-container collector config for inferring
// additional attributes from database query text (e.g. db.operation.name,
// db.collection.name). Configuration options will be added later.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type InferDbAttributesConfig struct{}
