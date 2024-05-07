package report

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	endpoint = "https://349trlge22.execute-api.us-east-1.amazonaws.com/default/odigos_events_lambda"
)

func Start(c client.Client) {
	installationId := uuid.New().String()
	time.Sleep(1 * time.Minute)
	reportEvent(c, installationId)

	time.Sleep(5 * time.Minute)
	reportEvent(c, installationId)

	for range time.Tick(24 * time.Hour) {
		reportEvent(c, installationId)
	}
}

func reportEvent(c client.Client, installationId string) {
	err := report(c, installationId)
	if err != nil {
		ctrl.Log.Error(err, "error reporting event")
	}
}

type event struct {
	SendingTraces    bool     `json:"sending_traces"`
	SendingMetrics   bool     `json:"sending_metrics"`
	SendingLogs      bool     `json:"sending_logs"`
	Backends         []string `json:"backends"`
	GoApps           int      `json:"go_apps"`
	JavaApps         int      `json:"java_apps"`
	PythonApps       int      `json:"python_apps"`
	DotnetApps       int      `json:"dotnet_apps"`
	JsApps           int      `json:"js_apps"`
	UnrecognizedApps int      `json:"unrecognized_apps"`
	NumOfNodes       int      `json:"num_of_nodes"`
	InstallationID   string   `json:"installation_id"`
}

func report(c client.Client, installationId string) error {
	ctx := context.Background()
	var dests odigosv1.DestinationList
	err := c.List(ctx, &dests)
	if err != nil {
		return err
	}

	traces := false
	metrics := false
	logs := false
	var backends []string
	for _, dest := range dests.Items {
		backends = append(backends, string(dest.Spec.Type))
		for _, s := range dest.Spec.Signals {
			if s == common.TracesObservabilitySignal {
				traces = true
			} else if s == common.MetricsObservabilitySignal {
				metrics = true
			} else if s == common.LogsObservabilitySignal {
				logs = true
			}
		}
	}

	var nodes corev1.NodeList
	err = c.List(ctx, &nodes)
	if err != nil {
		return err
	}

	var apps odigosv1.InstrumentedApplicationList
	err = c.List(ctx, &apps)
	if err != nil {
		return err
	}

	goApps := 0
	javaApps := 0
	pythonApps := 0
	dotnetApps := 0
	jsApps := 0
	unrecognizedApps := 0
	for _, app := range apps.Items {
		for _, l := range app.Spec.RuntimeDetails {
			switch l.Language {
			case common.GoProgrammingLanguage:
				goApps++
			case common.JavaProgrammingLanguage:
				javaApps++
			case common.PythonProgrammingLanguage:
				pythonApps++
			case common.DotNetProgrammingLanguage:
				dotnetApps++
			case common.JavascriptProgrammingLanguage:
				jsApps++
			default:
				unrecognizedApps++
			}
		}

		if len(app.Spec.RuntimeDetails) == 0 {
			unrecognizedApps++
		}
	}

	reportedEvent := &event{
		Backends:         backends,
		SendingTraces:    traces,
		SendingMetrics:   metrics,
		SendingLogs:      logs,
		NumOfNodes:       len(nodes.Items),
		GoApps:           goApps,
		JavaApps:         javaApps,
		PythonApps:       pythonApps,
		DotnetApps:       dotnetApps,
		JsApps:           jsApps,
		UnrecognizedApps: unrecognizedApps,
		InstallationID:   installationId,
	}
	jsonReport, err := json.Marshal(reportedEvent)
	if err != nil {
		return err
	}

	_, err = http.Post(endpoint, "application/json", bytes.NewBuffer(jsonReport))
	return err
}
