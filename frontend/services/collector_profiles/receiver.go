package collectorprofiles

// ProfileStoreRef is used by GraphQL resolvers for on-demand profiling (OTLP → buffer → merge).
type ProfileStoreRef interface {
	StartViewing(sourceKey string)
	GetProfileData(sourceKey string) [][]byte
	MaxSlots() int
	DebugSlots() (activeKeys []string, keysWithData []string)
	MemoryStats() (totalBytes int, maxSlots int, slotMaxBytes int, ttlSeconds int)
}
