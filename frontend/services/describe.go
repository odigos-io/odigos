package services

import (
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func DescribeOdigos(c *gin.Context) {
	ctx := c.Request.Context()
	odiogosNs := env.GetCurrentNamespace()
	desc, err := describe.DescribeOdigos(ctx, kube.DefaultClient, kube.DefaultClient.OdigosClient, odiogosNs)
	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}

	// construct the http response code based on the status of the odigos
	returnCode := 200

	// Check for the Accept header
	acceptHeader := c.GetHeader("Accept")

	if acceptHeader == "application/json" {
		// Return JSON response if Accept header is "application/json"
		c.JSON(returnCode, desc)
	} else {
		describeText := describe.DescribeOdigosToText(desc)
		c.String(returnCode, describeText)
	}
}

func DescribeSource(c *gin.Context) {
	ctx := c.Request.Context()

	ns := c.Param("namespace")
	name := c.Param("name")
	kind := c.Param("kind")

	var desc *source.SourceAnalyze
	var err error
	switch kind {
	case "deployment":
		desc, err = describe.DescribeDeployment(ctx, kube.DefaultClient.Interface, kube.DefaultClient.OdigosClient, ns, name)
	case "daemonset":
		desc, err = describe.DescribeDaemonSet(ctx, kube.DefaultClient.Interface, kube.DefaultClient.OdigosClient, ns, name)
	case "statefulset":
		desc, err = describe.DescribeStatefulSet(ctx, kube.DefaultClient.Interface, kube.DefaultClient.OdigosClient, ns, name)
	case "staticpod":
		desc, err = describe.DescribeStaticPod(ctx, kube.DefaultClient.Interface, kube.DefaultClient.OdigosClient, ns, name)
	case "deploymentconfig":
		desc, err = describe.DescribeDeploymentConfig(ctx, kube.DefaultClient.Interface, kube.DefaultClient.DynamicClient, kube.DefaultClient.OdigosClient, ns, name)
	default:
		c.JSON(404, gin.H{
			"message": "kind not supported",
		})
		return
	}

	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}

	// Check for the Accept header
	acceptHeader := c.GetHeader("Accept")

	if acceptHeader == "application/json" {
		// Return JSON response if Accept header is "application/json"
		c.JSON(200, desc)
	} else {
		describeText := describe.DescribeSourceToText(desc)
		c.String(200, describeText)
	}
}
