package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	semconv1_21 "go.opentelemetry.io/otel/semconv/v1.21.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type K8sAttributesResolverReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

/*
An example configuration for the k8sattributes processor:
k8sattributes:
auth_type: "serviceAccount"
passthrough: false
filter:
	node_from_env_var: NODE_NAME
extract:
	metadata:
	- k8s.pod.name
	- k8s.pod.uid
	- k8s.deployment.name
	- k8s.namespace.name
	- k8s.node.name
	- k8s.pod.start_time
	labels:
	- tag_name: app.label.component
	  key: app.kubernetes.io/component
	  from: pod
pod_association:
	- sources:
	  - from: resource_attribute
		name: k8s.pod.name
	  - from: resource_attribute
		name: k8s.namespace.name
*/

var (
	workloadUIDAttributes = []string{
		string(semconv.K8SDeploymentUIDKey),
		string(semconv.K8SDaemonSetUIDKey),
		string(semconv.K8SStatefulSetUIDKey),
	}

	workloadNameAttributes = []string{
		string(semconv.K8SDeploymentNameKey),
		string(semconv.K8SDaemonSetNameKey),
		string(semconv.K8SStatefulSetNameKey),
	}

	containerAttributes = []string{
		string(semconv.K8SContainerNameKey),
		string(semconv.ContainerIDKey),
		string(semconv.ContainerImageNameKey),
		// this attribute was changes after 1.21.0 of the semantic conventions
		// the collector processor can't handle the new attribute key
		// versions <= 1.21.0 have container.image.tag
		// versions > 1.21.0 have container.image.tags
		string(semconv1_21.ContainerImageTagKey),
	}
)

type podAssociationFrom string

const (
	ResourceAttribute podAssociationFrom = "resource_attribute"
	Connection        podAssociationFrom = "connection"
)

type k8sAttributesPodsAssociationSource struct {
	From podAssociationFrom `json:"from"`
	Name string             `json:"name"`
}

type k8sAttributesPodsAssociationRule struct {
	Sources []k8sAttributesPodsAssociationSource `json:"sources"`
}

type k8sAttributesPodsAssociation []k8sAttributesPodsAssociationRule

type k8sAttributesFilter struct {
	NodeFromEnvVar string `json:"node_from_env_var"`
}

type k8sTagAttribute struct {
	Tag  string `json:"tag_name"`
	Key  string `json:"key"`
	From string `json:"from"`
}

type k8sAttributeExtract struct {
	MetadataAttributes   []string          `json:"metadata,omitempty"`
	LabelAttributes      []k8sTagAttribute `json:"labels,omitempty"`
	AnnotationAttributes []k8sTagAttribute `json:"annotations,omitempty"`
}

type k8sAttributesConfig struct {
	AuthType       string                       `json:"auth_type"`
	Passthrough    bool                         `json:"passthrough"`
	Filter         k8sAttributesFilter          `json:"filter"`
	Extract        k8sAttributeExtract          `json:"extract"`
	PodAssociation k8sAttributesPodsAssociation `json:"pod_association"`
}

