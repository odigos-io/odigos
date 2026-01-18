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
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/confirm"
	clihelm "github.com/odigos-io/odigos/cli/pkg/helm"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"

	"github.com/spf13/cobra"
	helmaction "helm.sh/helm/v3/pkg/action"
	helmcli "helm.sh/helm/v3/pkg/cli"
	corev1 "k8s.io/api/core/v1"
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
)

var centralVersionRegex = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:-(?:pre|rc)\d+)?$`)

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
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		onPremToken := cmd.Flag("onprem-token").Value.String()
		var storageClassNamePtr *string
		if cmd.Flags().Changed("central-storage-class-name") {
			storageClassNamePtr = &centralAuthStorageClassName
		}
		if err := installOrUpgradeCentralWithHelm(ctx, client, proNamespaceFlag, versionFlag, onPremToken, storageClassNamePtr); err != nil {
			fmt.Println("\033[31mERROR\033[0m Failed to install Odigos Central:")
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

var centralUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Odigos Central backend and UI components",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to read namespace flag: %s\n", err)
			os.Exit(1)
		}

		if !cmd.Flag("yes").Changed {
			fmt.Printf("About to uninstall Odigos Central from namespace %s\n", ns)
			confirmed, err := confirm.Ask("Are you sure?")
			if err != nil || !confirmed {
				fmt.Println("Aborting uninstall")
				return
			}
		}

		fmt.Println("Starting Odigos Central uninstallation (Helm)...")
		if err := uninstallCentralWithHelm(ns); err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to uninstall Odigos Central with Helm: %s\n", err)
			os.Exit(1)
		}

		hasCentralLabel, err := kube.NamespaceHasLabel(ctx, client, ns, k8sconsts.OdigosSystemLabelCentralKey)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if namespace %s has Odigos Central label: %s\n", ns, err)
			os.Exit(1)
		}
		if hasCentralLabel {
			createKubeResourceWithLogging(ctx, fmt.Sprintf("Uninstalling Namespace %s", ns),
				client, ns, k8sconsts.OdigosSystemLabelCentralKey, uninstallNamespace)
			waitForNamespaceDeletion(ctx, client, ns)
		}

		fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos Central uninstalled.\n")
	},
}

var centralUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Odigos Central UI in the central namespace",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to read namespace flag: %s\n", err)
			os.Exit(1)
		}

		if !cmd.Flag("yes").Changed {
			fmt.Printf("About to upgrade Odigos Central UI in namespace %s to version %s\n", ns, versionFlag)
			confirmed, err := confirm.Ask("Are you sure?")
			if err != nil || !confirmed {
				fmt.Println("Aborting upgrade")
				return
			}
		}

		if !centralVersionRegex.MatchString(versionFlag) {
			fmt.Printf("\033[31mERROR\033[0m Invalid --version value %q. Expected formats: vX.Y.Z, vX.Y.Z-preN, or vX.Y.Z-rcN\n", versionFlag)
			os.Exit(1)
		}

		// Helm upgrade behaves like install+upgrade (same semantics as `odigos install`).
		// If the Helm release does not exist yet, user must provide --onprem-token to perform a fresh install.
		onPremToken, _ := cmd.Flags().GetString("onprem-token")
		var storageClassNamePtr *string
		if cmd.Flags().Changed("central-storage-class-name") {
			storageClassNamePtr = &centralAuthStorageClassName
		}
		if err := installOrUpgradeCentralWithHelm(ctx, client, ns, versionFlag, onPremToken, storageClassNamePtr); err != nil {
			fmt.Println("\033[31mERROR\033[0m Failed to upgrade Odigos Central with Helm:")
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos Central upgraded to %s.\n", versionFlag)
	},
}

func installOrUpgradeCentralWithHelm(ctx context.Context, client *kube.Client, ns string, version string, onPremToken string, storageClassNamePtr *string) error {
	// Keep existing behavior: create/label namespace before install so we can delete it on uninstall.
	createKubeResourceWithLogging(ctx, fmt.Sprintf("> Ensuring namespace %s", ns), client, ns, k8sconsts.OdigosSystemLabelCentralKey, createNamespace)

	settings := helmcli.New()
	settings.KubeConfig = kubeConfig
	settings.KubeContext = kubeContext

	actionConfig := new(helmaction.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), ns, "secret", clihelm.CustomInstallLogger); err != nil {
		return err
	}

	// If the release does not exist, we need a token (or an existing secret) to render the chart
	// because the chart gates most resources on the token secret.
	exists, err := clihelm.ReleaseExists(actionConfig, clihelm.DefaultCentralReleaseName)
	if err != nil {
		return err
	}
	if !exists && onPremToken == "" {
		// Allow a first Helm install if the token secret already exists (chart will reuse it via lookup).
		_, secErr := client.CoreV1().Secrets(ns).Get(ctx, "odigos-central", metav1.GetOptions{})
		if secErr != nil && apierrors.IsNotFound(secErr) {
			return fmt.Errorf("odigos central is not installed in namespace %q; provide --onprem-token to perform a fresh install", ns)
		}
		if secErr != nil && !apierrors.IsNotFound(secErr) {
			return secErr
		}
	}

	ch, valuesMap, err := clihelm.PrepareCentralChartAndValues(settings, version, clihelm.CentralValues{
		OnPremToken:          onPremToken,
		AdminUsername:        centralAdminUser,
		AdminPassword:        centralAdminPassword,
		KeycloakStorageClass: storageClassNamePtr,
		MaxMessageSize:       centralMaxMessageSize,
		ImageTag:             version,
		ImagePullSecrets:     centralImagePullSecrets,
	})
	if err != nil {
		return err
	}

	result, err := clihelm.InstallOrUpgrade(actionConfig, ch, valuesMap, ns, clihelm.DefaultCentralReleaseName, clihelm.InstallOrUpgradeOptions{
		CreateNamespace:      true,
		ResetThenReuseValues: true,
	})
	if err != nil {
		return err
	}

	clihelm.PrintSummary()
	fmt.Printf("\n✅ %s\n", clihelm.FormatInstallOrUpgradeMessage(result, ch.Metadata.Version))
	return nil
}

func uninstallCentralWithHelm(ns string) error {
	settings := helmcli.New()
	// Honor CLI flags like --kubeconfig/--kube-context.
	settings.KubeConfig = kubeConfig
	settings.KubeContext = kubeContext

	actionConfig := new(helmaction.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), ns, "secret", clihelm.CustomUninstallLogger); err != nil {
		return err
	}

	_, err := clihelm.RunUninstall(actionConfig, clihelm.DefaultCentralReleaseName)
	if err != nil {
		return err
	}

	clihelm.PrintSummary()
	return nil
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

	proCmd.Flags().String("onprem-token", "", "On-prem token for Odigos")
	proCmd.MarkFlagRequired("onprem-token")
	proCmd.PersistentFlags().BoolVarP(&updateRemoteFlag, "remote", "r", false, "use odigos ui service in the cluster to update the onprem token")

	proCmd.AddCommand(offsetsCmd)
	offsetsCmd.Flags().BoolVar(&useDefault, "default", false, "revert to using the default offsets data shipped with the current version of Odigos")
	offsetsCmd.Flags().StringVar(&downloadFile, "download-file", "", "download the offsets file to the specified location without updating the cluster")
	offsetsCmd.Flags().StringVar(&fromFile, "from-file", "", "use the offsets file from the specified location instead of downloading it")
	proCmd.AddCommand(centralCmd)
	// central subcommands
	centralCmd.AddCommand(centralInstallCmd)
	centralInstallCmd.Flags().String("onprem-token", "", "On-prem token for Odigos")
	centralInstallCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "Specify version to install")
	centralInstallCmd.MarkFlagRequired("onprem-token")
	centralInstallCmd.Flags().StringVarP(&proNamespaceFlag, "namespace", "n", consts.DefaultOdigosCentralNamespace, "Target namespace for Odigos Central installation")
	centralInstallCmd.Flags().StringSliceVar(&centralImagePullSecrets, "image-pull-secrets", nil, "Secret names for imagePullSecrets (repeat or comma-separated)")

	// register and configure central uninstall command
	centralCmd.AddCommand(centralUninstallCmd)
	centralUninstallCmd.Flags().Bool("yes", false, "Confirm the uninstall without prompting")
	centralUninstallCmd.Flags().StringVarP(&proNamespaceFlag, "namespace", "n", consts.DefaultOdigosCentralNamespace, "Target namespace for Odigos Central uninstallation")

	// register and configure central upgrade command
	centralCmd.AddCommand(centralUpgradeCmd)
	centralUpgradeCmd.Flags().Bool("yes", false, "Confirm the upgrade without prompting")
	centralUpgradeCmd.Flags().StringVarP(&proNamespaceFlag, "namespace", "n", consts.DefaultOdigosCentralNamespace, "Target namespace for Odigos Central upgrade")
	centralUpgradeCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "Specify version to upgrade to")
	centralUpgradeCmd.MarkFlagRequired("version")
	centralUpgradeCmd.Flags().StringSliceVar(&centralImagePullSecrets, "image-pull-secrets", nil, "Secret names for imagePullSecrets (repeat or comma-separated)")
	centralUpgradeCmd.Flags().StringVar(&centralMaxMessageSize, "central-max-message-size", "", "Maximum message size in bytes for gRPC messages (empty = use default)")
	centralUpgradeCmd.Flags().String("onprem-token", "", "On-prem token for Odigos (required only if Central is not installed yet)")

	// Central configuration flags
	centralInstallCmd.Flags().StringVar(&centralAdminUser, "central-admin-user", "admin", "Central admin username")
	centralInstallCmd.Flags().StringVar(&centralAdminPassword, "central-admin-password", "", "Central admin password (auto-generated if not provided)")
	centralInstallCmd.Flags().StringVar(&centralAuthStorageClassName, "central-storage-class-name", "", "StorageClassName for Keycloak PVC (omit to use cluster default; set '' to disable)")
	centralInstallCmd.Flags().StringVar(&centralMaxMessageSize, "central-max-message-size", "", "Maximum message size in bytes for gRPC messages (empty = use default)")
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

func GetImageReferences(odigosTier common.OdigosTier, openshift bool) resourcemanager.ImageReferences {
	var imageReferences resourcemanager.ImageReferences
	if openshift {
		imageReferences = resourcemanager.ImageReferences{
			AutoscalerImage:    k8sconsts.AutoScalerImageCertified,
			CollectorImage:     k8sconsts.OdigosClusterCollectorImageCertified,
			InitContainerImage: k8sconsts.OdigosInitContainerImageCertified,
			InstrumentorImage:  k8sconsts.InstrumentorImageCertified,
			OdigletImage:       k8sconsts.OdigletImageCertified,
			KeyvalProxyImage:   k8sconsts.KeyvalProxyImage,
			SchedulerImage:     k8sconsts.SchedulerImageCertified,
			UIImage:            k8sconsts.UIImageCertified,
		}
	} else {
		imageReferences = resourcemanager.ImageReferences{
			AutoscalerImage:    k8sconsts.AutoScalerImageName,
			CollectorImage:     k8sconsts.OdigosClusterCollectorImage,
			InitContainerImage: k8sconsts.OdigosInitContainerImage,
			InstrumentorImage:  k8sconsts.InstrumentorImage,
			OdigletImage:       k8sconsts.OdigletImageName,
			KeyvalProxyImage:   k8sconsts.KeyvalProxyImage,
			SchedulerImage:     k8sconsts.SchedulerImage,
			UIImage:            k8sconsts.UIImage,
		}
	}

	if odigosTier == common.OnPremOdigosTier {
		if openshift {
			imageReferences.InstrumentorImage = k8sconsts.InstrumentorEnterpriseImageCertified
			imageReferences.OdigletImage = k8sconsts.OdigletEnterpriseImageCertified
			imageReferences.InitContainerImage = k8sconsts.OdigosInitContainerEnterpriseImageCertified
		} else {
			imageReferences.InstrumentorImage = k8sconsts.InstrumentorEnterpriseImage
			imageReferences.OdigletImage = k8sconsts.OdigletEnterpriseImageName
			imageReferences.CentralProxyImage = k8sconsts.CentralProxyImage
			imageReferences.CentralBackendImage = k8sconsts.CentralBackendImage
			imageReferences.CentralUIImage = k8sconsts.CentralUIImage
			imageReferences.InitContainerImage = k8sconsts.OdigosInitContainerEnterpriseImage
		}
	}
	return imageReferences
}

type ResourceCreationFunc func(ctx context.Context, client *kube.Client, ns string, labelKey string) error

func createNamespace(ctx context.Context, client *kube.Client, ns string, labelKey string) error {
	_, err := client.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err == nil {
		// Namespace already exists, nothing to do
		return nil
	}

	if apierrors.IsNotFound(err) {
		nsObj := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
				Labels: map[string]string{
					labelKey: k8sconsts.OdigosSystemLabelValue,
				},
			},
		}
		_, err := client.CoreV1().Namespaces().Create(ctx, nsObj, metav1.CreateOptions{})
		return err
	}

	return err
}
