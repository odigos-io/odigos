package pro

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func NewEnterpriseRegistryPullSecret(namespace, token string) (*corev1.Secret, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("odigos:%s", token)))
	dockerConfigJSON, err := json.Marshal(map[string]any{
		"auths": map[string]any{
			k8sconsts.OdigosImagePrefix: map[string]string{
				"username": "odigos",
				"password": token,
				"auth":     auth,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enterprise registry docker config: %w", err)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosEnterpriseRegistryPullSecretName,
			Namespace: namespace,
			Labels:    EnterpriseRegistryPullSecretLabels(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}, nil
}

// EnterpriseRegistryPullSecretLabels returns labels matching Helm-managed pull secrets.
func EnterpriseRegistryPullSecretLabels() map[string]string {
	return map[string]string{
		k8sconsts.OdigosSystemLabelKey: k8sconsts.OdigosSystemLabelValue,
	}
}

// CopyImagePullSecretsIfMissing copies each named pull secret from sourceNs into destNs
// when it is not already present. Missing source secrets are skipped.
// secretNames should come from effective config (user secrets and enterprise, if any).
func CopyImagePullSecretsIfMissing(
	ctx context.Context, r client.Reader, w client.Writer, sourceNs, destNs string, secretNames []string,
) error {
	if sourceNs == "" || destNs == "" || sourceNs == destNs {
		return nil
	}

	var errs error
	for _, name := range secretNames {
		if name == "" {
			continue
		}
		if err := copyOneImagePullSecret(ctx, r, w, sourceNs, destNs, name); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

// copyOneImagePullSecret creates dest if missing. It does not GET dest first:
// instrumentor can only get (and cache) secrets in the Odigos namespace, so a dest
// GET would need cluster-wide secrets get. Create + AlreadyExists is the
// already-copied path and also covers create races.
func copyOneImagePullSecret(ctx context.Context, r client.Reader, w client.Writer, sourceNs, destNs, secretName string) error {
	var src corev1.Secret
	err := r.Get(ctx, client.ObjectKey{Namespace: sourceNs, Name: secretName}, &src)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to read image pull secret %q in namespace %q: %w", secretName, sourceNs, err)
	}

	data := make(map[string][]byte, len(src.Data))
	for k, v := range src.Data {
		data[k] = append([]byte(nil), v...)
	}

	dest := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: destNs,
			Labels:    EnterpriseRegistryPullSecretLabels(),
		},
		Type: src.Type,
		Data: data,
	}
	err = w.Create(ctx, dest)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to copy image pull secret %q to namespace %q: %w", secretName, destNs, err)
	}
	return nil
}
