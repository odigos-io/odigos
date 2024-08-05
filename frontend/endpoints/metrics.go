package endpoints

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	collectormetrics "github.com/odigos-io/odigos/frontend/endpoints/collector_metrics"
	"github.com/odigos-io/odigos/frontend/endpoints/common"
)

type singleSourceMetricsResponse struct {
	common.SourceID
	TotalDataSent int64 `json:"totalDataSent"`
	Throughput    int64 `json:"throughput"`
}

func GetSingleSourceMetrics(c *gin.Context, m *collectormetrics.OdigosMetricsConsumer) {
	ns := c.Param("namespace")
	kind := c.Param("kind")
	name := c.Param("name")

	sID := common.SourceID{
		Namespace: ns,
		Kind:      kind,
		Name:      name,
	}
	metric, ok := m.GetSingleSourceMetrics(sID)
	if !ok {
		returnError(c, fmt.Errorf("source not found %v", sID))
		return
	}

	c.JSON(http.StatusOK,
		singleSourceMetricsResponse{
			SourceID:      sID,
			TotalDataSent: metric.TotalDataSent(),
			Throughput:    metric.TotalThroughput(),
		},
	)
}

type sourcesMetricsResponse struct {
	Sources []singleSourceMetricsResponse `json:"sources"`
}

func GetSourcesMetrics(c *gin.Context, m *collectormetrics.OdigosMetricsConsumer) {
	metrics := m.GetSourcesMetrics()

	var sources []singleSourceMetricsResponse
	for sID, metric := range metrics {
		sources = append(sources, singleSourceMetricsResponse{
			SourceID:      sID,
			TotalDataSent: metric.TotalDataSent(),
			Throughput:    metric.TotalThroughput(),
		})
	}

	c.JSON(http.StatusOK, sourcesMetricsResponse{Sources: sources})
}

type singleDestinationMetricsResponse struct {
	ID            string `json:"id"`
	TotalDataSent int64  `json:"totalDataSent"`
	Throughput    int64  `json:"throughput"`
}

func GetSingleDestinationMetrics(c *gin.Context, m *collectormetrics.OdigosMetricsConsumer) {
	destId := c.Param("id")

	metric, ok := m.GetSingleDestinationMetrics(destId)
	if !ok {
		returnError(c, fmt.Errorf("destination not found %v", destId))
		return
	}

	c.JSON(http.StatusOK,
		singleDestinationMetricsResponse{
			ID:            destId,
			TotalDataSent: metric.TotalDataSent(),
			Throughput:    metric.TotalThroughput(),
		},
	)
}

type destinationsMetricsResponse struct {
	Destinations []singleDestinationMetricsResponse `json:"destinations"`
}

func GetDestinationsMetrics(c *gin.Context, m *collectormetrics.OdigosMetricsConsumer) {
	metrics := m.GetDestinationsMetrics()

	var destinations []singleDestinationMetricsResponse
	for destId, metric := range metrics {
		destinations = append(destinations, singleDestinationMetricsResponse{
			ID:            destId,
			TotalDataSent: metric.TotalDataSent(),
			Throughput:    metric.TotalThroughput(),
		})
	}

	c.JSON(http.StatusOK, destinationsMetricsResponse{Destinations: destinations})
}

type overviewMetricsResponse struct {
	sourcesMetricsResponse
	destinationsMetricsResponse
}

func GetOverviewMetrics(c *gin.Context, m *collectormetrics.OdigosMetricsConsumer) {
	sources := m.GetSourcesMetrics()
	destinations := m.GetDestinationsMetrics()

	var sourcesResp []singleSourceMetricsResponse
	for sID, metric := range sources {
		sourcesResp = append(sourcesResp, singleSourceMetricsResponse{
			SourceID:      sID,
			TotalDataSent: metric.TotalDataSent(),
			Throughput:    metric.TotalThroughput(),
		})
	}

	var destinationsResp []singleDestinationMetricsResponse
	for destId, metric := range destinations {
		destinationsResp = append(destinationsResp, singleDestinationMetricsResponse{
			ID:            destId,
			TotalDataSent: metric.TotalDataSent(),
			Throughput:    metric.TotalThroughput(),
		})
	}

	c.JSON(http.StatusOK, overviewMetricsResponse{
		sourcesMetricsResponse{Sources: sourcesResp},
		destinationsMetricsResponse{Destinations: destinationsResp},
	})
}
