/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/cli/pkg/helm"
	"github.com/odigos-io/odigos/common"
	operatorv1alpha1 "github.com/odigos-io/odigos/operator/api/v1alpha1"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// relatedImageEnvVars maps helm chart component names to their corresponding
// RELATED_IMAGE_* environment variable names used in OpenShift operator deployments.
// These env vars contain the full image URLs for certified container images.
var relatedImageEnvVars = map[string]string{
	"autoscaler":              "RELATED_IMAGE_AUTOSCALER",
	"cli":                     "RELATED_IMAGE_CLI",
	"collector":               "RELATED_IMAGE_COLLECTOR",
	"ui":                      "RELATED_IMAGE_FRONTEND",
	"instrumentor":            "RELATED_IMAGE_INSTRUMENTOR",
	"enterprise-instrumentor": "RELATED_IMAGE_ENTERPRISE_INSTRUMENTOR",
	"odiglet":                 "RELATED_IMAGE_ODIGLET",
	"enterprise-odiglet":      "RELATED_IMAGE_ENTERPRISE_ODIGLET",
	"scheduler":               "RELATED_IMAGE_SCHEDULER",
}

// restClientGetter implements genericclioptions.RESTClientGetter using an existing rest.Config.
// This allows us to use the Helm SDK with the controller's existing k8s configuration.
type restClientGetter struct {
	config    *rest.Config
	namespace string
}

func newRESTClientGetter(config *rest.Config, namespace string) *restClientGetter {
	return &restClientGetter{
		config:    config,
		namespace: namespace,
	}
}

func (r *restClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.config, nil
}

func (r *restClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := r.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(discoveryClient), nil
}

func (r *restClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient), nil
}

func (r *restClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	// Return a minimal ClientConfig that provides the namespace
	return &simpleClientConfig{namespace: r.namespace}
}

// simpleClientConfig implements clientcmd.ClientConfig for basic namespace support
type simpleClientConfig struct {
	namespace string
}

func (c *simpleClientConfig) RawConfig() (clientcmdapi.Config, error) {
	return clientcmdapi.Config{}, nil
}

func (c *simpleClientConfig) ClientConfig() (*rest.Config, error) {
	return nil, nil
}

func (c *simpleClientConfig) Namespace() (string, bool, error) {
	return c.namespace, true, nil
}

func (c *simpleClientConfig) ConfigAccess() clientcmd.ConfigAccess {
	return nil
}

// helmLogger implements helm's DebugLog function to use our logr logger
type helmLogger struct {
	logger logr.Logger
	mu     sync.Mutex
	stats  helmStats
}

type helmStats struct {
	created   int
	updated   int
	deleted   int
	unchanged int
}

func newHelmLogger(logger logr.Logger) *helmLogger {
	return &helmLogger{logger: logger}
}

func (h *helmLogger) logFunc(format string, v ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Count operations without printing every line
	msg := ""
	if len(v) > 0 {
		msg = format
	}
	switch {
	case strings.HasPrefix(msg, "creating"):
		h.stats.created++
	case strings.HasPrefix(msg, "Patch"):
		h.stats.updated++
	case strings.HasPrefix(msg, "Deleting"), strings.HasPrefix(msg, "Starting delete"):
		h.stats.deleted++
	case strings.HasPrefix(msg, "no changes"):
		h.stats.unchanged++
	}
}

func (h *helmLogger) printSummary() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.logger.Info("Helm operation summary",
		"created", h.stats.created,
		"updated", h.stats.updated,
		"deleted", h.stats.deleted,
		"unchanged", h.stats.unchanged)

	// Reset counters
	h.stats = helmStats{}
}

// helmInstall performs a Helm install or upgrade of Odigos using the shared CLI helm package
func helmInstall(config *rest.Config, namespace string, odigos *operatorv1alpha1.Odigos, version string, openshiftEnabled bool, logger logr.Logger) error {
	helmLog := newHelmLogger(logger)
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(newRESTClientGetter(config, namespace), namespace, "secret", helmLog.logFunc); err != nil {
		return err
	}

	// Load the embedded chart from CLI package
	chartVersion := strings.TrimPrefix(version, "v")
	ch, err := helm.LoadEmbeddedChart(chartVersion, "odigos")
	if err != nil {
		return fmt.Errorf("failed to load embedded chart: %w", err)
	}

	// Convert Odigos CR spec to Helm values
	vals := odigosSpecToHelmValues(odigos, openshiftEnabled)

	// Set the image tag from version
	if _, ok := vals["image"]; !ok {
		vals["image"] = map[string]interface{}{}
	}
	if imgVals, ok := vals["image"].(map[string]interface{}); ok {
		if _, hasTag := imgVals["tag"]; !hasTag || imgVals["tag"] == "" {
			imgVals["tag"] = version
		}
	}

	// Use shared install or upgrade logic from CLI package
	// CreateNamespace is false because the operator runs in the namespace where Odigos will be installed
	result, err := helm.InstallOrUpgrade(actionConfig, ch, vals, namespace, helm.DefaultReleaseName, helm.InstallOrUpgradeOptions{
		CreateNamespace:      false,
		ResetThenReuseValues: true,
	})
	if err != nil {
		return err
	}

	helmLog.printSummary()
	if result.Installed {
		logger.Info("Helm install completed", "release", helm.DefaultReleaseName, "namespace", namespace, "chartVersion", ch.Metadata.Version)
	} else {
		logger.Info("Helm upgrade completed", "release", helm.DefaultReleaseName, "namespace", namespace, "chartVersion", ch.Metadata.Version)
	}
	return nil
}

