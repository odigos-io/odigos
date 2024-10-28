package describe

import (
	"context"
	"fmt"
	"strings"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/source"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func printWorkloadManifestInfo(analyze *source.SourceAnalyze, sb *strings.Builder) bool {
	printProperty(sb, 0, &analyze.Name)
	printProperty(sb, 0, &analyze.Kind)
	printProperty(sb, 0, &analyze.Namespace)

	sb.WriteString("Labels:\n")
	printProperty(sb, 1, &analyze.Labels.Instrumented)
	printProperty(sb, 1, analyze.Labels.Workload)
	printProperty(sb, 1, analyze.Labels.Namespace)
	printProperty(sb, 1, &analyze.Labels.InstrumentedText)

	return analyze.Labels.Instrumented.Value.(bool)
}

func printInstrumentationConfigInfo(analyze *source.SourceAnalyze, sb *strings.Builder) {
	describeText(sb, 0, "\nInstrumentation Config:")
	printProperty(sb, 1, &analyze.InstrumentationConfig.Created)
	printProperty(sb, 1, analyze.InstrumentationConfig.CreateTime)
}

func printRuntimeDetails(analyze *source.SourceAnalyze, sb *strings.Builder) {
	describeText(sb, 0, "\nRuntime Inspection Details (new):")

	if analyze.RuntimeInfo == nil {
		describeText(sb, 1, "No runtime details")
		return
	}

	printProperty(sb, 1, &analyze.RuntimeInfo.Generation)
	describeText(sb, 1, "Detected Containers:")
	for _, container := range analyze.RuntimeInfo.Containers {
		printProperty(sb, 2, &container.ContainerName)
		printProperty(sb, 3, &container.Language)
		printProperty(sb, 3, &container.RuntimeVersion)
		if len(container.EnvVars) > 0 {
			describeText(sb, 3, "Relevant Environment Variables:")
			for _, envVar := range container.EnvVars {
				describeText(sb, 4, fmt.Sprintf("%s: %s", envVar.Name, envVar.Value))
			}
		}
	}
}

func printInstrumentedApplicationInfo(analyze *source.SourceAnalyze, sb *strings.Builder) {

	describeText(sb, 0, "\nRuntime Inspection Details (old):")
	printProperty(sb, 1, &analyze.InstrumentedApplication.Created)
	printProperty(sb, 1, analyze.InstrumentedApplication.CreateTime)

	printProperty(sb, 1, &analyze.RuntimeInfo.Generation)
	describeText(sb, 1, "Detected Containers:")
	for _, container := range analyze.RuntimeInfo.Containers {
		printProperty(sb, 2, &container.ContainerName)
		printProperty(sb, 3, &container.Language)
		printProperty(sb, 3, &container.RuntimeVersion)
		if len(container.EnvVars) > 0 {
			describeText(sb, 3, "Relevant Environment Variables:")
			for _, envVar := range container.EnvVars {
				describeText(sb, 4, fmt.Sprintf("%s: %s", envVar.Name, envVar.Value))
			}
		}
	}
}

func printAppliedInstrumentationDeviceInfo(analyze *source.SourceAnalyze, workloadObj *source.K8sSourceObject, instrumented bool, sb *strings.Builder) map[string][]string {

	describeText(sb, 0, "\nInstrumentation Device:")
	printProperty(sb, 1, &analyze.InstrumentationDevice.StatusText)
	describeText(sb, 1, "Containers:")
	for _, container := range analyze.InstrumentationDevice.Containers {
		printProperty(sb, 2, &container.ContainerName)
		printProperty(sb, 3, &container.Devices)
		if len(container.OriginalEnv) > 0 {
			describeText(sb, 3, "Original Environment Variables:")
			for _, envVar := range container.OriginalEnv {
				printProperty(sb, 4, &envVar)
			}
		}
	}

	containerNameToExpectedDevices := make(map[string][]string)
	for _, container := range workloadObj.PodTemplateSpec.Spec.Containers {
		odigosDevices := make([]string, 0)
		for resourceName := range container.Resources.Limits {
			deviceName, found := strings.CutPrefix(resourceName.String(), common.OdigosResourceNamespace+"/")
			if found {
				odigosDevices = append(odigosDevices, deviceName)
			}
		}
		containerNameToExpectedDevices[container.Name] = odigosDevices
	}

	return containerNameToExpectedDevices
}

