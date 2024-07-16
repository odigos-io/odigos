package endpoints

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	collectormetrics "github.com/odigos-io/odigos/frontend/endpoints/collector_metrics"
	"github.com/odigos-io/odigos/frontend/endpoints/common"
)

type sourceMetricsResponse struct {
	common.SourceID
	TotalDataSent int64   `json:"totalDataSent"`
	Throughput    int64   `json:"throughput"`
}

func GetMetrics(c *gin.Context, m *collectormetrics.OdigosMetricsConsumer) {
	ns := c.Param("namespace")
	kind := c.Param("kind")
	name := c.Param("name")

	sID := common.SourceID{
		Namespace: ns,
		Kind:      kind,
		Name:      name,
	}
	metric, ok := m.GetSourceTrafficMetrics(sID)
	if !ok {
		returnError(c, fmt.Errorf("source not found %v", sID))
		return
	}

	c.JSON(http.StatusOK,
		sourceMetricsResponse{
			SourceID:     sID,
			TotalDataSent: metric.TotalDataSent(),
			Throughput:    metric.TotalThroughput(),
		},
	)
}