// helmUninstall performs a Helm uninstall of Odigos using the shared CLI helm package
func helmUninstall(config *rest.Config, namespace string, logger logr.Logger) error {
	helmLog := newHelmLogger(logger)
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(newRESTClientGetter(config, namespace), namespace, "secret", helmLog.logFunc); err != nil {
		return err
	}

	// Use shared uninstall logic from CLI package
	res, err := helm.RunUninstall(actionConfig, helm.DefaultReleaseName)
	if err != nil {
		return err
	}

	if res == nil {
		// Release was not found, already uninstalled
		logger.Info("Helm release not found, considering already uninstalled", "release", helm.DefaultReleaseName)
		return nil
	}

	helmLog.printSummary()
	logger.Info("Helm uninstall completed", "release", helm.DefaultReleaseName, "namespace", namespace)
	return nil
}

// odigosSpecToHelmValues converts the Odigos CR spec to Helm values
func odigosSpecToHelmValues(odigos *operatorv1alpha1.Odigos, openshiftEnabled bool) map[string]interface{} {
	vals := make(map[string]interface{})

	// OnPremToken
	if odigos.Spec.OnPremToken != "" {
		vals["onPremToken"] = odigos.Spec.OnPremToken
	}

	// UI settings
	ui := make(map[string]interface{})
	if odigos.Spec.UIMode != "" {
		// "normal" is deprecated, map to "default"
		uiMode := string(odigos.Spec.UIMode)
		if uiMode == "normal" {
			uiMode = string(common.UiModeDefault)
		}
		ui["uiMode"] = uiMode
	}
	if len(ui) > 0 {
		vals["ui"] = ui
	}

	// Telemetry
	telemetry := make(map[string]interface{})
	telemetry["enabled"] = odigos.Spec.TelemetryEnabled
	vals["telemetry"] = telemetry

	// Ignored namespaces
	if len(odigos.Spec.IgnoredNamespaces) > 0 {
		vals["ignoredNamespaces"] = odigos.Spec.IgnoredNamespaces
	}

	// Ignored containers
	if len(odigos.Spec.IgnoredContainers) > 0 {
		vals["ignoredContainers"] = odigos.Spec.IgnoredContainers
	}

	// Profiles
	if len(odigos.Spec.Profiles) > 0 {
		profiles := make([]string, len(odigos.Spec.Profiles))
		for i, p := range odigos.Spec.Profiles {
			profiles[i] = string(p)
		}
		vals["profiles"] = profiles
	}

	// Image prefix
	if odigos.Spec.ImagePrefix != "" {
		vals["imagePrefix"] = odigos.Spec.ImagePrefix
	}

	// Instrumentor settings
	instrumentor := make(map[string]interface{})

	// AgentEnvVarsInjectionMethod
	if odigos.Spec.AgentEnvVarsInjectionMethod != "" {
		instrumentor["agentEnvVarsInjectionMethod"] = string(odigos.Spec.AgentEnvVarsInjectionMethod)
	}

	// MountMethod
	if odigos.Spec.MountMethod != "" {
		instrumentor["mountMethod"] = string(odigos.Spec.MountMethod)
	}

	// SkipWebhookIssuerCreation
	if odigos.Spec.SkipWebhookIssuerCreation {
		instrumentor["skipWebhookIssuerCreation"] = true
	}

	if len(instrumentor) > 0 {
		vals["instrumentor"] = instrumentor
	}

	// Node selector (global)
	if len(odigos.Spec.NodeSelector) > 0 {
		vals["nodeSelector"] = odigos.Spec.NodeSelector
	}

	// Pod Security Policy
	if odigos.Spec.PodSecurityPolicy {
		psp := make(map[string]interface{})
		psp["enabled"] = true
		vals["psp"] = psp
	}

	// OpenShift
	openshift := make(map[string]interface{})
	openshift["enabled"] = openshiftEnabled
	vals["openshift"] = openshift

	// Per-component image overrides from RELATED_IMAGE_* env vars (used in OpenShift)
	// These env vars are set by the operator deployment and contain full image URLs
	// for certified container images from the Red Hat registry.
	images := getRelatedImageOverrides()
	if len(images) > 0 {
		vals["images"] = images
	}

	return vals
}

// getRelatedImageOverrides returns a map of component names to full image URLs
// from the RELATED_IMAGE_* environment variables. These are used in OpenShift
// operator deployments to specify certified container images.
func getRelatedImageOverrides() map[string]interface{} {
	images := make(map[string]interface{})
	for component, envVar := range relatedImageEnvVars {
		if imageURL := os.Getenv(envVar); imageURL != "" {
			images[component] = imageURL
		}
	}

	// TODO: Remove this hardcoded fallback once CLI image is properly configured
	// Temporarily hardcode CLI image if not set via env var
	if _, ok := images["cli"]; !ok {
		images["cli"] = "registry.odigos.io/odigos-cli-ubi9:v1.14.0"
	}

	return images
}
