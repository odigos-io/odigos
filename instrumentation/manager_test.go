package instrumentation

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentation/detector"
	"go.opentelemetry.io/otel/metric/noop"
)

type testProcessDetails struct {
	distribution *distro.OtelDistro
}

func (t testProcessDetails) String() string {
	return "test-process-details"
}

func (t testProcessDetails) ConfigGroup(context.Context) (string, error) {
	return "test-config-group", nil
}

func (t testProcessDetails) ProcessGroup(context.Context) (string, error) {
	return "test-process-group", nil
}

func (t testProcessDetails) Distribution(context.Context) (*distro.OtelDistro, error) {
	return t.distribution, nil
}

type testReporter struct{}

func (testReporter) OnInit(context.Context, int, error, testProcessDetails) error {
	return nil
}

func (testReporter) OnLoad(context.Context, int, error, testProcessDetails, Status) error {
	return nil
}

func (testReporter) OnRun(context.Context, int, error, testProcessDetails) error {
	return nil
}

func (testReporter) OnExit(context.Context, int, testProcessDetails) error {
	return nil
}

type testSettingsGetter struct{}

func (testSettingsGetter) Settings(context.Context, logr.Logger, testProcessDetails, *distro.OtelDistro) (Settings, error) {
	return Settings{}, nil
}

type failingFactory struct {
	err error
}

func (f failingFactory) CreateInstrumentation(context.Context, int, Settings) (Instrumentation, error) {
	return nil, f.err
}

type trackingDetector struct {
	tracked []int
}

func (d *trackingDetector) Run(context.Context) error {
	return nil
}

func (d *trackingDetector) TrackProcesses(pids []int) error {
	d.tracked = append(d.tracked, pids...)
	return nil
}

func newTestManager(t *testing.T, detector detector.Detector, factories map[string]Factory) *manager[string, string, testProcessDetails] {
	t.Helper()

	metrics, err := newManagerMetrics(noop.NewMeterProvider().Meter("test"))
	if err != nil {
		t.Fatalf("newManagerMetrics returned error: %v", err)
	}

	return &manager[string, string, testProcessDetails]{
		detector: detector,
		handler: &Handler[string, string, testProcessDetails]{
			ProcessDetailsResolver: nil,
			Reporter:               testReporter{},
			SettingsGetter:         testSettingsGetter{},
		},
		factories:             factories,
		logger:                commonlogger.LoggerCompat(),
		detailsByPid:          make(map[int]*instrumentationDetails[string, string, testProcessDetails]),
		detailsByConfigGroup:  make(map[string]map[int]*instrumentationDetails[string, string, testProcessDetails]),
		detailsByProcessGroup: make(map[string]map[int]*instrumentationDetails[string, string, testProcessDetails]),
		metrics:               metrics,
	}
}

func TestInstrumentFromDetailsTracksFailedAttemptsForExitCleanup(t *testing.T) {
	const pid = 1234
	initErr := errors.New("init failed")
	tracker := &trackingDetector{}
	m := newTestManager(t, tracker, map[string]Factory{
		"go": failingFactory{err: initErr},
	})

	m.instrumentFromDetails(context.Background(), map[int]testProcessDetails{
		pid: {
			distribution: &distro.OtelDistro{
				Name:     "go",
				Language: common.GoProgrammingLanguage,
			},
		},
	})

	if len(tracker.tracked) != 1 || tracker.tracked[0] != pid {
		t.Fatalf("tracked pids got %v want [%d]", tracker.tracked, pid)
	}
	details, ok := m.detailsByPid[pid]
	if !ok {
		t.Fatalf("pid %d was not tracked after failed instrumentation", pid)
	}
	if details.inst != nil {
		t.Fatalf("failed instrumentation stored non-nil instance: %#v", details.inst)
	}
}
