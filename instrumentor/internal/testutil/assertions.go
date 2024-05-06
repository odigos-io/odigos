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

func AssertInstrumentedApplicationDeleted(ctx context.Context, k8sClient client.Client, instrumentedApplication *odigosv1.InstrumentedApplication) {
	key := client.ObjectKey{Namespace: instrumentedApplication.GetNamespace(), Name: instrumentedApplication.GetName()}
	Eventually(func() bool {
		var runtimeDetails odigosv1.InstrumentedApplication
		err := k8sClient.Get(ctx, key, &runtimeDetails)
		return apierrors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())
}

func AssertInstrumentedApplicationRetained(ctx context.Context, k8sClient client.Client, instrumentedApplication *odigosv1.InstrumentedApplication) {
	key := client.ObjectKey{Namespace: instrumentedApplication.GetNamespace(), Name: instrumentedApplication.GetName()}
	Consistently(func() bool {
		var runtimeDetails odigosv1.InstrumentedApplication
		err := k8sClient.Get(ctx, key, &runtimeDetails)
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
	return found
}
