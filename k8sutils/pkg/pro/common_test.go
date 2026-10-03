package pro

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
)

func TestUpdateSecretTokenStoresTheTokenUnderTheKeyTheTierCheckReads(t *testing.T) {
	client := proClient(proTokenSecret(map[string][]byte{
		k8sconsts.OdigosOnpremTokenSecretKey: []byte("previous-token"),
		"unrelated-key":                      []byte("must survive"),
	}))

	require.NoError(t, updateSecretToken(context.Background(), client, proNamespace, "new-token"))

	stored := proStoredSecret(t, client, k8sconsts.OdigosProSecretName)
	assert.Equal(t, []byte("new-token"), stored.Data[k8sconsts.OdigosOnpremTokenSecretKey])
	assert.Equal(t, []byte("must survive"), stored.Data["unrelated-key"])
}

// The odigos-pro secret only exists on pro installs, so its absence is reported as a licensing
// message rather than as a Kubernetes NotFound the caller might retry.
func TestUpdateSecretTokenReportsACommunityInstallWhenTheProSecretIsMissing(t *testing.T) {
	client := proClient()

	err := updateSecretToken(context.Background(), client, proNamespace, "new-token")

	require.Error(t, err)
	assert.Equal(t, "tokens are not available in the community version of Odigos. Please contact Odigos team to inquire about pro version", err.Error())
	assert.False(t, apierrors.IsNotFound(err))
}

func TestUpdateSecretTokenPropagatesANonNotFoundReadError(t *testing.T) {
	readErr := apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, k8sconsts.OdigosProSecretName, errors.New("no access"))
	client := proClient(proTokenSecret(map[string][]byte{}))
	proFailOn(client, "get", "secrets", readErr)

	err := updateSecretToken(context.Background(), client, proNamespace, "new-token")

	require.Error(t, err)
	assert.True(t, apierrors.IsForbidden(err))
	// A permission problem must not be reported as "you are on the community version".
	assert.NotContains(t, err.Error(), "community version")
}

func TestUpdateSecretTokenPropagatesAWriteError(t *testing.T) {
	writeErr := apierrors.NewConflict(schema.GroupResource{Resource: "secrets"}, k8sconsts.OdigosProSecretName, errors.New("modified"))
	client := proClient(proTokenSecret(map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: []byte("previous")}))
	proFailOn(client, "update", "secrets", writeErr)

	err := updateSecretToken(context.Background(), client, proNamespace, "new-token")

	require.Error(t, err)
	assert.True(t, apierrors.IsConflict(err))
}

func TestUpdateSecretTokenOnlyTouchesTheNamespaceItWasGiven(t *testing.T) {
	other := proTokenSecret(map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: []byte("other-namespace-token")})
	other.Namespace = "some-other-namespace"
	client := proClient(other)

	err := updateSecretToken(context.Background(), client, proNamespace, "new-token")

	require.Error(t, err)
	stored, getErr := client.CoreV1().Secrets("some-other-namespace").Get(context.Background(), k8sconsts.OdigosProSecretName, metav1.GetOptions{})
	require.NoError(t, getErr)
	assert.Equal(t, []byte("other-namespace-token"), stored.Data[k8sconsts.OdigosOnpremTokenSecretKey])
}

