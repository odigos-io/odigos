package cmd

import (
	"bytes"
	"context"
	"encoding/json"

	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/helm"
	clihelm "github.com/odigos-io/odigos/cli/pkg/helm"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	helmcli "helm.sh/helm/v3/pkg/cli"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	updateRemoteFlag        bool
	proNamespaceFlag        string
	useDefault              bool
	downloadFile            string
	fromFile                string
	centralImagePullSecrets []string
	versionFlag             string

	// Helm-like flags for `odigos pro central` (mirrors `odigos install`)
	centralHelmReleaseName          string
	centralHelmChart                string
	centralHelmValuesFile           string
	centralHelmSetArgs              []string
	centralHelmResetThenReuseValues = true

	// Backward-compat flags (previous central installer scripts)
	centralImagePrefixFlag  string
	centralUIModeFlag       string
	centralBackendURLFlag   string
	centralNodeSelectorFlag string

	// Accepted for compatibility; not used by the `odigos-central` chart today.
	centralSkipWait                  bool
	centralTelemetryEnabled          bool
	centralOpenshiftEnabled          bool
	centralSkipWebhookIssuerCreation bool
	centralPSPEnabled                bool
	centralIgnoredNamespaces         []string
	centralIgnoredContainers         []string
	centralInstallProfiles           []string
	centralRuntimeSocketPath         string
	centralK8sNodeLogsDirectory      string
	centralInstrumentorImage         string
	centralOdigletImage              string
	centralAutoScalerImage           string
	centralKarpenterEnabled          bool
)

var proCmd = &cobra.Command{
	Use:   "pro",
	Short: "Manage Odigos onprem tier for enterprise users",
	Long:  `The pro command provides various operations and functionalities specifically designed for enterprise users. Use this command to access advanced features and manage your pro account.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		onPremToken := cmd.Flag("onprem-token").Value.String()

		ns, err := resources.GetOdigosNamespace(client, ctx)
		if resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Println("\033[31mERROR\033[0m no odigos installation found in the current cluster")
			os.Exit(1)
		} else if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already installed: %s\n", err)
			os.Exit(1)
		}

		if updateRemoteFlag {
			err = executeRemoteUpdateToken(ctx, client, ns, onPremToken)
		} else {
			err = pro.UpdateOdigosToken(ctx, client, ns, onPremToken)
		}

		if err != nil {
			fmt.Println("\033[31mERROR\033[0m Failed to update token:")
			fmt.Println(err)
			os.Exit(1)
		} else {
			fmt.Println()
			fmt.Println("\u001B[32mSUCCESS:\u001B[0m Token updated successfully")
		}
	},
	Example: `
# Renew the on-premises token for Odigos,
odigos pro --onprem-token <token>


`,
}

func createTokenPayload(onpremToken string) (string, error) {
	tokenPayload := pro.TokenPayload{OnpremToken: onpremToken}
	jsonBytes, err := json.Marshal(tokenPayload)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func executeRemoteUpdateToken(ctx context.Context, client *kube.Client, namespace string, onPremToken string) error {
	uiSvcProxyEndpoint := fmt.Sprintf(
		"/api/v1/namespaces/%s/services/%s:%d/proxy/api/token/update",
		namespace,
		k8sconsts.OdigosUiServiceName,
		k8sconsts.OdigosUiServicePort,
	)

	tokenPayload, err := createTokenPayload(onPremToken)
	if err != nil {
		return fmt.Errorf("failed to create token payload: %v", err)
	}
	body := bytes.NewBuffer([]byte(tokenPayload))

	request := client.Clientset.RESTClient().Post().
		AbsPath(uiSvcProxyEndpoint).
		Body(body).
		SetHeader("Content-Type", "application/json").
		Do(ctx)

	if err := request.Error(); err != nil {
		return fmt.Errorf("failed to update token: %v", err)
	}

	return nil
}

var offsetsCmd = &cobra.Command{
	Use:   "update-offsets",
	Short: "Update Odiglet to use the latest available Go instrumentation offsets",
	Long: `This command pulls the latest available Go struct and field offsets information from Odigos public server.
Internet access is required to fetch latest offset manifests.
It stores this data in a ConfigMap in the Odigos Namespace and updates the Odiglet DaemonSet to mount it.

Use this command when instrumenting apps that depend on very new dependencies that aren't currently supported
with the installed version of Odigos.

Note that updating offsets does not guarantee instrumentation for libraries with significant changes that
require an update to Odigos. See docs for more info: https://docs.odigos.io/instrumentations/golang/ebpf#about-go-offsets
`,
	Example: `
# Pull the latest offsets and restart Odiglet
odigos pro update-offsets

# Revert to using the default offsets data shipped with Odigos
odigos pro update-offsets --default

# Download the offsets file to a specific location without updating the cluster
odigos pro update-offsets --download-file /path/to/save/offsets.json

# Use a local offsets file instead of downloading it
odigos pro update-offsets --from-file /path/to/local/offsets.json
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)
		ns, err := resources.GetOdigosNamespace(client, ctx)
		if err != nil {
			fmt.Println("Unable to get Odigos namespace")
			os.Exit(1)
		}

		currentTier, err := odigospro.GetCurrentOdigosTier(ctx, client, ns)
		if err != nil {
			fmt.Println("Odigos pro update-offsets failed - unable to read the current Odigos tier.")
			os.Exit(1)
		}
		if currentTier == common.CommunityOdigosTier {
			fmt.Println("Custom Offsets support is only available in Odigos pro tier.")
			os.Exit(1)
		}

		data, err := getLatestOffsets(useDefault)
		if err != nil {
			fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m %+s", err))
			os.Exit(1)
		}

		// If download file is specified, just save the file and exit
		if downloadFile != "" {
			err = os.WriteFile(downloadFile, data, 0644)
			if err != nil {
				fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to write offsets file: %s", err))
				os.Exit(1)
			}
			fmt.Printf("Successfully downloaded offsets to %s\n", downloadFile)
			return
		}

		cm, err := client.Clientset.CoreV1().ConfigMaps(ns).Get(ctx, k8sconsts.GoOffsetsConfigMap, metav1.GetOptions{})
		if err != nil {
			fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to get Go offsets ConfigMap: %s", err))
			os.Exit(1)
		}

		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}

		var escaped []byte
		if len(data) == 0 {
			escaped = []byte{}
		} else {
			escaped, err = json.Marshal(string(data))
			if err != nil {
				fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to encode json string: %s", err))
				os.Exit(1)
			}
		}

		cm.Data[k8sconsts.GoOffsetsFileName] = string(escaped)
		_, err = client.Clientset.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to update Go offsets ConfigMap: %s", err))
			os.Exit(1)
		}

		fmt.Println("Updated Go Offsets, restarting Odiglet to use the new offsets.")
		err = restartOdiglet(ctx, client, ns)
		if err != nil {
			fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to restart Odiglet: %s", err))
			os.Exit(1)
		}
		fmt.Println("Odiglet restarted successfully.")
	},
}

