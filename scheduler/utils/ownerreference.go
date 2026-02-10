package utils

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetOwnerControllerToSchedulerDeployment(ctx context.Context, c client.Client, cg *odigosv1.CollectorsGroup, scheme *runtime.Scheme) error {
	schedluerDeployment := appsv1.Deployment{}
	err := c.Get(ctx, client.ObjectKey{Namespace: cg.GetNamespace(), Name: env.GetComponentDeploymentNameOrDefault(k8sconsts.SchedulerDeploymentName)}, &schedluerDeployment)
	if err != nil {
		return err
	}
	return ctrl.SetControllerReference(&schedluerDeployment, cg, scheme)
}
