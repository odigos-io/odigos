package destination_recognition

import (
	"github.com/gin-gonic/gin"
	k8s "k8s.io/api/core/v1"
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
	isPotentialService(k8s.Service) bool
	fetchDestinationDetails(k8s.Service) DestinationDetails
}

type DestinationFinder struct {
	destinationFinder IDestinationFinder
}

func (d *DestinationFinder) isPotentialService(service k8s.Service) bool {
	return d.destinationFinder.isPotentialService(service)
}

func (d *DestinationFinder) fetchDestinationDetails(service k8s.Service) DestinationDetails {
	return d.destinationFinder.fetchDestinationDetails(service)
}

func GetAllPotentialDestinationDetails(ctx *gin.Context, namespaces []k8s.Namespace) ([]DestinationDetails, error) {
	helmManagedServices := getAllHelmManagedServices(ctx, namespaces)

	var destinationFinder *DestinationFinder
	var destinationDetails []DestinationDetails
	for _, service := range helmManagedServices {
		for _, destinationType := range SupportedDestinationType {
			destinationFinder = getDestinationFinder(destinationType)
			if destinationFinder.isPotentialService(service) {
				destinationDetails = append(destinationDetails, destinationFinder.fetchDestinationDetails(service))
				break
			}
		}
	}

	return destinationDetails, nil
}

func getDestinationFinder(destinationType DestinationType) *DestinationFinder {
	switch destinationType {
	case JaegerDestinationType:
		return &DestinationFinder{
			destinationFinder: &JaegerDestinationFinder{},
		}
	}

	return nil
}
