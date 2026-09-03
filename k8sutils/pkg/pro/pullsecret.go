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

// ImagePullSecretNamesToCopy returns the pull secret names that should be copied
// into instrumented namespaces. It includes user-configured secrets and the
// enterprise registry secret when that secret exists (copy is a no-op if it does not).
func ImagePullSecretNamesToCopy(configured []string) []string {
	seen := make(map[string]struct{}, len(configured)+1)
	names := make([]string, 0, len(configured)+1)
	add := func(name string) {
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for _, name := range configured {
		add(name)
	}
	add(k8sconsts.OdigosEnterpriseRegistryPullSecretName)
	return names
}

// CopyImagePullSecretsIfMissing copies each named pull secret from sourceNs into destNs
// when it is not already present. Missing source secrets are skipped.
// Returns the secret names that destNs should reference on pods.
func CopyImagePullSecretsIfMissing(
	ctx context.Context, r client.Reader, w client.Writer, sourceNs, destNs string, secretNames []string,
) ([]string, error) {
	if sourceNs == "" || destNs == "" {
		return nil, nil
	}

	var present []string
	var errs error
	for _, name := range ImagePullSecretNamesToCopy(secretNames) {
		ok, err := copyImagePullSecretIfMissing(ctx, r, w, sourceNs, destNs, name)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if ok {
			present = append(present, name)
		}
	}
	return present, errs
}

func copyImagePullSecretIfMissing(ctx context.Context, r client.Reader, w client.Writer, sourceNs, destNs, secretName string) (bool, error) {
	var src corev1.Secret
	err := r.Get(ctx, client.ObjectKey{Namespace: sourceNs, Name: secretName}, &src)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read image pull secret %q in namespace %q: %w", secretName, sourceNs, err)
	}

	if sourceNs == destNs {
		return true, nil
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
			return true, nil
		}
		return false, fmt.Errorf("failed to copy image pull secret %q to namespace %q: %w", secretName, destNs, err)
	}
	return true, nil
}
