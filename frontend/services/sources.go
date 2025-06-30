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
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/version"
)

const (
	WorkloadKindNamespace   model.K8sResourceKind = "Namespace"
	WorkloadKindDeployment  model.K8sResourceKind = "Deployment"
	WorkloadKindStatefulSet model.K8sResourceKind = "StatefulSet"
	WorkloadKindDaemonSet   model.K8sResourceKind = "DaemonSet"
	WorkloadKindCronJob     model.K8sResourceKind = "CronJob"
)

func GetWorkloadsInNamespace(ctx context.Context, nsName string) ([]model.K8sActualSource, error) {
	namespace, err := kube.DefaultClient.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	g, ctx := errgroup.WithContext(ctx)
	var (
		deps      []model.K8sActualSource
		statefuls []model.K8sActualSource
		daemons   []model.K8sActualSource
		crons     []model.K8sActualSource
	)

	g.Go(func() error {
		var err error
		deps, err = getDeployments(ctx, *namespace)
		return err
	})

	g.Go(func() error {
		var err error
		statefuls, err = getStatefulSets(ctx, *namespace)
		return err
	})

	g.Go(func() error {
		var err error
		daemons, err = getDaemonSets(ctx, *namespace)
		return err
	})

	g.Go(func() error {
		var err error
		crons, err = getCronJobs(ctx, *namespace)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	items := make([]model.K8sActualSource, len(deps)+len(statefuls)+len(daemons)+len(crons))
	copy(items, deps)
	copy(items[len(deps):], statefuls)
	copy(items[len(deps)+len(statefuls):], daemons)
	copy(items[len(deps)+len(statefuls)+len(daemons):], crons)

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
				Kind:              WorkloadKindDeployment,
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
				Kind:              WorkloadKindDaemonSet,
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
				Kind:              WorkloadKindStatefulSet,
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

func getCronJobs(ctx context.Context, namespace corev1.Namespace) ([]model.K8sActualSource, error) {
	var response []model.K8sActualSource

	ver, err := getKubeVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to detect Kubernetes version: %w", err)
	}

	// Note: batchv1beta1 is deprecated in Kubernetes 1.21 and removed in 1.25
	// so we use batchv1beta1 for versions < 1.21 and batchv1 for >= 1.21
	// this is to ensure compatibility with older Kubernetes versions.
	if ver.LessThan(version.MustParseSemantic("1.21.0")) {
		err = client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.BatchV1beta1().CronJobs(namespace.Name).List, ctx, &metav1.ListOptions{}, func(cjs *batchv1beta1.CronJobList) error {
			for _, cj := range cjs.Items {
				numberOfInstances := len(cj.Status.Active)
				response = append(response, model.K8sActualSource{
					Namespace:         cj.Namespace,
					Name:              cj.Name,
					Kind:              WorkloadKindCronJob,
					NumberOfInstances: &numberOfInstances,
				})
			}
			return nil
		})
	} else {
		err = client.ListWithPages(client.DefaultPageSize, kube.DefaultClient.BatchV1().CronJobs(namespace.Name).List, ctx, &metav1.ListOptions{}, func(cjs *batchv1.CronJobList) error {
			for _, cj := range cjs.Items {
				numberOfInstances := len(cj.Status.Active)
				response = append(response, model.K8sActualSource{
					Namespace:         cj.Namespace,
					Name:              cj.Name,
					Kind:              WorkloadKindCronJob,
					NumberOfInstances: &numberOfInstances,
				})
			}
			return nil
		})
	}

	if err != nil {
		return nil, err
	}

	return response, nil
}

func RolloutRestartWorkload(ctx context.Context, namespace string, name string, kind model.K8sResourceKind) error {
	now := time.Now().Format(time.RFC3339)

	switch kind {
	case WorkloadKindDeployment:
		dep, err := kube.DefaultClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get deployment: %w", err)
		}

		if dep.Spec.Template.Annotations == nil {
			dep.Spec.Template.Annotations = map[string]string{}
		}
		dep.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = now

		_, err = kube.DefaultClient.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update deployment: %w", err)
		}

	case WorkloadKindStatefulSet:
		sts, err := kube.DefaultClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get statefulset: %w", err)
		}

		if sts.Spec.Template.Annotations == nil {
			sts.Spec.Template.Annotations = map[string]string{}
		}
		sts.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = now

		_, err = kube.DefaultClient.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update statefulset: %w", err)
		}

	case WorkloadKindDaemonSet:
		ds, err := kube.DefaultClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get daemonset: %w", err)
		}

		if ds.Spec.Template.Annotations == nil {
			ds.Spec.Template.Annotations = map[string]string{}
		}
		ds.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = now

		_, err = kube.DefaultClient.AppsV1().DaemonSets(namespace).Update(ctx, ds, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update daemonset: %w", err)
		}

	case WorkloadKindCronJob:
		// CronJobs do not support rolling restarts.
		// We return nil here to prevent an error, as this is a no-op.
		return nil

	default:
		return fmt.Errorf("unsupported kind: %s (must be Deployment, StatefulSet, DaemonSet or CronJob)", kind)
	}

	return nil
}

func GetSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind model.K8sResourceKind) (*v1alpha1.Source, error) {
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

func stringToWorkloadKind(workloadKind string) (model.K8sResourceKind, bool) {
	switch strings.ToLower(workloadKind) {
	case "namespace":
		return WorkloadKindNamespace, true
	case "deployment":
		return WorkloadKindDeployment, true
	case "statefulset":
		return WorkloadKindStatefulSet, true
	case "daemonset":
		return WorkloadKindDaemonSet, true
	case "cronjob":
		return WorkloadKindCronJob, true
	}

	return "", false
}

func EnsureSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind model.K8sResourceKind, currentStreamName string) (*v1alpha1.Source, error) {
	streamLabel := ""
	if currentStreamName != "" {
		streamLabel = k8sconsts.SourceDataStreamLabelPrefix + currentStreamName
	}

	switch workloadKind {
	// Namespace is not a workload, but we need it to "select future apps" by creating a Source CRD for it
	case WorkloadKindNamespace, WorkloadKindDeployment, WorkloadKindStatefulSet, WorkloadKindDaemonSet, WorkloadKindCronJob:
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
		if streamLabel != "" {
			source, err = UpdateSourceCRDLabel(ctx, nsName, source.Name, streamLabel, "true")
			if err != nil {
				return nil, err
			}
		}
		return source, nil
	}

	newSource := &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "source-",
		},
		Spec: v1alpha1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Namespace: nsName,
				Name:      workloadName,
				Kind:      k8sconsts.WorkloadKind(workloadKind),
			},
		},
	}
	if currentStreamName != "" {
		newSource.ObjectMeta.Labels = map[string]string{
			streamLabel: "true",
		}
	}

	return CreateResourceWithGenerateName(ctx, func() (*v1alpha1.Source, error) {
		return kube.DefaultClient.OdigosClient.Sources(nsName).Create(ctx, newSource, metav1.CreateOptions{})
	})
}

func deleteSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind model.K8sResourceKind, currentStreamName string) error {
	source, err := EnsureSourceCRD(ctx, nsName, workloadName, workloadKind, currentStreamName)
	if err != nil {
		return err
	}

	// check for namespace source first
	var nsSource *v1alpha1.Source
	if workloadKind != WorkloadKindNamespace {
		nsSource, err = GetSourceCRD(ctx, nsName, nsName, WorkloadKindNamespace)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	dataStreamNames := ExtractDataStreamsFromSource(source, nsSource)
	isWorkloadWithNamespace := workloadKind != WorkloadKindNamespace && nsSource != nil

	// we remove the current data-stream
	if currentStreamName != "" {
		dataStreamLabelKey := k8sconsts.SourceDataStreamLabelPrefix + currentStreamName

		if isWorkloadWithNamespace {
			_, err = UpdateSourceCRDLabel(ctx, nsName, source.Name, dataStreamLabelKey, "false")
		} else {
			_, err = UpdateSourceCRDLabel(ctx, nsName, source.Name, dataStreamLabelKey, "")
		}

		// if there are more labels for data-streams, we exit and don't delete the source
		if len(dataStreamNames) > 1 && currentStreamName != "" {
			return err
		}
	}

	if isWorkloadWithNamespace {
		// we add "DisableInstrumentation" label to the source
		_, err = UpdateSourceCRDSpec(ctx, nsName, source.Name, common.DisableInstrumentationJsonKey, true)
		return err
	} else {
		// namespace source does not exist.
		// we need to delete the workload source,
		// or remove the relevant data-stream label (if source is in multiple streams)
		if len(dataStreamNames) > 1 && currentStreamName != "" {
			_, err = UpdateSourceCRDLabel(ctx, nsName, source.Name, k8sconsts.SourceDataStreamLabelPrefix+currentStreamName, "")
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

	patchOps := []map[string]interface{}{}

	if newValue == "" {
		// if no value, remove the label
		patchOps = append(patchOps, map[string]interface{}{
			"op":   "remove",
			"path": fmt.Sprintf("/metadata/labels/%s", escapedLabel),
		})
	} else {
		// if value is provided, replace the label
		patchOps = append(patchOps, map[string]interface{}{
			"op":    "replace",
			"path":  fmt.Sprintf("/metadata/labels/%s", escapedLabel),
			"value": newValue,
		})
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

func ToggleSourceCRD(ctx context.Context, nsName string, workloadName string, workloadKind model.K8sResourceKind, enabled bool, currentStreamName string) error {
	if enabled {
		_, err := EnsureSourceCRD(ctx, nsName, workloadName, workloadKind, currentStreamName)
		return err
	} else {
		return deleteSourceCRD(ctx, nsName, workloadName, workloadKind, currentStreamName)
	}
}

type InstanceCounts struct {
	TotalInstances   int
	HealthyInstances int
}

func getInstrumentationInstancesConditions(ctx context.Context, namespace string, name string, kind string) ([]*model.SourceConditions, error) {
	result := make([]*model.SourceConditions, 0)
	conditionsMap := make(map[string]*model.SourceConditions)
	instanceCountsMap := make(map[string]*InstanceCounts)

	listOptions := metav1.ListOptions{}

	if namespace != "" && name != "" && kind != "" {
		objectName := workload.CalculateWorkloadRuntimeObjectName(name, kind)
		if len(objectName) > 63 {
			// prevents k8s error: must be no more than 63 characters
			// see https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
			return result, nil
		}
		listOptions.LabelSelector = fmt.Sprintf("%s=%s", consts.InstrumentedAppNameLabel, objectName)
	}

	list, err := kube.DefaultClient.OdigosClient.InstrumentationInstances("").List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	// Count instances and group by workload
	for _, instance := range list.Items {
		objectName, exists := instance.Labels[consts.InstrumentedAppNameLabel]
		if !exists {
			continue
		}
		pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(objectName, instance.Namespace)
		if err != nil {
			continue
		}
		key := fmt.Sprintf("%s/%s/%s", pw.Namespace, pw.Name, pw.Kind)

		if _, exists := conditionsMap[key]; !exists {
			conditionsMap[key] = &model.SourceConditions{
				Namespace:  pw.Namespace,
				Name:       pw.Name,
				Kind:       model.K8sResourceKind(pw.Kind),
				Conditions: []*model.Condition{},
			}
			instanceCountsMap[key] = &InstanceCounts{
				TotalInstances:   0,
				HealthyInstances: 0,
			}
		}

		instanceCountsMap[key].TotalInstances++
		if instance.Status.Healthy != nil && *instance.Status.Healthy {
			instanceCountsMap[key].HealthyInstances++
		}
	}

	// Create conditions for each workload
	for _, item := range conditionsMap {
		key := fmt.Sprintf("%s/%s/%s", item.Namespace, item.Name, item.Kind)
		instanceCounts := instanceCountsMap[key]

		status := model.ConditionStatusSuccess
		if instanceCounts.HealthyInstances < instanceCounts.TotalInstances {
			status = model.ConditionStatusError
		}

		reason := v1alpha1.InstrumentationInstancesHealth
		message := fmt.Sprintf("%d/%d instances are healthy", instanceCounts.HealthyInstances, instanceCounts.TotalInstances)
		lastTransitionTime := Metav1TimeToString(metav1.NewTime(time.Now()))

		item.Conditions = append(item.Conditions, &model.Condition{
			Type:               reason,
			Status:             status,
			Reason:             &reason,
			Message:            &message,
			LastTransitionTime: &lastTransitionTime,
		})

		result = append(result, item)
	}

	return result, nil
}

func getWorkloadsConditions(ctx context.Context, namespace string, name string, kind string) ([]*model.SourceConditions, error) {
	result := make([]*model.SourceConditions, 0)
	conditionsMap := make(map[string]*model.SourceConditions)

	// Deployments
	if model.K8sResourceKind(kind) == model.K8sResourceKindDeployment || kind == "" {
		deployments := make([]appsv1.Deployment, 0)

		if namespace == "" && name == "" && kind == "" {
			list, err := kube.DefaultClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, err
			}

			deployments = append(deployments, list.Items...)
		} else {
			dep, err := kube.DefaultClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			deployments = append(deployments, *dep)
		}

		for _, dep := range deployments {
			key := fmt.Sprintf("%s/%s/%s", dep.Namespace, dep.Name, model.K8sResourceKindDeployment)

			if _, exists := conditionsMap[key]; !exists {
				conditionsMap[key] = &model.SourceConditions{
					Namespace:  dep.Namespace,
					Name:       dep.Name,
					Kind:       model.K8sResourceKindDeployment,
					Conditions: []*model.Condition{},
				}
			}

			for _, c := range dep.Status.Conditions {
				status := TransformConditionStatus(metav1.ConditionStatus(c.Status), string(c.Type), c.Reason)
				lastTransitionTime := Metav1TimeToString(c.LastTransitionTime)

				conditionsMap[key].Conditions = append(conditionsMap[key].Conditions, &model.Condition{
					Status:             status,
					Type:               string(c.Type),
					Reason:             &c.Reason,
					Message:            &c.Message,
					LastTransitionTime: &lastTransitionTime,
				})
			}
		}
	}

	// DaemonSets
	if model.K8sResourceKind(kind) == model.K8sResourceKindDaemonSet || kind == "" {
		daemonSets := make([]appsv1.DaemonSet, 0)

		if namespace == "" && name == "" && kind == "" {
			list, err := kube.DefaultClient.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, err
			}

			daemonSets = append(daemonSets, list.Items...)
		} else {
			ds, err := kube.DefaultClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			daemonSets = append(daemonSets, *ds)
		}

		for _, ds := range daemonSets {
			key := fmt.Sprintf("%s/%s/%s", ds.Namespace, ds.Name, model.K8sResourceKindDaemonSet)

			if _, exists := conditionsMap[key]; !exists {
				conditionsMap[key] = &model.SourceConditions{
					Namespace:  ds.Namespace,
					Name:       ds.Name,
					Kind:       model.K8sResourceKindDaemonSet,
					Conditions: []*model.Condition{},
				}
			}

			for _, c := range ds.Status.Conditions {
				status := TransformConditionStatus(metav1.ConditionStatus(c.Status), string(c.Type), c.Reason)
				lastTransitionTime := Metav1TimeToString(c.LastTransitionTime)

				conditionsMap[key].Conditions = append(conditionsMap[key].Conditions, &model.Condition{
					Status:             status,
					Type:               string(c.Type),
					Reason:             &c.Reason,
					Message:            &c.Message,
					LastTransitionTime: &lastTransitionTime,
				})
			}
		}
	}

	// StatefulSets
	if model.K8sResourceKind(kind) == model.K8sResourceKindStatefulSet || kind == "" {
		statefulSets := make([]appsv1.StatefulSet, 0)

		if namespace == "" && name == "" && kind == "" {
			list, err := kube.DefaultClient.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, err
			}

			statefulSets = append(statefulSets, list.Items...)
		} else {
			ss, err := kube.DefaultClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			statefulSets = append(statefulSets, *ss)
		}

		for _, ss := range statefulSets {
			key := fmt.Sprintf("%s/%s/%s", ss.Namespace, ss.Name, model.K8sResourceKindStatefulSet)

			if _, exists := conditionsMap[key]; !exists {
				conditionsMap[key] = &model.SourceConditions{
					Namespace:  ss.Namespace,
					Name:       ss.Name,
					Kind:       model.K8sResourceKindStatefulSet,
					Conditions: []*model.Condition{},
				}
			}

			for _, c := range ss.Status.Conditions {
				status := TransformConditionStatus(metav1.ConditionStatus(c.Status), string(c.Type), c.Reason)
				lastTransitionTime := Metav1TimeToString(c.LastTransitionTime)

				conditionsMap[key].Conditions = append(conditionsMap[key].Conditions, &model.Condition{
					Status:             status,
					Type:               string(c.Type),
					Reason:             &c.Reason,
					Message:            &c.Message,
					LastTransitionTime: &lastTransitionTime,
				})
			}
		}
	}

	// Final result
	for _, item := range conditionsMap {
		result = append(result, item)
	}

	return result, nil
}

func GetOtherConditionsForSources(ctx context.Context, namespace string, name string, kind string) ([]*model.SourceConditions, error) {
	result := make([]*model.SourceConditions, 0)
	conditionsMap := make(map[string]*model.SourceConditions)

	instancesConditions, err := getInstrumentationInstancesConditions(ctx, namespace, name, kind)
	if err != nil {
		return nil, err
	}
	for _, instanceItem := range instancesConditions {
		key := fmt.Sprintf("%s/%s/%s", instanceItem.Namespace, instanceItem.Name, instanceItem.Kind)
		if _, exists := conditionsMap[key]; !exists {
			conditionsMap[key] = instanceItem
		} else {
			conditionsMap[key].Conditions = append(conditionsMap[key].Conditions, instanceItem.Conditions...)
			SortConditions(conditionsMap[key].Conditions)
		}
	}

	workloadsConditions, err := getWorkloadsConditions(ctx, namespace, name, kind)
	if err != nil {
		return nil, err
	}
	for _, workloadItem := range workloadsConditions {
		key := fmt.Sprintf("%s/%s/%s", workloadItem.Namespace, workloadItem.Name, workloadItem.Kind)
		if _, exists := conditionsMap[key]; !exists {
			conditionsMap[key] = workloadItem
		} else {
			conditionsMap[key].Conditions = append(conditionsMap[key].Conditions, workloadItem.Conditions...)
			SortConditions(conditionsMap[key].Conditions)
		}
	}

	for _, item := range conditionsMap {
		result = append(result, item)
	}

	return result, nil
}
