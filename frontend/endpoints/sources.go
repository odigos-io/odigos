package endpoints

import (
	"fmt"
	"strings"

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

// this object contains only part of the source fields. It is used to display the sources in the frontend
type ThinSource struct {

	// combination of namespace, kind and name is unique
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`

	Languages []SourceLanguage `json:"languages"`
}

type Source struct {
	ThinSource
	ReportedName string `json:"reported_name"`
}

type PatchSourceRequest struct {
	ReportedName string `json:"reported_name"`
}

func GetSources(c *gin.Context) {
	instrumentedApplications, err := kube.DefaultClient.OdigosClient.InstrumentedApplications("").List(c, metav1.ListOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	sources := []ThinSource{}
	for _, app := range instrumentedApplications.Items {
		sources = append(sources, k8sInstrumentedAppToThinSource(&app))
	}

	c.JSON(200, sources)
}

func GetSource(c *gin.Context) {
	ns := c.Param("namespace")
	kind := strings.ToLower(c.Param("kind"))
	name := c.Param("name")
	k8sObjectName := fmt.Sprintf("%s-%s", kind, name)

	instrumentedApplication, err := kube.DefaultClient.OdigosClient.InstrumentedApplications(ns).Get(c, k8sObjectName, metav1.GetOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	owner := getK8sObject(c, ns, kind, name)
	ownerAnnotations := owner.GetAnnotations()
	var reportedName string
	if ownerAnnotations != nil {
		reportedName = ownerAnnotations[consts.OdigosReportedNameAnnotation]
	}

	c.JSON(200, Source{
		ThinSource:   k8sInstrumentedAppToThinSource(instrumentedApplication),
		ReportedName: reportedName,
	})
}

func PatchSource(c *gin.Context) {
	ns := c.Param("namespace")
	kind := strings.ToLower(c.Param("kind"))
	name := c.Param("name")

	request := PatchSourceRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		returnError(c, err)
		return
	}

	if request.ReportedName != "" {

		switch kind {
		case "deployment":
			deployment, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(c, name, metav1.GetOptions{})
			if err != nil {
				c.JSON(404, gin.H{"error": "could not find a deployment with the given name in the given namespace"})
				return
			}
			deployment.SetAnnotations(updateReportedName(deployment.GetAnnotations(), request.ReportedName))
			_, err = kube.DefaultClient.AppsV1().Deployments(ns).Update(c, deployment, metav1.UpdateOptions{})
			if err != nil {
				returnError(c, err)
				return
			}
		case "statefulset":
			statefulset, err := kube.DefaultClient.AppsV1().StatefulSets(ns).Get(c, name, metav1.GetOptions{})
			if err != nil {
				c.JSON(404, gin.H{"error": "could not find a statefulset with the given name in the given namespace"})
				return
			}
			statefulset.SetAnnotations(updateReportedName(statefulset.GetAnnotations(), request.ReportedName))
			_, err = kube.DefaultClient.AppsV1().StatefulSets(ns).Update(c, statefulset, metav1.UpdateOptions{})
			if err != nil {
				returnError(c, err)
				return
			}
		case "daemonset":
			daemonset, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(c, name, metav1.GetOptions{})
			if err != nil {
				c.JSON(404, gin.H{"error": "could not find a daemonset with the given name in the given namespace"})
				return
			}
			daemonset.SetAnnotations(updateReportedName(daemonset.GetAnnotations(), request.ReportedName))
			_, err = kube.DefaultClient.AppsV1().DaemonSets(ns).Update(c, daemonset, metav1.UpdateOptions{})
			if err != nil {
				returnError(c, err)
				return
			}
		default:
			c.JSON(400, gin.H{"error": "kind must be one of deployment, statefulset or daemonset"})
			return
		}
	}

	c.Status(200)
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
		deployment.SetAnnotations(deleteReportedNameAnnotation(deployment.GetAnnotations()))
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
		statefulSet.SetAnnotations(deleteReportedNameAnnotation(statefulSet.GetAnnotations()))
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
		daemonSet.SetAnnotations(deleteReportedNameAnnotation(daemonSet.GetAnnotations()))
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

func k8sInstrumentedAppToThinSource(app *v1alpha1.InstrumentedApplication) ThinSource {
	var source ThinSource
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

func deleteReportedNameAnnotation(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}
	delete(annotations, consts.OdigosReportedNameAnnotation)
	return annotations
}

func getK8sObject(c *gin.Context, ns string, kind string, name string) metav1.Object {
	switch kind {
	case "deployment":
		deployment, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return deployment
	case "statefulset":
		statefulSet, err := kube.DefaultClient.AppsV1().StatefulSets(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return statefulSet
	case "daemonset":
		daemonSet, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return daemonSet
	default:
		return nil
	}
}

func updateReportedName(annotations map[string]string, reportedName string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[consts.OdigosReportedNameAnnotation] = reportedName
	return annotations
}
