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
	EnsureSlot(sourceKey string)
	RemoveSlot(sourceKey string)
	ClearSlotBuffer(sourceKey string) bool
	GetProfileData(sourceKey string) [][]byte
	MaxSlots() int
	ActiveSlots() (activeKeys []string, keysWithData []string)
	MemoryStats() ProfileMemoryStats
}