func getLatestOffsets(revert bool) ([]byte, error) {
	if revert {
		return []byte{}, nil
	}

	// If fromFile is specified, read from local file
	if fromFile != "" {
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read offsets file: %s", err)
		}
		return data, nil
	}

	resp, err := http.Get(consts.GoOffsetsPublicURL)
	if err != nil {
		return nil, fmt.Errorf("cannot get latest offsets: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot get latest offsets: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %s", err)
	}
	return data, nil
}

var centralCmd = &cobra.Command{
	Use:   "central",
	Short: "Manage Odigos Central (Enterprise tier)",
	Long:  "Manage Odigos Central backend and UI components used in enterprise deployments.",
}

var (
	centralAdminUser            string
	centralAdminPassword        string
	centralAuthStorageClassName string
	centralMaxMessageSize       string
)

var centralInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Odigos Central backend and UI components",
	RunE:  runCentralInstallOrUpgradeWithLegacyCheck,
	Example: `
# Install Odigos Central
odigos pro central install
`,
}

var centralUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Odigos Central backend and UI components",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCentralHelmUninstall(cmd)
	},
	Example: `
# Uninstall Odigos Central
odigos pro central uninstall
`,
}

var centralUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Odigos Central UI in the central namespace",
	RunE:  runCentralInstallOrUpgradeWithLegacyCheck,
	Example: `
# Upgrade Odigos
odigos pro central upgrade

# Upgrade Odigos with custom values
odigos pro central upgrade --set collectorGateway.maxReplicas=10

# Reset all values to chart defaults (opt out of reuse)
odigos pro central upgrade --reset-then-reuse-values=false
`,
}

