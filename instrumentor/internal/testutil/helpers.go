package testutil

import (
	"github.com/odigos-io/odigos/common/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetOdigosInstrumentationEnabled[W client.Object](obj W) W {
	copy := obj.DeepCopyObject().(W)
	copy.SetLabels(map[string]string{consts.OdigosInstrumentationLabel: consts.InstrumentationEnabled})
	return copy
}

func SetOdigosInstrumentationDisabled[W client.Object](obj W) W {
	copy := obj.DeepCopyObject().(W)
	copy.SetLabels(map[string]string{consts.OdigosInstrumentationLabel: consts.InstrumentationDisabled})
	return copy
}

func DeleteOdigosInstrumentationLabel[W client.Object](obj W) W {
	copy := obj.DeepCopyObject().(W)
	delete(copy.GetLabels(), consts.OdigosInstrumentationLabel)
	return copy
}

func SetReportedNameAnnotation[W client.Object](obj W, reportedName string) W {
	copy := obj.DeepCopyObject().(W)
	copy.SetAnnotations(map[string]string{consts.OdigosReportedNameAnnotation: reportedName})
	return copy
}

func SetDeploymentContainerEnv(obj *appsv1.Deployment, envName string, envValue string) *appsv1.Deployment {
	copy := obj.DeepCopy()
	envVar := corev1.EnvVar{Name: envName, Value: envValue}
	if len(copy.Spec.Template.Spec.Containers[0].Env) == 0 {
		copy.Spec.Template.Spec.Containers[0].Env = append(copy.Spec.Template.Spec.Containers[0].Env, envVar)
	} else {
		copy.Spec.Template.Spec.Containers[0].Env[0] = envVar
	}

	return copy
}

func IsDeploymentSingleContainerSingleEnv(obj *appsv1.Deployment, envName string, envValue string) bool {
	return len(obj.Spec.Template.Spec.Containers) == 1 &&
		len(obj.Spec.Template.Spec.Containers[0].Env) == 1 &&
		obj.Spec.Template.Spec.Containers[0].Env[0].Name == envName &&
		obj.Spec.Template.Spec.Containers[0].Env[0].Value == envValue
}
