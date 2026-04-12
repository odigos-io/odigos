package profiles

import (
	"github.com/odigos-io/odigos/frontend/services/common"
)

// ChunksForSourceKey returns a shallow snapshot of buffered OTLP profile chunks for the given
// source key. Each element is one protobuf-encoded ExportProfilesServiceRequest (see pdata
// ProtoMarshaler.MarshalProfiles). The returned inner byte slices are read-only: do not mutate
// them; they may share backing arrays with the live buffer until the request finishes.
func ChunksForSourceKey(store common.ProfileStoreRef, sourceKey string) [][]byte {
	if store == nil {
		return nil
	}
	return store.GetProfileData(sourceKey)
}
