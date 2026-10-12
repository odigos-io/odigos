package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

type OpenStackSwift struct{}

func (s *OpenStackSwift) DestType() common.DestinationType {
	return common.OpenStackSwiftDestinationType
}

func (s *OpenStackSwift) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if dest.GetConfig()[s3EndpointKey] == "" {
		return nil, errors.New("S3_ENDPOINT is required for OpenStack Swift S3 Gateway")
	}
	return configureS3(dest, currentConfig, "openstackswift")
}
