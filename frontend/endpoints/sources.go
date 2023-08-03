package endpoints

import (
	"github.com/gin-gonic/gin"
	"github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common/consts"
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

func DeleteSource(c *gin.Context) {
	// to delete a source, we need to set it's instrumentation label to disable
	// afterwards, odigos will detect the change and remove the instrumented application object from k8s

	ns := c.Param("namespace")
	kind := c.Param("kind")
	name := c.Param("name")

	switch kind {
	case "Deployment":
		deployment, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		markK8sObjectInstrumentationDisabled(deployment)
		_, err = kube.DefaultClient.AppsV1().Deployments(ns).Update(c, deployment, metav1.UpdateOptions{})
		if err != nil {
			returnError(c, err)
			return
		}
	case "StatefulSet":
		statefulSet, err := kube.DefaultClient.AppsV1().StatefulSets(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		markK8sObjectInstrumentationDisabled(statefulSet)
		_, err = kube.DefaultClient.AppsV1().StatefulSets(ns).Update(c, statefulSet, metav1.UpdateOptions{})
		if err != nil {
			returnError(c, err)
			return
		}
	case "DaemonSet":
		daemonSet, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		markK8sObjectInstrumentationDisabled(daemonSet)
		_, err = kube.DefaultClient.AppsV1().DaemonSets(ns).Update(c, daemonSet, metav1.UpdateOptions{})
		if err != nil {
			returnError(c, err)
			return
		}
	default:
		c.JSON(400, gin.H{"error": "kind not supported"})
		return
	}
	c.JSON(200, gin.H{"message": "ok"})
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

func markK8sObjectInstrumentationDisabled(obj metav1.Object) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[consts.OdigosInstrumentationLabel] = consts.InstrumentationDisabled
	obj.SetLabels(labels)
}
