package actions

import (
	"github.com/gin-gonic/gin"
	"github.com/keyval-dev/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IcaInstanceResponse struct {
	Id   string      `json:"id"`
	Type string      `json:"type"`
	Spec interface{} `json:"spec"`
}

func GetActions(c *gin.Context, odigosns string) {

	icaActions, err := kube.DefaultClient.ActionsClient.InsertClusterAttributes(odigosns).List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	response := []IcaInstanceResponse{}
	for _, action := range icaActions.Items {
		response = append(response, IcaInstanceResponse{
			Id:   action.Name,
			Type: action.Kind,
			Spec: action.Spec,
		})
	}

	c.JSON(200, response)
}
