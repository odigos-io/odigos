package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe"
	"github.com/spf13/cobra"
)

var (
	describeNamespaceFlag string
	describeRemoteFlag    bool
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Show details on odigos deployment",
	Long:  `Print detailed description odigos deployment, which can be used to troubleshoot issues`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		odigosNs, err := resources.GetOdigosNamespace(client, ctx)
		if err != nil {
			if resources.IsErrNoOdigosNamespaceFound(err) {
				fmt.Println("\033[31mERROR\033[0m Odigos is NOT yet installed in the current cluster")
			} else {
				fmt.Println("\033[31mERROR\033[0m Error detecting Odigos namespace in the current cluster")
			}
			return
		}

		var describeText string
		if describeRemoteFlag {
			describeText = executeRemoteOdigosDescribe(ctx, client, odigosNs)
		} else {
			describeAnalyze, err := describe.DescribeOdigos(ctx, client, client.OdigosClient, odigosNs)
			if err != nil {
				describeText = fmt.Sprintf("Failed to describe odigos: %s", err)
			} else {
				describeText = describe.DescribeOdigosToText(describeAnalyze)
			}
		}
		fmt.Println(describeText)
	},
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
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		name := args[0]
		ns := cmd.Flag("namespace").Value.String()

		var describeText string
		if describeRemoteFlag {
			describeText = executeRemoteSourceDescribe(ctx, client, "deployment", ns, name)
		} else {
			desc, err := describe.DescribeDeployment(ctx, client.Interface, client.OdigosClient, ns, name)
			if err != nil {
				describeText = fmt.Sprintf("Failed to describe deployment: %s", err)
			} else {
				describeText = describe.DescribeSourceToText(desc)
			}
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
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		name := args[0]
		ns := cmd.Flag("namespace").Value.String()

		var describeText string
		if describeRemoteFlag {
			describeText = executeRemoteSourceDescribe(ctx, client, "daemonset", ns, name)
		} else {
			desc, err := describe.DescribeDaemonSet(ctx, client.Interface, client.OdigosClient, ns, name)
			if err != nil {
				describeText = fmt.Sprintf("Failed to describe daemonset: %s", err)
			} else {
				describeText = describe.DescribeSourceToText(desc)
			}
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
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		name := args[0]
		ns := cmd.Flag("namespace").Value.String()

		var describeText string
		if describeRemoteFlag {
			describeText = executeRemoteSourceDescribe(ctx, client, "statefulset", ns, name)
		} else {
			desc, err := describe.DescribeStatefulSet(ctx, client.Interface, client.OdigosClient, ns, name)
			if err != nil {
				describeText = fmt.Sprintf("Failed to describe statefulset: %s", err)
			} else {
				describeText = describe.DescribeSourceToText(desc)
			}
		}
		fmt.Println(describeText)
	},
}

func executeRemoteOdigosDescribe(ctx context.Context, client *kube.Client, odigosNs string) string {
	uiSvcProxyEndpoint := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/api/describe/odigos", odigosNs, k8sconsts.OdigosUiServiceName, k8sconsts.OdigosUiServicePort)
	request := client.Clientset.RESTClient().Get().AbsPath(uiSvcProxyEndpoint).Do(ctx)
	response, err := request.Raw()
	if err != nil {
		return "Remote describe failed: " + err.Error()
	} else {
		return string(response)
	}
}

func executeRemoteSourceDescribe(ctx context.Context, client *kube.Client, workloadKind string, workloadNs string, workloadName string) string {
	uiSvcProxyEndpoint := getUiServiceSourceEndpoint(ctx, client, workloadKind, workloadNs, workloadName)
	request := client.Clientset.RESTClient().Get().AbsPath(uiSvcProxyEndpoint).Do(ctx)
	response, err := request.Raw()
	if err != nil {
		return "Remote describe failed: " + err.Error()
	} else {
		return string(response)
	}
}

func getUiServiceSourceEndpoint(ctx context.Context, client *kube.Client, workloadKind string, workloadNs string, workloadName string) string {
	ns, err := resources.GetOdigosNamespace(client, ctx)
	if resources.IsErrNoOdigosNamespaceFound(err) {
		fmt.Println("\033[31mERROR\033[0m no odigos installation found in the current cluster. use \"odigos install\" to install odigos in the cluster or check that kubeconfig is pointing to the correct cluster.")
		os.Exit(1)
	} else if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already installed: %s\n", err)
		os.Exit(1)
	}

	return fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/api/describe/source/namespace/%s/kind/%s/name/%s", ns, k8sconsts.OdigosUiServiceName, k8sconsts.OdigosUiServicePort, workloadNs, workloadKind, workloadName)
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
