package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const interrogationEffectiveConfig = `interrogation:
  enabled: true
  model:
    provider: openai
    name: gpt-4o-mini
    apiBase: https://proxy.example.com/v1
`

func interrogationModelInput(m model.InterrogationModelConfigInput) model.LocalUIConfigInput {
	return model.LocalUIConfigInput{Interrogation: &model.InterrogationConfigInput{Model: &m}}
}

func aiConfigMap(name, config string) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: env.GetCurrentNamespace(), UID: types.UID(name)}, Data: map[string]string{consts.OdigosConfigurationFileName: config}}
}

func aiTestClient(objects ...client.Object) client.Client {
	objects = append([]client.Object{aiConfigMap(consts.OdigosConfigurationName, interrogationEffectiveConfig)}, objects...)
	return fake.NewClientBuilder().WithObjects(objects...).Build()
}

func TestInterrogationPartialEditPreservesInheritanceAndExplicitClear(t *testing.T) {
	ctx := context.Background()
	c := aiTestClient()
	require.NoError(t, UpdateLocalUIConfig(ctx, c, interrogationModelInput(model.InterrogationModelConfigInput{Name: StringPtr("gpt-4.1"), APIBase: StringPtr("")})))
	cfg, err := GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	require.Nil(t, cfg.Interrogation.Model.Provider, "editing the name must not pin the inherited provider")
	require.Equal(t, "gpt-4.1", *cfg.Interrogation.Model.Name)
	require.NotNil(t, cfg.Interrogation.Model.ApiBase)
	require.Empty(t, *cfg.Interrogation.Model.ApiBase)
	provenance := map[string]string{}
	recordOverlayProvenance(cfg, provenance, consts.OdigosLocalUiConfigName)
	require.Equal(t, consts.OdigosLocalUiConfigName, provenance["interrogation.model.apiBase"])
	require.Empty(t, provenance["interrogation.model.provider"])
	require.NoError(t, UpdateLocalUIConfig(ctx, c, model.LocalUIConfigInput{Interrogation: &model.InterrogationConfigInput{ResetPaths: []string{"interrogation.model.name"}}}))
	cfg, err = GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	require.Nil(t, cfg.Interrogation.Model.Name)
	require.NotNil(t, cfg.Interrogation.Model.ApiBase)
}

func TestInterrogationCredentialOverrideIsAtomicAndHelmIndependent(t *testing.T) {
	ctx := context.Background()
	baseline := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "odigos-litellm", Namespace: env.GetCurrentNamespace()}, Data: map[string][]byte{"master-key": []byte("master-original"), "upstream-api-key": []byte("helm-original")}}
	paused := aiConfigMap(consts.OdigosLocalUiConfigName, "interrogation:\n  analysisEnabled: false\n")
	c := aiTestClient(baseline, paused)
	input := interrogationModelInput(model.InterrogationModelConfigInput{Name: StringPtr("gpt-4.1"), APIKey: StringPtr("  test-new-key  ")})
	require.NoError(t, UpdateLocalUIConfig(ctx, c, input))
	cfg, err := GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	require.False(t, *cfg.Interrogation.AnalysisEnabled)
	require.NotEmpty(t, cfg.Interrogation.CredentialSecretName)
	selected := &corev1.Secret{}
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: env.GetCurrentNamespace(), Name: cfg.Interrogation.CredentialSecretName}, selected))
	require.True(t, *selected.Immutable)
	require.Equal(t, "test-new-key", string(selected.Data["upstream-api-key"]))
	cm := &corev1.ConfigMap{}
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: env.GetCurrentNamespace(), Name: consts.OdigosLocalUiConfigName}, cm))
	require.NotContains(t, cm.Data[consts.OdigosConfigurationFileName], "test-new-key")
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(baseline), baseline))
	require.Equal(t, "helm-original", string(baseline.Data["upstream-api-key"]))
	require.Equal(t, "master-original", string(baseline.Data["master-key"]))
	status, err := GetInterrogationSettingsStatus(ctx, c)
	require.NoError(t, err)
	require.Equal(t, "APPLYING", status.State)
	oldName := cfg.Interrogation.CredentialSecretName
	require.NoError(t, UpdateLocalUIConfig(ctx, c, interrogationModelInput(model.InterrogationModelConfigInput{APIKey: StringPtr(" ")})))
	cfg, err = GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	require.Equal(t, oldName, cfg.Interrogation.CredentialSecretName)
	require.NoError(t, UpdateLocalUIConfig(ctx, c, model.LocalUIConfigInput{Interrogation: &model.InterrogationConfigInput{ResetPaths: []string{"interrogation.model.apiKey"}}}))
	status, err = GetInterrogationSettingsStatus(ctx, c)
	require.NoError(t, err)
	cfg, err = GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	require.Empty(t, cfg.Interrogation.CredentialSecretName)
}

