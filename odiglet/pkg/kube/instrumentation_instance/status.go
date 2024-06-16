package instrumentation_instance

import (
	"context"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type InstrumentationInstanceConfig struct {
	healthy                  *bool
	identifyingAttributes    []odigosv1.Attribute
	nonIdentifyingAttributes []odigosv1.Attribute
	message                  string
	reason                   string
}

type InstrumentationInstanceOption interface {
	applyInstrumentationInstance(InstrumentationInstanceConfig) InstrumentationInstanceConfig
}

type fnOpt func(InstrumentationInstanceConfig) InstrumentationInstanceConfig

func (o fnOpt) applyInstrumentationInstance(c InstrumentationInstanceConfig) InstrumentationInstanceConfig {
	return o(c)
}

func WithHealthy(healthy *bool) InstrumentationInstanceOption {
	return fnOpt(func(c InstrumentationInstanceConfig) InstrumentationInstanceConfig {
		c.healthy = healthy
		return c
	})
}

func WithIdentifyingAttributes(attributes []odigosv1.Attribute) InstrumentationInstanceOption {
	return fnOpt(func(c InstrumentationInstanceConfig) InstrumentationInstanceConfig {
		c.identifyingAttributes = attributes
		return c
	})
}

func WithNonIdentifyingAttributes(attributes []odigosv1.Attribute) InstrumentationInstanceOption {
	return fnOpt(func(c InstrumentationInstanceConfig) InstrumentationInstanceConfig {
		c.nonIdentifyingAttributes = attributes
		return c
	})
}

func WithMessage(message string) InstrumentationInstanceOption {
	return fnOpt(func(c InstrumentationInstanceConfig) InstrumentationInstanceConfig {
		c.message = message
		return c
	})
}

func WithReason(reason string) InstrumentationInstanceOption {
	return fnOpt(func(c InstrumentationInstanceConfig) InstrumentationInstanceConfig {
		c.reason = reason
		return c
	})
}

func newInstrumentationInstanceConfig(options ...InstrumentationInstanceOption) InstrumentationInstanceConfig {
	var c InstrumentationInstanceConfig
	for _, option := range options {
		c = option.applyInstrumentationInstance(c)
	}
	return c
}

func newInstrumentationInstanceStatus(options ...InstrumentationInstanceOption) *odigosv1.InstrumentationInstanceStatus {
	c := newInstrumentationInstanceConfig(options...)
	return &odigosv1.InstrumentationInstanceStatus{
		Healthy:                  c.healthy,
		IdentifyingAttributes:    c.identifyingAttributes,
		NonIdentifyingAttributes: c.nonIdentifyingAttributes,
		Message:                  c.message,
		Reason:                   c.reason,
		LastStatusTime:           metav1.Now(),
	}
}

func instrumentationInstanceName(owner client.Object, pid int) string {
	return fmt.Sprintf("%s-%d", owner.GetName(), pid)
}

func PersistInstrumentationInstanceStatus(ctx context.Context, owner client.Object, kubeClient client.Client, instrumentedAppName string, pid int, scheme *runtime.Scheme, options ...InstrumentationInstanceOption) error {
	instrumentationInstanceName := instrumentationInstanceName(owner, pid)
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
		}}

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

	updatedInstance.Status = *newInstrumentationInstanceStatus(options...)

	err = kubeClient.Status().Update(ctx, updatedInstance)
	if err != nil {
		return err
	}
	return nil
}
