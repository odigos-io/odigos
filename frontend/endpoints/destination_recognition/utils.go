package destination_recognition

import (
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getAllHelmManagedServices(ctx *gin.Context, namespaces []v1.Namespace) ([]v1.Service, error) {
	var helmManagedServices []v1.Service
	var err error
	for _, ns := range namespaces {
		err = client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.CoreV1().Services(ns.Name).List,
			ctx, metav1.ListOptions{}, func(services *v1.ServiceList) error {
				for _, service := range services.Items {
					if isHelmManagedService(service) {
						helmManagedServices = append(helmManagedServices, service)
					}
				}
				return nil
			})
	}

	if err != nil {
		return nil, err
	}

	return helmManagedServices, nil
}

// isHelmManagedService checks if a Service was created by Helm
func isHelmManagedService(service v1.Service) bool {
	annotations := service.GetAnnotations()
	labels := service.GetLabels()

	_, hasHelmReleaseName := annotations["meta.helm.sh/release-name"]
	managedByHelm := labels["app.kubernetes.io/managed-by"] == "Helm"

	return hasHelmReleaseName && managedByHelm
}
