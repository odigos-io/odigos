package controllers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type OdigletPodsWebhook struct {
	client.Client
	Decoder admission.Decoder
}

var _ admission.Handler = &OdigletPodsWebhook{}

func (o *OdigletPodsWebhook) InjectDecoder(d admission.Decoder) error {
	o.Decoder = d
	return nil
}

// Handle implements the admission.Handler interface to mutate Odiglet Pod objects at creation/update time.
// This webhook modifies the odiglet pod based on NodeDetails for the target node.
// If NodeDetails doesn't exist, the pod is allowed through unchanged so it can run discovery.
// If NodeDetails exists, discovery is removed and node-specific configurations are applied.
func (o *OdigletPodsWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx)

	pod := &corev1.Pod{}
	err := o.Decoder.Decode(req, pod)
	if err != nil {
		logger.Error(err, "Failed to decode pod")
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Extract target node name from pod affinity
	nodeName := extractTargetNodeName(pod)
	if nodeName == "" {
		// Can't determine target node, allow it through unchanged
		return admission.Allowed("unable to determine target node")
	}

	// Get NodeDetails for this node (if exists)
	nodeDetails := &odigosv1.NodeDetails{}
	err = o.Client.Get(ctx, types.NamespacedName{Name: nodeName, Namespace: req.Namespace}, nodeDetails)
	nodeDetailsExists := err == nil

	// Apply modifications to the odiglet container
	modified := false
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == k8sconsts.OdigletDaemonSetName {
			containerModified := applyOdigletContainerModifications(&pod.Spec.Containers[i], nodeDetails, nodeDetailsExists, nodeName, logger)
			modified = modified || containerModified
			break
		}
	}

	if !modified {
		// No changes needed
		return admission.Allowed("no modifications applied")
	}

	// Marshal the modified pod
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		logger.Error(err, "Failed to marshal modified pod")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Return patch response
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// extractTargetNodeName extracts the node name from the pod's NodeAffinity (used by DaemonSets).
// Falls back to pod.Spec.NodeName if affinity is not set.
func extractTargetNodeName(pod *corev1.Pod) string {
	// Try to get node name from NodeAffinity (DaemonSets use this)
	if pod.Spec.Affinity != nil && pod.Spec.Affinity.NodeAffinity != nil &&
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		nodeSelectorTerms := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
		for _, term := range nodeSelectorTerms {
			for _, matchField := range term.MatchFields {
				if matchField.Key == "metadata.name" && len(matchField.Values) > 0 {
					return matchField.Values[0]
				}
			}
		}
	}

	// Fallback to NodeName if already scheduled
	return pod.Spec.NodeName
}

// applyOdigletContainerModifications applies all NodeDetails-based modifications to the odiglet container.
// Returns true if any modifications were made.
func applyOdigletContainerModifications(container *corev1.Container, nodeDetails *odigosv1.NodeDetails, nodeDetailsExists bool, nodeName string, logger logr.Logger) bool {
	modified := false

	// Only apply modifications if NodeDetails exists
	// If NodeDetails doesn't exist, the pod needs to run discovery first
	if nodeDetailsExists {
		// Modification 1: Remove "discovery" argument
		// NodeDetails exists, so discovery has already been completed
		if removeDiscoveryArgument(container, nodeName, logger) {
			modified = true
		}

		// Modification 2: Add "--wasp-enabled" argument if WaspRequired is true
		if addWaspEnabledArgument(container, nodeDetails, nodeName, logger) {
			modified = true
		}

		// Future modifications can be added here as separate functions
		// Example: if addSomeOtherArgument(container, nodeDetails, nodeName, logger) { modified = true }
	}

	return modified
}

// removeDiscoveryArgument removes the "discovery" argument from the container args.
// This is called only when NodeDetails exists, preventing the pod from re-running discovery.
// Returns true if the argument was found and removed.
func removeDiscoveryArgument(container *corev1.Container, nodeName string, logger logr.Logger) bool {
	newArgs := []string{}
	found := false

	for _, arg := range container.Args {
		if arg != "discovery" {
			newArgs = append(newArgs, arg)
		} else {
			found = true
			logger.Info("Removing 'discovery' argument from odiglet container", "node", nodeName)
		}
	}

	if found {
		container.Args = newArgs
	}

	return found
}

// addWaspEnabledArgument adds the "--wasp-enabled" argument if WaspRequired is true in NodeDetails.
// This enables WASP instrumentation for the odiglet on this node.
// Returns true if the argument was added.
func addWaspEnabledArgument(container *corev1.Container, nodeDetails *odigosv1.NodeDetails, nodeName string, logger logr.Logger) bool {
	if !nodeDetails.Spec.WaspRequired {
		return false
	}

	// Check if --wasp-enabled already exists
	for _, arg := range container.Args {
		if arg == "--wasp-enabled" {
			return false // Already present
		}
	}

	// Add the argument
	container.Args = append(container.Args, "--wasp-enabled")
	logger.Info("Adding '--wasp-enabled' argument to odiglet container",
		"node", nodeName, "nodeDetails", nodeDetails.Name)

	return true
}
