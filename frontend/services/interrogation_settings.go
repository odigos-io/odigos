package services

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	interrogationCredentialPrefix     = k8sconsts.OdigosLocalUIAICredentialSecretPrefix
	interrogationCredentialLabel      = k8sconsts.OdigosLocalUIAIConfigSourceLabel
	interrogationCredentialLabelValue = k8sconsts.OdigosLocalUIAIConfigSourceValue
)

// InterrogationAPIKeyProvenancePath is reported only while the local UI selects
// its own credential revision.
const InterrogationAPIKeyProvenancePath = "interrogation.model.apiKey"

var interrogationResetPaths = map[string]bool{
	"interrogation.model.provider":    true,
	"interrogation.model.name":        true,
	"interrogation.model.apiBase":     true,
	InterrogationAPIKeyProvenancePath: true,
}

func validateInterrogationInput(ctx context.Context, c client.Client, input *model.InterrogationConfigInput) error {
	base, err := GetHelmDeploymentConfig(ctx, c)
	if err != nil {
		return err
	}
	if base == nil || !base.InterrogationEnabled() {
		return fmt.Errorf("AI settings require Helm provisioning: insights.enabled=true and insights.interrogation.enabled=true with an enterprise token")
	}
	for _, path := range input.ResetPaths {
		if !interrogationResetPaths[path] {
			return fmt.Errorf("unsupported AI settings reset path")
		}
	}
	return nil
}

func stageInterrogationCredential(ctx context.Context, c client.Client, input *model.InterrogationConfigInput) (string, error) {
	if input.Model == nil || input.Model.APIKey == nil || strings.TrimSpace(*input.Model.APIKey) == "" {
		return "", nil
	}
	// Validate the candidate before creating even an inactive credential revision.
	local, err := GetLocalUIConfig(ctx, c)
	if err != nil {
		return "", err
	}
	if local == nil {
		local = &common.OdigosConfiguration{}
	}
	applyInterrogationInput(local, input, nil)
	if err := validateResolvedInterrogation(ctx, c, local, input); err != nil {
		return "", err
	}
	owner := &corev1.ConfigMap{}
	ns := env.GetCurrentNamespace()
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosConfigurationName}, owner); err != nil {
		return "", err
	}
	immutable := true
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: interrogationCredentialPrefix, Namespace: ns,
			Labels:          map[string]string{interrogationCredentialLabel: interrogationCredentialLabelValue},
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ConfigMap", Name: owner.Name, UID: owner.UID}},
		},
		Immutable: &immutable,
		Data:      map[string][]byte{"upstream-api-key": []byte(strings.TrimSpace(*input.Model.APIKey))},
	}
	if err := c.Create(ctx, secret); err != nil {
		return "", err
	}
	// A failed ConfigMap commit deliberately leaves this revision inactive. The
	// controller later collects unreferenced revisions; an uncertain write must
	// never delete a Secret that the API server may already have committed.
	return secret.Name, nil
}

func applyStagedCredential(cfg *common.OdigosConfiguration, name string) {
	if name == "" {
		return
	}
	if cfg.Interrogation == nil {
		cfg.Interrogation = &common.InterrogationConfiguration{}
	}
	if cfg.Interrogation.CredentialSecretName != "" {
		cfg.Interrogation.PreviousCredentialSecretName = cfg.Interrogation.CredentialSecretName
	}
	cfg.Interrogation.CredentialSecretName = name
}

func validateResolvedInterrogation(ctx context.Context, c client.Client, local *common.OdigosConfiguration, input *model.InterrogationConfigInput) error {
	if input == nil {
		return nil
	}
	base, err := GetHelmDeploymentConfig(ctx, c)
	if err != nil {
		return err
	}
	if base == nil || !base.InterrogationEnabled() {
		return fmt.Errorf("interrogation is not provisioned")
	}
	resolved := base.Interrogation.DeepCopy()
	remote, err := GetRemoteConfig(ctx, c)
	if err != nil {
		return err
	}
	if remote != nil {
		common.MergeInterrogationSettings(resolved, remote.Interrogation)
	}
	common.MergeInterrogationSettings(resolved, local.Interrogation)
	// Older charts omit the model because they rendered LiteLLM directly.
	provider, name := "openai", "gpt-4o-mini"
	endpoint := ""
	if m := resolved.Model; m != nil {
		if m.Provider != nil {
			provider = strings.TrimSpace(*m.Provider)
		}
		if m.Name != nil {
			name = strings.TrimSpace(*m.Name)
		}
		if m.ApiBase != nil {
			endpoint = strings.TrimSpace(*m.ApiBase)
		}
	}
	if provider == "" || name == "" || strings.Contains(provider, "/") {
		return fmt.Errorf("AI provider and model must not be blank")
	}
	if endpoint != "" {
		u, err := url.Parse(endpoint)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return fmt.Errorf("AI endpoint must be an HTTP(S) URL without embedded credentials")
		}
	}
	return nil
}

func GetInterrogationAPIKey(ctx context.Context, c client.Client) (*string, error) {
	cfg, err := GetEffectiveConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("AI settings configuration is unavailable")
	}
	if cfg.UiMode == common.UiModeReadonly || !cfg.InterrogationEnabled() {
		return nil, nil
	}
	local, err := GetLocalUIConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	name := "odigos-litellm"
	localOverride := local != nil && local.Interrogation != nil && local.Interrogation.CredentialSecretName != ""
	if localOverride {
		name = local.Interrogation.CredentialSecretName
		if !strings.HasPrefix(name, interrogationCredentialPrefix) {
			return nil, fmt.Errorf("AI credential reference is invalid")
		}
	}
	secret := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: env.GetCurrentNamespace(), Name: name}, secret); err != nil {
		if apierrors.IsNotFound(err) && !localOverride {
			return nil, nil
		}
		return nil, fmt.Errorf("AI credential could not be read")
	}
	if localOverride && (secret.Immutable == nil || !*secret.Immutable || secret.Labels[interrogationCredentialLabel] != interrogationCredentialLabelValue) {
		return nil, fmt.Errorf("AI credential revision is invalid")
	}
	key := string(secret.Data["upstream-api-key"])
	if key == "" {
		return nil, nil
	}
	return &key, nil
}
