package describe

import (
	"context"
	"fmt"
	"strings"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/envoverwrite"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type K8sSourceObject struct {
	metav1.ObjectMeta
	Kind            string
	PodTemplateSpec *corev1.PodTemplateSpec
	LabelSelector   *metav1.LabelSelector
}

func wrapTextInRed(text string) string {
	return "\033[31m" + text + "\033[0m"
}

func wrapTextInGreen(text string) string {
	return "\033[32m" + text + "\033[0m"
}

func wrapTextSuccessOfFailure(text string, success bool) string {
	if success {
		return wrapTextInGreen(text)
	} else {
		return wrapTextInRed(text)
	}
}

func getInstrumentationLabelTexts(workload *K8sSourceObject, ns *corev1.Namespace) (workloadText, nsText, decisionText string, instrumented bool) {
	workloadLabel, workloadFound := workload.GetLabels()[consts.OdigosInstrumentationLabel]
	nsLabel, nsFound := ns.GetLabels()[consts.OdigosInstrumentationLabel]

	if workloadFound {
		workloadText = consts.OdigosInstrumentationLabel + "=" + workloadLabel
	} else {
		workloadText = consts.OdigosInstrumentationLabel + " label not set"
	}

	if nsFound {
		nsText = consts.OdigosInstrumentationLabel + "=" + nsLabel
	} else {
		nsText = consts.OdigosInstrumentationLabel + " label not set"
	}

	if workloadFound {
		instrumented = workloadLabel == consts.InstrumentationEnabled
		if instrumented {
			decisionText = "Workload is instrumented because the " + workload.Kind + " contains the label '" + consts.OdigosInstrumentationLabel + "=" + workloadLabel + "'"
		} else {
			decisionText = "Workload is NOT instrumented because the " + workload.Kind + " contains the label '" + consts.OdigosInstrumentationLabel + "=" + workloadLabel + "'"
		}
	} else {
		instrumented = nsLabel == consts.InstrumentationEnabled
		if instrumented {
			decisionText = "Workload is instrumented because the " + workload.Kind + " is not labeled, and the namespace is labeled with '" + consts.OdigosInstrumentationLabel + "=" + nsLabel + "'"
		} else {
			if nsFound {
				decisionText = "Workload is NOT instrumented because the " + workload.Kind + " is not labeled, and the namespace is labeled with '" + consts.OdigosInstrumentationLabel + "=" + nsLabel + "'"
			} else {
				decisionText = "Workload is NOT instrumented because neither the workload nor the namespace has the '" + consts.OdigosInstrumentationLabel + "' label set"
			}
		}
	}

	return
}

func getRelevantResources(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, workloadObj *K8sSourceObject) (namespace *corev1.Namespace, instrumentationConfig *odigosv1.InstrumentationConfig, instrumentedApplication *odigosv1.InstrumentedApplication, instrumentationInstances *odigosv1.InstrumentationInstanceList, pods *corev1.PodList, err error) {

	ns := workloadObj.GetNamespace()
	namespace, err = kubeClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err != nil {
		return
	}

	runtimeObjectName := workload.CalculateWorkloadRuntimeObjectName(workloadObj.GetName(), workloadObj.Kind)
	instrumentationConfig, err = odigosClient.InstrumentationConfigs(ns).Get(ctx, runtimeObjectName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// it is ok if the instrumentation config is not found
			err = nil
			instrumentationConfig = nil
		} else {
			return
		}
	}

	instrumentedApplication, err = odigosClient.InstrumentedApplications(ns).Get(ctx, runtimeObjectName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// it is ok if the instrumented application is not found
			err = nil
			instrumentedApplication = nil
		} else {
			return
		}
	}

	instrumentedAppSelector := labels.SelectorFromSet(labels.Set{
		"instrumented-app": runtimeObjectName,
	})
	instrumentationInstances, err = odigosClient.InstrumentationInstances(ns).List(ctx, metav1.ListOptions{LabelSelector: instrumentedAppSelector.String()})
	if err != nil {
		// if no instrumentation instances are found, it should not error, so any error is returned
		return
	}

	podLabelSelector := metav1.FormatLabelSelector(workloadObj.LabelSelector)
	if err != nil {
		// if pod info cannot be extracted, it is an unrecoverable error
		return
	}
	pods, err = kubeClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: podLabelSelector})
	if err != nil {
		// if no pods are found, it should not error, so any error is returned
		return
	}

	return
}

func printWorkloadManifestInfo(workloadObj *K8sSourceObject, namespace *corev1.Namespace, sb *strings.Builder) bool {
	sb.WriteString(fmt.Sprintf("Name: %s\n", workloadObj.GetName()))
	sb.WriteString(fmt.Sprintf("Kind: %s\n", workloadObj.Kind))
	sb.WriteString(fmt.Sprintf("Namespace: %s\n\n", workloadObj.GetNamespace()))

	sb.WriteString("Labels:\n")
	workloadText, nsText, decisionText, instrumented := getInstrumentationLabelTexts(workloadObj, namespace)
	if instrumented {
		sb.WriteString("  Instrumented: " + wrapTextInGreen("true") + "\n")
	} else {
		sb.WriteString("  Instrumented: " + wrapTextInRed("false") + "\n")
	}
	sb.WriteString("  Workload: " + workloadText + "\n")
	sb.WriteString("  Namespace: " + nsText + "\n")
	sb.WriteString("  Decision: " + decisionText + "\n")
	sb.WriteString("  Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#1-odigos-instrumentation-label\n")

	return instrumented
}

