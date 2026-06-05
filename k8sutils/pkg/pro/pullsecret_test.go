package pro

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func TestEnterpriseRegistryPullSecretLabels(t *testing.T) {
	t.Parallel()

	odigosLabels := EnterpriseRegistryPullSecretLabels(false)
	if odigosLabels[k8sconsts.OdigosSystemLabelKey] != k8sconsts.OdigosSystemLabelValue {
		t.Fatalf("expected system-object label for odigos namespace secret")
	}
	if _, ok := odigosLabels[k8sconsts.OdigosSystemLabelCentralKey]; ok {
		t.Fatalf("did not expect central-system-object label for odigos namespace secret")
	}

	centralLabels := EnterpriseRegistryPullSecretLabels(true)
	if centralLabels[k8sconsts.OdigosSystemLabelKey] != k8sconsts.OdigosSystemLabelValue {
		t.Fatalf("expected system-object label for central secret")
	}
	if centralLabels[k8sconsts.OdigosSystemLabelCentralKey] != k8sconsts.OdigosSystemLabelValue {
		t.Fatalf("expected central-system-object label for central secret")
	}
}
