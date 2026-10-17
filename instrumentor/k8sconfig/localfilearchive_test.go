package k8sconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

func TestLocalFileArchivePVCMount(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	for _, mode := range []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany, corev1.ReadWriteOnce} {
		t.Run(string(mode), func(t *testing.T) {
			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "archive", Namespace: "odigos"}, Spec: corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{mode}}}
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
			deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "odigos"}, Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: k8sconsts.OdigosClusterCollectorContainerName}}}}}}
			dest := &v1alpha1.Destination{ObjectMeta: metav1.ObjectMeta{Name: "archive.example"}, Spec: v1alpha1.DestinationSpec{Type: common.LocalFileArchiveDestinationType, Data: map[string]string{config.LocalFileArchivePathKey: "/archive", config.LocalFileArchivePVCKey: "archive"}}}
			configer := LoadK8sConfigers()[common.LocalFileArchiveDestinationType]
			require.NotNil(t, configer)
			err := configer.ModifyGatewayCollectorDeployment(context.Background(), c, dest, deployment)
			if mode != corev1.ReadWriteMany {
				require.ErrorContains(t, err, "ReadWriteMany")
				assert.Empty(t, deployment.Spec.Template.Spec.Volumes)
				return
			}
			require.NoError(t, err)
			require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), c, dest, deployment))
			assert.Len(t, deployment.Spec.Template.Spec.Volumes, 1)
			assert.Equal(t, "archive", deployment.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName)
			assert.Equal(t, []corev1.VolumeMount{{Name: "file-archive-archive-example", MountPath: "/archive"}}, deployment.Spec.Template.Spec.Containers[0].VolumeMounts)
			dest.Name = "other"
			require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), c, dest, deployment))
			assert.Len(t, deployment.Spec.Template.Spec.Volumes, 1)
			dest.Spec.Data[config.LocalFileArchivePVCKey] = "missing"
			require.Error(t, configer.ModifyGatewayCollectorDeployment(context.Background(), c, dest, deployment))
			dest.Spec.Data[config.LocalFileArchivePVCKey] = ""
			require.ErrorContains(t, configer.ModifyGatewayCollectorDeployment(context.Background(), c, dest, deployment), config.LocalFileArchivePVCKey)
			dest.Spec.Data[config.LocalFileArchivePVCKey] = "archive"
			dest.Spec.Data[config.LocalFileArchivePathKey] = "/archive/nested"
			require.ErrorContains(t, configer.ModifyGatewayCollectorDeployment(context.Background(), c, dest, deployment), "overlaps")
		})
	}
}
