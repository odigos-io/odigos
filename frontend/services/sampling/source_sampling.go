package sampling

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetSourceSamplingForWorkload computes the sampling rules that apply to each
// container of the given workload.
//
// All sampling rules in the cluster are evaluated against the workload identity
// and the container's detected language using SourceScopeMatchesContainer; only
// rules whose source scope matches are included in the result. The container set
// is taken from the workload's InstrumentationConfig spec (one entry per pod
// manifest container) so that containers without runtime-detected languages are
// still represented (with a null language and only un-language-scoped rules).
func GetSourceSamplingForWorkload(ctx context.Context, k8sCacheClient client.Client, workloadID model.K8sWorkloadIDInput) (*model.SourceSampling, error) {
	pw := k8sconsts.PodWorkload{
		Namespace: workloadID.Namespace,
		Kind:      k8sconsts.WorkloadKind(workloadID.Kind),
		Name:      workloadID.Name,
	}

	containerLanguages, err := resolveContainerLanguages(ctx, k8sCacheClient, pw)
	if err != nil {
		return nil, err
	}

	samplings, err := listSamplingCRs(ctx, k8sCacheClient)
	if err != nil {
		return nil, err
	}

	containers := make([]*model.SourceContainerSampling, 0, len(containerLanguages))
	for _, cl := range containerLanguages {
		containers = append(containers, computeContainerSampling(samplings, pw, cl))
	}

	return &model.SourceSampling{
		WorkloadID: &model.K8sWorkloadID{
			Namespace: workloadID.Namespace,
			Kind:      workloadID.Kind,
			Name:      workloadID.Name,
		},
		Containers: containers,
	}, nil
}

// containerLanguage pairs a container's name with the detected programming
// language (or zero value when detection has not produced one yet).
type containerLanguage struct {
	name     string
	language common.ProgrammingLanguage
	// detected is false when the container exists in the pod manifest but
	// runtime detection has not yet produced a language for it.
	detected bool
}

// resolveContainerLanguages returns one entry per container in the workload's
// pod manifest, populating the detected language when available. Containers
// without a runtime-detected language are still included so the UI can render
// them and only language-scoped sampling rules will be filtered out.
func resolveContainerLanguages(ctx context.Context, k8sCacheClient client.Client, pw k8sconsts.PodWorkload) ([]containerLanguage, error) {
	var ic v1alpha1.InstrumentationConfig
	icName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)
	err := k8sCacheClient.Get(ctx, client.ObjectKey{Namespace: pw.Namespace, Name: icName}, &ic)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// no IC means the workload is not marked for instrumentation;
			// no sampling rules can be evaluated for it.
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch instrumentation config for %s/%s/%s: %w", pw.Namespace, pw.Kind, pw.Name, err)
	}

	languageByContainer := make(map[string]common.ProgrammingLanguage, len(ic.Status.RuntimeDetailsByContainer))
	for _, rd := range ic.Status.RuntimeDetailsByContainer {
		languageByContainer[rd.ContainerName] = rd.Language
	}

	seen := make(map[string]struct{})
	result := make([]containerLanguage, 0, len(ic.Spec.Containers))

	addContainer := func(name string) {
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		lang, ok := languageByContainer[name]
		result = append(result, containerLanguage{
			name:     name,
			language: lang,
			detected: ok,
		})
	}

	for _, c := range ic.Spec.Containers {
		addContainer(c.ContainerName)
	}
	// include any runtime-detected container that's missing from spec (defensive).
	for _, rd := range ic.Status.RuntimeDetailsByContainer {
		addContainer(rd.ContainerName)
	}

	return result, nil
}

// listSamplingCRs returns every Sampling CR in the odigos namespace.
func listSamplingCRs(ctx context.Context, k8sCacheClient client.Client) ([]v1alpha1.Sampling, error) {
	odigosNs := env.GetCurrentNamespace()
	var list v1alpha1.SamplingList
	if err := k8sCacheClient.List(ctx, &list, client.InNamespace(odigosNs)); err != nil {
		return nil, fmt.Errorf("failed to list sampling CRs: %w", err)
	}
	return list.Items, nil
}

// computeContainerSampling returns the sampling rules that match the given
// container by evaluating each rule's source scope against the workload identity
// and the container's language. When the language is undetected, language-scoped
// rules will not match — see SourceScopeMatchesContainer for the semantics.
func computeContainerSampling(samplings []v1alpha1.Sampling, pw k8sconsts.PodWorkload, cl containerLanguage) *model.SourceContainerSampling {
	noisy := []*model.NoisyOperationRule{}
	relevant := []*model.HighlyRelevantOperationRule{}
	cost := []*model.CostReductionRule{}

	language := cl.language
	if !cl.detected {
		language = common.UnknownProgrammingLanguage
	}

	for i := range samplings {
		spec := &samplings[i].Spec

		for j := range spec.NoisyOperations {
			rule := &spec.NoisyOperations[j]
			if scope.SourceScopeMatchesContainer(rule.SourceScopes, pw, language) {
				noisy = append(noisy, convertNoisyOperationToModel(rule))
			}
		}

		for j := range spec.HighlyRelevantOperations {
			rule := &spec.HighlyRelevantOperations[j]
			if scope.SourceScopeMatchesContainer(rule.SourceScopes, pw, language) {
				relevant = append(relevant, convertHighlyRelevantOperationToModel(rule))
			}
		}

		for j := range spec.CostReductionRules {
			rule := &spec.CostReductionRules[j]
			if scope.SourceScopeMatchesContainer(rule.SourceScopes, pw, language) {
				cost = append(cost, convertCostReductionRuleToModel(rule))
			}
		}
	}

	return &model.SourceContainerSampling{
		ContainerName:            cl.name,
		NoisyOperations:          noisy,
		HighlyRelevantOperations: relevant,
		CostReductionRules:       cost,
	}
}
