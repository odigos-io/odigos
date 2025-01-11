package instrumentation_instance

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
)

type InstrumentationInstanceOption interface {
	applyInstrumentationInstance(odigosv1.InstrumentationInstanceStatus) odigosv1.InstrumentationInstanceStatus
}

type updateInstrumentationInstanceStatusOpt func(odigosv1.InstrumentationInstanceStatus) odigosv1.InstrumentationInstanceStatus

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

func UpdateInstrumentationInstanceStatus(ctx context.Context, owner client.Object, containerName string, kubeClient client.Client, instrumentedAppName string, pid int, scheme *runtime.Scheme, options ...InstrumentationInstanceOption) error {
	instrumentationInstanceName := InstrumentationInstanceName(owner.GetName(), pid)
	updatedInstance := &odigosv1.InstrumentationInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "InstrumentationInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instrumentationInstanceName,
			Namespace: owner.GetNamespace(),
			Labels: map[string]string{
				consts.InstrumentedAppNameLabel: instrumentedAppName,
			},
		},
		Spec: odigosv1.InstrumentationInstanceSpec{
			ContainerName: containerName,
		},
	}

	err := controllerutil.SetControllerReference(owner, updatedInstance, scheme)
	if err != nil {
		return err
	}

	if err = kubeClient.Create(ctx, updatedInstance); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
		err := kubeClient.Get(ctx, client.ObjectKeyFromObject(updatedInstance), updatedInstance)
		if err != nil {
			return err
		}
	}

	updatedInstance.Status = updateInstrumentationInstanceStatus(updatedInstance.Status, options...)

	err = kubeClient.Status().Update(ctx, updatedInstance)
	if err != nil {
		return err
	}
	return nil
}
