package destination_recognition

import (
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/kube"
	k8s "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DestinationType string

const (
	JaegerDestinationType DestinationType = "jaeger"
)

var SupportedDestinationType = []DestinationType{JaegerDestinationType}

type DestinationDetails struct {
	Name      string `json:"name"`
	UrlString string `json:"urlString"`
}

type IDestinationFinder interface {
	findPotentialServices([]k8s.Service) []k8s.Service
	fetchDestinationDetails([]k8s.Service) []DestinationDetails
}

type DestinationFinder struct {
	destinationFinder IDestinationFinder
}

func (d *DestinationFinder) findPotentialServices(services []k8s.Service) []k8s.Service {
	return d.destinationFinder.findPotentialServices(services)
}

func (d *DestinationFinder) fetchDestinationDetails(services []k8s.Service) []DestinationDetails {
	return d.destinationFinder.fetchDestinationDetails(services)
}

func GetAllPotentialDestinationDetails(ctx *gin.Context, namespaces []k8s.Namespace) ([]DestinationDetails, error) {
	helmManagedServices := findHelmManagedServices(ctx, namespaces)

	var destinationFinder DestinationFinder
	for _, destinationType := range SupportedDestinationType {
		switch destinationType {
		case JaegerDestinationType:
			destinationFinder = DestinationFinder{
				destinationFinder: &JaegerDestinationFinder{},
			}
		}
	}

	potentialServices := destinationFinder.findPotentialServices(helmManagedServices)
	destinationDetails := destinationFinder.fetchDestinationDetails(potentialServices)

	return destinationDetails, nil
}

func findHelmManagedServices(ctx *gin.Context, namespaces []k8s.Namespace) []k8s.Service {
	var helmManagedServices []k8s.Service
	for _, ns := range namespaces {
		services, _ := kube.DefaultClient.CoreV1().Services(ns.Name).List(ctx, metav1.ListOptions{})

		for _, service := range services.Items {
			if isHelmManagedPod(service) {
				helmManagedServices = append(helmManagedServices, service)
			}
		}
	}

	return helmManagedServices
}

// isHelmManagedPod checks if a Pod was created by Helm
func isHelmManagedPod(service k8s.Service) bool {
	annotations := service.GetAnnotations()
	labels := service.GetLabels()

	_, hasHelmReleaseName := annotations["meta.helm.sh/release-name"]
	managedByHelm := labels["app.kubernetes.io/managed-by"] == "Helm"

	return hasHelmReleaseName && managedByHelm
}
