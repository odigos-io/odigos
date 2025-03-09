package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
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
	- "ui-mode": Sets the UI mode(normal/readonly).
	- "ignored-namespaces": List of namespaces to be ignored.
	- "ignored-containers": List of containers to be ignored.
	- "mount-method": Determines how Odigos agent files are mounted into the pod's container filesystem. Options include k8s-host-path (direct hostPath mount) and k8s-virtual-device (virtual device-based injection).
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

		resourceManagers := resources.CreateResourceManagers(client, ns, currentTier, nil, config, currentOdigosVersion, installationmethod.K8sInstallationMethodOdigosCli)
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
		consts.SkipWebhookIssuerCreationProperty, consts.AllowConcurrentAgentsProperty:

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
		}

	case consts.ImagePrefixProperty, consts.OdigletImageProperty, consts.InstrumentorImageProperty,
		consts.AutoscalerImageProperty, consts.UiModeProperty:

		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}
		switch property {
		case consts.ImagePrefixProperty:
			config.ImagePrefix = value[0]
		case consts.OdigletImageProperty:
			config.OdigletImage = value[0]
		case consts.InstrumentorImageProperty:
			config.InstrumentorImage = value[0]
		case consts.AutoscalerImageProperty:
			config.AutoscalerImage = value[0]
		case consts.UiModeProperty:
			config.UiMode = common.UiMode(value[0])
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
