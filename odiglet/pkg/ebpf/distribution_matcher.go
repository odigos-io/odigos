package ebpf

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type podDeviceDistributionMatcher struct {
	distributionGetter *distros.Getter
}

func (dm *podDeviceDistributionMatcher) Distribution(ctx context.Context, e K8sProcessDetails) (*distro.OtelDistro, error) {
	otelDistro := dm.distributionGetter.GetDistroByName(e.DistroName)
	if otelDistro == nil {
		return nil, fmt.Errorf("no districution is registered for '%s'", e.DistroName)
	}

	// verify the language of the process event
	if ok := inspectors.VerifyLanguage(process.Details{
		ProcessID: e.ProcEvent.PID,
		ExePath:   e.ProcEvent.ExecDetails.ExePath,
		CmdLine:   e.ProcEvent.ExecDetails.CmdLine,
		Environments: process.ProcessEnvs{
			DetailedEnvs: e.ProcEvent.ExecDetails.Environments,
		},
	}, otelDistro.Language, log.Logger); !ok {
		return nil,
			fmt.Errorf("process language does not match the detected language (%s) for container: %s. exe path: %s", otelDistro.Language, e.ContainerName, e.ProcEvent.ExecDetails.ExePath)
	}

	return otelDistro, nil
}
