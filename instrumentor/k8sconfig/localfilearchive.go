package k8sconfig

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

type LocalFileArchive struct{}

func (f *LocalFileArchive) DestType() common.DestinationType {
	return common.LocalFileArchiveDestinationType
}

func (f *LocalFileArchive) ModifyGatewayCollectorDeployment(ctx context.Context, c client.Client, dest K8sExporterConfigurer, deployment *appsv1.Deployment) error {
	directory, err := config.LocalFileArchiveDirectory(dest.GetConfig())
	if err != nil {
		return err
	}
	claim := dest.GetConfig()[config.LocalFileArchivePVCKey]
	if claim == "" {
		return fmt.Errorf("local file archive on Kubernetes requires %s", config.LocalFileArchivePVCKey)
	}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: deployment.Namespace, Name: claim}, pvc); err != nil {
		return fmt.Errorf("local file archive PVC: %w", err)
	}
	if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == corev1.PersistentVolumeBlock {
		return fmt.Errorf("local file archive PVC must use Filesystem volume mode")
	}
	shared := false
	for _, mode := range pvc.Spec.AccessModes {
		if mode == corev1.ReadWriteMany {
			shared = true
		}
	}
	if !shared {
		return fmt.Errorf("local file archive PVC requires ReadWriteMany for gateway scaling and rolling updates")
	}
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		if container.Name != k8sconsts.OdigosClusterCollectorContainerName {
			continue
		}
		for _, mount := range container.VolumeMounts {
			if mount.MountPath == directory && !mount.ReadOnly {
				for _, volume := range deployment.Spec.Template.Spec.Volumes {
					if volume.Name == mount.Name && volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == claim && !volume.PersistentVolumeClaim.ReadOnly {
						return nil
					}
				}
			}
			if mount.MountPath == directory || strings.HasPrefix(directory, mount.MountPath+"/") || strings.HasPrefix(mount.MountPath, directory+"/") {
				return fmt.Errorf("local file archive path overlaps volume mount %s", mount.MountPath)
			}
		}
		volumeName := "file-archive-" + config.SanitizeDestinationID(dest.GetID())
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{Name: volumeName, MountPath: directory})
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: volumeName, VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claim}},
		})
		return nil
	}
	return fmt.Errorf("gateway collector container %q not found", k8sconsts.OdigosClusterCollectorContainerName)
}
