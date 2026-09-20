package pro

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
)

const proNamespace = "odigos-pro-test-ns"

// proTokenWithClaims builds an unsigned JWT carrying the given payload. Odigos only base64-decodes
// and inspects the payload, so the header and signature segments are arbitrary.
func proTokenWithClaims(t *testing.T, claims map[string]any) string {
	t.Helper()

	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

// proValidToken is a token that passes validation, so a test using it observes what the code under
// test does rather than how it rejects input.
func proValidToken(t *testing.T) string {
	t.Helper()

	return proTokenWithClaims(t, map[string]any{
		"exp": time.Now().Add(time.Hour).Unix(),
		"iss": "https://odigos.io",
		"sub": "https://odigos.io/onprem",
		"aud": "acme-corp",
	})
}

func proExpiredToken(t *testing.T) string {
	t.Helper()

	return proTokenWithClaims(t, map[string]any{
		"exp": time.Now().Add(-2 * time.Hour).Unix(),
		"iss": "https://odigos.io",
		"sub": "https://odigos.io/onprem",
		"aud": "acme-corp",
	})
}

// proTokenSecret is the odigos-pro secret the CLI updates in place.
func proTokenSecret(data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosProSecretName,
			Namespace: proNamespace,
		},
		Data: data,
	}
}

func proOdigletDaemonSet(annotations map[string]string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigletDaemonSetName,
			Namespace: proNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: annotations},
			},
		},
	}
}

// proConfigMap renders an OdigosConfiguration into the ConfigMap the pull-secret gate reads.
func proConfigMap(t *testing.T, config common.OdigosConfiguration) *corev1.ConfigMap {
	t.Helper()

	raw, err := yaml.Marshal(config)
	require.NoError(t, err)
	return proRawConfigMap(map[string]string{odigosconsts.OdigosConfigurationFileName: string(raw)})
}

func proRawConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      odigosconsts.OdigosConfigurationName,
			Namespace: proNamespace,
		},
		Data: data,
	}
}

func proClient(objects ...runtime.Object) *k8sfake.Clientset {
	return k8sfake.NewClientset(objects...)
}

// proStoredSecret reads a secret back through the client, failing the test when it is absent.
func proStoredSecret(t *testing.T, client kubernetes.Interface, name string) *corev1.Secret {
	t.Helper()

	secret, err := client.CoreV1().Secrets(proNamespace).Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(t, err)
	return secret
}

func proSecretExists(t *testing.T, client kubernetes.Interface, name string) bool {
	t.Helper()

	_, err := client.CoreV1().Secrets(proNamespace).Get(context.Background(), name, metav1.GetOptions{})
	return err == nil
}

func proStoredDaemonSet(t *testing.T, client kubernetes.Interface, name string) *appsv1.DaemonSet {
	t.Helper()

	daemonSet, err := client.AppsV1().DaemonSets(proNamespace).Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(t, err)
	return daemonSet
}

// proDockerAuths decodes the .dockerconfigjson payload of a pull secret.
func proDockerAuths(t *testing.T, secret *corev1.Secret) map[string]map[string]string {
	t.Helper()

	var decoded struct {
		Auths map[string]map[string]string `json:"auths"`
	}
	require.NoError(t, json.Unmarshal(secret.Data[corev1.DockerConfigJsonKey], &decoded))
	return decoded.Auths
}

// proFailOn makes the named verb on the named resource fail, so an error branch can be reached
// without taking the whole client down.
func proFailOn(client *k8sfake.Clientset, verb, resource string, err error) {
	client.PrependReactor(verb, resource, func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, err
	})
}

// proRejectAll is a "must not be reached" assertion for short-circuit paths.
func proRejectAll(t *testing.T, client *k8sfake.Clientset, verb, resource string) {
	t.Helper()

	client.PrependReactor(verb, resource, func(action k8stesting.Action) (bool, runtime.Object, error) {
		t.Errorf("unexpected %s on %s", action.GetVerb(), action.GetResource().Resource)
		return false, nil, nil
	})
}
