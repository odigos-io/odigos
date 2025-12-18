package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
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
	odigosCloudApiKeyFlag            string
	odigosOnPremToken                string
	namespaceFlag                    string
	versionFlag                      string
	skipWait                         bool
	telemetryEnabled                 bool
	openshiftEnabled                 bool
	skipWebhookIssuerCreation        bool
	psp                              bool
	userInputIgnoredNamespaces       []string
	userInputIgnoredContainers       []string
	userInputInstallProfiles         []string
	uiMode                           string
	customContainerRuntimeSocketPath string
	k8sNodeLogsDirectory             string
	instrumentorImage                string
	odigletImage                     string
	autoScalerImage                  string
	imagePrefix                      string
	nodeSelectorFlag                 string
	karpenterEnabled                 bool

	clusterName       string
	centralBackendURL string

	userInstrumentationEnvsRaw string

	autoRollbackDisabled         bool
	autoRollbackGraceTime        string
	autoRollbackStabilityWindows string
)

type ResourceCreationFunc func(ctx context.Context, client *kube.Client, ns string, labelKey string) error

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install-deprecated",
	Short: "Install Odigos in your kubernetes cluster.",
	Long: `This command is deprecated. Please use ` + "`odigos install`" + ` instead. which uses the Helm SDK under the hood.
This sub command will Install Odigos in your kubernetes cluster.
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

		if installed {
			fmt.Printf("\033[31mERROR\033[0m Odigos is already installed in namespace\n")
			os.Exit(1)
		}

		if clusterName == "" && centralBackendURL != "" {
			fmt.Printf("\033[33mWARNING\033[0m You provided a central backend URL but no cluster name.\n")
			fmt.Println("Odigos will be installed, but this cluster will NOT be connected to the centralized Odigos backend.")
			fmt.Println("To connect it later, run: \033[36modigos config set --cluster-name <your-cluster-name> \033[0m")
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
		odigosTier := common.CommunityOdigosTier
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
		if centralBackendURL != "" && odigosTier != common.OnPremOdigosTier {
			fmt.Printf("\033[31mERROR\033[0m Central backend connection is only available in the OnPrem tier.\n")
			fmt.Println("Please upgrade to the OnPrem tier or remove the --central-backend-url flag.")
		}
		// validate user input profiles against available profiles
		err = ValidateUserInputProfiles(odigosTier)
		if err != nil {
			os.Exit(1)
		}

		nodeSelector, err := parseNodeSelectorFlag()
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Unable to parse node-selector flag.\n")
			os.Exit(1)
		}

		config := CreateOdigosConfiguration(odigosTier, nodeSelector)

		err = installOdigos(ctx, client, ns, &config, &odigosProToken, odigosTier, "Creating")
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

# Install Odigos and connect the cluster to forward data to the centralized backend
odigos install --cluster-name ${YOUR_CLUSTER_NAME} --central-backend-url ${YOUR_CENTRAL_BACKEND_URL}
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

func installOdigos(ctx context.Context, client *kube.Client, ns string, config *common.OdigosConfiguration, token *string, odigosTier common.OdigosTier, label string) error {
	fmt.Printf("Installing Odigos version %s in namespace %s ...\n", versionFlag, ns)

	managerOpts := resourcemanager.ManagerOpts{
		ImageReferences: GetImageReferences(odigosTier, openshiftEnabled),
	}

	createKubeResourceWithLogging(ctx, fmt.Sprintf("> Creating namespace %s", ns), client, ns, k8sconsts.OdigosSystemLabelKey, createNamespace)

	resourceManagers := resources.CreateResourceManagers(client, ns, odigosTier, token, config, versionFlag, installationmethod.K8sInstallationMethodOdigosCli, managerOpts)
	return resources.ApplyResourceManagers(ctx, client, resourceManagers, label)
}

func parseNodeSelectorFlag() (map[string]string, error) {
	nodeSelector := make(map[string]string)
	if len(nodeSelectorFlag) == 0 {
		return nodeSelector, nil
	}
	selectors := strings.Split(nodeSelectorFlag, ",")
	for _, selector := range selectors {
		s := strings.Split(selector, "=")
		if len(s) != 2 {
			return nodeSelector, errors.New(fmt.Sprintf("invalid node selector, must be in form 'key=value': %s", selector))
		}
		nodeSelector[s[0]] = s[1]
	}
	return nodeSelector, nil
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

func CreateOdigosConfiguration(odigosTier common.OdigosTier, nodeSelector map[string]string) common.OdigosConfiguration {
	selectedProfiles := []common.ProfileName{}
	for _, profile := range userInputInstallProfiles {
		selectedProfiles = append(selectedProfiles, common.ProfileName(profile))
	}

	if openshiftEnabled {
		if imagePrefix == "" {
			imagePrefix = k8sconsts.RedHatImagePrefix
		}
		odigletImage = k8sconsts.OdigletImageCertified
		instrumentorImage = k8sconsts.InstrumentorImageCertified
		autoScalerImage = k8sconsts.AutoScalerImageCertified
	}

	var parsedUserJson *common.UserInstrumentationEnvs
	if userInstrumentationEnvsRaw != "" {
		parsedUserJson = &common.UserInstrumentationEnvs{}
		if err := json.Unmarshal([]byte(userInstrumentationEnvsRaw), &parsedUserJson); err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to parse --user-instrumentation-envs: %v\n", err)
		}
	}

	return common.OdigosConfiguration{
		ConfigVersion:                    1, // config version starts at 1 and incremented on every config change
		TelemetryEnabled:                 telemetryEnabled,
		OpenshiftEnabled:                 openshiftEnabled,
		IgnoredNamespaces:                userInputIgnoredNamespaces,
		CustomContainerRuntimeSocketPath: customContainerRuntimeSocketPath,
		CollectorNode: &common.CollectorNodeConfiguration{
			K8sNodeLogsDirectory: k8sNodeLogsDirectory,
		},
		IgnoredContainers:         userInputIgnoredContainers,
		SkipWebhookIssuerCreation: skipWebhookIssuerCreation,
		Psp:                       psp,
		ImagePrefix:               imagePrefix,
		Profiles:                  selectedProfiles,
		UiMode:                    common.UiMode(uiMode),
		UiPaginationLimit:         100,
		ClusterName:               clusterName,
		CentralBackendURL:         centralBackendURL,
		UserInstrumentationEnvs:   parsedUserJson,
		NodeSelector:              nodeSelector,
		RollbackDisabled:          &autoRollbackDisabled,
		RollbackGraceTime:         autoRollbackGraceTime,
		RollbackStabilityWindow:   autoRollbackStabilityWindows,
	}

}

func createKubeResourceWithLogging(ctx context.Context, msg string, client *kube.Client, ns string, labelScope string, create ResourceCreationFunc) {
	l := log.Print(msg)
	err := create(ctx, client, ns, labelScope)
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
	installCmd.Flags().StringVar(&imagePrefix, consts.ImagePrefixProperty, k8sconsts.OdigosImagePrefix, "prefix for all container images.")
	installCmd.Flags().BoolVar(&psp, consts.PspProperty, false, "enable pod security policy")
	installCmd.Flags().StringSliceVar(&userInputIgnoredNamespaces, "ignore-namespace", k8sconsts.DefaultIgnoredNamespaces, "namespaces not to show in odigos ui")
	installCmd.Flags().StringVar(&customContainerRuntimeSocketPath, "container-runtime-socket-path", "", "custom configuration of a path to the container runtime socket path (e.g. /var/lib/rancher/rke2/agent/containerd/containerd.sock)")
	installCmd.Flags().StringVar(&k8sNodeLogsDirectory, consts.K8sNodeLogsDirectory, "", "custom configuration of a path to the directory where Kubernetes logs are symlinked in a node (e.g. /mnt/var/log)")
	installCmd.Flags().StringSliceVar(&userInputIgnoredContainers, "ignore-container", k8sconsts.DefaultIgnoredContainers, "container names to exclude from instrumentation (useful for sidecar container)")
	installCmd.Flags().StringSliceVar(&userInputInstallProfiles, "profile", []string{}, "install preset profiles with a specific configuration")
	installCmd.Flags().StringVarP(&uiMode, consts.UiModeProperty, "", string(common.UiModeDefault), "set the UI mode (one-of: default, readonly)")
	installCmd.Flags().StringVar(&nodeSelectorFlag, "node-selector", "", "comma-separated key=value pair of Kubernetes NodeSelectors to set on Odigos components. Example: kubernetes.io/hostname=myhost")

	installCmd.Flags().StringVar(&clusterName, "cluster-name", "", "name of the cluster to be used in the centralized backend")
	installCmd.Flags().StringVar(&centralBackendURL, "central-backend-url", "", "use to connect this cluster to the centralized odigos cluster")
	installCmd.Flags().StringVar(
		&userInstrumentationEnvsRaw,
		"user-instrumentation-envs",
		"",
		"JSON string to configure per-language instrumentation envs, e.g. '{\"languages\":{\"go\":{\"enabled\":true,\"env\":{\"OTEL_GO_ENABLED\":\"true\"}}}}'",
	)
	installCmd.Flags().BoolVar(&autoRollbackDisabled, consts.RollbackDisabledProperty, false, "Disabled the auto rollback feature")
	installCmd.Flags().StringVar(&autoRollbackGraceTime, consts.RollbackGraceTimeProperty, consts.DefaultAutoRollbackGraceTime, "Auto rollback grace time")
	installCmd.Flags().StringVar(&autoRollbackStabilityWindows, consts.RollbackStabilityWindow, consts.DefaultAutoRollbackStabilityWindow, "Auto rollback stability windows time")
	if OdigosVersion != "" {
		versionFlag = OdigosVersion
	} else {
		installCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "for development purposes only")
	}
}
