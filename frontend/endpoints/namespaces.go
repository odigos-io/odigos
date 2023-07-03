package endpoints

import (
	"log"
	"net/http"

	"github.com/keyval-dev/odigos/common/consts"

	"github.com/keyval-dev/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"
)

type GetNamespacesResponse struct {
	Namespaces []GetNamespaceItem `json:"namespaces"`
}

type GetNamespaceItem struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

func GetNamespaces(c *gin.Context) {
	log.Println("GetNamespaces")
	list, err := kube.DefaultClient.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})

	if err != nil {
		log.Println(err)

		returnError(c, err)
		return
	}

	var response GetNamespacesResponse
	for _, namespace := range list.Items {
		selected := false
		if val, exists := namespace.Labels[consts.OdigosInstrumentationLabel]; exists {
			if val == consts.InstrumentationEnabled {
				selected = true
			}
		}

		response.Namespaces = append(response.Namespaces, GetNamespaceItem{
			Name:     namespace.Name,
			Selected: selected,
		})
	}

	c.JSON(http.StatusOK, response)
}
