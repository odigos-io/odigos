package pprof

import (
	"fmt"

	profilev1 "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
)

type ProfileBuilder struct {
	*profilev1.Profile
	strings                       map[string]int
	externalLocationID2LocationID map[ExternalLocationID]PPROFLocationID
	externalFunctionID2FunctionID map[ExternalFunctionID]PPROFFunctionID
	externalSampleID2SampleIndex  map[sampleID]uint32
	metricName                    string

	truncatedLoc uint64
}

type sampleID struct {
	locationsID uint64
	correlation StacktraceCorrelation
}

// NewProfileBuilderWithLabels creates a new ProfileBuilder with the given nanoseconds timestamp and labels.
func NewProfileBuilderWithLabels(ts int64) *ProfileBuilder {
	profile := new(profilev1.Profile)
	profile.TimeNanos = ts
	profile.Mapping = append(profile.Mapping, &profilev1.Mapping{
		Id: 1, HasFunctions: true,
	})
	p := &ProfileBuilder{
		Profile:                       profile,
		strings:                       map[string]int{},
		externalFunctionID2FunctionID: map[ExternalFunctionID]PPROFFunctionID{},
		externalLocationID2LocationID: map[ExternalLocationID]PPROFLocationID{},
	}
	p.addString("")
	return p
}

type ExternalFunctionID uint32
type ExternalLocationID struct {
	ExternalFunctionID ExternalFunctionID
	Line               uint32
}
type PPROFFunctionID uint64
type PPROFLocationID uint64

func (m *ProfileBuilder) AddSampleType(typ, unit string) {
	m.Profile.SampleType = append(m.Profile.SampleType, &profilev1.ValueType{
		Type: m.addString(typ),
		Unit: m.addString(unit),
	})
}

func (m *ProfileBuilder) MetricName(name string) {
	m.metricName = name
}

func (m *ProfileBuilder) PeriodType(periodType string, periodUnit string) {
	m.Profile.PeriodType = &profilev1.ValueType{
		Type: m.addString(periodType),
		Unit: m.addString(periodUnit),
	}
}

func (m *ProfileBuilder) addString(s string) int64 {
	i, ok := m.strings[s]
	if !ok {
		i = len(m.strings)
		m.strings[s] = i
		m.StringTable = append(m.StringTable, s)
	}
	return int64(i)
}

func (m *ProfileBuilder) FindLocationByExternalID(externalLocationID ExternalLocationID) (PPROFLocationID, bool) {
	loc, ok := m.externalLocationID2LocationID[externalLocationID]
	return loc, ok
}

func (m *ProfileBuilder) FindFunctionByExternalID(externalFunctionID ExternalFunctionID) (PPROFFunctionID, bool) {
	loc, ok := m.externalFunctionID2FunctionID[externalFunctionID]
	return loc, ok
}

func (m *ProfileBuilder) AddExternalFunction(frame string, id ExternalFunctionID) PPROFFunctionID {
	ret := m.addFunction(frame)
	m.externalFunctionID2FunctionID[id] = ret
	return ret
}

func (m *ProfileBuilder) addFunction(frame string) PPROFFunctionID {
	fname := m.addString(frame)
	funcID := uint64(len(m.Function)) + 1
	m.Function = append(m.Function, &profilev1.Function{
		Id:   funcID,
		Name: fname,
	})
	ret := PPROFFunctionID(funcID)
	return ret
}

func (m *ProfileBuilder) AddExternalLocation(id ExternalLocationID, pprofFunctionID PPROFFunctionID) PPROFLocationID {
	ret := m.addLocation(pprofFunctionID, id.Line)
	m.externalLocationID2LocationID[id] = ret
	return ret
}

func (m *ProfileBuilder) addLocation(pprofFunctionID PPROFFunctionID, line uint32) PPROFLocationID {
	locID := uint64(len(m.Location)) + 1
	m.Location = append(m.Location, &profilev1.Location{
		Id:        locID,
		MappingId: uint64(1),
		Line:      []*profilev1.Line{{FunctionId: uint64(pprofFunctionID), Line: int64(line)}},
	})
	ret := PPROFLocationID(locID)
	return ret
}

func (m *ProfileBuilder) AddExternalSampleWithLabels(locs []uint64, values []int64, labelsCtx *Context, labelsSnapshot *LabelsSnapshot, locationsID uint64, correlation StacktraceCorrelation) {
	sample := &profilev1.Sample{
		LocationId: locs,
		Value:      values,
	}
	if m.externalSampleID2SampleIndex == nil {
		m.externalSampleID2SampleIndex = map[sampleID]uint32{}
	}
	m.externalSampleID2SampleIndex[sampleID{locationsID: locationsID, correlation: correlation}] = uint32(len(m.Profile.Sample))
	m.Profile.Sample = append(m.Profile.Sample, sample)
	if labelsSnapshot == nil {
		return
	}
	const LabelProfileId = "profile_id"
	const LabelSpanName = "span_name"
	capacity := 0
	if labelsCtx != nil {
		capacity += len(labelsCtx.Labels)
	}
	if correlation.SpanId != 0 {
		capacity++
	}
	if correlation.SpanName != 0 {
		capacity++
	}
	if labelsCtx != nil {
		sample.Label = make([]*profilev1.Label, 0, capacity)
		for k, v := range labelsCtx.Labels {
			sample.Label = append(sample.Label, &profilev1.Label{
				Key: m.addString(labelsSnapshot.Strings[k]),
				Str: m.addString(labelsSnapshot.Strings[v]),
			})
		}

	}
	if correlation.SpanId != 0 {
		sample.Label = append(sample.Label, &profilev1.Label{
			Key: m.addString(LabelProfileId),
			Str: m.addString(profileIdString(correlation.SpanId)),
		})
	}
	if correlation.SpanName != 0 {
		spanName := labelsSnapshot.Strings[int64(correlation.SpanName)]
		if spanName != "" {
			sample.Label = append(sample.Label, &profilev1.Label{
				Key: m.addString(LabelSpanName),
				Str: m.addString(spanName),
			})
		}
	}
}

func profileIdString(profileId uint64) string {
	//todo how to do with no sprintf
	return fmt.Sprintf("%016x", profileId)
	//return strconv.FormatUint(profileId, 16)
}

type StacktraceCorrelation struct {
	ContextId uint64
	SpanId    uint64
	SpanName  uint64
}

// FindExternalSampleWithLabels deprecated
func (m *ProfileBuilder) FindExternalSampleWithLabels(locationsID uint64, correlation StacktraceCorrelation) *profilev1.Sample {
	return m.FindExternalSampleWithCorrelation(locationsID, correlation)
}

func (m *ProfileBuilder) FindExternalSampleWithCorrelation(locationsID uint64, correlation StacktraceCorrelation) *profilev1.Sample {
	sampleIndex, ok := m.externalSampleID2SampleIndex[sampleID{locationsID: locationsID, correlation: correlation}]
	if !ok {
		return nil
	}
	sample := m.Profile.Sample[sampleIndex]
	return sample
}

func (m *ProfileBuilder) getTruncatedLocation() uint64 {
	if m.truncatedLoc != 0 {
		return m.truncatedLoc
	}
	const truncatedFrameName = "[truncated]"
	f := m.addFunction(truncatedFrameName)
	location := m.addLocation(f, 0)
	m.truncatedLoc = uint64(location)
	return m.truncatedLoc
}
