package testutil

import (
	"context"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 10
	duration = time.Second * 2
	interval = time.Millisecond * 250
)

func AssertInstrumentationConfigDeleted(ctx context.Context, k8sClient client.Client, instrumentationConfig *odigosv1.InstrumentationConfig) {
	key := client.ObjectKey{Namespace: instrumentationConfig.GetNamespace(), Name: instrumentationConfig.GetName()}
	Eventually(func() bool {
		var ic odigosv1.InstrumentationConfig
		err := k8sClient.Get(ctx, key, &ic)
		return apierrors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())
}

func AssertInstrumentationConfigRetained(ctx context.Context, k8sClient client.Client, instrumentationConfig *odigosv1.InstrumentationConfig) {
	key := client.ObjectKey{Namespace: instrumentationConfig.GetNamespace(), Name: instrumentationConfig.GetName()}
	Consistently(func() bool {
		var ic odigosv1.InstrumentationConfig
		err := k8sClient.Get(ctx, key, &ic)
		return err == nil
	}, duration, interval).Should(BeTrue())
}

func AssertReportedNameAnnotationDeletedDeployment(ctx context.Context, k8sClient client.Client, dep *appsv1.Deployment) {
	key := client.ObjectKey{Namespace: dep.GetNamespace(), Name: dep.GetName()}
	Eventually(func() bool {
		var dep appsv1.Deployment
		err := k8sClient.Get(ctx, key, &dep)
		return isReportedNameDeleted(&dep, err)
	}, timeout, interval).Should(BeTrue())
}

func AssertReportedNameAnnotationDeletedDaemonSet(ctx context.Context, k8sClient client.Client, ds *appsv1.DaemonSet) {
	key := client.ObjectKey{Namespace: ds.GetNamespace(), Name: ds.GetName()}
	Eventually(func() bool {
		var ds appsv1.DaemonSet
		err := k8sClient.Get(ctx, key, &ds)
		return isReportedNameDeleted(&ds, err)
	}, timeout, interval).Should(BeTrue())
}

func AssertReportedNameAnnotationDeletedStatefulSet(ctx context.Context, k8sClient client.Client, sts *appsv1.StatefulSet) {
	key := client.ObjectKey{Namespace: sts.GetNamespace(), Name: sts.GetName()}
	Eventually(func() bool {
		var sts appsv1.StatefulSet
		err := k8sClient.Get(ctx, key, &sts)
		return isReportedNameDeleted(&sts, err)
	}, timeout, interval).Should(BeTrue())
}

func AssertDeploymentAnnotationRetained(ctx context.Context, k8sClient client.Client, dep *appsv1.Deployment, annotationKey string, annotationValue string) {
	key := client.ObjectKey{Namespace: dep.GetNamespace(), Name: dep.GetName()}
	Consistently(func() bool {
		var dep appsv1.Deployment
		err := k8sClient.Get(ctx, key, &dep)
		if err != nil {
			return false
		}
		value, foundAnnotation := dep.Annotations[annotationKey]
		return foundAnnotation && value == annotationValue
	}, duration, interval).Should(BeTrue())
}

func isReportedNameDeleted(obj client.Object, err error) bool {
	if err != nil {
		return false
	}
	_, found := obj.GetAnnotations()[consts.OdigosReportedNameAnnotation]
	return !found
}

func AssertDepContainerEnvRemainEmpty(ctx context.Context, k8sClient client.Client, dep *appsv1.Deployment) {
	key := client.ObjectKey{Namespace: dep.GetNamespace(), Name: dep.GetName()}
	Consistently(func() bool {
		var currentDeployment appsv1.Deployment
		err := k8sClient.Get(ctx, key, &currentDeployment)
		if err != nil {
			return false
		}
		for _, container := range currentDeployment.Spec.Template.Spec.Containers {
			if len(container.Env) > 0 {
				return false
			}
		}
		return true
	}, duration, interval).Should(BeTrue())
}

func AssertDepContainerSingleEnvBecomesEmpty(ctx context.Context, k8sClient client.Client, dep *appsv1.Deployment) {
	key := client.ObjectKey{Namespace: dep.GetNamespace(), Name: dep.GetName()}
	Eventually(func() bool {
		var currentDeployment appsv1.Deployment
		err := k8sClient.Get(ctx, key, &currentDeployment)
		if err != nil {
			return false
		}
		for _, container := range currentDeployment.Spec.Template.Spec.Containers {
			if len(container.Env) > 0 {
				return false
			}
		}
		return true
	}, duration, interval).Should(BeTrue())
}

func AssertDepContainerSingleEnv(ctx context.Context, k8sClient client.Client, dep *appsv1.Deployment, envName string, envValue string) {
	key := client.ObjectKey{Namespace: dep.GetNamespace(), Name: dep.GetName()}
	Eventually(func() bool {
		var currentDeployment appsv1.Deployment
		err := k8sClient.Get(ctx, key, &currentDeployment)
		if err != nil {
			return false
		}
		return IsDeploymentSingleContainerSingleEnv(&currentDeployment, envName, envValue)
	}, duration, interval).Should(BeTrue())
}

func AssertDepContainerSingleEnvRemainsSame(ctx context.Context, k8sClient client.Client, dep *appsv1.Deployment, envName string, envValue string) {
	key := client.ObjectKey{Namespace: dep.GetNamespace(), Name: dep.GetName()}
	Consistently(func() bool {
		var currentDeployment appsv1.Deployment
		err := k8sClient.Get(ctx, key, &currentDeployment)
		if err != nil {
			return false
		}
		return IsDeploymentSingleContainerSingleEnv(&currentDeployment, envName, envValue)
	}, duration, interval).Should(BeTrue())
}
