package cmd

import (
	"context"
	"fmt"
	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/cmd/resources/crds"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/keyval-dev/odigos/cli/pkg/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultNamespace = "odigos-system"
)

var (
	namespaceFlag string
)

type ResourceCreationFunc func(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := kube.CreateClient(cmd)
		ctx := cmd.Context()
		ns := cmd.Flag("namespace").Value.String()
		createKubeResourceWithLogging(ctx, fmt.Sprintf("Creating namespace %s", ns),
			client, cmd, ns, createNamespace)
		createKubeResourceWithLogging(ctx, "Creating CRDs",
			client, cmd, ns, createCRDs)
		createKubeResourceWithLogging(ctx, "Creating Leader Election Role",
			client, cmd, ns, createLeaderElectionRole)
		createKubeResourceWithLogging(ctx, "Deploying Instrumentor",
			client, cmd, ns, createInstrumentor)
	},
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
}
