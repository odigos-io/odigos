package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

type DigitalOceanSpaces struct{}

func (s *DigitalOceanSpaces) DestType() common.DestinationType {
	return common.DigitalOceanSpacesDestinationType
}

func (s *DigitalOceanSpaces) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	endpoint, err := url.Parse(dest.GetConfig()["S3_ENDPOINT"])
	if err != nil || endpoint.Scheme != "https" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Path != "" && endpoint.Path != "/") || endpoint.Port() != "" {
		return nil, fmt.Errorf("Spaces requires an HTTPS regional origin endpoint")
	}
	host := strings.TrimSuffix(endpoint.Hostname(), ".digitaloceanspaces.com")
	if host == endpoint.Hostname() || host == "" || strings.Contains(host, ".") || strings.HasSuffix(host, "-static") {
		return nil, fmt.Errorf("Spaces requires a regional origin endpoint, for example https://nyc3.digitaloceanspaces.com")
	}
	if compression := dest.GetConfig()["S3_COMPRESSION"]; compression != "" && compression != "gzip" && compression != "zstd" {
		return nil, fmt.Errorf("unsupported S3 compression")
	}
	if strings.TrimSpace(dest.GetConfig()[s3BucketKey]) == "" {
		return nil, ErrS3BucketNotSpecified
	}
	if strings.TrimSpace(dest.GetConfig()[s3RegionKey]) == "" {
		return nil, ErrS3RegionNotSpecified
	}
	return configureS3(dest, currentConfig, true)
}
