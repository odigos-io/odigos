package odigospro

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// proTokenExpiry renders as "18 May 2033 03:33:20 AM" in UTC and as
// "18 May 2033 10:33:20 AM" in the +07:00 zone used by the UTC test below.
const proTokenExpiry = int64(2000000000)

const proTokenExpiryRenderedUTC = "18 May 2033 03:33:20 AM"

// proToken builds a parsable on-prem token. The controller uses ParseUnverified,
// so the signing key is irrelevant.
func proToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("not-verified"))
	require.NoError(t, err)
	return signed
}

func proValidClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"aud": "odigos-pro-customer",
		"exp": float64(proTokenExpiry),
	}
}

// proSecret is the odigos-pro secret as helm/the CLI writes it for an on-prem install.
func proSecret(ns string, token string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OdigosProSecretName},
		Data:       map[string][]byte{k8sconsts.OdigosProSecretTokenKeyName: []byte(token)},
	}
}

// proDeploymentConfigMap always carries the version key helm installs, so every test can
// assert that reconciling pro info never disturbs the rest of the document.
func proDeploymentConfigMap(ns string, proInfo map[string]string) *corev1.ConfigMap {
	data := map[string]string{k8sconsts.OdigosDeploymentConfigMapVersionKey: "v1.2.3"}
	for k, v := range proInfo {
		data[k] = v
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OdigosDeploymentConfigMapName},
		Data:       data,
	}
}

func newSecretReconciler(objs ...client.Object) *odigossecretController {
	return newSecretReconcilerWithInterceptor(interceptor.Funcs{}, objs...)
}

func newSecretReconcilerWithInterceptor(funcs interceptor.Funcs, objs ...client.Object) *odigossecretController {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	return &odigossecretController{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).
			WithInterceptorFuncs(funcs).Build(),
	}
}

func proDeploymentData(t *testing.T, c client.Client, ns string) map[string]string {
	t.Helper()
	cm := &corev1.ConfigMap{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: k8sconsts.OdigosDeploymentConfigMapName}, cm))
	return cm.Data
}

// ****************
// odigossecretController.Reconcile
// ****************

