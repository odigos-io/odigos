package configgrpc

import (
	"sync/atomic"
	_ "unsafe"
)

// This package is used to protect the collector from crashing due to out of memory.
//
// Why avoiding the collector from crashing (in general) is so important:
// - if pod is crashing short while after it started, the metrics might not have been recorded for HPA to scale up which prevents autohealing.
// - crashing pods can show up in dashboards, create alert and introcue operational noise.
// - when a pod crashes it loses all the telemetry items it was processing and not yet exported.
// - crashing collector can effectively can deny service to the user (no telemetry data makes it to destinations).
// - degrade user experience confidence and trust over time.
//
// Odigos uses k8s best practice and sets memory and cpu resource request and limits.
// this prevents the node from degraded performance in case one pod is (intentionally or not)
// consuming too many resources.
// memory limit is a fixed number of memory a container can consume, after which it is OOM killed by the OS.
// So the challenge of protecting the collector from OOM crashing,
// is the challenge of making sure that it's memory ALWAYS stays below the configured limit.
//
// In ideal case, the rate at which we export data out from the process is larger than the rate at which we injest data.
// So the memory the collector consumes is roughly used:
// - while telemetry data is being processed in the collector.
// - to batch telemetry data (for some time) before sending it down the pipeline.
// - in tail sampler storage.
// - in exporter queues waiting for successful ack on the export.
//
// Each of the above cases can cause the collector to build up pressure and progress till the hard limit.
//
// Examples:
// - downstream (where we send data from collector) can experience pressue and run with inadquate resources.
//   it may take longer time to response, return "retriable" error, or timeout.
//   this may be transient or long-lasting state.
// - destination might be down or misconfigured (due to missconfiguration, downtime, network issues etc).
// - the spans might have many or attributes with large payload (increase allocations in heap).
// - many senders can overwhelm the collector in large clusters.
//
// To protect the collector and avoid dropping data, we need to make sure we do not allow new allocations
// if we are above or nearing the danguages limit.
//
// To do it, we use the go runtime garbage collector to read internal gc state,
// and compute a boolean value indicating wether to accept or reject new allocations.

// miror the types from go runtime which uses sysMemStat.
// https://github.com/golang/go/blob/44c5956bf7454ca178c596eb87578ea61d6c9dee/src/runtime/mstats.go#L643
type sysMemStat uint64

func (s *sysMemStat) load() uint64 {
	return atomic.LoadUint64((*uint64)(s))
}

// using go linkname makes it so that the following variable is the same as
// go runtime internal garbage collector variable.
// this allows us to get internal values using by the GC.
// it is considered bad practice and should be avoided.
// used here since:
// - there is no other way to obtain those value which is cheap for high frequency tests.
// - we know what version of go it's going to use and control when it's upgraded.
// - the odigos collector is an application and not 3rd party library, thus we do not need to supply guarantees to arbitrary users.
//
// it would be nice to do it in an idomatic way not needing the go linkname if that is ever added in the future.
//
//go:linkname runtimeGCController runtime.gcController
var runtimeGCController gcControllerState

//go:linkname runtimeHeapGoal runtime.(*gcControllerState).heapGoal
func runtimeHeapGoal(*gcControllerState) uint64

// following struct is a mirror of the exact struct used by the go runtime.
// notice that it must match exactly (field order and types).
// if go ever changes the internal struct, this need to be updated as well,
// or we can get invalid values when accessing those fields.
type gcControllerState struct {
	gcPercent                  atomic.Int32
	memoryLimit                atomic.Int64
	heapMinimum                uint64
	runway                     atomic.Uint64
	consMark                   float64
	lastConsMark               [4]float64
	gcPercentHeapGoal          atomic.Uint64
	sweepDistMinTrigger        atomic.Uint64
	triggered                  uint64
	lastHeapGoal               uint64
	heapLive                   atomic.Uint64
	heapScan                   atomic.Uint64
	lastHeapScan               uint64
	lastStackScan              atomic.Uint64
	maxStackScan               atomic.Uint64
	globalsScan                atomic.Uint64
	heapMarked                 uint64
	heapScanWork               atomic.Int64
	stackScanWork              atomic.Int64
	globalsScanWork            atomic.Int64
	bgScanCredit               atomic.Int64
	assistTime                 atomic.Int64
	dedicatedMarkTime          atomic.Int64
	fractionalMarkTime         atomic.Int64
	idleMarkTime               atomic.Int64
	markStartTime              int64
	dedicatedMarkWorkersNeeded atomic.Int64
	idleMarkWorkers            atomic.Uint64
	assistWorkPerByte          atomic.Uint64 // This was Float64 originally (from go internals). not used so don't matter
	assistBytesPerWork         atomic.Uint64 // This was Float64 originally (from go internals). not used so don't matter
	fractionalUtilizationGoal  float64

	// fields used for memory limiting goal calculation
	heapInUse    sysMemStat
	heapReleased sysMemStat
	heapFree     sysMemStat
	totalAlloc   atomic.Uint64
	totalFree    atomic.Uint64
	mappedReady  atomic.Uint64

	test bool
	_    [64]byte
}

func isMemLimitReached() bool {

	// fast check - if the mapped memory is below the limit, we are good.
	// this check is expected to cover most cases (normal operationwhen memory limit is not reached)
	memoryLimit := runtimeGCController.memoryLimit.Load()
	mappedReady := runtimeGCController.mappedReady.Load()
	if uint64(memoryLimit) > mappedReady {
		return false
	}

	// any bytes in heap free are accounted for in mappedReady,
	// but is available space to make new allocations.
	heapFree := runtimeGCController.heapFree.load()
	if uint64(memoryLimit) > (mappedReady - heapFree) {
		return false
	}

	// this is the "correct" check to make (which follows what go runtime is doing).
	// it will compare the heap live with the heap goal.
	// if we are above the goal, it means a GC cycle could not lower the memory limit to acceptable level.
	heapGoal := runtimeHeapGoal(&runtimeGCController)
	heapLive := runtimeGCController.heapLive.Load()

	if heapLive < heapGoal {
		// we are below the goal, we are good, no garbage collection is needed.
		return false
	}

	// live heap is above the goal => we are not able to make new allocations safely.
	return true
}

// for debug perpuses
// func debugPrintMemLimitNumbers() {
// 	fmt.Println("--------------------------------")
// 	fmt.Println("memoryLimit", runtimeGCController.memoryLimit.Load())
// 	heapGoal := runtimeHeapGoal(&runtimeGCController)
// 	fmt.Println("heapGoal", heapGoal)
// 	fmt.Println("heapLive", runtimeGCController.heapLive.Load())
// 	fmt.Println("mappedReady", runtimeGCController.mappedReady.Load())
// 	fmt.Println("totalFree", runtimeGCController.totalFree.Load())
// }
