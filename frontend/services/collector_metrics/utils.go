package collectormetrics

import "time"

func calculateThroughput(diff float64, currentTime, prevTime time.Time) int64 {
	elapsed := currentTime.Sub(prevTime).Seconds()

	var throughput int64
	if diff > 0 && elapsed > 0 {
		throughput = int64(diff / elapsed)
	}

	return throughput
}