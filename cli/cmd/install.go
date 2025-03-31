package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/odigos-io/odigos/cli/pkg/autodetect"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/installationmethod"
	"github.com/odigos-io/odigos/profiles"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/odigos-io/odigos/common"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	odigosCloudApiKeyFlag      string
	odigosOnPremToken          string
	namespaceFlag              string
	versionFlag                string
	skipWait                   bool
	telemetryEnabled           bool
	openshiftEnabled           bool
	skipWebhookIssuerCreation  bool
	psp                        bool
	userInputIgnoredNamespaces []string
	userInputIgnoredContainers []string
	userInputInstallProfiles   []string
	uiMode                     string

	instrumentorImage string
	odigletImage      string
	autoScalerImage   string
	imagePrefix       string

	installCentralized bool
	clusterName        string
	centralBackendURL  string
)

type ResourceCreationFunc func(ctx context.Context, client *kube.Client, ns string) error

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Odigos in your kubernetes cluster.",
	Long: `This sub command will Install Odigos in your kubernetes cluster.
It will install k8s components that will auto-instrument your applications with OpenTelemetry and send traces, metrics and logs to any telemetry backend`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)
		ns := cmd.Flag("namespace").Value.String()

		installed, err := isOdigosInstalled(ctx, client, ns)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check Odigos installation: %v\n", err)
			os.Exit(1)
		}

		shouldInstallProxy := clusterName != "" && centralBackendURL != ""

		if installed {
			fmt.Printf("\033[31mERROR\033[0m Odigos is already installed in namespace\n")
			os.Exit(1)
		}

		// Check if the cluster meets the minimum requirements
		clusterKind := cmdcontext.ClusterKindFromContext(ctx)
		if clusterKind == autodetect.KindUnknown {
			fmt.Println("Unknown Kubernetes cluster detected, proceeding with installation")
		} else {
			fmt.Printf("Detected cluster: Kubernetes kind: %s\n", clusterKind)
		}

		k8sVersion := cmdcontext.K8SVersionFromContext(ctx)
		if k8sVersion != nil {
			if k8sVersion.LessThan(k8sconsts.MinK8SVersionForInstallation) {
				fmt.Printf("\033[31mERROR\033[0m Odigos requires Kubernetes version %s or higher but found %s, aborting\n", k8sconsts.MinK8SVersionForInstallation.String(), k8sVersion.String())
				os.Exit(1)
			}
			fmt.Printf("Detected cluster: Kubernetes version: %s\n", k8sVersion.String())
		}

		var odigosProToken string
		odigosTier := common.OnPremOdigosTier
		if odigosCloudApiKeyFlag != "" {
			odigosTier = common.CloudOdigosTier
			odigosProToken = odigosCloudApiKeyFlag
			err = VerifyOdigosCloudApiKey(odigosCloudApiKeyFlag)
			if err != nil {
				fmt.Println("Odigos install failed - invalid api-key format.")
				os.Exit(1)
			}
		} else if odigosOnPremToken != "" {
			odigosTier = common.OnPremOdigosTier
			odigosProToken = odigosOnPremToken
		}

		if installCentralized {
			if err := installCentralBackendAndUI(ctx, client, ns); err != nil {
				fmt.Printf("\033[31mERROR\033[0m %s\n", err)
				os.Exit(1)
			}
			return
		}

		// validate user input profiles against available profiles
		err = ValidateUserInputProfiles(odigosTier)
		if err != nil {
			os.Exit(1)
		}

		config, err := getOrCreateConfig(ctx, client, ns, installed, odigosTier)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to prepare config: %v\n", err)
			os.Exit(1)
		}

		if shouldInstallProxy {
			config.ClusterName = clusterName
			config.CentralBackendURL = centralBackendURL
		}

		err = installOdigos(ctx, client, ns, config, &odigosProToken, odigosTier, "Creating", shouldInstallProxy, installed)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to install Odigos: %s\n", err)
			os.Exit(1)
		}

		if !skipWait {
			l := log.Print("Waiting for Odigos pods to be ready ...")
			err := wait.PollImmediate(1*time.Second, 3*time.Minute, arePodsReady(ctx, client, ns))
			if err != nil {
				l.Error(err)
			}

			l.Success()
		}

		fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos installed.\n")

	},
	Example: `
# Install Odigos open-source in your cluster.
odigos install

# Install Odigos cloud in your cluster.
odigos install --api-key <your-api-key>

# Install Odigos cloud in a specific cluster
odigos install --kubeconfig <path-to-kubeconfig>

# Install Odigos onprem tier for enterprise users
odigos install --onprem-token ${ODIGOS_TOKEN} --profile ${YOUR_ENTERPRISE_PROFILE_NAME}

# Install centralized backend and UI (must run in the central cluster after installing Odigos)
odigos install --centralized

# Install Odigos and connect the cluster to forward data to the centralized backend
odigos install --cluster-name my-cluster --central-backend-url https://central.odigos.local
`,
}