func runCentralHelmUninstall(cmd *cobra.Command) error {
	ns := proNamespaceFlag
	if ns == "" {
		ns = consts.DefaultOdigosCentralNamespace
	}
	releaseName := centralHelmReleaseName
	if releaseName == "" {
		releaseName = clihelm.DefaultCentralReleaseName
	}

	fmt.Printf("🗑️  Starting uninstall of release %q from namespace %q...\n", releaseName, ns)

	settings := cli.New()
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), ns, "secret", helm.CustomUninstallLogger); err != nil {
		return err
	}

	res, err := helm.RunUninstall(actionConfig, releaseName)
	if err != nil {
		return err
	}

	if res == nil {
		// Release was not found, already uninstalled
		fmt.Printf("\n🗑️  Release %q not found in namespace %q (already uninstalled)\n", releaseName, ns)
		return nil
	}

	helm.PrintSummary()

	fmt.Printf("\n🗑️  Uninstalled release %q from namespace %q\n", releaseName, ns)
	if res.Info != "" {
		fmt.Printf("Info: %s\n", res.Info)
	}
	return nil
}

func runCentralInstallOrUpgrade() error {

	helm.HelmNamespace = consts.DefaultOdigosCentralNamespace
	helm.HelmReleaseName = clihelm.DefaultCentralReleaseName
	helm.HelmChart = k8sconsts.DefaultCentralHelmChart
	helm.HelmValuesFile = centralHelmValuesFile
	helm.HelmSetArgs = centralHelmSetArgs
	helm.HelmChartVersion = versionFlag
	helm.HelmResetThenReuseValues = centralHelmResetThenReuseValues

	settings := helmcli.New()
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), helm.HelmNamespace, "secret", helm.CustomInstallLogger); err != nil {
		return err
	}

	ch, vals, err := helm.PrepareCentralChartAndValues(settings, "odigos-central")

	if err != nil {
		return err
	}

	result, err := helm.InstallOrUpgrade(actionConfig, ch, vals, helm.HelmNamespace, helm.HelmReleaseName, helm.InstallOrUpgradeOptions{
		CreateNamespace:      true,
		ResetThenReuseValues: helm.HelmResetThenReuseValues,
	})
	if err != nil {
		return err
	}

	helm.PrintSummary()

	fmt.Printf("\n✅ %s\n", helm.FormatInstallOrUpgradeMessage(result, ch.Metadata.Version))
	return nil
}

func runCentralInstallOrUpgradeWithLegacyCheck(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	kubeClient := cmdcontext.KubeClientFromContextOrExit(ctx)

	isLegacy, err := helm.IsLegacyInstallation(ctx, kubeClient.Clientset.CoreV1(), helm.HelmNamespace)
	if err != nil {
		return err
	}
	if isLegacy {
		msg := fmt.Sprintf(`
⚠️  Detected that Odigos was originally installed using an older CLI-based method (without Helm) in namespace "%s".

The current version of the Odigos CLI installs and upgrades Odigos using Helm under the hood,
and cannot automatically upgrade installations created with the legacy method.

👉  To proceed, please do one of the following:
   • Run 'odigos uninstall-deprecated' to remove the old installation, then reinstall using 'odigos install'
   • Or continue using 'odigos upgrade-deprecated' until you are ready to migrate

`, helm.HelmNamespace)

		fmt.Printf("%s\n", msg)
		os.Exit(1)
		return nil
	}

	return runCentralInstallOrUpgrade()
}

var portForwardCentralCmd = &cobra.Command{
	Use:   "ui",
	Short: "Port-forward Odigos Central UI and Backend to localhost",
	Long:  "Port-forward the Central UI (port 3000) and Central Backend (port 8081) to enable local access to Odigos UI. Use --address to bind to specific interfaces.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		var wg sync.WaitGroup
		localAddress := cmd.Flag("address").Value.String()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		// Start resilient port forwarding for backend
		kube.StartResilientPortForward(ctx, kube.ResilientPortForwardConfig{
			WaitGroup:    &wg,
			Client:       client,
			LocalPort:    k8sconsts.CentralBackendPort,
			RemotePort:   k8sconsts.CentralBackendPort,
			LocalAddress: localAddress,
			Namespace:    proNamespaceFlag,
			Name:         "Backend",
			AppLabel:     k8sconsts.CentralBackendAppName,
		})

		// Start resilient port forwarding for UI
		kube.StartResilientPortForward(ctx, kube.ResilientPortForwardConfig{
			WaitGroup:    &wg,
			Client:       client,
			LocalPort:    k8sconsts.CentralUIPort,
			RemotePort:   k8sconsts.CentralUIPort,
			LocalAddress: localAddress,
			Namespace:    proNamespaceFlag,
			Name:         "UI",
			AppLabel:     k8sconsts.CentralUIAppName,
		})

		fmt.Printf("Odigos Central UI is available at: http://%s:%s\n", localAddress, k8sconsts.CentralUIPort)
		fmt.Printf("Odigos Central Backend is available at: http://%s:%s\n", localAddress, k8sconsts.CentralBackendPort)
		fmt.Printf("Press Ctrl+C to stop\n")

		<-sigCh
		fmt.Println("\nReceived interrupt. Stopping port forwarding...")
		cancel()
		wg.Wait()
	},
}

