package pro

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func TestEnterpriseRegistryPullSecretLabels(t *testing.T) {
	t.Parallel()

	labels := EnterpriseRegistryPullSecretLabels()
	if labels[k8sconsts.OdigosSystemLabelKey] != k8sconsts.OdigosSystemLabelValue {
		t.Fatalf("expected system-object label for pull secret")
	}
	if _, ok := labels[k8sconsts.OdigosSystemLabelCentralKey]; ok {
		t.Fatalf("did not expect central-system-object label for odigos namespace secret")
	}
}
