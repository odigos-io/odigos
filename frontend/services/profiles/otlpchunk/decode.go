// Package otlpchunk decodes stored profiling buffer chunks: each chunk is one OTLP
// ExportProfilesServiceRequest protobuf (same wire as pdata.ProtoMarshaler.MarshalProfiles).
package otlpchunk

import (
	"fmt"

	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	"google.golang.org/protobuf/proto"
)

// UnmarshalExportProfilesRequest decodes one stored chunk (protobuf wire) into an export request.
// Empty input returns an error.
func UnmarshalExportProfilesRequest(chunk []byte) (*pprofileotlp.ExportProfilesServiceRequest, error) {
	if len(chunk) == 0 {
		return nil, fmt.Errorf("empty chunk")
	}
	req := &pprofileotlp.ExportProfilesServiceRequest{}
	if err := proto.Unmarshal(chunk, req); err != nil {
		return nil, err
	}
	return req, nil
}
