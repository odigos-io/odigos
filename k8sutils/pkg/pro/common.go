package pro

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/odigosauth"
)

func UpdateOdigosToken(ctx context.Context, client kubernetes.Interface, namespace string, onPremToken string) error {
	if _, err := odigosauth.ValidateToken(onPremToken); err != nil {
		return err
	}
	if err := updateSecretToken(ctx, client, namespace, onPremToken); err != nil {
		return fmt.Errorf("failed to update secret token: %w", err)
	}
	if err := EnsureEnterpriseRegistryPullSecret(ctx, client, namespace, onPremToken); err != nil {
		return fmt.Errorf("failed to update enterprise registry pull secret: %w", err)
	}
	if err := odigletRolloutTrigger(ctx, client, namespace); err != nil {
		return fmt.Errorf("failed to trigger odiglet rollout: %w", err)
	}
	return nil
}

func updateSecretToken(ctx context.Context, client kubernetes.Interface, namespace string, onPremToken string) error {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, k8sconsts.OdigosProSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("tokens are not available in the community version of Odigos. Please contact Odigos team to inquire about pro version")
		}
		return err
	}
	secret.Data[k8sconsts.OdigosOnpremTokenSecretKey] = []byte(onPremToken)

	_, err = client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// ShouldUseEnterpriseRegistryPullSecret reports whether Odigos should manage the
// registry.odigos.io pull secret (skipped when a custom imagePrefix is configured).
func ShouldUseEnterpriseRegistryPullSecret(ctx context.Context, client kubernetes.Interface, namespace string) (bool, error) {
	return shouldUseEnterpriseRegistryPullSecret(ctx, client, namespace)
}

// EnsureEnterpriseRegistryPullSecret creates or updates the registry pull secret for on-prem installs.
// It is a no-op when a custom imagePrefix is configured.
func EnsureEnterpriseRegistryPullSecret(ctx context.Context, client kubernetes.Interface, namespace, onPremToken string) error {
	useEnterprisePullSecret, err := shouldUseEnterpriseRegistryPullSecret(ctx, client, namespace)
	if err != nil {
		return err
	}
	if !useEnterprisePullSecret {
		return nil
	}

	pullSecret, err := NewEnterpriseRegistryPullSecret(namespace, onPremToken)
	if err != nil {
		return fmt.Errorf("failed to build enterprise registry pull secret: %w", err)
	}

	existing, err := client.CoreV1().Secrets(namespace).Get(ctx, k8sconsts.OdigosEnterpriseRegistryPullSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = client.CoreV1().Secrets(namespace).Create(ctx, pullSecret, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create enterprise registry pull secret %q: %w", k8sconsts.OdigosEnterpriseRegistryPullSecretName, err)
			}
			return nil
		}
		return fmt.Errorf("failed to read enterprise registry pull secret %q: %w", k8sconsts.OdigosEnterpriseRegistryPullSecretName, err)
	}

	existing.Type = pullSecret.Type
	existing.Data = pullSecret.Data
	existing.Labels = pullSecret.Labels
	_, err = client.CoreV1().Secrets(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update enterprise registry pull secret %q: %w", k8sconsts.OdigosEnterpriseRegistryPullSecretName, err)
	}
	return nil
}

func odigletRolloutTrigger(ctx context.Context, client kubernetes.Interface, namespace string) error {
	daemonSet, err := client.AppsV1().DaemonSets(namespace).Get(ctx, "odiglet", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("\033[31mERROR\033[0m failed to get odiglet DaemonSet in namespace %s: %v", namespace, err)
	}

	// Modify the DaemonSet spec.template to trigger a rollout
	if daemonSet.Spec.Template.Annotations == nil {
		daemonSet.Spec.Template.Annotations = make(map[string]string)
	}
	daemonSet.Spec.Template.Annotations[odigosconsts.RolloutTriggerAnnotation] = time.Now().Format(time.RFC3339)

	_, err = client.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	if err != nil {
		command := fmt.Sprintf("kubectl rollout restart daemonset odiglet -n %s", daemonSet.Namespace)
		return fmt.Errorf("failed to restart Odiglets: %w. To trigger a restart manually, run the following command: %s", err, command)
	}
	return nil
}

type TokenPayload struct {
	OnpremToken string `json:"token"`
}

// UsesOdigosRegistry reports whether images pull from the default Odigos registry.
func UsesOdigosRegistry(config *common.OdigosConfiguration) bool {
	if config.ImagePrefix != "" {
		return config.ImagePrefix == k8sconsts.OdigosImagePrefix
	}
	return !config.OpenshiftEnabled
}

func shouldUseEnterpriseRegistryPullSecret(ctx context.Context, client kubernetes.Interface, namespace string) (bool, error) {
	configMap, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, odigosconsts.OdigosConfigurationName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	configYAML, ok := configMap.Data[odigosconsts.OdigosConfigurationFileName]
	if !ok || configYAML == "" {
		return true, nil
	}

	var config common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		return false, fmt.Errorf("failed to parse odigos configuration: %w", err)
	}

	return UsesOdigosRegistry(&config), nil
}
