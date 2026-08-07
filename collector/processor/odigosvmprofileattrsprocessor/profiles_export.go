package odigosvmprofileattrsprocessor

import "go.opentelemetry.io/collector/pdata/pprofile"

// profilesExportable reports whether the profiles payload should be sent to a downstream exporter.
// Pyroscope OTLP ingest rejects requests with no resource profiles (InvalidArgument:
// "missing resource profiles").
func profilesExportable(md pprofile.Profiles) bool {
	return md.ResourceProfiles().Len() > 0
}
