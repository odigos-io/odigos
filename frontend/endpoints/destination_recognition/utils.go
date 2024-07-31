package destination_recognition

import (
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/kube"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getAllHelmManagedServices(ctx *gin.Context, namespaces []v1.Namespace) []v1.Service {
	var helmManagedServices []v1.Service
	for _, ns := range namespaces {
		services, _ := kube.DefaultClient.CoreV1().Services(ns.Name).List(ctx, v12.ListOptions{})

		for _, service := range services.Items {
			if isHelmManagedService(service) {
				helmManagedServices = append(helmManagedServices, service)
			}
		}
	}

	return helmManagedServices
}

// isHelmManagedService checks if a Pod was created by Helm
func isHelmManagedService(service v1.Service) bool {
	annotations := service.GetAnnotations()
	labels := service.GetLabels()

	_, hasHelmReleaseName := annotations["meta.helm.sh/release-name"]
	managedByHelm := labels["app.kubernetes.io/managed-by"] == "Helm"

	return hasHelmReleaseName && managedByHelm
}
