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
	helmManagedServices := getAllHelmManagedServices(ctx, namespaces)

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
