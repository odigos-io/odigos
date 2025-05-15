package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"golang.org/x/sync/errgroup"
)

type WorkloadKind string

const (
	WorkloadKindNamespace   WorkloadKind = "Namespace"
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
	err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.AppsV1().Deployments(namespace.Name).List, ctx, &metav1.ListOptions{}, func(deps *appsv1.DeploymentList) error {
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
	err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.AppsV1().DaemonSets(namespace.Name).List, ctx, &metav1.ListOptions{}, func(dss *appsv1.DaemonSetList) error {
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
	err := client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.AppsV1().StatefulSets(namespace.Name).List, ctx, &metav1.ListOptions{}, func(sss *appsv1.StatefulSetList) error {
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

func GetSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind) (*v1alpha1.Source, error) {
	list, err := kube.DefaultClient.OdigosClient.Sources(nsName).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			k8sconsts.WorkloadNamespaceLabel: nsName,
			k8sconsts.WorkloadNameLabel:      workloadName,
			k8sconsts.WorkloadKindLabel:      string(workloadKind),
		}).String(),
	})

	if err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "source"}, workloadName)
	}
	if len(list.Items) > 1 {
		return nil, fmt.Errorf(`expected to get 1 source "%s", got %d`, workloadName, len(list.Items))
	}

	return &list.Items[0], err
}

func DeleteSourceWithAPI(c *gin.Context) {
	toggleSourceWithAPI(c, false)
}

func CreateSourceWithAPI(c *gin.Context) {
	toggleSourceWithAPI(c, true)
}

func toggleSourceWithAPI(c *gin.Context, enabled bool) {
	ctx := c.Request.Context()
	ns := c.Param("namespace")
	name := c.Param("name")
	kind := c.Param("kind")
	wk, ok := stringToWorkloadKind(kind)
	if !ok {
		c.JSON(400, gin.H{
			"message": fmt.Sprintf("invalid kind: %s", kind),
		})
		return
	}

	// TODO: check if we need to handle a stream name for remote API requests
	err := ToggleSourceCRD(ctx, ns, name, wk, enabled, "default")
	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"message": "toggle successful",
	})
}

func stringToWorkloadKind(workloadKind string) (WorkloadKind, bool) {
	switch strings.ToLower(workloadKind) {
	case "namespace":
		return WorkloadKindNamespace, true
	case "deployment":
		return WorkloadKindDeployment, true
	case "statefulset":
		return WorkloadKindStatefulSet, true
	case "daemonset":
		return WorkloadKindDaemonSet, true
	}

	return "", false
}

func EnsureSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind, currentStreamName string) (*v1alpha1.Source, error) {
	streamLabel := k8sconsts.SourceGroupLabelPrefix + currentStreamName

	switch workloadKind {
	// Namespace is not a workload, but we need it to "select future apps" by creating a Source CRD for it
	case WorkloadKindNamespace, WorkloadKindDeployment, WorkloadKindStatefulSet, WorkloadKindDaemonSet:
		break
	default:
		return nil, errors.New("unsupported workload kind: " + string(workloadKind))
	}

	source, err := GetSourceCRD(ctx, nsName, workloadName, workloadKind)
	if err != nil && !apierrors.IsNotFound(err) {
		// unexpected error occurred while trying to get the source
		return nil, err
	}

	if source != nil {
		// source already exists, do not create a new one, instead update so it's not disabled anymore
		source, err = UpdateSourceCRDSpec(ctx, nsName, source.Name, common.DisableInstrumentationJsonKey, false)
		if err != nil {
			return nil, err
		}
		source, err = UpdateSourceCRDLabel(ctx, nsName, source.Name, streamLabel, "true")
		if err != nil {
			return nil, err
		}
		return source, nil
	}

	newSource := &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "source-",
			Labels: map[string]string{
				streamLabel: "true",
			},
		},
		Spec: v1alpha1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Namespace: nsName,
				Name:      workloadName,
				Kind:      k8sconsts.WorkloadKind(workloadKind),
			},
		},
	}

	source, err = kube.DefaultClient.OdigosClient.Sources(nsName).Create(ctx, newSource, metav1.CreateOptions{})
	return source, err
}

func deleteSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind, currentStreamName string) error {
	source, err := GetSourceCRD(ctx, nsName, workloadName, workloadKind)
	if err != nil {
		return err
	}

	if workloadKind == WorkloadKindNamespace {
		// if is a namespace source, then proceed to delete it
		return kube.DefaultClient.OdigosClient.Sources(nsName).Delete(ctx, source.Name, metav1.DeleteOptions{})
	}

	// if is a regular workload, then check for namespace source first
	nsSource, err := GetSourceCRD(ctx, nsName, nsName, WorkloadKindNamespace)
	if err != nil && !apierrors.IsNotFound(err) {
		// unexpected error occurred while trying to get the namespace source
		return err
	}

	if nsSource != nil {
		// namespace source exists.
		// we need to create a workload source and add "DisableInstrumentation" label,
		// or remove the relevant data-stream label (if source is in multiple streams)

		// note: create will also return an existing crd (if exists) without throwing an error
		source, err := EnsureSourceCRD(ctx, nsName, workloadName, workloadKind, currentStreamName)
		if err != nil {
			return err
		}

		streamNames := GetSourceStreamNames(source)

		if len(streamNames) > 1 {
			_, err = RemoveSourceCRDLabel(ctx, nsName, source.Name, k8sconsts.SourceGroupLabelPrefix+currentStreamName)
			return err
		}

		_, err = UpdateSourceCRDSpec(ctx, nsName, source.Name, common.DisableInstrumentationJsonKey, true)
		return err
	} else {
		// namespace source does not exist.
		// we need to delete the workload source,
		// or remove the relevant data-stream label (if source is in multiple streams)

		streamNames := GetSourceStreamNames(source)

		if len(streamNames) > 1 {
			_, err = RemoveSourceCRDLabel(ctx, nsName, source.Name, k8sconsts.SourceGroupLabelPrefix+currentStreamName)
			return err
		}

		err = kube.DefaultClient.OdigosClient.Sources(nsName).Delete(ctx, source.Name, metav1.DeleteOptions{})
		return err

	}
}

func UpdateSourceCRDSpec(ctx context.Context, nsName string, crdName string, specField string, newValue any) (*v1alpha1.Source, error) {
	patch := fmt.Sprintf(`[{"op": "replace", "path": "/spec/%s", "value": %v}]`, specField, newValue)

	source, err := kube.DefaultClient.OdigosClient.Sources(nsName).Patch(
		ctx, crdName, types.JSONPatchType, []byte(patch), metav1.PatchOptions{},
	)

	return source, err
}

func UpdateSourceCRDLabel(ctx context.Context, nsName string, crdName string, labelKey string, newValue string) (*v1alpha1.Source, error) {
	escapedLabel := strings.ReplaceAll(labelKey, "/", "~1")   // replace "/" with "~1" to escape it for JSON patch
	escapedLabel = strings.ReplaceAll(escapedLabel, "\"", "") // remove quotes to avoid JSON parsing issues

	patchOps := []map[string]interface{}{
		{
			"op":    "replace",
			"path":  fmt.Sprintf("/metadata/labels/%s", escapedLabel),
			"value": newValue,
		},
	}

	patchBytes, err := json.Marshal(patchOps)
	if err != nil {
		return nil, err
	}

	source, err := kube.DefaultClient.OdigosClient.Sources(nsName).Patch(
		ctx, crdName, types.JSONPatchType, patchBytes, metav1.PatchOptions{},
	)

	return source, err
}

func RemoveSourceCRDLabel(ctx context.Context, nsName string, crdName string, labelKey string) (*v1alpha1.Source, error) {
	escapedLabel := strings.ReplaceAll(labelKey, "/", "~1")
	patch := fmt.Sprintf(`[{"op": "remove", "path": "/metadata/labels/%s"}]`, escapedLabel)

	source, err := kube.DefaultClient.OdigosClient.Sources(nsName).Patch(
		ctx, crdName, types.JSONPatchType, []byte(patch), metav1.PatchOptions{},
	)

	return source, err
}

func ToggleSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind, enabled bool, currentStreamName string) error {
	if enabled {
		_, err := EnsureSourceCRD(ctx, nsName, workloadName, workloadKind, currentStreamName)
		return err
	} else {
		return deleteSourceCRD(ctx, nsName, workloadName, workloadKind, currentStreamName)
	}
}

func GetInstrumentationInstancesHealthCondition(ctx context.Context, namespace string, name string, kind string) (model.Condition, error) {
	objectName := workload.CalculateWorkloadRuntimeObjectName(name, kind)
	if len(objectName) > 63 {
		// prevents k8s error: must be no more than 63 characters
		// see https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
		return model.Condition{}, nil
	}

	var message string
	labelSelector := fmt.Sprintf("%s=%s", consts.InstrumentedAppNameLabel, objectName)
	list, err := kube.DefaultClient.OdigosClient.InstrumentationInstances(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		message = err.Error()
	}

	totalInstances := len(list.Items)
	if totalInstances == 0 {
		// no instances so nothing to report
		return model.Condition{}, nil
	}

	healthyInstances := 0
	for _, instance := range list.Items {
		if instance.Status.Healthy != nil && *instance.Status.Healthy {
			healthyInstances++
		}
	}

	status := model.ConditionStatusSuccess
	if healthyInstances < totalInstances || message != "" {
		status = model.ConditionStatusError
	}

	reason := v1alpha1.InstrumentationInstancesHealth
	lastTransitionTime := Metav1TimeToString(metav1.NewTime(time.Time{}))
	if message == "" {
		message = fmt.Sprintf("%d/%d instances are healthy", healthyInstances, totalInstances)
	}

	condition := model.Condition{
		Type:               reason,
		Status:             status,
		Reason:             &reason,
		Message:            &message,
		LastTransitionTime: &lastTransitionTime,
	}

	return condition, nil
}

func GetInstrumentationInstancesHealthConditions(ctx context.Context) ([]*model.InstrumentationInstanceHealth, error) {
	resultMap := make(map[string]*model.InstrumentationInstanceHealth)
	list, err := kube.DefaultClient.OdigosClient.InstrumentationInstances("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, instance := range list.Items {
		namespace := instance.Namespace
		objectName, exists := instance.Labels[consts.InstrumentedAppNameLabel]
		if !exists {
			continue
		}

		name, kind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(objectName)
		if err != nil {
			continue
		}

		key := fmt.Sprintf("%s/%s/%s", namespace, name, kind)

		if _, exists := resultMap[key]; !exists {
			resultMap[key] = &model.InstrumentationInstanceHealth{
				Namespace:        namespace,
				Name:             name,
				Kind:             model.K8sResourceKind(kind),
				TotalInstances:   0,
				HealthyInstances: 0,
			}
		}

		resultMap[key].TotalInstances++
		if instance.Status.Healthy != nil && *instance.Status.Healthy {
			resultMap[key].HealthyInstances++
		}
	}

	result := make([]*model.InstrumentationInstanceHealth, 0, len(resultMap))
	for _, item := range resultMap {
		status := model.ConditionStatusSuccess
		if item.HealthyInstances < item.TotalInstances {
			status = model.ConditionStatusError
		}

		reason := v1alpha1.InstrumentationInstancesHealth
		message := fmt.Sprintf("%d/%d instances are healthy", item.HealthyInstances, item.TotalInstances)
		lastTransitionTime := Metav1TimeToString(metav1.NewTime(time.Now()))

		item.Condition = &model.Condition{
			Type:               reason,
			Status:             status,
			Reason:             &reason,
			Message:            &message,
			LastTransitionTime: &lastTransitionTime,
		}

		result = append(result, item)
	}

	return result, nil
}

func GetSourceStreamNames(source *v1alpha1.Source) []*string {
	streamNames := make([]*string, 0)

	for labelKey, labelValue := range source.Labels {
		if strings.Contains(labelKey, k8sconsts.SourceGroupLabelPrefix) && labelValue == "true" {
			streamName := strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix)
			streamNames = append(streamNames, &streamName)
		}
	}

	return streamNames
}
