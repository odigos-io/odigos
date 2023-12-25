package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/pkg/confirm"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/keyval-dev/odigos/cli/pkg/labels"
	"github.com/keyval-dev/odigos/cli/pkg/log"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/consts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/spf13/cobra"
)

const (
	goAgentImage                = "keyval/otel-go-agent"
	golangKernelDebugVolumeName = "kernel-debug"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Unistall Odigos from your cluster",
	Run: func(cmd *cobra.Command, args []string) {
		println("Unistall Odigos from your cluster...")

		client, err := kube.CreateClient(cmd)

		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}

		ctx := cmd.Context()

		ns, err := resources.GetOdigosNamespace(client, ctx)
		if resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Println("\033[31mERROR\033[0m odigos is not currently installed in the cluster, so there is nothing to uninstall")
			os.Exit(1)
		} else if !resources.IsErrNoOdigosNamespaceFound(err) && err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already uninstalled: %s\n", err)
			os.Exit(1)
		}
		println("Odigos namespace: " + ns)
		if !cmd.Flag("yes").Changed {
			fmt.Printf("About to uninstall Odigos from namespace %s\n", ns)
			confirmed, err := confirm.Ask("Are you sure?")
			if err != nil || !confirmed {
				fmt.Println("Aborting uninstall")
				return
			}
		}

		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Deployments",
			client, cmd, ns, uninstallDeployments)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos DaemonSets",
			client, cmd, ns, uninstallDaemonSets)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos ConfigMaps",
			client, cmd, ns, uninstallConfigMaps)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Services",
			client, cmd, ns, uninstallServices)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos CRDs",
			client, cmd, ns, uninstallCRDs)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos RBAC",
			client, cmd, ns, uninstallRBAC)
		createKubeResourceWithLogging(ctx, "Uninstalling Odigos Secrets",
			client, cmd, ns, uninstallSecrets)
		createKubeResourceWithLogging(ctx, fmt.Sprintf("Uninstalling Namespace %s", ns),
			client, cmd, ns, uninstallNamespace)

		// Wait for namespace to be deleted
		waitForNamespaceDeletion(ctx, client, ns)

		l := log.Print("Rolling back odigos changes to pods")
		err = rollbackPodChanges(ctx, client)
		if err != nil {
			l.Error(err)
		} else {
			l.Success()
		}

		l = log.Print("Rolling back odigos changes to namespaces")
		err = rollbackNamespaceChanges(ctx, client)
		if err != nil {
			l.Error(err)
		} else {
			l.Success()
		}

		fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos uninstalled.\n")
	},
}

func waitForNamespaceDeletion(ctx context.Context, client *kube.Client, ns string) {
	l := log.Print("Waiting for namespace to be deleted")
	wait.PollImmediate(1*time.Second, 5*time.Minute, func() (bool, error) {
		_, err := client.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err != nil {
			l.Success()
			return true, nil
		}
		return false, nil
	})
}

