package destination_recognition

import (
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	k8s "k8s.io/api/core/v1"
)

type OdigosDestinationFinder struct{}

const OdigosGrpcOtlpPort int32 = 4317

func (j *OdigosDestinationFinder) isPotentialService(service k8s.Service) bool {
	for _, port := range service.Spec.Ports {
		if isOdigosService(port.Port, service.Name) {
			return true
		}
	}

	return false
}

func isOdigosService(portNumber int32, name string) bool {
	return portNumber == OdigosGrpcOtlpPort && strings.Contains(name, k8sconsts.IngesterServiceName)
}

func (j *OdigosDestinationFinder) fetchDestinationDetails(service k8s.Service) DestinationDetails {
	return DestinationDetails{
		Type: common.OdigosDestinationType,
	}
}

func (j *OdigosDestinationFinder) getServiceURL() string {
	return ""
}
