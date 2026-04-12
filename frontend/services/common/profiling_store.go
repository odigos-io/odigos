package common

// ProfileMemoryStats summarizes buffered profiling data and configured limits for the UI / GraphQL.
type ProfileMemoryStats struct {
	TotalBytes          int
	MaxSlots            int
	SlotMaxBytes        int
	SlotTTLSeconds      int
	MaxTotalBytesBudget int // worst-case if every slot uses its full rolling buffer (maxSlots × slotMaxBytes)
}

// ProfileStoreRef is the narrow API GraphQL and OTLP use from the profiling buffer.
type ProfileStoreRef interface {
	StartViewing(sourceKey string)
	RemoveSlot(sourceKey string)
	GetProfileData(sourceKey string) [][]byte
	MaxSlots() int
	DebugSlots() (activeKeys []string, keysWithData []string)
	MemoryStats() ProfileMemoryStats
}
