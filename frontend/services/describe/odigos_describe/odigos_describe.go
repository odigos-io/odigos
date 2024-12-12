package odigos_describe

import (
	"context"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	describe_utils "github.com/odigos-io/odigos/frontend/services/describe/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/odigos"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

type OdigosService struct{}

func GetOdigosDescription(ctx context.Context) (*model.OdigosAnalyze, error) {

	namespace := env.GetCurrentNamespace()
	desc, err := describe.DescribeOdigos(ctx, kube.DefaultClient, kube.DefaultClient.OdigosClient, namespace)
	if err != nil {
		return nil, err
	}

	return convertOdigosToGQL(desc), nil
}

func convertOdigosToGQL(odigos *odigos.OdigosAnalyze) *model.OdigosAnalyze {
	if odigos == nil {
		return nil
	}
	return &model.OdigosAnalyze{
		OdigosVersion:        describe_utils.ConvertEntityPropertyToGQL(&odigos.OdigosVersion),
		NumberOfDestinations: odigos.NumberOfDestinations,
		NumberOfSources:      odigos.NumberOfSources,
		ClusterCollector:     convertClusterCollectorToGQL(&odigos.ClusterCollector),
		NodeCollector:        convertNodeCollectorToGQL(&odigos.NodeCollector),
		IsSettled:            odigos.IsSettled,
		HasErrors:            odigos.HasErrors,
	}
}

func convertClusterCollectorToGQL(collector *odigos.ClusterCollectorAnalyze) *model.ClusterCollectorAnalyze {
	return &model.ClusterCollectorAnalyze{
		Enabled:              describe_utils.ConvertEntityPropertyToGQL(&collector.Enabled),
		CollectorGroup:       describe_utils.ConvertEntityPropertyToGQL(&collector.CollectorGroup),
		Deployed:             describe_utils.ConvertEntityPropertyToGQL(collector.Deployed),
		DeployedError:        describe_utils.ConvertEntityPropertyToGQL(collector.DeployedError),
		CollectorReady:       describe_utils.ConvertEntityPropertyToGQL(collector.CollectorReady),
		DeploymentCreated:    describe_utils.ConvertEntityPropertyToGQL(&collector.DeploymentCreated),
		ExpectedReplicas:     describe_utils.ConvertEntityPropertyToGQL(collector.ExpectedReplicas),
		HealthyReplicas:      describe_utils.ConvertEntityPropertyToGQL(collector.HealthyReplicas),
		FailedReplicas:       describe_utils.ConvertEntityPropertyToGQL(collector.FailedReplicas),
		FailedReplicasReason: describe_utils.ConvertEntityPropertyToGQL(collector.FailedReplicasReason),
	}
}

func convertNodeCollectorToGQL(collector *odigos.NodeCollectorAnalyze) *model.NodeCollectorAnalyze {
	return &model.NodeCollectorAnalyze{
		Enabled:        describe_utils.ConvertEntityPropertyToGQL(&collector.Enabled),
		CollectorGroup: describe_utils.ConvertEntityPropertyToGQL(&collector.CollectorGroup),
		Deployed:       describe_utils.ConvertEntityPropertyToGQL(collector.Deployed),
		DeployedError:  describe_utils.ConvertEntityPropertyToGQL(collector.DeployedError),
		CollectorReady: describe_utils.ConvertEntityPropertyToGQL(collector.CollectorReady),
		DaemonSet:      describe_utils.ConvertEntityPropertyToGQL(&collector.DaemonSet),
		DesiredNodes:   describe_utils.ConvertEntityPropertyToGQL(collector.DesiredNodes),
		CurrentNodes:   describe_utils.ConvertEntityPropertyToGQL(collector.CurrentNodes),
		UpdatedNodes:   describe_utils.ConvertEntityPropertyToGQL(collector.UpdatedNodes),
		AvailableNodes: describe_utils.ConvertEntityPropertyToGQL(collector.AvailableNodes),
	}
}
