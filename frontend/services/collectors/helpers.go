package collectors

import (
	containersutil "github.com/odigos-io/odigos/k8sutils/pkg/containers"
	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services"
)

func extractResourcesForContainer(containers []corev1.Container, containerName string) *model.Resources {
	c := containersutil.GetContainerByName(containers, containerName)
	if c == nil {
		return nil
	}
	req := buildResourceAmounts(c.Resources.Requests)
	lim := buildResourceAmounts(c.Resources.Limits)
	return &model.Resources{Requests: req, Limits: lim}
}

// extractImageVersionForContainer finds a container by name and returns its parsed image version (tag).
// Returns empty string if the container is not found or the image has no tag.
func extractImageVersionForContainer(containers []corev1.Container, containerName string) string {
	c := containersutil.GetContainerByName(containers, containerName)
	if c == nil {
		return ""
	}
	return services.ExtractImageVersion(c.Image)
}
