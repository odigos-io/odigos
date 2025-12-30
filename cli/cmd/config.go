package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/getters"
	"github.com/odigos-io/odigos/k8sutils/pkg/installationmethod"
	"github.com/odigos-io/odigos/k8sutils/pkg/restart"
	"github.com/odigos-io/odigos/k8sutils/pkg/sizing"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Odigos configuration",
	Long: fmt.Sprintf(`
	This command will be deprecated as setting values now happens via the `+"`"+`odigos install`+"`"+` command with the "--set" flag.

	Manage Odigos configuration settings to customize system behavior.

	Configurable properties:
	- "%s": Enables or disables telemetry (true/false).
	- "%s": Enables or disables OpenShift support (true/false).
	- "%s": Enables or disables Pod Security Policies (true/false).
	- "%s": Skips webhook issuer creation (true/false).
	- "%s": Allows concurrent agents (true/false).
	- "%s": Sets the image prefix.
	- "%s": Sets the UI mode (default/readonly).
	- "%s": Controls the number of items to fetch per paginated-batch in the UI.
	- "%s": Sets the public URL of a remotely, self-hosted UI.
	- "%s": Sets the URL of the Odigos Central Backend.
	- "%s": Sets the name of this cluster, for Odigos Central.
	- "%s": List of namespaces to be ignored.
	- "%s": List of containers to be ignored.
	- "%s": Determines how Odigos agent files are mounted into the pod's container filesystem. Options include k8s-host-path (direct hostPath mount) and k8s-virtual-device (virtual device-based injection).
	- "%s": Path to the custom container runtime socket (e.g /var/lib/rancher/rke2/agent/containerd/containerd.sock).
	- "%s": Directory where Kubernetes logs are symlinked in a node (e.g /mnt/var/log).
	- "%s": JSON string defining per-language env vars to customize instrumentation, e.g., `+"`"+`{"languages":{"java":{"enabled":true,"env":{"OTEL_INSTRUMENTATION_COMMON_EXPERIMENTAL_VIEW_TELEMETRY_ENABLED":"true"}}}}`+"`"+`
	- "%s": Method for injecting agent environment variables into the instrumented processes. Options include loader, pod-manifest and loader-fallback-to-pod-manifest.
	- "%s": Apply a space-separated list of Kubernetes NodeSelectors to all Odigos components (ex: "kubernetes.io/os=linux mylabel=foo").
	- "%s": Enables or disables Karpenter support (true/false).
	- "%s": Disable auto rollback feature for failing instrumentations.
	- "%s": Grace time before uninstrumenting an application [default: 5m].
	- "%s": Time windows where the auto rollback can happen [default: 1h].
	- "%s": Disable auto rollout feature for workloads when instrumenting or uninstrumenting.
	- "%s": Sets the URL of the OIDC tenant.
	- "%s": Sets the client ID of the OIDC application.
	- "%s": Sets the client secret of the OIDC application.
	- "%s": Sets the port for the Odiglet health probes (readiness/liveness).
  	- "%s": Enable or disable the service graph feature [default: false].
	- "%s": Cron schedule for automatic Go offsets updates (e.g. "0 0 * * *" for daily at midnight). Set to empty string to disable.
	- "%s": Mode for automatic Go offsets updates. Options include direct (default) and image.
	- "%s": Enable or disable ClickHouse JSON column support. When enabled, telemetry data is written using a new schema with JSON-typed columns (requires ClickHouse v25.3+). [default: false]
	- "%s": List of allowed domains for test connection endpoints (e.g., "https://api.honeycomb.io", "https://otel.example.com"). Use "*" to allow all domains. Empty list allows all domains for backward compatibility.
	- "%s": Enable or disable data compression before sending data to the Gateway collector. [default: false],
	- "%s": Set the sizing configuration for the Odigos components (size_s, size_m [default], size_l).
	- "%s": Enable wasp.
	- "%s": Enable HyperDX log normalization processor. This parses JSON from log bodies, infers severity levels, and normalizes log attributes for better querying. [default: false]
	`,
		consts.TelemetryEnabledProperty,
		consts.OpenshiftEnabledProperty,
		consts.PspProperty,
		consts.SkipWebhookIssuerCreationProperty,
		consts.AllowConcurrentAgentsProperty,
		consts.ImagePrefixProperty,
		consts.UiModeProperty,
		consts.UiPaginationLimitProperty,
		consts.UiRemoteUrlProperty,
		consts.CentralBackendURLProperty,
		consts.ClusterNameProperty,
		consts.IgnoredNamespacesProperty,
		consts.IgnoredContainersProperty,
		consts.MountMethodProperty,
		consts.CustomContainerRuntimeSocketPath,
		consts.K8sNodeLogsDirectory,
		consts.UserInstrumentationEnvsProperty,
		consts.AgentEnvVarsInjectionMethod,
		consts.NodeSelectorProperty,
		consts.KarpenterEnabledProperty,
		consts.RollbackDisabledProperty,
		consts.RollbackGraceTimeProperty,
		consts.RollbackStabilityWindow,
		consts.AutomaticRolloutDisabledProperty,
		consts.OidcTenantUrlProperty,
		consts.OidcClientIdProperty,
		consts.OidcClientSecretProperty,
		consts.OdigletHealthProbeBindPortProperty,
		consts.ServiceGraphDisabledProperty,
		consts.GoAutoOffsetsCronProperty,
		consts.GoAutoOffsetsModeProperty,
		consts.ClickhouseJsonTypeEnabledProperty,
		consts.AllowedTestConnectionHostsProperty,
		consts.EnableDataCompressionProperty,
		consts.ResourceSizePresetProperty,
		consts.WaspEnabledProperty,
		consts.HyperdxLogNormalizerProperty,
	),
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

		err = validatePropertyValue(property, value)
		if err != nil {
			l.Error(err)
			os.Exit(1)
		}
		config.ConfigVersion += 1
		err = setConfigProperty(ctx, client, config, property, value, ns)
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

		// central proxy depends on central-backend-url / cluster-name, ensure it restarts when those change
		if (property == consts.CentralBackendURLProperty || property == consts.ClusterNameProperty) &&
			currentTier == common.OnPremOdigosTier &&
			config.CentralBackendURL != "" && config.ClusterName != "" {
			if err := restart.RestartDeployment(ctx, client.Interface, ns, k8sconsts.CentralProxyDeploymentName); err != nil {
				fmt.Printf("Warning: failed to restart central-proxy: %v\n", err)
			}
		}

		l.Success()

	},
}

