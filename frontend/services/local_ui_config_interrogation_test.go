package services

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/require"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const interrogationEffectiveConfig = `
interrogation:
  enabled: true
  model:
    provider: openai
    name: gpt-4o-mini
    apiBase: https://proxy.example.com/v1
`

func interrogationModelInput(m model.InterrogationModelConfigInput) model.LocalUIConfigInput {
	return model.LocalUIConfigInput{Interrogation: &model.InterrogationConfigInput{Model: &m}}
}

func effectiveInterrogationModel() *common.OdigosConfiguration {
	enabled := true
	return &common.OdigosConfiguration{Interrogation: &common.InterrogationConfiguration{
		Enabled: &enabled,
		Model:   &common.InterrogationModelConfiguration{Provider: "openai", Name: "gpt-4o-mini", ApiBase: "https://proxy.example.com/v1"},
	}}
}

func liteLLMSecret(data map[string][]byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.OdigosLiteLLMName, Namespace: env.GetCurrentNamespace()},
		Data:       data,
	}
}

func setLiteLLMTestClient(t *testing.T, objs ...runtime.Object) {
	t.Helper()
	kube.SetDefaultClient(&kube.Client{Interface: k8sfake.NewSimpleClientset(objs...)})
	t.Cleanup(func() { kube.SetDefaultClient(nil) })
}

func readLiteLLMSecret(t *testing.T) map[string][]byte {
	t.Helper()
	secret, err := kube.DefaultClient.CoreV1().Secrets(env.GetCurrentNamespace()).
		Get(context.Background(), k8sconsts.OdigosLiteLLMName, metav1.GetOptions{})
	require.NoError(t, err)
	return secret.Data
}

func TestApplyInterrogationInput_SnapshotsEffectiveModel(t *testing.T) {
	cfg := &common.OdigosConfiguration{}
	applyLocalUiConfigInput(cfg, interrogationModelInput(model.InterrogationModelConfigInput{Name: StringPtr("gpt-4.1")}), effectiveInterrogationModel())

	require.Equal(t, &common.InterrogationModelConfiguration{
		Provider: "openai",
		Name:     "gpt-4.1",
		ApiBase:  "https://proxy.example.com/v1",
	}, cfg.Interrogation.Model)
	require.Nil(t, cfg.Interrogation.Enabled)
}

func TestApplyInterrogationInput_SnapshotsOverlayModel(t *testing.T) {
	cfg := &common.OdigosConfiguration{Interrogation: &common.InterrogationConfiguration{
		Model: &common.InterrogationModelConfiguration{Provider: "anthropic", Name: "claude-sonnet-4-5"},
	}}
	applyLocalUiConfigInput(cfg, interrogationModelInput(model.InterrogationModelConfigInput{APIBase: StringPtr("https://gateway.example.com")}), effectiveInterrogationModel())

	require.Equal(t, &common.InterrogationModelConfiguration{
		Provider: "anthropic",
		Name:     "claude-sonnet-4-5",
		ApiBase:  "https://gateway.example.com",
	}, cfg.Interrogation.Model)
}

func TestApplyInterrogationInput_EmptyApiBaseClears(t *testing.T) {
	cfg := &common.OdigosConfiguration{}
	applyLocalUiConfigInput(cfg, interrogationModelInput(model.InterrogationModelConfigInput{
		Provider: StringPtr("gemini"),
		APIBase:  StringPtr(""),
	}), effectiveInterrogationModel())

	require.Equal(t, &common.InterrogationModelConfiguration{Provider: "gemini", Name: "gpt-4o-mini"}, cfg.Interrogation.Model)
}

func TestApplyInterrogationInput_ApiKeyOnlyLeavesModelUnset(t *testing.T) {
	cfg := &common.OdigosConfiguration{}
	applyLocalUiConfigInput(cfg, interrogationModelInput(model.InterrogationModelConfigInput{APIKey: StringPtr("sk-test")}), effectiveInterrogationModel())

	require.Nil(t, cfg.Interrogation)
}

func TestSetInterrogationUpstreamApiKey_PreservesOtherKeys(t *testing.T) {
	setLiteLLMTestClient(t, liteLLMSecret(map[string][]byte{
		"master-key": []byte("sk-master"),
		k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey: []byte("sk-old"),
	}))

	err := setInterrogationUpstreamApiKey(context.Background(), env.GetCurrentNamespace(), effectiveInterrogationModel(), "  sk-new\n")
	require.NoError(t, err)

	data := readLiteLLMSecret(t)
	require.Equal(t, "sk-new", string(data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey]))
	require.Equal(t, "sk-master", string(data["master-key"]))
}

func TestSetInterrogationUpstreamApiKey_EmptyKeepsCurrent(t *testing.T) {
	setLiteLLMTestClient(t, liteLLMSecret(map[string][]byte{k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey: []byte("sk-old")}))

	err := setInterrogationUpstreamApiKey(context.Background(), env.GetCurrentNamespace(), effectiveInterrogationModel(), "   ")
	require.NoError(t, err)
	require.Equal(t, "sk-old", string(readLiteLLMSecret(t)[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey]))
}

func TestSetInterrogationUpstreamApiKey_InterrogationDisabled(t *testing.T) {
	setLiteLLMTestClient(t, liteLLMSecret(nil))

	err := setInterrogationUpstreamApiKey(context.Background(), env.GetCurrentNamespace(), &common.OdigosConfiguration{}, "sk-new")
	require.ErrorContains(t, err, "interrogation is not enabled")
	require.Empty(t, readLiteLLMSecret(t))
}

func TestSetInterrogationUpstreamApiKey_SecretMissing(t *testing.T) {
	setLiteLLMTestClient(t)

	err := setInterrogationUpstreamApiKey(context.Background(), env.GetCurrentNamespace(), effectiveInterrogationModel(), "sk-new")
	require.ErrorContains(t, err, "LiteLLM is not deployed")
}

func TestUpdateLocalUIConfig_InterrogationModelAndApiKey(t *testing.T) {
	ns := env.GetCurrentNamespace()
	setLiteLLMTestClient(t, liteLLMSecret(map[string][]byte{"master-key": []byte("sk-master")}))
	c := fake.NewClientBuilder().WithObjects(
		effectiveConfigCM(interrogationEffectiveConfig),
		&v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosConfigurationName, Namespace: ns, UID: "owner-uid"}},
	).Build()

	err := UpdateLocalUIConfig(context.Background(), c, interrogationModelInput(model.InterrogationModelConfigInput{
		Name:   StringPtr("gpt-4.1"),
		APIKey: StringPtr("sk-new"),
	}))
	require.NoError(t, err)

	var cm v1.ConfigMap
	require.NoError(t, c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: consts.OdigosLocalUiConfigName}, &cm))
	raw := cm.Data[consts.OdigosConfigurationFileName]
	require.NotContains(t, raw, "sk-new")

	local := common.OdigosConfiguration{}
	require.NoError(t, yaml.Unmarshal([]byte(raw), &local))
	require.Equal(t, &common.InterrogationModelConfiguration{
		Provider: "openai",
		Name:     "gpt-4.1",
		ApiBase:  "https://proxy.example.com/v1",
	}, local.Interrogation.Model)

	data := readLiteLLMSecret(t)
	require.Equal(t, "sk-new", string(data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey]))
	require.Equal(t, "sk-master", string(data["master-key"]))
}
