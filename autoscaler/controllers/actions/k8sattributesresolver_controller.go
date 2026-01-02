package actions

import (
	"context"
	"fmt"
	"sort"

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

// getEffectiveSources returns the list of sources to use for label extraction.
// It handles backward compatibility with the deprecated From field.
// If FromSources is specified, it returns those sources.
// If only From is specified (deprecated), it returns a single-element slice with that source.
// If neither is specified, it defaults to pod.
func getEffectiveSources(from *actionv1.K8sAttributeSource, fromSources []actionv1.K8sAttributeSource) []actionv1.K8sAttributeSource {
	if len(fromSources) > 0 {
		return fromSources
	}
	if from != nil {
		return []actionv1.K8sAttributeSource{*from}
	}
	return []actionv1.K8sAttributeSource{actionv1.PodAttributeSource}
}

// getEffectiveSourcesFromString is similar to getEffectiveSources but handles the string pointer type
// used in K8sAnnotationAttribute.From for backward compatibility.
func getEffectiveSourcesFromString(from *string, fromSources []actionv1.K8sAttributeSource) []actionv1.K8sAttributeSource {
	if len(fromSources) > 0 {
		return fromSources
	}
	if from != nil {
		return []actionv1.K8sAttributeSource{actionv1.K8sAttributeSource(*from)}
	}
	return []actionv1.K8sAttributeSource{actionv1.PodAttributeSource}
}

// sourcePrecedence returns the precedence value for a source.
// Lower values mean lower precedence (will be overwritten by higher precedence).
func sourcePrecedence(source actionv1.K8sAttributeSource) int {
	for i, s := range actionv1.K8sAttributeSourcePrecedence {
		if s == source {
			return i
		}
	}
	// Unknown sources get lowest precedence
	return -1
}

// sortByPrecedence converts a map of k8sTagAttribute to a sorted slice.
// The slice is sorted by source precedence (lower precedence first),
// so that when the k8sattributes processor iterates through them,
// higher precedence sources overwrite lower precedence ones.
func sortByPrecedence(attrs map[string]k8sTagAttribute) []k8sTagAttribute {
	result := make([]k8sTagAttribute, 0, len(attrs))
	for _, attr := range attrs {
		result = append(result, attr)
	}
	sort.Slice(result, func(i, j int) bool {
		// Sort by source precedence (lower precedence first)
		precedenceI := sourcePrecedence(actionv1.K8sAttributeSource(result[i].From))
		precedenceJ := sourcePrecedence(actionv1.K8sAttributeSource(result[j].From))
		if precedenceI != precedenceJ {
			return precedenceI < precedenceJ
		}
		// For same precedence, sort by key for deterministic ordering
		return result[i].Key < result[j].Key
	})
	return result
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

		// Add label attributes
		// For each label, we may have multiple sources (e.g., pod and namespace).
		// We need to generate separate processor config entries for each source,
		// with the correct precedence order (lower precedence sources processed first,
		// so higher precedence sources can override).
		for _, label := range config.LabelsAttributes {
			sources := getEffectiveSources(label.From, label.FromSources)
			for _, source := range sources {
				// Use a composite labelKeyWithSource of LabelKey and source to allow multiple sources for the same label
				labelKeyWithSource := fmt.Sprintf("%s:%s", label.LabelKey, string(source))
				labelAttributes[labelKeyWithSource] = k8sTagAttribute{
					Tag:  label.AttributeKey,
					Key:  label.LabelKey,
					From: string(source),
				}
			}
		}

		// Add annotation attributes
		// For each annotation, we may have multiple sources (e.g., pod and namespace).
		// We need to generate separate processor config entries for each source,
		// with the correct precedence order.
		for _, annotation := range config.AnnotationsAttributes {
			sources := getEffectiveSourcesFromString(annotation.From, annotation.FromSources)
			for _, source := range sources {
				// Use a composite key of AnnotationKey and source to allow multiple sources for the same annotation
				annotationKeyWithSource := fmt.Sprintf("%s:%s", annotation.AnnotationKey, string(source))
				annotationAttributes[annotationKeyWithSource] = k8sTagAttribute{
					Tag:  annotation.AttributeKey,
					Key:  annotation.AnnotationKey,
					From: string(source),
				}
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

	// Convert maps back to slices, sorted by precedence (lower precedence first)
	// This ensures that when the k8sattributes processor processes these entries,
	// higher precedence sources (e.g., pod) are processed last and overwrite
	// lower precedence sources (e.g., namespace).
	labelAttrs := sortByPrecedence(labelAttributes)
	annotationAttrs := sortByPrecedence(annotationAttributes)

	if len(metadataAttributes) == 0 {
		// when metadata attributes are not set, the collector will take the default
		// attributes for extract.metadata with k8s.deployment.name which can be very expensive.
		// using just the pod name should bypass this flaw in the collector.
		metadataAttributes = append(metadataAttributes, string(semconv.K8SPodNameKey))
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
