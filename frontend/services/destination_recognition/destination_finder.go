package destination_recognition

import (
	"context"
	"strings"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	k8s "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	var destinationDetails []DestinationDetails

	for _, ns := range namespaces {
		err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.CoreV1().Services(ns.Name).List, ctx, metav1.ListOptions{},
			func(svc *k8s.ServiceList) error {
				for _, service := range svc.Items {
					df := getDestinationFinder(service.Name)

					if df != nil && df.isPotentialService(service) {
						pd := df.fetchDestinationDetails(service)

						if !destinationExist(dests, pd, df) {
							destinationDetails = append(destinationDetails, pd)
						}
						break
					}
				}

				return nil
			},
		)

		if err != nil {
			return nil, err
		}
	}

	return destinationDetails, nil
}

func getDestinationFinder(serviceName string) IDestinationFinder {
	if strings.Contains(serviceName, string(common.JaegerDestinationType)) {
		return &JaegerDestinationFinder{}
	}

	if strings.Contains(serviceName, string(common.ElasticsearchDestinationType)) {
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
