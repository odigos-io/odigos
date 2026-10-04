package collectormetrics

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func destinationDataPoint(destinationID string, value float64, t time.Time) pmetric.NumberDataPoint {
	dp := pmetric.NewNumberDataPoint()
	dp.Attributes().PutStr(exporterMetricAttributesKey, "otlp/"+destinationID)
	dp.SetDoubleValue(value)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(t))
	return dp
}

func TestRemoveClusterCollectorKeepsOtherCollectorsTraffic(t *testing.T) {
	dm := newClusterCollectorMetrics()
	now := time.Now()

	dm.updateDestinationMetricsByExporter(destinationDataPoint("odigos.io.dest.jaeger", 100, now), exporterSentSpansMetricName, "gateway-a")
	dm.updateDestinationMetricsByExporter(destinationDataPoint("odigos.io.dest.jaeger", 200, now), exporterSentSpansMetricName, "gateway-b")

	dm.removeClusterCollector("gateway-a")

	sdm, ok := dm.destinations["odigos.io.dest.jaeger"]
	if !ok {
		t.Fatal("removeClusterCollector dropped the destination itself")
	}
	if _, ok := sdm.clusterCollectorsTraffic["gateway-a"]; ok {
		t.Error("traffic of the removed cluster collector was kept")
	}
	if _, ok := sdm.clusterCollectorsTraffic["gateway-b"]; !ok {
		t.Error("traffic of a live cluster collector was removed")
	}
}

// The OTLP receive path inserts into the destinations map from the gRPC receiver goroutine,
// while the delete notifications loop and the GraphQL resolvers read the same map from their
// own goroutines. Every one of them must hold destinationsMu: an unsynchronized read against a
// concurrent insert makes the Go runtime abort the whole frontend process with a fatal error
// that no recover can catch.
func TestClusterCollectorMetricsConcurrentDestinationAccess(t *testing.T) {
	dm := newClusterCollectorMetrics()

	const destinations = 5000

	ingestDone := make(chan struct{})
	var wg sync.WaitGroup

	// the OTLP gRPC receive path, seeing each destination for the first time
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ingestDone)
		now := time.Now()
		for i := 0; i < destinations; i++ {
			destinationID := fmt.Sprintf("odigos.io.dest.jaeger-%d", i)
			dm.updateDestinationMetricsByExporter(destinationDataPoint(destinationID, float64(i), now), exporterSentSpansMetricName, "gateway-a")
		}
	}()

	// the delete notifications loop, reacting to a deleted gateway pod
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ingestDone:
				return
			default:
				dm.removeClusterCollector("gateway-b")
			}
		}
	}()

	// a GraphQL resolver reading the metrics of a single destination
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ingestDone:
				return
			default:
				dm.metricsByID("odigos.io.dest.jaeger-0")
			}
		}
	}()

	wg.Wait()

	if got := len(dm.destinations); got != destinations {
		t.Fatalf("expected %d tracked destinations, got %d", destinations, got)
	}
}