func printPodContainerInstrumentationInstancesInfo(instances []*odigosv1.InstrumentationInstance, sb *strings.Builder) {
	if len(instances) == 0 {
		sb.WriteString("    No instrumentation instances\n")
		return
	}

	sb.WriteString("    Instrumentation Instances:\n")
	for _, instance := range instances {
		unhealthy := false
		healthyText := "unknown"
		if instance.Status.Healthy != nil {
			if *instance.Status.Healthy {
				healthyText = wrapTextInGreen("true")
			} else {
				healthyText = wrapTextInRed("false")
				unhealthy = true
			}
		}
		sb.WriteString(fmt.Sprintf("    - Healthy: %s\n", healthyText))
		if instance.Status.Message != "" {
			sb.WriteString(fmt.Sprintf("      Message: %s\n", instance.Status.Message))
		}
		if instance.Status.Reason != "" && instance.Status.Reason != string(common.AgentHealthStatusHealthy) {
			sb.WriteString(fmt.Sprintf("      Reason: %s\n", instance.Status.Reason))
		}
		if unhealthy {
			sb.WriteString("      Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#7-instrumentation-instance-unhealthy\n")
		}
	}
}

func printPodContainerInfo(pod *corev1.Pod, container *corev1.Container, instrumentationInstances *odigosv1.InstrumentationInstanceList, containerNameToExpectedDevices map[string][]string, sb *strings.Builder) {
	instrumentationDevices := make([]string, 0)
	sb.WriteString(fmt.Sprintf("  - Container Name: %s\n", container.Name))
	for resourceName := range container.Resources.Limits {
		deviceName, found := strings.CutPrefix(resourceName.String(), common.OdigosResourceNamespace+"/")
		if found {
			instrumentationDevices = append(instrumentationDevices, deviceName)
		}
	}
	expectedDevices, foundExpectedDevices := containerNameToExpectedDevices[container.Name]
	if len(instrumentationDevices) == 0 {
		isMatch := !foundExpectedDevices || len(expectedDevices) == 0
		sb.WriteString(wrapTextSuccessOfFailure("    No instrumentation devices", isMatch) + "\n")
		if !isMatch {
			sb.WriteString("      Expected Devices: " + strings.Join(expectedDevices, ", ") + "\n")
			sb.WriteString("      Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#6-pods-instrumentation-devices-not-matching-manifest\n")
		}
	} else {
		actualDevicesText := strings.Join(instrumentationDevices, ", ")
		expectedDevicesText := strings.Join(expectedDevices, ", ")
		isMatch := actualDevicesText == expectedDevicesText
		sb.WriteString("    Instrumentation Devices: " + wrapTextSuccessOfFailure(actualDevicesText, isMatch) + "\n")
		if !isMatch {
			sb.WriteString("      Expected Devices: " + expectedDevicesText + "\n")
			sb.WriteString("      Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#6-pods-instrumentation-devices-not-matching-manifest\n")
		}
	}

	// find the instrumentation instances for this pod
	thisPodInstrumentationInstances := make([]*odigosv1.InstrumentationInstance, 0)
	for _, instance := range instrumentationInstances.Items {
		if len(instance.OwnerReferences) != 1 || instance.OwnerReferences[0].Kind != "Pod" {
			continue
		}
		if instance.OwnerReferences[0].Name != pod.GetName() {
			continue
		}
		if instance.Spec.ContainerName != container.Name {
			continue
		}
		thisPodInstrumentationInstances = append(thisPodInstrumentationInstances, &instance)
	}
	printPodContainerInstrumentationInstancesInfo(thisPodInstrumentationInstances, sb)
}

