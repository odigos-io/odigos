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
			if isJaegerService(port.Port, service.Name) {
				potentialServices = append(potentialServices, service)
				break
			}
		}
	}

	return potentialServices
}

func isJaegerService(portNumber int32, name string) bool {
	return portNumber == JaegerGrpcOtlpPort && strings.Contains(name, string(common.JaegerDestinationType))
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
