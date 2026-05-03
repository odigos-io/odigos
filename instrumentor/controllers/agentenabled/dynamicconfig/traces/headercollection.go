package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/distros/distro"
)

func DistroSupportsTracesHeadersCollection(distro *distro.OtelDistro) bool {
	return distro.Traces != nil && distro.Traces.HeadersCollection != nil && distro.Traces.HeadersCollection.Supported
}

func CalculateHeaderCollectionConfig(distro *distro.OtelDistro, irls *[]odigosv1.InstrumentationRule) *odigosv1.HeadersCollectionConfig {

	if !DistroSupportsTracesHeadersCollection(distro) {
		return nil
	}

	// http headers collection configuration
	headerKeysToCollectHttp := []string{}
	for _, irl := range *irls {
		if irl.Spec.HeadersCollection != nil {
			headerKeysToCollectHttp = append(headerKeysToCollectHttp, irl.Spec.HeadersCollection.HeaderKeys...)
		}
	}
	if len(headerKeysToCollectHttp) == 0 {
		return nil
	}

	return &odigosv1.HeadersCollectionConfig{
		HttpHeaderKeys: headerKeysToCollectHttp,
	}
}
