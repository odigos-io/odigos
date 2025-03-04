package instrumentation_instance

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
)

const (
	// label key used to associate the instrumentation instance with the owner pod
	ownerPodNameLabel = "ownerPodName"

	// maxInstrumentationInstancesPerPod is the maximum number of instrumentation instances that can be created per pod
	// this is required to bound the number of instrumentation instances that can be created.
	maxInstrumentationInstancesPerPod = 16
)

type InstrumentationInstanceOption interface {
	applyInstrumentationInstance(odigosv1.InstrumentationInstanceStatus) odigosv1.InstrumentationInstanceStatus
}

type updateInstrumentationInstanceStatusOpt func(odigosv1.InstrumentationInstanceStatus) odigosv1.InstrumentationInstanceStatus

//nolint:all
func (o updateInstrumentationInstanceStatusOpt) applyInstrumentationInstance(s odigosv1.InstrumentationInstanceStatus) odigosv1.InstrumentationInstanceStatus {
	return o(s)
}

// set Healthy and related fields in InstrumentationInstanceStatus
func WithHealthy(healthy *bool, reason string, message *string) InstrumentationInstanceOption {
	return updateInstrumentationInstanceStatusOpt(func(s odigosv1.InstrumentationInstanceStatus) odigosv1.InstrumentationInstanceStatus {
		s.Healthy = healthy
		s.Reason = reason
		if message != nil {
			s.Message = *message
		} else {
			s.Message = ""
		}
		return s
	})
}

func WithAttributes(identifying []odigosv1.Attribute, nonIdentifying []odigosv1.Attribute) InstrumentationInstanceOption {
	return updateInstrumentationInstanceStatusOpt(func(s odigosv1.InstrumentationInstanceStatus) odigosv1.InstrumentationInstanceStatus {
		s.IdentifyingAttributes = identifying
		s.NonIdentifyingAttributes = nonIdentifying
		return s
	})
}

//nolint:all
func updateInstrumentationInstanceStatus(status odigosv1.InstrumentationInstanceStatus, options ...InstrumentationInstanceOption) odigosv1.InstrumentationInstanceStatus {
	for _, option := range options {
		status = option.applyInstrumentationInstance(status)
	}
	status.LastStatusTime = metav1.Now()
	return status
}

func InstrumentationInstanceName(ownerName string, pid int) string {
	return fmt.Sprintf("%s-%d", ownerName, pid)
}

func UpdateInstrumentationInstanceStatus(ctx context.Context, owner client.Object, containerName string, kubeClient client.Client,
	instrumentedAppName string, pid int, scheme *runtime.Scheme, options ...InstrumentationInstanceOption) error {
	instrumentationInstanceName := InstrumentationInstanceName(owner.GetName(), pid)
	instance := odigosv1.InstrumentationInstance{}

	err := kubeClient.Get(ctx,
		client.ObjectKey{Namespace: owner.GetNamespace(), Name: instrumentationInstanceName},
		&instance,
	)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		// check if we can create a new instance
		if instrumentationInstancesCountReachedLimit(ctx, owner, kubeClient) {
			return fmt.Errorf("instrumentation instances count per pod is over the limit of %d", maxInstrumentationInstancesPerPod)
		}

		// create new instance
		instance = odigosv1.InstrumentationInstance{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "odigos.io/v1alpha1",
				Kind:       "InstrumentationInstance",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      instrumentationInstanceName,
				Namespace: owner.GetNamespace(),
				Labels: map[string]string{
					consts.InstrumentedAppNameLabel: instrumentedAppName,
					ownerPodNameLabel:               owner.GetName(),
				},
			},
			Spec: odigosv1.InstrumentationInstanceSpec{
				ContainerName: containerName,
			},
		}

		err := controllerutil.SetControllerReference(owner, &instance, scheme)
		if err != nil {
			return err
		}

		err = kubeClient.Create(ctx, &instance)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	instance.Status = updateInstrumentationInstanceStatus(instance.Status, options...)
	err = kubeClient.Status().Update(ctx, &instance)

	if err != nil {
		if apierrors.IsConflict(err) {
			return retry.RetryOnConflict(retry.DefaultRetry, func() error {
				// Re-fetch latest version to avoid conflict errors
				fmt.Println("retrying")
				instance := odigosv1.InstrumentationInstance{}
				err := kubeClient.Get(ctx, client.ObjectKey{Namespace: owner.GetNamespace(), Name: instrumentationInstanceName}, &instance)
				if err != nil {
					return err
				}

				instance.Status = updateInstrumentationInstanceStatus(instance.Status, options...)

				return kubeClient.Status().Update(ctx, &instance)
			})
		}
		return err
	}

	return nil
}

func instrumentationInstancesCountReachedLimit(ctx context.Context, owner client.Object, kubeClient client.Client) bool {
	instances := &odigosv1.InstrumentationInstanceList{}
	err := kubeClient.List(ctx, instances,
		client.InNamespace(owner.GetNamespace()),
		client.MatchingLabels{ownerPodNameLabel: owner.GetName()},
	)
	if err != nil {
		return true
	}
	return len(instances.Items) >= maxInstrumentationInstancesPerPod
}
