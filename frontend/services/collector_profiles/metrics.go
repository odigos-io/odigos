package collectorprofiles

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const promNamespace = "odigos_ui"

var (
	profilingOtelBatchesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: "profiling",
		Name:      "otlp_batches_total",
		Help:      "OTLP profile batches handled by the UI (ConsumeProfiles invocations with at least one ResourceProfile).",
	})
	profilingResourceProfilesReceivedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: "profiling",
		Name:      "resource_profiles_received_total",
		Help:      "ResourceProfile entries seen across all profiling batches.",
	})
	profilingResourceProfilesDroppedNoSourceKeyTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: "profiling",
		Name:      "resource_profiles_dropped_no_source_key_total",
		Help:      "ResourceProfile entries dropped because namespace/workload attributes could not be derived.",
	})
	profilingResourceProfilesDroppedNoSlotTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: "profiling",
		Name:      "resource_profiles_dropped_no_slot_total",
		Help:      "ResourceProfile entries dropped because no in-memory profiling slot exists for the workload (open the Profiler tab or call enableSourceProfiling).",
	})
	profilingResourceProfilesStoredTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: "profiling",
		Name:      "resource_profiles_stored_total",
		Help:      "ResourceProfile entries written to the in-memory profiling buffer.",
	})
	profilingBatchesFullyDroppedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: "profiling",
		Name:      "otlp_batches_fully_dropped_total",
		Help:      "Batches where every ResourceProfile was dropped (no slot and/or no source key).",
	})
	profilingChunkMarshalErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: "profiling",
		Name:      "chunk_marshal_errors_total",
		Help:      "Errors marshaling a ResourceProfile to JSON for storage.",
	})
)
