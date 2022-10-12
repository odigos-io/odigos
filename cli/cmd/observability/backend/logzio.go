package backend

import (
	"fmt"
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
)

const (
	LogzioTracingToken = "logzio-tracing-token"
	LogzioMetricsToken = "logzio-metrics-token"
	LogzioLogsToken    = "logzio-logs-token"
)

type LogzIO struct{}

func (l *LogzIO) Name() common.DestinationType {
	return common.LogzioDestinationType
}

func (l *LogzIO) ParseFlags(cmd *cobra.Command, selectedSignals []common.ObservabilitySignal) (*ObservabilityArgs, error) {
	region := cmd.Flag("region").Value.String()
	if region == "" {
		return nil, fmt.Errorf("region required when using Logz.io backend. pleease specify --region")
	}

	args := &ObservabilityArgs{
		Data: map[string]string{
			"LOGZIO_REGION": region,
		},
		Secret: make(map[string]string),
	}

	for _, s := range selectedSignals {
		switch s {
		case common.TracesObservabilitySignal:
			tracingToken := cmd.Flag(LogzioTracingToken).Value.String()
			if tracingToken == "" {
				return nil, fmt.Errorf("tracing token required when using Logz.io backend with traces signal. pleease specify --%s", LogzioTracingToken)
			}

			args.Secret["LOGZIO_TRACING_TOKEN"] = tracingToken
		case common.MetricsObservabilitySignal:
			metricsToken := cmd.Flag(LogzioMetricsToken).Value.String()
			if metricsToken == "" {
				return nil, fmt.Errorf("metrics token required when using Logz.io backend with metrics signal. pleease specify --%s", LogzioMetricsToken)
			}

			args.Secret["LOGZIO_METRICS_TOKEN"] = metricsToken
		case common.LogsObservabilitySignal:
			loggingToken := cmd.Flag(LogzioLogsToken).Value.String()
			if loggingToken == "" {
				return nil, fmt.Errorf("logging token required when using Logz.io backend with logs signal. pleease specify --%s", LogzioLogsToken)
			}

			args.Secret["LOGZIO_LOGS_TOKEN"] = loggingToken
		}
	}

	return args, nil
}

func (l *LogzIO) SupportedSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}
}
