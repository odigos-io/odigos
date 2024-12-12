package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
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
		Name:      convertEntityPropertyToGQL(&analyze.Name),
		Kind:      convertEntityPropertyToGQL(&analyze.Kind),
		Namespace: convertEntityPropertyToGQL(&analyze.Namespace),
		Labels: &model.InstrumentationLabelsAnalyze{
			Instrumented:     convertEntityPropertyToGQL(&analyze.Labels.Instrumented),
			Workload:         convertEntityPropertyToGQL(analyze.Labels.Workload),
			Namespace:        convertEntityPropertyToGQL(analyze.Labels.Namespace),
			InstrumentedText: convertEntityPropertyToGQL(&analyze.Labels.InstrumentedText),
		},
		InstrumentationConfig: &model.InstrumentationConfigAnalyze{
			Created:    convertEntityPropertyToGQL(&analyze.InstrumentationConfig.Created),
			CreateTime: convertEntityPropertyToGQL(analyze.InstrumentationConfig.CreateTime),
		},
		RuntimeInfo: convertRuntimeInfoToGQL(analyze.RuntimeInfo),
		InstrumentedApplication: &model.InstrumentedApplicationAnalyze{
			Created:    convertEntityPropertyToGQL(&analyze.InstrumentedApplication.Created),
			CreateTime: convertEntityPropertyToGQL(analyze.InstrumentedApplication.CreateTime),
			Containers: convertRuntimeInfoContainersToGQL(analyze.InstrumentedApplication.Containers),
		},
		InstrumentationDevice: &model.InstrumentationDeviceAnalyze{
			StatusText: convertEntityPropertyToGQL(&analyze.InstrumentationDevice.StatusText),
			Containers: convertWorkloadManifestContainersToGQL(analyze.InstrumentationDevice.Containers),
		},
		TotalPods:       analyze.TotalPods,
		PodsPhasesCount: analyze.PodsPhasesCount,
		Pods:            convertPodsToGQL(analyze.Pods),
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

func convertRuntimeInfoToGQL(info *source.RuntimeInfoAnalyze) *model.RuntimeInfoAnalyze {
	if info == nil {
		return nil
	}
	return &model.RuntimeInfoAnalyze{
		Generation: convertEntityPropertyToGQL(&info.Generation),
		Containers: convertRuntimeInfoContainersToGQL(info.Containers),
	}
}

func convertRuntimeInfoContainersToGQL(containers []source.ContainerRuntimeInfoAnalyze) []*model.ContainerRuntimeInfoAnalyze {
	gqlContainers := make([]*model.ContainerRuntimeInfoAnalyze, 0, len(containers))
	for _, container := range containers {
		gqlContainers = append(gqlContainers, &model.ContainerRuntimeInfoAnalyze{
			ContainerName:  convertEntityPropertyToGQL(&container.ContainerName),
			Language:       convertEntityPropertyToGQL(&container.Language),
			RuntimeVersion: convertEntityPropertyToGQL(&container.RuntimeVersion),
			EnvVars:        convertEntityPropertiesToGQL(container.EnvVars),
		})
	}
	return gqlContainers
}

func convertEntityPropertiesToGQL(props []properties.EntityProperty) []*model.EntityProperty {
	gqlProps := make([]*model.EntityProperty, 0, len(props))
	for _, prop := range props {
		gqlProps = append(gqlProps, convertEntityPropertyToGQL(&prop))
	}
	return gqlProps
}

func convertWorkloadManifestContainersToGQL(containers []source.ContainerWorkloadManifestAnalyze) []*model.ContainerWorkloadManifestAnalyze {
	gqlContainers := make([]*model.ContainerWorkloadManifestAnalyze, 0, len(containers))
	for _, container := range containers {
		gqlContainers = append(gqlContainers, &model.ContainerWorkloadManifestAnalyze{
			ContainerName: convertEntityPropertyToGQL(&container.ContainerName),
			Devices:       convertEntityPropertyToGQL(&container.Devices),
			OriginalEnv:   convertEntityPropertiesToGQL(container.OriginalEnv),
		})
	}
	return gqlContainers
}

func convertPodsToGQL(pods []source.PodAnalyze) []*model.PodAnalyze {
	gqlPods := make([]*model.PodAnalyze, 0, len(pods))
	for _, pod := range pods {
		gqlPods = append(gqlPods, &model.PodAnalyze{
			PodName:    convertEntityPropertyToGQL(&pod.PodName),
			NodeName:   convertEntityPropertyToGQL(&pod.NodeName),
			Phase:      convertEntityPropertyToGQL(&pod.Phase),
			Containers: convertPodContainersToGQL(pod.Containers),
		})
	}
	return gqlPods
}

func convertPodContainersToGQL(containers []source.PodContainerAnalyze) []*model.PodContainerAnalyze {
	gqlContainers := make([]*model.PodContainerAnalyze, 0, len(containers))
	for _, container := range containers {
		gqlContainers = append(gqlContainers, &model.PodContainerAnalyze{
			ContainerName:            convertEntityPropertyToGQL(&container.ContainerName),
			ActualDevices:            convertEntityPropertyToGQL(&container.ActualDevices),
			InstrumentationInstances: convertInstrumentationInstancesToGQL(container.InstrumentationInstances),
		})
	}
	return gqlContainers
}

func convertInstrumentationInstancesToGQL(instances []source.InstrumentationInstanceAnalyze) []*model.InstrumentationInstanceAnalyze {
	gqlInstances := make([]*model.InstrumentationInstanceAnalyze, 0, len(instances))
	for _, instance := range instances {
		gqlInstances = append(gqlInstances, &model.InstrumentationInstanceAnalyze{
			Healthy:               convertEntityPropertyToGQL(&instance.Healthy),
			Message:               convertEntityPropertyToGQL(instance.Message),
			IdentifyingAttributes: convertEntityPropertiesToGQL(instance.IdentifyingAttributes),
		})
	}
	return gqlInstances
}
