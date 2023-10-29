package cmd

import (
	"context"
	"fmt"
	"strings"

	_ "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/cli/cmd/observability/backend"
	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/pkg/confirm"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// observabilityCmd represents the observability command
var observabilityCmd = &cobra.Command{
    Use:   "observability",
    Short: "Configure and manage observability for your applications and services",
    Long: `The observability command allows you to set up and manage observability and monitoring for your applications and services. It provides you with the tools to choose an observability backend, specify the signals you want to collect (e.g., traces, metrics, logs), and define the settings for your observability infrastructure.

Observability is crucial for understanding how your applications are performing, diagnosing issues, and ensuring the reliability of your services. It enables you to gain valuable insights into the behavior of your systems.

You can select from a variety of observability backends, such as Grafana Cloud, Logz.io, and more. Each backend offers unique features and integrations. You can pick the one that aligns best with your specific needs.

Usage Examples:
  - Configure observability with Grafana Cloud:
    odigos observability --backend grafana-cloud --signal traces,metrics,logs --api-key <YOUR_API_KEY>

  - Configure observability with Logz.io:
    odigos observability --backend logzio --signal traces,metrics,logs --tracing-token <YOUR_TRACING_TOKEN> --metrics-token <YOUR_METRICS_TOKEN> --logs-token <YOUR_LOGS_TOKEN>

You have the flexibility to customize your observability setup according to your precise requirements. Utilize the provided flags to specify the backend, select the signals to capture, and set other options. Make informed decisions to effectively monitor, analyze, and enhance the performance of your applications and services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := isValidBackend(backendFlag); err != nil {
			return err
		}

		be := backend.Get(backendFlag)

		signals, err := calculateSignals(signalsFlag, be.SupportedSignals(), be.Name())
		if err != nil {
			return err
		}

		parsedArgs, err := be.ParseFlags(cmd, signals)
		if err != nil {
			return err
		}

		fmt.Println("About to install the following observability skill:")
		fmt.Println("Target applications: all recognized applications")
		fmt.Println("Infra: OpenTelemetry Collector")
		fmt.Printf("Signals: %s\n", strings.Join(signalsToString(signals), ","))
		if !skipConfirm {
			confirmed, err := confirm.Ask("Do you want to continue?")
			if err != nil {
				return err
			}

			if !confirmed {
				fmt.Println("Aborting installation.")
				return nil
			}
		}

		if err := persistArgs(parsedArgs, cmd, signals, be.Name()); err != nil {
			return err
		}

		fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Observability skill installed.\n")
		return nil
	},
}

var (
	backendFlag string
	apiKeyFlag  string
	urlFlag     string
	regionFlag  string
	signalsFlag []string
	skipConfirm bool

	// Grafana Cloud flags
	grafanaTempoUrl       string
	grafanaTempoUser      string
	grafanaRemoteWriteUrl string
	grafanaPromUser       string
	grafanaLokiUrl        string
	grafanaLokiUser       string

	// Logz.io flags
	logzioTracingToken string
	logzioMetricsToken string
	logzioLoggingToken string
)

func isValidBackend(name string) error {
	avail := backend.GetAvailableBackends()
	if name == "" {
		return fmt.Errorf("please specify an observability backend via --backend flag, choose one from %+v", avail)
	}

	for _, s := range avail {
		if name == s {
			return nil
		}
	}

	return fmt.Errorf("invalid backend %s, choose from %+v", name, avail)
}

func persistArgs(args *backend.ObservabilityArgs, cmd *cobra.Command, signals []common.ObservabilitySignal, backendName common.DestinationType) error {
	kc, err := kube.CreateClient(cmd)
	if err != nil {
		kube.PrintClientErrorAndExit(err)
	}
	ns, err := getOdigosNamespace(kc, cmd.Context())
	if err != nil {
		return err
	}

	skillName := "observability" // TODO
	_, err = kc.CoreV1().Secrets(ns).Create(cmd.Context(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: skillName,
		},
		StringData: args.Secret,
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = kc.OdigosClient.Destinations(ns).Create(cmd.Context(), &odigosv1.Destination{
		ObjectMeta: metav1.ObjectMeta{
			Name: skillName,
		},
		Spec: odigosv1.DestinationSpec{
			Type: backendName,
			Data: args.Data,
			SecretRef: &v1.LocalObjectReference{
				Name: skillName,
			},
			Signals: signals,
		},
	}, metav1.CreateOptions{})

	return err
}

func calculateSignals(args []string, supported []common.ObservabilitySignal, beName common.DestinationType) ([]common.ObservabilitySignal, error) {
	if len(args) == 0 {
		return supported, nil
	}

	supportedMap := make(map[common.ObservabilitySignal]interface{}, len(supported))
	for _, s := range supported {
		supportedMap[s] = nil
	}

	var result []common.ObservabilitySignal
	for _, s := range args {
		signal, ok := common.GetSignal(s)
		if !ok {
			return nil, fmt.Errorf("%s is not a valid signal choose from %+v", s, signalsToString(supported))
		}

		if _, exists := supportedMap[signal]; !exists {
			return nil, fmt.Errorf("%s is not supported as a %s signal. Choose from the following signals %+v or choose a different backend", s, beName, signalsToString(supported))
		}

		result = append(result, signal)
	}

	return result, nil
}

func signalsToString(signals []common.ObservabilitySignal) []string {
	var result []string
	for _, s := range signals {
		result = append(result, strings.ToLower(string(s)))
	}
	return result
}

func getOdigosNamespace(kubeClient *kube.Client, ctx context.Context) (string, error) {
	return resources.GetOdigosNamespace(kubeClient, ctx)
}

func init() {
		observabilityCmd.Flags().StringVar(&backendFlag, "backend", "", "Specify the observability backend (e.g., Grafana, Logz.io)")
    	observabilityCmd.Flags().StringSliceVarP(&signalsFlag, "signal", "s", nil, "Specify reported signals (e.g., traces, metrics, logs)")
    	observabilityCmd.Flags().BoolVarP(&skipConfirm, "no-prompt", "y", false, "Skip installation confirmation")
    	observabilityCmd.Flags().StringVarP(&urlFlag, "url", "u", "", "Set the URL of the observability backend")
    	observabilityCmd.Flags().StringVar(&apiKeyFlag, "api-key", "", "Provide the API key for the selected backend")
    	observabilityCmd.Flags().StringVar(&regionFlag, "region", "", "Specify the region for the selected backend")

    // Grafana Cloud Flags
    	observabilityCmd.Flags().StringVar(&grafanaTempoUrl, backend.GrafanaTempoUrlFlag, "", "Set the URL for Grafana Cloud Tempo instance")
    	observabilityCmd.Flags().StringVar(&grafanaTempoUser, backend.GrafanaTempoUserFlag, "", "Specify the user for Grafana Cloud Tempo instance")
    	observabilityCmd.Flags().StringVar(&grafanaRemoteWriteUrl, backend.GrafanaPromUrlFlag, "", "Set the RemoteWrite URL for Grafana Cloud Prometheus instance")
    	observabilityCmd.Flags().StringVar(&grafanaPromUser, backend.GrafanaPromUserFlag, "", "Specify the user for Grafana Cloud Prometheus instance")
    	observabilityCmd.Flags().StringVar(&grafanaLokiUrl, backend.GrafanaLokiUrlFlag, "", "Set the URL for Grafana Cloud Loki instance")
    	observabilityCmd.Flags().StringVar(&grafanaLokiUser, backend.GrafanaLokiUserFlag, "", "Specify the user for Grafana Cloud Loki instance")

    // Logz.io Flags
    	observabilityCmd.Flags().StringVar(&logzioTracingToken, backend.LogzioTracingToken, "", "Set the tracing token for Logz.io")
    	observabilityCmd.Flags().StringVar(&logzioMetricsToken, backend.LogzioMetricsToken, "", "Set the metrics token for Logz.io")
    	observabilityCmd.Flags().StringVar(&logzioLoggingToken, backend.LogzioLogsToken, "", "Set the logging token for Logz.io")
}