type failingAIConfigClient struct{ client.Client }

func (c failingAIConfigClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if obj.GetName() == consts.OdigosLocalUiConfigName {
		return errors.New("simulated ConfigMap write failure")
	}
	return c.Client.Update(ctx, obj, opts...)
}

func TestInterrogationFailedCommitCannotActivateStagedCredential(t *testing.T) {
	ctx := context.Background()
	base := aiTestClient(aiConfigMap(consts.OdigosLocalUiConfigName, "{}"))
	err := UpdateLocalUIConfig(ctx, failingAIConfigClient{base}, interrogationModelInput(model.InterrogationModelConfigInput{Name: StringPtr("gpt-4.1"), APIKey: StringPtr("inactive-key")}))
	require.ErrorContains(t, err, "write failure")
	cfg, err := GetLocalUIConfig(ctx, base)
	require.NoError(t, err)
	require.Nil(t, cfg.Interrogation)
	secrets := &corev1.SecretList{}
	require.NoError(t, base.List(ctx, secrets))
	require.Len(t, secrets.Items, 1, "the uncommitted immutable revision is safe to collect later")
}

func TestInterrogationValidationPrecedesWrites(t *testing.T) {
	tests := []struct {
		name  string
		input *model.InterrogationConfigInput
	}{
		{"blank provider", &model.InterrogationConfigInput{Model: &model.InterrogationModelConfigInput{Provider: StringPtr(" "), APIKey: StringPtr("must-not-be-written")}}},
		{"blank model", &model.InterrogationConfigInput{Model: &model.InterrogationModelConfigInput{Name: StringPtr("")}}},
		{"invalid endpoint", &model.InterrogationConfigInput{Model: &model.InterrogationModelConfigInput{APIBase: StringPtr("file:///tmp/model")}}},
		{"unallowed reset", &model.InterrogationConfigInput{ResetPaths: []string{"interrogation.enabled"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := aiTestClient()
			require.Error(t, UpdateLocalUIConfig(context.Background(), c, model.LocalUIConfigInput{Interrogation: test.input}))
			secrets := &corev1.SecretList{}
			require.NoError(t, c.List(context.Background(), secrets))
			require.Empty(t, secrets.Items)
			cfg, err := GetLocalUIConfig(context.Background(), c)
			require.NoError(t, err)
			require.Nil(t, cfg)
		})
	}
	c := fake.NewClientBuilder().WithObjects(aiConfigMap(consts.OdigosConfigurationName, "{}")).Build()
	enabled := true
	require.ErrorContains(t, UpdateLocalUIConfig(context.Background(), c, model.LocalUIConfigInput{Interrogation: &model.InterrogationConfigInput{AnalysisEnabled: &enabled}}), "Helm provisioning")
}

func TestInterrogationGlobalResetClearsCredentialAndAcknowledgesRevision(t *testing.T) {
	ctx := context.Background()
	c := aiTestClient()
	require.NoError(t, UpdateLocalUIConfig(ctx, c, interrogationModelInput(model.InterrogationModelConfigInput{APIKey: StringPtr("local-key")})))
	cfg, err := GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	old := cfg.Interrogation.CredentialSecretName
	require.NoError(t, ResetLocalUiConfigToFactoryDefaults(ctx, c, "reset-request"))
	cfg, err = GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	require.Equal(t, "reset-request", cfg.Interrogation.UpdateID)
	require.Empty(t, cfg.Interrogation.CredentialSecretName)
	require.Equal(t, old, cfg.Interrogation.PreviousCredentialSecretName)
	require.Nil(t, cfg.Interrogation.Model)
}

func TestInterrogationStatusRejectsFormerLeaderAndWrongRevision(t *testing.T) {
	ctx := context.Background()
	identity := "current-leader"
	duration := int32(15)
	renew := metav1.NewMicroTime(time.Now())
	lease := &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Name: "odigos-insights-leader", Namespace: env.GetCurrentNamespace()}, Spec: coordinationv1.LeaseSpec{HolderIdentity: &identity, LeaseDurationSeconds: &duration, RenewTime: &renew}}
	encode := func(a interrogationAcknowledgement) string {
		raw, err := json.Marshal(a)
		require.NoError(t, err)
		return string(raw)
	}
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: interrogationStatusName, Namespace: env.GetCurrentNamespace()}, Data: map[string]string{
		"proxy":    encode(interrogationAcknowledgement{UpdateID: "request", Revision: "revision", State: "READY", ModelHash: common.InterrogationModelHash(&common.InterrogationModelConfiguration{Provider: StringPtr("openai"), Name: StringPtr("gpt-4o-mini"), ApiBase: StringPtr("https://proxy.example.com/v1")})}),
		"analysis": encode(interrogationAcknowledgement{UpdateID: "request", Revision: "revision", State: "PAUSED", HolderIdentity: "former-leader"}),
	}}
	c := aiTestClient(lease, cm, aiConfigMap(consts.OdigosEffectiveConfigName, interrogationEffectiveConfig+"  analysisEnabled: false\n"), aiConfigMap(consts.OdigosLocalUiConfigName, "interrogation:\n  updateId: request\n"))
	status, err := GetInterrogationSettingsStatus(ctx, c)
	require.NoError(t, err)
	require.Equal(t, "APPLYING", status.State)
	cm.Data["analysis"] = encode(interrogationAcknowledgement{UpdateID: "request", Revision: "revision", State: "PAUSED", HolderIdentity: identity})
	require.NoError(t, c.Update(ctx, cm))
	status, err = GetInterrogationSettingsStatus(ctx, c)
	require.NoError(t, err)
	require.Equal(t, "PAUSED", status.State)
	require.Equal(t, "request", status.ObservedUpdateID)
	cm.Data["analysis"] = encode(interrogationAcknowledgement{UpdateID: "request", Revision: "older-revision", State: "RUNNING", HolderIdentity: identity})
	require.NoError(t, c.Update(ctx, cm))
	status, err = GetInterrogationSettingsStatus(ctx, c)
	require.NoError(t, err)
	require.Equal(t, "APPLYING", status.State)
}

