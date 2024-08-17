package endpoints

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/endpoints/common"
	"github.com/odigos-io/odigos/frontend/kube"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SourceLanguage struct {
	ContainerName string `json:"container_name"`
	Language      string `json:"language"`
}

type InstrumentedApplicationDetails struct {
	Languages  []SourceLanguage   `json:"languages,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// this object contains only part of the source fields. It is used to display the sources in the frontend
type ThinSource struct {
	common.SourceID
	NumberOfRunningInstances int                             `json:"number_of_running_instances"`
	IaDetails                *InstrumentedApplicationDetails `json:"instrumented_application_details"`
}

type Source struct {
	ThinSource
	ReportedName string `json:"reported_name,omitempty"`
}

type PatchSourceRequest struct {
	ReportedName *string `json:"reported_name"`
}

func GetSources(c *gin.Context, odigosns string) {
	reqCtx := c.Request.Context()
	effectiveInstrumentedSources := map[common.SourceID]ThinSource{}

	var (
		items                    []GetApplicationItem
		instrumentedApplications *v1alpha1.InstrumentedApplicationList
	)

	g, errCtx := errgroup.WithContext(reqCtx)
	g.Go(func() error {
		relevantNamespaces, err := getRelevantNameSpaces(errCtx, odigosns)
		if err != nil {
			return err
		}
		nsInstrumentedMap := map[string]*bool{}
		for _, ns := range relevantNamespaces {
			nsInstrumentedMap[ns.Name] = isObjectLabeledForInstrumentation(ns.ObjectMeta)
		}
		// get all the applications in all the namespaces,
		// passing an empty string here is more efficient compared to iterating over the namespaces
		// since it will make a single request per workload type to the k8s api server
		items, err = getApplicationsInNamespace(errCtx, "", nsInstrumentedMap)
		return err
	})

	g.Go(func() error {
		var err error
		instrumentedApplications, err = kube.DefaultClient.OdigosClient.InstrumentedApplications("").List(errCtx, metav1.ListOptions{})
		return err
	})

	if err := g.Wait(); err != nil {
		returnError(c, err)
		return
	}

	for _, item := range items {
		if item.nsItem.InstrumentationEffective {
			id := common.SourceID{Namespace: item.namespace, Kind: string(item.nsItem.Kind), Name: item.nsItem.Name}
			effectiveInstrumentedSources[id] = ThinSource{
				NumberOfRunningInstances: item.nsItem.Instances,
				SourceID:                 id,
			}
		}
	}

	sourcesResult := []ThinSource{}
	// go over the instrumented applications and update the languages of the effective sources.
	// Not all effective sources necessarily have a corresponding instrumented application,
	// it may take some time for the instrumented application to be created. In that case the languages
	// slice will be empty.
	for _, app := range instrumentedApplications.Items {
		thinSource := k8sInstrumentedAppToThinSource(&app)
		if source, ok := effectiveInstrumentedSources[thinSource.SourceID]; ok {
			source.IaDetails = thinSource.IaDetails
			err := addHealthyInstrumentationInstancesCondition(reqCtx, &app, &source)
			if err != nil {
				returnError(c, err)
				return
			}
			effectiveInstrumentedSources[thinSource.SourceID] = source
		}
	}

	for _, source := range effectiveInstrumentedSources {
		sourcesResult = append(sourcesResult, source)
	}

	c.JSON(200, sourcesResult)
}

func GetSource(c *gin.Context) {
	ns := c.Param("namespace")
	kind := c.Param("kind")
	name := c.Param("name")
	k8sObjectName := workload.CalculateWorkloadRuntimeObjectName(name, kind)

	owner, numberOfRunningInstances := getWorkloadObject(c, ns, kind, name)
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

	ts := ThinSource{
		SourceID: common.SourceID{
			Namespace: ns,
			Kind:      kind,
			Name:      name,
		},
		NumberOfRunningInstances: numberOfRunningInstances,
	}

	instrumentedApplication, err := kube.DefaultClient.OdigosClient.InstrumentedApplications(ns).Get(c, k8sObjectName, metav1.GetOptions{})
	if err == nil {
		// valid instrumented application, grab the runtime details
		ts.IaDetails = k8sInstrumentedAppToThinSource(instrumentedApplication).IaDetails
		// potentially add a condition for healthy instrumentation instances
		err = addHealthyInstrumentationInstancesCondition(c, instrumentedApplication, &ts)
		if err != nil {
			returnError(c, err)
			return
		}
	}

	c.JSON(200, Source{
		ThinSource:   ts,
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
	var conditions []metav1.Condition
	for _, condition := range app.Status.Conditions {
		conditions = append(conditions, metav1.Condition{
			Type:               condition.Type,
			Status:             condition.Status,
			Message:            condition.Message,
			LastTransitionTime: condition.LastTransitionTime,
		})
	}
	source.IaDetails = &InstrumentedApplicationDetails{
		Languages:  []SourceLanguage{},
		Conditions: conditions,
	}
	for _, language := range app.Spec.RuntimeDetails {
		source.IaDetails.Languages = append(source.IaDetails.Languages, SourceLanguage{
			ContainerName: language.ContainerName,
			Language:      string(language.Language),
		})
	}
	return source
}

func addHealthyInstrumentationInstancesCondition(ctx context.Context, app *v1alpha1.InstrumentedApplication, source *ThinSource) error {
	labelSelector := fmt.Sprintf("%s=%s", consts.InstrumentedAppNameLabel, app.Name)
	instancesList, err := kube.DefaultClient.OdigosClient.InstrumentationInstances(app.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		return err
	}

	totalInstances := len(instancesList.Items)
	if totalInstances == 0 {
		// no instances so nothing to report
		return nil
	}

	healthyInstances := 0
	latestStatusTime := metav1.NewTime(time.Time{})
	for _, instance := range instancesList.Items {
		if instance.Status.Healthy != nil && *instance.Status.Healthy {
			healthyInstances++
		}
		if instance.Status.LastStatusTime.After(latestStatusTime.Time) {
			latestStatusTime = instance.Status.LastStatusTime
		}
	}

	status := metav1.ConditionTrue
	if healthyInstances < totalInstances {
		status = metav1.ConditionFalse
	}

	source.IaDetails.Conditions = append(source.IaDetails.Conditions, metav1.Condition{
		Type:               "HealthyInstrumentationInstances",
		Status:             status,
		LastTransitionTime: latestStatusTime,
		Message:            fmt.Sprintf("%d/%d instances are healthy", healthyInstances, totalInstances),
	})

	return nil
}

func getWorkloadObject(c *gin.Context, ns string, kind string, name string) (metav1.Object, int) {
	switch kind {
	case "Deployment":
		deployment, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, 0
		}
		return deployment, int(deployment.Status.AvailableReplicas)
	case "StatefulSet":
		statefulSet, err := kube.DefaultClient.AppsV1().StatefulSets(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, 0
		}
		return statefulSet, int(statefulSet.Status.ReadyReplicas)
	case "DaemonSet":
		daemonSet, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, 0
		}
		return daemonSet, int(daemonSet.Status.NumberReady)
	default:
		return nil, 0
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
