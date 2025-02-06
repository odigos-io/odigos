package source_describe

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	describe_utils "github.com/odigos-io/odigos/frontend/services/describe/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/source"
)

func GetSourceDescription(ctx context.Context, namespace string, kind string, name string) (*model.SourceAnalyze, error) {
	var desc *source.SourceAnalyze
	var err error

	switch kind {
	case "Deployment":
		desc, err = describe.DescribeDeployment(ctx, kube.DefaultClient.Interface, kube.DefaultClient.OdigosClient, namespace, name)
	case "DaemonSet":
		desc, err = describe.DescribeDaemonSet(ctx, kube.DefaultClient.Interface, kube.DefaultClient.OdigosClient, namespace, name)
	case "StatefulSet":
		desc, err = describe.DescribeStatefulSet(ctx, kube.DefaultClient.Interface, kube.DefaultClient.OdigosClient, namespace, name)
	default:
		return nil, fmt.Errorf("kind %s is not supported", kind)
	}

	if err != nil {
		return nil, err
	}

	gqlResponse := ConvertSourceAnalyzeToGQL(desc)
	return gqlResponse, nil
}

func ConvertSourceAnalyzeToGQL(analyze *source.SourceAnalyze) *model.SourceAnalyze {
	if analyze == nil {
		return nil
	}

	return &model.SourceAnalyze{
		Name:      describe_utils.ConvertEntityPropertyToGQL(&analyze.Name),
		Kind:      describe_utils.ConvertEntityPropertyToGQL(&analyze.Kind),
		Namespace: describe_utils.ConvertEntityPropertyToGQL(&analyze.Namespace),
		SourceObjects: &model.InstrumentationSourcesAnalyze{
			Instrumented:     describe_utils.ConvertEntityPropertyToGQL(&analyze.SourceObjectsAnalysis.Instrumented),
			Workload:         describe_utils.ConvertEntityPropertyToGQL(analyze.SourceObjectsAnalysis.Workload),
			Namespace:        describe_utils.ConvertEntityPropertyToGQL(analyze.SourceObjectsAnalysis.Namespace),
			InstrumentedText: describe_utils.ConvertEntityPropertyToGQL(&analyze.SourceObjectsAnalysis.InstrumentedText),
		},
		RuntimeInfo: convertRuntimeInfoToGQL(analyze.RuntimeInfo),
		OtelAgents: &model.OtelAgentsAnalyze{
			Created:    describe_utils.ConvertEntityPropertyToGQL(&analyze.OtelAgents.Created),
			CreateTime: describe_utils.ConvertEntityPropertyToGQL(analyze.OtelAgents.CreateTime),
			Containers: convertOtelAgentContainersToGQL(analyze.OtelAgents.Containers),
		},
		TotalPods:       analyze.TotalPods,
		PodsPhasesCount: analyze.PodsPhasesCount,
		Pods:            convertPodsToGQL(analyze.Pods),
	}
}

func convertRuntimeInfoToGQL(info *source.RuntimeInfoAnalyze) *model.RuntimeInfoAnalyze {
	if info == nil {
		return nil
	}
	return &model.RuntimeInfoAnalyze{
		Containers: convertRuntimeInfoContainersToGQL(info.Containers),
	}
}

func convertRuntimeInfoContainersToGQL(containers []source.ContainerRuntimeInfoAnalyze) []*model.ContainerRuntimeInfoAnalyze {
	gqlContainers := make([]*model.ContainerRuntimeInfoAnalyze, 0, len(containers))
	for _, container := range containers {
		gqlContainers = append(gqlContainers, &model.ContainerRuntimeInfoAnalyze{
			ContainerName:  describe_utils.ConvertEntityPropertyToGQL(&container.ContainerName),
			Language:       describe_utils.ConvertEntityPropertyToGQL(&container.Language),
			RuntimeVersion: describe_utils.ConvertEntityPropertyToGQL(&container.RuntimeVersion),
			EnvVars:        convertEntityPropertiesToGQL(container.EnvVars),
		})
	}
	return gqlContainers
}

func convertOtelAgentContainersToGQL(containers []source.ContainerAgentConfigAnalyze) []*model.ContainerAgentConfigAnalyze {
	gqlContainers := make([]*model.ContainerAgentConfigAnalyze, 0, len(containers))
	for _, container := range containers {
		gqlContainers = append(gqlContainers, &model.ContainerAgentConfigAnalyze{
			ContainerName:  describe_utils.ConvertEntityPropertyToGQL(&container.ContainerName),
			AgentEnabled:   describe_utils.ConvertEntityPropertyToGQL(&container.AgentEnabled),
			Reason:         describe_utils.ConvertEntityPropertyToGQL(container.Reason),
			Message:        describe_utils.ConvertEntityPropertyToGQL(container.Message),
			OtelDistroName: describe_utils.ConvertEntityPropertyToGQL(container.OtelDistroName),
		})
	}
	return gqlContainers
}

func convertEntityPropertiesToGQL(props []properties.EntityProperty) []*model.EntityProperty {
	gqlProps := make([]*model.EntityProperty, 0, len(props))
	for _, prop := range props {
		gqlProps = append(gqlProps, describe_utils.ConvertEntityPropertyToGQL(&prop))
	}
	return gqlProps
}

func convertPodsToGQL(pods []source.PodAnalyze) []*model.PodAnalyze {
	gqlPods := make([]*model.PodAnalyze, 0, len(pods))
	for _, pod := range pods {
		gqlPods = append(gqlPods, &model.PodAnalyze{
			PodName:    describe_utils.ConvertEntityPropertyToGQL(&pod.PodName),
			NodeName:   describe_utils.ConvertEntityPropertyToGQL(&pod.NodeName),
			Phase:      describe_utils.ConvertEntityPropertyToGQL(&pod.Phase),
			Containers: convertPodContainersToGQL(pod.Containers),
		})
	}
	return gqlPods
}

func convertPodContainersToGQL(containers []source.PodContainerAnalyze) []*model.PodContainerAnalyze {
	gqlContainers := make([]*model.PodContainerAnalyze, 0, len(containers))
	for _, container := range containers {
		gqlContainers = append(gqlContainers, &model.PodContainerAnalyze{
			ContainerName:            describe_utils.ConvertEntityPropertyToGQL(&container.ContainerName),
			ActualDevices:            describe_utils.ConvertEntityPropertyToGQL(&container.ActualDevices),
			InstrumentationInstances: convertInstrumentationInstancesToGQL(container.InstrumentationInstances),
		})
	}
	return gqlContainers
}

func convertInstrumentationInstancesToGQL(instances []source.InstrumentationInstanceAnalyze) []*model.InstrumentationInstanceAnalyze {
	gqlInstances := make([]*model.InstrumentationInstanceAnalyze, 0, len(instances))
	for _, instance := range instances {
		gqlInstances = append(gqlInstances, &model.InstrumentationInstanceAnalyze{
			Healthy:               describe_utils.ConvertEntityPropertyToGQL(&instance.Healthy),
			Message:               describe_utils.ConvertEntityPropertyToGQL(instance.Message),
			IdentifyingAttributes: convertEntityPropertiesToGQL(instance.IdentifyingAttributes),
		})
	}
	return gqlInstances
}
