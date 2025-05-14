package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/getters"
	"github.com/odigos-io/odigos/k8sutils/pkg/installationmethod"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Odigos configuration",
	Long: `Manage Odigos configuration settings to customize system behavior.

	Configurable properties:
	- "telemetry-enabled": Enables or disables telemetry (true/false).
	- "openshift-enabled": Enables or disables OpenShift support (true/false).
	- "psp": Enables or disables Pod Security Policies (true/false).
	- "skip-webhook-issuer-creation": Skips webhook issuer creation (true/false).
	- "allow-concurrent-agents": Allows concurrent agents (true/false).
	- "image-prefix": Sets the image prefix.
	- "ui-mode": Sets the UI mode (normal/readonly).
	- "ui-pagination-limit": Controls the number of items to fetch per paginated-batch in the UI.
	- "ignored-namespaces": List of namespaces to be ignored.
	- "ignored-containers": List of containers to be ignored.
	- "mount-method": Determines how Odigos agent files are mounted into the pod's container filesystem. Options include k8s-host-path (direct hostPath mount) and k8s-virtual-device (virtual device-based injection).
	- "container-runtime-socket-path": Path to the custom container runtime socket (e.g /var/lib/rancher/rke2/agent/containerd/containerd.sock).
	- "k8s-node-logs-directory": Directory where Kubernetes logs are symlinked in a node (e.g /mnt/var/log).
	- "avoid-java-opts-env-var": Avoid injecting the Odigos value in JAVA_OPTS environment variable into Java applications.
	- "agent-env-vars-injection-method": Method for injecting agent environment variables into the instrumented processes. Options include loader, pod-manifest and loader-fallback-to-pod-manifest.
	- "node-selector": Apply a space-separated list of Kubernetes NodeSelectors to all Odigos components (ex: "kubernetes.io/os=linux mylabel=foo").
	`,
}

// `odigos config set <property> <value>`
var setConfigCmd = &cobra.Command{
	Use:   "set <property> <value>",
	Short: "Set a configuration property in Odigos",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		property := args[0]
		value := args[1:]

		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)
		ns, err := resources.GetOdigosNamespace(client, ctx)

		l := log.Print(fmt.Sprintf("Updating %s to %s...", property, value))

		config, err := resources.GetCurrentConfig(ctx, client, ns)
		if err != nil {
			l.Error(fmt.Errorf("unable to read the current Odigos configuration: %w", err))
			os.Exit(1)
		}

		config.ConfigVersion += 1
		err = setConfigProperty(config, property, value)
		if err != nil {
			l.Error(err)
			os.Exit(1)
		}

		currentTier, err := odigospro.GetCurrentOdigosTier(ctx, client, ns)
		if err != nil {
			l.Error(fmt.Errorf("unable to read the current Odigos tier: %w", err))
			os.Exit(1)
		}

		currentOdigosVersion, err := getters.GetOdigosVersionInClusterFromConfigMap(ctx, client.Clientset, ns)
		if err != nil {
			fmt.Println("Odigos config failed - unable to read the current Odigos version.")
			os.Exit(1)
		}

		managerOpts := resourcemanager.ManagerOpts{
			ImageReferences: GetImageReferences(currentTier, openshiftEnabled),
		}

		resourceManagers := resources.CreateResourceManagers(client, ns, currentTier, nil, config, currentOdigosVersion, installationmethod.K8sInstallationMethodOdigosCli, managerOpts)
		err = resources.ApplyResourceManagers(ctx, client, resourceManagers, "Updating Config")
		if err != nil {
			l.Error(fmt.Errorf("failed to apply updated configuration: %w", err))
			os.Exit(1)
		}

		err = resources.DeleteOldOdigosSystemObjects(ctx, client, ns, config)
		if err != nil {
			fmt.Println("Odigos config update failed - unable to cleanup old Odigos resources.")
			os.Exit(1)
		}

		l.Success()

	},
}

