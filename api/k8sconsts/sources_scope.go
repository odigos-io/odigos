package k8sconsts

import "github.com/odigos-io/odigos/common"

// filter specific sources based on criterias.
// sources can be matched based on the workload details (namespace, kind, name),
// it's namespace, or the programming language of the container.
//
// if no criterias are provided, all sources are matched.
// empty list means "match all".
// each criteria is matched using OR semantics, so if any of the criteria is matched, the source is matched.
// if multiple criteria are provided, a source must match all of them to be pass the filter.
type SourcesScopes struct {
	Sources    []PodWorkload                `json:"sources,omitempty"`
	Namespaces []string                     `json:"namespaces,omitempty"`
	Languages  []common.ProgrammingLanguage `json:"languages,omitempty"`
}
