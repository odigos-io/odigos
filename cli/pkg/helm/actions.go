package helm

import (
	"errors"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

const (
	// DefaultReleaseName is the default Helm release name for Odigos
	DefaultReleaseName = "odigos"
	// DefaultNamespace is the default namespace for Odigos installation
	DefaultNamespace = "odigos-system"
)

// InstallOrUpgradeResult contains the result of an install or upgrade operation
type InstallOrUpgradeResult struct {
	Release   *release.Release
	Installed bool // true if this was a fresh install, false if upgrade
}

// InstallOrUpgrade performs a Helm install or upgrade of the given chart.
// It first checks if the release exists, and performs install or upgrade accordingly.
// This is the shared logic used by both CLI and operator.
func InstallOrUpgrade(actionConfig *action.Configuration, ch *chart.Chart, vals map[string]interface{}, namespace, releaseName string, resetThenReuseValues bool) (*InstallOrUpgradeResult, error) {
	get := action.NewGet(actionConfig)
	_, getErr := get.Run(releaseName)
	if getErr != nil {
		if errors.Is(getErr, driver.ErrReleaseNotFound) {
			// Release does not exist → install
			rel, err := RunInstall(actionConfig, ch, vals, namespace, releaseName)
			if err != nil {
				return nil, err
			}
			return &InstallOrUpgradeResult{Release: rel, Installed: true}, nil
		}
		return nil, getErr // Some other error
	}

	// Release exists → upgrade
	rel, err := RunUpgrade(actionConfig, ch, vals, namespace, releaseName, resetThenReuseValues)
	if err != nil {
		return nil, err
	}
	return &InstallOrUpgradeResult{Release: rel, Installed: false}, nil
}

// RunInstall performs a fresh Helm installation
func RunInstall(actionConfig *action.Configuration, ch *chart.Chart, vals map[string]interface{}, namespace, releaseName string) (*release.Release, error) {
	install := action.NewInstall(actionConfig)
	install.ReleaseName = releaseName
	install.Namespace = namespace
	install.CreateNamespace = true
	install.ChartPathOptions.Version = ch.Metadata.Version
	return install.Run(ch, vals)
}

// RunUpgrade performs a Helm upgrade on an existing release
func RunUpgrade(actionConfig *action.Configuration, ch *chart.Chart, vals map[string]interface{}, namespace, releaseName string, resetThenReuseValues bool) (*release.Release, error) {
	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = namespace
	upgrade.Install = false // we handle install fallback ourselves
	upgrade.ChartPathOptions.Version = ch.Metadata.Version
	upgrade.ResetThenReuseValues = resetThenReuseValues
	return upgrade.Run(releaseName, ch, vals)
}

// RunUninstall performs a Helm uninstall of the given release.
// Returns nil, nil if the release was not found (already uninstalled).
func RunUninstall(actionConfig *action.Configuration, releaseName string) (*release.UninstallReleaseResponse, error) {
	uninstall := action.NewUninstall(actionConfig)
	res, err := uninstall.Run(releaseName)
	if err != nil {
		// If release not found, consider it already uninstalled
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

// ReleaseExists checks if a Helm release exists
func ReleaseExists(actionConfig *action.Configuration, releaseName string) (bool, error) {
	get := action.NewGet(actionConfig)
	_, err := get.Run(releaseName)
	if err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// FormatInstallOrUpgradeMessage returns a formatted message for the result
func FormatInstallOrUpgradeMessage(result *InstallOrUpgradeResult, chartVersion string) string {
	if result.Installed {
		return fmt.Sprintf("Installed release %q in namespace %q (chart version: %s)",
			result.Release.Name, result.Release.Namespace, chartVersion)
	}
	return fmt.Sprintf("Upgraded release %q in namespace %q (chart version: %s)",
		result.Release.Name, result.Release.Namespace, chartVersion)
}
