package common

import "github.com/odigos-io/odigos/common/profilecache"

// ProfileStoreRef is the read/lifecycle contract the GraphQL layer depends on; it
// lives in the shared common/profilecache package (used by frontend + vm-agent).
type ProfileStoreRef = profilecache.StoreRef
