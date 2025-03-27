package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common/consts"
)

var (
	offsetFileMountPath = "/offsets"
	useDefault          bool
)

var offsetsCmd = &cobra.Command{
	Use:   "update-offsets",
	Short: "Update Odiglet to use the latest available Go instrumentation offsets",
	Long: `This command pulls the latest available Go struct and field offsets information from Odigos.
It stores this data in a ConfigMap in the Odigos Namespace and updates the Odiglet DaemonSet to mount it.

Use this command when instrumenting apps that depend on very new dependencies that aren't currently supported
with the previous release of Odigos.

Note that updating offsets does not guarantee instrumentation for libraries with significant changes that
require an update to Odigos. See docs for more info: https://docs.odigos.io/instrumentations/golang/ebpf#about-go-offsets
`,
	Example: `
# Pull the latest offsets and restart Odiglet
odigos update-offsets

# Revert to using the default offsets data shipped with Odigos
odigos update-offsets --default
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)
		ns := cmd.Flag("namespace").Value.String()

		resp, err := http.Get(consts.GoOffsetsPublicURL)
		if err != nil {
			fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Cannot get latest offsets: %s", err))
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Cannot get latest offsets: %d", resp.StatusCode))
			os.Exit(1)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to read response body: %s", err))
			os.Exit(1)
		}

		ds := updateOdigletDaemonSet(ctx, client, ns, useDefault)

		if useDefault {
			// Update odiglet to remove references to configmap (before we delete the configmap)
			_, err = client.Clientset.AppsV1().DaemonSets(ns).Update(ctx, ds, metav1.UpdateOptions{})
			if err != nil {
				fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to update Odiglet DaemonSet: %s", err))
				os.Exit(1)
			}

			// Delete offsets configmap
			err = client.Clientset.CoreV1().ConfigMaps(ns).Delete(ctx, consts.GoOffsetsConfigMap, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to delete Go offsets ConfigMap: %s", err))
				os.Exit(1)
			}

			fmt.Println("Reverted to using default Go offsets data.")
		} else {
			// Create or update the offsets configmap (before we update the odiglet to refer to it)
			cm, err := client.Clientset.CoreV1().ConfigMaps(ns).Get(ctx, consts.GoOffsetsConfigMap, metav1.GetOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to get Go offsets ConfigMap: %s", err))
					os.Exit(1)
				}
				cm = &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      consts.GoOffsetsConfigMap,
						Namespace: ns,
					},
					Data: map[string]string{
						consts.GoOffsetsFileName: string(data),
					},
				}

				cm, err = client.Clientset.CoreV1().ConfigMaps(ns).Create(ctx, cm, metav1.CreateOptions{})
				if err != nil {
					fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to create Go offsets ConfigMap: %s", err))
					os.Exit(1)
				}
			} else {
				if cm.Data == nil {
					cm.Data = make(map[string]string)
				}

				cm.Data[consts.GoOffsetsFileName] = string(data)
				_, err = client.Clientset.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
				if err != nil {
					fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to update Go offsets ConfigMap: %s", err))
					os.Exit(1)
				}
			}

			_, err = client.Clientset.AppsV1().DaemonSets(ns).Update(ctx, ds, metav1.UpdateOptions{})
			if err != nil {
				fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to update Odiglet DaemonSet: %s", err))
				os.Exit(1)
			}
			fmt.Println("Updated Go offsets to latest.")
		}
	},
}

func updateOdigletDaemonSet(ctx context.Context, client *kube.Client, ns string, revert bool) *appsv1.DaemonSet {
	ds, err := client.Clientset.AppsV1().DaemonSets(ns).Get(ctx, k8sconsts.OdigletDaemonSetName, metav1.GetOptions{})
	if err != nil {
		fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to get Odiglet DaemonSet: %s", err))
		os.Exit(1)
	}

	if revert {
		// Remove offsets file env var (if present)
		envVars := ds.Spec.Template.Spec.Containers[0].Env
		for i, env := range envVars {
			if env.Name == consts.GoOffsetsEnvVar {
				ds.Spec.Template.Spec.Containers[0].Env = append(envVars[:i], envVars[i+1:]...)
				break
			}
		}

		// Remove offsets volumemount (if present)
		volumeMounts := ds.Spec.Template.Spec.Containers[0].VolumeMounts
		for i, vm := range volumeMounts {
			if vm.Name == consts.GoOffsetsConfigMap {
				ds.Spec.Template.Spec.Containers[0].VolumeMounts = append(volumeMounts[:i], volumeMounts[i+1:]...)
				break
			}
		}

		// Remove offsets volume (if present)
		volumes := ds.Spec.Template.Spec.Volumes
		for i, vol := range volumes {
			if vol.Name == consts.GoOffsetsConfigMap {
				ds.Spec.Template.Spec.Volumes = append(volumes[:i], volumes[i+1:]...)
				break
			}
		}
	} else {
		// Add offsets volume (if not already exist)
		volumes := ds.Spec.Template.Spec.Volumes
		if volumes == nil {
			volumes = make([]v1.Volume, 0)
		}
		addVolume := true
		for _, vol := range volumes {
			if vol.Name == consts.GoOffsetsConfigMap {
				addVolume = false
				break
			}
		}
		if addVolume {
			volumes = append(volumes, v1.Volume{Name: consts.GoOffsetsConfigMap,
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: consts.GoOffsetsConfigMap,
						},
					},
				},
			})
			ds.Spec.Template.Spec.Volumes = volumes
		}

		// Add offsets volume mounnt (if not already exist)
		volumeMounts := ds.Spec.Template.Spec.Containers[0].VolumeMounts
		if volumeMounts == nil {
			volumeMounts = make([]v1.VolumeMount, 0)
		}
		addVolumeMount := true
		for _, vm := range volumeMounts {
			if vm.Name == consts.GoOffsetsConfigMap {
				addVolumeMount = false
				break
			}
		}
		if addVolumeMount {
			volumeMounts = append(volumeMounts, v1.VolumeMount{Name: consts.GoOffsetsConfigMap, MountPath: offsetFileMountPath})
			ds.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
		}

		// Add offsets Env Var (if not already exist)
		envVars := ds.Spec.Template.Spec.Containers[0].Env
		if envVars == nil {
			envVars = make([]v1.EnvVar, 0)
		}
		addEnvVar := true
		for _, env := range envVars {
			if env.Name == consts.GoOffsetsEnvVar {
				addEnvVar = false
				break
			}
		}
		if addEnvVar {
			envVars = append(envVars, v1.EnvVar{Name: consts.GoOffsetsEnvVar, Value: offsetFileMountPath + "/" + consts.GoOffsetsFileName})
			ds.Spec.Template.Spec.Containers[0].Env = envVars
		}
	}
	return ds
}

func init() {
	rootCmd.AddCommand(offsetsCmd)
	offsetsCmd.Flags().StringVarP(&namespaceFlag, "namespace", "n", consts.DefaultOdigosNamespace, "target k8s namespace for Odigos installation")
	offsetsCmd.Flags().BoolVar(&useDefault, "default", false, "revert to using the default offsets data shipped with Odigos")
}
