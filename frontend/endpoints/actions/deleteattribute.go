package actions

import (
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/frontend/kube"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetDeleteAttribute(c *gin.Context, odigosns string, id string) {
	action, err := kube.DefaultClient.ActionsClient.DeleteAttributes(odigosns).Get(c, id, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.JSON(404, gin.H{
				"error": "not found",
			})
			return
		} else {
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
	}
	c.JSON(200, action.Spec)
}

func CreateDeleteAttribute(c *gin.Context, odigosns string) {
	var action v1alpha1.DeleteAttribute
	if err := c.ShouldBindJSON(&action.Spec); err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}
	action.GenerateName = "da-"
	generatedAction, err := kube.DefaultClient.ActionsClient.DeleteAttributes(odigosns).Create(c, &action, metav1.CreateOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(201, gin.H{
		"id": generatedAction.Name,
	})
}

func UpdateDeleteAttribute(c *gin.Context, odigosns string, id string) {
	action, err := kube.DefaultClient.ActionsClient.DeleteAttributes(odigosns).Get(c, id, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.JSON(404, gin.H{
				"error": "not found",
			})
			return
		} else {
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
		}
		return
	}
	action.Spec = v1alpha1.DeleteAttributeSpec{}
	if err := c.ShouldBindJSON(&action.Spec); err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}
	action.Name = id

	_, err = kube.DefaultClient.ActionsClient.DeleteAttributes(odigosns).Update(c, action, metav1.UpdateOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(204, nil)
}

func DeleteDeleteAttribute(c *gin.Context, odigosns string, id string) {
	err := kube.DefaultClient.ActionsClient.DeleteAttributes(odigosns).Delete(c, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.JSON(404, gin.H{
				"error": "not found",
			})
			return
		} else {
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
	}
	c.JSON(204, nil)
}
