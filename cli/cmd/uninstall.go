package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/k8sutils/pkg/envoverwrite"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/confirm"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/labels"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
	"go.uber.org/multierr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8slabels "k8s.io/apimachinery/pkg/labels"

	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use: "uninstall",
	Short: `Revert all the changes made by the ` + "`odigos install`" + ` command.
This command will uninstall Odigos from your cluster. It will delete all Odigos objects
and rollback any metadata changes made to your objects.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		ns, err := resources.GetOdigosNamespace(client, ctx)
		if err != nil && !resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already uninstalled: %s\n", err)
			os.Exit(1)
		}
		odigosNsFound := !resources.IsErrNoOdigosNamespaceFound(err)

		if odigosNsFound {
			if !cmd.Flag("yes").Changed {
				fmt.Printf("About to uninstall Odigos from namespace %s\n", ns)
				confirmed, err := confirm.Ask("Are you sure?")
				if err != nil || !confirmed {
					fmt.Println("Aborting uninstall")
					return
				}
			}

			// delete all sources, and wait for the pods to rollout without instrumentation
			// this is done before the instrumentor is removed, to ensure that the instrumentation is removed
			err := removeAllSources(ctx, client)
			if err != nil {
				fmt.Printf("\033[31mERROR\033[0m Failed to remove all sources: %s\n", err)
				os.Exit(1)
			}
			if !cmd.Flag("no-wait").Changed {
				waitForPodsToRolloutWithoutInstrumentation(ctx, client)
			}

			UninstallOdigosResources(ctx, client, ns)

			// The CLI is running in Kubernetes via a Helm chart [pre-delete hook] to clean up Odigos resources.
			// Deleting the namespace during uninstallation will cause Helm to fail due to the loss of the release state.
			if !k8sutils.IsRunningInKubernetes() {
				createKubeResourceWithLogging(ctx, fmt.Sprintf("Uninstalling Namespace %s", ns),
					client, ns, k8sconsts.OdigosSystemLabelKey, uninstallNamespace)

				waitForNamespaceDeletion(ctx, client, ns)
			}

		} else {
			fmt.Println("Odigos is not installed in any namespace. cleaning up any other Odigos resources that might be left in the cluster...")
		}

		UninstallClusterResources(ctx, client, ns)

		fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos uninstalled.\n")
	},
	Example: `
# Uninstall Odigos open-source or cloud from the cluster in your kubeconfig active context.
odigos uninstall

# Uninstall Odigos without confirmation
odigos uninstall --yes

# Uninstall Odigos without waiting for pods to rollout without instrumentation
odigos uninstall --no-wait

# Uninstall Odigos cloud from a specific cluster
odigos uninstall --kubeconfig <path-to-kubeconfig>

