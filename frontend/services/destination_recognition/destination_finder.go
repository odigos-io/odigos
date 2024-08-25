package destination_recognition

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	k8s "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var SupportedDestinationType = []common.DestinationType{common.JaegerDestinationType, common.ElasticsearchDestinationType}

type DestinationDetails struct {
	Type   common.DestinationType `json:"type"`
	Fields map[string]string      `json:"fields"`
}

type IDestinationFinder interface {
	isPotentialService(k8s.Service) bool
	fetchDestinationDetails(k8s.Service) DestinationDetails
	getServiceURL() string
}

func GetAllPotentialDestinationDetails(ctx context.Context, namespaces []k8s.Namespace, dests *odigosv1.DestinationList) ([]DestinationDetails, error) {
	var destinationFinder IDestinationFinder
	var destinationDetails []DestinationDetails
	var err error

	for _, ns := range namespaces {
		err = client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.CoreV1().Services(ns.Name).List,
			ctx, metav1.ListOptions{}, func(services *k8s.ServiceList) error {
				for _, service := range services.Items {
					for _, destinationType := range SupportedDestinationType {
						destinationFinder = getDestinationFinder(destinationType)

						if destinationFinder.isPotentialService(service) {
							potentialDestination := destinationFinder.fetchDestinationDetails(service)

							if !destinationExist(dests, potentialDestination, destinationFinder) {
								destinationDetails = append(destinationDetails, potentialDestination)
							}
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

func getDestinationFinder(destinationType common.DestinationType) IDestinationFinder {
	switch destinationType {
	case common.JaegerDestinationType:
		return &JaegerDestinationFinder{}
	case common.ElasticsearchDestinationType:
		return &ElasticSearchDestinationFinder{}
	}

	return nil
}

func destinationExist(dests *odigosv1.DestinationList, potentialDestination DestinationDetails, destinationFinder IDestinationFinder) bool {
	for _, dest := range dests.Items {
		if dest.Spec.Type == potentialDestination.Type && dest.GetConfig()[destinationFinder.getServiceURL()] == potentialDestination.Fields[destinationFinder.getServiceURL()] {
			return true
		}
	}

	return false
}
