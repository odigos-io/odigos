//go:build ignore

// Run from frontend/services/collector_profiles: go run testdata/gen_chunk_with_symbols.go
package main

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	otel "go.opentelemetry.io/proto/otlp/profiles/v1development"
)

func main() {
	// Indices: 0=""; align with Pyroscope profileType "samples:count:cpu:nanoseconds" for CPU path.
	st := []string{
		"", "samples", "count", "cpu", "nanoseconds",
		"service.name", "test-svc",
		"runtime.main", "main.foo", "main.bar",
		"lib.so",
	}
	dict := &otel.ProfilesDictionary{
		StringTable: st,
		AttributeTable: []*otel.KeyValueAndUnit{{
			KeyStrindex: 5,
			Value:       &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "test-svc"}},
		}},
		FunctionTable: []*otel.Function{
			{NameStrindex: 7, SystemNameStrindex: 0, FilenameStrindex: 0},
			{NameStrindex: 8, SystemNameStrindex: 0, FilenameStrindex: 0},
			{NameStrindex: 9, SystemNameStrindex: 0, FilenameStrindex: 0},
		},
		MappingTable: []*otel.Mapping{{
			MemoryStart:      0x1000,
			MemoryLimit:      0x2000,
			FilenameStrindex: 10,
		}},
		LocationTable: []*otel.Location{
			{MappingIndex: 0, Address: 0x1100, Lines: []*otel.Line{{FunctionIndex: 0, Line: 1}}},
			{MappingIndex: 0, Address: 0x1200, Lines: []*otel.Line{{FunctionIndex: 1, Line: 2}}},
			{MappingIndex: 0, Address: 0x1300, Lines: []*otel.Line{{FunctionIndex: 2, Line: 3}}},
		},
		StackTable: []*otel.Stack{
			{LocationIndices: []int32{2, 1, 0}}, // leaf-first in OTLP stack
			{LocationIndices: []int32{1, 0}},
		},
	}
	prof := &otel.Profile{
		SampleType: &otel.ValueType{TypeStrindex: 1, UnitStrindex: 2},
		PeriodType: &otel.ValueType{TypeStrindex: 3, UnitStrindex: 4},
		Period:     1000000,
		TimeUnixNano: 1000000000,
		Samples: []*otel.Sample{
			{StackIndex: 0, Values: []int64{2}, AttributeIndices: []int32{0}},
			{StackIndex: 1, Values: []int64{1}, AttributeIndices: []int32{0}},
		},
	}
	req := &pprofileotlp.ExportProfilesServiceRequest{
		Dictionary: dict,
		ResourceProfiles: []*otel.ResourceProfiles{{
			ScopeProfiles: []*otel.ScopeProfiles{{
				Profiles: []*otel.Profile{prof},
			}},
		}},
	}
	opts := protojson.MarshalOptions{Indent: "  "}
	b, err := opts.Marshal(req)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("testdata/chunk-with-symbols.json", b, 0644); err != nil {
		panic(err)
	}
	if err := os.WriteFile("testdata/accounting-merged.json", b, 0644); err != nil {
		panic(err)
	}
	fmt.Println("wrote testdata/chunk-with-symbols.json and testdata/accounting-merged.json")
}