func (r *K8sAttributesResolverReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling K8sAttributes action")

	actions := actionv1.K8sAttributesResolverList{}
	err := r.List(ctx, &actions, client.InNamespace(req.Namespace))
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	processor, err := r.convertToUnifiedProcessor(&actions, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Patch(ctx, processor, client.Apply, client.FieldOwner("odigos-k8sattributesresolver"), client.ForceOwnership)
	reportErr := r.reportActionsStatuses(ctx, &actions, err)
	return ctrl.Result{}, errors.Join(err, reportErr)
}

func (r *K8sAttributesResolverReconciler) convertToUnifiedProcessor(actions *actionv1.K8sAttributesResolverList, ns string) (*odigosv1alpha1.Processor, error) {
	processor := odigosv1alpha1.Processor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-k8sattributes",
			Namespace: ns,
		},
		Spec: odigosv1alpha1.ProcessorSpec{
			Type:          "k8sattributes",
			ProcessorName: "Unified Kubernetes Attributes",
			CollectorRoles: []odigosv1alpha1.CollectorsGroupRole{
				odigosv1alpha1.CollectorsGroupRoleNodeCollector,
			},
			OrderHint: 0,
			Disabled:  false,
		},
	}

	// first, initialize the config with our configuration fields which are not configurable by the user
	config := k8sAttributesConfig{
		AuthType:    "serviceAccount",
		Passthrough: false,
		// restrict the collector to query pods running on the same node only - reducing resource requirements.
		Filter: k8sAttributesFilter{
			NodeFromEnvVar: k8sconsts.NodeNameEnvVar,
		},
		// each trace/metric/log will be associated with the pod it originated from
		// based on the pod name and namespace resource attributes
		PodAssociation: k8sAttributesPodsAssociation{
			{
				Sources: []k8sAttributesPodsAssociationSource{
					{
						From: ResourceAttribute,
						Name: string(semconv.K8SPodNameKey),
					},
					{
						From: ResourceAttribute,
						Name: string(semconv.K8SNamespaceNameKey),
					},
				},
			},
		},
	}

	// annotation key -> attribute key
	annotation := map[string]string{}
	// label key -> attribute key
	labels := map[string]string{}
	// observability signals
	signals := map[common.ObservabilitySignal]struct{}{}
	collectWorkloadUID := false
	collectContainerAttributes := false
	collectClusterUID := false
	collectWorkloadNames := false
	collectReplicaSetAttributes := false

	// create a union of all the actions' configuration to one processor
	for actionIndex := range actions.Items {
		currentAction := &actions.Items[actionIndex]

		if currentAction.Spec.Disabled {
			continue
		}

		collectContainerAttributes = (collectContainerAttributes || currentAction.Spec.CollectContainerAttributes)
		collectWorkloadUID = (collectWorkloadUID || currentAction.Spec.CollectWorkloadUID)
		collectReplicaSetAttributes = (collectReplicaSetAttributes || currentAction.Spec.CollectReplicaSetAttributes)
		collectClusterUID = (collectClusterUID || currentAction.Spec.CollectClusterUID)
		// traces should already contain workload name (if they originated from odigos)
		// logs collected from filelog receiver will lack this info thus needs to be added
		collectWorkloadNames = (collectWorkloadNames || slices.Contains(currentAction.Spec.Signals, common.LogsObservabilitySignal))

		for labelIndex := range currentAction.Spec.LabelsAttributes {
			labels[currentAction.Spec.LabelsAttributes[labelIndex].LabelKey] = currentAction.Spec.LabelsAttributes[labelIndex].AttributeKey
		}
		for annotationIndex := range currentAction.Spec.AnnotationsAttributes {
			annotation[currentAction.Spec.AnnotationsAttributes[annotationIndex].AnnotationKey] = currentAction.Spec.AnnotationsAttributes[annotationIndex].AttributeKey
		}
		for signalIndex := range currentAction.Spec.Signals {
			signals[currentAction.Spec.Signals[signalIndex]] = struct{}{}
		}

		processor.ObjectMeta.OwnerReferences = append(processor.ObjectMeta.OwnerReferences, metav1.OwnerReference{
			APIVersion: currentAction.APIVersion,
			Kind:       currentAction.Kind,
			Name:       currentAction.Name,
			UID:        currentAction.UID,
		})
	}

	if collectWorkloadUID {
		config.Extract.MetadataAttributes = append(config.Extract.MetadataAttributes, workloadUIDAttributes...)
		if collectReplicaSetAttributes {
			config.Extract.MetadataAttributes = append(config.Extract.MetadataAttributes, string(semconv.K8SReplicaSetUIDKey))
		}
	}
	if collectWorkloadNames {
		config.Extract.MetadataAttributes = append(config.Extract.MetadataAttributes, workloadNameAttributes...)
	}
	if collectReplicaSetAttributes {
		config.Extract.MetadataAttributes = append(config.Extract.MetadataAttributes, string(semconv.K8SReplicaSetNameKey))
	}
	if collectContainerAttributes {
		config.Extract.MetadataAttributes = append(config.Extract.MetadataAttributes, containerAttributes...)
	}
	if collectClusterUID {
		config.Extract.MetadataAttributes = append(config.Extract.MetadataAttributes, string(semconv.K8SClusterUIDKey))
	}

	for key, value := range labels {
		// The naming by the collector processor is:
		//	- tag == resource attribute key
		//	- key == label key
		config.Extract.LabelAttributes = append(config.Extract.LabelAttributes, k8sTagAttribute{
			Tag:  value,
			Key:  key,
			From: "pod",
		})
	}

	for key, value := range annotation {
		// The naming by the collector processor is:
		//	- tag == resource attribute key
		//	- key == annotation key
		config.Extract.AnnotationAttributes = append(config.Extract.AnnotationAttributes, k8sTagAttribute{
			Tag:  value,
			Key:  key,
			From: "pod",
		})
	}

	configJson, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	processor.Spec.ProcessorConfig = runtime.RawExtension{Raw: configJson}
	processor.Spec.Signals = []common.ObservabilitySignal{}
	for signal := range signals {
		processor.Spec.Signals = append(processor.Spec.Signals, signal)
	}

	return &processor, nil
}

func (r *K8sAttributesResolverReconciler) reportActionsStatuses(ctx context.Context, actions *actionv1.K8sAttributesResolverList, processorErr error) error {
	var updateErr error
	status := metav1.ConditionTrue
	message := "The action successfully transformed to a unified processor"
	reason := "ProcessorCreated"

	if processorErr != nil {
		status = metav1.ConditionFalse
		message = fmt.Sprintf("Failed to transform the action to a unified processor: %s", processorErr.Error())
		reason = "ProcessorCreationFailed"
	}

	for actionIndex := range actions.Items {
		action := &actions.Items[actionIndex]
		changed := meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
			Type:    "ActionTransformedToProcessorType",
			Status:  status,
			Reason:  reason,
			Message: message,
		})

		if changed {
			err := r.Status().Update(ctx, action)
			updateErr = errors.Join(updateErr, err)
		}
	}

	return updateErr
}
