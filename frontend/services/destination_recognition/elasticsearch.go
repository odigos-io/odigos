package destination_recognition

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	k8s "k8s.io/api/core/v1"
)

type ElasticSearchDestinationFinder struct{}

const ElasticSearchHttpPort int32 = 9200
const ElasticSearchHttpUrlFormat = "https://%s.%s:%d"

func (j *ElasticSearchDestinationFinder) isPotentialService(service k8s.Service) bool {
	for _, port := range service.Spec.Ports {
		if isElasticSearchService(port.Port, service.Name) {
			return true
		}
	}

	return false
}

func isElasticSearchService(portNumber int32, name string) bool {
	return portNumber == ElasticSearchHttpPort && strings.Contains(name, string(common.ElasticsearchDestinationType))
}

func (j *ElasticSearchDestinationFinder) fetchDestinationDetails(service k8s.Service) DestinationDetails {
	urlString := fmt.Sprintf(ElasticSearchHttpUrlFormat, service.Name, service.Namespace, ElasticSearchHttpPort)
	elasticServiceURL := j.getServiceURL()
	fields := make(map[string]string)
	fields[elasticServiceURL] = urlString

	return DestinationDetails{
		Type:   common.ElasticsearchDestinationType,
		Fields: fields,
	}
}

func (j *ElasticSearchDestinationFinder) getServiceURL() string {
	return config.ElasticsearchUrlKey
}
