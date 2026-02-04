package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/centralodigos"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/cmdutil"
	"github.com/odigos-io/odigos/cli/pkg/confirm"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	versionFlagDep      string
	openshiftEnabledDep bool
)

var (
	updateRemoteFlagDep        bool
	proNamespaceFlagDep        string
	centralImagePullSecretsDep []string
)

var centralVersionRegexDep = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:-(?:pre|rc)\d+)?$`)

var proCmdDep = &cobra.Command{
	Use:   "pro-dep",
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

		if updateRemoteFlagDep {
			err = kube.ExecuteRemoteUpdateToken(ctx, client, ns, onPremToken)
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

var centralCmdDep = &cobra.Command{
	Use:   "central",
	Short: "Manage Odigos Central (Enterprise tier)",
	Long:  "Manage Odigos Central backend and UI components used in enterprise deployments.",
}

var (
	centralAdminUserDep            string
	centralAdminPasswordDep        string
	centralAuthStorageClassNameDep string
	centralMaxMessageSizeDep       string
)

var centralInstallCmdDep = &cobra.Command{
	Use:   "install",
	Short: "Install Odigos Central backend and UI components",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		onPremToken := cmd.Flag("onprem-token").Value.String()
		var storageClassNamePtr *string
		if cmd.Flags().Changed("central-storage-class-name") {
			storageClassNamePtr = &centralAuthStorageClassNameDep
		}
		if err := installCentralBackendAndUIDep(ctx, client, proNamespaceFlagDep, onPremToken, storageClassNamePtr); err != nil {
			fmt.Println("\033[31mERROR\033[0m Failed to install Odigos Central:")
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

var centralUninstallCmdDep = &cobra.Command{
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

		cmdutil.CreateKubeResourceWithLogging(ctx, "Uninstalling Odigos Central Deployments",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteDeploymentsByLabel)
		cmdutil.CreateKubeResourceWithLogging(ctx, "Uninstalling Odigos Central Services",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteServicesByLabel)
		cmdutil.CreateKubeResourceWithLogging(ctx, "Uninstalling Odigos Central Roles",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteRolesByLabel)
		cmdutil.CreateKubeResourceWithLogging(ctx, "Uninstalling Odigos Central RoleBindings",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteRoleBindingsByLabel)
		cmdutil.CreateKubeResourceWithLogging(ctx, "Uninstalling Odigos Central ClusterRoles",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, deleteClusterRolesByLabelAdapterDep)
		cmdutil.CreateKubeResourceWithLogging(ctx, "Uninstalling Odigos Central ClusterRoleBindings",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, deleteClusterRoleBindingsByLabelAdapterDep)
		cmdutil.CreateKubeResourceWithLogging(ctx, "Uninstalling Odigos Central ServiceAccounts",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteServiceAccountsByLabel)
		cmdutil.CreateKubeResourceWithLogging(ctx, "Uninstalling Odigos Central Secrets",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteSecretsByLabel)
		cmdutil.CreateKubeResourceWithLogging(ctx, "Uninstalling Odigos Central ConfigMaps",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteConfigMapsByLabel)
		cmdutil.CreateKubeResourceWithLogging(ctx, "Uninstalling Odigos Central HPAs",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, kube.DeleteHPAsByLabel)

		cmdutil.CreateKubeResourceWithLogging(ctx, "Deleting Odigos Central token secret",
			client, ns, k8sconsts.OdigosSystemLabelCentralKey, deleteCentralTokenSecretAdapterDep)

		hasCentralLabel, err := kube.NamespaceHasLabel(ctx, client, ns, k8sconsts.OdigosSystemLabelCentralKey)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if namespace %s has Odigos Central label: %s\n", ns, err)
			os.Exit(1)
		}
		if hasCentralLabel {
			cmdutil.CreateKubeResourceWithLogging(ctx, fmt.Sprintf("Uninstalling Namespace %s", ns),
				client, ns, k8sconsts.OdigosSystemLabelCentralKey, uninstallNamespace)
			waitForNamespaceDeletion(ctx, client, ns)
		}

		fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos Central uninstalled.\n")
	},
}

var centralUpgradeCmdDep = &cobra.Command{
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
			fmt.Printf("About to upgrade Odigos Central UI in namespace %s to version %s\n", ns, versionFlagDep)
			confirmed, err := confirm.Ask("Are you sure?")
			if err != nil || !confirmed {
				fmt.Println("Aborting upgrade")
				return
			}
		}

		if !centralVersionRegexDep.MatchString(versionFlagDep) {
			fmt.Printf("\033[31mERROR\033[0m Invalid --version value %q. Expected formats: vX.Y.Z, vX.Y.Z-preN, or vX.Y.Z-rcN\n", versionFlagDep)
			os.Exit(1)
		}

		var imagePullSecrets []string
		if len(centralImagePullSecretsDep) > 0 {
			imagePullSecrets = append(imagePullSecrets, centralImagePullSecretsDep...)
		}

		managerOpts := resourcemanager.ManagerOpts{
			ImageReferences:      GetImageReferencesDep(common.OnPremOdigosTier, openshiftEnabledDep),
			SystemObjectLabelKey: k8sconsts.OdigosSystemLabelCentralKey,
			ImagePullSecrets:     imagePullSecrets,
		}

		uiManager := centralodigos.NewCentralUIResourceManager(client, ns, managerOpts, versionFlagDep)
		backendConfig := centralodigos.CentralBackendConfig{
			MaxMessageSize: centralMaxMessageSizeDep,
		}
		backendManager := centralodigos.NewCentralBackendResourceManager(client, ns, versionFlagDep, managerOpts, backendConfig)
		if err := resources.ApplyResourceManagers(ctx, client, []resourcemanager.ResourceManager{uiManager, backendManager}, "Upgrading"); err != nil {
			fmt.Println("\033[31mERROR\033[0m Failed to upgrade Odigos Central UI/Backend:")
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos Central UI and Backend upgraded to %s.\n", versionFlagDep)
	},
}

func createOdigosCentralSecretDep(ctx context.Context, client *kube.Client, ns, token string) error {
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

func installCentralBackendAndUIDep(ctx context.Context, client *kube.Client, ns string, onPremToken string, storageClassNamePtr *string) error {

	_, err := client.AppsV1().Deployments(ns).Get(ctx, k8sconsts.CentralBackendName, metav1.GetOptions{})
	if err == nil {
		fmt.Printf("\n\u001B[33mINFO:\u001B[0m Odigos Central is already installed in namespace %s\n", ns)
		return nil
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check existing central backend: %w", err)
	}

	fmt.Println("Installing Odigos Central backend and UI ...")

	var imagePullSecrets []string
	if len(centralImagePullSecretsDep) > 0 {
		imagePullSecrets = append(imagePullSecrets, centralImagePullSecretsDep...)
	}

	managerOpts := resourcemanager.ManagerOpts{
		ImageReferences:      GetImageReferencesDep(common.OnPremOdigosTier, openshiftEnabledDep),
		SystemObjectLabelKey: k8sconsts.OdigosSystemLabelCentralKey,
		ImagePullSecrets:     imagePullSecrets,
	}

	cmdutil.CreateKubeResourceWithLogging(ctx, fmt.Sprintf("> Creating namespace %s", ns), client, ns, k8sconsts.OdigosSystemLabelCentralKey, createNamespaceDep)

	if err := createOdigosCentralSecretDep(ctx, client, ns, onPremToken); err != nil {
		return err
	}
	config := resources.CentralManagersConfig{
		Auth: centralodigos.AuthConfig{
			AdminUsername:    centralAdminUserDep,
			AdminPassword:    centralAdminPasswordDep,
			StorageClassName: storageClassNamePtr,
		},
		CentralBackend: centralodigos.CentralBackendConfig{
			MaxMessageSize: centralMaxMessageSizeDep,
		},
	}
	resourceManagers := resources.CreateCentralizedManagers(client, managerOpts, ns, versionFlagDep, config)
	if err := resources.ApplyResourceManagers(ctx, client, resourceManagers, "Creating"); err != nil {
		return fmt.Errorf("failed to install Odigos Central: %w", err)
	}

	fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos Central installed.\n")
	return nil
}

func deleteCentralTokenSecretAdapterDep(ctx context.Context, client *kube.Client, ns string, _ string) error {
	return kube.DeleteCentralTokenSecret(ctx, client, ns)
}

func deleteClusterRolesByLabelAdapterDep(ctx context.Context, client *kube.Client, _ string, labelKey string) error {
	return kube.DeleteClusterRolesByLabel(ctx, client, labelKey)
}

func deleteClusterRoleBindingsByLabelAdapterDep(ctx context.Context, client *kube.Client, _ string, labelKey string) error {
	return kube.DeleteClusterRoleBindingsByLabel(ctx, client, labelKey)
}

var portForwardCentralCmdDep = &cobra.Command{
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
			Namespace:    proNamespaceFlagDep,
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
			Namespace:    proNamespaceFlagDep,
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

func init() {
	// Deprecated/legacy flow entrypoint.
	// This is intentionally separate from `odigos pro` to avoid collisions with the Helm-based flow.
	rootCmd.AddCommand(proCmdDep)

	proCmdDep.Flags().String("onprem-token", "", "On-prem token for Odigos")
	proCmdDep.MarkFlagRequired("onprem-token")
	proCmdDep.PersistentFlags().BoolVarP(&updateRemoteFlagDep, "remote", "r", false, "use odigos ui service in the cluster to update the onprem token")

	proCmdDep.AddCommand(centralCmdDep)

	// central subcommands (dep variants)
	centralCmdDep.AddCommand(centralInstallCmdDep)
	centralInstallCmdDep.Flags().String("onprem-token", "", "On-prem token for Odigos")
	centralInstallCmdDep.Flags().StringVar(&versionFlagDep, "version", OdigosVersion, "Specify version to install")
	centralInstallCmdDep.MarkFlagRequired("onprem-token")
	centralInstallCmdDep.Flags().StringVarP(&proNamespaceFlagDep, "namespace", "n", consts.DefaultOdigosCentralNamespace, "Target namespace for Odigos Central installation")
	centralInstallCmdDep.Flags().StringSliceVar(&centralImagePullSecretsDep, "image-pull-secrets", nil, "Secret names for imagePullSecrets (repeat or comma-separated)")

	// register and configure central uninstall command
	centralCmdDep.AddCommand(centralUninstallCmdDep)
	centralUninstallCmdDep.Flags().Bool("yes", false, "Confirm the uninstall without prompting")
	centralUninstallCmdDep.Flags().StringVarP(&proNamespaceFlagDep, "namespace", "n", consts.DefaultOdigosCentralNamespace, "Target namespace for Odigos Central uninstallation")

	// register and configure central upgrade command
	centralCmdDep.AddCommand(centralUpgradeCmdDep)
	centralUpgradeCmdDep.Flags().Bool("yes", false, "Confirm the upgrade without prompting")
	centralUpgradeCmdDep.Flags().StringVarP(&proNamespaceFlagDep, "namespace", "n", consts.DefaultOdigosCentralNamespace, "Target namespace for Odigos Central upgrade")
	centralUpgradeCmdDep.Flags().StringVar(&versionFlagDep, "version", OdigosVersion, "Specify version to upgrade to")
	centralUpgradeCmdDep.MarkFlagRequired("version")
	centralUpgradeCmdDep.Flags().StringSliceVar(&centralImagePullSecretsDep, "image-pull-secrets", nil, "Secret names for imagePullSecrets (repeat or comma-separated)")
	centralUpgradeCmdDep.Flags().StringVar(&centralMaxMessageSizeDep, "central-max-message-size", "", "Maximum message size in bytes for gRPC messages (empty = use default)")

	// Central configuration flags
	centralInstallCmdDep.Flags().StringVar(&centralAdminUserDep, "central-admin-user", "admin", "Central admin username")
	centralInstallCmdDep.Flags().StringVar(&centralAdminPasswordDep, "central-admin-password", "", "Central admin password (auto-generated if not provided)")
	centralInstallCmdDep.Flags().StringVar(&centralAuthStorageClassNameDep, "central-storage-class-name", "", "StorageClassName for Keycloak PVC (omit to use cluster default; set '' to disable)")
	centralInstallCmdDep.Flags().StringVar(&centralMaxMessageSizeDep, "central-max-message-size", "", "Maximum message size in bytes for gRPC messages (empty = use default)")

	centralCmdDep.AddCommand(portForwardCentralCmdDep)
	portForwardCentralCmdDep.Flags().String("address", "localhost", "Address to serve the UI on")
}

func GetImageReferencesDep(odigosTier common.OdigosTier, openshift bool) resourcemanager.ImageReferences {
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

func createNamespaceDep(ctx context.Context, client *kube.Client, ns string, labelKey string) error {
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
