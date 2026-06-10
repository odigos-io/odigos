package instrumentation

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/distros/distro"
)

type testProcessGroup string
type testConfigGroup string

type testProcessDetails struct {
	distro *distro.OtelDistro
}

func (d *testProcessDetails) String() string {
	return "test-process-details"
}

func (d *testProcessDetails) ConfigGroup(context.Context) (testConfigGroup, error) {
	return testConfigGroup("test-config"), nil
}

func (d *testProcessDetails) ProcessGroup(context.Context) (testProcessGroup, error) {
	return testProcessGroup("test-process"), nil
}

func (d *testProcessDetails) Distribution(context.Context) (*distro.OtelDistro, error) {
	return d.distro, nil
}

type trackingDetector struct {
	tracked []int
}

func (d *trackingDetector) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (d *trackingDetector) TrackProcesses(pids []int) error {
	d.tracked = append(d.tracked, pids...)
	return nil
}

type testReporter struct{}

func (testReporter) OnInit(context.Context, int, error, *testProcessDetails) error {
	return nil
}

func (testReporter) OnLoad(context.Context, int, error, *testProcessDetails, Status) error {
	return nil
}

func (testReporter) OnRun(context.Context, int, error, *testProcessDetails) error {
	return nil
}

func (testReporter) OnExit(context.Context, int, *testProcessDetails) error {
	return nil
}

type testSettingsGetter struct{}

func (testSettingsGetter) Settings(context.Context, logr.Logger, *testProcessDetails, *distro.OtelDistro) (Settings, error) {
	return Settings{}, nil
}

type testFactory struct {
	inst Instrumentation
	err  error
}

func (f testFactory) CreateInstrumentation(context.Context, int, Settings) (Instrumentation, error) {
	return f.inst, f.err
}

type testInstrumentation struct {
	loadErr error
}

func (i *testInstrumentation) Load(context.Context) (Status, error) {
	return Status{}, i.loadErr
}

func (i *testInstrumentation) Run(context.Context) error {
	return nil
}

func (i *testInstrumentation) Close(context.Context) error {
	return nil
}

func (i *testInstrumentation) ApplyConfig(context.Context, Config) error {
	return nil
}

func newTestManager(t *testing.T, factory Factory) (*manager[testProcessGroup, testConfigGroup, *testProcessDetails], *trackingDetector) {
	t.Helper()

	managerMetrics, err := newManagerMetrics(meter)
	if err != nil {
		t.Fatalf("failed to create manager metrics: %v", err)
	}

	detector := &trackingDetector{}
	return &manager[testProcessGroup, testConfigGroup, *testProcessDetails]{
		detector:              detector,
		handler:               &Handler[testProcessGroup, testConfigGroup, *testProcessDetails]{Reporter: testReporter{}, SettingsGetter: testSettingsGetter{}},
		factories:             map[string]Factory{"test-distro": factory},
		logger:                commonlogger.LoggerCompat().With("subsystem", "test"),
		detailsByPid:          make(map[int]*instrumentationDetails[testProcessGroup, testConfigGroup, *testProcessDetails]),
		detailsByConfigGroup:  make(map[testConfigGroup]map[int]*instrumentationDetails[testProcessGroup, testConfigGroup, *testProcessDetails]),
		detailsByProcessGroup: make(map[testProcessGroup]map[int]*instrumentationDetails[testProcessGroup, testConfigGroup, *testProcessDetails]),
		metrics:               managerMetrics,
	}, detector
}

func testDetails() *testProcessDetails {
	return &testProcessDetails{
		distro: &distro.OtelDistro{
			Name:     "test-distro",
			Language: common.GoProgrammingLanguage,
		},
	}
}

func TestTryInstrumentTracksFailedInstrumentationForExitCleanup(t *testing.T) {
	initErr := errors.New("init failed")
	loadErr := errors.New("load failed")

	tests := []struct {
		name    string
		factory Factory
		wantErr error
	}{
		{
			name:    "init failure",
			factory: testFactory{err: initErr},
			wantErr: initErr,
		},
		{
			name:    "load failure",
			factory: testFactory{inst: &testInstrumentation{loadErr: loadErr}},
			wantErr: loadErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const pid = 1234
			manager, detector := newTestManager(t, tt.factory)

			err := manager.tryInstrument(context.Background(), testDetails(), pid)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}

			if len(detector.tracked) != 1 || detector.tracked[0] != pid {
				t.Fatalf("expected failed pid %d to be tracked for exit cleanup, got %v", pid, detector.tracked)
			}

			details, found := manager.detailsByPid[pid]
			if !found {
				t.Fatalf("expected failed pid %d to be retained for status cleanup and retry", pid)
			}
			if details.inst != nil {
				t.Fatalf("expected failed pid %d to be retained without a live instrumentation", pid)
			}
		})
	}
}