func TestInterrogationStatusErrorsAndExpiredAcknowledgements(t *testing.T) {
	for _, test := range []struct {
		name, proxyState, analysisState, want string
		renewAge, observedAge                 time.Duration
	}{
		{name: "proxy error", proxyState: "ERROR", want: "ERROR"},
		{name: "analysis error", proxyState: "READY", analysisState: "ERROR", want: "ERROR"},
		{name: "expired leader", proxyState: "READY", analysisState: "PAUSED", renewAge: time.Minute, want: "APPLYING"},
		{name: "stale acknowledgement", proxyState: "READY", analysisState: "PAUSED", observedAge: 2 * time.Minute, want: "APPLYING"},
	} {
		t.Run(test.name, func(t *testing.T) {
			identity, duration := "current-leader", int32(15)
			renew := metav1.NewMicroTime(time.Now().Add(-test.renewAge))
			lease := &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Name: "odigos-insights-leader", Namespace: env.GetCurrentNamespace()}, Spec: coordinationv1.LeaseSpec{HolderIdentity: &identity, LeaseDurationSeconds: &duration, RenewTime: &renew}}
			encode := func(a interrogationAcknowledgement) string {
				raw, err := json.Marshal(a)
				require.NoError(t, err)
				return string(raw)
			}
			cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: interrogationStatusName, Namespace: env.GetCurrentNamespace()}, Data: map[string]string{
				"proxy":    encode(interrogationAcknowledgement{UpdateID: "request", Revision: "revision", State: test.proxyState, ModelHash: common.InterrogationModelHash(&common.InterrogationModelConfiguration{Provider: StringPtr("openai"), Name: StringPtr("gpt-4o-mini"), ApiBase: StringPtr("https://proxy.example.com/v1")})}),
				"analysis": encode(interrogationAcknowledgement{UpdateID: "request", Revision: "revision", State: test.analysisState, HolderIdentity: identity, ObservedAt: time.Now().Add(-test.observedAge).UTC().Format(time.RFC3339Nano)}),
			}}
			c := aiTestClient(lease, cm, aiConfigMap(consts.OdigosEffectiveConfigName, interrogationEffectiveConfig+"  analysisEnabled: false\n"), aiConfigMap(consts.OdigosLocalUiConfigName, "interrogation:\n  updateId: request\n"))
			status, err := GetInterrogationSettingsStatus(context.Background(), c)
			require.NoError(t, err)
			require.Equal(t, test.want, status.State)
			require.Equal(t, "request", status.UpdateID)
			require.Empty(t, status.ObservedUpdateID)
			if test.want == "ERROR" {
				require.NotNil(t, status.Error)
			}
		})
	}
}

