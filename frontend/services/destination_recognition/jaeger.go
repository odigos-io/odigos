package destination_recognition

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	k8s "k8s.io/api/core/v1"
)

type JaegerDestinationFinder struct{}

const JaegerGrpcOtlpPort int32 = 4317
const JaegerGrpcUrlFormat = "%s.%s:%d"

func (j *JaegerDestinationFinder) isPotentialService(service k8s.Service) bool {
	for _, port := range service.Spec.Ports {
		if isJaegerService(port.Port, service.Name) {
			return true
		}
	}

	return false
}

func isJaegerService(portNumber int32, name string) bool {
	return portNumber == JaegerGrpcOtlpPort && strings.Contains(name, string(common.JaegerDestinationType))
}

func (j *JaegerDestinationFinder) fetchDestinationDetails(service k8s.Service) DestinationDetails {
	urlString := fmt.Sprintf(JaegerGrpcUrlFormat, service.Name, service.Namespace, JaegerGrpcOtlpPort)

	jaegerServiceURL := j.getServiceURL()
	fields := make(map[string]string)
	fields[jaegerServiceURL] = urlString

	return DestinationDetails{
		Type:   common.JaegerDestinationType,
		Fields: fields,
	}
}

func (j *JaegerDestinationFinder) getServiceURL() string {
	return config.JaegerUrlKey
}
