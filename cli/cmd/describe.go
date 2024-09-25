package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe"
	"github.com/spf13/cobra"
)

var (
	describeNamespaceFlag string
	describeRemoteFlag    bool
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Show details of a specific odigos entity",
	Long:  `Print detailed description of a specific odigos entity, which can be used to troubleshoot issues`,
}

var describeSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Show details of a specific odigos source",
	Long:  `Print detailed description of a specific odigos source, which can be used to troubleshoot issues`,
}

var describeSourceDeploymentCmd = &cobra.Command{
	Use:     "deployment <name>",
	Short:   "Show details of a specific odigos source of type deployment",
	Long:    `Print detailed description of a specific odigos source of type deployment, which can be used to troubleshoot issues`,
	Aliases: []string{"deploy", "deployments", "deploy.apps", "deployment.apps", "deployments.apps"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}
		ctx := cmd.Context()
		name := args[0]
		ns := cmd.Flag("namespace").Value.String()

		var describeText string
		if describeRemoteFlag {
			describeText = executeRemoteDescribe(ctx, client, "deployment", ns, name)
		} else {
			describeText = describe.DescribeDeployment(ctx, client.Interface, client.OdigosClient, ns, name)
		}
		fmt.Println(describeText)
	},
}

var describeSourceDaemonSetCmd = &cobra.Command{
	Use:     "daemonset <name>",
	Short:   "Show details of a specific odigos source of type daemonset",
	Long:    `Print detailed description of a specific odigos source of type daemonset, which can be used to troubleshoot issues`,
	Aliases: []string{"ds", "daemonsets", "ds.apps", "daemonset.apps", "daemonsets.apps"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}

		ctx := cmd.Context()
		name := args[0]
		ns := cmd.Flag("namespace").Value.String()

		var describeText string
		if describeRemoteFlag {
			describeText = executeRemoteDescribe(ctx, client, "daemonset", ns, name)
		} else {
			describeText = describe.DescribeDaemonSet(ctx, client.Interface, client.OdigosClient, ns, name)
		}
		fmt.Println(describeText)
	},
}

var describeSourceStatefulSetCmd = &cobra.Command{
	Use:     "statefulset <name>",
	Short:   "Show details of a specific odigos source of type statefulset",
	Long:    `Print detailed description of a specific odigos source of type statefulset, which can be used to troubleshoot issues`,
	Aliases: []string{"sts", "statefulsets", "sts.apps", "statefulset.apps", "statefulsets.apps"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}

		ctx := cmd.Context()
		name := args[0]
		ns := cmd.Flag("namespace").Value.String()

		var describeText string
		if describeRemoteFlag {
			describeText = executeRemoteDescribe(ctx, client, "statefulset", ns, name)
		} else {
			describeText = describe.DescribeStatefulSet(ctx, client.Interface, client.OdigosClient, ns, name)
		}
		fmt.Println(describeText)
	},
}

func executeRemoteDescribe(ctx context.Context, client *kube.Client, workloadKind string, workloadNs string, workloadName string) string {
	uiSvcProxyEndpoint := getUiServiceEndpoint(ctx, client, workloadKind, workloadNs, workloadName)
	// Send the HTTP GET request
	request := client.Clientset.RESTClient().Get().AbsPath(uiSvcProxyEndpoint).Do(ctx)

	// Handle the response
	response, err := request.Raw()
	if err != nil {
		return "Remote describe failed: " + err.Error()
	} else {
		return string(response)
	}
}

func getUiServiceEndpoint(ctx context.Context, client *kube.Client, workloadKind string, workloadNs string, workloadName string) string {
	ns, err := resources.GetOdigosNamespace(client, ctx)
	if resources.IsErrNoOdigosNamespaceFound(err) {
		fmt.Println("\033[31mERROR\033[0m no odigos installation found in the current cluster. use \"odigos install\" to install odigos in the cluster or check that kubeconfig is pointing to the correct cluster.")
		os.Exit(1)
	} else if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already installed: %s\n", err)
		os.Exit(1)
	}

	uiServiceName := "ui"
	uiServicePort := 3000

	return fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/api/describe/source/kind/%s/namespace/%s/name/%s", ns, uiServiceName, uiServicePort, workloadKind, workloadNs, workloadName)
	// return fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy", ns, uiServiceName, uiServicePort)
}

func init() {

	// describe
	rootCmd.AddCommand(describeCmd)
	describeCmd.PersistentFlags().BoolVarP(&describeRemoteFlag, "remote", "r", false, "use odigos ui service in the cluster to describe the entity")

	// source
	describeCmd.AddCommand(describeSourceCmd)
	describeSourceCmd.PersistentFlags().StringVarP(&describeNamespaceFlag, "namespace", "n", "default", "namespace of the source being described")

	// source kinds
	describeSourceCmd.AddCommand(describeSourceDeploymentCmd)
	describeSourceCmd.AddCommand(describeSourceDaemonSetCmd)
	describeSourceCmd.AddCommand(describeSourceStatefulSetCmd)
}
