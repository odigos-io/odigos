package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
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
		string(semconv.K8SCronJobUIDKey),
		string(semconv.K8SJobUIDKey),
	}

	workloadNameAttributes = []string{
		string(semconv.K8SDeploymentNameKey),
		string(semconv.K8SDaemonSetNameKey),
		string(semconv.K8SStatefulSetNameKey),
		string(semconv.K8SCronJobNameKey),
		string(semconv.K8SJobNameKey),
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

	processor, err := r.convertToUnifiedProcessor(ctx, req.Namespace, &actions)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Patch(ctx, processor, client.Apply, client.FieldOwner("odigos-k8sattributesresolver"), client.ForceOwnership)
	reportErr := r.reportActionsStatuses(ctx, &actions, err)
	return ctrl.Result{}, errors.Join(err, reportErr)
}

func (r *K8sAttributesResolverReconciler) convertToUnifiedProcessor(ctx context.Context, ns string, actions *actionv1.K8sAttributesResolverList) (*odigosv1.Processor, error) {
	processor := odigosv1.Processor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-k8sattributes",
			Namespace: ns,
		},
		Spec: odigosv1.ProcessorSpec{
			Type:          "k8sattributes",
			ProcessorName: "Unified Kubernetes Attributes",
			CollectorRoles: []odigosv1.CollectorsGroupRole{
				odigosv1.CollectorsGroupRoleNodeCollector,
			},
			OrderHint: 0,
			Disabled:  false,
		},
	}

	// Convert legacy K8sAttributesResolver to Action format
	var legacyConfigs []*odigosv1.Action
	for i := range actions.Items {
		legacyConfigs = append(legacyConfigs, convertK8sAttributesResolverToAction(&actions.Items[i]))
	}

	config, signals, ownerReferences, err := k8sAttributeConfig(ctx, r.Client, ns, legacyConfigs)
	if err != nil {
		return nil, err
	}

	processor.ObjectMeta.OwnerReferences = ownerReferences
	for signal := range signals {
		processor.Spec.Signals = append(processor.Spec.Signals, signal)
	}
	configJson, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	processor.Spec.ProcessorConfig = runtime.RawExtension{Raw: configJson}
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

// convertK8sAttributesResolverToAction converts a K8sAttributesResolver to an Action
func convertK8sAttributesResolverToAction(resolver *actionv1.K8sAttributesResolver) *odigosv1.Action {
	return &odigosv1.Action{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Action",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resolver.Name,
			Namespace: resolver.Namespace,
		},
		Spec: odigosv1.ActionSpec{
			ActionName: resolver.Spec.ActionName,
			Notes:      resolver.Spec.Notes,
			Disabled:   resolver.Spec.Disabled,
			Signals:    resolver.Spec.Signals,
			K8sAttributes: &actionv1.K8sAttributesConfig{
				CollectContainerAttributes:  resolver.Spec.CollectContainerAttributes,
				CollectReplicaSetAttributes: resolver.Spec.CollectReplicaSetAttributes,
				CollectWorkloadUID:          resolver.Spec.CollectWorkloadUID,
				CollectClusterUID:           resolver.Spec.CollectClusterUID,
				LabelsAttributes:            resolver.Spec.LabelsAttributes,
				AnnotationsAttributes:       resolver.Spec.AnnotationsAttributes,
			},
		},
	}
}

