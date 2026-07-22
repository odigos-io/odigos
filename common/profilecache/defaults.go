package profilecache

import "time"

// Default cache limits, applied by NewStore for any non-positive argument.
const (
	DefaultMaxSlots        = 100
	DefaultSlotMaxBytes    = 5 << 20 // 5 MiB per source
	DefaultSlotTTLSeconds  = 15 * 60 // 15 minutes
	DefaultCleanupInterval = time.Minute
)
