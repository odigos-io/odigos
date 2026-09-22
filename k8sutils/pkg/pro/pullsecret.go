package pro

import (
	"bytes"
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

// ErrDestAlreadyExists is returned when a destination secret exists but was
// not created by Odigos, so it is not overwritten.
var ErrDestAlreadyExists = errors.New("destination image pull secret already exists and is not managed by Odigos")

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

// CopiedImagePullSecretLabels marks a secret as an Odigos-managed copy in an
// instrumented namespace. Re-sync may update these; unlabeled secrets are not replaced.
func CopiedImagePullSecretLabels() map[string]string {
	return map[string]string{
		k8sconsts.OdigosSystemLabelKey:             k8sconsts.OdigosSystemLabelValue,
		k8sconsts.OdigosCopiedImagePullSecretLabel: k8sconsts.OdigosSystemLabelValue,
	}
}

func isOdigosCopiedPullSecret(secret *corev1.Secret) bool {
	return secret.Labels[k8sconsts.OdigosCopiedImagePullSecretLabel] == k8sconsts.OdigosSystemLabelValue
}

// CopyImagePullSecrets copies each named pull secret from sourceNs into destNs.
// secretNames should come from effective config (user secrets and enterprise, if any).
// Existing destination secrets labeled as Odigos copies are updated; others are left alone.
func CopyImagePullSecrets(
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

// IgnoreDestAlreadyExists drops ErrDestAlreadyExists from err so a later
// workload in the same namespace does not fail after the first successful copy.
func IgnoreDestAlreadyExists(err error) error {
	if err == nil {
		return nil
	}
	if u, ok := err.(interface{ Unwrap() []error }); ok {
		var kept []error
		for _, e := range u.Unwrap() {
			if !errors.Is(e, ErrDestAlreadyExists) {
				kept = append(kept, e)
			}
		}
		return errors.Join(kept...)
	}
	if errors.Is(err, ErrDestAlreadyExists) {
		return nil
	}
	return err
}

// copyOneImagePullSecret creates dest, or updates it when it is an Odigos copy.
// Dest GET/update is limited to the configured pull-secret names (resourceNames RBAC).
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

	var dest corev1.Secret
	err = r.Get(ctx, client.ObjectKey{Namespace: destNs, Name: secretName}, &dest)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to read image pull secret %q in namespace %q: %w", secretName, destNs, err)
		}
		dest := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: destNs,
				Labels:    CopiedImagePullSecretLabels(),
			},
			Type: src.Type,
			Data: data,
		}
		if createErr := w.Create(ctx, dest); createErr != nil {
			if apierrors.IsAlreadyExists(createErr) {
				return fmt.Errorf("%w: %q in namespace %q", ErrDestAlreadyExists, secretName, destNs)
			}
			return fmt.Errorf("failed to copy image pull secret %q to namespace %q: %w", secretName, destNs, createErr)
		}
		return nil
	}

	if !isOdigosCopiedPullSecret(&dest) {
		return fmt.Errorf("%w: %q in namespace %q", ErrDestAlreadyExists, secretName, destNs)
	}
	if dest.Type == src.Type && secretDataEqual(dest.Data, data) {
		return nil
	}
	dest.Type = src.Type
	dest.Data = data
	if updateErr := w.Update(ctx, &dest); updateErr != nil {
		return fmt.Errorf("failed to update copied image pull secret %q in namespace %q: %w", secretName, destNs, updateErr)
	}
	return nil
}

func secretDataEqual(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || !bytes.Equal(av, bv) {
			return false
		}
	}
	return true
}