// k8sAttributeConfig combines multiple k8sattributes configurations into a single unified processor config
func k8sAttributeConfig(ctx context.Context, k8sclient client.Client, namespace string, legacyConfigs []*odigosv1.Action) (*k8sAttributesConfig, map[common.ObservabilitySignal]struct{}, []metav1.OwnerReference, error) {
	// Get all actions in the namespace
	actionList := &odigosv1.ActionList{}
	err := k8sclient.List(ctx, actionList, client.InNamespace(namespace))
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		metadataAttributes   []string
		labelAttributes      = make(map[string]k8sTagAttribute)
		annotationAttributes = make(map[string]k8sTagAttribute)
		signals              = map[common.ObservabilitySignal]struct{}{}
		ownerReferences      = []metav1.OwnerReference{}
		collectContainer     = false
		collectReplicaSet    = false
		collectWorkloadUID   = false
		collectClusterUID    = false
		collectWorkloadNames = false
	)

	// Merge legacy configs with current actions
	var allActions []*odigosv1.Action
	allActions = append(allActions, legacyConfigs...)
	for i := range actionList.Items {
		allActions = append(allActions, &actionList.Items[i])
	}

	// Collect all k8sattributes configurations
	for _, currentAction := range allActions {
		if currentAction.Spec.K8sAttributes == nil || currentAction.Spec.Disabled {
			continue
		}

		config := currentAction.Spec.K8sAttributes

		// create a union of all the actions' configuration to one processor
		collectContainer = collectContainer || config.CollectContainerAttributes
		collectReplicaSet = collectReplicaSet || config.CollectReplicaSetAttributes
		collectWorkloadUID = collectWorkloadUID || config.CollectWorkloadUID
		collectClusterUID = collectClusterUID || config.CollectClusterUID
		// traces should already contain workload name (if they originated from odigos)
		// logs collected from filelog receiver will lack this info thus needs to be added
		collectWorkloadNames = (collectWorkloadNames || slices.Contains(currentAction.Spec.Signals, common.LogsObservabilitySignal))

		// Add label attributes, newer configs override older ones with same Tag
		for _, label := range config.LabelsAttributes {
			labelAttributes[label.LabelKey] = k8sTagAttribute{
				Tag:  label.AttributeKey,
				Key:  label.LabelKey,
				From: "pod",
			}
		}

		// Add annotation attributes, newer configs override older ones with same Tag
		for _, annotation := range config.AnnotationsAttributes {
			annotationAttributes[annotation.AnnotationKey] = k8sTagAttribute{
				Tag:  annotation.AttributeKey,
				Key:  annotation.AnnotationKey,
				From: "pod",
			}
		}

		for signalIndex := range currentAction.Spec.Signals {
			signals[currentAction.Spec.Signals[signalIndex]] = struct{}{}
		}

		ownerReferences = append(ownerReferences, metav1.OwnerReference{
			APIVersion: currentAction.APIVersion,
			Kind:       currentAction.Kind,
			Name:       currentAction.Name,
			UID:        currentAction.UID,
		})
	}

	if collectWorkloadUID {
		metadataAttributes = append(metadataAttributes, workloadUIDAttributes...)
		if collectReplicaSet {
			metadataAttributes = append(metadataAttributes, string(semconv.K8SReplicaSetUIDKey))
		}
	}

	if collectWorkloadNames {
		metadataAttributes = append(metadataAttributes, workloadNameAttributes...)
	}

	if collectReplicaSet {
		metadataAttributes = append(metadataAttributes, string(semconv.K8SReplicaSetNameKey))
	}

	if collectContainer {
		metadataAttributes = append(metadataAttributes, containerAttributes...)
	}

	if collectClusterUID {
		metadataAttributes = append(metadataAttributes, string(semconv.K8SClusterUIDKey))
	}

	// Convert maps back to slices
	var labelAttrs []k8sTagAttribute
	for _, attr := range labelAttributes {
		labelAttrs = append(labelAttrs, attr)
	}

	var annotationAttrs []k8sTagAttribute
	for _, attr := range annotationAttributes {
		annotationAttrs = append(annotationAttrs, attr)
	}

	return &k8sAttributesConfig{
		AuthType:    "serviceAccount",
		Passthrough: false,
		Filter: k8sAttributesFilter{
			NodeFromEnvVar: k8sconsts.NodeNameEnvVar,
		},
		Extract: k8sAttributeExtract{
			MetadataAttributes:   metadataAttributes,
			LabelAttributes:      labelAttrs,
			AnnotationAttributes: annotationAttrs,
		},
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
	}, signals, ownerReferences, nil
}