func validatePropertyValue(property string, value []string) error {
	switch property {
	case consts.IgnoredNamespacesProperty,
		consts.IgnoredContainersProperty,
		consts.AllowedTestConnectionHostsProperty:
		if len(value) < 1 {
			return fmt.Errorf("%s expects at least one value", property)
		}

	case consts.TelemetryEnabledProperty,
		consts.OpenshiftEnabledProperty,
		consts.PspProperty,
		consts.SkipWebhookIssuerCreationProperty,
		consts.AllowConcurrentAgentsProperty,
		consts.ImagePrefixProperty,
		consts.UiModeProperty,
		consts.UiPaginationLimitProperty,
		consts.UiRemoteUrlProperty,
		consts.CentralBackendURLProperty,
		consts.ClusterNameProperty,
		consts.MountMethodProperty,
		consts.CustomContainerRuntimeSocketPath,
		consts.K8sNodeLogsDirectory,
		consts.UserInstrumentationEnvsProperty,
		consts.AgentEnvVarsInjectionMethod,
		consts.KarpenterEnabledProperty,
		consts.RollbackDisabledProperty,
		consts.RollbackGraceTimeProperty,
		consts.RollbackStabilityWindow,
		consts.AutomaticRolloutDisabledProperty,
		consts.OidcTenantUrlProperty,
		consts.OidcClientIdProperty,
		consts.OidcClientSecretProperty,
		consts.OdigletHealthProbeBindPortProperty,
		consts.GoAutoOffsetsCronProperty,
		consts.GoAutoOffsetsModeProperty,
		consts.ServiceGraphDisabledProperty,
		consts.ClickhouseJsonTypeEnabledProperty,
		consts.EnableDataCompressionProperty,
		consts.ResourceSizePresetProperty,
		consts.WaspEnabledProperty:

		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}

		switch property {
		case consts.TelemetryEnabledProperty,
			consts.OpenshiftEnabledProperty,
			consts.PspProperty,
			consts.SkipWebhookIssuerCreationProperty,
			consts.AllowConcurrentAgentsProperty,
			consts.KarpenterEnabledProperty,
			consts.RollbackDisabledProperty,
			consts.AutomaticRolloutDisabledProperty,
			consts.ServiceGraphDisabledProperty,
			consts.EnableDataCompressionProperty,
			consts.WaspEnabledProperty:
			_, err := strconv.ParseBool(value[0])
			if err != nil {
				return fmt.Errorf("invalid boolean value for %s: %s", property, value[0])
			}

		case consts.UiPaginationLimitProperty,
			consts.OdigletHealthProbeBindPortProperty:
			_, err := strconv.Atoi(value[0])
			if err != nil {
				return fmt.Errorf("invalid integer value for %s: %s", property, value[0])
			}

		case consts.UserInstrumentationEnvsProperty:
			var uie common.UserInstrumentationEnvs
			if err := json.Unmarshal([]byte(value[0]), &uie); err != nil {
				return fmt.Errorf("invalid JSON for %s: %w", property, err)
			}

		case consts.UiModeProperty:
			uiMode := common.UiMode(value[0])
			if uiMode != common.UiModeDefault && uiMode != common.UiModeReadonly {
				return fmt.Errorf("invalid UI mode: %s (valid values: %s, %s)", value[0], common.UiModeDefault, common.UiModeReadonly)
			}

		case consts.MountMethodProperty:
			mountMethod := common.MountMethod(value[0])
			if mountMethod != common.K8sHostPathMountMethod && mountMethod != common.K8sVirtualDeviceMountMethod && mountMethod != common.K8sInitContainerMountMethod {
				return fmt.Errorf("invalid mount method: %s (valid values: %s, %s, %s)", value[0], common.K8sHostPathMountMethod, common.K8sVirtualDeviceMountMethod, common.K8sInitContainerMountMethod)
			}

		case consts.AgentEnvVarsInjectionMethod:
			injectionMethod := common.EnvInjectionMethod(value[0])
			if injectionMethod != common.LoaderEnvInjectionMethod && injectionMethod != common.PodManifestEnvInjectionMethod && injectionMethod != common.LoaderFallbackToPodManifestInjectionMethod {
				return fmt.Errorf("invalid agent env vars injection method: %s (valid values: %s, %s, %s)", value[0], common.LoaderEnvInjectionMethod, common.PodManifestEnvInjectionMethod, common.LoaderFallbackToPodManifestInjectionMethod)
			}

		case consts.NodeSelectorProperty:
			for _, v := range value {
				label := strings.Split(v, "=")
				if len(label) != 2 {
					return fmt.Errorf("nodeselector must be a valid key=value, got %s", value)
				}
			}
		}

	default:
		return fmt.Errorf("invalid property: %s", property)
	}

	return nil
}

