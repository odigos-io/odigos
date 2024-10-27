package endpoints

import (
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func DescribeOdigos(c *gin.Context) {
	ctx := c.Request.Context()
	odiogosNs := env.GetCurrentNamespace()
	describeText := describe.DescribeOdigos(ctx, kube.DefaultClient, kube.DefaultClient.OdigosClient, odiogosNs)
	c.Writer.WriteString(describeText)
}

func DescribeSource(c *gin.Context, ns string, kind string, name string) {
	ctx := c.Request.Context()
	switch kind {
	case "deployment":
		describeText := describe.DescribeDeployment(ctx, kube.DefaultClient.Interface, kube.DefaultClient.OdigosClient, ns, name)
		c.Writer.WriteString(describeText)
	case "daemonset":
		describeText := describe.DescribeDaemonSet(ctx, kube.DefaultClient.Interface, kube.DefaultClient.OdigosClient, ns, name)
		c.Writer.WriteString(describeText)
	case "statefulset":
		describeText := describe.DescribeStatefulSet(ctx, kube.DefaultClient.Interface, kube.DefaultClient.OdigosClient, ns, name)
		c.Writer.WriteString(describeText)
	default:
		c.JSON(404, gin.H{
			"message": "kind not supported",
		})
	}
}
