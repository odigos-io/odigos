package endpoints

import (
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/utils"
	"github.com/odigos-io/odigos/frontend/kube"
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
	ReportedName string `json:"reported_name,omitempty"`
}

type PatchSourceRequest struct {
	ReportedName *string `json:"reported_name"`
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
	kind := c.Param("kind")
	name := c.Param("name")
	k8sObjectName := utils.GetRuntimeObjectName(name, kind)

	instrumentedApplication, err := kube.DefaultClient.OdigosClient.InstrumentedApplications(ns).Get(c, k8sObjectName, metav1.GetOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	owner := getK8sObject(c, ns, kind, name)
	if owner == nil {
		c.JSON(500, gin.H{
			"message": "could not find owner of instrumented application",
		})
		return
	}
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
	kind := c.Param("kind")
	name := c.Param("name")

	request := PatchSourceRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		returnError(c, err)
		return
	}

	if request.ReportedName != nil {

		newReportedName := *request.ReportedName

		switch kind {
		case "Deployment":
			deployment, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(c, name, metav1.GetOptions{})
			if err != nil {
				c.JSON(404, gin.H{"error": "could not find a deployment with the given name in the given namespace"})
				return
			}
			deployment.SetAnnotations(updateReportedName(deployment.GetAnnotations(), newReportedName))
			_, err = kube.DefaultClient.AppsV1().Deployments(ns).Update(c, deployment, metav1.UpdateOptions{})
			if err != nil {
				returnError(c, err)
				return
			}
		case "StatefulSet":
			statefulset, err := kube.DefaultClient.AppsV1().StatefulSets(ns).Get(c, name, metav1.GetOptions{})
			if err != nil {
				c.JSON(404, gin.H{"error": "could not find a statefulset with the given name in the given namespace"})
				return
			}
			statefulset.SetAnnotations(updateReportedName(statefulset.GetAnnotations(), newReportedName))
			_, err = kube.DefaultClient.AppsV1().StatefulSets(ns).Update(c, statefulset, metav1.UpdateOptions{})
			if err != nil {
				returnError(c, err)
				return
			}
		case "DaemonSet":
			daemonset, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(c, name, metav1.GetOptions{})
			if err != nil {
				c.JSON(404, gin.H{"error": "could not find a daemonset with the given name in the given namespace"})
				return
			}
			daemonset.SetAnnotations(updateReportedName(daemonset.GetAnnotations(), newReportedName))
			_, err = kube.DefaultClient.AppsV1().DaemonSets(ns).Update(c, daemonset, metav1.UpdateOptions{})
			if err != nil {
				returnError(c, err)
				return
			}
		default:
			c.JSON(400, gin.H{"error": "kind must be one of Deployment, StatefulSet or DaemonSet"})
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

	var kindAsEnum WorkloadKind
	switch kind {
	case "Deployment":
		kindAsEnum = WorkloadKindDeployment
	case "StatefulSet":
		kindAsEnum = WorkloadKindStatefulSet
	case "DaemonSet":
		kindAsEnum = WorkloadKindDaemonSet
	default:
		c.JSON(400, gin.H{"error": "kind must be one of Deployment, StatefulSet or DaemonSet"})
		return
	}

	instrumented := false
	err := setWorkloadInstrumentationLabel(c, ns, name, kindAsEnum, &instrumented)
	if err != nil {
		returnError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "ok"})
}

func k8sInstrumentedAppToThinSource(app *v1alpha1.InstrumentedApplication) ThinSource {
	var source ThinSource
	source.Name = app.OwnerReferences[0].Name
	source.Kind = app.OwnerReferences[0].Kind
	source.Namespace = app.Namespace
	for _, language := range app.Spec.RuntimeDetails {
		source.Languages = append(source.Languages, SourceLanguage{
			ContainerName: language.ContainerName,
			Language:      string(language.Language),
		})
	}
	return source
}

func getK8sObject(c *gin.Context, ns string, kind string, name string) metav1.Object {
	switch kind {
	case "Deployment":
		deployment, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return deployment
	case "StatefulSet":
		statefulSet, err := kube.DefaultClient.AppsV1().StatefulSets(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return statefulSet
	case "DaemonSet":
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
	if reportedName == "" {
		// delete the reported name if it is empty, so we pick up the name from the k8s object
		if annotations == nil {
			return nil
		} else {
			delete(annotations, consts.OdigosReportedNameAnnotation)
			return annotations
		}
	} else {
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[consts.OdigosReportedNameAnnotation] = reportedName
		return annotations
	}
}