func setConfigProperty(ctx context.Context, client *kube.Client, config *common.OdigosConfiguration, property string, value []string, namespace string) error {
	switch property {
	case consts.TelemetryEnabledProperty:
		boolValue, _ := strconv.ParseBool(value[0])
		config.TelemetryEnabled = boolValue

	case consts.OpenshiftEnabledProperty:
		boolValue, _ := strconv.ParseBool(value[0])
		config.OpenshiftEnabled = boolValue

	case consts.PspProperty:
		boolValue, _ := strconv.ParseBool(value[0])
		config.Psp = boolValue

	case consts.SkipWebhookIssuerCreationProperty:
		boolValue, _ := strconv.ParseBool(value[0])
		config.SkipWebhookIssuerCreation = boolValue

	case consts.AllowConcurrentAgentsProperty:
		boolValue, _ := strconv.ParseBool(value[0])
		config.AllowConcurrentAgents = &boolValue

	case consts.ImagePrefixProperty:
		config.ImagePrefix = value[0]

	case consts.UiModeProperty:
		config.UiMode = common.UiMode(value[0])

	case consts.UiPaginationLimitProperty:
		intValue, _ := strconv.Atoi(value[0])
		config.UiPaginationLimit = intValue

	case consts.UiRemoteUrlProperty:
		config.UiRemoteUrl = value[0]

	case consts.CentralBackendURLProperty:
		config.CentralBackendURL = value[0]

	case consts.ClusterNameProperty:
		config.ClusterName = value[0]

	case consts.IgnoredNamespacesProperty:
		config.IgnoredNamespaces = value

	case consts.IgnoredContainersProperty:
		config.IgnoredContainers = value

	case consts.MountMethodProperty:
		mountMethod := common.MountMethod(value[0])
		config.MountMethod = &mountMethod

	case consts.CustomContainerRuntimeSocketPath:
		config.CustomContainerRuntimeSocketPath = value[0]

	case consts.K8sNodeLogsDirectory:
		if config.CollectorNode == nil {
			config.CollectorNode = &common.CollectorNodeConfiguration{}
		}
		config.CollectorNode.K8sNodeLogsDirectory = value[0]

	case consts.UserInstrumentationEnvsProperty:
		var uie common.UserInstrumentationEnvs
		json.Unmarshal([]byte(value[0]), &uie)
		config.UserInstrumentationEnvs = &uie

	case consts.AgentEnvVarsInjectionMethod:
		injectionMethod := common.EnvInjectionMethod(value[0])
		config.AgentEnvVarsInjectionMethod = &injectionMethod

	case consts.NodeSelectorProperty:
		nodeSelectorMap := make(map[string]string)
		for _, v := range value {
			label := strings.Split(v, "=")
			nodeSelectorMap[label[0]] = label[1]
		}
		config.NodeSelector = nodeSelectorMap

	case consts.KarpenterEnabledProperty:
		boolValue, _ := strconv.ParseBool(value[0])
		config.KarpenterEnabled = &boolValue

	case consts.RollbackDisabledProperty:
		boolValue, _ := strconv.ParseBool(value[0])
		config.RollbackDisabled = &boolValue

	case consts.RollbackGraceTimeProperty:
		config.RollbackGraceTime = value[0]

	case consts.RollbackStabilityWindow:
		config.RollbackStabilityWindow = value[0]

	case consts.AutomaticRolloutDisabledProperty:
		if config.Rollout == nil {
			config.Rollout = &common.RolloutConfiguration{}
		}
		boolValue, _ := strconv.ParseBool(value[0])
		config.Rollout.AutomaticRolloutDisabled = &boolValue

	case consts.ServiceGraphDisabledProperty:
		if config.CollectorGateway == nil {
			config.CollectorGateway = &common.CollectorGatewayConfiguration{}
		}
		boolValue, _ := strconv.ParseBool(value[0])
		config.CollectorGateway.ServiceGraphDisabled = &boolValue

	case consts.EnableDataCompressionProperty:
		if config.CollectorNode == nil {
			config.CollectorNode = &common.CollectorNodeConfiguration{}
		}
		boolValue, _ := strconv.ParseBool(value[0])
		config.CollectorNode.EnableDataCompression = &boolValue

	case consts.OidcTenantUrlProperty:
		if config.Oidc == nil {
			config.Oidc = &common.OidcConfiguration{}
		}
		config.Oidc.TenantUrl = value[0]

	case consts.OidcClientIdProperty:
		if config.Oidc == nil {
			config.Oidc = &common.OidcConfiguration{}
		}
		config.Oidc.ClientId = value[0]

	case consts.OidcClientSecretProperty:
		// get existing secret, do not throw on not found
		secret, err := client.CoreV1().Secrets(namespace).Get(ctx, consts.OidcSecretName, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		if secret == nil || apierrors.IsNotFound(err) {
			// if the secret doesn't exist, create it
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: consts.OidcSecretName},
				Data:       map[string][]byte{consts.OidcClientSecretProperty: []byte(value[0])},
			}
			secret, err = client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		} else {
			// if the secret exists, update it
			secret.Data[consts.OidcClientSecretProperty] = []byte(value[0])
			secret, err = client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}

		// update the odigos configmap with the secret name
		if config.Oidc == nil {
			config.Oidc = &common.OidcConfiguration{}
		}
		config.Oidc.ClientSecret = fmt.Sprintf("secretRef:%s", consts.OidcSecretName)

	case consts.OdigletHealthProbeBindPortProperty:
		intValue, _ := strconv.Atoi(value[0])
		config.OdigletHealthProbeBindPort = intValue
	case consts.GoAutoOffsetsCronProperty:
		currentTier, err := odigospro.GetCurrentOdigosTier(ctx, client, namespace)
		if err != nil {
			return fmt.Errorf("unable to read the current Odigos tier: %w", err)
		}
		if currentTier == common.CommunityOdigosTier {
			return fmt.Errorf("custom offsets support is only available in Odigos pro tier.")
		}
		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}
		cronValue := value[0]
		if cronValue != "" {
			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
			if _, err := parser.Parse(cronValue); err != nil {
				return fmt.Errorf("invalid cron schedule: %v", err)
			}
		}
		config.GoAutoOffsetsCron = cronValue

	case consts.GoAutoOffsetsModeProperty:
		currentTier, err := odigospro.GetCurrentOdigosTier(ctx, client, namespace)
		if err != nil {
			return fmt.Errorf("unable to read the current Odigos tier: %w", err)
		}
		if currentTier == common.CommunityOdigosTier {
			return fmt.Errorf("custom offsets support is only available in Odigos pro tier.")
		}
		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}
		modeValue := value[0]
		if modeValue != "" {
			mode := k8sconsts.OffsetCronJobMode(modeValue)
			if !mode.IsValid() {
				return fmt.Errorf("invalid mode: %s. Must be one of: %s, %s, %s",
					modeValue, k8sconsts.OffsetCronJobModeDirect, k8sconsts.OffsetCronJobModeImage, k8sconsts.OffsetCronJobModeOff)
			}
		}
		config.GoAutoOffsetsMode = modeValue

	case consts.ClickhouseJsonTypeEnabledProperty:
		boolValue, _ := strconv.ParseBool(value[0])
		config.ClickhouseJsonTypeEnabledProperty = &boolValue

	case consts.HyperdxLogNormalizerProperty:
		boolValue, _ := strconv.ParseBool(value[0])
		config.HyperdxLogNormalizer = &boolValue

	case consts.AllowedTestConnectionHostsProperty:
		config.AllowedTestConnectionHosts = value

	case consts.ResourceSizePresetProperty:
		if !sizing.IsValidSizing(value[0]) {
			return fmt.Errorf("invalid sizing config: %s (valid values: %s, %s, %s)", value[0], sizing.SizeSmall, sizing.SizeMedium, sizing.SizeLarge)
		}
		config.ResourceSizePreset = value[0]

	case consts.WaspEnabledProperty:
		boolValue, _ := strconv.ParseBool(value[0])
		config.WaspEnabled = &boolValue

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
