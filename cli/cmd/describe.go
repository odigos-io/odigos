package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
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

		if err != nil {
			log.Print("unable to get the Odigos Namespace")
			os.Exit(1)
		}

		config, err := resources.GetCurrentConfig(ctx, client, ns)

		if err != nil {
			log.Print("unable to read the current Odigos configuration")
			os.Exit(1)
		}

		log.Print(`Manage Odigos configuration settings to customize system behavior.` + "\n")

		log.Print(`Configurable properties` + "\n")

		populateConfValues(config)
		printAll()
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

// work on later
func populateConfValues(config *common.OdigosConfiguration) {
	consts.ConfigValues[consts.TelemetryEnabledProperty] = config.TelemetryEnabled
	consts.ConfigValues[consts.OpenshiftEnabledProperty] = config.OpenshiftEnabled
	consts.ConfigValues[consts.PspProperty] = config.Psp
	consts.ConfigValues[consts.SkipWebhookIssuerCreationProperty] = config.SkipWebhookIssuerCreation
	consts.ConfigValues[consts.AllowConcurrentAgentsProperty] = config.AllowConcurrentAgents
	consts.ConfigValues[consts.ImagePrefixProperty] = config.ImagePrefix
	consts.ConfigValues[consts.UiModeProperty] = config.UiMode
	consts.ConfigValues[consts.UiPaginationLimitProperty] = config.UiPaginationLimit
	consts.ConfigValues[consts.UiRemoteUrlProperty] = config.UiRemoteUrl
	consts.ConfigValues[consts.CentralBackendURLProperty] = config.CentralBackendURL
	consts.ConfigValues[consts.ClusterNameProperty] = config.ClusterName
	consts.ConfigValues[consts.IgnoredNamespacesProperty] = config.IgnoredNamespaces
	consts.ConfigValues[consts.IgnoredContainersProperty] = config.IgnoredContainers
	consts.ConfigValues[consts.MountMethodProperty] = config.MountMethod
	consts.ConfigValues[consts.CustomContainerRuntimeSocketPath] = config.CustomContainerRuntimeSocketPath
	consts.ConfigValues[consts.K8sNodeLogsDirectory] = config.CollectorNode.K8sNodeLogsDirectory
	consts.ConfigValues[consts.UserInstrumentationEnvsProperty] = config.UserInstrumentationEnvs
	consts.ConfigValues[consts.AgentEnvVarsInjectionMethod] = config.AgentEnvVarsInjectionMethod
	consts.ConfigValues[consts.NodeSelectorProperty] = config.NodeSelector
	consts.ConfigValues[consts.KarpenterEnabledProperty] = config.KarpenterEnabled
	consts.ConfigValues[consts.RollbackDisabledProperty] = config.RollbackDisabled
	consts.ConfigValues[consts.RollbackGraceTimeProperty] = config.RollbackGraceTime
	consts.ConfigValues[consts.RollbackStabilityWindow] = config.RollbackStabilityWindow
	if config.Rollout == nil {
		consts.ConfigValues[consts.AutomaticRolloutDisabledProperty] = nil
	} else {
		consts.ConfigValues[consts.AutomaticRolloutDisabledProperty] = config.Rollout
	}
	if config.Oidc == nil {
		consts.ConfigValues[consts.OidcTenantUrlProperty] = nil
		consts.ConfigValues[consts.OidcClientIdProperty] = nil
		consts.ConfigValues[consts.OidcClientSecretProperty] = nil
	} else {
		consts.ConfigValues[consts.OidcTenantUrlProperty] = config.Oidc.TenantUrl
		consts.ConfigValues[consts.OidcClientIdProperty] = config.Oidc.ClientId
		consts.ConfigValues[consts.OidcClientSecretProperty] = config.Oidc.ClientSecret
	}
	consts.ConfigValues[consts.OdigletHealthProbeBindPortProperty] = config.OdigletHealthProbeBindPort
	if config.CollectorGateway == nil {
		consts.ConfigValues[consts.ServiceGraphDisabledProperty] = nil
	} else {
		consts.ConfigValues[consts.ServiceGraphDisabledProperty] = config.CollectorGateway.ServiceGraphDisabled
	}
}

func printAll() {
	for key, value := range consts.ConfigValues {
		switch v := value.(type) {
		case string:
			printString(v, key)
		case bool:
			printBool(v, key)
		case *bool:
			printPointerBool(v, key)
		case *common.MountMethod:
			printPointerFileStuff(v, key)
		case *common.EnvInjectionMethod:
			printPointerFileStuff2(v, key)
		case []string:
			printLists(v, key)
		case *common.CollectorNodeConfiguration:
			printStructStuff(v, key)
		case *common.UserInstrumentationEnvs:
			printUserEnv(v, key)
		case map[string]string:
			printNodeSelector(v, key)
		case *common.RolloutConfiguration:
			printStructStuff2(v, key)
		case *common.OidcConfiguration:
			printStructStuff3(v, key)
		case *common.CollectorGatewayConfiguration:
			printStructStuff4(v, key)
		default:
			printDefault(v, key)
		}
	}

}

