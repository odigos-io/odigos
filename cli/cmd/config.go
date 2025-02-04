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
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Odigos configuration",
	Long:  "Manage Odigos configuration settings to customize system behavior.",
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
			odigosConfig, err := resources.GetDeprecatedConfig(ctx, client, ns)
			if err != nil {
				l.Error(fmt.Errorf("unable to read the current Odigos configuration: %w", err))
				os.Exit(1)
			}
			config = odigosConfig.ToCommonConfig()
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

		resourceManagers := resources.CreateResourceManagers(client, ns, currentTier, nil, config, "Updating Config")
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
	case "central-backend-url":
		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}
		config.CentralBackendURL = value[0]

	case "telemetry-enabled", "openshift-enabled", "psp", "skip-webhook-issuer-creation", "allow-concurrent-agents":
		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value (true/false)", property)
		}
		boolValue, err := strconv.ParseBool(value[0])
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", property, value[0])
		}

		switch property {
		case "telemetry-enabled":
			config.TelemetryEnabled = boolValue
		case "openshift-enabled":
			config.OpenshiftEnabled = boolValue
		case "psp":
			config.Psp = boolValue
		case "skip-webhook-issuer-creation":
			config.SkipWebhookIssuerCreation = boolValue
		case "allow-concurrent-agents":
			config.AllowConcurrentAgents = &boolValue
		}

	case "image-prefix", "odiglet-image", "instrumentor-image", "autoscaler-image", "ui-mode":
		if len(value) != 1 {
			return fmt.Errorf("%s expects exactly one value", property)
		}
		switch property {
		case "image-prefix":
			config.ImagePrefix = value[0]
		case "odiglet-image":
			config.OdigletImage = value[0]
		case "instrumentor-image":
			config.InstrumentorImage = value[0]
		case "autoscaler-image":
			config.AutoscalerImage = value[0]
		case "ui-mode":
			config.UiMode = common.UiMode(value[0])
		}

	case "ignored-namespaces":
		if len(value) < 1 {
			return fmt.Errorf("%s expects at least one value", property)
		}
		config.IgnoredNamespaces = value

	case "ignored-containers":
		if len(value) < 1 {
			return fmt.Errorf("%s expects at least one value", property)
		}
		config.IgnoredContainers = value

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
