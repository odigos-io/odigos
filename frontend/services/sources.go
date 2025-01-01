package services

import (
	"context"
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

	"golang.org/x/sync/errgroup"
)

type WorkloadKind string

const (
	WorkloadKindDeployment  WorkloadKind = "Deployment"
	WorkloadKindStatefulSet WorkloadKind = "StatefulSet"
	WorkloadKindDaemonSet   WorkloadKind = "DaemonSet"
)

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

func AddHealthyInstrumentationInstancesCondition(ctx context.Context, instruConfig *v1alpha1.InstrumentationConfig, source *model.K8sActualSource) error {
	labelSelector := fmt.Sprintf("%s=%s", consts.InstrumentedAppNameLabel, instruConfig.Name)
	instancesList, err := kube.DefaultClient.OdigosClient.InstrumentationInstances(instruConfig.Namespace).List(ctx, metav1.ListOptions{
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
	source.Conditions = append(source.Conditions, &model.Condition{
		Type:               "HealthyInstrumentationInstances",
		Status:             status,
		LastTransitionTime: &lastTransitionTime,
		Message:            &message,
	})

	return nil
}

func GetWorkloadsInNamespace(ctx context.Context, nsName string) ([]model.K8sActualSource, error) {
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
		deps, err = getDeployments(ctx, *namespace)
		return err
	})

	g.Go(func() error {
		var err error
		ss, err = getStatefulSets(ctx, *namespace)
		return err
	})

	g.Go(func() error {
		var err error
		dss, err = getDaemonSets(ctx, *namespace)
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

func getDeployments(ctx context.Context, namespace corev1.Namespace) ([]model.K8sActualSource, error) {
	var response []model.K8sActualSource
	err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.AppsV1().Deployments(namespace.Name).List, ctx, metav1.ListOptions{}, func(deps *appsv1.DeploymentList) error {
		for _, dep := range deps.Items {
			numberOfInstances := int(dep.Status.ReadyReplicas)
			response = append(response, model.K8sActualSource{
				Namespace:         dep.Namespace,
				Name:              dep.Name,
				Kind:              k8sKindToGql(string(WorkloadKindDeployment)),
				NumberOfInstances: &numberOfInstances,
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

func getDaemonSets(ctx context.Context, namespace corev1.Namespace) ([]model.K8sActualSource, error) {
	var response []model.K8sActualSource
	err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.AppsV1().DaemonSets(namespace.Name).List, ctx, metav1.ListOptions{}, func(dss *appsv1.DaemonSetList) error {
		for _, ds := range dss.Items {
			numberOfInstances := int(ds.Status.NumberReady)
			response = append(response, model.K8sActualSource{
				Namespace:         ds.Namespace,
				Name:              ds.Name,
				Kind:              k8sKindToGql(string(WorkloadKindDaemonSet)),
				NumberOfInstances: &numberOfInstances,
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

func getStatefulSets(ctx context.Context, namespace corev1.Namespace) ([]model.K8sActualSource, error) {
	var response []model.K8sActualSource
	err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.AppsV1().StatefulSets(namespace.Name).List, ctx, metav1.ListOptions{}, func(sss *appsv1.StatefulSetList) error {
		for _, ss := range sss.Items {
			numberOfInstances := int(ss.Status.ReadyReplicas)
			response = append(response, model.K8sActualSource{
				Namespace:         ss.Namespace,
				Name:              ss.Name,
				Kind:              k8sKindToGql(string(WorkloadKindStatefulSet)),
				NumberOfInstances: &numberOfInstances,
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

func GetSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind) (*v1alpha1.Source, error) {
	source, err := kube.DefaultClient.OdigosClient.Sources(nsName).List(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{
		consts.OdigosNamespaceAnnotation:    nsName,
		consts.OdigosWorkloadNameAnnotation: workloadName,
		consts.OdigosWorkloadKindAnnotation: string(workloadKind),
	}).String()})

	if err != nil {
		return nil, err
	}
	if len(source.Items) == 0 {
		return nil, fmt.Errorf(`source "%s" not found`, workloadName)
	}
	if len(source.Items) > 1 {
		return nil, fmt.Errorf(`expected to get 1 source "%s", got %d`, workloadName, len(source.Items))
	}

	crdName := source.Items[0].Name
	crd, err := kube.DefaultClient.OdigosClient.Sources(nsName).Get(ctx, crdName, metav1.GetOptions{})

	return crd, err
}

func createSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind) error {
	err := CheckWorkloadKind(workloadKind)
	if err != nil {
		return err
	}

	source, err := GetSourceCRD(ctx, nsName, workloadName, workloadKind)
	if source != nil && err == nil {
		return err
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

	_, err = kube.DefaultClient.OdigosClient.Sources(nsName).Create(ctx, newSource, metav1.CreateOptions{})
	return err
}

func deleteSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind) error {
	err := CheckWorkloadKind(workloadKind)
	if err != nil {
		return err
	}

	source, err := GetSourceCRD(ctx, nsName, workloadName, workloadKind)
	if err != nil {
		return err
	}

	err = kube.DefaultClient.OdigosClient.Sources(nsName).Delete(ctx, source.Name, metav1.DeleteOptions{})
	return err
}

func ToggleSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind, enabled *bool) error {
	if enabled == nil {
		return fmt.Errorf("enabled must be provided")
	}

	if *enabled {
		return createSourceCRD(ctx, nsName, workloadName, workloadKind)
	} else {
		return deleteSourceCRD(ctx, nsName, workloadName, workloadKind)
	}
}
