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
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type K8sSourceObject struct {
	metav1.ObjectMeta
	Kind            string
	PodTemplateSpec *v1.PodTemplateSpec
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

func getInstrumentationLabelTexts(workload *K8sSourceObject, ns *v1.Namespace) (workloadText, nsText, decisionText string, instrumented bool) {
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

func printWorkloadManifestInfo(workloadObj *K8sSourceObject, namespace *corev1.Namespace) bool {
	fmt.Println("Name: ", workloadObj.GetName())
	fmt.Println("Kind: ", workloadObj.Kind)
	fmt.Println("Namespace: ", workloadObj.GetNamespace())

	fmt.Println("")
	fmt.Println("Labels:")
	workloadText, nsText, decisionText, instrumented := getInstrumentationLabelTexts(workloadObj, namespace)
	if instrumented {
		fmt.Println("  Instrumented: ", wrapTextInGreen("true"))
	} else {
		fmt.Println("  Instrumented: ", wrapTextInRed("false"))
	}
	fmt.Println("  Workload: " + workloadText)
	fmt.Println("  Namespace: " + nsText)
	fmt.Println("  Decision: " + decisionText)
	fmt.Println("  Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#1-odigos-instrumentation-label")

	return instrumented
}

func printInstrumentationConfigInfo(instrumentationConfig *odigosv1.InstrumentationConfig, instrumented bool) {
	instrumentationConfigNotFound := instrumentationConfig == nil
	statusAsExpected := instrumentationConfigNotFound == !instrumented
	fmt.Println("")
	fmt.Println("Instrumentation Config:")
	if instrumentationConfigNotFound {
		if statusAsExpected {
			fmt.Println(wrapTextInGreen("  Workload not instrumented, no instrumentation config"))
		} else {
			fmt.Println("  Not yet created")
		}
	} else {
		createAtText := "  Created at " + instrumentationConfig.GetCreationTimestamp().String()
		fmt.Println(wrapTextSuccessOfFailure(createAtText, statusAsExpected))
	}

	if !statusAsExpected {
		fmt.Println("  Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#2-odigos-instrumentation-config")
	}
}

func printInstrumentedApplicationInfo(instrumentedApplication *odigosv1.InstrumentedApplication, instrumented bool) {
	instrumentedApplicationNotFound := instrumentedApplication == nil
	statusAsExpected := instrumentedApplicationNotFound == !instrumented
	fmt.Println("")
	fmt.Println("Runtime inspection details:")
	if instrumentedApplicationNotFound {
		if instrumented {
			fmt.Println("  Not yet created")
		} else {
			fmt.Println(wrapTextInGreen("  Workload not instrumented, no runtime details"))
		}
	} else {
		createdAtText := "  Created at " + instrumentedApplication.GetCreationTimestamp().String()
		fmt.Println(wrapTextSuccessOfFailure(createdAtText, statusAsExpected))
		fmt.Println("  Detected Containers:")
		for _, container := range instrumentedApplication.Spec.RuntimeDetails {
			fmt.Println("    - Container Name:", container.ContainerName)
			colorfulLanguage := string(container.Language)
			isUnknown := container.Language == common.UnknownProgrammingLanguage
			if isUnknown {
				colorfulLanguage = wrapTextInRed(string(container.Language))
			} else if container.Language != common.IgnoredProgrammingLanguage {
				colorfulLanguage = wrapTextInGreen(string(container.Language))
			}
			fmt.Println("      Language:      ", colorfulLanguage)
			if isUnknown {
				fmt.Println("      Troubleshooting: http://localhost:3000/architecture/troubleshooting#4-language-not-detected")
			}

			// calculate env vars for this container
			if container.EnvVars != nil && len(container.EnvVars) > 0 {
				fmt.Println("      Relevant Environment Variables:")
				for _, envVar := range container.EnvVars {
					fmt.Println("        -", envVar.Name, ":", envVar.Value)
				}
			}
		}
	}
	if !statusAsExpected {
		fmt.Println("  Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#3-odigos-instrumented-application")
	}
}

func printAppliedInstrumentationDeviceInfo(workloadObj *K8sSourceObject, instrumentedApplication *odigosv1.InstrumentedApplication, instrumented bool) map[string][]string {
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
	fmt.Println("")
	fmt.Println("Instrumentation Device:")
	fmt.Println("  Status:", appliedInstrumentationDeviceStatusMessage)
	containerNameToExpectedDevices := make(map[string][]string)
	for _, container := range workloadObj.PodTemplateSpec.Spec.Containers {
		fmt.Println("  - Container Name:", container.Name)
		odigosDevices := make([]string, 0)
		for resourceName := range container.Resources.Limits {
			deviceName, found := strings.CutPrefix(resourceName.String(), common.OdigosResourceNamespace+"/")
			if found {
				odigosDevices = append(odigosDevices, deviceName)
			}
		}
		if len(odigosDevices) == 0 {
			if !instrumented {
				fmt.Println(wrapTextInGreen("    No instrumentation devices"))
			} else {
				fmt.Println("    No instrumentation devices")
				fmt.Println("    Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#5-odigos-instrumentation-devices-not-added")
			}
		} else {
			fmt.Println("    Instrumentation Devices:", wrapTextSuccessOfFailure(strings.Join(odigosDevices, ", "), instrumented))
			if !instrumented {
				fmt.Println("	 Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#5-odigos-instrumentation-devices-not-added")
			}
		}
		containerNameToExpectedDevices[container.Name] = odigosDevices

		// override environment variables
		originalContainerEnvs := origWorkloadEnvValues.GetContainerStoredEnvs(container.Name)
		if originalContainerEnvs != nil && len(originalContainerEnvs) > 0 {
			fmt.Println("    Original Environment Variables:")
			for envName, envVarOriginalValue := range originalContainerEnvs {
				if envVarOriginalValue == nil {
					fmt.Println("    - " + envName + "=null (not set in manifest)")
				} else {
					fmt.Println("    - " + envName + "=" + *envVarOriginalValue)
				}
			}
		}
	}

	return containerNameToExpectedDevices
}

func printPodContainerInstrumentationInstancesInfo(instances []*odigosv1.InstrumentationInstance) {
	if len(instances) == 0 {
		fmt.Println("    No instrumentation instances")
		return
	}

	fmt.Println("    Instrumentation Instances:")
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
		fmt.Println("    - Healthy:", healthyText)
		if instance.Status.Message != "" {
			fmt.Println("      Message:", instance.Status.Message)
		}
		if instance.Status.Reason != "" && instance.Status.Reason != string(common.AgentHealthStatusHealthy) {
			fmt.Println("      Reason:", instance.Status.Reason)
		}
		if unhealthy {
			fmt.Println("      Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#7-instrumentation-instance-unhealthy")
		}
	}
}

func printPodContainerInfo(pod *corev1.Pod, container *corev1.Container, instrumentationInstances *odigosv1.InstrumentationInstanceList, containerNameToExpectedDevices map[string][]string) {
	instrumentationDevices := make([]string, 0)
	fmt.Println("  - Container Name:", container.Name)
	for resourceName := range container.Resources.Limits {
		deviceName, found := strings.CutPrefix(resourceName.String(), common.OdigosResourceNamespace+"/")
		if found {
			instrumentationDevices = append(instrumentationDevices, deviceName)
		}
	}
	expectedDevices, foundExpectedDevices := containerNameToExpectedDevices[container.Name]
	if len(instrumentationDevices) == 0 {
		isMatch := !foundExpectedDevices || len(expectedDevices) == 0
		fmt.Println(wrapTextSuccessOfFailure("    No instrumentation devices", isMatch))
		if !isMatch {
			fmt.Println("      Expected Devices:", strings.Join(expectedDevices, ", "))
			fmt.Println("      Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#6-pods-instrumentation-devices-not-matching-manifest")
		}
	} else {
		actualDevicesText := strings.Join(instrumentationDevices, ", ")
		expectedDevicesText := strings.Join(expectedDevices, ", ")
		isMatch := actualDevicesText == expectedDevicesText
		fmt.Println("    Instrumentation Devices:", wrapTextSuccessOfFailure(actualDevicesText, isMatch))
		if !isMatch {
			fmt.Println("      Expected Devices:", expectedDevicesText)
			fmt.Println("      Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#6-pods-instrumentation-devices-not-matching-manifest")
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
	printPodContainerInstrumentationInstancesInfo(thisPodInstrumentationInstances)
}

func printSinglePodInfo(pod *corev1.Pod, instrumentationInstances *odigosv1.InstrumentationInstanceList, containerNameToExpectedDevices map[string][]string) {
	fmt.Println("")
	fmt.Println("  Pod Name:", pod.GetName())
	fmt.Println("  Pod Phase:", pod.Status.Phase)
	fmt.Println("  Pod Node Name:", pod.Spec.NodeName)
	fmt.Println("  Containers:")
	for _, container := range pod.Spec.Containers {
		printPodContainerInfo(pod, &container, instrumentationInstances, containerNameToExpectedDevices)
	}
}

func printPodsInfo(pods *corev1.PodList, instrumentationInstances *odigosv1.InstrumentationInstanceList, containerNameToExpectedDevices map[string][]string) {
	podsStatuses := make(map[v1.PodPhase]int)
	for _, pod := range pods.Items {
		podsStatuses[pod.Status.Phase]++
	}
	podPhasesTexts := make([]string, 0)
	for phase, count := range podsStatuses {
		podPhasesTexts = append(podPhasesTexts, fmt.Sprintf("%s %d", phase, count))
	}
	podPhasesText := strings.Join(podPhasesTexts, ", ")
	fmt.Println("")
	fmt.Printf("Pods (Total %d, %s):\n", len(pods.Items), podPhasesText)
	for _, pod := range pods.Items {
		printSinglePodInfo(&pod, instrumentationInstances, containerNameToExpectedDevices)
	}
}

func PrintDescribeSource(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, workloadObj *K8sSourceObject) {
	namespace, instrumentationConfig, instrumentedApplication, instrumentationInstances, pods, err := getRelevantResources(ctx, kubeClient, odigosClient, workloadObj)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	instrumented := printWorkloadManifestInfo(workloadObj, namespace)
	printInstrumentationConfigInfo(instrumentationConfig, instrumented)
	printInstrumentedApplicationInfo(instrumentedApplication, instrumented)
	containerNameToExpectedDevices := printAppliedInstrumentationDeviceInfo(workloadObj, instrumentedApplication, instrumented)
	printPodsInfo(pods, instrumentationInstances, containerNameToExpectedDevices)
}
