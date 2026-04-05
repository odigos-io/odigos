package common

// ProfileStoreRef is the subset of the in-memory profiling store used by GraphQL and OTLP ingestion.
type ProfileStoreRef interface {
	StartViewing(sourceKey string)
	RemoveSlot(sourceKey string)
	GetProfileData(sourceKey string) [][]byte
	MaxSlots() int
	DebugSlots() (activeKeys []string, keysWithData []string)
	MemoryStats() (totalBytes int, maxSlots int, slotMaxBytes int, ttlSeconds int)
}