func setConfigProperty(config *common.OdigosConfiguration, property string, value []string) error {
	switch property {
	case consts.CentralBackendURLProperty:
		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}
		config.CentralBackendURL = value[0]

	case consts.TelemetryEnabledProperty, consts.OpenshiftEnabledProperty, consts.PspProperty,
		consts.SkipWebhookIssuerCreationProperty, consts.AllowConcurrentAgentsProperty, consts.AvoidJavaOptsEnvVar,
		consts.KarpenterEnabledProperty:

		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value (true/false)", property)
		}
		boolValue, err := strconv.ParseBool(value[0])
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", property, value[0])
		}

		switch property {
		case consts.TelemetryEnabledProperty:
			config.TelemetryEnabled = boolValue
		case consts.OpenshiftEnabledProperty:
			config.OpenshiftEnabled = boolValue
		case consts.PspProperty:
			config.Psp = boolValue
		case consts.SkipWebhookIssuerCreationProperty:
			config.SkipWebhookIssuerCreation = boolValue
		case consts.AllowConcurrentAgentsProperty:
			config.AllowConcurrentAgents = &boolValue
		case consts.AvoidJavaOptsEnvVar:
			config.AvoidInjectingJavaOptsEnvVar = &boolValue
		case consts.KarpenterEnabledProperty:
			config.KarpenterEnabled = &boolValue
		}

	case consts.ImagePrefixProperty, consts.UiModeProperty, consts.UiPaginationLimit:

		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}
		switch property {
		case consts.ImagePrefixProperty:
			config.ImagePrefix = value[0]
		case consts.UiModeProperty:
			config.UiMode = common.UiMode(value[0])
		case consts.UiPaginationLimit:
			intValue, err := strconv.Atoi(value[0])
			if err != nil {
				return fmt.Errorf("invalid integer value for %s: %s", property, value[0])
			}
			config.UiPaginationLimit = intValue
		}

	case consts.IgnoredNamespacesProperty:
		if len(value) < 1 {
			return fmt.Errorf("%s expects at least one value", property)
		}
		config.IgnoredNamespaces = value

	case consts.IgnoredContainersProperty:
		if len(value) < 1 {
			return fmt.Errorf("%s expects at least one value", property)
		}
		config.IgnoredContainers = value

	case consts.CustomContainerRuntimeSocketPath:
		if len(value) != 1 {
			return fmt.Errorf("%s expects one value", property)
		}
		config.CustomContainerRuntimeSocketPath = value[0]

	case consts.K8sNodeLogsDirectory:
		if len(value) != 1 {
			return fmt.Errorf("%s expects one value", property)
		}
		if config.CollectorNode == nil {
			config.CollectorNode = &common.CollectorNodeConfiguration{}
		}
		config.CollectorNode.K8sNodeLogsDirectory = value[0]

	case consts.MountMethodProperty:
		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}
		mountMethod := common.MountMethod(value[0])
		switch mountMethod {
		case common.K8sHostPathMountMethod, common.K8sVirtualDeviceMountMethod:
			config.MountMethod = &mountMethod
		default:
			return fmt.Errorf("invalid mount method: %s (valid values: %s, %s)", value[0],
				common.K8sHostPathMountMethod, common.K8sVirtualDeviceMountMethod)
		}

	case consts.ClusterNameProperty:
		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}
		config.ClusterName = value[0]

	case consts.AgentEnvVarsInjectionMethod:
		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}

		injectionMethod := common.EnvInjectionMethod(value[0])
		switch injectionMethod {
		case common.LoaderEnvInjectionMethod, common.PodManifestEnvInjectionMethod, common.LoaderFallbackToPodManifestInjectionMethod:
			config.AgentEnvVarsInjectionMethod = &injectionMethod
		default:
			return fmt.Errorf("invalid agent env vars injection method: %s (valid values: %s, %s, %s)", value[0],
				common.LoaderEnvInjectionMethod, common.PodManifestEnvInjectionMethod, common.LoaderFallbackToPodManifestInjectionMethod)
		}
	case consts.NodeSelectorProperty:
		nodeSelectorMap := make(map[string]string)
		for _, v := range value {
			label := strings.Split(v, "=")
			if len(label) != 2 {
				return fmt.Errorf("nodeselector must be a valid key=value, got %s", value)
			}
			nodeSelectorMap[label[0]] = label[1]
		}
		config.NodeSelector = nodeSelectorMap

	default:
		return fmt.Errorf("invalid property: %s", property)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(setConfigCmd)

	setConfigCmd.Flags().StringP("namespace", "n", consts.DefaultOdigosNamespace, "Namespace where Odigos is installed")
}
