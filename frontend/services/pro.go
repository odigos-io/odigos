package services

import (
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"
)

func UpdateToken(c *gin.Context) {
	ctx := c.Request.Context()

	err := pro.UpdateOdigosToken(ctx, kube.DefaultClient, env.GetCurrentNamespace(), c.Param("onPremToken"))
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
