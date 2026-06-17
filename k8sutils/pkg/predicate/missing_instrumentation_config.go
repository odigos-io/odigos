package predicate

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	cr_predicate "sigs.k8s.io/controller-runtime/pkg/predicate"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

// MissingInstrumentationConfigPredicate allows update events when the workload's InstrumentationConfig
// does not exist and the workload is still covered by an active Source.
type MissingInstrumentationConfigPredicate struct {
	Client client.Client
}

func (p MissingInstrumentationConfigPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (p MissingInstrumentationConfigPredicate) Update(e event.UpdateEvent) bool {
	return p.workloadMissingInstrumentationConfig(e.ObjectNew)
}

func (p MissingInstrumentationConfigPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (p MissingInstrumentationConfigPredicate) Generic(e event.GenericEvent) bool {
	return false
}

func (p MissingInstrumentationConfigPredicate) workloadMissingInstrumentationConfig(obj client.Object) bool {
	if obj == nil {
		return false
	}

	pw, err := workload.PodWorkloadFromObject(obj)
	if err != nil {
		return false
	}

	icName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)
	ic := &odigosv1.InstrumentationConfig{}
	err = p.Client.Get(context.Background(), client.ObjectKey{Namespace: pw.Namespace, Name: icName}, ic)
	if !apierrors.IsNotFound(err) {
		return false
	}

	sources, err := odigosv1.GetSources(context.Background(), p.Client, pw)
	enabled, _, err := sourceutils.IsObjectInstrumentedBySource(context.Background(), sources, err)
	if err != nil || !enabled {
		return false
	}

	return true
}

// WorkloadCreateOrMissingInstrumentationConfig reconciles workload creates and updates that happen while the
// workload's InstrumentationConfig is missing (for example after a GitOps replace cascades IC deletion).
func WorkloadCreateOrMissingInstrumentationConfig(c client.Client) cr_predicate.Predicate {
	return cr_predicate.Or(&CreationPredicate{}, &MissingInstrumentationConfigPredicate{Client: c})
}

var _ cr_predicate.Predicate = &MissingInstrumentationConfigPredicate{}