type credentialReadCounter struct {
	client.Client
	reads int
}

func (c *credentialReadCounter) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if _, ok := obj.(*corev1.Secret); ok {
		c.reads++
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

func TestInterrogationAPIKeyRespectsReadonlyAndCredentialOwnership(t *testing.T) {
	for _, test := range []struct {
		name, config, localRef, want string
		invalid, wantError           bool
		wantReads                    int
	}{
		{name: "Helm credential", config: interrogationEffectiveConfig, want: "helm-key", wantReads: 1},
		{name: "UI credential", config: interrogationEffectiveConfig, localRef: "odigos-local-ui-ai-test", want: "ui-key", wantReads: 1},
		{name: "readonly", config: "uiMode: readonly\n" + interrogationEffectiveConfig},
		{name: "unprovisioned", config: "{}"},
		{name: "missing configuration", wantError: true},
		{name: "invalid local reference", config: interrogationEffectiveConfig, localRef: "another-secret", wantError: true},
		{name: "invalid local revision", config: interrogationEffectiveConfig, localRef: "odigos-local-ui-ai-test", invalid: true, wantError: true, wantReads: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			immutable := true
			label := interrogationCredentialLabelValue
			if test.invalid {
				label = "other-owner"
			}
			objects := []client.Object{
				aiConfigMap(consts.OdigosEffectiveConfigName, test.config),
				&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "odigos-litellm", Namespace: env.GetCurrentNamespace()}, Data: map[string][]byte{"upstream-api-key": []byte("helm-key"), "master-key": []byte("never-return-master")}},
				&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "odigos-local-ui-ai-test", Namespace: env.GetCurrentNamespace(), Labels: map[string]string{interrogationCredentialLabel: label}}, Immutable: &immutable, Data: map[string][]byte{"upstream-api-key": []byte("ui-key")}},
			}
			if test.localRef != "" {
				objects = append(objects, aiConfigMap(consts.OdigosLocalUiConfigName, "interrogation:\n  credentialSecretName: "+test.localRef+"\n"))
			}
			c := &credentialReadCounter{Client: aiTestClient(objects...)}
			got, err := GetInterrogationAPIKey(context.Background(), c)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if test.want == "" {
				require.Nil(t, got)
			} else {
				require.NotNil(t, got)
				require.Equal(t, test.want, *got)
			}
			require.Equal(t, test.wantReads, c.reads)
		})
	}
}
