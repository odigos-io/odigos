package source

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// IsObjectInstrumentedBySource returns true if the given object has an active, non-excluding Source.
// 1) Is the object actively included by a workload Source: true
// 2) Is the object active excluded by a workload Source (overrides namespace instrumentation): false
// 3) Is the object actively included by a namespace Source: true
// 4) False
func IsObjectInstrumentedBySource(ctx context.Context, k8sClient client.Client, obj client.Object) (bool, error) {
	// Check if a Source object exists for this object
	sources, err := v1alpha1.GetSources(ctx, k8sClient, obj)
	if err != nil {
		return false, err
	}

	if sources.Workload != nil {
		if IsActiveSource(sources.Workload) {
			return true, nil
		}
		if v1alpha1.IsExcludedSource(sources.Workload) && !k8sutils.IsTerminating(sources.Workload) {
			return false, nil
		}
	}

	if sources.Namespace != nil && IsActiveSource(sources.Namespace) {
		return true, nil
	}

	return false, nil
}

// IsActiveSource returns true if the Source enables instrumentation.
// Specifically, the Source must be either:
// 1) Inclusive AND NOT terminating, or
// 2) Exclusive AND terminating
func IsActiveSource(source *v1alpha1.Source) bool {
	return v1alpha1.IsExcludedSource(source) == k8sutils.IsTerminating(source)
}

// CreateOrUpdateSourceForObject creates a Source for an object if one does not exist and
// applies a label to the object referencing the new Source.
// The created Source will have a randomly generated name and be in the object's Namespace.
func CreateOrUpdateSourceForObject(ctx context.Context, k8sClient client.Client, obj client.Object, kind workload.WorkloadKind, disableInstrumentation bool) error {
	if !workload.IsValidWorkloadKind(kind) && kind != "Namespace" {
		return fmt.Errorf("invalid workload kind %s", kind)
	}

	namespace := obj.GetNamespace()
	if len(namespace) == 0 && kind == "Namespace" {
		namespace = obj.GetName()
	}

	sources, err := v1alpha1.GetSources(ctx, k8sClient, obj)
	if err != nil {
		return err
	}
	var source *v1alpha1.Source

	if kind == "Namespace" {
		if sources.Namespace != nil {
			source = sources.Namespace
		}
	} else {
		if sources.Workload != nil {
			source = sources.Workload
		}
	}

	create := false
	if source == nil {
		create = true
		source = &v1alpha1.Source{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("source-%s-%s-", strings.ToLower(string(kind)), strings.ToLower(obj.GetName())),
				Namespace:    namespace,
			},
			Spec: v1alpha1.SourceSpec{
				Workload: workload.PodWorkload{
					Name:      obj.GetName(),
					Namespace: namespace,
					Kind:      kind,
				},
			},
		}
	}
	source.Spec.DisableInstrumentation = disableInstrumentation

	if create {
		log.FromContext(ctx).Info("creating source", "source", source.Spec)
		return client.IgnoreAlreadyExists(k8sClient.Create(ctx, source))
	}
	log.FromContext(ctx).Info("updating source", "source", source.Spec)
	return k8sClient.Update(ctx, source)
}

// MigrateInstrumentationLabelToSource checks if an object is enabled by the legacy label.
// If so, it will create (or update) a Source for the object.
func MigrateInstrumentationLabelToSource(ctx context.Context, k8sClient client.Client, obj client.Object, kind workload.WorkloadKind) error {
	logger := log.FromContext(ctx)

	if workload.IsObjectLabeledForInstrumentation(obj) {
		logger.Info("legacy instrumentation label is deprecated; creating source for workload", "name", obj.GetName(), "namespace", obj.GetNamespace(), "kind", kind)
		err := CreateOrUpdateSourceForObject(ctx, k8sClient, obj, kind, false)
		if err != nil {
			return err
		}
	}

	return nil
}

// MigrateInstrumentationLabelToDisabledSource checks if an object is explicitly disabled by the legacy label.
// If so, it will create (or update) a disabled Source for the object.
func MigrateInstrumentationLabelToDisabledSource(ctx context.Context, k8sClient client.Client, obj client.Object, kind workload.WorkloadKind) error {
	logger := log.FromContext(ctx)

	if workload.IsInstrumentationDisabledExplicitly(obj) {
		logger.Info("legacy instrumentation label is deprecated; excluding source for workload", "name", obj.GetName(), "namespace", obj.GetNamespace(), "kind", kind)
		err := CreateOrUpdateSourceForObject(ctx, k8sClient, obj, kind, true)
		if err != nil {
			return err
		}
	}

	return nil
}