func TestShouldUseEnterpriseRegistryPullSecret(t *testing.T) {
	tests := []struct {
		name     string
		config   *common.OdigosConfiguration
		rawData  map[string]string
		expected bool
	}{
		{
			name:     "no odigos configuration yet",
			expected: true,
		},
		{
			name:     "configuration exists but carries no config file",
			rawData:  map[string]string{"other-key": "value"},
			expected: true,
		},
		{
			name:     "config file is present but empty",
			rawData:  map[string]string{odigosconsts.OdigosConfigurationFileName: ""},
			expected: true,
		},
		{
			name:     "default configuration pulls from the odigos registry",
			config:   &common.OdigosConfiguration{},
			expected: true,
		},
		{
			name:     "openshift installs pull from the openshift registry",
			config:   &common.OdigosConfiguration{OpenshiftEnabled: true},
			expected: false,
		},
		{
			name:     "a custom image prefix means the user manages their own registry",
			config:   &common.OdigosConfiguration{ImagePrefix: "my-mirror.example.test/odigos"},
			expected: false,
		},
		{
			name:     "an explicit odigos image prefix still needs the pull secret",
			config:   &common.OdigosConfiguration{ImagePrefix: k8sconsts.OdigosImagePrefix},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := proClient()
			switch {
			case tt.config != nil:
				client = proClient(proConfigMap(t, *tt.config))
			case tt.rawData != nil:
				client = proClient(proRawConfigMap(tt.rawData))
			}

			got, err := ShouldUseEnterpriseRegistryPullSecret(context.Background(), client, proNamespace)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestShouldUseEnterpriseRegistryPullSecretRejectsUnparsableConfiguration(t *testing.T) {
	client := proClient(proRawConfigMap(map[string]string{
		odigosconsts.OdigosConfigurationFileName: "imagePrefix: [this is not a scalar",
	}))

	got, err := ShouldUseEnterpriseRegistryPullSecret(context.Background(), client, proNamespace)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to parse odigos configuration")
	// Defaulting to true on a parse failure would create a pull secret an air-gapped install
	// deliberately does not want.
	assert.False(t, got)
}

func TestShouldUseEnterpriseRegistryPullSecretPropagatesAReadError(t *testing.T) {
	client := proClient()
	proFailOn(client, "get", "configmaps", apierrors.NewForbidden(
		schema.GroupResource{Resource: "configmaps"}, odigosconsts.OdigosConfigurationName, errors.New("no access")))

	got, err := ShouldUseEnterpriseRegistryPullSecret(context.Background(), client, proNamespace)

	require.Error(t, err)
	assert.True(t, apierrors.IsForbidden(err))
	assert.False(t, got)
}

// The gate reads the configuration of the namespace it is asked about; a configuration in some
// other namespace must not decide whether this install needs a pull secret.
func TestShouldUseEnterpriseRegistryPullSecretIgnoresAnotherNamespacesConfiguration(t *testing.T) {
	elsewhere := proConfigMap(t, common.OdigosConfiguration{ImagePrefix: "my-mirror.example.test/odigos"})
	elsewhere.Namespace = "some-other-namespace"
	client := proClient(elsewhere)

	got, err := ShouldUseEnterpriseRegistryPullSecret(context.Background(), client, proNamespace)

	require.NoError(t, err)
	assert.True(t, got)
}

func TestEnsureEnterpriseRegistryPullSecretCreatesTheSecret(t *testing.T) {
	token := "onprem-token-value"
	client := proClient()

	require.NoError(t, EnsureEnterpriseRegistryPullSecret(context.Background(), client, proNamespace, token))

	stored := proStoredSecret(t, client, k8sconsts.OdigosEnterpriseRegistryPullSecretName)
	assert.Equal(t, proNamespace, stored.Namespace)
	assert.Equal(t, corev1.SecretTypeDockerConfigJson, stored.Type)
	assert.Equal(t, k8sconsts.OdigosSystemLabelValue, stored.Labels[k8sconsts.OdigosSystemLabelKey])

	auths := proDockerAuths(t, stored)
	// The registry host is the literal the kubelet matches an image reference against; it has to
	// stay in step with the images Odigos actually ships.
	require.Contains(t, auths, "registry.odigos.io")
	assert.Equal(t, k8sconsts.OdigosImagePrefix, "registry.odigos.io")
	assert.Equal(t, "odigos", auths["registry.odigos.io"]["username"])
	assert.Equal(t, token, auths["registry.odigos.io"]["password"])
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("odigos:"+token)), auths["registry.odigos.io"]["auth"])
}

func TestEnsureEnterpriseRegistryPullSecretRefreshesAnExistingSecret(t *testing.T) {
	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        k8sconsts.OdigosEnterpriseRegistryPullSecretName,
			Namespace:   proNamespace,
			Labels:      map[string]string{"stale-label": "stale"},
			Annotations: map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "{}"},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte("{}"), "stale-key": []byte("stale")},
	}
	client := proClient(existing)

	require.NoError(t, EnsureEnterpriseRegistryPullSecret(context.Background(), client, proNamespace, "rotated-token"))

	stored := proStoredSecret(t, client, k8sconsts.OdigosEnterpriseRegistryPullSecretName)
	assert.Equal(t, corev1.SecretTypeDockerConfigJson, stored.Type)
	assert.Equal(t, "rotated-token", proDockerAuths(t, stored)["registry.odigos.io"]["password"])
	// Stale data and labels are replaced wholesale, so a rotated token cannot leave the previous
	// credentials behind.
	assert.NotContains(t, stored.Data, "stale-key")
	assert.NotContains(t, stored.Labels, "stale-label")
	assert.Equal(t, k8sconsts.OdigosSystemLabelValue, stored.Labels[k8sconsts.OdigosSystemLabelKey])
	// Annotations are left alone; they are not part of what Odigos owns on this secret.
	assert.Equal(t, "{}", stored.Annotations["kubectl.kubernetes.io/last-applied-configuration"])
}

