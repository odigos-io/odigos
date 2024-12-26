package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"golang.org/x/sync/errgroup"
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

func GetWorkload(c context.Context, ns string, kind string, name string) (metav1.Object, int) {
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

func AddHealthyInstrumentationInstancesCondition(ctx context.Context, app *v1alpha1.InstrumentedApplication, source *model.K8sActualSource) error {
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

	status := model.ConditionStatusTrue
	if healthyInstances < totalInstances {
		status = model.ConditionStatusFalse
	}

	message := fmt.Sprintf("%d/%d instances are healthy", healthyInstances, totalInstances)
	lastTransitionTime := Metav1TimeToString(latestStatusTime)
	source.InstrumentedApplicationDetails.Conditions = append(source.InstrumentedApplicationDetails.Conditions, &model.Condition{
		Type:               "HealthyInstrumentationInstances",
		Status:             status,
		LastTransitionTime: &lastTransitionTime,
		Message:            &message,
	})

	return nil
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

func UpdateReportedName(
	ctx context.Context,
	ns, kind, name, reportedName string,
) error {
	switch kind {
	case "Deployment":
		deployment, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("could not find deployment: %w", err)
		}
		deployment.SetAnnotations(updateAnnotations(deployment.GetAnnotations(), reportedName))
		_, err = kube.DefaultClient.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	case "StatefulSet":
		statefulSet, err := kube.DefaultClient.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("could not find statefulset: %w", err)
		}
		statefulSet.SetAnnotations(updateAnnotations(statefulSet.GetAnnotations(), reportedName))
		_, err = kube.DefaultClient.AppsV1().StatefulSets(ns).Update(ctx, statefulSet, metav1.UpdateOptions{})
		return err
	case "DaemonSet":
		daemonSet, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("could not find daemonset: %w", err)
		}
		daemonSet.SetAnnotations(updateAnnotations(daemonSet.GetAnnotations(), reportedName))
		_, err = kube.DefaultClient.AppsV1().DaemonSets(ns).Update(ctx, daemonSet, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("unsupported kind: %s", kind)
	}
}

func updateAnnotations(annotations map[string]string, reportedName string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	if reportedName == "" {
		delete(annotations, consts.OdigosReportedNameAnnotation)
	} else {
		annotations[consts.OdigosReportedNameAnnotation] = reportedName
	}
	return annotations
}

func GetSourceCRDs(ctx context.Context, args ...interface{}) ([]*v1alpha1.Source, error) {
	var nsName, workloadName string
	var workloadKind WorkloadKind
	if len(args) > 0 {
		nsName, _ = args[0].(string)
	}
	if len(args) > 1 {
		workloadName, _ = args[1].(string)
	}
	if len(args) > 2 {
		workloadKind, _ = args[2].(WorkloadKind)
		if workloadKind != WorkloadKindDeployment && workloadKind != WorkloadKindStatefulSet && workloadKind != WorkloadKindDaemonSet {
			return nil, errors.New("unsupported workload kind " + string(workloadKind))
		}
	}

	labelsSet := labels.Set{}
	if nsName != "" {
		labelsSet["odigos.io/workload-namespace"] = nsName
	}
	if workloadName != "" {
		labelsSet["odigos.io/workload-name"] = workloadName
	}
	if string(workloadKind) != "" {
		labelsSet["odigos.io/workload-kind"] = string(workloadKind)
	}

	sourceList, err := kube.DefaultClient.OdigosClient.Sources(consts.DefaultOdigosNamespace).List(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labelsSet).String()})
	if err != nil {
		return nil, err
	}
	if workloadName != "" && len(sourceList.Items) == 0 {
		return nil, errors.New("source not found" + workloadName)
	}

	var sources []*v1alpha1.Source

	for _, crd := range sourceList.Items {
		crdName := crd.Name
		source, err := kube.DefaultClient.OdigosClient.Sources(consts.DefaultOdigosNamespace).Get(ctx, crdName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}

	return sources, nil
}

func CreateSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind) error {
	if workloadKind != WorkloadKindDeployment && workloadKind != WorkloadKindStatefulSet && workloadKind != WorkloadKindDaemonSet {
		return errors.New("unsupported workload kind " + string(workloadKind))
	}

	newSource := &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "source-",
		},
		Spec: v1alpha1.SourceSpec{
			Workload: workload.PodWorkload{
				Namespace: nsName,
				Name:      workloadName,
				Kind:      workload.WorkloadKind(workloadKind),
			},
		},
	}

	_, err := kube.DefaultClient.OdigosClient.Sources(consts.DefaultOdigosNamespace).Create(ctx, newSource, metav1.CreateOptions{})
	return err
}

func DeleteSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind) error {
	if workloadKind != WorkloadKindDeployment && workloadKind != WorkloadKindStatefulSet && workloadKind != WorkloadKindDaemonSet {
		return errors.New("unsupported workload kind " + string(workloadKind))
	}

	sources, err := GetSourceCRDs(ctx, nsName, workloadName, workloadKind)
	if err != nil {
		return err
	}

	err = kube.DefaultClient.OdigosClient.Sources(consts.DefaultOdigosNamespace).Delete(ctx, sources[0].Name, metav1.DeleteOptions{})
	return err
}

func ToggleSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind, enabled *bool) error {
	if enabled == nil {
		return errors.New("enabled must be provided")
	}

	if *enabled {
		return CreateSourceCRD(ctx, nsName, workloadName, workloadKind)
	} else {
		return DeleteSourceCRD(ctx, nsName, workloadName, workloadKind)
	}
}

// TODO: remove this after a fix was made in the backend to correctly handle the InstrumentedApplication on-create Source CRD
func SetWorkloadInstrumentationLabel(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind, enabled *bool) error {
	jsonMergePatchData := GetJsonMergePatchForInstrumentationLabel(enabled)

	switch workloadKind {
	case WorkloadKindDeployment:
		_, err := kube.DefaultClient.AppsV1().Deployments(nsName).Patch(ctx, workloadName, types.MergePatchType, jsonMergePatchData, metav1.PatchOptions{})
		return err
	case WorkloadKindStatefulSet:
		_, err := kube.DefaultClient.AppsV1().StatefulSets(nsName).Patch(ctx, workloadName, types.MergePatchType, jsonMergePatchData, metav1.PatchOptions{})
		return err
	case WorkloadKindDaemonSet:
		_, err := kube.DefaultClient.AppsV1().DaemonSets(nsName).Patch(ctx, workloadName, types.MergePatchType, jsonMergePatchData, metav1.PatchOptions{})
		return err
	default:
		return errors.New("unsupported workload kind " + string(workloadKind))
	}
}
