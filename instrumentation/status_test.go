package instrumentation

import (
	"context"
	"errors"
	"testing"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

type statusTestDetails struct{ ProcessDetails[string, string] }

type statusTestInstrumentation struct {
	Instrumentation
	revision uint64
	reads    int
}

func (s *statusTestInstrumentation) StatusRevision() uint64   { return s.revision }
func (s *statusTestInstrumentation) Status() (Status, uint64) { s.reads++; return Status{}, s.revision }

type statusTestReporter struct {
	Reporter[string, string, statusTestDetails]
	calls   []int
	failPID int
}

func (s *statusTestReporter) OnStatus(_ context.Context, pid int, _ statusTestDetails, _ Status) error {
	s.calls = append(s.calls, pid)
	if pid == s.failPID {
		return errors.New("report unavailable")
	}
	return nil
}

func TestChangedStatusCoalescesAndRetriesWithoutStarvation(t *testing.T) {
	r := &statusTestReporter{failPID: 1}
	first, second := &statusTestInstrumentation{revision: 30}, &statusTestInstrumentation{revision: 1}
	m := &manager[string, string, statusTestDetails]{logger: commonlogger.LoggerCompat(), handler: &Handler[string, string, statusTestDetails]{Reporter: r}, detailsByPid: map[int]*instrumentationDetails[string, string, statusTestDetails]{1: {distroInst: first}, 2: {distroInst: second}}}
	m.reportChangedStatus(t.Context())
	if len(r.calls) != 1 || r.calls[0] != 1 || m.detailsByPid[1].statusRevision != 0 {
		t.Fatal("failed report must remain pending; only one report per tick")
	}
	m.reportChangedStatus(t.Context())
	if len(r.calls) != 2 || r.calls[1] != 2 {
		t.Fatal("failed process starved another process")
	}
	first.revision = 31
	r.failPID = 0
	m.reportChangedStatus(t.Context())
	if m.detailsByPid[1].statusRevision != 31 {
		t.Fatal("retry did not coalesce to the newest revision")
	}
	for i := 0; i < 10000; i++ {
		m.reportChangedStatus(t.Context())
	}
	if len(r.calls) != 3 || first.reads != 2 || second.reads != 1 {
		t.Fatal("unchanged state performed snapshot or API work")
	}
	// A new SDK lifetime reusing this PID must not inherit its predecessor's revision.
	m.detailsByPid[1] = &instrumentationDetails[string, string, statusTestDetails]{distroInst: &statusTestInstrumentation{revision: 1}}
	m.reportChangedStatus(t.Context())
	if len(r.calls) != 4 || m.detailsByPid[1].statusRevision != 1 {
		t.Fatal("replacement status was lost")
	}
	delete(m.detailsByPid, 1)
	m.reportChangedStatus(t.Context())
	if len(r.calls) != 4 {
		t.Fatal("reported an exited process")
	}
}
