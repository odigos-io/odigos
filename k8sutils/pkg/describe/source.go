package describe

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/source"
)

func printWorkloadManifestInfo(analyze *source.SourceAnalyze, sb *strings.Builder) {
	printProperty(sb, 0, &analyze.Name)
	printProperty(sb, 0, &analyze.Kind)
	printProperty(sb, 0, &analyze.Namespace)

	sb.WriteString("Source Custom Resources:\n")
	printProperty(sb, 1, &analyze.SourceObjectsAnalysis.Instrumented)
	printProperty(sb, 1, analyze.SourceObjectsAnalysis.Workload)
	printProperty(sb, 1, analyze.SourceObjectsAnalysis.Namespace)
	printProperty(sb, 1, &analyze.SourceObjectsAnalysis.InstrumentedText)
}

func printRuntimeDetails(analyze *source.SourceAnalyze, sb *strings.Builder) {
	describeText(sb, 0, false, "\nRuntime Inspection Details:")

	if analyze.RuntimeInfo == nil {
		describeText(sb, 1, false, "No runtime details")
		return
	}

	describeText(sb, 1, false, "Detected Containers:")
	for i := range analyze.RuntimeInfo.Containers {
		container := analyze.RuntimeInfo.Containers[i]
		printProperty(sb, 2, &container.ContainerName)
		printProperty(sb, 2, &container.Language)
		printProperty(sb, 2, &container.RuntimeVersion)
		if len(container.EnvVars) > 0 {
			describeText(sb, 2, false, "Relevant Environment Variables:")
			for _, envVar := range container.EnvVars {
				describeText(sb, 3, true, "%s", fmt.Sprintf("%s: %s", envVar.Name, envVar.Value))
			}
		}
	}
}

func printInstrumentationConfigInfo(analyze *source.SourceAnalyze, sb *strings.Builder) {
	describeText(sb, 0, false, "\nInstrumentation Config:")
	printProperty(sb, 1, &analyze.OtelAgents.Created)
	printProperty(sb, 1, analyze.OtelAgents.CreateTime)

	describeText(sb, 1, false, "Containers:")
	for i := range analyze.OtelAgents.Containers {
		containerConfig := analyze.OtelAgents.Containers[i]
		printProperty(sb, 2, &containerConfig.ContainerName)
		printProperty(sb, 2, &containerConfig.Instrumented)
		printProperty(sb, 2, containerConfig.Reason)
		printProperty(sb, 2, containerConfig.Message)
		printProperty(sb, 2, containerConfig.OtelDistroName)
	}
}

func printPodsInfo(analyze *source.SourceAnalyze, sb *strings.Builder) {
	describeText(sb, 0, false, "\nPods (Total %d, %s):", analyze.TotalPods, analyze.PodsPhasesCount)

	for i := range analyze.Pods {
		pod := analyze.Pods[i]
		describeText(sb, 0, false, "")
		printProperty(sb, 1, &pod.PodName)
		printProperty(sb, 1, &pod.NodeName)
		printProperty(sb, 1, &pod.Phase)
		describeText(sb, 1, false, "Containers:")
		for i := range pod.Containers {
			container := pod.Containers[i]
			printProperty(sb, 2, &container.ContainerName)
			printProperty(sb, 2, &container.ActualDevices)
			describeText(sb, 2, false, "")
			describeText(sb, 2, false, "Instrumentation Instances:")
			for _, ii := range container.InstrumentationInstances {
				printProperty(sb, 3, &ii.Healthy)
				printProperty(sb, 3, ii.Message)
				if len(ii.IdentifyingAttributes) > 0 {
					describeText(sb, 3, false, "Identifying Attributes:")
					for _, attr := range ii.IdentifyingAttributes {
						printProperty(sb, 4, &attr)
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
	printPodsInfo(analyze, &sb)

	return sb.String()
}

func DescribeSource(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface,
	workloadObj *source.K8sSourceObject) (*source.SourceAnalyze, error) {
	resources, err := source.GetRelevantSourceResources(ctx, kubeClient, odigosClient, workloadObj)
	if err != nil {
		return nil, err
	}
	analyze := source.AnalyzeSource(resources, workloadObj)
	return analyze, nil
}

func DescribeDeployment(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface,
	ns string, name string) (*source.SourceAnalyze, error) {
	deployment, err := kubeClient.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	workloadObj := &source.K8sSourceObject{
		Kind:            k8sconsts.WorkloadKindDeployment,
		ObjectMeta:      deployment.ObjectMeta,
		PodTemplateSpec: &deployment.Spec.Template,
		LabelSelector:   deployment.Spec.Selector,
	}
	return DescribeSource(ctx, kubeClient, odigosClient, workloadObj)
}

func DescribeDaemonSet(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface,
	ns string, name string) (*source.SourceAnalyze, error) {
	ds, err := kubeClient.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	workloadObj := &source.K8sSourceObject{
		Kind:            k8sconsts.WorkloadKindDaemonSet,
		ObjectMeta:      ds.ObjectMeta,
		PodTemplateSpec: &ds.Spec.Template,
		LabelSelector:   ds.Spec.Selector,
	}
	return DescribeSource(ctx, kubeClient, odigosClient, workloadObj)
}

func DescribeStatefulSet(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface,
	ns string, name string) (*source.SourceAnalyze, error) {
	ss, err := kubeClient.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	workloadObj := &source.K8sSourceObject{
		Kind:            k8sconsts.WorkloadKindStatefulSet,
		ObjectMeta:      ss.ObjectMeta,
		PodTemplateSpec: &ss.Spec.Template,
		LabelSelector:   ss.Spec.Selector,
	}
	return DescribeSource(ctx, kubeClient, odigosClient, workloadObj)
}
