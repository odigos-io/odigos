package nodecollector

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestUpdateNodeCollectorLocalTrafficSvcPreservesAllocatedClusterIP(t *testing.T) {
	local := corev1.ServiceInternalTrafficPolicyLocal
	existing := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            k8sconsts.OdigosNodeCollectorLocalTrafficServiceName,
			Namespace:       "odigos",
			ResourceVersion: "1",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:             "10.96.1.50",
			ClusterIPs:            []string{"10.96.1.50"},
			InternalTrafficPolicy: &local,
			IPFamilies:            []corev1.IPFamily{corev1.IPv4Protocol},
			IPFamilyPolicy:        ptr.To(corev1.IPFamilyPolicySingleStack),
			Ports: []corev1.ServicePort{
				{Name: "otlp", Protocol: "TCP", Port: 4317, TargetPort: intstr.FromInt(4317)},
			},
			Selector: map[string]string{
				k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	modified := existing.DeepCopy()
	updateNodeCollectorLocalTrafficSvc(modified)

	// CreateOrPatch merge-patches from the mutated object; wiping Spec would zero
	// ClusterIP/Type and produce immutable-field patch failures on every reconcile.
	require.Equal(t, "10.96.1.50", modified.Spec.ClusterIP)
	require.Equal(t, []string{"10.96.1.50"}, modified.Spec.ClusterIPs)
	require.Equal(t, corev1.ServiceTypeClusterIP, modified.Spec.Type)
	require.Equal(t, existing.Spec.IPFamilies, modified.Spec.IPFamilies)
	require.NotNil(t, modified.Spec.Ports[0].AppProtocol)
	require.Equal(t, "grpc", *modified.Spec.Ports[0].AppProtocol)
}
