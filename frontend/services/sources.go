package services

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
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
type SourceID struct {
	// combination of namespace, kind and name is unique
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
}

type Source struct {
	ThinSource
	ReportedName string `json:"reported_name,omitempty"`
}

type PatchSourceRequest struct {
	ReportedName *string `json:"reported_name"`
}

// this object contains only part of the source fields. It is used to display the sources in the frontend
type ThinSource struct {
	SourceID
	NumberOfRunningInstances int                             `json:"number_of_running_instances"`
	IaDetails                *InstrumentedApplicationDetails `json:"instrumented_application_details"`
}

func GetActualSources(ctx context.Context, odigosns string) []ThinSource {
	return getSourcesForNamespace(ctx, odigosns)
}

func GetNamespaceActualSources(ctx context.Context, namespace string) []ThinSource {
	return getSourcesForNamespace(ctx, namespace)
}

func getSourcesForNamespace(ctx context.Context, namespace string) []ThinSource {
	effectiveInstrumentedSources := map[SourceID]ThinSource{}

	var (
		items                    []GetApplicationItem
		instrumentedApplications *v1alpha1.InstrumentedApplicationList
	)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		relevantNamespaces, err := getRelevantNameSpaces(ctx, namespace)
		if err != nil {
			return err
		}
		nsInstrumentedMap := map[string]*bool{}
		for _, ns := range relevantNamespaces {
			nsInstrumentedMap[ns.Name] = isObjectLabeledForInstrumentation(ns.ObjectMeta)
		}
		items, err = getApplicationsInNamespace(ctx, "", nsInstrumentedMap)
		return err
	})

	g.Go(func() error {
		var err error
		instrumentedApplications, err = kube.DefaultClient.OdigosClient.InstrumentedApplications("").List(ctx, metav1.ListOptions{})
		return err
	})

	if err := g.Wait(); err != nil {
		return nil
	}

	for _, item := range items {
		if item.nsItem.InstrumentationEffective {
			id := SourceID{Namespace: item.namespace, Kind: string(item.nsItem.Kind), Name: item.nsItem.Name}
			effectiveInstrumentedSources[id] = ThinSource{
				NumberOfRunningInstances: item.nsItem.Instances,
				SourceID:                 id,
			}
		}
	}

	sourcesResult := []ThinSource{}
	for _, app := range instrumentedApplications.Items {
		thinSource := k8sInstrumentedAppToThinSource(&app)
		if source, ok := effectiveInstrumentedSources[thinSource.SourceID]; ok {
			source.IaDetails = thinSource.IaDetails
			effectiveInstrumentedSources[thinSource.SourceID] = source
		}
	}

	for _, source := range effectiveInstrumentedSources {
		sourcesResult = append(sourcesResult, source)
	}

	return sourcesResult
}

func GetActualSource(ctx context.Context, ns string, kind string, name string) (*Source, error) {
	k8sObjectName := workload.GetRuntimeObjectName(name, kind)
	owner, numberOfRunningInstances := getWorkload(ctx, ns, kind, name)
	if owner == nil {
		return nil, fmt.Errorf("owner not found")
	}
	ownerAnnotations := owner.GetAnnotations()
	var reportedName string
	if ownerAnnotations != nil {
		reportedName = ownerAnnotations[consts.OdigosReportedNameAnnotation]
	}

	ts := ThinSource{
		SourceID: SourceID{
			Namespace: ns,
			Kind:      kind,
			Name:      name,
		},
		NumberOfRunningInstances: numberOfRunningInstances,
	}

	instrumentedApplication, err := kube.DefaultClient.OdigosClient.InstrumentedApplications(ns).Get(ctx, k8sObjectName, metav1.GetOptions{})
	if err == nil {
		ts.IaDetails = k8sInstrumentedAppToThinSource(instrumentedApplication).IaDetails
		err = addHealthyInstrumentationInstancesCondition(ctx, instrumentedApplication, &ts)
		if err != nil {
			return nil, err
		}
	}

	return &Source{
		ThinSource:   ts,
		ReportedName: reportedName,
	}, nil
}

func getWorkload(c context.Context, ns string, kind string, name string) (metav1.Object, int) {
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
