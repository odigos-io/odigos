package actions

import (
	"context"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"

	semconv1_21 "go.opentelemetry.io/otel/semconv/v1.21.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DEPRECATED: Use odigosv1.Action instead
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
	logger.V(0).Info("WARNING: K8sAttributes action is deprecated and will be removed in a future version. Migrate to odigosv1.Action instead.")

	// Get the specific K8sAttributesResolver that triggered this reconcile
	action := &actionv1.K8sAttributesResolver{}
	err := r.Get(ctx, req.NamespacedName, action)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Migrate to odigosv1.Action
	migratedActionName := odigosv1.ActionMigratedLegacyPrefix + action.Name
	odigosAction := &odigosv1.Action{}
	err = r.Get(ctx, client.ObjectKey{Name: migratedActionName, Namespace: action.Namespace}, odigosAction)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		logger.V(0).Info("Migrating legacy Action to odigosv1.Action. This is a one-way change, and modifications to the legacy Action will not be reflected in the migrated Action.")
		// Action doesn't exist, create new one
		odigosAction = r.createMigratedAction(action, migratedActionName)
		err = r.Create(ctx, odigosAction)
		if err != nil {
			return ctrl.Result{}, err
		}
		action.OwnerReferences = append(action.OwnerReferences, metav1.OwnerReference{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Action",
			Name:       odigosAction.Name,
			UID:        odigosAction.UID,
		})
		err = r.Update(ctx, action)
		return ctrl.Result{}, err
	}

	logger.V(0).Info("Migrated Action already exists, skipping update")
	return ctrl.Result{}, nil
}

func (r *K8sAttributesResolverReconciler) createMigratedAction(action *actionv1.K8sAttributesResolver, migratedActionName string) *odigosv1.Action {
	config := actionv1.K8sAttributesConfig{
		CollectContainerAttributes: action.Spec.CollectContainerAttributes,
		CollectClusterUID:          action.Spec.CollectClusterUID,
		LabelsAttributes:           action.Spec.LabelsAttributes,
		AnnotationsAttributes:      action.Spec.AnnotationsAttributes,
	}

	odigosAction := &odigosv1.Action{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Action",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      migratedActionName,
			Namespace: action.Namespace,
		},
		Spec: odigosv1.ActionSpec{
			ActionName:    action.Spec.ActionName,
			Notes:         action.Spec.Notes,
			Disabled:      action.Spec.Disabled,
			Signals:       action.Spec.Signals,
			K8sAttributes: &config,
		},
	}

	return odigosAction
}

// k8sAttributeConfig combines multiple k8sattributes configurations into a single unified processor config
func k8sAttributeConfig(ctx context.Context, k8sclient client.Client, namespace string) (*k8sAttributesConfig, map[common.ObservabilitySignal]struct{}, []metav1.OwnerReference, error) {
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
		collectClusterUID    = false
	)

	for i := range actionList.Items {
		ownerReferences = append(ownerReferences, metav1.OwnerReference{
			APIVersion: actionList.Items[i].APIVersion,
			Kind:       actionList.Items[i].Kind,
			Name:       actionList.Items[i].Name,
			UID:        actionList.Items[i].UID,
		})
	}

	// Collect all k8sattributes configurations
	for _, currentAction := range actionList.Items {
		if currentAction.Spec.K8sAttributes == nil || currentAction.Spec.Disabled {
			continue
		}

		config := currentAction.Spec.K8sAttributes

		// create a union of all the actions' configuration to one processor
		collectContainer = collectContainer || config.CollectContainerAttributes
		collectClusterUID = collectClusterUID || config.CollectClusterUID

		// Add label attributes, newer configs override older ones with same Tag
		for _, label := range config.LabelsAttributes {
			from := actionv1.PodAttributeSource
			if label.From != nil {
				from = actionv1.K8sAttributeSource(*label.From)
			}
			labelAttributes[label.LabelKey] = k8sTagAttribute{
				Tag:  label.AttributeKey,
				Key:  label.LabelKey,
				From: string(from),
			}
		}

		// Add annotation attributes, newer configs override older ones with same Tag
		for _, annotation := range config.AnnotationsAttributes {
			from := actionv1.PodAttributeSource
			if annotation.From != nil {
				from = actionv1.K8sAttributeSource(*annotation.From)
			}
			annotationAttributes[annotation.AnnotationKey] = k8sTagAttribute{
				Tag:  annotation.AttributeKey,
				Key:  annotation.AnnotationKey,
				From: string(from),
			}
		}

		for signalIndex := range currentAction.Spec.Signals {
			signals[currentAction.Spec.Signals[signalIndex]] = struct{}{}
		}

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