func printSinglePodInfo(pod *corev1.Pod, instrumentationInstances *odigosv1.InstrumentationInstanceList, containerNameToExpectedDevices map[string][]string, sb *strings.Builder) {
	sb.WriteString(fmt.Sprintf("\n  Pod Name: %s\n", pod.GetName()))
	sb.WriteString(fmt.Sprintf("  Pod Phase: %s\n", pod.Status.Phase))
	sb.WriteString(fmt.Sprintf("  Pod Node Name: %s\n", pod.Spec.NodeName))
	sb.WriteString("  Containers:\n")
	for _, container := range pod.Spec.Containers {
		printPodContainerInfo(pod, &container, instrumentationInstances, containerNameToExpectedDevices, sb)
	}
}

func printPodsInfo(pods *corev1.PodList, instrumentationInstances *odigosv1.InstrumentationInstanceList, containerNameToExpectedDevices map[string][]string, sb *strings.Builder) {
	podsStatuses := make(map[corev1.PodPhase]int)
	for _, pod := range pods.Items {
		podsStatuses[pod.Status.Phase]++
	}
	podPhasesTexts := make([]string, 0)
	for phase, count := range podsStatuses {
		podPhasesTexts = append(podPhasesTexts, fmt.Sprintf("%s %d", phase, count))
	}
	podPhasesText := strings.Join(podPhasesTexts, ", ")
	sb.WriteString(fmt.Sprintf("\nPods (Total %d, %s):\n", len(pods.Items), podPhasesText))
	for _, pod := range pods.Items {
		printSinglePodInfo(&pod, instrumentationInstances, containerNameToExpectedDevices, sb)
	}
}

func PrintDescribeSource(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, workloadObj *source.K8sSourceObject) string {
	var sb strings.Builder

	resources, err := source.GetRelevantSourceResources(ctx, kubeClient, odigosClient, workloadObj)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error: %v\n", err))
		return sb.String()
	}

	analyze := source.AnalyzeSource(resources, workloadObj)

	instrumented := printWorkloadManifestInfo(analyze, &sb)
	printInstrumentationConfigInfo(analyze, &sb)
	printRuntimeDetails(analyze, &sb)
	printInstrumentedApplicationInfo(analyze, &sb)
	containerNameToExpectedDevices := printAppliedInstrumentationDeviceInfo(analyze, workloadObj, instrumented, &sb)
	printPodsInfo(resources.Pods, resources.InstrumentationInstances, containerNameToExpectedDevices, &sb)

	return sb.String()
}

func DescribeDeployment(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, ns string, name string) string {
	deployment, err := kubeClient.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}
	workloadObj := &source.K8sSourceObject{
		Kind:            "deployment",
		ObjectMeta:      deployment.ObjectMeta,
		PodTemplateSpec: &deployment.Spec.Template,
		LabelSelector:   deployment.Spec.Selector,
	}
	return PrintDescribeSource(ctx, kubeClient, odigosClient, workloadObj)
}

func DescribeDaemonSet(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, ns string, name string) string {
	ds, err := kubeClient.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}
	workloadObj := &source.K8sSourceObject{
		Kind:            "daemonset",
		ObjectMeta:      ds.ObjectMeta,
		PodTemplateSpec: &ds.Spec.Template,
		LabelSelector:   ds.Spec.Selector,
	}
	return PrintDescribeSource(ctx, kubeClient, odigosClient, workloadObj)
}

func DescribeStatefulSet(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, ns string, name string) string {
	ss, err := kubeClient.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}
	workloadObj := &source.K8sSourceObject{
		Kind:            "statefulset",
		ObjectMeta:      ss.ObjectMeta,
		PodTemplateSpec: &ss.Spec.Template,
		LabelSelector:   ss.Spec.Selector,
	}
	return PrintDescribeSource(ctx, kubeClient, odigosClient, workloadObj)
}
