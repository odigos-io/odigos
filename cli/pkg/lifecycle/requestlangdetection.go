package lifecycle

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/pkg/remote"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RequestLangDetection struct {
	BaseTransition
}

var _ Transition = &RequestLangDetection{}

func (r *RequestLangDetection) From() State {
	return PreflightChecksPassed
}

func (r *RequestLangDetection) To() State {
	return LangDetectionInProgress
}

func (r *RequestLangDetection) Execute(ctx context.Context, obj client.Object, isRemote bool) error {
	workloadKind := workload.WorkloadKindFromClientObject(obj)
	if !isRemote {
		var source *odigosv1.Source
		selector := labels.SelectorFromSet(labels.Set{
			k8sconsts.WorkloadNameLabel:      obj.GetName(),
			k8sconsts.WorkloadNamespaceLabel: obj.GetNamespace(),
			k8sconsts.WorkloadKindLabel:      string(workloadKind),
		})
		sources, err := r.client.OdigosClient.Sources(obj.GetNamespace()).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return err
		}
		if len(sources.Items) > 0 {
			source = &sources.Items[0]
		} else {
			source = &odigosv1.Source{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: workload.CalculateWorkloadRuntimeObjectName(obj.GetName(), workloadKind),
					Namespace:    obj.GetNamespace(),
				},
				Spec: odigosv1.SourceSpec{
					Workload: k8sconsts.PodWorkload{
						Kind:      workloadKind,
						Name:      obj.GetName(),
						Namespace: obj.GetNamespace(),
					},
				},
			}
		}

		if len(sources.Items) > 0 {
			source, err = r.client.OdigosClient.Sources(obj.GetNamespace()).Update(ctx, source, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		} else {
			source, err = r.client.OdigosClient.Sources(obj.GetNamespace()).Create(ctx, source, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	} else {
		err := remote.CreateSource(ctx, r.client, obj.GetNamespace(), string(workloadKind), obj.GetNamespace(), obj.GetName())
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RequestLangDetection) GetTransitionState(ctx context.Context, obj client.Object, isRemote bool, odigosNamespace string) (State, error) {
	workloadKind := workload.WorkloadKindFromClientObject(obj)
	if !isRemote {
		labeled := labels.Set{
			k8sconsts.WorkloadNameLabel:      obj.GetName(),
			k8sconsts.WorkloadNamespaceLabel: obj.GetNamespace(),
			k8sconsts.WorkloadKindLabel:      string(workloadKind),
		}
		sources, err := r.client.OdigosClient.Sources(obj.GetNamespace()).List(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labeled).String()})
		if err != nil {
			return UnknownState, err
		}
		if len(sources.Items) == 0 {
			return r.From(), nil
		}
	} else {
		des, err := remote.DescribeSource(ctx, r.client, odigosNamespace, string(workloadKind), obj.GetNamespace(), obj.GetName())
		if err != nil || des.Name.Value == nil {
			// name value will be nil for unsupported kinds
			return UnknownState, err
		}

		if des.SourceObjectsAnalysis.Instrumented.Value != true {
			return r.From(), nil
		}
	}
	return r.To(), nil
}
