package collectorconfig

import (
	b64 "encoding/base64"
	"fmt"
	"github.com/ghodss/yaml"
	v1 "github.com/keyval-dev/odigos/cooper/api/v1"
	"strings"
)

type genericMap map[string]interface{}

type Config struct {
	Receivers  genericMap `json:"receivers"`
	Exporters  genericMap `json:"exporters"`
	Processors genericMap `json:"processors"`
	Extensions genericMap `json:"extensions"`
	Service    genericMap `json:"service"`
}

func getExporters(dest *v1.Destination) genericMap {
	if dest.Spec.Type == v1.GrafanaDestinationType {
		authString := fmt.Sprintf("%s:%s", dest.Spec.Data.Grafana.User, dest.Spec.Data.Grafana.ApiKey)
		encodedAuthString := b64.StdEncoding.EncodeToString([]byte(authString))
		url := strings.TrimSuffix(dest.Spec.Data.Grafana.Url, "/tempo")
		return genericMap{
			"otlp": genericMap{
				"endpoint": fmt.Sprintf("%s:%d", url, 443),
				"headers": genericMap{
					"authorization": fmt.Sprintf("Basic %s", encodedAuthString),
				},
			},
		}
	}

	return genericMap{}
}

func GetConfigForCollector(dest *v1.Destination) (string, error) {
	empty := struct{}{}
	exporters := getExporters(dest)
	c := &Config{
		Receivers: genericMap{
			"zipkin": empty,
			"otlp": genericMap{
				"protocols": genericMap{
					"grpc": empty,
					"http": empty,
				},
			},
		},
		Exporters: exporters,
		Processors: genericMap{
			"batch": empty,
		},
		Extensions: genericMap{
			"health_check": empty,
			"zpages":       empty,
		},
		Service: getService(exporters),
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func getService(exporters genericMap) genericMap {
	var exp []string
	for e, _ := range exporters {
		exp = append(exp, e)
	}

	return genericMap{
		"extensions": []string{"health_check", "zpages"},
		"pipelines": genericMap{
			"traces": genericMap{
				"receivers":  []string{"otlp", "zipkin"},
				"processors": []string{"batch"},
				"exporters":  exp,
			},
		},
	}
}
