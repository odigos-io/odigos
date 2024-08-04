package destination_recognition

import (
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	k8s "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DestinationType string

const (
	JaegerDestinationType        DestinationType = "jaeger"
	ElasticSearchDestinationType DestinationType = "elasticsearch"
)

var SupportedDestinationType = []DestinationType{JaegerDestinationType, ElasticSearchDestinationType}

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
	var destinationFinder *DestinationFinder
	var destinationDetails []DestinationDetails
	var err error

	for _, ns := range namespaces {
		err = client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.CoreV1().Services(ns.Name).List,
			ctx, metav1.ListOptions{}, func(services *k8s.ServiceList) error {
				for _, service := range services.Items {
					for _, destinationType := range SupportedDestinationType {
						destinationFinder = getDestinationFinder(destinationType)
						if destinationFinder.isPotentialService(service) {
							destinationDetails = append(destinationDetails, destinationFinder.fetchDestinationDetails(service))
							break
						}
					}
				}
				return nil
			})
	}

	if err != nil {
		return nil, err
	}

	return destinationDetails, nil
}

func getDestinationFinder(destinationType DestinationType) *DestinationFinder {
	switch destinationType {
	case JaegerDestinationType:
		return &DestinationFinder{
			destinationFinder: &JaegerDestinationFinder{},
		}
	case ElasticSearchDestinationType:
		return &DestinationFinder{
			destinationFinder: &ElasticSearchDestinationFinder{},
		}
	}

	return nil
}
