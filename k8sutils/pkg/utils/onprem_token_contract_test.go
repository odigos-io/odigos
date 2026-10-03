package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"
)

// onPremTokenForContract builds a token that passes validation; only the payload is inspected.
func onPremTokenForContract(t *testing.T) string {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"exp": time.Now().Add(time.Hour).Unix(),
		"iss": "https://odigos.io",
		"sub": "https://odigos.io/onprem",
		"aud": "acme-corp",
	})
	require.NoError(t, err)
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

// pro.UpdateOdigosToken writes the token and GetCurrentOdigosTier decides the tier by looking for
// it, but the two packages spell the secret key out separately. Driving both against one cluster
// is the only thing that proves a rotated token actually leaves the install on the on-prem tier.
func TestRotatingTheOnPremTokenLeavesTheClusterOnTheOnPremTier(t *testing.T) {
	server := newTierAPIServer()
	server.put(t, server.secretPath(tierNamespace, k8sconsts.OdigosProSecretName), tierProSecret(map[string][]byte{
		k8sconsts.OdigosOnpremTokenSecretKey: []byte("previous-token"),
	}))
	server.put(t, server.daemonSetPath(tierNamespace, k8sconsts.OdigletDaemonSetName), &appsv1.DaemonSet{
		TypeMeta:   metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.OdigletDaemonSetName, Namespace: tierNamespace},
	})
	clientset := tierClientset(t, server)

	token := onPremTokenForContract(t)
	require.NoError(t, pro.UpdateOdigosToken(context.Background(), clientset, tierNamespace, token))

	tier, err := GetCurrentOdigosTier(context.Background(), tierNamespace, clientset)
	require.NoError(t, err)
	assert.Equal(t, common.OnPremOdigosTier, tier)

	var rotated corev1.Secret
	server.stored(t, server.secretPath(tierNamespace, k8sconsts.OdigosProSecretName), &rotated)
	assert.Equal(t, []byte(token), rotated.Data[k8sconsts.OdigosOnpremTokenSecretKey])

	// The same rotation has to reach the image pull secret and restart the odiglet, otherwise the
	// agents keep running with credentials that are about to expire.
	var pullSecret corev1.Secret
	server.stored(t, server.secretPath(tierNamespace, k8sconsts.OdigosEnterpriseRegistryPullSecretName), &pullSecret)
	assert.Equal(t, corev1.SecretTypeDockerConfigJson, pullSecret.Type)

	var odiglet appsv1.DaemonSet
	server.stored(t, server.daemonSetPath(tierNamespace, k8sconsts.OdigletDaemonSetName), &odiglet)
	assert.Contains(t, odiglet.Spec.Template.Annotations, odigosconsts.RolloutTriggerAnnotation)
}
