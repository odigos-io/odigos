package describe

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/source"
)

func printWorkloadManifestInfo(analyze *source.SourceAnalyze, sb *strings.Builder) {
	printProperty(sb, 0, &analyze.Name)
	printProperty(sb, 0, &analyze.Kind)
	printProperty(sb, 0, &analyze.Namespace)

	sb.WriteString("Labels:\n")
	printProperty(sb, 1, &analyze.Labels.Instrumented)
	printProperty(sb, 1, analyze.Labels.Workload)
	printProperty(sb, 1, analyze.Labels.Namespace)
	printProperty(sb, 1, &analyze.Labels.InstrumentedText)
}

func printRuntimeDetails(analyze *source.SourceAnalyze, sb *strings.Builder) {
	describeText(sb, 0, "\nRuntime Inspection Details:")

	if analyze.RuntimeInfo == nil {
		describeText(sb, 1, "No runtime details")
		return
	}

	describeText(sb, 1, "Detected Containers:")
	for i := range analyze.RuntimeInfo.Containers {
		container := analyze.RuntimeInfo.Containers[i]
		printProperty(sb, 2, &container.ContainerName)
		printProperty(sb, 3, &container.Language)
		printProperty(sb, 3, &container.RuntimeVersion)
		if len(container.EnvVars) > 0 {
			describeText(sb, 3, "Relevant Environment Variables:")
			for _, envVar := range container.EnvVars {
				describeText(sb, 4, "%s", fmt.Sprintf("%s: %s", envVar.Name, envVar.Value))
			}
		}
	}
}

func printInstrumentationConfigInfo(analyze *source.SourceAnalyze, sb *strings.Builder) {
	describeText(sb, 0, "\nInstrumentation Config:")
	printProperty(sb, 1, &analyze.InstrumentationConfig.Created)
	printProperty(sb, 1, analyze.InstrumentationConfig.CreateTime)

	describeText(sb, 1, "Detected Containers:")
	for i := range analyze.InstrumentationConfig.Containers {
		container := analyze.InstrumentationConfig.Containers[i]
		printProperty(sb, 2, &container.ContainerName)
		printProperty(sb, 3, &container.Language)
		printProperty(sb, 3, &container.RuntimeVersion)
		printProperty(sb, 3, &container.CriError)
		if len(container.EnvVars) > 0 {
			describeText(sb, 3, "Relevant Environment Variables:")
			for _, envVar := range container.EnvVars {
				describeText(sb, 4, "%s", fmt.Sprintf("%s: %s", envVar.Name, envVar.Value))
			}
		}
		if len(container.ContainerRuntimeEnvs) > 0 {
			describeText(sb, 3, "Relevant Container Runtime Environment Variables:")
			for _, containerRuntimeEnvVar := range container.ContainerRuntimeEnvs {
				describeText(sb, 4, "%s", fmt.Sprintf("%s: %s", containerRuntimeEnvVar.Name, containerRuntimeEnvVar.Value))
			}
		}
	}
}

func printAppliedInstrumentationDeviceInfo(analyze *source.SourceAnalyze, sb *strings.Builder) {
	describeText(sb, 0, "\nInstrumentation Device:")
	printProperty(sb, 1, &analyze.InstrumentationDevice.StatusText)
	describeText(sb, 1, "Containers:")
	for i := range analyze.InstrumentationDevice.Containers {
		container := analyze.InstrumentationDevice.Containers[i]
		printProperty(sb, 2, &container.ContainerName)
		printProperty(sb, 3, &container.Devices)
		if len(container.OriginalEnv) > 0 {
			describeText(sb, 3, "Original Environment Variables:")
			for _, envVar := range container.OriginalEnv {
				printProperty(sb, 4, &envVar)
			}
		}
	}
}

func printPodsInfo(analyze *source.SourceAnalyze, sb *strings.Builder) {
	describeText(sb, 0, "\nPods (Total %d, %s):", analyze.TotalPods, analyze.PodsPhasesCount)

	for i := range analyze.Pods {
		pod := analyze.Pods[i]
		describeText(sb, 0, "")
		printProperty(sb, 1, &pod.PodName)
		printProperty(sb, 1, &pod.NodeName)
		printProperty(sb, 1, &pod.Phase)
		describeText(sb, 1, "Containers:")
		for i := range pod.Containers {
			container := pod.Containers[i]
			printProperty(sb, 2, &container.ContainerName)
			printProperty(sb, 3, &container.ActualDevices)
			describeText(sb, 3, "")
			describeText(sb, 3, "Instrumentation Instances:")
			for _, ii := range container.InstrumentationInstances {
				printProperty(sb, 4, &ii.Healthy)
				printProperty(sb, 4, ii.Message)
				if len(ii.IdentifyingAttributes) > 0 {
					describeText(sb, 4, "Identifying Attributes:")
					for _, attr := range ii.IdentifyingAttributes {
						printProperty(sb, 5, &attr)
					}
				}
			}
		}
	}
}

func DescribeSourceToText(analyze *source.SourceAnalyze) string {
	var sb strings.Builder

	printWorkloadManifestInfo(analyze, &sb)
	printRuntimeDetails(analyze, &sb)
	printInstrumentationConfigInfo(analyze, &sb)
	printAppliedInstrumentationDeviceInfo(analyze, &sb)
	printPodsInfo(analyze, &sb)

	return sb.String()
}

func DescribeSource(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, workloadObj *source.K8sSourceObject) (*source.SourceAnalyze, error) {
	resources, err := source.GetRelevantSourceResources(ctx, kubeClient, odigosClient, workloadObj)
	if err != nil {
		return nil, err
	}
	analyze := source.AnalyzeSource(resources, workloadObj)
	return analyze, nil
}

func DescribeDeployment(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, ns string, name string) (*source.SourceAnalyze, error) {
	deployment, err := kubeClient.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	workloadObj := &source.K8sSourceObject{
		Kind:            "deployment",
		ObjectMeta:      deployment.ObjectMeta,
		PodTemplateSpec: &deployment.Spec.Template,
		LabelSelector:   deployment.Spec.Selector,
	}
	return DescribeSource(ctx, kubeClient, odigosClient, workloadObj)
}

func DescribeDaemonSet(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, ns string, name string) (*source.SourceAnalyze, error) {
	ds, err := kubeClient.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	workloadObj := &source.K8sSourceObject{
		Kind:            "daemonset",
		ObjectMeta:      ds.ObjectMeta,
		PodTemplateSpec: &ds.Spec.Template,
		LabelSelector:   ds.Spec.Selector,
	}
	return DescribeSource(ctx, kubeClient, odigosClient, workloadObj)
}

func DescribeStatefulSet(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, ns string, name string) (*source.SourceAnalyze, error) {
	ss, err := kubeClient.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	workloadObj := &source.K8sSourceObject{
		Kind:            "statefulset",
		ObjectMeta:      ss.ObjectMeta,
		PodTemplateSpec: &ss.Spec.Template,
		LabelSelector:   ss.Spec.Selector,
	}
	return DescribeSource(ctx, kubeClient, odigosClient, workloadObj)
}
