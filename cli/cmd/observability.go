package cmd

import (
	"context"
	"fmt"
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
	"strings"
)

// observabilityCmd represents the observability command
var observabilityCmd = &cobra.Command{
	Use:   "observability",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
		return fmt.Errorf("please specifiy an observability backend via --backend flag, choose one from %+v", avail)
	}

	for _, s := range avail {
		if name == s {
			return nil
		}
	}

	return fmt.Errorf("invalid backend %s, choose from %+v", name, avail)
}

func persistArgs(args *backend.ObservabilityArgs, cmd *cobra.Command, signals []common.ObservabilitySignal, backendName common.DestinationType) error {
	kc := kube.CreateClient(cmd)
	ns, err := getOdigosNamespace(kc, cmd.Context())
	if err != nil {
		return err
	}

	_, err = kc.OdigosClient.OdigosConfigurations(ns).Create(cmd.Context(), &odigosv1.OdigosConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-config",
		},
		Spec: odigosv1.OdigosConfigurationSpec{
			InstrumentationMode: odigosv1.OptOutInstrumentationMode,
		},
	}, metav1.CreateOptions{})
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
	observabilityCmd.Flags().StringVar(&backendFlag, "backend", "", "Backend for observability data")
	observabilityCmd.Flags().StringSliceVarP(&signalsFlag, "signal", "s", nil, "Reported signals [traces,metrics,logs]")
	observabilityCmd.Flags().BoolVarP(&skipConfirm, "no-prompt", "y", false, "Skip install confirmation")
	observabilityCmd.Flags().StringVarP(&urlFlag, "url", "u", "", "URL of the backend for observability data")
	observabilityCmd.Flags().StringVar(&apiKeyFlag, "api-key", "", "API key for the selected backend")
	observabilityCmd.Flags().StringVar(&regionFlag, "region", "", "Region for the selected backend")

	// Grafana Cloud Flags
	observabilityCmd.Flags().StringVar(&grafanaTempoUrl, backend.GrafanaTempoUrlFlag, "", "URL for Grafana Cloud Tempo instance")
	observabilityCmd.Flags().StringVar(&grafanaTempoUser, backend.GrafanaTempoUserFlag, "", "User for Grafana Cloud Tempo instance")
	observabilityCmd.Flags().StringVar(&grafanaRemoteWriteUrl, backend.GrafanaPromUrlFlag, "", "RemoteWrite URL for Grafana Cloud prometheus instance")
	observabilityCmd.Flags().StringVar(&grafanaPromUser, backend.GrafanaPromUserFlag, "", "User for Grafana Cloud prometheus instance")
	observabilityCmd.Flags().StringVar(&grafanaLokiUrl, backend.GrafanaLokiUrlFlag, "", "URL for Grafana Cloud Loki instance")
	observabilityCmd.Flags().StringVar(&grafanaLokiUser, backend.GrafanaLokiUserFlag, "", "User for Grafana Cloud Loki instance")

	// Logz.io Flags
	observabilityCmd.Flags().StringVar(&logzioTracingToken, backend.LogzioTracingToken, "", "Tracing token for Logz.io")
	observabilityCmd.Flags().StringVar(&logzioMetricsToken, backend.LogzioMetricsToken, "", "Metrics token for Logz.io")
	observabilityCmd.Flags().StringVar(&logzioLoggingToken, backend.LogzioLogsToken, "", "Logging token for Logz.io")
}
