package pprof

import (
	"fmt"
	"io"

	"github.com/grafana/jfr-parser/parser"
)

type pprofOptions struct {
	truncatedFrame       bool
	disablePanicRecovery bool
}
type Option func(*pprofOptions)

func WithTruncatedFrame(v bool) Option {
	return func(o *pprofOptions) {
		o.truncatedFrame = v
	}
}

func WithDisablePanicRecovery(v bool) Option {
	return func(o *pprofOptions) {
		o.disablePanicRecovery = v
	}
}

func ParseJFR(body []byte, pi *ParseInput, jfrLabels *LabelsSnapshot, opts ...Option) (res *Profiles, err error) {
	o := &pprofOptions{
		truncatedFrame:       false,
		disablePanicRecovery: false,
	}
	for i := range opts {
		opts[i](o)
	}

	if !o.disablePanicRecovery {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("jfr parser panic: %v", r)
			}
		}()
	}

	p := parser.NewParser(body, parser.Options{
		SymbolProcessor: parser.ProcessSymbols,
	})
	return parse(p, pi, jfrLabels, o)
}

func parse(parser *parser.Parser, piOriginal *ParseInput, jfrLabels *LabelsSnapshot, opt *pprofOptions) (result *Profiles, err error) {
	var event string

	builders := newJfrPprofBuilders(parser, jfrLabels, piOriginal, opt)

	var values = [2]int64{1, 0}

	for {
		typ, err := parser.ParseEvent()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("jfr parser ParseEvent error: %w", err)
		}
		switch typ {
		case parser.TypeMap.T_EXECUTION_SAMPLE:
			ts := parser.GetThreadState(parser.ExecutionSample.State)
			correlation := StacktraceCorrelation{
				ContextId: parser.ExecutionSample.ContextId,
				SpanId:    parser.ExecutionSample.SpanId,
				SpanName:  parser.ExecutionSample.SpanName,
			}
			if ts != nil && ts.Name != "STATE_SLEEPING" {
				builders.addStacktrace(sampleTypeCPU, correlation, parser.ExecutionSample.StackTrace, values[:1])
			}
			if event == "wall" {
				builders.addStacktrace(sampleTypeWall, correlation, parser.ExecutionSample.StackTrace, values[:1])
			}
		case parser.TypeMap.T_WALL_CLOCK_SAMPLE:
			values[0] = int64(parser.WallClockSample.Samples)
			correlation := StacktraceCorrelation{
				ContextId: parser.WallClockSample.ContextId,
				SpanId:    parser.WallClockSample.SpanId,
				SpanName:  parser.WallClockSample.SpanName,
			}
			ts := parser.GetThreadState(parser.WallClockSample.State)
			if ts != nil && ts.Name == "STATE_RUNNABLE" && event == "wall" {
				builders.addStacktrace(sampleTypeCPU, correlation, parser.WallClockSample.StackTrace, values[:1])
			}
			builders.addStacktrace(sampleTypeWall, correlation, parser.WallClockSample.StackTrace, values[:1])
		case parser.TypeMap.T_ALLOC_IN_NEW_TLAB:
			values[1] = int64(parser.ObjectAllocationInNewTLAB.TlabSize)
			correlation := StacktraceCorrelation{
				ContextId: parser.ObjectAllocationInNewTLAB.ContextId,
				SpanId:    parser.ObjectAllocationInNewTLAB.SpanId,
				SpanName:  parser.ObjectAllocationInNewTLAB.SpanName,
			}
			builders.addStacktrace(sampleTypeInTLAB, correlation, parser.ObjectAllocationInNewTLAB.StackTrace, values[:2])
		case parser.TypeMap.T_ALLOC_OUTSIDE_TLAB:
			values[1] = int64(parser.ObjectAllocationOutsideTLAB.AllocationSize)
			correlation := StacktraceCorrelation{
				ContextId: parser.ObjectAllocationOutsideTLAB.ContextId,
				SpanId:    parser.ObjectAllocationOutsideTLAB.SpanId,
				SpanName:  parser.ObjectAllocationOutsideTLAB.SpanName,
			}
			builders.addStacktrace(sampleTypeOutTLAB, correlation, parser.ObjectAllocationOutsideTLAB.StackTrace, values[:2])
		case parser.TypeMap.T_ALLOC_SAMPLE:
			values[1] = int64(parser.ObjectAllocationSample.Weight)
			builders.addStacktrace(sampleTypeAllocSample, StacktraceCorrelation{}, parser.ObjectAllocationSample.StackTrace, values[:2])
		case parser.TypeMap.T_MONITOR_ENTER:
			values[1] = int64(parser.JavaMonitorEnter.Duration)
			correlation := StacktraceCorrelation{
				ContextId: parser.JavaMonitorEnter.ContextId,
				SpanId:    parser.JavaMonitorEnter.SpanId,
				SpanName:  parser.JavaMonitorEnter.SpanName,
			}
			builders.addStacktrace(sampleTypeLock, correlation, parser.JavaMonitorEnter.StackTrace, values[:2])
		case parser.TypeMap.T_THREAD_PARK:
			values[1] = int64(parser.ThreadPark.Duration)
			builders.addStacktrace(sampleTypeThreadPark, StacktraceCorrelation{}, parser.ThreadPark.StackTrace, values[:2])
		case parser.TypeMap.T_LIVE_OBJECT:
			builders.addStacktrace(sampleTypeLiveObject, StacktraceCorrelation{}, parser.LiveObject.StackTrace, values[:1])
		case parser.TypeMap.T_MALLOC:
			values[1] = int64(parser.Malloc.Size)
			builders.addStacktrace(sampleTypeMalloc, StacktraceCorrelation{}, parser.Malloc.StackTrace, values[:2])
		case parser.TypeMap.T_ACTIVE_SETTING:
			if parser.ActiveSetting.Name == "event" {
				event = parser.ActiveSetting.Value
			}
		}
	}

	result = builders.build(event)

	return result, nil
}