func isOdigosInstalled(ctx context.Context, client *kube.Client, ns string) (bool, error) {
	cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, k8sconsts.OdigosDeploymentConfigMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return cm != nil, nil
}

func installOdigos(ctx context.Context, client *kube.Client, ns string, config *common.OdigosConfiguration, token *string, odigosTier common.OdigosTier, label string, includeProxy bool, isOdigosInstall bool) error {
	managerOpts := resourcemanager.ManagerOpts{
		ImageReferences: GetImageReferences(odigosTier, openshiftEnabled),
		IncludeProxy:    includeProxy,
	}
	if isOdigosInstall {
		if err := resources.DeleteOldOdigosSystemObjects(ctx, client, ns, config); err != nil {
			return fmt.Errorf("cleanup old Odigos resources failed: %w", err)
		}
	}

	createKubeResourceWithLogging(ctx, fmt.Sprintf("> Creating namespace %s", ns), client, ns, createNamespace)

	resourceManagers := resources.CreateResourceManagers(client, ns, odigosTier, token, config, versionFlag, installationmethod.K8sInstallationMethodOdigosCli, managerOpts)
	return resources.ApplyResourceManagers(ctx, client, resourceManagers, label)

}

func installCentralBackendAndUI(ctx context.Context, client *kube.Client, ns string) error {
	fmt.Println("Installing centralized Odigos backend and UI ...")

	managerOpts := resourcemanager.ManagerOpts{
		ImageReferences: GetImageReferences(common.OnPremOdigosTier, openshiftEnabled),
	}

	centralNamespace := consts.DefaultOdigosCentralNamespace
	if ns != consts.DefaultOdigosNamespace {
		centralNamespace = ns
	}

	createKubeResourceWithLogging(ctx, fmt.Sprintf("> Creating namespace %s", centralNamespace),
		client, centralNamespace, createNamespace)

	resourceManagers := resources.CreateCentralizedManagers(client, managerOpts)
	if err := resources.ApplyResourceManagers(ctx, client, resourceManagers, "Creating"); err != nil {
		return fmt.Errorf("failed to install centralized Odigos: %w", err)
	}

	fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Centralized Odigos installed.\n")
	return nil
}

func getOrCreateConfig(ctx context.Context, client *kube.Client, ns string, installed bool, odigosTier common.OdigosTier) (*common.OdigosConfiguration, error) {
	if installed {
		config, err := resources.GetCurrentConfig(ctx, client, ns)
		if err != nil {
			return nil, fmt.Errorf("unable to read current Odigos config: %w", err)
		}
		config.ConfigVersion += 1
		return config, nil
	}

	cfg := CreateOdigosConfig(odigosTier)
	return &cfg, nil
}

