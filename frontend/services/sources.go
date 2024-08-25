package services

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/odigos-io/odigos/frontend/graph/model"

	"github.com/odigos-io/odigos/k8sutils/pkg/client"

	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
)

type WorkloadKind string

const (
	WorkloadKindDeployment  WorkloadKind = "Deployment"
	WorkloadKindStatefulSet WorkloadKind = "StatefulSet"
	WorkloadKindDaemonSet   WorkloadKind = "DaemonSet"
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

func GetActualSource(ctx context.Context, ns string, kind string, name string) (*Source, error) {
	k8sObjectName := workload.CalculateWorkloadRuntimeObjectName(name, kind)
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

func GetWorkloadsInNamespace(ctx context.Context, nsName string, instrumentationLabeled *bool) ([]model.K8sActualSource, error) {

	namespace, err := kube.DefaultClient.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	g, ctx := errgroup.WithContext(ctx)
	var (
		deps []model.K8sActualSource
		ss   []model.K8sActualSource
		dss  []model.K8sActualSource
	)

	g.Go(func() error {
		var err error
		deps, err = getDeployments(ctx, *namespace, instrumentationLabeled)
		return err
	})

	g.Go(func() error {
		var err error
		ss, err = getStatefulSets(ctx, *namespace, instrumentationLabeled)
		return err
	})

	g.Go(func() error {
		var err error
		dss, err = getDaemonSets(ctx, *namespace, instrumentationLabeled)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	items := make([]model.K8sActualSource, len(deps)+len(ss)+len(dss))
	copy(items, deps)
	copy(items[len(deps):], ss)
	copy(items[len(deps)+len(ss):], dss)

	return items, nil
}

func getDeployments(ctx context.Context, namespace corev1.Namespace, instrumentationLabeled *bool) ([]model.K8sActualSource, error) {
	var response []model.K8sActualSource
	err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.AppsV1().Deployments(namespace.Name).List, ctx, metav1.ListOptions{}, func(deps *appsv1.DeploymentList) error {
		for _, dep := range deps.Items {
			_, _, decisionText, autoInstrumented := workload.GetInstrumentationLabelTexts(dep.GetLabels(), string(WorkloadKindDeployment), namespace.GetLabels())
			if instrumentationLabeled != nil && *instrumentationLabeled != autoInstrumented {
				continue
			}
			numberOfInstances := int(dep.Status.ReadyReplicas)
			response = append(response, model.K8sActualSource{
				Namespace:                      dep.Namespace,
				Name:                           dep.Name,
				Kind:                           k8sKindToGql(string(WorkloadKindDeployment)),
				NumberOfInstances:              &numberOfInstances,
				AutoInstrumented:               autoInstrumented,
				AutoInstrumentedDecision:       decisionText,
				InstrumentedApplicationDetails: nil, // TODO: fill this
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

func getDaemonSets(ctx context.Context, namespace corev1.Namespace, instrumentationLabeled *bool) ([]model.K8sActualSource, error) {
	var response []model.K8sActualSource
	err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.AppsV1().DaemonSets(namespace.Name).List, ctx, metav1.ListOptions{}, func(dss *appsv1.DaemonSetList) error {
		for _, ds := range dss.Items {
			_, _, decisionText, autoInstrumented := workload.GetInstrumentationLabelTexts(ds.GetLabels(), string(WorkloadKindDaemonSet), namespace.GetLabels())
			if instrumentationLabeled != nil && *instrumentationLabeled != autoInstrumented {
				continue
			}
			numberOfInstances := int(ds.Status.NumberReady)
			response = append(response, model.K8sActualSource{
				Namespace:                      ds.Namespace,
				Name:                           ds.Name,
				Kind:                           k8sKindToGql(string(WorkloadKindDaemonSet)),
				NumberOfInstances:              &numberOfInstances,
				AutoInstrumented:               autoInstrumented,
				AutoInstrumentedDecision:       decisionText,
				InstrumentedApplicationDetails: nil, // TODO: fill this
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

func getStatefulSets(ctx context.Context, namespace corev1.Namespace, instrumentationLabeled *bool) ([]model.K8sActualSource, error) {
	var response []model.K8sActualSource
	err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.AppsV1().StatefulSets(namespace.Name).List, ctx, metav1.ListOptions{}, func(sss *appsv1.StatefulSetList) error {
		for _, ss := range sss.Items {
			_, _, decisionText, autoInstrumented := workload.GetInstrumentationLabelTexts(ss.GetLabels(), string(WorkloadKindStatefulSet), namespace.GetLabels())
			if instrumentationLabeled != nil && *instrumentationLabeled != autoInstrumented {
				continue
			}
			numberOfInstances := int(ss.Status.ReadyReplicas)
			response = append(response, model.K8sActualSource{
				Namespace:                      ss.Namespace,
				Name:                           ss.Name,
				Kind:                           k8sKindToGql(string(WorkloadKindStatefulSet)),
				NumberOfInstances:              &numberOfInstances,
				AutoInstrumented:               autoInstrumented,
				AutoInstrumentedDecision:       decisionText,
				InstrumentedApplicationDetails: nil, // TODO: fill this
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

func k8sKindToGql(k8sResourceKind string) model.K8sResourceKind {
	switch k8sResourceKind {
	case "Deployment":
		return model.K8sResourceKindDeployment
	case "StatefulSet":
		return model.K8sResourceKindStatefulSet
	case "DaemonSet":
		return model.K8sResourceKindDaemonSet
	}
	return ""
}
