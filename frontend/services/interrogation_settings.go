package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	interrogationCredentialPrefix     = k8sconsts.OdigosLocalUIAICredentialSecretPrefix
	interrogationStatusName           = k8sconsts.OdigosInterrogationStatusConfigMapName
	interrogationCredentialLabel      = k8sconsts.OdigosLocalUIAIConfigSourceLabel
	interrogationCredentialLabelValue = k8sconsts.OdigosLocalUIAIConfigSourceValue
)

var interrogationResetPaths = map[string]bool{
	"interrogation.analysisEnabled": true,
	"interrogation.model.provider":  true,
	"interrogation.model.name":      true,
	"interrogation.model.apiBase":   true,
	"interrogation.model.apiKey":    true,
}

func validateInterrogationInput(ctx context.Context, c client.Client, input *model.InterrogationConfigInput) error {
	base, err := GetHelmDeploymentConfig(ctx, c)
	if err != nil {
		return err
	}
	if base == nil || !base.InterrogationEnabled() {
		return fmt.Errorf("AI settings require Helm provisioning: insights.enabled=true and insights.interrogation.enabled=true with an enterprise token")
	}
	if input.UpdateID != nil && len(*input.UpdateID) > 128 {
		return fmt.Errorf("AI settings update identifier is too long")
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

type interrogationAcknowledgement struct {
	UpdateID         string `json:"updateId"`
	Revision         string `json:"revision"`
	ModelHash        string `json:"modelHash"`
	APIKeyConfigured bool   `json:"apiKeyConfigured"`
	APIKeySource     string `json:"apiKeySource"`
	ObservedAt       string `json:"observedAt"`
	State            string `json:"state"`
	Error            string `json:"error"`
	HolderIdentity   string `json:"holderIdentity"`
}

// GetInterrogationSettingsStatus separates a persisted preference from the
// configuration actually acknowledged by the proxy and elected analysis worker.
func GetInterrogationSettingsStatus(ctx context.Context, c client.Client) (*model.InterrogationSettingsStatus, error) {
	status := &model.InterrogationSettingsStatus{State: "NOT_INSTALLED", APIKeySource: "NONE"}
	base, err := GetHelmDeploymentConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	status.Provisioned = base != nil && base.InterrogationEnabled()
	if !status.Provisioned {
		return status, nil
	}
	status.State = "APPLYING"
	local, err := GetLocalUIConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	if local != nil && local.Interrogation != nil {
		status.UpdateID = local.Interrogation.UpdateID
	}
	ns := env.GetCurrentNamespace()
	cm := &corev1.ConfigMap{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: interrogationStatusName}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return status, nil
		}
		return nil, err
	}
	var proxy, analysis interrogationAcknowledgement
	if err := json.Unmarshal([]byte(cm.Data["proxy"]), &proxy); err != nil {
		return status, nil
	}
	effective, err := GetEffectiveConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	if effective == nil || effective.Interrogation == nil || proxy.ModelHash != common.InterrogationModelHash(effective.Interrogation.Model) {
		return status, nil
	}
	if proxy.UpdateID != status.UpdateID {
		return status, nil
	}
	status.APIKeyConfigured = proxy.APIKeyConfigured
	status.APIKeySource = proxy.APIKeySource
	if status.APIKeySource != "HELM" && status.APIKeySource != "LOCAL_UI" {
		status.APIKeySource = "NONE"
	}
	if proxy.State == "ERROR" {
		status.State = "ERROR"
		message := "AI proxy configuration could not be applied. Check the instrumentor logs."
		status.Error = &message
		return status, nil
	}
	if proxy.State != "READY" {
		return status, nil
	}
	if err := json.Unmarshal([]byte(cm.Data["analysis"]), &analysis); err != nil {
		return status, nil
	}
	if analysis.UpdateID != status.UpdateID || analysis.Revision != proxy.Revision {
		return status, nil
	}
	// A former leader's acknowledgement cannot make the UI report success.
	lease := &coordinationv1.Lease{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "odigos-insights-leader"}, lease); err != nil {
		if apierrors.IsNotFound(err) {
			return status, nil
		}
		return nil, err
	}
	if lease.Spec.HolderIdentity == nil || *lease.Spec.HolderIdentity != analysis.HolderIdentity || lease.Spec.RenewTime == nil || lease.Spec.LeaseDurationSeconds == nil || time.Since(lease.Spec.RenewTime.Time) > time.Duration(*lease.Spec.LeaseDurationSeconds)*time.Second {
		return status, nil
	}
	if effective.InterrogationAnalysisEnabled() && analysis.State == "PAUSED" || !effective.InterrogationAnalysisEnabled() && analysis.State == "RUNNING" {
		return status, nil
	}
	if analysis.ObservedAt != "" {
		observed, err := time.Parse(time.RFC3339Nano, analysis.ObservedAt)
		if err != nil || time.Since(observed) > time.Minute {
			return status, nil
		}
	}
	switch analysis.State {
	case "RUNNING", "PAUSED":
		status.State = analysis.State
		status.ObservedUpdateID = analysis.UpdateID
	case "ERROR":
		status.State = "ERROR"
		message := "AI analysis settings could not be applied. Check the Insights logs."
		status.Error = &message
	}
	return status, nil
}
