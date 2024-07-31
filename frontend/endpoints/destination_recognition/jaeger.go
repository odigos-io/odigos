package destination_recognition

import (
	"fmt"
	"github.com/odigos-io/odigos/common"
	k8s "k8s.io/api/core/v1"
	"strings"
)

type JaegerDestinationFinder struct{}

const JaegerGrpcOtlpPort int32 = 4317

func (j *JaegerDestinationFinder) findPotentialServices(services []k8s.Service) []k8s.Service {
	var potentialServices []k8s.Service
	for _, service := range services {
		for _, port := range service.Spec.Ports {
			if port.Port == JaegerGrpcOtlpPort && strings.Contains(service.Name, string(common.JaegerDestinationType)) {
				potentialServices = append(potentialServices, service)
			}
		}
	}

	return potentialServices
}

func (j *JaegerDestinationFinder) fetchDestinationDetails(services []k8s.Service) []DestinationDetails {
	var destinationDetails []DestinationDetails
	for _, service := range services {
		urlString := fmt.Sprintf("url: %s.%s:%d", service.Name, service.Namespace, JaegerGrpcOtlpPort)
		destinationDetails = append(destinationDetails, DestinationDetails{
			Name:      string(common.JaegerDestinationType),
			UrlString: urlString,
		})
	}

	return destinationDetails
}