func printInstrumentationConfigInfo(instrumentationConfig *odigosv1.InstrumentationConfig, instrumented bool, sb *strings.Builder) {
	instrumentationConfigNotFound := instrumentationConfig == nil
	statusAsExpected := instrumentationConfigNotFound == !instrumented
	sb.WriteString("\nInstrumentation Config:\n")
	if instrumentationConfigNotFound {
		if statusAsExpected {
			sb.WriteString(wrapTextInGreen("  Workload not instrumented, no instrumentation config\n"))
		} else {
			sb.WriteString("  Not yet created\n")
		}
	} else {
		createAtText := "  Created at " + instrumentationConfig.GetCreationTimestamp().String()
		sb.WriteString(wrapTextSuccessOfFailure(createAtText, statusAsExpected) + "\n")
	}

	if !statusAsExpected {
		sb.WriteString("  Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#2-odigos-instrumentation-config\n")
	}
}

func printRuntimeDetails(instrumentationConfig *odigosv1.InstrumentationConfig, instrumented bool, sb *strings.Builder) {
	if instrumentationConfig == nil {
		sb.WriteString("No runtime details\n")
		return
	}

	sb.WriteString("\nRuntime inspection details (new):\n")
	sb.WriteString(fmt.Sprintf("Workload generation: %d\n", instrumentationConfig.Status.ObservedWorkloadGeneration))
	for _, container := range instrumentationConfig.Status.RuntimeDetailsByContainer {
		sb.WriteString(fmt.Sprintf("  Container Name: %s\n", container.ContainerName))
		colorfulLanguage := string(container.Language)
		isUnknown := container.Language == common.UnknownProgrammingLanguage
		if isUnknown {
			colorfulLanguage = wrapTextInRed(string(container.Language))
		} else if container.Language != common.IgnoredProgrammingLanguage {
			colorfulLanguage = wrapTextInGreen(string(container.Language))
		}
		sb.WriteString("    Language:      " + colorfulLanguage + "\n")
		if isUnknown {
			sb.WriteString("    Troubleshooting: http://localhost:3000/architecture/troubleshooting#4-language-not-detected\n")
		}
		if container.RuntimeVersion != "" {
			sb.WriteString("    Runtime Version: " + container.RuntimeVersion + "\n")
		} else {
			sb.WriteString("    Runtime Version: not detected\n")
		}

		// calculate env vars for this container
		if container.EnvVars != nil && len(container.EnvVars) > 0 {
			sb.WriteString("    Relevant Environment Variables:\n")
			for _, envVar := range container.EnvVars {
				sb.WriteString(fmt.Sprintf("      - %s: %s\n", envVar.Name, envVar.Value))
			}
		}
	}
}

func printInstrumentedApplicationInfo(instrumentedApplication *odigosv1.InstrumentedApplication, instrumented bool, sb *strings.Builder) {
	instrumentedApplicationNotFound := instrumentedApplication == nil
	statusAsExpected := instrumentedApplicationNotFound == !instrumented
	sb.WriteString("\nRuntime inspection details (old):\n")
	if instrumentedApplicationNotFound {
		if instrumented {
			sb.WriteString("  Not yet created\n")
		} else {
			sb.WriteString(wrapTextInGreen("  Workload not instrumented, no runtime details\n"))
		}
	} else {
		createdAtText := "  Created at " + instrumentedApplication.GetCreationTimestamp().String()
		sb.WriteString(wrapTextSuccessOfFailure(createdAtText, statusAsExpected) + "\n")
		sb.WriteString("  Detected Containers:\n")
		for _, container := range instrumentedApplication.Spec.RuntimeDetails {
			sb.WriteString(fmt.Sprintf("    - Container Name: %s\n", container.ContainerName))
			colorfulLanguage := string(container.Language)
			isUnknown := container.Language == common.UnknownProgrammingLanguage
			if isUnknown {
				colorfulLanguage = wrapTextInRed(string(container.Language))
			} else if container.Language != common.IgnoredProgrammingLanguage {
				colorfulLanguage = wrapTextInGreen(string(container.Language))
			}
			sb.WriteString("      Language:      " + colorfulLanguage + "\n")
			if isUnknown {
				sb.WriteString("      Troubleshooting: http://localhost:3000/architecture/troubleshooting#4-language-not-detected\n")
			}
			if container.RuntimeVersion != "" {
				sb.WriteString("      Runtime Version: " + container.RuntimeVersion + "\n")
			} else {
				sb.WriteString("      Runtime Version: not detected\n")
			}

			// calculate env vars for this container
			if container.EnvVars != nil && len(container.EnvVars) > 0 {
				sb.WriteString("      Relevant Environment Variables:\n")
				for _, envVar := range container.EnvVars {
					sb.WriteString(fmt.Sprintf("        - %s: %s\n", envVar.Name, envVar.Value))
				}
			}
		}
	}
	if !statusAsExpected {
		sb.WriteString("  Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#3-odigos-instrumented-application\n")
	}
}