func TestEnsureEnterpriseRegistryPullSecretIsANoOpForACustomRegistry(t *testing.T) {
	client := proClient(proConfigMap(t, common.OdigosConfiguration{ImagePrefix: "my-mirror.example.test/odigos"}))
	proRejectAll(t, client, "create", "secrets")
	proRejectAll(t, client, "update", "secrets")

	require.NoError(t, EnsureEnterpriseRegistryPullSecret(context.Background(), client, proNamespace, "onprem-token-value"))

	assert.False(t, proSecretExists(t, client, k8sconsts.OdigosEnterpriseRegistryPullSecretName))
}

func TestEnsureEnterpriseRegistryPullSecretPropagatesTheGateError(t *testing.T) {
	client := proClient()
	proFailOn(client, "get", "configmaps", apierrors.NewForbidden(
		schema.GroupResource{Resource: "configmaps"}, odigosconsts.OdigosConfigurationName, errors.New("no access")))
	proRejectAll(t, client, "create", "secrets")

	err := EnsureEnterpriseRegistryPullSecret(context.Background(), client, proNamespace, "onprem-token-value")

	require.Error(t, err)
	assert.True(t, apierrors.IsForbidden(err))
}

// The three failure points around the pull secret are reported with different verbs, so an
// operator reading the message knows whether Odigos could not read, create or update it.
func TestEnsureEnterpriseRegistryPullSecretNamesTheFailedOperation(t *testing.T) {
	tests := []struct {
		name    string
		verb    string
		failure error
		seed    bool
		phrase  string
	}{
		{
			name:    "create failure",
			verb:    "create",
			failure: apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, k8sconsts.OdigosEnterpriseRegistryPullSecretName, errors.New("no access")),
			phrase:  `failed to create enterprise registry pull secret "odigos-enterprise-registry"`,
		},
		{
			name:    "read failure",
			verb:    "get",
			failure: apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, k8sconsts.OdigosEnterpriseRegistryPullSecretName, errors.New("no access")),
			phrase:  `failed to read enterprise registry pull secret "odigos-enterprise-registry"`,
		},
		{
			name:    "update failure",
			verb:    "update",
			failure: apierrors.NewConflict(schema.GroupResource{Resource: "secrets"}, k8sconsts.OdigosEnterpriseRegistryPullSecretName, errors.New("modified")),
			seed:    true,
			phrase:  `failed to update enterprise registry pull secret "odigos-enterprise-registry"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := proClient()
			if tt.seed {
				client = proClient(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
					Name:      k8sconsts.OdigosEnterpriseRegistryPullSecretName,
					Namespace: proNamespace,
				}})
			}
			proFailOn(client, tt.verb, "secrets", tt.failure)

			err := EnsureEnterpriseRegistryPullSecret(context.Background(), client, proNamespace, "onprem-token-value")

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.phrase)
			assert.ErrorIs(t, err, tt.failure)
		})
	}
}

func TestOdigletRolloutTriggerStampsThePodTemplate(t *testing.T) {
	client := proClient(proOdigletDaemonSet(map[string]string{"unrelated": "kept"}))
	before := time.Now().Add(-time.Second)

	require.NoError(t, odigletRolloutTrigger(context.Background(), client, proNamespace))

	stored := proStoredDaemonSet(t, client, k8sconsts.OdigletDaemonSetName)
	annotations := stored.Spec.Template.Annotations
	assert.Equal(t, "kept", annotations["unrelated"])

	stamp, ok := annotations[odigosconsts.RolloutTriggerAnnotation]
	require.True(t, ok, "the rollout trigger annotation drives the restart; without it the odiglet keeps the old token")
	parsed, err := time.Parse(time.RFC3339, stamp)
	require.NoError(t, err)
	assert.WithinRange(t, parsed, before, time.Now().Add(time.Second))
}

// The odiglet DaemonSet is created by Helm under a fixed name; a mismatch here means the token
// rotation silently never reaches the agents.
func TestOdigletRolloutTriggerTargetsTheOdigletDaemonSet(t *testing.T) {
	assert.Equal(t, "odiglet", k8sconsts.OdigletDaemonSetName)

	client := proClient(proOdigletDaemonSet(nil))
	otherDaemonSet := proOdigletDaemonSet(nil)
	otherDaemonSet.Name = "odigos-data-collection"
	_, err := client.AppsV1().DaemonSets(proNamespace).Create(context.Background(), otherDaemonSet, metav1.CreateOptions{})
	require.NoError(t, err)

	require.NoError(t, odigletRolloutTrigger(context.Background(), client, proNamespace))

	assert.Contains(t, proStoredDaemonSet(t, client, k8sconsts.OdigletDaemonSetName).Spec.Template.Annotations, odigosconsts.RolloutTriggerAnnotation)
	assert.NotContains(t, proStoredDaemonSet(t, client, "odigos-data-collection").Spec.Template.Annotations, odigosconsts.RolloutTriggerAnnotation)
}

func TestOdigletRolloutTriggerCreatesTheAnnotationMapWhenThereIsNone(t *testing.T) {
	client := proClient(proOdigletDaemonSet(nil))

	require.NoError(t, odigletRolloutTrigger(context.Background(), client, proNamespace))

	assert.Contains(t, proStoredDaemonSet(t, client, k8sconsts.OdigletDaemonSetName).Spec.Template.Annotations, odigosconsts.RolloutTriggerAnnotation)
}

func TestOdigletRolloutTriggerReportsAMissingDaemonSetWithItsNamespace(t *testing.T) {
	client := proClient()

	err := odigletRolloutTrigger(context.Background(), client, proNamespace)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to get odiglet DaemonSet in namespace "+proNamespace)
}

func TestOdigletRolloutTriggerSuggestsAManualRestartWhenTheUpdateFails(t *testing.T) {
	client := proClient(proOdigletDaemonSet(nil))
	updateErr := apierrors.NewConflict(schema.GroupResource{Resource: "daemonsets"}, k8sconsts.OdigletDaemonSetName, errors.New("modified"))
	proFailOn(client, "update", "daemonsets", updateErr)

	err := odigletRolloutTrigger(context.Background(), client, proNamespace)

	require.ErrorIs(t, err, updateErr)
	assert.ErrorContains(t, err, "kubectl rollout restart daemonset odiglet -n "+proNamespace)
}

func TestUpdateOdigosTokenAppliesTheWholeFlow(t *testing.T) {
	token := proValidToken(t)
	client := proClient(
		proTokenSecret(map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: []byte("previous-token")}),
		proOdigletDaemonSet(nil),
	)

	require.NoError(t, UpdateOdigosToken(context.Background(), client, proNamespace, token))

	assert.Equal(t, []byte(token), proStoredSecret(t, client, k8sconsts.OdigosProSecretName).Data[k8sconsts.OdigosOnpremTokenSecretKey])
	assert.Equal(t, token, proDockerAuths(t, proStoredSecret(t, client, k8sconsts.OdigosEnterpriseRegistryPullSecretName))["registry.odigos.io"]["password"])
	assert.Contains(t, proStoredDaemonSet(t, client, k8sconsts.OdigletDaemonSetName).Spec.Template.Annotations, odigosconsts.RolloutTriggerAnnotation)
}

func TestUpdateOdigosTokenRejectsAnInvalidToken(t *testing.T) {
	tests := []struct {
		name   string
		token  func(t *testing.T) string
		phrase string
	}{
		{
			name:   "empty",
			token:  func(*testing.T) string { return "" },
			phrase: "missing Odigos Pro token",
		},
		{
			name:   "not a jwt",
			token:  func(*testing.T) string { return "not-a-jwt" },
			phrase: "invalid JWT token format",
		},
		{
			name:   "expired",
			token:  proExpiredToken,
			phrase: "Token has expired",
		},
		{
			name: "issued by someone else",
			token: func(t *testing.T) string {
				return proTokenWithClaims(t, map[string]any{
					"exp": time.Now().Add(time.Hour).Unix(),
					"iss": "https://attacker.example.test",
					"sub": "https://odigos.io/onprem",
					"aud": "acme-corp",
				})
			},
			phrase: "Invalid iss",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := proClient(
				proTokenSecret(map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: []byte("previous-token")}),
				proOdigletDaemonSet(nil),
			)
			proRejectAll(t, client, "update", "secrets")
			proRejectAll(t, client, "create", "secrets")
			proRejectAll(t, client, "update", "daemonsets")

			err := UpdateOdigosToken(context.Background(), client, proNamespace, tt.token(t))

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.phrase)
			// A rejected token must leave the previous one in place.
			assert.Equal(t, []byte("previous-token"), proStoredSecret(t, client, k8sconsts.OdigosProSecretName).Data[k8sconsts.OdigosOnpremTokenSecretKey])
		})
	}
}

// Each stage of the update names itself, and a stage that fails stops the ones after it rather
// than leaving the install half-rotated in a way the message does not describe.
func TestUpdateOdigosTokenStopsAtTheFirstFailedStage(t *testing.T) {
	tests := []struct {
		name          string
		seedProSecret bool
		seedOdiglet   bool
		failVerb      string
		failResource  string
		phrase        string
		rejectVerbs   map[string]string
	}{
		{
			name:          "token secret update fails",
			seedProSecret: true,
			seedOdiglet:   true,
			failVerb:      "update",
			failResource:  "secrets",
			phrase:        "failed to update secret token: ",
			rejectVerbs:   map[string]string{"update": "daemonsets"},
		},
		{
			name:          "pull secret creation fails",
			seedProSecret: true,
			seedOdiglet:   true,
			failVerb:      "create",
			failResource:  "secrets",
			phrase:        "failed to update enterprise registry pull secret: ",
			rejectVerbs:   map[string]string{"update": "daemonsets"},
		},
		{
			name:          "odiglet rollout fails",
			seedProSecret: true,
			seedOdiglet:   false,
			phrase:        "failed to trigger odiglet rollout: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []runtime.Object{}
			if tt.seedProSecret {
				objects = append(objects, proTokenSecret(map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: []byte("previous-token")}))
			}
			if tt.seedOdiglet {
				objects = append(objects, proOdigletDaemonSet(nil))
			}
			client := proClient(objects...)

			if tt.failVerb != "" {
				proFailOn(client, tt.failVerb, tt.failResource, apierrors.NewForbidden(
					schema.GroupResource{Resource: tt.failResource}, "", errors.New("no access")))
			}
			for verb, resource := range tt.rejectVerbs {
				proRejectAll(t, client, verb, resource)
			}

			err := UpdateOdigosToken(context.Background(), client, proNamespace, proValidToken(t))

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.phrase)
		})
	}
}
