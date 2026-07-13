package services

import (
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestValidateDestinationURLsAgainstAllowedHostsRejectsGrpcEndpoint(t *testing.T) {
	destination := model.DestinationInput{
		Fields: []*model.FieldInput{
			{Key: "OTLP_GRPC_ENDPOINT", Value: "internal-service.odigos-system.svc:4317"},
		},
	}

	err := validateDestinationURLsAgainstAllowedHosts(destination, []string{"allowed.example.com:4317"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "internal-service.odigos-system.svc:4317")
}

func TestValidateDestinationURLsAgainstAllowedHostsRejectsPerSignalHTTPEndpoint(t *testing.T) {
	destination := model.DestinationInput{
		Fields: []*model.FieldInput{
			{Key: "OTLP_HTTP_TRACES_ENDPOINT", Value: "http://internal-service.odigos-system.svc:4318/v1/traces"},
		},
	}

	err := validateDestinationURLsAgainstAllowedHosts(destination, []string{"https://allowed.example.com:4318"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "internal-service.odigos-system.svc")
}

func TestValidateDestinationURLsAgainstAllowedHostsAllowsConfiguredGrpcEndpoint(t *testing.T) {
	destination := model.DestinationInput{
		Fields: []*model.FieldInput{
			{Key: "OTLP_GRPC_ENDPOINT", Value: "collector.example.com:4317"},
		},
	}

	err := validateDestinationURLsAgainstAllowedHosts(destination, []string{"collector.example.com:4317"})

	require.NoError(t, err)
}

func TestValidateDestinationURLsAgainstAllowedHostsAllowsWildcardConfig(t *testing.T) {
	destination := model.DestinationInput{
		Fields: []*model.FieldInput{
			{Key: "OTLP_GRPC_ENDPOINT", Value: "internal-service.odigos-system.svc:4317"},
		},
	}

	err := validateDestinationURLsAgainstAllowedHosts(destination, []string{"*"})

	require.NoError(t, err)
}
