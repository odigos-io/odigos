package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	useDefault          bool
	updateRemoteFlag    bool
)

var proCmd = &cobra.Command{
	Use:   "pro",
	Short: "Manage Odigos onprem tier for enterprise users",
	Long:  `The pro command provides various operations and functionalities specifically designed for enterprise users. Use this command to access advanced features and manage your pro account.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)
		ns, err := resources.GetOdigosNamespace(client, ctx)
		if resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Println("\033[31mERROR\033[0m no odigos installation found in the current cluster")
			os.Exit(1)
		} else if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already installed: %s\n", err)
			os.Exit(1)
		}
		onPremToken := cmd.Flag("onprem-token").Value.String()
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
odigos pro --onprem-token ${ODIGOS_TOKEN}
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
with the previous release of Odigos.

Note that updating offsets does not guarantee instrumentation for libraries with significant changes that
require an update to Odigos. See docs for more info: https://docs.odigos.io/instrumentations/golang/ebpf#about-go-offsets
`,
	Example: `
# Pull the latest offsets and restart Odiglet
odigos pro update-offsets

# Revert to using the default offsets data shipped with Odigos
odigos pro update-offsets --default
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)
		ns := cmd.Flag("namespace").Value.String()

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

		cm, err := client.Clientset.CoreV1().ConfigMaps(ns).Get(ctx, k8sconsts.GoOffsetsConfigMap, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to get Go offsets ConfigMap: %s", err))
				os.Exit(1)
			}
			cm = &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k8sconsts.GoOffsetsConfigMap,
					Namespace: ns,
				},
				Data: map[string]string{
					k8sconsts.GoOffsetsFileName: string(data),
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

			cm.Data[k8sconsts.GoOffsetsFileName] = string(data)
			_, err = client.Clientset.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
			if err != nil {
				fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to update Go offsets ConfigMap: %s", err))
				os.Exit(1)
			}
		}

		ds := updateOdigletDaemonSet(ctx, client, ns, useDefault)
		_, err = client.Clientset.AppsV1().DaemonSets(ns).Update(ctx, ds, metav1.UpdateOptions{})
		if err != nil {
			fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to update Odiglet DaemonSet: %s", err))
			os.Exit(1)
		}

		fmt.Println("Updated Go Offsets.")
	},
}

func getLatestOffsets(revert bool) ([]byte, error) {
	if revert {
		return []byte{}, nil
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

func updateOdigletDaemonSet(ctx context.Context, client *kube.Client, ns string, revert bool) *appsv1.DaemonSet {
	ds, err := client.Clientset.AppsV1().DaemonSets(ns).Get(ctx, k8sconsts.OdigletDaemonSetName, metav1.GetOptions{})
	if err != nil {
		fmt.Println(fmt.Sprintf("\033[31mERROR\033[0m Unable to get Odiglet DaemonSet: %s", err))
		os.Exit(1)
	}

	// Add offsets volume (if not already exist)
	volumes := ds.Spec.Template.Spec.Volumes
	if volumes == nil {
		volumes = make([]v1.Volume, 0)
	}
	addVolume := true
	for _, vol := range volumes {
		if vol.Name == k8sconsts.GoOffsetsConfigMap {
			addVolume = false
			break
		}
	}
	if addVolume {
		volumes = append(volumes, v1.Volume{Name: k8sconsts.GoOffsetsConfigMap,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: k8sconsts.GoOffsetsConfigMap,
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
		if vm.Name == k8sconsts.GoOffsetsConfigMap {
			addVolumeMount = false
			break
		}
	}
	if addVolumeMount {
		volumeMounts = append(volumeMounts, v1.VolumeMount{Name: k8sconsts.GoOffsetsConfigMap, MountPath: k8sconsts.OffsetFileMountPath})
		ds.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
	}

	// Add offsets Env Var (if not already exist)
	envVars := ds.Spec.Template.Spec.Containers[0].Env
	if envVars == nil {
		envVars = make([]v1.EnvVar, 0)
	}
	addEnvVar := true
	for _, env := range envVars {
		if env.Name == k8sconsts.GoOffsetsEnvVar {
			addEnvVar = false
			break
		}
	}
	if addEnvVar {
		envVars = append(envVars, v1.EnvVar{Name: k8sconsts.GoOffsetsEnvVar, Value: k8sconsts.OffsetFileMountPath + "/" + k8sconsts.GoOffsetsFileName})
		ds.Spec.Template.Spec.Containers[0].Env = envVars
	}
	return ds
}

func init() {
	rootCmd.AddCommand(proCmd)

	proCmd.Flags().String("onprem-token", "", "On-prem token for Odigos")
	proCmd.MarkFlagRequired("onprem-token")
	proCmd.PersistentFlags().BoolVarP(&updateRemoteFlag, "remote", "r", false, "use odigos ui service in the cluster to update the onprem token")

	proCmd.AddCommand(offsetsCmd)
	offsetsCmd.Flags().StringVarP(&namespaceFlag, "namespace", "n", consts.DefaultOdigosNamespace, "target k8s namespace for Odigos installation")
	offsetsCmd.Flags().BoolVar(&useDefault, "default", false, "revert to using the default offsets data shipped with the current version of Odigos")
}