// The three pro-info keys are the only published view of the on-prem license: `odigos describe`
// renders the audience and expiry, and the odigosconfiguration controller turns the profiles key
// into the cluster's effective profiles.
func TestProSecretReconcileWritesEveryProInfoKey(t *testing.T) {
	ns := env.GetCurrentNamespace()

	claims := proValidClaims()
	claims["profiles"] = []interface{}{"java-native-instrumentations", "allow_concurrent_agents"}

	r := newSecretReconciler(
		proDeploymentConfigMap(ns, nil),
		proSecret(ns, proToken(t, claims)),
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	require.Equal(t, map[string]string{
		k8sconsts.OdigosDeploymentConfigMapVersionKey:              "v1.2.3",
		k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey:       "odigos-pro-customer",
		k8sconsts.OdigosDeploymentConfigMapOnPremTokenExpKey:       proTokenExpiryRenderedUTC,
		k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey: "java-native-instrumentations, allow_concurrent_agents",
	}, proDeploymentData(t, r.Client, ns))
}

// The expiry is rendered for humans (`odigos describe`, the UI), so it must not depend on the
// time zone of whichever node the instrumentor happens to be scheduled on.
func TestProSecretReconcileRendersTheExpiryInUTC(t *testing.T) {
	ns := env.GetCurrentNamespace()

	originalLocal := time.Local
	t.Cleanup(func() { time.Local = originalLocal })
	time.Local = time.FixedZone("TEST+07", 7*60*60)

	r := newSecretReconciler(
		proDeploymentConfigMap(ns, nil),
		proSecret(ns, proToken(t, proValidClaims())),
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	data := proDeploymentData(t, r.Client, ns)
	assert.Equal(t, proTokenExpiryRenderedUTC, data[k8sconsts.OdigosDeploymentConfigMapOnPremTokenExpKey])
}

// A secret deleted while the controller was down (a downgrade to community, or a revoked
// license) has to clear every pro-info key. Only the audience key is reported back as "was
// present", so a dropped delete for the expiry or the profiles key is otherwise invisible and
// would leave the cluster entitled to profiles it no longer pays for.
func TestProSecretReconcileRemovesEveryProInfoKeyWhenTheSecretIsGone(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := newSecretReconciler(proDeploymentConfigMap(ns, map[string]string{
		k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey:       "odigos-pro-customer",
		k8sconsts.OdigosDeploymentConfigMapOnPremTokenExpKey:       proTokenExpiryRenderedUTC,
		k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey: "java-native-instrumentations",
	}))

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	require.Equal(t, map[string]string{
		k8sconsts.OdigosDeploymentConfigMapVersionKey: "v1.2.3",
	}, proDeploymentData(t, r.Client, ns))
}

// Profiles granted by a previous token must not survive a token that no longer grants them.
// The audience and expiry are overwritten unconditionally, but the profiles key is only
// removed by an explicit delete in the "no profiles in this token" branch.
func TestProSecretReconcileRevokesProfilesWhenTheNewTokenGrantsNone(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := newSecretReconciler(
		proDeploymentConfigMap(ns, map[string]string{
			k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey: "java-native-instrumentations",
		}),
		proSecret(ns, proToken(t, proValidClaims())),
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	data := proDeploymentData(t, r.Client, ns)
	assert.NotContains(t, data, k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey)
	assert.Equal(t, "odigos-pro-customer", data[k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey])
	assert.Equal(t, proTokenExpiryRenderedUTC, data[k8sconsts.OdigosDeploymentConfigMapOnPremTokenExpKey])
}

// An empty profiles claim is the same grant as no claim at all, and must not publish an empty
// profiles key: the entitlement reader splits the value unconditionally, so an empty string
// there becomes a single profile named "".
func TestProSecretReconcileTreatsAnEmptyProfilesClaimAsNoProfiles(t *testing.T) {
	ns := env.GetCurrentNamespace()

	claims := proValidClaims()
	claims["profiles"] = []interface{}{}

	r := newSecretReconciler(
		proDeploymentConfigMap(ns, nil),
		proSecret(ns, proToken(t, claims)),
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	assert.NotContains(t, proDeploymentData(t, r.Client, ns), k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey)
}

// odigos-deployment is created by helm at install time. Its absence is not something a retry
// can fix, so the reconcile must be terminal rather than requeue forever.
func TestProSecretReconcileIsTerminalWhenTheDeploymentConfigMapIsMissing(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := newSecretReconciler(proSecret(ns, proToken(t, proValidClaims())))

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.Error(t, err)
	assert.ErrorIs(t, err, reconcile.TerminalError(nil))
}

// A token the controller cannot read must not be retried either, and - just as importantly -
// must not partially overwrite the pro info that the previous good token published.
func TestProSecretReconcileKeepsTheExistingProInfoWhenTheTokenIsUnreadable(t *testing.T) {
	ns := env.GetCurrentNamespace()

	existing := map[string]string{
		k8sconsts.OdigosDeploymentConfigMapVersionKey:              "v1.2.3",
		k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey:       "odigos-pro-customer",
		k8sconsts.OdigosDeploymentConfigMapOnPremTokenExpKey:       proTokenExpiryRenderedUTC,
		k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey: "java-native-instrumentations",
	}

	r := newSecretReconciler(
		proDeploymentConfigMap(ns, existing),
		proSecret(ns, "this-is-not-a-jwt"),
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.Error(t, err)
	assert.ErrorIs(t, err, reconcile.TerminalError(nil))
	assert.Equal(t, existing, proDeploymentData(t, r.Client, ns))
}

// odigos-deployment is written by several controllers, so a conflict is routine. It has to
// requeue: giving up would leave the cluster without the pro info - and so without the
// profiles the token grants - until the next unrelated event happens to wake the controller.
func TestProSecretReconcileRequeuesOnAConflictingDeploymentConfigMapUpdate(t *testing.T) {
	ns := env.GetCurrentNamespace()

	tests := []struct {
		name         string
		updateErr    error
		wantRequeue  bool
		wantReconErr bool
	}{
		{
			name:        "a conflicting write is retried",
			updateErr:   apierrors.NewConflict(schema.GroupResource{Resource: "configmaps"}, k8sconsts.OdigosDeploymentConfigMapName, assert.AnError),
			wantRequeue: true,
		},
		{
			name:         "a denied write is reported",
			updateErr:    apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, k8sconsts.OdigosDeploymentConfigMapName, assert.AnError),
			wantReconErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newSecretReconcilerWithInterceptor(interceptor.Funcs{
				Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					return tt.updateErr
				},
			},
				proDeploymentConfigMap(ns, nil),
				proSecret(ns, proToken(t, proValidClaims())),
			)

			res, err := r.Reconcile(context.Background(), ctrl.Request{})

			if tt.wantReconErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantRequeue, res.Requeue)
		})
	}
}

// ****************
// updateProInfoInConfigMap
// ****************

// Every rejection reason must be distinguishable in the logs: the controller returns a
// terminal error, so the message is all an operator has to go on when an install silently
// stays on the community tier.
func TestUpdateProInfoInConfigMapRejectsUnreadableTokens(t *testing.T) {
	arrayAudienceClaims := proValidClaims()
	arrayAudienceClaims["aud"] = []interface{}{"odigos-pro-customer"}

	stringExpiryClaims := proValidClaims()
	stringExpiryClaims["exp"] = "2033-05-18T03:33:20Z"

	noExpiryClaims := jwt.MapClaims{"aud": "odigos-pro-customer"}

	stringProfilesClaims := proValidClaims()
	stringProfilesClaims["profiles"] = "java-native-instrumentations"

	nonStringProfileClaims := proValidClaims()
	nonStringProfileClaims["profiles"] = []interface{}{"java-native-instrumentations", 7}

	tests := []struct {
		name    string
		secret  *corev1.Secret
		wantErr string
	}{
		{
			name:    "the secret carries no token key",
			secret:  &corev1.Secret{Data: map[string][]byte{"some-other-key": []byte("x")}},
			wantErr: "token not found in secret",
		},
		{
			name:    "the token is not a JWT",
			secret:  proSecret("", "this-is-not-a-jwt"),
			wantErr: "failed to parse JWT token",
		},
		{
			name:    "the audience claim is missing",
			secret:  proSecret("", proToken(t, jwt.MapClaims{"exp": float64(proTokenExpiry)})),
			wantErr: "failed to parse JWT token audience",
		},
		{
			name:    "the audience claim is a list",
			secret:  proSecret("", proToken(t, arrayAudienceClaims)),
			wantErr: "failed to parse JWT token audience",
		},
		{
			name:    "the expiry claim is missing",
			secret:  proSecret("", proToken(t, noExpiryClaims)),
			wantErr: "failed to parse JWT token expiry",
		},
		{
			name:    "the expiry claim is a string",
			secret:  proSecret("", proToken(t, stringExpiryClaims)),
			wantErr: "failed to parse JWT token expiry",
		},
		{
			name:    "the profiles claim is a bare string",
			secret:  proSecret("", proToken(t, stringProfilesClaims)),
			wantErr: "failed to parse JWT token profiles",
		},
		{
			name:    "a profile in the list is not a string",
			secret:  proSecret("", proToken(t, nonStringProfileClaims)),
			wantErr: "found JWT profile which is not a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := proDeploymentConfigMap("", nil)

			err := updateProInfoInConfigMap(cm, tt.secret)

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			// a rejected token must not leave half of the pro info behind
			assert.Equal(t, map[string]string{k8sconsts.OdigosDeploymentConfigMapVersionKey: "v1.2.3"}, cm.Data)
		})
	}

	// the "failed to parse JWT token <part>" messages differ only by their last word, so a
	// copy-pasted message would otherwise pass every row above: rows sharing a reason must
	// share a message, and rows with different reasons must not.
	messagesByReason := map[string]map[string]bool{}
	for _, tt := range tests {
		err := updateProInfoInConfigMap(proDeploymentConfigMap("", nil), tt.secret)
		require.Error(t, err)
		if messagesByReason[tt.wantErr] == nil {
			messagesByReason[tt.wantErr] = map[string]bool{}
		}
		messagesByReason[tt.wantErr][err.Error()] = true
	}
	allMessages := map[string]bool{}
	for reason, messages := range messagesByReason {
		assert.Len(t, messages, 1, "rows rejected for %q must report the same message", reason)
		for message := range messages {
			allMessages[message] = true
		}
	}
	assert.Len(t, allMessages, len(messagesByReason), "every rejection reason must have its own message")
}

