package odigos_describe

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/odigos"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
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
		OdigosVersion:        convertEntityPropertyToGQL(&odigos.OdigosVersion),
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
		Enabled:              convertEntityPropertyToGQL(&collector.Enabled),
		CollectorGroup:       convertEntityPropertyToGQL(&collector.CollectorGroup),
		Deployed:             convertEntityPropertyToGQL(collector.Deployed),
		DeployedError:        convertEntityPropertyToGQL(collector.DeployedError),
		CollectorReady:       convertEntityPropertyToGQL(collector.CollectorReady),
		DeploymentCreated:    convertEntityPropertyToGQL(&collector.DeploymentCreated),
		ExpectedReplicas:     convertEntityPropertyToGQL(collector.ExpectedReplicas),
		HealthyReplicas:      convertEntityPropertyToGQL(collector.HealthyReplicas),
		FailedReplicas:       convertEntityPropertyToGQL(collector.FailedReplicas),
		FailedReplicasReason: convertEntityPropertyToGQL(collector.FailedReplicasReason),
	}
}

func convertNodeCollectorToGQL(collector *odigos.NodeCollectorAnalyze) *model.NodeCollectorAnalyze {
	return &model.NodeCollectorAnalyze{
		Enabled:        convertEntityPropertyToGQL(&collector.Enabled),
		CollectorGroup: convertEntityPropertyToGQL(&collector.CollectorGroup),
		Deployed:       convertEntityPropertyToGQL(collector.Deployed),
		DeployedError:  convertEntityPropertyToGQL(collector.DeployedError),
		CollectorReady: convertEntityPropertyToGQL(collector.CollectorReady),
		DaemonSet:      convertEntityPropertyToGQL(&collector.DaemonSet),
		DesiredNodes:   convertEntityPropertyToGQL(collector.DesiredNodes),
		CurrentNodes:   convertEntityPropertyToGQL(collector.CurrentNodes),
		UpdatedNodes:   convertEntityPropertyToGQL(collector.UpdatedNodes),
		AvailableNodes: convertEntityPropertyToGQL(collector.AvailableNodes),
	}
}

func convertEntityPropertyToGQL(prop *properties.EntityProperty) *model.EntityProperty {
	if prop == nil {
		return nil
	}

	var value string
	if strValue, ok := prop.Value.(string); ok {
		value = strValue
	} else {
		value = fmt.Sprintf("%v", prop.Value)
	}

	var status *string
	if prop.Status != "" {
		statusStr := string(prop.Status)
		status = &statusStr
	}

	var explain *string
	if prop.Explain != "" {
		explain = &prop.Explain
	}

	return &model.EntityProperty{
		Name:    prop.Name,
		Value:   value,
		Status:  status,
		Explain: explain,
	}
}
