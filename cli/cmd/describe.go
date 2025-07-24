package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe"
	"github.com/spf13/cobra"

	// new ones for the odigos describe config command
	"github.com/odigos-io/odigos/cli/pkg/log"
	//"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	// "github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	// "github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
)

var (
	describeNamespaceFlag string
	describeRemoteFlag    bool
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Show details of a specific odigos entity",
	Long:  "This command can be used for troubleshooting and observing odigos state for its various entities.<br />It is similar to `kubectl describe` but for Odigos entities.",
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
	Example: `
# Describe a source of kind deployment and name myservice in the default namespace
odigos describe source deployment myservice -n default

Output:

Name:  myservice
Kind:  deployment
Namespace:  default

Labels:
  Instrumented:  true
  Workload: odigos-instrumentation=enabled
  Namespace: odigos-instrumentation=enabled
  Decision: Workload is instrumented because the deployment contains the label 'odigos-instrumentation=enabled'
  Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#1-odigos-instrumentation-label

Instrumentation Config:
  Created at 2024-07-30 19:00:40 +0300 IDT

Runtime inspection details:
  Created at 2024-07-30 19:00:40 +0300 IDT
  Detected Containers:
    - Container Name: myservice
      Language:       javascript
      Relevant Environment Variables:
        - NODE_OPTIONS : --require /var/odigos/nodejs/autoinstrumentation.js

Instrumentation Device:
  Status: Successfully applied instrumentation device to pod template
  - Container Name: myservice
    Instrumentation Devices: javascript-native-community

Pods (Total 1, Running 1):
  Pod Name: myservice-ffd68d8c-qqmxl
  Pod Phase: Running
  Pod Node Name: kind-control-plane
  Containers:
  - Container Name: myservice
    Instrumentation Devices: javascript-native-community
    Instrumentation Instances:
    - Healthy: true
      Reason: Healthy
`,
}

var describeSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Show details of a specific odigos source",
	Long:  `Print detailed description of a specific odigos source, which can be used to troubleshoot issues`,
}

var describeConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show details of odigos-config configmap",
	Long:  "Print detailed description of the odigos-config map giving info on whether certain features are on or off",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		ns, err := resources.GetOdigosNamespace(client, ctx)
		config, err := resources.GetCurrentConfig(ctx, client, ns)

		if err != nil {
			log.Print("unable to read the current Odigos configuration")
			os.Exit(1)
		}

		log.Print(`Manage Odigos configuration settings to customize system behavior. 
		
		Configurable properties:` + "\n")

		log.Print(fmt.Sprintf("- %s: Enables or disables telemetry %t.\n", consts.TelemetryEnabledProperty, config.TelemetryEnabled))

		log.Print(fmt.Sprintf("- %s: Enables or disables OpenShift support %t.\n", consts.OpenshiftEnabledProperty, config.OpenshiftEnabled))

		log.Print(fmt.Sprintf("- %s: Enables or disables Pod Security Policies %t.\n", consts.PspProperty, config.Psp))

		log.Print(fmt.Sprintf("- %s: Skips webhook issuer creation %t.\n", consts.SkipWebhookIssuerCreationProperty, config.SkipWebhookIssuerCreation))

		if config.AllowConcurrentAgents != nil {
			log.Print(fmt.Sprintf("- %s: Allows concurrent agents %t.\n", consts.AllowConcurrentAgentsProperty, *config.AllowConcurrentAgents))
		} else {
			log.Print(fmt.Sprintf("- %s: Allows concurrent agents (not set)\n", consts.AllowConcurrentAgentsProperty))
		}

		if config.ImagePrefix == "" {
			log.Print(fmt.Sprintf("- %s: Sets the image prefix (not set)\n", consts.ImagePrefixProperty))
		} else {
			log.Print(fmt.Sprintf("- %s: Sets the image prefix. %s\n", consts.ImagePrefixProperty, config.ImagePrefix))
		}

		if config.UiMode == "" {
			log.Print(fmt.Sprintf("- %s: Sets the UI mode. (not set)\n", consts.UiModeProperty))
		} else {
			log.Print(fmt.Sprintf("- %s: Sets the UI mode. %s.\n", consts.UiModeProperty, config.UiMode))
		}

		log.Print(fmt.Sprintf("- %s: Controls the number of items to fetch per paginated-batch in the UI. %d.\n",
			consts.UiPaginationLimitProperty, config.UiPaginationLimit))

		if config.UiRemoteUrl == "" {
			log.Print(fmt.Sprintf("- %s: Sets the URL of the Odigos Central Backend. (not set)\n", consts.UiRemoteUrlProperty))
		} else {
			log.Print(fmt.Sprintf("- %s: Sets the URL of the Odigos Central Backend. %s.\n", consts.UiRemoteUrlProperty, config.UiRemoteUrl))
		}

		if config.CentralBackendURL == "" {
			log.Print(fmt.Sprintf("- %s: Sets the name of this cluster, for Odigos Central. (not set)\n", consts.CentralBackendURLProperty))
		} else {
			log.Print(fmt.Sprintf("- %s: Sets the name of this cluster, for Odigos Central. %s.\n", consts.CentralBackendURLProperty, config.CentralBackendURL))
		}

		log.Print(fmt.Sprintf("- %s: List of namespaces to be ignored.\n", consts.IgnoredNamespacesProperty))
		if len(config.IgnoredNamespaces) == 0 {
			log.Print("none found\n")
		} else {
			for i := 0; i < len(config.IgnoredNamespaces); i++ {
				log.Print(fmt.Sprintf("- %s\n", config.IgnoredNamespaces[i]))
			}
		}

		log.Print(fmt.Sprintf("- %s: List of containers to be ignored.\n", consts.IgnoredContainersProperty))
		if len(config.IgnoredContainers) == 0 {
			log.Print("none found\n")
		} else {
			for i := 0; i < len(config.IgnoredContainers); i++ {
				log.Print(fmt.Sprintf("- %s\n", config.IgnoredContainers[i]))
			}
		}

		if config.MountMethod != nil {
			log.Print(fmt.Sprintf("- %s: Determines how Odigos agent files are mounted into the pod's container filesystem. Options include k8s-host-path (direct hostPath mount) and k8s-virtual-device (virtual device-based injection). %s\n", consts.MountMethodProperty, *config.MountMethod))
		} else {
			log.Print(fmt.Sprintf("- %s: Determines how Odigos agent files are mounted into the pod's container filesystem. Options include k8s-host-path (direct hostPath mount) and k8s-virtual-device (virtual device-based injection). (not set)\n", consts.MountMethodProperty))
		}

		if config.CustomContainerRuntimeSocketPath == "" {
			log.Print(fmt.Sprintf("- %s: Path to the custom container runtime socket (e.g /var/lib/rancher/rke2/agent/containerd/containerd.sock). (not set)\n", consts.CustomContainerRuntimeSocketPath))
		} else {
			log.Print(fmt.Sprintf("- %s: Path to the custom container runtime socket (e.g /var/lib/rancher/rke2/agent/containerd/containerd.sock). %s.\n", consts.CustomContainerRuntimeSocketPath, config.CustomContainerRuntimeSocketPath))
		}

		if config.CollectorNode == nil {
			log.Print(fmt.Sprintf("- %s: Directory where Kubernetes logs are symlinked in a node (e.g /mnt/var/log). (not set)\n", consts.K8sNodeLogsDirectory))
		} else {
			if config.CollectorNode.K8sNodeLogsDirectory == "" {
				log.Print(fmt.Sprintf("- %s: Directory where Kubernetes logs are symlinked in a node (e.g /mnt/var/log). (not set)\n", consts.K8sNodeLogsDirectory))
			} else {
				log.Print(fmt.Sprintf("- %s: Directory where Kubernetes logs are symlinked in a node (e.g /mnt/var/log). %s.\n", consts.K8sNodeLogsDirectory, config.CollectorNode.K8sNodeLogsDirectory))
			}
		}

		if config.UserInstrumentationEnvs == nil {
			log.Print(fmt.Sprintf("- %s: JSON string defining per-language env vars to customize instrumentation. (not set)\n", consts.UserInstrumentationEnvsProperty))
		} else if len(config.UserInstrumentationEnvs.Languages) == 0 {
			log.Print(fmt.Sprintf("- %s: JSON string defining per-language env vars to customize instrumentation. (not set)\n", consts.UserInstrumentationEnvsProperty))
		} else {
			log.Print(fmt.Sprintf("- %s: JSON string defining per-language env vars to customize instrumentation. \n", consts.UserInstrumentationEnvsProperty))
			for lang, env := range config.UserInstrumentationEnvs.Languages {
				fmt.Printf("Language: %+v, Mode: %+v\n", lang, env)
			}
		}

		if config.AgentEnvVarsInjectionMethod == nil {
			log.Print(fmt.Sprintf("- %s: Directory where Kubernetes logs are symlinked in a node (e.g /mnt/var/log). (not set)\n", consts.AgentEnvVarsInjectionMethod))
		} else {
			if *config.AgentEnvVarsInjectionMethod == "" {
				log.Print(fmt.Sprintf("- %s: Directory where Kubernetes logs are symlinked in a node (e.g /mnt/var/log). (not set)\n", consts.AgentEnvVarsInjectionMethod))
			} else {
				log.Print(fmt.Sprintf("- %s: Directory where Kubernetes logs are symlinked in a node (e.g /mnt/var/log). %s.\n", consts.AgentEnvVarsInjectionMethod, *config.AgentEnvVarsInjectionMethod))
			}
		}

		if len(config.NodeSelector) == 0 {
			log.Print(fmt.Sprintf("- %s: Apply a space-separated list of Kubernetes NodeSelectors to all Odigos components (ex: 'kubernetes.io/os=linux mylabel=foo'). (not set)\n", consts.UserInstrumentationEnvsProperty))
		} else {
			log.Print(fmt.Sprintf("- %s: Apply a space-separated list of Kubernetes NodeSelectors to all Odigos components (ex: 'kubernetes.io/os=linux mylabel=foo'). \n", consts.UserInstrumentationEnvsProperty))
			for key, val := range config.NodeSelector {
				fmt.Printf("key: %+v, value: %+v\n", key, val)
			}
		}

		if config.KarpenterEnabled != nil {
			log.Print(fmt.Sprintf("- %s: Enables or disables Karpenter support (true/false). %t.\n", consts.KarpenterEnabledProperty, *config.KarpenterEnabled))
		} else {
			log.Print(fmt.Sprintf("- %s: Enables or disables Karpenter support (true/false). (not set)\n", consts.KarpenterEnabledProperty))
		}

		if config.RollbackDisabled != nil {
			log.Print(fmt.Sprintf("- %s: Disable auto rollback feature for failing instrumentations. %t.\n", consts.RollbackDisabledProperty, *config.RollbackDisabled))
		} else {
			log.Print(fmt.Sprintf("- %s: Disable auto rollback feature for failing instrumentations. (not set)\n", consts.RollbackDisabledProperty))
		}

		log.Print(fmt.Sprintf("- %s: Grace time before uninstrumenting an application [default: 5m]. %s.\n", consts.RollbackGraceTimeProperty, config.RollbackGraceTime))

		log.Print(fmt.Sprintf("- %s: Time windows where the auto rollback can happen [default: 1h]. %s.\n", consts.RollbackStabilityWindow, config.RollbackStabilityWindow))

		if config.Rollout.AutomaticRolloutDisabled == nil {
			log.Print(fmt.Sprintf("- %s: Disable auto rollout feature for workloads when instrumenting or uninstrumenting. (not set).\n", consts.AutomaticRolloutDisabledProperty))
		} else {
			log.Print(fmt.Sprintf("- %s: Disable auto rollout feature for workloads when instrumenting or uninstrumenting. %t\n", consts.AutomaticRolloutDisabledProperty, *config.Rollout.AutomaticRolloutDisabled))
		}

		if config.Oidc == nil {
			log.Print(fmt.Sprintf("- %s: Sets the URL of the OIDC tenant. (not set)\n", consts.OidcTenantUrlProperty))
		} else {
			if config.Oidc.TenantUrl == "" {
				log.Print(fmt.Sprintf("- %s: Sets the URL of the OIDC tenant. (not set)\n", consts.OidcTenantUrlProperty))
			} else {
				log.Print(fmt.Sprintf("- %s: Sets the URL of the OIDC tenant. %s\n", consts.OidcTenantUrlProperty, config.Oidc.TenantUrl))
			}
		}

		if config.Oidc == nil {
			log.Print(fmt.Sprintf("- %s: Sets the client ID of the OIDC application. (not set)\n", consts.OidcClientIdProperty))
		} else {
			if config.Oidc.ClientId == "" {
				log.Print(fmt.Sprintf("- %s: Sets the client ID of the OIDC application. (not set)\n", consts.OidcClientIdProperty))
			} else {
				log.Print(fmt.Sprintf("- %s: Sets the client ID of the OIDC application. %s\n", consts.OidcClientIdProperty, config.Oidc.ClientId))
			}
		}

		// maybe don't show the secret for security reasons?
		if config.Oidc == nil {
			log.Print(fmt.Sprintf("- %s: Sets the client secret of the OIDC application. (not set)\n", consts.OidcClientSecretProperty))
		} else {
			if config.Oidc.ClientSecret == "" {
				log.Print(fmt.Sprintf("- %s: Sets the client secret of the OIDC application. (not set)\n", consts.OidcClientSecretProperty))
			} else {
				log.Print(fmt.Sprintf("- %s: Sets the client secret of the OIDC application. %s\n", consts.OidcClientSecretProperty, config.Oidc.ClientSecret))
			}
		}

		log.Print(fmt.Sprintf("- %s: Time windows where the auto rollback can happen [default: 1h]. %d.\n", consts.OdigletHealthProbeBindPortProperty, config.OdigletHealthProbeBindPort))

		if config.CollectorGateway.ServiceGraphDisabled == nil {
			log.Print(fmt.Sprintf("- %s: Enable or disable the service graph feature [default: false]. (not set).\n", consts.ServiceGraphConnectorName))
		} else {
			log.Print(fmt.Sprintf("- %s: Enable or disable the service graph feature [default: false]. %t\n", consts.ServiceGraphConnectorName, *config.CollectorGateway.ServiceGraphDisabled))
		}
	},
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

	// config
	describeSourceCmd.AddCommand(describeConfigCmd)
}
