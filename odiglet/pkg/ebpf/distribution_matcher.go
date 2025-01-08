package ebpf

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/instrumentation"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/container"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type podDeviceDistributionMatcher struct{}

func (dm *podDeviceDistributionMatcher) Distribution(ctx context.Context, e K8sProcessDetails) (instrumentation.OtelDistribution, error) {
	// get the language and sdk for this process event
	// based on the pod spec and the container name from the process event
	lang, sdk, err := odgiosK8s.LanguageSdkFromPodContainer(e.pod, e.containerName)
	if err != nil {
		return instrumentation.OtelDistribution{}, fmt.Errorf("failed to get language and sdk: %w", err)
	}
	// verify the language of the process event
	if ok := inspectors.VerifyLanguage(process.Details{
		ProcessID: e.procEvent.PID,
		ExeName:   e.procEvent.ExecDetails.ExeName,
		CmdLine:   e.procEvent.ExecDetails.CmdLine,
		Environments: process.ProcessEnvs{
			DetailedEnvs:  e.procEvent.ExecDetails.Environments,
		},
	}, lang); ok {
		return instrumentation.OtelDistribution{Language: lang, OtelSdk: sdk}, nil
	}

	return instrumentation.OtelDistribution{},
	 	fmt.Errorf("process language does not match the detected language (%s) for container: %s. exe name: %s", lang, e.containerName, e.procEvent.ExecDetails.ExeName)
}
