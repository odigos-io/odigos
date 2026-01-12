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
	"github.com/odigos-io/odigos/cli/cmd/resources/centralodigos"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/confirm"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"

	"github.com/spf13/cobra"
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
		if err := installCentralBackendAndUI(ctx, client, proNamespaceFlag, onPremToken, storageClassNamePtr); err != nil {
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

		fmt.Println("Starting Odigos Central uninstallation...")

		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Central Deployments",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteDeploymentsByLabel)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Central Services",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteServicesByLabel)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Central Roles",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteRolesByLabel)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Central RoleBindings",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteRoleBindingsByLabel)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Central ClusterRoles",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, deleteClusterRolesByLabelAdapter)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Central ClusterRoleBindings",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, deleteClusterRoleBindingsByLabelAdapter)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Central ServiceAccounts",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteServiceAccountsByLabel)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Central Secrets",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteSecretsByLabel)

		createKubeResourceWithLogging(ctx, "Deleting Odigos Central token secret",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, deleteCentralTokenSecretAdapter)

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

		var imagePullSecrets []string
		if len(centralImagePullSecrets) > 0 {
			imagePullSecrets = append(imagePullSecrets, centralImagePullSecrets...)
		}

		managerOpts := resourcemanager.ManagerOpts{
			ImageReferences:      GetImageReferences(common.OnPremOdigosTier, openshiftEnabled),
			SystemObjectLabelKey: k8sconsts.OdigosSystemLabelCentralKey,
			ImagePullSecrets:     imagePullSecrets,
		}

		uiManager := centralodigos.NewCentralUIResourceManager(client, ns, managerOpts, versionFlag)
		backendConfig := centralodigos.CentralBackendConfig{
			MaxMessageSize: centralMaxMessageSize,
		}
		backendManager := centralodigos.NewCentralBackendResourceManager(client, ns, versionFlag, managerOpts, backendConfig)
		if err := resources.ApplyResourceManagers(ctx, client, []resourcemanager.ResourceManager{uiManager, backendManager}, "Upgrading"); err != nil {
			fmt.Println("\033[31mERROR\033[0m Failed to upgrade Odigos Central UI/Backend:")
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos Central UI and Backend upgraded to %s.\n", versionFlag)
	},
}

func createOdigosCentralSecret(ctx context.Context, client *kube.Client, ns, token string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosCentralSecretName,
			Namespace: ns,
		},
		StringData: map[string]string{
			k8sconsts.OdigosOnpremTokenSecretKey: token,
		},
	}
	_, err := client.CoreV1().Secrets(ns).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create odigos-central secret: %w", err)
	}
	return nil
}

func installCentralBackendAndUI(ctx context.Context, client *kube.Client, ns string, onPremToken string, storageClassNamePtr *string) error {

	_, err := client.AppsV1().Deployments(ns).Get(ctx, k8sconsts.CentralBackendName, metav1.GetOptions{})
	if err == nil {
		fmt.Printf("\n\u001B[33mINFO:\u001B[0m Odigos Central is already installed in namespace %s\n", ns)
		return nil
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check existing central backend: %w", err)
	}

	fmt.Println("Installing Odigos Central backend and UI ...")

	var imagePullSecrets []string
	if len(centralImagePullSecrets) > 0 {
		imagePullSecrets = append(imagePullSecrets, centralImagePullSecrets...)
	}

	managerOpts := resourcemanager.ManagerOpts{
		ImageReferences:      GetImageReferences(common.OnPremOdigosTier, openshiftEnabled),
		SystemObjectLabelKey: k8sconsts.OdigosSystemLabelCentralKey,
		ImagePullSecrets:     imagePullSecrets,
	}

	createKubeResourceWithLogging(ctx, fmt.Sprintf("> Creating namespace %s", ns), client, ns, k8sconsts.OdigosSystemLabelCentralKey, createNamespace)

	if err := createOdigosCentralSecret(ctx, client, ns, onPremToken); err != nil {
		return err
	}
	config := resources.CentralManagersConfig{
		Auth: centralodigos.AuthConfig{
			AdminUsername:    centralAdminUser,
			AdminPassword:    centralAdminPassword,
			StorageClassName: storageClassNamePtr,
		},
		CentralBackend: centralodigos.CentralBackendConfig{
			MaxMessageSize: centralMaxMessageSize,
		},
	}
	resourceManagers := resources.CreateCentralizedManagers(client, managerOpts, ns, versionFlag, config)
	if err := resources.ApplyResourceManagers(ctx, client, resourceManagers, "Creating"); err != nil {
		return fmt.Errorf("failed to install Odigos Central: %w", err)
	}

	fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos Central installed.\n")
	return nil
}

func deleteCentralTokenSecretAdapter(ctx context.Context, client *kube.Client, ns string, _ string) error {
	return kube.DeleteCentralTokenSecret(ctx, client, ns)
}

func deleteClusterRolesByLabelAdapter(ctx context.Context, client *kube.Client, _ string, labelKey string) error {
	return kube.DeleteClusterRolesByLabel(ctx, client, labelKey)
}

func deleteClusterRoleBindingsByLabelAdapter(ctx context.Context, client *kube.Client, _ string, labelKey string) error {
	return kube.DeleteClusterRoleBindingsByLabel(ctx, client, labelKey)
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

	// Central configuration flags
	centralInstallCmd.Flags().StringVar(&centralAdminUser, "central-admin-user", "admin", "Central admin username")
	centralInstallCmd.Flags().StringVar(&centralAdminPassword, "central-admin-password", "", "Central admin password (auto-generated if not provided)")
	centralInstallCmd.Flags().StringVar(&centralAuthStorageClassName, "central-storage-class-name", "", "StorageClassName for Keycloak PVC (omit to use cluster default; set '' to disable)")
	centralInstallCmd.Flags().StringVar(&centralMaxMessageSize, "central-max-message-size", "", "Maximum message size in bytes for gRPC messages (empty = use default)")
	centralCmd.AddCommand(portForwardCentralCmd)
	portForwardCentralCmd.Flags().String("address", "localhost", "Address to serve the UI on")
}