func restartOdiglet(ctx context.Context, client *kube.Client, ns string) error {
	// Create patch to add/update the restartedAt annotation
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`,
		time.Now().Format(time.RFC3339))

	// Patch the Odiglet daemonset
	_, err := client.AppsV1().DaemonSets(ns).Patch(
		ctx,
		k8sconsts.OdigletDaemonSetName,
		types.StrategicMergePatchType,
		[]byte(patch),
		metav1.PatchOptions{},
	)
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("odiglet daemonset not found in namespace %s", ns)
	}
	return err
}

func init() {
	rootCmd.AddCommand(proCmd)

	// proCmd.Flags().String("onprem-token", "", "On-prem token for Odigos")
	// proCmd.MarkFlagRequired("onprem-token")
	// proCmd.PersistentFlags().BoolVarP(&updateRemoteFlag, "remote", "r", false, "use odigos ui service in the cluster to update the onprem token")

	// proCmd.AddCommand(offsetsCmd)
	// offsetsCmd.Flags().BoolVar(&useDefault, "default", false, "revert to using the default offsets data shipped with the current version of Odigos")
	// offsetsCmd.Flags().StringVar(&downloadFile, "download-file", "", "download the offsets file to the specified location without updating the cluster")
	// offsetsCmd.Flags().StringVar(&fromFile, "from-file", "", "use the offsets file from the specified location instead of downloading it")
	proCmd.AddCommand(centralCmd)
	// central subcommands
	centralCmd.AddCommand(centralInstallCmd)
	// centralInstallCmd.Flags().String("onprem-token", "", "On-prem token for Odigos")
	// centralInstallCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "Specify version to install")
	// centralInstallCmd.Flags().StringVarP(&proNamespaceFlag, "namespace", "n", consts.DefaultOdigosCentralNamespace, "Target namespace for Odigos Central installation")
	// centralInstallCmd.Flags().StringSliceVar(&centralImagePullSecrets, "image-pull-secrets", nil, "Secret names for imagePullSecrets (repeat or comma-separated)")

	// register and configure central uninstall command
	centralCmd.AddCommand(centralUninstallCmd)
	centralUninstallCmd.Flags().Bool("yes", false, "Confirm the uninstall without prompting")
	centralUninstallCmd.Flags().StringVarP(&proNamespaceFlag, "namespace", "n", consts.DefaultOdigosCentralNamespace, "Target namespace for Odigos Central uninstallation")

	// register and configure central upgrade command
	centralCmd.AddCommand(centralUpgradeCmd)
	// centralUpgradeCmd.Flags().Bool("yes", false, "Confirm the upgrade without prompting")
	// centralUpgradeCmd.Flags().StringVarP(&proNamespaceFlag, "namespace", "n", consts.DefaultOdigosCentralNamespace, "Target namespace for Odigos Central upgrade")
	// centralUpgradeCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "Specify version to upgrade to")
	// centralUpgradeCmd.MarkFlagRequired("version")
	// centralUpgradeCmd.Flags().StringSliceVar(&centralImagePullSecrets, "image-pull-secrets", nil, "Secret names for imagePullSecrets (repeat or comma-separated)")
	// centralUpgradeCmd.Flags().StringVar(&centralMaxMessageSize, "central-max-message-size", "", "Maximum message size in bytes for gRPC messages (empty = use default)")
	// centralUpgradeCmd.Flags().String("onprem-token", "", "On-prem token for Odigos (required only if Central is not installed yet)")

	// Central configuration flags
	// centralInstallCmd.Flags().StringVar(&centralAdminUser, "central-admin-user", "admin", "Central admin username")
	// centralInstallCmd.Flags().StringVar(&centralAdminPassword, "central-admin-password", "", "Central admin password (auto-generated if not provided)")
	// centralInstallCmd.Flags().StringVar(&centralAuthStorageClassName, "central-storage-class-name", "", "StorageClassName for Keycloak PVC (omit to use cluster default; set '' to disable)")
	// centralInstallCmd.Flags().StringVar(&centralMaxMessageSize, "central-max-message-size", "", "Maximum message size in bytes for gRPC messages (empty = use default)")

	// Helm-style flags for `odigos pro central` (same shape as `odigos install`)
	for _, c := range []*cobra.Command{centralInstallCmd, centralUpgradeCmd} {
		c.Flags().StringVar(&centralHelmReleaseName, "release-name", clihelm.DefaultCentralReleaseName, "Helm release name")
		c.Flags().StringVar(&centralHelmChart, "chart", k8sconsts.DefaultCentralHelmChart, "Helm chart to install (repo/name, local path, or URL)")
		c.Flags().StringVarP(&centralHelmValuesFile, "values", "f", "", "Path to a custom values.yaml file")
		c.Flags().StringSliceVar(&centralHelmSetArgs, "set", []string{}, "Set values on the command line (key=value)")
		c.Flags().StringVarP(&proNamespaceFlag, "namespace", "n", consts.DefaultOdigosCentralNamespace, "Target namespace for Odigos Central installation")
		c.Flags().BoolVar(
			&centralHelmResetThenReuseValues,
			"reset-then-reuse-values",
			true,
			"Reset to chart defaults, then reuse values from the previous release (default: true).",
		)
		c.Flags().StringVar(&versionFlag, "version", OdigosVersion, "Specify version to upgrade to")
	}

	// Backward-compat flags (mapped where possible; otherwise ignored with a warning).
	// for _, c := range []*cobra.Command{centralInstallCmd, centralUpgradeCmd} {
	// 	c.Flags().StringVar(&centralImagePrefixFlag, "image-prefix", "", "Image registry/prefix override for Odigos Central images")
	// 	c.Flags().StringVar(&centralUIModeFlag, "ui-mode", "", "Central UI mode (maps to centralUI.uiMode)")
	// 	c.Flags().StringVar(&centralBackendURLFlag, "central-backend-url", "", "Override Central backend URL (maps to centralUI.centralBackendURL)")
	// 	c.Flags().StringVar(&centralNodeSelectorFlag, "node-selector", "", "Node selector for central components (key=value[,key=value...])")

	// 	c.Flags().BoolVar(&centralSkipWait, "skip-wait", false, "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().BoolVar(&centralTelemetryEnabled, "telemetry-enabled", false, "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().BoolVar(&centralOpenshiftEnabled, "openshift-enabled", false, "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().BoolVar(&centralSkipWebhookIssuerCreation, "skip-webhook-issuer-creation", false, "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().BoolVar(&centralPSPEnabled, "psp", false, "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().StringSliceVar(&centralIgnoredNamespaces, "ignored-namespaces", nil, "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().StringSliceVar(&centralIgnoredContainers, "ignored-containers", nil, "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().StringSliceVar(&centralInstallProfiles, "profiles", nil, "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().StringVar(&centralRuntimeSocketPath, "custom-container-runtime-socket-path", "", "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().StringVar(&centralK8sNodeLogsDirectory, "k8s-node-logs-directory", "", "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().StringVar(&centralInstrumentorImage, "instrumentor-image", "", "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().StringVar(&centralOdigletImage, "odiglet-image", "", "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().StringVar(&centralAutoScalerImage, "autoscaler-image", "", "Compatibility flag (ignored for Helm-based Central)")
	// 	c.Flags().BoolVar(&centralKarpenterEnabled, "karpenter-enabled", false, "Compatibility flag (ignored for Helm-based Central)")
	// }

	centralCmd.AddCommand(portForwardCentralCmd)
	portForwardCentralCmd.Flags().String("address", "localhost", "Address to serve the UI on")
}

func createKubeResourceWithLogging(ctx context.Context, msg string, client *kube.Client, ns string, labelScope string, create ResourceCreationFunc) {
	l := log.Print(msg)
	err := create(ctx, client, ns, labelScope)
	if err != nil {
		l.Error(err)
	}

	l.Success()
}

type ResourceCreationFunc func(ctx context.Context, client *kube.Client, ns string, labelKey string) error