# Install a fresh setup of Odigos
odigos uninstall
odigos install
`,
}

// UninstallOdigosResources removes Odigos system resources from the Odigos namespace,
// such as component deployments, daemonsets, configmaps, services, RBAC, and secrets.
func UninstallOdigosResources(ctx context.Context, client *kube.Client, ns string) {
	createKubeResourceWithLogging(ctx, "Uninstalling Odigos Deployments",
		client, ns, k8sconsts.OdigosSystemLabelKey, uninstallDeployments)
	createKubeResourceWithLogging(ctx, "Uninstalling Odigos DaemonSets",
		client, ns, k8sconsts.OdigosSystemLabelKey, uninstallDaemonSets)
	createKubeResourceWithLogging(ctx, "Uninstalling Odigos ConfigMaps",
		client, ns, k8sconsts.OdigosSystemLabelKey, uninstallConfigMaps)
	createKubeResourceWithLogging(ctx, "Uninstalling Odigos Services",
		client, ns, k8sconsts.OdigosSystemLabelKey, uninstallServices)
	createKubeResourceWithLogging(ctx, "Uninstalling Odigos RBAC",
		client, ns, k8sconsts.OdigosSystemLabelKey, uninstallRBAC)
	createKubeResourceWithLogging(ctx, "Uninstalling Odigos Secrets",
		client, ns, k8sconsts.OdigosSystemLabelKey, uninstallSecrets)
	// Without deleting the mutating and validating webhook configurations the CRDs cannot be deleted.
	// E.g deleting "Sources" at later stage will fail as the CRD is still in use.
	createKubeResourceWithLogging(ctx, "Uninstalling Odigos MutatingWebhookConfigurations",
		client, ns, k8sconsts.OdigosSystemLabelKey, uninstallMutatingWebhookConfigs)

	createKubeResourceWithLogging(ctx, "Uninstalling Odigos ValidatingWebhookConfigurations",
		client, ns, k8sconsts.OdigosSystemLabelKey, uninstallValidatingWebhookConfigs)
}

// UninstallClusterResources removes cluster-wide Odigos resources, such as node labels,
// pod and namespace changes, CRDs, and webhook configurations.
func UninstallClusterResources(ctx context.Context, client *kube.Client, ns string) {
	createKubeResourceWithLogging(ctx, "Cleaning up Odigos node labels",
		client, ns, k8sconsts.OdigosSystemLabelKey, cleanupNodeOdigosLabels)

	l := log.Print("Rolling back odigos changes to pods")
	err := rollbackPodChanges(ctx, client)
	if err != nil {
		l.Warn(err.Error())
	} else {
		l.Success()
	}

	l = log.Print("Rolling back odigos changes to namespaces")
	err = rollbackNamespaceChanges(ctx, client)
	if err != nil {
		l.Warn(err.Error())
	} else {
		l.Success()
	}

	createKubeResourceWithLogging(ctx, "Uninstalling Odigos CRDs",
		client, ns, k8sconsts.OdigosSystemLabelKey, uninstallCRDs)

}

func waitForPodsToRolloutWithoutInstrumentation(ctx context.Context, client *kube.Client) {
	instrumentedPodReq, _ := k8slabels.NewRequirement(k8sconsts.OdigosAgentsMetaHashLabel, selection.Exists, []string{})
	fmt.Printf("Waiting for pods to rollout without instrumentation... this might take a while\n")

	pollErr := wait.PollUntilContextTimeout(ctx, 10*time.Second, 5*time.Minute, true, func(innerCtx context.Context) (bool, error) {
		pods, err := client.CoreV1().Pods("").List(innerCtx, metav1.ListOptions{
			LabelSelector: instrumentedPodReq.String(),
		})
		if err != nil {
			return false, err
		}
		if len(pods.Items) == 0 {
			l := log.Print("All pods rolled out without instrumentation")
			l.Success()
			return true, nil
		}
		log.Print(fmt.Sprintf("\tWaiting for %d pods to rollout without instrumentation...\n", len(pods.Items)))
		return false, nil
	})

	if pollErr != nil {
		if errors.Is(pollErr, context.DeadlineExceeded) {
			fmt.Printf("\033[33m!\tWARN\033[0m deadline exceeded for waiting pods to roll out cleanly, consider re-running uninstall or rollout the un cleaned workloads\n")
		}
		if errors.Is(pollErr, context.Canceled) {
			fmt.Printf("\033[33m!\tWARN\033[0m canceled while waiting pods to roll out cleanly\n")
		}
	}
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
	var errs error

	deps, err := client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, dep := range deps.Items {
		jsonPatchPayloadBytes, err := getWorkloadRolloutJsonPatch(&dep, &dep.Spec.Template)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		if len(jsonPatchPayloadBytes) > 0 {
			_, err = client.AppsV1().Deployments(dep.Namespace).Patch(ctx, dep.Name, types.JSONPatchType, jsonPatchPayloadBytes, metav1.PatchOptions{})
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	ss, err := client.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, s := range ss.Items {
		jsonPatchPayloadBytes, err := getWorkloadRolloutJsonPatch(&s, &s.Spec.Template)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		if len(jsonPatchPayloadBytes) > 0 {
			_, err = client.AppsV1().StatefulSets(s.Namespace).Patch(ctx, s.Name, types.JSONPatchType, jsonPatchPayloadBytes, metav1.PatchOptions{})
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	dd, err := client.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, d := range dd.Items {
		jsonPatchPayloadBytes, err := getWorkloadRolloutJsonPatch(&d, &d.Spec.Template)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		if len(jsonPatchPayloadBytes) > 0 {
			_, err = client.AppsV1().DaemonSets(d.Namespace).Patch(ctx, d.Name, types.JSONPatchType, jsonPatchPayloadBytes, metav1.PatchOptions{})
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	return errs
}

// with json patch, "/" is used to separate levels in the JSON structure.
// to escape it, we replace it with "~1"
func jsonPatchEscapeKey(key string) string {
	return strings.Replace(key, "/", "~1", 1)
}

func getWorkloadRolloutJsonPatch(obj kube.Object, pts *v1.PodTemplateSpec) ([]byte, error) {
	patchOperations := []map[string]interface{}{}

	// Remove odigos instrumentation label
	if obj.GetLabels() != nil {
		if _, found := obj.GetLabels()[consts.OdigosInstrumentationLabel]; found {
			patchOperations = append(patchOperations, map[string]interface{}{
				"op":   "remove",
				"path": "/metadata/labels/" + consts.OdigosInstrumentationLabel,
			})
		}
	}
	odigosInjectInstrumentationLabel := "odigos.io/inject-instrumentation"
	if _, found := pts.ObjectMeta.Labels[odigosInjectInstrumentationLabel]; found {
		patchOperations = append(patchOperations, map[string]interface{}{
			"op":   "remove",
			"path": "/spec/template/metadata/labels/" + jsonPatchEscapeKey(odigosInjectInstrumentationLabel),
		})
	}

	// remove odigos reported name annotation
	if obj.GetAnnotations() != nil {
		if _, found := obj.GetAnnotations()[consts.OdigosReportedNameAnnotation]; found {
			patchOperations = append(patchOperations, map[string]interface{}{
				"op":   "remove",
				"path": "/metadata/annotations/" + jsonPatchEscapeKey(consts.OdigosReportedNameAnnotation),
			})
		}
	}

	// read the original env vars (of the manifest) from the annotation
	manifestEnvOriginal, err := envoverwrite.NewOrigWorkloadEnvValues(obj.GetAnnotations())
	if err != nil {
		fmt.Println("Failed to get original env vars from annotation: ", err)
		manifestEnvOriginal = &envoverwrite.OrigWorkloadEnvValues{}
	}

	var origManifestEnv map[string]map[string]string
	if obj.GetAnnotations() != nil {
		manifestEnvAnnotation, ok := obj.GetAnnotations()[consts.ManifestEnvOriginalValAnnotation]
		if ok {
			err := json.Unmarshal([]byte(manifestEnvAnnotation), &origManifestEnv)
			if err != nil {
				fmt.Printf("Failed to unmarshal original env vars from annotation: %s. %s: %s\n", err, obj.GetName(), obj.GetNamespace())
			}
		}
	}

	// remove odigos instrumentation device from containers
	for iContainer, c := range pts.Spec.Containers {
		if c.Resources.Limits != nil {
			for val := range c.Resources.Limits {
				if strings.HasPrefix(val.String(), common.OdigosResourceNamespace) {
					patchOperations = append(patchOperations, map[string]interface{}{
						"op":   "remove",
						"path": fmt.Sprintf("/spec/template/spec/containers/%d/resources/limits/%s", iContainer, jsonPatchEscapeKey(val.String())),
					})
				}
			}
		}

		for envName, originalEnvValue := range manifestEnvOriginal.GetContainerStoredEnvs(c.Name) {
			// find the index of the env var in the env array:
			iEnv := -1
			for i, env := range c.Env {
				if env.Name == envName {
					iEnv = i
					break
				}
			}
			if iEnv == -1 {
				return nil, fmt.Errorf("env var %s not found in container %s", envName, c.Name)
			}

			if originalEnvValue == nil {
				// originally the value was absent, so we remove it
				patchOperations = append(patchOperations, map[string]interface{}{
					"op":   "remove",
					"path": fmt.Sprintf("/spec/template/spec/containers/%d/env/%d", iContainer, iEnv),
				})
			} else {
				// revert the env var to its original value
				sanitizedEnvVar := envOverwrite.CleanupEnvValueFromOdigosAdditions(envName, *originalEnvValue)
				patchOperations = append(patchOperations, map[string]interface{}{
					"op":    "replace",
					"path":  fmt.Sprintf("/spec/template/spec/containers/%d/env/%d/value", iContainer, iEnv),
					"value": sanitizedEnvVar,
				})
			}
		}
	}

	// remove the env var original value annotation
	if obj.GetAnnotations() != nil {
		if _, found := obj.GetAnnotations()[consts.ManifestEnvOriginalValAnnotation]; found {
			patchOperations = append(patchOperations, map[string]interface{}{
				"op":   "remove",
				"path": "/metadata/annotations/" + jsonPatchEscapeKey(consts.ManifestEnvOriginalValAnnotation),
			})
		}
	}

	return json.Marshal(patchOperations)
}

func rollbackNamespaceChanges(ctx context.Context, client *kube.Client) error {
	ns, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	var errs error
	for _, n := range ns.Items {
		if n.GetLabels() == nil {
			continue
		}
		_, found := n.GetLabels()[consts.OdigosInstrumentationLabel]
		if !found {
			continue
		}
		_, err := client.CoreV1().Namespaces().Patch(ctx, n.Name, types.JSONPatchType, []byte(`[{"op": "remove", "path": "/metadata/labels/`+consts.OdigosInstrumentationLabel+`"}]`), metav1.PatchOptions{})
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return errs
}

func uninstallDeployments(ctx context.Context, client *kube.Client, ns, _ string) error {
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

func uninstallServices(ctx context.Context, client *kube.Client, ns, _ string) error {
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

func uninstallDaemonSets(ctx context.Context, client *kube.Client, ns, _ string) error {
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

func uninstallConfigMaps(ctx context.Context, client *kube.Client, ns, _ string) error {
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

func removeAllSources(ctx context.Context, client *kube.Client) error {
	l := log.Print("Removing Odigos Sources...")
	sources, err := client.OdigosClient.Sources("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var deleteErr error
	for _, i := range sources.Items {
		e := client.OdigosClient.Sources(i.Namespace).Delete(ctx, i.Name, metav1.DeleteOptions{})
		if e != nil {
			deleteErr = errors.Join(deleteErr, e)
		}
	}

	if deleteErr != nil {
		return deleteErr
	}

	// make sure all sources are deleted, this is required regardless of the --no-wait flag,
	// in order to make sure the Source CRD can be deleted later in the uninstall process.
	// failing to remove all the sources may cause the Source CRD to not get removed - since kubernetes
	// has a finalizer on a CRD, waiting for all the CR instances to be deleted before removing the CRD.
	pollErr := wait.PollUntilContextTimeout(ctx, 5*time.Second, 1*time.Minute, true, func(innerCtx context.Context) (bool, error) {
		sources, err := client.OdigosClient.Sources("").List(innerCtx, metav1.ListOptions{
			Limit: 1,
		})
		if err != nil {
			return false, err
		}
		if len(sources.Items) == 0 {
			l.Success()
			return true, nil
		}
		return false, nil
	})

	var returnErr error
	if pollErr != nil {
		if errors.Is(pollErr, context.DeadlineExceeded) {
			returnErr = fmt.Errorf("deadline exceeded for waiting sources to be deleted\n")
		} else if errors.Is(pollErr, context.Canceled) {
			returnErr = fmt.Errorf("canceled while waiting sources to be deleted\n")
		} else {
			returnErr = fmt.Errorf("error while waiting for sources to be deleted: %s\n", err)
		}
	}
	return returnErr
}

func uninstallCRDs(ctx context.Context, client *kube.Client, ns string, _ string) error {
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

func uninstallMutatingWebhookConfigs(ctx context.Context, client *kube.Client, ns, _ string) error {
	list, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return err
	}

	for _, webhook := range list.Items {
		err = client.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, webhook.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func uninstallValidatingWebhookConfigs(ctx context.Context, client *kube.Client, ns, _ string) error {
	list, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return err
	}

	for _, webhook := range list.Items {
		err = client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, webhook.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func uninstallRBAC(ctx context.Context, client *kube.Client, ns, _ string) error {
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

func cleanupNodeOdigosLabels(ctx context.Context, client *kube.Client, ns, _ string) error {
	nodeSet := make(map[string]struct{})

	// Step 1: Get OSS nodes
	ossNodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: k8sconsts.OdigletOSSInstalledLabel,
	})
	if err != nil {
		return fmt.Errorf("failed to list nodes with %s: %w", k8sconsts.OdigletOSSInstalledLabel, err)
	}
	for _, node := range ossNodes.Items {
		nodeSet[node.Name] = struct{}{}
	}

	// Step 2: Get Enterprise nodes
	enterpriseNodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: k8sconsts.OdigletEnterpriseInstalledLabel,
	})
	if err != nil {
		return fmt.Errorf("failed to list nodes with %s: %w", k8sconsts.OdigletEnterpriseInstalledLabel, err)
	}
	for _, node := range enterpriseNodes.Items {
		nodeSet[node.Name] = struct{}{}
	}

	for nodeName := range nodeSet {
		patchData := map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": map[string]interface{}{
					// Setting to `nil` removes the labels if exists, otherwise will ignore
					k8sconsts.OdigletOSSInstalledLabel:        nil,
					k8sconsts.OdigletEnterpriseInstalledLabel: nil,
				},
			},
		}

		patchBytes, err := json.Marshal(patchData)
		if err != nil {
			return fmt.Errorf("failed to marshal patch data: %w", err)
		}

		_, err = client.CoreV1().Nodes().Patch(ctx, nodeName, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("failed to patch node %s: %w", nodeName, err)
		}
	}

	return nil
}

func uninstallSecrets(ctx context.Context, client *kube.Client, ns, _ string) error {
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

func uninstallNamespace(ctx context.Context, client *kube.Client, ns, _ string) error {
	err := client.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
	return err
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().Bool("yes", false, "skip the confirmation prompt")
	uninstallCmd.Flags().Bool("no-wait", false, "skip waiting for pods to rollout without instrumentation")
}
