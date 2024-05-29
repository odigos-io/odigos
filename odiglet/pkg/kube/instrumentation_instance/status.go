package instrumentation_instance

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

type InstrumentationInstanceConfig struct {
	healthy *bool
	nonIdentifyingAttributes []odigosv1.Attribute
	message string
	reason string
}

type InstrumentationInstanceOption interface {
	applyInstrumentationInstance(InstrumentationInstanceConfig) InstrumentationInstanceConfig
}

type fnOpt func(InstrumentationInstanceConfig) InstrumentationInstanceConfig

func (o fnOpt) applyInstrumentationInstance(c InstrumentationInstanceConfig) InstrumentationInstanceConfig { return o(c) }

func WithHealthy(healthy *bool) InstrumentationInstanceOption {
	return fnOpt(func(c InstrumentationInstanceConfig) InstrumentationInstanceConfig {
		c.healthy = healthy
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
		Healthy: c.healthy,
		NonIdentifyingAttributes: c.nonIdentifyingAttributes,
		Message: c.message,
		Reason: c.reason,
	}
}

func PersistInstrumentationInstanceStatus(ctx context.Context, owner client.Object, kubeClient client.Client, instrumentedAppName string, scheme *runtime.Scheme, options ...InstrumentationInstanceOption) error {
	updatedInstance := &odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: owner.GetName(),
			Namespace: owner.GetNamespace(),
			Labels: map[string]string{
				consts.InstrumentedAppNameLabel: instrumentedAppName,
			},
		},
		Status: *newInstrumentationInstanceStatus(options...),
	}

	err := controllerutil.SetControllerReference(owner, updatedInstance, scheme)
	if err != nil {
		return err
	}

	instrumentationInstanceBytes, err := yaml.Marshal(updatedInstance)
	if err != nil {
		return err
	}

	err = kubeClient.Patch(ctx, updatedInstance, client.RawPatch(types.ApplyPatchType, instrumentationInstanceBytes))
	if err != nil {
		return err
	}

	// TODO: remove this log
	log.Logger.V(0).Info("Updated instrumentation instance status", "name", owner.GetName(), "namespace", owner.GetNamespace())

	return nil
}

// func mutateInstrumentationStatus(current, desired *odigosv1.InstrumentationInstanceStatus) error {
// 	if current == nil || desired == nil {
// 		return nil
// 	}

// 	if desired.Healthy != nil {
// 		current.Healthy = desired.Healthy
// 		current.LastStatusTime.Time = metav1.Now().Time
// 		current.Message = ""
// 		current.Reason = ""
// 	}
// 	if len(desired.Message) > 0 {
// 		current.Message = desired.Message
// 	}
// 	if len(desired.Reason) > 0 {
// 		current.Reason = desired.Reason
// 	}
// 	if current.StartTime.Time.IsZero() {
// 		current.StartTime = metav1.Now()
// 	}
// 	if len(desired.NonIdentifyingAttributes) > 0 {
// 		current.NonIdentifyingAttributes = MergeAttributes(current.NonIdentifyingAttributes, desired.NonIdentifyingAttributes)
// 	}

// 	return nil
// }

// // MergeAttributes updates the current slice of attributes based on the desired slice
// // for each attribute in desired, if the key exists in current, the value is updated
// // if the key does not exist in current, the attribute is appended
// func MergeAttributes(current, desired []odigosv1.Attribute) []odigosv1.Attribute {
// 	// Create a map to track keys in the current slice
// 	currentMap := make(map[string]int)
// 	for i, attr := range current {
// 		currentMap[attr.Key] = i
// 	}

// 	// Iterate over the desired attributes
// 	for _, d := range desired {
// 		if index, exists := currentMap[d.Key]; exists {
// 			// Update the value if the key exists in current
// 			current[index].Value = d.Value
// 		} else {
// 			// Append the attribute if the key does not exist in current
// 			current = append(current, d)
// 		}
// 	}

// 	return current
// }