func rollbackPodChanges(ctx context.Context, client *kube.Client) error {
	deps, err := client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	
	for _, dep := range deps.Items {
		if dep.Namespace == "odigos-system" {
			continue
		}

		// Remove the "odigos-instrumentation" label
		if labels := dep.GetLabels(); labels != nil {
			delete(labels, "odigos-instrumentation")
			dep.SetLabels(labels)
		}

		rollbackPodTemplateSpec(ctx, client, &dep.Spec.Template)
		if _, err := client.AppsV1().Deployments(dep.Namespace).Update(ctx, &dep, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	ss, err := client.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, s := range ss.Items {
		if s.Namespace == "odigos-system" {
			continue
		}

		// Remove the "odigos-instrumentation" label
		if labels := s.GetLabels(); labels != nil {
			delete(labels, "odigos-instrumentation")
			s.SetLabels(labels)
		}

		rollbackPodTemplateSpec(ctx, client, &s.Spec.Template)
		if _, err := client.AppsV1().StatefulSets(s.Namespace).Update(ctx, &s, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	dd, err := client.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, d := range dd.Items {
		if d.Namespace == "odigos-system" {
			continue
		}

		// Remove the "odigos-instrumentation" label
		if labels := d.GetLabels(); labels != nil {
			delete(labels, "odigos-instrumentation")
			d.SetLabels(labels)
		}

		rollbackPodTemplateSpec(ctx, client, &d.Spec.Template)
		if _, err := client.AppsV1().DaemonSets(d.Namespace).Update(ctx, &d, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func rollbackPodTemplateSpec(ctx context.Context, client *kube.Client, pts *v1.PodTemplateSpec) {
	// Odigos instruments pods in two ways:
	// A. For Java/.NET/Python/NodeJS apps, it adds a resource limit to the container
	instrumentedViaResourceLimit := false
	for i, c := range pts.Spec.Containers {
		if c.Resources.Limits != nil {
			for val := range c.Resources.Limits {
				if strings.Contains(val.String(), common.OdigosResourceNamespace) {
					instrumentedViaResourceLimit = true
					delete(pts.Spec.Containers[i].Resources.Limits, val)
				}
			}
		}
	}

	if instrumentedViaResourceLimit {
		return
	}

	// B. For Go apps, it adds a sidecar container

	// Remove containers with go agent image
	for i, c := range pts.Spec.Containers {
		if strings.Contains(c.Image, goAgentImage) {
			pts.Spec.Containers = append(pts.Spec.Containers[:i], pts.Spec.Containers[i+1:]...)
		}
	}

	// Roll back shared process namespace
	pts.Spec.ShareProcessNamespace = nil

	// Remove odigos volumes
	for i, v := range pts.Spec.Volumes {
		if v.Name == golangKernelDebugVolumeName {
			pts.Spec.Volumes = append(pts.Spec.Volumes[:i], pts.Spec.Volumes[i+1:]...)
		}
	}
}

func rollbackNamespaceChanges(ctx context.Context, client *kube.Client) error {
	var enabledOdigosInstrumentationLabel = map[string]string{
		consts.OdigosInstrumentationLabel: consts.InstrumentationEnabled,
	}
	ns, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: enabledOdigosInstrumentationLabel,
		}),
	})
	if err != nil {
		return err
	}
	for _, n := range ns.Items {
		delete(n.Labels, consts.OdigosInstrumentationLabel)
		_, err := client.CoreV1().Namespaces().Update(ctx, &n, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func uninstallDeployments(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	list, err := client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return err
	}

	for _, i := range list.Items {
		err = client.AppsV1().Deployments(ns).Delete(ctx, i.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}



func uninstallServices(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	list, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return err
	}

	for _, i := range list.Items {
		err = client.CoreV1().Services(ns).Delete(ctx, i.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func uninstallDaemonSets(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	list, err := client.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return err
	}

	for _, i := range list.Items {
		err = client.AppsV1().DaemonSets(ns).Delete(ctx, i.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func uninstallConfigMaps(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	list, err := client.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return err
	}

	for _, i := range list.Items {
		err = client.CoreV1().ConfigMaps(ns).Delete(ctx, i.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func uninstallCRDs(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	list, err := client.ApiExtensions.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return err
	}

	for _, i := range list.Items {
		err = client.ApiExtensions.ApiextensionsV1().CustomResourceDefinitions().Delete(ctx, i.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func uninstallRBAC(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	list, err := client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return err
	}

	for _, i := range list.Items {
		err = client.RbacV1().ClusterRoles().Delete(ctx, i.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	list2, err := client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return err
	}

	for _, i := range list2.Items {
		err = client.RbacV1().ClusterRoleBindings().Delete(ctx, i.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func uninstallSecrets(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	list, err := client.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return err
	}

	for _, i := range list.Items {
		err = client.CoreV1().Secrets(ns).Delete(ctx, i.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func uninstallNamespace(ctx context.Context, cmd *cobra.Command, client *kube.Client, ns string) error {
	err := client.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
	return err
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().Bool("yes", false, "Skip the confirmation prompt")
}
