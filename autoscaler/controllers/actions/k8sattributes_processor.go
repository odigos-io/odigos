package actions

import (
	"context"
	"slices"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	metadataAttributes   []string
	labelAttributes      = make(map[string]k8sTagAttribute)
	annotationAttributes = make(map[string]k8sTagAttribute)
	collectContainer     = false
	collectReplicaSet    = false
	collectWorkloadUID   = false
	collectClusterUID    = false
	collectWorkloadNames = false
)

type k8sAttributesProcessorConfig struct {
	AuthType       string                       `json:"auth_type"`
	Passthrough    bool                         `json:"passthrough"`
	Filter         k8sAttributesFilter          `json:"filter"`
	Extract        k8sAttributeExtract          `json:"extract"`
	PodAssociation k8sAttributesPodsAssociation `json:"pod_association"`
}

// k8sAttributeConfig combines multiple k8sattributes configurations into a single unified processor config
func k8sAttributeConfig(ctx context.Context, k8sclient client.Client, namespace string) (*k8sAttributesProcessorConfig, error) {
	// Get all actions in the namespace
	actionList := &odigosv1.ActionList{}
	err := k8sclient.List(ctx, actionList, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	// Collect all k8sattributes configurations
	var k8sAttributesConfigs []odigosv1.Action
	for _, a := range actionList.Items {
		if a.Spec.K8sAttributes != nil && !a.Spec.Disabled {
			k8sAttributesConfigs = append(k8sAttributesConfigs, a)
		}
	}

	// create a union of all the actions' configuration to one processor
	for _, config := range k8sAttributesConfigs {
		if config.Spec.K8sAttributes == nil {
			continue
		}

		collectContainer = collectContainer || config.Spec.K8sAttributes.CollectContainerAttributes
		collectReplicaSet = collectReplicaSet || config.Spec.K8sAttributes.CollectReplicaSetAttributes
		collectWorkloadUID = collectWorkloadUID || config.Spec.K8sAttributes.CollectWorkloadUID
		collectClusterUID = collectClusterUID || config.Spec.K8sAttributes.CollectClusterUID
		// traces should already contain workload name (if they originated from odigos)
		// logs collected from filelog receiver will lack this info thus needs to be added
		collectWorkloadNames = (collectWorkloadNames || slices.Contains(config.Spec.Signals, common.LogsObservabilitySignal))

		// Add label attributes, newer configs override older ones with same Tag
		for _, label := range config.Spec.K8sAttributes.LabelsAttributes {
			labelAttributes[label.AttributeKey] = k8sTagAttribute{
				Tag:  label.AttributeKey,
				Key:  label.LabelKey,
				From: "pod",
			}
		}

		// Add annotation attributes, newer configs override older ones with same Tag
		for _, annotation := range config.Spec.K8sAttributes.AnnotationsAttributes {
			annotationAttributes[annotation.AttributeKey] = k8sTagAttribute{
				Tag:  annotation.AttributeKey,
				Key:  annotation.AnnotationKey,
				From: "pod",
			}
		}
	}

	// Build metadata attributes list based on combined configuration
	if collectContainer {
		metadataAttributes = append(metadataAttributes, containerAttributes...)
	}

	if collectWorkloadNames {
		metadataAttributes = append(metadataAttributes, workloadNameAttributes...)
	}

	if collectReplicaSet {
		metadataAttributes = append(metadataAttributes, string(semconv.K8SReplicaSetNameKey))
		if collectWorkloadUID {
			metadataAttributes = append(metadataAttributes, string(semconv.K8SReplicaSetUIDKey))
		}
	}

	if collectWorkloadUID {
		metadataAttributes = append(metadataAttributes, workloadUIDAttributes...)
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

	return &k8sAttributesProcessorConfig{
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
	}, nil
}
