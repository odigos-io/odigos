package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/keyval-dev/odigos/cli/pkg/containers"

	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/cmd/resources/crds"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/keyval-dev/odigos/cli/pkg/log"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	defaultNamespace        = "odigos-system"
	odigosCloudProxyVersion = "v0.3.0"
)

var (
	odigosCloudApiKeyFlag    string
	namespaceFlag            string
	versionFlag              string
	skipWait                 bool
	telemetryEnabled         bool
	sidecarInstrumentation   bool
	psp                      bool
	ignoredNamespaces        []string
	DefaultIgnoredNamespaces = []string{"odigos-system", "kube-system", "local-path-storage", "istio-system", "linkerd"}
)

type ResourceCreationFunc func(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Odigos",
	Long:  `Install Odigos in your cluster. This command will install the Odigos CRDs, the Odigos Instrumentor, Scheduler, Autoscaler and Odiglet.`,
	Run: func(cmd *cobra.Command, args []string) {

		client := kube.CreateClient(cmd)
		ctx := cmd.Context()
		ns := cmd.Flag("namespace").Value.String()
		cmd.Flags().StringSliceVar(&ignoredNamespaces, "ignore-namespace", DefaultIgnoredNamespaces, "--ignore-namespace foo logging")
		fmt.Printf("Installing Odigos version %s in namespace %s ...\n", versionFlag, ns)
		createKubeResourceWithLogging(ctx, fmt.Sprintf("Creating namespace %s", ns),
			client, cmd, ns, createNamespace)
		createKubeResourceWithLogging(ctx, "Creating CRDs",
			client, cmd, ns, createCRDs)
		createKubeResourceWithLogging(ctx, "Creating Leader Election Role",
			client, cmd, ns, createLeaderElectionRole)
		createKubeResourceWithLogging(ctx, "Creating RBAC",
			client, cmd, ns, createDataCollectionRBAC)
		createKubeResourceWithLogging(ctx, "Deploying Instrumentor",
			client, cmd, ns, createInstrumentor)
		createKubeResourceWithLogging(ctx, "Deploying Scheduler",
			client, cmd, ns, createScheduler)
		createKubeResourceWithLogging(ctx, "Deploying Odiglet",
			client, cmd, ns, createOdiglet)
		createKubeResourceWithLogging(ctx, "Deploying Autoscaler",
			client, cmd, ns, createAutoscaler)

		if odigosCloudApiKeyFlag != "" {
			createKubeResourceWithLogging(ctx, "Deploying Odigos Cloud Proxy",
				client, cmd, ns, createKeyvalProxy)
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
	_, err := client.CoreV1().Namespaces().Create(ctx, resources.NewNamespace(ns), metav1.CreateOptions{})
	return err
}

func createCRDs(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	for _, crd := range crds.NewCRDs() {
		_, err := client.ApiExtensions.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, &crd, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func createLeaderElectionRole(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	_, err := client.RbacV1().Roles(ns).Create(ctx, resources.NewLeaderElectionRole(), metav1.CreateOptions{})
	return err
}

func createDataCollectionRBAC(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	_, err := client.CoreV1().ServiceAccounts(ns).Create(ctx, resources.NewDataCollectionServiceAccount(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoles().Create(ctx, resources.NewDataCollectionClusterRole(psp), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, resources.NewDataCollectionClusterRoleBinding(ns), metav1.CreateOptions{})
	return err
}

func createInstrumentor(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	_, err := client.CoreV1().ServiceAccounts(ns).Create(ctx, resources.NewInstrumentorServiceAccount(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().RoleBindings(ns).Create(ctx, resources.NewInstrumentorRoleBinding(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoles().Create(ctx, resources.NewInstrumentorClusterRole(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, resources.NewInstrumentorClusterRoleBinding(ns), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(ns).Create(ctx, resources.NewInstrumentorDeployment(versionFlag, telemetryEnabled, sidecarInstrumentation, ignoredNamespaces), metav1.CreateOptions{})
	return err
}

func createScheduler(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	_, err := client.CoreV1().ServiceAccounts(ns).Create(ctx, resources.NewSchedulerServiceAccount(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().RoleBindings(ns).Create(ctx, resources.NewSchedulerRoleBinding(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoles().Create(ctx, resources.NewSchedulerClusterRole(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, resources.NewSchedulerClusterRoleBinding(ns), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(ns).Create(ctx, resources.NewSchedulerDeployment(versionFlag), metav1.CreateOptions{})
	return err
}

func createAutoscaler(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	_, err := client.CoreV1().ServiceAccounts(ns).Create(ctx, resources.NewAutoscalerServiceAccount(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().Roles(ns).Create(ctx, resources.NewAutoscalerRole(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().RoleBindings(ns).Create(ctx, resources.NewAutoscalerRoleBinding(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoles().Create(ctx, resources.NewAutoscalerClusterRole(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, resources.NewAutoscalerClusterRoleBinding(ns), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().RoleBindings(ns).Create(ctx, resources.NewAutoscalerLeaderElectionRoleBinding(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(ns).Create(ctx, resources.NewAutoscalerDeployment(versionFlag), metav1.CreateOptions{})
	return err
}

func createOdiglet(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	_, err := client.CoreV1().ServiceAccounts(ns).Create(ctx, resources.NewOdigletServiceAccount(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoles().Create(ctx, resources.NewOdigletClusterRole(psp), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, resources.NewOdigletClusterRoleBinding(ns), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.AppsV1().DaemonSets(ns).Create(ctx, resources.NewOdigletDaemonSet(versionFlag), metav1.CreateOptions{})
	return err
}

func createKeyvalProxy(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {

	_, err := client.CoreV1().ServiceAccounts(ns).Create(ctx, resources.NewKeyvalProxyServiceAccount(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().Roles(ns).Create(ctx, resources.NewKeyvalProxyRole(ns), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().RoleBindings(ns).Create(ctx, resources.NewKeyvalProxyRoleBinding(ns), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoles().Create(ctx, resources.NewKeyvalProxyClusterRole(), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, resources.NewKeyvalProxyClusterRoleBinding(ns), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.CoreV1().Secrets(ns).Create(ctx, resources.NewKeyvalSecret(odigosCloudApiKeyFlag), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(ns).Create(ctx, resources.NewKeyvalProxyDeployment(odigosCloudProxyVersion, ns), metav1.CreateOptions{})
	return err
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
	installCmd.Flags().StringVarP(&namespaceFlag, "namespace", "n", defaultNamespace, "target namespace for Odigos installation")
	installCmd.Flags().StringVarP(&odigosCloudApiKeyFlag, "api-key", "k", "", "api key for managed odigos")
	installCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "target version for Odigos installation")
	installCmd.Flags().BoolVar(&skipWait, "nowait", false, "Skip waiting for pods to be ready")
	installCmd.Flags().BoolVar(&telemetryEnabled, "telemetry", true, "Enable telemetry")
	installCmd.Flags().BoolVar(&sidecarInstrumentation, "sidecar-instrumentation", false, "Used sidecars for eBPF instrumentations")
	installCmd.Flags().StringVar(&resources.OdigletImage, "odiglet-image", "keyval/odigos-odiglet", "odiglet container image")
	installCmd.Flags().StringVar(&resources.InstrumentorImage, "instrumentor-image", "keyval/odigos-instrumentor", "instrumentor container image")
	installCmd.Flags().StringVar(&resources.AutoscalerImage, "autoscaler-image", "keyval/odigos-autoscaler", "autoscaler container image")
	installCmd.Flags().StringVar(&containers.ImagePrefix, "image-prefix", "", "Prefix for all container images")
	installCmd.Flags().BoolVar(&psp, "psp", false, "Enable pod security policy")
}
