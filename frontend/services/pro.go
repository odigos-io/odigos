package services

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"
)

func UpdateToken(c *gin.Context) {
	var request common.TokenPayload

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	onPremToken := request.OnpremToken
	if onPremToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "onprem-token is required",
		})
		return
	}
	ctx := c.Request.Context()

	err := pro.UpdateOdigosToken(ctx, kube.DefaultClient, env.GetCurrentNamespace(), onPremToken)
	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}

	statusCode := 200
	acceptHeader := c.GetHeader("Accept")

	if acceptHeader == "application/json" {
		c.JSON(statusCode, struct{}{})
	} else {
		c.String(statusCode, "")
	}
}