func arePodsReady(ctx context.Context, client *kube.Client, ns string) func() (bool, error) {
	return func() (bool, error) {
		// ensure all DaemonSets in the odigos namespace have all their pods ready
		daemonSets, err := client.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		for _, ds := range daemonSets.Items {
			desiredPods := ds.Status.DesiredNumberScheduled
			readyPods := ds.Status.NumberReady
			if readyPods == 0 || readyPods != desiredPods {
				return false, nil
			}
		}

		// ensure all pods in the odigos namespace are running
		pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		runningPods := 0
		for _, p := range pods.Items {
			if p.Status.Phase == corev1.PodFailed {
				return false, fmt.Errorf("pod %s failed", p.Name)
			}

			if p.Status.Phase == corev1.PodRunning {
				runningPods++
			}
		}

		return runningPods == len(pods.Items), nil
	}
}

func createNamespace(ctx context.Context, client *kube.Client, ns string) error {
	nsObj, err := client.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err := client.CoreV1().Namespaces().Create(ctx, resources.NewNamespace(ns), metav1.CreateOptions{})
			return err
		}
		return err
	}

	val, exists := nsObj.Labels[k8sconsts.OdigosSystemLabelKey]
	if !exists || val != k8sconsts.OdigosSystemLabelValue {
		return fmt.Errorf("namespace %s does not contain %s label", ns, k8sconsts.OdigosSystemLabelKey)
	}

	return nil
}

func ValidateUserInputProfiles(tier common.OdigosTier) error {
	// Fetch available profiles for the given tier
	availableProfiles := profiles.GetAvailableProfilesForTier(tier)

	// Create a map for fast lookups of valid profile names
	profileMap := make(map[string]struct{})
	for _, profile := range availableProfiles {
		profileMap[string(profile.ProfileName)] = struct{}{}
	}

	// Check each user input profile against the map
	for _, input := range userInputInstallProfiles {
		if _, exists := profileMap[input]; !exists {
			fmt.Printf("\033[31mERROR\033[0m Profile '%s' not available.\n", input)
			return errors.New("profile " + input + " not available")
		}
	}
	return nil
}

func GetImageReferences(odigosTier common.OdigosTier, openshift bool) resourcemanager.ImageReferences {
	var imageReferences resourcemanager.ImageReferences
	if openshift {
		imageReferences = resourcemanager.ImageReferences{
			AutoscalerImage:   k8sconsts.AutoScalerImageUBI9,
			CollectorImage:    k8sconsts.OdigosClusterCollectorImageUBI9,
			InstrumentorImage: k8sconsts.InstrumentorImageUBI9,
			OdigletImage:      k8sconsts.OdigletImageUBI9,
			KeyvalProxyImage:  k8sconsts.KeyvalProxyImage,
			SchedulerImage:    k8sconsts.SchedulerImageUBI9,
			UIImage:           k8sconsts.UIImageUBI9,
		}
	} else {
		imageReferences = resourcemanager.ImageReferences{
			AutoscalerImage:   k8sconsts.AutoScalerImageName,
			CollectorImage:    k8sconsts.OdigosClusterCollectorImage,
			InstrumentorImage: k8sconsts.InstrumentorImage,
			OdigletImage:      k8sconsts.OdigletImageName,
			KeyvalProxyImage:  k8sconsts.KeyvalProxyImage,
			SchedulerImage:    k8sconsts.SchedulerImage,
			UIImage:           k8sconsts.UIImage,
		}
	}

	if odigosTier == common.OnPremOdigosTier {
		if openshift {
			imageReferences.InstrumentorImage = k8sconsts.InstrumentorEnterpriseImageUBI9
			imageReferences.OdigletImage = k8sconsts.OdigletEnterpriseImageUBI9
		} else {
			imageReferences.InstrumentorImage = k8sconsts.InstrumentorEnterpriseImage
			imageReferences.OdigletImage = k8sconsts.OdigletEnterpriseImageName
		}
	}
	return imageReferences
}

