package services

import (
	"context"
	"errors"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/require"
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
	c := aiTestClient(baseline)
	input := interrogationModelInput(model.InterrogationModelConfigInput{Name: StringPtr("gpt-4.1"), APIKey: StringPtr("  test-new-key  ")})
	require.NoError(t, UpdateLocalUIConfig(ctx, c, input))
	cfg, err := GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	require.Equal(t, "gpt-4.1", *cfg.Interrogation.Model.Name)
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
	oldName := cfg.Interrogation.CredentialSecretName
	require.NoError(t, UpdateLocalUIConfig(ctx, c, interrogationModelInput(model.InterrogationModelConfigInput{APIKey: StringPtr(" ")})))
	cfg, err = GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	require.Equal(t, oldName, cfg.Interrogation.CredentialSecretName)
	require.NoError(t, UpdateLocalUIConfig(ctx, c, model.LocalUIConfigInput{Interrogation: &model.InterrogationConfigInput{ResetPaths: []string{"interrogation.model.apiKey"}}}))
	cfg, err = GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	require.Empty(t, cfg.Interrogation.CredentialSecretName)
	require.Equal(t, oldName, cfg.Interrogation.PreviousCredentialSecretName)
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
	require.ErrorContains(t, UpdateLocalUIConfig(context.Background(), c, interrogationModelInput(model.InterrogationModelConfigInput{Name: StringPtr("gpt-4.1")})), "Helm provisioning")
}

func TestInterrogationGlobalResetClearsCredentialAndRetainsPrevious(t *testing.T) {
	ctx := context.Background()
	c := aiTestClient()
	require.NoError(t, UpdateLocalUIConfig(ctx, c, interrogationModelInput(model.InterrogationModelConfigInput{APIKey: StringPtr("local-key")})))
	cfg, err := GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	old := cfg.Interrogation.CredentialSecretName
	require.NoError(t, ResetLocalUiConfigToFactoryDefaults(ctx, c))
	cfg, err = GetLocalUIConfig(ctx, c)
	require.NoError(t, err)
	require.Empty(t, cfg.Interrogation.CredentialSecretName)
	require.Equal(t, old, cfg.Interrogation.PreviousCredentialSecretName)
	require.Nil(t, cfg.Interrogation.Model)
}

func TestInterrogationAPIKeyProvenanceTracksLocalCredentialOnly(t *testing.T) {
	ctx := context.Background()
	effective := &common.OdigosConfiguration{}
	remote := aiConfigMap(consts.OdigosRemoteConfigName, "interrogation:\n  credentialSecretName: odigos-local-ui-ai-forged\n")
	for _, test := range []struct {
		name, local, want string
	}{
		{name: "Helm credential", local: "interrogation:\n  model:\n    name: gpt-4.1\n"},
		{name: "no local config"},
		{name: "local credential", local: "interrogation:\n  credentialSecretName: odigos-local-ui-ai-test\n", want: consts.OdigosLocalUiConfigName},
		{name: "previous credential only", local: "interrogation:\n  previousCredentialSecretName: odigos-local-ui-ai-old\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			objects := []client.Object{remote}
			if test.local != "" {
				objects = append(objects, aiConfigMap(consts.OdigosLocalUiConfigName, test.local))
			}
			provenance, err := ComputeProvenance(ctx, aiTestClient(objects...), effective)
			require.NoError(t, err)
			got, ok := provenance[InterrogationAPIKeyProvenancePath]
			require.Equal(t, test.want != "", ok)
			require.Equal(t, test.want, got)
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
