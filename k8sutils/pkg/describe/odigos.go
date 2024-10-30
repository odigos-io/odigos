package describe

import (
	"context"
	"strings"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	odigos "github.com/odigos-io/odigos/k8sutils/pkg/describe/odigos"
	"k8s.io/client-go/kubernetes"
)

func printClusterCollectorStatus(analyze *odigos.OdigosAnalyze, sb *strings.Builder) {
	describeText(sb, 1, "Cluster Collector:")
	printProperty(sb, 2, &analyze.ClusterCollector.Enabled)
	printProperty(sb, 2, &analyze.ClusterCollector.CollectorGroup)
	printProperty(sb, 2, analyze.ClusterCollector.Deployed)
	printProperty(sb, 2, analyze.ClusterCollector.DeployedError)
	printProperty(sb, 2, analyze.ClusterCollector.CollectorReady)
	printProperty(sb, 2, &analyze.ClusterCollector.DeploymentCreated)
	printProperty(sb, 2, analyze.ClusterCollector.ExpectedReplicas)
	printProperty(sb, 2, analyze.ClusterCollector.HealthyReplicas)
	printProperty(sb, 2, analyze.ClusterCollector.FailedReplicas)
	printProperty(sb, 2, analyze.ClusterCollector.FailedReplicasReason)
}

func printNodeCollectorStatus(analyze *odigos.OdigosAnalyze, sb *strings.Builder) {
	describeText(sb, 1, "Node Collector:")
	printProperty(sb, 2, &analyze.NodeCollector.Enabled)
	printProperty(sb, 2, &analyze.NodeCollector.CollectorGroup)
	printProperty(sb, 2, analyze.NodeCollector.Deployed)
	printProperty(sb, 2, analyze.NodeCollector.DeployedError)
	printProperty(sb, 2, analyze.NodeCollector.CollectorReady)
	printProperty(sb, 2, &analyze.NodeCollector.DaemonSet)
	printProperty(sb, 2, analyze.NodeCollector.DesiredNodes)
	printProperty(sb, 2, analyze.NodeCollector.CurrentNodes)
	printProperty(sb, 2, analyze.NodeCollector.UpdatedNodes)
	printProperty(sb, 2, analyze.NodeCollector.AvailableNodes)
}

func printOdigosPipeline(analyze *odigos.OdigosAnalyze, sb *strings.Builder) {
	describeText(sb, 0, "Odigos Pipeline:")
	describeText(sb, 1, "Status: there are %d sources and %d destinations\n", analyze.NumberOfSources, analyze.NumberOfDestinations)
	printClusterCollectorStatus(analyze, sb)
	sb.WriteString("\n")
	printNodeCollectorStatus(analyze, sb)
}

func DescribeOdigosToText(analyze *odigos.OdigosAnalyze) string {
	var sb strings.Builder

	printProperty(&sb, 0, &analyze.OdigosVersion)
	sb.WriteString("\n")
	printOdigosPipeline(analyze, &sb)

	return sb.String()
}

func DescribeOdigos(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) (*odigos.OdigosAnalyze, error) {

	odigosResources, err := odigos.GetRelevantOdigosResources(ctx, kubeClient, odigosClient, odigosNs)
	if err != nil {
		return nil, err
	}

	return odigos.AnalyzeOdigos(odigosResources), nil
}
