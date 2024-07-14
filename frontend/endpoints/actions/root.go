package actions

import (
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IcaInstanceResponse struct {
	Id   string      `json:"id"`
	Type string      `json:"type"`
	Spec interface{} `json:"spec"`
}

func GetActions(c *gin.Context, odigosns string) {

	response := []IcaInstanceResponse{}

	icaActions, err := kube.DefaultClient.ActionsClient.AddClusterInfos(odigosns).List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	for _, action := range icaActions.Items {
		response = append(response, IcaInstanceResponse{
			Id:   action.Name,
			Type: action.Kind,
			Spec: action.Spec,
		})
	}

	daActions, err := kube.DefaultClient.ActionsClient.DeleteAttributes(odigosns).List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	for _, action := range daActions.Items {
		response = append(response, IcaInstanceResponse{
			Id:   action.Name,
			Type: action.Kind,
			Spec: action.Spec,
		})
	}

	raActions, err := kube.DefaultClient.ActionsClient.RenameAttributes(odigosns).List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	for _, action := range raActions.Items {
		response = append(response, IcaInstanceResponse{
			Id:   action.Name,
			Type: action.Kind,
			Spec: action.Spec,
		})
	}

	esActions, err := kube.DefaultClient.ActionsClient.ErrorSamplers(odigosns).List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	for _, action := range esActions.Items {
		response = append(response, IcaInstanceResponse{
			Id:   action.Name,
			Type: action.Kind,
			Spec: action.Spec,
		})
	}

	lsActions, err := kube.DefaultClient.ActionsClient.LatencySamplers(odigosns).List(c, metav1.ListOptions{})

	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	for _, action := range lsActions.Items {
		response = append(response, IcaInstanceResponse{
			Id:   action.Name,
			Type: action.Kind,
			Spec: action.Spec,
		})
	}

	psActions, err := kube.DefaultClient.ActionsClient.ProbabilisticSamplers(odigosns).List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	for _, action := range psActions.Items {
		response = append(response, IcaInstanceResponse{
			Id:   action.Name,
			Type: action.Kind,
			Spec: action.Spec,
		})
	}

	piActions, err := kube.DefaultClient.ActionsClient.PiiMaskings(odigosns).List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	for _, action := range piActions.Items {
		response = append(response, IcaInstanceResponse{
			Id:   action.Name,
			Type: action.Kind,
			Spec: action.Spec,
		})
	}

	c.JSON(200, response)
}
