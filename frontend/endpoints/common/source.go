package common

type SourceID struct {
	// combination of namespace, kind and name is unique
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
}