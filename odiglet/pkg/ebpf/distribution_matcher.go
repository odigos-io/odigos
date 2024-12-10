package ebpf

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common/instrumentation/types"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/container"
)

type distributionMatcher struct{}

func (dm *distributionMatcher) Distribution(ctx context.Context, e K8sDetails) (types.OtelDistribution, error) {
	// get the language and sdk for this process event
	// based on the pod spec and the container name from the process event
	// TODO: We should have all the required information in the process event
	// to determine the language - hence in the future we can improve this
	lang, sdk, err := odgiosK8s.LanguageSdkFromPodContainer(e.pod, e.containerName)
	if err != nil {
		return types.OtelDistribution{}, fmt.Errorf("failed to get language and sdk: %w", err)
	}
	return types.OtelDistribution{Language: lang, OtelSdk: sdk}, nil
}