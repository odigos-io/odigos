package endpoints

import (
	"github.com/gin-gonic/gin"
	"github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SourceLanguage struct {
	ContainerName string `json:"container_name"`
	Language      string `json:"language"`
}

type Source struct {
	Name      string           `json:"name"`
	Kind      string           `json:"kind"`
	Namespace string           `json:"namespace"`
	Languages []SourceLanguage `json:"languages"`
}

func GetSources(c *gin.Context) {
	instrumentedApplications, err := kube.DefaultClient.OdigosClient.InstrumentedApplications("").List(c, metav1.ListOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	sources := []Source{}
	for _, app := range instrumentedApplications.Items {
		sources = append(sources, k8sInstrumentedAppToSource(&app))
	}

	c.JSON(200, sources)
}

func k8sInstrumentedAppToSource(app *v1alpha1.InstrumentedApplication) Source {
	var source Source
	source.Name = app.OwnerReferences[0].Name
	source.Kind = app.OwnerReferences[0].Kind
	source.Namespace = app.Namespace
	for _, language := range app.Spec.Languages {
		source.Languages = append(source.Languages, SourceLanguage{
			ContainerName: language.ContainerName,
			Language:      string(language.Language),
		})
	}
	return source
}