func printDefault(featureSetting interface{}, featureName string) {
	if featureSetting == nil {
		log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
	} else {
		log.Print(fmt.Sprintf("- %s: %s status: %v\n", featureName, consts.ConfigDisplay[featureName], featureSetting))
	}
}

func printString(featureSetting string, featureName string) {
	if featureSetting == "" {
		log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
	} else {
		log.Print(fmt.Sprintf("- %s: %s status: %s\n", featureName, consts.ConfigDisplay[featureName], featureSetting))
	}
}

func printBool(featureSetting bool, featureName string) {
	log.Print(fmt.Sprintf("- %s: %s status: %t\n", featureName, consts.ConfigDisplay[featureName], featureSetting))
}

func printPointerBool(featureSetting *bool, featureName string) {
	if featureSetting != nil {
		log.Print(fmt.Sprintf("- %s: %s status: %t\n", featureName, consts.ConfigDisplay[featureName], *featureSetting))
	} else {
		log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
	}
}

func printLists(featureSetting []string, featureName string) {
	log.Print(fmt.Sprintf("- %s: List of what should be ignored.\n", featureName))
	if len(featureSetting) == 0 {
		log.Print("none found\n")
	} else {
		for i := 0; i < len(featureSetting); i++ {
			log.Print(fmt.Sprintf("- %s\n", featureSetting[i]))
		}
	}
}

func printPointerFileStuff(featureSetting *common.MountMethod, featureName string) {
	if featureSetting != nil {
		log.Print(fmt.Sprintf("- %s: %s status: %s\n", featureName, consts.ConfigDisplay[featureName], *featureSetting))
	} else {
		log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
	}
}

func printPointerFileStuff2(featureSetting *common.EnvInjectionMethod, featureName string) {
	if featureSetting != nil {
		log.Print(fmt.Sprintf("- %s: %s status: %s\n", featureName, consts.ConfigDisplay[featureName], *featureSetting))
	} else {
		log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
	}
}

func printStructStuff(featureSetting *common.CollectorNodeConfiguration, featureName string) {
	if featureSetting == nil {
		log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
	} else {
		if featureSetting.K8sNodeLogsDirectory == "" {
			log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
		} else {
			log.Print(fmt.Sprintf("- %s: %s status: %s\n", featureName, consts.ConfigDisplay[featureName], featureSetting.K8sNodeLogsDirectory))
		}
	}
}

func printStructStuff2(featureSetting *common.RolloutConfiguration, featureName string) {
	if featureSetting.AutomaticRolloutDisabled == nil {
		log.Print(fmt.Sprintf("- %s: %s status: not set \n", featureName, consts.ConfigDisplay[featureName]))
	} else {
		printPointerBool(featureSetting.AutomaticRolloutDisabled, featureName)
	}
}

func printStructStuff3(featureSetting *common.OidcConfiguration, featureName string) {
	switch featureName {
	case consts.OidcTenantUrlProperty:
		printString(featureSetting.TenantUrl, featureName)
	case consts.OidcClientIdProperty:
		printString(featureSetting.ClientId, featureName)
	case consts.OidcClientSecretProperty:
		printString(featureSetting.ClientSecret, featureName)
	}

}

func printStructStuff4(featureSetting *common.CollectorGatewayConfiguration, featureName string) {
	if featureSetting.ServiceGraphDisabled == nil {
		log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
	} else {
		printPointerBool(featureSetting.ServiceGraphDisabled, featureName)
	}
}

func printUserEnv(featureSetting *common.UserInstrumentationEnvs, featureName string) {
	if featureSetting == nil {
		log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
	} else if len(featureSetting.Languages) == 0 {
		log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
	} else {
		log.Print(fmt.Sprintf("- %s: %s status: \n", featureName, consts.ConfigDisplay[featureName]))
		for lang, env := range featureSetting.Languages {
			fmt.Printf("Language: %+v, Mode: %+v\n", lang, env)
		}
	}
}

func printNodeSelector(featureSetting map[string]string, featureName string) {
	if len(featureSetting) == 0 {
		log.Print(fmt.Sprintf("- %s: %s status: not set\n", featureName, consts.ConfigDisplay[featureName]))
	} else {
		log.Print(fmt.Sprintf("- %s: %s\n", featureName, consts.ConfigDisplay[featureName]))
		for key, val := range featureName {
			fmt.Printf("key: %+v, value: %+v\n", key, val)
		}
	}
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

	// config
	describeCmd.AddCommand(describeConfigCmd)

	// source kinds
	describeSourceCmd.AddCommand(describeSourceDeploymentCmd)
	describeSourceCmd.AddCommand(describeSourceDaemonSetCmd)
	describeSourceCmd.AddCommand(describeSourceStatefulSetCmd)

}