func printAppliedInstrumentationDeviceInfo(workloadObj *K8sSourceObject, instrumentedApplication *odigosv1.InstrumentedApplication, instrumented bool, sb *strings.Builder) map[string][]string {
	appliedInstrumentationDeviceStatusMessage := "Unknown"
	if !instrumented {
		// if the workload is not instrumented, the instrumentation device expected
		appliedInstrumentationDeviceStatusMessage = "No instrumentation devices expected"
	}
	if instrumentedApplication != nil && instrumentedApplication.Status.Conditions != nil {
		for _, condition := range instrumentedApplication.Status.Conditions {
			if condition.Type == "AppliedInstrumentationDevice" { // TODO: share this constant with instrumentor
				if condition.ObservedGeneration == instrumentedApplication.GetGeneration() {
					appliedInstrumentationDeviceStatusMessage = wrapTextSuccessOfFailure(condition.Message, condition.Status == metav1.ConditionTrue)
				} else {
					appliedInstrumentationDeviceStatusMessage = "Not yet reconciled"
				}
				break
			}
		}
	}
	// get original env vars:
	origWorkloadEnvValues, _ := envoverwrite.NewOrigWorkloadEnvValues(workloadObj.GetAnnotations())
	sb.WriteString("\nInstrumentation Device:\n")
	sb.WriteString("  Status: " + appliedInstrumentationDeviceStatusMessage + "\n")
	containerNameToExpectedDevices := make(map[string][]string)
	for _, container := range workloadObj.PodTemplateSpec.Spec.Containers {
		sb.WriteString(fmt.Sprintf("  - Container Name: %s\n", container.Name))
		odigosDevices := make([]string, 0)
		for resourceName := range container.Resources.Limits {
			deviceName, found := strings.CutPrefix(resourceName.String(), common.OdigosResourceNamespace+"/")
			if found {
				odigosDevices = append(odigosDevices, deviceName)
			}
		}
		if len(odigosDevices) == 0 {
			if !instrumented {
				sb.WriteString(wrapTextInGreen("    No instrumentation devices\n"))
			} else {
				sb.WriteString("    No instrumentation devices\n")
				sb.WriteString("    Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#5-odigos-instrumentation-devices-not-added\n")
			}
		} else {
			sb.WriteString("    Instrumentation Devices: " + wrapTextSuccessOfFailure(strings.Join(odigosDevices, ", "), instrumented) + "\n")
			if !instrumented {
				sb.WriteString("	 Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#5-odigos-instrumentation-devices-not-added\n")
			}
		}
		containerNameToExpectedDevices[container.Name] = odigosDevices

		// override environment variables
		originalContainerEnvs := origWorkloadEnvValues.GetContainerStoredEnvs(container.Name)
		if originalContainerEnvs != nil && len(originalContainerEnvs) > 0 {
			sb.WriteString("    Original Environment Variables:\n")
			for envName, envVarOriginalValue := range originalContainerEnvs {
				if envVarOriginalValue == nil {
					sb.WriteString("    - " + envName + "=null (not set in manifest)\n")
				} else {
					sb.WriteString("    - " + envName + "=" + *envVarOriginalValue + "\n")
				}
			}
		}
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

func PrintDescribeSource(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, workloadObj *K8sSourceObject) string {
	var sb strings.Builder

	namespace, instrumentationConfig, instrumentedApplication, instrumentationInstances, pods, err := getRelevantResources(ctx, kubeClient, odigosClient, workloadObj)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error: %v\n", err))
		return sb.String()
	}

	instrumented := printWorkloadManifestInfo(workloadObj, namespace, &sb)
	printInstrumentationConfigInfo(instrumentationConfig, instrumented, &sb)
	printRuntimeDetails(instrumentationConfig, instrumented, &sb)
	printInstrumentedApplicationInfo(instrumentedApplication, instrumented, &sb)
	containerNameToExpectedDevices := printAppliedInstrumentationDeviceInfo(workloadObj, instrumentedApplication, instrumented, &sb)
	printPodsInfo(pods, instrumentationInstances, containerNameToExpectedDevices, &sb)

	return sb.String()
}

func DescribeDeployment(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, ns string, name string) string {
	deployment, err := kubeClient.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}
	workloadObj := &K8sSourceObject{
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
	workloadObj := &K8sSourceObject{
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
	workloadObj := &K8sSourceObject{
		Kind:            "statefulset",
		ObjectMeta:      ss.ObjectMeta,
		PodTemplateSpec: &ss.Spec.Template,
		LabelSelector:   ss.Spec.Selector,
	}
	return PrintDescribeSource(ctx, kubeClient, odigosClient, workloadObj)
}