// ****************
// deleteProInfoFromConfigMap
// ****************

// The return value only drives a log line, but it is keyed on the audience alone: a document
// holding just an expiry or just profiles is still pro info that has to be reported as removed
// or silently vanish.
func TestDeleteProInfoFromConfigMapReportsOnTheAudienceKey(t *testing.T) {
	tests := []struct {
		name    string
		proInfo map[string]string
		want    bool
	}{
		{
			name:    "no pro info at all",
			proInfo: nil,
			want:    false,
		},
		{
			name: "a full pro info document",
			proInfo: map[string]string{
				k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey:       "odigos-pro-customer",
				k8sconsts.OdigosDeploymentConfigMapOnPremTokenExpKey:       proTokenExpiryRenderedUTC,
				k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey: "java-native-instrumentations",
			},
			want: true,
		},
		{
			name:    "an empty audience is still an audience",
			proInfo: map[string]string{k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey: ""},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := proDeploymentConfigMap("", tt.proInfo)

			assert.Equal(t, tt.want, deleteProInfoFromConfigMap(cm))
			assert.Equal(t, map[string]string{k8sconsts.OdigosDeploymentConfigMapVersionKey: "v1.2.3"}, cm.Data)
		})
	}
}

// ****************
// getProfilesString
// ****************

