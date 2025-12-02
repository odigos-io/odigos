package services

import (
	"context"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetPodDetails(ctx context.Context, namespace, name string) (*model.PodDetails, error) {
	pod, err := kube.DefaultClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var nodePtr *string
	if pod.Spec.NodeName != "" {
		n := pod.Spec.NodeName
		nodePtr = &n
	}

	var rolePtr *string
	if pod.Labels != nil {
		if role, ok := pod.Labels[k8sconsts.OdigosCollectorRoleLabel]; ok && role != "" {
			r := role
			rolePtr = &r
		}
	}

	var statusPtr *string
	if string(pod.Status.Phase) != "" {
		s := string(pod.Status.Phase)
		statusPtr = &s
	}

	conditions := convertPodConditions(pod.Status.Conditions)

	containers := buildContainersOverview(pod)

	manifestYAML, err := K8sManifest(ctx, namespace, model.K8sResourceKindPod, name)
	if err != nil {
		return nil, err
	}

	return &model.PodDetails{
		Name:         pod.Name,
		Namespace:    pod.Namespace,
		Node:         nodePtr,
		Role:         rolePtr,
		Status:       statusPtr,
		Conditions:   conditions,
		Containers:   containers,
		ManifestYaml: manifestYAML,
	}, nil
}

// TODO: Create a dedicated services conversion file and move all conversion helpers there.
func convertPodConditions(k8sConds []corev1.PodCondition) []*model.PodCondition {
	conds := make([]*model.PodCondition, 0, len(k8sConds))
	for _, c := range k8sConds {
		var lttPtr *string
		if !c.LastTransitionTime.IsZero() {
			ltt := c.LastTransitionTime.Time.Format(time.RFC3339)
			lttPtr = &ltt
		}
		typeEnum := mapPodConditionType(c.Type)
		statusEnum := mapK8sConditionStatus(c.Status)
		var reasonPtr, messagePtr *string
		if c.Reason != "" {
			r := c.Reason
			reasonPtr = &r
		}
		if c.Message != "" {
			m := c.Message
			messagePtr = &m
		}
		conds = append(conds, &model.PodCondition{
			Type:               typeEnum,
			Status:             statusEnum,
			LastTransitionTime: lttPtr,
			Reason:             reasonPtr,
			Message:            messagePtr,
		})
	}
	return conds
}

func buildContainersOverview(pod *corev1.Pod) []*model.ContainerOverview {

	statusByName := make(map[string]corev1.ContainerStatus, len(pod.Status.ContainerStatuses))
	for _, cs := range pod.Status.ContainerStatuses {
		statusByName[cs.Name] = cs
	}

	containers := make([]*model.ContainerOverview, 0, len(pod.Spec.Containers))
	for _, c := range pod.Spec.Containers {
		cs, ok := statusByName[c.Name]

		ready := false
		restarts := 0
		status := model.ContainerLifecycleStatusWaiting
		var stateReasonPtr *string
		var startedAtPtr *string

		if ok {
			ready = cs.Ready
			restarts = int(cs.RestartCount)
			if cs.State.Running != nil {
				status = model.ContainerLifecycleStatusRunning
				if !cs.State.Running.StartedAt.IsZero() {
					startedAt := cs.State.Running.StartedAt.Time.Format(time.RFC3339)
					startedAtPtr = &startedAt
				}
			} else if cs.State.Waiting != nil {
				status = model.ContainerLifecycleStatusWaiting
				if cs.State.Waiting.Reason != "" {
					reason := cs.State.Waiting.Reason
					stateReasonPtr = &reason
				}
			} else if cs.State.Terminated != nil {
				status = model.ContainerLifecycleStatusTerminated
				if cs.State.Terminated.Reason != "" {
					reason := cs.State.Terminated.Reason
					stateReasonPtr = &reason
				}
			}
		}

		containers = append(containers, &model.ContainerOverview{
			Name:        c.Name,
			Image:       &c.Image,
			Status:      status,
			StateReason: stateReasonPtr,
			Ready:       ready,
			Restarts:    restarts,
			StartedAt:   startedAtPtr,
			Resources:   buildContainerResources(c.Resources),
		})
	}
	return containers
}

func mapK8sConditionStatus(s corev1.ConditionStatus) *model.K8sConditionStatus {
	switch s {
	case corev1.ConditionTrue:
		v := model.K8sConditionStatusTrue
		return &v
	case corev1.ConditionFalse:
		v := model.K8sConditionStatusFalse
		return &v
	case corev1.ConditionUnknown:
		v := model.K8sConditionStatusUnknown
		return &v
	default:
		return nil
	}
}

func mapPodConditionType(t corev1.PodConditionType) *model.PodConditionType {
	switch t {
	case corev1.PodScheduled:
		v := model.PodConditionTypePodScheduled
		return &v
	case corev1.PodInitialized:
		v := model.PodConditionTypeInitialized
		return &v
	case corev1.ContainersReady:
		v := model.PodConditionTypeContainersReady
		return &v
	case corev1.PodReady:
		v := model.PodConditionTypeReady
		return &v
	default:
		v := model.PodConditionTypeOther
		return &v
	}
}

func buildContainerResources(reqs corev1.ResourceRequirements) *model.Resources {
	if reqs.Requests == nil && reqs.Limits == nil {
		return nil
	}

	var requests *model.ResourceAmounts
	if len(reqs.Requests) > 0 {
		var cpuPtr, memPtr *string
		if q, ok := reqs.Requests[corev1.ResourceCPU]; ok {
			s := q.String()
			cpuPtr = &s
		}
		if q, ok := reqs.Requests[corev1.ResourceMemory]; ok {
			s := q.String()
			memPtr = &s
		}
		if cpuPtr != nil || memPtr != nil {
			requests = &model.ResourceAmounts{
				CPU:    cpuPtr,
				Memory: memPtr,
			}
		}
	}

	var limits *model.ResourceAmounts
	if len(reqs.Limits) > 0 {
		var cpuPtr, memPtr *string
		if q, ok := reqs.Limits[corev1.ResourceCPU]; ok {
			s := q.String()
			cpuPtr = &s
		}
		if q, ok := reqs.Limits[corev1.ResourceMemory]; ok {
			s := q.String()
			memPtr = &s
		}
		if cpuPtr != nil || memPtr != nil {
			limits = &model.ResourceAmounts{
				CPU:    cpuPtr,
				Memory: memPtr,
			}
		}
	}

	if requests == nil && limits == nil {
		return nil
	}

	return &model.Resources{
		Requests: requests,
		Limits:   limits,
	}
}