func CreateOdigosConfig(odigosTier common.OdigosTier) common.OdigosConfiguration {
	selectedProfiles := []common.ProfileName{}
	for _, profile := range userInputInstallProfiles {
		selectedProfiles = append(selectedProfiles, common.ProfileName(profile))
	}

	if openshiftEnabled {
		if imagePrefix == "" {
			imagePrefix = k8sconsts.RedHatImagePrefix
		}
		odigletImage = k8sconsts.OdigletImageUBI9
		instrumentorImage = k8sconsts.InstrumentorImageUBI9
		autoScalerImage = k8sconsts.AutoScalerImageUBI9
	}

	return common.OdigosConfiguration{
		ConfigVersion:             1, // config version starts at 1 and incremented on every config change
		TelemetryEnabled:          telemetryEnabled,
		OpenshiftEnabled:          openshiftEnabled,
		IgnoredNamespaces:         userInputIgnoredNamespaces,
		IgnoredContainers:         userInputIgnoredContainers,
		SkipWebhookIssuerCreation: skipWebhookIssuerCreation,
		Psp:                       psp,
		ImagePrefix:               imagePrefix,
		Profiles:                  selectedProfiles,
		UiMode:                    common.UiMode(uiMode),
	}
}

func createKubeResourceWithLogging(ctx context.Context, msg string, client *kube.Client, ns string, create ResourceCreationFunc) {
	l := log.Print(msg)
	err := create(ctx, client, ns)
	if err != nil {
		l.Error(err)
	}

	l.Success()
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVarP(&namespaceFlag, "namespace", "n", consts.DefaultOdigosNamespace, "target k8s namespace for Odigos installation")
	installCmd.Flags().StringVarP(&odigosCloudApiKeyFlag, "api-key", "k", "", "api key for odigos cloud")
	installCmd.Flags().StringVarP(&odigosOnPremToken, "onprem-token", "", "", "authentication token for odigos enterprise on-premises")
	installCmd.Flags().BoolVar(&skipWait, "nowait", false, "skip waiting for odigos pods to be ready")
	installCmd.Flags().BoolVar(&telemetryEnabled, "telemetry", true, "send general telemetry regarding Odigos usage")
	installCmd.Flags().BoolVar(&openshiftEnabled, "openshift", false, "configure requirements for OpenShift: required selinux settings, RBAC roles, and will use OpenShift certified images (if --image-prefix is not set)")
	installCmd.Flags().BoolVar(&skipWebhookIssuerCreation, consts.SkipWebhookIssuerCreationProperty, false, "Skip creating the Issuer and Certificate for the Instrumentor pod webhook if cert-manager is installed.")
	installCmd.Flags().StringVar(&imagePrefix, consts.ImagePrefixProperty, "registry.odigos.io", "prefix for all container images.")
	installCmd.Flags().BoolVar(&psp, consts.PspProperty, false, "enable pod security policy")
	installCmd.Flags().StringSliceVar(&userInputIgnoredNamespaces, "ignore-namespace", k8sconsts.DefaultIgnoredNamespaces, "namespaces not to show in odigos ui")
	installCmd.Flags().StringSliceVar(&userInputIgnoredContainers, "ignore-container", k8sconsts.DefaultIgnoredContainers, "container names to exclude from instrumentation (useful for sidecar container)")
	installCmd.Flags().StringSliceVar(&userInputInstallProfiles, "profile", []string{}, "install preset profiles with a specific configuration")
	installCmd.Flags().StringVarP(&uiMode, consts.UiModeProperty, "", string(common.NormalUiMode), "set the UI mode (one-of: normal, readonly)")
	installCmd.Flags().BoolVar(&installCentralized, "centralized", false, "Install centralized Odigos UI and backend")
	installCmd.Flags().StringVar(&clusterName, "cluster-name", "", "name of the cluster to be used in the centralized backend")
	installCmd.Flags().StringVar(&centralBackendURL, "central-backend-url", "", "use to connect this cluster to the centralized odigos cluster")

	if OdigosVersion != "" {
		versionFlag = OdigosVersion
	} else {
		installCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "for development purposes only")
	}
}
