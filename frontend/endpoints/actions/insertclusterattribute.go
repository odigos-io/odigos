package actions

import (
	"github.com/gin-gonic/gin"
	"github.com/keyval-dev/odigos/api/odigos/actions/v1alpha1"
	"github.com/keyval-dev/odigos/frontend/kube"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InsertClusterAttributesResponse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func GetActionInsertClusterAttributes(c *gin.Context, odigosns string) {

	actions, err := kube.DefaultClient.ActionsClient.InsertClusterAttributes(odigosns).List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	response := make([]InsertClusterAttributesResponse, 0, len(actions.Items))
	for _, action := range actions.Items {
		response = append(response, InsertClusterAttributesResponse{
			Id:   action.Name,
			Name: action.Spec.ActionName,
		})
	}

	c.JSON(200, response)
}

func GetInsertClusterAttribute(c *gin.Context, odigosns string, id string) {
	action, err := kube.DefaultClient.ActionsClient.InsertClusterAttributes(odigosns).Get(c, id, metav1.GetOptions{})
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

func CreateInsertClusterAttribute(c *gin.Context, odigosns string) {
	var action v1alpha1.InsertClusterAttribute
	if err := c.ShouldBindJSON(&action.Spec); err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}
	action.GenerateName = "ica-"
	generatedAction, err := kube.DefaultClient.ActionsClient.InsertClusterAttributes(odigosns).Create(c, &action, metav1.CreateOptions{})
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

func UpdateInsertClusterAttribute(c *gin.Context, odigosns string, id string) {
	action, err := kube.DefaultClient.ActionsClient.InsertClusterAttributes(odigosns).Get(c, id, metav1.GetOptions{})
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
	action.Spec = v1alpha1.InsertClusterAttributeSpec{}
	if err := c.ShouldBindJSON(&action.Spec); err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}
	action.Name = id

	_, err = kube.DefaultClient.ActionsClient.InsertClusterAttributes(odigosns).Update(c, action, metav1.UpdateOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(204, nil)
}

func DeleteInsertClusterAttribute(c *gin.Context, odigosns string, id string) {
	err := kube.DefaultClient.ActionsClient.InsertClusterAttributes(odigosns).Delete(c, id, metav1.DeleteOptions{})
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