func TestGetProfilesString(t *testing.T) {
	tests := []struct {
		name       string
		profiles   interface{}
		absent     bool
		want       string
		wantExists bool
		wantErr    string
	}{
		{
			name:   "the claim is absent",
			absent: true,
		},
		{
			name:     "the claim is an empty list",
			profiles: []interface{}{},
		},
		{
			name:       "a single profile",
			profiles:   []interface{}{"java-native-instrumentations"},
			want:       "java-native-instrumentations",
			wantExists: true,
		},
		{
			name:       "several profiles",
			profiles:   []interface{}{"java-native-instrumentations", "allow_concurrent_agents", "ebpf-log-capture"},
			want:       "java-native-instrumentations, allow_concurrent_agents, ebpf-log-capture",
			wantExists: true,
		},
		{
			name:     "the claim is a bare string",
			profiles: "java-native-instrumentations",
			wantErr:  "failed to parse JWT token profiles",
		},
		{
			name:     "a profile is not a string",
			profiles: []interface{}{"java-native-instrumentations", 7},
			wantErr:  "found JWT profile which is not a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := jwt.MapClaims{}
			if !tt.absent {
				claims["profiles"] = tt.profiles
			}

			got, exists, err := getProfilesString(claims)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.False(t, exists)
				assert.Empty(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantExists, exists)
		})
	}
}

// The profiles are published as one ConfigMap value and split back apart by
// instrumentor/controllers/odigosconfiguration, which does strings.Split(value, ", ").
// Changing the separator here drops every profile but the first out of the effective
// profiles - the cluster quietly loses the features the token pays for, with no error
// anywhere. TestOnPremTokenProfilesReachTheEffectiveConfig pins the reading half against
// the same encoding.
func TestProfilesStringUsesTheSeparatorTheEntitlementReaderSplitsOn(t *testing.T) {
	const separator = ", "

	granted := []string{"java-native-instrumentations", "allow_concurrent_agents"}
	claim := make([]interface{}, 0, len(granted))
	for _, p := range granted {
		claim = append(claim, p)
	}

	got, exists, err := getProfilesString(jwt.MapClaims{"profiles": claim})
	require.NoError(t, err)
	require.True(t, exists)

	assert.Equal(t, granted, strings.Split(got, separator))
	// a single profile must round trip through the same split with no separator at all
	single, _, err := getProfilesString(jwt.MapClaims{"profiles": []interface{}{granted[0]}})
	require.NoError(t, err)
	assert.Equal(t, granted[:1], strings.Split(single, separator))
}
