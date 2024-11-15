package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/odigos-io/odigos/cli/pkg/autodetect"

	"github.com/odigos-io/odigos/cli/pkg/labels"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/utils"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/version"
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
	psp                        bool
	userInputIgnoredNamespaces []string
	userInputIgnoredContainers []string
	userInputInstallProfiles   []string

	instrumentorImage string
	odigletImage      string
	autoScalerImage   string
	imagePrefix       string
)

var (
	// minK8SVersionForInstallation is the minimum Kubernetes version required for Odigos installation
	// this value must be in sync with the one defined in the kubeVersion field in Chart.yaml
	minK8SVersionForInstallation = version.MustParse("v1.20.15")
)

type ResourceCreationFunc func(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Odigos",
	Long: `Install Odigos in your kubernetes cluster.
This command will install k8s components that will auto-instrument your applications with OpenTelemetry and send traces, metrics and logs to any telemetry backend`,
	Run: func(cmd *cobra.Command, args []string) {

		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}
		ctx := cmd.Context()
		ns := cmd.Flag("namespace").Value.String()

		// Check if Odigos already installed
		cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, k8sconsts.OdigosDeploymentConfigMapName, metav1.GetOptions{})
		if err == nil && cm != nil {
			fmt.Printf("\033[31mERROR\033[0m Odigos is already installed in namespace\n")
			os.Exit(1)
		}

		// Check if the cluster meets the minimum requirements
		kc := cmd.Flag("kubeconfig").Value.String()
		details, err := autodetect.DetectK8SClusterDetails(ctx, kc, client)
		if !errors.Is(err, autodetect.ErrCannotDetectClusterKind) {
			autodetect.CurrentKubernetesVersion.Kind = details.Kind
			fmt.Printf("Detected cluster: Kubernetes kind: %s\n", details.Kind)
		} else {
			fmt.Println("Unknown Kubernetes cluster detected, proceeding with installation")
		}

		if !errors.Is(err, autodetect.ErrCannotDetectK8sVersion) {
			autodetect.CurrentKubernetesVersion.Version = details.K8SVersion
			if details.K8SVersion.LessThan(minK8SVersionForInstallation) {
				fmt.Printf("\033[31mERROR\033[0m Odigos requires Kubernetes version %s or higher but found %s, aborting\n", minK8SVersionForInstallation.String(), details.K8SVersion.String())
				os.Exit(1)
			}
			fmt.Printf("Detected cluster: Kubernetes version: %s\n", details.K8SVersion.String())
		} else {
			fmt.Println("Unknown Kubernetes version detected, proceeding with installation")
		}

		var odigosProToken string
		odigosTier := common.CommunityOdigosTier
		if odigosCloudApiKeyFlag != "" {
			odigosTier = common.CloudOdigosTier
			odigosProToken = odigosCloudApiKeyFlag
			err = verifyOdigosCloudApiKey(odigosCloudApiKeyFlag)
			if err != nil {
				fmt.Println("Odigos install failed - invalid api-key format.")
				os.Exit(1)
			}
		} else if odigosOnPremToken != "" {
			odigosTier = common.OnPremOdigosTier
			odigosProToken = odigosOnPremToken
		}

		config := createOdigosConfig(odigosTier)

		fmt.Printf("Installing Odigos version %s in namespace %s ...\n", versionFlag, ns)

		// namespace is created on "install" and is not managed by resource manager
		createKubeResourceWithLogging(ctx, fmt.Sprintf("> Creating namespace %s", ns),
			client, cmd, ns, createNamespace)

		resourceManagers := resources.CreateResourceManagers(client, ns, odigosTier, &odigosProToken, &config, versionFlag)
		err = resources.ApplyResourceManagers(ctx, client, resourceManagers, "Creating")
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

func createNamespace(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	nsObj, err := client.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err := client.CoreV1().Namespaces().Create(ctx, resources.NewNamespace(ns), metav1.CreateOptions{})
			return err
		}
		return err
	}

	val, exists := nsObj.Labels[labels.OdigosSystemLabelKey]
	if !exists || val != labels.OdigosSystemLabelValue {
		return fmt.Errorf("namespace %s does not contain %s label", ns, labels.OdigosSystemLabelKey)
	}

	return nil
}

func createOdigosConfig(odigosTier common.OdigosTier) common.OdigosConfiguration {
	fullIgnoredNamespaces := utils.MergeDefaultIgnoreWithUserInput(userInputIgnoredNamespaces, consts.SystemNamespaces)
	fullIgnoredContainers := utils.MergeDefaultIgnoreWithUserInput(userInputIgnoredContainers, consts.IgnoredContainers)

	selectedProfiles := []common.ProfileName{}
	profiles := resources.GetAvailableProfilesForTier(odigosTier)
	for _, profile := range userInputInstallProfiles {
		found := false
		for _, p := range profiles {
			if string(p.ProfileName) == profile {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("\033[34mINFO\033[0m Profile '%s' skipped - not available for tier '%s'.\n", profile, odigosTier)
		} else {
			selectedProfiles = append(selectedProfiles, common.ProfileName(profile))
		}
	}

	return common.OdigosConfiguration{
		ConfigVersion:     1, // config version starts at 1 and incremented on every config change
		TelemetryEnabled:  telemetryEnabled,
		OpenshiftEnabled:  openshiftEnabled,
		IgnoredNamespaces: fullIgnoredNamespaces,
		IgnoredContainers: fullIgnoredContainers,
		Psp:               psp,
		ImagePrefix:       imagePrefix,
		OdigletImage:      odigletImage,
		InstrumentorImage: instrumentorImage,
		AutoscalerImage:   autoScalerImage,
		Profiles:          selectedProfiles,
	}
}

func createKubeResourceWithLogging(ctx context.Context, msg string, client *kube.Client, cmd *cobra.Command, ns string, create ResourceCreationFunc) {
	l := log.Print(msg)
	err := create(ctx, cmd, client, ns)
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
	installCmd.Flags().BoolVar(&openshiftEnabled, "openshift", false, "configure selinux on openshift nodes")
	installCmd.Flags().StringVar(&odigletImage, "odiglet-image", "", "odiglet container image name")
	installCmd.Flags().StringVar(&instrumentorImage, "instrumentor-image", "keyval/odigos-instrumentor", "instrumentor container image name")
	installCmd.Flags().StringVar(&autoScalerImage, "autoscaler-image", "keyval/odigos-autoscaler", "autoscaler container image name")
	installCmd.Flags().StringVar(&imagePrefix, "image-prefix", "", "prefix for all container images. used when your cluster doesn't have access to docker hub")
	installCmd.Flags().BoolVar(&psp, "psp", false, "enable pod security policy")
	installCmd.Flags().StringSliceVar(&userInputIgnoredNamespaces, "ignore-namespace", consts.SystemNamespaces, "namespaces not to show in odigos ui")
	installCmd.Flags().StringSliceVar(&userInputIgnoredContainers, "ignore-container", consts.IgnoredContainers, "container names to exclude from instrumentation (useful for sidecar container)")
	installCmd.Flags().StringSliceVar(&userInputInstallProfiles, "profile", []string{}, "install preset profiles with a specific configuration")

	if OdigosVersion != "" {
		versionFlag = OdigosVersion
	} else {
		installCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "for development purposes only")
	}
}
