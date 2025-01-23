package effecticeprofiles

import (
	"context"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stormcat24/protodep/pkg/logger"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Sync(ctx context.Context, k8sClient client.Client, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string) error {

	odigosNs := env.GetCurrentNamespace()
	effectiveConfig := &v1.ConfigMap{}

	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: consts.OdigosEffectiveConfigName}, effectiveConfig)
	if err != nil {
		// collectors group is created by the scheduler, after the first destination is added.
		// it is however possible that some reconciler (like deployment) triggered and the collectors group will be created shortly.
		return client.IgnoreNotFound(err)
	}

	err = syncGateway(&dests, &processors, &gatewayCollectorGroup, ctx, k8sClient, scheme, imagePullSecrets, odigosVersion, config)
	statusPatchString := commonconf.GetCollectorsGroupDeployedConditionsPatch(err)
	statusErr := k8sClient.Status().Patch(ctx, &gatewayCollectorGroup, client.RawPatch(types.MergePatchType, []byte(statusPatchString)))
	if statusErr != nil {
		logger.Error(statusErr, "Failed to patch collectors group status")
		// just log the error, do not fail the reconciliation
	}
	return err
}
