package services

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func SourcesScopesInputToCRD(in *model.SourcesScopesInput) *k8sconsts.SourcesScopes {
	if in == nil {
		return nil
	}
	out := &k8sconsts.SourcesScopes{}
	if len(in.Sources) > 0 {
		out.Sources = make([]k8sconsts.PodWorkload, 0, len(in.Sources))
		for _, s := range in.Sources {
			if s == nil {
				continue
			}
			out.Sources = append(out.Sources, k8sconsts.PodWorkload{
				Name:      s.Name,
				Namespace: s.Namespace,
				Kind:      k8sconsts.WorkloadKind(s.Kind),
			})
		}
	}
	if len(in.Namespaces) > 0 {
		out.Namespaces = append([]string(nil), in.Namespaces...)
	}
	if len(in.Languages) > 0 {
		out.Languages = make([]common.ProgrammingLanguage, 0, len(in.Languages))
		for _, l := range in.Languages {
			out.Languages = append(out.Languages, common.ProgrammingLanguage(l))
		}
	}
	return out
}

func SourcesScopesCRDToModel(in *k8sconsts.SourcesScopes) *model.SourcesScopes {
	if in == nil {
		return nil
	}
	out := &model.SourcesScopes{}
	if len(in.Sources) > 0 {
		out.Sources = make([]*model.K8sWorkloadID, 0, len(in.Sources))
		for _, s := range in.Sources {
			out.Sources = append(out.Sources, &model.K8sWorkloadID{
				Name:      s.Name,
				Namespace: s.Namespace,
				Kind:      model.K8sResourceKind(s.Kind),
			})
		}
	}
	if len(in.Namespaces) > 0 {
		out.Namespaces = append([]string(nil), in.Namespaces...)
	}
	if len(in.Languages) > 0 {
		out.Languages = make([]model.SamplingWorkloadLanguage, 0, len(in.Languages))
		for _, l := range in.Languages {
			out.Languages = append(out.Languages, model.SamplingWorkloadLanguage(l))
		}
	}
	return out
}
